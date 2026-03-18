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
		return nil, NewAPIError(ErrAPIError, "获取规则列表失败", err)
	}
	return &result, nil
}

// DisableRules 禁用指定规则
func (c *Client) DisableRules(ctx context.Context, ruleIDs []int) error {
	if len(ruleIDs) == 0 {
		return NewAPIError(ErrInvalidArgs, "规则索引列表不能为空", nil)
	}

	request := types.DisableRulesRequest{
		RuleIDs: ruleIDs,
	}

	err := c.Patch(ctx, "/rules/disable", nil, &request, nil)
	if err != nil {
		return NewAPIError(ErrAPIError, "禁用规则失败", err)
	}

	return nil
}

// EnableRules 启用指定规则
func (c *Client) EnableRules(ctx context.Context, ruleIDs []int) error {
	if len(ruleIDs) == 0 {
		return NewAPIError(ErrInvalidArgs, "规则索引列表不能为空", nil)
	}

	request := types.EnableRulesRequest{
		RuleIDs: ruleIDs,
	}

	err := c.Patch(ctx, "/rules/enable", nil, &request, nil)
	if err != nil {
		return NewAPIError(ErrAPIError, "启用规则失败", err)
	}

	return nil
}
