//go:build windows

package service

import (
	"syscall"

	"golang.org/x/sys/windows/svc/mgr"

	pkgerrors "github.com/kkkqkx123/mihomo-cli/pkg/errors"
)

// windowsServiceManager Windows 服务管理器
type windowsServiceManager struct {
	serviceName string
	displayName string
	description string
	exePath     string
}

// newWindowsServiceManager 创建新的 Windows 服务管理器
func newWindowsServiceManager(serviceName, displayName, description, exePath string) ServiceManager {
	return &windowsServiceManager{
		serviceName: serviceName,
		displayName: displayName,
		description: description,
		exePath:     exePath,
	}
}

// OpenManager 打开服务管理器
func (sm *windowsServiceManager) OpenManager() (*mgr.Mgr, error) {
	m, err := mgr.Connect()
	if err != nil {
		return nil, pkgerrors.ErrService("failed to connect to service manager", err)
	}
	return m, nil
}

// OpenService 打开指定服务
func (sm *windowsServiceManager) OpenService(m *mgr.Mgr) (*mgr.Service, error) {
	s, err := m.OpenService(sm.serviceName)
	if err != nil {
		// 错误码 1060 表示服务不存在
		if errno, ok := err.(syscall.Errno); ok && errno == 1060 {
			return nil, pkgerrors.ErrService("service "+sm.serviceName+" does not exist", nil)
		}
		return nil, pkgerrors.ErrService("failed to open service "+sm.serviceName, err)
	}
	return s, nil
}

// ServiceExists 检查服务是否存在
func (sm *windowsServiceManager) ServiceExists() (bool, error) {
	m, err := sm.OpenManager()
	if err != nil {
		return false, err
	}
	defer func() { _ = m.Disconnect() }()

	s, err := sm.OpenService(m)
	if err != nil {
		// 服务不存在
		return false, nil
	}
	defer s.Close()

	return true, nil
}

// GetServiceName 获取服务名称
func (sm *windowsServiceManager) GetServiceName() string {
	return sm.serviceName
}

// GetDisplayName 获取显示名称
func (sm *windowsServiceManager) GetDisplayName() string {
	return sm.displayName
}

// GetExePath 获取可执行文件路径
func (sm *windowsServiceManager) GetExePath() string {
	return sm.exePath
}

// IsSupported 检查当前平台是否支持服务管理
func (sm *windowsServiceManager) IsSupported() bool {
	return true
}
