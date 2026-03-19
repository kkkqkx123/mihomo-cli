package cmd

import (
	"context"
	"fmt"
	"time"

	"github.com/spf13/cobra"

	"github.com/kkkqkx123/mihomo-cli/internal/output"
	"github.com/kkkqkx123/mihomo-cli/internal/recovery"
	"github.com/kkkqkx123/mihomo-cli/internal/system"
	"github.com/kkkqkx123/mihomo-cli/internal/util"
	pkgerrors "github.com/kkkqkx123/mihomo-cli/pkg/errors"
)

var (
	recoveryAuto      bool
	recoveryComponent string
	recoveryInterval  int
)

var recoveryCmd = &cobra.Command{
	Use:   "recovery",
	Short: "自动恢复管理",
	Long:  "管理系统自动恢复功能，包括问题检测和自动修复。",
}

var recoveryDetectCmd = &cobra.Command{
	Use:   "detect",
	Short: "检测问题",
	Long:  "检测系统配置问题。",
	RunE:  runRecoveryDetect,
}

var recoveryExecuteCmd = &cobra.Command{
	Use:   "execute",
	Short: "执行恢复",
	Long:  "执行系统配置恢复。",
	RunE:  runRecoveryExecute,
}

var recoveryStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "查询恢复状态",
	Long:  "查询自动恢复的状态和配置。",
	RunE:  runRecoveryStatus,
}

var recoveryEnableCmd = &cobra.Command{
	Use:   "enable",
	Short: "启用自动恢复",
	Long:  "启用自动恢复功能。",
	RunE:  runRecoveryEnable,
}

var recoveryDisableCmd = &cobra.Command{
	Use:   "disable",
	Short: "禁用自动恢复",
	Long:  "禁用自动恢复功能。",
	RunE:  runRecoveryDisable,
}

func init() {
	rootCmd.AddCommand(recoveryCmd)

	// 添加子命令
	recoveryCmd.AddCommand(recoveryDetectCmd)
	recoveryCmd.AddCommand(recoveryExecuteCmd)
	recoveryCmd.AddCommand(recoveryStatusCmd)
	recoveryCmd.AddCommand(recoveryEnableCmd)
	recoveryCmd.AddCommand(recoveryDisableCmd)

	// execute 命令标志
	recoveryExecuteCmd.Flags().BoolVarP(&recoveryAuto, "auto", "a", false, "仅自动恢复可自动处理的问题")
	recoveryExecuteCmd.Flags().StringVarP(&recoveryComponent, "component", "c", "", "指定组件 (sysproxy, tun, route)")

	// enable 命令标志
	recoveryEnableCmd.Flags().IntVarP(&recoveryInterval, "interval", "i", 300, "检查间隔（秒）")
}

func createRecoveryManager() (*recovery.RecoveryManager, error) {
	// 创建系统配置管理器
	sysMgr, err := system.NewSystemConfigManager()
	if err != nil {
		return nil, pkgerrors.ErrService("failed to create system config manager", err)
	}

	// 创建恢复管理器
	recoveryMgr, err := recovery.NewRecoveryManager(sysMgr, nil)
	if err != nil {
		return nil, pkgerrors.ErrService("failed to create recovery manager", err)
	}

	return recoveryMgr, nil
}

func runRecoveryDetect(cmd *cobra.Command, args []string) error {
	// 创建恢复管理器
	mgr, err := createRecoveryManager()
	if err != nil {
		return err
	}

	// 检测问题
	problems, err := mgr.Detect()
	if err != nil {
		return pkgerrors.ErrService("failed to detect problems", err)
	}

	if len(problems) == 0 {
		output.Success("未检测到问题")
		return nil
	}

	fmt.Fprintf(output.GetGlobalStdout(), "检测到 %d 个问题:\n\n", len(problems))
	for i, problem := range problems {
		fmt.Fprintf(output.GetGlobalStdout(), "%d. [%s] %s\n", i+1, problem.Severity, problem.Description)
		fmt.Fprintf(output.GetGlobalStdout(), "   类型: %s\n", problem.Type)
		if len(problem.Solutions) > 0 {
			fmt.Fprintf(output.GetGlobalStdout(), "   解决方案:\n")
			for j, solution := range problem.Solutions {
				auto := ""
				if solution.Auto {
					auto = " (可自动执行)"
				}
				fmt.Fprintf(output.GetGlobalStdout(), "   %d. %s%s\n", j+1, solution.Description, auto)
				if solution.Command != "" {
					fmt.Fprintf(output.GetGlobalStdout(), "      命令: %s\n", solution.Command)
				}
			}
		}
		fmt.Fprintf(output.GetGlobalStdout(), "\n")
	}

	return nil
}

func runRecoveryExecute(cmd *cobra.Command, args []string) error {
	// 检查管理员权限
	if !util.IsAdmin() {
		return pkgerrors.ErrService("this operation requires administrator privileges, please run as administrator", nil)
	}

	// 创建恢复管理器
	mgr, err := createRecoveryManager()
	if err != nil {
		return err
	}

	ctx := context.Background()

	var report *recovery.RecoveryReport

	if recoveryAuto {
		// 自动恢复
		report, err = mgr.AutoRecover(ctx)
	} else {
		// 手动恢复
		report, err = mgr.CheckAndRecover(ctx)
	}

	if err != nil {
		return pkgerrors.ErrService("failed to execute recovery", err)
	}

	// 显示报告
	fmt.Fprintf(output.GetGlobalStdout(), "恢复报告:\n")
	fmt.Fprintf(output.GetGlobalStdout(), "  时间: %s\n", report.Timestamp.Format("2006-01-02 15:04:05"))
	fmt.Fprintf(output.GetGlobalStdout(), "  问题数量: %d\n", len(report.Problems))
	fmt.Fprintf(output.GetGlobalStdout(), "  执行动作: %d\n", len(report.Actions))
	fmt.Fprintf(output.GetGlobalStdout(), "  结果: ")
	if report.Success {
		output.Success("成功")
	} else {
		output.Error("失败")
		if report.ErrorMessage != "" {
			fmt.Fprintf(output.GetGlobalStderr(), "  错误: %s\n", report.ErrorMessage)
		}
	}
	fmt.Fprintf(output.GetGlobalStdout(), "  耗时: %v\n", report.Duration)

	if len(report.Actions) > 0 {
		fmt.Fprintf(output.GetGlobalStdout(), "\n执行的动作:\n")
		for i, action := range report.Actions {
			fmt.Fprintf(output.GetGlobalStdout(), "%d. %s - %s\n", i+1, action.Action, action.Problem.Type)
			if action.Success {
				output.Success("   结果: 成功")
			} else {
				output.Error("   结果: 失败")
				if action.ErrorMessage != "" {
					fmt.Fprintf(output.GetGlobalStderr(), "   错误: %s\n", action.ErrorMessage)
				}
			}
			fmt.Fprintf(output.GetGlobalStdout(), "   耗时: %v\n", action.Duration)
		}
	}

	return nil
}

func runRecoveryStatus(cmd *cobra.Command, args []string) error {
	// 创建恢复管理器
	mgr, err := createRecoveryManager()
	if err != nil {
		return err
	}

	// 获取状态
	status := mgr.GetStatus()

	fmt.Fprintf(output.GetGlobalStdout(), "自动恢复状态:\n")
	fmt.Fprintf(output.GetGlobalStdout(), "  启用: %v\n", status.Enabled)
	if !status.LastCheckTime.IsZero() {
		fmt.Fprintf(output.GetGlobalStdout(), "  上次检查: %s\n", status.LastCheckTime.Format("2006-01-02 15:04:05"))
	}
	if status.LastRecovery != nil {
		fmt.Fprintf(output.GetGlobalStdout(), "\n上次恢复:\n")
		fmt.Fprintf(output.GetGlobalStdout(), "  时间: %s\n", status.LastRecovery.Timestamp.Format("2006-01-02 15:04:05"))
		fmt.Fprintf(output.GetGlobalStdout(), "  问题数量: %d\n", len(status.LastRecovery.Problems))
		fmt.Fprintf(output.GetGlobalStdout(), "  结果: ")
		if status.LastRecovery.Success {
			output.Success("成功")
		} else {
			output.Error("失败")
		}
		fmt.Fprintf(output.GetGlobalStdout(), "  耗时: %v\n", status.LastRecovery.Duration)
	}

	// 显示配置
	config := mgr.GetConfig()
	fmt.Fprintf(output.GetGlobalStdout(), "\n恢复配置:\n")
	fmt.Fprintf(output.GetGlobalStdout(), "  自动恢复: %v\n", config.AutoRecover)
	fmt.Fprintf(output.GetGlobalStdout(), "  备份后恢复: %v\n", config.BackupBeforeRecover)
	fmt.Fprintf(output.GetGlobalStdout(), "  最大重试次数: %d\n", config.MaxRetryCount)
	fmt.Fprintf(output.GetGlobalStdout(), "  重试间隔: %v\n", config.RetryInterval)
	fmt.Fprintf(output.GetGlobalStdout(), "  检查组件: %v\n", config.Components)

	return nil
}

func runRecoveryEnable(cmd *cobra.Command, args []string) error {
	// 创建恢复管理器
	mgr, err := createRecoveryManager()
	if err != nil {
		return err
	}

	// 更新配置
	config := mgr.GetConfig()
	config.Enabled = true
	config.AutoRecover = true
	mgr.SetConfig(config)

	// 启动定期检查
	ctx := context.Background()
	interval := time.Duration(recoveryInterval) * time.Second
	if err := mgr.StartPeriodicCheck(ctx, interval); err != nil {
		return pkgerrors.ErrService("failed to start periodic check", err)
	}

	output.Success("自动恢复已启用")
	fmt.Fprintf(output.GetGlobalStdout(), "检查间隔: %v\n", interval)

	return nil
}

func runRecoveryDisable(cmd *cobra.Command, args []string) error {
	// 创建恢复管理器
	mgr, err := createRecoveryManager()
	if err != nil {
		return err
	}

	// 更新配置
	config := mgr.GetConfig()
	config.Enabled = false
	config.AutoRecover = false
	mgr.SetConfig(config)

	output.Success("自动恢复已禁用")

	return nil
}
