//go:build linux

package mihomo

import (
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"

	pkgerrors "github.com/kkkqkx123/mihomo-cli/pkg/errors"
)

// linuxProcessChecker Linux 平台进程检查器
type linuxProcessChecker struct{}

// newProcessChecker 创建进程检查器（Linux 平台）
func newProcessChecker() ProcessChecker {
	return &linuxProcessChecker{}
}

// IsProcessRunning 检查进程是否正在运行
func (l *linuxProcessChecker) IsProcessRunning(pid int) bool {
	// 在 Linux 系统上，通过 /proc/<pid> 目录检查进程是否存在
	procPath := filepath.Join("/proc", strconv.Itoa(pid))
	
	// 尝试读取 /proc/<pid>/stat 文件
	statPath := filepath.Join(procPath, "stat")
	data, err := os.ReadFile(statPath)
	if err != nil {
		return false
	}
	
	// 如果文件存在且不为空，进程正在运行
	return len(data) > 0
}

// GetProcessExecutable 获取进程的可执行文件路径
func (l *linuxProcessChecker) GetProcessExecutable(pid int) (string, error) {
	// 在 Linux 系统上，通过 /proc/<pid>/exe 符号链接获取可执行文件路径
	procPath := filepath.Join("/proc", strconv.Itoa(pid), "exe")
	
	// 读取符号链接
	execPath, err := os.Readlink(procPath)
	if err != nil {
		// 如果符号链接读取失败，尝试从 cmdline 获取
		cmdlinePath := filepath.Join("/proc", strconv.Itoa(pid), "cmdline")
		data, err := os.ReadFile(cmdlinePath)
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

// SetSysProcAttr 设置进程的系统属性（Linux 平台无需特殊设置）
func (l *linuxProcessChecker) SetSysProcAttr(cmd *exec.Cmd) {
	// Linux 平台不需要设置特殊的系统属性
	// 保持 cmd.SysProcAttr 为 nil
}

// SendGracefulSignal 发送优雅关闭信号（Linux 使用 SIGTERM）
// SIGTERM 是 Linux 系统上标准的优雅终止信号
func (l *linuxProcessChecker) SendGracefulSignal(proc *os.Process) error {
	// Linux 系统使用 SIGTERM 信号
	// 这会给进程机会执行清理操作
	if err := proc.Signal(syscall.SIGTERM); err != nil {
		return pkgerrors.ErrService("failed to send SIGTERM signal", err)
	}
	return nil
}
