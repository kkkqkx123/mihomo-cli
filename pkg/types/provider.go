package types

// ProviderInfo 代理提供者信息
type ProviderInfo struct {
	Name        string       `json:"name"`
	Type        string       `json:"type"`
	VehicleType string       `json:"vehicleType"`
	Proxies     []*ProxyInfo `json:"proxies"`
	UpdatedAt   string       `json:"updatedAt"`
}

// ProvidersResponse 代理提供者列表响应
type ProvidersResponse struct {
	Providers map[string]*ProviderInfo `json:"providers"`
}
