package system

import (
	"fmt"

	"github.com/kkkqkx123/mihomo-cli/internal/operation"
	"github.com/kkkqkx123/mihomo-cli/internal/sysproxy"
)

// SysProxyManager 系统代理管理器
type SysProxyManager struct {
	operation *operation.Manager
}

// NewSysProxyManager 创建系统代理管理器
func NewSysProxyManager(op *operation.Manager) *SysProxyManager {
	return &SysProxyManager{
		operation: op,
	}
}

// Enable 启用系统代理
func (spm *SysProxyManager) Enable(server, bypassList string) error {
	proxy := sysproxy.NewSysProxy()
	if !proxy.IsSupported() {
		return fmt.Errorf("system proxy is not supported on this platform")
	}

	err := proxy.Enable(server, bypassList)
	if spm.operation != nil {
		details := fmt.Sprintf("server=%s, bypass=%s", server, bypassList)
		result := "success"
		if err != nil {
			result = "failed"
		}
		_ = spm.operation.Record("enable", "sysproxy", details, result, err)
	}

	return err
}

// Disable 禁用系统代理
func (spm *SysProxyManager) Disable() error {
	proxy := sysproxy.NewSysProxy()
	if !proxy.IsSupported() {
		return fmt.Errorf("system proxy is not supported on this platform")
	}

	err := proxy.Disable()
	if spm.operation != nil {
		result := "success"
		if err != nil {
			result = "failed"
		}
		_ = spm.operation.Record("disable", "sysproxy", "", result, err)
	}

	return err
}

// GetStatus 获取系统代理状态
func (spm *SysProxyManager) GetStatus() (*ProxySettings, error) {
	proxy := sysproxy.NewSysProxy()
	if !proxy.IsSupported() {
		return nil, fmt.Errorf("system proxy is not supported on this platform")
	}

	settings, err := proxy.GetStatus()
	if err != nil {
		return nil, err
	}

	return &ProxySettings{
		Enabled:    settings.Enabled,
		Server:     settings.Server,
		BypassList: settings.BypassList,
	}, nil
}

// CheckResidual 检查是否有残留配置
func (spm *SysProxyManager) CheckResidual() (*Problem, error) {
	status, err := spm.GetStatus()
	if err != nil {
		return nil, err
	}

	// 如果代理已启用，可能需要清理
	if status.Enabled {
		return &Problem{
			Type:        ProblemConfigResidual,
			Severity:    SeverityMedium,
			Description: "System proxy is enabled, may need cleanup",
			Details: map[string]interface{}{
				"server":      status.Server,
				"bypass_list": status.BypassList,
			},
			Solutions: []Solution{
				{
					Description: "Disable system proxy",
					Command:     "mihomo-cli sysproxy set off",
					Auto:        true,
				},
				{
					Description: "Disable through Windows Settings",
					Command:     "Settings > Network & Internet > Proxy",
					Auto:        false,
				},
			},
		}, nil
	}

	return nil, nil
}

// Cleanup 清理系统代理配置
func (spm *SysProxyManager) Cleanup() error {
	status, err := spm.GetStatus()
	if err != nil {
		return err
	}

	// 如果代理已启用，禁用它
	if status.Enabled {
		return spm.Disable()
	}

	return nil
}
