package connection

import (
	"fmt"

	"github.com/kkkqkx123/mihomo-cli/internal/output"
	"github.com/kkkqkx123/mihomo-cli/pkg/types"

	"github.com/olekukonko/tablewriter"
	"github.com/olekukonko/tablewriter/tw"
)

// FormatConnectionList 格式化连接列表输出
func FormatConnectionList(connResp *types.ConnectionsResponse, outputFormat string) error {
	if outputFormat == "json" {
		return formatConnectionJSON(connResp)
	}
	return formatConnectionTable(connResp)
}

// formatConnectionJSON 以 JSON 格式输出连接列表
func formatConnectionJSON(connResp *types.ConnectionsResponse) error {
	return output.PrintJSON(connResp)
}

// formatConnectionTable 以表格格式输出连接列表
func formatConnectionTable(connResp *types.ConnectionsResponse) error {
	// 显示总流量信息
	fmt.Fprintf(output.GetGlobalStdout(), "总连接数: %d\n", len(connResp.Connections))
	fmt.Fprintf(output.GetGlobalStdout(), "上传速度: %s/s\n", formatBytes(connResp.UploadSpeed))
	fmt.Fprintf(output.GetGlobalStdout(), "下载速度: %s/s\n\n", formatBytes(connResp.DownloadSpeed))

	if len(connResp.Connections) == 0 {
		output.Info("当前没有活跃连接")
		return nil
	}

	// 创建表格
	table := tablewriter.NewTable(output.GetGlobalStdout(),
		tablewriter.WithHeader([]string{"ID", "源地址", "目标地址", "上传", "下载", "代理", "规则"}),
		tablewriter.WithHeaderAutoFormat(tw.On),
		tablewriter.WithRowAlignment(tw.AlignLeft),
		tablewriter.WithRendition(tw.Rendition{Borders: tw.Border{Left: tw.Off, Right: tw.Off, Top: tw.Off, Bottom: tw.Off}}),
	)

	// 遍历连接并添加到表格
	for _, conn := range connResp.Connections {
		// 源地址
		source := formatAddress(conn.Metadata.SourceIP, conn.Metadata.SourcePort)
		// 目标地址
		destination := formatDestination(conn.Metadata)
		// 代理链
		proxy := formatChains(conn.Chains)
		// 规则
		rule := formatRule(conn.Rule, conn.RulePayload)

		if err := table.Append([]string{
			truncateID(conn.ID),
			source,
			destination,
			formatBytes(conn.Upload),
			formatBytes(conn.Download),
			proxy,
			rule,
		}); err != nil {
			return err
		}
	}

	return table.Render()
}

// formatAddress 格式化地址
func formatAddress(ip, port string) string {
	if port == "" {
		return ip
	}
	return fmt.Sprintf("%s:%s", ip, port)
}

// formatDestination 格式化目标地址
func formatDestination(meta types.Metadata) string {
	// 优先使用 Host
	if meta.Host != "" {
		if meta.DestinationPort != "" {
			return fmt.Sprintf("%s:%s", meta.Host, meta.DestinationPort)
		}
		return meta.Host
	}
	// 使用 IP
	return formatAddress(meta.DestinationIP, meta.DestinationPort)
}

// formatChains 格式化代理链
func formatChains(chains []string) string {
	if len(chains) == 0 {
		return "-"
	}
	// 只显示最后一个代理（实际使用的代理）
	return chains[len(chains)-1]
}

// formatRule 格式化规则
func formatRule(rule, payload string) string {
	if rule == "" {
		return "-"
	}
	if payload != "" {
		return fmt.Sprintf("%s(%s)", rule, payload)
	}
	return rule
}

// formatBytes 格式化字节数
func formatBytes(bytes int64) string {
	const (
		KB = 1024
		MB = KB * 1024
		GB = MB * 1024
	)

	switch {
	case bytes >= GB:
		return fmt.Sprintf("%.2fGB", float64(bytes)/float64(GB))
	case bytes >= MB:
		return fmt.Sprintf("%.2fMB", float64(bytes)/float64(MB))
	case bytes >= KB:
		return fmt.Sprintf("%.2fKB", float64(bytes)/float64(KB))
	default:
		return fmt.Sprintf("%dB", bytes)
	}
}

// truncateID 截断 ID 显示
func truncateID(id string) string {
	if len(id) > 12 {
		return id[:12] + "..."
	}
	return id
}

// FormatCloseResult 格式化关闭连接结果
func FormatCloseResult(id string, err error) error {
	if err != nil {
		return output.PrintError(fmt.Sprintf("关闭连接失败: %v", err))
	}

	output.Success("连接已关闭")
	fmt.Fprintf(output.GetGlobalStdout(), "  连接 ID: %s\n", id)
	return nil
}

// FormatCloseAllResult 格式化关闭所有连接结果
func FormatCloseAllResult(count int, err error) error {
	if err != nil {
		return output.PrintError(fmt.Sprintf("关闭所有连接失败: %v", err))
	}

	output.Success("所有连接已关闭")
	fmt.Fprintf(output.GetGlobalStdout(), "  关闭连接数: %d\n", count)
	return nil
}
