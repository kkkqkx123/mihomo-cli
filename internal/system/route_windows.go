//go:build windows

package system

import (
	"bytes"
	"fmt"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
)

// listRoutes 列出 Windows 系统路由表
func (rm *RouteManager) listRoutes() ([]RouteEntry, error) {
	cmd := exec.Command("route", "print")
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to execute route print: %w, stderr: %s", err, stderr.String())
	}
	return parseWindowsRouteOutput(output)
}

// deleteRoute 删除 Windows 系统路由
func (rm *RouteManager) deleteRoute(route RouteEntry) error {
	// route delete 目标
	args := []string{"delete", route.Destination}

	// 对于 IPv4 路由，添加 mask 参数以提高精确性
	if route.IPVersion == IPVersion4 && route.Netmask != "" {
		args = append(args, "mask", route.Netmask)
	}

	// 添加网关信息（如果有），提高删除精确性
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

// parseWindowsRouteOutput 解析 Windows route print 命令输出
// Windows route print 输出格式示例 (IPv4):
// ===========================================================================
// Active Routes:
// Network Destination        Netmask          Gateway       Interface  Metric
//           0.0.0.0          0.0.0.0      192.168.1.1    192.168.1.100     25
//         127.0.0.0        255.0.0.0         On-link         127.0.0.1    331
// ===========================================================================
//
// Windows route print 输出格式示例 (IPv6):
// ===========================================================================
// Active Routes:
//  If Metric Network Destination      Gateway
//  1    331  ::1/128                  On-link
//  1     25  ::/0                     fe80::1
// ===========================================================================
func parseWindowsRouteOutput(output []byte) ([]RouteEntry, error) {
	var routes []RouteEntry
	lines := strings.Split(string(output), "\n")

	// IPv4 正则匹配路由行
	// 格式: Network Destination    Netmask    Gateway    Interface    Metric
	ipv4RoutePattern := regexp.MustCompile(`^\s*(\S+)\s+(\S+)\s+(\S+)\s+(\S+)\s+(\d+)\s*$`)

	// IPv6 正则匹配路由行
	// 格式: If Metric Network Destination      Gateway
	ipv6RoutePattern := regexp.MustCompile(`^\s*(\d+)\s+(\d+)\s+(\S+)\s+(\S+)\s*$`)

	inActiveRoutes := false
	for _, line := range lines {
		line = strings.TrimSpace(line)

		// 检测 Active Routes 段落开始
		if strings.Contains(line, "Active Routes:") {
			inActiveRoutes = true
			continue
		}

		// 检测段落结束
		if strings.HasPrefix(line, "==========") && inActiveRoutes {
			inActiveRoutes = false
			continue
		}

		if !inActiveRoutes {
			continue
		}

		// 跳过空行和标题行
		if line == "" || strings.Contains(line, "Network Destination") {
			continue
		}

		// 尝试匹配 IPv4 路由
		ipv4Matches := ipv4RoutePattern.FindStringSubmatch(line)
		if ipv4Matches != nil {
			destination := ipv4Matches[1]
			netmask := ipv4Matches[2]
			gateway := ipv4Matches[3]
			iface := ipv4Matches[4]
			metric, _ := strconv.Atoi(ipv4Matches[5])

			// 跳过 On-link 路由（本地路由）
			if gateway == "On-link" {
				gateway = ""
			}

			// 构建完整的目的地址（包含子网掩码）
			if netmask != "255.255.255.255" && netmask != "0.0.0.0" {
				// 对于非主机路由，计算 CIDR 前缀
				prefix := netmaskToCIDR(netmask)
				if prefix > 0 && prefix < 32 {
					destination = fmt.Sprintf("%s/%d", destination, prefix)
				}
			} else if netmask == "0.0.0.0" {
				// 默认路由
				destination = "0.0.0.0/0"
			} else {
				// 主机路由
				destination = destination + "/32"
			}

			routes = append(routes, RouteEntry{
				Destination: destination,
				Gateway:     gateway,
				Interface:   iface,
				Metric:      metric,
				IPVersion:   IPVersion4,
				Netmask:     netmask,
			})
			continue
		}

		// 尝试匹配 IPv6 路由
		ipv6Matches := ipv6RoutePattern.FindStringSubmatch(line)
		if ipv6Matches != nil {
			ifaceIndex, _ := strconv.Atoi(ipv6Matches[1])
			metric, _ := strconv.Atoi(ipv6Matches[2])
			destination := ipv6Matches[3]
			gateway := ipv6Matches[4]

			// 跳过 On-link 路由（本地路由）
			if gateway == "On-link" {
				gateway = ""
			}

			// Windows IPv6 路由通常已经包含前缀长度
			// 如果没有，对于主机路由添加 /128
			if !strings.Contains(destination, "/") {
				destination = destination + "/128"
			}

			// 将接口索引转换为接口名称（这里简化处理，使用索引）
			iface := fmt.Sprintf("%d", ifaceIndex)

			routes = append(routes, RouteEntry{
				Destination: destination,
				Gateway:     gateway,
				Interface:   iface,
				Metric:      metric,
				IPVersion:   IPVersion6,
			})
			continue
		}
	}

	return routes, nil
}

// netmaskToCIDR 将子网掩码转换为 CIDR 前缀长度
func netmaskToCIDR(netmask string) int {
	parts := strings.Split(netmask, ".")
	if len(parts) != 4 {
		return 0
	}

	bits := 0
	for _, part := range parts {
		val, err := strconv.Atoi(part)
		if err != nil {
			return 0
		}
		// 计算每个字节中 1 的个数
		for val > 0 {
			bits += val & 1
			val >>= 1
		}
	}
	return bits
}
