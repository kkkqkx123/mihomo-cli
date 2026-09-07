//go:build linux

package mihomo

import (
	"context"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
	"time"

	"github.com/kkkqkx123/mihomo-cli/internal/output"
	pkgerrors "github.com/kkkqkx123/mihomo-cli/pkg/errors"
)

// LinuxDaemonManager Linux 平台守护进程管理器
type LinuxDaemonManager struct {
	*DaemonManagerCommon
}

// NewLinuxDaemonManager 创建 Linux 平台守护进程管理器
func NewLinuxDaemonManager(
	config *DaemonConfig,
	pidFile, secret, apiAddr, execPath, configFile string,
) *LinuxDaemonManager {
	base := NewDaemonManagerBase(config, pidFile, secret, apiAddr, execPath, configFile)
	return &LinuxDaemonManager{
		DaemonManagerCommon: NewDaemonManagerCommon(base),
	}
}

// StartAsDaemon 以守护进程方式启动
func (ldm *LinuxDaemonManager) StartAsDaemon(ctx context.Context, cfg interface{}) error {
	// 构建命令
	cmd := exec.Command(ldm.Base().GetExecutablePath(), "-f", ldm.Base().GetConfigFile())

	// 设置工作目录
	if workDir := ldm.Base().GetWorkDir(); workDir != "" {
		cmd.Dir = workDir
	}

	// 创建进程组和会话
	if err := ldm.CreateProcessGroup(cmd); err != nil {
		return err
	}

	// 重定向 I/O
	logFile := ""
	if ldm.Base().GetConfig() != nil {
		logFile = ldm.Base().GetConfig().LogFile
	}
	closers, err := ldm.RedirectIO(cmd, logFile)
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

	// 保存 PID
	if err := ldm.SavePID(cmd.Process.Pid); err != nil {
		output.Warning("failed to save PID file: " + err.Error())
	}

	output.Success("Mihomo daemon started successfully (PID: %d)", cmd.Process.Pid)
	return nil
}

// StopDaemon 停止守护进程
func (ldm *LinuxDaemonManager) StopDaemon(pid int) error {
	// 检查进程是否运行
	if !ldm.IsDaemonRunning(pid) {
		return pkgerrors.ErrService("daemon is not running", nil)
	}

	// 优先使用 API 优雅关闭
	apiAddr := ldm.Base().GetAPIAddress()
	secret := ldm.Base().GetSecret()

	if apiAddr != "" && secret != "" {
		output.Printf("Attempting to shutdown daemon via API...\n")
		if err := StopProcessByPID(pid, apiAddr, secret); err == nil {
			ldm.CleanupPID()
			return nil
		}
		output.Warning("API shutdown failed, using force kill")
	}

	// 执行分级终止：先 SIGTERM，等待 5 秒，再 SIGKILL
	if err := ldm.GracefulKillDaemon(pid); err != nil {
		return err
	}

	ldm.CleanupPID()
	return nil
}

// IsDaemonRunning 检查守护进程是否运行
func (ldm *LinuxDaemonManager) IsDaemonRunning(pid int) bool {
	return ldm.DaemonManagerCommon.IsDaemonRunning(pid)
}

// GetDaemonPID 获取守护进程 PID
func (ldm *LinuxDaemonManager) GetDaemonPID() (int, error) {
	return ldm.DaemonManagerCommon.GetDaemonPID()
}

// CreateProcessGroup 创建进程组
func (ldm *LinuxDaemonManager) CreateProcessGroup(cmd *exec.Cmd) error {
	// Linux 使用 Setpgid 创建独立进程组
	// Setpgid: 创建新进程组，使进程成为进程组组长
	// 这确保了进程不会受到终端关闭的影响（SIGHUP 不会传递到新进程组）
	//
	// 注意：不使用 Setsid，因为在某些受限环境（容器、沙箱）中
	// Setsid + Setpgid 组合会导致 EPERM 错误。
	// Setpgid 足以让子进程脱离父进程的进程组，满足 daemon 需求。
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Setpgid: true,
	}
	return nil
}

// GracefulKillDaemon 分级终止守护进程：先 SIGTERM，等待 5 秒，再 SIGKILL
func (ldm *LinuxDaemonManager) GracefulKillDaemon(pid int) error {
	output.Printf("Sending SIGTERM to process %d...\n", pid)

	// 发送 SIGTERM 信号
	if err := syscall.Kill(pid, syscall.SIGTERM); err != nil {
		if err == syscall.ESRCH {
			output.Info("Process %d already exited", pid)
			return nil
		}
		return pkgerrors.ErrService("failed to send SIGTERM", err)
	}

	// 等待进程退出（最多 5 秒）
	timeout := 5 * time.Second
	checkInterval := 200 * time.Millisecond
	deadline := time.Now().Add(timeout)

	for time.Now().Before(deadline) {
		if !IsProcessRunning(pid) {
			output.Success("Process %d has gracefully exited", pid)
			return nil
		}
		time.Sleep(checkInterval)
	}

	// 如果进程仍未退出，发送 SIGKILL
	output.Warning("Process %d did not exit within %v, sending SIGKILL...", pid, timeout)
	if err := syscall.Kill(pid, syscall.SIGKILL); err != nil {
		if err == syscall.ESRCH {
			output.Info("Process %d already exited", pid)
			return nil
		}
		return pkgerrors.ErrService("failed to send SIGKILL", err)
	}

	output.Success("Process %d has been killed", pid)
	return nil
}

// RedirectIO 重定向标准输入输出。
// 返回需要在 cmd.Start() 之后由调用方关闭的父进程端文件句柄。
func (ldm *LinuxDaemonManager) RedirectIO(cmd *exec.Cmd, logFile string) ([]io.Closer, error) {
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
		// 重定向到 /dev/null
		devNull, err := os.OpenFile("/dev/null", os.O_RDWR, 0)
		if err != nil {
			return nil, pkgerrors.ErrConfig("failed to open /dev/null", err)
		}
		cmd.Stdout = devNull
		cmd.Stderr = devNull
		closers = append(closers, devNull)
	}

	// 重定向 stdin 到 /dev/null
	devNull, err := os.OpenFile("/dev/null", os.O_RDONLY, 0)
	if err != nil {
		return nil, pkgerrors.ErrConfig("failed to open /dev/null for stdin", err)
	}
	cmd.Stdin = devNull
	closers = append(closers, devNull)

	return closers, nil
}

// GetDaemonManager 获取守护进程管理器（工厂函数）
func GetDaemonManager(
	config *DaemonConfig,
	pidFile, secret, apiAddr, execPath, configFile string,
) DaemonManager {
	return NewLinuxDaemonManager(config, pidFile, secret, apiAddr, execPath, configFile)
}