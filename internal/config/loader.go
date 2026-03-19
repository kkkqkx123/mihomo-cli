package config

import (
	"os"
	"path/filepath"

	"github.com/kkkqkx123/mihomo-cli/internal/output"
	"github.com/kkkqkx123/mihomo-cli/pkg/errors"
	"github.com/spf13/viper"
)

// Loader 配置加载器
type Loader struct {
	v *viper.Viper
}

// NewLoader 创建配置加载器
func NewLoader() *Loader {
	return &Loader{
		v: viper.New(),
	}
}

// Load 从指定路径加载配置
func (l *Loader) Load(configPath string) (*CLIConfig, error) {
	output.Info("加载配置文件: %s", configPath)

	// 设置配置文件路径
	l.v.SetConfigFile(configPath)

	// 读取配置文件
	if err := l.v.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			return nil, errors.ErrConfig("config file not found", nil)
		}
		return nil, errors.ErrConfig("failed to read config file", err)
	}

	// 解析配置到结构体
	cfg := &CLIConfig{}
	if err := l.v.Unmarshal(cfg); err != nil {
		return nil, errors.ErrConfig("failed to unmarshal config", err)
	}

	// 验证配置
	output.Info("验证配置...")
	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	output.Success("配置加载成功")
	return cfg, nil
}

// LoadFromViper 从全局 viper 实例加载配置
func LoadFromViper() (*CLIConfig, error) {
	// 重新读取配置文件以确保获取最新配置
	if err := viper.ReadInConfig(); err != nil {
		// 配置文件不存在不是错误，使用默认配置
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, errors.ErrConfig("failed to read config", err)
		}
	}

	cfg := &CLIConfig{}
	if err := viper.Unmarshal(cfg); err != nil {
		return nil, errors.ErrConfig("failed to unmarshal config", err)
	}

	// 验证配置
	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	return cfg, nil
}

// Save 保存配置到指定路径
func (l *Loader) Save(cfg *CLIConfig, configPath string) error {
	output.Info("保存配置到: %s", configPath)

	// 验证配置
	if err := cfg.Validate(); err != nil {
		return err
	}

	// 设置 API 配置
	l.v.Set("api.address", cfg.API.Address)
	l.v.Set("api.secret", cfg.API.Secret)
	l.v.Set("api.timeout", cfg.API.Timeout)

	// 设置 Proxy 配置
	l.v.Set("proxy.test_url", cfg.Proxy.TestURL)
	l.v.Set("proxy.timeout", cfg.Proxy.Timeout)
	l.v.Set("proxy.concurrent", cfg.Proxy.Concurrent)

	// 设置 Log 配置
	l.v.Set("log.file", cfg.Log.File)
	l.v.Set("log.mode", cfg.Log.Mode)
	l.v.Set("log.append", cfg.Log.Append)

	// 确保目录存在
	configDir := filepath.Dir(configPath)
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return errors.ErrConfig("failed to create config directory", err)
	}

	// 设置输出文件
	l.v.SetConfigFile(configPath)

	// 写入配置文件
	if err := l.v.WriteConfig(); err != nil {
		return errors.ErrConfig("failed to write config file", err)
	}

	output.Success("配置保存成功")
	return nil
}

// GetDefaultConfigPath 获取默认配置文件路径
func GetDefaultConfigPath() (string, error) {
	paths, err := GetPaths()
	if err != nil {
		return "", err
	}
	return paths.ConfigFile, nil
}