package api

import (
	"context"
	"net/url"

	"github.com/kkkqkx123/mihomo-cli/pkg/types"
)

// ListProviders 获取所有代理提供者信息
func (c *Client) ListProviders(ctx context.Context) (map[string]*types.ProviderInfo, error) {
	var result types.ProvidersResponse
	err := c.Get(ctx, "/providers/proxies", nil, &result)
	if err != nil {
		// 直接透传 HTTP 层 APIError，避免双重包装导致错误码丢失
		return nil, err
	}
	for _, provider := range result.Providers {
		for _, proxy := range provider.Proxies {
			fillDelayFromHistory(proxy)
		}
	}
	return result.Providers, nil
}

// ListProviderProxyMap 获取全部 provider 节点并按节点名索引
// provider 节点不在 /proxies 注册表中，组内子节点补全与当前节点查询依赖该索引
func (c *Client) ListProviderProxyMap(ctx context.Context) (map[string]*types.ProxyInfo, error) {
	providers, err := c.ListProviders(ctx)
	if err != nil {
		return nil, err
	}

	index := make(map[string]*types.ProxyInfo)
	for _, provider := range providers {
		for _, proxy := range provider.Proxies {
			if proxy == nil {
				continue
			}
			index[proxy.Name] = proxy
		}
	}
	return index, nil
}

// UpdateProvider 更新指定代理提供者的订阅
func (c *Client) UpdateProvider(ctx context.Context, name string) error {
	// URL 编码提供者名称
	encodedName := url.PathEscape(name)

	err := c.Put(ctx, "/providers/proxies/"+encodedName, nil, nil, nil)
	if err != nil {
		return err
	}

	return nil
}

// ListRuleProviders 获取所有规则提供者信息
func (c *Client) ListRuleProviders(ctx context.Context) (map[string]*types.RuleProviderInfo, error) {
	var result types.RuleProvidersResponse
	err := c.Get(ctx, "/providers/rules", nil, &result)
	if err != nil {
		// 直接透传 HTTP 层 APIError，避免双重包装导致错误码丢失
		return nil, err
	}
	return result.Providers, nil
}

// UpdateRuleProvider 更新指定规则提供者的订阅
func (c *Client) UpdateRuleProvider(ctx context.Context, name string) error {
	// URL 编码提供者名称
	encodedName := url.PathEscape(name)

	err := c.Put(ctx, "/providers/rules/"+encodedName, nil, nil, nil)
	if err != nil {
		return err
	}

	return nil
}

// HealthCheckProvider 触发指定代理提供者的健康检查
func (c *Client) HealthCheckProvider(ctx context.Context, name string) error {
	// URL 编码提供者名称
	encodedName := url.PathEscape(name)

	err := c.Get(ctx, "/providers/proxies/"+encodedName+"/healthcheck", nil, nil)
	if err != nil {
		return err
	}

	return nil
}
