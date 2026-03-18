//go:build !windows

package sysproxy

// newWindowsSysProxy is a stub for non-Windows platforms.
func newWindowsSysProxy() SysProxy {
	return &stubSysProxy{}
}
