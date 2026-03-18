//go:build windows

package service

import (
	"syscall"

	"golang.org/x/sys/windows/svc/mgr"

	pkgerrors "github.com/kkkqkx123/mihomo-cli/pkg/errors"
)

// ServiceStatus 服务状态
type ServiceStatus string

const (
	StatusRunning      ServiceStatus = "running"
	StatusStopped      ServiceStatus = "stopped"
	StatusNotInstalled ServiceStatus = "not-installed"
	StatusUnknown      ServiceStatus = "unknown"
)

// ServiceManager Windows 服务管理器
type ServiceManager struct {
	serviceName string
	displayName string
	description string
	exePath     string
}

// NewServiceManager 创建新的服务管理器
func NewServiceManager(serviceName, displayName, description, exePath string) *ServiceManager {
	return &ServiceManager{
		serviceName: serviceName,
		displayName: displayName,
		description: description,
		exePath:     exePath,
	}
}

// OpenManager 打开服务管理器
func (sm *ServiceManager) OpenManager() (*mgr.Mgr, error) {
	m, err := mgr.Connect()
	if err != nil {
		return nil, pkgerrors.ErrService("failed to connect to service manager", err)
	}
	return m, nil
}

// OpenService 打开指定服务
func (sm *ServiceManager) OpenService(m *mgr.Mgr) (*mgr.Service, error) {
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
func (sm *ServiceManager) ServiceExists() (bool, error) {
	m, err := sm.OpenManager()
	if err != nil {
		return false, err
	}
	defer m.Disconnect()

	s, err := sm.OpenService(m)
	if err != nil {
		// 服务不存在
		return false, nil
	}
	defer s.Close()

	return true, nil
}

// GetServiceName 获取服务名称
func (sm *ServiceManager) GetServiceName() string {
	return sm.serviceName
}

// GetDisplayName 获取显示名称
func (sm *ServiceManager) GetDisplayName() string {
	return sm.displayName
}

// GetExePath 获取可执行文件路径
func (sm *ServiceManager) GetExePath() string {
	return sm.exePath
}
