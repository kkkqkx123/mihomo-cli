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
			// 健康检查超时
			pm.Stop()
			return nil, pkgerrors.ErrService("mihomo health check timeout: process may have failed to start", nil)

		case <-ticker.C:
			// 检查进程是否还在运行
			if !pm.IsRunning() {
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

			// 尝试连接 API 进行健康检查
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			_, err := apiClient.GetMode(ctx)
			cancel()

			if err == nil {
				// 健康检查成功
				fmt.Println("Mihomo 内核启动成功！")
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

	// 如果指定了 PID 参数
	if len(args) == 1 {
		var pid int
		_, err := fmt.Sscanf(args[0], "%d", &pid)
		if err != nil {
			return nil, pkgerrors.ErrInvalidArg("invalid PID: "+args[0], nil)
		}

		// 验证进程是否在运行
		if !IsProcessRunning(pid) {
			return nil, pkgerrors.ErrService("process "+fmt.Sprintf("%d", pid)+" is not running", nil)
		}

		// 停止进程
		if err := StopProcessByPID(pid); err != nil {
			return nil, pkgerrors.ErrService("failed to stop process", err)
		}

		return &StopResult{PID: pid}, nil
	}

	// 默认：停止当前配置的实例
	pm := NewProcessManager(cfg)

	// 从 PID 文件读取进程 ID
	pid, err := pm.GetPIDFromPIDFile()
	if err != nil {
		return nil, pkgerrors.ErrService("mihomo is not running", err)
	}

	// 停止进程
	if err := pm.Stop(); err != nil {
		return nil, pkgerrors.ErrService("failed to stop mihomo", err)
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