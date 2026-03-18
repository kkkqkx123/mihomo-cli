package sysproxy

import "runtime"

// NewSysProxy 创建系统代理管理器
func NewSysProxy() SysProxy {
	switch runtime.GOOS {
	case "windows":
		return newWindowsSysProxy()
	case "linux":
		return newLinuxSysProxy()
	default:
		return &stubSysProxy{}
	}
}

// stubSysProxy is a stub implementation for unsupported platforms.
type stubSysProxy struct{}

func (sp *stubSysProxy) GetStatus() (*ProxySettings, error) {
	return nil, ErrPlatformNotSupported
}

func (sp *stubSysProxy) Enable(server, bypassList string) error {
	return ErrPlatformNotSupported
}

func (sp *stubSysProxy) Disable() error {
	return ErrPlatformNotSupported
}

func (sp *stubSysProxy) IsSupported() bool {
	return false
}
