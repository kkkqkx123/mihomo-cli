package auditlog

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"sync"
	"time"
)

// AuditLogger 审计日志记录器
type AuditLogger struct {
	storage *LogStorage
	rotate  *LogRotate
	level   LogLevel
	mu      sync.Mutex
}

// NewAuditLogger 创建审计日志记录器
func NewAuditLogger(dbPath string, rotateConfig *LogRotateConfig) (*AuditLogger, error) {
	// 创建存储
	storage, err := NewLogStorage(dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to create log storage: %w", err)
	}

	// 创建轮转
	if rotateConfig == nil {
		rotateConfig = DefaultLogRotateConfig()
	}
	rotate := NewLogRotate(rotateConfig, storage)

	return &AuditLogger{
		storage: storage,
		rotate:  rotate,
		level:   LevelInfo,
	}, nil
}

// SetLevel 设置日志级别
func (al *AuditLogger) SetLevel(level LogLevel) {
	al.mu.Lock()
	defer al.mu.Unlock()
	al.level = level
}

// GetLevel 获取日志级别
func (al *AuditLogger) GetLevel() LogLevel {
	al.mu.Lock()
	defer al.mu.Unlock()
	return al.level
}

// Log 记录日志
func (al *AuditLogger) Log(level LogLevel, category LogCategory, operation, component string, details map[string]interface{}, result string, err error, duration time.Duration) error {
	// 检查日志级别
	if level < al.level {
		return nil
	}

	// 创建记录
	record := &AuditRecord{
		ID:        generateID(),
		Timestamp: time.Now(),
		Level:     level,
		Category:  category,
		Operation: operation,
		Component: component,
		Details:   details,
		Result:    result,
		Duration:  duration,
	}

	if err != nil {
		record.Error = err.Error()
	}

	// 插入记录
	if err := al.storage.Insert(record); err != nil {
		return fmt.Errorf("failed to insert audit record: %w", err)
	}

	// 检查是否需要轮转
	if al.rotate.ShouldRotate() {
		go al.rotate.Rotate()
	}

	return nil
}

// LogOperation 记录操作日志
func (al *AuditLogger) LogOperation(operation, component string, details map[string]interface{}, result string, err error) error {
	return al.Log(LevelInfo, CategoryUser, operation, component, details, result, err, 0)
}

// LogConfigChange 记录配置变更
func (al *AuditLogger) LogConfigChange(operation, component string, details map[string]interface{}, result string, err error) error {
	return al.Log(LevelInfo, CategoryConfig, operation, component, details, result, err, 0)
}

// LogProcessEvent 记录进程事件
func (al *AuditLogger) LogProcessEvent(operation string, pid int, result string, err error) error {
	details := map[string]interface{}{
		"pid": pid,
	}
	return al.Log(LevelInfo, CategoryProcess, operation, "mihomo", details, result, err, 0)
}

// LogSystemChange 记录系统配置变更
func (al *AuditLogger) LogSystemChange(operation, component string, details map[string]interface{}, result string, err error) error {
	return al.Log(LevelInfo, CategorySystem, operation, component, details, result, err, 0)
}

// LogAPICall 记录 API 调用
func (al *AuditLogger) LogAPICall(method, path string, statusCode int, duration time.Duration, err error) error {
	details := map[string]interface{}{
		"method":      method,
		"path":        path,
		"status_code": statusCode,
	}
	result := "success"
	if err != nil || statusCode >= 400 {
		result = "failed"
	}
	return al.Log(LevelDebug, CategoryAPI, "call", "api", details, result, err, duration)
}

// LogRecovery 记录恢复操作
func (al *AuditLogger) LogRecovery(operation string, details map[string]interface{}, result string, err error) error {
	return al.Log(LevelWarn, CategoryRecovery, operation, "recovery", details, result, err, 0)
}

// Debug 记录调试日志
func (al *AuditLogger) Debug(category LogCategory, operation, component string, details map[string]interface{}) error {
	return al.Log(LevelDebug, category, operation, component, details, "success", nil, 0)
}

// Info 记录信息日志
func (al *AuditLogger) Info(category LogCategory, operation, component string, details map[string]interface{}) error {
	return al.Log(LevelInfo, category, operation, component, details, "success", nil, 0)
}

// Warn 记录警告日志
func (al *AuditLogger) Warn(category LogCategory, operation, component string, details map[string]interface{}, err error) error {
	return al.Log(LevelWarn, category, operation, component, details, "warning", err, 0)
}

// Error 记录错误日志
func (al *AuditLogger) Error(category LogCategory, operation, component string, details map[string]interface{}, err error) error {
	return al.Log(LevelError, category, operation, component, details, "failed", err, 0)
}

// Query 查询日志
func (al *AuditLogger) Query(query *LogQuery) (*QueryResult, error) {
	return al.storage.Query(query)
}

// QueryByTimeRange 按时间范围查询
func (al *AuditLogger) QueryByTimeRange(start, end time.Time, limit int) ([]*AuditRecord, error) {
	query := &LogQuery{
		StartTime:  &start,
		EndTime:    &end,
		Limit:      limit,
		Descending: true,
	}
	result, err := al.storage.Query(query)
	if err != nil {
		return nil, err
	}
	return result.Records, nil
}

// QueryByCategory 按分类查询
func (al *AuditLogger) QueryByCategory(category LogCategory, limit int) ([]*AuditRecord, error) {
	query := &LogQuery{
		Category:   &category,
		Limit:      limit,
		Descending: true,
	}
	result, err := al.storage.Query(query)
	if err != nil {
		return nil, err
	}
	return result.Records, nil
}

// QueryErrors 查询错误日志
func (al *AuditLogger) QueryErrors(limit int) ([]*AuditRecord, error) {
	level := LevelError
	query := &LogQuery{
		Level:      &level,
		Limit:      limit,
		Descending: true,
	}
	result, err := al.storage.Query(query)
	if err != nil {
		return nil, err
	}
	return result.Records, nil
}

// Clear 清空日志
func (al *AuditLogger) Clear() error {
	return al.storage.Clear()
}

// Close 关闭日志记录器
func (al *AuditLogger) Close() error {
	return al.storage.Close()
}

// Rotate 执行日志轮转
func (al *AuditLogger) Rotate() error {
	return al.rotate.Rotate()
}

// CleanOldLogs 清理旧日志
func (al *AuditLogger) CleanOldLogs() error {
	return al.rotate.CleanOldLogs()
}

// generateID 生成唯一 ID
func generateID() string {
	b := make([]byte, 8)
	rand.Read(b)
	return hex.EncodeToString(b) + "-" + time.Now().Format("20060102150405")
}
