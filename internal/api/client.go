package api

import (
	"context"
	"time"
)

// Client Mihomo API 客户端
type Client struct {
	baseURL    string        // API 基础地址
	secret     string        // API 密钥
	httpClient *HTTPClient   // HTTP 客户端
	timeout    time.Duration // 请求超时
}

// ClientOption 客户端配置选项
type ClientOption func(*Client)

// WithTimeout 设置请求超时时间
func WithTimeout(timeout time.Duration) ClientOption {
	return func(c *Client) {
		c.timeout = timeout
	}
}

// NewClient 创建新的 API 客户端
func NewClient(baseURL, secret string, opts ...ClientOption) *Client {
	client := &Client{
		baseURL: baseURL,
		secret:  secret,
		timeout: 10 * time.Second, // 默认 10 秒超时
	}

	// 应用配置选项
	for _, opt := range opts {
		opt(client)
	}

	// 初始化 HTTP 客户端
	client.httpClient = NewHTTPClient(client.secret, client.timeout)

	return client
}

// NewClientWithTimeout 创建带有指定超时的 API 客户端（兼容旧接口）
func NewClientWithTimeout(baseURL, secret string, timeout int) *Client {
	return NewClient(baseURL, secret, WithTimeout(time.Duration(timeout)*time.Second))
}

// GetTimeout 获取当前超时时间
func (c *Client) GetTimeout() time.Duration {
	return c.timeout
}

// SetTimeout 设置超时时间
func (c *Client) SetTimeout(timeout time.Duration) {
	c.timeout = timeout
	c.httpClient.SetTimeout(timeout)
}

// GetBaseURL 获取 API 基础地址
func (c *Client) GetBaseURL() string {
	return c.baseURL
}

// GetSecret 获取 API 密钥（谨慎使用）
func (c *Client) GetSecret() string {
	return c.secret
}

// Get 执行 GET 请求
func (c *Client) Get(ctx context.Context, endpoint string, queryParams map[string]string, target interface{}) error {
	return c.httpClient.Get(ctx, c.baseURL, endpoint, queryParams, target)
}

// Post 执行 POST 请求
func (c *Client) Post(ctx context.Context, endpoint string, queryParams map[string]string, body interface{}, target interface{}) error {
	return c.httpClient.Post(ctx, c.baseURL, endpoint, queryParams, body, target)
}

// Put 执行 PUT 请求
func (c *Client) Put(ctx context.Context, endpoint string, queryParams map[string]string, body interface{}, target interface{}) error {
	return c.httpClient.Put(ctx, c.baseURL, endpoint, queryParams, body, target)
}

// Patch 执行 PATCH 请求
func (c *Client) Patch(ctx context.Context, endpoint string, queryParams map[string]string, body interface{}, target interface{}) error {
	return c.httpClient.Patch(ctx, c.baseURL, endpoint, queryParams, body, target)
}

// Delete 执行 DELETE 请求
func (c *Client) Delete(ctx context.Context, endpoint string, queryParams map[string]string, target interface{}) error {
	return c.httpClient.Delete(ctx, c.baseURL, endpoint, queryParams, target)
}