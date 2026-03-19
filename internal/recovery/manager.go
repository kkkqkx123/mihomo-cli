package recovery

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/kkkqkx123/mihomo-cli/internal/output"
	"github.com/kkkqkx123/mihomo-cli/internal/system"
)

// RecoveryManager 恢复管理器
type RecoveryManager struct {
	detector  *ProblemDetector
	executor  *RecoveryExecutor
	strategy  *RecoveryStrategy
	manager   *system.SystemConfigManager
	config    *RecoveryConfig
	mu        sync.RWMutex
	lastCheck time.Time
	lastReport *RecoveryReport
}

// NewRecoveryManager 创建恢复管理器
func NewRecoveryManager(manager *system.SystemConfigManager, config *RecoveryConfig) (*RecoveryManager, error) {
	if config == nil {
		config = DefaultRecoveryConfig()
	}

	// 创建检测器
	detector := NewProblemDetector()

	// 注册默认检查器
	detector.RegisterChecker(NewSysProxyChecker(manager))
	detector.RegisterChecker(NewTUNChecker(manager))
	detector.RegisterChecker(NewRouteChecker(manager))

	// 创建执行器
	executor := NewRecoveryExecutor()

	// 注册默认处理器
	executor.RegisterHandler(system.ProblemConfigResidual, NewCleanupHandler(manager))
	executor.RegisterHandler(system.ProblemConfigInconsistent, NewRollbackHandler(manager))

	// 创建策略
	strategy := DefaultRecoveryStrategy()

	return &RecoveryManager{
		detector: detector,
		executor: executor,
		strategy: strategy,
		manager:  manager,
		config:   config,
	}, nil
}

// Detect 检测问题
func (rm *RecoveryManager) Detect() ([]*system.Problem, error) {
	rm.mu.Lock()
	rm.lastCheck = time.Now()
	rm.mu.Unlock()

	return rm.detector.CheckAll()
}

// Recover 执行恢复
func (rm *RecoveryManager) Recover(ctx context.Context, problems []*system.Problem) (*RecoveryReport, error) {
	startTime := time.Now()

	report := &RecoveryReport{
		Timestamp: startTime,
		Problems:  problems,
		Actions:   []RecoveryActionRecord{},
	}

	// 备份当前状态（如果启用）
	if rm.config.BackupBeforeRecover {
		if _, err := rm.manager.CreateSnapshot("before-recovery"); err != nil {
			// 备份失败，记录但不影响恢复
			output.Warning("failed to create backup snapshot: " + err.Error())
		}
	}

	// 处理每个问题
	var errors []error
	for _, problem := range problems {
		// 检查是否需要确认
		if rm.strategy.RequireConfirm(problem) {
			// TODO: 实现用户确认
			// 这里暂时跳过需要确认的问题
			continue
		}

		// 获取恢复动作
		action := rm.strategy.GetAction(problem)
		maxRetry := rm.strategy.GetMaxRetry(problem)

		// 执行恢复
		actionStart := time.Now()
		err := rm.executor.ExecuteWithRetry(ctx, problem, maxRetry, rm.config.RetryInterval)
		actionDuration := time.Since(actionStart)

		// 记录动作
		actionRecord := RecoveryActionRecord{
			Problem:   problem,
			Action:    action,
			Success:   err == nil,
			Duration:  actionDuration,
			Timestamp: actionStart,
		}
		if err != nil {
			actionRecord.ErrorMessage = err.Error()
			errors = append(errors, err)
		}

		report.Actions = append(report.Actions, actionRecord)
	}

	// 设置报告结果
	report.Duration = time.Since(startTime)
	report.Success = len(errors) == 0
	if len(errors) > 0 {
		report.ErrorMessage = fmt.Sprintf("%d errors occurred: %v", len(errors), errors)
	}

	// 保存报告
	rm.mu.Lock()
	rm.lastReport = report
	rm.mu.Unlock()

	return report, nil
}

// AutoRecover 自动恢复
func (rm *RecoveryManager) AutoRecover(ctx context.Context) (*RecoveryReport, error) {
	if !rm.config.Enabled || !rm.config.AutoRecover {
		return nil, fmt.Errorf("auto recovery is disabled")
	}

	// 检测问题
	problems, err := rm.Detect()
	if err != nil {
		return nil, fmt.Errorf("failed to detect problems: %w", err)
	}

	if len(problems) == 0 {
		return &RecoveryReport{
			Timestamp: time.Now(),
			Problems:  problems,
			Success:   true,
		}, nil
	}

	// 过滤可以自动恢复的问题
	var autoProblems []*system.Problem
	for _, problem := range problems {
		if rm.strategy.ShouldAutoRecover(problem) {
			autoProblems = append(autoProblems, problem)
		}
	}

	if len(autoProblems) == 0 {
		return &RecoveryReport{
			Timestamp: time.Now(),
			Problems:  problems,
			Success:   true,
		}, nil
	}

	// 执行恢复
	return rm.Recover(ctx, autoProblems)
}

// GetRecoveryReport 获取恢复报告
func (rm *RecoveryManager) GetRecoveryReport() *RecoveryReport {
	rm.mu.RLock()
	defer rm.mu.RUnlock()
	return rm.lastReport
}

// GetStatus 获取恢复状态
func (rm *RecoveryManager) GetStatus() *RecoveryStatus {
	rm.mu.RLock()
	defer rm.mu.RUnlock()

	return &RecoveryStatus{
		LastCheckTime: rm.lastCheck,
		LastRecovery:  rm.lastReport,
		Enabled:       rm.config.Enabled,
	}
}

// SetConfig 设置配置
func (rm *RecoveryManager) SetConfig(config *RecoveryConfig) {
	rm.mu.Lock()
	defer rm.mu.Unlock()
	rm.config = config
}

// GetConfig 获取配置
func (rm *RecoveryManager) GetConfig() *RecoveryConfig {
	rm.mu.RLock()
	defer rm.mu.RUnlock()
	return rm.config
}

// RegisterChecker 注册检查器
func (rm *RecoveryManager) RegisterChecker(checker ProblemChecker) {
	rm.detector.RegisterChecker(checker)
}

// RegisterHandler 注册处理器
func (rm *RecoveryManager) RegisterHandler(problemType system.ProblemType, handler RecoveryHandler) {
	rm.executor.RegisterHandler(problemType, handler)
}

// AddRule 添加恢复规则
func (rm *RecoveryManager) AddRule(rule RecoveryRule) {
	rm.strategy.AddRule(rule)
}

// StartPeriodicCheck 启动定期检查
func (rm *RecoveryManager) StartPeriodicCheck(ctx context.Context, interval time.Duration) error {
	if interval <= 0 {
		interval = 5 * time.Minute
	}

	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				// 执行自动恢复
				_, err := rm.AutoRecover(ctx)
				if err != nil {
					output.Error("Auto recovery failed: " + err.Error())
				}
			}
		}
	}()

	return nil
}

// CheckAndRecover 检查并恢复（手动触发）
func (rm *RecoveryManager) CheckAndRecover(ctx context.Context) (*RecoveryReport, error) {
	// 检测问题
	problems, err := rm.Detect()
	if err != nil {
		return nil, fmt.Errorf("failed to detect problems: %w", err)
	}

	if len(problems) == 0 {
		return &RecoveryReport{
			Timestamp: time.Now(),
			Problems:  problems,
			Success:   true,
		}, nil
	}

	// 执行恢复
	return rm.Recover(ctx, problems)
}
