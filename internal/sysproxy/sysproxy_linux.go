//go:build linux

package sysproxy

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/kkkqkx123/mihomo-cli/internal/output"
	pkgerrors "github.com/kkkqkx123/mihomo-cli/pkg/errors"
)

const (
	// ProxyEnvFile proxy environment variable config file (systemd environment.d)
	ProxyEnvFile = "/etc/environment.d/proxy.conf"
	// ProxyEnvFileFallback fallback config file (/etc/environment)
	ProxyEnvFileFallback = "/etc/environment"
)

// linuxSysProxy Linux system proxy manager
type linuxSysProxy struct{}

// newPlatformSysProxy creates a new Linux system proxy manager
func newPlatformSysProxy() SysProxy {
	return &linuxSysProxy{}
}

// GetStatus gets the proxy status
func (sp *linuxSysProxy) GetStatus() (*ProxySettings, error) {
	settings := &ProxySettings{}

	// First read environment variables
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

	// If environment variables are empty, try reading config files
	if !settings.Enabled {
		// Try reading systemd environment.d config
		if data, err := os.ReadFile(ProxyEnvFile); err == nil {
			parseProxyConfig(string(data), settings)
		}
	}

	// Still empty, try reading /etc/environment
	if !settings.Enabled {
		if data, err := os.ReadFile(ProxyEnvFileFallback); err == nil {
			parseProxyConfig(string(data), settings)
		}
	}

	return settings, nil
}

// Enable enables the system proxy
func (sp *linuxSysProxy) Enable(server, bypassList string) error {
	// Build environment variable content
	content := fmt.Sprintf(
		"HTTP_PROXY=%s\n"+
			"HTTPS_PROXY=%s\n"+
			"http_proxy=%s\n"+
			"https_proxy=%s\n",
		server, server, server, server,
	)

	// Add bypass list
	if bypassList != "" {
		content += fmt.Sprintf("NO_PROXY=%s\nno_proxy=%s\n", bypassList, bypassList)
	}

	// Try writing to systemd environment.d directory
	if err := writeProxyConfig(ProxyEnvFile, content); err == nil {
		// Warning: environment variables won't take effect immediately in current terminal session
		output.Warning("Proxy settings have been saved to configuration file.")
		output.Println("Note: The current terminal session will not reflect these changes immediately.")
		output.Println("To apply the proxy settings to your current session, run:")
		output.Printf("  source /etc/environment.d/proxy.conf\n")
		output.Println("Or start a new terminal session.")
		return nil
	}

	// Fallback to /etc/environment (safely add config, won't overwrite existing content)
	if err := addToEtcEnvironment(content); err != nil {
		return pkgerrors.ErrService("failed to write proxy config", err)
	}

	// Warning: environment variables won't take effect immediately in current terminal session
	output.Warning("Proxy settings have been saved to /etc/environment.")
	output.Println("Note: The current terminal session will not reflect these changes immediately.")
	output.Println("To apply the proxy settings to your current session, run:")
	output.Printf("  source /etc/environment\n")
	output.Println("Or start a new terminal session.")

	return nil
}

// Disable disables the system proxy
func (sp *linuxSysProxy) Disable() error {
	// Delete systemd environment.d config
	if err := removeProxyConfig(ProxyEnvFile); err != nil {
		return err
	}

	// Remove proxy config from /etc/environment
	if err := removeFromEtcEnvironment(); err != nil {
		return pkgerrors.ErrService("failed to remove proxy from /etc/environment", err)
	}

	return nil
}

// IsSupported checks if the current platform supports system proxy management
func (sp *linuxSysProxy) IsSupported() bool {
	return true
}

// writeProxyConfig writes proxy config file
func writeProxyConfig(path, content string) error {
	// Ensure directory exists
	dir := filepath.Dir(path)
	if dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return err
		}
	}

	return os.WriteFile(path, []byte(content), 0644)
}

// removeProxyConfig deletes proxy config file
func removeProxyConfig(path string) error {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return nil // File doesn't exist, no need to delete
	}
	return os.Remove(path)
}

// removeFromEtcEnvironment removes proxy related config from /etc/environment
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
		// Skip proxy related lines
		parts := strings.SplitN(line, "=", 2)
		if len(parts) >= 1 {
			key := strings.TrimSpace(parts[0])
			if proxyKeys[key] {
				continue
			}
		}
		newLines = append(newLines, line)
	}

	// Write back to file
	newContent := strings.Join(newLines, "\n")
	return os.WriteFile(ProxyEnvFileFallback, []byte(newContent), 0644)
}

// addToEtcEnvironment safely adds proxy config to /etc/environment
// This function reads existing content, removes old proxy variables, then appends new proxy config
func addToEtcEnvironment(content string) error {
	data, err := os.ReadFile(ProxyEnvFileFallback)
	if err != nil && !os.IsNotExist(err) {
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
		// Remove old proxy related lines
		parts := strings.SplitN(line, "=", 2)
		if len(parts) >= 1 {
			key := strings.TrimSpace(parts[0])
			if proxyKeys[key] {
				continue
			}
		}
		newLines = append(newLines, line)
	}

	// Append new proxy config
	proxyLines := strings.Split(strings.TrimSpace(content), "\n")
	for _, line := range proxyLines {
		line = strings.TrimSpace(line)
		if line != "" {
			newLines = append(newLines, line)
		}
	}

	// Write back to file
	newContent := strings.Join(newLines, "\n")
	if string(data) != "" && !strings.HasSuffix(string(data), "\n") {
		newContent = strings.Join(newLines, "\n")
	}
	return os.WriteFile(ProxyEnvFileFallback, []byte(newContent), 0644)
}

// parseProxyConfig parses proxy config file
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

		// Remove possible quotes
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
