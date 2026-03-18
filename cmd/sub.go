package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/kkkqkx123/mihomo-cli/internal/api"
	"github.com/kkkqkx123/mihomo-cli/internal/errors"
)

// NewSubCmd 创建订阅管理命令
func NewSubCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "sub",
		Short: "管理代理订阅",
		Long:  `管理 Mihomo 的代理订阅，包括更新订阅等操作。`,
	}

	cmd.AddCommand(newSubUpdateCmd())

	return cmd
}

// newSubUpdateCmd 创建更新订阅命令
func newSubUpdateCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "update",
		Short: "更新代理订阅",
		Long:  `触发 Mihomo 更新所有代理提供者的订阅配置。`,
		Example: `  mihomo-cli sub update`,
		RunE: runSubUpdate,
	}
}

// runSubUpdate 执行更新订阅命令
func runSubUpdate(cmd *cobra.Command, args []string) error {
	// 创建 API 客户端
	client := api.NewClientWithTimeout(
		viper.GetString("api.address"),
		viper.GetString("api.secret"),
		viper.GetInt("api.timeout"),
	)

	// 获取所有代理提供者
	providers, err := client.ListProviders(cmd.Context())
	if err != nil {
		return errors.WrapAPIError("failed to get proxy provider list", err)
	}

	if len(providers) == 0 {
		fmt.Println("未找到任何代理提供者")
		return nil
	}

	fmt.Printf("找到 %d 个代理提供者，开始更新...\n", len(providers))
	fmt.Println()

	// 更新每个提供者
	successCount := 0
	failCount := 0
	for name, provider := range providers {
		fmt.Printf("正在更新 %s (%s)...\n", name, provider.VehicleType)
		
		err := client.UpdateProvider(cmd.Context(), name)
		if err != nil {
			fmt.Printf("  ✗ 更新失败: %v\n", err)
			failCount++
		} else {
			fmt.Printf("  ✓ 更新成功\n")
			successCount++
		}
	}

	// 显示汇总信息
	fmt.Println()
	fmt.Printf("更新完成: 成功 %d 个，失败 %d 个\n", successCount, failCount)

	return nil
}
