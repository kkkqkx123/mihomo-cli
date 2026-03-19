package cmd

import (
	"strconv"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/kkkqkx123/mihomo-cli/internal/api"
	"github.com/kkkqkx123/mihomo-cli/internal/errors"
	"github.com/kkkqkx123/mihomo-cli/internal/rule"
)

var (
	ruleType string
)

// NewRuleCmd 创建规则管理命令
func NewRuleCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "rule",
		Short: "管理规则",
		Long:  `管理规则，包括列出、禁用和启用规则。`,
	}

	cmd.AddCommand(newRuleListCmd())
	cmd.AddCommand(newRuleProviderCmd())
	cmd.AddCommand(newRuleDisableCmd())
	cmd.AddCommand(newRuleEnableCmd())

	return cmd
}

// newRuleListCmd 创建列出规则命令
func newRuleListCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "列出所有规则",
		Long:  `列出所有规则及其统计信息。`,
		Example: `  mihomo-cli rule list
  mihomo-cli rule list -o json
  mihomo-cli rule list --type DOMAIN`,
		Args: cobra.NoArgs,
		RunE: runRuleList,
	}

	cmd.Flags().StringVar(&ruleType, "type", "", "过滤规则类型（如 DOMAIN, IP-CIDR 等）")

	return cmd
}

// runRuleList 执行列出规则命令
func runRuleList(cmd *cobra.Command, args []string) error {
	// 创建 API 客户端
	client := api.NewClientWithTimeout(
		viper.GetString("api.address"),
		viper.GetString("api.secret"),
		viper.GetInt("api.timeout"),
	)

	// 获取所有规则
	rulesResp, err := client.GetRules(cmd.Context())
	if err != nil {
		return errors.WrapAPIError("获取规则列表失败", err)
	}

	// 格式化输出
	return rule.FormatRuleListWithFilter(rulesResp.Rules, ruleType, outputFmt)
}

// newRuleDisableCmd 创建禁用规则命令
func newRuleDisableCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "disable <index> [index...]",
		Short: "禁用指定规则",
		Long:  `禁用一个或多个规则。使用规则索引指定要禁用的规则。`,
		Example: `  mihomo-cli rule disable 0
  mihomo-cli rule disable 0 1 2
  mihomo-cli rule disable 0-5`,
		Args: cobra.MinimumNArgs(1),
		RunE: runRuleDisable,
	}

	return cmd
}

// runRuleDisable 执行禁用规则命令
func runRuleDisable(cmd *cobra.Command, args []string) error {
	// 创建 API 客户端
	client := api.NewClientWithTimeout(
		viper.GetString("api.address"),
		viper.GetString("api.secret"),
		viper.GetInt("api.timeout"),
	)

	// 获取所有规则以验证索引
	rulesResp, err := client.GetRules(cmd.Context())
	if err != nil {
		return errors.WrapAPIError("获取规则列表失败", err)
	}

	// 解析规则索引
	indices, err := parseRuleIndices(args, len(rulesResp.Rules))
	if err != nil {
		return err
	}

	// 验证索引
	if err := rule.ValidateRuleIndices(indices, len(rulesResp.Rules)); err != nil {
		return rule.FormatValidationError(err)
	}

	// 禁用规则
	err = client.DisableRules(cmd.Context(), indices)
	if err != nil {
		return errors.WrapAPIError("禁用规则失败", err)
	}

	// 格式化输出结果
	return rule.FormatDisableResult(indices, nil)
}

// newRuleEnableCmd 创建启用规则命令
func newRuleEnableCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "enable <index> [index...]",
		Short: "启用指定规则",
		Long:  `启用一个或多个规则。使用规则索引指定要启用的规则。`,
		Example: `  mihomo-cli rule enable 0
  mihomo-cli rule enable 0 1 2
  mihomo-cli rule enable 0-5`,
		Args: cobra.MinimumNArgs(1),
		RunE: runRuleEnable,
	}

	return cmd
}

// runRuleEnable 执行启用规则命令
func runRuleEnable(cmd *cobra.Command, args []string) error {
	// 创建 API 客户端
	client := api.NewClientWithTimeout(
		viper.GetString("api.address"),
		viper.GetString("api.secret"),
		viper.GetInt("api.timeout"),
	)

	// 获取所有规则以验证索引
	rulesResp, err := client.GetRules(cmd.Context())
	if err != nil {
		return errors.WrapAPIError("获取规则列表失败", err)
	}

	// 解析规则索引
	indices, err := parseRuleIndices(args, len(rulesResp.Rules))
	if err != nil {
		return err
	}

	// 验证索引
	if err := rule.ValidateRuleIndices(indices, len(rulesResp.Rules)); err != nil {
		return rule.FormatValidationError(err)
	}

	// 启用规则
	err = client.EnableRules(cmd.Context(), indices)
	if err != nil {
		return errors.WrapAPIError("启用规则失败", err)
	}

	// 格式化输出结果
	return rule.FormatEnableResult(indices, nil)
}

// parseRuleIndices 解析规则索引参数
// 支持格式: "0", "0 1 2", "0-5"
func parseRuleIndices(args []string, totalRules int) ([]int, error) {
	var indices []int
	seen := make(map[int]bool)

	for _, arg := range args {
		// 检查是否是范围格式 (如 "0-5")
		if strings.Contains(arg, "-") {
			parts := strings.Split(arg, "-")
			if len(parts) != 2 {
				return nil, errors.NewValidationError("无效的范围格式: %s", arg)
			}

			start, err := strconv.Atoi(parts[0])
			if err != nil {
				return nil, errors.NewValidationError("无效的起始索引: %s", parts[0])
			}

			end, err := strconv.Atoi(parts[1])
			if err != nil {
				return nil, errors.NewValidationError("无效的结束索引: %s", parts[1])
			}

			if start < 0 || end < 0 {
				return nil, errors.NewValidationError("索引不能为负数: start=%d, end=%d", start, end)
			}

			if start > end {
				return nil, errors.NewValidationError("起始索引不能大于结束索引: %d > %d", start, end)
			}

			if end >= totalRules {
				return nil, errors.NewValidationError("结束索引超出范围: %d >= %d (总规则数)", end, totalRules)
			}

			// 添加范围内的所有索引
			for i := start; i <= end; i++ {
				if !seen[i] {
					indices = append(indices, i)
					seen[i] = true
				}
			}
		} else {
			// 单个索引
			index, err := strconv.Atoi(arg)
			if err != nil {
				return nil, errors.NewValidationError("无效的索引: %s", arg)
			}

			if index < 0 {
				return nil, errors.NewValidationError("索引不能为负数: %d", index)
			}

			if index >= totalRules {
				return nil, errors.NewValidationError("索引超出范围: %d >= %d (总规则数)", index, totalRules)
			}

			if !seen[index] {
				indices = append(indices, index)
				seen[index] = true
			}
		}
	}

	return indices, nil
}

// newRuleProviderCmd 创建规则提供者命令
func newRuleProviderCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "provider",
		Short: "列出规则提供者",
		Long:  `列出所有规则提供者及其信息。`,
		Example: `  mihomo-cli rule provider
  mihomo-cli rule provider -o json`,
		Args: cobra.NoArgs,
		RunE: runRuleProvider,
	}

	return cmd
}

// runRuleProvider 执行列出规则提供者命令
func runRuleProvider(cmd *cobra.Command, args []string) error {
	// 创建 API 客户端
	client := api.NewClientWithTimeout(
		viper.GetString("api.address"),
		viper.GetString("api.secret"),
		viper.GetInt("api.timeout"),
	)

	// 获取所有规则提供者
	providers, err := client.ListRuleProviders(cmd.Context())
	if err != nil {
		return errors.WrapAPIError("获取规则提供者列表失败", err)
	}

	// 格式化输出
	return rule.FormatRuleProviderList(providers, outputFmt)
}
