package mihomo

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/kkkqkx123/mihomo-cli/internal/config"
	"github.com/kkkqkx123/mihomo-cli/internal/output"
	pkgerrors "github.com/kkkqkx123/mihomo-cli/pkg/errors"
)

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

// getConfigPathFromHash 从 hash 反推配置文件路径（简化版）
func getConfigPathFromHash(hash string) string {
	// 这是一个简化的实现，实际上需要维护一个映射表
	// 在实际应用中，可以在 PID 文件中存储配置文件的完整路径
	return ""
}

// extractAPIPortFromConfig 从配置文件提取 API 端口（简化版）
func extractAPIPortFromConfig(configFile string) (string, error) {
	// 这是一个简化的实现，实际需要解析配置文件
	// 可以使用 yaml 或 toml 解析器
	return "", pkgerrors.ErrService("not implemented", nil)
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
