package mihomo

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/kkkqkx123/mihomo-cli/internal/api"
	"github.com/kkkqkx123/mihomo-cli/internal/output"
	pkgerrors "github.com/kkkqkx123/mihomo-cli/pkg/errors"
)

// PIDFileManager PID 文件管理器（跨平台通用）
type PIDFileManager struct {
	pidFile string
}

// NewPIDFileManager 创建 PID 文件管理器
func NewPIDFileManager(pidFile string) *PIDFileManager {
	return &PIDFileManager{pidFile: pidFile}
}

// Save 保存 PID 到文件
func (p *PIDFileManager) Save(pid int) error {
	if p.pidFile == "" {
		return nil
	}

	// 确保目录存在
	pidDir := filepath.Dir(p.pidFile)
	if err := os.MkdirAll(pidDir, 0755); err != nil {
		return pkgerrors.ErrConfig("failed to create PID directory", err)
	}

	data := []byte(strconv.Itoa(pid))
	if err := os.WriteFile(p.pidFile, data, 0644); err != nil {
		return pkgerrors.ErrConfig("failed to write PID file", err)
	}

	return nil
}

// Read 从文件读取 PID
func (p *PIDFileManager) Read() (int, error) {
	if p.pidFile == "" {
		return 0, pkgerrors.ErrConfig("PID file not configured", nil)
	}

	data, err := os.ReadFile(p.pidFile)
	if err != nil {
		return 0, pkgerrors.ErrConfig("failed to read PID file", err)
	}

	pid, err := strconv.Atoi(string(data))
	if err != nil {
		return 0, pkgerrors.ErrConfig("invalid PID format", err)
	}

	return pid, nil
}

// Cleanup 清理 PID 文件
func (p *PIDFileManager) Cleanup() {
	if p.pidFile != "" {
		os.Remove(p.pidFile)
	}
}

// Exists 检查 PID 文件是否存在
func (p *PIDFileManager) Exists() bool {
	_, err := os.Stat(p.pidFile)
	return err == nil
}

// ForceKill 强制终止进程（跨平台通用）
func ForceKill(pid int) error {
	proc, err := os.FindProcess(pid)
	if err != nil {
		return pkgerrors.ErrService("failed to find process", err)
	}

	if err := proc.Kill(); err != nil {
		return pkgerrors.ErrService("failed to kill process", err)
	}

	// 等待进程退出
	state, err := proc.Wait()
	if err != nil {
		return pkgerrors.ErrService("failed to wait for process exit", err)
	}

	if !state.Exited() {
		return pkgerrors.ErrService("process did not exit as expected", nil)
	}

	return nil
}

// DaemonManagerCommon 守护进程管理器通用功能
type DaemonManagerCommon struct {
	base *DaemonManagerBase
	pid  *PIDFileManager
}

// NewDaemonManagerCommon 创建通用守护进程管理器
func NewDaemonManagerCommon(base *DaemonManagerBase) *DaemonManagerCommon {
	return &DaemonManagerCommon{
		base: base,
		pid:  NewPIDFileManager(base.pidFile),
	}
}

// SavePID 保存 PID
func (d *DaemonManagerCommon) SavePID(pid int) error {
	return d.pid.Save(pid)
}

// ReadPID 读取 PID
func (d *DaemonManagerCommon) ReadPID() (int, error) {
	return d.pid.Read()
}

// CleanupPID 清理 PID
func (d *DaemonManagerCommon) CleanupPID() {
	d.pid.Cleanup()
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

// PIDManager 获取 PID 管理器
func (d *DaemonManagerCommon) PIDManager() *PIDFileManager {
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
