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

// RouteBackup 路由表备份
type RouteBackup struct {
	ID        string       `json:"id"`
	Timestamp time.Time    `json:"timestamp"`
	Routes    []RouteEntry `json:"routes"`
	Note      string       `json:"note,omitempty"`
}

// RouteDiff 路由差异
type RouteDiff struct {
	BackupTimestamp  time.Time     `json:"backup_timestamp"`
	CurrentTimestamp time.Time     `json:"current_timestamp"`
	AddedRoutes      []RouteEntry  `json:"added_routes"`
	RemovedRoutes    []RouteEntry  `json:"removed_routes"`
	ModifiedRoutes   []RouteChange `json:"modified_routes"`
}

// RouteChange 路由变更
type RouteChange struct {
	Old RouteEntry `json:"old"`
	New RouteEntry `json:"new"`
}

// RestoreResult 恢复结果
type RestoreResult struct {
	BackupID      string       `json:"backup_id"`
	Mode          string       `json:"mode"`
	Success       bool         `json:"success"`
	AddedRoutes   []RouteEntry `json:"added_routes"`
	DeletedRoutes []RouteEntry `json:"deleted_routes"`
	FailedRoutes  []RouteEntry `json:"failed_routes"`
	Errors        []error      `json:"errors"`
}

// BackupRoutes 备份当前路由表
func (rm *RouteManager) BackupRoutes(note string) (*RouteBackup, error) {
	routes, err := rm.ListRoutes()
	if err != nil {
		return nil, fmt.Errorf("failed to list routes: %w", err)
	}

	// 生成备份 ID
	id := time.Now().Format("20060102-150405")

	backup := &RouteBackup{
		ID:        id,
		Timestamp: time.Now(),
		Routes:    routes,
		Note:      note,
	}

	// 记录操作日志
	if rm.operation != nil {
		_ = rm.operation.Record("backup", "route", fmt.Sprintf("backed up %d routes", len(routes)), "success", nil)
	}

	return backup, nil
}

// SaveBackup 保存路由表备份到文件
func (rm *RouteManager) SaveBackup(backup *RouteBackup) (string, error) {
	dataDir, err := GetDataDir()
	if err != nil {
		return "", fmt.Errorf("failed to get data directory: %w", err)
	}

	// 确保目录存在
	backupDir := filepath.Join(dataDir, "route-backups")
	if err := os.MkdirAll(backupDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create backup directory: %w", err)
	}

	// 序列化备份
	data, err := json.MarshalIndent(backup, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal backup: %w", err)
	}

	// 保存到文件
	filename := fmt.Sprintf("route-backup-%s.json", backup.ID)
	backupPath := filepath.Join(backupDir, filename)
	if err := os.WriteFile(backupPath, data, 0644); err != nil {
		return "", fmt.Errorf("failed to save backup: %w", err)
	}

	return backupPath, nil
}

// LoadBackup 从文件加载路由表备份
func (rm *RouteManager) LoadBackup(id string) (*RouteBackup, error) {
	dataDir, err := GetDataDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get data directory: %w", err)
	}

	backupDir := filepath.Join(dataDir, "route-backups")
	filename := fmt.Sprintf("route-backup-%s.json", id)
	backupPath := filepath.Join(backupDir, filename)

	data, err := os.ReadFile(backupPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read backup: %w", err)
	}

	var backup RouteBackup
	if err := json.Unmarshal(data, &backup); err != nil {
		return nil, fmt.Errorf("failed to unmarshal backup: %w", err)
	}

	return &backup, nil
}

// ListBackups 列出所有路由表备份
func (rm *RouteManager) ListBackups() ([]RouteBackup, error) {
	dataDir, err := GetDataDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get data directory: %w", err)
	}

	backupDir := filepath.Join(dataDir, "route-backups")
	entries, err := os.ReadDir(backupDir)
	if err != nil {
		if os.IsNotExist(err) {
			return []RouteBackup{}, nil
		}
		return nil, fmt.Errorf("failed to read backup directory: %w", err)
	}

	var backups []RouteBackup
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		// 只处理备份文件
		if !isRouteBackupFile(entry.Name()) {
			continue
		}

		// 提取备份 ID
		id := extractBackupID(entry.Name())
		if id == "" {
			continue
		}

		// 加载备份
		backup, err := rm.LoadBackup(id)
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

// CompareRoutes 比较当前路由表与备份的差异
func (rm *RouteManager) CompareRoutes(backup *RouteBackup) (*RouteDiff, error) {
	currentRoutes, err := rm.ListRoutes()
	if err != nil {
		return nil, fmt.Errorf("failed to list current routes: %w", err)
	}

	diff := &RouteDiff{
		BackupTimestamp:  backup.Timestamp,
		CurrentTimestamp: time.Now(),
		AddedRoutes:      []RouteEntry{},
		RemovedRoutes:    []RouteEntry{},
		ModifiedRoutes:   []RouteChange{},
	}

	// 创建备份路由的索引
	backupIndex := make(map[string]RouteEntry)
	for _, route := range backup.Routes {
		key := routeKey(route)
		backupIndex[key] = route
	}

	// 创建当前路由的索引
	currentIndex := make(map[string]RouteEntry)
	for _, route := range currentRoutes {
		key := routeKey(route)
		currentIndex[key] = route
	}

	// 查找新增的路由（存在于当前但不存在于备份中）
	for _, route := range currentRoutes {
		key := routeKey(route)
		if _, exists := backupIndex[key]; !exists {
			diff.AddedRoutes = append(diff.AddedRoutes, route)
		}
	}

	// 查找删除的路由（存在于备份但不存在于当前中）
	for _, route := range backup.Routes {
		key := routeKey(route)
		if _, exists := currentIndex[key]; !exists {
			diff.RemovedRoutes = append(diff.RemovedRoutes, route)
		}
	}

	// 查找修改的路由（两者都存在但属性不同）
	for _, currentRoute := range currentRoutes {
		key := routeKey(currentRoute)
		if backupRoute, exists := backupIndex[key]; exists {
			if !routesEqual(currentRoute, backupRoute) {
				diff.ModifiedRoutes = append(diff.ModifiedRoutes, RouteChange{
					Old: backupRoute,
					New: currentRoute,
				})
			}
		}
	}

	return diff, nil
}

// RestoreRoutes 恢复路由表
// 注意：此功能主要用于清理残留路由，而不是完全恢复整个路由表
func (rm *RouteManager) RestoreRoutes(backupID string, mode string) (*RestoreResult, error) {
	// 加载备份
	backup, err := rm.LoadBackup(backupID)
	if err != nil {
		return nil, fmt.Errorf("failed to load backup: %w", err)
	}

	result := &RestoreResult{
		BackupID: backupID,
		Mode:     mode,
	}

	currentRoutes, err := rm.ListRoutes()
	if err != nil {
		return nil, fmt.Errorf("failed to list current routes: %w", err)
	}

	// 创建当前路由的索引
	currentIndex := make(map[string]RouteEntry)
	for _, route := range currentRoutes {
		key := routeKey(route)
		currentIndex[key] = route
	}

	// 根据模式执行恢复
	switch mode {
	case "cleanup":
		// 清理模式：删除所有存在于备份中但不存在于当前中的路由
		// 实际上，这应该是删除当前路由中存在于备份中但不应该存在的路由
		// 更准确地说：删除当前路由中不是 Mihomo 路由的路由
		// 这里我们使用另一种策略：删除当前路由中不在备份中且是 Mihomo 路由的路由

		// 创建备份路由的索引
		backupIndex := make(map[string]RouteEntry)
		for _, route := range backup.Routes {
			key := routeKey(route)
			backupIndex[key] = route
		}

		for _, currentRoute := range currentRoutes {
			key := routeKey(currentRoute)
			if _, exists := backupIndex[key]; !exists {
				// 这个路由不在备份中，检查是否是 Mihomo 路由
				if isMihomoRoute(currentRoute) {
					if err := rm.DeleteRoute(currentRoute); err != nil {
						result.Errors = append(result.Errors, err)
						result.FailedRoutes = append(result.FailedRoutes, currentRoute)
					} else {
						result.DeletedRoutes = append(result.DeletedRoutes, currentRoute)
					}
				}
			}
		}

	case "restore":
		// 恢复模式：恢复备份中的路由（仅恢复 Mihomo 路由）
		// 注意：这不会恢复系统路由，只会恢复 Mihomo 添加的路由
		for _, backupRoute := range backup.Routes {
			if isMihomoRoute(backupRoute) {
				// 检查当前是否存在
				key := routeKey(backupRoute)
				if _, exists := currentIndex[key]; !exists {
					// 不存在，添加
					if err := rm.AddRoute(backupRoute); err != nil {
						result.Errors = append(result.Errors, err)
						result.FailedRoutes = append(result.FailedRoutes, backupRoute)
					} else {
						result.AddedRoutes = append(result.AddedRoutes, backupRoute)
					}
				}
			}
		}

	default:
		return nil, fmt.Errorf("unsupported restore mode: %s", mode)
	}

	result.Success = len(result.Errors) == 0

	// 记录操作日志
	if rm.operation != nil {
		_ = rm.operation.Record("restore", "route", fmt.Sprintf("mode=%s, added=%d, deleted=%d", mode, len(result.AddedRoutes), len(result.DeletedRoutes)), "success", nil)
	}

	return result, nil
}

// DeleteBackup 删除路由表备份
func (rm *RouteManager) DeleteBackup(id string) error {
	dataDir, err := GetDataDir()
	if err != nil {
		return fmt.Errorf("failed to get data directory: %w", err)
	}

	backupDir := filepath.Join(dataDir, "route-backups")
	filename := fmt.Sprintf("route-backup-%s.json", id)
	backupPath := filepath.Join(backupDir, filename)

	if err := os.Remove(backupPath); err != nil {
		return fmt.Errorf("failed to delete backup: %w", err)
	}

	// 记录操作日志
	if rm.operation != nil {
		_ = rm.operation.Record("delete", "route", id, "success", nil)
	}

	return nil
}

// PruneBackups 清理旧备份
func (rm *RouteManager) PruneBackups(keep int, olderThan time.Duration) ([]string, error) {
	backups, err := rm.ListBackups()
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
			if err := rm.DeleteBackup(backup.ID); err != nil {
				return deleted, err
			}
			deleted = append(deleted, backup.ID)
		}
	}

	return deleted, nil
}

// isRouteBackupFile 检查是否是路由备份文件
func isRouteBackupFile(filename string) bool {
	return len(filename) > len("route-backup-") &&
		filename[:len("route-backup-")] == "route-backup-" &&
		len(filename) > 5 &&
		filename[len(filename)-5:] == ".json"
}

// extractBackupID 从文件名提取备份 ID
func extractBackupID(filename string) string {
	prefix := "route-backup-"
	suffix := ".json"

	if !strings.HasPrefix(filename, prefix) {
		return ""
	}

	if !strings.HasSuffix(filename, suffix) {
		return ""
	}

	return filename[len(prefix) : len(filename)-len(suffix)]
}
