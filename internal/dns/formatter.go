package dns

import (
	"fmt"
	"strings"

	"github.com/kkkqkx123/mihomo-cli/internal/output"
	"github.com/kkkqkx123/mihomo-cli/internal/systemdns"
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

// FormatSystemDNSConfig 格式化系统 DNS 配置输出
func FormatSystemDNSConfig(config *systemdns.SystemDNSConfig, outputFormat string) error {
	if outputFormat == "json" {
		return output.PrintJSON(config)
	}
	return formatSystemDNSConfigTable(config)
}

// formatSystemDNSConfigTable 以表格格式输出系统 DNS 配置
func formatSystemDNSConfigTable(config *systemdns.SystemDNSConfig) error {
	output.Info("系统 DNS 配置:")
	output.PrintKeyValue("配置来源", config.Source)

	if len(config.Nameservers) == 0 {
		output.Warning("未检测到 DNS 服务器")
	} else {
		output.PrintEmptyLine()
		output.Info("DNS 服务器:")
		for _, ns := range config.Nameservers {
			output.Printf("  - %s\n", ns)
		}
	}

	if len(config.SearchDomains) > 0 {
		output.PrintEmptyLine()
		output.Info("搜索域:")
		for _, domain := range config.SearchDomains {
			output.Printf("  - %s\n", domain)
		}
	}

	if len(config.Options) > 0 {
		output.PrintEmptyLine()
		output.Info("解析器选项:")
		for _, opt := range config.Options {
			output.Printf("  - %s\n", opt)
		}
	}

	return nil
}
