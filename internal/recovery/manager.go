package recovery

import (
	"context"
	"fmt"
	"time"

	"github.com/kkkqkx123/mihomo-cli/internal/output"
	"github.com/kkkqkx123/mihomo-cli/internal/system"
)

// RecoveryManager 恢复管理器（无状态设计）
type RecoveryManager struct {
	detector *ProblemDetector
	executor *RecoveryExecutor
	strategy *RecoveryStrategy
	manager  *system.SystemConfigManager
	config   *RecoveryConfig
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
	return rm.detector.CheckAll()
}

// Recover 执行恢复（无状态，每次独立执行）
// force 为 true 时跳过需要确认的问题检查
func (rm *RecoveryManager) Recover(ctx context.Context, problems []*system.Problem, force bool) (*RecoveryReport, error) {
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

	// 处理每个问题（非阻塞，单次执行）
	var errors []error
	var skippedProblems []*system.Problem

	for _, problem := range problems {
		// 检查是否需要确认
		if rm.strategy.RequireConfirm(problem) && !force {
			// 记录跳过的问题
			skippedProblems = append(skippedProblems, problem)
			continue
		}

		// 获取恢复动作
		action := rm.strategy.GetAction(problem)

		// 高风险操作警告
		if rm.strategy.RequireConfirm(problem) && force {
			output.Warningf("Executing high-risk action: %s for problem: %s", action, problem.Type)
		}

		// 单次执行，使用 context 超时控制
		actionStart := time.Now()
		err := rm.executor.Execute(ctx, problem)
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

	// 记录跳过的问题
	report.SkippedProblems = skippedProblems

	// 设置报告结果
	report.Duration = time.Since(startTime)
	report.Success = len(errors) == 0
	if len(errors) > 0 {
		report.ErrorMessage = fmt.Sprintf("%d errors occurred: %v", len(errors), errors)
	}

	// 不保存状态，直接返回
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

	// 执行恢复（自动恢复不强制执行高风险操作）
	return rm.Recover(ctx, autoProblems, false)
}

// GetConfig 获取配置
func (rm *RecoveryManager) GetConfig() *RecoveryConfig {
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

// CheckAndRecover 检查并恢复（手动触发）
// force 为 true 时强制执行高风险操作
func (rm *RecoveryManager) CheckAndRecover(ctx context.Context, force bool) (*RecoveryReport, error) {
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
	return rm.Recover(ctx, problems, force)
}

// CheckAndRecoverWithFilter 检查并恢复（手动触发，支持过滤问题类型）
// force 为 true 时强制执行高风险操作
// problemType 为空时处理所有问题，否则只处理指定类型的问题
func (rm *RecoveryManager) CheckAndRecoverWithFilter(ctx context.Context, force bool, problemType string) (*RecoveryReport, error) {
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

	// 过滤问题类型
	var filteredProblems []*system.Problem
	if problemType != "" {
		targetType := system.ProblemType(problemType)
		for _, problem := range problems {
			if problem.Type == targetType {
				filteredProblems = append(filteredProblems, problem)
			}
		}
		if len(filteredProblems) == 0 {
			return &RecoveryReport{
				Timestamp:       time.Now(),
				Problems:        problems,
				SkippedProblems: problems,
				Success:         true,
			}, nil
		}
	} else {
		filteredProblems = problems
	}

	// 执行恢复
	return rm.Recover(ctx, filteredProblems, force)
}
