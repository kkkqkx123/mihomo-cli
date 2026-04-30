//go:build !windows

package mihomo

import (
	"fmt"
	"os"
	"time"

	pkgerrors "github.com/kkkqkx123/mihomo-cli/pkg/errors"
)

// forceKillPlatform Unix/Linux/macOS 平台专用的进程终止实现
func forceKillPlatform(pid int, timeout time.Duration) error {
	proc, err := os.FindProcess(pid)
	if err != nil {
		return pkgerrors.ErrService("failed to find process", err)
	}

	if err := proc.Kill(); err != nil {
		return pkgerrors.ErrService("failed to kill process", err)
	}

	// 等待进程退出
	done := make(chan error, 1)
	go func() {
		state, err := proc.Wait()
		if err != nil {
			done <- err
			return
		}
		if !state.Exited() {
			done <- pkgerrors.ErrService("process did not exit as expected", nil)
			return
		}
		done <- nil
	}()

	select {
	case err := <-done:
		return err
	case <-time.After(timeout):
		return pkgerrors.ErrService(fmt.Sprintf("wait for process exit timeout after %v", timeout), nil)
	}
}
