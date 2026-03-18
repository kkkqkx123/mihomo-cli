package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestNewEditor(t *testing.T) {
	path := "/tmp/test-config.yaml"
	editor := NewEditor(path)
	if editor == nil {
		t.Fatal("NewEditor() returned nil")
	}
	if editor.configPath != path {
		t.Errorf("NewEditor() configPath = %v, want %v", editor.configPath, path)
	}
}

func TestEditor_SetBackupDir(t *testing.T) {
	editor := NewEditor("/tmp/test-config.yaml")
	backupDir := "/tmp/backups"
	editor.SetBackupDir(backupDir)
	if editor.backupDir != backupDir {
		t.Errorf("SetBackupDir() backupDir = %v, want %v", editor.backupDir, backupDir)
	}
}

func TestEditor_GetConfigPath(t *testing.T) {
	path := "/tmp/test-config.yaml"
	editor := NewEditor(path)
	if editor.GetConfigPath() != path {
		t.Errorf("GetConfigPath() = %v, want %v", editor.GetConfigPath(), path)
	}
}

func TestEditor_ReadConfig(t *testing.T) {
	// 创建临时测试文件
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "test.yaml")

	// 测试读取不存在的文件
	editor := NewEditor(configPath)
	_, err := editor.ReadConfig()
	if err == nil {
		t.Error("ReadConfig() expected error for non-existent file, got nil")
	}

	// 创建有效的 YAML 文件
	yamlContent := `mode: rule
allow-lan: true
port: 7890
`
	if err := os.WriteFile(configPath, []byte(yamlContent), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	config, err := editor.ReadConfig()
	if err != nil {
		t.Fatalf("ReadConfig() error: %v", err)
	}

	if config["mode"] != "rule" {
		t.Errorf("ReadConfig() mode = %v, want rule", config["mode"])
	}
	if config["allow-lan"] != true {
		t.Errorf("ReadConfig() allow-lan = %v, want true", config["allow-lan"])
	}
}

func TestEditor_BackupConfig(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "test.yaml")

	// 创建测试文件
	yamlContent := `mode: rule`
	if err := os.WriteFile(configPath, []byte(yamlContent), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	editor := NewEditor(configPath)
	editor.SetBackupDir(tmpDir)

	backupPath, err := editor.BackupConfig()
	if err != nil {
		t.Fatalf("BackupConfig() error: %v", err)
	}

	// 验证备份文件存在
	if _, err := os.Stat(backupPath); os.IsNotExist(err) {
		t.Errorf("Backup file not created at %s", backupPath)
	}

	// 验证备份文件内容
	backupData, err := os.ReadFile(backupPath)
	if err != nil {
		t.Fatalf("Failed to read backup file: %v", err)
	}
	if string(backupData) != yamlContent {
		t.Errorf("Backup content = %v, want %v", string(backupData), yamlContent)
	}
}

func TestEditor_WriteConfig(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "test.yaml")

	editor := NewEditor(configPath)

	config := map[string]interface{}{
		"mode":      "rule",
		"allow-lan": true,
		"port":      7890,
	}

	if err := editor.WriteConfig(config); err != nil {
		t.Fatalf("WriteConfig() error: %v", err)
	}

	// 验证文件已创建
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		t.Error("Config file not created")
	}

	// 读取并验证内容
	readConfig, err := editor.ReadConfig()
	if err != nil {
		t.Fatalf("ReadConfig() error: %v", err)
	}

	if readConfig["mode"] != "rule" {
		t.Errorf("Written config mode = %v, want rule", readConfig["mode"])
	}
}

func TestEditor_MergeConfig(t *testing.T) {
	editor := NewEditor("/tmp/test.yaml")

	base := map[string]interface{}{
		"mode":      "rule",
		"allow-lan": false,
		"port":      7890,
	}

	updates := map[string]interface{}{
		"allow-lan": true,
		"log-level": "debug",
	}

	result := editor.MergeConfig(base, updates)

	// 验证基础配置被保留
	if result["mode"] != "rule" {
		t.Errorf("MergeConfig() mode = %v, want rule", result["mode"])
	}
	if result["port"] != 7890 {
		t.Errorf("MergeConfig() port = %v, want 7890", result["port"])
	}

	// 验证更新被应用
	if result["allow-lan"] != true {
		t.Errorf("MergeConfig() allow-lan = %v, want true", result["allow-lan"])
	}
	if result["log-level"] != "debug" {
		t.Errorf("MergeConfig() log-level = %v, want debug", result["log-level"])
	}
}

func TestEditor_Edit(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "test.yaml")

	// 创建初始配置
	yamlContent := `mode: rule
allow-lan: false
`
	if err := os.WriteFile(configPath, []byte(yamlContent), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	editor := NewEditor(configPath)
	editor.SetBackupDir(tmpDir)

	// 测试编辑配置（不备份）
	_, err := editor.Edit("allow-lan", true, true)
	if err != nil {
		t.Fatalf("Edit() error: %v", err)
	}

	// 验证配置已更新
	config, err := editor.ReadConfig()
	if err != nil {
		t.Fatalf("ReadConfig() error: %v", err)
	}

	if config["allow-lan"] != true {
		t.Errorf("Edit() allow-lan = %v, want true", config["allow-lan"])
	}

	// 测试编辑配置（带备份）
	_, err = editor.Edit("mode", "global", false)
	if err != nil {
		t.Fatalf("Edit() with backup error: %v", err)
	}

	config, err = editor.ReadConfig()
	if err != nil {
		t.Fatalf("ReadConfig() error: %v", err)
	}

	if config["mode"] != "global" {
		t.Errorf("Edit() mode = %v, want global", config["mode"])
	}
}

func TestEditor_EditMultiple(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "test.yaml")

	// 创建初始配置
	yamlContent := `mode: rule
allow-lan: false
port: 7890
`
	if err := os.WriteFile(configPath, []byte(yamlContent), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	editor := NewEditor(configPath)
	editor.SetBackupDir(tmpDir)

	updates := map[string]interface{}{
		"allow-lan": true,
		"mode":      "global",
	}

	_, err := editor.EditMultiple(updates, true)
	if err != nil {
		t.Fatalf("EditMultiple() error: %v", err)
	}

	config, err := editor.ReadConfig()
	if err != nil {
		t.Fatalf("ReadConfig() error: %v", err)
	}

	if config["allow-lan"] != true {
		t.Errorf("EditMultiple() allow-lan = %v, want true", config["allow-lan"])
	}
	if config["mode"] != "global" {
		t.Errorf("EditMultiple() mode = %v, want global", config["mode"])
	}
}

func TestEditor_EditNonExistentFile(t *testing.T) {
	editor := NewEditor("/non/existent/path/config.yaml")

	_, err := editor.Edit("mode", "rule", true)
	if err == nil {
		t.Error("Edit() expected error for non-existent file, got nil")
	}
}

func TestSetNestedValue(t *testing.T) {
	tests := []struct {
		name     string
		config   map[string]interface{}
		key      string
		value    interface{}
		expected interface{}
	}{
		{
			name:     "simple key",
			config:   map[string]interface{}{},
			key:      "mode",
			value:    "rule",
			expected: "rule",
		},
		{
			name:     "nested key - new",
			config:   map[string]interface{}{},
			key:      "tun.enable",
			value:    true,
			expected: true,
		},
		{
			name: "nested key - existing",
			config: map[string]interface{}{
				"tun": map[string]interface{}{
					"enable": false,
				},
			},
			key:      "tun.enable",
			value:    true,
			expected: true,
		},
		{
			name: "deeply nested key",
			config: map[string]interface{}{
				"tun": map[string]interface{}{
					"stack": "system",
				},
			},
			key:      "tun.mtu",
			value:     1500,
			expected: 1500,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			setNestedValue(tt.config, tt.key, tt.value)

			// 获取值进行验证
			parts := splitKey(tt.key)
			var current interface{} = tt.config
			for _, part := range parts[:len(parts)-1] {
				if m, ok := current.(map[string]interface{}); ok {
					current = m[part]
				}
			}
			if m, ok := current.(map[string]interface{}); ok {
				lastPart := parts[len(parts)-1]
				if m[lastPart] != tt.expected {
					t.Errorf("setNestedValue() = %v, want %v", m[lastPart], tt.expected)
				}
			}
		})
	}
}

// 辅助函数：分割键
func splitKey(key string) []string {
	result := []string{}
	current := ""
	for _, c := range key {
		if c == '.' {
			result = append(result, current)
			current = ""
		} else {
			current += string(c)
		}
	}
	if current != "" {
		result = append(result, current)
	}
	return result
}
