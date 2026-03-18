package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	pkgerrors "github.com/kkkqkx123/mihomo-cli/pkg/errors"
)

// Paths 统一管理所有路径
type Paths struct {
	BaseDir    string // 基础配置目录 (~/.config/.mihomo-cli)
	PIDDir     string // PID 文件目录
	BackupDir  string // 备份目录
	ConfigFile string // CLI 配置文件路径
}

// GetPaths 获取统一的路径配置
func GetPaths() (*Paths, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, pkgerrors.ErrConfig("failed to get user home directory", err)
	}

	baseDir := filepath.Join(home, ".config", ".mihomo-cli")

	return &Paths{
		BaseDir:    baseDir,
		PIDDir:     baseDir,
		BackupDir:  filepath.Join(baseDir, "backups"),
		ConfigFile: filepath.Join(baseDir, "config.yaml"),
	}, nil
}

// GetBaseDir 获取基础配置目录
func GetBaseDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", pkgerrors.ErrConfig("failed to get user home directory", err)
	}
	return filepath.Join(home, ".config", ".mihomo-cli"), nil
}

// GetPIDDir 获取 PID 文件目录
func GetPIDDir() (string, error) {
	return GetBaseDir()
}

// GetBackupDir 获取备份目录
func GetBackupDir() (string, error) {
	baseDir, err := GetBaseDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(baseDir, "backups"), nil
}

// GetDataDir 获取数据目录（用于存储审计日志、快照等）
func GetDataDir() (string, error) {
	baseDir, err := GetBaseDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(baseDir, "data"), nil
}

// GetPIDFilePath 获取 PID 文件路径（基于配置文件路径）
func GetPIDFilePath(configFile string) (string, error) {
	pidDir, err := GetPIDDir()
	if err != nil {
		return "", err
	}

	// 如果配置文件为空，使用默认名称
	if configFile == "" {
		return filepath.Join(pidDir, "mihomo.pid"), nil
	}

	// 根据配置文件路径生成唯一的 hash
	hash := generateConfigHash(configFile)
	return filepath.Join(pidDir, fmt.Sprintf("mihomo-%s.pid", hash)), nil
}

// EnsureDirExists 确保目录存在，如果不存在则创建
func EnsureDirExists(dir string) error {
	if err := os.MkdirAll(dir, 0755); err != nil {
		return pkgerrors.ErrConfig("failed to create directory", err)
	}
	return nil
}

// generateConfigHash 根据配置文件路径生成短 hash
func generateConfigHash(configFile string) string {
	// 使用配置文件的绝对路径作为输入
	absPath, err := filepath.Abs(configFile)
	if err != nil {
		absPath = configFile
	}

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