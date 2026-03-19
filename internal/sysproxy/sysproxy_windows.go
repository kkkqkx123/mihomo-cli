//go:build windows

package sysproxy

import (
	"fmt"
	"syscall"
	"time"

	"golang.org/x/sys/windows/registry"

	pkgerrors "github.com/kkkqkx123/mihomo-cli/pkg/errors"
)

const (
	// InternetSetOption constants
	INTERNET_OPTION_SETTINGS_CHANGED = 39
	INTERNET_OPTION_REFRESH          = 37
)

var (
	// wininet.dll dynamic library and functions
	wininet              = syscall.NewLazyDLL("wininet.dll")
	procInternetSetOption = wininet.NewProc("InternetSetOptionW")
)

const (
	// InternetSettingsKey Internet Settings registry key path
	InternetSettingsKey = `SOFTWARE\Microsoft\Windows\CurrentVersion\Internet Settings`

	// ProxyEnableValue proxy enable flag
	ProxyEnableValue = "ProxyEnable"
	// ProxyServerValue proxy server address
	ProxyServerValue = "ProxyServer"
	// ProxyOverrideValue proxy bypass list
	ProxyOverrideValue = "ProxyOverride"
)

// windowsSysProxy Windows system proxy manager
type windowsSysProxy struct{}

// newPlatformSysProxy creates a new Windows system proxy manager
func newPlatformSysProxy() SysProxy {
	return &windowsSysProxy{}
}

// refreshProxy notifies the system to refresh proxy settings
// This function is mainly used to notify long-running legacy applications or specific system components to refresh proxy settings
// For modern applications (like Chrome, Terminal), registry modifications usually take effect immediately
func refreshProxy() error {
	// Notify settings changed
	ret, _, _ := procInternetSetOption.Call(
		0,
		uintptr(INTERNET_OPTION_SETTINGS_CHANGED),
		0,
		0,
	)
	if ret == 0 {
		// Even if it fails, don't return an error, as this doesn't affect the main functionality
		// This is a compatibility enhancement, not required
		return nil
	}

	// Refresh settings
	ret, _, _ = procInternetSetOption.Call(
		0,
		uintptr(INTERNET_OPTION_REFRESH),
		0,
		0,
	)
	if ret == 0 {
		return nil
	}

	return nil
}

// WindowsRegistry Windows registry operations
type WindowsRegistry struct {
	key registry.Key
}

// NewWindowsRegistry creates a new registry operation instance
func NewWindowsRegistry() (*WindowsRegistry, error) {
	// Open the Internet Settings key for the current user
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

// Close closes the registry key
func (wr *WindowsRegistry) Close() error {
	return wr.key.Close()
}

// GetSettings gets the current system proxy settings
func (wr *WindowsRegistry) GetSettings() (*ProxySettings, error) {
	settings := &ProxySettings{}

	// Read ProxyEnable value
	enabled, _, err := wr.key.GetIntegerValue(ProxyEnableValue)
	if err != nil {
		// If value doesn't exist, default to disabled
		settings.Enabled = false
	} else {
		settings.Enabled = enabled != 0
	}

	// Read ProxyServer value
	server, _, err := wr.key.GetStringValue(ProxyServerValue)
	if err != nil {
		settings.Server = ""
	} else {
		settings.Server = server
	}

	// Read ProxyOverride value
	bypass, _, err := wr.key.GetStringValue(ProxyOverrideValue)
	if err != nil {
		settings.BypassList = ""
	} else {
		settings.BypassList = bypass
	}

	return settings, nil
}

// SetSettings sets the system proxy
func (wr *WindowsRegistry) SetSettings(settings *ProxySettings) error {
	// Set ProxyEnable value
	var enabled uint32
	if settings.Enabled {
		enabled = 1
	}
	err := wr.key.SetDWordValue(ProxyEnableValue, enabled)
	if err != nil {
		return pkgerrors.ErrService("failed to set ProxyEnable", err)
	}

	// Set ProxyServer value
	if settings.Server != "" {
		err = wr.key.SetStringValue(ProxyServerValue, settings.Server)
		if err != nil {
			return pkgerrors.ErrService("failed to set ProxyServer", err)
		}
	}

	// Set ProxyOverride value
	if settings.BypassList != "" {
		err = wr.key.SetStringValue(ProxyOverrideValue, settings.BypassList)
		if err != nil {
			return pkgerrors.ErrService("failed to set ProxyOverride", err)
		}
	}

	return nil
}

// GetStatus gets the system proxy status
func (sp *windowsSysProxy) GetStatus() (*ProxySettings, error) {
	wr, err := NewWindowsRegistry()
	if err != nil {
		return nil, err
	}
	defer wr.Close()

	return wr.GetSettings()
}

// Enable enables the system proxy
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

	// Notify system to refresh proxy settings (compatibility enhancement for legacy applications)
	_ = refreshProxy()

	return nil
}

// Disable disables the system proxy
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
			fmt.Sprintf("failed to disable system proxy: %v\n\nRecovery suggestions:\n  1. Check registry permissions\n  2. Close processes that may lock the registry\n  3. Manually disable proxy through Windows Settings\n  4. Restart computer", err), err)
	}

	// Notify system to refresh proxy settings (compatibility enhancement for legacy applications)
	_ = refreshProxy()

	return nil
}

// IsSupported checks if the current platform supports system proxy management
func (sp *windowsSysProxy) IsSupported() bool {
	return true
}

// BackupRegistrySettings 备份当前注册表设置
func (sp *windowsSysProxy) BackupRegistrySettings(note string) (*RegistryBackup, error) {
	wr, err := NewWindowsRegistry()
	if err != nil {
		return nil, pkgerrors.ErrService("failed to open registry key", err)
	}
	defer wr.Close()

	settings, err := wr.GetSettings()
	if err != nil {
		return nil, pkgerrors.ErrService("failed to get registry settings", err)
	}

	// 生成备份 ID
	id := time.Now().Format("20060102-150405")

	backup := &RegistryBackup{
		ID:        id,
		Timestamp: time.Now(),
		Settings:  settings,
		Note:      note,
	}

	return backup, nil
}

// RestoreRegistrySettings 恢复注册表设置
func (sp *windowsSysProxy) RestoreRegistrySettings(settings *ProxySettings) error {
	wr, err := NewWindowsRegistry()
	if err != nil {
		return pkgerrors.ErrService("failed to open registry key", err)
	}
	defer wr.Close()

	if err := wr.SetSettings(settings); err != nil {
		return pkgerrors.ErrService("failed to restore registry settings", err)
	}

	// 通知系统刷新代理设置
	_ = refreshProxy()

	return nil
}

// RegistryBackup 注册表备份
type RegistryBackup struct {
	ID        string        `json:"id"`
	Timestamp time.Time     `json:"timestamp"`
	Settings  *ProxySettings `json:"settings"`
	Note      string        `json:"note,omitempty"`
}
