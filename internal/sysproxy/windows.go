//go:build windows

package sysproxy

import (
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

// SystemProxySettings 系统代理设置
type SystemProxySettings struct {
	Enabled    bool   // 是否启用代理
	Server     string // 代理服务器地址
	BypassList string // 绕过代理的地址列表
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
func (wr *WindowsRegistry) GetSettings() (*SystemProxySettings, error) {
	settings := &SystemProxySettings{}

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
func (wr *WindowsRegistry) SetSettings(settings *SystemProxySettings) error {
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

// EnableProxy 启用系统代理
func EnableProxy(server, bypassList string) error {
	wr, err := NewWindowsRegistry()
	if err != nil {
		return err
	}
	defer wr.Close()

	settings := &SystemProxySettings{
		Enabled:    true,
		Server:     server,
		BypassList: bypassList,
	}

	return wr.SetSettings(settings)
}

// DisableProxy 禁用系统代理
func DisableProxy() error {
	wr, err := NewWindowsRegistry()
	if err != nil {
		return err
	}
	defer wr.Close()

	settings := &SystemProxySettings{
		Enabled: false,
	}

	return wr.SetSettings(settings)
}

// GetProxyStatus 获取系统代理状态
func GetProxyStatus() (*SystemProxySettings, error) {
	wr, err := NewWindowsRegistry()
	if err != nil {
		return nil, err
	}
	defer wr.Close()

	return wr.GetSettings()
}
