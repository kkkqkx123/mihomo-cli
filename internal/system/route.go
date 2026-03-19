package system

import (
	"fmt"
	"log"
	"strings"

	"github.com/kkkqkx123/mihomo-cli/internal/operation"
)

// RouteManager 路由表管理器
type RouteManager struct {
	operation *operation.Manager
}

// NewRouteManager 创建路由表管理器
func NewRouteManager(op *operation.Manager) *RouteManager {
	return &RouteManager{
		operation: op,
	}
}

// ListRoutes 列出所有路由
func (rm *RouteManager) ListRoutes() ([]RouteEntry, error) {
	return rm.listRoutes()
}

// DeleteRoute 删除路由
func (rm *RouteManager) DeleteRoute(route RouteEntry) error {
	err := rm.deleteRoute(route)

	if rm.operation != nil {
		details := fmt.Sprintf("%s via %s", route.Destination, route.Gateway)
		result := "success"
		if err != nil {
			result = "failed"
		}
		_ = rm.operation.Record("delete", "route", details, result, err)
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

	if rm.operation != nil {
		details := fmt.Sprintf("%s via %s", route.Destination, route.Gateway)
		result := "success"
		if err != nil {
			result = "failed"
		}
		_ = rm.operation.Record("add", "route", details, result, err)
	}

	return err
}

// AddRoutes 批量添加路由
func (rm *RouteManager) AddRoutes(routes []RouteEntry) error {
	var addedRoutes []RouteEntry

	for i, route := range routes {
		if err := rm.AddRoute(route); err != nil {
			// 添加失败时，回滚已添加的路由
			var rollbackErrors []string
			for _, addedRoute := range addedRoutes {
				if delErr := rm.DeleteRoute(addedRoute); delErr != nil {
					rollbackErrors = append(rollbackErrors,
						fmt.Sprintf("failed to delete %s: %v", addedRoute.Destination, delErr))
				}
			}

			// 构建详细的错误信息
			errMsg := fmt.Sprintf("failed to add route #%d (%s): %v", i+1, route.Destination, err)
			if len(addedRoutes) > 0 {
				if len(rollbackErrors) > 0 {
					errMsg = fmt.Sprintf("%s (rolled back %d routes, but %d rollback(s) failed: %s)",
						errMsg, len(addedRoutes), len(rollbackErrors), strings.Join(rollbackErrors, "; "))
				} else {
					errMsg = fmt.Sprintf("%s (successfully rolled back %d routes)", errMsg, len(addedRoutes))
				}
			}
			return fmt.Errorf("%s", errMsg)
		}
		addedRoutes = append(addedRoutes, route)
	}

	return nil
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
	switch route.IPVersion {
	case IPVersion4:
		// 检查是否包含 IPv6 地址
		if strings.Contains(route.Destination, ":") || (route.Gateway != "" && strings.Contains(route.Gateway, ":")) {
			return fmt.Errorf("invalid IPv6 address in IPv4 route")
		}
	case IPVersion6:
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
			// 相同前缀但不同网关，记录警告日志
			log.Printf("Warning: route conflict detected - same destination %s with different gateways: %s vs %s",
				route.Destination, existingRoute.Gateway, route.Gateway)
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

// routeKey 生成路由的唯一键
func routeKey(route RouteEntry) string {
	return fmt.Sprintf("%s|%s|%s|%d",
		route.Destination,
		route.Gateway,
		route.Interface,
		route.Metric,
	)
}

// routesEqual 比较两个路由是否相等
func routesEqual(a, b RouteEntry) bool {
	return a.Destination == b.Destination &&
		a.Gateway == b.Gateway &&
		a.Interface == b.Interface &&
		a.Metric == b.Metric &&
		a.IPVersion == b.IPVersion &&
		a.Netmask == b.Netmask
}
