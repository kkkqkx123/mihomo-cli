package service

import (
	"os"
	"path/filepath"
	"runtime"

	"github.com/kkkqkx123/mihomo-cli/internal/config"
	"github.com/kkkqkx123/mihomo-cli/internal/util"
	pkgerrors "github.com/kkkqkx123/mihomo-cli/pkg/errors"
)

const (
	defaultServiceName = "Mihomo"
	defaultDisplayName = "Mihomo Service"
	defaultDescription = "Mihomo Proxy Service"
)

// serviceFactory 服务工厂
type serviceFactory struct {
	serviceName string
	displayName string
	description string
}

// NewServiceFactory 创建服务工厂
func NewServiceFactory() ServiceFactory {
	return &serviceFactory{
		serviceName: defaultServiceName,
		displayName: defaultDisplayName,
		description: defaultDescription,
	}
}

// CreateServiceManager 创建服务管理器
func (sf *serviceFactory) CreateServiceManager() (ServiceManager, error) {
	// 检查管理员权限
	if !util.IsAdmin() {
		return nil, pkgerrors.ErrService("this operation requires administrator privileges, please run as administrator", nil)
	}

	// 获取当前可执行文件路径
	exePath, err := sf.findMihomoExecutable()
	if err != nil {
		return nil, pkgerrors.ErrConfig("failed to determine mihomo executable path", err)
	}

	// 根据平台创建对应的服务管理器
	switch runtime.GOOS {
	case "windows":
		return newWindowsServiceManager(
			sf.serviceName,
			sf.displayName,
			sf.description,
			exePath,
		), nil
	default:
		// 返回 stub 实现
		return &stubServiceManager{
			serviceName: sf.serviceName,
			displayName: sf.displayName,
			exePath:     exePath,
		}, nil
	}
}

// findMihomoExecutable 查找 Mihomo 可执行文件路径（复用 config.ExecutableResolver 统一解析规则）
func (sf *serviceFactory) findMihomoExecutable() (string, error) {
	// 服务安装场景没有 config.toml 显式配置，走"空配置 + 允许搜索"链：
	// LookPath("mihomo") → 平台默认候选目录（CLI 同目录、CWD、/usr/local/bin、/usr/bin）。
	cwd, err := os.Getwd()
	if err != nil {
		cwd = ""
	}
	resolver := config.NewExecutableResolver(cwd)
	exePath, _, err := resolver.Resolve("", nil)
	if err != nil {
		return "", pkgerrors.ErrConfig("failed to determine mihomo executable path", err)
	}
	return filepath.Clean(exePath), nil
}

// SetServiceName 设置服务名称
func (sf *serviceFactory) SetServiceName(name string) {
	sf.serviceName = name
}

// SetDisplayName 设置显示名称
func (sf *serviceFactory) SetDisplayName(name string) {
	sf.displayName = name
}

// SetDescription 设置描述
func (sf *serviceFactory) SetDescription(desc string) {
	sf.description = desc
}

// stubServiceManager is a stub implementation for unsupported platforms.
type stubServiceManager struct {
	serviceName string
	displayName string
	exePath     string
}

func (sm *stubServiceManager) Start(async bool) error {
	return ErrPlatformNotSupported
}

func (sm *stubServiceManager) Stop(async bool) error {
	return ErrPlatformNotSupported
}

func (sm *stubServiceManager) Install() error {
	return ErrPlatformNotSupported
}

func (sm *stubServiceManager) Uninstall() error {
	return ErrPlatformNotSupported
}

func (sm *stubServiceManager) Status() (ServiceStatus, error) {
	return StatusUnknown, ErrPlatformNotSupported
}

func (sm *stubServiceManager) GetServiceName() string {
	return sm.serviceName
}

func (sm *stubServiceManager) GetDisplayName() string {
	return sm.displayName
}

func (sm *stubServiceManager) GetExePath() string {
	return sm.exePath
}

func (sm *stubServiceManager) IsSupported() bool {
	return false
}
