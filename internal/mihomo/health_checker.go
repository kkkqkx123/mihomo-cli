package mihomo

import (
	"context"
	"fmt"
	"time"

	"github.com/kkkqkx123/mihomo-cli/internal/api"
	"github.com/kkkqkx123/mihomo-cli/internal/output"
)

// HealthChecker 健康检查器
type HealthChecker struct {
	client      *api.Client
	timeout     time.Duration
	configPath  string
}

// HealthStatus 健康状态
type HealthStatus struct {
	APIHealthy     bool   // API是否健康
	TunnelHealthy  bool   // 隧道是否健康
	TunEnabled     bool   // TUN是否启用
	TunHealthy     bool   // TUN是否健康
	ProxyHealthy   bool   // 代理是否健康
	Errors         []string // 错误信息
	Warnings       []string // 警告信息
}

// NewHealthChecker 创建健康检查器
func NewHealthChecker(client *api.Client, configPath string, timeout time.Duration) *HealthChecker {
	return &HealthChecker{
		client:     client,
		timeout:    timeout,
		configPath: configPath,
	}
}

// CheckHealth 执行健康检查
func (hc *HealthChecker) CheckHealth(ctx context.Context) (*HealthStatus, error) {
	status := &HealthStatus{
		Errors:   make([]string, 0),
		Warnings: make([]string, 0),
	}

	// 1. 检查API是否健康
	if err := hc.checkAPI(ctx, status); err != nil {
		status.Errors = append(status.Errors, fmt.Sprintf("API check failed: %v", err))
		return status, fmt.Errorf("API health check failed: %w", err)
	}

	// 2. 检查配置信息
	configInfo, err := hc.client.GetConfig(ctx)
	if err != nil {
		status.Errors = append(status.Errors, fmt.Sprintf("Failed to get config: %v", err))
		return status, fmt.Errorf("failed to get config: %w", err)
	}

	// 3. 检查TUN状态
	if configInfo.Tun != nil && configInfo.Tun.Enable {
		status.TunEnabled = true
		if err := hc.checkTunStatus(status); err != nil {
			status.Warnings = append(status.Warnings, 
				fmt.Sprintf("TUN mode is enabled but cannot verify status: %v", err),
				"This may affect network connectivity if the process crashes")
		}
	}

	// 4. 检查代理状态
	if err := hc.checkProxyStatus(ctx, status); err != nil {
		status.Warnings = append(status.Warnings, 
			fmt.Sprintf("Proxy status check failed: %v", err))
	}

	// 5. 检查隧道状态
	status.TunnelHealthy = status.APIHealthy && status.ProxyHealthy

	return status, nil
}

// checkAPI 检查API是否健康
func (hc *HealthChecker) checkAPI(ctx context.Context, status *HealthStatus) error {
	// 尝试获取版本信息
	version, err := hc.client.GetVersion(ctx)
	if err != nil {
		status.APIHealthy = false
		return err
	}

	status.APIHealthy = true
	if version != nil {
		output.Printf("  Mihomo version: %s\n", version.Version)
	}

	return nil
}

// checkTunStatus 检查TUN状态
func (hc *HealthChecker) checkTunStatus(status *HealthStatus) error {
	// TODO: 实现实际的TUN状态检查
	// 目前只能通过配置信息判断是否启用
	// 实际实现可能需要调用系统API检查TUN网卡状态
	
	status.TunHealthy = true // 假设启用就是健康的
	status.Warnings = append(status.Warnings, 
		"TUN mode is enabled. If the process crashes or is forcefully terminated,",
		"you may need to manually clean up the TUN network adapter and routing table.")
	
	return nil
}

// checkProxyStatus 检查代理状态
func (hc *HealthChecker) checkProxyStatus(ctx context.Context, status *HealthStatus) error {
	// 尝试获取代理列表
	proxies, err := hc.client.ListProxies(ctx)
	if err != nil {
		status.ProxyHealthy = false
		return err
	}

	status.ProxyHealthy = true

	// 检查代理组状态
	for name, proxy := range proxies {
		if proxy.Type == "Selector" || proxy.Type == "URLTest" || proxy.Type == "LoadBalance" {
			// 这是一个代理组
			if proxy.Now == "" {
				status.Warnings = append(status.Warnings, 
					fmt.Sprintf("Proxy group '%s' has no selected proxy", name))
			}
		}
	}

	return nil
}

// PrintHealthStatus 打印健康状态
func (hc *HealthChecker) PrintHealthStatus(status *HealthStatus) {
	output.PrintEmptyLine()
	output.PrintSection("Health Check Results")
	output.PrintSeparator("=", 80)

	// 打印API状态
	output.Printf("API Status: ")
	if status.APIHealthy {
		output.Success("Healthy")
	} else {
		output.Error("Unhealthy")
	}

	// 打印TUN状态
	output.Printf("TUN Mode: ")
	if status.TunEnabled {
		output.Printf("Enabled (Status: ")
		if status.TunHealthy {
			output.Printf("Healthy)\n")
		} else {
			output.Printf("Unhealthy)\n")
		}
	} else {
		output.Println("Disabled")
	}

	// 打印代理状态
	output.Printf("Proxy Status: ")
	if status.ProxyHealthy {
		output.Success("Healthy")
	} else {
		output.Error("Unhealthy")
	}

	// 打印隧道状态
	output.Printf("Tunnel Status: ")
	if status.TunnelHealthy {
		output.Success("Healthy")
	} else {
		output.Error("Unhealthy")
	}

	// 打印警告
	if len(status.Warnings) > 0 {
		output.PrintSection("Warnings")
		output.PrintSeparator("-", 80)
		for i, warning := range status.Warnings {
			output.Printf("%d. %s\n", i+1, warning)
		}
	}

	// 打印错误
	if len(status.Errors) > 0 {
		output.PrintSection("Errors")
		output.PrintSeparator("-", 80)
		for i, err := range status.Errors {
			output.Printf("%d. %s\n", i+1, err)
		}
	}

	output.PrintSeparator("=", 80)
}

// IsHealthy 判断是否健康
func (hc *HealthChecker) IsHealthy(status *HealthStatus) bool {
	return status.APIHealthy && status.TunnelHealthy
}
