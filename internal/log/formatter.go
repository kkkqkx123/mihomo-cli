package log

import (
	"fmt"
	"strings"

	"github.com/kkkqkx123/mihomo-cli/internal/output"
	"github.com/kkkqkx123/mihomo-cli/pkg/types"

	"github.com/fatih/color"
)

var (
	// 颜色定义
	infoColor    = color.New(color.FgGreen)
	warnColor    = color.New(color.FgYellow)
	errorColor   = color.New(color.FgRed)
	debugColor   = color.New(color.FgCyan)
	silentColor  = color.New(color.FgHiBlack)
)

// FormatLogMessage 格式化单条日志消息
func FormatLogMessage(log *types.LogInfo) string {
	var prefix string
	var msgColor *color.Color

	switch strings.ToLower(log.LogType) {
	case "info":
		prefix = "[INFO]"
		msgColor = infoColor
	case "warning", "warn":
		prefix = "[WARN]"
		msgColor = warnColor
	case "error":
		prefix = "[ERROR]"
		msgColor = errorColor
	case "debug":
		prefix = "[DEBUG]"
		msgColor = debugColor
	case "silent":
		prefix = "[SILENT]"
		msgColor = silentColor
	default:
		prefix = "[" + strings.ToUpper(log.LogType) + "]"
		msgColor = infoColor
	}

	return msgColor.Sprintf("%s %s", prefix, log.Payload)
}

// PrintLogMessage 打印单条日志消息
func PrintLogMessage(log *types.LogInfo) {
	fmt.Fprintln(output.GetGlobalStdout(), FormatLogMessage(log))
}

// FormatLogHeader 打印日志流头部信息
func FormatLogHeader() {
	output.Info("开始接收日志流 (按 Ctrl+C 停止)...")
	fmt.Fprintln(output.GetGlobalStdout())
}
