//go:build darwin

package mihomo

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"syscall"

	pkgerrors "github.com/kkkqkx123/mihomo-cli/pkg/errors"
)

// darwinProcessChecker macOS 平台进程检查器
type darwinProcessChecker struct{}

// newProcessChecker 创建进程检查器（macOS 平台）
func newProcessChecker() ProcessChecker {
	return &darwinProcessChecker{}
}

// IsProcessRunning 检查进程是否正在运行
func (d *darwinProcessChecker) IsProcessRunning(pid int) bool {
	// 在 macOS 上，使用 os.FindProcess 和信号 0 检查进程是否存在
	// 信号 0 不会实际发送信号，只是检查进程是否存在
	proc, err := os.FindProcess(pid)
	if err != nil {
		return false
	}
	
	// 发送信号 0 检查进程是否存在
	err = proc.Signal(syscall.Signal(0))
	return err == nil
}

// GetProcessExecutable 获取进程的可执行文件路径
func (d *darwinProcessChecker) GetProcessExecutable(pid int) (string, error) {
	// 在 macOS 上，使用 ps 命令获取可执行文件路径
	// ps -p <pid> -o comm= 只返回命令名
	// ps -p <pid> -o args= 返回完整命令行
	cmd := exec.Command("ps", "-p", fmt.Sprintf("%d", pid), "-o", "comm=")
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	
	if err := cmd.Run(); err != nil {
		return "", pkgerrors.ErrService("failed to get process executable: "+stderr.String(), err)
	}
	
	execPath := strings.TrimSpace(stdout.String())
	if execPath == "" {
		return "", pkgerrors.ErrService("empty process executable path", nil)
	}
	
	return execPath, nil
}

// SetSysProcAttr 设置进程的系统属性（macOS 平台无需特殊设置）
func (d *darwinProcessChecker) SetSysProcAttr(cmd *exec.Cmd) {
	// macOS 平台不需要设置特殊的系统属性
	// 保持 cmd.SysProcAttr 为 nil
}

// SendGracefulSignal 发送优雅关闭信号（macOS 使用 SIGTERM）
// SIGTERM 是 macOS 系统上标准的优雅终止信号
func (d *darwinProcessChecker) SendGracefulSignal(proc *os.Process) error {
	// macOS 系统使用 SIGTERM 信号
	// 这会给进程机会执行清理操作
	if err := proc.Signal(syscall.SIGTERM); err != nil {
		return pkgerrors.ErrService("failed to send SIGTERM signal", err)
	}
	return nil
}
