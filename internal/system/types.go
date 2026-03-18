package system

import (
	"time"
)

// ConfigState 系统配置状态
type ConfigState struct {
	SysProxy  *ProxySettings `json:"sysproxy"`
	TUN       *TUNState      `json:"tun"`
	Routes    []RouteEntry   `json:"routes"`
	IPTables  []IPTablesRule `json:"iptables,omitempty"` // Linux only
	Timestamp time.Time      `json:"timestamp"`
}

// ProxySettings 代理设置
type ProxySettings struct {
	Enabled    bool   `json:"enabled"`
	Server     string `json:"server"`
	BypassList string `json:"bypass_list"`
}

// TUNState TUN 网卡状态
type TUNState struct {
	Name      string `json:"name"`
	Enabled   bool   `json:"enabled"`
	IPAddress string `json:"ip_address"`
	MTU       int    `json:"mtu"`
}

// RouteEntry 路由表项
type RouteEntry struct {
	Destination string `json:"destination"`
	Gateway     string `json:"gateway"`
	Interface   string `json:"interface"`
	Metric      int    `json:"metric"`
}

// IPTablesRule iptables 规则
type IPTablesRule struct {
	Table    string `json:"table"`
	Chain    string `json:"chain"`
	Rule     string `json:"rule"`
	Target   string `json:"target"`
	Protocol string `json:"protocol"`
}

// ConfigSnapshot 配置快照
type ConfigSnapshot struct {
	ID        string      `json:"id"`
	State     ConfigState `json:"state"`
	CreatedAt time.Time   `json:"created_at"`
	Note      string      `json:"note"`
}

// ProblemType 问题类型
type ProblemType string

const (
	ProblemConfigResidual     ProblemType = "config-residual"     // 配置残留
	ProblemProcessAbnormal    ProblemType = "process-abnormal"    // 进程异常
	ProblemConfigInconsistent ProblemType = "config-inconsistent" // 配置不一致
	ProblemPortConflict       ProblemType = "port-conflict"       // 端口冲突
	ProblemPermissionDenied   ProblemType = "permission-denied"   // 权限不足
)

// Severity 严重程度
type Severity string

const (
	SeverityCritical Severity = "critical" // 严重
	SeverityHigh     Severity = "high"     // 高
	SeverityMedium   Severity = "medium"   // 中
	SeverityLow      Severity = "low"      // 低
)

// Problem 检测到的问题
type Problem struct {
	Type        ProblemType            `json:"type"`
	Severity    Severity               `json:"severity"`
	Description string                 `json:"description"`
	Details     map[string]interface{} `json:"details"`
	Solutions   []Solution             `json:"solutions"`
}

// Solution 解决方案
type Solution struct {
	Description string `json:"description"`
	Command     string `json:"command"`
	Auto        bool   `json:"auto"` // 是否可以自动执行
}
