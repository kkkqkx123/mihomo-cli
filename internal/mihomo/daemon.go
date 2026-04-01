package mihomo

import (
	"context"
	"os/exec"
)

// DaemonManager 守护进程管理器接口
type DaemonManager interface {
	// StartAsDaemon 以守护进程方式启动 Mihomo 内核
	StartAsDaemon(ctx context.Context, cfg interface{}) error

	// StopDaemon 停止守护进程
	StopDaemon(pid int) error

	// IsDaemonRunning 检查守护进程是否运行
	IsDaemonRunning(pid int) bool

	// GetDaemonPID 获取守护进程 PID
	GetDaemonPID() (int, error)

	// RedirectIO 重定向标准输入输出
	RedirectIO(cmd *exec.Cmd, logFile string) error

	// CreateProcessGroup 创建进程组
	CreateProcessGroup(cmd *exec.Cmd) error
}

// DaemonConfig 守护进程配置
type DaemonConfig struct {
	Enabled       bool   `toml:"enabled" json:"enabled"`
	WorkDir       string `toml:"work_dir" json:"work_dir"`
	LogFile       string `toml:"log_file" json:"log_file"`
	LogLevel      string `toml:"log_level" json:"log_level"`
	LogMaxSize    string `toml:"log_max_size" json:"log_max_size"`
	LogMaxBackups int    `toml:"log_max_backups" json:"log_max_backups"`
	LogMaxAge     int    `toml:"log_max_age" json:"log_max_age"`

	AutoRestart AutoRestartConfig `toml:"auto_restart" json:"auto_restart"`
	HealthCheck HealthCheckConfig `toml:"health_check" json:"health_check"`
}

// AutoRestartConfig 自动重启配置
type AutoRestartConfig struct {
	Enabled      bool   `toml:"enabled" json:"enabled"`
	MaxRestarts  int    `toml:"max_restarts" json:"max_restarts"`
	RestartDelay string `toml:"restart_delay" json:"restart_delay"`
}

// HealthCheckConfig 健康检查配置
type HealthCheckConfig struct {
	Enabled  bool   `toml:"enabled" json:"enabled"`
	Interval string `toml:"interval" json:"interval"`
	Timeout  string `toml:"timeout" json:"timeout"`
}

// DaemonManagerBase 守护进程管理器基类
type DaemonManagerBase struct {
	config    *DaemonConfig
	pidFile   string
	secret    string
	apiAddr   string
	execPath  string
	configFile string
}

// NewDaemonManagerBase 创建守护进程管理器基类
func NewDaemonManagerBase(
	config *DaemonConfig,
	pidFile, secret, apiAddr, execPath, configFile string,
) *DaemonManagerBase {
	return &DaemonManagerBase{
		config:     config,
		pidFile:    pidFile,
		secret:     secret,
		apiAddr:    apiAddr,
		execPath:   execPath,
		configFile: configFile,
	}
}

// GetConfig 获取配置
func (dmb *DaemonManagerBase) GetConfig() *DaemonConfig {
	return dmb.config
}

// GetPIDFile 获取 PID 文件路径
func (dmb *DaemonManagerBase) GetPIDFile() string {
	return dmb.pidFile
}

// GetSecret 获取密钥
func (dmb *DaemonManagerBase) GetSecret() string {
	return dmb.secret
}

// GetAPIAddress 获取 API 地址
func (dmb *DaemonManagerBase) GetAPIAddress() string {
	return dmb.apiAddr
}

// GetExecutablePath 获取可执行文件路径
func (dmb *DaemonManagerBase) GetExecutablePath() string {
	return dmb.execPath
}

// GetConfigFile 获取配置文件路径
func (dmb *DaemonManagerBase) GetConfigFile() string {
	return dmb.configFile
}

// GetWorkDir 获取工作目录
func (dmb *DaemonManagerBase) GetWorkDir() string {
	if dmb.config != nil && dmb.config.WorkDir != "" {
		return dmb.config.WorkDir
	}
	return ""
}
