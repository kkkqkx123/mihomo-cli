//go:build !windows

package mihomo

import (
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	pkgerrors "github.com/kkkqkx123/mihomo-cli/pkg/errors"
)

// unixProcessChecker Unix 平台进程检查器
type unixProcessChecker struct{}

// newProcessChecker 创建进程检查器（Unix 平台）
func newProcessChecker() ProcessChecker {
	return &unixProcessChecker{}
}

// newUnixProcessChecker 创建 Unix 进程检查器
func newUnixProcessChecker() ProcessChecker {
	return &unixProcessChecker{}
}

// IsProcessRunning 检查进程是否正在运行
func (u *unixProcessChecker) IsProcessRunning(pid int) bool {
	// 在 Unix 系统上，通过 /proc/<pid> 目录检查进程是否存在
	procPath := filepath.Join("/proc", strconv.Itoa(pid))
	
	// 尝试读取 /proc/<pid>/stat 文件
	statPath := filepath.Join(procPath, "stat")
	data, err := readFile(statPath)
	if err != nil {
		return false
	}
	
	// 如果文件存在且不为空，进程正在运行
	return len(data) > 0
}

// GetProcessExecutable 获取进程的可执行文件路径
func (u *unixProcessChecker) GetProcessExecutable(pid int) (string, error) {
	// 在 Unix 系统上，通过 /proc/<pid>/exe 符号链接获取可执行文件路径
	procPath := filepath.Join("/proc", strconv.Itoa(pid), "exe")
	
	// 读取符号链接
	execPath, err := readLink(procPath)
	if err != nil {
		// 如果符号链接读取失败，尝试从 cmdline 获取
		cmdlinePath := filepath.Join("/proc", strconv.Itoa(pid), "cmdline")
		data, err := readFile(cmdlinePath)
		if err != nil {
			return "", pkgerrors.ErrService("failed to get process executable", err)
		}
		
		// cmdline 以 null 分隔参数，取第一个参数
		parts := strings.Split(string(data), "\x00")
		if len(parts) > 0 && parts[0] != "" {
			return parts[0], nil
		}
		
		return "", pkgerrors.ErrService("failed to get process executable", nil)
	}
	
	return execPath, nil
}

// SetSysProcAttr 设置进程的系统属性（Unix 平台无需特殊设置）
func (u *unixProcessChecker) SetSysProcAttr(cmd *exec.Cmd) {
	// Unix 平台不需要设置特殊的系统属性
	// 保持 cmd.SysProcAttr 为 nil
}

// readFile 读取文件内容（辅助函数）
func readFile(path string) ([]byte, error) {
	return os.ReadFile(path)
}

// readLink 读取符号链接（辅助函数）
func readLink(path string) (string, error) {
	return os.Readlink(path)
}
