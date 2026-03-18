package cmd

import (
	"fmt"

	"github.com/schollz/progressbar/v3"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/kkkqkx123/mihomo-cli/internal/api"
	"github.com/kkkqkx123/mihomo-cli/internal/errors"
	"github.com/kkkqkx123/mihomo-cli/internal/proxy"
	"github.com/kkkqkx123/mihomo-cli/pkg/types"
)

var (
	testURL    string
	testTimeout int
	concurrent  int
	showProgress bool
	// 过滤参数
	filterType        string
	filterStatus      string
	excludePattern    string
	excludeLogical    bool
	groupsOnly        bool
	nodesOnly         bool
)

// NewProxyCmd 创建代理管理命令
func NewProxyCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "proxy",
		Short: "管理代理节点",
		Long:  `管理代理节点，包括列出、切换、测试和自动选择节点。`,
	}

	cmd.AddCommand(newProxyListCmd())
	cmd.AddCommand(newProxySwitchCmd())
	cmd.AddCommand(newProxyTestCmd())
	cmd.AddCommand(newProxyAutoCmd())
	cmd.AddCommand(newProxyUnfixCmd())

	return cmd
}

// newProxyListCmd 创建列出代理命令
func newProxyListCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list [group]",
		Short: "列出代理节点",
		Long:  `列出所有代理组的节点列表。如果指定代理组名称，只显示该代理组的节点。`,
		Example: `  mihomo-cli proxy list
  mihomo-cli proxy list Proxy
  mihomo-cli proxy list --type Vmess
  mihomo-cli proxy list --exclude-logical
  mihomo-cli proxy list --status alive
  mihomo-cli proxy list -o json`,
		Args: cobra.MaximumNArgs(1),
		RunE: runProxyList,
	}

	// 添加过滤标志
	cmd.Flags().StringVar(&filterType, "type", "", "按类型过滤（如 Vmess, Selector, URLTest 等）")
	cmd.Flags().StringVar(&filterStatus, "status", "", "按状态过滤（alive/dead）")
	cmd.Flags().StringVar(&excludePattern, "exclude", "", "排除名称匹配正则表达式的节点")
	cmd.Flags().BoolVar(&excludeLogical, "exclude-logical", false, "排除逻辑节点（DIRECT, REJECT 等）")
	cmd.Flags().BoolVar(&groupsOnly, "groups-only", false, "只显示代理组")
	cmd.Flags().BoolVar(&nodesOnly, "nodes-only", false, "只显示节点（排除代理组）")

	return cmd
}

// runProxyList 执行列出代理命令
func runProxyList(cmd *cobra.Command, args []string) error {
	// 创建 API 客户端
	client := api.NewClientWithTimeout(
		viper.GetString("api.address"),
		viper.GetString("api.secret"),
		viper.GetInt("api.timeout"),
	)

	// 获取所有代理
	proxies, err := client.ListProxies(cmd.Context())
	if err != nil {
		return errors.WrapAPIError("failed to get proxy list", err)
	}

	// 解析组过滤参数
	groupFilter := ""
	if len(args) > 0 {
		groupFilter = args[0]
	}

	// 构建过滤选项
	filterOpts := proxy.FilterOptions{
		Type:           filterType,
		Status:         filterStatus,
		ExcludeRegex:   excludePattern,
		ExcludeLogical: excludeLogical,
		GroupsOnly:     groupsOnly,
		NodesOnly:      nodesOnly,
	}

	// 格式化输出
	return proxy.FormatProxyList(proxies, groupFilter, outputFmt, filterOpts)
}

// newProxySwitchCmd 创建切换代理命令
func newProxySwitchCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "switch <group> <node>",
		Short: "切换代理节点",
		Long:  `切换指定代理组的选中节点。`,
		Example: `  mihomo-cli proxy switch Proxy Node1`,
		Args: cobra.ExactArgs(2),
		RunE: runProxySwitch,
	}

	return cmd
}

// runProxySwitch 执行切换代理命令
func runProxySwitch(cmd *cobra.Command, args []string) error {
	groupName := args[0]
	nodeName := args[1]

	// 创建 API 客户端
	client := api.NewClientWithTimeout(
		viper.GetString("api.address"),
		viper.GetString("api.secret"),
		viper.GetInt("api.timeout"),
	)

	// 切换代理
	err := client.SwitchProxy(cmd.Context(), groupName, nodeName)
	if err != nil {
		return errors.WrapAPIError("failed to switch proxy", err)
	}

	// 格式化输出结果
	return proxy.FormatSwitchResult(groupName, nodeName, nil)
}

// newProxyTestCmd 创建测试延迟命令
func newProxyTestCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "test <group> [node]",
		Short: "测试节点延迟",
		Long:  `测试指定代理组或节点的延迟。如果只指定代理组，测试该组内所有节点的延迟。`,
		Example: `  mihomo-cli proxy test Proxy
  mihomo-cli proxy test Proxy Node1
  mihomo-cli proxy test Proxy --url https://www.google.com/generate_204 --timeout 5000 --progress`,
		Args: cobra.RangeArgs(1, 2),
		RunE: runProxyTest,
	}

	cmd.Flags().StringVar(&testURL, "url", "", "测试 URL（可选，默认使用配置中的 URL）")
	cmd.Flags().IntVar(&testTimeout, "timeout", 5000, "超时时间（毫秒，默认 5000）")
	cmd.Flags().IntVar(&concurrent, "concurrent", 10, "并发测试数（默认 10）")
	cmd.Flags().BoolVar(&showProgress, "progress", false, "显示进度条")

	return cmd
}

// runProxyTest 执行测试延迟命令
func runProxyTest(cmd *cobra.Command, args []string) error {
	groupName := args[0]

	// 创建 API 客户端
	client := api.NewClientWithTimeout(
		viper.GetString("api.address"),
		viper.GetString("api.secret"),
		viper.GetInt("api.timeout"),
	)

	// 创建延迟测试器
	tester := proxy.NewDelayTester(client)
	if testURL != "" {
		tester.SetTestURL(testURL)
	}
	if testTimeout > 0 {
		tester.SetTimeout(testTimeout)
	}
	if concurrent > 0 {
		tester.SetConcurrent(concurrent)
	}

	var results []types.DelayResult

	// 如果指定了节点名称，测试单个节点
	if len(args) == 2 {
		nodeName := args[1]
		result := tester.TestSingle(cmd.Context(), nodeName)
		results = []types.DelayResult{result}
	} else {
		// 获取代理组信息以确定节点数量
		proxyGroup, err := client.GetProxy(cmd.Context(), groupName)
		if err != nil {
			return errors.WrapAPIError("failed to get proxy group info", err)
		}

		nodeCount := len(proxyGroup.All)

		// 如果需要显示进度条
		if showProgress && nodeCount > 0 {
			bar := progressbar.NewOptions(nodeCount,
				progressbar.OptionSetDescription("测速中"),
				progressbar.OptionShowCount(),
				progressbar.OptionShowIts(),
				progressbar.OptionClearOnFinish(),
			)

			// 设置进度回调
			tester.SetProgress(func(current, total int, nodeName string) {
				bar.Set(current)
			})
		}

		// 测试代理组中所有节点
		results, err = tester.TestGroup(cmd.Context(), groupName)
		if err != nil {
			return errors.WrapAPIError("failed to test delay", err)
		}

		// 完成进度条
		if showProgress && nodeCount > 0 {
			fmt.Println()
		}
	}

	// 格式化输出结果
	return proxy.FormatTestResults(results, outputFmt)
}

// newProxyAutoCmd 创建自动选择命令
func newProxyAutoCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "auto <group>",
		Short: "自动选择最快节点",
		Long:  `测试代理组中所有节点的延迟，并自动切换到延迟最低的节点。`,
		Example: `  mihomo-cli proxy auto Proxy
  mihomo-cli proxy auto Proxy --url https://www.google.com/generate_204 --timeout 5000 --progress`,
		Args: cobra.ExactArgs(1),
		RunE: runProxyAuto,
	}

	cmd.Flags().StringVar(&testURL, "url", "", "测试 URL（可选，默认使用配置中的 URL）")
	cmd.Flags().IntVar(&testTimeout, "timeout", 5000, "超时时间（毫秒，默认 5000）")
	cmd.Flags().IntVar(&concurrent, "concurrent", 10, "并发测试数（默认 10）")
	cmd.Flags().BoolVar(&showProgress, "progress", false, "显示进度条")

	return cmd
}

// runProxyAuto 执行自动选择命令
func runProxyAuto(cmd *cobra.Command, args []string) error {
	groupName := args[0]

	// 创建 API 客户端
	client := api.NewClientWithTimeout(
		viper.GetString("api.address"),
		viper.GetString("api.secret"),
		viper.GetInt("api.timeout"),
	)

	// 获取代理组信息以确定节点数量
	proxyGroup, err := client.GetProxy(cmd.Context(), groupName)
	if err != nil {
		return errors.WrapAPIError("failed to get proxy group info", err)
	}

	nodeCount := len(proxyGroup.All)

	// 创建延迟测试器
	tester := proxy.NewDelayTester(client)
	if testURL != "" {
		tester.SetTestURL(testURL)
	}
	if testTimeout > 0 {
		tester.SetTimeout(testTimeout)
	}
	if concurrent > 0 {
		tester.SetConcurrent(concurrent)
	}

	// 如果需要显示进度条
	if showProgress && nodeCount > 0 {
		bar := progressbar.NewOptions(nodeCount,
			progressbar.OptionSetDescription("测速中"),
			progressbar.OptionShowCount(),
			progressbar.OptionShowIts(),
			progressbar.OptionClearOnFinish(),
		)

		// 设置进度回调
		tester.SetProgress(func(current, total int, nodeName string) {
			bar.Set(current)
		})
	}

	// 创建节点选择器并配置测试器
	selector := proxy.NewSelector(client)
	selector.SetTester(tester)

	// 选择并切换到最快节点
	bestNode, delay, err := selector.SelectAndSwitch(cmd.Context(), groupName)
	if err != nil {
		return errors.WrapAPIError("failed to auto select", err)
	}

	// 完成进度条
	if showProgress && nodeCount > 0 {
		fmt.Println()
	}

	// 格式化输出结果
	return proxy.FormatAutoSelectResult(groupName, bestNode, delay, nil)
}

// newProxyUnfixCmd 创建取消固定代理命令
func newProxyUnfixCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "unfix <group>",
		Short: "取消固定代理",
		Long:  `取消代理组中固定的代理，恢复自动选择模式。`,
		Example: `  mihomo-cli proxy unfix Proxy`,
		Args: cobra.ExactArgs(1),
		RunE: runProxyUnfix,
	}

	return cmd
}

// runProxyUnfix 执行取消固定代理命令
func runProxyUnfix(cmd *cobra.Command, args []string) error {
	groupName := args[0]

	// 创建 API 客户端
	client := api.NewClientWithTimeout(
		viper.GetString("api.address"),
		viper.GetString("api.secret"),
		viper.GetInt("api.timeout"),
	)

	// 取消固定代理
	err := client.UnfixProxy(cmd.Context(), groupName)
	if err != nil {
		return errors.WrapAPIError("failed to unfix proxy", err)
	}

	// 格式化输出结果
	return proxy.FormatUnfixResult(groupName, nil)
}