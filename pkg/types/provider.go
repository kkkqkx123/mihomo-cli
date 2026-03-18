package types

// ProviderInfo 代理提供者信息
type ProviderInfo struct {
	Name        string                `json:"name"`
	Type        string                `json:"type"`
	VehicleType string                `json:"vehicleType"`
	Proxies     []ProviderProxyInfo   `json:"proxies"`
	UpdatedAt   string                `json:"updatedAt"`
}

// ProviderProxyInfo 提供者中的代理信息
type ProviderProxyInfo struct {
	Name    string         `json:"name"`
	Type    string         `json:"type"`
	UDP     bool           `json:"udp"`
	XUDP    bool           `json:"xudp"`
	History []DelayHistory `json:"history"`
	Alive   bool           `json:"alive"`
}

// ProvidersResponse 代理提供者列表响应
type ProvidersResponse struct {
	Providers map[string]*ProviderInfo `json:"providers"`
}
