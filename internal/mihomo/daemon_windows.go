//go:build windows

package mihomo

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"syscall"
	"time"

	"github.com/kkkqkx123/mihomo-cli/internal/api"
	"github.com/kkkqkx123/mihomo-cli/internal/output"
	pkgerrors "github.com/kkkqkx123/mihomo-cli/pkg/errors"
)

// WindowsDaemonManager Windows 平台守护进程管理器
type WindowsDaemonManager struct {
	*DaemonManagerBase
}

// NewWindowsDaemonManager 创建 Windows 平台守护进程管理器
func NewWindowsDaemonManager(
	config *DaemonConfig,
	pidFile, secret, apiAddr, execPath, configFile string,
) *WindowsDaemonManager {
	return &WindowsDaemonManager{
		DaemonManagerBase: NewDaemonManagerBase(config, pidFile, secret, apiAddr, execPath, configFile),
	}
}

// StartAsDaemon 以守护进程方式启动
func (wdm *WindowsDaemonManager) StartAsDaemon(ctx context.Context, cfg interface{}) error {
	// 构建命令
	cmd := exec.Command(wdm.GetExecutablePath(), "-f", wdm.GetConfigFile())

	// 设置工作目录
	if workDir := wdm.GetWorkDir(); workDir != "" {
		cmd.Dir = workDir
	}

	// 创建进程组并隐藏窗口
	if err := wdm.CreateProcessGroup(cmd); err != nil {
		return err
	}

	// 重定向 I/O
	logFile := ""
	if wdm.GetConfig() != nil {
		logFile = wdm.GetConfig().LogFile
	}
	if err := wdm.RedirectIO(cmd, logFile); err != nil {
		return err
	}

	// 启动进程
	if err := cmd.Start(); err != nil {
		return pkgerrors.ErrService("failed to start mihomo daemon", err)
	}

	// 保存 PID
	pid := cmd.Process.Pid
	if err := wdm.savePID(pid); err != nil {
		output.Warning("failed to save PID file: " + err.Error())
	}

	output.Success("Mihomo daemon started successfully (PID: %d)", pid)
	return nil
}

// StopDaemon 停止守护进程
func (wdm *WindowsDaemonManager) StopDaemon(pid int) error {
	// 检查进程是否运行
	if !wdm.IsDaemonRunning(pid) {
		return pkgerrors.ErrService("daemon is not running", nil)
	}

	// 优先使用 API 优雅关闭
	apiAddr := wdm.GetAPIAddress()
	secret := wdm.GetSecret()

	if apiAddr != "" && secret != "" {
		output.Printf("Attempting to shutdown daemon via API...\n")

		// 使用 API 客户端关闭
		client := api.NewClient("http://"+apiAddr, secret, api.WithTimeout(10*time.Second))
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		if err := client.Shutdown(ctx); err == nil {
			// API 关闭成功，等待进程退出
			output.Printf("Waiting for process to exit (max 10 seconds)...\n")

			timeout := 10 * time.Second
			checkInterval := 500 * time.Millisecond
			checkTicker := time.NewTicker(checkInterval)
			defer checkTicker.Stop()

			deadline := time.Now().Add(timeout)

			for range checkTicker.C {
				if !IsProcessRunning(pid) {
					output.Success("Process %d has gracefully exited", pid)
					wdm.cleanupPID()
					return nil
				}

				if time.Now().After(deadline) {
					output.Warning("Process did not exit within timeout")
					break
				}
			}
		} else {
			output.Warning("API shutdown failed: " + err.Error())
		}
		output.Warning("Using force kill")
	}

	// 强制关闭
	return wdm.forceKill(pid)
}

// IsDaemonRunning 检查守护进程是否运行
func (wdm *WindowsDaemonManager) IsDaemonRunning(pid int) bool {
	if pid == 0 {
		// 从 PID 文件读取
		var err error
		pid, err = wdm.readPID()
		if err != nil {
			return false
		}
	}
	return IsProcessRunning(pid)
}

// GetDaemonPID 获取守护进程 PID
func (wdm *WindowsDaemonManager) GetDaemonPID() (int, error) {
	return wdm.readPID()
}

// CreateProcessGroup 创建进程组
func (wdm *WindowsDaemonManager) CreateProcessGroup(cmd *exec.Cmd) error {
	// Windows 使用 CREATE_NEW_PROCESS_GROUP 标志
	// 这样创建的进程不会收到父进程的 Ctrl+C 信号
	cmd.SysProcAttr = &syscall.SysProcAttr{
		CreationFlags: syscall.CREATE_NEW_PROCESS_GROUP,
		HideWindow:    true, // 隐藏窗口
	}
	return nil
}

// RedirectIO 重定向标准输入输出
func (wdm *WindowsDaemonManager) RedirectIO(cmd *exec.Cmd, logFile string) error {
	if logFile != "" {
		// 确保日志目录存在
		logDir := filepath.Dir(logFile)
		if err := os.MkdirAll(logDir, 0755); err != nil {
			return pkgerrors.ErrConfig("failed to create log directory", err)
		}

		// 重定向到日志文件
		logFH, err := os.OpenFile(logFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
		if err != nil {
			return pkgerrors.ErrConfig("failed to open log file", err)
		}
		// 注意：不关闭文件句柄，因为子进程需要使用它

		cmd.Stdout = logFH
		cmd.Stderr = logFH
	} else {
		// 重定向到 NUL（Windows 的 /dev/null）
		nul, err := os.OpenFile("NUL", os.O_RDWR, 0)
		if err != nil {
			return pkgerrors.ErrConfig("failed to open NUL", err)
		}
		defer nul.Close()

		cmd.Stdout = nul
		cmd.Stderr = nul
	}

	// 重定向 stdin 到 NUL
	nul, err := os.OpenFile("NUL", os.O_RDONLY, 0)
	if err != nil {
		return pkgerrors.ErrConfig("failed to open NUL for stdin", err)
	}
	defer nul.Close()

	cmd.Stdin = nul

	return nil
}

// savePID 保存 PID 到文件
func (wdm *WindowsDaemonManager) savePID(pid int) error {
	pidFile := wdm.GetPIDFile()
	if pidFile == "" {
		return nil
	}

	// 确保目录存在
	pidDir := filepath.Dir(pidFile)
	if err := os.MkdirAll(pidDir, 0755); err != nil {
		return pkgerrors.ErrConfig("failed to create PID directory", err)
	}

	data := []byte(strconv.Itoa(pid))
	if err := os.WriteFile(pidFile, data, 0644); err != nil {
		return pkgerrors.ErrConfig("failed to write PID file", err)
	}

	return nil
}

// readPID 从文件读取 PID
func (wdm *WindowsDaemonManager) readPID() (int, error) {
	pidFile := wdm.GetPIDFile()
	if pidFile == "" {
		return 0, pkgerrors.ErrConfig("PID file not configured", nil)
	}

	data, err := os.ReadFile(pidFile)
	if err != nil {
		return 0, pkgerrors.ErrConfig("failed to read PID file", err)
	}

	pid, err := strconv.Atoi(string(data))
	if err != nil {
		return 0, pkgerrors.ErrConfig("invalid PID format", err)
	}

	return pid, nil
}

// cleanupPID 清理 PID 文件
func (wdm *WindowsDaemonManager) cleanupPID() {
	pidFile := wdm.GetPIDFile()
	if pidFile != "" {
		os.Remove(pidFile)
	}
}

// forceKill 强制终止进程
func (wdm *WindowsDaemonManager) forceKill(pid int) error {
	output.Printf("Force killing daemon process %d...\n", pid)

	proc, err := os.FindProcess(pid)
	if err != nil {
		return pkgerrors.ErrService("failed to find process", err)
	}

	if err := proc.Kill(); err != nil {
		return pkgerrors.ErrService("failed to kill process", err)
	}

	// 等待进程退出
	state, err := proc.Wait()
	if err != nil {
		return pkgerrors.ErrService("failed to wait for process exit", err)
	}

	if !state.Exited() {
		return pkgerrors.ErrService("process did not exit as expected", nil)
	}

	output.Success("Daemon process %d has been killed", pid)
	wdm.cleanupPID()

	return nil
}

// WindowsJobObjectManager Windows Job Object 管理器（可选的高级功能）
// 用于更严格的进程组管理和资源控制
type WindowsJobObjectManager struct {
	jobHandle syscall.Handle
}

// NewWindowsJobObjectManager 创建 Job Object 管理器
func NewWindowsJobObjectManager() *WindowsJobObjectManager {
	return &WindowsJobObjectManager{}
}

// CreateJobObject 创建 Job Object
func (wjom *WindowsJobObjectManager) CreateJobObject() error {
	// 注意：这里需要调用 Windows API 创建 Job Object
	// 为了简化，这里只是占位符
	// 实际实现需要使用 syscall 调用 CreateJobObject
	return nil
}

// AssignProcessToJobObject 将进程分配到 Job Object
func (wjom *WindowsJobObjectManager) AssignProcessToJobObject(pid int) error {
	// 注意：这里需要调用 Windows API 分配进程
	// 为了简化，这里只是占位符
	// 实际实现需要使用 syscall 调用 AssignProcessToJobObject
	return nil
}

// SetJobObjectLimit 设置 Job Object 限制
func (wjom *WindowsJobObjectManager) SetJobObjectLimit(limitType uint32, limitData uintptr) error {
	// 注意：这里需要调用 Windows API 设置限制
	// 为了简化，这里只是占位符
	// 实际实现需要使用 syscall 调用 SetInformationJobObject
	return nil
}

// Close 关闭 Job Object
func (wjom *WindowsJobObjectManager) Close() error {
	if wjom.jobHandle != 0 {
		_ = syscall.CloseHandle(wjom.jobHandle)
		wjom.jobHandle = 0
	}
	return nil
}

// GetDaemonManager 获取守护进程管理器（工厂函数）
func GetDaemonManager(
	config *DaemonConfig,
	pidFile, secret, apiAddr, execPath, configFile string,
) DaemonManager {
	return NewWindowsDaemonManager(config, pidFile, secret, apiAddr, execPath, configFile)
}
