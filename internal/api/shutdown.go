package api

import (
	"context"
)

// Shutdown 通过 API 关闭 Mihomo 内核
func (c *Client) Shutdown(ctx context.Context) error {
	var result struct {
		Status string `json:"status"`
	}
	err := c.Post(ctx, "/shutdown", nil, nil, &result)
	if err != nil {
		return NewAPIError(ErrAPIError, "关闭 Mihomo 失败", err)
	}
	return nil
}
