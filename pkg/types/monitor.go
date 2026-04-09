package types

// TrafficInfo 流量统计信息
type TrafficInfo struct {
	Up        int64 `json:"up"`        // 当前上传速度 (bytes/s)
	Down      int64 `json:"down"`      // 当前下载速度 (bytes/s)
	UpTotal   int64 `json:"upTotal"`   // 总上传流量 (bytes)
	DownTotal int64 `json:"downTotal"` // 总下载流量 (bytes)
}

// MemoryInfo 内存使用信息
type MemoryInfo struct {
	Inuse   uint64 `json:"inuse"`   // 当前内存使用量 (bytes)
	OSLimit uint64 `json:"oslimit"` // 系统内存限制 (bytes)
}

// TrafficSnapshot 流量快照（包含累计数据）
type TrafficSnapshot struct {
	CurrentUpSpeed   int64 `json:"current_up_speed"`    // 当前上传速度 (bytes/s)
	CurrentDownSpeed int64 `json:"current_down_speed"`  // 当前下载速度 (bytes/s)
	TotalUpload      int64 `json:"total_upload"`        // 总上传流量 (bytes)
	TotalDownload    int64 `json:"total_download"`      // 总下载流量 (bytes)
}

// MemorySnapshot 内存快照
type MemorySnapshot struct {
	Inuse   int64  `json:"inuse"`             // 当前内存使用量 (bytes)
	OSLimit *int64 `json:"oslimit,omitempty"` // 系统内存限制 (bytes)
}
