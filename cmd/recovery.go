package cmd

import (
	"context"
	"fmt"

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
	recoveryForce     bool
	recoveryProblem   string
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
	Long: `执行系统配置恢复。

高风险操作（如进程重启、配置回滚）默认跳过，需要使用 -F/--force 参数强制执行。

可使用 -p/--problem 指定要修复的问题类型，支持以下类型：
  - config-residual     : 配置残留
  - process-abnormal    : 进程异常
  - config-inconsistent : 配置不一致
  - port-conflict       : 端口冲突
  - permission-denied   : 权限不足`,
	RunE: runRecoveryExecute,
}

var recoveryStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "查询恢复状态",
	Long:  "查询自动恢复的状态和配置。",
	RunE:  runRecoveryStatus,
}

func init() {
	rootCmd.AddCommand(recoveryCmd)

	// 添加子命令
	recoveryCmd.AddCommand(recoveryDetectCmd)
	recoveryCmd.AddCommand(recoveryExecuteCmd)
	recoveryCmd.AddCommand(recoveryStatusCmd)

	// execute 命令标志
	recoveryExecuteCmd.Flags().BoolVarP(&recoveryAuto, "auto", "a", false, "仅自动恢复可自动处理的问题")
	recoveryExecuteCmd.Flags().StringVarP(&recoveryComponent, "component", "c", "", "指定组件 (sysproxy, tun, route)")
	recoveryExecuteCmd.Flags().BoolVarP(&recoveryForce, "force", "F", false, "强制执行高风险操作（跳过确认）")
	recoveryExecuteCmd.Flags().StringVarP(&recoveryProblem, "problem", "p", "", "指定要修复的问题类型")
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

	output.Printf("检测到 %d 个问题:\n\n", len(problems))
	for i, problem := range problems {
		output.Printf("%d. [%s] %s\n", i+1, problem.Severity, problem.Description)
		output.Printf("   类型: %s\n", problem.Type)
		if len(problem.Solutions) > 0 {
			output.Println("   解决方案:")
			for j, solution := range problem.Solutions {
				auto := ""
				if solution.Auto {
					auto = " (可自动执行)"
				}
				output.Printf("   %d. %s%s\n", j+1, solution.Description, auto)
				if solution.Command != "" {
					output.Printf("      命令: %s\n", solution.Command)
				}
			}
		}
		output.PrintEmptyLine()
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

	// 使用配置中的超时时间
	config := mgr.GetConfig()
	ctx, cancel := context.WithTimeout(context.Background(), config.Timeout)
	defer cancel()

	var report *recovery.RecoveryReport

	if recoveryAuto {
		// 自动恢复（不强制执行高风险操作）
		report, err = mgr.AutoRecover(ctx)
	} else {
		// 手动恢复
		report, err = mgr.CheckAndRecoverWithFilter(ctx, recoveryForce, recoveryProblem)
	}

	if err != nil {
		return pkgerrors.ErrService("failed to execute recovery", err)
	}

	// 显示报告
	output.Println("恢复报告:")
	output.PrintKeyValue("时间", report.Timestamp.Format("2006-01-02 15:04:05"))
	output.PrintKeyValue("问题数量", len(report.Problems))
	output.PrintKeyValue("执行动作", len(report.Actions))
	output.PrintKeyValue("跳过问题", len(report.SkippedProblems))
	fmt.Fprint(output.GetGlobalStdout(), "  结果: ")
	if report.Success {
		output.Println("成功")
	} else {
		output.Println("失败")
		if report.ErrorMessage != "" {
			output.Printf("  错误: %s\n", report.ErrorMessage)
		}
	}
	output.PrintKeyValue("耗时", report.Duration)

	// 显示跳过的问题
	if len(report.SkippedProblems) > 0 {
		output.PrintEmptyLine()
		output.Warning("以下问题需要确认，已跳过（使用 -F/--force 强制执行）:")
		for i, problem := range report.SkippedProblems {
			output.Printf("%d. [%s] %s\n", i+1, problem.Severity, problem.Description)
			output.Printf("   类型: %s\n", problem.Type)
		}
	}

	if len(report.Actions) > 0 {
		output.PrintEmptyLine()
		output.Println("执行的动作:")
		for i, action := range report.Actions {
			output.Printf("%d. %s - %s\n", i+1, action.Action, action.Problem.Type)
			if action.Success {
				output.Println("   结果: 成功")
			} else {
				output.Println("   结果: 失败")
				if action.ErrorMessage != "" {
					output.Printf("   错误: %s\n", action.ErrorMessage)
				}
			}
			output.Printf("   耗时: %v\n", action.Duration)
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

	// 显示配置
	config := mgr.GetConfig()

	output.Println("恢复配置:")
	output.PrintKeyValue("启用", config.Enabled)
	output.PrintKeyValue("自动恢复", config.AutoRecover)
	output.PrintKeyValue("备份后恢复", config.BackupBeforeRecover)
	output.PrintKeyValue("超时时间", config.Timeout)
	output.PrintKeyValue("检查组件", config.Components)

	return nil
}
