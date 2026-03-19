package cmd

import (
	"time"

	"github.com/spf13/cobra"

	"github.com/kkkqkx123/mihomo-cli/internal/output"
	"github.com/kkkqkx123/mihomo-cli/internal/system"
	"github.com/kkkqkx123/mihomo-cli/internal/util"
	pkgerrors "github.com/kkkqkx123/mihomo-cli/pkg/errors"
)

var (
	cleanupSysProxy bool
	cleanupTUN      bool
	cleanupRoute    bool
	snapshotNote    string
	snapshotID      string
	auditComponent  string
	auditLimit      int
	auditSince      string
)

var systemCmd = &cobra.Command{
	Use:   "system",
	Short: "系统配置管理",
	Long:  "管理系统配置，包括系统代理、TUN 设备、路由表等。",
}

var systemStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "查询系统配置状态",
	Long:  "查询当前系统配置状态，包括系统代理、TUN 设备、路由表等。",
	RunE:  runSystemStatus,
}

var systemCleanupCmd = &cobra.Command{
	Use:   "cleanup",
	Short: "清理系统配置",
	Long:  "清理系统配置残留，包括系统代理、TUN 设备、路由表等。",
	RunE:  runSystemCleanup,
}

var systemValidateCmd = &cobra.Command{
	Use:   "validate",
	Short: "验证系统配置",
	Long:  "验证系统配置是否正常，检测是否有残留配置。",
	RunE:  runSystemValidate,
}

var systemSnapshotCmd = &cobra.Command{
	Use:   "snapshot",
	Short: "配置快照管理",
	Long:  "管理系统配置快照。",
}

var systemSnapshotCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "创建配置快照",
	Long:  "创建当前系统配置的快照。",
	RunE:  runSystemSnapshotCreate,
}

var systemSnapshotListCmd = &cobra.Command{
	Use:   "list",
	Short: "列出所有快照",
	Long:  "列出所有系统配置快照。",
	RunE:  runSystemSnapshotList,
}

var systemSnapshotRestoreCmd = &cobra.Command{
	Use:   "restore <snapshot-id>",
	Short: "恢复配置快照",
	Long:  "恢复指定的系统配置快照。",
	Args:  cobra.ExactArgs(1),
	RunE:  runSystemSnapshotRestore,
}

var systemSnapshotDeleteCmd = &cobra.Command{
	Use:   "delete <snapshot-id>",
	Short: "删除配置快照",
	Long:  "删除指定的系统配置快照。",
	Args:  cobra.ExactArgs(1),
	RunE:  runSystemSnapshotDelete,
}

var systemAuditCmd = &cobra.Command{
	Use:   "audit",
	Short: "审计日志管理",
	Long:  "管理系统配置审计日志。",
}

var systemAuditQueryCmd = &cobra.Command{
	Use:   "query",
	Short: "查询审计日志",
	Long:  "查询系统配置审计日志。",
	RunE:  runSystemAuditQuery,
}

var systemAuditClearCmd = &cobra.Command{
	Use:   "clear",
	Short: "清空审计日志",
	Long:  "清空所有系统配置审计日志。",
	RunE:  runSystemAuditClear,
}

func init() {
	rootCmd.AddCommand(systemCmd)

	// 添加子命令
	systemCmd.AddCommand(systemStatusCmd)
	systemCmd.AddCommand(systemCleanupCmd)
	systemCmd.AddCommand(systemValidateCmd)
	systemCmd.AddCommand(systemSnapshotCmd)
	systemCmd.AddCommand(systemAuditCmd)

	// 快照子命令
	systemSnapshotCmd.AddCommand(systemSnapshotCreateCmd)
	systemSnapshotCmd.AddCommand(systemSnapshotListCmd)
	systemSnapshotCmd.AddCommand(systemSnapshotRestoreCmd)
	systemSnapshotCmd.AddCommand(systemSnapshotDeleteCmd)

	// 审计子命令
	systemAuditCmd.AddCommand(systemAuditQueryCmd)
	systemAuditCmd.AddCommand(systemAuditClearCmd)

	// cleanup 命令标志
	systemCleanupCmd.Flags().BoolVar(&cleanupSysProxy, "sysproxy", true, "清理系统代理")
	systemCleanupCmd.Flags().BoolVar(&cleanupTUN, "tun", true, "清理 TUN 设备")
	systemCleanupCmd.Flags().BoolVar(&cleanupRoute, "route", true, "清理路由表")

	// snapshot create 命令标志
	systemSnapshotCreateCmd.Flags().StringVarP(&snapshotNote, "note", "n", "", "快照备注")

	// audit query 命令标志
	systemAuditQueryCmd.Flags().StringVarP(&auditComponent, "component", "c", "", "过滤组件 (sysproxy, tun, route)")
	systemAuditQueryCmd.Flags().IntVarP(&auditLimit, "limit", "l", 20, "限制返回数量")
	systemAuditQueryCmd.Flags().StringVar(&auditSince, "since", "", "起始时间 (格式: 2006-01-02)")
}

func runSystemStatus(cmd *cobra.Command, args []string) error {
	// 创建系统配置管理器
	mgr, err := system.NewSystemConfigManager()
	if err != nil {
		return pkgerrors.ErrService("failed to create system config manager", err)
	}

	// 获取配置状态
	state, err := mgr.GetConfigState()
	if err != nil {
		return pkgerrors.ErrService("failed to get system config state", err)
	}

	// 显示状态
	output.Println("系统配置状态:")
	output.Printf("  时间: %s\n", state.Timestamp.Format("2006-01-02 15:04:05"))
	output.Println()

	// 系统代理状态
	output.Println("系统代理:")
	if state.SysProxy != nil {
		if state.SysProxy.Enabled {
			output.Println("  状态: 已启用")
			output.Printf("  代理服务器: %s\n", state.SysProxy.Server)
			if state.SysProxy.BypassList != "" {
				output.Printf("  绕过列表: %s\n", state.SysProxy.BypassList)
			}
		} else {
			output.Println("  状态: 已禁用")
		}
	} else {
		output.Println("  状态: 未知")
	}
	output.Println()

	// TUN 设备状态
	output.Println("TUN 设备:")
	if state.TUN != nil {
		if state.TUN.Enabled {
			output.Println("  状态: 已启用")
			output.Printf("  设备名: %s\n", state.TUN.Name)
			if state.TUN.IPAddress != "" {
				output.Printf("  IP 地址: %s\n", state.TUN.IPAddress)
			}
			if state.TUN.MTU > 0 {
				output.Printf("  MTU: %d\n", state.TUN.MTU)
			}
		} else {
			output.Println("  状态: 未启用")
		}
	} else {
		output.Println("  状态: 未知")
	}
	output.Println()

	// 路由表状态
	output.Println("路由表:")
	if len(state.Routes) > 0 {
		output.Printf("  路由数量: %d\n", len(state.Routes))
		// 只显示前 5 条路由
		limit := 5
		if len(state.Routes) < limit {
			limit = len(state.Routes)
		}
		for i := 0; i < limit; i++ {
			route := state.Routes[i]
			output.Printf("  - %s via %s dev %s\n", route.Destination, route.Gateway, route.Interface)
		}
		if len(state.Routes) > 5 {
			output.Printf("  ... 还有 %d 条路由\n", len(state.Routes)-5)
		}
	} else {
		output.Println("  路由数量: 0")
	}

	return nil
}

func runSystemCleanup(cmd *cobra.Command, args []string) error {
	// 检查管理员权限
	if !util.IsAdmin() {
		return pkgerrors.ErrService("this operation requires administrator privileges, please run as administrator", nil)
	}

	// 创建系统配置管理器
	mgr, err := system.NewSystemConfigManager()
	if err != nil {
		return pkgerrors.ErrService("failed to create system config manager", err)
	}

	output.Println("开始清理系统配置...")

	// 清理系统代理
	if cleanupSysProxy {
		output.Println("清理系统代理...")
		if err := mgr.GetSysProxyManager().Cleanup(); err != nil {
			output.Printf("  警告: %v\n", err)
		} else {
			output.Println("  完成")
		}
	}

	// 清理 TUN 设备
	if cleanupTUN {
		output.Println("清理 TUN 设备...")
		if err := mgr.GetTUNManager().Cleanup(); err != nil {
			output.Printf("  警告: %v\n", err)
		} else {
			output.Println("  完成")
		}
	}

	// 清理路由表
	if cleanupRoute {
		output.Println("清理路由表...")
		if err := mgr.GetRouteManager().Cleanup(); err != nil {
			output.Printf("  警告: %v\n", err)
		} else {
			output.Println("  完成")
		}
	}

	output.Println("清理完成")

	return nil
}

func runSystemValidate(cmd *cobra.Command, args []string) error {
	// 创建系统配置管理器
	mgr, err := system.NewSystemConfigManager()
	if err != nil {
		return pkgerrors.ErrService("failed to create system config manager", err)
	}

	// 验证配置状态
	problems, err := mgr.ValidateState()
	if err != nil {
		return pkgerrors.ErrService("failed to validate system config", err)
	}

	if len(problems) == 0 {
		output.Println("系统配置状态正常，未检测到问题")
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
		output.Println()
	}

	return nil
}

func runSystemSnapshotCreate(cmd *cobra.Command, args []string) error {
	// 创建系统配置管理器
	mgr, err := system.NewSystemConfigManager()
	if err != nil {
		return pkgerrors.ErrService("failed to create system config manager", err)
	}

	// 创建快照
	snapshot, err := mgr.CreateSnapshot(snapshotNote)
	if err != nil {
		return pkgerrors.ErrService("failed to create snapshot", err)
	}

	output.Println("配置快照已创建:")
	output.Printf("  ID: %s\n", snapshot.ID)
	output.Printf("  时间: %s\n", snapshot.CreatedAt.Format("2006-01-02 15:04:05"))
	if snapshot.Note != "" {
		output.Printf("  备注: %s\n", snapshot.Note)
	}

	return nil
}

func runSystemSnapshotList(cmd *cobra.Command, args []string) error {
	// 创建系统配置管理器
	mgr, err := system.NewSystemConfigManager()
	if err != nil {
		return pkgerrors.ErrService("failed to create system config manager", err)
	}

	// 列出快照
	snapshots, err := mgr.ListSnapshots()
	if err != nil {
		return pkgerrors.ErrService("failed to list snapshots", err)
	}

	if len(snapshots) == 0 {
		output.Println("没有找到配置快照")
		return nil
	}

	output.Printf("找到 %d 个配置快照:\n\n", len(snapshots))
	for i, snapshot := range snapshots {
		output.Printf("%d. ID: %s\n", i+1, snapshot.ID)
		output.Printf("   时间: %s\n", snapshot.CreatedAt.Format("2006-01-02 15:04:05"))
		if snapshot.Note != "" {
			output.Printf("   备注: %s\n", snapshot.Note)
		}
		output.Println()
	}

	return nil
}

func runSystemSnapshotRestore(cmd *cobra.Command, args []string) error {
	// 检查管理员权限
	if !util.IsAdmin() {
		return pkgerrors.ErrService("this operation requires administrator privileges, please run as administrator", nil)
	}

	snapshotID := args[0]

	// 创建系统配置管理器
	mgr, err := system.NewSystemConfigManager()
	if err != nil {
		return pkgerrors.ErrService("failed to create system config manager", err)
	}

	// 恢复快照
	if err := mgr.RestoreSnapshot(snapshotID); err != nil {
		return pkgerrors.ErrService("failed to restore snapshot", err)
	}

	output.Printf("配置快照 %s 已恢复\n", snapshotID)

	return nil
}

func runSystemSnapshotDelete(cmd *cobra.Command, args []string) error {
	snapshotID := args[0]

	// 创建系统配置管理器
	mgr, err := system.NewSystemConfigManager()
	if err != nil {
		return pkgerrors.ErrService("failed to create system config manager", err)
	}

	// 删除快照
	if err := mgr.DeleteSnapshot(snapshotID); err != nil {
		return pkgerrors.ErrService("failed to delete snapshot", err)
	}

	output.Printf("配置快照 %s 已删除\n", snapshotID)

	return nil
}

func runSystemAuditQuery(cmd *cobra.Command, args []string) error {
	// 创建系统配置管理器
	mgr, err := system.NewSystemConfigManager()
	if err != nil {
		return pkgerrors.ErrService("failed to create system config manager", err)
	}

	// 解析时间
	var since time.Time
	if auditSince != "" {
		since, err = time.Parse("2006-01-02", auditSince)
		if err != nil {
			return pkgerrors.ErrInvalidArg("invalid time format, use YYYY-MM-DD", nil)
		}
	}

	// 查询审计日志
	records, err := mgr.QueryAuditLog(auditComponent, since, auditLimit)
	if err != nil {
		return pkgerrors.ErrService("failed to query audit log", err)
	}

	if len(records) == 0 {
		output.Println("没有找到审计日志")
		return nil
	}

	output.Printf("找到 %d 条审计日志:\n\n", len(records))
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

func runSystemAuditClear(cmd *cobra.Command, args []string) error {
	// 创建系统配置管理器
	mgr, err := system.NewSystemConfigManager()
	if err != nil {
		return pkgerrors.ErrService("failed to create system config manager", err)
	}

	// 清空审计日志
	if err := mgr.ClearAuditLog(); err != nil {
		return pkgerrors.ErrService("failed to clear audit log", err)
	}

	output.Println("审计日志已清空")

	return nil
}
