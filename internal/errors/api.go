package errors

import (
	"fmt"

	"github.com/kkkqkx123/mihomo-cli/internal/api"
	"github.com/kkkqkx123/mihomo-cli/pkg/errors"
)

// APIErrorToCLIError 将 API 错误转换为 CLI 错误
func APIErrorToCLIError(apiErr *api.APIError) *errors.CLIError {
	if apiErr == nil {
		return nil
	}

	switch apiErr.Code {
	case api.ErrAPIConnection:
		return errors.ErrNetwork("API 连接失败", apiErr.Cause)

	case api.ErrAPIAuth:
		return errors.ErrAuth("API 认证失败", apiErr.Cause)

	case api.ErrTimeout:
		return errors.ErrTimeout("请求超时", apiErr.Cause)

	case api.ErrNotFound:
		return errors.ErrInvalidArg("资源不存在", apiErr.Cause)

	case api.ErrInvalidArgs:
		return errors.ErrInvalidArg("无效参数", apiErr.Cause)

	case api.ErrPermission:
		return errors.ErrAuth("权限不足", apiErr.Cause)

	case api.ErrFileOperation:
		return errors.ErrConfig("文件操作失败", apiErr.Cause)

	case api.ErrYAMLParse:
		return errors.ErrConfig("配置解析失败", apiErr.Cause)

	default:
		return errors.ErrAPI("API 错误", apiErr.Cause)
	}
}

// WrapAPIError 包装 API 错误，自动转换为 CLI 错误
func WrapAPIError(message string, err error) *errors.CLIError {
	if err == nil {
		return nil
	}

	// 如果已经是 CLI 错误，直接返回
	if cliErr := errors.GetCLIError(err); cliErr != nil {
		return errors.WrapError(message, cliErr)
	}

	// 如果是 API 错误，转换后包装
	if apiErr, ok := err.(*api.APIError); ok {
		cliErr := APIErrorToCLIError(apiErr)
		if cliErr != nil {
			return errors.WrapError(message, cliErr)
		}
	}

	// 其他错误，使用默认包装
	return errors.WrapError(message, err)
}

// WrapAPIErrorWithCode 包装 API 错误并指定退出码
func WrapAPIErrorWithCode(code int, message string, err error) *errors.CLIError {
	if err == nil {
		return nil
	}

	// 如果已经是 CLI 错误，保留原错误码
	if cliErr := errors.GetCLIError(err); cliErr != nil {
		return errors.WrapErrorWithCode(cliErr.Code, message, cliErr)
	}

	// 如果是 API 错误，转换后包装
	if apiErr, ok := err.(*api.APIError); ok {
		cliErr := APIErrorToCLIError(apiErr)
		if cliErr != nil {
			return errors.WrapErrorWithCode(cliErr.Code, message, cliErr)
		}
	}

	// 其他错误，使用指定的退出码
	return errors.WrapErrorWithCode(code, message, err)
}

// IsAPIConnectionError 检查是否为 API 连接错误
func IsAPIConnectionError(err error) bool {
	if err == nil {
		return false
	}

	// 检查是否是 CLI 网络错误
	if cliErr := errors.GetCLIError(err); cliErr != nil {
		return cliErr.Code == errors.ExitNetwork
	}

	// 检查是否是 API 连接错误
	if apiErr, ok := err.(*api.APIError); ok {
		return api.IsAPIConnectionError(apiErr)
	}

	return false
}

// IsAPIAuthError 检查是否为 API 认证错误
func IsAPIAuthError(err error) bool {
	if err == nil {
		return false
	}

	// 检查是否是 CLI 认证错误
	if cliErr := errors.GetCLIError(err); cliErr != nil {
		return cliErr.Code == errors.ExitAuth
	}

	// 检查是否是 API 认证错误
	if apiErr, ok := err.(*api.APIError); ok {
		return api.IsAPIAuthError(apiErr)
	}

	return false
}

// IsAPITimeoutError 检查是否为 API 超时错误
func IsAPITimeoutError(err error) bool {
	if err == nil {
		return false
	}

	// 检查是否是 CLI 超时错误
	if cliErr := errors.GetCLIError(err); cliErr != nil {
		return cliErr.Code == errors.ExitTimeout
	}

	// 检查是否是 API 超时错误
	if apiErr, ok := err.(*api.APIError); ok {
		return api.IsTimeoutError(apiErr)
	}

	return false
}

// IsAPINotFoundError 检查是否为 API 未找到错误
func IsAPINotFoundError(err error) bool {
	if err == nil {
		return false
	}

	// 检查是否是 CLI 无效参数错误
	if cliErr := errors.GetCLIError(err); cliErr != nil {
		return cliErr.Code == errors.ExitInvalid
	}

	// 检查是否是 API 未找到错误
	if apiErr, ok := err.(*api.APIError); ok {
		return api.IsNotFoundError(apiErr)
	}

	return false
}

// FormatAPIError 格式化 API 错误信息
func FormatAPIError(err error) string {
	if err == nil {
		return ""
	}

	// 如果是 API 错误，格式化输出
	if apiErr, ok := err.(*api.APIError); ok {
		statusCode := apiErr.StatusCode
		if statusCode == 0 {
			return fmt.Sprintf("API 错误: %s", apiErr.Message)
		}
		return fmt.Sprintf("API 错误 (HTTP %d): %s", statusCode, apiErr.Message)
	}

	// 其他错误，使用默认格式化
	return FormatError(err)
}