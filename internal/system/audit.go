package system

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// AuditLogger 审计日志记录器
type AuditLogger struct {
	logFile string
	mu      sync.Mutex
}

// NewAuditLogger 创建审计日志记录器
func NewAuditLogger(logFile string) (*AuditLogger, error) {
	// 确保目录存在
	logDir := filepath.Dir(logFile)
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create audit log directory: %w", err)
	}

	return &AuditLogger{
		logFile: logFile,
	}, nil
}

// Record 记录审计日志
func (al *AuditLogger) Record(operation, component, details, result string, err error) error {
	al.mu.Lock()
	defer al.mu.Unlock()

	record := AuditRecord{
		Timestamp: time.Now(),
		Operation: operation,
		Component: component,
		Details:   details,
		Result:    result,
	}

	if err != nil {
		record.Error = err.Error()
	}

	// 追加写入日志文件
	f, err := os.OpenFile(al.logFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("failed to open audit log file: %w", err)
	}
	defer f.Close()

	// 每条记录一行 JSON
	data, err := json.Marshal(record)
	if err != nil {
		return fmt.Errorf("failed to marshal audit record: %w", err)
	}

	if _, err := f.WriteString(string(data) + "\n"); err != nil {
		return fmt.Errorf("failed to write audit record: %w", err)
	}

	return nil
}

// Query 查询审计日志
func (al *AuditLogger) Query(component string, since time.Time, limit int) ([]AuditRecord, error) {
	al.mu.Lock()
	defer al.mu.Unlock()

	// 读取日志文件
	data, err := os.ReadFile(al.logFile)
	if err != nil {
		if os.IsNotExist(err) {
			return []AuditRecord{}, nil
		}
		return nil, fmt.Errorf("failed to read audit log file: %w", err)
	}

	// 解析每一行
	var records []AuditRecord
	lines := strings.Split(string(data), "\n")
	for _, line := range lines {
		if line == "" {
			continue
		}

		var record AuditRecord
		if err := json.Unmarshal([]byte(line), &record); err != nil {
			continue
		}

		// 过滤条件
		if component != "" && record.Component != component {
			continue
		}
		if !since.IsZero() && record.Timestamp.Before(since) {
			continue
		}

		records = append(records, record)
	}

	// 限制数量
	if limit > 0 && len(records) > limit {
		records = records[len(records)-limit:]
	}

	return records, nil
}

// Clear 清空审计日志
func (al *AuditLogger) Clear() error {
	al.mu.Lock()
	defer al.mu.Unlock()

	if err := os.Remove(al.logFile); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to clear audit log: %w", err)
	}

	return nil
}

// GetLogFile 获取日志文件路径
func (al *AuditLogger) GetLogFile() string {
	return al.logFile
}


