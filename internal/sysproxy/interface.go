package sysproxy

import "errors"

// ErrPlatformNotSupported indicates that system proxy management is not supported.
var ErrPlatformNotSupported = errors.New("system proxy not supported on this platform")

// ProxySettings holds the configuration for system proxies.
type ProxySettings struct {
	Enabled    bool
	Server     string
	BypassList string
}

// SysProxy defines the interface for managing system proxies.
type SysProxy interface {
	// Get Proxy Status
	GetStatus() (*ProxySettings, error)

	// Configure Proxy
	Enable(server, bypassList string) error
	Disable() error

	// Platform Detection
	IsSupported() bool
}
