package cmd

import (
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/kkkqkx123/mihomo-cli/internal/config"
	"github.com/kkkqkx123/mihomo-cli/internal/history"
	"github.com/kkkqkx123/mihomo-cli/internal/output"
	pkgerrors "github.com/kkkqkx123/mihomo-cli/pkg/errors"
)

var (
	cfgFile   string
	outputFmt string
	apiURL    string
	secret    string
	timeout   int
)

// streamOutput 输出流管理器
var streamOutput *output.StreamOutput

// historyManager 历史记录管理器
var historyManager *history.Manager

var rootCmd = &cobra.Command{
	Use:   "mihomo-cli",
	Short: "Mihomo CLI 管理工具",
	Long: `Mihomo CLI 是一个非交互式的 Mihomo 代理核心管理工具，
通过命令行界面提供对 Mihomo RESTful API 的完整管理能力。

主要功能：
  - 模式管理：查询和切换运行模式
  - 代理管理：列出、切换、测试代理节点
  - 服务管理：Windows 服务的安装、启动、停止、卸载
  - 配置管理：初始化、查看、设置配置
  - 规则管理：列出、启用、禁用规则
  - 连接管理：查看、关闭连接
  - 缓存管理：清空 FakeIP 和 DNS 缓存
  - 监控功能：实时流量和内存监控`,
	PersistentPreRunE:  preRun,
	PersistentPostRunE: postRun,
}

// preRun 命令执行前的初始化
func preRun(cmd *cobra.Command, args []string) error {
	// 初始化配置
	initConfig()

	// 设置输出格式
	output.SetGlobalFormat(outputFmt)

	// 创建配置管理器
	cfgManager, err := config.NewConfigManager()
	if err != nil {
		return pkgerrors.ErrService("初始化配置管理器失败", err)
	}

	// 初始化输出流（从配置文件读取）
	cfg, err := cfgManager.LoadCLIConfigFromViper()
	if err == nil && cfg.Output.Mode != "console" && cfg.Output.Mode != "" {
		streamOutput, err = output.InitStreamOutput(cfg.Output.File, cfg.Output.Mode, cfg.Output.Append)
		if err != nil {
			return pkgerrors.ErrService("初始化输出流失败", err)
		}

		// 设置全局输出
		output.SetGlobalStdout(streamOutput.GetWriter())
		output.SetGlobalStderr(streamOutput.GetWriter())
	}

	// 初始化历史记录管理器
	pathResolver := cfgManager.GetPathResolver()
	historyDir := pathResolver.GetHistoryDir()
	historyFile := historyDir + "/commands.jsonl"
	historyManager = history.NewManager(historyFile)

	return nil
}

// postRun 命令执行后的清理
func postRun(cmd *cobra.Command, args []string) error {
	// 记录历史
	if historyManager != nil && cmd.Name() != "history" {
		// 构建完整命令
		fullCmd := cmd.CommandPath()
		if len(args) > 0 {
			fullCmd += " " + cmd.Flags().Arg(0)
		}

		entry := history.Entry{
			Timestamp: time.Now(),
			Command:   fullCmd,
			Success:   true,
		}

		// 忽略记录错误
		_ = historyManager.Add(entry)
	}

	// 关闭输出流
	if streamOutput != nil {
		if err := streamOutput.Close(); err != nil {
			return pkgerrors.ErrService("关闭输出流失败", err)
		}
		streamOutput = nil
	}
	return nil
}

// Execute 执行根命令
func Execute(ver, com string) error {
	SetVersionInfo(ver, com, "unknown")
	return rootCmd.Execute()
}

func init() {
	cobra.OnInitialize(initConfig)

	// 添加子命令
	rootCmd.AddCommand(NewModeCmd())
	rootCmd.AddCommand(NewProxyCmd())
	rootCmd.AddCommand(NewRuleCmd())
	rootCmd.AddCommand(NewCacheCmd())
	rootCmd.AddCommand(NewConnCmd())
	rootCmd.AddCommand(NewDNSCmd())
	rootCmd.AddCommand(NewServiceCmd())
	rootCmd.AddCommand(NewSysproxyCmd())
	rootCmd.AddCommand(NewSubCmd())
	rootCmd.AddCommand(NewVersionCmd())
	rootCmd.AddCommand(NewGeoIPCmd())
	rootCmd.AddCommand(NewMonitorCmd())
	rootCmd.AddCommand(NewLogsCmd())
	rootCmd.AddCommand(NewHistoryCmd())

	// 全局标志
	rootCmd.PersistentFlags().StringVarP(&cfgFile, "config", "c", "", "配置文件路径 (默认: ~/.config/.mihomo-cli/config.yaml)")
	rootCmd.PersistentFlags().StringVarP(&outputFmt, "output", "o", "table", "输出格式 (table/json)")
	rootCmd.PersistentFlags().StringVar(&apiURL, "api", "", "API 地址 (覆盖配置文件)")
	rootCmd.PersistentFlags().StringVar(&secret, "secret", "", "API 密钥 (覆盖配置文件)")
	rootCmd.PersistentFlags().IntVarP(&timeout, "timeout", "t", 10, "请求超时时间（秒）")
}

// initConfig 初始化配置
func initConfig() {
	// 创建配置管理器
	cfgManager, err := config.NewConfigManager()
	if err != nil {
		cobra.CheckErr(err)
		return
	}

	// 初始化 Viper 配置
	cfgManager.InitViperConfig(cfgFile)

	// 命令行参数优先级高于配置文件
	if apiURL != "" {
		viper.Set("api.address", apiURL)
	}
	if secret != "" {
		viper.Set("api.secret", secret)
	}
	if timeout != 0 {
		viper.Set("api.timeout", timeout)
	}
}
