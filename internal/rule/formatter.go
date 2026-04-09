package rule

import (
	"fmt"

	"github.com/kkkqkx123/mihomo-cli/internal/output"
	"github.com/kkkqkx123/mihomo-cli/pkg/types"

	"github.com/fatih/color"
	"github.com/olekukonko/tablewriter"
	"github.com/olekukonko/tablewriter/tw"
)

// FormatRuleList 格式化规则列表输出
func FormatRuleList(rules []types.RuleInfo, outputFormat string) error {
	if len(rules) == 0 {
		output.Info("没有找到规则")
		return nil
	}

	if outputFormat == "json" {
		return formatRuleJSON(rules)
	}
	return formatRuleTable(rules)
}

// formatRuleJSON 以 JSON 格式输出规则列表
func formatRuleJSON(rules []types.RuleInfo) error {
	return output.PrintJSON(rules)
}

// formatRuleTable 以表格格式输出规则列表
func formatRuleTable(rules []types.RuleInfo) error {
	// 创建表格
	table := tablewriter.NewTable(output.GetGlobalStdout(),
		tablewriter.WithHeader([]string{"索引", "类型", "匹配内容", "代理", "命中次数"}),
		tablewriter.WithHeaderAutoFormat(tw.On),
		tablewriter.WithRowAlignment(tw.AlignLeft),
		tablewriter.WithRendition(tw.Rendition{Borders: tw.Border{Left: tw.Off, Right: tw.Off, Top: tw.Off, Bottom: tw.Off}}),
	)

	// 统计信息
	stats := calculateStats(rules)

	// 遍历规则并添加到表格
	for _, rule := range rules {
		// 截断过长的匹配内容
		payload := rule.Payload
		if len(payload) > 50 {
			payload = payload[:47] + "..."
		}

		// 获取命中次数：优先使用 Extra.HitCount，否则使用 Size（仅 GEOIP/GEOSITE 有效）
		hitCount := getHitCount(rule)

		if err := table.Append([]string{
			fmt.Sprintf("%d", rule.Index),
			rule.Type,
			payload,
			rule.Proxy,
			fmt.Sprintf("%d", hitCount),
		}); err != nil {
			return err
		}
	}

	if err := table.Render(); err != nil {
		return err
	}

	// 输出统计信息
	fmt.Fprintf(output.GetGlobalStdout(), "\n")
	output.Info("统计信息:")
	fmt.Fprintf(output.GetGlobalStdout(), "  总规则数: %d\n", stats.Total)
	fmt.Fprintf(output.GetGlobalStdout(), "  总命中次数: %d\n", stats.TotalHits)

	// 按类型统计
	if len(stats.ByType) > 0 {
		fmt.Fprintf(output.GetGlobalStdout(), "  按类型统计:\n")
		for ruleType, count := range stats.ByType {
			fmt.Fprintf(output.GetGlobalStdout(), "    - %s: %d\n", ruleType, count)
		}
	}

	return nil
}

// RuleStats 规则统计
type RuleStats struct {
	Total     int
	TotalHits int
	ByType    map[string]int
}

// calculateStats 计算规则统计信息
func calculateStats(rules []types.RuleInfo) *RuleStats {
	stats := &RuleStats{
		Total:  len(rules),
		ByType: make(map[string]int),
	}

	for _, rule := range rules {
		stats.TotalHits += int(getHitCount(rule))
		stats.ByType[rule.Type]++
	}

	return stats
}

// getHitCount 获取规则的命中次数
// 优先使用 Extra.HitCount（真实命中统计），否则使用 Size（仅 GEOIP/GEOSITE 有效）
func getHitCount(rule types.RuleInfo) uint64 {
	if rule.Extra != nil {
		return rule.Extra.HitCount
	}
	// Size 对于非 GEOIP/GEOSITE 规则是 -1，视为 0
	if rule.Size < 0 {
		return 0
	}
	return uint64(rule.Size)
}

// FormatDisableResult 格式化禁用规则结果
func FormatDisableResult(ruleIDs []int, err error) error {
	if err != nil {
		return output.PrintError(fmt.Sprintf("禁用规则失败: %v", err))
	}

	output.Success("规则禁用成功")
	if len(ruleIDs) == 1 {
		fmt.Fprintf(output.GetGlobalStdout(), "  规则索引: %d\n", ruleIDs[0])
	} else {
		fmt.Fprintf(output.GetGlobalStdout(), "  规则索引: %v\n", ruleIDs)
	}
	return nil
}

// FormatEnableResult 格式化启用规则结果
func FormatEnableResult(ruleIDs []int, err error) error {
	if err != nil {
		return output.PrintError(fmt.Sprintf("启用规则失败: %v", err))
	}

	output.Success("规则启用成功")
	if len(ruleIDs) == 1 {
		fmt.Fprintf(output.GetGlobalStdout(), "  规则索引: %d\n", ruleIDs[0])
	} else {
		fmt.Fprintf(output.GetGlobalStdout(), "  规则索引: %v\n", ruleIDs)
	}
	return nil
}

// ValidateRuleIndex 验证规则索引是否有效
func ValidateRuleIndex(index int, totalRules int) error {
	if index < 0 {
		return fmt.Errorf("规则索引不能为负数: %d", index)
	}
	if index >= totalRules {
		return fmt.Errorf("规则索引超出范围: %d (总规则数: %d)", index, totalRules)
	}
	return nil
}

// ValidateRuleIndices 验证多个规则索引是否有效
func ValidateRuleIndices(indices []int, totalRules int) error {
	if len(indices) == 0 {
		return fmt.Errorf("规则索引列表不能为空")
	}

	for _, index := range indices {
		if err := ValidateRuleIndex(index, totalRules); err != nil {
			return err
		}
	}
	return nil
}

// FormatValidationError 格式化验证错误
func FormatValidationError(err error) error {
	return output.PrintError(fmt.Sprintf("索引验证失败: %v", err))
}

// PrintRuleInfo 打印单个规则信息
func PrintRuleInfo(index int, rule types.RuleInfo) {
	hitCount := getHitCount(rule)
	fmt.Fprintf(output.GetGlobalStdout(), "规则详情:\n")
	fmt.Fprintf(output.GetGlobalStdout(), "  索引: %d\n", index)
	fmt.Fprintf(output.GetGlobalStdout(), "  类型: %s\n", rule.Type)
	fmt.Fprintf(output.GetGlobalStdout(), "  匹配内容: %s\n", rule.Payload)
	fmt.Fprintf(output.GetGlobalStdout(), "  代理: %s\n", rule.Proxy)
	fmt.Fprintf(output.GetGlobalStdout(), "  命中次数: %d\n", hitCount)
}

// FormatRuleListWithFilter 格式化规则列表输出（带过滤）
func FormatRuleListWithFilter(rules []types.RuleInfo, ruleType string, outputFormat string) error {
	if ruleType != "" {
		// 过滤指定类型的规则
		var filtered []types.RuleInfo
		for _, rule := range rules {
			if rule.Type == ruleType {
				filtered = append(filtered, rule)
			}
		}
		if len(filtered) == 0 {
			output.Warning("没有找到类型为 '%s' 的规则", ruleType)
			return nil
		}
		return FormatRuleList(filtered, outputFormat)
	}
	return FormatRuleList(rules, outputFormat)
}

// ColorStatus 根据状态返回带颜色的状态字符串
func ColorStatus(enabled bool) string {
	if enabled {
		return color.GreenString("启用")
	}
	return color.RedString("禁用")
}

// FormatRuleProviderList 格式化规则提供者列表输出
func FormatRuleProviderList(providers map[string]*types.RuleProviderInfo, outputFormat string) error {
	if len(providers) == 0 {
		output.Info("没有找到规则提供者")
		return nil
	}

	if outputFormat == "json" {
		return formatRuleProviderJSON(providers)
	}
	return formatRuleProviderTable(providers)
}

// formatRuleProviderJSON 以 JSON 格式输出规则提供者列表
func formatRuleProviderJSON(providers map[string]*types.RuleProviderInfo) error {
	return output.PrintJSON(providers)
}

// formatRuleProviderTable 以表格格式输出规则提供者列表
func formatRuleProviderTable(providers map[string]*types.RuleProviderInfo) error {
	// 创建表格
	table := tablewriter.NewTable(output.GetGlobalStdout(),
		tablewriter.WithHeader([]string{"名称", "类型", "行为", "格式", "来源类型", "规则数量", "更新时间"}),
		tablewriter.WithHeaderAutoFormat(tw.On),
		tablewriter.WithRowAlignment(tw.AlignLeft),
		tablewriter.WithRendition(tw.Rendition{Borders: tw.Border{Left: tw.Off, Right: tw.Off, Top: tw.Off, Bottom: tw.Off}}),
	)

	// 遍历提供者并添加到表格
	for name, provider := range providers {
		// 截断过长的更新时间
		updatedAt := provider.UpdatedAt
		if len(updatedAt) > 19 {
			updatedAt = updatedAt[:19]
		}

		if err := table.Append([]string{
			name,
			provider.Type,
			provider.Behavior,
			provider.Format,
			provider.VehicleType,
			fmt.Sprintf("%d", provider.RuleCount),
			updatedAt,
		}); err != nil {
			return err
		}
	}

	if err := table.Render(); err != nil {
		return err
	}

	// 输出统计信息
	fmt.Fprintf(output.GetGlobalStdout(), "\n")
	output.Info("统计信息:")
	fmt.Fprintf(output.GetGlobalStdout(), "  总提供者数: %d\n", len(providers))

	// 计算总规则数
	totalRules := 0
	for _, provider := range providers {
		totalRules += provider.RuleCount
	}
	fmt.Fprintf(output.GetGlobalStdout(), "  总规则数: %d\n", totalRules)

	return nil
}
