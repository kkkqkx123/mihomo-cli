package mihomo

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/kkkqkx123/mihomo-cli/internal/api"
	"github.com/kkkqkx123/mihomo-cli/internal/config"
	"github.com/kkkqkx123/mihomo-cli/internal/output"
	"github.com/kkkqkx123/mihomo-cli/internal/system"
	pkgerrors "github.com/kkkqkx123/mihomo-cli/pkg/errors"
)

// ProcessHandler 进程管理处理器
type ProcessHandler struct {
	configPath string
}

// NewProcessHandler 创建进程处理器
func NewProcessHandler(configPath string) *ProcessHandler {
	return &ProcessHandler{
		configPath: configPath,
	}
}

// StartResult 启动结果
type StartResult struct {
	APIAddress string
	Secret     string
	PID        int
}

// Start 启动 Mihomo 内核
func (ph *ProcessHandler) Start(cfg *config.TomlConfig) (*StartResult, error) {
	// 检查是否启用自动启动
	if !cfg.Mihomo.Enabled {
		return nil, pkgerrors.ErrConfig("mihomo auto-start is disabled in config.toml", nil)
	}

	// 检查可执行文件是否存在
	if _, err := os.Stat(cfg.Mihomo.Executable); os.IsNotExist(err) {
		return nil, pkgerrors.ErrConfig("mihomo executable not found: "+cfg.Mihomo.Executable, nil)
	}

	// 检查是否已经在运行
	pm := NewProcessManager(cfg)
	if pid, err := pm.GetPIDFromPIDFile(); err == nil {
		return nil, pkgerrors.ErrService("mihomo is already running (PID: "+fmt.Sprintf("%d", pid)+"), use 'mihomo-cli stop' to stop it first", nil)
	}

	// 启动前配置检查 - 检查是否启用了高风险配置（TUN/TProxy）
	hasTUN := false
	hasTProxy := false
	if cfg.Mihomo.ConfigFile != "" {
		validator := config.NewConfigValidator(cfg.Mihomo.ConfigFile)
		if err := validator.ValidateAndWarn(); err != nil {
			// 配置检查失败不影响启动，只记录警告
			output.Warning("config validation failed: " + err.Error())
		}

		// 检测是否启用了 TUN 或 TProxy
		content, _ := os.ReadFile(cfg.Mihomo.ConfigFile)
		if content != nil {
			lowerContent := strings.ToLower(string(content))
			if strings.Contains(lowerContent, "tun:") && strings.Contains(lowerContent, "enable: true") {
				hasTUN = true
			}
			if strings.Contains(lowerContent, "tproxy-port:") {
				hasTProxy = true
			}
		}
	}

	// 启动前备份系统配置
	if hasTUN || hasTProxy {
		output.Info("Creating system configuration backup...")
		if err := ph.backupSystemConfig(cfg, hasTUN, hasTProxy); err != nil {
			output.Warning("failed to create backup: " + err.Error())
		} else {
			output.Success("System configuration backup created")
		}
	}

	// 启动内核
	if err := pm.Start(); err != nil {
		return nil, pkgerrors.ErrService("failed to start mihomo", err)
	}

	result := &StartResult{
		APIAddress: pm.GetAPIAddress(),
		Secret:     pm.GetSecret(),
	}

	// 获取健康检查超时时间
	healthCheckTimeout := cfg.Mihomo.HealthCheckTimeout
	if healthCheckTimeout <= 0 {
		healthCheckTimeout = 5 // 默认 5 秒
	}

	// 创建 API 客户端进行健康检查
	apiClient := api.NewClient(
		"http://"+pm.GetAPIAddress(),
		pm.GetSecret(),
		api.WithTimeout(3*time.Second),
	)

	// 等待并检查健康状况
	checkCtx, cancel := context.WithTimeout(context.Background(), time.Duration(healthCheckTimeout)*time.Second)
	defer cancel()

	output.Printf("等待 Mihomo 内核启动（最多 %d 秒）...\n", healthCheckTimeout)

	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-checkCtx.Done():
			// 健康检查超时，通过 PID 停止进程
			if pid, err := pm.GetPIDFromPIDFile(); err == nil {
				_ = StopProcessByPID(pid)
			}
			return nil, pkgerrors.ErrService("mihomo health check timeout: process may have failed to start", nil)

		case <-ticker.C:
			// 检查进程是否还在运行
			if pid, err := pm.GetPIDFromPIDFile(); err == nil {
				if !IsProcessRunning(pid) {
					// 获取错误输出
					stderr := pm.GetErrorOutput()
					stdout := pm.GetStandardOutput()

					errMsg := "mihomo process exited unexpectedly"
					if stderr != "" {
						errMsg += fmt.Sprintf("\n错误输出:\n%s", stderr)
					}
					if stdout != "" {
						errMsg += fmt.Sprintf("\n标准输出:\n%s", stdout)
					}

					return nil, pkgerrors.ErrService(errMsg, nil)
				}
			}

			// 尝试连接 API 进行健康检查
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			_, err := apiClient.GetMode(ctx)
			cancel()

			if err == nil {
				// 基础健康检查成功，进行增强健康检查
				output.PrintEmptyLine()
				output.Info("Performing detailed health check...")

				// 创建新的 context 用于详细健康检查
				healthCtx, healthCancel := context.WithTimeout(context.Background(), 5*time.Second)
				defer healthCancel()

				healthChecker := NewHealthChecker(apiClient, cfg.Mihomo.ConfigFile, 5*time.Second)
				healthStatus, err := healthChecker.CheckHealth(healthCtx)
				if err != nil {
					output.Warning("detailed health check failed: " + err.Error())
					output.Warning("Process started but may have issues")
				} else {
					healthChecker.PrintHealthStatus(healthStatus)

					if !healthChecker.IsHealthy(healthStatus) {
						output.PrintEmptyLine()
						output.Warning("⚠ Mihomo started but some components may not be working properly")
						output.Printf("  Check the warnings above for details\n")
					}
				}

				// 健康检查成功
				output.PrintEmptyLine()
				output.Success("Mihomo 内核启动成功！")
				return result, nil
			}
		}
	}
}

// StopResult 停止结果
type StopResult struct {
	PID int
}

// Stop 停止 Mihomo 内核
func (ph *ProcessHandler) Stop(cfg *config.TomlConfig, stopAll bool, stopConfig string, args []string) (*StopResult, error) {
	// 如果指定了 --all，停止所有进程
	if stopAll {
		return nil, StopAllMihomoProcesses()
	}

	var pid int
	var err error

	// 如果指定了 PID 参数
	if len(args) == 1 {
		_, err := fmt.Sscanf(args[0], "%d", &pid)
		if err != nil {
			return nil, pkgerrors.ErrInvalidArg("invalid PID: "+args[0], nil)
		}

		// 验证进程是否在运行
		if !IsProcessRunning(pid) {
			return nil, pkgerrors.ErrService("process "+fmt.Sprintf("%d", pid)+" is not running", nil)
		}
	} else {
		// 默认：停止当前配置的实例
		pm := NewProcessManager(cfg)

		// 从 PID 文件读取进程 ID
		pid, err = pm.GetPIDFromPIDFile()
		if err != nil {
			return nil, pkgerrors.ErrService("mihomo is not running", err)
		}
	}

	// 停止进程
	if err := StopProcessByPID(pid); err != nil {
		return nil, pkgerrors.ErrService("failed to stop process", err)
	}

	// 删除 PID 文件
	if len(args) == 0 {
		pm := NewProcessManager(cfg)
		os.Remove(pm.pidFile)
	}

	// 检查系统配置状态并尝试清理
	output.Info("Checking system configuration...")
	if err := ph.checkAndCleanupAfterStop(cfg); err != nil {
		output.Warning("failed to check system configuration: " + err.Error())
	}

	return &StopResult{PID: pid}, nil
}

// StatusResult 状态结果
type StatusResult struct {
	IsRunning  bool
	PID        int
	APIAddress string
}

// Status 查询 Mihomo 内核状态
func (ph *ProcessHandler) Status(cfg *config.TomlConfig) (*StatusResult, error) {
	pm := NewProcessManager(cfg)

	// 从 PID 文件读取进程 ID
	pid, err := pm.GetPIDFromPIDFile()
	if err != nil {
		return &StatusResult{
			IsRunning: false,
		}, nil
	}

	return &StatusResult{
		IsRunning:  true,
		PID:        pid,
		APIAddress: pm.GetAPIAddress(),
	}, nil
}

// backupSystemConfig 备份系统配置
func (ph *ProcessHandler) backupSystemConfig(cfg *config.TomlConfig, hasTUN, hasTProxy bool) error {
	dataDir, err := config.GetDataDir()
	if err != nil {
		return err
	}

	// 确保目录存在
	backupDir := filepath.Join(dataDir, "system-backups")
	if err := os.MkdirAll(backupDir, 0755); err != nil {
		return err
	}

	// 创建系统配置管理器
	scm, err := system.NewSystemConfigManager()
	if err != nil {
		return err
	}

	// 备份路由表
	if hasTUN || hasTProxy {
		routeManager := scm.GetRouteManager()
		routeBackup, err := routeManager.BackupRoutes("pre-start backup")
		if err == nil {
			_, err = routeManager.SaveBackup(routeBackup)
			if err != nil {
				output.Warning("failed to save route backup: " + err.Error())
			}
		}
	}

	// 备份 TUN 接口状态
	if hasTUN {
		tunManager := scm.GetTUNManager()
		tunBackup, err := tunManager.BackupTUNState("pre-start backup")
		if err == nil {
			_, err = tunManager.SaveTUNBackup(tunBackup)
			if err != nil {
				output.Warning("failed to save TUN backup: " + err.Error())
			}
		}
	}

	// 备份注册表设置（仅 Windows）
	spm := scm.GetSysProxyManager()
	if spm != nil {
		proxyStatus, err := spm.GetStatus()
		if err == nil {
			// 保存注册表备份
			backupData, err := json.MarshalIndent(proxyStatus, "", "  ")
			if err == nil {
				backupFile := filepath.Join(backupDir, fmt.Sprintf("registry-backup-pre-start.json"))
				_ = os.WriteFile(backupFile, backupData, 0644)
			}
		}
	}

	return nil
}

// checkAndCleanupAfterStop 停止后检查并清理系统配置
func (ph *ProcessHandler) checkAndCleanupAfterStop(cfg *config.TomlConfig) error {
	scm, err := system.NewSystemConfigManager()
	if err != nil {
		return err
	}

	// 检查残留配置
	problems, err := scm.ValidateState()
	if err != nil {
		return err
	}

	if len(problems) > 0 {
		output.Warning("Detected %d residual configuration issues", len(problems))
		for _, problem := range problems {
			output.Printf("  - %s (severity: %s)\n", problem.Description, problem.Severity)
		}

		// 尝试自动清理
		output.Info("Attempting automatic cleanup...")
		if err := scm.CleanupAll(); err != nil {
			output.Warning("Automatic cleanup failed: " + err.Error())
			output.Println("Manual cleanup may be required")
		} else {
			output.Success("Automatic cleanup completed")
		}
	}

	return nil
}