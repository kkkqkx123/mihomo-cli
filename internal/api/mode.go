package api

import (
	"context"
	"fmt"

	"github.com/kkkqkx123/mihomo-cli/pkg/types"
)

// GetMode 获取当前运行模式
func (c *Client) GetMode(ctx context.Context) (*types.ModeInfo, error) {
	var result types.ModeInfo
	err := c.Get(ctx, "/configs", nil, &result)
	if err != nil {
		return nil, fmt.Errorf("获取模式失败: %w", err)
	}
	return &result, nil
}

// SetMode 设置运行模式
func (c *Client) SetMode(ctx context.Context, mode types.TunnelMode) error {
	// 验证模式
	if !types.IsValidMode(string(mode)) {
		return fmt.Errorf("无效的模式: %s, 有效选项: %v", mode, types.ValidModes)
	}

	// 使用 PATCH /configs 更新模式
	patchData := map[string]interface{}{
		"mode": mode,
	}

	err := c.Patch(ctx, "/configs", nil, patchData, nil)
	if err != nil {
		return fmt.Errorf("设置模式失败: %w", err)
	}

	return nil
}