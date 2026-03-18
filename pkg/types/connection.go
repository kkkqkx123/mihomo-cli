package types

// ConnectionInfo 连接信息
type ConnectionInfo struct {
	ID          string `json:"id"`
	Metadata    Metadata `json:"metadata"`
	Upload      int64   `json:"upload"`
	Download    int64   `json:"download"`
	UploadSpeed int64   `json:"uploadSpeed"`
	DownloadSpeed int64 `json:"downloadSpeed"`
	Rule        string  `json:"rule"`
	RulePayload string  `json:"rulePayload"`
	Chains      []string `json:"chains"`
}

// Metadata 连接元数据
type Metadata struct {
	Network     string `json:"network"`
	Type        string `json:"type"`
	SourceIP    string `json:"sourceIP"`
	SourcePort  string `json:"sourcePort"`
	DestinationIP string `json:"destinationIP"`
	DestinationPort string `json:"destinationPort"`
	Host        string `json:"host"`
	DNSMode     string `json:"dnsMode"`
	Process     string `json:"process"`
	ProcessPath string `json:"processPath"`
}

// ConnectionsResponse 连接列表响应
type ConnectionsResponse struct {
	DownloadSpeed int64            `json:"downloadSpeed"`
	UploadSpeed   int64            `json:"uploadSpeed"`
	Connections   []ConnectionInfo `json:"connections"`
}
