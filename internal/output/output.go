package output

import (
	"fmt"
	"io"
)

// Print 打印输出（根据格式，使用默认 stdout）
func Print(data interface{}, format string) error {
	return PrintWithWriter(GetGlobalStdout(), data, format)
}

// PrintWithWriter 使用指定 Writer 打印输出
func PrintWithWriter(w io.Writer, data interface{}, format string) error {
	if format == "json" {
		return PrintJSONWithWriter(w, data)
	}
	_, err := fmt.Fprintf(w, "%v\n", data)
	return err
}

// PrintError 打印错误信息（使用默认 stderr）
func PrintError(msg string) error {
	return PrintErrorWithWriter(GetGlobalStderr(), msg)
}

// PrintErrorWithWriter 使用指定 Writer 打印错误信息
func PrintErrorWithWriter(w io.Writer, msg string) error {
	FError(w, msg)
	return nil
}

// PrintSuccess 打印成功信息（使用默认 stdout）
func PrintSuccess(msg string) {
	Success(msg)
}

// PrintSuccessWithWriter 使用指定 Writer 打印成功信息
func PrintSuccessWithWriter(w io.Writer, msg string) {
	FSuccess(w, msg)
}

// PrintWarning 打印警告信息（使用默认 stdout）
func PrintWarning(msg string) {
	Warning(msg)
}

// PrintWarningWithWriter 使用指定 Writer 打印警告信息
func PrintWarningWithWriter(w io.Writer, msg string) {
	FWarning(w, msg)
}

// PrintInfo 打印信息（使用默认 stdout）
func PrintInfo(msg string) {
	Info(msg)
}

// PrintInfoWithWriter 使用指定 Writer 打印信息
func PrintInfoWithWriter(w io.Writer, msg string) {
	FInfo(w, msg)
}
