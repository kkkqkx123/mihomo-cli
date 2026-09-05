package api

import (
	"context"

	"github.com/kkkqkx123/mihomo-cli/pkg/types"
)

// GetRules 获取所有规则
func (c *Client) GetRules(ctx context.Context) (*types.RulesResponse, error) {
	var result types.RulesResponse
	err := c.Get(ctx, "/rules", nil, &result)
	if err != nil {
		// 直接透传 HTTP 层 APIError，避免双重包装导致错误码丢失
		return nil, err
	}
	return &result, nil
}

// DisableRules 禁用指定规则
// 核心仅提供 PATCH /rules/disable 端点，请求体为 map[int]bool（key 为规则索引，value 为禁用状态）
func (c *Client) DisableRules(ctx context.Context, ruleIDs []int) error {
	if len(ruleIDs) == 0 {
		return NewAPIError(ErrInvalidArgs, "规则索引列表不能为空", nil)
	}

	payload := make(map[int]bool, len(ruleIDs))
	for _, id := range ruleIDs {
		payload[id] = true
	}

	err := c.Patch(ctx, "/rules/disable", nil, payload, nil)
	if err != nil {
		return err
	}

	return nil
}

// EnableRules 启用指定规则
// 核心仅提供 PATCH /rules/disable 端点，value 为 false 时表示启用
func (c *Client) EnableRules(ctx context.Context, ruleIDs []int) error {
	if len(ruleIDs) == 0 {
		return NewAPIError(ErrInvalidArgs, "规则索引列表不能为空", nil)
	}

	payload := make(map[int]bool, len(ruleIDs))
	for _, id := range ruleIDs {
		payload[id] = false
	}

	err := c.Patch(ctx, "/rules/disable", nil, payload, nil)
	if err != nil {
		return err
	}

	return nil
}
