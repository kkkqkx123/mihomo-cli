package api

import (
	"context"

	"github.com/kkkqkx123/mihomo-cli/pkg/types"
)

// GetTraffic 获取流量统计信息
func (c *Client) GetTraffic(ctx context.Context) (*types.TrafficInfo, error) {
	var result types.TrafficInfo
	err := c.Get(ctx, "/traffic", nil, &result)
	if err != nil {
		return nil, NewAPIError(ErrAPIError, "获取流量统计失败", err)
	}
	return &result, nil
}

// GetMemory 获取内存使用信息
func (c *Client) GetMemory(ctx context.Context) (*types.MemoryInfo, error) {
	var result types.MemoryInfo
	err := c.Get(ctx, "/memory", nil, &result)
	if err != nil {
		return nil, NewAPIError(ErrAPIError, "获取内存使用失败", err)
	}
	return &result, nil
}
