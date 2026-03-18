package types

// ProxyInfo 代理信息
type ProxyInfo struct {
	Name     string         `json:"name"`
	Type     string         `json:"type"`
	UDP      bool           `json:"udp"`
	XUDP     bool           `json:"xudp"`
	History  []DelayHistory `json:"history"`
	Alive    bool           `json:"alive"`
	Now      string         `json:"now,omitempty"`
	All      []string       `json:"all,omitempty"`
	Provider string         `json:"provider,omitempty"`
	Delay    uint16         `json:"delay"`
}

// DelayHistory 延迟历史
type DelayHistory struct {
	Time  string `json:"time"`
	Delay int    `json:"delay"`
}

// DelayResult 延迟测试结果
type DelayResult struct {
	Name   string
	Delay  uint16
	Error  error
	Status string // 状态描述：优秀/良好/较差/超时/未知
	Time   int64  // 测速耗时（毫秒）
}

// SwitchRequest 切换代理请求
type SwitchRequest struct {
	Name string `json:"name"`
}

// ProxiesResponse 代理列表响应
type ProxiesResponse struct {
	Proxies map[string]*ProxyInfo `json:"proxies"`
}

// DelayResponse 延迟响应
type DelayResponse struct {
	Delay uint16 `json:"delay"`
}