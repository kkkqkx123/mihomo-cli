package mihomo

// ProcessChecker 进程检查器接口（跨平台抽象，三平台语义一致）
type ProcessChecker interface {
	// IsProcessRunning 检查进程是否正在运行。
	// 权限不足（Access Denied / EACCES / EPERM）时保守返回 true，避免误判进程不存在。
	IsProcessRunning(pid int) bool

	// GetProcessExecutable 获取进程的可执行文件绝对路径（规范化）。
	// 不可用时返回可读错误。
	GetProcessExecutable(pid int) (string, error)

	// GetProcessCommandLine 获取进程的完整命令行（含参数，可含空格）。
	// 不可用时返回可读错误。
	GetProcessCommandLine(pid int) (string, error)
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

// GetProcessCommandLine 获取进程的完整命令行（跨平台入口）
func GetProcessCommandLine(pid int) (string, error) {
	return processChecker.GetProcessCommandLine(pid)
}
