package config

import (
	"os"
	"path/filepath"

	"github.com/spf13/viper"
	"gopkg.in/yaml.v3"

	"github.com/kkkqkx123/mihomo-cli/pkg/errors"
)

// Loader 配置加载器
type Loader struct {
}

// NewLoader 创建配置加载器
func NewLoader() *Loader {
	return &Loader{}
}

// Load 从指定路径加载配置
func (l *Loader) Load(configPath string) (*CLIConfig, error) {
	data, err := os.ReadFile(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, errors.ErrConfig("config file not found", err)
		}
		return nil, errors.ErrConfig("failed to read config file", err)
	}

	cfg := &CLIConfig{}
	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, errors.ErrConfig("failed to unmarshal config", err)
	}

	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	return cfg, nil
}

// LoadFromViper 从全局 viper 实例加载配置
func LoadFromViper() (*CLIConfig, error) {
	// 重新读取配置文件以确保获取最新配置
	if err := viper.ReadInConfig(); err != nil {
		// 配置文件不存在不是错误，使用默认配置
		if !os.IsNotExist(err) {
			if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
				return nil, errors.ErrConfig("failed to read config", err)
			}
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
	// 验证配置
	if err := cfg.Validate(); err != nil {
		return err
	}

	// 确保目录存在
	configDir := filepath.Dir(configPath)
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return errors.ErrConfig("failed to create config directory", err)
	}

	data, err := yaml.Marshal(cfg)
	if err != nil {
		return errors.ErrConfig("failed to marshal config", err)
	}

	tmpFile, err := os.CreateTemp(configDir, ".config-*.tmp")
	if err != nil {
		return errors.ErrConfig("failed to create temp config file", err)
	}
	tmpName := tmpFile.Name()
	if _, err := tmpFile.Write(data); err != nil {
		tmpFile.Close()
		os.Remove(tmpName)
		return errors.ErrConfig("failed to write config file", err)
	}
	if err := tmpFile.Close(); err != nil {
		os.Remove(tmpName)
		return errors.ErrConfig("failed to write config file", err)
	}
	if err := os.Chmod(tmpName, 0600); err != nil {
		os.Remove(tmpName)
		return errors.ErrConfig("failed to write config file", err)
	}
	if err := os.Rename(tmpName, configPath); err != nil {
		os.Remove(tmpName)
		return errors.ErrConfig("failed to write config file", err)
	}

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
