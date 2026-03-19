package cmd

import (
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/kkkqkx123/mihomo-cli/internal/api"
	"github.com/kkkqkx123/mihomo-cli/internal/errors"
	"github.com/kkkqkx123/mihomo-cli/internal/output"
)

// NewCacheCmd 创建缓存管理命令
func NewCacheCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "cache",
		Short: "管理缓存",
		Long:  `管理缓存，包括清空 FakeIP 池和 DNS 缓存。`,
	}

	cmd.AddCommand(newCacheClearCmd())

	return cmd
}

// newCacheClearCmd 创建清空缓存命令
func newCacheClearCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "clear",
		Short: "清空缓存",
		Long:  `清空指定类型的缓存（fakeip 或 dns）。`,
	}

	cmd.AddCommand(newCacheClearFakeIPCmd())
	cmd.AddCommand(newCacheClearDNSCmd())

	return cmd
}

// newCacheClearFakeIPCmd 创建清空 FakeIP 缓存命令
func newCacheClearFakeIPCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "fakeip",
		Short:   "清空 FakeIP 池",
		Long:    `清空 FakeIP 地址池。`,
		Example: `  mihomo-cli cache clear fakeip`,
		Args:    cobra.NoArgs,
		RunE:    runCacheClearFakeIP,
	}

	return cmd
}

// runCacheClearFakeIP 执行清空 FakeIP 命令
func runCacheClearFakeIP(cmd *cobra.Command, args []string) error {
	// 创建 API 客户端
	client := api.NewClientWithTimeout(
		viper.GetString("api.address"),
		viper.GetString("api.secret"),
		viper.GetInt("api.timeout"),
	)

	// 清空 FakeIP 池
	err := client.FlushFakeIP(cmd.Context())
	if err != nil {
		return errors.WrapAPIError("清空 FakeIP 池失败", err)
	}

	// 显示成功信息
	if outputFmt == "json" {
		output.Success("操作成功", map[string]interface{}{
			"message": "FakeIP 池已清空",
			"action":  "cache_clear_fakeip",
		})
	} else {
		output.Println("✓ FakeIP 池已清空")
	}

	return nil
}

// newCacheClearDNSCmd 创建清空 DNS 缓存命令
func newCacheClearDNSCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "dns",
		Short:   "清空 DNS 缓存",
		Long:    `清空 DNS 缓存。`,
		Example: `  mihomo-cli cache clear dns`,
		Args:    cobra.NoArgs,
		RunE:    runCacheClearDNS,
	}

	return cmd
}

// runCacheClearDNS 执行清空 DNS 缓存命令
func runCacheClearDNS(cmd *cobra.Command, args []string) error {
	// 创建 API 客户端
	client := api.NewClientWithTimeout(
		viper.GetString("api.address"),
		viper.GetString("api.secret"),
		viper.GetInt("api.timeout"),
	)

	// 清空 DNS 缓存
	err := client.FlushDNS(cmd.Context())
	if err != nil {
		return errors.WrapAPIError("清空 DNS 缓存失败", err)
	}

	// 显示成功信息
	if outputFmt == "json" {
		output.Success("操作成功", map[string]interface{}{
			"message": "DNS 缓存已清空",
			"action":  "cache_clear_dns",
		})
	} else {
		output.Println("✓ DNS 缓存已清空")
	}

	return nil
}
