//go:build darwin

package systemdns

import (
	"reflect"
	"testing"
)

func TestParseScutilDNSOutput(t *testing.T) {
	output := `DNS configuration

resolver #1
  search domain[0] : example.com
  nameserver[0] : 8.8.8.8
  nameserver[1] : 8.8.4.4
  flags    : Request A records, Request AAAA records
  reach    : 0x00020002 (Reachable,Directly Reachable Address)

resolver #2
  domain   : local
  nameserver[0] : 127.0.0.1
  nameserver[1] : 8.8.8.8
  flags    : Request A records, Request AAAA records
`

	config, err := parseScutilDNSOutput(output)
	if err != nil {
		t.Fatalf("parseScutilDNSOutput failed: %v", err)
	}

	// 跨 resolver 去重且保持出现顺序
	if !reflect.DeepEqual(config.Nameservers, []string{"8.8.8.8", "8.8.4.4", "127.0.0.1"}) {
		t.Errorf("Expected nameservers [8.8.8.8 8.8.4.4 127.0.0.1], got %v", config.Nameservers)
	}

	if !reflect.DeepEqual(config.SearchDomains, []string{"example.com"}) {
		t.Errorf("Expected search domains [example.com], got %v", config.SearchDomains)
	}
}

func TestParseScutilDNSOutputNoNameserver(t *testing.T) {
	output := "DNS configuration\n\nresolver #1\n  flags    : Request A records\n"

	_, err := parseScutilDNSOutput(output)
	if err == nil {
		t.Fatal("Expected error for empty nameserver list, got nil")
	}
}
