package mihomo

// Source 进程来源
type Source string

const (
	// SourceManaged 由本 CLI 启动并留有元数据
	SourceManaged Source = "managed"
	// SourceExternal 全进程表枚举发现（外部启动）
	SourceExternal Source = "external"
)

// VerifyLevel 验证级别
type VerifyLevel string

const (
	// VerifyConfirmed 命中元数据且可执行路径一致（或配置显式匹配）
	VerifyConfirmed VerifyLevel = "confirmed"
	// VerifyLikely 仅名称/命令行匹配（external 或元数据缺失）
	VerifyLikely VerifyLevel = "likely"
)

// ProcessInfo 进程信息（跨平台定义）
type ProcessInfo struct {
	PID        int         // 进程 ID
	ExecPath   string      // 可执行文件绝对路径
	APIPort    string      // API 端口（从元数据/配置文件解析）
	StartTime  string      // 启动时间
	CmdLine    string      // 真实命令行（可含参数）；不可用时为空
	IsVerified bool        // 是否验证为 Mihomo 进程（confirmed 或 likely）
	Source     Source      // managed / external
	Verified   VerifyLevel // confirmed / likely
	ConfigFile string      // managed 实例对应的 mihomo 配置文件
}
