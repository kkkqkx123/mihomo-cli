# Cross-Platform Unified Architecture Design Proposal

## 1. Overview

This document outlines a design proposal to provide a unified cross-platform abstraction for the `service` and `sysproxy` modules, addressing architectural asymmetry issues inherent in the current conditional compilation approach.

## 2. Current Status Analysis

### 2.1 Current Implementation Approach

The current implementation relies on **conditional compilation + stub implementations**:

```
cmd/
├── service.go           // Full Windows implementation
├── service_other.go     // Shell for other platforms
├── sysproxy.go          // Full Windows implementation
└── sysproxy_other.go    // Shell for other platforms
```

### 2.2 Existing Issues

1.  **Architectural Asymmetry**: Non-Windows platforms only have shell commands, lacking a unified interface.
2.  **Difficult Extension**: Adding support for Linux systemd in the future would require refactoring.
3.  **Code Fragmentation**: Related logic is scattered across multiple files with conditional compilation tags.
4.  **Testing Difficulty**: Writing unified cross-platform tests is challenging.

## 3. Unified Abstraction Scheme Design

### 3.1 Core Design Principles

- **Separation of Interface and Implementation**: Define platform-agnostic interfaces.
- **Factory Pattern for Creation**: Select platform-specific implementations at runtime.
- **Graceful Degradation**: Return clear errors for unsupported platforms.
- **Extensibility**: Adding support for new platforms requires only implementing the interface.

### 3.2 Service Module Abstraction Design

#### 3.2.1 Interface Definition

```go
// internal/service/interface.go

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
```

#### 3.2.2 Platform Implementation Structure

```
internal/service/
├── interface.go           // Interface definition (Cross-platform)
├── factory.go             // Factory implementation (Cross-platform)
├── manager_windows.go     // Windows implementation
├── manager_linux.go       // Linux systemd implementation (Future)
├── manager_darwin.go      // macOS launchd implementation (Future)
└── manager_stub.go        // Stub implementation for unsupported platforms
```

#### 3.2.3 Windows Implementation Example

```go
// internal/service/manager_windows.go
//go:build windows

package service

import (
    "golang.org/x/sys/windows/svc/mgr"
    // ...
)

type windowsServiceManager struct {
    serviceName string
    displayName string
    description string
    exePath     string
}

func (sm *windowsServiceManager) Start(async bool) error {
    // Windows service start implementation
}

func (sm *windowsServiceManager) IsSupported() bool {
    return true
}

// ... Implementations for other methods
```

#### 3.2.4 Stub Implementation Example

```go
// internal/service/manager_stub.go
//go:build !windows && !linux && !darwin

package service

type stubServiceManager struct {
    serviceName string
}

func (sm *stubServiceManager) Start(async bool) error {
    return ErrPlatformNotSupported
}

func (sm *stubServiceManager) IsSupported() bool {
    return false
}

// ... Other methods return ErrPlatformNotSupported
```

### 3.3 SysProxy Module Abstraction Design

#### 3.3.1 Interface Definition

```go
// internal/sysproxy/interface.go

package sysproxy

import "errors"

// ErrPlatformNotSupported indicates that system proxy management is not supported.
var ErrPlatformNotSupported = errors.New("system proxy not supported on this platform")

// ProxySettings holds the configuration for system proxies.
type ProxySettings struct {
    Enabled    bool
    Server     string
    BypassList string
}

// SysProxy defines the interface for managing system proxies.
type SysProxy interface {
    // Get Proxy Status
    GetStatus() (*ProxySettings, error)

    // Configure Proxy
    Enable(server, bypassList string) error
    Disable() error

    // Platform Detection
    IsSupported() bool
}
```

#### 3.3.2 Platform Implementation Structure

```
internal/sysproxy/
├── interface.go           // Interface definition (Cross-platform)
├── proxy.go               // Factory function (Cross-platform)
├── windows.go             // Windows Registry implementation
├── linux.go               // Linux Environment Variables/GNOME implementation
├── darwin.go              // macOS networksetup implementation
└── stub.go                // Stub implementation for unsupported platforms
```

### 3.4 Unified Command Layer Implementation

#### 3.4.1 Service Command

```go
// cmd/service.go (Cross-platform)

package cmd

import (
    "fmt"
    "runtime"

    "github.com/spf13/cobra"
    "github.com/kkkqkx123/mihomo-cli/internal/service"
)

var serviceCmd = &cobra.Command{
    Use:   "service",
    Short: "Service Management",
    Long:  "Manage Mihomo system services.",
    RunE: func(cmd *cobra.Command, args []string) error {
        factory := service.NewServiceFactory()
        sm, err := factory.CreateServiceManager()
        if err != nil {
            return err
        }

        if !sm.IsSupported() {
            return fmt.Errorf("service command not supported on %s", runtime.GOOS)
        }

        return cmd.Help()
    },
}

// Sub-command implementation uses the ServiceManager interface
func runServiceStart(cmd *cobra.Command, args []string) error {
    factory := service.NewServiceFactory()
    sm, err := factory.CreateServiceManager()
    if err != nil {
        return err
    }

    if !sm.IsSupported() {
        return service.ErrPlatformNotSupported
    }

    return sm.Start(asyncMode)
}
```

## 4. Implementation Plan

### 4.1 Phase 1: Interface Definition

1.  Create `internal/service/interface.go`.
2.  Create `internal/sysproxy/interface.go`.
3.  Define all necessary interfaces and types.

### 4.2 Phase 2: Refactor Existing Implementations

1.  Convert `manager.go`'s `ServiceManager` to a struct implementing the interface.
2.  Create `manager_stub.go` for stub implementations.
3.  Refactor `factory.go` to return the interface type.

### 4.3 Phase 3: Command Layer Adaptation

1.  Update `cmd/service.go` to use the interface.
2.  Update `cmd/sysproxy.go` to use the interface.
3.  Remove conditional compilation tags; replace with runtime detection.

### 4.4 Phase 4: Testing and Verification

1.  Perform cross-platform compilation tests.
2.  Conduct functional testing on Windows.
3.  Test error message display on Linux/macOS.

## 5. Future Extensions

### 5.1 Linux systemd Support

```go
// internal/service/manager_linux.go
//go:build linux

package service

import "github.com/coreos/go-systemd/v22/sd-daemon"

type systemdServiceManager struct {
    unitName string
    exePath  string
}

func (sm *systemdServiceManager) Start(async bool) error {
    // systemctl start implementation
}
```

### 5.2 macOS launchd Support

```go
// internal/service/manager_darwin.go
//go:build darwin

package service

type launchdServiceManager struct {
    plistPath string
    exePath   string
}

func (sm *launchdServiceManager) Start(async bool) error {
    // launchctl load implementation
}
```

## 6. Summary of Advantages

| Aspect                     | Current Solution | Unified Abstraction Solution |
| :------------------------- | :--------------- | :--------------------------- |
| **Architectural Symmetry** | ❌ Asymmetric    | ✅ Symmetric                 |
| **Extensibility**          | ❌ Difficult     | ✅ Easy to Extend            |
| **Code Organization**      | ❌ Scattered     | ✅ Centralized               |
| **Testability**            | ❌ Difficult     | ✅ Easy to Test              |
| **Runtime Overhead**       | ✅ None          | ⚠️ Minimal (Interface Call)  |
| **Build Consistency**      | ❌ Poor          | ✅ Good                      |

## 7. Considerations

1.  **Backward Compatibility**: Maintain existing APIs unchanged during refactoring.
2.  **Error Handling**: Return clear error messages for unsupported platforms.
3.  **Permission Checks**: Unify permission checks during factory creation.
4.  **Logging**: Add platform detection logs to facilitate debugging.

## 8. References

- [Go Interface Design Best Practices](https://go.dev/doc/effective_go#interfaces)
- [Cross-Platform Go Project Layout](https://github.com/golang-standards/project-layout)
- [systemd Go Bindings](https://github.com/coreos/go-systemd)
