package cmd

import (
	"context"
	"fmt"
	"time"

	"github.com/spf13/cobra"

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
		fmt.Println("未检测到问题")
		return nil
	}

	fmt.Printf("检测到 %d 个问题:\n\n", len(problems))
	for i, problem := range problems {
		fmt.Printf("%d. [%s] %s\n", i+1, problem.Severity, problem.Description)
		fmt.Printf("   类型: %s\n", problem.Type)
		if len(problem.Solutions) > 0 {
			fmt.Println("   解决方案:")
			for j, solution := range problem.Solutions {
				auto := ""
				if solution.Auto {
					auto = " (可自动执行)"
				}
				fmt.Printf("   %d. %s%s\n", j+1, solution.Description, auto)
				if solution.Command != "" {
					fmt.Printf("      命令: %s\n", solution.Command)
				}
			}
		}
		fmt.Println()
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
	fmt.Println("恢复报告:")
	fmt.Printf("  时间: %s\n", report.Timestamp.Format("2006-01-02 15:04:05"))
	fmt.Printf("  问题数量: %d\n", len(report.Problems))
	fmt.Printf("  执行动作: %d\n", len(report.Actions))
	fmt.Printf("  结果: ", )
	if report.Success {
		fmt.Println("成功")
	} else {
		fmt.Println("失败")
		if report.ErrorMessage != "" {
			fmt.Printf("  错误: %s\n", report.ErrorMessage)
		}
	}
	fmt.Printf("  耗时: %v\n", report.Duration)

	if len(report.Actions) > 0 {
		fmt.Println("\n执行的动作:")
		for i, action := range report.Actions {
			fmt.Printf("%d. %s - %s\n", i+1, action.Action, action.Problem.Type)
			if action.Success {
				fmt.Println("   结果: 成功")
			} else {
				fmt.Println("   结果: 失败")
				if action.ErrorMessage != "" {
					fmt.Printf("   错误: %s\n", action.ErrorMessage)
				}
			}
			fmt.Printf("   耗时: %v\n", action.Duration)
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

	fmt.Println("自动恢复状态:")
	fmt.Printf("  启用: %v\n", status.Enabled)
	if !status.LastCheckTime.IsZero() {
		fmt.Printf("  上次检查: %s\n", status.LastCheckTime.Format("2006-01-02 15:04:05"))
	}
	if status.LastRecovery != nil {
		fmt.Println("\n上次恢复:")
		fmt.Printf("  时间: %s\n", status.LastRecovery.Timestamp.Format("2006-01-02 15:04:05"))
		fmt.Printf("  问题数量: %d\n", len(status.LastRecovery.Problems))
		fmt.Printf("  结果: ", )
		if status.LastRecovery.Success {
			fmt.Println("成功")
		} else {
			fmt.Println("失败")
		}
		fmt.Printf("  耗时: %v\n", status.LastRecovery.Duration)
	}

	// 显示配置
	config := mgr.GetConfig()
	fmt.Println("\n恢复配置:")
	fmt.Printf("  自动恢复: %v\n", config.AutoRecover)
	fmt.Printf("  备份后恢复: %v\n", config.BackupBeforeRecover)
	fmt.Printf("  最大重试次数: %d\n", config.MaxRetryCount)
	fmt.Printf("  重试间隔: %v\n", config.RetryInterval)
	fmt.Printf("  检查组件: %v\n", config.Components)

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

	fmt.Println("自动恢复已启用")
	fmt.Printf("检查间隔: %v\n", interval)

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

	fmt.Println("自动恢复已禁用")

	return nil
}
