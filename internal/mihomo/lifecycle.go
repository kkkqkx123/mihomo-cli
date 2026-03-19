package mihomo

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/kkkqkx123/mihomo-cli/internal/config"
	"github.com/kkkqkx123/mihomo-cli/internal/output"
	pkgerrors "github.com/kkkqkx123/mihomo-cli/pkg/errors"
)

// LifecycleHook 生命周期钩子接口
type LifecycleHook interface {
	OnPreStart(ctx context.Context, cfg *config.TomlConfig) error
	OnPostStart(ctx context.Context, pid int) error
	OnPreStop(ctx context.Context, pid int) error
	OnPostStop(ctx context.Context) error
	OnFailure(ctx context.Context, stage LifecycleStage, err error)
}

// LifecycleManager 生命周期管理器
type LifecycleManager struct {
	pm         *ProcessManager
	state      *StateManager
	lock       *ProcessLock
	monitor    *ProcessMonitor
	hooks      []LifecycleHook
	mu         sync.RWMutex
	configFile string
}

// NewLifecycleManager 创建生命周期管理器
func NewLifecycleManager(cfg *config.TomlConfig) (*LifecycleManager, error) {
	configFile := cfg.Mihomo.ConfigFile

	// 创建状态管理器
	state, err := NewStateManager(configFile)
	if err != nil {
		return nil, fmt.Errorf("failed to create state manager: %w", err)
	}

	// 创建进程锁
	lock, err := NewProcessLock(configFile)
	if err != nil {
		return nil, fmt.Errorf("failed to create process lock: %w", err)
	}

	// 创建进程管理器
	pm := NewProcessManager(cfg)

	return &LifecycleManager{
		pm:         pm,
		state:      state,
		lock:       lock,
		configFile: configFile,
	}, nil
}

// RegisterHook 注册生命周期钩子
func (lm *LifecycleManager) RegisterHook(hook LifecycleHook) {
	lm.mu.Lock()
	defer lm.mu.Unlock()
	lm.hooks = append(lm.hooks, hook)
}

// Start 启动进程（包含所有生命周期阶段）
func (lm *LifecycleManager) Start(ctx context.Context, cfg *config.TomlConfig) error {
	// 阶段 1: PreStart
	if err := lm.executeStage(ctx, StagePreStart, func() error {
		return lm.executeHooks(ctx, func(hook LifecycleHook) error {
			return hook.OnPreStart(ctx, cfg)
		})
	}); err != nil {
		return err
	}

	// 获取进程锁
	if err := lm.lock.Acquire(); err != nil {
		return pkgerrors.ErrService("failed to acquire process lock", err)
	}
	defer func() {
		if err := lm.lock.Release(); err != nil {
			output.Warning("failed to release process lock: " + err.Error())
		}
	}()

	// 阶段 2: Starting
	if err := lm.executeStage(ctx, StageStarting, func() error {
		// 设置状态
		if err := lm.state.SetStage(StageStarting); err != nil {
			return err
		}

		// 启动进程
		return lm.pm.Start()
	}); err != nil {
		return err
	}

	// 获取进程信息
	pid := lm.pm.process.Pid
	apiAddress := lm.pm.GetAPIAddress()
	secret := lm.pm.GetSecret()

	// 更新状态
	if err := lm.state.Update(func(state *ProcessState) {
		state.PID = pid
		state.APIAddress = apiAddress
		state.Secret = secret
		state.ConfigFile = lm.configFile
		state.StartedAt = time.Now()
		state.Stage = StageRunning
	}); err != nil {
		return pkgerrors.ErrService("failed to update state", err)
	}

	// 阶段 3: Running
	if err := lm.executeStage(ctx, StageRunning, func() error {
		// 启动监控
		lm.monitor = NewProcessMonitor(pid, 5*time.Second)
		if err := lm.monitor.Start(); err != nil {
			output.Warning("failed to start process monitor: " + err.Error())
		}

		// 执行 PostStart 钩子
		return lm.executeHooks(ctx, func(hook LifecycleHook) error {
			return hook.OnPostStart(ctx, pid)
		})
	}); err != nil {
		return err
	}

	return nil
}

// Stop 停止进程（包含所有生命周期阶段）
func (lm *LifecycleManager) Stop(ctx context.Context, pid int) error {
	// 阶段 1: PreStop
	if err := lm.executeStage(ctx, StagePreStop, func() error {
		// 设置状态
		if err := lm.state.SetStage(StagePreStop); err != nil {
			return err
		}

		// 执行 PreStop 钩子
		return lm.executeHooks(ctx, func(hook LifecycleHook) error {
			return hook.OnPreStop(ctx, pid)
		})
	}); err != nil {
		return err
	}

	// 阶段 2: Stopping
	if err := lm.executeStage(ctx, StageStopping, func() error {
		// 设置状态
		if err := lm.state.SetStage(StageStopping); err != nil {
			return err
		}

		// 停止监控
		if lm.monitor != nil {
			lm.monitor.Stop()
		}

		// 停止进程
		return StopProcessByPID(pid)
	}); err != nil {
		return err
	}

	// 阶段 3: Stopped
	if err := lm.executeStage(ctx, StageStopped, func() error {
		// 清除状态
		if err := lm.state.Clear(); err != nil {
			output.Warning("failed to clear state: " + err.Error())
		}

		// 执行 PostStop 钩子
		return lm.executeHooks(ctx, func(hook LifecycleHook) error {
			return hook.OnPostStop(ctx)
		})
	}); err != nil {
		return err
	}

	return nil
}

// Restart 重启进程
func (lm *LifecycleManager) Restart(ctx context.Context, cfg *config.TomlConfig) error {
	// 获取当前进程状态
	state := lm.state.Get()
	if state == nil || !IsProcessRunning(state.PID) {
		// 进程未运行，直接启动
		return lm.Start(ctx, cfg)
	}

	// 停止进程
	if err := lm.Stop(ctx, state.PID); err != nil {
		return fmt.Errorf("failed to stop process: %w", err)
	}

	// 等待一段时间
	time.Sleep(1 * time.Second)

	// 启动进程
	return lm.Start(ctx, cfg)
}

// GetStage 获取当前生命周期阶段
func (lm *LifecycleManager) GetStage() LifecycleStage {
	state := lm.state.Get()
	if state == nil {
		return StageStopped
	}
	return state.Stage
}

// GetState 获取进程状态
func (lm *LifecycleManager) GetState() *ProcessState {
	return lm.state.Get()
}

// IsRunning 检查进程是否运行
func (lm *LifecycleManager) IsRunning() bool {
	state := lm.state.Get()
	if state == nil {
		return false
	}
	return IsProcessRunning(state.PID)
}

// executeStage 执行生命周期阶段
func (lm *LifecycleManager) executeStage(ctx context.Context, stage LifecycleStage, fn func() error) error {
	if err := fn(); err != nil {
		// 执行失败，调用失败钩子
		lm.executeHooks(ctx, func(hook LifecycleHook) error {
			hook.OnFailure(ctx, stage, err)
			return nil
		})

		// 设置失败状态
		_ = lm.state.SetStage(StageFailed)

		return pkgerrors.ErrService(fmt.Sprintf("failed at stage %s: %v", stage, err), err)
	}
	return nil
}

// executeHooks 执行所有钩子
func (lm *LifecycleManager) executeHooks(ctx context.Context, fn func(hook LifecycleHook) error) error {
	lm.mu.RLock()
	hooks := make([]LifecycleHook, len(lm.hooks))
	copy(hooks, lm.hooks)
	lm.mu.RUnlock()

	var errors []error
	for _, hook := range hooks {
		if err := fn(hook); err != nil {
			errors = append(errors, err)
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("hook execution failed: %v", errors)
	}
	return nil
}

// DefaultLifecycleHooks 默认生命周期钩子
type DefaultLifecycleHooks struct{}

// OnPreStart 启动前钩子
func (d *DefaultLifecycleHooks) OnPreStart(ctx context.Context, cfg *config.TomlConfig) error {
	// 检查可执行文件是否存在
	// 检查配置文件是否有效
	// 检查端口是否被占用
	return nil
}

// OnPostStart 启动后钩子
func (d *DefaultLifecycleHooks) OnPostStart(ctx context.Context, pid int) error {
	// 执行健康检查
	// 记录启动日志
	return nil
}

// OnPreStop 停止前钩子
func (d *DefaultLifecycleHooks) OnPreStop(ctx context.Context, pid int) error {
	// 通知进程准备停止
	// 保存当前状态
	return nil
}

// OnPostStop 停止后钩子
func (d *DefaultLifecycleHooks) OnPostStop(ctx context.Context) error {
	// 清理系统配置
	// 记录停止日志
	return nil
}

// OnFailure 失败钩子
func (d *DefaultLifecycleHooks) OnFailure(ctx context.Context, stage LifecycleStage, err error) {
	// 记录失败日志
	output.Error("Lifecycle failure at stage " + string(stage) + ": " + err.Error())
}
