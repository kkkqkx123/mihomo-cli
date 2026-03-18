package auditlog

import (
	"time"
)

// LogLevel 日志级别
type LogLevel int

const (
	LevelDebug LogLevel = iota
	LevelInfo
	LevelWarn
	LevelError
)

// String 返回日志级别字符串
func (l LogLevel) String() string {
	switch l {
	case LevelDebug:
		return "DEBUG"
	case LevelInfo:
		return "INFO"
	case LevelWarn:
		return "WARN"
	case LevelError:
		return "ERROR"
	default:
		return "UNKNOWN"
	}
}

// ParseLogLevel 解析日志级别
func ParseLogLevel(s string) LogLevel {
	switch s {
	case "DEBUG", "debug":
		return LevelDebug
	case "INFO", "info":
		return LevelInfo
	case "WARN", "warn", "WARNING", "warning":
		return LevelWarn
	case "ERROR", "error":
		return LevelError
	default:
		return LevelInfo
	}
}

// LogCategory 日志分类
type LogCategory string

const (
	CategoryConfig   LogCategory = "config"    // 配置操作
	CategoryProcess  LogCategory = "process"   // 进程操作
	CategorySystem   LogCategory = "system"    // 系统配置
	CategoryAPI      LogCategory = "api"       // API 调用
	CategoryRecovery LogCategory = "recovery"  // 恢复操作
	CategoryUser     LogCategory = "user"      // 用户操作
)

// AuditRecord 审计记录
type AuditRecord struct {
	ID        string                 `json:"id"`
	Timestamp time.Time              `json:"timestamp"`
	Level     LogLevel               `json:"level"`
	Category  LogCategory            `json:"category"`
	Operation string                 `json:"operation"`
	Component string                 `json:"component"`
	Details   map[string]interface{} `json:"details,omitempty"`
	Result    string                 `json:"result"`
	Error     string                 `json:"error,omitempty"`
	Duration  time.Duration          `json:"duration,omitempty"`
}

// LogQuery 日志查询条件
type LogQuery struct {
	StartTime  *time.Time
	EndTime    *time.Time
	Level      *LogLevel
	Category   *LogCategory
	Component  string
	Operation  string
	Result     string
	Limit      int
	Offset     int
	OrderBy    string // "time" or "level"
	Descending bool
}

// QueryResult 查询结果
type QueryResult struct {
	Records []*AuditRecord `json:"records"`
	Total   int            `json:"total"`
}

// LogRotateConfig 日志轮转配置
type LogRotateConfig struct {
	MaxSize    int64         // 最大文件大小（字节）
	MaxAge     time.Duration // 最大保留时间
	MaxBackups int           // 最大备份数量
	Compress   bool          // 是否压缩
}

// DefaultLogRotateConfig 默认日志轮转配置
func DefaultLogRotateConfig() *LogRotateConfig {
	return &LogRotateConfig{
		MaxSize:    10 * 1024 * 1024, // 10 MB
		MaxAge:     7 * 24 * time.Hour, // 7 天
		MaxBackups: 5,
		Compress:   false,
	}
}
