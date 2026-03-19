package cmd

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/spf13/cobra"

	"github.com/kkkqkx123/mihomo-cli/internal/config"
	"github.com/kkkqkx123/mihomo-cli/internal/mihomo"
	"github.com/kkkqkx123/mihomo-cli/internal/output"
	pkgerrors "github.com/kkkqkx123/mihomo-cli/pkg/errors"
)

var stopAll bool
var stopConfig string

var startCmd = &cobra.Command{
	Use:   "start",
	Short: "启动 Mihomo 内核",
	Long: `启动 Mihomo 内核并自动生成随机密钥。

该命令会：
1. 读取 config.toml 配置文件（优先当前目录）
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
	// 查找配置文件路径
	configPath := config.FindTomlConfigPath(cfgFile)

	// 加载配置
	cfg, err := config.LoadTomlConfig(configPath)
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
		fmt.Fprintf(output.GetGlobalStdout(), "=====================================\n")
		fmt.Fprintf(output.GetGlobalStdout(), "  Mihomo 内核已启动\n")
		fmt.Fprintf(output.GetGlobalStdout(), "=====================================\n")
		fmt.Fprintf(output.GetGlobalStdout(), "API 地址: http://%s\n", result.APIAddress)
		fmt.Fprintf(output.GetGlobalStdout(), "密钥: %s\n", result.Secret)
		fmt.Fprintf(output.GetGlobalStdout(), "\n")
		fmt.Fprintf(output.GetGlobalStdout(), "使用以下命令管理：\n")
		fmt.Fprintf(output.GetGlobalStdout(), "  mihomo-cli status  - 查询运行状态\n")
		fmt.Fprintf(output.GetGlobalStdout(), "  mihomo-cli stop    - 停止内核\n")
		fmt.Fprintf(output.GetGlobalStdout(), "=====================================\n")

		return nil

	case sig := <-sigChan:
		// 收到中断信号
		fmt.Fprintf(output.GetGlobalStdout(), "\nReceived signal %v, stopping Mihomo kernel...\n", sig)

		// 通过 PID 文件停止进程
		pm := mihomo.NewProcessManager(cfg)
		pid, err := pm.GetPIDFromPIDFile()
		if err != nil {
			fmt.Fprintf(output.GetGlobalStderr(), "Failed to get PID: %v\n", err)
			fmt.Fprintf(output.GetGlobalStdout(), "Kernel may have already stopped or PID file is missing\n")
			return nil
		}

		fmt.Fprintf(output.GetGlobalStdout(), "Stopping Mihomo kernel (PID: %d)...\n", pid)
		
		if err := mihomo.StopProcessByPID(pid); err != nil {
			fmt.Fprintf(output.GetGlobalStderr(), "Failed to stop kernel: %v\n", err)
			fmt.Fprintf(output.GetGlobalStdout(), "\nRecovery suggestions:\n")
			fmt.Fprintf(output.GetGlobalStdout(), "  1. Check if the process is still running: mihomo-cli status\n")
			fmt.Fprintf(output.GetGlobalStdout(), "  2. If process is running, try stopping again: mihomo-cli stop\n")
			fmt.Fprintf(output.GetGlobalStdout(), "  3. If process is unresponsive, force kill: mihomo-cli stop --force\n")
			fmt.Fprintf(output.GetGlobalStdout(), "  4. If system configuration is not cleaned up, restart the system\n")
			return err
		}

		fmt.Fprintf(output.GetGlobalStdout(), "\nKernel stopped successfully\n")
		fmt.Fprintf(output.GetGlobalStdout(), "System configuration should be cleaned up\n")
		fmt.Fprintf(output.GetGlobalStdout(), "If you experience network issues, try:\n")
		fmt.Fprintf(output.GetGlobalStdout(), "  1. Restarting Mihomo: mihomo-cli start\n")
		fmt.Fprintf(output.GetGlobalStdout(), "  2. Or restart the system\n")
		return nil
	}
}

func runStop(cmd *cobra.Command, args []string) error {
	// 查找配置文件路径
	configPath := config.FindTomlConfigPath(cfgFile)

	// 加载配置
	cfg, err := config.LoadTomlConfig(configPath)
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
		fmt.Fprintf(output.GetGlobalStdout(), "Mihomo 内核已停止 (PID: %d)\n", result.PID)
	}

	return nil
}

func runStatus(cmd *cobra.Command, args []string) error {
	// 查找配置文件路径
	configPath := config.FindTomlConfigPath(cfgFile)

	// 加载配置
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
		fmt.Fprintf(output.GetGlobalStdout(), "状态: 运行中\n")
		fmt.Fprintf(output.GetGlobalStdout(), "PID: %d\n", result.PID)
		fmt.Fprintf(output.GetGlobalStdout(), "API 地址: http://%s\n", result.APIAddress)
	} else {
		fmt.Fprintf(output.GetGlobalStdout(), "状态: 未运行\n")
	}

	return nil
}
