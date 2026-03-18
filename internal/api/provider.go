package api

import (
	"context"
	"fmt"
	"net/url"

	"github.com/kkkqkx123/mihomo-cli/pkg/types"
)

// ListProviders 获取所有代理提供者信息
func (c *Client) ListProviders(ctx context.Context) (map[string]*types.ProviderInfo, error) {
	var result types.ProvidersResponse
	err := c.Get(ctx, "/providers/proxies", nil, &result)
	if err != nil {
		return nil, NewAPIError(ErrAPIError, "获取代理提供者列表失败", err)
	}
	return result.Providers, nil
}

// UpdateProvider 更新指定代理提供者的订阅
func (c *Client) UpdateProvider(ctx context.Context, name string) error {
	// URL 编码提供者名称
	encodedName := url.PathEscape(name)

	err := c.Put(ctx, "/providers/proxies/"+encodedName, nil, nil, nil)
	if err != nil {
		return NewAPIError(ErrNotFound, fmt.Sprintf("更新代理提供者 %s 失败", name), err)
	}

	return nil
}

// ListRuleProviders 获取所有规则提供者信息
func (c *Client) ListRuleProviders(ctx context.Context) (map[string]*types.RuleProviderInfo, error) {
	var result types.RuleProvidersResponse
	err := c.Get(ctx, "/providers/rules", nil, &result)
	if err != nil {
		return nil, NewAPIError(ErrAPIError, "获取规则提供者列表失败", err)
	}
	return result.Providers, nil
}
