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

// addRoute 添加 Linux 系统路由
func (rm *RouteManager) addRoute(route RouteEntry) error {
	// 根据路由 IP 版本选择合适的命令参数
	args := []string{"route", "add"}

	if route.IPVersion == IPVersion6 {
		args = []string{"-6", "route", "add"}
	}

	args = append(args, route.Destination)

	// 添加网关信息（如果有）
	if route.Gateway != "" {
		args = append(args, "via", route.Gateway)
	}

	// 添加设备信息（如果有）
	if route.Interface != "" {
		args = append(args, "dev", route.Interface)
	}

	// 添加度量值（如果有）
	if route.Metric > 0 {
		args = append(args, "metric", strconv.Itoa(route.Metric))
	}

	cmd := exec.Command("ip", args...)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to add route %s: %w, stderr: %s", route.Destination, err, stderr.String())
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

// Linux 平台特定实现
func checkInterfaceExistsImpl(iface string) bool {
	if iface == "" {
		return false
	}

	// 使用 ip link show 命令检查接口
	cmd := exec.Command("ip", "link", "show", iface)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	err := cmd.Run()
	return err == nil
}

func checkGatewayReachableImpl(gateway string) bool {
	if gateway == "" {
		return true // 空网关不需要检查（直连路由）
	}

	// 使用 ping 命令检测网关是否可达（仅发送 1 个包，超时 1 秒）
	cmd := exec.Command("ping", "-c", "1", "-W", "1", gateway)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	err := cmd.Run()
	return err == nil
}

// checkMihomoRouteFlagsImpl 检查 Linux 路由标志是否表明是 Mihomo 添加的路由
// Linux 使用 ip route 命令,路由标志与 BSD/macOS 不同
// Linux 路由属性包括:
// - proto: 路由协议(kernel, boot, static, etc.)
// - scope: 路由范围(link, host, global)
// - metric: 路由度量值
// - dev: 接口设备
func checkMihomoRouteFlagsImpl(_ string) bool {
	// Linux 的路由标志格式与 BSD/macOS 不同
	// Linux ip route 输出中,flags 字段通常包含协议和范围信息
	// 对于 Linux,我们主要依赖接口名称和网关地址来判断
	// 这里保留接口,但返回 false,因为 Linux 不使用 BSD 风格的标志
	return false
}

// GetInterfaceInfo 获取 Linux 接口详细信息
func (rm *RouteManager) GetInterfaceInfo(iface string) (map[string]string, error) {
	if iface == "" {
		return nil, fmt.Errorf("interface name is empty")
	}

	cmd := exec.Command("ip", "addr", "show", iface)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to get interface info: %w", err)
	}

	info := make(map[string]string)
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.Contains(line, ":") {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) == 2 {
				key := strings.TrimSpace(parts[0])
				value := strings.TrimSpace(parts[1])
				info[key] = value
			}
		}
	}

	// 解析状态信息
	for _, line := range lines {
		if strings.Contains(line, "state") {
			parts := strings.Fields(line)
			for i, part := range parts {
				if part == "state" && i+1 < len(parts) {
					info["state"] = parts[i+1]
					break
				}
			}
		}
	}

	return info, nil
}

// GetActiveInterfaceList 获取 Linux 活动接口列表
func (rm *RouteManager) GetActiveInterfaceList() ([]string, error) {
	cmd := exec.Command("ip", "link", "show")
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to get interface list: %w", err)
	}

	var interfaces []string
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.Contains(line, "state UP") || strings.Contains(line, "state") {
			// 提取接口名称（格式如：2: eth0: <BROADCAST,MULTICAST,UP,LOWER_UP>）
			fields := strings.Fields(line)
			if len(fields) > 1 {
				ifaceName := strings.TrimSuffix(fields[1], ":")
				ifaceName = strings.TrimPrefix(ifaceName, "@") // 处理别名接口
				interfaces = append(interfaces, ifaceName)
			}
		}
	}

	return interfaces, nil
}
