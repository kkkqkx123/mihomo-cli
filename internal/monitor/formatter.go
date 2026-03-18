package monitor

import (
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/kkkqkx123/mihomo-cli/internal/output"
	"github.com/kkkqkx123/mihomo-cli/pkg/types"
)

// formatBytes 格式化字节数为人类可读格式
func formatBytes(bytes int64) string {
	const (
		KB = 1024
		MB = KB * 1024
		GB = MB * 1024
		TB = GB * 1024
	)

	switch {
	case bytes >= TB:
		return fmt.Sprintf("%.2f TB", float64(bytes)/float64(TB))
	case bytes >= GB:
		return fmt.Sprintf("%.2f GB", float64(bytes)/float64(GB))
	case bytes >= MB:
		return fmt.Sprintf("%.2f MB", float64(bytes)/float64(MB))
	case bytes >= KB:
		return fmt.Sprintf("%.2f KB", float64(bytes)/float64(KB))
	default:
		return fmt.Sprintf("%d B", bytes)
	}
}

// formatSpeed 格式化速度（字节/秒）
func formatSpeed(bytesPerSec int64) string {
	return formatBytes(bytesPerSec) + "/s"
}

// TrafficFormatter 流量格式化器
type TrafficFormatter struct {
	w io.Writer
}

// NewTrafficFormatter 创建流量格式化器
func NewTrafficFormatter(w io.Writer) *TrafficFormatter {
	if w == nil {
		w = os.Stdout
	}
	return &TrafficFormatter{w: w}
}

// FormatOnce 格式化单次流量数据
func (f *TrafficFormatter) FormatOnce(traffic *types.TrafficInfo) error {
	fmt.Fprintf(f.w, "流量统计:\n")
	fmt.Fprintf(f.w, "  上传速度: %s\n", formatSpeed(traffic.Up))
	fmt.Fprintf(f.w, "  下载速度: %s\n", formatSpeed(traffic.Down))
	return nil
}

// FormatWatchHeader 格式化 Watch 模式头部
func (f *TrafficFormatter) FormatWatchHeader() {
	fmt.Fprintf(f.w, "\033[2J\033[H") // 清屏并移动光标到左上角
	fmt.Fprintf(f.w, "实时流量监控 (按 Ctrl+C 退出)\n")
	fmt.Fprintf(f.w, "%s\n", strings.Repeat("-", 50))
}

// FormatWatchLine 格式化 Watch 模式单行数据
func (f *TrafficFormatter) FormatWatchLine(traffic *types.TrafficInfo, totalUp, totalDown int64) {
	fmt.Fprintf(f.w, "\033[4;0H") // 移动到第4行
	fmt.Fprintf(f.w, "\033[K")   // 清除该行
	fmt.Fprintf(f.w, "时间: %s\n", time.Now().Format("2006-01-02 15:04:05"))
	fmt.Fprintf(f.w, "\033[K")
	fmt.Fprintf(f.w, "上传速度: %s\n", formatSpeed(traffic.Up))
	fmt.Fprintf(f.w, "\033[K")
	fmt.Fprintf(f.w, "下载速度: %s\n", formatSpeed(traffic.Down))
	fmt.Fprintf(f.w, "\033[K")
	fmt.Fprintf(f.w, "累计上传: %s\n", formatBytes(totalUp))
	fmt.Fprintf(f.w, "\033[K")
	fmt.Fprintf(f.w, "累计下载: %s\n", formatBytes(totalDown))
}

// FormatJSON 以 JSON 格式输出
func (f *TrafficFormatter) FormatJSON(traffic *types.TrafficInfo) error {
	return output.PrintJSONWithWriter(f.w, traffic)
}

// MemoryFormatter 内存格式化器
type MemoryFormatter struct {
	w io.Writer
}

// NewMemoryFormatter 创建内存格式化器
func NewMemoryFormatter(w io.Writer) *MemoryFormatter {
	if w == nil {
		w = os.Stdout
	}
	return &MemoryFormatter{w: w}
}

// FormatOnce 格式化单次内存数据
func (f *MemoryFormatter) FormatOnce(memory *types.MemoryInfo) error {
	fmt.Fprintf(f.w, "内存使用:\n")
	fmt.Fprintf(f.w, "  当前使用: %s\n", formatBytes(memory.Inuse))
	if memory.OSLimit != nil {
		fmt.Fprintf(f.w, "  系统限制: %s\n", formatBytes(*memory.OSLimit))
		usagePercent := float64(memory.Inuse) / float64(*memory.OSLimit) * 100
		fmt.Fprintf(f.w, "  使用率: %.2f%%\n", usagePercent)
	}
	return nil
}

// FormatWatchHeader 格式化 Watch 模式头部
func (f *MemoryFormatter) FormatWatchHeader() {
	fmt.Fprintf(f.w, "\033[2J\033[H") // 清屏并移动光标到左上角
	fmt.Fprintf(f.w, "实时内存监控 (按 Ctrl+C 退出)\n")
	fmt.Fprintf(f.w, "%s\n", strings.Repeat("-", 50))
}

// FormatWatchLine 格式化 Watch 模式单行数据
func (f *MemoryFormatter) FormatWatchLine(memory *types.MemoryInfo) {
	fmt.Fprintf(f.w, "\033[4;0H") // 移动到第4行
	fmt.Fprintf(f.w, "\033[K")   // 清除该行
	fmt.Fprintf(f.w, "时间: %s\n", time.Now().Format("2006-01-02 15:04:05"))
	fmt.Fprintf(f.w, "\033[K")
	fmt.Fprintf(f.w, "当前使用: %s\n", formatBytes(memory.Inuse))
	if memory.OSLimit != nil {
		fmt.Fprintf(f.w, "\033[K")
		fmt.Fprintf(f.w, "系统限制: %s\n", formatBytes(*memory.OSLimit))
		usagePercent := float64(memory.Inuse) / float64(*memory.OSLimit) * 100
		fmt.Fprintf(f.w, "\033[K")
		fmt.Fprintf(f.w, "使用率: %.2f%%\n", usagePercent)
	}
}

// FormatJSON 以 JSON 格式输出
func (f *MemoryFormatter) FormatJSON(memory *types.MemoryInfo) error {
	return output.PrintJSONWithWriter(f.w, memory)
}
