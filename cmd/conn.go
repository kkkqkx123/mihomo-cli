package cmd

import (
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/kkkqkx123/mihomo-cli/internal/api"
	"github.com/kkkqkx123/mihomo-cli/internal/connection"
	"github.com/kkkqkx123/mihomo-cli/internal/errors"
)

// NewConnCmd 创建连接管理命令
func NewConnCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "conn",
		Short: "管理连接",
		Long:  `管理活跃连接，包括列出、关闭指定连接和关闭所有连接。`,
	}

	cmd.AddCommand(newConnListCmd())
	cmd.AddCommand(newConnCloseCmd())
	cmd.AddCommand(newConnCloseAllCmd())

	return cmd
}

// newConnListCmd 创建列出连接命令
func newConnListCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "列出活跃连接",
		Long:  `列出当前所有活跃的连接信息。`,
		Example: `  mihomo-cli conn list
  mihomo-cli conn list -o json`,
		Args: cobra.NoArgs,
		RunE: runConnList,
	}

	return cmd
}

// runConnList 执行列出连接命令
func runConnList(cmd *cobra.Command, args []string) error {
	// 创建 API 客户端
	client := api.NewClientWithTimeout(
		viper.GetString("api.address"),
		viper.GetString("api.secret"),
		viper.GetInt("api.timeout"),
	)

	// 获取连接列表
	connResp, err := client.GetConnections(cmd.Context())
	if err != nil {
		return errors.WrapAPIError("failed to get connection list", err)
	}

	// 格式化输出
	return connection.FormatConnectionList(connResp, outputFmt)
}

// newConnCloseCmd 创建关闭连接命令
func newConnCloseCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "close <id>",
		Short: "关闭指定连接",
		Long:  `关闭指定 ID 的连接。`,
		Example: `  mihomo-cli conn close abc123`,
		Args: cobra.ExactArgs(1),
		RunE: runConnClose,
	}

	return cmd
}

// runConnClose 执行关闭连接命令
func runConnClose(cmd *cobra.Command, args []string) error {
	connID := args[0]

	// 创建 API 客户端
	client := api.NewClientWithTimeout(
		viper.GetString("api.address"),
		viper.GetString("api.secret"),
		viper.GetInt("api.timeout"),
	)

	// 关闭连接
	err := client.CloseConnection(cmd.Context(), connID)
	if err != nil {
		return errors.WrapAPIError("failed to close connection", err)
	}

	// 格式化输出结果
	return connection.FormatCloseResult(connID, nil)
}

// newConnCloseAllCmd 创建关闭所有连接命令
func newConnCloseAllCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "close-all",
		Short: "关闭所有连接",
		Long:  `关闭所有活跃的连接。`,
		Example: `  mihomo-cli conn close-all`,
		Args: cobra.NoArgs,
		RunE: runConnCloseAll,
	}

	return cmd
}

// runConnCloseAll 执行关闭所有连接命令
func runConnCloseAll(cmd *cobra.Command, args []string) error {
	// 创建 API 客户端
	client := api.NewClientWithTimeout(
		viper.GetString("api.address"),
		viper.GetString("api.secret"),
		viper.GetInt("api.timeout"),
	)

	// 先获取连接数量
	connResp, err := client.GetConnections(cmd.Context())
	if err != nil {
		return errors.WrapAPIError("failed to get connection list", err)
	}

	count := len(connResp.Connections)

	// 关闭所有连接
	err = client.CloseAllConnections(cmd.Context())
	if err != nil {
		return errors.WrapAPIError("failed to close all connections", err)
	}

	// 格式化输出结果
	return connection.FormatCloseAllResult(count, nil)
}
