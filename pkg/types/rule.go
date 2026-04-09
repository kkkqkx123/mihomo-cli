package types

import "time"

// RuleExtra 规则额外信息（命中统计）
type RuleExtra struct {
	Disabled  bool      `json:"disabled"`
	HitCount  uint64    `json:"hitCount"`
	HitAt     time.Time `json:"hitAt"`
	MissCount uint64    `json:"missCount"`
	MissAt    time.Time `json:"missAt"`
}

// RuleInfo 规则信息
type RuleInfo struct {
	Index   int        `json:"index"`
	Type    string     `json:"type"`
	Payload string     `json:"payload"`
	Proxy   string     `json:"proxy"`
	Size    int        `json:"size"`
	Extra   *RuleExtra `json:"extra,omitempty"`
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
	Behavior    string   `json:"behavior"`
	Format      string   `json:"format"`
	VehicleType string   `json:"vehicleType"`
	RuleCount   int      `json:"ruleCount"`
	UpdatedAt   string   `json:"updatedAt"`
	Payload     []string `json:"payload,omitempty"`
}

// RuleProvidersResponse 规则提供者列表响应
type RuleProvidersResponse struct {
	Providers map[string]*RuleProviderInfo `json:"providers"`
}
