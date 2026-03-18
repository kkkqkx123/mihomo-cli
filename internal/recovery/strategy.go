package recovery

import (
	"github.com/kkkqkx123/mihomo-cli/internal/system"
)

// RecoveryStrategy 恢复策略
type RecoveryStrategy struct {
	rules []RecoveryRule
}

// NewRecoveryStrategy 创建恢复策略
func NewRecoveryStrategy() *RecoveryStrategy {
	return &RecoveryStrategy{
		rules: []RecoveryRule{},
	}
}

// AddRule 添加规则
func (rs *RecoveryStrategy) AddRule(rule RecoveryRule) {
	rs.rules = append(rs.rules, rule)
}

// GetAction 获取恢复动作
func (rs *RecoveryStrategy) GetAction(problem *system.Problem) RecoveryAction {
	for _, rule := range rs.rules {
		if rule.ProblemType == problem.Type && rule.Severity == problem.Severity {
			return rule.Action
		}
	}

	// 默认动作
	return getDefaultAction(problem)
}

// ShouldAutoRecover 是否自动恢复
func (rs *RecoveryStrategy) ShouldAutoRecover(problem *system.Problem) bool {
	for _, rule := range rs.rules {
		if rule.ProblemType == problem.Type && rule.Severity == problem.Severity {
			return rule.AutoRecover
		}
	}

	// 默认不自动恢复
	return false
}

// RequireConfirm 是否需要确认
func (rs *RecoveryStrategy) RequireConfirm(problem *system.Problem) bool {
	for _, rule := range rs.rules {
		if rule.ProblemType == problem.Type && rule.Severity == problem.Severity {
			return rule.RequireConfirm
		}
	}

	// 默认需要确认
	return true
}

// GetMaxRetry 获取最大重试次数
func (rs *RecoveryStrategy) GetMaxRetry(problem *system.Problem) int {
	for _, rule := range rs.rules {
		if rule.ProblemType == problem.Type && rule.Severity == problem.Severity {
			return rule.MaxRetry
		}
	}

	// 默认重试 3 次
	return 3
}

// GetRules 获取所有规则
func (rs *RecoveryStrategy) GetRules() []RecoveryRule {
	return rs.rules
}

// getDefaultAction 获取默认动作
func getDefaultAction(problem *system.Problem) RecoveryAction {
	switch problem.Type {
	case system.ProblemConfigResidual:
		return ActionCleanup
	case system.ProblemProcessAbnormal:
		return ActionRestart
	case system.ProblemConfigInconsistent:
		return ActionRollback
	case system.ProblemPortConflict:
		return ActionNotify
	case system.ProblemPermissionDenied:
		return ActionNotify
	default:
		return ActionNotify
	}
}

// DefaultRecoveryStrategy 默认恢复策略
func DefaultRecoveryStrategy() *RecoveryStrategy {
	strategy := NewRecoveryStrategy()

	// 配置残留 - 清理
	strategy.AddRule(RecoveryRule{
		ProblemType:    system.ProblemConfigResidual,
		Severity:       system.SeverityHigh,
		Action:         ActionCleanup,
		RequireConfirm: false,
		AutoRecover:    true,
		MaxRetry:       3,
	})

	// 配置残留 - 中等严重度
	strategy.AddRule(RecoveryRule{
		ProblemType:    system.ProblemConfigResidual,
		Severity:       system.SeverityMedium,
		Action:         ActionCleanup,
		RequireConfirm: false,
		AutoRecover:    true,
		MaxRetry:       3,
	})

	// 进程异常 - 重启
	strategy.AddRule(RecoveryRule{
		ProblemType:    system.ProblemProcessAbnormal,
		Severity:       system.SeverityCritical,
		Action:         ActionRestart,
		RequireConfirm: true,
		AutoRecover:    false,
		MaxRetry:       2,
	})

	// 配置不一致 - 回滚
	strategy.AddRule(RecoveryRule{
		ProblemType:    system.ProblemConfigInconsistent,
		Severity:       system.SeverityHigh,
		Action:         ActionRollback,
		RequireConfirm: true,
		AutoRecover:    false,
		MaxRetry:       1,
	})

	// 端口冲突 - 通知
	strategy.AddRule(RecoveryRule{
		ProblemType:    system.ProblemPortConflict,
		Severity:       system.SeverityHigh,
		Action:         ActionNotify,
		RequireConfirm: true,
		AutoRecover:    false,
		MaxRetry:       0,
	})

	// 权限不足 - 通知
	strategy.AddRule(RecoveryRule{
		ProblemType:    system.ProblemPermissionDenied,
		Severity:       system.SeverityCritical,
		Action:         ActionNotify,
		RequireConfirm: true,
		AutoRecover:    false,
		MaxRetry:       0,
	})

	return strategy
}
