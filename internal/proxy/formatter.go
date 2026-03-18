package proxy

import (
	"fmt"

	"github.com/kkkqkx123/mihomo-cli/internal/output"
	"github.com/kkkqkx123/mihomo-cli/pkg/types"

	"github.com/fatih/color"
	"github.com/olekukonko/tablewriter"
	"github.com/olekukonko/tablewriter/tw"

	pkgerrors "github.com/kkkqkx123/mihomo-cli/pkg/errors"
)

// FormatProxyList 格式化代理列表输出
func FormatProxyList(proxies map[string]*types.ProxyInfo, groupFilter string, outputFormat string, filterOpts FilterOptions) error {
	// 应用过滤条件
	filteredProxies := FilterProxies(proxies, filterOpts)

	// 如果有组过滤，只显示指定的代理组
	if groupFilter != "" {
		if proxy, exists := filteredProxies[groupFilter]; exists {
			// 格式化单个代理组
			if outputFormat == "json" {
				return formatProxyJSON(map[string]*types.ProxyInfo{groupFilter: proxy})
			}
			return formatProxyTable(map[string]*types.ProxyInfo{groupFilter: proxy}, true)
		}
		return pkgerrors.ErrInvalidArg("proxy group '"+groupFilter+"' does not exist", nil)
	}

	// 显示所有代理
	if outputFormat == "json" {
		return formatProxyJSON(filteredProxies)
	}
	return formatProxyTable(filteredProxies, false)
}

// formatProxyJSON 以 JSON 格式输出代理列表
func formatProxyJSON(proxies map[string]*types.ProxyInfo) error {
	return output.PrintJSON(proxies)
}

// formatProxyTable 以表格格式输出代理列表
func formatProxyTable(proxies map[string]*types.ProxyInfo, showOnlyOneGroup bool) error {
	// 创建表格
	table := tablewriter.NewTable(output.GetGlobalStdout(),
		tablewriter.WithHeader([]string{"名称", "类型", "当前", "节点数", "延迟", "状态"}),
		tablewriter.WithHeaderAutoFormat(tw.On),
		tablewriter.WithRowAlignment(tw.AlignLeft),
		tablewriter.WithBorders(tw.Border{Left: tw.Off, Right: tw.Off, Top: tw.Off, Bottom: tw.Off}),
	)

	// 遍历代理并添加到表格
	for name, proxy := range proxies {
		// 判断是否是代理组（有 all 字段）
		if len(proxy.All) > 0 {
			// 这是一个代理组
			current := proxy.Now
			if current == "" {
				current = "-"
			}

			// 获取延迟
			delayStr := formatDelay(proxy.Delay)

			// 获取状态
			status := "✓"
			if !proxy.Alive {
				status = "✗"
				status = color.RedString(status)
			}

			table.Append([]string{
				name,
				proxy.Type,
				current,
				fmt.Sprintf("%d", len(proxy.All)),
				delayStr,
				status,
			})

			// 如果只显示一个代理组，显示所有节点
			if showOnlyOneGroup {
				// 添加缩进的节点列表
				for _, nodeName := range proxy.All {
					table.Append([]string{
						"  └ " + nodeName,
						"-",
						"",
						"",
						"",
						"",
					})
				}
			}
		} else {
			// 这是一个单独的代理节点
			delayStr := formatDelay(proxy.Delay)

			// 获取状态
			status := "✓"
			if !proxy.Alive {
				status = "✗"
				status = color.RedString(status)
			}

			table.Append([]string{
				name,
				proxy.Type,
				"-",
				"-",
				delayStr,
				status,
			})
		}
	}

	table.Render()
	return nil
}

// formatDelay 格式化延迟显示
func formatDelay(delay uint16) string {
	if delay == 0 {
		return "-"
	}
	return fmt.Sprintf("%dms", delay)
}

// FormatTestResults 格式化延迟测试结果
func FormatTestResults(results []types.DelayResult, outputFormat string) error {
	if outputFormat == "json" {
		return formatTestResultsJSON(results)
	}
	return formatTestResultsTable(results)
}

// formatTestResultsJSON 以 JSON 格式输出测试结果
func formatTestResultsJSON(results []types.DelayResult) error {
	return output.PrintJSON(results)
}

// formatTestResultsTable 以表格格式输出测试结果
func formatTestResultsTable(results []types.DelayResult) error {
	table := tablewriter.NewTable(output.GetGlobalStdout(),
		tablewriter.WithHeader([]string{"节点名称", "延迟", "耗时", "状态"}),
		tablewriter.WithHeaderAutoFormat(tw.On),
		tablewriter.WithRowAlignment(tw.AlignLeft),
		tablewriter.WithBorders(tw.Border{Left: tw.Off, Right: tw.Off, Top: tw.Off, Bottom: tw.Off}),
	)

	for _, result := range results {
		var delayStr string
		var timeStr string
		var status string

		if result.Error != nil {
			delayStr = "-"
			timeStr = fmt.Sprintf("%dms", result.Time)
			status = color.RedString(result.Status)
		} else if result.Delay == 0 {
			delayStr = "-"
			timeStr = fmt.Sprintf("%dms", result.Time)
			status = color.YellowString(result.Status)
		} else {
			delayStr = fmt.Sprintf("%dms", result.Delay)
			timeStr = fmt.Sprintf("%dms", result.Time)
			if result.Delay < 100 {
				status = color.GreenString(result.Status)
			} else if result.Delay < 300 {
				status = color.YellowString(result.Status)
			} else {
				status = color.RedString(result.Status)
			}
		}

		table.Append([]string{
			result.Name,
			delayStr,
			timeStr,
			status,
		})
	}

	table.Render()
	return nil
}

// FormatAutoSelectResult 格式化自动选择结果
func FormatAutoSelectResult(groupName, bestNode string, delay uint16, err error) error {
	if err != nil {
		return output.PrintError(fmt.Sprintf("自动选择失败: %v", err))
	}

	if bestNode == "" {
		output.Warning("代理组 '%s' 中没有可用的节点", groupName)
		return nil
	}

	output.Success("已自动切换到最快节点")
	fmt.Fprintf(output.GetGlobalStdout(), "  代理组: %s\n", groupName)
	fmt.Fprintf(output.GetGlobalStdout(), "  节点: %s\n", bestNode)
	fmt.Fprintf(output.GetGlobalStdout(), "  延迟: %dms\n", delay)
	return nil
}

// FormatSwitchResult 格式化切换代理结果
func FormatSwitchResult(groupName, nodeName string, err error) error {
	if err != nil {
		return output.PrintError(fmt.Sprintf("切换代理失败: %v", err))
	}

	output.Success("代理切换成功")
	fmt.Fprintf(output.GetGlobalStdout(), "  代理组: %s\n", groupName)
	fmt.Fprintf(output.GetGlobalStdout(), "  节点: %s\n", nodeName)
	return nil
}

// FormatUnfixResult 格式化取消固定代理结果
func FormatUnfixResult(groupName string, err error) error {
	if err != nil {
		return output.PrintError(fmt.Sprintf("取消固定代理失败: %v", err))
	}

	output.Success("已取消固定代理，恢复自动选择")
	fmt.Fprintf(output.GetGlobalStdout(), "  代理组: %s\n", groupName)
	return nil
}

// GetGroupsFromProxies 从代理列表中提取所有代理组
func GetGroupsFromProxies(proxies map[string]*types.ProxyInfo) []string {
	var groups []string
	for name, proxy := range proxies {
		if len(proxy.All) > 0 {
			groups = append(groups, name)
		}
	}
	return groups
}

// FormatGroupList 格式化代理组列表
func FormatGroupList(proxies map[string]*types.ProxyInfo) error {
	groups := GetGroupsFromProxies(proxies)

	if len(groups) == 0 {
		output.Info("没有找到代理组")
		return nil
	}

	fmt.Fprintf(output.GetGlobalStdout(), "可用的代理组:\n")
	for _, group := range groups {
		fmt.Fprintf(output.GetGlobalStdout(), "  - %s\n", group)
	}
	return nil
}
