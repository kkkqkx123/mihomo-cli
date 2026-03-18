package recovery

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/kkkqkx123/mihomo-cli/internal/system"
)

// RecoveryHandler 恢复处理器接口
type RecoveryHandler interface {
	CanHandle(problem *system.Problem) bool
	Handle(ctx context.Context, problem *system.Problem) error
}

// RecoveryExecutor 恢复执行器
type RecoveryExecutor struct {
	handlers map[system.ProblemType]RecoveryHandler
	mu       sync.RWMutex
}

// NewRecoveryExecutor 创建恢复执行器
func NewRecoveryExecutor() *RecoveryExecutor {
	return &RecoveryExecutor{
		handlers: make(map[system.ProblemType]RecoveryHandler),
	}
}

// RegisterHandler 注册处理器
func (re *RecoveryExecutor) RegisterHandler(problemType system.ProblemType, handler RecoveryHandler) {
	re.mu.Lock()
	defer re.mu.Unlock()
	re.handlers[problemType] = handler
}

// Execute 执行恢复
func (re *RecoveryExecutor) Execute(ctx context.Context, problem *system.Problem) error {
	re.mu.RLock()
	handler, ok := re.handlers[problem.Type]
	re.mu.RUnlock()

	if !ok {
		return fmt.Errorf("no handler for problem type: %s", problem.Type)
	}

	if !handler.CanHandle(problem) {
		return fmt.Errorf("handler cannot handle problem: %s", problem.Type)
	}

	return handler.Handle(ctx, problem)
}

// ExecuteWithRetry 带重试的恢复
func (re *RecoveryExecutor) ExecuteWithRetry(ctx context.Context, problem *system.Problem, maxRetry int, interval time.Duration) error {
	var lastErr error

	for i := 0; i < maxRetry; i++ {
		err := re.Execute(ctx, problem)
		if err == nil {
			return nil
		}

		lastErr = err

		// 等待重试间隔
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(interval):
		}
	}

	return fmt.Errorf("recovery failed after %d retries: %w", maxRetry, lastErr)
}

// CanHandle 检查是否能处理问题
func (re *RecoveryExecutor) CanHandle(problem *system.Problem) bool {
	re.mu.RLock()
	handler, ok := re.handlers[problem.Type]
	re.mu.RUnlock()

	if !ok {
		return false
	}

	return handler.CanHandle(problem)
}

// GetHandlerTypes 获取所有处理器类型
func (re *RecoveryExecutor) GetHandlerTypes() []system.ProblemType {
	re.mu.RLock()
	defer re.mu.RUnlock()

	types := make([]system.ProblemType, 0, len(re.handlers))
	for t := range re.handlers {
		types = append(types, t)
	}
	return types
}

// CleanupHandler 清理处理器
type CleanupHandler struct {
	manager *system.SystemConfigManager
}

// NewCleanupHandler 创建清理处理器
func NewCleanupHandler(manager *system.SystemConfigManager) *CleanupHandler {
	return &CleanupHandler{
		manager: manager,
	}
}

// CanHandle 检查是否能处理问题
func (ch *CleanupHandler) CanHandle(problem *system.Problem) bool {
	return problem.Type == system.ProblemConfigResidual
}

// Handle 处理问题
func (ch *CleanupHandler) Handle(ctx context.Context, problem *system.Problem) error {
	// 执行清理
	return ch.manager.CleanupAll()
}

// RestartHandler 重启处理器
type RestartHandler struct {
	// restartFunc 重启函数
	restartFunc func(ctx context.Context) error
}

// NewRestartHandler 创建重启处理器
func NewRestartHandler(restartFunc func(ctx context.Context) error) *RestartHandler {
	return &RestartHandler{
		restartFunc: restartFunc,
	}
}

// CanHandle 检查是否能处理问题
func (rh *RestartHandler) CanHandle(problem *system.Problem) bool {
	return problem.Type == system.ProblemProcessAbnormal
}

// Handle 处理问题
func (rh *RestartHandler) Handle(ctx context.Context, problem *system.Problem) error {
	if rh.restartFunc == nil {
		return fmt.Errorf("restart function not set")
	}
	return rh.restartFunc(ctx)
}

// RollbackHandler 回滚处理器
type RollbackHandler struct {
	manager *system.SystemConfigManager
}

// NewRollbackHandler 创建回滚处理器
func NewRollbackHandler(manager *system.SystemConfigManager) *RollbackHandler {
	return &RollbackHandler{
		manager: manager,
	}
}

// CanHandle 检查是否能处理问题
func (rh *RollbackHandler) CanHandle(problem *system.Problem) bool {
	return problem.Type == system.ProblemConfigInconsistent
}

// Handle 处理问题
func (rh *RollbackHandler) Handle(ctx context.Context, problem *system.Problem) error {
	// 获取最新的快照
	snapshot, err := rh.manager.GetSnapshotManager().GetLatestSnapshot()
	if err != nil {
		return fmt.Errorf("failed to get latest snapshot: %w", err)
	}

	// 恢复快照
	return rh.manager.RestoreSnapshot(snapshot.ID)
}

// NotifyHandler 通知处理器
type NotifyHandler struct {
	notifyFunc func(problem *system.Problem)
}

// NewNotifyHandler 创建通知处理器
func NewNotifyHandler(notifyFunc func(problem *system.Problem)) *NotifyHandler {
	return &NotifyHandler{
		notifyFunc: notifyFunc,
	}
}

// CanHandle 检查是否能处理问题
func (nh *NotifyHandler) CanHandle(problem *system.Problem) bool {
	return true
}

// Handle 处理问题
func (nh *NotifyHandler) Handle(ctx context.Context, problem *system.Problem) error {
	if nh.notifyFunc != nil {
		nh.notifyFunc(problem)
	}
	return nil
}
