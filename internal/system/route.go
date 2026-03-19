package system

import (
	"fmt"
	"strconv"
	"strings"
	"time"
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
				Route: route,
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

// DiagnoseNetworkRouting 诊断网络路由问题（综合检查）
func (rm *RouteManager) DiagnoseNetworkRouting() (*NetworkDiagnosis, error) {
	diagnosis := &NetworkDiagnosis{
		Timestamp: time.Now(),
	}

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

	// 3. 检查活动接口
	activeInterfaces, err := getActiveInterfaceList()
	if err != nil {
		diagnosis.Error = err
		return diagnosis, err
	}
	diagnosis.ActiveInterfaces = activeInterfaces

	// 4. 综合判断网络状态
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

// RouteConflict 路由冲突信息
type RouteConflict struct {
	Type           string      `json:"type"`
	Severity       string      `json:"severity"` // Critical, High, Medium, Low
	Message        string      `json:"message"`
	Routes         []RouteEntry `json:"routes"`
	ActiveRoute    RouteEntry  `json:"active_route"`
	Recommendation string      `json:"recommendation"`
}

// NetworkDiagnosis 网络路由诊断结果
type NetworkDiagnosis struct {
	Timestamp              time.Time        `json:"timestamp"`
	Health                 string           `json:"health"` // Healthy, Warning, Critical
	DefaultRouteConflicts  []RouteConflict  `json:"default_route_conflicts"`
	ResidualRoutes         []ResidualRoute  `json:"residual_routes"`
	ActiveInterfaces       []string         `json:"active_interfaces"`
	Error                  error            `json:"error,omitempty"`
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

// checkInterfaceExistsImpl 平台特定的接口检测实现（由各平台文件实现）
func checkInterfaceExistsImpl(iface string) bool {
	return false // 默认实现，子类覆盖
}

// checkGatewayReachableImpl 平台特定的网关可达性检测实现（由各平台文件实现）
func checkGatewayReachableImpl(gateway string) bool {
	return false // 默认实现，子类覆盖
}

// getInterfaceInfoImpl 平台特定的接口信息获取实现（由各平台文件实现）
func getInterfaceInfoImpl(iface string) (map[string]string, error) {
	return nil, fmt.Errorf("not implemented")
}

// getActiveInterfaceListImpl 平台特定的活动接口列表获取实现（由各平台文件实现）
func getActiveInterfaceListImpl() ([]string, error) {
	return nil, fmt.Errorf("not implemented")
}

// ResidualRoute 残留路由诊断信息
type ResidualRoute struct {
	Route            RouteEntry `json:"route"`
	Reason           string     `json:"reason"`
	InterfaceExists  bool       `json:"interface_exists"`
	GatewayReachable bool       `json:"gateway_reachable"`
	Issue            string     `json:"issue,omitempty"`
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

// CleanupMihomoResidualRoutes 清理 Mihomo 残留路由并返回详细报告
func (rm *RouteManager) CleanupMihomoResidualRoutes() (*CleanupReport, error) {
	residualRoutes, err := rm.CheckMihomoResidualRoutes()
	if err != nil {
		return nil, err
	}

	report := &CleanupReport{
		TotalFound:    len(residualRoutes),
		TotalRemoved:  0,
		Failed:        []RouteEntry{},
		Skipped:       []RouteEntry{},
	}

	for _, residual := range residualRoutes {
		route := residual.Route

		// 检查是否需要跳过（例如度量值不是 0 的默认路由）
		if shouldSkipRoute(route) {
			report.Skipped = append(report.Skipped, route)
			continue
		}

		// 尝试删除路由
		if err := rm.DeleteRoute(route); err != nil {
			report.Failed = append(report.Failed, route)
			report.LastError = err
		} else {
			report.TotalRemoved++
			report.Removed = append(report.Removed, route)
		}
	}

	return report, nil
}

// shouldSkipRoute 判断是否应该跳过删除某些路由
func shouldSkipRoute(route RouteEntry) bool {
	// 不要删除直连路由（On-link）
	if strings.Contains(route.Gateway, "On-link") {
		return true
	}

	// 不要删除不是 Mihomo 路由的配置
	if !isMihomoGateway(route.Gateway) && !strings.HasPrefix(route.Interface, "198.18.") {
		return true
	}

	// 对于非默认路由，如果度量值较高，可能是系统自动添加的，跳过
	if route.Destination != "0.0.0.0" && route.Metric > 100 {
		return true
	}

	return false
}

// CleanupReport 清理报告
type CleanupReport struct {
	TotalFound   int          `json:"total_found"`
	TotalRemoved int          `json:"total_removed"`
	Removed      []RouteEntry `json:"removed"`
	Failed       []RouteEntry `json:"failed"`
	Skipped      []RouteEntry `json:"skipped"`
	LastError    error        `json:"last_error,omitempty"`
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
