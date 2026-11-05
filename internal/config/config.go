package config

import (
	"net"
	"net/url"
	"strconv"
	"strings"

	"github.com/kkkqkx123/mihomo-cli/internal/output"
	"github.com/kkkqkx123/mihomo-cli/pkg/errors"
)

// CLIConfig CLI 工具配置
type CLIConfig struct {
	API    APIConfig           `mapstructure:"api" yaml:"api"`
	Proxy  ProxyConfig         `mapstructure:"proxy" yaml:"proxy"`
	Output output.OutputConfig `mapstructure:"output" yaml:"output"` // 输出配置
}

// APIConfig API 连接配置
type APIConfig struct {
	Address string `mapstructure:"address" yaml:"address"` // API 地址，如 http://127.0.0.1:9090
	Secret  string `mapstructure:"secret" yaml:"secret"`   // API 密钥
	Timeout int    `mapstructure:"timeout" yaml:"timeout"` // 请求超时（秒）
}

// ProxyConfig 代理测试配置
type ProxyConfig struct {
	TestURL    string `mapstructure:"test_url" yaml:"test_url"`     // 测试 URL
	Timeout    int    `mapstructure:"timeout" yaml:"timeout"`       // 测速超时时间（毫秒）
	Concurrent int    `mapstructure:"concurrent" yaml:"concurrent"` // 并发测试数
	BatchSize  int    `mapstructure:"batch_size" yaml:"batch_size"` // 每批次测试的节点数
	MaxNodes   int    `mapstructure:"max_nodes" yaml:"max_nodes"`   // 最大测试节点数
}

// APIHostPort 返回 API 地址的 host:port 形式（用于内核控制接口）。
func (c *TomlConfig) APIHostPort() string {
	parsedURL, err := url.Parse(c.API.Address)
	if err == nil && parsedURL.Host != "" {
		return parsedURL.Host
	}
	return strings.TrimPrefix(strings.TrimPrefix(c.API.Address, "https://"), "http://")
}

// Validate 验证配置
func (c *CLIConfig) Validate() error {
	if err := c.API.Validate(); err != nil {
		return errors.WrapError("API config validation failed", err)
	}
	if err := c.Proxy.Validate(); err != nil {
		return errors.WrapError("Proxy config validation failed", err)
	}
	if err := c.Output.Validate(); err != nil {
		return errors.WrapError("Output config validation failed", err)
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

// Validate 验证代理配置
func (p *ProxyConfig) Validate() error {
	// 超时时间范围：1秒到60秒
	if p.Timeout != 0 && (p.Timeout < 1000 || p.Timeout > 60000) {
		return errors.ErrConfig("proxy timeout must be between 1000 and 60000 milliseconds", nil)
	}

	// 并发数范围：1到100
	if p.Concurrent != 0 && (p.Concurrent < 1 || p.Concurrent > 100) {
		return errors.ErrConfig("proxy concurrent must be between 1 and 100", nil)
	}

	// 每批次节点数范围：1到10000，零值表示使用默认值
	if p.BatchSize != 0 && (p.BatchSize < 1 || p.BatchSize > 10000) {
		return errors.ErrConfig("proxy batch_size must be between 1 and 10000", nil)
	}

	// 最大节点数范围：1到10000，零值表示使用默认值
	if p.MaxNodes != 0 && (p.MaxNodes < 1 || p.MaxNodes > 10000) {
		return errors.ErrConfig("proxy max_nodes must be between 1 and 10000", nil)
	}

	// 测试地址非空时要求为 HTTP 地址
	if p.TestURL != "" && !strings.HasPrefix(p.TestURL, "http://") && !strings.HasPrefix(p.TestURL, "https://") {
		return errors.ErrConfig("proxy test_url must start with http:// or https://", nil)
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
		Proxy: ProxyConfig{
			TestURL:    "",
			Timeout:    10000, // 默认 10 秒
			Concurrent: 10,
			BatchSize:  100, // 默认每批次 100 个节点
			MaxNodes:   500, // 默认最多测试 500 个节点
		},
		Output: output.OutputConfig{
			File:   "",
			Mode:   "console",
			Append: false,
		},
	}
}
