package config

import (
	"fmt"
	"os"
	"path/filepath"

	pkgerrors "github.com/kkkqkx123/mihomo-cli/pkg/errors"
)

// Paths 统一管理所有路径
type Paths struct {
	BaseDir    string // 基础配置目录 (~/.config/.mihomo-cli)
	PIDDir     string // PID 文件目录
	BackupDir  string // 备份目录
	HistoryDir string // 历史记录目录
	ConfigFile string // CLI 配置文件路径
}

// GetPaths 获取统一的路径配置
func GetPaths() (*Paths, error) {
	baseDir, err := GetBaseDir()
	if err != nil {
		return nil, err
	}

	return &Paths{
		BaseDir:    baseDir,
		PIDDir:     baseDir,
		BackupDir:  filepath.Join(baseDir, "backups"),
		HistoryDir: filepath.Join(baseDir, "history"),
		ConfigFile: filepath.Join(baseDir, "config.yaml"),
	}, nil
}

// GetBaseDir 获取基础配置目录（平台规范目录 os.UserConfigDir()/mihomo-cli）。
// - Windows: %AppData%\Roaming\mihomo-cli
// - macOS:   ~/Library/Application Support/mihomo-cli
// - Linux:   $XDG_CONFIG_HOME 或 ~/.config 下的 mihomo-cli
func GetBaseDir() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", pkgerrors.ErrConfig("failed to get user config directory", err)
	}
	return filepath.Join(dir, "mihomo-cli"), nil
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

// GetHistoryDir 获取历史记录目录
func GetHistoryDir() (string, error) {
	baseDir, err := GetBaseDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(baseDir, "history"), nil
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

	// 根据配置文件路径生成唯一的 identity
	identity := IdentityOf(configFile)
	return filepath.Join(pidDir, fmt.Sprintf("mihomo-%s.pid", identity)), nil
}

// EnsureDirExists 确保目录存在，如果不存在则创建
func EnsureDirExists(dir string) error {
	if err := os.MkdirAll(dir, 0755); err != nil {
		return pkgerrors.ErrConfig("failed to create directory", err)
	}
	return nil
}
