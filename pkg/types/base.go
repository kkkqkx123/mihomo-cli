package types

// VersionInfo 版本信息响应
type VersionInfo struct {
	Version     string `json:"version"`
	PreRelease  bool   `json:"premium"`
	HomeDir     string `json:"homeDir,omitempty"`
	ConfigPath  string `json:"configPath,omitempty"`
}

// LogInfo 日志信息（用于 WebSocket 流）
type LogInfo struct {
	LogType string `json:"type"` // info, warning, error, debug, silent
	Payload string `json:"payload"`
}
