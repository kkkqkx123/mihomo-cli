package api

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// ErrorCode 错误码
type ErrorCode int

const (
	ErrSuccess       ErrorCode = 0 // 成功
	ErrGeneral       ErrorCode = 1 // 通用错误
	ErrAPIConnection ErrorCode = 2 // API 连接错误
	ErrAPIAuth       ErrorCode = 3 // API 认证错误
	ErrInvalidArgs   ErrorCode = 4 // 参数无效
	ErrNotFound      ErrorCode = 5 // 资源不存在
	ErrPermission    ErrorCode = 6 // 权限不足
	ErrFileOperation ErrorCode = 7 // 文件操作错误
	ErrYAMLParse     ErrorCode = 8 // YAML 解析错误
	ErrTimeout       ErrorCode = 9 // 请求超时
	ErrAPIError      ErrorCode = 10 // API 返回错误
)

// APIError API 错误类型
type APIError struct {
	Code       ErrorCode `json:"code"`       // 错误码
	Message    string    `json:"message"`    // 错误消息
	StatusCode int       `json:"statusCode"` // HTTP 状态码
	Cause      error     `json:"-"`          // 原始错误
}

// Error 实现 error 接口
func (e *APIError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("[%d] %s: %v", e.Code, e.Message, e.Cause)
	}
	return fmt.Sprintf("[%d] %s", e.Code, e.Message)
}

// Unwrap 返回原始错误
func (e *APIError) Unwrap() error {
	return e.Cause
}

// ExitCode 返回退出码
func (e *APIError) ExitCode() int {
	return int(e.Code)
}

// NewAPIError 创建新的 API 错误
func NewAPIError(code ErrorCode, message string, cause error) *APIError {
	return &APIError{
		Code:    code,
		Message: message,
		Cause:   cause,
	}
}

// NewConnectionError 创建连接错误
func NewConnectionError(err error) *APIError {
	return NewAPIError(ErrAPIConnection, "failed to connect to API server", err)
}

// NewAuthError 创建认证错误
func NewAuthError(err error) *APIError {
	return NewAPIError(ErrAPIAuth, "authentication failed", err)
}

// NewTimeoutError 创建超时错误
func NewTimeoutError(err error) *APIError {
	return NewAPIError(ErrTimeout, "request timeout", err)
}

// NewNotFoundError 创建未找到错误
func NewNotFoundError(resource string) *APIError {
	return NewAPIError(ErrNotFound, fmt.Sprintf("resource not found: %s", resource), nil)
}

// ParseErrorResponse 解析 API 错误响应
func ParseErrorResponse(resp *http.Response) *APIError {
	// 读取响应体
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return NewAPIError(ErrAPIError, "failed to read error response", err)
	}

	// 尝试解析 JSON 格式错误
	var errorResp struct {
		Message string `json:"message"`
		Error   string `json:"error"`
	}

	if err := json.Unmarshal(body, &errorResp); err == nil {
		message := errorResp.Message
		if message == "" {
			message = errorResp.Error
		}
		if message == "" {
			message = http.StatusText(resp.StatusCode)
		}

		return &APIError{
			Code:       mapStatusCodeToErrorCode(resp.StatusCode),
			Message:    message,
			StatusCode: resp.StatusCode,
		}
	}

	// 如果不是 JSON 格式，使用 HTTP 状态文本
	return &APIError{
		Code:       mapStatusCodeToErrorCode(resp.StatusCode),
		Message:    http.StatusText(resp.StatusCode),
		StatusCode: resp.StatusCode,
	}
}

// mapStatusCodeToErrorCode 将 HTTP 状态码映射到错误码
func mapStatusCodeToErrorCode(statusCode int) ErrorCode {
	switch statusCode {
	case http.StatusUnauthorized:
		return ErrAPIAuth
	case http.StatusForbidden:
		return ErrPermission
	case http.StatusNotFound:
		return ErrNotFound
	case http.StatusBadRequest:
		return ErrInvalidArgs
	case http.StatusGatewayTimeout, http.StatusRequestTimeout:
		return ErrTimeout
	case http.StatusInternalServerError, http.StatusServiceUnavailable:
		return ErrAPIError
	default:
		return ErrGeneral
	}
}

// IsAPIConnectionError 检查是否为连接错误
func IsAPIConnectionError(err error) bool {
	apiErr, ok := err.(*APIError)
	return ok && apiErr.Code == ErrAPIConnection
}

// IsAPIAuthError 检查是否为认证错误
func IsAPIAuthError(err error) bool {
	apiErr, ok := err.(*APIError)
	return ok && apiErr.Code == ErrAPIAuth
}

// IsTimeoutError 检查是否为超时错误
func IsTimeoutError(err error) bool {
	apiErr, ok := err.(*APIError)
	return ok && apiErr.Code == ErrTimeout
}

// IsNotFoundError 检查是否为未找到错误
func IsNotFoundError(err error) bool {
	apiErr, ok := err.(*APIError)
	return ok && apiErr.Code == ErrNotFound
}