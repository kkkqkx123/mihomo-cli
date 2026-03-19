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
func parseDarwinRouteOutput(output []byte) ([]RouteEntry, error) {
	var routes []RouteEntry
	lines := strings.Split(string(output), "\n")

	inInternet := false
	for _, line := range lines {
		line = strings.TrimSpace(line)

		// 检测 Internet 段落开始
		if line == "Internet:" {
			inInternet = true
			continue
		}

		// 检测其他段落（Internet6 等）
		if strings.HasSuffix(line, ":") && line != "Internet:" {
			inInternet = false
			continue
		}

		if !inInternet {
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
	} else {
		route.Destination = destination
		// 如果没有 CIDR 后缀，检查是否是网络地址
		if !strings.Contains(destination, "/") && !strings.Contains(destination, ".") {
			// 可能是网络前缀如 "192.168.1"，需要添加 CIDR
			// macOS 通常省略 .0 后缀
			if strings.Count(destination, ".") == 2 {
				route.Destination = destination + ".0/24"
			} else if strings.Count(destination, ".") == 1 {
				route.Destination = destination + ".0.0/16"
			} else if strings.Count(destination, ".") == 0 && destination != "127" {
				// 单个数字，可能是 A 类网络
				route.Destination = destination + ".0.0.0/8"
			} else if destination == "127" {
				route.Destination = "127.0.0.0/8"
			} else {
				route.Destination = destination + "/32"
			}
		} else if !strings.Contains(destination, "/") {
			// 有完整 IP 但没有 CIDR，默认是主机路由
			route.Destination = destination + "/32"
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
