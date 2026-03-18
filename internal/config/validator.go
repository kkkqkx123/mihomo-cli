package config

import (
	"fmt"
	"os"
	"strings"

	"gopkg.in/yaml.v3"

	pkgerrors "github.com/kkkqkx123/mihomo-cli/pkg/errors"
)

// ConfigValidator 配置验证器
type ConfigValidator struct {
	configPath string
}

// NewConfigValidator 创建配置验证器
func NewConfigValidator(configPath string) *ConfigValidator {
	return &ConfigValidator{
		configPath: configPath,
	}
}

// MihomoYAMLConfig Mihomo YAML 配置结构
type MihomoYAMLConfig struct {
	Tun *TunConfig `yaml:"tun"`
}

// TunConfig TUN 配置
type TunConfig struct {
	Enable bool `yaml:"enable"`
}

// ValidateConfigSyntax 验证配置文件语法
func (cv *ConfigValidator) ValidateConfigSyntax() error {
	// 读取配置文件
	data, err := os.ReadFile(cv.configPath)
	if err != nil {
		return pkgerrors.ErrConfig("failed to read config file", err)
	}

	// 尝试解析 YAML
	var config interface{}
	if err := yaml.Unmarshal(data, &config); err != nil {
		return pkgerrors.ErrConfig("invalid YAML syntax: "+err.Error(), err)
	}

	// 验证必需的字段
	configMap, ok := config.(map[string]interface{})
	if !ok {
		return pkgerrors.ErrConfig("invalid config structure: expected YAML object", nil)
	}

	// 验证基本配置项
	requiredFields := []string{
		"mixed-port",
		"port",
		"socks-port",
		"external-controller",
	}

	for _, field := range requiredFields {
		if _, exists := configMap[field]; !exists {
			// 这不是致命错误，只是警告
			fmt.Printf("Warning: missing recommended field '%s'\n", field)
		}
	}

	// 检查代理组配置
	if proxyGroups, ok := configMap["proxy-groups"]; ok {
		groups, ok := proxyGroups.([]interface{})
		if !ok {
			return pkgerrors.ErrConfig("invalid proxy-groups format: expected array", nil)
		}

		for i, group := range groups {
			groupMap, ok := group.(map[string]interface{})
			if !ok {
				return pkgerrors.ErrConfig(fmt.Sprintf("invalid proxy-group[%d] format: expected object", i), nil)
			}

			if _, ok := groupMap["name"]; !ok {
				return pkgerrors.ErrConfig(fmt.Sprintf("proxy-group[%d] missing required field 'name'", i), nil)
			}
			if _, ok := groupMap["type"]; !ok {
				return pkgerrors.ErrConfig(fmt.Sprintf("proxy-group[%d] missing required field 'type'", i), nil)
			}
		}
	}

	// 检查规则配置
	if rules, ok := configMap["rules"]; ok {
		ruleList, ok := rules.([]interface{})
		if !ok {
			return pkgerrors.ErrConfig("invalid rules format: expected array", nil)
		}

		for i, rule := range ruleList {
			ruleStr, ok := rule.(string)
			if !ok {
				return pkgerrors.ErrConfig(fmt.Sprintf("invalid rule[%d] format: expected string", i), nil)
			}

			// 验证规则格式
			parts := strings.Split(ruleStr, ",")
			if len(parts) < 2 {
				return pkgerrors.ErrConfig(fmt.Sprintf("invalid rule[%d] format: expected at least 2 parts", i), nil)
			}
		}
	}

	return nil
}

// ValidateAndWarn 验证配置并发出警告
func (cv *ConfigValidator) ValidateAndWarn() error {
	// 读取配置文件
	data, err := os.ReadFile(cv.configPath)
	if err != nil {
		return pkgerrors.ErrConfig("failed to read config file", err)
	}

	// 解析配置
	var mihomoConfig MihomoYAMLConfig
	if err := yaml.Unmarshal(data, &mihomoConfig); err != nil {
		// 解析失败不返回错误，因为配置可能格式不标准
		// 但我们仍然尝试通过文本分析来检测高风险配置
		cv.warnByTextAnalysis(string(data))
		return nil
	}

	// 检查 TUN 配置
	if mihomoConfig.Tun != nil && mihomoConfig.Tun.Enable {
		cv.warnTunEnabled()
	}

	return nil
}

// warnByTextAnalysis 通过文本分析检测高风险配置
func (cv *ConfigValidator) warnByTextAnalysis(content string) {
	lowerContent := strings.ToLower(content)

	// 检查是否启用了 TUN 模式
	if strings.Contains(lowerContent, "tun:") && strings.Contains(lowerContent, "enable: true") {
		cv.warnTunEnabled()
	}

	// 检查是否启用了 TProxy 模式
	if strings.Contains(lowerContent, "tproxy-port:") {
		cv.warnTProxyEnabled()
	}
}

// warnTunEnabled 警告 TUN 模式已启用
func (cv *ConfigValidator) warnTunEnabled() {
	fmt.Println("=" + strings.Repeat("=", 60))
	fmt.Println("WARNING: TUN mode is enabled in the configuration file")
	fmt.Println("=" + strings.Repeat("=", 60))
	fmt.Println()
	fmt.Println("TUN mode creates a virtual network adapter and modifies the routing table.")
	fmt.Println("If the process is forcibly terminated or crashes, the following may remain:")
	fmt.Println("  - Virtual network adapter (TUN device)")
	fmt.Println("  - Modified routing table")
	fmt.Println("  - DNS redirect settings")
	fmt.Println()
	fmt.Println("Recovery suggestions:")
	fmt.Println("  1. Use graceful shutdown: press Ctrl+C or run 'mihomo-cli stop'")
	fmt.Println("  2. If process is forcefully terminated, manually clean up:")
	fmt.Println("     - Delete TUN network adapter (Windows: Network Adapter Settings)")
	fmt.Println("     - Clean up routing table")
	fmt.Println("  3. Restart the system (simplest and most reliable)")
	fmt.Println()
	fmt.Println("It is recommended to test the configuration before use:")
	fmt.Println("  1. Start Mihomo")
	fmt.Println("  2. Test network connectivity")
	fmt.Println("  3. Stop Mihomo gracefully")
	fmt.Println("  4. Verify system configuration is cleaned up")
	fmt.Println()
}

// warnTProxyEnabled 警告 TProxy 模式已启用
func (cv *ConfigValidator) warnTProxyEnabled() {
	fmt.Println("=" + strings.Repeat("=", 60))
	fmt.Println("WARNING: TProxy mode is enabled in the configuration file")
	fmt.Println("=" + strings.Repeat("=", 60))
	fmt.Println()
	fmt.Println("TProxy mode modifies iptables rules to implement transparent proxying.")
	fmt.Println("If the process is forcibly terminated or crashes, the following may remain:")
	fmt.Println("  - iptables rules")
	fmt.Println("  - Routing table modifications")
	fmt.Println()
	fmt.Println("Recovery suggestions:")
	fmt.Println("  1. Use graceful shutdown: press Ctrl+C or run 'mihomo-cli stop'")
	fmt.Println("  2. If process is forcefully terminated, manually clean up:")
	fmt.Println("     - Clean up iptables rules")
	fmt.Println("     - Clean up routing table")
	fmt.Println("  3. Restart the system (simplest and most reliable)")
	fmt.Println()
	fmt.Println("It is recommended to test the configuration before use:")
	fmt.Println("  1. Start Mihomo")
	fmt.Println("  2. Test network connectivity")
	fmt.Println("  3. Stop Mihomo gracefully")
	fmt.Println("  4. Verify system configuration is cleaned up")
	fmt.Println()
}