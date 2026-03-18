package cmd

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/kkkqkx123/mihomo-cli/internal/config"
	"github.com/kkkqkx123/mihomo-cli/internal/mihomo"
	pkgerrors "github.com/kkkqkx123/mihomo-cli/pkg/errors"
)

var foregroundMode bool
var stopAll bool
var stopForce bool
var stopConfig string

var startCmd = &cobra.Command{
	Use:   "start",
	Short: "启动 Mihomo 内核",
	Long: `启动 Mihomo 内核并自动生成随机密钥。

该命令会：
1. 读取 config.toml 配置文件
2. 自动生成 SHA256 随机密钥
3. 启动 Mihomo 内核进程（后台模式）
4. 输出 API 地址和密钥信息

默认为后台模式，启动后立即返回。使用 --foreground 可切换到前台模式。`,
	RunE: runStart,
}

var stopCmd = &cobra.Command{
	Use:   "stop",
	Short: "停止 Mihomo 内核",
	Long:  `停止正在运行的 Mihomo 内核进程。

可以指定 PID 或使用 --all 停止所有进程。`,
	Example: `  mihomo-cli stop           # 停止默认配置的实例
  mihomo-cli stop 12345      # 停止指定 PID 的实例
  mihomo-cli stop --all      # 停止所有 Mihomo 进程
  mihomo-cli stop --force    # 强制停止（不验证）`,
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

	startCmd.Flags().BoolVarP(&foregroundMode, "foreground", "F", false, "前台模式（阻塞终端，用于调试）")

	// stop 命令的标志
	stopCmd.Flags().BoolVarP(&stopAll, "all", "a", false, "停止所有 Mihomo 进程")
	stopCmd.Flags().BoolVarP(&stopForce, "force", "F", false, "强制停止（不验证进程）")
	stopCmd.Flags().StringVarP(&stopConfig, "config", "c", "", "指定配置文件路径")
}

func runStart(cmd *cobra.Command, args []string) error {
	// 加载配置
	cfg, err := config.LoadTomlConfig("config.toml")
	if err != nil {
		return pkgerrors.ErrConfig("failed to load config", err)
	}

	// 创建进程处理器
	handler := mihomo.NewProcessHandler("")

	// 启动内核
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	result, err := handler.Start(ctx, cfg, foregroundMode)
	if err != nil {
		return err
	}

	// 输出信息
	fmt.Println("=====================================")
	fmt.Println("  Mihomo 内核已启动")
	fmt.Println("=====================================")
	fmt.Printf("API 地址: http://%s\n", result.APIAddress)
	fmt.Printf("密钥: %s\n", result.Secret)
	fmt.Println()

	if foregroundMode {
		// 前台模式：等待中断信号
		fmt.Println("提示: 请保存密钥，用于 API 认证")
		fmt.Println("按 Ctrl+C 停止内核")
		fmt.Println("=====================================")
	} else {
		// 后台模式：立即返回
		fmt.Println("提示: 内核已在后台运行")
		fmt.Println("使用以下命令管理：")
		fmt.Println("  mihomo-cli status  - 查询运行状态")
		fmt.Println("  mihomo-cli stop    - 停止内核")
		fmt.Println("=====================================")
	}

	return nil
}

func runStop(cmd *cobra.Command, args []string) error {
	// 加载配置
	cfg, err := config.LoadTomlConfig("config.toml")
	if err != nil {
		return pkgerrors.ErrConfig("failed to load config", err)
	}

	// 创建进程处理器
	handler := mihomo.NewProcessHandler("")

	// 停止内核
	result, err := handler.Stop(cfg, stopAll, stopForce, stopConfig, args)
	if err != nil {
		return err
	}

	if result != nil {
		fmt.Printf("Mihomo 内核已停止 (PID: %d)\n", result.PID)
	}

	return nil
}

func runStatus(cmd *cobra.Command, args []string) error {
	// 加载配置
	cfg, err := config.LoadTomlConfig("config.toml")
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
		fmt.Printf("状态: 运行中\n")
		fmt.Printf("PID: %d\n", result.PID)
		fmt.Printf("API 地址: http://%s\n", result.APIAddress)
	} else {
		fmt.Println("状态: 未运行")
	}

	return nil
}
