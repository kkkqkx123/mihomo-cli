package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	pkgerrors "github.com/kkkqkx123/mihomo-cli/pkg/errors"
)

// PathResolver 统一的路径解析器
type PathResolver struct {
	baseDir string
}

// NewPathResolver 创建路径解析器
func NewPathResolver() (*PathResolver, error) {
	baseDir, err := GetBaseDir()
	if err != nil {
		return nil, err
	}

	return &PathResolver{
		baseDir: baseDir,
	}, nil
}

// GetBaseDir 获取基础配置目录
func (pr *PathResolver) GetBaseDir() string {
	return pr.baseDir
}

// GetAbsolutePath 将路径转换为绝对路径
func (pr *PathResolver) GetAbsolutePath(path string) string {
	if path == "" {
		return ""
	}
	absPath, err := filepath.Abs(path)
	if err != nil {
		return path
	}
	return absPath
}

// GetPIDDir 获取 PID 文件目录
func (pr *PathResolver) GetPIDDir() string {
	return pr.baseDir
}

// GetBackupDir 获取备份目录
func (pr *PathResolver) GetBackupDir() string {
	return filepath.Join(pr.baseDir, "backups")
}

// GetDataDir 获取数据目录
func (pr *PathResolver) GetDataDir() string {
	return filepath.Join(pr.baseDir, "data")
}

// GetHistoryDir 获取历史记录目录
func (pr *PathResolver) GetHistoryDir() string {
	return filepath.Join(pr.baseDir, "history")
}

// GetConfigFile 获取 CLI 配置文件路径
func (pr *PathResolver) GetConfigFile() string {
	return filepath.Join(pr.baseDir, "config.yaml")
}

// GetPIDFilePath 获取 PID 文件路径（基于配置文件路径）
func (pr *PathResolver) GetPIDFilePath(configFile string) string {
	// 如果配置文件为空，使用默认名称
	if configFile == "" {
		return filepath.Join(pr.GetPIDDir(), "mihomo.pid")
	}

	// 根据配置文件路径生成唯一的 hash
	hash := pr.generateConfigHash(configFile)
	return filepath.Join(pr.GetPIDDir(), fmt.Sprintf("mihomo-%s.pid", hash))
}

// GetStateFilePath 获取状态文件路径（基于配置文件路径）
func (pr *PathResolver) GetStateFilePath(configFile string) string {
	hash := pr.generateConfigHash(configFile)
	return filepath.Join(pr.baseDir, fmt.Sprintf("state-%s.json", hash))
}

// generateConfigHash 根据配置文件路径生成短 hash
func (pr *PathResolver) generateConfigHash(configFile string) string {
	// 使用配置文件的绝对路径作为输入
	absPath := pr.GetAbsolutePath(configFile)

	// 使用文件名作为简单的 hash（避免依赖 crypto 包）
	// 取文件名的最后部分，去除扩展名
	filename := filepath.Base(absPath)
	nameWithoutExt := strings.TrimSuffix(filename, filepath.Ext(filename))

	// 如果名称太长，截取前 8 个字符
	if len(nameWithoutExt) > 8 {
		nameWithoutExt = nameWithoutExt[:8]
	}

	// 如果名称为空，使用默认
	if nameWithoutExt == "" {
		nameWithoutExt = "default"
	}

	return nameWithoutExt
}

// EnsureDirExists 确保目录存在，如果不存在则创建
func (pr *PathResolver) EnsureDirExists(dir string) error {
	if err := os.MkdirAll(dir, 0755); err != nil {
		return pkgerrors.ErrConfig("failed to create directory", err)
	}
	return nil
}

// EnsureBaseDirs 确保所有基础目录存在
func (pr *PathResolver) EnsureBaseDirs() error {
	dirs := []string{
		pr.baseDir,
		pr.GetPIDDir(),
		pr.GetBackupDir(),
		pr.GetDataDir(),
		pr.GetHistoryDir(),
	}

	for _, dir := range dirs {
		if err := pr.EnsureDirExists(dir); err != nil {
			return err
		}
	}

	return nil
}
