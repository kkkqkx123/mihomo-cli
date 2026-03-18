package config

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	pkgerrors "github.com/kkkqkx123/mihomo-cli/pkg/errors"
)

// BackupInfo 备份文件信息
type BackupInfo struct {
	Path      string    // 备份文件路径
	Size      int64     // 文件大小（字节）
	CreatedAt time.Time // 创建时间
	Note      string    // 备份备注
	Checksum  string    // 文件校验和（MD5）
}

// BackupManager 备份管理器
type BackupManager struct {
	configPath string // 配置文件路径
	backupDir  string // 备份目录
}

// NewBackupManager 创建备份管理器
func NewBackupManager(configPath string) *BackupManager {
	// 默认备份目录为 ~/.config/.mihomo-cli/backups/
	backupDir, _ := GetBackupDir()

	return &BackupManager{
		configPath: configPath,
		backupDir:  backupDir,
	}
}

// SetBackupDir 设置备份目录
func (bm *BackupManager) SetBackupDir(dir string) {
	bm.backupDir = dir
}

// GetBackupDir 获取备份目录
func (bm *BackupManager) GetBackupDir() string {
	return bm.backupDir
}

// CreateBackup 创建备份
func (bm *BackupManager) CreateBackup(note string) (*BackupInfo, error) {
	// 确保备份目录存在
	if err := os.MkdirAll(bm.backupDir, 0755); err != nil {
		return nil, pkgerrors.ErrService("创建备份目录失败", err)
	}

	// 读取原配置文件
	data, err := os.ReadFile(bm.configPath)
	if err != nil {
		return nil, pkgerrors.ErrConfig("读取配置文件失败", err)
	}

	// 计算校验和
	checksum := calculateChecksum(data)

	// 生成备份文件名
	timestamp := time.Now().Format("20060102-150405")
	baseName := filepath.Base(bm.configPath)
	ext := filepath.Ext(baseName)
	nameWithoutExt := strings.TrimSuffix(baseName, ext)

	var backupName string
	if note != "" {
		// 清理备注中的特殊字符
		cleanNote := sanitizeNote(note)
		backupName = fmt.Sprintf("%s.%s.%s%s", nameWithoutExt, timestamp, cleanNote, ext)
	} else {
		backupName = fmt.Sprintf("%s.%s%s", nameWithoutExt, timestamp, ext)
	}
	backupPath := filepath.Join(bm.backupDir, backupName)

	// 写入备份文件
	if err := os.WriteFile(backupPath, data, 0644); err != nil {
		return nil, pkgerrors.ErrService("创建备份文件失败", err)
	}

	// 获取文件信息
	fileInfo, err := os.Stat(backupPath)
	if err != nil {
		return nil, pkgerrors.ErrService("获取备份文件信息失败", err)
	}

	return &BackupInfo{
		Path:      backupPath,
		Size:      fileInfo.Size(),
		CreatedAt: fileInfo.ModTime(),
		Note:      note,
		Checksum:  checksum,
	}, nil
}

// ListBackups 列出所有备份
func (bm *BackupManager) ListBackups() ([]*BackupInfo, error) {
	// 检查备份目录是否存在
	if _, err := os.Stat(bm.backupDir); os.IsNotExist(err) {
		return []*BackupInfo{}, nil
	}

	// 获取配置文件基础名
	baseName := filepath.Base(bm.configPath)
	ext := filepath.Ext(baseName)
	nameWithoutExt := strings.TrimSuffix(baseName, ext)
	prefix := nameWithoutExt + "."

	// 读取备份目录
	entries, err := os.ReadDir(bm.backupDir)
	if err != nil {
		return nil, pkgerrors.ErrService("读取备份目录失败", err)
	}

	var backups []*BackupInfo
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		// 检查是否为该配置文件的备份
		if !strings.HasPrefix(entry.Name(), prefix) {
			continue
		}

		// 解析备份信息
		info, err := bm.parseBackupFile(filepath.Join(bm.backupDir, entry.Name()))
		if err != nil {
			continue
		}

		backups = append(backups, info)
	}

	// 按时间倒序排列（最新的在前）
	sort.Slice(backups, func(i, j int) bool {
		return backups[i].CreatedAt.After(backups[j].CreatedAt)
	})

	return backups, nil
}

// parseBackupFile 解析备份文件信息
func (bm *BackupManager) parseBackupFile(path string) (*BackupInfo, error) {
	fileInfo, err := os.Stat(path)
	if err != nil {
		return nil, err
	}

	// 读取文件内容计算校验和
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	// 解析文件名获取备注
	fileName := filepath.Base(path)
	note := bm.parseNoteFromFilename(fileName)

	return &BackupInfo{
		Path:      path,
		Size:      fileInfo.Size(),
		CreatedAt: fileInfo.ModTime(),
		Note:      note,
		Checksum:  calculateChecksum(data),
	}, nil
}

// parseNoteFromFilename 从文件名解析备注
func (bm *BackupManager) parseNoteFromFilename(filename string) string {
	// 文件名格式: {name}.{timestamp}.{note}.yaml 或 {name}.{timestamp}.yaml
	baseName := filepath.Base(bm.configPath)
	ext := filepath.Ext(baseName)
	nameWithoutExt := strings.TrimSuffix(baseName, ext)
	prefix := nameWithoutExt + "."

	// 移除前缀和扩展名
	name := strings.TrimPrefix(filename, prefix)
	name = strings.TrimSuffix(name, ext)

	// 分割时间戳和备注
	parts := strings.SplitN(name, ".", 2)
	if len(parts) == 2 {
		return parts[1]
	}

	return ""
}

// RestoreBackup 恢复备份
func (bm *BackupManager) RestoreBackup(backupPath string) error {
	// 读取备份文件
	data, err := os.ReadFile(backupPath)
	if err != nil {
		return pkgerrors.ErrConfig("读取备份文件失败", err)
	}

	// 写入配置文件
	if err := os.WriteFile(bm.configPath, data, 0644); err != nil {
		return pkgerrors.ErrService("恢复配置文件失败", err)
	}

	return nil
}

// DeleteBackup 删除备份
func (bm *BackupManager) DeleteBackup(backupPath string) error {
	if err := os.Remove(backupPath); err != nil {
		return pkgerrors.ErrService("删除备份文件失败", err)
	}
	return nil
}

// PruneBackups 清理旧备份
// keep: 保留最近 N 个备份
// olderThan: 删除超过指定天数的备份（0 表示不按时间删除）
func (bm *BackupManager) PruneBackups(keep int, olderThan time.Duration) ([]string, error) {
	backups, err := bm.ListBackups()
	if err != nil {
		return nil, err
	}

	var deleted []string
	now := time.Now()

	for i, backup := range backups {
		shouldDelete := false

		// 检查是否超过保留数量
		if keep > 0 && i >= keep {
			shouldDelete = true
		}

		// 检查是否超过保留时间
		if olderThan > 0 {
			age := now.Sub(backup.CreatedAt)
			if age > olderThan {
				shouldDelete = true
			}
		}

		if shouldDelete {
			if err := bm.DeleteBackup(backup.Path); err != nil {
				return deleted, err
			}
			deleted = append(deleted, backup.Path)
		}
	}

	return deleted, nil
}

// GetBackupByIndex 通过序号获取备份
func (bm *BackupManager) GetBackupByIndex(index int) (*BackupInfo, error) {
	backups, err := bm.ListBackups()
	if err != nil {
		return nil, err
	}

	if index < 1 || index > len(backups) {
		return nil, pkgerrors.ErrInvalidArg(fmt.Sprintf("无效的备份序号: %d，有效范围: 1-%d", index, len(backups)), nil)
	}

	return backups[index-1], nil
}

// GetBackupByPath 通过路径获取备份信息
func (bm *BackupManager) GetBackupByPath(path string) (*BackupInfo, error) {
	return bm.parseBackupFile(path)
}

// calculateChecksum 计算文件校验和
func calculateChecksum(data []byte) string {
	hash := md5.Sum(data)
	return hex.EncodeToString(hash[:])
}

// sanitizeNote 清理备注中的特殊字符
func sanitizeNote(note string) string {
	// 替换不安全的文件名字符
	replacer := strings.NewReplacer(
		"/", "-",
		"\\", "-",
		":", "-",
		"*", "-",
		"?", "-",
		"\"", "-",
		"<", "-",
		">", "-",
		"|", "-",
		" ", "_",
	)
	return replacer.Replace(note)
}

// FormatSize 格式化文件大小
func FormatSize(bytes int64) string {
	const (
		KB = 1024
		MB = KB * 1024
		GB = MB * 1024
	)

	switch {
	case bytes >= GB:
		return fmt.Sprintf("%.2fGB", float64(bytes)/float64(GB))
	case bytes >= MB:
		return fmt.Sprintf("%.2fMB", float64(bytes)/float64(MB))
	case bytes >= KB:
		return fmt.Sprintf("%.2fKB", float64(bytes)/float64(KB))
	default:
		return fmt.Sprintf("%dB", bytes)
	}
}
