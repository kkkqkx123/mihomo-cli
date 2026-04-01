package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/kkkqkx123/mihomo-cli/internal/config"
	"github.com/kkkqkx123/mihomo-cli/internal/mihomo"
	"github.com/kkkqkx123/mihomo-cli/internal/output"
	pkgerrors "github.com/kkkqkx123/mihomo-cli/pkg/errors"
)

var stopAll bool
var stopConfig string
var stopForce bool

var startCmd = &cobra.Command{
	Use:   "start",
	Short: "启动 Mihomo 内核（守护进程模式）",
	Long: `启动 Mihomo 内核并自动生成随机密钥。

该命令会：
1. 读取 config.toml 配置文件（优先当前目录）
2. 自动生成 SHA256 随机密钥
3. 以守护进程模式启动 Mihomo 内核（独立进程）
4. 等待并验证内核启动成功
5. 输出 API 地址和密钥信息

守护进程模式下，内核作为独立进程运行，关闭终端不会影响内核运行。
启动后会进行健康检查，确保内核成功启动。如果启动失败或超时，会自动停止内核。`,
	RunE: runStart,
}

var stopCmd = &cobra.Command{
	Use:   "stop",
	Short: "停止 Mihomo 内核",
	Long:  `停止正在运行的 Mihomo 内核进程。

可以指定 PID 或使用 --all 停止所有进程。
停止操作会等待进程完全退出后才返回。

默认通过 API 优雅关闭，如果 API 不可用，需要使用 -F 强制关闭。`,
	Example: `  mihomo-cli stop           # 停止默认配置的实例（通过 API）
  mihomo-cli stop 12345      # 停止指定 PID 的实例（通过 API）
  mihomo-cli stop -F         # 强制关闭默认配置的实例
  mihomo-cli stop -F 12345   # 强制关闭指定 PID 的实例
  mihomo-cli stop --all      # 停止所有 Mihomo 进程`,
	Args:  cobra.MaximumNArgs(1),
	RunE: runStop,
}

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "查询 Mihomo 内核状态",
	Long:  `查询 Mihomo 内核进程的运行状态。`,
	RunE: runStatus,
}

func init() {
	rootCmd.AddCommand(startCmd)
	rootCmd.AddCommand(stopCmd)
	rootCmd.AddCommand(statusCmd)

	// stop 命令的标志
	stopCmd.Flags().BoolVarP(&stopAll, "all", "a", false, "停止所有 Mihomo 进程")
	stopCmd.Flags().StringVarP(&stopConfig, "config", "c", "", "指定配置文件路径")
	stopCmd.Flags().BoolVarP(&stopForce, "force", "F", false, "强制关闭进程（不通过 API）")
}

func runStart(cmd *cobra.Command, args []string) error {
	// 查找配置文件路径
	output.Info("查找配置文件...")
	configPath := config.FindTomlConfigPath(cfgFile)

	// 加载配置
	output.Info("加载配置: %s", configPath)
	cfg, err := config.LoadTomlConfig(configPath)
	if err != nil {
		return pkgerrors.ErrConfig("failed to load config", err)
	}

	// 创建进程处理器
	handler := mihomo.NewProcessHandler("")

	// 启动内核（守护进程模式或传统模式）
	output.Info("启动 Mihomo 进程...")
	result, err := handler.Start(cfg)
	if err != nil {
		return err
	}

	// 启动成功
	output.Println("=====================================")
	output.Println("  Mihomo 内核已启动")
	output.Println("=====================================")
	output.PrintKeyValue("API 地址", fmt.Sprintf("http://%s", result.APIAddress))
	output.PrintKeyValue("密钥", result.Secret)
	if result.PID > 0 {
		output.PrintKeyValue("PID", result.PID)
	}
	output.PrintEmptyLine()

	output.Println("守护进程模式：内核将在后台独立运行")
	output.Println("关闭终端不会影响内核运行")

	output.PrintEmptyLine()
	output.Println("使用以下命令管理：")
	output.Println("  mihomo-cli status  - 查询运行状态")
	output.Println("  mihomo-cli stop    - 停止内核")
	output.Println("=====================================")

	return nil
}

func runStop(cmd *cobra.Command, args []string) error {
	// 查找配置文件路径
	output.Info("查找配置文件...")
	configPath := config.FindTomlConfigPath(cfgFile)

	// 加载配置
	output.Info("加载配置: %s", configPath)
	cfg, err := config.LoadTomlConfig(configPath)
	if err != nil {
		return pkgerrors.ErrConfig("failed to load config", err)
	}

	// 创建进程处理器
	handler := mihomo.NewProcessHandler("")

	// 停止内核
	output.Info("停止 Mihomo 进程...")
	result, err := handler.Stop(cfg, stopAll, stopConfig, stopForce, args)
	if err != nil {
		return err
	}

	if result != nil {
		output.Success("Mihomo 内核已停止 (PID: %d)", result.PID)
	}

	return nil
}

func runStatus(cmd *cobra.Command, args []string) error {
	// 查找配置文件路径
	output.Info("查找配置文件...")
	configPath := config.FindTomlConfigPath(cfgFile)

	// 加载配置
	output.Info("加载配置: %s", configPath)
	cfg, err := config.LoadTomlConfig(configPath)
	if err != nil {
		return pkgerrors.ErrConfig("failed to load config", err)
	}

	// 创建进程处理器
	handler := mihomo.NewProcessHandler("")

	// 查询状态
	result, err := handler.Status(cfg)
	if err != nil {
		return err
	}

	if result.IsRunning {
		output.Println("状态: 运行中")
		output.PrintKeyValue("PID", result.PID)
		output.PrintKeyValue("API 地址", fmt.Sprintf("http://%s", result.APIAddress))
	} else {
		output.Println("状态: 未运行")
	}

	return nil
}
