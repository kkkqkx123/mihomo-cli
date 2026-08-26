package systemdns

// NewSystemDNS creates a system DNS config provider for the current platform.
func NewSystemDNS() SystemDNS {
	return newPlatformSystemDNS()
}
