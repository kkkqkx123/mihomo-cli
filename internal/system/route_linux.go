//go:build linux

package system

import (
	"bytes"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
)

// listRoutes 列出 Linux 系统路由表
func (rm *RouteManager) listRoutes() ([]RouteEntry, error) {
	cmd := exec.Command("ip", "route", "show")
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to execute ip route show: %w, stderr: %s", err, stderr.String())
	}
	return parseLinuxRouteOutput(output)
}

// deleteRoute 删除 Linux 系统路由
func (rm *RouteManager) deleteRoute(route RouteEntry) error {
	args := []string{"route", "del", route.Destination}
	if route.Gateway != "" {
		args = append(args, "via", route.Gateway)
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
// Linux ip route show 输出格式示例:
// default via 192.168.1.1 dev eth0 proto dhcp metric 100
// 192.168.1.0/24 dev eth0 proto kernel scope link src 192.168.1.100 metric 100
// 172.17.0.0/16 dev docker0 proto kernel scope link src 172.17.0.1
func parseLinuxRouteOutput(output []byte) ([]RouteEntry, error) {
	var routes []RouteEntry
	lines := strings.Split(string(output), "\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		route, err := parseLinuxRouteLine(line)
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
func parseLinuxRouteLine(line string) (RouteEntry, error) {
	var route RouteEntry
	parts := strings.Fields(line)
	if len(parts) == 0 {
		return route, fmt.Errorf("empty line")
	}

	// 第一个字段是目的地址
	// 可能是 "default" 或 "192.168.1.0/24" 或 "192.168.1.0"
	if parts[0] == "default" {
		route.Destination = "0.0.0.0/0"
	} else {
		route.Destination = parts[0]
		// 如果没有 CIDR 后缀，默认是 /32（主机路由）
		if !strings.Contains(route.Destination, "/") {
			route.Destination = route.Destination + "/32"
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
