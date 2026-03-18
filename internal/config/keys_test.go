package config

import (
	"testing"
)

func TestGetConfigKeyInfo(t *testing.T) {
	tests := []struct {
		name      string
		key       string
		wantFound bool
		wantType  ConfigKeyType
	}{
		{
			name:      "existing key - mode",
			key:       "mode",
			wantFound: true,
			wantType:  ConfigTypeString,
		},
		{
			name:      "existing key - allow-lan",
			key:       "allow-lan",
			wantFound: true,
			wantType:  ConfigTypeBool,
		},
		{
			name:      "existing key - port",
			key:       "port",
			wantFound: true,
			wantType:  ConfigTypeInt,
		},
		{
			name:      "existing key - tun",
			key:       "tun",
			wantFound: true,
			wantType:  ConfigTypeObject,
		},
		{
			name:      "non-existing key",
			key:       "non-existent-key",
			wantFound: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			info, found := GetConfigKeyInfo(tt.key)
			if found != tt.wantFound {
				t.Errorf("GetConfigKeyInfo() found = %v, want %v", found, tt.wantFound)
				return
			}
			if tt.wantFound && info.Type != tt.wantType {
				t.Errorf("GetConfigKeyInfo() type = %v, want %v", info.Type, tt.wantType)
			}
		})
	}
}

func TestIsConfigKeySupported(t *testing.T) {
	tests := []struct {
		name     string
		key      string
		expected bool
	}{
		{
			name:     "supported key - mode",
			key:      "mode",
			expected: true,
		},
		{
			name:     "supported key - tun.enable",
			key:      "tun.enable",
			expected: true,
		},
		{
			name:     "unsupported key",
			key:      "invalid-key",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if result := IsConfigKeySupported(tt.key); result != tt.expected {
				t.Errorf("IsConfigKeySupported() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestIsHotUpdateSupported(t *testing.T) {
	tests := []struct {
		name     string
		key      string
		expected bool
	}{
		{
			name:     "hot update supported - mode",
			key:      "mode",
			expected: true,
		},
		{
			name:     "hot update supported - allow-lan",
			key:      "allow-lan",
			expected: true,
		},
		{
			name:     "unsupported key",
			key:      "invalid-key",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if result := IsHotUpdateSupported(tt.key); result != tt.expected {
				t.Errorf("IsHotUpdateSupported() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestParseConfigValue(t *testing.T) {
	tests := []struct {
		name     string
		key      string
		value    string
		expected interface{}
		wantErr  bool
	}{
		{
			name:     "parse string value",
			key:      "mode",
			value:    "rule",
			expected: "rule",
			wantErr:  false,
		},
		{
			name:     "parse bool value - true",
			key:      "allow-lan",
			value:    "true",
			expected: true,
			wantErr:  false,
		},
		{
			name:     "parse bool value - false",
			key:      "allow-lan",
			value:    "false",
			expected: false,
			wantErr:  false,
		},
		{
			name:     "parse bool value - 1",
			key:      "allow-lan",
			value:    "1",
			expected: true,
			wantErr:  false,
		},
		{
			name:     "parse bool value - 0",
			key:      "allow-lan",
			value:    "0",
			expected: false,
			wantErr:  false,
		},
		{
			name:     "parse int value",
			key:      "port",
			value:    "7890",
			expected: 7890,
			wantErr:  false,
		},
		{
			name:    "parse object value - should error",
			key:     "tun",
			value:   "{}",
			wantErr: true,
		},
		{
			name:    "unsupported key",
			key:     "invalid-key",
			value:   "value",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ParseConfigValue(tt.key, tt.value)
			if tt.wantErr {
				if err == nil {
					t.Errorf("ParseConfigValue() expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Errorf("ParseConfigValue() unexpected error: %v", err)
				return
			}
			if result != tt.expected {
				t.Errorf("ParseConfigValue() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestParseBool(t *testing.T) {
	tests := []struct {
		name     string
		value    string
		expected bool
		wantErr  bool
	}{
		{"true", "true", true, false},
		{"True", "True", true, false},
		{"TRUE", "TRUE", true, false},
		{"1", "1", true, false},
		{"yes", "yes", true, false},
		{"on", "on", true, false},
		{"false", "false", false, false},
		{"False", "False", false, false},
		{"FALSE", "FALSE", false, false},
		{"0", "0", false, false},
		{"no", "no", false, false},
		{"off", "off", false, false},
		{"invalid", "invalid", false, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := parseBool(tt.value)
			if tt.wantErr {
				if err == nil {
					t.Errorf("parseBool() expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Errorf("parseBool() unexpected error: %v", err)
				return
			}
			if result != tt.expected {
				t.Errorf("parseBool() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestParseInt(t *testing.T) {
	tests := []struct {
		name     string
		value    string
		expected int
		wantErr  bool
	}{
		{"zero", "0", 0, false},
		{"positive", "123", 123, false},
		{"negative", "-456", -456, false},
		{"large", "65535", 65535, false},
		{"invalid", "abc", 0, true},
		{"float string", "12.34", 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := parseInt(tt.value)
			if tt.wantErr {
				if err == nil {
					t.Errorf("parseInt() expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Errorf("parseInt() unexpected error: %v", err)
				return
			}
			if result != tt.expected {
				t.Errorf("parseInt() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestListSupportedConfigKeys(t *testing.T) {
	keys := ListSupportedConfigKeys()
	if len(keys) == 0 {
		t.Error("ListSupportedConfigKeys() returned empty list")
	}

	// 验证返回的键都是有效的
	for _, key := range keys {
		if key.Key == "" {
			t.Error("ListSupportedConfigKeys() returned key with empty name")
		}
		if key.Type == "" {
			t.Error("ListSupportedConfigKeys() returned key with empty type")
		}
	}
}

func TestListHotUpdateConfigKeys(t *testing.T) {
	keys := ListHotUpdateConfigKeys()
	if len(keys) == 0 {
		t.Error("ListHotUpdateConfigKeys() returned empty list")
	}

	// 验证所有返回的键都支持热更新
	for _, key := range keys {
		if !key.HotUpdate {
			t.Errorf("ListHotUpdateConfigKeys() returned key %s that does not support hot update", key.Key)
		}
	}
}

func TestValidateConfigKey(t *testing.T) {
	tests := []struct {
		name    string
		key     string
		value   interface{}
		wantErr bool
	}{
		{
			name:    "valid string",
			key:     "mode",
			value:   "rule",
			wantErr: false,
		},
		{
			name:    "invalid string - wrong type",
			key:     "mode",
			value:   123,
			wantErr: true,
		},
		{
			name:    "valid bool",
			key:     "allow-lan",
			value:   true,
			wantErr: false,
		},
		{
			name:    "invalid bool - wrong type",
			key:     "allow-lan",
			value:   "true",
			wantErr: true,
		},
		{
			name:    "valid int",
			key:     "port",
			value:   7890,
			wantErr: false,
		},
		{
			name:    "valid int64",
			key:     "port",
			value:   int64(7890),
			wantErr: false,
		},
		{
			name:    "valid float64 as int",
			key:     "port",
			value:   float64(7890),
			wantErr: false,
		},
		{
			name:    "invalid float64 - not integer",
			key:     "port",
			value:   78.90,
			wantErr: true,
		},
		{
			name:    "invalid int - wrong type",
			key:     "port",
			value:   "7890",
			wantErr: true,
		},
		{
			name:    "valid object",
			key:     "tun",
			value:   map[string]interface{}{"enable": true},
			wantErr: false,
		},
		{
			name:    "invalid object - wrong type",
			key:     "tun",
			value:   "not an object",
			wantErr: true,
		},
		{
			name:    "unsupported key",
			key:     "invalid-key",
			value:   "value",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateConfigKey(tt.key, tt.value)
			if tt.wantErr && err == nil {
				t.Errorf("ValidateConfigKey() expected error, got nil")
			}
			if !tt.wantErr && err != nil {
				t.Errorf("ValidateConfigKey() unexpected error: %v", err)
			}
		})
	}
}
