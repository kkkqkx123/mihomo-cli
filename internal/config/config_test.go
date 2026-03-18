package config

import (
	"testing"
)

func TestAPIConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		config  APIConfig
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid config",
			config: APIConfig{
				Address: "http://127.0.0.1:9090",
				Secret:  "test-secret",
				Timeout: 10,
			},
			wantErr: false,
		},
		{
			name: "valid config with https",
			config: APIConfig{
				Address: "https://example.com:8080",
				Secret:  "",
				Timeout: 30,
			},
			wantErr: false,
		},
		{
			name: "empty address",
			config: APIConfig{
				Address: "",
				Secret:  "",
				Timeout: 10,
			},
			wantErr: true,
			errMsg:  "API address is required",
		},
		{
			name: "missing protocol scheme",
			config: APIConfig{
				Address: "127.0.0.1:9090",
				Secret:  "",
				Timeout: 10,
			},
			wantErr: true,
			errMsg:  "API address must start with http:// or https://",
		},
		{
			name: "invalid timeout - zero",
			config: APIConfig{
				Address: "http://127.0.0.1:9090",
				Secret:  "",
				Timeout: 0,
			},
			wantErr: true,
			errMsg:  "timeout must be between 1 and 300 seconds",
		},
		{
			name: "invalid timeout - too large",
			config: APIConfig{
				Address: "http://127.0.0.1:9090",
				Secret:  "",
				Timeout: 301,
			},
			wantErr: true,
			errMsg:  "timeout must be between 1 and 300 seconds",
		},
		{
			name: "valid timeout at boundary - 1",
			config: APIConfig{
				Address: "http://127.0.0.1:9090",
				Secret:  "",
				Timeout: 1,
			},
			wantErr: false,
		},
		{
			name: "valid timeout at boundary - 300",
			config: APIConfig{
				Address: "http://127.0.0.1:9090",
				Secret:  "",
				Timeout: 300,
			},
			wantErr: false,
		},
		{
			name: "address without port",
			config: APIConfig{
				Address: "http://localhost",
				Secret:  "",
				Timeout: 10,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if tt.wantErr {
				if err == nil {
					t.Errorf("APIConfig.Validate() expected error, got nil")
					return
				}
				if tt.errMsg != "" && err.Error() != tt.errMsg {
					t.Errorf("APIConfig.Validate() error = %v, wantErr %v", err.Error(), tt.errMsg)
				}
			} else {
				if err != nil {
					t.Errorf("APIConfig.Validate() unexpected error: %v", err)
				}
			}
		})
	}
}

func TestCLIConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		config  CLIConfig
		wantErr bool
	}{
		{
			name: "valid config",
			config: CLIConfig{
				API: APIConfig{
					Address: "http://127.0.0.1:9090",
					Secret:  "test",
					Timeout: 10,
				},
			},
			wantErr: false,
		},
		{
			name: "invalid API config",
			config: CLIConfig{
				API: APIConfig{
					Address: "",
					Secret:  "",
					Timeout: 10,
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if tt.wantErr && err == nil {
				t.Errorf("CLIConfig.Validate() expected error, got nil")
			}
			if !tt.wantErr && err != nil {
				t.Errorf("CLIConfig.Validate() unexpected error: %v", err)
			}
		})
	}
}

func TestGetDefaultConfig(t *testing.T) {
	cfg := GetDefaultConfig()
	if cfg == nil {
		t.Fatal("GetDefaultConfig() returned nil")
	}

	if cfg.API.Address != "http://127.0.0.1:9090" {
		t.Errorf("GetDefaultConfig() API.Address = %v, want http://127.0.0.1:9090", cfg.API.Address)
	}

	if cfg.API.Secret != "" {
		t.Errorf("GetDefaultConfig() API.Secret = %v, want empty string", cfg.API.Secret)
	}

	if cfg.API.Timeout != 10 {
		t.Errorf("GetDefaultConfig() API.Timeout = %v, want 10", cfg.API.Timeout)
	}

	// 验证默认配置是有效的
	if err := cfg.Validate(); err != nil {
		t.Errorf("GetDefaultConfig() returned invalid config: %v", err)
	}
}
