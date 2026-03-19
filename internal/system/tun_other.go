//go:build !windows && !linux && !darwin

package system

import "fmt"

// listTUNDevices 列出 TUN 设备（不支持的平台）
func (tm *TUNManager) listTUNDevices() ([]TUNState, error) {
	return nil, fmt.Errorf("TUN device listing not supported on this platform")
}

// removeTUN 删除 TUN 设备（不支持的平台）
func (tm *TUNManager) removeTUN(name string) error {
	return fmt.Errorf("TUN device removal not supported on this platform")
}
