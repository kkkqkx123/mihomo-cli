package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestGenerateRandomSecret(t *testing.T) {
	secret1, err := GenerateRandomSecret()
	if err != nil {
		t.Fatalf("GenerateRandomSecret() error: %v", err)
	}

	if secret1 == "" {
		t.Error("GenerateRandomSecret() returned empty string")
	}

	// SHA256 哈希应该是 64 个十六进制字符
	if len(secret1) != 64 {
		t.Errorf("GenerateRandomSecret() length = %d, want 64", len(secret1))
	}

	// 生成另一个密钥，验证它们不同
	secret2, err := GenerateRandomSecret()
	if err != nil {
		t.Fatalf("GenerateRandomSecret() second call error: %v", err)
	}

	if secret1 == secret2 {
		t.Error("GenerateRandomSecret() generated identical secrets (extremely unlikely)")
	}
}

func TestGetDefaultTomlConfig(t *testing.T) {
	cfg := GetDefaultTomlConfig()
	if cfg == nil {
		t.Fatal("GetDefaultTomlConfig() returned nil")
	}

	// 验证默认 API 配置
	if cfg.API.Address != "http://127.0.0.1:9090" {
		t.Errorf("GetDefaultTomlConfig() API.Address = %v, want http://127.0.0.1:9090", cfg.API.Address)
	}
	if cfg.API.Timeout != 10 {
		t.Errorf("GetDefaultTomlConfig() API.Timeout = %v, want 10", cfg.API.Timeout)
	}

	// 验证默认 Mihomo 配置
	if !cfg.Mihomo.Enabled {
		t.Error("GetDefaultTomlConfig() Mihomo.Enabled should be true")
	}
	if cfg.Mihomo.Executable != "mihomo.exe" {
		t.Errorf("GetDefaultTomlConfig() Mihomo.Executable = %v, want mihomo.exe", cfg.Mihomo.Executable)
	}
	if !cfg.Mihomo.AutoGenerateSecret {
		t.Error("GetDefaultTomlConfig() Mihomo.AutoGenerateSecret should be true")
	}
	if cfg.Mihomo.API.ExternalController != "127.0.0.1:9090" {
		t.Errorf("GetDefaultTomlConfig() Mihomo.API.ExternalController = %v, want 127.0.0.1:9090", cfg.Mihomo.API.ExternalController)
	}
	if cfg.Mihomo.Log.Level != "info" {
		t.Errorf("GetDefaultTomlConfig() Mihomo.Log.Level = %v, want info", cfg.Mihomo.Log.Level)
	}
}

func TestLoadTomlConfig(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.toml")

	// 测试加载不存在的文件 - 应返回默认配置
	cfg, err := LoadTomlConfig(configPath)
	if err != nil {
		t.Fatalf("LoadTomlConfig() for non-existent file error: %v", err)
	}
	if cfg == nil {
		t.Fatal("LoadTomlConfig() returned nil for non-existent file")
	}
	// 验证返回的是默认配置
	if cfg.API.Address != "http://127.0.0.1:9090" {
		t.Errorf("LoadTomlConfig() for non-existent file should return default config")
	}

	// 创建有效的 TOML 配置文件
	validToml := `[api]
address = "http://192.168.1.1:8080"
secret = "my-secret"
timeout = 30

[mihomo]
enabled = false
executable = "custom-mihomo.exe"
config_file = "/path/to/config.yaml"
auto_generate_secret = false
health_check_timeout = 5

[mihomo.api]
external_controller = "0.0.0.0:9090"
external_controller_tls = ""
external_controller_unix = ""
external_controller_pipe = ""

[mihomo.log]
level = "debug"
file = "/var/log/mihomo.log"
`
	if err := os.WriteFile(configPath, []byte(validToml), 0644); err != nil {
		t.Fatalf("Failed to create test TOML file: %v", err)
	}

	cfg, err = LoadTomlConfig(configPath)
	if err != nil {
		t.Fatalf("LoadTomlConfig() error: %v", err)
	}

	// 验证加载的配置
	if cfg.API.Address != "http://192.168.1.1:8080" {
		t.Errorf("LoadTomlConfig() API.Address = %v, want http://192.168.1.1:8080", cfg.API.Address)
	}
	if cfg.API.Secret != "my-secret" {
		t.Errorf("LoadTomlConfig() API.Secret = %v, want my-secret", cfg.API.Secret)
	}
	if cfg.API.Timeout != 30 {
		t.Errorf("LoadTomlConfig() API.Timeout = %v, want 30", cfg.API.Timeout)
	}
	if cfg.Mihomo.Enabled {
		t.Error("LoadTomlConfig() Mihomo.Enabled should be false")
	}
	if cfg.Mihomo.Executable != "custom-mihomo.exe" {
		t.Errorf("LoadTomlConfig() Mihomo.Executable = %v, want custom-mihomo.exe", cfg.Mihomo.Executable)
	}
}

func TestTomlConfig_Save(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.toml")

	cfg := &TomlConfig{
		API: APIConfig{
			Address: "http://127.0.0.1:9090",
			Secret:  "test-secret",
			Timeout: 15,
		},
		Mihomo: MihomoConfig{
			Enabled:            true,
			Executable:         "mihomo.exe",
			ConfigFile:         "/etc/mihomo/config.yaml",
			AutoGenerateSecret: false,
			HealthCheckTimeout: 5,
			API: MihomoAPIConfig{
				ExternalController: "127.0.0.1:9090",
			},
			Log: MihomoLogConfig{
				Level: "warning",
				File:  "/var/log/mihomo.log",
			},
		},
	}

	if err := cfg.Save(configPath); err != nil {
		t.Fatalf("Save() error: %v", err)
	}

	// 验证文件已创建
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		t.Error("Save() did not create config file")
	}

	// 重新加载并验证
	loadedCfg, err := LoadTomlConfig(configPath)
	if err != nil {
		t.Fatalf("LoadTomlConfig() after Save() error: %v", err)
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
	if loadedCfg.Mihomo.Executable != cfg.Mihomo.Executable {
		t.Errorf("Loaded Mihomo.Executable = %v, want %v", loadedCfg.Mihomo.Executable, cfg.Mihomo.Executable)
	}
	if loadedCfg.Mihomo.Log.Level != cfg.Mihomo.Log.Level {
		t.Errorf("Loaded Mihomo.Log.Level = %v, want %v", loadedCfg.Mihomo.Log.Level, cfg.Mihomo.Log.Level)
	}
}

func TestTomlConfig_SaveToNestedPath(t *testing.T) {
	tmpDir := t.TempDir()
	// 使用嵌套目录路径
	configPath := filepath.Join(tmpDir, "nested", "dir", "config.toml")

	cfg := GetDefaultTomlConfig()

	if err := cfg.Save(configPath); err != nil {
		t.Fatalf("Save() to nested path error: %v", err)
	}

	// 验证文件已创建
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		t.Error("Save() did not create config file in nested directory")
	}
}
