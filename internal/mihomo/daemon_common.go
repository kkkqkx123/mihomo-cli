package mihomo

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/kkkqkx123/mihomo-cli/internal/api"
	"github.com/kkkqkx123/mihomo-cli/internal/output"
	pkgerrors "github.com/kkkqkx123/mihomo-cli/pkg/errors"
)

// ForceKill 强制终止进程（跨平台通用）
func ForceKill(pid int) error {
	return ForceKillWithTimeout(pid, 5*time.Second)
}

// ForceKillWithTimeout 带超时机制的强制终止进程
func ForceKillWithTimeout(pid int, timeout time.Duration) error {
	// 使用平台专用的实现
	return forceKillPlatform(pid, timeout)
}

// isAccessDeniedError 检查是否为权限不足错误（主要用于 Windows）
func isAccessDeniedError(err error) bool {
	if err == nil {
		return false
	}
	errStr := err.Error()
	// Windows 权限错误（使用包含匹配，因为可能有前缀如 "TerminateProcess: "）
	if strings.Contains(errStr, "Access is denied") {
		return true
	}
	// Unix 权限错误
	if strings.Contains(errStr, "operation not permitted") {
		return true
	}
	return false
}

// DaemonManagerCommon 守护进程管理器通用功能
type DaemonManagerCommon struct {
	base *DaemonManagerBase
	pid  *InstanceRegistry
}

// NewDaemonManagerCommon 创建通用守护进程管理器
func NewDaemonManagerCommon(base *DaemonManagerBase) *DaemonManagerCommon {
	return &DaemonManagerCommon{
		base: base,
		pid:  NewInstanceRegistry(base.pidFile),
	}
}

// SavePID 保存 PID 元数据（v1 JSON，含配置文件/可执行文件/API 地址/启动时间）。
// 三平台守护进程管理器（windows/linux/darwin）均通过该方法写入完整元数据，
// 使 ps/status 等进程发现可直接读取实例信息，无需文件名反推。
func (d *DaemonManagerCommon) SavePID(pid int) error {
	meta := InstanceMeta{
		PID:        pid,
		ConfigFile: d.base.GetConfigFile(),
		ExecPath:   d.base.GetExecutablePath(),
		APIAddr:    d.base.GetAPIAddress(),
		StartedAt:  time.Now().Format(time.RFC3339),
	}
	return d.pid.Save(meta)
}

// ReadPID 读取 PID
func (d *DaemonManagerCommon) ReadPID() (int, error) {
	meta, err := d.pid.Load()
	if err != nil {
		return 0, err
	}
	return meta.PID, nil
}

// CleanupPID 清理 PID 文件
func (d *DaemonManagerCommon) CleanupPID() {
	_ = d.pid.Remove()
}

// IsDaemonRunning 检查守护进程是否运行
func (d *DaemonManagerCommon) IsDaemonRunning(pid int) bool {
	if pid == 0 {
		var err error
		pid, err = d.ReadPID()
		if err != nil {
			return false
		}
	}
	return IsProcessRunning(pid)
}

// GetDaemonPID 获取守护进程 PID
func (d *DaemonManagerCommon) GetDaemonPID() (int, error) {
	return d.ReadPID()
}

// ForceKillDaemon 强制终止守护进程
func (d *DaemonManagerCommon) ForceKillDaemon(pid int) error {
	output.Printf("Force killing daemon process %d...\n", pid)

	if err := ForceKill(pid); err != nil {
		return err
	}

	output.Success("Daemon process %d has been killed", pid)
	d.CleanupPID()

	return nil
}

// Base 获取基础配置
func (d *DaemonManagerCommon) Base() *DaemonManagerBase {
	return d.base
}

// PIDManager 获取 PID 元数据注册器
func (d *DaemonManagerCommon) PIDManager() *InstanceRegistry {
	return d.pid
}

// StopProcessByPID 通过 API 停止指定 PID 的进程
func StopProcessByPID(pid int, apiAddr, secret string) error {
	// 检查进程是否还在运行
	if !IsProcessRunning(pid) {
		return pkgerrors.ErrService(fmt.Sprintf("process %d is not running", pid), nil)
	}

	// 创建 API 客户端
	client := api.NewClient(
		"http://"+apiAddr,
		secret,
		api.WithTimeout(10*time.Second),
	)

	// 使用 API 关闭进程
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := client.Shutdown(ctx); err != nil {
		return pkgerrors.ErrService("API shutdown failed", err)
	}

	// 等待进程退出
	timeout := 10 * time.Second
	deadline := time.Now().Add(timeout)

	for time.Now().Before(deadline) {
		if !IsProcessRunning(pid) {
			return nil
		}
		time.Sleep(100 * time.Millisecond)
	}

	return pkgerrors.ErrService("process did not exit within timeout", nil)
}
