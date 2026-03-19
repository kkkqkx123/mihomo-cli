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

// addRoute 添加 Windows 系统路由
func (rm *RouteManager) addRoute(route RouteEntry) error {
	// route add 目标 mask 子网掩码 网关 metric 度量值
	args := []string{"add", route.Destination}

	// 对于 IPv4 路由，添加 mask 参数
	if route.IPVersion == IPVersion4 {
		if route.Netmask != "" {
			args = append(args, "mask", route.Netmask)
		} else {
			// 如果没有提供子网掩码，尝试从目的地址中提取
			if strings.Contains(route.Destination, "/") {
				parts := strings.Split(route.Destination, "/")
				if len(parts) == 2 {
					cidr := parts[1]
					netmask := cidrToNetmask(cidr)
					args = append(args, "mask", netmask)
				}
			}
		}
	}

	// 添加网关
	if route.Gateway != "" {
		args = append(args, route.Gateway)
	}

	// 添加度量值
	if route.Metric > 0 {
		args = append(args, "metric", strconv.Itoa(route.Metric))
	}

	// 添加接口（如果有）
	if route.Interface != "" {
		args = append(args, "if", route.Interface)
	}

	cmd := exec.Command("route", args...)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to add route %s: %w, stderr: %s", route.Destination, err, stderr.String())
	}
	return nil
}

// parseWindowsRouteOutput 解析 Windows route print 命令输出
// Windows route print 输出格式示例 (IPv4):
// ===========================================================================
// Active Routes:
// Network Destination        Netmask          Gateway       Interface  Metric
//
//	  0.0.0.0          0.0.0.0      192.168.1.1    192.168.1.100     25
//	127.0.0.0        255.0.0.0         On-link         127.0.0.1    331
//
// ===========================================================================
//
// Windows route print 输出格式示例 (IPv6):
// ===========================================================================
// Active Routes:
//
//	If Metric Network Destination      Gateway
//	1    331  ::1/128                  On-link
//	1     25  ::/0                     fe80::1
//
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

// cidrToNetmask 将 CIDR 前缀长度转换为子网掩码
func cidrToNetmask(cidrStr string) string {
	cidr, err := strconv.Atoi(cidrStr)
	if err != nil {
		return ""
	}

	if cidr < 0 || cidr > 32 {
		return ""
	}

	var mask uint32
	for i := 0; i < cidr; i++ {
		mask |= 1 << (31 - i)
	}

	return fmt.Sprintf("%d.%d.%d.%d",
		(mask>>24)&0xFF,
		(mask>>16)&0xFF,
		(mask>>8)&0xFF,
		mask&0xFF)
}

// Windows 平台特定实现
func checkInterfaceExistsImpl(iface string) bool {
	if iface == "" {
		return false
	}

	// 使用 netsh interface show interface 命令检查接口
	cmd := exec.Command("netsh", "interface", "show", "interface")
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	output, err := cmd.Output()
	if err != nil {
		return false
	}

	// 检查输出中是否包含该接口
	outputStr := strings.ToLower(string(output))
	searchStr := strings.ToLower(iface)

	// 支持通过接口名称或 IP 地址匹配
	lines := strings.Split(outputStr, "\n")
	for _, line := range lines {
		if strings.Contains(line, searchStr) {
			// 检查是否是启用状态
			if strings.Contains(line, "connected") || strings.Contains(line, "已连接") {
				return true
			}
		}
	}

	return false
}

func checkGatewayReachableImpl(gateway string) bool {
	if gateway == "" || gateway == "On-link" {
		return true // 直连路由或空网关不需要检查
	}

	// 使用 ping 命令检测网关是否可达
	cmd := exec.Command("ping", "-n", "1", "-w", "1000", gateway)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	err := cmd.Run()
	return err == nil
}

// checkMihomoRouteFlagsImpl 检查 Windows 路由标志是否表明是 Mihomo 添加的路由
// Windows 路由表不使用 BSD 风格的标志,而是使用 route print 命令的输出格式
// Windows 路由标志通常包括:
// - 活跃路由: 在路由表中标记为 "Active"
// - 永久路由: 在路由表中标记为 "Permanent"
// - 接口索引: 用于标识网络接口
func checkMihomoRouteFlagsImpl(_ string) bool {
	// Windows 的路由标志格式与 BSD/macOS 不同
	// Windows route print 输出中,flags 字段通常为空或包含特定标记
	// 对于 Windows,我们主要依赖接口名称和网关地址来判断
	// 这里保留接口,但返回 false,因为 Windows 不使用 BSD 风格的标志
	return false
}

// GetInterfaceInfo 获取 Windows 接口详细信息
func (rm *RouteManager) GetInterfaceInfo(iface string) (map[string]string, error) {
	if iface == "" {
		return nil, fmt.Errorf("interface name is empty")
	}

	// 使用 netsh 命令获取接口详细信息
	cmd := exec.Command("netsh", "interface", "ip", "show", "config", "name="+iface)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to get interface info: %w, stderr: %s", err, stderr.String())
	}

	info := make(map[string]string)
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// 解析配置信息，格式如：
		// Configuration for interface "Ethernet"
		// IP Address: 192.168.1.100
		// Subnet Prefix: 192.168.1.0/24 (mask 255.255.255.0)
		// Default Gateway: 192.168.1.1
		// Gateway Metric: 0
		// Interface Metric: 25
		if strings.Contains(line, ":") {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) == 2 {
				key := strings.TrimSpace(parts[0])
				value := strings.TrimSpace(parts[1])
				info[key] = value
			}
		}
	}

	// 获取接口状态
	statusCmd := exec.Command("netsh", "interface", "show", "interface", "name="+iface)
	var statusStderr bytes.Buffer
	statusCmd.Stderr = &statusStderr
	statusOutput, err := statusCmd.Output()
	if err == nil {
		statusLines := strings.Split(string(statusOutput), "\n")
		for _, line := range statusLines {
			line = strings.TrimSpace(line)
			// 解析状态信息
			// 格式：已启用  已连接  专用    Ethernet
			if strings.Contains(line, "connected") || strings.Contains(line, "已连接") {
				info["status"] = "connected"
			} else if strings.Contains(line, "disconnected") || strings.Contains(line, "已断开") {
				info["status"] = "disconnected"
			}
		}
	}

	return info, nil
}

// GetActiveInterfaceList 获取 Windows 活动接口列表
func (rm *RouteManager) GetActiveInterfaceList() ([]string, error) {
	// 使用 netsh 命令获取接口列表
	cmd := exec.Command("netsh", "interface", "show", "interface")
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to get interface list: %w, stderr: %s", err, stderr.String())
	}

	var interfaces []string
	lines := strings.Split(string(output), "\n")

	// Windows netsh interface show interface 输出格式：
	// 管理状态    状态          类型             接口名称
	// ---------------------------------------------------------------------------
	// 已启用            已连接            专用                Ethernet
	// 已启用            已连接            专用                Wi-Fi
	//
	// 或英文版：
	// Admin State    State          Type             Interface Name
	// ---------------------------------------------------------------------------
	// Enabled        Connected      Dedicated        Ethernet
	// Enabled        Connected      Dedicated        Wi-Fi

	for _, line := range lines {
		line = strings.TrimSpace(line)
		// 跳过空行和标题行
		if line == "" || strings.Contains(line, "Admin State") || strings.Contains(line, "管理状态") || strings.Contains(line, "---") {
			continue
		}

		// 解析接口行
		fields := strings.Fields(line)
		if len(fields) >= 4 {
			// 检查是否是已连接状态
			// 中文版：已启用  已连接  ...
			// 英文版：Enabled Connected ...
			isConnected := strings.Contains(line, "connected") || strings.Contains(line, "已连接")
			isEnabled := strings.Contains(line, "enabled") || strings.Contains(line, "已启用")

			if isConnected && isEnabled {
				// 接口名称是最后一个字段
				ifaceName := fields[len(fields)-1]
				interfaces = append(interfaces, ifaceName)
			}
		}
	}

	return interfaces, nil
}
