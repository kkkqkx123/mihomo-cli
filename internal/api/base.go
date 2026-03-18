package api

import (
	"context"

	"github.com/kkkqkx123/mihomo-cli/pkg/types"
)

// GetVersion 获取 Mihomo 内核版本信息
func (c *Client) GetVersion(ctx context.Context) (*types.VersionInfo, error) {
	var result types.VersionInfo
	err := c.Get(ctx, "/version", nil, &result)
	if err != nil {
		return nil, NewAPIError(ErrAPIError, "获取版本信息失败", err)
	}
	return &result, nil
}
