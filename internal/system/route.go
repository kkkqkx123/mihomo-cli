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
	// 1. 检查目的地址是否为空
	if route.Destination == "" {
		return true
	}

	// 2. 检查度量值是否异常
	if route.Metric < 0 || route.Metric > 9999 {
		return true
	}

	// 3. 处理直连路由（On-link）
	// 直连路由没有网关，但有接口，这是正常的
	isOnLink := strings.Contains(route.Gateway, "On-link") ||
		route.Gateway == "" && route.Interface != ""

	if isOnLink {
		// 直连路由，检查接口是否有效
		if route.Interface == "" {
			return true
		}
		return false // 直连路由是正常的
	}

	// 4. 对于非直连路由，检查网关
	if route.Gateway == "" {
		// 非直连路由必须有网关
		return true
	}

	// 5. 检查网关是否为无效地址
	invalidGateways := []string{"0.0.0.0", "::", "127.0.0.1", "::1"}
	for _, invalid := range invalidGateways {
		if route.Gateway == invalid {
			// 如果网关是无效地址，且接口为空，则异常
			if route.Interface == "" {
				return true
			}
		}
	}

	// 6. 检查接口和网关的兼容性
	// 如果有网关但接口为空，可能有问题
	if route.Gateway != "" && route.Interface == "" {
		// 某些情况下可能是正常的（如默认路由）
		// 但对于非默认路由，应该有接口
		if route.Destination != "0.0.0.0/0" &&
			route.Destination != "default" &&
			route.Destination != "::/0" {
			return true
		}
	}

	// 7. 检查 IPv4 子网掩码格式（仅 Windows）
	if route.IPVersion == IPVersion4 && route.Netmask != "" {
		if !isValidNetmask(route.Netmask) {
			return true
		}
	}

	// 8. 检查 Mihomo 相关的异常路由（重点）
	// 例如：Mihomo 添加的路由但没有对应的 TUN 接口
	mihomoGateway := isMihomoGateway(route.Gateway)
	hasTunInterface := isTunInterface(route.Interface)

	if mihomoGateway && !hasTunInterface {
		// 网关指向 Mihomo 但没有 TUN 接口，可能是残留路由
		// 这正是你遇到的情况：网关 198.18.0.2，但接口不存在
		return true
	}

	// 9. 检查路由标志（仅 Unix-like 系统）
	if route.Flags != "" {
		// 某些异常标志可能表示路由问题
		// 例如：RTF_REJECT, RTF_BLACKHOLE 等
		rejectFlags := []string{"reject", "blackhole", "unreachable"}
		lowerFlags := strings.ToLower(route.Flags)
		for _, rejectFlag := range rejectFlags {
			if strings.Contains(lowerFlags, rejectFlag) {
				return true
			}
		}
	}

	return false
}

// isValidNetmask 验证子网掩码是否有效
func isValidNetmask(netmask string) bool {
	parts := strings.Split(netmask, ".")
	if len(parts) != 4 {
		return false
	}

	var value uint32
	for _, part := range parts {
		val, err := strconv.Atoi(part)
		if err != nil || val < 0 || val > 255 {
			return false
		}
		value = (value << 8) | uint32(val)
	}

	// 检查是否是有效的子网掩码（连续的 1 后跟连续的 0）
	invertedValue := ^value
	if invertedValue == 0 {
		return true // 255.255.255.255
	}

	// 检查是否是连续的
	if (invertedValue & (invertedValue + 1)) == 0 {
		return true
	}

	return false
}

// isMihomoGateway 检查网关是否指向 Mihomo
func isMihomoGateway(gateway string) bool {
	mihomoGateways := []string{"198.18.0.1", "198.18.0.2", "198.18.0.3"}
	for _, mg := range mihomoGateways {
		if gateway == mg {
			return true
		}
	}
	return false
}

// isTunInterface 检查是否是 TUN 接口
func isTunInterface(iface string) bool {
	if iface == "" {
		return false
	}

	tunPrefixes := []string{"utun", "tun", "clash", "mihomo", "wintun", "Meta Tunnel"}
	lowerIface := strings.ToLower(iface)

	for _, prefix := range tunPrefixes {
		if strings.Contains(lowerIface, prefix) {
			return true
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
