package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// Editor 配置文件编辑器
type Editor struct {
	configPath string
	backupDir  string
}

// NewEditor 创建配置文件编辑器
func NewEditor(configPath string) *Editor {
	return &Editor{
		configPath: configPath,
		backupDir:  filepath.Dir(configPath),
	}
}

// SetBackupDir 设置备份目录
func (e *Editor) SetBackupDir(dir string) {
	e.backupDir = dir
}

// ReadConfig 读取现有 YAML 配置文件
func (e *Editor) ReadConfig() (map[string]interface{}, error) {
	data, err := os.ReadFile(e.configPath)
	if err != nil {
		return nil, fmt.Errorf("读取配置文件失败: %w", err)
	}

	var config map[string]interface{}
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("解析配置文件失败: %w", err)
	}

	return config, nil
}

// BackupConfig 备份原配置文件
func (e *Editor) BackupConfig() (string, error) {
	return e.BackupConfigWithNote("")
}

// BackupConfigWithNote 备份原配置文件并添加备注
func (e *Editor) BackupConfigWithNote(note string) (string, error) {
	// 读取原配置文件
	data, err := os.ReadFile(e.configPath)
	if err != nil {
		return "", fmt.Errorf("读取配置文件失败: %w", err)
	}

	// 生成备份文件名
	timestamp := time.Now().Format("20060102-150405")
	baseName := filepath.Base(e.configPath)
	ext := filepath.Ext(baseName)
	nameWithoutExt := strings.TrimSuffix(baseName, ext)

	var backupName string
	if note != "" {
		// 清理备注中的特殊字符
		cleanNote := sanitizeNote(note)
		backupName = fmt.Sprintf("%s.%s.%s%s", nameWithoutExt, timestamp, cleanNote, ext)
	} else {
		backupName = fmt.Sprintf("%s.%s%s", nameWithoutExt, timestamp, ext)
	}
	backupPath := filepath.Join(e.backupDir, backupName)

	// 确保备份目录存在
	if err := os.MkdirAll(e.backupDir, 0755); err != nil {
		return "", fmt.Errorf("创建备份目录失败: %w", err)
	}

	// 写入备份文件
	if err := os.WriteFile(backupPath, data, 0644); err != nil {
		return "", fmt.Errorf("创建备份文件失败: %w", err)
	}

	return backupPath, nil
}

// WriteConfig 写入新配置文件
func (e *Editor) WriteConfig(config map[string]interface{}) error {
	data, err := yaml.Marshal(config)
	if err != nil {
		return fmt.Errorf("序列化配置失败: %w", err)
	}

	if err := os.WriteFile(e.configPath, data, 0644); err != nil {
		return fmt.Errorf("写入配置文件失败: %w", err)
	}

	return nil
}

// MergeConfig 合并用户指定的配置项
func (e *Editor) MergeConfig(base map[string]interface{}, updates map[string]interface{}) map[string]interface{} {
	result := make(map[string]interface{})

	// 复制基础配置
	for k, v := range base {
		result[k] = v
	}

	// 合并更新
	for k, v := range updates {
		result[k] = v
	}

	return result
}

// Edit 编辑配置文件
// key: 配置键，支持点分隔符如 "tun.enable"
// value: 配置值
// noBackup: 是否跳过备份
func (e *Editor) Edit(key string, value interface{}, noBackup bool) (string, error) {
	return e.EditWithNote(key, value, noBackup, "")
}

// EditWithNote 编辑配置文件并添加备份备注
// key: 配置键，支持点分隔符如 "tun.enable"
// value: 配置值
// noBackup: 是否跳过备份
// note: 备份备注
func (e *Editor) EditWithNote(key string, value interface{}, noBackup bool, note string) (string, error) {
	// 检查配置文件是否存在
	if _, err := os.Stat(e.configPath); os.IsNotExist(err) {
		return "", fmt.Errorf("配置文件不存在: %s", e.configPath)
	}

	// 读取现有配置
	config, err := e.ReadConfig()
	if err != nil {
		return "", err
	}

	// 备份原配置
	var backupPath string
	if !noBackup {
		backupPath, err = e.BackupConfigWithNote(note)
		if err != nil {
			return "", err
		}
	}

	// 设置配置值
	setNestedValue(config, key, value)

	// 写入新配置
	if err := e.WriteConfig(config); err != nil {
		return "", err
	}

	return backupPath, nil
}

// EditMultiple 批量编辑配置文件
func (e *Editor) EditMultiple(updates map[string]interface{}, noBackup bool) (string, error) {
	return e.EditMultipleWithNote(updates, noBackup, "")
}

// EditMultipleWithNote 批量编辑配置文件并添加备份备注
func (e *Editor) EditMultipleWithNote(updates map[string]interface{}, noBackup bool, note string) (string, error) {
	// 检查配置文件是否存在
	if _, err := os.Stat(e.configPath); os.IsNotExist(err) {
		return "", fmt.Errorf("配置文件不存在: %s", e.configPath)
	}

	// 读取现有配置
	config, err := e.ReadConfig()
	if err != nil {
		return "", err
	}

	// 备份原配置
	var backupPath string
	if !noBackup {
		backupPath, err = e.BackupConfigWithNote(note)
		if err != nil {
			return "", err
		}
	}

	// 设置配置值
	for key, value := range updates {
		setNestedValue(config, key, value)
	}

	// 写入新配置
	if err := e.WriteConfig(config); err != nil {
		return "", err
	}

	return backupPath, nil
}

// setNestedValue 设置嵌套配置值
// 支持点分隔符如 "tun.enable"
func setNestedValue(config map[string]interface{}, key string, value interface{}) {
	parts := strings.Split(key, ".")
	if len(parts) == 1 {
		config[key] = value
		return
	}

	// 处理嵌套键
	current := config
	for i := 0; i < len(parts)-1; i++ {
		part := parts[i]
		if next, ok := current[part]; ok {
			if nextMap, ok := next.(map[string]interface{}); ok {
				current = nextMap
			} else {
				// 如果不是 map，创建新的 map
				newMap := make(map[string]interface{})
				current[part] = newMap
				current = newMap
			}
		} else {
			// 如果键不存在，创建新的 map
			newMap := make(map[string]interface{})
			current[part] = newMap
			current = newMap
		}
	}

	// 设置最终值
	current[parts[len(parts)-1]] = value
}

// GetConfigPath 获取配置文件路径
func (e *Editor) GetConfigPath() string {
	return e.configPath
}
