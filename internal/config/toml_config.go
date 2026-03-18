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
