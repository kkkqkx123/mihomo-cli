package dns

import (
	"fmt"
	"strings"

	"github.com/kkkqkx123/mihomo-cli/internal/output"
	"github.com/kkkqkx123/mihomo-cli/pkg/types"

	"github.com/olekukonko/tablewriter"
	"github.com/olekukonko/tablewriter/tw"
)

// FormatDNSQueryResult 格式化 DNS 查询结果
func FormatDNSQueryResult(resp *types.DNSQueryResponse, outputFormat string) error {
	if outputFormat == "json" {
		return formatDNSJSON(resp)
	}
	return formatDNSTable(resp)
}

// formatDNSJSON 以 JSON 格式输出 DNS 查询结果
func formatDNSJSON(resp *types.DNSQueryResponse) error {
	return output.PrintJSON(resp)
}

// formatDNSTable 以表格格式输出 DNS 查询结果
func formatDNSTable(resp *types.DNSQueryResponse) error {
	// 显示查询信息
	if len(resp.Question) > 0 {
		question := resp.Question[0]
		fmt.Fprintf(output.GetGlobalStdout(), "查询域名: %s\n", strings.TrimSuffix(question.Name, "."))
		fmt.Fprintf(output.GetGlobalStdout(), "记录类型: %s\n", types.DNSTypeToString(question.Type))
		fmt.Fprintf(output.GetGlobalStdout(), "响应状态: %s\n\n", formatDNSStatus(resp.Status))
	}

	// 显示 Answer 记录
	if len(resp.Answer) > 0 {
		output.Info("Answer 记录:")
		table := tablewriter.NewTable(output.GetGlobalStdout(),
			tablewriter.WithHeader([]string{"名称", "类型", "TTL", "数据"}),
			tablewriter.WithHeaderAutoFormat(tw.On),
			tablewriter.WithRowAlignment(tw.AlignLeft),
			tablewriter.WithBorders(tw.Border{Left: tw.Off, Right: tw.Off, Top: tw.Off, Bottom: tw.Off}),
		)

		for _, answer := range resp.Answer {
			table.Append([]string{
				strings.TrimSuffix(answer.Name, "."),
				types.DNSTypeToString(answer.Type),
				fmt.Sprintf("%d", answer.TTL),
				answer.Data,
			})
		}
		table.Render()
		fmt.Fprintln(output.GetGlobalStdout())
	}

	// 显示 Authority 记录
	if len(resp.Authority) > 0 {
		output.Info("Authority 记录:")
		table := tablewriter.NewTable(output.GetGlobalStdout(),
			tablewriter.WithHeader([]string{"名称", "类型", "TTL", "数据"}),
			tablewriter.WithHeaderAutoFormat(tw.On),
			tablewriter.WithRowAlignment(tw.AlignLeft),
			tablewriter.WithBorders(tw.Border{Left: tw.Off, Right: tw.Off, Top: tw.Off, Bottom: tw.Off}),
		)

		for _, answer := range resp.Authority {
			table.Append([]string{
				strings.TrimSuffix(answer.Name, "."),
				types.DNSTypeToString(answer.Type),
				fmt.Sprintf("%d", answer.TTL),
				answer.Data,
			})
		}
		table.Render()
		fmt.Fprintln(output.GetGlobalStdout())
	}

	// 显示 Additional 记录
	if len(resp.Additional) > 0 {
		output.Info("Additional 记录:")
		table := tablewriter.NewTable(output.GetGlobalStdout(),
			tablewriter.WithHeader([]string{"名称", "类型", "TTL", "数据"}),
			tablewriter.WithHeaderAutoFormat(tw.On),
			tablewriter.WithRowAlignment(tw.AlignLeft),
			tablewriter.WithBorders(tw.Border{Left: tw.Off, Right: tw.Off, Top: tw.Off, Bottom: tw.Off}),
		)

		for _, answer := range resp.Additional {
			table.Append([]string{
				strings.TrimSuffix(answer.Name, "."),
				types.DNSTypeToString(answer.Type),
				fmt.Sprintf("%d", answer.TTL),
				answer.Data,
			})
		}
		table.Render()
		fmt.Fprintln(output.GetGlobalStdout())
	}

	// 如果没有记录
	if len(resp.Answer) == 0 && len(resp.Authority) == 0 && len(resp.Additional) == 0 {
		output.Warning("未找到 DNS 记录")
	}

	return nil
}

// formatDNSStatus 格式化 DNS 响应状态码
func formatDNSStatus(status int) string {
	switch status {
	case 0:
		return "成功 (NOERROR)"
	case 1:
		return "格式错误 (FORMERR)"
	case 2:
		return "服务器失败 (SERVFAIL)"
	case 3:
		return "域名不存在 (NXDOMAIN)"
	case 4:
		return "不支持 (NOTIMP)"
	case 5:
		return "拒绝 (REFUSED)"
	default:
		return fmt.Sprintf("未知状态 (%d)", status)
	}
}

// FormatDNSConfig 格式化 DNS 配置输出
func FormatDNSConfig(config *types.DNSConfig, outputFormat string) error {
	if outputFormat == "json" {
		return formatDNSConfigJSON(config)
	}
	return formatDNSConfigTable(config)
}

// formatDNSConfigJSON 以 JSON 格式输出 DNS 配置
func formatDNSConfigJSON(config *types.DNSConfig) error {
	return output.PrintJSON(config)
}

// formatDNSConfigTable 以表格格式输出 DNS 配置
func formatDNSConfigTable(config *types.DNSConfig) error {
	// 基本信息
	output.Info("DNS 配置:")
	fmt.Fprintf(output.GetGlobalStdout(), "  启用状态: %s\n", formatBool(config.Enable))
	fmt.Fprintf(output.GetGlobalStdout(), "  IPv6 支持: %s\n", formatBool(config.IPv6))
	fmt.Fprintf(output.GetGlobalStdout(), "  增强模式: %s\n", config.EnhancedMode)

	if config.Listen != "" {
		fmt.Fprintf(output.GetGlobalStdout(), "  监听地址: %s\n", config.Listen)
	}

	if config.FakeIPRange != "" {
		fmt.Fprintf(output.GetGlobalStdout(), "  FakeIP 范围: %s\n", config.FakeIPRange)
	}

	// Nameserver
	if len(config.Nameserver) > 0 {
		fmt.Fprintf(output.GetGlobalStdout(), "\n")
		output.Info("Nameserver:")
		for _, ns := range config.Nameserver {
			fmt.Fprintf(output.GetGlobalStdout(), "  - %s\n", ns)
		}
	}

	// Fallback
	if len(config.Fallback) > 0 {
		fmt.Fprintf(output.GetGlobalStdout(), "\n")
		output.Info("Fallback:")
		for _, fb := range config.Fallback {
			fmt.Fprintf(output.GetGlobalStdout(), "  - %s\n", fb)
		}
	}

	// Default Nameserver
	if len(config.DefaultNameserver) > 0 {
		fmt.Fprintf(output.GetGlobalStdout(), "\n")
		output.Info("Default Nameserver:")
		for _, ns := range config.DefaultNameserver {
			fmt.Fprintf(output.GetGlobalStdout(), "  - %s\n", ns)
		}
	}

	// FakeIP Filter
	if len(config.FakeIPFilter) > 0 {
		fmt.Fprintf(output.GetGlobalStdout(), "\n")
		output.Info("FakeIP 过滤规则:")
		for _, filter := range config.FakeIPFilter {
			fmt.Fprintf(output.GetGlobalStdout(), "  - %s\n", filter)
		}
	}

	// Fallback Filter
	if config.FallbackFilter.GeoIP || len(config.FallbackFilter.IPCIDR) > 0 || len(config.FallbackFilter.Domain) > 0 {
		fmt.Fprintf(output.GetGlobalStdout(), "\n")
		output.Info("Fallback 过滤器:")
		if config.FallbackFilter.GeoIP {
			fmt.Fprintf(output.GetGlobalStdout(), "  GeoIP: 启用 (代码: %s)\n", config.FallbackFilter.GeoIPCode)
		}
		if len(config.FallbackFilter.IPCIDR) > 0 {
			fmt.Fprintf(output.GetGlobalStdout(), "  IP CIDR:\n")
			for _, cidr := range config.FallbackFilter.IPCIDR {
				fmt.Fprintf(output.GetGlobalStdout(), "    - %s\n", cidr)
			}
		}
		if len(config.FallbackFilter.Domain) > 0 {
			fmt.Fprintf(output.GetGlobalStdout(), "  Domain:\n")
			for _, domain := range config.FallbackFilter.Domain {
				fmt.Fprintf(output.GetGlobalStdout(), "    - %s\n", domain)
			}
		}
	}

	return nil
}

// formatBool 格式化布尔值
func formatBool(b bool) string {
	if b {
		return "启用"
	}
	return "禁用"
}
