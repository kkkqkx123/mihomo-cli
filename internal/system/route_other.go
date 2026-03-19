//go:build !windows && !linux && !darwin

package system

import "fmt"

// listRoutes 列出路由表（不支持的平台）
func (rm *RouteManager) listRoutes() ([]RouteEntry, error) {
	return nil, fmt.Errorf("route listing not supported on this platform")
}

// deleteRoute 删除路由（不支持的平台）
func (rm *RouteManager) deleteRoute(_ RouteEntry) error {
	return fmt.Errorf("route deletion not supported on this platform")
}

// addRoute 添加路由（不支持的平台）
func (rm *RouteManager) addRoute(_ RouteEntry) error {
	return fmt.Errorf("route addition not supported on this platform")
}

// checkInterfaceExistsImpl 检查接口是否存在（不支持的平台）
func checkInterfaceExistsImpl(_ string) bool {
	// 对于不支持的平台，返回 false
	return false
}

// checkGatewayReachableImpl 检查网关是否可达（不支持的平台）
func checkGatewayReachableImpl(_ string) bool {
	// 对于不支持的平台，返回 true（假设网关可达）
	return true
}

// checkMihomoRouteFlagsImpl 检查路由标志是否表明是 Mihomo 添加的路由（不支持的平台）
func checkMihomoRouteFlagsImpl(_ string) bool {
	// 对于不支持的平台，返回 false
	return false
}

// GetInterfaceInfo 获取接口详细信息（不支持的平台）
func (rm *RouteManager) GetInterfaceInfo(_ string) (map[string]string, error) {
	return nil, fmt.Errorf("interface info not supported on this platform")
}

// GetActiveInterfaceList 获取活动接口列表（不支持的平台）
func (rm *RouteManager) GetActiveInterfaceList() ([]string, error) {
	return nil, fmt.Errorf("interface list not supported on this platform")
}
