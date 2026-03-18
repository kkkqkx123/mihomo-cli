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

// RuleDetail 规则详细信息（带索引）
type RuleDetail struct {
	Index   int    `json:"index"`
	Type    string `json:"type"`
	Payload string `json:"payload"`
	Proxy   string `json:"proxy"`
	Size    int    `json:"size"`
	Enabled bool   `json:"enabled"`
}

// DisableRulesRequest 禁用规则请求
type DisableRulesRequest struct {
	RuleIDs []int `json:"rule_ids"`
}

// EnableRulesRequest 启用规则请求
type EnableRulesRequest struct {
	RuleIDs []int `json:"rule_ids"`
}
