package service

import (
	"os/exec"
	"path/filepath"
	"runtime"

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

// findMihomoExecutable 查找 Mihomo 可执行文件路径
func (sf *serviceFactory) findMihomoExecutable() (string, error) {
	// 优先在 PATH 中查找
	if exePath, err := exec.LookPath("mihomo"); err == nil {
		// 返回绝对路径
		if absPath, err := filepath.Abs(exePath); err == nil {
			return absPath, nil
		}
		return exePath, nil
	}

	// 在当前目录和父目录查找匹配 mihomo*.exe 模式的文件
	searchDirs := []string{".", ".."}

	for _, dir := range searchDirs {
		absDir, err := filepath.Abs(dir)
		if err != nil {
			continue
		}

		// 根据平台选择可执行文件模式
		var pattern string
		if runtime.GOOS == "windows" {
			pattern = "mihomo*.exe"
		} else {
			pattern = "mihomo*"
		}

		// 查找所有匹配的文件
		matches, err := filepath.Glob(filepath.Join(absDir, pattern))
		if err != nil {
			continue
		}

		if len(matches) == 0 {
			continue
		}

		// 过滤掉 mihomo-cli（当前项目的可执行文件）
		var filteredMatches []string
		cliName := "mihomo-cli"
		if runtime.GOOS == "windows" {
			cliName = "mihomo-cli.exe"
		}
		for _, match := range matches {
			if filepath.Base(match) != cliName {
				filteredMatches = append(filteredMatches, match)
			}
		}

		if len(filteredMatches) == 0 {
			continue
		}

		// 优先选择简单的 mihomo
		targetName := "mihomo"
		if runtime.GOOS == "windows" {
			targetName = "mihomo.exe"
		}
		for _, match := range filteredMatches {
			if filepath.Base(match) == targetName {
				return match, nil
			}
		}

		// 如果没有找到简单的 mihomo，返回第一个匹配的文件
		return filteredMatches[0], nil
	}

	return "", pkgerrors.ErrConfig("mihomo executable not found", nil)
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
