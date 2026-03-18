package config

import (
	"fmt"

	"github.com/kkkqkx123/mihomo-cli/internal/sysproxy"
)

// SystemChecker 系统配置检查器
type SystemChecker struct{}

// NewSystemChecker 创建系统配置检查器
func NewSystemChecker() *SystemChecker {
	return &SystemChecker{}
}

// SystemStatus 系统配置状态
type SystemStatus struct {
	ProxyEnabled bool   // 系统代理是否启用
	ProxyServer  string // 代理服务器地址
	TunDetected  bool   // 是否检测到TUN设备
	Warnings     []string // 警告信息
}

// CheckSystemConfig 检查系统配置状态
func (sc *SystemChecker) CheckSystemConfig() (*SystemStatus, error) {
	status := &SystemStatus{
		Warnings: make([]string, 0),
	}

	// 检查系统代理状态
	sp := sysproxy.NewSysProxy()
	if sp.IsSupported() {
		proxyStatus, err := sp.GetStatus()
		if err != nil {
			status.Warnings = append(status.Warnings, fmt.Sprintf("Failed to check system proxy status: %v", err))
		} else {
			status.ProxyEnabled = proxyStatus.Enabled
			status.ProxyServer = proxyStatus.Server

			// 如果系统代理启用，发出警告
			if proxyStatus.Enabled {
				status.Warnings = append(status.Warnings, 
					fmt.Sprintf("System proxy is enabled: %s", proxyStatus.Server),
					"This may affect network connectivity if Mihomo is not running",
					"To disable, run: mihomo-cli sysproxy set off")
			}
		}
	}

	// 检查TUN设备（目前仅提示，不实际检测）
	// TODO: 实现实际的TUN设备检测
	status.TunDetected = false
	// 如果启用了TUN模式，这里应该检测TUN设备

	return status, nil
}

// PrintSystemStatus 打印系统配置状态
func (sc *SystemChecker) PrintSystemStatus(status *SystemStatus) {
	fmt.Println("System Configuration Status:")
	fmt.Println("=============================")

	// 打印系统代理状态
	fmt.Printf("System Proxy: ")
	if status.ProxyEnabled {
		fmt.Printf("ENABLED (%s)\n", status.ProxyServer)
	} else {
		fmt.Println("DISABLED")
	}

	// 打印TUN设备状态
	fmt.Printf("TUN Device: ")
	if status.TunDetected {
		fmt.Println("DETECTED")
	} else {
		fmt.Println("NOT DETECTED")
	}

	// 打印警告信息
	if len(status.Warnings) > 0 {
		fmt.Println("\nWarnings:")
		fmt.Println("---------")
		for i, warning := range status.Warnings {
			fmt.Printf("%d. %s\n", i+1, warning)
		}
	}

	fmt.Println("=============================")
}

// CheckAfterStop 在停止Mihomo后检查系统配置
func (sc *SystemChecker) CheckAfterStop() error {
	fmt.Println("\nChecking system configuration after stopping Mihomo...")
	fmt.Println("------------------------------------------------")

	status, err := sc.CheckSystemConfig()
	if err != nil {
		return fmt.Errorf("failed to check system configuration: %w", err)
	}

	sc.PrintSystemStatus(status)

	// 如果有警告，提供恢复建议
	if len(status.Warnings) > 0 {
		fmt.Println("\nRecovery Suggestions:")
		fmt.Println("---------------------")
		
		if status.ProxyEnabled {
			fmt.Println("System proxy is still enabled. To disable:")
			fmt.Println("  1. Run: mihomo-cli sysproxy set off")
			fmt.Println("  2. Or manually disable through Windows Settings")
			fmt.Println("  3. Or restart the computer")
		}

		if status.TunDetected {
			fmt.Println("\nTUN device may still exist. To clean up:")
			fmt.Println("  1. Delete TUN network adapter (Windows: Network Adapter Settings)")
			fmt.Println("  2. Clean up routing table")
			fmt.Println("  3. Or restart the computer")
		}

		fmt.Println("\nIf you continue to experience network issues, restart the computer")
		fmt.Println("to ensure all system configurations are cleaned up.")
	} else {
		fmt.Println("\n✓ System configuration appears to be clean")
	}

	fmt.Println("------------------------------------------------")
	return nil
}
