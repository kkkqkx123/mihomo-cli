package cmd

import (
	"fmt"
	"strings"

	"github.com/fatih/color"
	"github.com/spf13/cobra"

	"github.com/kkkqkx123/mihomo-cli/internal/mihomo"
	pkgerrors "github.com/kkkqkx123/mihomo-cli/pkg/errors"
)

var psCmd = &cobra.Command{
	Use:   "ps",
	Short: "列出所有 Mihomo 进程",
	Long:  `列出所有正在运行的 Mihomo 进程及其详细信息。`,
	RunE: runPs,
}

func init() {
	rootCmd.AddCommand(psCmd)
}

func runPs(cmd *cobra.Command, args []string) error {
	// 扫描所有 Mihomo 进程
	processes, err := mihomo.ScanMihomoProcesses()
	if err != nil {
		return pkgerrors.ErrService("failed to scan processes", err)
	}

	if len(processes) == 0 {
		color.Yellow("没有找到正在运行的 Mihomo 进程")
		return nil
	}

	// 输出表头
	fmt.Printf("%-8s %-6s %-50s %-15s\n", "PID", "状态", "可执行文件", "API 端口")
	fmt.Println(strings.Repeat("-", 80))

	// 输出进程信息
	for _, proc := range processes {
		// 状态图标
		statusIcon := "✓"
		if !proc.IsVerified {
			statusIcon = "?"
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

		fmt.Printf("%-8d %-6s %-50s %-15s\n", proc.PID, statusIcon, execPath, apiPort)
	}

	// 输出汇总信息
	fmt.Println(strings.Repeat("-", 80))
	fmt.Printf("总计: %d 个进程\n", len(processes))
	if len(processes) > 0 {
		verifiedCount := 0
		for _, proc := range processes {
			if proc.IsVerified {
				verifiedCount++
			}
		}
		fmt.Printf("已验证: %d 个\n", verifiedCount)
	}

	// 输出提示
	fmt.Println()
	fmt.Println("提示:")
	fmt.Println("  mihomo-cli stop <pid>  - 停止指定进程")
	fmt.Println("  mihomo-cli stop --all  - 停止所有进程")
	fmt.Println("  mihomo-cli cleanup     - 清理残留的 PID 文件")

	return nil
}
