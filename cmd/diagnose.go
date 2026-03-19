package cmd

import (
	"fmt"
	"path/filepath"
	"sort"

	"github.com/kkkqkx123/mihomo-cli/internal/config"
	"github.com/kkkqkx123/mihomo-cli/internal/operation"
	"github.com/kkkqkx123/mihomo-cli/internal/output"
	"github.com/kkkqkx123/mihomo-cli/internal/system"
	"github.com/spf13/cobra"
)

// diagnoseCmd 诊断命令
var diagnoseCmd = &cobra.Command{
	Use:   "diagnose",
	Short: "Diagnose system issues",
	Long:  "Diagnose system issues including routes, network, and Mihomo residual configurations",
}

var (
	diagnoseFix        bool
	diagnoseFormat     string
)

func init() {
	rootCmd.AddCommand(diagnoseCmd)

	// 子命令
	diagnoseCmd.AddCommand(diagnoseRouteCmd)
	diagnoseCmd.AddCommand(diagnoseNetworkCmd)

	// 全局标志
	diagnoseCmd.PersistentFlags().BoolVarP(&diagnoseFix, "fix", "f", false, "Automatically fix issues")
	diagnoseCmd.PersistentFlags().StringVarP(&diagnoseFormat, "output", "o", "table", "Output format (table, json)")
}

// diagnoseRouteCmd 诊断路由命令
var diagnoseRouteCmd = &cobra.Command{
	Use:   "route",
	Short: "Diagnose route table issues",
	Long:  "Diagnose route table issues including residual routes and conflicts",
	RunE: func(cmd *cobra.Command, args []string) error {
		return runDiagnoseRoute()
	},
}

// diagnoseNetworkCmd 诊断网络命令
var diagnoseNetworkCmd = &cobra.Command{
	Use:   "network",
	Short: "Diagnose network issues",
	Long:  "Diagnose network issues including routing and connectivity",
	RunE: func(cmd *cobra.Command, args []string) error {
		return runDiagnoseNetwork()
	},
}

func runDiagnoseRoute() error {
	// 创建路由管理器
	op, err := createOperationManager()
	if err != nil {
		return err
	}
	routeManager := system.NewRouteManager(op)

	// 诊断路由
	diagnosis, err := routeManager.DiagnoseNetworkRouting()
	if err != nil {
		output.Error("failed to diagnose routes: %v", err)
		return err
	}

	// 输出诊断结果
	if diagnoseFormat == "json" {
		return output.PrintJSON(diagnosis)
	}

	// 表格格式输出
	return printRouteDiagnosis(diagnosis)
}

func runDiagnoseNetwork() error {
	// 创建路由管理器
	op, err := createOperationManager()
	if err != nil {
		return err
	}
	routeManager := system.NewRouteManager(op)

	// 诊断网络
	diagnosis, err := routeManager.DiagnoseNetworkRouting()
	if err != nil {
		output.Error("failed to diagnose network: %v", err)
		return err
	}

	// 输出诊断结果
	if diagnoseFormat == "json" {
		return output.PrintJSON(diagnosis)
	}

	// 表格格式输出
	return printNetworkDiagnosis(diagnosis)
}

// createOperationManager 创建操作记录管理器（使用统一的数据目录）
func createOperationManager() (*operation.Manager, error) {
	dataDir, err := config.GetDataDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get data directory: %w", err)
	}
	operationFile := filepath.Join(dataDir, "operation.log")
	return operation.NewManager(operationFile)
}

// printRouteDiagnosis 打印路由诊断结果（表格格式）
func printRouteDiagnosis(diagnosis *system.NetworkDiagnosis) error {
	// 打印健康状态
	output.Success(fmt.Sprintf("Route Health: %s", diagnosis.Health))

	// 打印默认路由冲突
	if len(diagnosis.DefaultRouteConflicts) > 0 {
		output.Warning(fmt.Sprintf("\nFound %d default route conflicts:", len(diagnosis.DefaultRouteConflicts)))

		table := output.NewTable()
		table.SetHeader([]string{"Type", "Severity", "Message", "Recommendation"})

		for _, conflict := range diagnosis.DefaultRouteConflicts {
			if err := table.Append([]string{
				conflict.Type,
				conflict.Severity,
				conflict.Message,
				conflict.Recommendation,
			}); err != nil {
				return fmt.Errorf("failed to append table row: %w", err)
			}
		}

		if err := table.Render(); err != nil {
			return fmt.Errorf("failed to render table: %w", err)
		}
	}

	// 打印残留路由
	if len(diagnosis.ResidualRoutes) > 0 {
		output.Warning(fmt.Sprintf("\nFound %d residual routes:", len(diagnosis.ResidualRoutes)))

		table := output.NewTable()
		table.SetHeader([]string{"Destination", "Gateway", "Interface", "Reason", "Issue"})

		for _, residual := range diagnosis.ResidualRoutes {
			if err := table.Append([]string{
				residual.Route.Destination,
				residual.Route.Gateway,
				residual.Route.Interface,
				residual.Reason,
				residual.Issue,
			}); err != nil {
				return fmt.Errorf("failed to append table row: %w", err)
			}
		}

		if err := table.Render(); err != nil {
			return fmt.Errorf("failed to render table: %w", err)
		}

		// 提示清理命令
		output.Info("\nTo cleanup residual routes, run:")
		output.Info("  mihomo-cli system cleanup --route")
	}

	// 如果没有问题
	if len(diagnosis.DefaultRouteConflicts) == 0 && len(diagnosis.ResidualRoutes) == 0 {
		output.Success("\nNo route issues found")
	}

	return nil
}

// printNetworkDiagnosis 打印网络诊断结果（表格格式）
func printNetworkDiagnosis(diagnosis *system.NetworkDiagnosis) error {
	// 打印健康状态
	output.Success(fmt.Sprintf("Network Health: %s", diagnosis.Health))

	// 打印默认路由冲突
	if len(diagnosis.DefaultRouteConflicts) > 0 {
		output.Warning(fmt.Sprintf("\nFound %d default route conflicts:", len(diagnosis.DefaultRouteConflicts)))

		// 按严重程度排序
		conflicts := diagnosis.DefaultRouteConflicts
		sort.Slice(conflicts, func(i, j int) bool {
			severityOrder := map[string]int{"Critical": 0, "High": 1, "Medium": 2, "Low": 3}
			return severityOrder[conflicts[i].Severity] < severityOrder[conflicts[j].Severity]
		})

		for _, conflict := range conflicts {
			if conflict.Severity == "Critical" {
				output.Error(fmt.Sprintf("\n[CRITICAL] %s", conflict.Message))
			} else if conflict.Severity == "High" {
				output.Warning(fmt.Sprintf("\n[WARNING] %s", conflict.Message))
			} else {
				output.Info(fmt.Sprintf("\n[INFO] %s", conflict.Message))
			}

			// 打印相关路由
			table := output.NewTable()
			table.SetHeader([]string{"Destination", "Gateway", "Interface", "Metric"})

			for _, route := range conflict.Routes {
				if err := table.Append([]string{
					route.Destination,
					route.Gateway,
					route.Interface,
					fmt.Sprintf("%d", route.Metric),
				}); err != nil {
					return fmt.Errorf("failed to append table row: %w", err)
				}
			}

			if err := table.Render(); err != nil {
				return fmt.Errorf("failed to render table: %w", err)
			}

			// 打印建议
			output.Info(fmt.Sprintf("Recommendation: %s", conflict.Recommendation))
		}
	}

	// 打印残留路由
	if len(diagnosis.ResidualRoutes) > 0 {
		output.Warning(fmt.Sprintf("\nFound %d residual routes:", len(diagnosis.ResidualRoutes)))

		table := output.NewTable()
		table.SetHeader([]string{"Destination", "Gateway", "Interface", "Interface Exists", "Gateway Reachable", "Reason"})

		for _, residual := range diagnosis.ResidualRoutes {
			ifaceStatus := "✓"
			if !residual.InterfaceExists {
				ifaceStatus = "✗"
			}

			gwStatus := "✓"
			if !residual.GatewayReachable {
				gwStatus = "✗"
			}

			if err := table.Append([]string{
				residual.Route.Destination,
				residual.Route.Gateway,
				residual.Route.Interface,
				ifaceStatus,
				gwStatus,
				residual.Reason,
			}); err != nil {
				return fmt.Errorf("failed to append table row: %w", err)
			}
		}

		if err := table.Render(); err != nil {
			return fmt.Errorf("failed to render table: %w", err)
		}

		// 提示清理命令
		output.Info("\nTo cleanup residual routes, run:")
		output.Info("  mihomo-cli system cleanup --route")
	}

	// 如果没有问题
	if len(diagnosis.DefaultRouteConflicts) == 0 && len(diagnosis.ResidualRoutes) == 0 {
		output.Success("\nNo network issues found")
	}

	return nil
}