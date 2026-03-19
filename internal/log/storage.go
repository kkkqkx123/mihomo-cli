package log

import (
	"encoding/csv"
	"fmt"
	"os"
	"strings"

	"github.com/kkkqkx123/mihomo-cli/internal/output"
	"github.com/kkkqkx123/mihomo-cli/pkg/types"
)

// ExportFormat 导出格式
type ExportFormat string

const (
	// FormatJSON JSON 格式
	FormatJSON ExportFormat = "json"
	// FormatTXT TXT 格式
	FormatTXT ExportFormat = "txt"
	// FormatCSV CSV 格式
	FormatCSV ExportFormat = "csv"
)

// LogExporter 日志导出器
type LogExporter struct {
	format   ExportFormat
	filter   *LogFilter
	filename string
}

// NewLogExporter 创建新的日志导出器
func NewLogExporter(filename string, format ExportFormat) *LogExporter {
	return &LogExporter{
		format:   format,
		filename: filename,
	}
}

// WithFilter 设置过滤器
func (le *LogExporter) WithFilter(filter *LogFilter) *LogExporter {
	le.filter = filter
	return le
}

// ExportToFile 导出日志到文件
func (le *LogExporter) ExportToFile(logs []*types.LogInfo) error {
	// 应用过滤器
	exportLogs := logs
	if le.filter != nil {
		exportLogs = FilterLogs(logs, le.filter)
	}

	// 创建文件
	file, err := os.Create(le.filename)
	if err != nil {
		return fmt.Errorf("创建文件失败: %w", err)
	}
	defer file.Close()

	// 根据格式导出
	switch le.format {
	case FormatJSON:
		return le.exportToJSON(file, exportLogs)
	case FormatTXT:
		return le.exportToTXT(file, exportLogs)
	case FormatCSV:
		return le.exportToCSV(file, exportLogs)
	default:
		return fmt.Errorf("不支持的导出格式: %s", le.format)
	}
}

// exportToJSON 导出为 JSON 格式
func (le *LogExporter) exportToJSON(file *os.File, logs []*types.LogInfo) error {
	encoder := output.NewJSONEncoderWithWriter(file)
	return encoder.Encode(logs)
}

// exportToTXT 导出为 TXT 格式
func (le *LogExporter) exportToTXT(file *os.File, logs []*types.LogInfo) error {
	for _, log := range logs {
		line := fmt.Sprintf("[%s] %s\n", log.LogType, log.Payload)
		if _, err := file.WriteString(line); err != nil {
			return err
		}
	}
	return nil
}

// exportToCSV 导出为 CSV 格式
func (le *LogExporter) exportToCSV(file *os.File, logs []*types.LogInfo) error {
	writer := csv.NewWriter(file)
	defer writer.Flush()

	// 写入表头
	header := []string{"Level", "Message"}
	if err := writer.Write(header); err != nil {
		return err
	}

	// 写入数据
	for _, log := range logs {
		// 处理包含逗号或换行符的消息
		payload := strings.ReplaceAll(log.Payload, "\n", "\\n")
		record := []string{log.LogType, payload}
		if err := writer.Write(record); err != nil {
			return err
		}
	}

	return nil
}

// ExportLogsToFile 便捷函数：导出日志到文件
func ExportLogsToFile(logs []*types.LogInfo, filename string, format ExportFormat, filter *LogFilter) error {
	exporter := NewLogExporter(filename, format)
	if filter != nil {
		exporter.WithFilter(filter)
	}
	return exporter.ExportToFile(logs)
}
