package system

import (
	"fmt"
	"strconv"
	"strings"
)

// RouteManager 路由表管理器
type RouteManager struct {
	audit *AuditLogger
}

// NewRouteManager 创建路由表管理器
func NewRouteManager(audit *AuditLogger) *RouteManager {
	return &RouteManager{
		audit: audit,
	}
}

// ListRoutes 列出所有路由
func (rm *RouteManager) ListRoutes() ([]RouteEntry, error) {
	return rm.listRoutes()
}

// CheckAbnormalRoutes 检查异常路由
func (rm *RouteManager) CheckAbnormalRoutes() ([]RouteEntry, error) {
	routes, err := rm.ListRoutes()
	if err != nil {
		return nil, err
	}

	// 检查异常路由（例如指向不存在的网关）
	var abnormal []RouteEntry
	for _, route := range routes {
		if isAbnormalRoute(route) {
			abnormal = append(abnormal, route)
		}
	}

	return abnormal, nil
}

// DeleteRoute 删除路由
func (rm *RouteManager) DeleteRoute(route RouteEntry) error {
	err := rm.deleteRoute(route)

	if rm.audit != nil {
		details := fmt.Sprintf("%s via %s", route.Destination, route.Gateway)
		result := "success"
		if err != nil {
			result = "failed"
		}
		_ = rm.audit.Record("delete", "route", details, result, err)
	}

	return err
}

// AddRoute 添加路由
func (rm *RouteManager) AddRoute(route RouteEntry) error {
	// 验证路由
	if err := rm.validateRoute(route); err != nil {
		return err
	}

	// 检查冲突
	if err := rm.checkRouteConflict(route); err != nil {
		return err
	}

	err := rm.addRoute(route)

	if rm.audit != nil {
		details := fmt.Sprintf("%s via %s", route.Destination, route.Gateway)
		result := "success"
		if err != nil {
			result = "failed"
		}
		_ = rm.audit.Record("add", "route", details, result, err)
	}

	return err
}

// CleanupMihomoRoutes 清理 Mihomo 添加的路由
func (rm *RouteManager) CleanupMihomoRoutes() error {
	routes, err := rm.ListRoutes()
	if err != nil {
		return err
	}

	var lastErr error
	for _, route := range routes {
		// 检查是否是 Mihomo 添加的路由
		if isMihomoRoute(route) {
			if err := rm.DeleteRoute(route); err != nil {
				lastErr = err
			}
		}
	}

	return lastErr
}

// CheckResidual 检查是否有残留路由
func (rm *RouteManager) CheckResidual() (*Problem, error) {
	routes, err := rm.ListRoutes()
	if err != nil {
		return nil, err
	}

	var mihomoRoutes []RouteEntry
	for _, route := range routes {
		if isMihomoRoute(route) {
			mihomoRoutes = append(mihomoRoutes, route)
		}
	}

	if len(mihomoRoutes) > 0 {
		routeStrs := make([]string, len(mihomoRoutes))
		for i, route := range mihomoRoutes {
			routeStrs[i] = fmt.Sprintf("%s via %s", route.Destination, route.Gateway)
		}

		return &Problem{
			Type:        ProblemConfigResidual,
			Severity:    SeverityHigh,
			Description: "Routes added by Mihomo still exist",
			Details: map[string]interface{}{
				"routes": routeStrs,
			},
			Solutions: []Solution{
				{
					Description: "Remove routes",
					Command:     "mihomo-cli system cleanup --route",
					Auto:        true,
				},
				{
					Description: "Restart Mihomo to cleanup",
					Command:     "mihomo-cli restart",
					Auto:        true,
				},
				{
					Description: "Restart system to cleanup",
					Command:     "restart computer",
					Auto:        false,
				},
			},
		}, nil
	}

	return nil, nil
}

// Cleanup 清理路由表
func (rm *RouteManager) Cleanup() error {
	return rm.CleanupMihomoRoutes()
}

// FilterRoutes 过滤路由
func (rm *RouteManager) FilterRoutes(filter RouteFilter) ([]RouteEntry, error) {
	routes, err := rm.ListRoutes()
	if err != nil {
		return nil, err
	}

	var filtered []RouteEntry
	for _, route := range routes {
		if filter.match(route) {
			filtered = append(filtered, route)
		}
	}

	return filtered, nil
}

// AddRoutes 批量添加路由
func (rm *RouteManager) AddRoutes(routes []RouteEntry) error {
	var addedRoutes []RouteEntry
	var lastErr error

	for _, route := range routes {
		if err := rm.AddRoute(route); err != nil {
			lastErr = err
			// 添加失败时，回滚已添加的路由
			for _, addedRoute := range addedRoutes {
				_ = rm.DeleteRoute(addedRoute)
			}
			return fmt.Errorf("failed to add route %s: %w (rolled back)", route.Destination, err)
		}
		addedRoutes = append(addedRoutes, route)
	}

	return lastErr
}

// DeleteRoutes 批量删除路由
func (rm *RouteManager) DeleteRoutes(routes []RouteEntry) error {
	var lastErr error
	for _, route := range routes {
		if err := rm.DeleteRoute(route); err != nil {
			lastErr = err
		}
	}
	return lastErr
}

// isAbnormalRoute 检查是否是异常路由
func isAbnormalRoute(route RouteEntry) bool {
	// 1. 检查网关是否为空但路由不是直连路由
	if route.Gateway == "" && route.Destination != "0.0.0.0/0" && route.Destination != "::/0" {
		// 如果没有网关，且不是默认路由，可能是异常路由
		// 但也可能是直连路由，需要进一步检查
		// 这里暂时不做严格检查
	}

	// 2. 检查接口是否为空
	if route.Interface == "" {
		// 没有接口的路由可能是异常的
		return true
	}

	// 3. 检查网关是否为本地地址（0.0.0.0 或 ::）
	if route.Gateway == "0.0.0.0" || route.Gateway == "::" {
		// 本地地址作为网关通常是异常的
		return true
	}

	// 4. 检查度量值是否为负数或异常大
	if route.Metric < 0 || route.Metric > 9999 {
		return true
	}

	// 5. 检查目的地址格式
	if route.Destination == "" {
		return true
	}

	// 6. 检查 IPv4 子网掩码格式（仅 Windows）
	if route.IPVersion == IPVersion4 && route.Netmask != "" {
		// 验证子网掩码格式
		parts := strings.Split(route.Netmask, ".")
		if len(parts) != 4 {
			return true
		}
		for _, part := range parts {
			val, err := strconv.Atoi(part)
			if err != nil || val < 0 || val > 255 {
				return true
			}
		}
	}

	return false
}

// isMihomoRoute 检查是否是 Mihomo 添加的路由
func isMihomoRoute(route RouteEntry) bool {
	// 1. 检查接口名称
	tunInterfaces := []string{"utun", "tun", "clash", "mihomo"}
	for _, iface := range tunInterfaces {
		if strings.Contains(strings.ToLower(route.Interface), iface) {
			return true
		}
	}

	// 2. 检查网关是否指向 Mihomo 常用的地址
	mihomoGateways := []string{"198.18.0.1", "198.18.0.2"}
	for _, gateway := range mihomoGateways {
		if route.Gateway == gateway {
			return true
		}
	}

	// 3. 检查目的地址是否是 Mihomo 常用的范围
	// Mihomo TUN 模式通常使用 198.18.0.0/16 或类似的私有地址
	if strings.HasPrefix(route.Destination, "198.18.") {
		return true
	}

	// 4. 检查路由标志（仅 macOS）
	if route.Flags != "" {
		// Mihomo 可能会设置特定的路由标志
		// 这里可以根据实际情况扩展
	}

	return false
}

// validateRoute 验证路由配置
func (rm *RouteManager) validateRoute(route RouteEntry) error {
	// 检查目的地址
	if route.Destination == "" {
		return fmt.Errorf("destination is required")
	}

	// 检查接口或网关至少有一个
	if route.Interface == "" && route.Gateway == "" {
		return fmt.Errorf("interface or gateway is required")
	}

	// 检查 IP 版本一致性
	if route.IPVersion == IPVersion4 {
		// 检查是否包含 IPv6 地址
		if strings.Contains(route.Destination, ":") || (route.Gateway != "" && strings.Contains(route.Gateway, ":")) {
			return fmt.Errorf("invalid IPv6 address in IPv4 route")
		}
	} else if route.IPVersion == IPVersion6 {
		// 检查是否包含 IPv4 地址
		if !strings.Contains(route.Destination, ":") && !strings.Contains(route.Destination, "default") {
			return fmt.Errorf("invalid IPv4 address in IPv6 route")
		}
		if route.Gateway != "" && !strings.Contains(route.Gateway, ":") && !strings.Contains(route.Gateway, "On-link") {
			return fmt.Errorf("invalid IPv4 address in IPv6 route gateway")
		}
	}

	return nil
}

// checkRouteConflict 检查路由冲突
func (rm *RouteManager) checkRouteConflict(route RouteEntry) error {
	routes, err := rm.ListRoutes()
	if err != nil {
		return err
	}

	for _, existingRoute := range routes {
		// 检查完全相同的路由
		if existingRoute.Destination == route.Destination &&
			existingRoute.Gateway == route.Gateway &&
			existingRoute.Interface == route.Interface {
			return fmt.Errorf("route already exists: %s via %s", route.Destination, route.Gateway)
		}

		// 检查相同目的地址但不同网关的路由冲突
		if existingRoute.Destination == route.Destination &&
			existingRoute.Gateway != route.Gateway &&
			existingRoute.Gateway != "" &&
			route.Gateway != "" {
			// 相同前缀但不同网关，可能需要警告
			// 这里暂时不阻止，但可以记录日志
		}
	}

	return nil
}

// match 检查路由是否匹配过滤器
func (rf *RouteFilter) match(route RouteEntry) bool {
	// 过滤 IP 版本
	if rf.IPVersion != "" && route.IPVersion != rf.IPVersion {
		return false
	}

	// 过滤接口
	if rf.Interface != "" && route.Interface != rf.Interface {
		return false
	}

	// 过滤网关
	if rf.Gateway != "" && route.Gateway != rf.Gateway {
		return false
	}

	// 过滤目的地址（支持前缀匹配）
	if rf.Destination != "" {
		if route.Destination != rf.Destination {
			// 检查是否是前缀匹配
			if !strings.HasPrefix(route.Destination, rf.Destination) {
				return false
			}
		}
	}

	return true
}
