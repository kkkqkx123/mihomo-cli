package types

// RuleInfo 规则信息
type RuleInfo struct {
	Type    string `json:"type"`
	Payload string `json:"payload"`
	Proxy   string `json:"proxy"`
	Size    int    `json:"size"`
}

// RulesResponse 规则列表响应
type RulesResponse struct {
	Rules []RuleInfo `json:"rules"`
}

// RuleStats 规则统计信息
type RuleStats struct {
	Total   int            `json:"total"`
	ByType  map[string]int `json:"by_type"`
	Enabled int            `json:"enabled"`
	Disabled int           `json:"disabled"`
}

// DisableRulesRequest 禁用规则请求
type DisableRulesRequest struct {
	RuleIDs []int `json:"rule_ids"`
}

// EnableRulesRequest 启用规则请求
type EnableRulesRequest struct {
	RuleIDs []int `json:"rule_ids"`
}

// RuleProviderInfo 规则提供者信息
type RuleProviderInfo struct {
	Name        string   `json:"name"`
	Type        string   `json:"type"`
	VehicleType string   `json:"vehicleType"`
	Rules       []string `json:"rules"`
	UpdatedAt   string   `json:"updatedAt"`
}

// RuleProvidersResponse 规则提供者列表响应
type RuleProvidersResponse struct {
	Providers map[string]*RuleProviderInfo `json:"providers"`
}
