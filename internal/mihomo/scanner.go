//go:build windows

package mihomo

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"unsafe"

	pkgerrors "github.com/kkkqkx123/mihomo-cli/pkg/errors"
)

// ProcessInfo 进程信息
type ProcessInfo struct {
	PID       int      // 进程 ID
	ExecPath  string   // 可执行文件路径
	APIPort   string   // API 端口（如果能从配置中获取）
	StartTime string   // 启动时间
	CmdLine   string   // 命令行参数
	IsVerified bool    // 是否已验证为 Mihomo 进程
}

var (
	// Windows API 函数
	modkernel32           = syscall.NewLazyDLL("kernel32.dll")
	procOpenProcess       = modkernel32.NewProc("OpenProcess")
	procQueryFullProcessImageName = modkernel32.NewProc("QueryFullProcessImageNameW")
	procCloseHandle       = modkernel32.NewProc("CloseHandle")
	procGetModuleFileNameEx = modkernel32.NewProc("GetModuleFileNameExW")
	modpsapi              = syscall.NewLazyDLL("psapi.dll")
)

const (
	PROCESS_QUERY_INFORMATION = 0x0400
	PROCESS_VM_READ           = 0x0010
	MAX_PATH                  = 260
)

// isProcessRunningWindows 使用 Windows API 检查进程是否存在
func isProcessRunningWindows(pid int) bool {
	// 使用 PROCESS_QUERY_INFORMATION 权限打开进程
	// 如果进程不存在，OpenProcess 会返回 0
	handle, _, _ := procOpenProcess.Call(
		uintptr(PROCESS_QUERY_INFORMATION),
		0,
		uintptr(pid),
	)
	if handle == 0 {
		return false
	}
	// 关闭句柄
	procCloseHandle.Call(handle)
	return true
}

// ScanMihomoProcesses 扫描所有 Mihomo 进程
func ScanMihomoProcesses() ([]ProcessInfo, error) {
	processes := []ProcessInfo{}

	// 获取当前用户目录，用于查找 PID 文件
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, pkgerrors.ErrConfig("failed to get user home dir", err)
	}
	pidDir := filepath.Join(home, ".mihomo-cli")

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
			CmdLine:    fmt.Sprintf("%s", execPath),
			IsVerified: isMihomo,
		})
	}

	return processes, nil
}

// readPIDFromFile 从文件读取 PID
func readPIDFromFile(pidFile string) (int, error) {
	data, err := os.ReadFile(pidFile)
	if err != nil {
		return 0, err
	}

	var pid int
	_, err = fmt.Sscanf(string(data), "%d", &pid)
	if err != nil {
		return 0, err
	}

	return pid, nil
}

// GetProcessExecutable 获取进程的可执行文件路径
func GetProcessExecutable(pid int) (string, error) {
	// 打开进程
	handle, _, err := procOpenProcess.Call(
		uintptr(PROCESS_QUERY_INFORMATION|PROCESS_VM_READ),
		0,
		uintptr(pid),
	)
	if handle == 0 {
		return "", pkgerrors.ErrService("failed to open process", nil)
	}
	defer procCloseHandle.Call(handle)

	// 获取可执行文件路径
	var path [MAX_PATH]uint16
	var size uint32 = MAX_PATH

	ret, _, err := procQueryFullProcessImageName.Call(
		handle,
		0,
		uintptr(unsafe.Pointer(&path[0])),
		uintptr(unsafe.Pointer(&size)),
	)
	if ret == 0 {
		return "", pkgerrors.ErrService("failed to query process image name", err)
	}

	return syscall.UTF16ToString(path[:]), nil
}

// VerifyMihomoProcess 验证进程是否是 Mihomo 进程
func VerifyMihomoProcess(pid int) (bool, error) {
	execPath, err := GetProcessExecutable(pid)
	if err != nil {
		return false, err
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
	home, err := os.UserHomeDir()
	if err != nil {
		return pkgerrors.ErrConfig("failed to get user home dir", err)
	}
	pidDir := filepath.Join(home, ".mihomo-cli")

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
		fmt.Printf("清理了 %d 个残留的 PID 文件\n", cleanedCount)
	} else {
		fmt.Println("没有需要清理的残留 PID 文件")
	}

	return nil
}

// GetAllMihomoPIDs 获取所有已验证的 Mihomo 进程 PID
func GetAllMihomoPIDs() ([]int, error) {
	processes, err := ScanMihomoProcesses()
	if err != nil {
		return nil, err
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
		fmt.Println("没有正在运行的 Mihomo 进程")
		return nil
	}

	fmt.Printf("找到 %d 个 Mihomo 进程，开始停止...\n", len(pids))

	stoppedCount := 0
	for _, pid := range pids {
		if IsProcessRunning(pid) {
			proc, err := os.FindProcess(pid)
			if err != nil {
				fmt.Printf("  ✗ 无法停止进程 %d: %v\n", pid, err)
				continue
			}

			if err := proc.Kill(); err != nil {
				fmt.Printf("  ✗ 无法停止进程 %d: %v\n", pid, err)
				continue
			}

			fmt.Printf("  ✓ 已停止进程 %d\n", pid)
			stoppedCount++
		}
	}

	// 清理所有 PID 文件
	home, _ := os.UserHomeDir()
	pidDir := filepath.Join(home, ".mihomo-cli")
	entries, _ := os.ReadDir(pidDir)
	for _, entry := range entries {
		if strings.HasSuffix(entry.Name(), ".pid") {
			os.Remove(filepath.Join(pidDir, entry.Name()))
		}
	}

	fmt.Printf("停止完成: 成功 %d 个\n", stoppedCount)
	return nil
}
