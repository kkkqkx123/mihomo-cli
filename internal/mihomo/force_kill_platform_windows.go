//go:build windows

package mihomo

import (
	"time"

	"github.com/kkkqkx123/mihomo-cli/internal/output"
)

// forceKillPlatform Windows 平台专用的进程终止实现
func forceKillPlatform(pid int, timeout time.Duration) error {
	output.Printf("Terminating process %d using taskkill...\n", pid)
	return forceKillWindows(pid)
}
