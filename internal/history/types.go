package history

import "time"

// Entry 命令历史记录条目
type Entry struct {
	Timestamp time.Time `json:"timestamp"` // 执行时间
	Command   string    `json:"command"`   // 完整命令（包含参数）
	Success   bool      `json:"success"`   // 是否成功
	Error     string    `json:"error,omitempty"` // 错误信息
}