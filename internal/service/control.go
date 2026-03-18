//go:build windows

package service

import (
	"time"

	"golang.org/x/sys/windows/svc"
	"golang.org/x/sys/windows/svc/mgr"

	pkgerrors "github.com/kkkqkx123/mihomo-cli/pkg/errors"
)

// Start 启动服务
func (sm *windowsServiceManager) Start(async bool) error {
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

	// 检查当前状态
	status, err := s.Query()
	if err != nil {
		return pkgerrors.ErrService("failed to query service status", err)
	}

	if status.State == svc.Running {
		return pkgerrors.ErrService("service "+sm.serviceName+" is already running", nil)
	}

	// 启动服务
	err = s.Start()
	if err != nil {
		return pkgerrors.ErrService("failed to start service", err)
	}

	// 如果不是异步模式，等待服务启动完成
	if !async {
		err = sm.waitForStatus(s, svc.Running, 10*time.Second)
		if err != nil {
			return pkgerrors.ErrService("timeout waiting for service to start", err)
		}
	}

	return nil
}

// Stop 停止服务
func (sm *windowsServiceManager) Stop(async bool) error {
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

	// 检查当前状态
	status, err := s.Query()
	if err != nil {
		return pkgerrors.ErrService("failed to query service status", err)
	}

	if status.State == svc.Stopped {
		return pkgerrors.ErrService("service "+sm.serviceName+" is already stopped", nil)
	}

	// 停止服务
	_, err = s.Control(svc.Stop)
	if err != nil {
		return pkgerrors.ErrService("failed to stop service", err)
	}

	// 如果不是异步模式，等待服务停止完成
	if !async {
		err = sm.waitForStatus(s, svc.Stopped, 10*time.Second)
		if err != nil {
			return pkgerrors.ErrService("timeout waiting for service to stop", err)
		}
	}

	return nil
}

// Status 查询服务状态
func (sm *windowsServiceManager) Status() (ServiceStatus, error) {
	// 检查服务是否存在
	exists, err := sm.ServiceExists()
	if err != nil {
		return StatusUnknown, err
	}
	if !exists {
		return StatusNotInstalled, nil
	}

	// 打开服务管理器
	m, err := sm.OpenManager()
	if err != nil {
		return StatusUnknown, err
	}
	defer m.Disconnect()

	// 打开服务
	s, err := sm.OpenService(m)
	if err != nil {
		return StatusUnknown, err
	}
	defer s.Close()

	// 查询服务状态
	status, err := s.Query()
	if err != nil {
		return StatusUnknown, pkgerrors.ErrService("failed to query service status", err)
	}

	switch status.State {
	case svc.Running:
		return StatusRunning, nil
	case svc.Stopped:
		return StatusStopped, nil
	default:
		return StatusUnknown, nil
	}
}

// waitForStatus 等待服务达到指定状态
func (sm *windowsServiceManager) waitForStatus(s *mgr.Service, targetState svc.State, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)

	for time.Now().Before(deadline) {
		status, err := s.Query()
		if err != nil {
			return err
		}

		if status.State == targetState {
			return nil
		}

		time.Sleep(500 * time.Millisecond)
	}

	return pkgerrors.ErrService("timeout", nil)
}
