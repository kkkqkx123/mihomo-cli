package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestNewLoader(t *testing.T) {
	loader := NewLoader()
	if loader == nil {
		t.Fatal("NewLoader() returned nil")
	}
	if loader.v == nil {
		t.Error("NewLoader() viper instance is nil")
	}
}

func TestLoader_Load(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	// 测试加载不存在的文件
	loader := NewLoader()
	_, err := loader.Load(configPath)
	if err == nil {
		t.Error("Load() expected error for non-existent file, got nil")
	}

	// 创建有效的配置文件
	validConfig := `api:
  address: http://127.0.0.1:9090
  secret: test-secret
  timeout: 10
`
	if err := os.WriteFile(configPath, []byte(validConfig), 0644); err != nil {
		t.Fatalf("Failed to create test config file: %v", err)
	}

	cfg, err := loader.Load(configPath)
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}

	if cfg.API.Address != "http://127.0.0.1:9090" {
		t.Errorf("Load() API.Address = %v, want http://127.0.0.1:9090", cfg.API.Address)
	}
	if cfg.API.Secret != "test-secret" {
		t.Errorf("Load() API.Secret = %v, want test-secret", cfg.API.Secret)
	}
	if cfg.API.Timeout != 10 {
		t.Errorf("Load() API.Timeout = %v, want 10", cfg.API.Timeout)
	}
}

func TestLoader_LoadInvalidConfig(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	// 创建无效的配置文件（缺少必要字段）
	invalidConfig := `api:
  address: ""
  secret: ""
  timeout: 0
`
	if err := os.WriteFile(configPath, []byte(invalidConfig), 0644); err != nil {
		t.Fatalf("Failed to create test config file: %v", err)
	}

	loader := NewLoader()
	_, err := loader.Load(configPath)
	if err == nil {
		t.Error("Load() expected error for invalid config, got nil")
	}
}

func TestLoader_Save(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	loader := NewLoader()

	cfg := &CLIConfig{
		API: APIConfig{
			Address: "http://127.0.0.1:9090",
			Secret:  "test-secret",
			Timeout: 10,
		},
	}

	if err := loader.Save(cfg, configPath); err != nil {
		t.Fatalf("Save() error: %v", err)
	}

	// 验证文件已创建
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		t.Error("Save() did not create config file")
	}

	// 重新加载并验证
	loadedCfg, err := loader.Load(configPath)
	if err != nil {
		t.Fatalf("Load() after Save() error: %v", err)
	}

	if loadedCfg.API.Address != cfg.API.Address {
		t.Errorf("Loaded API.Address = %v, want %v", loadedCfg.API.Address, cfg.API.Address)
	}
	if loadedCfg.API.Secret != cfg.API.Secret {
		t.Errorf("Loaded API.Secret = %v, want %v", loadedCfg.API.Secret, cfg.API.Secret)
	}
	if loadedCfg.API.Timeout != cfg.API.Timeout {
		t.Errorf("Loaded API.Timeout = %v, want %v", loadedCfg.API.Timeout, cfg.API.Timeout)
	}
}

func TestLoader_SaveInvalidConfig(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	loader := NewLoader()

	// 无效配置（空地址）
	cfg := &CLIConfig{
		API: APIConfig{
			Address: "",
			Secret:  "",
			Timeout: 10,
		},
	}

	err := loader.Save(cfg, configPath)
	if err == nil {
		t.Error("Save() expected error for invalid config, got nil")
	}
}

func TestGetDefaultConfigPath(t *testing.T) {
	path, err := GetDefaultConfigPath()
	if err != nil {
		t.Fatalf("GetDefaultConfigPath() error: %v", err)
	}

	if path == "" {
		t.Error("GetDefaultConfigPath() returned empty path")
	}

	// 验证路径包含预期的目录
	if !filepath.IsAbs(path) {
		t.Errorf("GetDefaultConfigPath() returned relative path: %s", path)
	}
}
