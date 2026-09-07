package mihomo

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/kkkqkx123/mihomo-cli/internal/api"
	"github.com/kkkqkx123/mihomo-cli/internal/config"
	"github.com/kkkqkx123/mihomo-cli/internal/output"
	pkgerrors "github.com/kkkqkx123/mihomo-cli/pkg/errors"
)

// DaemonLauncher 守护进程启动器（统一管理路径解析和状态持久化）
type DaemonLauncher struct {
	cfg          *config.TomlConfig
	stateMgr     *StateManager
	pathResolver *config.PathResolver
	execResolver *config.ExecutableResolver
	execPath     string // 解析后的内核可执行文件绝对路径（Start 时确定）
	secret       string
	apiAddress   string
}

// NewDaemonLauncher 创建守护进程启动器。
// configPath 为 config.toml 路径（可为空），用于确定内核相对路径的解析基准目录。
func NewDaemonLauncher(cfg *config.TomlConfig, configPath string) (*DaemonLauncher, error) {
	// 创建路径解析器
	pathResolver, err := config.NewPathResolver()
	if err != nil {
		return nil, err
	}

	// 创建状态管理器
	stateMgr, err := NewStateManagerWithResolver(cfg.Mihomo.ConfigFile, pathResolver)
	if err != nil {
		return nil, err
	}

	// 内核相对路径以 config.toml 所在目录为基准
	baseDir := ""
	if configPath != "" {
		baseDir = filepath.Dir(configPath)
	}

	return &DaemonLauncher{
		cfg:          cfg,
		stateMgr:     stateMgr,
		pathResolver: pathResolver,
		execResolver: config.NewExecutableResolver(baseDir),
	}, nil
}

// GetAbsolutePath 将路径转换为绝对路径
func (dl *DaemonLauncher) GetAbsolutePath(path string) string {
	return dl.pathResolver.GetAbsolutePath(path)
}

// GetExecutablePath 获取解析后的内核可执行文件绝对路径（需先 Start）
func (dl *DaemonLauncher) GetExecutablePath() string {
	return dl.execPath
}

// GetConfigFilePath 获取配置文件的绝对路径
func (dl *DaemonLauncher) GetConfigFilePath() string {
	return dl.GetAbsolutePath(dl.cfg.Mihomo.ConfigFile)
}

// GetWorkDir 获取工作目录（可执行文件所在目录的绝对路径）
func (dl *DaemonLauncher) GetWorkDir() string {
	execPath := dl.GetExecutablePath()
	if execPath == "" {
		return ""
	}
	dir := filepath.Dir(execPath)
	// 确保返回绝对路径
	if !filepath.IsAbs(dir) {
		if abs, err := filepath.Abs(dir); err == nil {
			return abs
		}
	}
	return dir
}

// GetAPIAddress 获取 API 地址
func (dl *DaemonLauncher) GetAPIAddress() string {
	return dl.cfg.Mihomo.API.ExternalController
}

// GetSecret 获取当前密钥（需先 Start）
func (dl *DaemonLauncher) GetSecret() string {
	return dl.secret
}

// PrepareSecret 准备 API 密钥
func (dl *DaemonLauncher) PrepareSecret() (string, error) {
	if dl.cfg.Mihomo.AutoGenerateSecret {
		return config.GenerateRandomSecret()
	}
	return dl.cfg.API.Secret, nil
}

// Start 启动守护进程
func (dl *DaemonLauncher) Start() error {
	// 检查是否已经在运行
	if pid, err := dl.GetRunningPID(); err == nil && pid > 0 {
		return pkgerrors.ErrService(fmt.Sprintf("mihomo is already running (PID: %d), use 'mihomo-cli stop' to stop it first", pid), nil)
	}

	// 解析内核可执行文件（全项目唯一校验点）
	execPath, source, err := dl.execResolver.Resolve(dl.cfg.Mihomo.Executable, dl.cfg.Mihomo.ExecutableSearch)
	if err != nil {
		return err
	}
	dl.execPath = execPath
	if source != config.ResolveSourceExplicit {
		output.Info("mihomo executable resolved via %s: %s", source, execPath)
	}

	// 准备密钥
	secret, err := dl.PrepareSecret()
	if err != nil {
		return err
	}
	dl.secret = secret
	dl.apiAddress = dl.GetAPIAddress()

	// 配置文件：未显式指定时生成临时配置
	configPath := dl.GetConfigFilePath()
	if configPath == "" {
		configPath = dl.prepareTempConfig(secret)
	}

	workDir := dl.GetWorkDir()

	// 构建守护进程配置
	daemonConfig := &DaemonConfig{
		Enabled:       true,
		WorkDir:       workDir,
		LogFile:       "",
		LogLevel:      "info",
		LogMaxSize:    "100M",
		LogMaxBackups: 10,
		LogMaxAge:     30,
	}

	// 获取 PID 文件路径
	pidFile := dl.pathResolver.GetPIDFilePath(dl.cfg.Mihomo.ConfigFile)

	// 创建守护进程管理器
	daemonMgr := GetDaemonManager(
		daemonConfig,
		pidFile,
		secret,
		dl.apiAddress,
		execPath,
		configPath,
	)

	// 设置状态
	dl.stateMgr.Update(func(state *ProcessState) {
		state.Stage = StageStarting
		state.APIAddress = dl.apiAddress
		state.Secret = secret
		state.ConfigFile = configPath
		state.StartedAt = time.Now()
	})

	// 启动进程
	ctx := context.Background()
	if err := daemonMgr.StartAsDaemon(ctx, nil); err != nil {
		dl.stateMgr.SetStage(StageFailed)
		return err
	}

	// 获取 PID
	pid, err := daemonMgr.GetDaemonPID()
	if err != nil {
		output.Warning("Failed to get PID, but daemon started successfully")
		pid = 0
	}

	// 更新状态
	dl.stateMgr.Update(func(state *ProcessState) {
		state.PID = pid
		state.Stage = StageRunning
	})

	return nil
}

// prepareTempConfig 未配置 mihomo 配置文件时生成临时配置（供内核 -f 参数使用）
func (dl *DaemonLauncher) prepareTempConfig(secret string) string {
	tempDir := os.TempDir()
	configFile := filepath.Join(tempDir, "mihomo-config.yaml")

	content := fmt.Sprintf(`# Auto-generated config by mihomo-cli
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
`, dl.cfg.Mihomo.Log.Level, dl.apiAddress, secret)

	if err := os.WriteFile(configFile, []byte(content), 0644); err != nil {
		output.Warning("failed to write temp config file: " + err.Error())
		return ""
	}
	return configFile
}

// Stop 停止守护进程
func (dl *DaemonLauncher) Stop(force bool) error {
	// 获取运行中的 PID
	pid, err := dl.GetRunningPID()
	if err != nil {
		return pkgerrors.ErrService("mihomo is not running", err)
	}

	// 获取保存的密钥（先从内存，再从磁盘加载兜底）
	state := dl.stateMgr.Get()
	if state == nil {
		// 内存状态为空（例如 ProcessHandler.Stop() 创建了新 Launcher），从磁盘加载
		_ = dl.stateMgr.Load()
		state = dl.stateMgr.Get()
	}

	secret := ""
	apiAddr := dl.GetAPIAddress()
	if state != nil {
		secret = state.Secret
		if state.APIAddress != "" {
			apiAddr = state.APIAddress
		}
	}

	// 设置状态
	dl.stateMgr.SetStage(StageStopping)

	// 优先通过 API 优雅关闭
	if apiAddr != "" && secret != "" && !force {
		output.Info("Attempting to shutdown via API...")
		if err := StopProcessByPID(pid, apiAddr, secret); err != nil {
			return pkgerrors.ErrService("API shutdown failed (use -F flag to force kill)", err)
		}
	} else {
		// 强制关闭
		output.Info("Force killing process...")
		if err := ForceKill(pid); err != nil {
			return pkgerrors.ErrService("failed to stop daemon", err)
		}
	}

	// 清理状态
	dl.stateMgr.Clear()

	// 清理 PID 文件
	pidFile := dl.pathResolver.GetPIDFilePath(dl.cfg.Mihomo.ConfigFile)
	if pidFile != "" {
		os.Remove(pidFile)
	}

	return nil
}

// GetRunningPID 获取运行中的进程 PID
func (dl *DaemonLauncher) GetRunningPID() (int, error) {
	// 先从状态文件读取
	state := dl.stateMgr.Get()
	if state != nil && state.PID > 0 {
		if IsProcessRunning(state.PID) {
			return state.PID, nil
		}
	}

	// 再从 PID 文件读取
	pidFile := dl.pathResolver.GetPIDFilePath(dl.cfg.Mihomo.ConfigFile)
	if pidFile == "" {
		return 0, pkgerrors.ErrService("PID file not configured", nil)
	}

	meta, err := NewInstanceRegistry(pidFile).Load()
	if err != nil {
		return 0, err
	}

	// 检查进程是否真的在运行
	if !IsProcessRunning(meta.PID) {
		return 0, pkgerrors.ErrService("process is not running", nil)
	}

	return meta.PID, nil
}

// GetStatus 获取运行状态
func (dl *DaemonLauncher) GetStatus() (bool, int, string, string) {
	pid, err := dl.GetRunningPID()
	if err != nil {
		return false, 0, "", ""
	}

	// 从状态文件获取密钥和 API 地址
	state := dl.stateMgr.Get()
	secret := ""
	apiAddr := dl.GetAPIAddress()
	if state != nil {
		secret = state.Secret
		if state.APIAddress != "" {
			apiAddr = state.APIAddress
		}
	}

	return true, pid, apiAddr, secret
}

// WaitForHealthy 等待进程健康
func (dl *DaemonLauncher) WaitForHealthy(timeout time.Duration) error {
	pid, err := dl.GetRunningPID()
	if err != nil {
		return err
	}

	// 获取密钥
	state := dl.stateMgr.Get()
	secret := ""
	if state != nil {
		secret = state.Secret
	}

	apiClient := api.NewClient(
		"http://"+dl.apiAddress,
		secret,
		api.WithTimeout(3*time.Second),
	)

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return pkgerrors.ErrService("health check timeout", nil)
		case <-ticker.C:
			// 检查进程是否还在运行
			if !IsProcessRunning(pid) {
				return pkgerrors.ErrService("daemon exited unexpectedly", nil)
			}

			// 尝试连接 API
			checkCtx, checkCancel := context.WithTimeout(context.Background(), 2*time.Second)
			_, err := apiClient.GetMode(checkCtx)
			checkCancel()

			if err == nil {
				dl.stateMgr.UpdateHealthCheck()
				return nil
			}
		}
	}
}
