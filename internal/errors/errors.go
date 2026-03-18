package errors

import (
	"fmt"

	"github.com/kkkqkx123/mihomo-cli/pkg/errors"
)

// FormatError 格式化错误信息，用于命令行输出
func FormatError(err error) string {
	if err == nil {
		return ""
	}

	// 如果是 CLI 错误，直接返回
	if cliErr := errors.GetCLIError(err); cliErr != nil {
		return cliErr.Error()
	}

	// 其他错误类型
	return err.Error()
}

// FormatErrorWithExitCode 格式化错误信息并返回退出码
func FormatErrorWithExitCode(err error) (string, int) {
	if err == nil {
		return "", errors.ExitSuccess
	}

	// 获取退出码
	exitCode := errors.GetExitCode(err)

	// 格式化错误信息
	message := FormatError(err)

	return message, exitCode
}

// IsRetryable 检查错误是否可重试
func IsRetryable(err error) bool {
	if err == nil {
		return false
	}

	// 检查是否是 CLI 错误
	cliErr := errors.GetCLIError(err)
	if cliErr != nil {
		// 网络错误和超时错误可以重试
		return cliErr.Code == errors.ExitNetwork || cliErr.Code == errors.ExitTimeout
	}

	return false
}

// IsUserError 检查是否是用户错误（如参数错误）
func IsUserError(err error) bool {
	if err == nil {
		return false
	}

	cliErr := errors.GetCLIError(err)
	if cliErr != nil {
		// 参数错误、配置错误属于用户错误
		return cliErr.Code == errors.ExitInvalid || cliErr.Code == errors.ExitConfig
	}

	return false
}

// WrapWithLocation 包装错误并添加位置信息
func WrapWithLocation(message string, err error, location string) error {
	if err == nil {
		return nil
	}

	fullMessage := fmt.Sprintf("%s (at %s)", message, location)
	return errors.WrapError(fullMessage, err)
}

// NewWithLocation 创建错误并添加位置信息
func NewWithLocation(code int, message string, location string) error {
	fullMessage := fmt.Sprintf("%s (at %s)", message, location)
	return errors.NewError(code, fullMessage, nil)
}

// CombineErrors 合并多个错误
func CombineErrors(errs []error) error {
	if len(errs) == 0 {
		return nil
	}

	var nonNilErrs []error
	for _, err := range errs {
		if err != nil {
			nonNilErrs = append(nonNilErrs, err)
		}
	}

	if len(nonNilErrs) == 0 {
		return nil
	}

	if len(nonNilErrs) == 1 {
		return nonNilErrs[0]
	}

	return fmt.Errorf("%d errors occurred: %v", len(nonNilErrs), nonNilErrs)
}

// NewValidationError 创建验证错误
func NewValidationError(format string, args ...interface{}) error {
	message := fmt.Sprintf(format, args...)
	return errors.NewError(errors.ExitInvalid, message, nil)
}