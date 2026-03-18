package mihomo

// ProcessInfo 进程信息（跨平台定义）
type ProcessInfo struct {
	PID       int      // 进程 ID
	ExecPath  string   // 可执行文件路径
	APIPort   string   // API 端口（如果能从配置中获取）
	StartTime string   // 启动时间
	CmdLine   string   // 命令行参数
	IsVerified bool    // 是否已验证为 Mihomo 进程
}
