//go:build !windows

package service

// newWindowsServiceManager is a stub for non-Windows platforms.
func newWindowsServiceManager(serviceName, displayName, description, exePath string) ServiceManager {
	return &stubServiceManager{
		serviceName: serviceName,
		displayName: displayName,
		exePath:     exePath,
	}
}
