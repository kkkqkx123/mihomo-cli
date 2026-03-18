//go:build !linux

package sysproxy

// newLinuxSysProxy is a stub for non-Linux platforms.
func newLinuxSysProxy() SysProxy {
	return &stubSysProxy{}
}
