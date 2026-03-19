package system

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// TUNManager TUN 网卡管理器
type TUNManager struct {
	audit *AuditLogger
}

// NewTUNManager 创建 TUN 管理器
func NewTUNManager(audit *AuditLogger) *TUNManager {
	return &TUNManager{
		audit: audit,
	}
}

// ListTUNDevices 列出所有 TUN 设备
func (tm *TUNManager) ListTUNDevices() ([]TUNState, error) {
	return tm.listTUNDevices()
}

// CheckMihomoTUN 检查 Mihomo 创建的 TUN 设备
func (tm *TUNManager) CheckMihomoTUN() ([]TUNState, error) {
	devices, err := tm.ListTUNDevices()
	if err != nil {
		return nil, err
	}

	// 过滤出 Mihomo 相关的 TUN 设备
	var mihomoDevices []TUNState
	for _, dev := range devices {
		// Mihomo 通常使用 "utun" 或 "tun" 作为前缀
		if tm.isMihomoTUNDevice(dev.Name) {
			mihomoDevices = append(mihomoDevices, dev)
		}
	}

	return mihomoDevices, nil
}

// isMihomoTUNDevice 检查是否是 Mihomo 创建的 TUN 设备
func (tm *TUNManager) isMihomoTUNDevice(name string) bool {
	// Mihomo 通常使用以下前缀
	prefixes := []string{"utun", "tun", "clash", "mihomo"}
	for _, prefix := range prefixes {
		if len(name) >= len(prefix) && name[:len(prefix)] == prefix {
			return true
		}
	}
	return false
}

// RemoveTUN 删除 TUN 设备
func (tm *TUNManager) RemoveTUN(name string) error {
	err := tm.removeTUN(name)

	if tm.audit != nil {
		result := "success"
		if err != nil {
			result = "failed"
		}
		_ = tm.audit.Record("remove", "tun", name, result, err)
	}

	return err
}

// GetState 获取 TUN 状态
func (tm *TUNManager) GetState() (*TUNState, error) {
	devices, err := tm.CheckMihomoTUN()
	if err != nil {
		return nil, err
	}

	if len(devices) == 0 {
		return &TUNState{Enabled: false}, nil
	}

	// 返回第一个设备的状态
	dev := devices[0]
	return &TUNState{
		Name:      dev.Name,
		Enabled:   true,
		IPAddress: dev.IPAddress,
		MTU:       dev.MTU,
	}, nil
}

// CheckResidual 检查是否有残留 TUN 设备
func (tm *TUNManager) CheckResidual() (*Problem, error) {
	devices, err := tm.CheckMihomoTUN()
	if err != nil {
		return nil, err
	}

	if len(devices) > 0 {
		deviceNames := make([]string, len(devices))
		for i, dev := range devices {
			deviceNames[i] = dev.Name
		}

		return &Problem{
			Type:        ProblemConfigResidual,
			Severity:    SeverityHigh,
			Description: "TUN devices created by Mihomo still exist",
			Details: map[string]interface{}{
				"devices": deviceNames,
			},
			Solutions: []Solution{
				{
					Description: "Remove TUN devices",
					Command:     "mihomo-cli system cleanup --tun",
					Auto:        true,
				},
				{
					Description: "Restart Mihomo to cleanup",
					Command:     "mihomo-cli restart",
					Auto:        true,
				},
				{
					Description: "Restart system to cleanup",
					Command:     "restart computer",
					Auto:        false,
				},
			},
		}, nil
	}

	return nil, nil
}

// Cleanup 清理 TUN 设备
func (tm *TUNManager) Cleanup() error {
	devices, err := tm.CheckMihomoTUN()
	if err != nil {
		return err
	}

	var lastErr error
	for _, dev := range devices {
		if err := tm.RemoveTUN(dev.Name); err != nil {
			lastErr = err
		}
	}

	return lastErr
}

// TUNBackup TUN 接口状态备份
type TUNBackup struct {
	ID        string    `json:"id"`
	Timestamp time.Time `json:"timestamp"`
	States    []TUNState `json:"states"`
	Note      string    `json:"note,omitempty"`
}

// BackupTUNState 备份当前 TUN 接口状态
func (tm *TUNManager) BackupTUNState(note string) (*TUNBackup, error) {
	devices, err := tm.ListTUNDevices()
	if err != nil {
		return nil, fmt.Errorf("failed to list TUN devices: %w", err)
	}

	// 生成备份 ID
	id := time.Now().Format("20060102-150405")

	backup := &TUNBackup{
		ID:        id,
		Timestamp: time.Now(),
		States:    devices,
		Note:      note,
	}

	// 记录审计日志
	if tm.audit != nil {
		_ = tm.audit.Record("backup", "tun", fmt.Sprintf("backed up %d TUN devices", len(devices)), "success", nil)
	}

	return backup, nil
}

// SaveTUNBackup 保存 TUN 备份到文件
func (tm *TUNManager) SaveTUNBackup(backup *TUNBackup) (string, error) {
	dataDir, err := GetDataDir()
	if err != nil {
		return "", fmt.Errorf("failed to get data directory: %w", err)
	}

	// 确保目录存在
	backupDir := filepath.Join(dataDir, "tun-backups")
	if err := os.MkdirAll(backupDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create backup directory: %w", err)
	}

	// 序列化备份
	data, err := json.MarshalIndent(backup, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal backup: %w", err)
	}

	// 保存到文件
	filename := fmt.Sprintf("tun-backup-%s.json", backup.ID)
	backupPath := filepath.Join(backupDir, filename)
	if err := os.WriteFile(backupPath, data, 0644); err != nil {
		return "", fmt.Errorf("failed to save backup: %w", err)
	}

	return backupPath, nil
}

// LoadTUNBackup 从文件加载 TUN 备份
func (tm *TUNManager) LoadTUNBackup(id string) (*TUNBackup, error) {
	dataDir, err := GetDataDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get data directory: %w", err)
	}

	backupDir := filepath.Join(dataDir, "tun-backups")
	filename := fmt.Sprintf("tun-backup-%s.json", id)
	backupPath := filepath.Join(backupDir, filename)

	data, err := os.ReadFile(backupPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read backup: %w", err)
	}

	var backup TUNBackup
	if err := json.Unmarshal(data, &backup); err != nil {
		return nil, fmt.Errorf("failed to unmarshal backup: %w", err)
	}

	return &backup, nil
}

// ListTUNBackups 列出所有 TUN 备份
func (tm *TUNManager) ListTUNBackups() ([]TUNBackup, error) {
	dataDir, err := GetDataDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get data directory: %w", err)
	}

	backupDir := filepath.Join(dataDir, "tun-backups")
	entries, err := os.ReadDir(backupDir)
	if err != nil {
		if os.IsNotExist(err) {
			return []TUNBackup{}, nil
		}
		return nil, fmt.Errorf("failed to read backup directory: %w", err)
	}

	var backups []TUNBackup
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		// 只处理备份文件
		if !isTUNBackupFile(entry.Name()) {
			continue
		}

		// 提取备份 ID
		id := extractTUNBackupID(entry.Name())
		if id == "" {
			continue
		}

		// 加载备份
		backup, err := tm.LoadTUNBackup(id)
		if err != nil {
			continue
		}

		backups = append(backups, *backup)
	}

	// 按时间倒序排列（最新的在前）
	sort.Slice(backups, func(i, j int) bool {
		return backups[i].Timestamp.After(backups[j].Timestamp)
	})

	return backups, nil
}

// CompareTUNState 比较当前 TUN 状态与备份的差异
func (tm *TUNManager) CompareTUNState(backup *TUNBackup) (*TUNDiff, error) {
	currentDevices, err := tm.ListTUNDevices()
	if err != nil {
		return nil, fmt.Errorf("failed to list current TUN devices: %w", err)
	}

	diff := &TUNDiff{
		BackupTimestamp: backup.Timestamp,
		CurrentTimestamp: time.Now(),
		AddedDevices:     []TUNState{},
		RemovedDevices:   []TUNState{},
		ModifiedDevices:  []TUNChange{},
	}

	// 创建备份设备的索引
	backupIndex := make(map[string]TUNState)
	for _, dev := range backup.States {
		key := dev.Name
		backupIndex[key] = dev
	}

	// 创建当前设备的索引
	currentIndex := make(map[string]TUNState)
	for _, dev := range currentDevices {
		key := dev.Name
		currentIndex[key] = dev
	}

	// 查找新增的设备（存在于当前但不存在于备份中）
	for _, dev := range currentDevices {
		key := dev.Name
		if _, exists := backupIndex[key]; !exists {
			diff.AddedDevices = append(diff.AddedDevices, dev)
		}
	}

	// 查找删除的设备（存在于备份但不存在于当前中）
	for _, dev := range backup.States {
		key := dev.Name
		if _, exists := currentIndex[key]; !exists {
			diff.RemovedDevices = append(diff.RemovedDevices, dev)
		}
	}

	// 查找修改的设备（两者都存在但属性不同）
	for _, currentDev := range currentDevices {
		key := currentDev.Name
		if backupDev, exists := backupIndex[key]; exists {
			if !tunStatesEqual(currentDev, backupDev) {
				diff.ModifiedDevices = append(diff.ModifiedDevices, TUNChange{
					Old: backupDev,
					New: currentDev,
				})
			}
		}
	}

	return diff, nil
}

// RestoreTUNState 恢复 TUN 接口状态
// 注意：此功能主要用于清理残留设备，而不是恢复整个 TUN 配置
func (tm *TUNManager) RestoreTUNState(backupID string, mode string) (*TUNRestoreResult, error) {
	// 加载备份
	backup, err := tm.LoadTUNBackup(backupID)
	if err != nil {
		return nil, fmt.Errorf("failed to load backup: %w", err)
	}

	result := &TUNRestoreResult{
		BackupID: backupID,
		Mode:     mode,
	}

	currentDevices, err := tm.ListTUNDevices()
	if err != nil {
		return nil, fmt.Errorf("failed to list current TUN devices: %w", err)
	}

	// 创建当前设备的索引
	currentIndex := make(map[string]TUNState)
	for _, dev := range currentDevices {
		currentIndex[dev.Name] = dev
	}

	// 根据模式执行恢复
	switch mode {
	case "cleanup":
		// 清理模式：删除当前存在但备份中不存在的设备
		// 或者删除所有 Mihomo 相关的设备

		// 创建备份设备的索引
		backupIndex := make(map[string]TUNState)
		for _, state := range backup.States {
			backupIndex[state.Name] = state
		}

		for _, currentDev := range currentDevices {
			key := currentDev.Name
			if _, exists := backupIndex[key]; !exists {
				// 这个设备不在备份中，检查是否是 Mihomo 设备
				if tm.isMihomoTUNDevice(currentDev.Name) {
					if err := tm.RemoveTUN(currentDev.Name); err != nil {
						result.Errors = append(result.Errors, err)
						result.FailedDevices = append(result.FailedDevices, currentDev)
					} else {
						result.RemovedDevices = append(result.RemovedDevices, currentDev)
					}
				}
			}
		}

	case "restore":
		// 恢复模式：恢复备份中的设备
		// 注意：TUN 设备通常由 Mihomo 创建，这里主要是记录状态
		// 实际恢复需要重启 Mihomo
		for _, backupDev := range backup.States {
			if tm.isMihomoTUNDevice(backupDev.Name) {
				// 检查当前是否存在
				key := backupDev.Name
				if _, exists := currentIndex[key]; !exists {
					// 不存在，记录需要恢复
					result.NeedRestoreDevices = append(result.NeedRestoreDevices, backupDev)
				}
			}
		}

	default:
		return nil, fmt.Errorf("unsupported restore mode: %s", mode)
	}

	result.Success = len(result.Errors) == 0

	// 记录审计日志
	if tm.audit != nil {
		_ = tm.audit.Record("restore", "tun", fmt.Sprintf("mode=%s, removed=%d, need_restore=%d", mode, len(result.RemovedDevices), len(result.NeedRestoreDevices)), "success", nil)
	}

	return result, nil
}

// DeleteTUNBackup 删除 TUN 备份
func (tm *TUNManager) DeleteTUNBackup(id string) error {
	dataDir, err := GetDataDir()
	if err != nil {
		return fmt.Errorf("failed to get data directory: %w", err)
	}

	backupDir := filepath.Join(dataDir, "tun-backups")
	filename := fmt.Sprintf("tun-backup-%s.json", id)
	backupPath := filepath.Join(backupDir, filename)

	if err := os.Remove(backupPath); err != nil {
		return fmt.Errorf("failed to delete backup: %w", err)
	}

	// 记录审计日志
	if tm.audit != nil {
		_ = tm.audit.Record("delete", "tun", id, "success", nil)
	}

	return nil
}

// PruneTUNBackups 清理旧备份
func (tm *TUNManager) PruneTUNBackups(keep int, olderThan time.Duration) ([]string, error) {
	backups, err := tm.ListTUNBackups()
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
			age := now.Sub(backup.Timestamp)
			if age > olderThan {
				shouldDelete = true
			}
		}

		if shouldDelete {
			if err := tm.DeleteTUNBackup(backup.ID); err != nil {
				return deleted, err
			}
			deleted = append(deleted, backup.ID)
		}
	}

	return deleted, nil
}

// TUNDiff TUN 差异
type TUNDiff struct {
	BackupTimestamp  time.Time   `json:"backup_timestamp"`
	CurrentTimestamp time.Time   `json:"current_timestamp"`
	AddedDevices     []TUNState  `json:"added_devices"`
	RemovedDevices   []TUNState  `json:"removed_devices"`
	ModifiedDevices  []TUNChange `json:"modified_devices"`
}

// TUNChange TUN 变更
type TUNChange struct {
	Old TUNState `json:"old"`
	New TUNState `json:"new"`
}

// TUNRestoreResult TUN 恢复结果
type TUNRestoreResult struct {
	BackupID           string     `json:"backup_id"`
	Mode               string     `json:"mode"`
	Success            bool       `json:"success"`
	RemovedDevices     []TUNState `json:"removed_devices"`
	NeedRestoreDevices []TUNState `json:"need_restore_devices"` // 需要恢复的设备（通常需要重启 Mihomo）
	FailedDevices      []TUNState `json:"failed_devices"`
	Errors             []error    `json:"errors"`
}

// tunStatesEqual 比较两个 TUN 状态是否相等
func tunStatesEqual(a, b TUNState) bool {
	return a.Name == b.Name &&
		a.Enabled == b.Enabled &&
		a.IPAddress == b.IPAddress &&
		a.MTU == b.MTU
}

// isTUNBackupFile 检查是否是 TUN 备份文件
func isTUNBackupFile(filename string) bool {
	return len(filename) > len("tun-backup-") &&
		filename[:len("tun-backup-")] == "tun-backup-" &&
		len(filename) > 5 &&
		filename[len(filename)-5:] == ".json"
}

// extractTUNBackupID 从文件名提取备份 ID
func extractTUNBackupID(filename string) string {
	prefix := "tun-backup-"
	suffix := ".json"

	if !strings.HasPrefix(filename, prefix) {
		return ""
	}

	if !strings.HasSuffix(filename, suffix) {
		return ""
	}

	return filename[len(prefix) : len(filename)-len(suffix)]
}
