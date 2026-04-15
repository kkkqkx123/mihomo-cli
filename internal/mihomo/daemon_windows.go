//go:build windows

package mihomo

import (
	"context"
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
	if err := wdm.RedirectIO(cmd, logFile); err != nil {
		return err
	}

	// 启动进程
	if err := cmd.Start(); err != nil {
		return pkgerrors.ErrService("failed to start mihomo daemon", err)
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

// StopDaemon 停止守护进程
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

			timeout := 10 * time.Second
			checkInterval := 500 * time.Millisecond
			deadline := time.Now().Add(timeout)

			for time.Now().Before(deadline) {
				if !IsProcessRunning(pid) {
					output.Success("Process %d has gracefully exited", pid)
					wdm.CleanupPID()
					return nil
				}
				time.Sleep(checkInterval)
			}
			output.Warning("Process did not exit within timeout")
		} else {
			output.Warning("API shutdown failed: " + err.Error())
		}
		output.Warning("Using force kill")
	}

	// 执行强制关闭
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

		cmd.Stdout = logFH
		cmd.Stderr = logFH
		// 注意: 不关闭文件句柄，因为子进程需要继承这个句柄
		// 当子进程启动后，这个句柄会自动被子进程继承
		// 父进程退出时，子进程仍然持有这个句柄的引用
	} else {
		// 重定向到 NUL
		nullFile, err := os.OpenFile("NUL", os.O_RDWR, 0)
		if err != nil {
			return pkgerrors.ErrConfig("failed to open NUL", err)
		}
		defer nullFile.Close()

		cmd.Stdout = nullFile
		cmd.Stderr = nullFile
	}

	// 重定向 stdin 到 NUL
	nullFile, err := os.OpenFile("NUL", os.O_RDONLY, 0)
	if err != nil {
		return pkgerrors.ErrConfig("failed to open NUL for stdin", err)
	}
	defer nullFile.Close()
	cmd.Stdin = nullFile

	return nil
}

// GetDaemonManager 获取守护进程管理器（工厂函数）
func GetDaemonManager(
	config *DaemonConfig,
	pidFile, secret, apiAddr, execPath, configFile string,
) DaemonManager {
	return NewWindowsDaemonManager(config, pidFile, secret, apiAddr, execPath, configFile)
}