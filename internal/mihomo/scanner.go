package mihomo

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"gopkg.in/yaml.v3"

	"github.com/kkkqkx123/mihomo-cli/internal/config"
	"github.com/kkkqkx123/mihomo-cli/internal/output"
	pkgerrors "github.com/kkkqkx123/mihomo-cli/pkg/errors"
)

// configHashMapping 配置文件 hash 到路径的映射表
var configHashMapping = make(map[string]string)
var configHashMutex sync.RWMutex

// RegisterConfigHash 注册配置文件的 hash 映射
func RegisterConfigHash(configPath string) error {
	if configPath == "" {
		return nil
	}

	// 计算配置文件路径的 hash
	hash := computeConfigHash(configPath)

	configHashMutex.Lock()
	configHashMapping[hash] = configPath
	configHashMutex.Unlock()

	return nil
}

// computeConfigHash 计算配置文件的 hash
func computeConfigHash(configPath string) string {
	// 使用文件路径的绝对路径作为 hash
	absPath, err := filepath.Abs(configPath)
	if err != nil {
		absPath = configPath
	}

	// 使用 SHA256 计算路径的 hash
	h := sha256.New()
	h.Write([]byte(absPath))
	return hex.EncodeToString(h.Sum(nil))[:8] // 取前8位作为短hash
}

// ScanMihomoProcesses 扫描所有 Mihomo 进程
func ScanMihomoProcesses() ([]ProcessInfo, error) {
	processes := []ProcessInfo{}

	// 获取 PID 文件目录
	pidDir, err := config.GetPIDDir()
	if err != nil {
		return nil, pkgerrors.ErrConfig("failed to get pid directory", err)
	}

	// 读取所有 PID 文件
	entries, err := os.ReadDir(pidDir)
	if err != nil {
		if os.IsNotExist(err) {
			// PID 目录不存在，返回空列表
			return processes, nil
		}
		return nil, pkgerrors.ErrConfig("failed to read pid directory", err)
	}

	// 遍历所有 PID 文件
	for _, entry := range entries {
		if !strings.HasSuffix(entry.Name(), ".pid") {
			continue
		}

		pidFile := filepath.Join(pidDir, entry.Name())
		pid, err := readPIDFromFile(pidFile)
		if err != nil {
			// PID 文件损坏，跳过
			continue
		}

		// 验证进程是否存在
		if !IsProcessRunning(pid) {
			// 进程不存在，标记为未验证
			continue
		}

		// 获取进程信息
		execPath, err := GetProcessExecutable(pid)
		if err != nil {
			continue
		}

		// 验证是否是 Mihomo 进程
		isMihomo := strings.Contains(strings.ToLower(filepath.Base(execPath)), "mihomo")

		// 从 PID 文件名推断配置文件
		configName := ""
		if strings.HasPrefix(entry.Name(), "mihomo-") && strings.HasSuffix(entry.Name(), ".pid") {
			configName = strings.TrimSuffix(strings.TrimPrefix(entry.Name(), "mihomo-"), ".pid")
		}

		// 从配置文件路径读取 API 端口
		apiPort := ""
		if configName != "" {
			configFile := getConfigPathFromHash(configName)
			if configFile != "" {
				if port, err := extractAPIPortFromConfig(configFile); err == nil {
					apiPort = port
				}
			}
		}

		processes = append(processes, ProcessInfo{
			PID:        pid,
			ExecPath:   execPath,
			APIPort:    apiPort,
			CmdLine:    execPath,
			IsVerified: isMihomo,
		})
	}

	return processes, nil
}

// readPIDFromFile 从文件读取 PID
func readPIDFromFile(pidFile string) (int, error) {
	data, err := os.ReadFile(pidFile)
	if err != nil {
		return 0, pkgerrors.ErrConfig("failed to read pid file", err)
	}

	var pid int
	_, err = fmt.Sscanf(string(data), "%d", &pid)
	if err != nil {
		return 0, pkgerrors.ErrConfig("invalid pid format", err)
	}

	return pid, nil
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

// getConfigPathFromHash 从 hash 反推配置文件路径
func getConfigPathFromHash(hash string) string {
	configHashMutex.RLock()
	configPath, exists := configHashMapping[hash]
	configHashMutex.RUnlock()

	if exists {
		return configPath
	}

	// 如果映射表中没有，尝试搜索基础目录
	baseDir, err := config.GetBaseDir()
	if err != nil {
		return ""
	}

	// 遍历基础目录，查找匹配的配置文件
	entries, err := os.ReadDir(baseDir)
	if err != nil {
		return ""
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		configFile := filepath.Join(baseDir, entry.Name())
		computedHash := computeConfigHash(configFile)

		if computedHash == hash {
			// 找到匹配的配置文件，添加到映射表
			configHashMutex.Lock()
			configHashMapping[hash] = configFile
			configHashMutex.Unlock()
			return configFile
		}
	}

	return ""
}

// extractAPIPortFromConfig 从配置文件提取 API 端口
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

	// 提取端口号，格式如 "127.0.0.1:9090" 或 ":9090"
	parts := strings.Split(apiAddress, ":")
	if len(parts) < 2 {
		return "", pkgerrors.ErrConfig("invalid external-controller format: "+apiAddress, nil)
	}

	return parts[len(parts)-1], nil
}

// CleanupPIDFiles 清理所有残留的 PID 文件
func CleanupPIDFiles() error {
	pidDir, err := config.GetPIDDir()
	if err != nil {
		return pkgerrors.ErrConfig("failed to get pid directory", err)
	}

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
		if !strings.HasSuffix(entry.Name(), ".pid") {
			continue
		}

		pidFile := filepath.Join(pidDir, entry.Name())
		pid, err := readPIDFromFile(pidFile)
		if err != nil {
			// PID 文件损坏，删除
			os.Remove(pidFile)
			cleanedCount++
			continue
		}

		// 检查进程是否存在
		if !IsProcessRunning(pid) {
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

// StopAllMihomoProcesses 停止所有 Mihomo 进程
func StopAllMihomoProcesses() error {
	pids, err := GetAllMihomoPIDs()
	if err != nil {
		return err
	}

	if len(pids) == 0 {
		output.Println("没有正在运行的 Mihomo 进程")
		return nil
	}

	output.Printf("找到 %d 个 Mihomo 进程，开始停止...\n", len(pids))

	stoppedCount := 0
	for _, pid := range pids {
		if IsProcessRunning(pid) {
			proc, err := os.FindProcess(pid)
			if err != nil {
				output.Error("  ✗ 无法停止进程 " + fmt.Sprintf("%d", pid) + ": " + err.Error())
				continue
			}

			if err := proc.Kill(); err != nil {
				output.Error("  ✗ 无法停止进程 " + fmt.Sprintf("%d", pid) + ": " + err.Error())
				continue
			}

			output.Success("  ✓ 已停止进程 " + fmt.Sprintf("%d", pid))
			stoppedCount++
		}
	}

	// 清理所有 PID 文件
	pidDir, _ := config.GetPIDDir()
	entries, _ := os.ReadDir(pidDir)
	for _, entry := range entries {
		if strings.HasSuffix(entry.Name(), ".pid") {
			os.Remove(filepath.Join(pidDir, entry.Name()))
		}
	}

	output.Printf("停止完成: 成功 %d 个\n", stoppedCount)
	return nil
}
