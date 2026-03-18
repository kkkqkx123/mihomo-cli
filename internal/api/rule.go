package api

import (
	"context"
	"fmt"

	"github.com/kkkqkx123/mihomo-cli/pkg/types"
)

// GetRules 获取所有规则
func (c *Client) GetRules(ctx context.Context) (*types.RulesResponse, error) {
	var result types.RulesResponse
	err := c.Get(ctx, "/rules", nil, &result)
	if err != nil {
		return nil, fmt.Errorf("获取规则列表失败: %w", err)
	}
	return &result, nil
}

// DisableRules 禁用指定规则
func (c *Client) DisableRules(ctx context.Context, ruleIDs []int) error {
	if len(ruleIDs) == 0 {
		return fmt.Errorf("规则索引列表不能为空")
	}

	request := types.DisableRulesRequest{
		RuleIDs: ruleIDs,
	}

	err := c.Patch(ctx, "/rules/disable", nil, &request, nil)
	if err != nil {
		return fmt.Errorf("禁用规则失败: %w", err)
	}

	return nil
}

// EnableRules 启用指定规则
func (c *Client) EnableRules(ctx context.Context, ruleIDs []int) error {
	if len(ruleIDs) == 0 {
		return fmt.Errorf("规则索引列表不能为空")
	}

	request := types.EnableRulesRequest{
		RuleIDs: ruleIDs,
	}

	err := c.Patch(ctx, "/rules/enable", nil, &request, nil)
	if err != nil {
		return fmt.Errorf("启用规则失败: %w", err)
	}

	return nil
}
