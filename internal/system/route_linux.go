//go:build linux

package system

import (
	"bytes"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
)

// listRoutes 列出 Linux 系统路由表（包括 IPv4 和 IPv6）
func (rm *RouteManager) listRoutes() ([]RouteEntry, error) {
	var allRoutes []RouteEntry

	// 获取 IPv4 路由
	cmd4 := exec.Command("ip", "route", "show")
	var stderr4 bytes.Buffer
	cmd4.Stderr = &stderr4
	output4, err := cmd4.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to execute ip route show: %w, stderr: %s", err, stderr4.String())
	}

	routes4, err := parseLinuxRouteOutput(output4, IPVersion4)
	if err != nil {
		return nil, fmt.Errorf("failed to parse IPv4 routes: %w", err)
	}
	allRoutes = append(allRoutes, routes4...)

	// 获取 IPv6 路由
	cmd6 := exec.Command("ip", "-6", "route", "show")
	var stderr6 bytes.Buffer
	cmd6.Stderr = &stderr6
	output6, err := cmd6.Output()
	if err != nil {
		// IPv6 路由获取失败不应该影响 IPv4 路由，记录警告但不返回错误
		return allRoutes, nil
	}

	routes6, err := parseLinuxRouteOutput(output6, IPVersion6)
	if err != nil {
		// IPv6 路由解析失败不应该影响 IPv4 路由，记录警告但不返回错误
		return allRoutes, nil
	}
	allRoutes = append(allRoutes, routes6...)

	return allRoutes, nil
}

// deleteRoute 删除 Linux 系统路由
func (rm *RouteManager) deleteRoute(route RouteEntry) error {
	// 根据路由 IP 版本选择合适的命令参数
	args := []string{"route", "del"}

	if route.IPVersion == IPVersion6 {
		args = []string{"-6", "route", "del"}
	}

	args = append(args, route.Destination)

	// 添加网关信息（如果有）
	if route.Gateway != "" {
		args = append(args, "via", route.Gateway)
	}

	// 添加设备信息（如果有），提高删除精确性
	if route.Interface != "" {
		args = append(args, "dev", route.Interface)
	}

	// 添加度量值（如果有），提高删除精确性
	if route.Metric > 0 {
		args = append(args, "metric", strconv.Itoa(route.Metric))
	}

	cmd := exec.Command("ip", args...)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to delete route %s: %w, stderr: %s", route.Destination, err, stderr.String())
	}
	return nil
}

// parseLinuxRouteOutput 解析 Linux ip route show 命令输出
// Linux ip route show 输出格式示例 (IPv4):
// default via 192.168.1.1 dev eth0 proto dhcp metric 100
// 192.168.1.0/24 dev eth0 proto kernel scope link src 192.168.1.100 metric 100
// 172.17.0.0/16 dev docker0 proto kernel scope link src 172.17.0.1
//
// Linux ip -6 route show 输出格式示例 (IPv6):
// default via fe80::1 dev eth0 proto ra metric 100
// 2001:db8::/64 dev eth0 proto kernel metric 256
func parseLinuxRouteOutput(output []byte, ipVersion IPVersion) ([]RouteEntry, error) {
	var routes []RouteEntry
	lines := strings.Split(string(output), "\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		route, err := parseLinuxRouteLine(line, ipVersion)
		if err != nil {
			continue // 跳过无法解析的行
		}
		if route.Destination != "" {
			routes = append(routes, route)
		}
	}

	return routes, nil
}

// parseLinuxRouteLine 解析单行 Linux 路由
func parseLinuxRouteLine(line string, ipVersion IPVersion) (RouteEntry, error) {
	var route RouteEntry
	parts := strings.Fields(line)
	if len(parts) == 0 {
		return route, fmt.Errorf("empty line")
	}

	route.IPVersion = ipVersion

	// 第一个字段是目的地址
	// 可能是 "default" 或 "192.168.1.0/24" 或 "192.168.1.0" (IPv4)
	// 或 "2001:db8::/64" (IPv6)
	if parts[0] == "default" {
		if ipVersion == IPVersion4 {
			route.Destination = "0.0.0.0/0"
		} else {
			route.Destination = "::/0"
		}
	} else {
		route.Destination = parts[0]
		// 如果没有 CIDR 后缀，根据 IP 版本添加默认前缀长度
		if !strings.Contains(route.Destination, "/") {
			if ipVersion == IPVersion4 {
				// IPv4 主机路由默认 /32
				route.Destination = route.Destination + "/32"
			} else {
				// IPv6 主机路由默认 /128
				route.Destination = route.Destination + "/128"
			}
		}
	}

	// 解析其他字段
	for i := 1; i < len(parts); i++ {
		switch parts[i] {
		case "via":
			// 下一跳网关
			if i+1 < len(parts) {
				route.Gateway = parts[i+1]
				i++
			}
		case "dev":
			// 出接口
			if i+1 < len(parts) {
				route.Interface = parts[i+1]
				i++
			}
		case "metric":
			// 路由度量值
			if i+1 < len(parts) {
				metric, err := strconv.Atoi(parts[i+1])
				if err == nil {
					route.Metric = metric
				}
				i++
			}
		}
	}

	return route, nil
}
