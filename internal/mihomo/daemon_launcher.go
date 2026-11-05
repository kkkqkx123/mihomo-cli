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

// GetWorkDir 获取工作目录（配置文件所在目录；未配置时返回空，由内核使用默认目录）
func (dl *DaemonLauncher) GetWorkDir() string {
	configFile := dl.GetConfigFilePath()
	if configFile == "" {
		return ""
	}
	return filepath.Dir(configFile)
}

// GetAPIAddress 获取 API 地址
func (dl *DaemonLauncher) GetAPIAddress() string {
	return dl.cfg.APIHostPort()
}

// GetSecret 获取当前密钥（需先 Start）
func (dl *DaemonLauncher) GetSecret() string {
	return dl.secret
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

	// API 密钥唯一来源为 [api] secret
	secret := dl.cfg.API.Secret
	dl.secret = secret
	dl.apiAddress = dl.GetAPIAddress()

	// 配置文件为空时直接以默认配置启动内核（内核自行定位 ~/.config/mihomo/config.yaml）
	configPath := dl.GetConfigFilePath()

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

// Stop 停止守护进程
func (dl *DaemonLauncher) Stop(force bool) error {
	// 获取运行中的 PID
	pid, err := dl.GetRunningPID()
	if err != nil {
		return pkgerrors.ErrService("mihomo is not running", err)
	}

	// API 地址与密钥由配置实时提供，不依赖状态文件
	apiAddr := dl.GetAPIAddress()
	secret := dl.cfg.API.Secret

	// 设置状态
	dl.stateMgr.SetStage(StageStopping)

	// 优先通过 API 优雅关闭（空 secret 表示内核未启用鉴权）
	if apiAddr != "" && !force {
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

	return true, pid, dl.GetAPIAddress(), dl.cfg.API.Secret
}

// WaitForHealthy 等待进程健康
func (dl *DaemonLauncher) WaitForHealthy(timeout time.Duration) error {
	pid, err := dl.GetRunningPID()
	if err != nil {
		return err
	}

	apiClient := api.NewClient(
		"http://"+dl.GetAPIAddress(),
		dl.cfg.API.Secret,
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
