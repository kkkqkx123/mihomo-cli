//go:build windows

package systemdns

import (
	"context"
	"os/exec"
	"strings"
	"time"

	pkgerrors "github.com/kkkqkx123/mihomo-cli/pkg/errors"
)

const (
	// NetshDNS 输出来源描述
	NetshDNS = "netsh interface ip show dns"
	// NetshTimeout netsh 命令执行超时
	NetshTimeout = 5 * time.Second
)

// windowsSystemDNS Windows system DNS provider
type windowsSystemDNS struct{}

// newPlatformSystemDNS creates a new Windows system DNS provider
func newPlatformSystemDNS() SystemDNS {
	return &windowsSystemDNS{}
}

// GetConfig queries DNS servers via `netsh interface ip show dns`.
func (sd *windowsSystemDNS) GetConfig() (*SystemDNSConfig, error) {
	ctx, cancel := context.WithTimeout(context.Background(), NetshTimeout)
	defer cancel()

	output, err := exec.CommandContext(ctx, "netsh", "interface", "ip", "show", "dns").Output()
	if err != nil {
		return nil, pkgerrors.ErrService("failed to run netsh interface ip show dns", err)
	}

	config, err := parseNetshDNSOutput(string(output))
	if err != nil {
		return nil, err
	}
	config.Source = NetshDNS
	return config, nil
}

// parseNetshDNSOutput 解析 `netsh interface ip show dns` 输出。
// 中英文系统输出格式不同，统一按行内包含 DNS 关键字与 "=" 取等号右侧 IP：
//
//	DNS Server = 8.8.8.8
//	DNS 服务器 = 10.0.0.1
func parseNetshDNSOutput(output string) (*SystemDNSConfig, error) {
	config := &SystemDNSConfig{}
	seenNS := make(map[string]bool)

	for _, line := range strings.Split(output, "\n") {
		line = strings.TrimSpace(line)
		if !strings.Contains(line, "DNS") || !strings.Contains(line, "=") {
			continue
		}

		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}

		ns := strings.TrimSpace(parts[1])
		if ns == "" {
			continue
		}

		if !seenNS[ns] {
			seenNS[ns] = true
			config.Nameservers = append(config.Nameservers, ns)
		}
	}

	if len(config.Nameservers) == 0 {
		return nil, pkgerrors.ErrService("no DNS server found in netsh output", nil)
	}

	return config, nil
}

// IsSupported returns true on Windows
func (sd *windowsSystemDNS) IsSupported() bool {
	return true
}
