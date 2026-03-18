package api

import (
	"context"
	"fmt"
)

// ConfigInfo Mihomo 配置信息
type ConfigInfo struct {
	Port           int          `json:"port"`
	SocksPort      int          `json:"socks-port"`
	RedirPort      int          `json:"redir-port"`
	TproxyPort     int          `json:"tproxy-port"`
	MixedPort      int          `json:"mixed-port"`
	AllowLan       bool         `json:"allow-lan"`
	BindAddress    string       `json:"bind-address"`
	Mode           string       `json:"mode"`
	LogLevel       string       `json:"log-level"`
	IPv6           bool         `json:"ipv6"`
	Sniffing       bool         `json:"sniffing"`
	TCPConcurrent  bool         `json:"tcp-concurrent"`
	FindProcessMode string      `json:"find-process-mode"`
	InterfaceName  string       `json:"interface-name"`
	Tun            *TunConfig   `json:"tun"`
	TuicServer     *TuicConfig  `json:"tuic-server"`
}

// TunConfig TUN 配置
type TunConfig struct {
	Enable              bool     `json:"enable"`
	Device              string   `json:"device"`
	Stack               string   `json:"stack"`
	DNSHijack           []string `json:"dns-hijack"`
	AutoRoute           bool     `json:"auto-route"`
	AutoDetectInterface bool     `json:"auto-detect-interface"`
	MTU                 int      `json:"mtu"`
	GSO                 bool     `json:"gso"`
	GSOMaxSize          int      `json:"gso-max-size"`
	Inet6Address        []string `json:"inet6-address"`
}

// TuicConfig TUIC 服务器配置
type TuicConfig struct {
	Enable       bool     `json:"enable"`
	Listen       string   `json:"listen"`
	Token        []string `json:"token"`
	Certificate  string   `json:"certificate"`
	PrivateKey   string   `json:"private-key"`
}

// ReloadConfigRequest 重载配置请求
type ReloadConfigRequest struct {
	Path    string `json:"path"`
	Payload string `json:"payload"`
}

// GetConfig 获取当前配置信息
func (c *Client) GetConfig(ctx context.Context) (*ConfigInfo, error) {
	var result ConfigInfo
	err := c.Get(ctx, "/configs", nil, &result)
	if err != nil {
		return nil, fmt.Errorf("获取配置失败: %w", err)
	}
	return &result, nil
}

// PatchConfig 部分更新配置（热更新）
func (c *Client) PatchConfig(ctx context.Context, patch map[string]interface{}) error {
	err := c.Patch(ctx, "/configs", nil, patch, nil)
	if err != nil {
		return fmt.Errorf("热更新配置失败: %w", err)
	}
	return nil
}

// ReloadConfig 重载完整配置文件
func (c *Client) ReloadConfig(ctx context.Context, path string, force bool) error {
	// 构建查询参数
	queryParams := make(map[string]string)
	if force {
		queryParams["force"] = "true"
	}

	// 构建请求体
	req := ReloadConfigRequest{
		Path: path,
	}

	err := c.Put(ctx, "/configs", queryParams, req, nil)
	if err != nil {
		return fmt.Errorf("重载配置失败: %w", err)
	}
	return nil
}

// UpdateGeo 更新 Geo 数据库
func (c *Client) UpdateGeo(ctx context.Context) error {
	err := c.Post(ctx, "/configs/geo", nil, nil, nil)
	if err != nil {
		return fmt.Errorf("更新 Geo 数据库失败: %w", err)
	}
	return nil
}
