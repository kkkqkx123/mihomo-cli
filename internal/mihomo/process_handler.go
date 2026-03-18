package mihomo

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

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
func (ph *ProcessHandler) Start(ctx context.Context, cfg *config.TomlConfig, foregroundMode bool) (*StartResult, error) {
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
	if err := pm.Start(ctx); err != nil {
		return nil, pkgerrors.ErrService("failed to start mihomo", err)
	}

	result := &StartResult{
		APIAddress: pm.GetAPIAddress(),
		Secret:     pm.GetSecret(),
	}

	// 前台模式：等待中断信号
	if foregroundMode {
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

		<-sigChan

		// 停止内核
		if err := pm.Stop(); err != nil {
			return nil, pkgerrors.ErrService("failed to stop mihomo", err)
		}
	}

	return result, nil
}

// StopResult 停止结果
type StopResult struct {
	PID int
}

// Stop 停止 Mihomo 内核
func (ph *ProcessHandler) Stop(cfg *config.TomlConfig, stopAll bool, stopForce bool, stopConfig string, args []string) (*StopResult, error) {
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

		// 验证进程
		if err := ValidateProcess(pid, stopForce); err != nil {
			return nil, err
		}

		// 停止进程
		if err := StopProcessByPID(pid); err != nil {
			return nil, err
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
	if err := pm.StopByPID(pid); err != nil {
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