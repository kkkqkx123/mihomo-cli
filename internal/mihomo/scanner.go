package mihomo

import (
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/kkkqkx123/mihomo-cli/internal/config"
	"github.com/kkkqkx123/mihomo-cli/internal/output"
	pkgerrors "github.com/kkkqkx123/mihomo-cli/pkg/errors"
)

// sameExecutablePath 比较两个可执行文件路径是否指向同一文件（规范化后忽略大小写）
func sameExecutablePath(a, b string) bool {
	return strings.EqualFold(filepath.Clean(a), filepath.Clean(b))
}

// apiPortFromAddr 从 external-controller 地址提取端口号
// 格式如 "127.0.0.1:9090" 或 ":9090"，解析失败返回空串
func apiPortFromAddr(apiAddr string) string {
	parts := strings.Split(apiAddr, ":")
	if len(parts) < 2 {
		return ""
	}
	port := parts[len(parts)-1]
	if port == "" {
		return ""
	}
	return port
}

// VerifyMihomoProcess 验证进程是否是 Mihomo 进程
func VerifyMihomoProcess(pid int) (bool, error) {
	execPath, err := GetProcessExecutable(pid)
	if err != nil {
		return false, pkgerrors.ErrService("failed to get process executable", err)
	}

	// 检查可执行文件名是否包含 "mihomo"
	basename := strings.ToLower(filepath.Base(execPath))
	return strings.Contains(basename, "mihomo"), nil
}

// extractAPIPortFromConfig 从配置文件提取 API 端口（兜底：元数据缺失 API 地址时使用）
func extractAPIPortFromConfig(configFile string) (string, error) {
	data, err := os.ReadFile(configFile)
	if err != nil {
		return "", pkgerrors.ErrConfig("failed to read config file", err)
	}

	// 解析 YAML 配置
	var cfg struct {
		ExternalController    string `yaml:"external-controller"`
		ExternalControllerTLS string `yaml:"external-controller-tls"`
	}

	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return "", pkgerrors.ErrConfig("failed to parse config file", err)
	}

	// 优先使用 TLS 端口
	apiAddress := cfg.ExternalControllerTLS
	if apiAddress == "" {
		apiAddress = cfg.ExternalController
	}

	if apiAddress == "" {
		return "", pkgerrors.ErrConfig("no external-controller found in config", nil)
	}

	port := apiPortFromAddr(apiAddress)
	if port == "" {
		return "", pkgerrors.ErrConfig("invalid external-controller format: "+apiAddress, nil)
	}

	return port, nil
}

// CleanupPIDFiles 清理所有残留的 PID 元数据文件
func CleanupPIDFiles() error {
	// 创建路径解析器
	pathResolver, err := config.NewPathResolver()
	if err != nil {
		return pkgerrors.ErrConfig("failed to create path resolver", err)
	}

	pidDir := pathResolver.GetPIDDir()

	// 读取所有 PID 文件
	entries, err := os.ReadDir(pidDir)
	if err != nil {
		if os.IsNotExist(err) {
			// PID 目录不存在，无需清理
			return nil
		}
		return pkgerrors.ErrConfig("failed to read pid directory", err)
	}

	cleanedCount := 0

	// 遍历所有 PID 文件
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".pid") {
			continue
		}

		pidFile := filepath.Join(pidDir, entry.Name())
		meta, err := ReadInstanceFile(pidFile)
		if err != nil {
			// PID 文件损坏，删除（不误判、不误杀）
			os.Remove(pidFile)
			cleanedCount++
			continue
		}

		// 检查进程是否存在
		if !IsProcessRunning(meta.PID) {
			// 进程不存在，删除 PID 文件
			os.Remove(pidFile)
			cleanedCount++
		}
	}

	if cleanedCount > 0 {
		output.Printf("清理了 %d 个残留的 PID 文件\n", cleanedCount)
	} else {
		output.Println("没有需要清理的残留 PID 文件")
	}

	return nil
}

// GetAllMihomoPIDs 获取所有已验证的 Mihomo 进程 PID
func GetAllMihomoPIDs() ([]int, error) {
	processes, err := ScanMihomoProcesses()
	if err != nil {
		return nil, pkgerrors.ErrService("failed to scan mihomo processes", err)
	}

	pids := []int{}
	for _, proc := range processes {
		if proc.IsVerified {
			pids = append(pids, proc.PID)
		}
	}

	return pids, nil
}

// StopAllMihomoProcesses 停止所有 Mihomo 进程。
// 默认只停止 confirmed（托管且路径一致）实例；
// includeUnmanaged=true 时才扩展到 likely（外部/未验证）实例。
// 对 confirmed 实例优先通过 API 优雅关闭，失败再 ForceKill；
// 对 likely 实例直接 ForceKill（无 API 信息）。
func StopAllMihomoProcesses(includeUnmanaged bool) error {
	processes, err := ScanMihomoProcesses()
	if err != nil {
		return pkgerrors.ErrService("failed to scan mihomo processes", err)
	}

	// 按验证级别过滤
	var targets []ProcessInfo
	for _, proc := range processes {
		if proc.Verified == VerifyConfirmed || (includeUnmanaged && proc.Verified == VerifyLikely) {
			targets = append(targets, proc)
		}
	}

	if len(targets) == 0 {
		output.Println("没有正在运行的 Mihomo 进程")
		return nil
	}

	output.Printf("找到 %d 个 Mihomo 进程，开始停止...\n", len(targets))

	// 加载配置以获取 API 密钥
	secret := ""
	if cliCfg, err := config.LoadFromViper(); err == nil && cliCfg != nil {
		secret = cliCfg.API.Secret
	}

	stoppedCount := 0
	for _, proc := range targets {
		pid := proc.PID
		if !IsProcessRunning(pid) {
			continue
		}

		// 尝试 API 优雅关闭（需要端口和密钥）
		if proc.APIPort != "" && secret != "" {
			apiAddr := "127.0.0.1:" + proc.APIPort
			if err := StopProcessByPID(pid, apiAddr, secret); err == nil {
				output.Success("  ✓ 已优雅停止进程 %d", pid)
				stoppedCount++
				continue
			}
			output.Warning("  API 优雅关闭进程 %d 失败，尝试强制终止...", pid)
		}

		// 强制终止
		if err := ForceKill(pid); err != nil {
			output.Error("  ✗ 无法停止进程 %d: %v", pid, err)
			continue
		}
		output.Success("  ✓ 已强制停止进程 %d", pid)
		stoppedCount++
	}

	// 清理已退出进程对应的 PID 元数据文件（不误删仍在运行的实例）
	pathResolver, _ := config.NewPathResolver()
	pidDir := pathResolver.GetPIDDir()
	entries, _ := os.ReadDir(pidDir)
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".pid") {
			continue
		}

		pidFile := filepath.Join(pidDir, entry.Name())
		meta, err := ReadInstanceFile(pidFile)
		if err != nil || !IsProcessRunning(meta.PID) {
			os.Remove(pidFile)
		}
	}

	output.Printf("停止完成: 成功 %d 个\n", stoppedCount)
	return nil
}
