//go:build linux

package sysproxy

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	pkgerrors "github.com/kkkqkx123/mihomo-cli/pkg/errors"
)

const (
	// ProxyEnvFile 代理环境变量配置文件 (systemd environment.d)
	ProxyEnvFile = "/etc/environment.d/proxy.conf"
	// ProxyEnvFileFallback 备用配置文件 (/etc/environment)
	ProxyEnvFileFallback = "/etc/environment"
)

// linuxSysProxy Linux 系统代理管理器
type linuxSysProxy struct{}

// newLinuxSysProxy 创建新的 Linux 系统代理管理器
func newLinuxSysProxy() SysProxy {
	return &linuxSysProxy{}
}

// GetStatus 获取代理状态
func (sp *linuxSysProxy) GetStatus() (*ProxySettings, error) {
	settings := &ProxySettings{}

	// 优先读取环境变量
	if httpProxy := os.Getenv("HTTP_PROXY"); httpProxy != "" {
		settings.Enabled = true
		settings.Server = httpProxy
	} else if httpProxy := os.Getenv("http_proxy"); httpProxy != "" {
		settings.Enabled = true
		settings.Server = httpProxy
	}

	if noProxy := os.Getenv("NO_PROXY"); noProxy != "" {
		settings.BypassList = noProxy
	} else if noProxy := os.Getenv("no_proxy"); noProxy != "" {
		settings.BypassList = noProxy
	}

	// 如果环境变量为空，尝试读取配置文件
	if !settings.Enabled {
		// 尝试读取 systemd environment.d 配置
		if data, err := os.ReadFile(ProxyEnvFile); err == nil {
			parseProxyConfig(string(data), settings)
		}
	}

	// 仍然为空，尝试读取 /etc/environment
	if !settings.Enabled {
		if data, err := os.ReadFile(ProxyEnvFileFallback); err == nil {
			parseProxyConfig(string(data), settings)
		}
	}

	return settings, nil
}

// Enable 启用系统代理
func (sp *linuxSysProxy) Enable(server, bypassList string) error {
	// 构建环境变量内容
	content := fmt.Sprintf(
		"HTTP_PROXY=%s\n"+
			"HTTPS_PROXY=%s\n"+
			"http_proxy=%s\n"+
			"https_proxy=%s\n",
		server, server, server, server,
	)

	// 添加绕过列表
	if bypassList != "" {
		content += fmt.Sprintf("NO_PROXY=%s\nno_proxy=%s\n", bypassList, bypassList)
	}

	// 尝试写入 systemd environment.d 目录
	if err := writeProxyConfig(ProxyEnvFile, content); err == nil {
		return nil
	}

	// 回退到 /etc/environment
	if err := writeProxyConfig(ProxyEnvFileFallback, content); err != nil {
		return pkgerrors.ErrService("failed to write proxy config", err)
	}

	return nil
}

// Disable 禁用系统代理
func (sp *linuxSysProxy) Disable() error {
	// 删除 systemd environment.d 配置
	if err := removeProxyConfig(ProxyEnvFile); err != nil {
		return err
	}

	// 从 /etc/environment 中移除代理配置
	if err := removeFromEtcEnvironment(); err != nil {
		return pkgerrors.ErrService("failed to remove proxy from /etc/environment", err)
	}

	return nil
}

// IsSupported 检查当前平台是否支持系统代理管理
func (sp *linuxSysProxy) IsSupported() bool {
	return true
}

// writeProxyConfig 写入代理配置文件
func writeProxyConfig(path, content string) error {
	// 确保目录存在
	dir := filepath.Dir(path)
	if dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return err
		}
	}

	return os.WriteFile(path, []byte(content), 0644)
}

// removeProxyConfig 删除代理配置文件
func removeProxyConfig(path string) error {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return nil // 文件不存在，无需删除
	}
	return os.Remove(path)
}

// removeFromEtcEnvironment 从 /etc/environment 中移除代理相关配置
func removeFromEtcEnvironment() error {
	data, err := os.ReadFile(ProxyEnvFileFallback)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	lines := strings.Split(string(data), "\n")
	var newLines []string
	proxyKeys := map[string]bool{
		"HTTP_PROXY":  true,
		"HTTPS_PROXY": true,
		"http_proxy":  true,
		"https_proxy": true,
		"NO_PROXY":    true,
		"no_proxy":    true,
	}

	for _, line := range lines {
		// 跳过代理相关的行
		parts := strings.SplitN(line, "=", 2)
		if len(parts) >= 1 {
			key := strings.TrimSpace(parts[0])
			if proxyKeys[key] {
				continue
			}
		}
		newLines = append(newLines, line)
	}

	// 写回文件
	newContent := strings.Join(newLines, "\n")
	return os.WriteFile(ProxyEnvFileFallback, []byte(newContent), 0644)
}

// parseProxyConfig 解析代理配置文件
func parseProxyConfig(content string, settings *ProxySettings) {
	lines := strings.Split(content, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}

		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])

		// 移除可能的引号
		value = strings.Trim(value, "\"'")

		switch key {
		case "HTTP_PROXY", "http_proxy":
			settings.Enabled = true
			settings.Server = value
		case "NO_PROXY", "no_proxy":
			settings.BypassList = value
		}
	}
}
