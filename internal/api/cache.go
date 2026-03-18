package api

import (
	"context"
)

// FlushFakeIP 清空 FakeIP 池
func (c *Client) FlushFakeIP(ctx context.Context) error {
	err := c.Post(ctx, "/cache/fakeip/flush", nil, nil, nil)
	if err != nil {
		// 检查是否为特定的 FakeIP 未启用错误
		if apiErr, ok := err.(*APIError); ok {
			if apiErr.StatusCode == 400 || apiErr.StatusCode == 404 {
				return NewAPIError(ErrInvalidArgs, "FakeIP 未启用", apiErr)
			}
		}
		return NewAPIError(ErrAPIError, "清空 FakeIP 池失败", err)
	}
	return nil
}

// FlushDNS 清空 DNS 缓存
func (c *Client) FlushDNS(ctx context.Context) error {
	err := c.Post(ctx, "/cache/dns/flush", nil, nil, nil)
	if err != nil {
		return NewAPIError(ErrAPIError, "清空 DNS 缓存失败", err)
	}
	return nil
}
