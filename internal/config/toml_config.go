package config

import (
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"

	pkgerrors "github.com/kkkqkx123/mihomo-cli/pkg/errors"
)

// TomlConfig 项目 TOML 配置
type TomlConfig struct {
	API    APIConfig     `toml:"api"`
	Mihomo MihomoConfig  `toml:"mihomo"`
	Daemon *DaemonConfig `toml:"daemon"`
}

// MihomoConfig Mihomo 内核配置
type MihomoConfig struct {
	Enabled            bool   `toml:"enabled"`
	Executable         string `toml:"executable"`
	ExecutableSearch   *bool  `toml:"executable_search"` // nil=默认 true：允许按查找链搜索内核；false=只使用显式路径
	ConfigFile         string `toml:"config_file"`
	HealthCheckTimeout int    `toml:"health_check_timeout"`
}

// DaemonConfig 守护进程配置
type DaemonConfig struct {
	Enabled       bool              `toml:"enabled"`
	WorkDir       string            `toml:"work_dir"`
	LogFile       string            `toml:"log_file"`
	LogLevel      string            `toml:"log_level"`
	LogMaxSize    string            `toml:"log_max_size"`
	LogMaxBackups int               `toml:"log_max_backups"`
	LogMaxAge     int               `toml:"log_max_age"`
	AutoRestart   AutoRestartConfig `toml:"auto_restart"`
	HealthCheck   HealthCheckConfig `toml:"health_check"`
}

// AutoRestartConfig 自动重启配置
type AutoRestartConfig struct {
	Enabled      bool   `toml:"enabled"`
	MaxRestarts  int    `toml:"max_restarts"`
	RestartDelay string `toml:"restart_delay"`
}

// HealthCheckConfig 健康检查配置
type HealthCheckConfig struct {
	Enabled  bool   `toml:"enabled"`
	Interval string `toml:"interval"`
	Timeout  string `toml:"timeout"`
}

// Validate 验证 TomlConfig 配置
func (c *TomlConfig) Validate() error {
	if err := c.API.Validate(); err != nil {
		return pkgerrors.WrapError("API config validation failed", err)
	}
	if err := c.Mihomo.Validate(); err != nil {
		return pkgerrors.WrapError("Mihomo config validation failed", err)
	}
	return nil
}

// Validate 验证 MihomoConfig 配置
func (m *MihomoConfig) Validate() error {
	// 验证健康检查超时
	if m.HealthCheckTimeout < 1 || m.HealthCheckTimeout > 60 {
		return pkgerrors.ErrConfig("health_check_timeout must be between 1 and 60 seconds", nil)
	}

	return nil
}

// FindTomlConfigPath 查找 TOML 配置文件路径
// 仅支持显式路径与用户配置目录，不探测当前目录
func FindTomlConfigPath(customPath string) (string, error) {
	if customPath != "" {
		abs, err := filepath.Abs(customPath)
		if err != nil {
			return "", pkgerrors.ErrConfig("invalid toml config path: "+customPath, err)
		}
		if _, err := os.Stat(abs); err != nil {
			return "", pkgerrors.ErrConfig("toml config file not found: "+abs, err)
		}
		return abs, nil
	}

	baseDir, err := GetBaseDir()
	if err != nil {
		return "", pkgerrors.ErrConfig("failed to get user config directory", err)
	}
	userConfig := filepath.Join(baseDir, "config.toml")
	if _, err := os.Stat(userConfig); err != nil {
		return "", pkgerrors.ErrConfig("toml config file not found: "+userConfig+" (specify with --toml)", err)
	}
	return userConfig, nil
}

// ResolveTomlRelativePath 将 TOML 文件内的相对路径以 TOML 所在目录为基准解析。
// 绝对路径直接清理后返回；TOML 路径为空时回退到相对当前目录解析。
// 调用方在读取 TOML 引用的内核文件前必须经此函数归一化，避免含义随工作目录漂移。
func ResolveTomlRelativePath(tomlPath, p string) string {
	if p == "" {
		return ""
	}
	if filepath.IsAbs(p) {
		return filepath.Clean(p)
	}
	if tomlPath != "" {
		if dir := filepath.Dir(tomlPath); dir != "" {
			return filepath.Clean(filepath.Join(dir, p))
		}
	}
	abs, err := filepath.Abs(p)
	if err != nil {
		return p
	}
	return abs
}

// NormalizeTomlConfigPaths 将 TOML 内相对引用的内核配置文件改写为绝对路径。
// 相对基准为 TOML 文件所在目录，与内核可执行文件解析器的基准保持一致。
func NormalizeTomlConfigPaths(tomlPath string, cfg *TomlConfig) {
	if cfg == nil {
		return
	}
	if cfg.Mihomo.ConfigFile != "" {
		cfg.Mihomo.ConfigFile = ResolveTomlRelativePath(tomlPath, cfg.Mihomo.ConfigFile)
	}
}

// LoadTomlConfig 加载 TOML 配置文件，路径必须显式存在
func LoadTomlConfig(path string) (*TomlConfig, error) {
	var config TomlConfig

	if path == "" {
		return nil, pkgerrors.ErrConfig("toml config path is required (specify with --toml)", nil)
	}

	if _, err := os.Stat(path); err != nil {
		return nil, pkgerrors.ErrConfig("toml config file not found: "+path, err)
	}

	// 读取配置文件
	if _, err := toml.DecodeFile(path, &config); err != nil {
		return nil, pkgerrors.ErrConfig("failed to decode TOML config", err)
	}

	// 验证配置
	if err := config.Validate(); err != nil {
		return nil, err
	}

	return &config, nil
}

// GetDefaultTomlConfig 获取默认 TOML 配置
func GetDefaultTomlConfig() *TomlConfig {
	return &TomlConfig{
		API: APIConfig{
			Address: "http://127.0.0.1:9090",
			Secret:  "",
			Timeout: 10,
		},
		Mihomo: MihomoConfig{
			Enabled:            true,
			Executable:         "",
			ConfigFile:         "",
			HealthCheckTimeout: 5,
		},
		Daemon: &DaemonConfig{
			Enabled:       true,
			WorkDir:       "",
			LogFile:       "",
			LogLevel:      "info",
			LogMaxSize:    "100M",
			LogMaxBackups: 10,
			LogMaxAge:     30,
			AutoRestart: AutoRestartConfig{
				Enabled:      false,
				MaxRestarts:  5,
				RestartDelay: "5s",
			},
			HealthCheck: HealthCheckConfig{
				Enabled:  false,
				Interval: "30s",
				Timeout:  "10s",
			},
		},
	}
}

// Save 保存配置到文件
func (c *TomlConfig) Save(path string) error {
	// 确保目录存在
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return pkgerrors.ErrService("failed to create config directory", err)
	}

	// 写入文件
	f, err := os.Create(path)
	if err != nil {
		return pkgerrors.ErrService("failed to create config file", err)
	}
	defer f.Close()

	encoder := toml.NewEncoder(f)
	return encoder.Encode(c)
}
