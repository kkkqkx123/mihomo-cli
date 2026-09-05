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
		// 直接透传 HTTP 层 APIError，避免双重包装导致错误码丢失
		return nil, err
	}
	return &result, nil
}

// Restart 通过 API 重启 Mihomo 内核
func (c *Client) Restart(ctx context.Context) error {
	err := c.Post(ctx, "/restart", nil, nil, nil)
	if err != nil {
		return err
	}
	return nil
}
