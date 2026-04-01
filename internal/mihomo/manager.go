package mihomo

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strconv"

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
	return &ProcessManager{
		config:  cfg,
		pidFile: pidFile,
	}
}

// getPIDFilePath 获取 PID 文件路径（基于配置文件路径）
func getPIDFilePath(configFile string) (string, error) {
	return config.GetPIDFilePath(configFile)
}

// Start 启动 Mihomo 内核
func (pm *ProcessManager) Start() error {
	// 生成随机密钥
	secret, err := pm.prepareSecret()
	if err != nil {
		return err
	}
	pm.secret = secret

	// 准备配置文件
	configFile, err := pm.prepareConfigFile(secret)
	if err != nil {
		return pkgerrors.ErrService("failed to prepare config file", err)
	}

	// 获取可执行文件的绝对路径
	execPath := pm.config.Mihomo.Executable
	if absExec, err := filepath.Abs(execPath); err == nil {
		execPath = absExec
	}

	// 创建守护进程管理器（只创建一次）
	daemonConfig := pm.buildDaemonConfig()
	pm.daemonManager = GetDaemonManager(
		daemonConfig,
		pm.pidFile,
		secret,
		pm.config.Mihomo.API.ExternalController,
		execPath,
		configFile,
	)

	// 使用守护进程管理器启动
	ctx := context.Background()
	if err := pm.daemonManager.StartAsDaemon(ctx, nil); err != nil {
		return err
	}

	return nil
}

// prepareSecret 准备 API 密钥
func (pm *ProcessManager) prepareSecret() (string, error) {
	if pm.config.Mihomo.AutoGenerateSecret {
		return config.GenerateRandomSecret()
	}
	return pm.config.API.Secret, nil
}

// buildDaemonConfig 构建守护进程配置
func (pm *ProcessManager) buildDaemonConfig() *DaemonConfig {
	// 默认工作目录为可执行文件所在目录（转换为绝对路径）
	defaultWorkDir := filepath.Dir(pm.config.Mihomo.Executable)
	if absDir, err := filepath.Abs(defaultWorkDir); err == nil {
		defaultWorkDir = absDir
	}

	daemonConfig := &DaemonConfig{
		Enabled:       true,
		WorkDir:       defaultWorkDir,
		LogFile:       "",
		LogLevel:      "info",
		LogMaxSize:    "100M",
		LogMaxBackups: 10,
		LogMaxAge:     30,
	}

	// 如果配置文件中有守护进程配置，使用配置文件的值
	if pm.config.Daemon != nil {
		// 只有当配置文件明确指定了 WorkDir 时才覆盖
		if pm.config.Daemon.WorkDir != "" {
			// 同样转换为绝对路径
			if absDir, err := filepath.Abs(pm.config.Daemon.WorkDir); err == nil {
				daemonConfig.WorkDir = absDir
			} else {
				daemonConfig.WorkDir = pm.config.Daemon.WorkDir
			}
		}
		daemonConfig.LogFile = pm.config.Daemon.LogFile
		daemonConfig.LogLevel = pm.config.Daemon.LogLevel
		daemonConfig.LogMaxSize = pm.config.Daemon.LogMaxSize
		daemonConfig.LogMaxBackups = pm.config.Daemon.LogMaxBackups
		daemonConfig.LogMaxAge = pm.config.Daemon.LogMaxAge
	}

	return daemonConfig
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
	// 如果有 daemonManager，优先使用它
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

	// 否则直接从 PID 文件读取（用于 status 命令等场景）
	if pm.pidFile == "" {
		return 0, pkgerrors.ErrService("PID file not configured", nil)
	}

	data, err := os.ReadFile(pm.pidFile)
	if err != nil {
		return 0, pkgerrors.ErrConfig("failed to read PID file", err)
	}

	pid, err := strconv.Atoi(string(data))
	if err != nil {
		return 0, pkgerrors.ErrConfig("invalid PID format", err)
	}

	// 检查进程是否真的在运行
	if !IsProcessRunning(pid) {
		return 0, pkgerrors.ErrService("process "+fmt.Sprintf("%d", pid)+" is not running", nil)
	}

	return pid, nil
}

// prepareConfigFile 准备配置文件
func (pm *ProcessManager) prepareConfigFile(secret string) (string, error) {
	// 如果指定了配置文件，转换为绝对路径后使用
	if pm.config.Mihomo.ConfigFile != "" {
		absPath, err := filepath.Abs(pm.config.Mihomo.ConfigFile)
		if err != nil {
			return "", pkgerrors.ErrConfig("failed to get absolute path of config file", err)
		}
		return absPath, nil
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