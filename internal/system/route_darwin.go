//go:build darwin

package system

import (
	"bytes"
	"fmt"
	"os/exec"
	"strings"
)

// listRoutes 列出 macOS 系统路由表
func (rm *RouteManager) listRoutes() ([]RouteEntry, error) {
	cmd := exec.Command("netstat", "-rn")
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to execute netstat -rn: %w, stderr: %s", err, stderr.String())
	}
	return parseDarwinRouteOutput(output)
}

// deleteRoute 删除 macOS 系统路由
func (rm *RouteManager) deleteRoute(route RouteEntry) error {
	// route -n delete 目标 [网关]
	args := []string{"-n", "delete", route.Destination}
	if route.Gateway != "" {
		args = append(args, route.Gateway)
	}
	cmd := exec.Command("route", args...)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to delete route %s: %w, stderr: %s", route.Destination, err, stderr.String())
	}
	return nil
}

// addRoute 添加 macOS 系统路由
func (rm *RouteManager) addRoute(route RouteEntry) error {
	// route -n add 目标 网关 [netmask] [metric]
	args := []string{"-n", "add", route.Destination}

	// 添加网关
	if route.Gateway != "" {
		args = append(args, route.Gateway)
	}

	// 对于 IPv4 路由，添加子网掩码
	if route.IPVersion == IPVersion4 && route.Netmask != "" {
		args = append(args, route.Netmask)
	}

	// 添加度量值（如果有）
	if route.Metric > 0 {
		args = append(args, strconv.Itoa(route.Metric))
	}

	cmd := exec.Command("route", args...)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to add route %s: %w, stderr: %s", route.Destination, err, stderr.String())
	}
	return nil
}

// parseDarwinRouteOutput 解析 macOS netstat -rn 命令输出
// macOS netstat -rn 输出格式示例:
// Routing tables
// Internet:
// Destination        Gateway            Flags        Netif Expire
// default            192.168.1.1        UGSc           en0
// 127                127.0.0.1          UCS            lo0
// 127.0.0.1          127.0.0.1          UH             lo0
// 192.168.1          link#1             UCS            en0
// 192.168.1.1        0:1a:2b:3c:4d:5e   UHLWIir        en0   1197
// 192.168.1.100      127.0.0.1          UHS            lo0
//
// Internet6:
// Destination                             Gateway                         Flags        Netif Expire
// default                                 fe80::1%en0                     UGSc          en0
// ::1                                     ::1                             UH            lo0
// fe80::/10                               fe80::1%en0                     UGc           en0
func parseDarwinRouteOutput(output []byte) ([]RouteEntry, error) {
	var routes []RouteEntry
	lines := strings.Split(string(output), "\n")

	var inInternet, inInternet6 bool
	for _, line := range lines {
		line = strings.TrimSpace(line)

		// 检测 Internet (IPv4) 段落开始
		if line == "Internet:" {
			inInternet = true
			inInternet6 = false
			continue
		}

		// 检测 Internet6 (IPv6) 段落开始
		if line == "Internet6:" {
			inInternet6 = true
			inInternet = false
			continue
		}

		// 检测其他段落
		if strings.HasSuffix(line, ":") && line != "Internet:" && line != "Internet6:" {
			inInternet = false
			inInternet6 = false
			continue
		}

		if !inInternet && !inInternet6 {
			continue
		}

		// 跳过空行和标题行
		if line == "" || strings.HasPrefix(line, "Destination") {
			continue
		}

		route, err := parseDarwinRouteLine(line)
		if err != nil {
			continue // 跳过无法解析的行
		}
		if route.Destination != "" {
			routes = append(routes, route)
		}
	}

	return routes, nil
}

// parseDarwinRouteLine 解析单行 macOS 路由
func parseDarwinRouteLine(line string) (RouteEntry, error) {
	var route RouteEntry
	fields := strings.Fields(line)
	if len(fields) < 4 {
		return route, fmt.Errorf("insufficient fields")
	}

	// 字段顺序: Destination Gateway Flags Netif [Expire]
	destination := fields[0]
	gateway := fields[1]
	flags := fields[2]
	iface := fields[3]

	// 处理目的地址
	if destination == "default" {
		route.Destination = "0.0.0.0/0"
		route.IPVersion = IPVersion4
	} else {
		route.Destination = destination
		// 检查是否是 IPv6 地址
		if strings.Contains(destination, ":") {
			route.IPVersion = IPVersion6
			// IPv6 地址通常已经有 CIDR 后缀，如果没有则添加 /128
			if !strings.Contains(destination, "/") {
				route.Destination = destination + "/128"
			}
		} else {
			route.IPVersion = IPVersion4
			// IPv4 地址处理
			if !strings.Contains(destination, "/") {
				// 没有 CIDR 后缀，检查是否是网络前缀
				if strings.Contains(destination, ".") {
					// 包含点，可能是网络前缀如 "192.168.1"
					dotCount := strings.Count(destination, ".")
					if dotCount == 2 {
						// 格式如 192.168.1，推断为 /24
						route.Destination = destination + ".0/24"
					} else if dotCount == 1 {
						// 格式如 192.168，推断为 /16
						route.Destination = destination + ".0.0/16"
					} else if dotCount == 0 && destination != "127" {
						// 单个数字，可能是 A 类网络
						route.Destination = destination + ".0.0.0/8"
					} else if destination == "127" {
						route.Destination = "127.0.0.0/8"
					} else {
						// 完整 IP 地址，默认是主机路由
						route.Destination = destination + "/32"
					}
				} else {
					// 不包含点，可能是特殊格式，默认是主机路由
					route.Destination = destination + "/32"
				}
			}
		}
	}

	// 处理网关
	// link#x 表示直连路由，没有网关
	if strings.HasPrefix(gateway, "link#") {
		route.Gateway = ""
	} else {
		route.Gateway = gateway
	}

	route.Interface = iface
	route.Flags = flags

	// 从 flags 中提取度量值（如果有）
	// macOS 的 flags 不直接包含 metric，这里设为 0
	route.Metric = 0

	// 如果有 Expire 字段，可以尝试解析（可选）
	if len(fields) > 4 {
		// Expire 字段，暂时忽略
		_ = fields[4]
	}

	// 跳过本地回环路由的特殊处理
	if flags == "UH" && strings.HasPrefix(route.Destination, "127.") {
		// 这是主机路由，保持原样
	}

	return route, nil
}

// parseMetricFromFlags 尝试从 flags 或其他信息推断 metric（可选）
func parseMetricFromFlags(flags string) int {
	// macOS 的路由标志不直接包含 metric
	// 可以根据需要扩展此函数
	return 0
}

// checkInterfaceExists 检查 macOS 接口是否存在
func checkInterfaceExists(iface string) bool {
	if iface == "" {
		return false
	}

	// 使用 ifconfig 命令检查接口
	cmd := exec.Command("ifconfig", iface)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	err := cmd.Run()
	return err == nil
}

// checkGatewayReachable 检查网关是否可达
func checkGatewayReachable(gateway string) bool {
	if gateway == "" {
		return true // 空网关不需要检查（直连路由）
	}

	// 使用 ping 命令检测网关是否可达（仅发送 1 个包，超时 1 秒）
	cmd := exec.Command("ping", "-c", "1", "-W", "1000", gateway)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	err := cmd.Run()
	return err == nil
}

// getInterfaceInfo 获取 macOS 接口详细信息
func getInterfaceInfo(iface string) (map[string]string, error) {
	if iface == "" {
		return nil, fmt.Errorf("interface name is empty")
	}

	cmd := exec.Command("ifconfig", iface)
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
		if strings.Contains(line, "status") {
			parts := strings.Fields(line)
			for i, part := range parts {
				if part == "status" && i+1 < len(parts) {
					info["status"] = parts[i+1]
					break
				}
			}
		}
	}

	return info, nil
}

// getActiveInterfaceList 获取 macOS 活动接口列表
func getActiveInterfaceList() ([]string, error) {
	cmd := exec.Command("ifconfig", "-a")
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
		// 检测接口名称行（格式如：en0: flags=8863<UP,BROADCAST,SMART,RUNNING,SIMPLEX,MULTICAST>）
		if strings.Contains(line, ":") && strings.Contains(line, "UP") {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) > 0 {
				ifaceName := strings.TrimSpace(parts[0])
				interfaces = append(interfaces, ifaceName)
			}
		}
	}

	return interfaces, nil
}

// macOS 平台特定实现
func checkInterfaceExistsImpl(iface string) bool {
	return checkInterfaceExists(iface)
}

func checkGatewayReachableImpl(gateway string) bool {
	return checkGatewayReachable(gateway)
}

func getInterfaceInfoImpl(iface string) (map[string]string, error) {
	return getInterfaceInfo(iface)
}

func getActiveInterfaceListImpl() ([]string, error) {
	return getActiveInterfaceList()
}
