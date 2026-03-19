package system

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/kkkqkx123/mihomo-cli/internal/config"
)

// SystemConfigManager 系统配置管理器
type SystemConfigManager struct {
	sysproxy       *SysProxyManager
	tun            *TUNManager
	route          *RouteManager
	snapshot       *SnapshotManager
	audit          *AuditLogger
	mu             sync.RWMutex
}

// NewSystemConfigManager 创建系统配置管理器
func NewSystemConfigManager() (*SystemConfigManager, error) {
	// 获取数据目录
	dataDir, err := config.GetDataDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get data directory: %w", err)
	}

	// 创建审计日志记录器
	auditFile := filepath.Join(dataDir, "audit.log")
	audit, err := NewAuditLogger(auditFile)
	if err != nil {
		return nil, fmt.Errorf("failed to create audit logger: %w", err)
	}

	// 创建快照管理器
	snapshotDir := filepath.Join(dataDir, "snapshots")
	snapshot, err := NewSnapshotManager(snapshotDir, audit)
	if err != nil {
		return nil, fmt.Errorf("failed to create snapshot manager: %w", err)
	}

	return &SystemConfigManager{
		sysproxy: NewSysProxyManager(audit),
		tun:      NewTUNManager(audit),
		route:    NewRouteManager(audit),
		snapshot: snapshot,
		audit:    audit,
	}, nil
}

// GetConfigState 获取当前系统配置状态
func (scm *SystemConfigManager) GetConfigState() (*ConfigState, error) {
	scm.mu.RLock()
	defer scm.mu.RUnlock()

	state := &ConfigState{
		Timestamp: time.Now(),
	}

	// 获取系统代理状态
	sysproxyStatus, err := scm.sysproxy.GetStatus()
	if err == nil {
		state.SysProxy = sysproxyStatus
	}

	// 获取 TUN 状态
	tunState, err := scm.tun.GetState()
	if err == nil {
		state.TUN = tunState
	}

	// 获取路由表
	routes, err := scm.route.ListRoutes()
	if err == nil {
		state.Routes = routes
	}

	return state, nil
}

// CleanupAll 清理所有系统配置
func (scm *SystemConfigManager) CleanupAll() error {
	scm.mu.Lock()
	defer scm.mu.Unlock()

	var errors []error

	// 清理系统代理
	if err := scm.sysproxy.Cleanup(); err != nil {
		errors = append(errors, fmt.Errorf("sysproxy cleanup failed: %w", err))
	}

	// 清理 TUN 设备
	if err := scm.tun.Cleanup(); err != nil {
		errors = append(errors, fmt.Errorf("tun cleanup failed: %w", err))
	}

	// 清理路由表
	if err := scm.route.Cleanup(); err != nil {
		errors = append(errors, fmt.Errorf("route cleanup failed: %w", err))
	}

	if len(errors) > 0 {
		return fmt.Errorf("cleanup completed with %d errors: %v", len(errors), errors)
	}

	return nil
}

// CreateSnapshot 创建配置快照
func (scm *SystemConfigManager) CreateSnapshot(note string) (*ConfigSnapshot, error) {
	state, err := scm.GetConfigState()
	if err != nil {
		return nil, fmt.Errorf("failed to get config state: %w", err)
	}

	return scm.snapshot.CreateSnapshot(*state, note)
}

// RestoreSnapshot 恢复配置快照
func (scm *SystemConfigManager) RestoreSnapshot(id string) error {
	snapshot, err := scm.snapshot.RestoreSnapshot(id)
	if err != nil {
		return fmt.Errorf("failed to restore snapshot: %w", err)
	}

	// 恢复系统代理设置
	if snapshot.State.SysProxy != nil {
		if snapshot.State.SysProxy.Enabled {
			if err := scm.sysproxy.Enable(snapshot.State.SysProxy.Server, snapshot.State.SysProxy.BypassList); err != nil {
				return fmt.Errorf("failed to restore sysproxy: %w", err)
			}
		} else {
			if err := scm.sysproxy.Disable(); err != nil {
				return fmt.Errorf("failed to restore sysproxy: %w", err)
			}
		}
	}

	// 注意：TUN 和路由表的恢复比较复杂，可能需要重启 Mihomo
	// 这里只记录日志，不执行实际恢复

	return nil
}

// ValidateState 验证配置状态是否正常
func (scm *SystemConfigManager) ValidateState() ([]Problem, error) {
	var problems []Problem

	// 检查系统代理残留
	if problem, err := scm.sysproxy.CheckResidual(); err == nil && problem != nil {
		problems = append(problems, *problem)
	}

	// 检查 TUN 设备残留
	if problem, err := scm.tun.CheckResidual(); err == nil && problem != nil {
		problems = append(problems, *problem)
	}

	// 检查路由表残留
	if problem, err := scm.route.CheckResidual(); err == nil && problem != nil {
		problems = append(problems, *problem)
	}

	return problems, nil
}

// GetSysProxyManager 获取系统代理管理器
func (scm *SystemConfigManager) GetSysProxyManager() *SysProxyManager {
	return scm.sysproxy
}

// GetTUNManager 获取 TUN 管理器
func (scm *SystemConfigManager) GetTUNManager() *TUNManager {
	return scm.tun
}

// GetRouteManager 获取路由表管理器
func (scm *SystemConfigManager) GetRouteManager() *RouteManager {
	return scm.route
}

// GetSnapshotManager 获取快照管理器
func (scm *SystemConfigManager) GetSnapshotManager() *SnapshotManager {
	return scm.snapshot
}

// GetAuditLogger 获取审计日志记录器
func (scm *SystemConfigManager) GetAuditLogger() *AuditLogger {
	return scm.audit
}

// ListSnapshots 列出所有快照
func (scm *SystemConfigManager) ListSnapshots() ([]ConfigSnapshot, error) {
	return scm.snapshot.ListSnapshots()
}

// DeleteSnapshot 删除快照
func (scm *SystemConfigManager) DeleteSnapshot(id string) error {
	return scm.snapshot.DeleteSnapshot(id)
}

// QueryAuditLog 查询审计日志
func (scm *SystemConfigManager) QueryAuditLog(component string, since time.Time, limit int) ([]AuditRecord, error) {
	return scm.audit.Query(component, since, limit)
}

// ClearAuditLog 清空审计日志
func (scm *SystemConfigManager) ClearAuditLog() error {
	return scm.audit.Clear()
}

// PruneAuditLog 清理指定时间之前的审计日志
func (scm *SystemConfigManager) PruneAuditLog(before time.Time) (int, error) {
	return scm.audit.Prune(before)
}

// GetDataDir 获取数据目录
func GetDataDir() (string, error) {
	return config.GetDataDir()
}

// EnsureDataDir 确保数据目录存在
func EnsureDataDir() error {
	dataDir, err := config.GetDataDir()
	if err != nil {
		return err
	}

	return os.MkdirAll(dataDir, 0755)
}
