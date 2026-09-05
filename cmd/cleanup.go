package cmd

import (
	"github.com/spf13/cobra"

	"github.com/kkkqkx123/mihomo-cli/internal/mihomo"
	"github.com/kkkqkx123/mihomo-cli/internal/output"
	pkgerrors "github.com/kkkqkx123/mihomo-cli/pkg/errors"
)

var cleanupCmd = &cobra.Command{
	Use:   "cleanup",
	Short: "清理残留的 PID 文件",
	Long:  `清理所有残留的 PID 文件（进程已退出但 PID 文件仍存在）。`,
	RunE:  runCleanup,
}

func init() {
	rootCmd.AddCommand(cleanupCmd)
}

func runCleanup(cmd *cobra.Command, args []string) error {
	output.Println("正在检查残留的 PID 文件...")
	output.Println()

	err := mihomo.CleanupPIDFiles()
	if err != nil {
		return pkgerrors.ErrService("清理 PID 文件失败", err)
	}

	output.Println()
	output.Success("清理完成")
	return nil
}
