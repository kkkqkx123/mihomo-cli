package cmd

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/spf13/cobra"

	"github.com/kkkqkx123/mihomo-cli/internal/config"
	"github.com/kkkqkx123/mihomo-cli/internal/mihomo"
	pkgerrors "github.com/kkkqkx123/mihomo-cli/pkg/errors"
)

var stopAll bool
var stopConfig string

var startCmd = &cobra.Command{
	Use:   "start",
	Short: "启动 Mihomo 内核",
	Long: `启动 Mihomo 内核并自动生成随机密钥。

该命令会：
1. 读取 config.toml 配置文件
2. 自动生成 SHA256 随机密钥
3. 启动 Mihomo 内核进程
4. 等待并验证内核启动成功
5. 输出 API 地址和密钥信息

启动后会进行健康检查，确保内核成功启动。如果启动失败或超时，会自动停止内核。`,
	RunE: runStart,
}

var stopCmd = &cobra.Command{
	Use:   "stop",
	Short: "停止 Mihomo 内核",
	Long:  `停止正在运行的 Mihomo 内核进程。

可以指定 PID 或使用 --all 停止所有进程。
停止操作会等待进程完全退出后才返回。`,
	Example: `  mihomo-cli stop           # 停止默认配置的实例
  mihomo-cli stop 12345      # 停止指定 PID 的实例
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
}

func runStart(cmd *cobra.Command, args []string) error {
	// 加载配置
	cfg, err := config.LoadTomlConfig("config.toml")
	if err != nil {
		return pkgerrors.ErrConfig("failed to load config", err)
	}

	// 创建进程处理器
	handler := mihomo.NewProcessHandler("")

	// 设置信号处理，确保在用户退出时停止内核
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// 在另一个 goroutine 中启动内核
	startErrChan := make(chan error, 1)
	var result *mihomo.StartResult

	go func() {
		var err error
		result, err = handler.Start(cfg)
		startErrChan <- err
	}()

	// 等待启动完成或收到中断信号
	select {
	case err := <-startErrChan:
		if err != nil {
			return err
		}

		// 启动成功
		fmt.Println("=====================================")
		fmt.Println("  Mihomo 内核已启动")
		fmt.Println("=====================================")
		fmt.Printf("API 地址: http://%s\n", result.APIAddress)
		fmt.Printf("密钥: %s\n", result.Secret)
		fmt.Println()
		fmt.Println("使用以下命令管理：")
		fmt.Println("  mihomo-cli status  - 查询运行状态")
		fmt.Println("  mihomo-cli stop    - 停止内核")
		fmt.Println("=====================================")

		return nil

	case sig := <-sigChan:
		// 收到中断信号
		fmt.Printf("\n收到信号 %v，正在停止内核...\n", sig)

		// 通过 PID 文件停止进程
		pm := mihomo.NewProcessManager(cfg)
		if pid, err := pm.GetPIDFromPIDFile(); err == nil {
			if err := mihomo.StopProcessByPID(pid); err != nil {
				fmt.Printf("停止内核失败: %v\n", err)
				return err
			}
		}

		fmt.Println("内核已停止")
		return nil
	}
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
	result, err := handler.Stop(cfg, stopAll, stopConfig, args)
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
