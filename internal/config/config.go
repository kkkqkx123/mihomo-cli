package config

import (
	"net"
	"net/url"
	"strconv"
	"strings"

	"github.com/kkkqkx123/mihomo-cli/pkg/errors"
)

// CLIConfig CLI 工具配置
type CLIConfig struct {
	API APIConfig `mapstructure:"api"`
}

// APIConfig API 连接配置
type APIConfig struct {
	Address string `mapstructure:"address"` // API 地址，如 http://127.0.0.1:9090
	Secret  string `mapstructure:"secret"`  // API 密钥
	Timeout int    `mapstructure:"timeout"` // 请求超时（秒）
}

// Validate 验证配置
func (c *CLIConfig) Validate() error {
	if err := c.API.Validate(); err != nil {
		return errors.WrapError("API config validation failed", err)
	}
	return nil
}

// Validate 验证 API 配置
func (a *APIConfig) Validate() error {
	// 验证 API 地址
	if a.Address == "" {
		return errors.ErrConfig("API address is required", nil)
	}

	// 验证 URL 格式
	if !strings.HasPrefix(a.Address, "http://") && !strings.HasPrefix(a.Address, "https://") {
		return errors.ErrConfig("API address must start with http:// or https://", nil)
	}

	parsedURL, err := url.Parse(a.Address)
	if err != nil {
		return errors.ErrConfig("invalid API address format", err)
	}

	if parsedURL.Host == "" {
		return errors.ErrConfig("API address must contain a host", nil)
	}

	// 验证端口
	_, port, err := net.SplitHostPort(parsedURL.Host)
	if err == nil {
		portNum, err := strconv.Atoi(port)
		if err != nil {
			return errors.ErrConfig("invalid port number", err)
		}
		if portNum < 1 || portNum > 65535 {
			return errors.ErrConfig("port must be between 1 and 65535", nil)
		}
	}

	// 验证超时
	if a.Timeout < 1 || a.Timeout > 300 {
		return errors.ErrConfig("timeout must be between 1 and 300 seconds", nil)
	}

	return nil
}

// GetDefaultConfig 获取默认配置
func GetDefaultConfig() *CLIConfig {
	return &CLIConfig{
		API: APIConfig{
			Address: "http://127.0.0.1:9090",
			Secret:  "",
			Timeout: 10,
		},
	}
}