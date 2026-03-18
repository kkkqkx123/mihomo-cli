package mihomo

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/kkkqkx123/mihomo-cli/internal/api"
	"github.com/kkkqkx123/mihomo-cli/internal/config"
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
	if cfg.Mihomo.ConfigFile != "" {
		validator := config.NewConfigValidator(cfg.Mihomo.ConfigFile)
		if err := validator.ValidateAndWarn(); err != nil {
			// 配置检查失败不影响启动，只记录警告
			fmt.Printf("Warning: config validation failed: %v\n", err)
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

	fmt.Printf("等待 Mihomo 内核启动（最多 %d 秒）...\n", healthCheckTimeout)

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
				fmt.Println("\nPerforming detailed health check...")
				
				healthChecker := NewHealthChecker(apiClient, cfg.Mihomo.ConfigFile, 5*time.Second)
				healthStatus, err := healthChecker.CheckHealth(ctx)
				if err != nil {
					fmt.Printf("Warning: detailed health check failed: %v\n", err)
					fmt.Println("Process started but may have issues")
				} else {
					healthChecker.PrintHealthStatus(healthStatus)
					
					if !healthChecker.IsHealthy(healthStatus) {
						fmt.Println("\n⚠ Mihomo started but some components may not be working properly")
						fmt.Println("  Check the warnings above for details")
					}
				}

				// 健康检查成功
				fmt.Println("\nMihomo 内核启动成功！")
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

	// 检查系统配置状态
	checker := config.NewSystemChecker()
	if err := checker.CheckAfterStop(); err != nil {
		// 检查失败不影响停止操作，只记录警告
		fmt.Printf("Warning: failed to check system configuration: %v\n", err)
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