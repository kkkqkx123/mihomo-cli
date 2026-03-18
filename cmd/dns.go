package cmd

import (
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/kkkqkx123/mihomo-cli/internal/api"
	"github.com/kkkqkx123/mihomo-cli/internal/dns"
	"github.com/kkkqkx123/mihomo-cli/internal/errors"
)

var (
	dnsType string
)

// NewDNSCmd 创建 DNS 管理命令
func NewDNSCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "dns",
		Short: "DNS 查询",
		Long:  `执行 DNS 查询，支持多种记录类型。`,
	}

	cmd.AddCommand(newDNSQueryCmd())
	cmd.AddCommand(newDNSConfigCmd())

	return cmd
}

// newDNSQueryCmd 创建 DNS 查询命令
func newDNSQueryCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "query <domain>",
		Short:   "执行 DNS 查询",
		Long:    `执行 DNS 查询，支持 A、AAAA、CNAME、MX、TXT、NS、SRV 等记录类型。`,
		Example: `  mihomo-cli dns query example.com
  mihomo-cli dns query example.com --type AAAA
  mihomo-cli dns query example.com --type MX
  mihomo-cli dns query example.com --type TXT -o json`,
		Args: cobra.ExactArgs(1),
		RunE: runDNSQuery,
	}

	cmd.Flags().StringVar(&dnsType, "type", "A", "DNS 记录类型 (A, AAAA, CNAME, MX, TXT, NS, SRV 等)")

	return cmd
}

// runDNSQuery 执行 DNS 查询命令
func runDNSQuery(cmd *cobra.Command, args []string) error {
	domain := args[0]

	// 创建 API 客户端
	client := api.NewClientWithTimeout(
		viper.GetString("api.address"),
		viper.GetString("api.secret"),
		viper.GetInt("api.timeout"),
	)

	// 执行 DNS 查询
	resp, err := client.QueryDNS(cmd.Context(), domain, dnsType)
	if err != nil {
		return errors.WrapAPIError("DNS 查询失败", err)
	}

	// 格式化输出
	return dns.FormatDNSQueryResult(resp, outputFmt)
}

// newDNSConfigCmd 创建 DNS 配置命令
func newDNSConfigCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "显示 DNS 配置",
		Long:  `显示当前 DNS 配置信息。`,
		Example: `  mihomo-cli dns config
  mihomo-cli dns config -o json`,
		Args: cobra.NoArgs,
		RunE: runDNSConfig,
	}

	return cmd
}

// runDNSConfig 执行 DNS 配置命令
func runDNSConfig(cmd *cobra.Command, args []string) error {
	// 创建 API 客户端
	client := api.NewClientWithTimeout(
		viper.GetString("api.address"),
		viper.GetString("api.secret"),
		viper.GetInt("api.timeout"),
	)

	// 获取 DNS 配置
	config, err := client.GetDNSConfig(cmd.Context())
	if err != nil {
		return errors.WrapAPIError("获取 DNS 配置失败", err)
	}

	// 格式化输出
	return dns.FormatDNSConfig(config, outputFmt)
}
