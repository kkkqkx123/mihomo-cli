package mihomo

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"syscall"

	"github.com/kkkqkx123/mihomo-cli/internal/config"
	pkgerrors "github.com/kkkqkx123/mihomo-cli/pkg/errors"
)

// ProcessManager Mihomo 进程管理器
type ProcessManager struct {
	config    *config.TomlConfig
	process   *os.Process
	secret    string
	mu        sync.RWMutex
	isRunning bool
	cmd       *exec.Cmd
	pidFile   string // PID 文件路径
	stderr    *bytes.Buffer // 捕获 stderr 输出
	stdout    *bytes.Buffer // 捕获 stdout 输出
}

// NewProcessManager 创建进程管理器
func NewProcessManager(cfg *config.TomlConfig) *ProcessManager {
	pm := &ProcessManager{
		config:  cfg,
		pidFile: getPIDFilePath(cfg.Mihomo.ConfigFile),
	}
	return pm
}

// NewProcessManagerWithConfig 创建进程管理器（指定配置文件路径）
func NewProcessManagerWithConfig(cfg *config.TomlConfig, configFile string) *ProcessManager {
	pm := &ProcessManager{
		config:  cfg,
		pidFile: getPIDFilePath(configFile),
	}
	return pm
}

// getPIDFilePath 获取 PID 文件路径（基于配置文件路径）
func getPIDFilePath(configFile string) string {
	home, err := os.UserHomeDir()
	if err != nil {
		// 如果无法获取用户目录，使用临时目录
		pidDir := os.TempDir()
		return filepath.Join(pidDir, "mihomo-cli.pid")
	}

	// 如果配置文件为空，使用默认名称
	if configFile == "" {
		return filepath.Join(home, ".mihomo-cli", "mihomo.pid")
	}

	// 根据配置文件路径生成唯一的 hash
	hash := generateConfigHash(configFile)
	return filepath.Join(home, ".mihomo-cli", fmt.Sprintf("mihomo-%s.pid", hash))
}

// generateConfigHash 根据配置文件路径生成短 hash
func generateConfigHash(configFile string) string {
	// 使用配置文件的绝对路径作为输入
	absPath, err := filepath.Abs(configFile)
	if err != nil {
		absPath = configFile
	}

	// 使用文件名作为简单的 hash（避免依赖 crypto 包）
	// 取文件名的最后部分，去除扩展名
	filename := filepath.Base(absPath)
	nameWithoutExt := strings.TrimSuffix(filename, filepath.Ext(filename))

	// 如果名称太长，截取前 8 个字符
	if len(nameWithoutExt) > 8 {
		nameWithoutExt = nameWithoutExt[:8]
	}

	// 如果名称为空，使用默认
	if nameWithoutExt == "" {
		nameWithoutExt = "default"
	}

	return nameWithoutExt
}

// Start 启动 Mihomo 内核
func (pm *ProcessManager) Start() error {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	if pm.isRunning {
		return pkgerrors.ErrService("mihomo is already running", nil)
	}

	// 生成随机密钥
	var secret string
	var err error

	if pm.config.Mihomo.AutoGenerateSecret {
		secret, err = config.GenerateRandomSecret()
		if err != nil {
			return pkgerrors.ErrService("failed to generate secret", err)
		}
	} else {
		secret = pm.config.API.Secret
	}

	pm.secret = secret

	// 准备配置文件
	configFile, err := pm.prepareConfigFile(secret)
	if err != nil {
		return pkgerrors.ErrService("failed to prepare config file", err)
	}

	// 构建命令（不使用 CommandContext，避免进程被取消）
	pm.cmd = exec.Command(pm.config.Mihomo.Executable, "-f", configFile)

	// 初始化输出缓冲区
	pm.stdout = &bytes.Buffer{}
	pm.stderr = &bytes.Buffer{}
	pm.cmd.Stdout = pm.stdout
	pm.cmd.Stderr = pm.stderr

	// 设置进程属性（Windows 下隐藏窗口）
	pm.cmd.SysProcAttr = &syscall.SysProcAttr{
		HideWindow: true,
	}

	// 设置工作目录
	workDir := filepath.Dir(pm.config.Mihomo.Executable)
	pm.cmd.Dir = workDir

	// 启动进程
	if err := pm.cmd.Start(); err != nil {
		return pkgerrors.ErrService("failed to start mihomo", err)
	}

	pm.process = pm.cmd.Process
	pm.isRunning = true

	// 保存 PID 到文件
	if err := pm.SavePID(pm.process.Pid); err != nil {
		// 如果保存 PID 失败，记录但不影响启动
		fmt.Printf("Warning: failed to save pid file: %v\n", err)
	}

	// 等待进程退出（后台模式）
	go func() {
		err := pm.cmd.Wait()
		pm.mu.Lock()
		pm.isRunning = false
		// 进程退出时删除 PID 文件
		os.Remove(pm.pidFile)
		
		// 如果进程异常退出，输出错误信息
		if err != nil {
			fmt.Printf("\n[Mihomo 进程异常退出] 错误: %v\n", err)
			if pm.stderr.Len() > 0 {
				fmt.Printf("错误输出:\n%s\n", pm.stderr.String())
			}
			if pm.stdout.Len() > 0 {
				fmt.Printf("标准输出:\n%s\n", pm.stdout.String())
			}
		}
		pm.mu.Unlock()
	}()

	return nil
}

// Stop 停止 Mihomo 内核并等待完全退出
func (pm *ProcessManager) Stop() error {
	pm.mu.Lock()

	if !pm.isRunning || pm.process == nil {
		pm.mu.Unlock()
		return pkgerrors.ErrService("mihomo is not running", nil)
	}

	// 发送终止信号
	if err := pm.process.Kill(); err != nil {
		pm.mu.Unlock()
		return pkgerrors.ErrService("failed to kill mihomo", err)
	}

	pm.isRunning = false
	pm.mu.Unlock()

	// 等待进程完全退出
	if pm.cmd != nil {
		pm.cmd.Wait()
	}

	// 删除 PID 文件
	os.Remove(pm.pidFile)

	return nil
}

// StopByPID 通过 PID 停止进程（用于后台模式）
func (pm *ProcessManager) StopByPID(pid int) error {
	proc, err := os.FindProcess(pid)
	if err != nil {
		return pkgerrors.ErrService("failed to find process "+fmt.Sprintf("%d", pid), err)
	}

	// 发送终止信号
	if err := proc.Kill(); err != nil {
		return pkgerrors.ErrService("failed to kill process "+fmt.Sprintf("%d", pid), err)
	}

	// 删除 PID 文件
	os.Remove(pm.pidFile)

	return nil
}

// IsRunning 检查是否运行中
func (pm *ProcessManager) IsRunning() bool {
	pm.mu.RLock()
	defer pm.mu.RUnlock()
	return pm.isRunning
}

// GetSecret 获取当前密钥
func (pm *ProcessManager) GetSecret() string {
	pm.mu.RLock()
	defer pm.mu.RUnlock()
	return pm.secret
}

// GetAPIAddress 获取 API 地址
func (pm *ProcessManager) GetAPIAddress() string {
	return pm.config.Mihomo.API.ExternalController
}

// GetErrorOutput 获取进程的错误输出
func (pm *ProcessManager) GetErrorOutput() string {
	pm.mu.RLock()
	defer pm.mu.RUnlock()
	if pm.stderr != nil {
		return pm.stderr.String()
	}
	return ""
}

// GetStandardOutput 获取进程的标准输出
func (pm *ProcessManager) GetStandardOutput() string {
	pm.mu.RLock()
	defer pm.mu.RUnlock()
	if pm.stdout != nil {
		return pm.stdout.String()
	}
	return ""
}

// SavePID 保存进程 PID 到文件
func (pm *ProcessManager) SavePID(pid int) error {
	// 确保目录存在
	pidDir := filepath.Dir(pm.pidFile)
	if err := os.MkdirAll(pidDir, 0755); err != nil {
		return pkgerrors.ErrConfig("failed to create pid directory", err)
	}

	data := []byte(strconv.Itoa(pid))
	if err := os.WriteFile(pm.pidFile, data, 0644); err != nil {
		return pkgerrors.ErrConfig("failed to write pid file", err)
	}
	return nil
}

// ReadPID 从文件读取进程 PID
func (pm *ProcessManager) ReadPID() (int, error) {
	data, err := os.ReadFile(pm.pidFile)
	if err != nil {
		return 0, pkgerrors.ErrConfig("failed to read pid file", err)
	}

	pid, err := strconv.Atoi(string(data))
	if err != nil {
		return 0, pkgerrors.ErrConfig("invalid pid format", err)
	}

	return pid, nil
}

// IsProcessRunning 检查进程是否正在运行
func IsProcessRunning(pid int) bool {
	// 使用 Windows API OpenProcess 检查进程是否存在
	// 这比 proc.Signal 更可靠
	return isProcessRunningWindows(pid)
}

// GetPIDFromPIDFile 从 PID 文件读取并检查进程是否运行
func (pm *ProcessManager) GetPIDFromPIDFile() (int, error) {
	pid, err := pm.ReadPID()
	if err != nil {
		return 0, err
	}

	// 检查进程是否真的在运行
	if !IsProcessRunning(pid) {
		return 0, pkgerrors.ErrService("process "+fmt.Sprintf("%d", pid)+" is not running", nil)
	}

	return pid, nil
}

// prepareConfigFile 准备配置文件
func (pm *ProcessManager) prepareConfigFile(secret string) (string, error) {
	// 如果指定了配置文件，直接使用
	if pm.config.Mihomo.ConfigFile != "" {
		return pm.config.Mihomo.ConfigFile, nil
	}

	// 否则生成临时配置文件
	tempDir := os.TempDir()
	configFile := filepath.Join(tempDir, "mihomo-config.yaml")

	// 生成配置内容
	configContent := pm.generateConfigContent(secret)

	// 写入文件
	if err := os.WriteFile(configFile, []byte(configContent), 0644); err != nil {
		return "", pkgerrors.ErrConfig("failed to write config file", err)
	}

	return configFile, nil
}

// generateConfigContent 生成配置内容
func (pm *ProcessManager) generateConfigContent(secret string) string {
	return fmt.Sprintf(`# Auto-generated config by mihomo-go
mixed-port: 7890
mode: rule
log-level: %s

# API 控制器
external-controller: %s
secret: "%s"

# DNS
dns:
  enable: true
  enhanced-mode: fake-ip
  nameserver:
    - 8.8.8.8

# 代理组配置
proxy-groups:
  - name: "Proxy"
    type: select
    proxies:
      - DIRECT

rules:
  - MATCH,Proxy
`, pm.config.Mihomo.Log.Level, pm.config.Mihomo.API.ExternalController, secret)
}

// ValidateProcess 验证进程是否是 Mihomo 进程
func ValidateProcess(pid int, force bool) error {
	// 验证进程
	if !force {
		if !IsProcessRunning(pid) {
			return pkgerrors.ErrService("process "+fmt.Sprintf("%d", pid)+" does not exist", nil)
		}

		verified, err := VerifyMihomoProcess(pid)
		if err != nil {
			return pkgerrors.ErrService("failed to verify process", err)
		}
		if !verified {
			return pkgerrors.ErrService("process "+fmt.Sprintf("%d", pid)+" is not a Mihomo process, use --force to stop", nil)
		}
	}
	return nil
}

// StopProcessByPID 通过 PID 停止进程并等待完全退出
func StopProcessByPID(pid int) error {
	proc, err := os.FindProcess(pid)
	if err != nil {
		return pkgerrors.ErrService("failed to find process "+fmt.Sprintf("%d", pid), err)
	}

	if err := proc.Kill(); err != nil {
		return pkgerrors.ErrService("failed to stop process", err)
	}

	// 等待进程完全退出
	state, err := proc.Wait()
	if err != nil {
		return pkgerrors.ErrService("failed to wait for process to exit", err)
	}

	// 验证进程确实已退出
	if !state.Exited() {
		return pkgerrors.ErrService("process did not exit as expected", nil)
	}

	return nil
}
