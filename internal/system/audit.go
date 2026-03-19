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

// AuditRecord 审计记录
type AuditRecord struct {
	Timestamp time.Time `json:"timestamp"`
	Operation string    `json:"operation"` // "enable", "disable", "cleanup", etc.
	Component string    `json:"component"` // "sysproxy", "tun", "route", etc.
	Details   string    `json:"details"`
	Result    string    `json:"result"` // "success", "failed"
	Error     string    `json:"error,omitempty"`
}

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

	// 追加写入日志文件（JSONL 格式）
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

	// 打开文件
	f, err := os.Open(al.logFile)
	if err != nil {
		if os.IsNotExist(err) {
			return []AuditRecord{}, nil
		}
		return nil, fmt.Errorf("failed to open audit log file: %w", err)
	}
	defer f.Close()

	// 获取文件大小
	stat, err := f.Stat()
	if err != nil {
		return nil, fmt.Errorf("failed to stat audit log file: %w", err)
	}
	fileSize := stat.Size()

	// 如果文件为空，直接返回
	if fileSize == 0 {
		return []AuditRecord{}, nil
	}

	// 从文件末尾向前读取，直到收集足够的记录
	var records []AuditRecord
	buf := make([]byte, 4096) // 4KB 缓冲区
	lineBuf := make([]byte, 0, 1024)
	offset := fileSize

	for offset > 0 && (limit <= 0 || len(records) < limit) {
		// 计算本次读取的大小
		readSize := int64(len(buf))
		if offset < readSize {
			readSize = offset
		}

		// 读取数据
		offset -= readSize
		_, err := f.Seek(offset, 0)
		if err != nil {
			break
		}
		n, err := f.Read(buf[:readSize])
		if err != nil {
			break
		}

		// 从后向前处理数据
		for i := n - 1; i >= 0; i-- {
			if buf[i] == '\n' {
				// 找到一行，解析它
				if len(lineBuf) > 0 {
					line := reverseBytes(lineBuf)
					if record, ok := al.parseAndFilterRecord(line, component, since); ok {
						records = append(records, record)
						if limit > 0 && len(records) >= limit {
							break
						}
					}
				}
				lineBuf = lineBuf[:0] // 重置行缓冲区
			} else {
				lineBuf = append(lineBuf, buf[i])
			}
		}
	}

	// 处理最后一行（文件开头可能没有换行符）
	if len(lineBuf) > 0 && (limit <= 0 || len(records) < limit) {
		line := reverseBytes(lineBuf)
		if record, ok := al.parseAndFilterRecord(line, component, since); ok {
			records = append(records, record)
		}
	}

	// 反转记录顺序（因为我们是从后向前读取的）
	for i, j := 0, len(records)-1; i < j; i, j = i+1, j-1 {
		records[i], records[j] = records[j], records[i]
	}

	return records, nil
}

// parseAndFilterRecord 解析并过滤记录
func (al *AuditLogger) parseAndFilterRecord(line []byte, component string, since time.Time) (AuditRecord, bool) {
	var record AuditRecord
	if err := json.Unmarshal(line, &record); err != nil {
		return AuditRecord{}, false
	}

	// 过滤条件
	if component != "" && record.Component != component {
		return AuditRecord{}, false
	}
	if !since.IsZero() && record.Timestamp.Before(since) {
		return AuditRecord{}, false
	}

	return record, true
}

// reverseBytes 反转字节切片
func reverseBytes(b []byte) []byte {
	result := make([]byte, len(b))
	for i := range b {
		result[i] = b[len(b)-1-i]
	}
	return result
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

// Prune 清理指定时间之前的审计日志
// 返回删除的记录数量
func (al *AuditLogger) Prune(before time.Time) (int, error) {
	al.mu.Lock()
	defer al.mu.Unlock()

	// 读取日志文件
	data, err := os.ReadFile(al.logFile)
	if err != nil {
		if os.IsNotExist(err) {
			return 0, nil
		}
		return 0, fmt.Errorf("failed to read audit log file: %w", err)
	}

	// 解析并过滤
	var keptRecords []string
	var removed int
	lines := strings.Split(string(data), "\n")
	for _, line := range lines {
		if line == "" {
			continue
		}

		var record AuditRecord
		if err := json.Unmarshal([]byte(line), &record); err != nil {
			continue
		}

		// 保留指定时间之后的记录
		if record.Timestamp.After(before) || record.Timestamp.Equal(before) {
			keptRecords = append(keptRecords, line)
		} else {
			removed++
		}
	}

	// 如果没有删除任何记录，直接返回
	if removed == 0 {
		return 0, nil
	}

	// 写回文件
	f, err := os.Create(al.logFile)
	if err != nil {
		return 0, fmt.Errorf("failed to create audit log file: %w", err)
	}
	defer f.Close()

	for _, line := range keptRecords {
		if _, err := f.WriteString(line + "\n"); err != nil {
			return 0, fmt.Errorf("failed to write audit log: %w", err)
		}
	}

	return removed, nil
}

// GetLogFile 获取日志文件路径
func (al *AuditLogger) GetLogFile() string {
	return al.logFile
}
