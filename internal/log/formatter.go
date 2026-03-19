package log

import (
	"fmt"
	"strings"

	"github.com/kkkqkx123/mihomo-cli/internal/output"
	"github.com/kkkqkx123/mihomo-cli/pkg/types"
)

// FormatLogMessage 格式化单条日志消息
func FormatLogMessage(log *types.LogInfo) string {
	var prefix string
	var formattedMsg string

	logType := strings.ToLower(log.LogType)

	switch logType {
	case "info":
		prefix = "[INFO]"
		formattedMsg = output.GreenString("%s %s", prefix, log.Payload)
	case "warning", "warn":
		prefix = "[WARN]"
		formattedMsg = output.YellowString("%s %s", prefix, log.Payload)
	case "error":
		prefix = "[ERROR]"
		formattedMsg = output.RedString("%s %s", prefix, log.Payload)
	case "debug":
		prefix = "[DEBUG]"
		formattedMsg = output.CyanString("%s %s", prefix, log.Payload)
	case "silent":
		prefix = "[SILENT]"
		formattedMsg = output.Dim("%s %s", prefix, log.Payload)
	default:
		prefix = "[" + strings.ToUpper(log.LogType) + "]"
		formattedMsg = output.GreenString("%s %s", prefix, log.Payload)
	}

	return formattedMsg
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
