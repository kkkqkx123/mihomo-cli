//go:build windows

package service

import (
	"golang.org/x/sys/windows/svc/mgr"

	pkgerrors "github.com/kkkqkx123/mihomo-cli/pkg/errors"
)

// Install 安装 Windows 服务
func (sm *windowsServiceManager) Install() error {
	// 检查服务是否已存在
	exists, err := sm.ServiceExists()
	if err != nil {
		return err
	}
	if exists {
		return pkgerrors.ErrService("service "+sm.serviceName+" already exists", nil)
	}

	// 打开服务管理器
	m, err := sm.OpenManager()
	if err != nil {
		return err
	}
	defer m.Disconnect()

	// 创建服务
	s, err := m.CreateService(
		sm.serviceName,
		sm.exePath,
		mgr.Config{
			DisplayName: sm.displayName,
			Description: sm.description,
			StartType:   mgr.StartAutomatic,
		},
	)
	if err != nil {
		return pkgerrors.ErrService("failed to create service", err)
	}
	defer s.Close()

	return nil
}

// Uninstall 卸载 Windows 服务
func (sm *windowsServiceManager) Uninstall() error {
	// 检查服务是否存在
	exists, err := sm.ServiceExists()
	if err != nil {
		return err
	}
	if !exists {
		return pkgerrors.ErrService("service "+sm.serviceName+" does not exist", nil)
	}

	// 打开服务管理器
	m, err := sm.OpenManager()
	if err != nil {
		return err
	}
	defer m.Disconnect()

	// 打开服务
	s, err := sm.OpenService(m)
	if err != nil {
		return err
	}
	defer s.Close()

	// 删除服务
	err = s.Delete()
	if err != nil {
		return pkgerrors.ErrService("failed to delete service", err)
	}

	return nil
}
