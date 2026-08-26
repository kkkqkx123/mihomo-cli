//go:build !windows && !linux && !darwin

package systemdns

import (
	"fmt"
	"runtime"
)

// unsupportedSystemDNS is an implementation for unsupported platforms
type unsupportedSystemDNS struct {
	platform string
}

// newPlatformSystemDNS creates a stub implementation for unsupported platforms
func newPlatformSystemDNS() SystemDNS {
	return &unsupportedSystemDNS{
		platform: runtime.GOOS,
	}
}

// GetConfig returns an error indicating the platform is not supported
func (sd *unsupportedSystemDNS) GetConfig() (*SystemDNSConfig, error) {
	return nil, fmt.Errorf("system DNS config query is not supported on %s: this feature is only available on Windows, Linux, and macOS", sd.platform)
}

// IsSupported returns false for unsupported platforms
func (sd *unsupportedSystemDNS) IsSupported() bool {
	return false
}
