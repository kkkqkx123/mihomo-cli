package config

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"

	pkgerrors "github.com/kkkqkx123/mihomo-cli/pkg/errors"
)

// TomlConfig 项目 TOML 配置
type TomlConfig struct {
	API    APIConfig    `toml:"api"`
	Mihomo MihomoConfig `toml:"mihomo"`
}

// MihomoConfig Mihomo 内核配置
type MihomoConfig struct {
	Enabled                bool             `toml:"enabled"`
	Executable             string           `toml:"executable"`
	ConfigFile             string           `toml:"config_file"`
	AutoGenerateSecret     bool             `toml:"auto_generate_secret"`
	HealthCheckTimeout     int              `toml:"health_check_timeout"`
	API                    MihomoAPIConfig  `toml:"api"`
	Log                    MihomoLogConfig  `toml:"log"`
}

// MihomoAPIConfig Mihomo API 配置
type MihomoAPIConfig struct {
	ExternalController     string `toml:"external_controller"`
	ExternalControllerTLS  string `toml:"external_controller_tls"`
	ExternalControllerUnix string `toml:"external_controller_unix"`
	ExternalControllerPipe string `toml:"external_controller_pipe"`
}

// MihomoLogConfig Mihomo 日志配置
type MihomoLogConfig struct {
	Level string `toml:"level"`
	File  string `toml:"file"`
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

	// 验证日志级别
	if m.Log.Level != "" {
		validLevels := map[string]bool{
			"debug": true, "info": true, "warning": true, "error": true, "silent": true,
		}
		if !validLevels[m.Log.Level] {
			return pkgerrors.ErrConfig("log level must be one of: debug, info, warning, error, silent", nil)
		}
	}

	return nil
}

// GenerateRandomSecret 生成随机 SHA256 密钥
func GenerateRandomSecret() (string, error) {
	// 生成 32 字节随机数
	randomBytes := make([]byte, 32)
	if _, err := rand.Read(randomBytes); err != nil {
		return "", pkgerrors.ErrService("failed to generate random bytes", err)
	}

	// 计算 SHA256
	hash := sha256.Sum256(randomBytes)
	return hex.EncodeToString(hash[:]), nil
}

// FindTomlConfigPath 查找 TOML 配置文件路径
// 优先级：1. 指定路径 2. 当前目录 3. 用户配置目录
func FindTomlConfigPath(customPath string) string {
	// 1. 如果指定了路径，直接使用
	if customPath != "" {
		if _, err := os.Stat(customPath); err == nil {
			return customPath
		}
	}

	// 2. 当前目录的 config.toml
	currentDirConfig := "config.toml"
	if _, err := os.Stat(currentDirConfig); err == nil {
		return currentDirConfig
	}

	// 3. 用户配置目录
	home, err := os.UserHomeDir()
	if err == nil {
		userConfig := filepath.Join(home, ".config", ".mihomo-cli", "config.toml")
		if _, err := os.Stat(userConfig); err == nil {
			return userConfig
		}
	}

	// 默认返回当前目录路径（即使不存在，LoadTomlConfig 会返回默认配置）
	return currentDirConfig
}

// LoadTomlConfig 加载 TOML 配置文件
func LoadTomlConfig(path string) (*TomlConfig, error) {
	var config TomlConfig

	// 如果文件不存在，返回默认配置
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return GetDefaultTomlConfig(), nil
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
			Enabled:                true,
			Executable:             "mihomo.exe",
			ConfigFile:             "",
			AutoGenerateSecret:     true,
			HealthCheckTimeout:     5,
			API: MihomoAPIConfig{
				ExternalController: "127.0.0.1:9090",
			},
			Log: MihomoLogConfig{
				Level: "info",
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
