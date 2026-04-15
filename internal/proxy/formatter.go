package proxy

import (
	"fmt"
	"sort"

	"github.com/kkkqkx123/mihomo-cli/internal/output"
	"github.com/kkkqkx123/mihomo-cli/pkg/types"

	"github.com/olekukonko/tablewriter"
	"github.com/olekukonko/tablewriter/tw"

	pkgerrors "github.com/kkkqkx123/mihomo-cli/pkg/errors"
)

// 逻辑节点类型列表
var logicalTypes = map[string]bool{
	"Direct":     true,
	"Reject":     true,
	"RejectDrop": true,
	"Pass":       true,
	"Compatible": true,
}

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
			return formatProxyTableWithSort(map[string]*types.ProxyInfo{groupFilter: proxy}, true, filterOpts.SortBy)
		}
		return pkgerrors.ErrInvalidArg("proxy group '"+groupFilter+"' does not exist", nil)
	}

	// 显示所有代理
	if outputFormat == "json" {
		return formatProxyJSON(filteredProxies)
	}
	return formatProxyTableWithSort(filteredProxies, false, filterOpts.SortBy)
}

// formatProxyJSON 以 JSON 格式输出代理列表
func formatProxyJSON(proxies map[string]*types.ProxyInfo) error {
	return output.PrintJSON(proxies)
}

// formatProxyTableWithSort 以表格格式输出代理列表（带排序）
func formatProxyTableWithSort(proxies map[string]*types.ProxyInfo, showOnlyOneGroup bool, sortBy string) error {
	// 创建表格
	table := tablewriter.NewTable(output.GetGlobalStdout(),
		tablewriter.WithHeader([]string{"名称", "类型", "当前", "节点数", "延迟"}),
		tablewriter.WithHeaderAutoFormat(tw.On),
		tablewriter.WithRowAlignment(tw.AlignLeft),
		tablewriter.WithRendition(tw.Rendition{Borders: tw.Border{Left: tw.Off, Right: tw.Off, Top: tw.Off, Bottom: tw.Off}}),
	)

	// 分离代理组和节点，以及逻辑节点和实际节点
	var groups, nodes, logicalNodes []string
	
	for name, proxy := range proxies {
		if len(proxy.All) > 0 {
			// 代理组
			groups = append(groups, name)
		} else {
			// 单独节点
			if logicalTypes[proxy.Type] {
				logicalNodes = append(logicalNodes, name)
			} else {
				nodes = append(nodes, name)
			}
		}
	}
	
	// 排序
	if sortBy == "delay" {
		// 按延迟排序时，代理组和节点一起排序
		allNames := make([]string, 0, len(groups)+len(nodes))
		allNames = append(allNames, groups...)
		allNames = append(allNames, nodes...)
		allNames = SortProxies(proxies, sortBy)
		
		// 先显示实际节点和代理组（已排序）
		for _, name := range allNames {
			proxy := proxies[name]
			if len(proxy.All) > 0 {
				// 代理组
				current := proxy.Now
				if current == "" {
					current = "-"
				}
				delayStr := formatDelayWithColor(proxy.Delay, proxy.Alive)
				if err := table.Append([]string{
					name,
					proxy.Type,
					current,
					fmt.Sprintf("%d", len(proxy.All)),
					delayStr,
				}); err != nil {
					return err
				}
				
				// 如果只显示一个代理组，显示所有节点
				if showOnlyOneGroup {
					for _, nodeName := range proxy.All {
						if err := table.Append([]string{
							"  └ " + nodeName,
							"-",
							"",
							"",
							"",
						}); err != nil {
							return err
						}
					}
				}
			} else {
				// 单独节点
				delayStr := formatDelayWithColor(proxy.Delay, proxy.Alive)
				if err := table.Append([]string{
					name,
					proxy.Type,
					"-",
					"-",
					delayStr,
				}); err != nil {
					return err
				}
			}
		}
		
		// 最后显示逻辑节点
		if len(logicalNodes) > 0 {
			sort.Strings(logicalNodes)
			output.Println()
			output.Info("逻辑节点")
			for _, name := range logicalNodes {
				proxy := proxies[name]
				delayStr := formatDelayWithColor(proxy.Delay, proxy.Alive)
				if err := table.Append([]string{
					name,
					proxy.Type,
					"-",
					"-",
					delayStr,
				}); err != nil {
					return err
				}
			}
		}
	} else {
		// 按名称排序，分开显示
		// 1. 先显示代理组
		sort.Strings(groups)
		for _, name := range groups {
			proxy := proxies[name]
			current := proxy.Now
			if current == "" {
				current = "-"
			}
			delayStr := formatDelayWithColor(proxy.Delay, proxy.Alive)
			if err := table.Append([]string{
				name,
				proxy.Type,
				current,
				fmt.Sprintf("%d", len(proxy.All)),
				delayStr,
			}); err != nil {
				return err
			}
			
			// 如果只显示一个代理组，显示所有节点
			if showOnlyOneGroup {
				for _, nodeName := range proxy.All {
					if err := table.Append([]string{
						"  └ " + nodeName,
						"-",
						"",
						"",
						"",
					}); err != nil {
						return err
					}
				}
			}
		}
		
		// 2. 显示实际节点
		sort.Strings(nodes)
		for _, name := range nodes {
			proxy := proxies[name]
			delayStr := formatDelayWithColor(proxy.Delay, proxy.Alive)
			if err := table.Append([]string{
				name,
				proxy.Type,
				"-",
				"-",
				delayStr,
			}); err != nil {
				return err
			}
		}
		
		// 3. 最后显示逻辑节点
		if len(logicalNodes) > 0 {
			sort.Strings(logicalNodes)
			output.Println()
			output.Info("逻辑节点")
			for _, name := range logicalNodes {
				proxy := proxies[name]
				delayStr := formatDelayWithColor(proxy.Delay, proxy.Alive)
				if err := table.Append([]string{
					name,
					proxy.Type,
					"-",
					"-",
					delayStr,
				}); err != nil {
					return err
				}
			}
		}
	}

	return table.Render()
}

// formatDelay 格式化延迟显示
func formatDelay(delay uint16) string {
	if delay == 0 {
		return "-"
	}
	return fmt.Sprintf("%dms", delay)
}

// formatDelayWithColor 格式化延迟显示（带颜色）
func formatDelayWithColor(delay uint16, alive bool) string {
	if !alive {
		return output.RedString("超时")
	}
	if delay == 0 {
		return output.GrayString("未测试")
	}
	if delay < 100 {
		return output.GreenString(fmt.Sprintf("%dms", delay))
	} else if delay < 300 {
		return output.YellowString(fmt.Sprintf("%dms", delay))
	}
	return output.RedString(fmt.Sprintf("%dms", delay))
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
		tablewriter.WithRendition(tw.Rendition{Borders: tw.Border{Left: tw.Off, Right: tw.Off, Top: tw.Off, Bottom: tw.Off}}),
	)

	for _, result := range results {
		var delayStr string
		var timeStr string
		var status string

		if result.Error != nil {
			delayStr = "-"
			timeStr = fmt.Sprintf("%dms", result.Time)
			status = output.RedString(result.Status)
		} else if result.Delay == 0 {
			delayStr = "-"
			timeStr = fmt.Sprintf("%dms", result.Time)
			status = output.YellowString(result.Status)
		} else {
			delayStr = fmt.Sprintf("%dms", result.Delay)
			timeStr = fmt.Sprintf("%dms", result.Time)
			if result.Delay < 100 {
				status = output.GreenString(result.Status)
			} else if result.Delay < 300 {
				status = output.YellowString(result.Status)
			} else {
				status = output.RedString(result.Status)
			}
		}

		if err := table.Append([]string{
			result.Name,
			delayStr,
			timeStr,
			status,
		}); err != nil {
			return err
		}
	}

	return table.Render()
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

// FormatCurrentProxy 格式化当前节点信息
func FormatCurrentProxy(groupName string, proxy *types.ProxyInfo) error {
	if len(proxy.All) == 0 {
		return pkgerrors.ErrInvalidArg("'"+groupName+"' is not a proxy group", nil)
	}

	current := proxy.Now
	if current == "" {
		output.Warning("代理组 '%s' 当前没有选中任何节点", groupName)
		return nil
	}

	output.Success("当前节点信息")
	fmt.Fprintf(output.GetGlobalStdout(), "  代理组：%s\n", groupName)
	fmt.Fprintf(output.GetGlobalStdout(), "  当前节点：%s\n", current)
	fmt.Fprintf(output.GetGlobalStdout(), "  类型：%s\n", proxy.Type)
	
	// 显示延迟（带颜色）
	if proxy.Delay > 0 {
		delayStr := formatDelayWithColor(proxy.Delay, true)
		fmt.Fprintf(output.GetGlobalStdout(), "  延迟：%s\n", delayStr)
	} else if proxy.Alive {
		fmt.Fprintf(output.GetGlobalStdout(), "  延迟：%s\n", output.GrayString("未测试"))
	} else {
		fmt.Fprintf(output.GetGlobalStdout(), "  状态：%s\n", output.RedString("超时"))
	}
	
	return nil
}
