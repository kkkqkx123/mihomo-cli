package api

import (
	"context"
	"net/url"
	"strconv"

	"github.com/kkkqkx123/mihomo-cli/pkg/types"
)

// ListProxies 获取所有代理信息
func (c *Client) ListProxies(ctx context.Context) (map[string]*types.ProxyInfo, error) {
	var result types.ProxiesResponse
	err := c.Get(ctx, "/proxies", nil, &result)
	if err != nil {
		// 直接透传 HTTP 层 APIError，避免双重包装导致错误码丢失
		return nil, err
	}

	// 根据 history 填充 Delay 字段
	for _, proxy := range result.Proxies {
		if proxy != nil && len(proxy.History) > 0 {
			// 使用最后一条记录的 delay 值
			lastHistory := proxy.History[len(proxy.History)-1]
			if lastHistory.Delay > 0 {
				proxy.Delay = uint16(lastHistory.Delay)
			}
		}
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
		// 直接透传 HTTP 层 APIError，避免双重包装导致错误码丢失
		return nil, err
	}

	// 根据 history 填充 Delay 字段
	if len(result.History) > 0 {
		lastHistory := result.History[len(result.History)-1]
		if lastHistory.Delay > 0 {
			result.Delay = uint16(lastHistory.Delay)
		}
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
		return err
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
	// 始终传递 timeout 参数，避免 mihomo API 因空参数而报错
	// 如果 timeout <= 0，使用默认值 5000ms
	if timeout <= 0 {
		timeout = 5000
	}
	queryParams["timeout"] = strconv.Itoa(timeout)

	var result types.DelayResponse
	err := c.Get(ctx, "/proxies/"+encodedName+"/delay", queryParams, &result)
	if err != nil {
		// 直接透传原始 API 错误，保留 HTTP 状态码，便于上层根据状态精确分类
		return 0, err
	}

	return result.Delay, nil
}

// UnfixProxy 取消代理组中固定的代理（恢复自动选择）
func (c *Client) UnfixProxy(ctx context.Context, group string) error {
	// URL 编码代理组名称
	encodedGroup := url.PathEscape(group)

	err := c.Delete(ctx, "/proxies/"+encodedGroup, nil, nil)
	if err != nil {
		return err
	}

	return nil
}
