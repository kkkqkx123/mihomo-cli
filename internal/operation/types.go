package operation

import "time"

// Record 操作记录
type Record struct {
	Timestamp time.Time `json:"timestamp"`
	Operation string    `json:"operation"` // "enable", "disable", "cleanup", etc.
	Component string    `json:"component"` // "sysproxy", "tun", "route", etc.
	Details   string    `json:"details"`
	Result    string    `json:"result"` // "success", "failed"
	Error     string    `json:"error,omitempty"`
}
