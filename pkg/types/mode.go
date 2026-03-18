package types

// TunnelMode 隧道模式
type TunnelMode string

const (
	ModeRule   TunnelMode = "rule"
	ModeGlobal TunnelMode = "global"
	ModeDirect TunnelMode = "direct"
)

// ValidModes 所有有效的模式
var ValidModes = []TunnelMode{ModeRule, ModeGlobal, ModeDirect}

// IsValidMode 检查模式是否有效
func IsValidMode(mode string) bool {
	for _, m := range ValidModes {
		if string(m) == mode {
			return true
		}
	}
	return false
}

// ModeInfo 模式信息
type ModeInfo struct {
	Mode string `json:"mode"`
}