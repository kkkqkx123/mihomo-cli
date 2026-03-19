package system

import (
	"fmt"
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

// isAbnormalRoute 检查是否是异常路由
func isAbnormalRoute(route RouteEntry) bool {
	// 检查路由是否异常（例如网关不可达）
	// 这里可以添加更多的检查逻辑
	return false
}

// isMihomoRoute 检查是否是 Mihomo 添加的路由
func isMihomoRoute(route RouteEntry) bool {
	// Mihomo 通常会添加特定的路由
	// 这里可以根据实际情况判断
	// 例如：检查路由是否指向 TUN 接口
	tunInterfaces := []string{"utun", "tun", "clash", "mihomo"}
	for _, iface := range tunInterfaces {
		if route.Interface == iface {
			return true
		}
	}
	return false
}
