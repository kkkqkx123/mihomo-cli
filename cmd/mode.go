package cmd

import (
	"fmt"

	"github.com/kkkqkx123/mihomo-cli/internal/api"
	"github.com/kkkqkx123/mihomo-cli/internal/errors"
	"github.com/kkkqkx123/mihomo-cli/internal/output"
	"github.com/kkkqkx123/mihomo-cli/pkg/types"
	pkgerrors "github.com/kkkqkx123/mihomo-cli/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// NewModeCmd 创建模式管理命令
func NewModeCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "mode",
		Short: "管理 Mihomo 运行模式",
		Long:  `管理 Mihomo 的运行模式，包括查询和切换模式。`,
	}

	cmd.AddCommand(newModeGetCmd())
	cmd.AddCommand(newModeSetCmd())

	return cmd
}

// newModeGetCmd 创建获取模式命令
func newModeGetCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "get",
		Short: "查询当前运行模式",
		Long:  `查询当前 Mihomo 的运行模式（rule/global/direct）。`,
		Example: `  mihomo-cli mode get
  mihomo-cli mode get -o json`,
		RunE: runModeGet,
	}
}

// runModeGet 执行获取模式命令
func runModeGet(cmd *cobra.Command, args []string) error {
	// 创建 API 客户端
	client := api.NewClientWithTimeout(
		viper.GetString("api.address"),
		viper.GetString("api.secret"),
		viper.GetInt("api.timeout"),
	)

	// 获取当前模式
	modeInfo, err := client.GetMode(cmd.Context())
	if err != nil {
		return errors.WrapAPIError("获取模式失败", err)
	}

	// 根据输出格式显示结果
	if outputFmt == "json" {
		// JSON 输出在 GetMode 中已经处理
		return nil
	}

	// 表格输出
	output.Printf("当前模式: %s\n", modeInfo.Mode)
	output.PrintEmptyLine()
	output.Println("可用模式:")
	output.Println("  - rule    规则模式：根据规则文件决定流量走向")
	output.Println("  - global  全局模式：所有流量通过代理")
	output.Println("  - direct  直连模式：所有流量不经过代理")

	return nil
}

// newModeSetCmd 创建设置模式命令
func newModeSetCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "set <mode>",
		Short: "设置运行模式",
		Long:  `设置 Mihomo 的运行模式（rule/global/direct）。`,
		Example: `  mihomo-cli mode set rule
  mihomo-cli mode set global
  mihomo-cli mode set direct`,
		Args: cobra.ExactArgs(1),
		RunE: runModeSet,
	}
}

// runModeSet 执行设置模式命令
func runModeSet(cmd *cobra.Command, args []string) error {
	modeStr := args[0]

	// 验证模式
	if !types.IsValidMode(modeStr) {
		return pkgerrors.ErrInvalidArg(fmt.Sprintf("无效的模式: %s", modeStr), nil)
	}

	// 创建 API 客户端
	client := api.NewClientWithTimeout(
		viper.GetString("api.address"),
		viper.GetString("api.secret"),
		viper.GetInt("api.timeout"),
	)

	// 设置模式
	err := client.SetMode(cmd.Context(), types.TunnelMode(modeStr))
	if err != nil {
		return errors.WrapAPIError("设置模式失败", err)
	}

	// 显示成功信息
	output.Success("已切换到 %s 模式", modeStr)

	return nil
}