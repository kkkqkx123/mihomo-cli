package api

import (
	"context"
	"fmt"

	"github.com/kkkqkx123/mihomo-cli/pkg/types"
)

// GetConnections 获取所有活跃连接
func (c *Client) GetConnections(ctx context.Context) (*types.ConnectionsResponse, error) {
	var result types.ConnectionsResponse
	err := c.Get(ctx, "/connections", nil, &result)
	if err != nil {
		return nil, NewAPIError(ErrAPIError, "获取连接列表失败", err)
	}
	return &result, nil
}

// CloseConnection 关闭指定连接
func (c *Client) CloseConnection(ctx context.Context, id string) error {
	err := c.Delete(ctx, "/connections/"+id, nil, nil)
	if err != nil {
		return NewAPIError(ErrNotFound, fmt.Sprintf("关闭连接 %s 失败", id), err)
	}
	return nil
}

// CloseAllConnections 关闭所有连接
func (c *Client) CloseAllConnections(ctx context.Context) error {
	err := c.Delete(ctx, "/connections", nil, nil)
	if err != nil {
		return NewAPIError(ErrAPIError, "关闭所有连接失败", err)
	}
	return nil
}
