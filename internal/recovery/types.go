package recovery

import (
	"time"

	"github.com/kkkqkx123/mihomo-cli/internal/system"
)

// RecoveryConfig 恢复配置
type RecoveryConfig struct {
	Enabled             bool          `json:"enabled"`
	AutoRecover         bool          `json:"auto_recover"`
	BackupBeforeRecover bool          `json:"backup_before_recover"`
	Timeout             time.Duration `json:"timeout"` // 单次执行超时时间
	Components          []string      `json:"components"`
}

// DefaultRecoveryConfig 默认恢复配置
func DefaultRecoveryConfig() *RecoveryConfig {
	return &RecoveryConfig{
		Enabled:             true,
		AutoRecover:         true,
		BackupBeforeRecover: true,
		Timeout:             30 * time.Second,
		Components:          []string{"sysproxy", "tun", "route"},
	}
}

// RecoveryAction 恢复动作
type RecoveryAction string

const (
	ActionCleanup  RecoveryAction = "cleanup"  // 清理配置
	ActionRestart  RecoveryAction = "restart"  // 重启进程
	ActionRollback RecoveryAction = "rollback" // 回滚配置
	ActionRepair   RecoveryAction = "repair"   // 修复配置
	ActionNotify   RecoveryAction = "notify"   // 仅通知
)

// RecoveryRule 恢复规则
type RecoveryRule struct {
	ProblemType    system.ProblemType `json:"problem_type"`
	Severity       system.Severity    `json:"severity"`
	Action         RecoveryAction     `json:"action"`
	RequireConfirm bool               `json:"require_confirm"`
	AutoRecover    bool               `json:"auto_recover"`
	MaxRetry       int                `json:"max_retry"`
}

// RecoveryActionRecord 恢复动作记录
type RecoveryActionRecord struct {
	Problem      *system.Problem `json:"problem"`
	Action       RecoveryAction  `json:"action"`
	Success      bool            `json:"success"`
	ErrorMessage string          `json:"error_message,omitempty"`
	Duration     time.Duration   `json:"duration"`
	Timestamp    time.Time       `json:"timestamp"`
}

// RecoveryReport 恢复报告
type RecoveryReport struct {
	Timestamp       time.Time              `json:"timestamp"`
	Problems        []*system.Problem      `json:"problems"`
	Actions         []RecoveryActionRecord `json:"actions"`
	SkippedProblems []*system.Problem      `json:"skipped_problems,omitempty"` // 跳过的问题（需要确认但未强制执行）
	Success         bool                   `json:"success"`
	ErrorMessage    string                 `json:"error_message,omitempty"`
	Duration        time.Duration          `json:"duration"`
}

// RecoveryStatus 恢复状态
type RecoveryStatus struct {
	Enabled bool `json:"enabled"`
}
