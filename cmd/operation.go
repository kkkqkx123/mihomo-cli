package cmd

import (
	"time"

	"github.com/spf13/cobra"

	"github.com/kkkqkx123/mihomo-cli/internal/output"
	"github.com/kkkqkx123/mihomo-cli/internal/system"
	pkgerrors "github.com/kkkqkx123/mihomo-cli/pkg/errors"
)

var (
	operationComponent string
	operationLimit     int
	operationSince     string
	operationBefore    string
)

var operationCmd = &cobra.Command{
	Use:   "operation",
	Short: "操作记录管理",
	Long:  "管理系统配置操作记录，包括查询、清理等。",
}

var operationQueryCmd = &cobra.Command{
	Use:   "query",
	Short: "查询操作记录",
	Long:  "查询系统配置操作记录。",
	RunE:  runOperationQuery,
}

var operationClearCmd = &cobra.Command{
	Use:   "clear",
	Short: "清空操作记录",
	Long:  "清空所有操作记录。",
	RunE:  runOperationClear,
}

var operationPruneCmd = &cobra.Command{
	Use:   "prune",
	Short: "清理旧操作记录",
	Long:  "清理指定时间之前的操作记录。",
	RunE:  runOperationPrune,
}

func init() {
	rootCmd.AddCommand(operationCmd)

	// 添加子命令
	operationCmd.AddCommand(operationQueryCmd)
	operationCmd.AddCommand(operationClearCmd)
	operationCmd.AddCommand(operationPruneCmd)

	// query 命令标志
	operationQueryCmd.Flags().StringVarP(&operationComponent, "component", "c", "", "过滤组件 (sysproxy, tun, route, snapshot)")
	operationQueryCmd.Flags().IntVarP(&operationLimit, "limit", "l", 20, "限制返回数量")
	operationQueryCmd.Flags().StringVar(&operationSince, "since", "", "起始时间 (格式: 2006-01-02)")

	// prune 命令标志
	operationPruneCmd.Flags().StringVar(&operationBefore, "before", "", "清理此时间之前的记录 (格式: 2006-01-02)")
}

func runOperationQuery(cmd *cobra.Command, args []string) error {
	// 创建系统配置管理器
	mgr, err := system.NewSystemConfigManager()
	if err != nil {
		return pkgerrors.ErrService("failed to create system config manager", err)
	}

	// 解析时间
	var since time.Time
	if operationSince != "" {
		since, err = time.Parse("2006-01-02", operationSince)
		if err != nil {
			return pkgerrors.ErrInvalidArg("invalid time format, use YYYY-MM-DD", nil)
		}
	}

	// 查询操作记录
	records, err := mgr.QueryOperationLog(operationComponent, since, operationLimit)
	if err != nil {
		return pkgerrors.ErrService("failed to query operation log", err)
	}

	if len(records) == 0 {
		output.Println("没有找到操作记录")
		return nil
	}

	output.Printf("找到 %d 条操作记录:\n\n", len(records))
	for i, record := range records {
		output.Printf("%d. [%s] %s.%s\n", i+1, record.Timestamp.Format("2006-01-02 15:04:05"), record.Component, record.Operation)
		if record.Details != "" {
			output.Printf("   详情: %s\n", record.Details)
		}
		output.Printf("   结果: %s\n", record.Result)
		if record.Error != "" {
			output.Printf("   错误: %s\n", record.Error)
		}
		output.Println()
	}

	return nil
}

func runOperationClear(cmd *cobra.Command, args []string) error {
	// 创建系统配置管理器
	mgr, err := system.NewSystemConfigManager()
	if err != nil {
		return pkgerrors.ErrService("failed to create system config manager", err)
	}

	// 清空操作记录
	if err := mgr.ClearOperationLog(); err != nil {
		return pkgerrors.ErrService("failed to clear operation log", err)
	}

	output.Println("操作记录已清空")

	return nil
}

func runOperationPrune(cmd *cobra.Command, args []string) error {
	// 解析时间
	if operationBefore == "" {
		return pkgerrors.ErrInvalidArg("--before is required, use format YYYY-MM-DD", nil)
	}
	before, err := time.Parse("2006-01-02", operationBefore)
	if err != nil {
		return pkgerrors.ErrInvalidArg("invalid time format, use YYYY-MM-DD", nil)
	}

	// 创建系统配置管理器
	mgr, err := system.NewSystemConfigManager()
	if err != nil {
		return pkgerrors.ErrService("failed to create system config manager", err)
	}

	// 清理操作记录
	removed, err := mgr.PruneOperationLog(before)
	if err != nil {
		return pkgerrors.ErrService("failed to prune operation log", err)
	}

	output.Printf("已清理 %d 条操作记录 (时间早于 %s)\n", removed, operationBefore)

	return nil
}
