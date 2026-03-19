package cmd

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/kkkqkx123/mihomo-cli/internal/api"
	"github.com/kkkqkx123/mihomo-cli/internal/log"
	"github.com/kkkqkx123/mihomo-cli/internal/output"
	pkgerrors "github.com/kkkqkx123/mihomo-cli/pkg/errors"
)

var (
	logFollow     bool
	logLevel      string
	logKeyword    []string
	logExclude    []string
	logRegex      string
	logDuration   string
	statsFormat   string
	searchFormat  string
	exportFormat  string
	exportFilename string
)

// NewLogsCmd 创建日志命令
func NewLogsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "logs",
		Short: "查看 Mihomo 日志",
		Long:  `查看 Mihomo 内核的日志输出。支持实时查看、统计、搜索和导出功能。`,
	}

	// 添加子命令
	cmd.AddCommand(
		newLogsViewCmd(),
		newLogsStatsCmd(),
		newLogsSearchCmd(),
		newLogsExportCmd(),
	)

	return cmd
}

// newLogsViewCmd 创建日志查看命令
func newLogsViewCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "view",
		Short: "实时查看 Mihomo 日志",
		Long:  `实时查看 Mihomo 内核的日志输出。支持按级别、关键词、正则表达式过滤。`,
		Example: `  mihomo-cli logs view
  mihomo-cli logs view --follow
  mihomo-cli logs view --level error
  mihomo-cli logs view --keyword "proxy"
  mihomo-cli logs view --exclude "keepalive"
  mihomo-cli logs view --regex "error"`,
		RunE: runLogsView,
	}

	cmd.Flags().BoolVarP(&logFollow, "follow", "f", true, "持续跟踪日志输出")
	cmd.Flags().StringVar(&logLevel, "level", "", "日志级别过滤 (silent/error/warning/info/debug)")
	cmd.Flags().StringSliceVar(&logKeyword, "keyword", nil, "包含关键词过滤（可多次使用，AND 逻辑）")
	cmd.Flags().StringSliceVar(&logExclude, "exclude", nil, "排除关键词过滤（可多次使用）")
	cmd.Flags().StringVar(&logRegex, "regex", "", "正则表达式过滤")

	return cmd
}

// runLogsView 执行日志查看命令
func runLogsView(cmd *cobra.Command, args []string) error {
	// 创建 API 客户端
	client := api.NewClientWithTimeout(
		viper.GetString("api.address"),
		viper.GetString("api.secret"),
		viper.GetInt("api.timeout"),
	)

	// 创建过滤器
	filter := log.NewLogFilter()
	if logLevel != "" {
		filter.WithLevel(logLevel)
	}
	if len(logKeyword) > 0 {
		filter.WithKeywords(logKeyword...)
	}
	if len(logExclude) > 0 {
		filter.WithExclude(logExclude...)
	}
	if logRegex != "" {
		if err := filter.WithRegex(logRegex); err != nil {
			return pkgerrors.WrapError("无效的正则表达式", err)
		}
	}

	// 设置信号处理
	ctx, cancel := context.WithCancel(cmd.Context())
	defer cancel()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigChan
		cancel()
	}()

	// 打印头部信息
	log.FormatLogHeader()
	if logLevel != "" || len(logKeyword) > 0 || len(logExclude) > 0 || logRegex != "" {
		output.Info("日志过滤器已启用")
		fmt.Fprintln(output.GetGlobalStdout())
	}

	// 获取日志流
	stream, err := client.StreamLogs(ctx)
	if err != nil {
		return pkgerrors.WrapError("获取日志流失败", err)
	}
	defer stream.Close()

	// 读取日志消息
	for {
		select {
		case <-ctx.Done():
			fmt.Fprintln(output.GetGlobalStdout())
			output.Info("日志流已停止")
			return nil
		case logMsg, ok := <-stream.Messages():
			if !ok {
				// 检查是否有错误
				if err := stream.Err(); err != nil {
					return pkgerrors.WrapError("读取日志流失败", err)
				}
				output.Info("日志流已关闭")
				return nil
			}
			// 应用过滤器
			if filter.Match(logMsg) {
				log.PrintLogMessage(logMsg)
			}
		case <-time.After(30 * time.Second):
			// 如果30秒没有收到日志，检查连接状态
			if err := stream.Err(); err != nil {
				return pkgerrors.WrapError("日志流连接异常", err)
			}
		}
	}
}

// newLogsStatsCmd 创建日志统计命令
func newLogsStatsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "stats",
		Short: "统计日志信息",
		Long:  `统计 Mihomo 日志信息，包括日志总数、错误率、常见错误等。`,
		Example: `  mihomo-cli logs stats
  mihomo-cli logs stats --duration 5m
  mihomo-cli logs stats -o json`,
		RunE: runLogsStats,
	}

	cmd.Flags().StringVar(&logDuration, "duration", "1m", "收集日志的时间范围（如 30s, 5m, 1h）")
	cmd.Flags().StringVarP(&statsFormat, "output", "o", "table", "输出格式 (table/json)")

	return cmd
}

// runLogsStats 执行日志统计命令
func runLogsStats(cmd *cobra.Command, args []string) error {
	// 解析时间范围
	duration, err := time.ParseDuration(logDuration)
	if err != nil {
		return pkgerrors.WrapError("无效的时间范围格式", err)
	}

	// 创建 API 客户端
	client := api.NewClientWithTimeout(
		viper.GetString("api.address"),
		viper.GetString("api.secret"),
		viper.GetInt("api.timeout"),
	)

	// 创建日志收集器
	collector := log.NewLogCollector(log.DefaultMaxLogs)

	// 收集日志
	ctx := cmd.Context()
	output.Info(fmt.Sprintf("正在收集最近 %s 的日志...", logDuration))
	if err := collector.CollectWithDuration(ctx, client, duration); err != nil {
		return pkgerrors.WrapError("收集日志失败", err)
	}

	logs := collector.GetLogs()
	if len(logs) == 0 {
		output.Info("未收集到日志")
		return nil
	}

	// 计算统计信息
	stats := log.CalculateStatistics(logs)

	// 根据输出格式显示结果
	if statsFormat == "json" {
		output.PrintJSON(stats)
	} else {
		printStatisticsTable(stats)
	}

	return nil
}

// printStatisticsTable 打印统计信息表格
func printStatisticsTable(stats *log.LogStatistics) {
	output.PrintSection("日志统计信息")
	output.PrintSeparator("=", 80)
	output.PrintKeyValue("总日志数", stats.TotalCount)
	output.PrintKeyValue("错误率", fmt.Sprintf("%.2f%%", stats.ErrorRate))
	output.PrintEmptyLine()

	output.PrintSection("按级别统计")
	levelMap := stats.GetLevelCountMap()
	output.PrintKeyValue("ERROR", levelMap["error"])
	output.PrintKeyValue("WARNING", levelMap["warning"])
	output.PrintKeyValue("INFO", levelMap["info"])
	output.PrintKeyValue("DEBUG", levelMap["debug"])
	output.PrintEmptyLine()

	if len(stats.TopErrors) > 0 {
		output.PrintSection("常见错误 (Top 10)")
		for i, err := range stats.TopErrors {
			output.Printf("  %d. %s (%d 次)\n", i+1, err.Message, err.Count)
		}
		output.PrintEmptyLine()
	}
}

// newLogsSearchCmd 创建日志搜索命令
func newLogsSearchCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "search",
		Short: "搜索日志",
		Long:  `在收集的日志中搜索关键词或正则表达式。`,
		Example: `  mihomo-cli logs search --keyword "proxy"
  mihomo-cli logs search --regex "error"
  mihomo-cli logs search --keyword "error" --level error
  mihomo-cli logs search --keyword "warning" --duration 10m`,
		RunE: runLogsSearch,
	}

	cmd.Flags().StringSliceVar(&logKeyword, "keyword", nil, "搜索关键词（可多次使用，AND 逻辑）")
	cmd.Flags().StringVar(&logRegex, "regex", "", "正则表达式搜索")
	cmd.Flags().StringVar(&logLevel, "level", "", "日志级别过滤")
	cmd.Flags().StringVar(&logDuration, "duration", "1m", "收集日志的时间范围（如 30s, 5m, 1h）")
	cmd.Flags().StringVarP(&searchFormat, "output", "o", "table", "输出格式 (table/json)")

	return cmd
}

// runLogsSearch 执行日志搜索命令
func runLogsSearch(cmd *cobra.Command, args []string) error {
	// 检查搜索条件
	if len(logKeyword) == 0 && logRegex == "" {
		return pkgerrors.ErrInvalidArg("请指定搜索关键词或正则表达式", nil)
	}

	// 解析时间范围
	duration, err := time.ParseDuration(logDuration)
	if err != nil {
		return pkgerrors.WrapError("无效的时间范围格式", err)
	}

	// 创建 API 客户端
	client := api.NewClientWithTimeout(
		viper.GetString("api.address"),
		viper.GetString("api.secret"),
		viper.GetInt("api.timeout"),
	)

	// 创建日志收集器
	collector := log.NewLogCollector(log.DefaultMaxLogs)

	// 收集日志
	ctx := cmd.Context()
	output.Info(fmt.Sprintf("正在收集最近 %s 的日志...", logDuration))
	if err := collector.CollectWithDuration(ctx, client, duration); err != nil {
		return pkgerrors.WrapError("收集日志失败", err)
	}

	logs := collector.GetLogs()
	if len(logs) == 0 {
		output.Info("未收集到日志")
		return nil
	}

	// 创建搜索查询
	query := log.NewSearchQuery()
	if len(logKeyword) > 0 {
		query.WithKeywords(logKeyword...)
	}
	if logRegex != "" {
		if _, err := query.WithRegex(logRegex); err != nil {
			return pkgerrors.WrapError("无效的正则表达式", err)
		}
	}
	if logLevel != "" {
		query.WithLevel(logLevel)
	}

	// 执行搜索
	searcher := log.NewLogSearcher()
	result := searcher.Search(logs, query)

	// 根据输出格式显示结果
	if searchFormat == "json" {
		output.PrintJSON(result)
	} else {
		printSearchResults(result)
	}

	return nil
}

// printSearchResults 打印搜索结果
func printSearchResults(result *log.SearchResult) {
	output.PrintEmptyLine()
	if result.Total == 0 {
		output.Info("未找到匹配的日志")
		return
	}

	output.Printf("找到 %d 条匹配的日志:\n", result.Total)
	output.PrintEmptyLine()

	for i, logInfo := range result.Matches {
		output.Printf("[%d] ", i+1)
		log.PrintLogMessage(logInfo)
	}
	output.PrintEmptyLine()
}

// newLogsExportCmd 创建日志导出命令
func newLogsExportCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "export",
		Short: "导出日志到文件",
		Long:  `将收集的日志导出为 JSON、TXT 或 CSV 文件。`,
		Example: `  mihomo-cli logs export -o logs.json --format json
  mihomo-cli logs export -o logs.csv --format csv
  mihomo-cli logs export -o errors.json --level error
  mihomo-cli logs export -o recent.json --duration 1h`,
		RunE: runLogsExport,
	}

	cmd.Flags().StringVarP(&exportFilename, "output", "o", "", "输出文件路径（必需）")
	cmd.Flags().StringVar(&exportFormat, "format", "json", "导出格式 (json/txt/csv)")
	cmd.Flags().StringVar(&logLevel, "level", "", "日志级别过滤")
	cmd.Flags().StringVar(&logDuration, "duration", "1m", "收集日志的时间范围（如 30s, 5m, 1h）")

	cmd.MarkFlagRequired("output")

	return cmd
}

// runLogsExport 执行日志导出命令
func runLogsExport(cmd *cobra.Command, args []string) error {
	// 验证导出格式
	format := log.ExportFormat(exportFormat)
	if format != log.FormatJSON && format != log.FormatTXT && format != log.FormatCSV {
		return pkgerrors.ErrInvalidArg("不支持的导出格式，请使用 json/txt/csv", nil)
	}

	// 解析时间范围
	duration, err := time.ParseDuration(logDuration)
	if err != nil {
		return pkgerrors.WrapError("无效的时间范围格式", err)
	}

	// 创建 API 客户端
	client := api.NewClientWithTimeout(
		viper.GetString("api.address"),
		viper.GetString("api.secret"),
		viper.GetInt("api.timeout"),
	)

	// 创建日志收集器
	collector := log.NewLogCollector(log.DefaultMaxLogs)

	// 收集日志
	ctx := cmd.Context()
	output.Info(fmt.Sprintf("正在收集最近 %s 的日志...", logDuration))
	if err := collector.CollectWithDuration(ctx, client, duration); err != nil {
		return pkgerrors.WrapError("收集日志失败", err)
	}

	logs := collector.GetLogs()
	if len(logs) == 0 {
		output.Info("未收集到日志")
		return nil
	}

	// 创建过滤器（如果有）
	var filter *log.LogFilter
	if logLevel != "" {
		filter = log.NewLogFilter().WithLevel(logLevel)
	}

	// 导出日志
	output.Info(fmt.Sprintf("正在导出日志到 %s...", exportFilename))
	if err := log.ExportLogsToFile(logs, exportFilename, format, filter); err != nil {
		return pkgerrors.WrapError("导出日志失败", err)
	}

	output.Success(fmt.Sprintf("成功导出 %d 条日志到 %s", len(logs), exportFilename))
	return nil
}
