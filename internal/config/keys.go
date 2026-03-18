package config

import (
	"fmt"
	"strconv"
	"strings"

	pkgerrors "github.com/kkkqkx123/mihomo-cli/pkg/errors"
)

// ConfigKeyType 配置键类型
type ConfigKeyType string

const (
	ConfigTypeString  ConfigKeyType = "string"
	ConfigTypeBool    ConfigKeyType = "bool"
	ConfigTypeInt     ConfigKeyType = "int"
	ConfigTypeObject  ConfigKeyType = "object"
)

// ConfigKeyInfo 配置键信息
type ConfigKeyInfo struct {
	Key         string        // 配置键名
	Type        ConfigKeyType // 值类型
	Description string        // 描述
	HotUpdate   bool          // 是否支持热更新
}

// SupportedConfigKeys 支持的配置键映射
var SupportedConfigKeys = map[string]ConfigKeyInfo{
	// 基础配置
	"mode": {
		Key:         "mode",
		Type:        ConfigTypeString,
		Description: "运行模式 (rule/global/direct)",
		HotUpdate:   true,
	},
	"allow-lan": {
		Key:         "allow-lan",
		Type:        ConfigTypeBool,
		Description: "允许局域网连接",
		HotUpdate:   true,
	},
	"log-level": {
		Key:         "log-level",
		Type:        ConfigTypeString,
		Description: "日志级别 (silent/error/warning/info/debug)",
		HotUpdate:   true,
	},
	"ipv6": {
		Key:         "ipv6",
		Type:        ConfigTypeBool,
		Description: "启用 IPv6",
		HotUpdate:   true,
	},
	"sniffing": {
		Key:         "sniffing",
		Type:        ConfigTypeBool,
		Description: "启用嗅探",
		HotUpdate:   true,
	},
	"tcp-concurrent": {
		Key:         "tcp-concurrent",
		Type:        ConfigTypeBool,
		Description: "启用 TCP 并发",
		HotUpdate:   true,
	},
	"find-process-mode": {
		Key:         "find-process-mode",
		Type:        ConfigTypeString,
		Description: "进程查找模式 (off/always/strict)",
		HotUpdate:   true,
	},
	"interface-name": {
		Key:         "interface-name",
		Type:        ConfigTypeString,
		Description: "绑定接口名称",
		HotUpdate:   true,
	},
	"bind-address": {
		Key:         "bind-address",
		Type:        ConfigTypeString,
		Description: "绑定地址",
		HotUpdate:   true,
	},

	// 端口配置
	"port": {
		Key:         "port",
		Type:        ConfigTypeInt,
		Description: "HTTP 代理端口",
		HotUpdate:   true,
	},
	"socks-port": {
		Key:         "socks-port",
		Type:        ConfigTypeInt,
		Description: "SOCKS5 代理端口",
		HotUpdate:   true,
	},
	"mixed-port": {
		Key:         "mixed-port",
		Type:        ConfigTypeInt,
		Description: "混合端口",
		HotUpdate:   true,
	},
	"redir-port": {
		Key:         "redir-port",
		Type:        ConfigTypeInt,
		Description: "透明代理端口",
		HotUpdate:   true,
	},
	"tproxy-port": {
		Key:         "tproxy-port",
		Type:        ConfigTypeInt,
		Description: "TPROXY 端口",
		HotUpdate:   true,
	},

	// TUN 配置
	"tun": {
		Key:         "tun",
		Type:        ConfigTypeObject,
		Description: "TUN 配置",
		HotUpdate:   true,
	},
	"tun.enable": {
		Key:         "tun.enable",
		Type:        ConfigTypeBool,
		Description: "启用 TUN",
		HotUpdate:   true,
	},
	"tun.device": {
		Key:         "tun.device",
		Type:        ConfigTypeString,
		Description: "TUN 设备名称",
		HotUpdate:   true,
	},
	"tun.stack": {
		Key:         "tun.stack",
		Type:        ConfigTypeString,
		Description: "TUN 协议栈 (system/gvisor/mixed)",
		HotUpdate:   true,
	},
	"tun.auto-route": {
		Key:         "tun.auto-route",
		Type:        ConfigTypeBool,
		Description: "自动路由",
		HotUpdate:   true,
	},
	"tun.auto-detect-interface": {
		Key:         "tun.auto-detect-interface",
		Type:        ConfigTypeBool,
		Description: "自动检测接口",
		HotUpdate:   true,
	},
	"tun.mtu": {
		Key:         "tun.mtu",
		Type:        ConfigTypeInt,
		Description: "MTU 大小",
		HotUpdate:   true,
	},

	// TUIC 服务器配置
	"tuic-server": {
		Key:         "tuic-server",
		Type:        ConfigTypeObject,
		Description: "TUIC 服务器配置",
		HotUpdate:   true,
	},
	"tuic-server.enable": {
		Key:         "tuic-server.enable",
		Type:        ConfigTypeBool,
		Description: "启用 TUIC 服务器",
		HotUpdate:   true,
	},
	"tuic-server.listen": {
		Key:         "tuic-server.listen",
		Type:        ConfigTypeString,
		Description: "TUIC 监听地址",
		HotUpdate:   true,
	},
}

// GetConfigKeyInfo 获取配置键信息
func GetConfigKeyInfo(key string) (ConfigKeyInfo, bool) {
	info, ok := SupportedConfigKeys[key]
	return info, ok
}

// IsConfigKeySupported 检查配置键是否支持
func IsConfigKeySupported(key string) bool {
	_, ok := SupportedConfigKeys[key]
	return ok
}

// IsHotUpdateSupported 检查配置键是否支持热更新
func IsHotUpdateSupported(key string) bool {
	info, ok := SupportedConfigKeys[key]
	if !ok {
		return false
	}
	return info.HotUpdate
}

// ParseConfigValue 解析配置值
func ParseConfigValue(key string, value string) (interface{}, error) {
	info, ok := SupportedConfigKeys[key]
	if !ok {
		return nil, pkgerrors.ErrInvalidArg(fmt.Sprintf("不支持的配置键: %s", key), nil)
	}

	switch info.Type {
	case ConfigTypeString:
		return value, nil
	case ConfigTypeBool:
		return parseBool(value)
	case ConfigTypeInt:
		return parseInt(value)
	case ConfigTypeObject:
		return nil, pkgerrors.ErrInvalidArg(fmt.Sprintf("配置键 %s 是对象类型，请使用 JSON 格式", key), nil)
	default:
		return nil, pkgerrors.ErrConfig(fmt.Sprintf("未知的配置类型: %s", info.Type), nil)
	}
}

// parseBool 解析布尔值
func parseBool(value string) (bool, error) {
	switch strings.ToLower(value) {
	case "true", "1", "yes", "on":
		return true, nil
	case "false", "0", "no", "off":
		return false, nil
	default:
		return false, pkgerrors.ErrInvalidArg(fmt.Sprintf("无效的布尔值: %s", value), nil)
	}
}

// parseInt 解析整数值
func parseInt(value string) (int, error) {
	val, err := strconv.Atoi(value)
	if err != nil {
		return 0, pkgerrors.ErrInvalidArg(fmt.Sprintf("无效的整数值: %s", value), nil)
	}
	return val, nil
}

// ListSupportedConfigKeys 列出所有支持的配置键
func ListSupportedConfigKeys() []ConfigKeyInfo {
	keys := make([]ConfigKeyInfo, 0, len(SupportedConfigKeys))
	for _, info := range SupportedConfigKeys {
		keys = append(keys, info)
	}
	return keys
}

// ListHotUpdateConfigKeys 列出支持热更新的配置键
func ListHotUpdateConfigKeys() []ConfigKeyInfo {
	keys := make([]ConfigKeyInfo, 0)
	for _, info := range SupportedConfigKeys {
		if info.HotUpdate {
			keys = append(keys, info)
		}
	}
	return keys
}

// ValidateConfigKey 验证配置键值
func ValidateConfigKey(key string, value interface{}) error {
	info, ok := SupportedConfigKeys[key]
	if !ok {
		return pkgerrors.ErrInvalidArg(fmt.Sprintf("不支持的配置键: %s", key), nil)
	}

	switch info.Type {
	case ConfigTypeString:
		if _, ok := value.(string); !ok {
			return pkgerrors.ErrInvalidArg(fmt.Sprintf("配置键 %s 需要字符串类型", key), nil)
		}
	case ConfigTypeBool:
		if _, ok := value.(bool); !ok {
			return pkgerrors.ErrInvalidArg(fmt.Sprintf("配置键 %s 需要布尔类型", key), nil)
		}
	case ConfigTypeInt:
		switch v := value.(type) {
		case int:
			// OK
		case int64:
			// OK
		case float64:
			// JSON 数字默认解析为 float64
			if v != float64(int(v)) {
				return pkgerrors.ErrInvalidArg(fmt.Sprintf("配置键 %s 需要整数类型", key), nil)
			}
		default:
			return pkgerrors.ErrInvalidArg(fmt.Sprintf("配置键 %s 需要整数类型", key), nil)
		}
	case ConfigTypeObject:
		if _, ok := value.(map[string]interface{}); !ok {
			return pkgerrors.ErrInvalidArg(fmt.Sprintf("配置键 %s 需要对象类型", key), nil)
		}
	}

	return nil
}
