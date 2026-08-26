package systemdns

import "strings"

const (
	// ResolvConf resolv.conf 路径（Linux 主用，macOS 作为 fallback）
	ResolvConf = "/etc/resolv.conf"
)

// parseResolvConf 解析 resolv.conf 文本，返回搜索域、DNS 服务器与解析器选项。
// 供 Linux（/etc/resolv.conf）与 macOS（scutil --dns 失败时 fallback）复用。
func parseResolvConf(content string) (searchDomains, nameservers, options []string) {
	seenNS := make(map[string]bool)

	for _, rawLine := range strings.Split(content, "\n") {
		line := strings.TrimSpace(rawLine)
		// 跳过空行与注释
		if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, ";") {
			continue
		}

		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}

		switch fields[0] {
		case "nameserver":
			for _, ns := range fields[1:] {
				if !seenNS[ns] {
					seenNS[ns] = true
					nameservers = append(nameservers, ns)
				}
			}
		case "search":
			searchDomains = append(searchDomains, fields[1:]...)
		case "domain":
			// domain 与 search 语义等价，仅在无 search 指令时使用
			if len(searchDomains) == 0 {
				searchDomains = append(searchDomains, fields[1:]...)
			}
		case "options":
			options = append(options, fields[1:]...)
		}
	}

	return searchDomains, nameservers, options
}
