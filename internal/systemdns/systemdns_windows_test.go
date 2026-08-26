//go:build windows

package systemdns

import (
	"reflect"
	"testing"
)

func TestParseNetshDNSOutputEnglish(t *testing.T) {
	output := `Configuration for interface "Ethernet"
    DNS servers configured through DHCP:
        DNS Server = 192.168.1.1
    Register with which suffix: Primary only

Configuration for interface "Wi-Fi"
    DNS servers configured manually:
        DNS Server = 8.8.8.8
        DNS Server = 192.168.1.1
`

	config, err := parseNetshDNSOutput(output)
	if err != nil {
		t.Fatalf("parseNetshDNSOutput failed: %v", err)
	}

	if !reflect.DeepEqual(config.Nameservers, []string{"192.168.1.1", "8.8.8.8"}) {
		t.Errorf("Expected nameservers [192.168.1.1 8.8.8.8], got %v", config.Nameservers)
	}
}

func TestParseNetshDNSOutputChinese(t *testing.T) {
	output := `配置 "以太网" 的 DNS 服务器:
    静态配置的 DNS 服务器:
        DNS 服务器 = 10.0.0.1
    通过 DHCP 配置的 DNS 服务器:
        DNS 服务器 = 10.0.0.2
`

	config, err := parseNetshDNSOutput(output)
	if err != nil {
		t.Fatalf("parseNetshDNSOutput failed: %v", err)
	}

	if !reflect.DeepEqual(config.Nameservers, []string{"10.0.0.1", "10.0.0.2"}) {
		t.Errorf("Expected nameservers [10.0.0.1 10.0.0.2], got %v", config.Nameservers)
	}
}

func TestParseNetshDNSOutputNoServer(t *testing.T) {
	output := `Configuration for interface "Ethernet"
    Register with which suffix: Primary only
`

	_, err := parseNetshDNSOutput(output)
	if err == nil {
		t.Fatal("Expected error for empty DNS server list, got nil")
	}
}
