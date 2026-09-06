//go:build !windows

package mihomo

import (
	"fmt"
	"syscall"
	"time"

	pkgerrors "github.com/kkkqkx123/mihomo-cli/pkg/errors"
)

// forceKillPlatform Unix/Linux/macOS 平台专用的进程终止实现。
// 采用轮询等待确认退出：SIGKILL 发送成功即视为完成终止动作，
// 等待仅用于确认退出；进程已退出（ESRCH）不算错误。
// 不使用 os.FindProcess + proc.Wait()（Wait 仅对 Start 的子进程有效，
// 对 FindProcess 得到的非子进程返回 ECHILD 误报失败）。
func forceKillPlatform(pid int, timeout time.Duration) error {
	if !IsProcessRunning(pid) {
		return nil
	}

	// 优先 SIGKILL
	if err := syscall.Kill(pid, syscall.SIGKILL); err != nil && err != syscall.ESRCH {
		return pkgerrors.ErrService("failed to kill process", err)
	}

	// 轮询等待进程退出
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if !IsProcessRunning(pid) {
			return nil
		}
		time.Sleep(100 * time.Millisecond)
	}

	return pkgerrors.ErrService(fmt.Sprintf("wait for process exit timeout after %v", timeout), nil)
}
