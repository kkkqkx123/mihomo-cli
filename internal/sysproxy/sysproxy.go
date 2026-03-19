package sysproxy

// NewSysProxy creates a system proxy manager for the current platform.
func NewSysProxy() SysProxy {
	return newPlatformSysProxy()
}
