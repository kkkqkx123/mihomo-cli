package systemdns

import "errors"

// ErrPlatformNotSupported indicates that system DNS query is not supported.
var ErrPlatformNotSupported = errors.New("system DNS config query not supported on this platform")

// SystemDNSConfig holds the system DNS resolver configuration.
type SystemDNSConfig struct {
	// SearchDomains 搜索域列表（resolv.conf 的 search/domain 指令）
	SearchDomains []string `json:"search_domains,omitempty"`
	// Nameservers DNS 服务器列表（按系统报告顺序去重）
	Nameservers []string `json:"nameservers"`
	// Options 解析器选项（如 timeout、attempts、rotate）
	Options []string `json:"options,omitempty"`
	// Source 配置来源描述（如 /etc/resolv.conf、scutil --dns）
	Source string `json:"source"`
}

// SystemDNS defines the interface for querying system DNS configuration.
type SystemDNS interface {
	// GetConfig 获取系统 DNS 配置
	GetConfig() (*SystemDNSConfig, error)

	// IsSupported 当前平台是否支持
	IsSupported() bool
}
