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
	"github.com/kkkqkx123/mihomo-cli/internal/errors"
	"github.com/kkkqkx123/mihomo-cli/internal/log"
	"github.com/kkkqkx123/mihomo-cli/internal/output"
)

var (
	logFollow bool
)

// NewLogsCmd 创建日志命令
func NewLogsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "logs",
		Short: "查看 Mihomo 日志",
		Long:  `实时查看 Mihomo 内核的日志输出。`,
		Example: `  mihomo-cli logs
  mihomo-cli logs --follow`,
		RunE: runLogs,
	}

	cmd.Flags().BoolVarP(&logFollow, "follow", "f", true, "持续跟踪日志输出")

	return cmd
}

// runLogs 执行日志命令
func runLogs(cmd *cobra.Command, args []string) error {
	// 创建 API 客户端
	client := api.NewClientWithTimeout(
		viper.GetString("api.address"),
		viper.GetString("api.secret"),
		viper.GetInt("api.timeout"),
	)

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

	// 获取日志流
	stream, err := client.StreamLogs(ctx)
	if err != nil {
		return errors.WrapAPIError("获取日志流失败", err)
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
					return errors.WrapAPIError("读取日志流失败", err)
				}
				output.Info("日志流已关闭")
				return nil
			}
			log.PrintLogMessage(logMsg)
		case <-time.After(30 * time.Second):
			// 如果30秒没有收到日志，检查连接状态
			if err := stream.Err(); err != nil {
				return errors.WrapAPIError("日志流连接异常", err)
			}
		}
	}
}
