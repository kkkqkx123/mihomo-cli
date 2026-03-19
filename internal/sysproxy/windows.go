//go:build windows

package sysproxy

import (
	"fmt"

	"golang.org/x/sys/windows/registry"

	pkgerrors "github.com/kkkqkx123/mihomo-cli/pkg/errors"
)

const (
	// InternetSettingsKey Internet Settings 注册表键路径
	InternetSettingsKey = `SOFTWARE\Microsoft\Windows\CurrentVersion\Internet Settings`

	// ProxyEnableValue 代理启用标志
	ProxyEnableValue = "ProxyEnable"
	// ProxyServerValue 代理服务器地址
	ProxyServerValue = "ProxyServer"
	// ProxyOverrideValue 代理绕过列表
	ProxyOverrideValue = "ProxyOverride"
)

// windowsSysProxy Windows 系统代理管理器
type windowsSysProxy struct{}

// newWindowsSysProxy 创建新的 Windows 系统代理管理器
func newWindowsSysProxy() SysProxy {
	return &windowsSysProxy{}
}

// WindowsRegistry Windows 注册表操作
type WindowsRegistry struct {
	key registry.Key
}

// NewWindowsRegistry 创建新的注册表操作实例
func NewWindowsRegistry() (*WindowsRegistry, error) {
	// 打开当前用户的 Internet Settings 键
	key, err := registry.OpenKey(
		registry.CURRENT_USER,
		InternetSettingsKey,
		registry.QUERY_VALUE|registry.SET_VALUE,
	)
	if err != nil {
		return nil, pkgerrors.ErrService("failed to open registry key", err)
	}

	return &WindowsRegistry{key: key}, nil
}

// Close 关闭注册表键
func (wr *WindowsRegistry) Close() error {
	return wr.key.Close()
}

// GetSettings 获取当前系统代理设置
func (wr *WindowsRegistry) GetSettings() (*ProxySettings, error) {
	settings := &ProxySettings{}

	// 读取 ProxyEnable 值
	enabled, _, err := wr.key.GetIntegerValue(ProxyEnableValue)
	if err != nil {
		// 如果值不存在，默认为禁用
		settings.Enabled = false
	} else {
		settings.Enabled = enabled != 0
	}

	// 读取 ProxyServer 值
	server, _, err := wr.key.GetStringValue(ProxyServerValue)
	if err != nil {
		settings.Server = ""
	} else {
		settings.Server = server
	}

	// 读取 ProxyOverride 值
	bypass, _, err := wr.key.GetStringValue(ProxyOverrideValue)
	if err != nil {
		settings.BypassList = ""
	} else {
		settings.BypassList = bypass
	}

	return settings, nil
}

// SetSettings 设置系统代理
func (wr *WindowsRegistry) SetSettings(settings *ProxySettings) error {
	// 设置 ProxyEnable 值
	var enabled uint32
	if settings.Enabled {
		enabled = 1
	}
	err := wr.key.SetDWordValue(ProxyEnableValue, enabled)
	if err != nil {
		return pkgerrors.ErrService("failed to set ProxyEnable", err)
	}

	// 设置 ProxyServer 值
	if settings.Server != "" {
		err = wr.key.SetStringValue(ProxyServerValue, settings.Server)
		if err != nil {
			return pkgerrors.ErrService("failed to set ProxyServer", err)
		}
	}

	// 设置 ProxyOverride 值
	if settings.BypassList != "" {
		err = wr.key.SetStringValue(ProxyOverrideValue, settings.BypassList)
		if err != nil {
			return pkgerrors.ErrService("failed to set ProxyOverride", err)
		}
	}

	return nil
}

// GetStatus 获取系统代理状态
func (sp *windowsSysProxy) GetStatus() (*ProxySettings, error) {
	wr, err := NewWindowsRegistry()
	if err != nil {
		return nil, err
	}
	defer wr.Close()

	return wr.GetSettings()
}

// Enable 启用系统代理
func (sp *windowsSysProxy) Enable(server, bypassList string) error {
	wr, err := NewWindowsRegistry()
	if err != nil {
		return pkgerrors.ErrService("failed to open registry key, please check permissions", err)
	}
	defer wr.Close()

	settings := &ProxySettings{
		Enabled:    true,
		Server:     server,
		BypassList: bypassList,
	}

	if err := wr.SetSettings(settings); err != nil {
		return pkgerrors.ErrService(
			fmt.Sprintf("failed to enable system proxy: %v\n\nRecovery suggestions:\n  1. Check registry permissions\n  2. Run manually: mihomo-cli sysproxy set off\n  3. Or disable system proxy through Windows Settings", err),
			err,
		)
	}

	return nil
}

// Disable 禁用系统代理
func (sp *windowsSysProxy) Disable() error {
	wr, err := NewWindowsRegistry()
	if err != nil {
		return pkgerrors.ErrService(
			"failed to open registry key, please check permissions\n\nRecovery suggestions:\n  1. Check registry permissions\n  2. Close processes that may lock the registry\n  3. Manually disable proxy through Windows Settings\n  4. Restart computer", err)
	}
	defer wr.Close()

	settings := &ProxySettings{
		Enabled: false,
	}

	if err := wr.SetSettings(settings); err != nil {
		return pkgerrors.ErrService(
			fmt.Sprintf("failed to disable system proxy: %v\n\nRecovery suggestions:\n  1. Check registry permissions\n  2. Close processes that may lock the registry\n  3. Manually disable proxy through Windows Settings\n  4. Restart computer"), err)
	}

	return nil
}

// IsSupported 检查当前平台是否支持系统代理管理
func (sp *windowsSysProxy) IsSupported() bool {
	return true
}
