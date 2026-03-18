package api

import (
	"context"
	"fmt"
	"net/url"
	"strconv"

	"github.com/kkkqkx123/mihomo-cli/pkg/types"
)

// ListProxies 获取所有代理信息
func (c *Client) ListProxies(ctx context.Context) (map[string]*types.ProxyInfo, error) {
	var result types.ProxiesResponse
	err := c.Get(ctx, "/proxies", nil, &result)
	if err != nil {
		return nil, fmt.Errorf("获取代理列表失败: %w", err)
	}
	return result.Proxies, nil
}

// GetProxy 获取指定代理的详细信息
func (c *Client) GetProxy(ctx context.Context, name string) (*types.ProxyInfo, error) {
	// URL 编码代理名称
	encodedName := url.PathEscape(name)

	var result types.ProxyInfo
	err := c.Get(ctx, "/proxies/"+encodedName, nil, &result)
	if err != nil {
		return nil, fmt.Errorf("获取代理 %s 失败: %w", name, err)
	}
	return &result, nil
}

// SwitchProxy 切换代理组中选中的代理
func (c *Client) SwitchProxy(ctx context.Context, group, proxy string) error {
	// URL 编码代理组名称
	encodedGroup := url.PathEscape(group)

	request := types.SwitchRequest{
		Name: proxy,
	}

	err := c.Put(ctx, "/proxies/"+encodedGroup, nil, &request, nil)
	if err != nil {
		return fmt.Errorf("切换代理失败: %w", err)
	}

	return nil
}

// TestDelay 测试指定代理的延迟
func (c *Client) TestDelay(ctx context.Context, name string, testURL string, timeout int) (uint16, error) {
	// URL 编码代理名称
	encodedName := url.PathEscape(name)

	queryParams := make(map[string]string)
	if testURL != "" {
		queryParams["url"] = testURL
	}
	if timeout > 0 {
		queryParams["timeout"] = strconv.Itoa(timeout)
	}

	var result types.DelayResponse
	err := c.Get(ctx, "/proxies/"+encodedName+"/delay", queryParams, &result)
	if err != nil {
		return 0, fmt.Errorf("测试延迟失败: %w", err)
	}

	return result.Delay, nil
}

// UnfixProxy 取消代理组中固定的代理（恢复自动选择）
func (c *Client) UnfixProxy(ctx context.Context, group string) error {
	// URL 编码代理组名称
	encodedGroup := url.PathEscape(group)

	err := c.Delete(ctx, "/proxies/"+encodedGroup, nil, nil)
	if err != nil {
		return fmt.Errorf("取消固定代理失败: %w", err)
	}

	return nil
}