package api

import (
	"context"
	"errors"
)

// FlushFakeIP 清空 FakeIP 池
func (c *Client) FlushFakeIP(ctx context.Context) error {
	err := c.Post(ctx, "/cache/fakeip/flush", nil, nil, nil)
	if err != nil {
		// 检查是否为特定的 FakeIP 未启用错误
		var apiErr *APIError
		if errors.As(err, &apiErr) {
			if apiErr.StatusCode == 400 || apiErr.StatusCode == 404 {
				return NewAPIError(ErrInvalidArgs, "FakeIP 未启用", apiErr)
			}
		}
		// 直接透传原始 API 错误，避免双重包装导致错误码丢失
		return err
	}
	return nil
}

// FlushDNS 清空 DNS 缓存
func (c *Client) FlushDNS(ctx context.Context) error {
	err := c.Post(ctx, "/cache/dns/flush", nil, nil, nil)
	if err != nil {
		return err
	}
	return nil
}
