//go:build windows

package mihomo

import (
	"context"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"golang.org/x/sys/windows"

	"github.com/kkkqkx123/mihomo-cli/internal/api"
	"github.com/kkkqkx123/mihomo-cli/internal/output"
	pkgerrors "github.com/kkkqkx123/mihomo-cli/pkg/errors"
)

// WindowsDaemonManager Windows 平台守护进程管理器
type WindowsDaemonManager struct {
	*DaemonManagerCommon
}

// NewWindowsDaemonManager 创建 Windows 平台守护进程管理器
func NewWindowsDaemonManager(
	config *DaemonConfig,
	pidFile, secret, apiAddr, execPath, configFile string,
) *WindowsDaemonManager {
	base := NewDaemonManagerBase(config, pidFile, secret, apiAddr, execPath, configFile)
	return &WindowsDaemonManager{
		DaemonManagerCommon: NewDaemonManagerCommon(base),
	}
}

// StartAsDaemon 以守护进程方式启动
func (wdm *WindowsDaemonManager) StartAsDaemon(ctx context.Context, cfg interface{}) error {
	// 构建命令
	cmd := exec.Command(wdm.Base().GetExecutablePath(), "-f", wdm.Base().GetConfigFile())

	// 设置工作目录
	if workDir := wdm.Base().GetWorkDir(); workDir != "" {
		cmd.Dir = workDir
	}

	// 创建进程组并隐藏窗口
	if err := wdm.CreateProcessGroup(cmd); err != nil {
		return err
	}

	// 重定向 I/O
	logFile := ""
	if wdm.Base().GetConfig() != nil {
		logFile = wdm.Base().GetConfig().LogFile
	}
	closers, err := wdm.RedirectIO(cmd, logFile)
	if err != nil {
		return err
	}

	// 启动进程
	if err := cmd.Start(); err != nil {
		// 关闭父进程端文件句柄（子进程已继承副本）
		for _, c := range closers {
			c.Close()
		}
		return pkgerrors.ErrService("failed to start mihomo daemon", err)
	}

	// 子进程已继承文件句柄，关闭父进程端副本以避免泄漏
	for _, c := range closers {
		c.Close()
	}

	pid := cmd.Process.Pid

	// 注意: 不再使用 Job Object
	// Job Object 的 JOB_OBJECT_LIMIT_KILL_ON_JOB_CLOSE 标志会导致父进程退出时子进程被终止
	// 这与守护进程的目标相反。我们使用 DETACHED_PROCESS 标志来确保进程独立运行。

	// 保存 PID
	if err := wdm.SavePID(pid); err != nil {
		output.Warning("failed to save PID file: " + err.Error())
	}

	output.Success("Mihomo daemon started successfully (PID: %d)", pid)
	return nil
}

// StopDaemon 停止守护进程。
// 三级停止策略：API 优雅关闭 → GenerateConsoleCtrlEvent（等效 SIGTERM）→ ForceKill。
func (wdm *WindowsDaemonManager) StopDaemon(pid int) error {
	// 检查进程是否运行
	if !wdm.IsDaemonRunning(pid) {
		return pkgerrors.ErrService("daemon is not running", nil)
	}

	// 优先使用 API 优雅关闭
	apiAddr := wdm.Base().GetAPIAddress()
	secret := wdm.Base().GetSecret()

	if apiAddr != "" && secret != "" {
		output.Printf("Attempting to shutdown daemon via API...\n")

		client := api.NewClient("http://"+apiAddr, secret, api.WithTimeout(10*time.Second))
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		if err := client.Shutdown(ctx); err == nil {
			output.Printf("Waiting for process to exit (max 10 seconds)...\n")

			if wdm.waitForExit(pid, 10*time.Second) {
				output.Success("Process %d has gracefully exited", pid)
				wdm.CleanupPID()
				return nil
			}
			output.Warning("Process did not exit within timeout")
		} else {
			output.Warning("API shutdown failed: " + err.Error())
		}
	}

	// 二级：发送 CTRL_C_EVENT（等效 Unix SIGTERM），等待进程自行退出
	output.Printf("Sending CTRL_C_EVENT to process %d...\n", pid)
	if wdm.sendCtrlC(pid) {
		if wdm.waitForExit(pid, 5*time.Second) {
			output.Success("Process %d has exited after CTRL_C_EVENT", pid)
			wdm.CleanupPID()
			return nil
		}
		output.Warning("Process did not exit after CTRL_C_EVENT")
	} else {
		output.Warning("Failed to send CTRL_C_EVENT")
	}

	// 三级：强制终止
	output.Warning("Using force kill")
	return wdm.ForceKillDaemon(pid)
}

// IsDaemonRunning 检查守护进程是否运行
func (wdm *WindowsDaemonManager) IsDaemonRunning(pid int) bool {
	return wdm.DaemonManagerCommon.IsDaemonRunning(pid)
}

// GetDaemonPID 获取守护进程 PID
func (wdm *WindowsDaemonManager) GetDaemonPID() (int, error) {
	return wdm.DaemonManagerCommon.GetDaemonPID()
}

// CreateProcessGroup 创建进程组
func (wdm *WindowsDaemonManager) CreateProcessGroup(cmd *exec.Cmd) error {
	// 使用 CREATE_NEW_PROCESS_GROUP | DETACHED_PROCESS 标志
	// CREATE_NEW_PROCESS_GROUP: 创建新的进程组，防止接收 Ctrl+C 信号
	// DETACHED_PROCESS: 创建独立的控制台进程，不继承父进程的控制台
	cmd.SysProcAttr = &windows.SysProcAttr{
		CreationFlags: windows.CREATE_NEW_PROCESS_GROUP | windows.DETACHED_PROCESS,
	}
	return nil
}

// RedirectIO 重定向标准输入输出。
// 返回需要在 cmd.Start() 之后由调用方关闭的父进程端文件句柄。
func (wdm *WindowsDaemonManager) RedirectIO(cmd *exec.Cmd, logFile string) ([]io.Closer, error) {
	var closers []io.Closer

	if logFile != "" {
		// 确保日志目录存在
		logDir := filepath.Dir(logFile)
		if err := os.MkdirAll(logDir, 0755); err != nil {
			return nil, pkgerrors.ErrConfig("failed to create log directory", err)
		}

		// 重定向到日志文件
		logFH, err := os.OpenFile(logFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
		if err != nil {
			return nil, pkgerrors.ErrConfig("failed to open log file", err)
		}

		cmd.Stdout = logFH
		cmd.Stderr = logFH
		closers = append(closers, logFH)
	} else {
		// 重定向到 NUL
		nullFile, err := os.OpenFile("NUL", os.O_RDWR, 0)
		if err != nil {
			return nil, pkgerrors.ErrConfig("failed to open NUL", err)
		}
		cmd.Stdout = nullFile
		cmd.Stderr = nullFile
		closers = append(closers, nullFile)
	}

	// 重定向 stdin 到 NUL
	nullFile, err := os.OpenFile("NUL", os.O_RDONLY, 0)
	if err != nil {
		return nil, pkgerrors.ErrConfig("failed to open NUL for stdin", err)
	}
	cmd.Stdin = nullFile
	closers = append(closers, nullFile)

	return closers, nil
}

// sendCtrlC 向指定 PID 的进程组发送 CTRL_C_EVENT（等效 Unix SIGTERM）。
// 子进程以 CREATE_NEW_PROCESS_GROUP 启动时，需使用该 PID 作为 dwProcessGroupId。
func (wdm *WindowsDaemonManager) sendCtrlC(pid int) bool {
	dll := windows.NewLazyDLL("kernel32.dll")
	proc := dll.NewProc("GenerateConsoleCtrlEvent")
	r, _, err := proc.Call(
		windows.CTRL_C_EVENT,
		uintptr(pid),
	)
	return r != 0 && err == nil
}

// waitForExit 轮询等待进程退出，超时返回 false。
func (wdm *WindowsDaemonManager) waitForExit(pid int, timeout time.Duration) bool {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if !IsProcessRunning(pid) {
			return true
		}
		time.Sleep(200 * time.Millisecond)
	}
	return false
}

// GetDaemonManager 获取守护进程管理器（工厂函数）
func GetDaemonManager(
	config *DaemonConfig,
	pidFile, secret, apiAddr, execPath, configFile string,
) DaemonManager {
	return NewWindowsDaemonManager(config, pidFile, secret, apiAddr, execPath, configFile)
}