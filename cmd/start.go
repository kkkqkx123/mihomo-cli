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

	// 设置信号处理，确保在用户退出时停止内核
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// 在另一个 goroutine 中启动内核
	output.Info("启动 Mihomo 进程...")
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
		output.Println("=====================================")
		output.Println("  Mihomo 内核已启动")
		output.Println("=====================================")
		output.PrintKeyValue("API 地址", fmt.Sprintf("http://%s", result.APIAddress))
		output.PrintKeyValue("密钥", result.Secret)
		output.PrintEmptyLine()
		output.Println("使用以下命令管理：")
		output.Println("  mihomo-cli status  - 查询运行状态")
		output.Println("  mihomo-cli stop    - 停止内核")
		output.Println("=====================================")

		return nil

	case sig := <-sigChan:
		// 收到中断信号
		output.PrintEmptyLine()
		output.Printf("Received signal %v, stopping Mihomo kernel...\n", sig)

		// 通过 PID 文件停止进程
		pm := mihomo.NewProcessManager(cfg)
		pid, err := pm.GetPIDFromPIDFile()
		if err != nil {
			_ = output.PrintError(fmt.Sprintf("Failed to get PID: %v", err))
			output.Println("Kernel may have already stopped or PID file is missing")
			return nil
		}

		output.Printf("Stopping Mihomo kernel (PID: %d)...\n", pid)

		if err := mihomo.StopProcessByPID(pid); err != nil {
			_ = output.PrintError(fmt.Sprintf("Failed to stop kernel: %v", err))
			output.PrintEmptyLine()
			output.Println("Recovery suggestions:")
			output.Println("  1. Check if the process is still running: mihomo-cli status")
			output.Println("  2. If process is running, try stopping again: mihomo-cli stop")
			output.Println("  3. If process is unresponsive, force kill: mihomo-cli stop --force")
			output.Println("  4. If system configuration is not cleaned up, restart the system")
			return err
		}

		output.PrintEmptyLine()
		output.Success("Kernel stopped successfully")

		// 检查系统配置是否已清理
		output.Info("Checking system configuration...")
		checker := config.NewSystemChecker()
		if err := checker.CheckAfterStop(); err != nil {
			output.Warning("System configuration check failed: " + err.Error())
			output.PrintEmptyLine()
			output.Println("Some system configurations may not have been cleaned up properly.")
			output.Println("If you experience network issues, try:")
			output.Println("  1. Restarting Mihomo: mihomo-cli start")
			output.Println("  2. Running cleanup command: mihomo-cli system cleanup")
			output.Println("  3. Restarting the system (recommended)")
		} else {
			output.Success("System configuration cleaned up successfully")
		}

		return nil
	}
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
	result, err := handler.Stop(cfg, stopAll, stopConfig, args)
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
