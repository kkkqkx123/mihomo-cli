//go:build darwin

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
	// ProxyEnvFile proxy environment variable config file
	ProxyEnvFile = "/etc/environment.d/proxy.conf"
	// ProxyEnvFileFallback fallback config file (/etc/environment)
	ProxyEnvFileFallback = "/etc/environment"
	// ProxyPlistFile macOS launchd plist file for proxy environment
	ProxyPlistFile = "/etc/profile.d/proxy.sh"
)

// darwinSysProxy macOS system proxy manager
type darwinSysProxy struct{}

// newPlatformSysProxy creates a new macOS system proxy manager
func newPlatformSysProxy() SysProxy {
	return &darwinSysProxy{}
}

// GetStatus gets the proxy status
func (sp *darwinSysProxy) GetStatus() (*ProxySettings, error) {
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
		// Try reading /etc/profile.d/proxy.sh
		if data, err := os.ReadFile(ProxyPlistFile); err == nil {
			parseDarwinProxyConfig(string(data), settings)
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
func (sp *darwinSysProxy) Enable(server, bypassList string) error {
	// Build shell export content for macOS
	content := fmt.Sprintf(
		"export HTTP_PROXY=%s\n"+
			"export HTTPS_PROXY=%s\n"+
			"export http_proxy=%s\n"+
			"export https_proxy=%s\n",
		server, server, server, server,
	)

	// Add bypass list
	if bypassList != "" {
		content += fmt.Sprintf("export NO_PROXY=%s\nexport no_proxy=%s\n", bypassList, bypassList)
	}

	// Try writing to /etc/profile.d/proxy.sh
	if err := writeProxyConfig(ProxyPlistFile, content); err == nil {
		output.Warning("Proxy settings have been saved to configuration file.")
		output.Println("Note: The current terminal session will not reflect these changes immediately.")
		output.Println("To apply the proxy settings to your current session, run:")
		output.Printf("  source /etc/profile.d/proxy.sh\n")
		output.Println("Or start a new terminal session.")
		return nil
	}

	// Fallback to /etc/environment
	if err := addToEtcEnvironment(content); err != nil {
		return pkgerrors.ErrService("failed to write proxy config", err)
	}

	output.Warning("Proxy settings have been saved to /etc/environment.")
	output.Println("Note: The current terminal session will not reflect these changes immediately.")
	output.Println("To apply the proxy settings to your current session, run:")
	output.Printf("  source /etc/environment\n")
	output.Println("Or start a new terminal session.")

	return nil
}

// Disable disables the system proxy
func (sp *darwinSysProxy) Disable() error {
	// Delete /etc/profile.d/proxy.sh
	if err := removeProxyConfig(ProxyPlistFile); err != nil {
		return err
	}

	// Remove proxy config from /etc/environment
	if err := removeFromEtcEnvironment(); err != nil {
		return pkgerrors.ErrService("failed to remove proxy from /etc/environment", err)
	}

	return nil
}

// IsSupported checks if the current platform supports system proxy management
func (sp *darwinSysProxy) IsSupported() bool {
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
			// Also handle "export KEY=value" format
			key = strings.TrimPrefix(key, "export ")
			key = strings.TrimSpace(key)
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
			key = strings.TrimPrefix(key, "export ")
			key = strings.TrimSpace(key)
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

// parseDarwinProxyConfig parses macOS proxy config file (with export prefix)
func parseDarwinProxyConfig(content string, settings *ProxySettings) {
	lines := strings.Split(content, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// Handle "export KEY=value" format
		line = strings.TrimPrefix(line, "export ")
		line = strings.TrimSpace(line)

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
