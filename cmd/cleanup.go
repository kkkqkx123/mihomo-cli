package cmd

import (
	"fmt"

	"github.com/fatih/color"
	"github.com/spf13/cobra"

	"github.com/kkkqkx123/mihomo-cli/internal/mihomo"
	pkgerrors "github.com/kkkqkx123/mihomo-cli/pkg/errors"
)

var cleanupCmd = &cobra.Command{
	Use:   "cleanup",
	Short: "清理残留的 PID 文件",
	Long:  `清理所有残留的 PID 文件（进程已退出但 PID 文件仍存在）。`,
	RunE: runCleanup,
}

func init() {
	rootCmd.AddCommand(cleanupCmd)
}

func runCleanup(cmd *cobra.Command, args []string) error {
	fmt.Println("正在检查残留的 PID 文件...")
	fmt.Println()

	err := mihomo.CleanupPIDFiles()
	if err != nil {
		return pkgerrors.ErrService("cleanup failed", err)
	}

	fmt.Println()
	color.Green("✓ 清理完成")
	return nil
}