//go:build !windows && !linux && !darwin

package mihomo

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"

	pkgerrors "github.com/kkkqkx123/mihomo-cli/pkg/errors"
)

// otherProcessChecker 其他平台进程检查器（不支持）
type otherProcessChecker struct{}

// newProcessChecker 创建进程检查器（不支持的平台）
func newProcessChecker() ProcessChecker {
	return &otherProcessChecker{}
}

// IsProcessRunning 检查进程是否正在运行（不支持的平台）
func (o *otherProcessChecker) IsProcessRunning(pid int) bool {
	// 尝试使用通用方法检查进程
	proc, err := os.FindProcess(pid)
	if err != nil {
		return false
	}
	
	// 尝试发送信号 0 检查进程是否存在
	// 这在某些平台上可能有效
	err = proc.Signal(nil)
	return err == nil
}

// GetProcessExecutable 获取进程的可执行文件路径（不支持的平台）
func (o *otherProcessChecker) GetProcessExecutable(pid int) (string, error) {
	return "", pkgerrors.ErrService(
		fmt.Sprintf("getting process executable not supported on %s", runtime.GOOS), nil)
}

// SetSysProcAttr 设置进程的系统属性（不支持的平台）
func (o *otherProcessChecker) SetSysProcAttr(cmd *exec.Cmd) {
	// 其他平台不需要设置特殊的系统属性
	// 保持 cmd.SysProcAttr 为 nil
}

// SendGracefulSignal 发送优雅关闭信号（不支持的平台）
func (o *otherProcessChecker) SendGracefulSignal(proc *os.Process) error {
	return pkgerrors.ErrService(
		fmt.Sprintf("graceful shutdown not supported on %s, use Kill instead", runtime.GOOS), nil)
}
