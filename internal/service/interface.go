package service

import "errors"

// ServiceStatus represents the status of a service.
type ServiceStatus string

const (
	StatusRunning      ServiceStatus = "running"
	StatusStopped      ServiceStatus = "stopped"
	StatusNotInstalled ServiceStatus = "not-installed"
	StatusUnknown      ServiceStatus = "unknown"
)

// ErrPlatformNotSupported indicates that service management is not supported on the current platform.
var ErrPlatformNotSupported = errors.New("service management not supported on this platform")

// ServiceManager defines the interface for managing system services.
type ServiceManager interface {
	// Lifecycle Management
	Start(async bool) error
	Stop(async bool) error
	Install() error
	Uninstall() error
	Status() (ServiceStatus, error)

	// Information Query
	GetServiceName() string
	GetDisplayName() string
	GetExePath() string

	// Platform Detection
	IsSupported() bool
}

// ServiceFactory defines the interface for creating service managers.
type ServiceFactory interface {
	CreateServiceManager() (ServiceManager, error)
	SetServiceName(name string)
	SetDisplayName(name string)
	SetDescription(desc string)
}
