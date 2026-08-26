//go:build linux

package systemdns

import (
	"os"

	pkgerrors "github.com/kkkqkx123/mihomo-cli/pkg/errors"
)

// linuxSystemDNS Linux system DNS provider
type linuxSystemDNS struct{}

// newPlatformSystemDNS creates a new Linux system DNS provider
func newPlatformSystemDNS() SystemDNS {
	return &linuxSystemDNS{}
}

// GetConfig reads and parses /etc/resolv.conf
func (sd *linuxSystemDNS) GetConfig() (*SystemDNSConfig, error) {
	data, err := os.ReadFile(ResolvConf)
	if err != nil {
		return nil, pkgerrors.ErrService("failed to read resolv.conf", err)
	}

	searchDomains, nameservers, options := parseResolvConf(string(data))

	return &SystemDNSConfig{
		SearchDomains: searchDomains,
		Nameservers:   nameservers,
		Options:       options,
		Source:        ResolvConf,
	}, nil
}

// IsSupported returns true on Linux
func (sd *linuxSystemDNS) IsSupported() bool {
	return true
}
