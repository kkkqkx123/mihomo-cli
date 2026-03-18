package history

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
)

// Manager 历史记录管理器
type Manager struct {
	filePath string
	mu       sync.Mutex
}

// NewManager 创建历史记录管理器
func NewManager(historyFile string) *Manager {
	return &Manager{
		filePath: historyFile,
	}
}

// Add 添加一条历史记录
func (m *Manager) Add(entry Entry) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// 确保目录存在
	if err := os.MkdirAll(filepath.Dir(m.filePath), 0755); err != nil {
		return err
	}

	// 打开文件（追加模式）
	file, err := os.OpenFile(m.filePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer file.Close()

	// 写入JSON行
	data, err := json.Marshal(entry)
	if err != nil {
		return err
	}

	_, err = file.Write(append(data, '\n'))
	return err
}

// Read 读取所有历史记录
func (m *Manager) Read() ([]Entry, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// 文件不存在返回空列表
	if _, err := os.Stat(m.filePath); os.IsNotExist(err) {
		return []Entry{}, nil
	}

	// 读取文件
	data, err := os.ReadFile(m.filePath)
	if err != nil {
		return nil, err
	}

	var entries []Entry
	lines := splitLines(data)
	for _, line := range lines {
		if len(line) == 0 {
			continue
		}
		var entry Entry
		if err := json.Unmarshal(line, &entry); err != nil {
			continue // 跳过无效行
		}
		entries = append(entries, entry)
	}

	return entries, nil
}

// Clear 清空历史记录
func (m *Manager) Clear() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	return os.Remove(m.filePath)
}

// splitLines 分割JSON行
func splitLines(data []byte) [][]byte {
	var lines [][]byte
	start := 0
	for i, b := range data {
		if b == '\n' {
			lines = append(lines, data[start:i])
			start = i + 1
		}
	}
	if start < len(data) {
		lines = append(lines, data[start:])
	}
	return lines
}