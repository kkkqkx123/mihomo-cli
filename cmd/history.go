package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/kkkqkx123/mihomo-cli/internal/config"
	"github.com/kkkqkx123/mihomo-cli/internal/history"
	"github.com/kkkqkx123/mihomo-cli/internal/output"
	pkgerrors "github.com/kkkqkx123/mihomo-cli/pkg/errors"
)

var (
	historyLimit int
	historyYes   bool
)

// NewHistoryCmd 创建历史记录命令
func NewHistoryCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "history",
		Short: "查看命令历史记录",
		Long:  "查看所有执行过的命令历史记录",
		RunE:  runHistory,
	}

	clearCmd := &cobra.Command{
		Use:   "clear",
		Short: "清除历史记录",
		Long:  "清除所有命令历史记录。非交互环境（无终端）下需使用 --yes 确认。",
		RunE:  runHistoryClear,
	}
	clearCmd.Flags().BoolVarP(&historyYes, "yes", "y", false, "跳过确认直接清除")
	cmd.AddCommand(clearCmd)

	cmd.Flags().IntVarP(&historyLimit, "limit", "l", 50, "显示记录数量")

	return cmd
}

// runHistory 执行查看历史记录
func runHistory(cmd *cobra.Command, args []string) error {
	// 获取历史目录
	historyDir, err := config.GetHistoryDir()
	if err != nil {
		return pkgerrors.ErrConfig("获取历史目录失败", err)
	}

	historyFile := historyDir + "/commands.jsonl"
	historyManager := history.NewManager(historyFile)

	// 读取历史记录
	entries, err := historyManager.Read()
	if err != nil {
		return pkgerrors.ErrService("读取历史记录失败", err)
	}

	if len(entries) == 0 {
		output.PrintInfo("暂无历史记录")
		return nil
	}

	// 限制显示数量
	if historyLimit > 0 && historyLimit < len(entries) {
		entries = entries[len(entries)-historyLimit:]
	}

	// 格式化输出
	if output.GetGlobalFormat() == "json" {
		return output.PrintJSON(entries)
	}

	// 表格输出
	headers := []string{"时间", "命令", "状态"}
	var rows [][]string

	for _, entry := range entries {
		status := "✓"
		if !entry.Success {
			status = "✗"
		}
		timeStr := entry.Timestamp.Format("2006-01-02 15:04:05")
		rows = append(rows, []string{timeStr, entry.Command, status})
	}

	return output.PrintTable(headers, rows)
}

// runHistoryClear 执行清除历史记录
func runHistoryClear(cmd *cobra.Command, args []string) error {
	// 确认：交互式终端提示输入；非交互环境（脚本/管道）必须显式使用 --yes，避免阻塞等待输入
	if !historyYes {
		if !stdinIsInteractive() {
			return pkgerrors.ErrInvalidArg("非交互环境清除历史记录需要确认，请使用 --yes 参数", nil)
		}

		output.PrintRaw("确定要清除所有历史记录吗？(y/N): ")
		var confirm string
		_, _ = fmt.Scanln(&confirm)

		if confirm != "y" && confirm != "Y" {
			output.PrintInfo("操作已取消")
			return nil
		}
	}

	// 获取历史目录
	historyDir, err := config.GetHistoryDir()
	if err != nil {
		return pkgerrors.ErrConfig("获取历史目录失败", err)
	}

	historyFile := historyDir + "/commands.jsonl"
	historyManager := history.NewManager(historyFile)

	// 清除历史记录
	if err := historyManager.Clear(); err != nil {
		return pkgerrors.ErrService("清除历史记录失败", err)
	}

	output.PrintSuccess("历史记录已清除")
	return nil
}

// stdinIsInteractive 判断标准输入是否为交互式终端
func stdinIsInteractive() bool {
	fi, err := os.Stdin.Stat()
	if err != nil {
		return false
	}
	return fi.Mode()&os.ModeCharDevice != 0
}
