package api

import (
	"context"

	"github.com/kkkqkx123/mihomo-cli/pkg/types"
)

// GetConnections 获取所有活跃连接
func (c *Client) GetConnections(ctx context.Context) (*types.ConnectionsResponse, error) {
	var result types.ConnectionsResponse
	err := c.Get(ctx, "/connections", nil, &result)
	if err != nil {
		// 直接透传 HTTP 层 APIError，避免双重包装导致错误码丢失
		return nil, err
	}
	return &result, nil
}

// CloseConnection 关闭指定连接
func (c *Client) CloseConnection(ctx context.Context, id string) error {
	err := c.Delete(ctx, "/connections/"+id, nil, nil)
	if err != nil {
		return err
	}
	return nil
}

// CloseAllConnections 关闭所有连接
func (c *Client) CloseAllConnections(ctx context.Context) error {
	err := c.Delete(ctx, "/connections", nil, nil)
	if err != nil {
		return err
	}
	return nil
}
