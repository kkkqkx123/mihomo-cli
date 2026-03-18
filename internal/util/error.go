package util

import (
	"fmt"
	"os"

	"github.com/fatih/color"
)

// ExitCode 定义退出码常量
const (
	ExitSuccess = 0    // 成功
	ExitError   = 1    // 一般错误
	ExitInvalid = 2    // 无效参数或用法错误
	ExitNetwork = 3    // 网络错误
	ExitAPI     = 4    // API 错误
	ExitConfig  = 5    // 配置错误
	ExitService = 6    // 服务错误
	ExitTimeout = 7    // 超时错误
	ExitAuth    = 8    // 认证错误
)

// CLIError CLI 错误类型
type CLIError struct {
	Code    int    // 退出码
	Message string // 错误消息
	Cause   error  // 原始错误
}

// Error 实现 error 接口
func (e *CLIError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("%s: %v", e.Message, e.Cause)
	}
	return e.Message
}

// Unwrap 实现错误解包
func (e *CLIError) Unwrap() error {
	return e.Cause
}

// NewError 创建新的 CLI 错误
func NewError(code int, message string, cause error) *CLIError {
	return &CLIError{
		Code:    code,
		Message: message,
		Cause:   cause,
	}
}

// WrapError 包装错误，添加上下文信息
func WrapError(message string, err error) *CLIError {
	code := ExitError
	if cliErr, ok := err.(*CLIError); ok {
		code = cliErr.Code
	}
	return &CLIError{
		Code:    code,
		Message: message,
		Cause:   err,
	}
}

// WrapErrorWithCode 包装错误并指定退出码
func WrapErrorWithCode(code int, message string, err error) *CLIError {
	return &CLIError{
		Code:    code,
		Message: message,
		Cause:   err,
	}
}

// 便捷错误创建函数

// ErrInvalidArg 创建无效参数错误
func ErrInvalidArg(message string, cause error) *CLIError {
	return NewError(ExitInvalid, message, cause)
}

// ErrNetwork 创建网络错误
func ErrNetwork(message string, cause error) *CLIError {
	return NewError(ExitNetwork, message, cause)
}

// ErrAPI 创建 API 错误
func ErrAPI(message string, cause error) *CLIError {
	return NewError(ExitAPI, message, cause)
}

// ErrConfig 创建配置错误
func ErrConfig(message string, cause error) *CLIError {
	return NewError(ExitConfig, message, cause)
}

// ErrService 创建服务错误
func ErrService(message string, cause error) *CLIError {
	return NewError(ExitService, message, cause)
}

// ErrTimeout 创建超时错误
func ErrTimeout(message string, cause error) *CLIError {
	return NewError(ExitTimeout, message, cause)
}

// ErrAuth 创建认证错误
func ErrAuth(message string, cause error) *CLIError {
	return NewError(ExitAuth, message, cause)
}

// PrintError 打印错误信息（彩色）
func PrintError(err error) {
	if err == nil {
		return
	}

	var message string
	if cliErr, ok := err.(*CLIError); ok {
		message = cliErr.Error()
	} else {
		message = err.Error()
	}

	fmt.Fprintln(os.Stderr, color.RedString("✗ "+message))
}

// PrintErrorAndExit 打印错误信息并退出
func PrintErrorAndExit(err error) {
	PrintError(err)

	code := ExitError
	if cliErr, ok := err.(*CLIError); ok {
		code = cliErr.Code
	}

	os.Exit(code)
}

// HandleError 统一错误处理
// 返回 true 表示已处理错误，false 表示无错误
func HandleError(err error) bool {
	if err == nil {
		return false
	}

	PrintError(err)
	return true
}

// HandleErrorAndExit 统一错误处理并退出
func HandleErrorAndExit(err error) {
	if err == nil {
		return
	}

	PrintErrorAndExit(err)
}

// GetExitCode 获取错误的退出码
func GetExitCode(err error) int {
	if err == nil {
		return ExitSuccess
	}

	if cliErr, ok := err.(*CLIError); ok {
		return cliErr.Code
	}

	return ExitError
}

// IsCLIError 检查是否是 CLI 错误
func IsCLIError(err error) bool {
	_, ok := err.(*CLIError)
	return ok
}

// GetCLIError 获取 CLI 错误
func GetCLIError(err error) *CLIError {
	if cliErr, ok := err.(*CLIError); ok {
		return cliErr
	}
	return nil
}
