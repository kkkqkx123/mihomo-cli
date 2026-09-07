package mihomo

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/kkkqkx123/mihomo-cli/internal/api"
	"github.com/kkkqkx123/mihomo-cli/internal/config"
	"github.com/kkkqkx123/mihomo-cli/internal/output"
	"github.com/kkkqkx123/mihomo-cli/internal/system"
	pkgerrors "github.com/kkkqkx123/mihomo-cli/pkg/errors"
)

// ProcessHandler 进程管理处理器
type ProcessHandler struct {
	configPath string
}

// NewProcessHandler 创建进程处理器
func NewProcessHandler(configPath string) *ProcessHandler {
	return &ProcessHandler{
		configPath: configPath,
	}
}

// StartResult 启动结果
type StartResult struct {
	APIAddress string
	Secret     string
	PID        int
}

// Start 启动 Mihomo 内核（守护进程模式）。
// 职责分工：ProcessHandler 管理业务逻辑（清理/备份/健康检查），
// LifecycleManager 管理生命周期（锁/状态/监控/钩子），
// DaemonLauncher 管理实际的进程操作。
func (ph *ProcessHandler) Start(cfg *config.TomlConfig) (*StartResult, error) {
	// 检查是否启用自动启动
	if !cfg.Mihomo.Enabled {
		return nil, pkgerrors.ErrConfig("mihomo auto-start is disabled in config.toml", nil)
	}

	// 创建守护进程启动器（内核可执行文件校验统一由 ExecutableResolver 完成）
	launcher, err := NewDaemonLauncher(cfg, ph.configPath)
	if err != nil {
		return nil, pkgerrors.ErrService("failed to create daemon launcher", err)
	}

	// 检查是否已经在运行
	if pid, err := launcher.GetRunningPID(); err == nil && pid > 0 {
		return nil, pkgerrors.ErrService(fmt.Sprintf("mihomo is already running (PID: %d), use 'mihomo-cli stop' to stop it first", pid), nil)
	}

	// 创建共享的 ProcessManager 和 LifecycleManager
	pm := NewProcessManager(cfg, ph.configPath)
	pm.SetLauncher(launcher)

	lm, err := NewLifecycleManager(cfg, pm)
	if err != nil {
		return nil, pkgerrors.ErrService("failed to create lifecycle manager", err)
	}
	lm.RegisterHook(&DefaultLifecycleHooks{})

	// === 业务逻辑：前置检查 ===

	// 启动前配置检查 - 检查是否启用了高风险配置（TUN/TProxy）
	hasTUN := false
	hasTProxy := false
	if cfg.Mihomo.ConfigFile != "" {
		validator := config.NewConfigValidator(cfg.Mihomo.ConfigFile)
		if err := validator.ValidateAndWarn(); err != nil {
			// 配置检查失败不影响启动，只记录警告
			output.Warning("config validation failed: " + err.Error())
		}

		// 检测是否启用了 TUN 或 TProxy
		content, _ := os.ReadFile(cfg.Mihomo.ConfigFile)
		if content != nil {
			lowerContent := strings.ToLower(string(content))
			if strings.Contains(lowerContent, "tun:") && strings.Contains(lowerContent, "enable: true") {
				hasTUN = true
			}
			if strings.Contains(lowerContent, "tproxy-port:") {
				hasTProxy = true
			}

		// 检测配置是否使用了需要地理数据库的规则（GEOIP/GEOSITE）
		needsGeoDB := strings.Contains(lowerContent, "geosite:") ||
			strings.Contains(lowerContent, "geoip,") ||
			strings.Contains(lowerContent, "geoip:")
		if needsGeoDB {
			// 检查地理数据库文件是否存在
			homeDir, _ := os.UserHomeDir()
			mihomoDir := filepath.Join(homeDir, ".config", "mihomo")
			hasMMDB := fileExists(filepath.Join(mihomoDir, "geoip.metadb")) ||
				fileExists(filepath.Join(mihomoDir, "Country.mmdb")) ||
				fileExists(filepath.Join(mihomoDir, "geoip.db"))
			hasGeoSite := fileExists(filepath.Join(mihomoDir, "GeoSite.dat"))

			if !hasMMDB || !hasGeoSite {
				output.PrintEmptyLine()
				output.Warning("Config uses GEOIP/GEOSITE rules but geo database files are missing")
				if !hasMMDB {
					output.Printf("  Missing: %s/geoip.metadb\n", mihomoDir)
				}
				if !hasGeoSite {
					output.Printf("  Missing: %s/GeoSite.dat\n", mihomoDir)
				}
				output.Println("")
				output.Println("Stripping GEOIP/GEOSITE rules from config to allow basic startup")
				output.Println("Download geo databases later to enable GEOIP/GEOSITE rules:")
				output.Printf("  curl -L -o %s/geoip.metadb https://github.com/MetaCubeX/meta-rules-dat/releases/download/latest/geoip.metadb\n", mihomoDir)
				output.Printf("  curl -L -o %s/GeoSite.dat https://github.com/MetaCubeX/meta-rules-dat/releases/download/latest/geosite.dat\n", mihomoDir)
				output.PrintEmptyLine()

				// 生成去除 GEOIP/GEOSITE 规则的临时配置文件
				strippedConfig, stripErr := stripGeoRules(cfg.Mihomo.ConfigFile)
				if stripErr != nil {
					output.Warning("failed to strip geo rules: " + stripErr.Error())
				} else {
					cfg.Mihomo.ConfigFile = strippedConfig
					output.Info("Using stripped config: %s", strippedConfig)
				}
			}
		}
		}
	}

	// 启动前检查并清理残留配置（确保系统状态干净）
	output.Info("Checking for residual configuration from abnormal exit...")
	if err := ph.checkAndCleanupBeforeStart(cfg); err != nil {
		return nil, pkgerrors.ErrService("failed to cleanup residual configuration", err)
	}

	// 启动前备份系统配置
	if hasTUN || hasTProxy {
		output.Info("Creating system configuration backup...")
		if err := ph.backupSystemConfig(cfg, hasTUN, hasTProxy); err != nil {
			output.Warning("failed to create backup: " + err.Error())
		} else {
			output.Success("System configuration backup created")
		}
	}

	// 确保 Mihomo 用户配置目录存在（用于 profile 文件持久化）
	homeDir, err := os.UserHomeDir()
	if err == nil {
		mihomoHomeDir := filepath.Join(homeDir, ".config", "mihomo")
		profilesDir := filepath.Join(mihomoHomeDir, "profiles")
		if err := os.MkdirAll(profilesDir, 0755); err != nil {
			output.Warning("failed to create profiles directory: " + err.Error())
		}
	}

	// === 生命周期：锁 + 状态 ===
	ctx := context.Background()
	if err := lm.Start(ctx, cfg); err != nil {
		return nil, err
	}

	// === 实际启动守护进程 ===
	if err := launcher.Start(); err != nil {
		_ = lm.state.SetStage(StageFailed)
		return nil, pkgerrors.ErrService("failed to start mihomo daemon", err)
	}

	// 获取状态
	isRunning, pid, apiAddr, secret := launcher.GetStatus()
	if !isRunning {
		_ = lm.state.SetStage(StageFailed)
		return nil, pkgerrors.ErrService("daemon started but status check failed", nil)
	}

	// 通知 LifecycleManager 启动成功（更新状态 + 启动监控 + 执行钩子）
	if err := lm.OnStarted(pid, apiAddr, secret); err != nil {
		output.Warning("lifecycle post-start notification failed: " + err.Error())
	}

	result := &StartResult{
		APIAddress: apiAddr,
		Secret:     secret,
		PID:        pid,
	}

	// === 业务逻辑：健康检查 ===
	healthCheckTimeout := cfg.Mihomo.HealthCheckTimeout
	if healthCheckTimeout <= 0 {
		healthCheckTimeout = 5 // 默认 5 秒
	}

	// 创建 API 客户端进行健康检查
	apiClient := api.NewClient(
		"http://"+apiAddr,
		secret,
		api.WithTimeout(3*time.Second),
	)

	// 等待并检查健康状况
	checkCtx, cancel := context.WithTimeout(context.Background(), time.Duration(healthCheckTimeout)*time.Second)
	defer cancel()

	output.Printf("等待 Mihomo 内核启动（最多 %d 秒）...\n", healthCheckTimeout)

	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-checkCtx.Done():
			// 健康检查超时，执行诊断并尝试停止守护进程
			output.PrintEmptyLine()
			output.Warning("Health check timeout, performing diagnostics...")

			// 诊断 1: 检查进程状态
			if pid > 0 {
				if !IsProcessRunning(pid) {
					output.Error("Process %d has exited", pid)
				} else if isZombieProcess(pid) {
					output.Error("Process %d is a zombie (crashed but not reaped)", pid)
					output.Info("Hint: the mihomo process likely crashed during startup, check mihomo logs for errors")
				} else {
					output.Info("Process %d is still running but API is not responding", pid)
				}
			}

			// 诊断 2: 检查端口占用
			if apiAddr != "" {
				host, port := parseHostPort(apiAddr)
				if occupyingPID := FindProcessByPort(host, port); occupyingPID > 0 {
					output.Error("Port %s is occupied by process %d", apiAddr, occupyingPID)
				}
			}

			// 诊断 3: 读取 mihomo 日志
			logRead := false
			// 优先读取配置的日志文件
			if cfg.Mihomo.Log.File != "" {
				if logContent, err := readLastLines(cfg.Mihomo.Log.File, 20); err == nil && logContent != "" {
					output.PrintEmptyLine()
					output.PrintSection("Mihomo Log (last 20 lines)")
					output.Printf("%s\n", logContent)
					logRead = true
				}
			}
			// 降级读取临时日志文件
			if !logRead {
				tempLog := filepath.Join(os.TempDir(), "mihomo-cli-daemon.log")
				if logContent, err := readLastLines(tempLog, 20); err == nil && logContent != "" {
					output.PrintEmptyLine()
					output.PrintSection("Mihomo Log (last 20 lines)")
					output.Printf("%s\n", logContent)
				}
			}

			// 尝试停止守护进程
			if pid > 0 {
				_ = launcher.Stop(true)
			}
			return nil, pkgerrors.ErrService("mihomo health check timeout: daemon may have failed to start", nil)

		case <-ticker.C:
			// 检查进程是否还在运行
			if pid > 0 {
				if !IsProcessRunning(pid) {
					errMsg := "mihomo daemon exited unexpectedly"
					return nil, pkgerrors.ErrService(errMsg, nil)
				}
			}

			// 尝试连接 API 进行健康检查
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			_, err := apiClient.GetMode(ctx)
			cancel()

			if err == nil {
				// 基础健康检查成功，进行增强健康检查
				output.PrintEmptyLine()
				output.Info("Performing detailed health check...")

				// 创建新的 context 用于详细健康检查
				healthCtx, healthCancel := context.WithTimeout(context.Background(), 5*time.Second)
				defer healthCancel()

				healthChecker := NewHealthChecker(apiClient, cfg.Mihomo.ConfigFile, 5*time.Second)
				healthStatus, err := healthChecker.CheckHealth(healthCtx)
				if err != nil {
					output.Warning("detailed health check failed: " + err.Error())
					output.Warning("Daemon started but may have issues")
				} else {
					healthChecker.PrintHealthStatus(healthStatus)

					if !healthChecker.IsHealthy(healthStatus) {
						output.PrintEmptyLine()
						output.Warning("⚠ Mihomo started but some components may not be working properly")
						output.Printf("  Check the warnings above for details\n")
					}
				}

				// 健康检查成功
				output.PrintEmptyLine()
				output.Success("Mihomo 内核启动成功！")
				return result, nil
			}
		}
	}
}

// StopResult 停止结果
type StopResult struct {
	PID int
}

// Stop 停止 Mihomo 内核。
// 职责分工：ProcessHandler 管理业务逻辑（系统配置清理），
// LifecycleManager 管理状态追踪和钩子，
// DaemonLauncher 管理实际的进程停止。
func (ph *ProcessHandler) Stop(cfg *config.TomlConfig, stopAll bool, stopConfig string, force bool, includeUnmanaged bool, args []string) (*StopResult, error) {
	// 如果指定了 --all，停止所有进程
	if stopAll {
		return nil, StopAllMihomoProcesses(includeUnmanaged)
	}

	// 创建守护进程启动器
	launcher, err := NewDaemonLauncher(cfg, ph.configPath)
	if err != nil {
		return nil, pkgerrors.ErrService("failed to create daemon launcher", err)
	}

	var pid int

	// 如果指定了 PID 参数
	if len(args) == 1 {
		_, err := fmt.Sscanf(args[0], "%d", &pid)
		if err != nil {
			return nil, pkgerrors.ErrInvalidArg("invalid PID: "+args[0], nil)
		}

		// 验证进程是否在运行
		if !IsProcessRunning(pid) {
			return nil, pkgerrors.ErrService("process "+fmt.Sprintf("%d", pid)+" is not running", nil)
		}
	} else {
		// 默认：停止当前配置的实例
		var err error
		pid, err = launcher.GetRunningPID()
		if err != nil {
			return nil, pkgerrors.ErrService("mihomo is not running", err)
		}
	}

	// 创建 LifecycleManager 用于状态追踪和钩子
	pm := NewProcessManager(cfg, ph.configPath)
	pm.SetLauncher(launcher)

	lm, err := NewLifecycleManager(cfg, pm)
	if err != nil {
		// 降级：不使用 LifecycleManager，直接停止
		if stopErr := launcher.Stop(force); stopErr != nil {
			return nil, stopErr
		}
	} else {
		lm.RegisterHook(&DefaultLifecycleHooks{})
		// 通过 LifecycleManager 停止（状态追踪 + 钩子 + 实际停止）
		if err := lm.Stop(context.Background(), pid); err != nil {
			// 降级：直接停止
			if stopErr := launcher.Stop(force); stopErr != nil {
				return nil, stopErr
			}
		}
	}

	// 检查系统配置状态并尝试清理
	output.Info("Checking system configuration...")
	if err := ph.checkAndCleanupAfterStop(cfg); err != nil {
		output.Warning("failed to check system configuration: " + err.Error())
	}

	return &StopResult{PID: pid}, nil
}

// StatusResult 状态结果
type StatusResult struct {
	IsRunning  bool
	PID        int
	APIAddress string
}

// Status 查询 Mihomo 内核状态
func (ph *ProcessHandler) Status(cfg *config.TomlConfig) (*StatusResult, error) {
	// 创建守护进程启动器
	launcher, err := NewDaemonLauncher(cfg, ph.configPath)
	if err != nil {
		return nil, pkgerrors.ErrService("failed to create daemon launcher", err)
	}

	// 获取状态
	isRunning, pid, apiAddr, _ := launcher.GetStatus()

	return &StatusResult{
		IsRunning:  isRunning,
		PID:        pid,
		APIAddress: apiAddr,
	}, nil
}

// checkAndCleanupBeforeStart 启动前检查并清理残留配置
func (ph *ProcessHandler) checkAndCleanupBeforeStart(cfg *config.TomlConfig) error {
	// 清理过期的 State 文件（stage=failed 或 pid=0），避免残留状态影响启动
	if cfg.Mihomo.ConfigFile != "" {
		pathResolver, prErr := config.NewPathResolver()
		if prErr == nil {
			stateFile := pathResolver.GetStateFilePath(cfg.Mihomo.ConfigFile)
			if stateFile != "" {
				data, readErr := os.ReadFile(stateFile)
				if readErr == nil {
					var state ProcessState
					if json.Unmarshal(data, &state) == nil {
						if state.Stage == StageFailed || state.PID == 0 {
							os.Remove(stateFile)
							output.Info("Cleaned up stale state file (stage=%s)", state.Stage)
						}
					}
				}
			}
		}
	}

	scm, err := system.NewSystemConfigManager()
	if err != nil {
		return err
	}

	// 检查是否有上次异常退出留下的残留
	problems, err := scm.ValidateState()
	if err != nil {
		return err
	}

	if len(problems) > 0 {
		output.Warning("Detected %d residual configuration issues from abnormal exit", len(problems))
		for _, problem := range problems {
			output.Printf("  - %s (severity: %s)\n", problem.Description, problem.Severity)
		}

		// 尝试自动清理
		output.Info("Cleaning up residual configuration before start...")
		if err := scm.CleanupAll(); err != nil {
			output.Warning("Automatic cleanup failed: " + err.Error())
			output.Println("Manual cleanup may be required")
			return fmt.Errorf("failed to cleanup residual configuration: %w", err)
		}
		output.Success("Automatic cleanup completed, system is now clean")
	}

	return nil
}

// backupSystemConfig 备份系统配置
func (ph *ProcessHandler) backupSystemConfig(_ *config.TomlConfig, hasTUN, hasTProxy bool) error {
	dataDir, err := config.GetDataDir()
	if err != nil {
		return err
	}

	// 确保目录存在
	backupDir := filepath.Join(dataDir, "system-backups")
	if err := os.MkdirAll(backupDir, 0755); err != nil {
		return err
	}

	// 创建系统配置管理器
	scm, err := system.NewSystemConfigManager()
	if err != nil {
		return err
	}

	// 备份路由表
	if hasTUN || hasTProxy {
		routeManager := scm.GetRouteManager()
		routeBackup, err := routeManager.BackupRoutes("pre-start backup")
		if err == nil {
			_, err = routeManager.SaveBackup(routeBackup)
			if err != nil {
				output.Warning("failed to save route backup: " + err.Error())
			}
		}
	}

	// 备份 TUN 接口状态
	if hasTUN {
		tunManager := scm.GetTUNManager()
		tunBackup, err := tunManager.BackupTUNState("pre-start backup")
		if err == nil {
			_, err = tunManager.SaveTUNBackup(tunBackup)
			if err != nil {
				output.Warning("failed to save TUN backup: " + err.Error())
			}
		}
	}

	// 备份注册表设置（仅 Windows）
	spm := scm.GetSysProxyManager()
	if spm != nil {
		proxyStatus, err := spm.GetStatus()
		if err == nil {
			// 保存注册表备份
			backupData, err := json.MarshalIndent(proxyStatus, "", "  ")
			if err == nil {
				backupFile := filepath.Join(backupDir, "registry-backup-pre-start.json")
				_ = os.WriteFile(backupFile, backupData, 0644)
			}
		}
	}

	return nil
}

// checkAndCleanupAfterStop 停止后检查并清理系统配置
func (ph *ProcessHandler) checkAndCleanupAfterStop(_ *config.TomlConfig) error {
	scm, err := system.NewSystemConfigManager()
	if err != nil {
		return err
	}

	// 检查残留配置
	problems, err := scm.ValidateState()
	if err != nil {
		return err
	}

	if len(problems) > 0 {
		output.Warning("Detected %d residual configuration issues", len(problems))
		for _, problem := range problems {
			output.Printf("  - %s (severity: %s)\n", problem.Description, problem.Severity)
		}

		// 尝试自动清理
		output.Info("Attempting automatic cleanup...")
		if err := scm.CleanupAll(); err != nil {
			output.Warning("Automatic cleanup failed: " + err.Error())
			output.Println("Manual cleanup may be required")
		} else {
			output.Success("Automatic cleanup completed")
		}
	}

	return nil
}

// isZombieProcess 检查进程是否为僵尸进程
func isZombieProcess(pid int) bool {
	statPath := filepath.Join("/proc", strconv.Itoa(pid), "stat")
	data, err := os.ReadFile(statPath)
	if err != nil {
		return false
	}
	content := string(data)
	lastParen := strings.LastIndex(content, ")")
	if lastParen == -1 || lastParen+2 >= len(content) {
		return false
	}
	return content[lastParen+2] == 'Z'
}

// parseHostPort 从 host:port 字符串中解析出 host 和 port
func parseHostPort(addr string) (string, string) {
	host, port, err := net.SplitHostPort(addr)
	if err != nil {
		return addr, "9090"
	}
	return host, port
}

// fileExists 检查文件是否存在
func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// stripGeoRules 生成去除 GEOIP/GEOSITE 规则的临时配置文件
func stripGeoRules(configPath string) (string, error) {
	content, err := os.ReadFile(configPath)
	if err != nil {
		return "", err
	}

	lines := strings.Split(string(content), "\n")
	var stripped []string
	inRules := false
	inNameServerPolicy := false
	inFallbackFilter := false
	skipNameServerPolicyValue := false
	geoipInjected := false
	blockBaseIndent := 0

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		currentIndent := len(line) - len(strings.TrimLeft(line, " \t"))

		// 跟踪 rules: 块
		if trimmed == "rules:" {
			inRules = true
			inNameServerPolicy = false
			inFallbackFilter = false
			stripped = append(stripped, line)
			continue
		}

		// 跟踪 fallback-filter: 块
		if strings.HasPrefix(trimmed, "fallback-filter:") {
			inFallbackFilter = true
			inRules = false
			inNameServerPolicy = false
			blockBaseIndent = currentIndent
			stripped = append(stripped, line)
			continue
		}

		// 在 fallback-filter 块中，替换 geoip 配置为 geoip: false
		if inFallbackFilter {
			if trimmed != "" && currentIndent <= blockBaseIndent && !strings.HasPrefix(trimmed, "#") {
				inFallbackFilter = false
				geoipInjected = false
			} else if trimmed == "geoip: true" || strings.HasPrefix(trimmed, "geoip-code:") {
				if !geoipInjected {
					// 替换为 geoip: false 禁用 GEOIP
					indent := strings.Repeat(" ", currentIndent)
					stripped = append(stripped, indent+"geoip: false")
					geoipInjected = true
				}
				continue
			}
		}

		// 跟踪 nameserver-policy: 块
		if strings.HasPrefix(trimmed, "nameserver-policy:") {
			inNameServerPolicy = true
			inRules = false
			inFallbackFilter = false
			blockBaseIndent = currentIndent
			stripped = append(stripped, line)
			continue
		}

		// 在 nameserver-policy 块中，跳过 geosite: 相关的条目
		if inNameServerPolicy {
			if trimmed != "" && currentIndent <= blockBaseIndent && !strings.HasPrefix(trimmed, "#") {
				inNameServerPolicy = false
				skipNameServerPolicyValue = false
			} else if strings.Contains(trimmed, "geosite:") {
				skipNameServerPolicyValue = true
				continue
			} else if skipNameServerPolicyValue {
				if strings.HasPrefix(trimmed, "-") {
					continue
				}
				skipNameServerPolicyValue = false
			}
		}

		// 在 rules 块中，跳过 GEOIP/GEOSITE 行
		if inRules {
			upper := strings.ToUpper(trimmed)
			if strings.HasPrefix(upper, "- GEOIP,") || strings.HasPrefix(upper, "- GEOSITE,") {
				continue
			}
			if trimmed != "" && !strings.HasPrefix(trimmed, " ") && !strings.HasPrefix(trimmed, "\t") && !strings.HasPrefix(trimmed, "-") {
				inRules = false
			}
		}

		stripped = append(stripped, line)
	}

	// 写入临时文件
	tmpFile, err := os.CreateTemp("", "mihomo-config-*.yaml")
	if err != nil {
		return "", err
	}
	defer tmpFile.Close()

	if _, err := tmpFile.WriteString(strings.Join(stripped, "\n")); err != nil {
		os.Remove(tmpFile.Name())
		return "", err
	}

	return tmpFile.Name(), nil
}

// readLastLines 读取文件最后 N 行
func readLastLines(filePath string, n int) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	stat, err := file.Stat()
	if err != nil {
		return "", err
	}

	fileSize := stat.Size()
	if fileSize == 0 {
		return "", nil
	}

	// 读取文件末尾（最多 8KB）
	readSize := int64(8192)
	if readSize > fileSize {
		readSize = fileSize
	}

	buf := make([]byte, readSize)
	_, err = file.ReadAt(buf, fileSize-readSize)
	if err != nil {
		return "", err
	}

	lines := strings.Split(string(buf), "\n")
	if len(lines) <= n {
		return strings.TrimSpace(string(buf)), nil
	}

	// 返回最后 n 行（跳过第一个可能不完整的行）
	result := strings.Join(lines[len(lines)-n:], "\n")
	return strings.TrimSpace(result), nil
}
