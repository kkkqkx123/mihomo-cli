package api

import (
	"bufio"
	"context"
	"encoding/json"
	"io"

	"github.com/kkkqkx123/mihomo-cli/pkg/types"
)

// GetTraffic 获取流量统计信息（单次查询）
// 注意：mihomo 的 /traffic API 是流式的，会持续推送数据
// 此方法只读取第一条数据然后关闭连接
func (c *Client) GetTraffic(ctx context.Context) (*types.TrafficInfo, error) {
	// 使用流式请求获取数据
	result, err := c.getStreamData(ctx, "/traffic", func() interface{} {
		return &types.TrafficInfo{}
	})
	if err != nil {
		return nil, NewAPIError(ErrAPIError, "获取流量统计失败", err)
	}

	if traffic, ok := result.(*types.TrafficInfo); ok {
		return traffic, nil
	}

	return nil, NewAPIError(ErrAPIError, "获取流量统计失败: 类型断言失败", nil)
}

// GetMemory 获取内存使用信息（单次查询）
// 注意：mihomo 的 /memory API 是流式的，会持续推送数据
// 此方法只读取第一条数据然后关闭连接
func (c *Client) GetMemory(ctx context.Context) (*types.MemoryInfo, error) {
	// 使用流式请求获取数据
	result, err := c.getStreamData(ctx, "/memory", func() interface{} {
		return &types.MemoryInfo{}
	})
	if err != nil {
		return nil, NewAPIError(ErrAPIError, "获取内存使用失败", err)
	}

	if memory, ok := result.(*types.MemoryInfo); ok {
		return memory, nil
	}

	return nil, NewAPIError(ErrAPIError, "获取内存使用失败: 类型断言失败", nil)
}

// getStreamData 从流式 API 获取单条数据
// mihomo 的某些 API（如 /traffic, /memory）会持续推送 JSON 数据
// 此方法只读取第一条数据然后关闭连接
func (c *Client) getStreamData(ctx context.Context, endpoint string, newTarget func() interface{}) (interface{}, error) {
	// 构建完整 URL
	fullURL, err := c.httpClient.buildURL(c.baseURL, endpoint, nil)
	if err != nil {
		return nil, err
	}

	// 创建请求
	req, err := c.httpClient.newRequest(ctx, "GET", fullURL, nil)
	if err != nil {
		return nil, err
	}

	// 执行请求
	resp, err := c.httpClient.doRequestWithReq(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// 检查状态码
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, ParseErrorResponse(resp)
	}

	// 创建 scanner 读取流式数据
	scanner := bufio.NewScanner(resp.Body)
	if !scanner.Scan() {
		if err := scanner.Err(); err != nil {
			return nil, NewAPIError(ErrGeneral, "读取流式数据失败", err)
		}
		return nil, NewAPIError(ErrGeneral, "流式数据为空", io.EOF)
	}

	// 解析第一条数据
	target := newTarget()
	if err := json.Unmarshal(scanner.Bytes(), target); err != nil {
		return nil, NewAPIError(ErrGeneral, "解析流式数据失败", err)
	}

	return target, nil
}
