package cmd

import (
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/kkkqkx123/mihomo-cli/internal/output"
)

var (
	cfgFile   string
	outputFmt string
	apiURL    string
	secret    string
	timeout   int
	outputFile string
	appendMode bool
)

// outputFileHandle 输出文件句柄
var outputFileHandle *os.File

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

	// 初始化文件输出
	if outputFile != "" {
		var err error
		if appendMode {
			outputFileHandle, err = os.OpenFile(outputFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
		} else {
			outputFileHandle, err = os.Create(outputFile)
		}
		if err != nil {
			return err
		}
		output.SetGlobalStdout(outputFileHandle)
		output.SetGlobalStderr(outputFileHandle)
	}

	return nil
}

// postRun 命令执行后的清理
func postRun(cmd *cobra.Command, args []string) error {
	// 关闭输出文件
	if outputFileHandle != nil {
		if err := outputFileHandle.Close(); err != nil {
			return err
		}
		outputFileHandle = nil
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
	rootCmd.AddCommand(NewServiceCmd())
	rootCmd.AddCommand(NewSysproxyCmd())
	rootCmd.AddCommand(NewSubCmd())
	rootCmd.AddCommand(NewVersionCmd())

	// 全局标志
	rootCmd.PersistentFlags().StringVarP(&cfgFile, "config", "c", "", "配置文件路径 (默认: ~/.mihomo-cli/config.yaml)")
	rootCmd.PersistentFlags().StringVarP(&outputFmt, "output", "o", "table", "输出格式 (table/json)")
	rootCmd.PersistentFlags().StringVar(&apiURL, "api", "", "API 地址 (覆盖配置文件)")
	rootCmd.PersistentFlags().StringVar(&secret, "secret", "", "API 密钥 (覆盖配置文件)")
	rootCmd.PersistentFlags().IntVarP(&timeout, "timeout", "t", 10, "请求超时时间（秒）")
	rootCmd.PersistentFlags().StringVarP(&outputFile, "file", "f", "", "输出到指定文件")
	rootCmd.PersistentFlags().BoolVar(&appendMode, "append", false, "追加模式（与 -f 一起使用）")
}

// initConfig 初始化配置
func initConfig() {
	if cfgFile != "" {
		// 使用指定的配置文件
		viper.SetConfigFile(cfgFile)
	} else {
		// 使用默认配置文件路径
		home, err := os.UserHomeDir()
		cobra.CheckErr(err)

		viper.AddConfigPath(home + "/.mihomo-cli")
		viper.SetConfigType("yaml")
		viper.SetConfigName("config")
	}

	// 支持环境变量
	viper.SetEnvPrefix("MIHOMO")
	viper.AutomaticEnv()

	// 读取配置文件（如果存在）
	if err := viper.ReadInConfig(); err == nil {
		// 配置文件读取成功
	}

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
