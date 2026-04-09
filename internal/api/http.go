package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path"
	"strings"
	"time"
)

// HTTPClient HTTP 客户端封装
type HTTPClient struct {
	secret  string
	client  *http.Client
	timeout time.Duration
}

// NewHTTPClient 创建新的 HTTP 客户端
func NewHTTPClient(secret string, timeout time.Duration) *HTTPClient {
	return &HTTPClient{
		secret:  secret,
		timeout: timeout,
		client: &http.Client{
			Timeout: timeout,
		},
	}
}

// SetTimeout 设置超时时间
func (c *HTTPClient) SetTimeout(timeout time.Duration) {
	c.timeout = timeout
	c.client.Timeout = timeout
}

// buildURL 构建完整的 API URL
func (c *HTTPClient) buildURL(baseURL, endpoint string, queryParams map[string]string) (string, error) {
	// 检查 baseURL 是否为空
	if baseURL == "" {
		return "", NewConnectionError(fmt.Errorf("API base URL is empty, please configure api.address or use --api flag"))
	}

	// 解析基础 URL
	u, err := url.Parse(baseURL)
	if err != nil {
		return "", NewConnectionError(err)
	}

	// 检查协议是否有效
	if u.Scheme != "http" && u.Scheme != "https" {
		return "", NewConnectionError(fmt.Errorf("unsupported protocol scheme %q, only http and https are supported", u.Scheme))
	}

	// 检查主机是否为空
	if u.Host == "" {
		return "", NewConnectionError(fmt.Errorf("API address must contain a host (e.g., http://127.0.0.1:9090)"))
	}

	// 拼接路径
	u.Path = path.Join(u.Path, endpoint)

	// 添加查询参数
	if len(queryParams) > 0 {
		q := u.Query()
		for key, value := range queryParams {
			q.Set(key, value)
		}
		u.RawQuery = q.Encode()
	}

	return u.String(), nil
}

// addAuthHeader 添加认证头
func (c *HTTPClient) addAuthHeader(req *http.Request) {
	if c.secret != "" {
		req.Header.Set("Authorization", "Bearer "+c.secret)
	}
}

// doRequest 执行 HTTP 请求
func (c *HTTPClient) doRequest(ctx context.Context, method, baseURL, endpoint string, queryParams map[string]string, body interface{}) (*http.Response, error) {
	// 构建完整 URL
	fullURL, err := c.buildURL(baseURL, endpoint, queryParams)
	if err != nil {
		return nil, NewConnectionError(err)
	}

	// 创建请求
	req, err := c.newRequest(ctx, method, fullURL, body)
	if err != nil {
		return nil, err
	}

	// 执行请求
	return c.doRequestWithReq(req)
}

// newRequest 创建 HTTP 请求
func (c *HTTPClient) newRequest(ctx context.Context, method, url string, body interface{}) (*http.Request, error) {
	// 准备请求体
	var reqBody io.Reader
	if body != nil {
		switch v := body.(type) {
		case []byte:
			reqBody = bytes.NewReader(v)
		case string:
			reqBody = strings.NewReader(v)
		default:
			jsonData, err := json.Marshal(body)
			if err != nil {
				return nil, NewAPIError(ErrGeneral, "failed to marshal request body", err)
			}
			reqBody = bytes.NewReader(jsonData)
		}
	}

	// 创建请求
	req, err := http.NewRequestWithContext(ctx, method, url, reqBody)
	if err != nil {
		return nil, NewConnectionError(err)
	}

	// 添加认证头
	c.addAuthHeader(req)

	// 设置 Content-Type
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	return req, nil
}

// doRequestWithReq 使用已创建的请求执行 HTTP 请求
func (c *HTTPClient) doRequestWithReq(req *http.Request) (*http.Response, error) {
	// 执行请求
	resp, err := c.client.Do(req)
	if err != nil {
		// 检查是否为超时错误
		if req.Context().Err() == context.DeadlineExceeded || err.Error() == "http: Client.Timeout exceeded" {
			return nil, NewTimeoutError(err)
		}
		return nil, NewConnectionError(err)
	}

	return resp, nil
}

// handleResponse 处理响应
func (c *HTTPClient) handleResponse(resp *http.Response, target interface{}) error {
	defer resp.Body.Close()

	// 检查状态码
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return ParseErrorResponse(resp)
	}

	// 如果没有目标，直接返回成功
	if target == nil {
		return nil
	}

	// 读取响应体
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return NewAPIError(ErrGeneral, "failed to read response body", err)
	}

	// 解析响应
	switch v := target.(type) {
	case *[]byte:
		*v = respBody
	case *string:
		*v = string(respBody)
	default:
		if err := json.Unmarshal(respBody, target); err != nil {
			return NewAPIError(ErrGeneral, "failed to unmarshal response", err)
		}
	}

	return nil
}

// Get 执行 GET 请求
func (c *HTTPClient) Get(ctx context.Context, baseURL, endpoint string, queryParams map[string]string, target interface{}) error {
	resp, err := c.doRequest(ctx, http.MethodGet, baseURL, endpoint, queryParams, nil)
	if err != nil {
		return err
	}
	return c.handleResponse(resp, target)
}

// Post 执行 POST 请求
func (c *HTTPClient) Post(ctx context.Context, baseURL, endpoint string, queryParams map[string]string, body interface{}, target interface{}) error {
	resp, err := c.doRequest(ctx, http.MethodPost, baseURL, endpoint, queryParams, body)
	if err != nil {
		return err
	}
	return c.handleResponse(resp, target)
}

// Put 执行 PUT 请求
func (c *HTTPClient) Put(ctx context.Context, baseURL, endpoint string, queryParams map[string]string, body interface{}, target interface{}) error {
	resp, err := c.doRequest(ctx, http.MethodPut, baseURL, endpoint, queryParams, body)
	if err != nil {
		return err
	}
	return c.handleResponse(resp, target)
}

// Patch 执行 PATCH 请求
func (c *HTTPClient) Patch(ctx context.Context, baseURL, endpoint string, queryParams map[string]string, body interface{}, target interface{}) error {
	resp, err := c.doRequest(ctx, http.MethodPatch, baseURL, endpoint, queryParams, body)
	if err != nil {
		return err
	}
	return c.handleResponse(resp, target)
}

// Delete 执行 DELETE 请求
func (c *HTTPClient) Delete(ctx context.Context, baseURL, endpoint string, queryParams map[string]string, target interface{}) error {
	resp, err := c.doRequest(ctx, http.MethodDelete, baseURL, endpoint, queryParams, nil)
	if err != nil {
		return err
	}
	return c.handleResponse(resp, target)
}