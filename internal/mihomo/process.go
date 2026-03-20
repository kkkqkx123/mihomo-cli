package mihomo

import (
	"os"
	"os/exec"
)

// ProcessChecker 进程检查器接口（跨平台抽象）
type ProcessChecker interface {
	// IsProcessRunning 检查进程是否正在运行
	IsProcessRunning(pid int) bool

	// GetProcessExecutable 获取进程的可执行文件路径
	GetProcessExecutable(pid int) (string, error)

	// SetSysProcAttr 设置进程的系统属性（如 Windows 下隐藏窗口）
	SetSysProcAttr(cmd *exec.Cmd)

	// SendGracefulSignal 发送优雅关闭信号给进程
	// Windows: 使用 SIGINT (Ctrl+C)
	// Unix: 使用 SIGTERM
	SendGracefulSignal(proc *os.Process) error
}

// processChecker 全局进程检查器实例
var processChecker ProcessChecker

// init 初始化进程检查器
func init() {
	processChecker = newProcessChecker()
}

// IsProcessRunning 检查进程是否正在运行（跨平台入口）
func IsProcessRunning(pid int) bool {
	return processChecker.IsProcessRunning(pid)
}

// GetProcessExecutable 获取进程的可执行文件路径（跨平台入口）
func GetProcessExecutable(pid int) (string, error) {
	return processChecker.GetProcessExecutable(pid)
}

// SetSysProcAttr 设置进程的系统属性（跨平台入口）
func SetSysProcAttr(cmd *exec.Cmd) {
	processChecker.SetSysProcAttr(cmd)
}

// SendGracefulSignal 发送优雅关闭信号（跨平台入口）
func SendGracefulSignal(proc *os.Process) error {
	return processChecker.SendGracefulSignal(proc)
}
