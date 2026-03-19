package system

import (
	"fmt"
	"strings"
)

// RouteConflict 路由冲突信息
type RouteConflict struct {
	Type           string       `json:"type"`
	Severity       string       `json:"severity"` // Critical, High, Medium, Low
	Message        string       `json:"message"`
	Routes         []RouteEntry `json:"routes"`
	ActiveRoute    RouteEntry   `json:"active_route"`
	Recommendation string       `json:"recommendation"`
}

// NetworkDiagnosis 网络路由诊断结果
type NetworkDiagnosis struct {
	Health                string         `json:"health"` // Healthy, Warning, Critical
	DefaultRouteConflicts []RouteConflict `json:"default_route_conflicts"`
	ResidualRoutes        []ResidualRoute `json:"residual_routes"`
	Error                 error           `json:"error,omitempty"`
}

// ResidualRoute 残留路由诊断信息
type ResidualRoute struct {
	Route            RouteEntry `json:"route"`
	Reason           string     `json:"reason"`
	InterfaceExists  bool       `json:"interface_exists"`
	GatewayReachable bool       `json:"gateway_reachable"`
	Issue            string     `json:"issue,omitempty"`
}

// StartCheckResult 启动前检查结果
type StartCheckResult struct {
	ReadyToStart         bool            `json:"ready_to_start"`
	HasResidualRoutes    bool            `json:"has_residual_routes"`
	HasCriticalConflicts bool            `json:"has_critical_conflicts"`
	ResidualRoutes       []ResidualRoute `json:"residual_routes,omitempty"`
	CriticalConflicts    []RouteConflict `json:"critical_conflicts,omitempty"`
	Recommendation       string          `json:"recommendation,omitempty"`
	CleanupCommand       string          `json:"cleanup_command,omitempty"`
}

// StopCheckResult 停止后检查结果
type StopCheckResult struct {
	HasResidualRoutes bool            `json:"has_residual_routes"`
	ResidualRoutes    []ResidualRoute `json:"residual_routes,omitempty"`
	CleanupCommand    string          `json:"cleanup_command,omitempty"`
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

// CheckMihomoResidualRoutes 检测 Mihomo 残留路由（针对用户遇到的特殊情况）
// 返回残留路由列表和详细诊断信息
func (rm *RouteManager) CheckMihomoResidualRoutes() ([]ResidualRoute, error) {
	routes, err := rm.ListRoutes()
	if err != nil {
		return nil, err
	}

	var residualRoutes []ResidualRoute
	for _, route := range routes {
		if isMihomoResidualRoute(route) {
			// 构建诊断信息
			diag := ResidualRoute{
				Route:  route,
				Reason: getResidualRouteReason(route),
			}

			// 尝试检测 TUN 接口状态
			if route.Interface != "" {
				diag.InterfaceExists = checkInterfaceExists(route.Interface)
			}

			// 检查度量值是否过低（可能导致网络问题）
			if route.Metric == 0 && route.Destination == "0.0.0.0" {
				diag.Issue = "Low priority default route (metric=0) may block network access"
			}

			// 检查网关是否可达
			if route.Gateway != "" && !strings.Contains(route.Gateway, "On-link") {
				diag.GatewayReachable = checkGatewayReachable(route.Gateway)
				if !diag.GatewayReachable {
					diag.Issue = fmt.Sprintf("Gateway %s is unreachable", route.Gateway)
				}
			}

			residualRoutes = append(residualRoutes, diag)
		}
	}

	return residualRoutes, nil
}

// CheckDefaultRouteConflicts 检查默认路由冲突（针对用户遇到的特殊情况）
// 特别检测是否有多个默认路由，以及是否有低优先级的默认路由
func (rm *RouteManager) CheckDefaultRouteConflicts() ([]RouteConflict, error) {
	routes, err := rm.ListRoutes()
	if err != nil {
		return nil, err
	}

	var defaultRoutes []RouteEntry
	for _, route := range routes {
		// 检查是否是默认路由
		if route.Destination == "0.0.0.0/0" || route.Destination == "::/0" || route.Destination == "default" {
			defaultRoutes = append(defaultRoutes, route)
		}
	}

	var conflicts []RouteConflict

	// 检查是否有多个默认路由
	if len(defaultRoutes) > 1 {
		conflict := RouteConflict{
			Type:     "MultipleDefaultRoutes",
			Severity: "High",
			Message:  fmt.Sprintf("Found %d default routes, which may cause network conflicts", len(defaultRoutes)),
			Routes:   defaultRoutes,
		}

		// 找出度量值最低的路由（优先级最高）
		var lowestMetricRoute *RouteEntry
		for i := range defaultRoutes {
			if lowestMetricRoute == nil || defaultRoutes[i].Metric < lowestMetricRoute.Metric {
				lowestMetricRoute = &defaultRoutes[i]
			}
		}

		if lowestMetricRoute != nil {
			conflict.ActiveRoute = *lowestMetricRoute
			conflict.Recommendation = fmt.Sprintf("Remove route: 0.0.0.0 mask 0.0.0.0 %s",
				lowestMetricRoute.Gateway)
		}

		conflicts = append(conflicts, conflict)
	}

	// 检查是否有指向 Mihomo 网关的默认路由
	for _, route := range defaultRoutes {
		if isMihomoGateway(route.Gateway) {
			ifaceExists := false
			if route.Interface != "" {
				ifaceExists = checkInterfaceExists(route.Interface)
			}

			if !ifaceExists {
				conflict := RouteConflict{
					Type:        "OrphanedMihomoRoute",
					Severity:    "Critical",
					Message:     "Default route points to Mihomo gateway but TUN interface does not exist",
					Routes:      []RouteEntry{route},
					ActiveRoute: route,
					Recommendation: fmt.Sprintf("Delete route: route delete 0.0.0.0 mask 0.0.0.0 %s",
						route.Gateway),
				}
				conflicts = append(conflicts, conflict)
			}
		}
	}

	return conflicts, nil
}

// DiagnoseNetworkRouting 诊断网络路由问题（按需检查）
// 用于用户遇到问题时主动调用诊断
func (rm *RouteManager) DiagnoseNetworkRouting() (*NetworkDiagnosis, error) {
	diagnosis := &NetworkDiagnosis{}

	// 1. 检查默认路由冲突
	conflicts, err := rm.CheckDefaultRouteConflicts()
	if err != nil {
		diagnosis.Error = err
		return diagnosis, err
	}
	diagnosis.DefaultRouteConflicts = conflicts

	// 2. 检查残留路由
	residualRoutes, err := rm.CheckMihomoResidualRoutes()
	if err != nil {
		diagnosis.Error = err
		return diagnosis, err
	}
	diagnosis.ResidualRoutes = residualRoutes

	// 3. 综合判断网络状态
	diagnosis.Health = "Healthy"
	if len(conflicts) > 0 {
		for _, conflict := range conflicts {
			if conflict.Severity == "Critical" {
				diagnosis.Health = "Critical"
				break
			}
			if diagnosis.Health == "Healthy" && conflict.Severity == "High" {
				diagnosis.Health = "Warning"
			}
		}
	}

	if len(residualRoutes) > 0 && diagnosis.Health == "Healthy" {
		diagnosis.Health = "Warning"
	}

	return diagnosis, nil
}

// CheckBeforeStart Mihomo 启动前检查
// 在启动 Mihomo 之前检查残留路由，避免冲突
func (rm *RouteManager) CheckBeforeStart() (*StartCheckResult, error) {
	result := &StartCheckResult{
		ReadyToStart: true,
	}

	// 检查残留路由
	residualRoutes, err := rm.CheckMihomoResidualRoutes()
	if err != nil {
		return result, fmt.Errorf("failed to check residual routes: %w", err)
	}

	if len(residualRoutes) > 0 {
		result.ReadyToStart = false
		result.HasResidualRoutes = true
		result.ResidualRoutes = residualRoutes
		result.Recommendation = "Clean up residual routes before starting Mihomo"
		result.CleanupCommand = "mihomo-cli system cleanup --route"
	}

	// 检查默认路由冲突
	conflicts, err := rm.CheckDefaultRouteConflicts()
	if err != nil {
		return result, fmt.Errorf("failed to check route conflicts: %w", err)
	}

	if len(conflicts) > 0 {
		for _, conflict := range conflicts {
			if conflict.Severity == "Critical" {
				result.ReadyToStart = false
				result.HasCriticalConflicts = true
				result.CriticalConflicts = append(result.CriticalConflicts, conflict)
			}
		}
	}

	return result, nil
}

// CheckAfterStop Mihomo 停止后检查
// 在停止 Mihomo 之后检查是否有残留路由
func (rm *RouteManager) CheckAfterStop() (*StopCheckResult, error) {
	result := &StopCheckResult{}

	// 检查残留路由
	residualRoutes, err := rm.CheckMihomoResidualRoutes()
	if err != nil {
		return result, fmt.Errorf("failed to check residual routes: %w", err)
	}

	if len(residualRoutes) > 0 {
		result.HasResidualRoutes = true
		result.ResidualRoutes = residualRoutes
		result.CleanupCommand = "mihomo-cli system cleanup --route"
	}

	return result, nil
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
		val := 0
		for _, c := range part {
			if c >= '0' && c <= '9' {
				val = val*10 + int(c-'0')
			}
		}
		if val < 0 || val > 255 {
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

// isMihomoResidualRoute 检测是否是 Mihomo 残留路由
func isMihomoResidualRoute(route RouteEntry) bool {
	// 情况1：网关指向 Mihomo 地址范围
	if strings.HasPrefix(route.Gateway, "198.18.") {
		// 检查接口是否存在 TUN 特征
		if !isTunInterface(route.Interface) {
			// 网关指向 Mihomo 但接口不是 TUN 接口，可能是残留路由
			return true
		}

		// 即使是 TUN 接口，如果度量值异常低也可能是残留
		if route.Metric == 0 && route.Destination == "0.0.0.0" {
			return true
		}
	}

	// 情况2：接口地址在 Mihomo 地址范围
	if route.Interface != "" && strings.HasPrefix(route.Interface, "198.18.") {
		// 但网关不是 Mihomo，可能是配置错误
		if !isMihomoGateway(route.Gateway) {
			return true
		}
	}

	// 情况3：目的地址在 Mihomo 范围但不是直连路由
	if strings.HasPrefix(route.Destination, "198.18.") &&
		!strings.Contains(route.Gateway, "On-link") &&
		route.Gateway != "" {
		return true
	}

	return false
}

// getResidualRouteReason 获取残留路由的原因
func getResidualRouteReason(route RouteEntry) string {
	if strings.HasPrefix(route.Gateway, "198.18.") && !isTunInterface(route.Interface) {
		return "Gateway points to Mihomo but TUN interface does not exist"
	}

	if route.Metric == 0 && route.Destination == "0.0.0.0" {
		return "Default route with metric 0 may cause network conflicts"
	}

	if strings.HasPrefix(route.Interface, "198.18.") && !isMihomoGateway(route.Gateway) {
		return "Interface in Mihomo range but gateway is not Mihomo"
	}

	return "Potential Mihomo residual route detected"
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

// checkInterfaceExists 检查接口是否存在
func checkInterfaceExists(iface string) bool {
	if iface == "" {
		return false
	}
	// 调用平台特定的实现
	return checkInterfaceExistsImpl(iface)
}

// checkGatewayReachable 检查网关是否可达
func checkGatewayReachable(gateway string) bool {
	// 调用平台特定的实现
	return checkGatewayReachableImpl(gateway)
}

// getInterfaceInfo 获取接口详细信息
func getInterfaceInfo(iface string) (map[string]string, error) {
	if iface == "" {
		return nil, fmt.Errorf("interface name is empty")
	}
	return getInterfaceInfoImpl(iface)
}

// getActiveInterfaceList 获取活动接口列表
func getActiveInterfaceList() ([]string, error) {
	return getActiveInterfaceListImpl()
}
