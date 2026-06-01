//go:build windows

package mihomo

import (
	"fmt"
	"os/exec"
	"time"

	"github.com/kkkqkx123/mihomo-cli/internal/output"
	pkgerrors "github.com/kkkqkx123/mihomo-cli/pkg/errors"
)

// forceKillPlatform Windows 平台专用的进程终止实现
// 使用 taskkill 命令可以跨会话终止 detached 进程
func forceKillPlatform(pid int, timeout time.Duration) error {
	output.Printf("Terminating process %d using taskkill...\n", pid)
	
	// 先检查进程是否存在
	if !IsProcessRunning(pid) {
		output.Info("Process %d already exited", pid)
		return nil
	}
	
	if err := forceKillWindows(pid); err != nil {
		return err
	}
	
	// 等待进程退出（使用 timeout 参数）
	done := make(chan error, 1)
	go func() {
		checkInterval := 100 * time.Millisecond
		deadline := time.Now().Add(timeout)
		
		for time.Now().Before(deadline) {
			if !IsProcessRunning(pid) {
				done <- nil
				return
			}
			time.Sleep(checkInterval)
		}
		done <- pkgerrors.ErrService(
			fmt.Sprintf("wait for process exit timeout after %v", timeout),
			nil,
		)
	}()
	
	return <-done
}

// forceKillWindows Windows 平台专用的进程终止实现
// 使用 taskkill 命令，可以跨会话终止 detached 进程
func forceKillWindows(pid int) error {
	// 策略 1: 使用 taskkill /F /T /PID（最可靠）
	output.Printf("Attempting to terminate process %d using taskkill...\n", pid)
	if err := killWithTaskkill(pid); err == nil {
		return nil
	}
	
	// taskkill 失败，记录警告但继续尝试其他方法
	output.Warning("taskkill failed, trying alternative method...")
	
	// 策略 2: 尝试通过 WMI 终止（需要管理员权限）
	// TODO: 可以实现 WMI 方法作为备选
	
	// 所有方法都失败，返回错误
	return pkgerrors.ErrService(
		fmt.Sprintf("all termination methods failed for process %d", pid),
		nil,
	)
}

// killWithTaskkill 使用 taskkill 命令终止进程
func killWithTaskkill(pid int) error {
	// 使用 taskkill /F /T /PID 强制终止进程及其子进程
	// /F - 强制终止
	// /T - 终止进程树（包括所有子进程）
	// /PID - 指定进程 ID
	cmd := exec.Command("taskkill", "/F", "/T", "/PID", fmt.Sprintf("%d", pid))
	
	// 执行命令，设置超时
	done := make(chan error, 1)
	go func() {
		done <- cmd.Run()
	}()

	select {
	case err := <-done:
		if err != nil {
			// taskkill 失败，记录详细错误信息
			output.Warning("taskkill command failed: %v", err)
			
			// 检查进程是否已被终止（可能部分成功）
			time.Sleep(100 * time.Millisecond)
			if !IsProcessRunning(pid) {
				output.Success("Process %d terminated despite taskkill error", pid)
				return nil // 进程已终止，忽略错误
			}
			
			// 进程仍在运行，返回详细错误
			return pkgerrors.ErrService(
				fmt.Sprintf("failed to terminate process %d (still running)", pid),
				err,
			)
		}
		output.Success("Process %d terminated successfully", pid)
		return nil

	case <-time.After(5 * time.Second):
		return pkgerrors.ErrService(
			fmt.Sprintf("taskkill timeout after 5 seconds for process %d", pid),
			nil,
		)
	}
}
