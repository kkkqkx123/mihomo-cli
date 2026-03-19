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
	cmd := exec.Command("route", args...)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to delete route %s: %w, stderr: %s", route.Destination, err, stderr.String())
	}
	return nil
}

// parseWindowsRouteOutput 解析 Windows route print 命令输出
// Windows route print 输出格式示例:
// ===========================================================================
// Active Routes:
// Network Destination        Netmask          Gateway       Interface  Metric
//           0.0.0.0          0.0.0.0      192.168.1.1    192.168.1.100     25
//         127.0.0.0        255.0.0.0         On-link         127.0.0.1    331
// ===========================================================================
func parseWindowsRouteOutput(output []byte) ([]RouteEntry, error) {
	var routes []RouteEntry
	lines := strings.Split(string(output), "\n")

	// 正则匹配路由行
	// 格式: Network Destination    Netmask    Gateway    Interface    Metric
	routePattern := regexp.MustCompile(`^\s*(\S+)\s+(\S+)\s+(\S+)\s+(\S+)\s+(\d+)\s*$`)

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

		matches := routePattern.FindStringSubmatch(line)
		if matches == nil {
			continue
		}

		destination := matches[1]
		netmask := matches[2]
		gateway := matches[3]
		iface := matches[4]
		metric, _ := strconv.Atoi(matches[5])

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
		}

		routes = append(routes, RouteEntry{
			Destination: destination,
			Gateway:     gateway,
			Interface:   iface,
			Metric:      metric,
		})
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
