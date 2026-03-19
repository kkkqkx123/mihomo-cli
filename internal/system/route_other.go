//go:build !windows && !linux && !darwin

package system

import "fmt"

// listRoutes 列出路由表（不支持的平台）
func (rm *RouteManager) listRoutes() ([]RouteEntry, error) {
	return nil, fmt.Errorf("route listing not supported on this platform")
}

// deleteRoute 删除路由（不支持的平台）
func (rm *RouteManager) deleteRoute(route RouteEntry) error {
	return fmt.Errorf("route deletion not supported on this platform")
}
