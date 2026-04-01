//go:build linux

package mihomo

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"syscall"

	"github.com/kkkqkx123/mihomo-cli/internal/output"
	pkgerrors "github.com/kkkqkx123/mihomo-cli/pkg/errors"
)

// LinuxDaemonManager Linux 平台守护进程管理器
type LinuxDaemonManager struct {
	*DaemonManagerBase
}

// NewLinuxDaemonManager 创建 Linux 平台守护进程管理器
func NewLinuxDaemonManager(
	config *DaemonConfig,
	pidFile, secret, apiAddr, execPath, configFile string,
) *LinuxDaemonManager {
	return &LinuxDaemonManager{
		DaemonManagerBase: NewDaemonManagerBase(config, pidFile, secret, apiAddr, execPath, configFile),
	}
}

// StartAsDaemon 以守护进程方式启动
func (ldm *LinuxDaemonManager) StartAsDaemon(ctx context.Context, cfg interface{}) error {
	// 构建命令
	cmd := exec.Command(ldm.GetExecutablePath(), "-f", ldm.GetConfigFile())

	// 设置工作目录
	if workDir := ldm.GetWorkDir(); workDir != "" {
		cmd.Dir = workDir
	}

	// 创建进程组和会话
	if err := ldm.CreateProcessGroup(cmd); err != nil {
		return err
	}

	// 重定向 I/O
	logFile := ""
	if ldm.GetConfig() != nil {
		logFile = ldm.GetConfig().LogFile
	}
	if err := ldm.RedirectIO(cmd, logFile); err != nil {
		return err
	}

	// 启动进程
	if err := cmd.Start(); err != nil {
		return pkgerrors.ErrService("failed to start mihomo daemon", err)
	}

	// 保存 PID
	pid := cmd.Process.Pid
	if err := ldm.savePID(pid); err != nil {
		output.Warning("failed to save PID file: " + err.Error())
	}

	output.Success("Mihomo daemon started successfully (PID: %d)", pid)
	return nil
}

// StopDaemon 停止守护进程
func (ldm *LinuxDaemonManager) StopDaemon(pid int) error {
	// 检查进程是否运行
	if !ldm.IsDaemonRunning(pid) {
		return pkgerrors.ErrService("daemon is not running", nil)
	}

	// 优先使用 API 优雅关闭
	apiAddr := ldm.GetAPIAddress()
	secret := ldm.GetSecret()

	if apiAddr != "" && secret != "" {
		output.Printf("Attempting to shutdown daemon via API...\n")
		if err := StopProcessByPID(pid, apiAddr, secret); err == nil {
			// API 关闭成功，删除 PID 文件
			ldm.cleanupPID()
			return nil
		}
		output.Warning("API shutdown failed, using force kill")
	}

	// 强制关闭
	return ldm.forceKill(pid)
}

// IsDaemonRunning 检查守护进程是否运行
func (ldm *LinuxDaemonManager) IsDaemonRunning(pid int) bool {
	if pid == 0 {
		// 从 PID 文件读取
		var err error
		pid, err = ldm.readPID()
		if err != nil {
			return false
		}
	}
	return IsProcessRunning(pid)
}

// GetDaemonPID 获取守护进程 PID
func (ldm *LinuxDaemonManager) GetDaemonPID() (int, error) {
	return ldm.readPID()
}

// CreateProcessGroup 创建进程组
func (ldm *LinuxDaemonManager) CreateProcessGroup(cmd *exec.Cmd) error {
	// Linux 使用 Setsid 和 Setpgid 创建独立进程组和会话
	// 这样可以脱离控制终端，避免 SIGHUP 信号影响
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
		// 注意：不关闭文件句柄，因为子进程需要使用它

		cmd.Stdout = logFH
		cmd.Stderr = logFH
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

// savePID 保存 PID 到文件
func (ldm *LinuxDaemonManager) savePID(pid int) error {
	pidFile := ldm.GetPIDFile()
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
func (ldm *LinuxDaemonManager) readPID() (int, error) {
	pidFile := ldm.GetPIDFile()
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
func (ldm *LinuxDaemonManager) cleanupPID() {
	pidFile := ldm.GetPIDFile()
	if pidFile != "" {
		os.Remove(pidFile)
	}
}

// forceKill 强制终止进程
func (ldm *LinuxDaemonManager) forceKill(pid int) error {
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
	ldm.cleanupPID()

	return nil
}

// StartAsTraditionalDaemon 使用传统的 double-fork 方法启动守护进程
// 这是 Unix 系统的经典守护进程创建方式
func (ldm *LinuxDaemonManager) StartAsTraditionalDaemon(ctx context.Context) error {
	// 第一个 fork：创建子进程，父进程退出
	// 这会让子进程成为孤儿进程，被 init 进程收养
	// 同时子进程会获得一个新的会话 ID

	// 注意：Go 的 os/exec 包已经处理了大部分守护进程的细节
	// 这里我们使用现代的方法（Setsid）而不是传统的 double-fork
	// 因为 Go 的运行时会更好地处理进程管理

	// 如果需要传统的 double-fork，可以考虑以下方式：
	// 1. 第一个 fork：父进程退出
	// 2. 子进程调用 setsid() 创建新会话
	// 3. 第二个 fork：会话领队进程退出
	// 4. 第二个子进程继续执行

	// 但是对于我们的用例，Setsid 已经足够了
	return ldm.StartAsDaemon(ctx, nil)
}

// GetDaemonManager 获取守护进程管理器（工厂函数）
func GetDaemonManager(
	config *DaemonConfig,
	pidFile, secret, apiAddr, execPath, configFile string,
) DaemonManager {
	return NewLinuxDaemonManager(config, pidFile, secret, apiAddr, execPath, configFile)
}
