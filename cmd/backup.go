package cmd

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/kkkqkx123/mihomo-cli/internal/api"
	"github.com/kkkqkx123/mihomo-cli/internal/config"
	"github.com/kkkqkx123/mihomo-cli/internal/output"
)

var backupCmd = &cobra.Command{
	Use:   "backup",
	Short: "管理配置备份",
	Long:  `管理 Mihomo 配置文件的备份，包括创建、查看、恢复和删除备份。`,
}

func init() {
	rootCmd.AddCommand(backupCmd)
	backupCmd.AddCommand(newBackupCreateCmd())
	backupCmd.AddCommand(newBackupListCmd())
	backupCmd.AddCommand(newBackupRestoreCmd())
	backupCmd.AddCommand(newBackupDeleteCmd())
	backupCmd.AddCommand(newBackupPruneCmd())
}

// newBackupCreateCmd 创建 backup create 命令
func newBackupCreateCmd() *cobra.Command {
	var mihomoConfigPath string
	var note string

	cmd := &cobra.Command{
		Use:   "create",
		Short: "创建配置备份",
		Long:  `手动创建 Mihomo 配置文件的备份。`,
		Example: `  mihomo-cli backup create
  mihomo-cli backup create -n "before-update"
  mihomo-cli backup create -p /path/to/config.yaml -n "manual-backup"`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runBackupCreate(mihomoConfigPath, note)
		},
	}

	cmd.Flags().StringVarP(&mihomoConfigPath, "path", "p", "", "Mihomo 配置文件路径")
	cmd.Flags().StringVarP(&note, "note", "n", "", "备份备注")

	return cmd
}

// runBackupCreate 执行创建备份
func runBackupCreate(mihomoConfigPath, note string) error {
	handler := config.NewBackupHandler("")

	// 创建备份
	info, err := handler.CreateBackup(mihomoConfigPath, note)
	if err != nil {
		return err
	}

	output.Success("备份创建成功")
	output.PrintKeyValue("配置文件", mihomoConfigPath)
	output.PrintKeyValue("备份文件", info.Path)
	output.PrintKeyValue("文件大小", config.FormatSize(info.Size))
	if note != "" {
		output.PrintKeyValue("备注", note)
	}

	return nil
}

// newBackupListCmd 创建 backup list 命令
func newBackupListCmd() *cobra.Command {
	var mihomoConfigPath string

	cmd := &cobra.Command{
		Use:   "list",
		Short: "列出所有备份",
		Long:  `列出 Mihomo 配置文件的所有备份。`,
		Example: `  mihomo-cli backup list
  mihomo-cli backup list -p /path/to/config.yaml`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runBackupList(mihomoConfigPath)
		},
	}

	cmd.Flags().StringVarP(&mihomoConfigPath, "path", "p", "", "Mihomo 配置文件路径")

	return cmd
}

// runBackupList 执行列出备份
func runBackupList(mihomoConfigPath string) error {
	handler := config.NewBackupHandler("")

	// 获取备份列表
	backups, err := handler.ListBackups(mihomoConfigPath)
	if err != nil {
		return err
	}

	if len(backups) == 0 {
		output.Warning("没有找到备份文件")
		return nil
	}

	// 显示备份列表
	output.Println("备份列表:")
	table := output.NewTable()
	table.SetHeader([]string{"序号", "时间", "大小", "备注"})

	for i, backup := range backups {
		timeStr := backup.CreatedAt.Format("2006-01-02 15:04:05")
		sizeStr := config.FormatSize(backup.Size)
		note := backup.Note
		if note == "" {
			note = "-"
		}
		_ = table.Append([]string{
			fmt.Sprintf("%d", i+1),
			timeStr,
			sizeStr,
			note,
		})
	}

	return table.Render()
}

// newBackupRestoreCmd 创建 backup restore 命令
func newBackupRestoreCmd() *cobra.Command {
	var mihomoConfigPath string
	var noReload bool

	cmd := &cobra.Command{
		Use:   "restore <备份文件|序号>",
		Short: "恢复配置备份",
		Long:  `从指定的备份文件恢复 Mihomo 配置。恢复前会自动备份当前配置。`,
		Example: `  mihomo-cli backup restore 1
  mihomo-cli backup restore /path/to/backup.yaml
  mihomo-cli backup restore 1 --no-reload`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runBackupRestore(cmd.Context(), mihomoConfigPath, args[0], noReload)
		},
	}

	cmd.Flags().StringVarP(&mihomoConfigPath, "path", "p", "", "Mihomo 配置文件路径")
	cmd.Flags().BoolVar(&noReload, "no-reload", false, "恢复后不自动重载配置")

	return cmd
}

// runBackupRestore 执行恢复备份
func runBackupRestore(ctx context.Context, mihomoConfigPath, backupRef string, noReload bool) error {
	handler := config.NewBackupHandler("")

	// 创建 API 客户端用于重载配置
	var client *api.Client
	if !noReload {
		client = api.NewClientWithTimeout(
			viper.GetString("api.address"),
			viper.GetString("api.secret"),
			viper.GetInt("api.timeout"),
		)
	}

	// 恢复备份
	result, err := handler.RestoreBackupWithClient(ctx, mihomoConfigPath, backupRef, noReload, client)
	if err != nil {
		return err
	}

	output.Cyan("已备份当前配置: %s", result.CurrentBackup.Path)
	output.Success("配置已恢复")
	output.PrintKeyValue("恢复源", result.BackupPath)
	output.PrintKeyValue("配置文件", result.ConfigPath)

	if result.ReloadError != nil {
		output.Warning("重载配置失败: %v", result.ReloadError)
		output.Warning("配置文件已恢复，但未生效，请手动重启服务")
	} else if result.Reloaded {
		output.Success("配置已重载生效")
	}

	return nil
}

// newBackupDeleteCmd 创建 backup delete 命令
func newBackupDeleteCmd() *cobra.Command {
	var mihomoConfigPath string
	var deleteAll bool
	var keep int
	var olderThan int

	cmd := &cobra.Command{
		Use:   "delete [备份文件|序号]",
		Short: "删除配置备份",
		Long:  `删除指定的 Mihomo 配置备份文件。`,
		Example: `  mihomo-cli backup delete 1
  mihomo-cli backup delete /path/to/backup.yaml
  mihomo-cli backup delete --all
  mihomo-cli backup delete --keep 5
  mihomo-cli backup delete --older-than 30`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runBackupDelete(mihomoConfigPath, args, deleteAll, keep, olderThan)
		},
	}

	cmd.Flags().StringVarP(&mihomoConfigPath, "path", "p", "", "Mihomo 配置文件路径")
	cmd.Flags().BoolVar(&deleteAll, "all", false, "删除所有备份")
	cmd.Flags().IntVarP(&keep, "keep", "k", 0, "保留最近 N 个备份，删除其余")
	cmd.Flags().IntVar(&olderThan, "older-than", 0, "删除超过指定天数的备份")

	return cmd
}

// runBackupDelete 执行删除备份
func runBackupDelete(mihomoConfigPath string, args []string, deleteAll bool, keep, olderThan int) error {
	handler := config.NewBackupHandler("")

	// 删除备份
	result, err := handler.DeleteBackup(mihomoConfigPath, args, deleteAll, keep, olderThan)
	if err != nil {
		return err
	}

	if deleteAll {
		for _, path := range result.Deleted {
			output.Printf("已删除: %s\n", path)
		}
		for path, err := range result.Failed {
			output.Warning("删除失败: %s - %v", path, err)
		}
		output.Success("已删除所有备份")
		return nil
	}

	if keep > 0 || olderThan > 0 {
		for _, path := range result.Deleted {
			output.Printf("已删除: %s\n", path)
		}
		output.Success("已删除 %d 个备份", len(result.Deleted))
		return nil
	}

	// 单个备份删除
	for _, path := range result.Deleted {
		output.Success("备份已删除: %s", path)
	}

	return nil
}

// newBackupPruneCmd 创建 backup prune 命令
func newBackupPruneCmd() *cobra.Command {
	var mihomoConfigPath string
	var keep int
	var olderThan int
	var dryRun bool

	cmd := &cobra.Command{
		Use:   "prune",
		Short: "清理旧备份",
		Long:  `按策略清理旧的 Mihomo 配置备份文件。`,
		Example: `  mihomo-cli backup prune
  mihomo-cli backup prune --keep 5
  mihomo-cli backup prune --older-than 30
  mihomo-cli backup prune --dry-run`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runBackupPrune(mihomoConfigPath, keep, olderThan, dryRun)
		},
	}

	cmd.Flags().StringVarP(&mihomoConfigPath, "path", "p", "", "Mihomo 配置文件路径")
	cmd.Flags().IntVarP(&keep, "keep", "k", 10, "保留最近 N 个备份")
	cmd.Flags().IntVar(&olderThan, "older-than", 0, "删除超过指定天数的备份")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "仅显示将被删除的备份，不实际删除")

	return cmd
}

// runBackupPrune 执行清理备份
func runBackupPrune(mihomoConfigPath string, keep, olderThan int, dryRun bool) error {
	handler := config.NewBackupHandler("")

	// 清理备份
	result, err := handler.PruneBackups(mihomoConfigPath, keep, olderThan, dryRun)
	if err != nil {
		return err
	}

	if len(result.ToDelete) == 0 {
		output.Success("没有需要清理的备份")
		return nil
	}

	if dryRun {
		output.Cyan("将删除以下 %d 个备份:", len(result.ToDelete))
		for _, path := range result.ToDelete {
			output.PrintIndent(1, path)
		}
		return nil
	}

	// 执行删除
	for _, path := range result.Deleted {
		output.Printf("已删除: %s\n", path)
	}
	for path, err := range result.Failed {
		output.Warning("删除失败: %s - %v", path, err)
	}

	output.Success("已清理 %d 个备份", len(result.Deleted))
	return nil
}
