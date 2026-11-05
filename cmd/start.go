package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/kkkqkx123/mihomo-cli/internal/config"
	"github.com/kkkqkx123/mihomo-cli/internal/mihomo"
	"github.com/kkkqkx123/mihomo-cli/internal/output"
	pkgerrors "github.com/kkkqkx123/mihomo-cli/pkg/errors"
)

var stopAll bool
var stopForce bool
var stopIncludeUnmanaged bool
var startForce bool

var startCmd = &cobra.Command{
	Use:   "start",
	Short: "启动 Mihomo 内核（后台进程模式）",
	Long: `启动 Mihomo 内核并以后台进程运行。

该命令会：
1. 读取 config.toml 配置文件（默认用户目录，可用 --toml 显式指定）
2. [mihomo] executable 为空时从系统 PATH 查找 mihomo
3. [mihomo] config_file 为空时直接启动 mihomo，由内核使用默认配置（~/.config/mihomo/config.yaml）
4. 等待并验证内核启动成功

启动后会进行健康检查，确保内核成功启动。如果启动失败或超时，会自动停止内核。`,
	RunE: runStart,
}

var stopCmd = &cobra.Command{
	Use:   "stop",
	Short: "停止 Mihomo 内核",
	Long: `停止正在运行的 Mihomo 内核进程。

可以指定 PID 或使用 --all 停止所有进程。
停止操作会等待进程完全退出后才返回。

默认通过 API 优雅关闭，如果 API 不可用，需要使用 -F 强制关闭。`,
	Example: `  mihomo-cli stop           # 停止默认配置的实例（通过 API）
  mihomo-cli stop 12345      # 停止指定 PID 的实例（通过 API）
  mihomo-cli stop -F         # 强制关闭默认配置的实例
  mihomo-cli stop -F 12345   # 强制关闭指定 PID 的实例
  mihomo-cli stop --all      # 停止所有 Mihomo 进程`,
	Args: cobra.MaximumNArgs(1),
	RunE: runStop,
}

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "查询 Mihomo 内核状态",
	Long: `查询 Mihomo 内核的运行状态。

状态通过 [api] 或 --api/--secret 指定的 API 实时探测，不需要 TOML 配置。
API 不可达时会扫描进程，区分“未运行”和“运行中但 API 不可达”。
使用 -o json 输出可被脚本解析的状态对象。`,
	Args: cobra.NoArgs,
	RunE: runStatus,
}

func init() {
	rootCmd.AddCommand(startCmd)
	rootCmd.AddCommand(stopCmd)
	rootCmd.AddCommand(statusCmd)

	// start 命令的标志
	startCmd.Flags().BoolVarP(&startForce, "force", "f", false, "强制启动，跳过清理检查")

	// stop 命令的标志
	stopCmd.Flags().BoolVarP(&stopAll, "all", "a", false, "停止所有 Mihomo 进程")
	stopCmd.Flags().BoolVarP(&stopForce, "force", "F", false, "强制关闭进程（不通过 API）")
	stopCmd.Flags().BoolVar(&stopIncludeUnmanaged, "include-unmanaged", false, "配合 --all 使用：同时停止未托管（外部启动/未验证）的 Mihomo 进程")
}

func runStart(cmd *cobra.Command, args []string) error {
	// 查找配置文件路径，仅支持显式 --toml 与用户目录
	output.Info("查找配置文件...")
	configPath, err := config.FindTomlConfigPath(tomlFile)
	if err != nil {
		return pkgerrors.ErrConfig("查找 TOML 配置失败", err)
	}

	// 加载配置
	output.Info("加载配置: %s", configPath)
	cfg, err := config.LoadTomlConfig(configPath)
	if err != nil {
		return pkgerrors.ErrConfig("加载配置失败", err)
	}

	// 创建进程处理器
	handler := mihomo.NewProcessHandler(configPath)

	// 启动内核（守护进程模式或传统模式）
	output.Info("启动 Mihomo 进程...")
	result, err := handler.Start(cfg, startForce)
	if err != nil {
		return pkgerrors.ErrService("启动 Mihomo 失败", err)
	}

	// 启动成功
	output.Println("=====================================")
	output.Println("  Mihomo 内核已启动")
	output.Println("=====================================")
	output.PrintKeyValue("API 地址", fmt.Sprintf("http://%s", result.APIAddress))
	if result.Secret != "" {
		output.PrintKeyValue("密钥", result.Secret)
	} else {
		output.PrintKeyValue("密钥", "(未设置)")
	}
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
	// 查找配置文件路径，仅支持显式 --toml 与用户目录
	output.Info("查找配置文件...")
	configPath, err := config.FindTomlConfigPath(tomlFile)
	if err != nil {
		return pkgerrors.ErrConfig("查找 TOML 配置失败", err)
	}

	// 加载配置
	output.Info("加载配置: %s", configPath)
	cfg, err := config.LoadTomlConfig(configPath)
	if err != nil {
		return pkgerrors.ErrConfig("加载配置失败", err)
	}

	// 创建进程处理器
	handler := mihomo.NewProcessHandler(configPath)

	// 停止内核
	output.Info("停止 Mihomo 进程...")
	result, err := handler.Stop(cfg, stopAll, stopForce, stopIncludeUnmanaged, args)
	if err != nil {
		return pkgerrors.ErrService("停止 Mihomo 失败", err)
	}

	if result != nil {
		output.Success("Mihomo 内核已停止 (PID: %d)", result.PID)
	}

	return nil
}

func runStatus(cmd *cobra.Command, args []string) error {
	result, err := mihomo.QueryStatus(viper.GetString("api.address"), viper.GetString("api.secret"))
	if err != nil {
		return pkgerrors.WrapError("查询 Mihomo 状态失败", err)
	}

	if outputFmt == "json" {
		return output.PrintJSON(result)
	}

	switch result.State {
	case mihomo.KernelStateRunning:
		output.Success("状态: 运行中")
	case mihomo.KernelStateUnreachable:
		if result.Running {
			output.Warning("状态: 运行中（API 不可达）")
		} else {
			output.Warning("状态: API 不可达")
		}
	default:
		output.Println("状态: 未运行")
	}

	if result.PID > 0 {
		output.PrintKeyValue("PID", result.PID)
	}
	if manager := formatStatusManager(result); manager != "" {
		output.PrintKeyValue("管理", manager)
	}
	if result.PID == 0 && result.ProcessCount > 1 {
		output.PrintKeyValue("进程数", result.ProcessCount)
	}
	if result.APIURL != "" {
		output.PrintKeyValue("API 地址", result.APIURL)
	}

	return nil
}

func formatStatusManager(result *mihomo.StatusResult) string {
	switch result.Manager {
	case mihomo.StatusManagerSystemd:
		if result.Unit != "" {
			return fmt.Sprintf("systemd (%s)", result.Unit)
		}
		return "systemd"
	case mihomo.StatusManagerCLI:
		return "mihomo-cli"
	case mihomo.StatusManagerExternal:
		return "外部进程"
	default:
		return ""
	}
}
