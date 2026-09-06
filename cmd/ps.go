package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/kkkqkx123/mihomo-cli/internal/mihomo"
	"github.com/kkkqkx123/mihomo-cli/internal/output"
	pkgerrors "github.com/kkkqkx123/mihomo-cli/pkg/errors"
)

var psCmd = &cobra.Command{
	Use:   "ps",
	Short: "列出所有 Mihomo 进程",
	Long:  `列出所有正在运行的 Mihomo 进程及其详细信息（托管实例与外部启动实例）。`,
	RunE:  runPs,
}

func init() {
	rootCmd.AddCommand(psCmd)
}

func runPs(cmd *cobra.Command, args []string) error {
	// 扫描所有 Mihomo 进程（managed + external 两源合并）
	processes, err := mihomo.ScanMihomoProcesses()
	if err != nil {
		return pkgerrors.ErrService("failed to scan processes", err)
	}

	if len(processes) == 0 {
		output.Info("没有找到正在运行的 Mihomo 进程")
		return nil
	}

	// 创建表格
	table := output.NewTable()
	table.SetHeader([]string{"PID", "状态", "来源", "验证", "可执行文件", "API 端口"})

	// 输出进程信息
	for _, proc := range processes {
		// 状态图标
		statusIcon := output.StatusOK()
		if !proc.IsVerified {
			statusIcon = output.StatusUnknown()
		}

		// 来源
		source := "外部"
		if proc.Source == mihomo.SourceManaged {
			source = "托管"
		}

		// 验证级别
		verified := "疑似"
		if proc.Verified == mihomo.VerifyConfirmed {
			verified = "确认"
		}

		// API 端口
		apiPort := proc.APIPort
		if apiPort == "" {
			apiPort = "未知"
		}

		// 可执行文件路径（截取显示）
		execPath := proc.ExecPath
		if len(execPath) > 50 {
			execPath = "..." + execPath[len(execPath)-47:]
		}

		_ = table.Append([]string{
			fmt.Sprintf("%d", proc.PID),
			statusIcon,
			source,
			verified,
			execPath,
			apiPort,
		})
	}

	// 渲染表格
	if err := table.Render(); err != nil {
		return err
	}

	// 输出汇总信息
	output.PrintSeparator("-", 80)
	output.Printf("总计: %d 个进程\n", len(processes))
	confirmedCount := 0
	managedCount := 0
	for _, proc := range processes {
		if proc.Verified == mihomo.VerifyConfirmed {
			confirmedCount++
		}
		if proc.Source == mihomo.SourceManaged {
			managedCount++
		}
	}
	output.Printf("已确认（托管且路径一致）: %d 个\n", confirmedCount)
	output.Printf("托管实例: %d 个，外部实例: %d 个\n", managedCount, len(processes)-managedCount)

	// 输出提示
	output.PrintEmptyLine()
	output.Println("提示:")
	output.Println("  mihomo-cli stop <pid>                - 停止指定进程")
	output.Println("  mihomo-cli stop --all                - 停止所有已确认的托管进程")
	output.Println("  mihomo-cli stop --all --include-unmanaged - 停止所有进程（含外部实例）")
	output.Println("  mihomo-cli cleanup                   - 清理残留的 PID 文件")

	return nil
}
