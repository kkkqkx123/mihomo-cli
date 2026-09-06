package mihomo

import (
	"fmt"

	"github.com/kkkqkx123/mihomo-cli/internal/config"
	pkgerrors "github.com/kkkqkx123/mihomo-cli/pkg/errors"
)

// ProcessManager Mihomo 进程管理器（DaemonLauncher 的薄包装，供 LifecycleManager 使用）。
// configPath 为 config.toml 路径，用于解析内核相对路径的基准目录。
type ProcessManager struct {
	config     *config.TomlConfig
	configPath string          // config.toml 路径
	pidFile    string          // PID 文件路径
	launcher   *DaemonLauncher // 缓存的启动器实例（Start 后赋值，Stop 复用）
}

// NewProcessManager 创建进程管理器
func NewProcessManager(cfg *config.TomlConfig, configPath string) *ProcessManager {
	pidFile, _ := getPIDFilePath(cfg.Mihomo.ConfigFile)
	return &ProcessManager{
		config:     cfg,
		configPath: configPath,
		pidFile:    pidFile,
	}
}

// getPIDFilePath 获取 PID 文件路径（基于配置文件路径）
func getPIDFilePath(configFile string) (string, error) {
	return config.GetPIDFilePath(configFile)
}

// Start 启动 Mihomo 内核（委托 DaemonLauncher，消除重复实现）
func (pm *ProcessManager) Start() error {
	launcher, err := NewDaemonLauncher(pm.config, "")
	if err != nil {
		return err
	}
	if err := launcher.Start(); err != nil {
		return err
	}
	pm.launcher = launcher
	return nil
}

// SetLauncher 设置缓存的启动器实例（由外部注入，Start/Stop 共享同一实例）
func (pm *ProcessManager) SetLauncher(launcher *DaemonLauncher) {
	pm.launcher = launcher
}

// GetLauncher 获取缓存的启动器实例
func (pm *ProcessManager) GetLauncher() *DaemonLauncher {
	return pm.launcher
}

// StopDaemon 停止守护进程（转发给 DaemonLauncher）
func (pm *ProcessManager) StopDaemon(force bool) error {
	if pm.launcher == nil {
		return pkgerrors.ErrService("daemon launcher not initialized", nil)
	}
	return pm.launcher.Stop(force)
}

// GetSecret 获取当前密钥
func (pm *ProcessManager) GetSecret() string {
	if pm.launcher != nil {
		return pm.launcher.GetSecret()
	}
	return ""
}

// GetAPIAddress 获取 API 地址
func (pm *ProcessManager) GetAPIAddress() string {
	return pm.config.Mihomo.API.ExternalController
}

// GetPIDFromPIDFile 从 PID 文件读取并检查进程是否运行
func (pm *ProcessManager) GetPIDFromPIDFile() (int, error) {
	if pm.pidFile == "" {
		return 0, pkgerrors.ErrService("PID file not configured", nil)
	}

	meta, err := NewInstanceRegistry(pm.pidFile).Load()
	if err != nil {
		return 0, err
	}
	pid := meta.PID

	// 检查进程是否真的在运行
	if !IsProcessRunning(pid) {
		return 0, pkgerrors.ErrService("process "+fmt.Sprintf("%d", pid)+" is not running", nil)
	}

	return pid, nil
}
