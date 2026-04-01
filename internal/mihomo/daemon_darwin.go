//go:build darwin

package mihomo

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"

	"github.com/kkkqkx123/mihomo-cli/internal/output"
	pkgerrors "github.com/kkkqkx123/mihomo-cli/pkg/errors"
)

// DarwinDaemonManager macOS 平台守护进程管理器
type DarwinDaemonManager struct {
	*DaemonManagerCommon
}

// NewDarwinDaemonManager 创建 macOS 平台守护进程管理器
func NewDarwinDaemonManager(
	config *DaemonConfig,
	pidFile, secret, apiAddr, execPath, configFile string,
) *DarwinDaemonManager {
	base := NewDaemonManagerBase(config, pidFile, secret, apiAddr, execPath, configFile)
	return &DarwinDaemonManager{
		DaemonManagerCommon: NewDaemonManagerCommon(base),
	}
}

// StartAsDaemon 以守护进程方式启动
func (ddm *DarwinDaemonManager) StartAsDaemon(ctx context.Context, cfg interface{}) error {
	// 构建命令
	cmd := exec.Command(ddm.Base().GetExecutablePath(), "-f", ddm.Base().GetConfigFile())

	// 设置工作目录
	if workDir := ddm.Base().GetWorkDir(); workDir != "" {
		cmd.Dir = workDir
	}

	// 创建进程组和会话
	if err := ddm.CreateProcessGroup(cmd); err != nil {
		return err
	}

	// 重定向 I/O
	logFile := ""
	if ddm.Base().GetConfig() != nil {
		logFile = ddm.Base().GetConfig().LogFile
	}
	if err := ddm.RedirectIO(cmd, logFile); err != nil {
		return err
	}

	// 启动进程
	if err := cmd.Start(); err != nil {
		return pkgerrors.ErrService("failed to start mihomo daemon", err)
	}

	// 保存 PID
	if err := ddm.SavePID(cmd.Process.Pid); err != nil {
		output.Warning("failed to save PID file: " + err.Error())
	}

	output.Success("Mihomo daemon started successfully (PID: %d)", cmd.Process.Pid)
	return nil
}

// StopDaemon 停止守护进程
func (ddm *DarwinDaemonManager) StopDaemon(pid int) error {
	// 检查进程是否运行
	if !ddm.IsDaemonRunning(pid) {
		return pkgerrors.ErrService("daemon is not running", nil)
	}

	// 优先使用 API 优雅关闭
	apiAddr := ddm.Base().GetAPIAddress()
	secret := ddm.Base().GetSecret()

	if apiAddr != "" && secret != "" {
		output.Printf("Attempting to shutdown daemon via API...\n")
		if err := StopProcessByPID(pid, apiAddr, secret); err == nil {
			ddm.CleanupPID()
			return nil
		}
		output.Warning("API shutdown failed, using force kill")
	}

	// 强制关闭
	return ddm.ForceKillDaemon(pid)
}

// IsDaemonRunning 检查守护进程是否运行
func (ddm *DarwinDaemonManager) IsDaemonRunning(pid int) bool {
	return ddm.DaemonManagerCommon.IsDaemonRunning(pid)
}

// GetDaemonPID 获取守护进程 PID
func (ddm *DarwinDaemonManager) GetDaemonPID() (int, error) {
	return ddm.DaemonManagerCommon.GetDaemonPID()
}

// CreateProcessGroup 创建进程组
func (ddm *DarwinDaemonManager) CreateProcessGroup(cmd *exec.Cmd) error {
	// macOS 与 Linux 类似，使用 Setsid 和 Setpgid
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Setsid:  true, // 创建新会话
		Setpgid: true, // 创建新进程组
	}
	return nil
}

// RedirectIO 重定向标准输入输出
func (ddm *DarwinDaemonManager) RedirectIO(cmd *exec.Cmd, logFile string) error {
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
		// 不关闭文件句柄，子进程需要使用
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

// LaunchdManager launchd 守护进程管理器
type LaunchdManager struct {
	plistPath string
	label     string
}

// NewLaunchdManager 创建 launchd 管理器
func NewLaunchdManager(label, plistPath string) *LaunchdManager {
	return &LaunchdManager{
		label:     label,
		plistPath: plistPath,
	}
}

// CreatePlist 创建 launchd plist 文件
func (lm *LaunchdManager) CreatePlist(execPath, configFile, logPath string) error {
	// 确保 plist 目录存在
	plistDir := filepath.Dir(lm.plistPath)
	if err := os.MkdirAll(plistDir, 0755); err != nil {
		return pkgerrors.ErrConfig("failed to create plist directory", err)
	}

	// 确保 log 目录存在
	logDir := filepath.Dir(logPath)
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return pkgerrors.ErrConfig("failed to create log directory", err)
	}

	// 构建 plist 内容
	plistContent := fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>Label</key>
    <string>%s</string>
    <key>ProgramArguments</key>
    <array>
        <string>%s</string>
        <string>-f</string>
        <string>%s</string>
    </array>
    <key>RunAtLoad</key>
    <true/>
    <key>KeepAlive</key>
    <true/>
    <key>StandardOutPath</key>
    <string>%s</string>
    <key>StandardErrorPath</key>
    <string>%s</string>
    <key>WorkingDirectory</key>
    <string>%s</string>
</dict>
</plist>`,
		lm.label,
		execPath,
		configFile,
		logPath+".stdout.log",
		logPath+".stderr.log",
		filepath.Dir(execPath),
	)

	return os.WriteFile(lm.plistPath, []byte(plistContent), 0644)
}

// Load 加载 launchd 服务
func (lm *LaunchdManager) Load() error {
	cmd := exec.Command("launchctl", "load", lm.plistPath)
	if err := cmd.Run(); err != nil {
		return pkgerrors.ErrService("failed to load launchd service", err)
	}
	output.Success("Launchd service loaded: %s", lm.label)
	return nil
}

// Unload 卸载 launchd 服务
func (lm *LaunchdManager) Unload() error {
	cmd := exec.Command("launchctl", "unload", lm.plistPath)
	if err := cmd.Run(); err != nil {
		return pkgerrors.ErrService("failed to unload launchd service", err)
	}
	output.Success("Launchd service unloaded: %s", lm.label)
	return nil
}

// Start 启动 launchd 服务
func (lm *LaunchdManager) Start() error {
	cmd := exec.Command("launchctl", "start", lm.label)
	if err := cmd.Run(); err != nil {
		return pkgerrors.ErrService("failed to start launchd service", err)
	}
	output.Success("Launchd service started: %s", lm.label)
	return nil
}

// Stop 停止 launchd 服务
func (lm *LaunchdManager) Stop() error {
	cmd := exec.Command("launchctl", "stop", lm.label)
	if err := cmd.Run(); err != nil {
		return pkgerrors.ErrService("failed to stop launchd service", err)
	}
	output.Success("Launchd service stopped: %s", lm.label)
	return nil
}

// GetStatus 获取服务状态
func (lm *LaunchdManager) GetStatus() (string, error) {
	cmd := exec.Command("launchctl", "list", lm.label)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", pkgerrors.ErrService("failed to get launchd service status", err)
	}
	return string(output), nil
}

// GetDaemonManager 获取守护进程管理器（工厂函数）
func GetDaemonManager(
	config *DaemonConfig,
	pidFile, secret, apiAddr, execPath, configFile string,
) DaemonManager {
	return NewDarwinDaemonManager(config, pidFile, secret, apiAddr, execPath, configFile)
}