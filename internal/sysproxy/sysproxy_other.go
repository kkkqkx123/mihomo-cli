//go:build !windows && !linux && !darwin

package sysproxy

import (
	"fmt"
	"runtime"
)

// unsupportedSysProxy is an implementation for unsupported platforms
type unsupportedSysProxy struct {
	platform string
}

// newPlatformSysProxy creates a stub implementation for unsupported platforms
func newPlatformSysProxy() SysProxy {
	return &unsupportedSysProxy{
		platform: runtime.GOOS,
	}
}

// GetStatus returns an error indicating the platform is not supported
func (sp *unsupportedSysProxy) GetStatus() (*ProxySettings, error) {
	return nil, fmt.Errorf("system proxy management is not supported on %s: this feature is only available on Windows, Linux, and macOS", sp.platform)
}

// Enable returns an error indicating the platform is not supported
func (sp *unsupportedSysProxy) Enable(server, bypassList string) error {
	return fmt.Errorf("system proxy management is not supported on %s: this feature is only available on Windows, Linux, and macOS", sp.platform)
}

// Disable returns an error indicating the platform is not supported
func (sp *unsupportedSysProxy) Disable() error {
	return fmt.Errorf("system proxy management is not supported on %s: this feature is only available on Windows, Linux, and macOS", sp.platform)
}

// IsSupported returns false for unsupported platforms
func (sp *unsupportedSysProxy) IsSupported() bool {
	return false
}
