//go:build !windows && !linux && !darwin

package mihomo

import (
	"fmt"
	"os"
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

// getProcessResourceUsage 获取进程资源使用情况 (不支持的平台)
func getProcessResourceUsage(pid int) (cpu, memory float64, err error) {
	return 0, 0, pkgerrors.ErrService(
		fmt.Sprintf("getting process resource usage not supported on %s", runtime.GOOS), nil)
}
