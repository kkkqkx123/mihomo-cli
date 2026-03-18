package service

import (
	"os/exec"
	"path/filepath"

	"github.com/kkkqkx123/mihomo-cli/internal/util"
	pkgerrors "github.com/kkkqkx123/mihomo-cli/pkg/errors"
)

const (
	defaultServiceName = "Mihomo"
	defaultDisplayName = "Mihomo Service"
	defaultDescription = "Mihomo Proxy Service"
)

// ServiceFactory 服务工厂
type ServiceFactory struct {
	serviceName string
	displayName string
	description string
}

// NewServiceFactory 创建服务工厂
func NewServiceFactory() *ServiceFactory {
	return &ServiceFactory{
		serviceName: defaultServiceName,
		displayName: defaultDisplayName,
		description: defaultDescription,
	}
}

// CreateServiceManager 创建服务管理器
func (sf *ServiceFactory) CreateServiceManager() (*ServiceManager, error) {
	// 检查管理员权限
	if !util.IsAdmin() {
		return nil, pkgerrors.ErrService("this operation requires administrator privileges, please run as administrator", nil)
	}

	// 获取当前可执行文件路径
	exePath, err := sf.findMihomoExecutable()
	if err != nil {
		return nil, pkgerrors.ErrConfig("failed to determine mihomo executable path", err)
	}

	return NewServiceManager(
		sf.serviceName,
		sf.displayName,
		sf.description,
		exePath,
	), nil
}

// findMihomoExecutable 查找 Mihomo 可执行文件路径
func (sf *ServiceFactory) findMihomoExecutable() (string, error) {
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

		// 查找所有匹配 mihomo*.exe 的文件
		matches, err := filepath.Glob(filepath.Join(absDir, "mihomo*.exe"))
		if err != nil {
			continue
		}

		if len(matches) == 0 {
			continue
		}

		// 过滤掉 mihomo-cli.exe（当前项目的可执行文件）
		var filteredMatches []string
		for _, match := range matches {
			if filepath.Base(match) != "mihomo-cli.exe" {
				filteredMatches = append(filteredMatches, match)
			}
		}

		if len(filteredMatches) == 0 {
			continue
		}

		// 优先选择简单的 mihomo.exe
		for _, match := range filteredMatches {
			if filepath.Base(match) == "mihomo.exe" {
				return match, nil
			}
		}

		// 如果没有找到简单的 mihomo.exe，返回第一个匹配的文件
		// 按文件名排序，确保选择一致性
		// 通常文件名格式如 @mihomo-windows-amd64.exe，第一个应该是最合适的
		return filteredMatches[0], nil
	}

	return "", pkgerrors.ErrConfig("mihomo executable not found", nil)
}

// SetServiceName 设置服务名称
func (sf *ServiceFactory) SetServiceName(name string) {
	sf.serviceName = name
}

// SetDisplayName 设置显示名称
func (sf *ServiceFactory) SetDisplayName(name string) {
	sf.displayName = name
}

// SetDescription 设置描述
func (sf *ServiceFactory) SetDescription(desc string) {
	sf.description = desc
}
