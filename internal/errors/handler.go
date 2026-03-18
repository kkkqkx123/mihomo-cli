package errors

import (
	"fmt"
	"os"

	"github.com/kkkqkx123/mihomo-cli/pkg/errors"
)

// ErrorHandler 错误处理器接口
type ErrorHandler interface {
	Handle(err error)
	HandleAndExit(err error)
	ShouldExit(err error) bool
}

// CLIHandler CLI 错误处理器
type CLIHandler struct {
	verbose bool
}

// NewCLIHandler 创建 CLI 错误处理器
func NewCLIHandler(verbose bool) *CLIHandler {
	return &CLIHandler{
		verbose: verbose,
	}
}

// Handle 处理错误
func (h *CLIHandler) Handle(err error) {
	if err == nil {
		return
	}

	// 使用 pkg/errors 的打印函数
	errors.PrintError(err)

	// 如果是详细模式，打印更多信息
	if h.verbose {
		h.printVerboseInfo(err)
	}
}

// HandleAndExit 处理错误并退出
func (h *CLIHandler) HandleAndExit(err error) {
	if err == nil {
		return
	}

	// 使用 pkg/errors 的打印和退出函数
	errors.PrintErrorAndExit(err)
}

// ShouldExit 检查是否应该退出
func (h *CLIHandler) ShouldExit(err error) bool {
	if err == nil {
		return false
	}

	// 所有错误都应该退出
	return true
}

// printVerboseInfo 打印详细错误信息
func (h *CLIHandler) printVerboseInfo(err error) {
	// 打印错误堆栈
	fmt.Fprintf(os.Stderr, "\nDebug info:\n")
	fmt.Fprintf(os.Stderr, "  Type: %T\n", err)
	fmt.Fprintf(os.Stderr, "  Message: %s\n", err.Error())

	// 如果是 CLI 错误，打印更多信息
	if cliErr := errors.GetCLIError(err); cliErr != nil {
		fmt.Fprintf(os.Stderr, "  Exit Code: %d\n", cliErr.Code)
		if cliErr.Cause != nil {
			fmt.Fprintf(os.Stderr, "  Cause: %v\n", cliErr.Cause)
		}
	}
}

// HandleCmdError 处理命令错误（Cobra 命令使用）
func HandleCmdError(err error, verbose bool) {
	if err == nil {
		return
	}

	handler := NewCLIHandler(verbose)
	handler.HandleAndExit(err)
}

// GetSuggestion 根据错误类型提供建议
func GetSuggestion(err error) string {
	if err == nil {
		return ""
	}

	cliErr := errors.GetCLIError(err)
	if cliErr == nil {
		return ""
	}

	switch cliErr.Code {
	case errors.ExitInvalid:
		return "请检查命令参数是否正确，使用 --help 查看帮助信息"
	case errors.ExitNetwork:
		return "请检查网络连接和 Mihomo 服务是否正常运行"
	case errors.ExitAPI:
		return "请检查 API 地址和密钥配置是否正确"
	case errors.ExitConfig:
		return "请检查配置文件格式和内容是否正确"
	case errors.ExitAuth:
		return "请检查 API 密钥是否正确"
	case errors.ExitTimeout:
		return "请检查网络连接或增加超时时间"
	case errors.ExitService:
		return "请检查 Mihomo 服务状态"
	default:
		return "如果问题持续存在，请查看日志获取更多信息"
	}
}

// PrintErrorWithSuggestion 打印错误并提供建议
func PrintErrorWithSuggestion(err error, verbose bool) {
	if err == nil {
		return
	}

	// 打印错误
	handler := NewCLIHandler(verbose)
	handler.Handle(err)

	// 打印建议
	suggestion := GetSuggestion(err)
	if suggestion != "" {
		fmt.Fprintf(os.Stderr, "\n建议: %s\n", suggestion)
	}
}

// ExitWithError 打印错误并退出（带建议）
func ExitWithError(err error, verbose bool) {
	if err == nil {
		return
	}

	// 打印错误和建议
	PrintErrorWithSuggestion(err, verbose)

	// 退出
	exitCode := errors.GetExitCode(err)
	os.Exit(exitCode)
}

// CheckError 检查错误，如果存在则返回格式化的错误信息
func CheckError(err error) (string, int) {
	if err == nil {
		return "", errors.ExitSuccess
	}

	return FormatErrorWithExitCode(err)
}

// PanicHandler 恢复 panic 并转换为错误
func PanicHandler() {
	if r := recover(); r != nil {
		var err error
		switch v := r.(type) {
		case error:
			err = v
		case string:
			err = fmt.Errorf("panic: %s", v)
		default:
			err = fmt.Errorf("panic: %v", v)
		}

		// 使用错误处理器处理
		handler := NewCLIHandler(true)
		handler.HandleAndExit(err)
	}
}

// SafeExecute 安全执行函数，捕获 panic
func SafeExecute(fn func() error) error {
	defer PanicHandler()

	return fn()
}

// SafeExecuteWithValue 安全执行函数，捕获 panic 并返回值
func SafeExecuteWithValue[T any](fn func() (T, error)) (T, error) {
	var zero T

	// 使用 recovered 变量来捕获 panic
	var recovered interface{}
	defer func() {
		recovered = recover()
	}()

	// 如果发生 panic，返回零值和错误
	if recovered != nil {
		var err error
		switch v := recovered.(type) {
		case error:
			err = v
		case string:
			err = fmt.Errorf("panic: %s", v)
		default:
			err = fmt.Errorf("panic: %v", v)
		}
		return zero, err
	}

	// 正常执行函数
	return fn()
}