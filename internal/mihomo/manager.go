package mihomo

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/kkkqkx123/mihomo-cli/internal/config"
	pkgerrors "github.com/kkkqkx123/mihomo-cli/pkg/errors"
)

// ProcessManager Mihomo 进程管理器（使用守护进程模式）
type ProcessManager struct {
	config        *config.TomlConfig
	secret        string
	pidFile       string       // PID 文件路径
	daemonManager DaemonManager // 守护进程管理器
}

// NewProcessManager 创建进程管理器
func NewProcessManager(cfg *config.TomlConfig) *ProcessManager {
	pidFile, _ := getPIDFilePath(cfg.Mihomo.ConfigFile)
	pm := &ProcessManager{
		config:  cfg,
		pidFile: pidFile,
	}

	// 创建守护进程配置
	daemonConfig := &DaemonConfig{
		Enabled:       true, // 默认启用守护进程模式
		WorkDir:       "",
		LogFile:       "",
		LogLevel:      "info",
		LogMaxSize:    "100M",
		LogMaxBackups: 10,
		LogMaxAge:     30,
	}

	// 如果配置文件中有守护进程配置，使用配置文件的值
	if cfg.Daemon != nil {
		daemonConfig.Enabled = true
		daemonConfig.WorkDir = cfg.Daemon.WorkDir
		daemonConfig.LogFile = cfg.Daemon.LogFile
		daemonConfig.LogLevel = cfg.Daemon.LogLevel
		daemonConfig.LogMaxSize = cfg.Daemon.LogMaxSize
		daemonConfig.LogMaxBackups = cfg.Daemon.LogMaxBackups
		daemonConfig.LogMaxAge = cfg.Daemon.LogMaxAge
	}

	// 创建守护进程管理器
	pm.daemonManager = GetDaemonManager(
		daemonConfig,
		pidFile,
		"", // secret 将在启动时设置
		cfg.Mihomo.API.ExternalController,
		cfg.Mihomo.Executable,
		cfg.Mihomo.ConfigFile,
	)

	return pm
}

// getPIDFilePath 获取 PID 文件路径（基于配置文件路径）
func getPIDFilePath(configFile string) (string, error) {
	return config.GetPIDFilePath(configFile)
}

// Start 启动 Mihomo 内核
func (pm *ProcessManager) Start() error {
	// 生成随机密钥
	var secret string
	var err error

	if pm.config.Mihomo.AutoGenerateSecret {
		secret, err = config.GenerateRandomSecret()
		if err != nil {
			return pkgerrors.ErrService("failed to generate secret", err)
		}
	} else {
		secret = pm.config.API.Secret
	}

	pm.secret = secret

	// 准备配置文件
	configFile, err := pm.prepareConfigFile(secret)
	if err != nil {
		return pkgerrors.ErrService("failed to prepare config file", err)
	}

	// 使用守护进程管理器启动
	return pm.startAsDaemon(configFile, secret)
}

// startAsDaemon 以守护进程方式启动
func (pm *ProcessManager) startAsDaemon(configFile, secret string) error {
	// 更新配置文件路径
	pm.config.Mihomo.ConfigFile = configFile

	// 创建新的守护进程管理器（包含 secret）
	daemonConfig := &DaemonConfig{
		Enabled:       true,
		WorkDir:       "",
		LogFile:       "",
		LogLevel:      "info",
		LogMaxSize:    "100M",
		LogMaxBackups: 10,
		LogMaxAge:     30,
	}

	// 如果配置文件中有守护进程配置，使用配置文件的值
	if pm.config.Daemon != nil {
		daemonConfig.WorkDir = pm.config.Daemon.WorkDir
		daemonConfig.LogFile = pm.config.Daemon.LogFile
		daemonConfig.LogLevel = pm.config.Daemon.LogLevel
		daemonConfig.LogMaxSize = pm.config.Daemon.LogMaxSize
		daemonConfig.LogMaxBackups = pm.config.Daemon.LogMaxBackups
		daemonConfig.LogMaxAge = pm.config.Daemon.LogMaxAge
	}

	pm.daemonManager = GetDaemonManager(
		daemonConfig,
		pm.pidFile,
		secret,
		pm.config.Mihomo.API.ExternalController,
		pm.config.Mihomo.Executable,
		configFile,
	)

	// 使用守护进程管理器启动
	ctx := context.Background()
	if err := pm.daemonManager.StartAsDaemon(ctx, nil); err != nil {
		return err
	}

	return nil
}

// GetSecret 获取当前密钥
func (pm *ProcessManager) GetSecret() string {
	return pm.secret
}

// GetAPIAddress 获取 API 地址
func (pm *ProcessManager) GetAPIAddress() string {
	return pm.config.Mihomo.API.ExternalController
}

// GetPIDFromPIDFile 从 PID 文件读取并检查进程是否运行
func (pm *ProcessManager) GetPIDFromPIDFile() (int, error) {
	// 从守护进程管理器获取 PID
	if pm.daemonManager != nil {
		pid, err := pm.daemonManager.GetDaemonPID()
		if err != nil {
			return 0, err
		}

		// 检查进程是否真的在运行
		if !IsProcessRunning(pid) {
			return 0, pkgerrors.ErrService("process "+fmt.Sprintf("%d", pid)+" is not running", nil)
		}

		return pid, nil
	}

	return 0, pkgerrors.ErrService("daemon manager not initialized", nil)
}

// prepareConfigFile 准备配置文件
func (pm *ProcessManager) prepareConfigFile(secret string) (string, error) {
	// 如果指定了配置文件，直接使用
	if pm.config.Mihomo.ConfigFile != "" {
		return pm.config.Mihomo.ConfigFile, nil
	}

	// 否则生成临时配置文件
	tempDir := os.TempDir()
	configFile := filepath.Join(tempDir, "mihomo-config.yaml")

	// 生成配置内容
	configContent := pm.generateConfigContent(secret)

	// 写入文件
	if err := os.WriteFile(configFile, []byte(configContent), 0644); err != nil {
		return "", pkgerrors.ErrConfig("failed to write config file", err)
	}

	return configFile, nil
}

// generateConfigContent 生成配置内容
func (pm *ProcessManager) generateConfigContent(secret string) string {
	return fmt.Sprintf(`# Auto-generated config by mihomo-go
mixed-port: 7890
mode: rule
log-level: %s

# API 控制器
external-controller: %s
secret: "%s"

# DNS
dns:
  enable: true
  enhanced-mode: fake-ip
  nameserver:
    - 8.8.8.8

# 代理组配置
proxy-groups:
  - name: "Proxy"
    type: select
    proxies:
      - DIRECT

rules:
  - MATCH,Proxy
`, pm.config.Mihomo.Log.Level, pm.config.Mihomo.API.ExternalController, secret)
}


