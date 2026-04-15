package cmd

import (
	"context"
	"time"

	"github.com/schollz/progressbar/v3"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/kkkqkx123/mihomo-cli/internal/api"
	"github.com/kkkqkx123/mihomo-cli/internal/errors"
	"github.com/kkkqkx123/mihomo-cli/internal/output"
	"github.com/kkkqkx123/mihomo-cli/internal/proxy"
	"github.com/kkkqkx123/mihomo-cli/pkg/types"
)

var (
	testURL       string
	testTimeout   int
	concurrent    int
	showProgress  bool
	testDelay     bool
	waitTest      bool
	batchSize     int
	maxNodes      int
	sortBy        string
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
	cmd.AddCommand(newProxyCurrentCmd())

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
  mihomo-cli proxy list --test-delay
  mihomo-cli proxy list --test-delay --concurrent 20
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
	
	// 添加测速标志
	cmd.Flags().BoolVar(&testDelay, "test-delay", false, "测试所有节点的延迟")
	cmd.Flags().IntVar(&concurrent, "concurrent", 10, "测速并发数（默认 10）")
	cmd.Flags().BoolVar(&showProgress, "progress", false, "显示测速进度条")
	
	// 添加测速控制标志
	cmd.Flags().BoolVar(&waitTest, "wait", true, "等待测速完成（默认 true，与 --async 互斥）")
	cmd.Flags().IntVar(&batchSize, "batch-size", 100, "每批次测试的节点数（默认 100）")
	cmd.Flags().IntVar(&maxNodes, "max-nodes", 500, "最大测试节点数（默认 500）")
	
	// 添加排序标志
	cmd.Flags().StringVar(&sortBy, "sort", "name", "排序方式（name/delay）")

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
		SortBy:         sortBy,
	}

	// 如果需要测试延迟
	if testDelay {
		tester := proxy.NewDelayTester(client)
		
		// 设置测速参数
		if testURL != "" {
			tester.SetTestURL(testURL)
		} else if viper.IsSet("proxy.test_url") {
			tester.SetTestURL(viper.GetString("proxy.test_url"))
		}
		
		if testTimeout > 0 {
			tester.SetTimeout(testTimeout)
		} else if viper.IsSet("proxy.timeout") {
			tester.SetTimeout(viper.GetInt("proxy.timeout"))
		} else {
			tester.SetTimeout(10000) // 默认 10 秒
		}
		
		if concurrent > 0 {
			tester.SetConcurrent(concurrent)
		}
		
		// 收集需要测试的节点
		var nodeNames []string
		for name, proxyInfo := range proxies {
			// 只测试单独的代理节点，不测试代理组
			if len(proxyInfo.All) == 0 {
				// 排除逻辑节点
				if shouldIncludeProxyForTest(name, proxyInfo) {
					nodeNames = append(nodeNames, name)
				}
			}
		}
		
		// 限制最大节点数
		if len(nodeNames) > maxNodes {
			output.Warning("节点数量过多（%d 个），只测试前 %d 个节点", len(nodeNames), maxNodes)
			nodeNames = nodeNames[:maxNodes]
		}
		
		// 如果需要显示进度条
		if showProgress && len(nodeNames) > 0 {
			bar := progressbar.NewOptions(len(nodeNames),
				progressbar.OptionSetDescription("测速中"),
				progressbar.OptionShowCount(),
				progressbar.OptionShowIts(),
				progressbar.OptionClearOnFinish(),
			)
			
			tester.SetProgress(func(current, total int, nodeName string) {
				_ = bar.Set(current)
			})
		}
		
		// 分批测速
		var results []types.DelayResult
		if waitTest {
			// 等待测速完成
			results, err = testNodesInBatches(tester, nodeNames, batchSize, cmd.Context())
			if err != nil {
				return errors.WrapAPIError("failed to test delay", err)
			}
		} else {
			// 异步测速：启动后台 goroutine，立即返回
			output.Info("后台测速已启动，结果将不会显示在当前列表中")
			go func() {
				testNodesInBatches(tester, nodeNames, batchSize, context.Background())
			}()
			// 不等待结果，直接返回
			return proxy.FormatProxyList(proxies, groupFilter, outputFmt, filterOpts)
		}
		
		// 更新代理信息的延迟数据
		for _, result := range results {
			if result.Error == nil && result.Delay > 0 {
				if proxyInfo, exists := proxies[result.Name]; exists {
					proxyInfo.Delay = result.Delay
					proxyInfo.Alive = true
				}
			} else {
				if proxyInfo, exists := proxies[result.Name]; exists {
					proxyInfo.Alive = false
				}
			}
		}
		
		if showProgress && len(nodeNames) > 0 {
			output.Println()
		}
	}

	// 格式化输出
	return proxy.FormatProxyList(proxies, groupFilter, outputFmt, filterOpts)
}

// shouldIncludeProxyForTest 判断是否应该测试此代理
func shouldIncludeProxyForTest(name string, proxyInfo *types.ProxyInfo) bool {
	if proxyInfo == nil {
		return false
	}
	
	// 排除空类型节点
	if proxyInfo.Type == "" {
		return false
	}
	
	// 排除逻辑节点（逻辑节点无法测试延迟）
	logicalTypes := map[string]bool{
		"Direct":     true,
		"Reject":     true,
		"RejectDrop": true,
		"Pass":       true,
		"Compatible": true,
	}
	if logicalTypes[proxyInfo.Type] {
		return false
	}
	
	return true
}

// testNodesInBatches 分批测试节点延迟
func testNodesInBatches(tester *proxy.DelayTester, nodeNames []string, batchSize int, ctx context.Context) ([]types.DelayResult, error) {
	var allResults []types.DelayResult
	
	// 分批测试
	for i := 0; i < len(nodeNames); i += batchSize {
		end := i + batchSize
		if end > len(nodeNames) {
			end = len(nodeNames)
		}
		
		batch := nodeNames[i:end]
		results, err := tester.TestNodes(ctx, batch)
		if err != nil {
			return allResults, err
		}
		allResults = append(allResults, results...)
		
		// 批次间短暂延迟，避免连接过多
		if end < len(nodeNames) {
			time.Sleep(500 * time.Millisecond)
		}
	}
	
	return allResults, nil
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
	
	// 优先使用命令行参数，其次使用配置文件
	if testURL != "" {
		tester.SetTestURL(testURL)
	} else if viper.IsSet("proxy.test_url") {
		tester.SetTestURL(viper.GetString("proxy.test_url"))
	}
	
	if testTimeout > 0 {
		tester.SetTimeout(testTimeout)
	} else if viper.IsSet("proxy.timeout") {
		tester.SetTimeout(viper.GetInt("proxy.timeout"))
	} else {
		tester.SetTimeout(10000) // 默认10秒
	}
	
	if concurrent > 0 {
		tester.SetConcurrent(concurrent)
	} else if viper.IsSet("proxy.concurrent") {
		tester.SetConcurrent(viper.GetInt("proxy.concurrent"))
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
				_ = bar.Set(current)
			})
		}

		// 测试代理组中所有节点
		results, err = tester.TestGroup(cmd.Context(), groupName)
		if err != nil {
			return errors.WrapAPIError("failed to test delay", err)
		}

		// 完成进度条
		if showProgress && nodeCount > 0 {
			output.Println()
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
	
	// 优先使用命令行参数，其次使用配置文件
	if testURL != "" {
		tester.SetTestURL(testURL)
	} else if viper.IsSet("proxy.test_url") {
		tester.SetTestURL(viper.GetString("proxy.test_url"))
	}
	
	if testTimeout > 0 {
		tester.SetTimeout(testTimeout)
	} else if viper.IsSet("proxy.timeout") {
		tester.SetTimeout(viper.GetInt("proxy.timeout"))
	} else {
		tester.SetTimeout(10000) // 默认10秒
	}
	
	if concurrent > 0 {
		tester.SetConcurrent(concurrent)
	} else if viper.IsSet("proxy.concurrent") {
		tester.SetConcurrent(viper.GetInt("proxy.concurrent"))
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
			_ = bar.Set(current)
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
		output.Println()
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

// newProxyCurrentCmd 创建获取当前节点命令
func newProxyCurrentCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "current <group>",
		Short: "获取当前使用的节点",
		Long:  `获取指定代理组当前使用的节点信息。`,
		Example: `  mihomo-cli proxy current Proxy
  mihomo-cli proxy current Proxy --test-delay
  mihomo-cli proxy current Proxy --test-delay --timeout 10000`,
		Args: cobra.ExactArgs(1),
		RunE: runProxyCurrent,
	}

	// 添加测速标志
	cmd.Flags().BoolVar(&testDelay, "test-delay", false, "测试当前节点的延迟")
	cmd.Flags().IntVar(&testTimeout, "timeout", 5000, "超时时间（毫秒，默认 5000）")
	cmd.Flags().StringVar(&testURL, "url", "", "测试 URL（可选，默认使用配置中的 URL）")

	return cmd
}

// runProxyCurrent 执行获取当前节点命令
func runProxyCurrent(cmd *cobra.Command, args []string) error {
	groupName := args[0]

	// 创建 API 客户端
	client := api.NewClientWithTimeout(
		viper.GetString("api.address"),
		viper.GetString("api.secret"),
		viper.GetInt("api.timeout"),
	)

	// 获取代理组信息
	proxyGroup, err := client.GetProxy(cmd.Context(), groupName)
	if err != nil {
		return errors.WrapAPIError("failed to get proxy group info", err)
	}

	// 如果需要测试延迟
	if testDelay && proxyGroup.Now != "" {
		tester := proxy.NewDelayTester(client)
		
		// 设置测速参数
		if testURL != "" {
			tester.SetTestURL(testURL)
		} else if viper.IsSet("proxy.test_url") {
			tester.SetTestURL(viper.GetString("proxy.test_url"))
		}
		
		if testTimeout > 0 {
			tester.SetTimeout(testTimeout)
		} else if viper.IsSet("proxy.timeout") {
			tester.SetTimeout(viper.GetInt("proxy.timeout"))
		} else {
			tester.SetTimeout(10000) // 默认 10 秒
		}
		
		// 测试当前节点的延迟
		result := tester.TestSingle(cmd.Context(), proxyGroup.Now)
		
		// 更新代理信息
		if result.Error == nil && result.Delay > 0 {
			proxyGroup.Delay = result.Delay
			proxyGroup.Alive = true
		} else {
			proxyGroup.Alive = false
		}
	}

	// 格式化输出结果
	return proxy.FormatCurrentProxy(groupName, proxyGroup)
}