package system

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/kkkqkx123/mihomo-cli/internal/operation"
)

// SnapshotManager 快照管理器
type SnapshotManager struct {
	snapshotDir string
	operation   *operation.Manager
}

// NewSnapshotManager 创建快照管理器
func NewSnapshotManager(snapshotDir string, op *operation.Manager) (*SnapshotManager, error) {
	// 确保目录存在
	if err := os.MkdirAll(snapshotDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create snapshot directory: %w", err)
	}

	return &SnapshotManager{
		snapshotDir: snapshotDir,
		operation:   op,
	}, nil
}

// CreateSnapshot 创建配置快照
func (sm *SnapshotManager) CreateSnapshot(state ConfigState, note string) (*ConfigSnapshot, error) {
	// 生成快照 ID
	id := time.Now().Format("20060102-150405")

	snapshot := ConfigSnapshot{
		ID:        id,
		State:     state,
		CreatedAt: time.Now(),
		Note:      note,
	}

	// 序列化快照
	data, err := json.MarshalIndent(snapshot, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal snapshot: %w", err)
	}

	// 保存快照文件
	filename := fmt.Sprintf("snapshot-%s.json", id)
	snapshotPath := filepath.Join(sm.snapshotDir, filename)
	if err := os.WriteFile(snapshotPath, data, 0644); err != nil {
		return nil, fmt.Errorf("failed to save snapshot: %w", err)
	}

	if sm.operation != nil {
		_ = sm.operation.Record("create", "snapshot", note, "success", nil)
	}

	return &snapshot, nil
}

// RestoreSnapshot 恢复配置快照
func (sm *SnapshotManager) RestoreSnapshot(id string) (*ConfigSnapshot, error) {
	// 读取快照文件
	filename := fmt.Sprintf("snapshot-%s.json", id)
	snapshotPath := filepath.Join(sm.snapshotDir, filename)

	data, err := os.ReadFile(snapshotPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read snapshot: %w", err)
	}

	var snapshot ConfigSnapshot
	if err := json.Unmarshal(data, &snapshot); err != nil {
		return nil, fmt.Errorf("failed to unmarshal snapshot: %w", err)
	}

	if sm.operation != nil {
		_ = sm.operation.Record("restore", "snapshot", id, "success", nil)
	}

	return &snapshot, nil
}

// ListSnapshots 列出所有快照
func (sm *SnapshotManager) ListSnapshots() ([]ConfigSnapshot, error) {
	// 读取快照目录
	entries, err := os.ReadDir(sm.snapshotDir)
	if err != nil {
		if os.IsNotExist(err) {
			return []ConfigSnapshot{}, nil
		}
		return nil, fmt.Errorf("failed to read snapshot directory: %w", err)
	}

	var snapshots []ConfigSnapshot
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		// 只处理快照文件
		if !isSnapshotFile(entry.Name()) {
			continue
		}

		// 读取快照
		snapshotPath := filepath.Join(sm.snapshotDir, entry.Name())
		data, err := os.ReadFile(snapshotPath)
		if err != nil {
			continue
		}

		var snapshot ConfigSnapshot
		if err := json.Unmarshal(data, &snapshot); err != nil {
			continue
		}

		snapshots = append(snapshots, snapshot)
	}

	// 按时间倒序排列（最新的在前）
	sort.Slice(snapshots, func(i, j int) bool {
		return snapshots[i].CreatedAt.After(snapshots[j].CreatedAt)
	})

	return snapshots, nil
}

// DeleteSnapshot 删除快照
func (sm *SnapshotManager) DeleteSnapshot(id string) error {
	filename := fmt.Sprintf("snapshot-%s.json", id)
	snapshotPath := filepath.Join(sm.snapshotDir, filename)

	if err := os.Remove(snapshotPath); err != nil {
		return fmt.Errorf("failed to delete snapshot: %w", err)
	}

	if sm.operation != nil {
		_ = sm.operation.Record("delete", "snapshot", id, "success", nil)
	}

	return nil
}

// GetLatestSnapshot 获取最新的快照
func (sm *SnapshotManager) GetLatestSnapshot() (*ConfigSnapshot, error) {
	snapshots, err := sm.ListSnapshots()
	if err != nil {
		return nil, err
	}

	if len(snapshots) == 0 {
		return nil, fmt.Errorf("no snapshots found")
	}

	return &snapshots[0], nil
}

// PruneSnapshots 清理旧快照
func (sm *SnapshotManager) PruneSnapshots(keep int, olderThan time.Duration) ([]string, error) {
	snapshots, err := sm.ListSnapshots()
	if err != nil {
		return nil, err
	}

	var deleted []string
	now := time.Now()

	for i, snapshot := range snapshots {
		shouldDelete := false

		// 检查是否超过保留数量
		if keep > 0 && i >= keep {
			shouldDelete = true
		}

		// 检查是否超过保留时间
		if olderThan > 0 {
			age := now.Sub(snapshot.CreatedAt)
			if age > olderThan {
				shouldDelete = true
			}
		}

		if shouldDelete {
			if err := sm.DeleteSnapshot(snapshot.ID); err != nil {
				return deleted, err
			}
			deleted = append(deleted, snapshot.ID)
		}
	}

	return deleted, nil
}

// isSnapshotFile 检查是否是快照文件
func isSnapshotFile(filename string) bool {
	return len(filename) > len("snapshot-") && filename[:len("snapshot-")] == "snapshot-" &&
		len(filename) > 5 && filename[len(filename)-5:] == ".json"
}
