package config

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/kkkqkx123/mihomo-cli/internal/api"
)

// BackupHandler 备份管理处理器
type BackupHandler struct {
	configPath string
	backupDir  string
}

// NewBackupHandler 创建备份处理器
func NewBackupHandler(configPath string) *BackupHandler {
	backupDir, _ := GetBackupDir()

	return &BackupHandler{
		configPath: configPath,
		backupDir:  backupDir,
	}
}

// SetBackupDir 设置备份目录
func (bh *BackupHandler) SetBackupDir(dir string) {
	bh.backupDir = dir
}

// FindConfigPath 查找 Mihomo 配置文件路径
func FindConfigPath(mihomoConfigPath string) (string, error) {
	if mihomoConfigPath != "" {
		// 检查文件是否存在
		if _, err := os.Stat(mihomoConfigPath); os.IsNotExist(err) {
			return "", fmt.Errorf("配置文件不存在: %s", mihomoConfigPath)
		}
		return mihomoConfigPath, nil
	}

	// 使用 FindTomlConfigPath 查找配置文件
	tomlConfigPath := FindTomlConfigPath("")
	tomlCfg, err := LoadTomlConfig(tomlConfigPath)
	if err == nil && tomlCfg.Mihomo.ConfigFile != "" {
		if _, err := os.Stat(tomlCfg.Mihomo.ConfigFile); err == nil {
			return tomlCfg.Mihomo.ConfigFile, nil
		}
	}

	// 尝试从标准位置查找
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("获取用户目录失败: %w", err)
	}

	// 只检查标准路径，保持一致性
	candidatePaths := []string{
		filepath.Join(home, ".config", "mihomo", "config.yaml"),
		"./config.yaml",
	}

	for _, p := range candidatePaths {
		if _, err := os.Stat(p); err == nil {
			return p, nil
		}
	}

	return "", fmt.Errorf("未找到 Mihomo 配置文件，请使用 --path 参数指定路径或配置 config.toml")
}

// CreateBackup 创建备份
func (bh *BackupHandler) CreateBackup(mihomoConfigPath, note string) (*BackupInfo, error) {
	// 确定配置文件路径
	configPath, err := FindConfigPath(mihomoConfigPath)
	if err != nil {
		return nil, err
	}

	// 创建备份管理器
	bm := NewBackupManager(configPath)

	// 创建备份
	info, err := bm.CreateBackup(note)
	if err != nil {
		return nil, fmt.Errorf("创建备份失败: %w", err)
	}

	return info, nil
}

// ListBackups 列出所有备份
func (bh *BackupHandler) ListBackups(mihomoConfigPath string) ([]*BackupInfo, error) {
	// 确定配置文件路径
	configPath, err := FindConfigPath(mihomoConfigPath)
	if err != nil {
		return nil, err
	}

	// 创建备份管理器
	bm := NewBackupManager(configPath)

	// 获取备份列表
	backups, err := bm.ListBackups()
	if err != nil {
		return nil, fmt.Errorf("获取备份列表失败: %w", err)
	}

	return backups, nil
}

// RestoreBackup 恢复备份
func (bh *BackupHandler) RestoreBackup(ctx context.Context, mihomoConfigPath, backupRef string, noReload bool) (*RestoreResult, error) {
	return bh.RestoreBackupWithClient(ctx, mihomoConfigPath, backupRef, noReload, nil)
}

// RestoreBackupWithClient 恢复备份（指定 API 客户端）
func (bh *BackupHandler) RestoreBackupWithClient(ctx context.Context, mihomoConfigPath, backupRef string, noReload bool, client *api.Client) (*RestoreResult, error) {
	// 确定配置文件路径
	configPath, err := FindConfigPath(mihomoConfigPath)
	if err != nil {
		return nil, err
	}

	// 创建备份管理器
	bm := NewBackupManager(configPath)

	// 确定备份文件路径
	var backupPath string
	if _, err := os.Stat(backupRef); err == nil {
		// 是有效的文件路径
		backupPath = backupRef
	} else {
		// 尝试解析为序号
		var index int
		if _, err := fmt.Sscanf(backupRef, "%d", &index); err == nil {
			backupInfo, err := bm.GetBackupByIndex(index)
			if err != nil {
				return nil, err
			}
			backupPath = backupInfo.Path
		} else {
			return nil, fmt.Errorf("无效的备份引用: %s，请使用序号或有效的备份文件路径", backupRef)
		}
	}

	// 恢复前先备份当前配置
	currentBackup, err := bm.CreateBackup("pre-restore")
	if err != nil {
		return nil, fmt.Errorf("恢复前备份当前配置失败: %w", err)
	}

	// 恢复备份
	if err := bm.RestoreBackup(backupPath); err != nil {
		return nil, fmt.Errorf("恢复备份失败: %w", err)
	}

	result := &RestoreResult{
		BackupPath:    backupPath,
		ConfigPath:    configPath,
		CurrentBackup: currentBackup,
	}

	// 如果需要重载
	if !noReload && client != nil {
		// 重载配置
		if err := client.ReloadConfig(ctx, configPath, false); err != nil {
			result.ReloadError = err
			return result, nil
		}

		result.Reloaded = true
	}

	return result, nil
}

// RestoreResult 恢复结果
type RestoreResult struct {
	BackupPath    string
	ConfigPath    string
	CurrentBackup *BackupInfo
	Reloaded      bool
	ReloadError   error
}

// DeleteBackup 删除备份
func (bh *BackupHandler) DeleteBackup(mihomoConfigPath string, args []string, deleteAll bool, keep, olderThan int) (*DeleteResult, error) {
	// 确定配置文件路径
	configPath, err := FindConfigPath(mihomoConfigPath)
	if err != nil {
		return nil, err
	}

	// 创建备份管理器
	bm := NewBackupManager(configPath)

	result := &DeleteResult{
		Deleted: []string{},
		Failed:  map[string]error{},
	}

	// 处理批量删除选项
	if deleteAll {
		backups, err := bm.ListBackups()
		if err != nil {
			return nil, err
		}

		if len(backups) == 0 {
			return result, nil
		}

		for _, backup := range backups {
			if err := bm.DeleteBackup(backup.Path); err != nil {
				result.Failed[backup.Path] = err
			} else {
				result.Deleted = append(result.Deleted, backup.Path)
			}
		}

		return result, nil
	}

	if keep > 0 || olderThan > 0 {
		deleted, err := bm.PruneBackups(keep, time.Duration(olderThan)*24*time.Hour)
		if err != nil {
			return nil, err
		}

		result.Deleted = deleted
		return result, nil
	}

	// 删除单个备份
	if len(args) == 0 {
		return nil, fmt.Errorf("请指定要删除的备份文件或序号")
	}

	backupRef := args[0]
	var backupPath string

	if _, err := os.Stat(backupRef); err == nil {
		backupPath = backupRef
	} else {
		var index int
		if _, err := fmt.Sscanf(backupRef, "%d", &index); err == nil {
			backupInfo, err := bm.GetBackupByIndex(index)
			if err != nil {
				return nil, err
			}
			backupPath = backupInfo.Path
		} else {
			return nil, fmt.Errorf("无效的备份引用: %s", backupRef)
		}
	}

	if err := bm.DeleteBackup(backupPath); err != nil {
		return nil, err
	}

	result.Deleted = append(result.Deleted, backupPath)
	return result, nil
}

// DeleteResult 删除结果
type DeleteResult struct {
	Deleted []string
	Failed  map[string]error
}

// PruneBackups 清理旧备份
func (bh *BackupHandler) PruneBackups(mihomoConfigPath string, keep, olderThan int, dryRun bool) (*PruneResult, error) {
	// 确定配置文件路径
	configPath, err := FindConfigPath(mihomoConfigPath)
	if err != nil {
		return nil, err
	}

	// 创建备份管理器
	bm := NewBackupManager(configPath)

	// 获取备份列表
	backups, err := bm.ListBackups()
	if err != nil {
		return nil, err
	}

	if len(backups) == 0 {
		return &PruneResult{}, nil
	}

	// 确定要删除的备份
	var toDelete []string
	now := time.Now()

	for i, backup := range backups {
		shouldDelete := false

		if keep > 0 && i >= keep {
			shouldDelete = true
		}

		if olderThan > 0 {
			age := now.Sub(backup.CreatedAt)
			if age > time.Duration(olderThan)*24*time.Hour {
				shouldDelete = true
			}
		}

		if shouldDelete {
			toDelete = append(toDelete, backup.Path)
		}
	}

	result := &PruneResult{
		ToDelete: toDelete,
	}

	if dryRun {
		return result, nil
	}

	// 执行删除
	for _, path := range toDelete {
		if err := bm.DeleteBackup(path); err != nil {
			result.Failed[path] = err
		} else {
			result.Deleted = append(result.Deleted, path)
		}
	}

	return result, nil
}

// PruneResult 清理结果
type PruneResult struct {
	ToDelete []string
	Deleted  []string
	Failed   map[string]error
}