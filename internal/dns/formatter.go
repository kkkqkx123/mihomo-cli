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
		output.Printf("查询域名: %s\n", strings.TrimSuffix(question.Name, "."))
		output.Printf("记录类型: %s\n", types.DNSTypeToString(question.Type))
		output.Printf("响应状态: %s\n\n", formatDNSStatus(resp.Status))
	}

	// 显示 Answer 记录
	if len(resp.Answer) > 0 {
		output.Info("Answer 记录:")
		table := tablewriter.NewTable(output.GetGlobalStdout(),
			tablewriter.WithHeader([]string{"名称", "类型", "TTL", "数据"}),
			tablewriter.WithHeaderAutoFormat(tw.On),
			tablewriter.WithRowAlignment(tw.AlignLeft),
			tablewriter.WithRendition(tw.Rendition{Borders: tw.Border{Left: tw.Off, Right: tw.Off, Top: tw.Off, Bottom: tw.Off}}),
		)

		for _, answer := range resp.Answer {
			if err := table.Append([]string{
				strings.TrimSuffix(answer.Name, "."),
				types.DNSTypeToString(answer.Type),
				fmt.Sprintf("%d", answer.TTL),
				answer.Data,
			}); err != nil {
				return err
			}
		}
		if err := table.Render(); err != nil {
			return err
		}
		output.PrintEmptyLine()
	}

	// 显示 Authority 记录
	if len(resp.Authority) > 0 {
		output.Info("Authority 记录:")
		table := tablewriter.NewTable(output.GetGlobalStdout(),
			tablewriter.WithHeader([]string{"名称", "类型", "TTL", "数据"}),
			tablewriter.WithHeaderAutoFormat(tw.On),
			tablewriter.WithRowAlignment(tw.AlignLeft),
			tablewriter.WithRendition(tw.Rendition{Borders: tw.Border{Left: tw.Off, Right: tw.Off, Top: tw.Off, Bottom: tw.Off}}),
		)

		for _, answer := range resp.Authority {
			if err := table.Append([]string{
				strings.TrimSuffix(answer.Name, "."),
				types.DNSTypeToString(answer.Type),
				fmt.Sprintf("%d", answer.TTL),
				answer.Data,
			}); err != nil {
				return err
			}
		}
		if err := table.Render(); err != nil {
			return err
		}
		output.PrintEmptyLine()
	}

	// 显示 Additional 记录
	if len(resp.Additional) > 0 {
		output.Info("Additional 记录:")
		table := tablewriter.NewTable(output.GetGlobalStdout(),
			tablewriter.WithHeader([]string{"名称", "类型", "TTL", "数据"}),
			tablewriter.WithHeaderAutoFormat(tw.On),
			tablewriter.WithRowAlignment(tw.AlignLeft),
			tablewriter.WithRendition(tw.Rendition{Borders: tw.Border{Left: tw.Off, Right: tw.Off, Top: tw.Off, Bottom: tw.Off}}),
		)

		for _, answer := range resp.Additional {
			if err := table.Append([]string{
				strings.TrimSuffix(answer.Name, "."),
				types.DNSTypeToString(answer.Type),
				fmt.Sprintf("%d", answer.TTL),
				answer.Data,
			}); err != nil {
				return err
			}
		}
		if err := table.Render(); err != nil {
			return err
		}
		output.PrintEmptyLine()
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
	output.PrintKeyValue("启用状态", formatBool(config.Enable))
	output.PrintKeyValue("IPv6 支持", formatBool(config.IPv6))
	output.PrintKeyValue("增强模式", config.EnhancedMode)

	if config.Listen != "" {
		output.PrintKeyValue("监听地址", config.Listen)
	}

	if config.FakeIPRange != "" {
		output.PrintKeyValue("FakeIP 范围", config.FakeIPRange)
	}

	// Nameserver
	if len(config.Nameserver) > 0 {
		output.PrintEmptyLine()
		output.Info("Nameserver:")
		for _, ns := range config.Nameserver {
			output.Printf("  - %s\n", ns)
		}
	}

	// Fallback
	if len(config.Fallback) > 0 {
		output.PrintEmptyLine()
		output.Info("Fallback:")
		for _, fb := range config.Fallback {
			output.Printf("  - %s\n", fb)
		}
	}

	// Default Nameserver
	if len(config.DefaultNameserver) > 0 {
		output.PrintEmptyLine()
		output.Info("Default Nameserver:")
		for _, ns := range config.DefaultNameserver {
			output.Printf("  - %s\n", ns)
		}
	}

	// FakeIP Filter
	if len(config.FakeIPFilter) > 0 {
		output.PrintEmptyLine()
		output.Info("FakeIP 过滤规则:")
		for _, filter := range config.FakeIPFilter {
			output.Printf("  - %s\n", filter)
		}
	}

	// Fallback Filter
	if config.FallbackFilter.GeoIP || len(config.FallbackFilter.IPCIDR) > 0 || len(config.FallbackFilter.Domain) > 0 {
		output.PrintEmptyLine()
		output.Info("Fallback 过滤器:")
		if config.FallbackFilter.GeoIP {
			output.Printf("  GeoIP: 启用 (代码: %s)\n", config.FallbackFilter.GeoIPCode)
		}
		if len(config.FallbackFilter.IPCIDR) > 0 {
			output.Println("  IP CIDR:")
			for _, cidr := range config.FallbackFilter.IPCIDR {
				output.Printf("    - %s\n", cidr)
			}
		}
		if len(config.FallbackFilter.Domain) > 0 {
			output.Println("  Domain:")
			for _, domain := range config.FallbackFilter.Domain {
				output.Printf("    - %s\n", domain)
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
