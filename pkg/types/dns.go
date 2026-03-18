package types

import "fmt"

// DNSQueryResponse DNS 查询响应
type DNSQueryResponse struct {
	Status     int           `json:"Status"`
	TC         bool          `json:"TC"`
	RD         bool          `json:"RD"`
	RA         bool          `json:"RA"`
	AD         bool          `json:"AD"`
	CD         bool          `json:"CD"`
	Question   []DNSQuestion `json:"Question"`
	Answer     []DNSAnswer   `json:"Answer,omitempty"`
	Authority  []DNSAnswer   `json:"Authority,omitempty"`
	Additional []DNSAnswer   `json:"Additional,omitempty"`
}

// DNSQuestion DNS 查询问题
type DNSQuestion struct {
	Name  string `json:"name"`
	Type  int    `json:"type"`
	Class int    `json:"class"`
}

// DNSAnswer DNS 回答记录
type DNSAnswer struct {
	Name string `json:"name"`
	Type int    `json:"type"`
	TTL  int    `json:"TTL"`
	Data string `json:"data"`
}

// DNSType DNS 记录类型映射
var DNSType = map[string]int{
	"A":     1,
	"NS":    2,
	"CNAME": 5,
	"SOA":   6,
	"PTR":   12,
	"MX":    15,
	"TXT":   16,
	"AAAA":  28,
	"SRV":   33,
}

// DNSTypeToString 将 DNS 类型值转换为字符串
func DNSTypeToString(typeValue int) string {
	for k, v := range DNSType {
		if v == typeValue {
			return k
		}
	}
	return fmt.Sprintf("TYPE%d", typeValue)
}

// DNSConfig DNS 配置信息
type DNSConfig struct {
	Enable       bool              `json:"enable"`
	IPv6         bool              `json:"ipv6"`
	EnhancedMode string            `json:"enhanced-mode"`
	FakeIPRange  string            `json:"fake-ip-range"`
	FakeIPFilter []string          `json:"fake-ip-filter"`
	Nameserver   []string          `json:"nameserver"`
	Fallback     []string          `json:"fallback"`
	FallbackFilter FallbackFilter  `json:"fallback-filter"`
	Listen       string            `json:"listen"`
	DefaultNameserver []string     `json:"default-nameserver"`
}

// FallbackFilter DNS 回退过滤器配置
type FallbackFilter struct {
	GeoIP     bool     `json:"geoip"`
	GeoIPCode string   `json:"geoip-code"`
	IPCIDR    []string `json:"ipcidr"`
	Domain    []string `json:"domain"`
}
