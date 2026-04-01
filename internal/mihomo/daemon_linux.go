//go:build linux

package mihomo

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"

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
	if err := ldm.RedirectIO(cmd, logFile); err != nil {
		return err
	}

	// 启动进程
	if err := cmd.Start(); err != nil {
		return pkgerrors.ErrService("failed to start mihomo daemon", err)
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

	// 强制关闭
	return ldm.ForceKillDaemon(pid)
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
	// Linux 使用 Setsid 和 Setpgid 创建独立进程组和会话
	// Setsid: 创建新会话，使进程脱离控制终端
	// Setpgid: 创建新进程组，使进程成为进程组组长
	// 这确保了进程完全独立于父进程，不会受到终端关闭的影响
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Setsid:  true, // 创建新会话
		Setpgid: true, // 创建新进程组
	}
	return nil
}

// RedirectIO 重定向标准输入输出
func (ldm *LinuxDaemonManager) RedirectIO(cmd *exec.Cmd, logFile string) error {
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
		// 重定向到 /dev/null
		devNull, err := os.OpenFile("/dev/null", os.O_RDWR, 0)
		if err != nil {
			return pkgerrors.ErrConfig("failed to open /dev/null", err)
		}
		defer devNull.Close()

		cmd.Stdout = devNull
		cmd.Stderr = devNull
	}

	// 重定向 stdin 到 /dev/null
	devNull, err := os.OpenFile("/dev/null", os.O_RDONLY, 0)
	if err != nil {
		return pkgerrors.ErrConfig("failed to open /dev/null for stdin", err)
	}
	defer devNull.Close()
	cmd.Stdin = devNull

	return nil
}

// GetDaemonManager 获取守护进程管理器（工厂函数）
func GetDaemonManager(
	config *DaemonConfig,
	pidFile, secret, apiAddr, execPath, configFile string,
) DaemonManager {
	return NewLinuxDaemonManager(config, pidFile, secret, apiAddr, execPath, configFile)
}