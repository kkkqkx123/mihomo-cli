package cmd

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/kkkqkx123/mihomo-cli/internal/api"
	"github.com/kkkqkx123/mihomo-cli/internal/errors"
	"github.com/kkkqkx123/mihomo-cli/internal/monitor"
	"github.com/kkkqkx123/mihomo-cli/internal/output"
)

var (
	watchMode     bool
	watchInterval int
)

// NewMonitorCmd 创建监控管理命令
func NewMonitorCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "monitor",
		Short: "监控管理",
		Long:  `监控 Mihomo 的流量和内存使用情况。`,
	}

	cmd.AddCommand(newMonitorTrafficCmd())
	cmd.AddCommand(newMonitorMemoryCmd())

	return cmd
}

// newMonitorTrafficCmd 创建流量监控命令
func newMonitorTrafficCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "traffic",
		Short: "获取流量统计",
		Long:  `获取实时流量统计信息，包括上传速度、下载速度、总上传流量和总下载流量。`,
		Example: `  mihomo-cli monitor traffic
  mihomo-cli monitor traffic --watch
  mihomo-cli monitor traffic -o json`,
		RunE: runMonitorTraffic,
	}

	cmd.Flags().BoolVarP(&watchMode, "watch", "w", false, "持续刷新显示实时流量")
	cmd.Flags().IntVarP(&watchInterval, "interval", "i", 1, "刷新间隔（秒，仅用于 --watch 模式）")

	return cmd
}

// runMonitorTraffic 执行流量监控命令
func runMonitorTraffic(cmd *cobra.Command, args []string) error {
	// 创建 API 客户端
	client := api.NewClientWithTimeout(
		viper.GetString("api.address"),
		viper.GetString("api.secret"),
		viper.GetInt("api.timeout"),
	)

	// 创建格式化器
	formatter := monitor.NewTrafficFormatter(nil)

	if watchMode {
		// Watch 模式
		return runTrafficWatch(cmd.Context(), client, formatter)
	}

	// 单次查询模式
	traffic, err := client.GetTraffic(cmd.Context())
	if err != nil {
		return errors.WrapAPIError("获取流量统计失败", err)
	}

	if outputFmt == "json" {
		return formatter.FormatJSON(traffic)
	}

	return formatter.FormatOnce(traffic)
}

// runTrafficWatch 运行流量 Watch 模式
func runTrafficWatch(ctx context.Context, client *api.Client, formatter *monitor.TrafficFormatter) error {
	// 设置信号处理
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigChan
		cancel()
	}()

	// 显示头部
	formatter.FormatWatchHeader()

	// 尝试使用 WebSocket
	streamClient := monitor.NewStreamClient(client.GetBaseURL(), client.GetSecret())
	dataChan, err := streamClient.StreamTraffic(ctx)
	if err != nil {
		// WebSocket 失败，使用 HTTP 轮询
		output.Printf("WebSocket 连接失败，使用 HTTP 轮询模式 (间隔: %d秒)\n", watchInterval)
		dataChan = monitor.WatchTraffic(ctx, client.GetTraffic, time.Duration(watchInterval)*time.Second)
	} else {
		defer streamClient.Close()
	}

	// 累计流量
	var totalUp, totalDown int64
	lastTime := time.Now()

	// 读取数据并显示
	for {
		select {
		case <-ctx.Done():
			output.Println("\n监控已停止")
			return nil
		case data, ok := <-dataChan:
			if !ok {
				output.Println("\n数据流已关闭")
				return nil
			}

			// 计算累计流量
			now := time.Now()
			elapsed := now.Sub(lastTime).Seconds()
			if elapsed > 0 {
				totalUp += data.Up
				totalDown += data.Down
			}
			lastTime = now

			// 更新显示
			formatter.FormatWatchLine(data, totalUp, totalDown)
		}
	}
}

// newMonitorMemoryCmd 创建内存监控命令
func newMonitorMemoryCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "memory",
		Short: "获取内存使用",
		Long:  `获取当前内存使用情况。`,
		Example: `  mihomo-cli monitor memory
  mihomo-cli monitor memory --watch
  mihomo-cli monitor memory -o json`,
		RunE: runMonitorMemory,
	}

	cmd.Flags().BoolVarP(&watchMode, "watch", "w", false, "持续刷新显示内存使用")
	cmd.Flags().IntVarP(&watchInterval, "interval", "i", 1, "刷新间隔（秒，仅用于 --watch 模式）")

	return cmd
}

// runMonitorMemory 执行内存监控命令
func runMonitorMemory(cmd *cobra.Command, args []string) error {
	// 创建 API 客户端
	client := api.NewClientWithTimeout(
		viper.GetString("api.address"),
		viper.GetString("api.secret"),
		viper.GetInt("api.timeout"),
	)

	// 创建格式化器
	formatter := monitor.NewMemoryFormatter(nil)

	if watchMode {
		// Watch 模式
		return runMemoryWatch(cmd.Context(), client, formatter)
	}

	// 单次查询模式
	memory, err := client.GetMemory(cmd.Context())
	if err != nil {
		return errors.WrapAPIError("获取内存使用失败", err)
	}

	if outputFmt == "json" {
		return formatter.FormatJSON(memory)
	}

	return formatter.FormatOnce(memory)
}

// runMemoryWatch 运行内存 Watch 模式
func runMemoryWatch(ctx context.Context, client *api.Client, formatter *monitor.MemoryFormatter) error {
	// 设置信号处理
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigChan
		cancel()
	}()

	// 显示头部
	formatter.FormatWatchHeader()

	// 尝试使用 WebSocket
	streamClient := monitor.NewStreamClient(client.GetBaseURL(), client.GetSecret())
	dataChan, err := streamClient.StreamMemory(ctx)
	if err != nil {
		// WebSocket 失败，使用 HTTP 轮询
		output.Printf("WebSocket 连接失败，使用 HTTP 轮询模式 (间隔: %d秒)\n", watchInterval)
		dataChan = monitor.WatchMemory(ctx, client.GetMemory, time.Duration(watchInterval)*time.Second)
	} else {
		defer streamClient.Close()
	}

	// 读取数据并显示
	for {
		select {
		case <-ctx.Done():
			output.Println("\n监控已停止")
			return nil
		case data, ok := <-dataChan:
			if !ok {
				output.Println("\n数据流已关闭")
				return nil
			}

			// 更新显示
			formatter.FormatWatchLine(data)
		}
	}
}
