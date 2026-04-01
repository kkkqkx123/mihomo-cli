# 守护进程模式增强功能实现方案

## 文档信息

- **创建日期**: 2026-04-01
- **版本**: 1.0
- **状态**: 规划阶段
- **目标**: 实现守护进程模式的增强功能

---

## 一、概述

当前守护进程模式已经实现了基础的进程分离功能，但还需要实现以下增强功能以提供更完善的生产级支持：

1. **日志轮转**：实现日志文件的自动轮转和管理
2. **自动重启**：实现进程崩溃检测和自动重启
3. **健康监控**：实现定期的健康检查和告警
4. **多实例支持**：支持同时运行多个 Mihomo 实例
5. **系统集成**：与系统原生服务管理深度集成

本文档将详细规划每个功能的实现方案。

---

## 二、日志轮转实现

### 2.1 功能需求

- 支持按文件大小轮转
- 支持按时间轮转
- 支持按数量保留备份
- 支持按时间保留备份
- 支持压缩备份文件
- 支持自定义命名规则

### 2.2 技术方案

使用第三方库 `lumberjack` 实现日志轮转功能：

```go
import "gopkg.in/natefinch/lumberjack.v2"
```

### 2.3 实现步骤

#### 步骤1：创建日志管理器

**文件**: `internal/log/rotation.go`

```go
package log

import (
	"io"
	"os"
	"path/filepath"
	"sync"

	"gopkg.in/natefinch/lumberjack.v2"
)

// LogRotator 日志轮转管理器
type LogRotator struct {
	mu      sync.RWMutex
	logger  *lumberjack.Logger
	config  *RotationConfig
	closers map[string]io.Closer
}

// RotationConfig 日志轮转配置
type RotationConfig struct {
	Filename   string `json:"filename"`   // 日志文件路径
	MaxSize    int    `json:"max_size"`   // 单个日志文件最大大小（MB）
	MaxBackups int    `json:"max_backups"` // 保留的旧日志文件的最大个数
	MaxAge     int    `json:"max_age"`     // 保留旧日志文件的最大天数
	Compress   bool   `json:"compress"`   // 是否压缩旧日志文件
	LocalTime  bool   `json:"local_time"`  // 是否使用本地时间
}

// NewLogRotator 创建日志轮转管理器
func NewLogRotator(config *RotationConfig) (*LogRotator, error) {
	if config.Filename == "" {
		return nil, fmt.Errorf("log filename is required")
	}

	// 确保日志目录存在
	logDir := filepath.Dir(config.Filename)
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create log directory: %w", err)
	}

	// 设置默认值
	if config.MaxSize == 0 {
		config.MaxSize = 100 // 默认 100MB
	}
	if config.MaxBackups == 0 {
		config.MaxBackups = 10 // 默认保留 10 个备份
	}
	if config.MaxAge == 0 {
		config.MaxAge = 30 // 默认保留 30 天
	}

	// 创建 lumberjack logger
	logger := &lumberjack.Logger{
		Filename:   config.Filename,
		MaxSize:    config.MaxSize,
		MaxBackups: config.MaxBackups,
		MaxAge:     config.MaxAge,
		Compress:   config.Compress,
		LocalTime:  config.LocalTime,
	}

	return &LogRotator{
		logger:  logger,
		config:  config,
		closers: make(map[string]io.Closer),
	}, nil
}

// Writer 返回日志写入器
func (lr *LogRotator) Writer() io.Writer {
	lr.mu.RLock()
	defer lr.mu.RUnlock()
	return lr.logger
}

// Close 关闭日志轮转管理器
func (lr *LogRotator) Close() error {
	lr.mu.Lock()
	defer lr.mu.Unlock()

	// 关闭 lumberjack logger
	if err := lr.logger.Close(); err != nil {
		return err
	}

	// 关闭所有附加的 closers
	for name, closer := range lr.closers {
		if err := closer.Close(); err != nil {
			return fmt.Errorf("failed to close %s: %w", name, err)
		}
	}

	return nil
}

// Rotate 手动触发日志轮转
func (lr *LogRotator) Rotate() error {
	lr.mu.RLock()
	defer lr.mu.RUnlock()
	return lr.logger.Rotate()
}

// GetCurrentLogFile 获取当前日志文件路径
func (lr *LogRotator) GetCurrentLogFile() string {
	lr.mu.RLock()
	defer lr.mu.RUnlock()
	return lr.logger.Filename
}

// GetBackupFiles 获取所有备份文件列表
func (lr *LogRotator) GetBackupFiles() ([]string, error) {
	lr.mu.RLock()
	defer lr.mu.RUnlock()

	dir := filepath.Dir(lr.config.Filename)
	base := filepath.Base(lr.config.Filename)

	files, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	var backups []string
	for _, file := range files {
		if file.IsDir() {
			continue
		}

		name := file.Name()
		// 匹配备份文件模式
		if name == base || (len(name) > len(base) && name[:len(base)] == base) {
			fullPath := filepath.Join(dir, name)
			backups = append(backups, fullPath)
		}
	}

	return backups, nil
}

// GetLogSize 获取当前日志文件大小
func (lr *LogRotator) GetLogSize() (int64, error) {
	fileInfo, err := os.Stat(lr.config.Filename)
	if err != nil {
		return 0, err
	}
	return fileInfo.Size(), nil
}
```

#### 步骤2：集成到守护进程管理器

修改 `internal/mihomo/daemon.go`：

```go
// DaemonManagerBase 守护进程管理器基类
type DaemonManagerBase struct {
	config      *DaemonConfig
	pidFile     string
	secret      string
	apiAddr     string
	execPath    string
	configFile  string
	logRotator  *log.LogRotator // 新增：日志轮转管理器
}

// InitLogRotator 初始化日志轮转
func (dmb *DaemonManagerBase) InitLogRotator() error {
	if dmb.config == nil || dmb.config.LogFile == "" {
		return nil // 不需要日志轮转
	}

	// 解析日志大小
	maxSize, err := parseSize(dmb.config.LogMaxSize)
	if err != nil {
		return fmt.Errorf("invalid log_max_size: %w", err)
	}

	// 创建日志轮转配置
	rotConfig := &log.RotationConfig{
		Filename:   dmb.config.LogFile,
		MaxSize:    maxSize,
		MaxBackups: dmb.config.LogMaxBackups,
		MaxAge:     dmb.config.LogMaxAge,
		Compress:   true, // 默认压缩
		LocalTime:  true,
	}

	// 创建日志轮转管理器
	logRotator, err := log.NewLogRotator(rotConfig)
	if err != nil {
		return fmt.Errorf("failed to create log rotator: %w", err)
	}

	dmb.logRotator = logRotator
	return nil
}

// parseSize 解析大小字符串（例如：100M, 1G）
func parseSize(sizeStr string) (int, error) {
	if sizeStr == "" {
		return 100, nil // 默认 100MB
	}

	var size int
	var unit string
	_, err := fmt.Sscanf(sizeStr, "%d%s", &size, &unit)
	if err != nil {
		return 0, fmt.Errorf("invalid size format: %w", err)
	}

	switch strings.ToUpper(unit) {
	case "M", "MB":
		return size, nil
	case "G", "GB":
		return size * 1024, nil
	case "K", "KB":
		return size / 1024, nil
	default:
		return size, nil // 默认为 MB
	}
}

// GetLogWriter 获取日志写入器
func (dmb *DaemonManagerBase) GetLogWriter() io.Writer {
	if dmb.logRotator != nil {
		return dmb.logRotator.Writer()
	}
	return nil
}

// Cleanup 清理资源
func (dmb *DaemonManagerBase) Cleanup() error {
	if dmb.logRotator != nil {
		return dmb.logRotator.Close()
	}
	return nil
}
```

#### 步骤3：修改平台特定实现

修改 `internal/mihomo/daemon_windows.go`、`daemon_linux.go`、`daemon_darwin.go` 中的 `RedirectIO` 方法：

```go
// RedirectIO 重定向标准输入输出
func (wdm *WindowsDaemonManager) RedirectIO(cmd *exec.Cmd, logFile string) error {
	if logFile != "" {
		// 使用日志轮转管理器
		if err := wdm.InitLogRotator(); err != nil {
			return err
		}

		// 获取日志写入器
		logWriter := wdm.GetLogWriter()
		if logWriter != nil {
			cmd.Stdout = logWriter
			cmd.Stderr = logWriter
		}
	} else {
		// 重定向到 NUL
		// ... 原有代码
	}

	// 重定向 stdin 到 NUL
	// ... 原有代码

	return nil
}
```

### 2.4 测试计划

1. **单元测试**
   - 日志轮转配置解析
   - 日志文件创建和写入
   - 日志轮转触发

2. **集成测试**
   - 守护进程启动和日志输出
   - 日志文件大小达到限制时的轮转
   - 备份文件清理

3. **压力测试**
   - 大量日志写入
   - 长时间运行
   - 磁盘空间不足处理

---

## 三、自动重启实现

### 3.1 功能需求

- 检测进程崩溃
- 自动重启进程
- 限制重启次数
- 延迟重启策略
- 记录重启历史
- 支持禁用自动重启

### 3.2 技术方案

实现一个重启管理器，监控进程状态并在崩溃时自动重启。

### 3.3 实现步骤

#### 步骤1：创建重启管理器

**文件**: `internal/mihomo/restart_manager.go`

```go
package mihomo

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// RestartManager 重启管理器
type RestartManager struct {
	mu               sync.RWMutex
	config           *AutoRestartConfig
	restartCount     int
	lastRestartTime  time.Time
	restartHistory   []RestartRecord
	maxRestarts      int
	restartDelay     time.Duration
	enabled          bool
	ctx              context.Context
	cancel           context.CancelFunc
	onRestartCallback func(pid int, err error)
}

// RestartRecord 重启记录
type RestartRecord struct {
	Timestamp time.Time `json:"timestamp"`
	PID       int       `json:"pid"`
	Reason    string    `json:"reason"`
	ExitCode  int       `json:"exit_code,omitempty"`
}

// NewRestartManager 创建重启管理器
func NewRestartManager(config *AutoRestartConfig) (*RestartManager, error) {
	if config == nil {
		return &RestartManager{enabled: false}, nil
	}

	// 解析重启延迟
	delay, err := time.ParseDuration(config.RestartDelay)
	if err != nil {
		return nil, fmt.Errorf("invalid restart_delay: %w", err)
	}

	ctx, cancel := context.WithCancel(context.Background())

	return &RestartManager{
		config:          config,
		maxRestarts:     config.MaxRestarts,
		restartDelay:    delay,
		enabled:         config.Enabled,
		ctx:            ctx,
		cancel:         cancel,
		restartHistory: make([]RestartRecord, 0),
	}, nil
}

// StartMonitoring 开始监控进程
func (rm *RestartManager) StartMonitoring(pid int, exitChan <-chan error) {
	if !rm.enabled {
		return
	}

	go func() {
		for {
			select {
			case <-rm.ctx.Done():
				// 监控被停止
				return

			case err := <-exitChan:
				if err != nil {
					// 进程异常退出，尝试重启
					rm.handleProcessExit(pid, err)
				} else {
					// 进程正常退出
					rm.recordRestart(pid, "normal_exit", 0)
					return
				}
			}
		}
	}()
}

// StopMonitoring 停止监控
func (rm *RestartManager) StopMonitoring() {
	if rm.cancel != nil {
		rm.cancel()
	}
}

// handleProcessExit 处理进程退出
func (rm *RestartManager) handleProcessExit(pid int, err error) {
	rm.mu.Lock()
	defer rm.mu.Unlock()

	// 记录退出
	exitCode := -1
	if exitErr, ok := err.(*exec.ExitError); ok {
		if status, ok := exitErr.Sys().(syscall.WaitStatus); ok {
			exitCode = status.ExitStatus()
		}
	}

	rm.recordRestart(pid, "crashed", exitCode)

	// 检查是否达到最大重启次数
	if rm.restartCount >= rm.maxRestarts {
		rm.notifyRestartLimit(pid, err)
		return
	}

	// 增加重启计数
	rm.restartCount++
	rm.lastRestartTime = time.Now()

	// 延迟重启
	time.Sleep(rm.restartDelay)

	// 触发重启回调
	if rm.onRestartCallback != nil {
		rm.onRestartCallback(pid, err)
	}
}

// recordRestart 记录重启
func (rm *RestartManager) recordRestart(pid int, reason string, exitCode int) {
	record := RestartRecord{
		Timestamp: time.Now(),
		PID:       pid,
		Reason:    reason,
		ExitCode:  exitCode,
	}

	rm.restartHistory = append(rm.restartHistory, record)

	// 保留最近 100 条记录
	if len(rm.restartHistory) > 100 {
		rm.restartHistory = rm.restartHistory[1:]
	}
}

// notifyRestartLimit 通知达到重启限制
func (rm *RestartManager) notifyRestartLimit(pid int, err error) {
	// 记录日志
	output.Error("Process %d has crashed %d times, reaching the maximum restart limit", pid, rm.maxRestarts)
	output.Error("Last error: %v", err)

	// 可以在这里发送告警通知
	// 例如：邮件、Slack、钉钉等
}

// SetRestartCallback 设置重启回调
func (rm *RestartManager) SetRestartCallback(callback func(pid int, err error)) {
	rm.mu.Lock()
	defer rm.mu.Unlock()
	rm.onRestartCallback = callback
}

// GetRestartCount 获取重启次数
func (rm *RestartManager) GetRestartCount() int {
	rm.mu.RLock()
	defer rm.mu.RUnlock()
	return rm.restartCount
}

// GetRestartHistory 获取重启历史
func (rm *RestartManager) GetRestartHistory() []RestartRecord {
	rm.mu.RLock()
	defer rm.mu.RUnlock()
	return rm.restartHistory
}

// Reset 重置重启计数
func (rm *RestartManager) Reset() {
	rm.mu.Lock()
	defer rm.mu.Unlock()
	rm.restartCount = 0
	rm.lastRestartTime = time.Time{}
}
```

#### 步骤2：集成到守护进程管理器

修改 `internal/mihomo/daemon.go`：

```go
// DaemonManagerBase 守护进程管理器基类
type DaemonManagerBase struct {
	config          *DaemonConfig
	pidFile         string
	secret          string
	apiAddr         string
	execPath        string
	configFile      string
	logRotator      *log.LogRotator
	restartManager  *RestartManager // 新增：重启管理器
}

// InitRestartManager 初始化重启管理器
func (dmb *DaemonManagerBase) InitRestartManager() error {
	if dmb.config == nil || dmb.config.AutoRestart == nil || !dmb.config.AutoRestart.Enabled {
		return nil // 不需要自动重启
	}

	// 创建重启管理器
	restartManager, err := NewRestartManager(&dmb.config.AutoRestart)
	if err != nil {
		return fmt.Errorf("failed to create restart manager: %w", err)
	}

	// 设置重启回调
	restartManager.SetRestartCallback(dmb.handleRestart)

	dmb.restartManager = restartManager
	return nil
}

// handleRestart 处理重启
func (dmb *DaemonManagerBase) handleRestart(pid int, err error) {
	output.Info("Auto-restarting process (PID: %d) after crash...", pid)

	// 重新启动进程
	ctx := context.Background()
	if err := dmb.StartAsDaemon(ctx, nil); err != nil {
		output.Error("Failed to auto-restart process: %v", err)
		return
	}

	output.Success("Process auto-restarted successfully")
}

// GetRestartManager 获取重启管理器
func (dmb *DaemonManagerBase) GetRestartManager() *RestartManager {
	return dmb.restartManager
}
```

#### 步骤3：修改启动流程

修改平台特定的守护进程管理器，在启动进程后开始监控：

```go
// StartAsDaemon 以守护进程方式启动
func (wdm *WindowsDaemonManager) StartAsDaemon(ctx context.Context, cfg interface{}) error {
	// ... 原有启动代码

	// 初始化重启管理器
	if err := wdm.InitRestartManager(); err != nil {
		output.Warning("failed to initialize restart manager: %v", err)
	}

	// 启动进程
	if err := cmd.Start(); err != nil {
		return pkgerrors.ErrService("failed to start mihomo daemon", err)
	}

	pid := cmd.Process.Pid

	// 如果启用了自动重启，开始监控
	if restartManager := wdm.GetRestartManager(); restartManager != nil {
		exitChan := make(chan error, 1)
		go func() {
			err := cmd.Wait()
			exitChan <- err
		}()

		restartManager.StartMonitoring(pid, exitChan)
	}

	// ... 原有保存 PID 代码

	return nil
}
```

### 3.4 测试计划

1. **单元测试**
   - 重启计数器
   - 重启延迟
   - 重启历史记录

2. **集成测试**
   - 进程崩溃检测
   - 自动重启触发
   - 重启限制

3. **场景测试**
   - 连续崩溃
   - 延迟重启
   - 达到重启限制

---

## 四、健康监控实现

### 4.1 功能需求

- 定期健康检查
- API 可用性检查
- 资源使用监控
- 异常告警
- 自动恢复
- 健康状态报告

### 4.2 技术方案

使用现有的 API 客户端进行健康检查，并结合系统监控。

### 4.3 实现步骤

#### 步骤1：创建健康监控器

**文件**: `internal/mihomo/health_monitor.go`

```go
package mihomo

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/kkkqkx123/mihomo-cli/internal/api"
	"github.com/kkkqkx123/mihomo-cli/internal/output"
)

// HealthMonitor 健康监控器
type HealthMonitor struct {
	mu                sync.RWMutex
	config            *HealthCheckConfig
	enabled           bool
	interval          time.Duration
	timeout           time.Duration
	apiClient         *api.Client
	ctx               context.Context
	cancel            context.CancelFunc
	onUnhealthyCallback func(report *HealthReport)
	ticker            *time.Ticker
}

// HealthReport 健康报告
type HealthReport struct {
	Timestamp   time.Time `json:"timestamp"`
	Healthy     bool      `json:"healthy"`
	APIStatus   string    `json:"api_status"`
	MemoryUsage float64   `json:"memory_usage_mb"`
	CPUUsage    float64   `json:"cpu_usage_percent"`
	Error       string    `json:"error,omitempty"`
}

// NewHealthMonitor 创建健康监控器
func NewHealthMonitor(config *HealthCheckConfig, apiAddr, secret string) (*HealthMonitor, error) {
	if config == nil || !config.Enabled {
		return &HealthMonitor{enabled: false}, nil
	}

	// 解析间隔和超时
	interval, err := time.ParseDuration(config.Interval)
	if err != nil {
		return nil, fmt.Errorf("invalid interval: %w", err)
	}

	timeout, err := time.ParseDuration(config.Timeout)
	if err != nil {
		return nil, fmt.Errorf("invalid timeout: %w", err)
	}

	// 创建 API 客户端
	apiClient := api.NewClient(
		"http://"+apiAddr,
		secret,
		api.WithTimeout(timeout),
	)

	ctx, cancel := context.WithCancel(context.Background())

	return &HealthMonitor{
		config:    config,
		enabled:   true,
		interval:  interval,
		timeout:   timeout,
		apiClient: apiClient,
		ctx:       ctx,
		cancel:    cancel,
	}, nil
}

// Start 开始健康监控
func (hm *HealthMonitor) Start(pid int) {
	if !hm.enabled {
		return
	}

	hm.ticker = time.NewTicker(hm.interval)

	go func() {
		defer hm.ticker.Stop()

		for {
			select {
			case <-hm.ctx.Done():
				// 监控被停止
				return

			case <-hm.ticker.C:
				// 执行健康检查
				report := hm.CheckHealth()

				// 如果不健康，触发回调
				if !report.Healthy && hm.onUnhealthyCallback != nil {
					hm.onUnhealthyCallback(report)
				}
			}
		}
	}()
}

// Stop 停止健康监控
func (hm *HealthMonitor) Stop() {
	if hm.cancel != nil {
		hm.cancel()
	}
	if hm.ticker != nil {
		hm.ticker.Stop()
	}
}

// CheckHealth 执行健康检查
func (hm *HealthMonitor) CheckHealth() *HealthReport {
	report := &HealthReport{
		Timestamp: time.Now(),
		Healthy:   true,
	}

	// 检查 API 可用性
	ctx, cancel := context.WithTimeout(context.Background(), hm.timeout)
	defer cancel()

	_, err := hm.apiClient.GetMode(ctx)
	if err != nil {
		report.Healthy = false
		report.APIStatus = "unavailable"
		report.Error = fmt.Sprintf("API check failed: %v", err)
		return report
	}

	report.APIStatus = "available"

	// 获取资源使用情况（需要实现）
	// report.MemoryUsage = ...
	// report.CPUUsage = ...

	return report
}

// SetUnhealthyCallback 设置不健康回调
func (hm *HealthMonitor) SetUnhealthyCallback(callback func(report *HealthReport)) {
	hm.mu.Lock()
	defer hm.mu.Unlock()
	hm.onUnhealthyCallback = callback
}

// GetLastReport 获取最后一次健康报告
func (hm *HealthMonitor) GetLastReport() *HealthReport {
	// 可以缓存最后一次报告
	return nil
}
```

#### 步骤2：集成到守护进程管理器

修改 `internal/mihomo/daemon.go`：

```go
// DaemonManagerBase 守护进程管理器基类
type DaemonManagerBase struct {
	config          *DaemonConfig
	pidFile         string
	secret          string
	apiAddr         string
	execPath        string
	configFile      string
	logRotator      *log.LogRotator
	restartManager  *RestartManager
	healthMonitor   *HealthMonitor // 新增：健康监控器
}

// InitHealthMonitor 初始化健康监控器
func (dmb *DaemonManagerBase) InitHealthMonitor() error {
	if dmb.config == nil || dmb.config.HealthCheck == nil || !dmb.config.HealthCheck.Enabled {
		return nil // 不需要健康监控
	}

	// 创建健康监控器
	healthMonitor, err := NewHealthMonitor(&dmb.config.HealthCheck, dmb.apiAddr, dmb.secret)
	if err != nil {
		return fmt.Errorf("failed to create health monitor: %w", err)
	}

	// 设置不健康回调
	healthMonitor.SetUnhealthyCallback(dmb.handleUnhealthy)

	dmb.healthMonitor = healthMonitor
	return nil
}

// handleUnhealthy 处理不健康状态
func (dmb *DaemonManagerBase) handleUnhealthy(report *HealthReport) {
	output.Warning("Health check failed at %s", report.Timestamp.Format(time.RFC3339))
	output.Warning("Error: %s", report.Error)

	// 可以在这里尝试恢复或发送告警
	// 例如：重启进程、发送通知等
}

// GetHealthMonitor 获取健康监控器
func (dmb *DaemonManagerBase) GetHealthMonitor() *HealthMonitor {
	return dmb.healthMonitor
}
```

#### 步骤3：修改启动流程

```go
// StartAsDaemon 以守护进程方式启动
func (wdm *WindowsDaemonManager) StartAsDaemon(ctx context.Context, cfg interface{}) error {
	// ... 原有启动代码

	pid := cmd.Process.Pid

	// 初始化健康监控器
	if err := wdm.InitHealthMonitor(); err != nil {
		output.Warning("failed to initialize health monitor: %v", err)
	}

	// 启动健康监控
	if healthMonitor := wdm.GetHealthMonitor(); healthMonitor != nil {
		healthMonitor.Start(pid)
	}

	// ... 原有保存 PID 代码

	return nil
}
```

### 4.4 测试计划

1. **单元测试**
   - 健康检查逻辑
   - 告警触发
   - 资源监控

2. **集成测试**
   - 定期健康检查
   - API 不可用检测
   - 自动恢复

3. **场景测试**
   - 网络中断
   - API 响应慢
   - 资源耗尽

---

## 五、多实例支持

### 5.1 功能需求

- 支持同时运行多个 Mihomo 实例
- 实例隔离（配置、日志、PID）
- 统一管理
- 实例标识
- 实例状态查询

### 5.2 技术方案

使用实例名称或 ID 来区分不同的实例，每个实例有独立的配置和资源。

### 5.3 实现步骤

#### 步骤1：创建实例管理器

**文件**: `internal/mihomo/instance_manager.go`

```go
package mihomo

import (
	"context"
	"fmt"
	"sync"
)

// InstanceManager 实例管理器
type InstanceManager struct {
	mu        sync.RWMutex
	instances map[string]*Instance
}

// Instance 实例信息
type Instance struct {
	ID          string            `json:"id"`
	Name        string            `json:"name"`
	PID         int               `json:"pid"`
	Config      *DaemonConfig     `json:"config"`
	APIAddress  string            `json:"api_address"`
	Secret      string            `json:"secret"`
	Status      string            `json:"status"`
	CreatedAt   time.Time         `json:"created_at"`
	StartedAt   time.Time         `json:"started_at"`
	Metadata    map[string]string `json:"metadata"`
}

// NewInstanceManager 创建实例管理器
func NewInstanceManager() *InstanceManager {
	return &InstanceManager{
		instances: make(map[string]*Instance),
	}
}

// CreateInstance 创建新实例
func (im *InstanceManager) CreateInstance(name string, config *DaemonConfig) (*Instance, error) {
	im.mu.Lock()
	defer im.mu.Unlock()

	// 检查实例名称是否已存在
	if _, exists := im.instances[name]; exists {
		return nil, fmt.Errorf("instance '%s' already exists", name)
	}

	// 生成实例 ID
	id := generateInstanceID()

	// 创建实例
	instance := &Instance{
		ID:        id,
		Name:      name,
		Config:    config,
		Status:    "created",
		CreatedAt: time.Now(),
		Metadata:  make(map[string]string),
	}

	im.instances[name] = instance
	return instance, nil
}

// StartInstance 启动实例
func (im *InstanceManager) StartInstance(name string) error {
	im.mu.Lock()
	defer im.mu.Unlock()

	instance, exists := im.instances[name]
	if !exists {
		return fmt.Errorf("instance '%s' not found", name)
	}

	if instance.Status == "running" {
		return fmt.Errorf("instance '%s' is already running", name)
	}

	// 启动实例（这里需要调用守护进程管理器）
	// ...

	instance.Status = "running"
	instance.StartedAt = time.Now()

	return nil
}

// StopInstance 停止实例
func (im *InstanceManager) StopInstance(name string) error {
	im.mu.Lock()
	defer im.mu.Unlock()

	instance, exists := im.instances[name]
	if !exists {
		return fmt.Errorf("instance '%s' not found", name)
	}

	if instance.Status != "running" {
		return fmt.Errorf("instance '%s' is not running", name)
	}

	// 停止实例（这里需要调用守护进程管理器）
	// ...

	instance.Status = "stopped"
	return nil
}

// GetInstance 获取实例
func (im *InstanceManager) GetInstance(name string) (*Instance, error) {
	im.mu.RLock()
	defer im.mu.RUnlock()

	instance, exists := im.instances[name]
	if !exists {
		return nil, fmt.Errorf("instance '%s' not found", name)
	}

	return instance, nil
}

// ListInstances 列出所有实例
func (im *InstanceManager) ListInstances() []*Instance {
	im.mu.RLock()
	defer im.mu.RUnlock()

	instances := make([]*Instance, 0, len(im.instances))
	for _, instance := range im.instances {
		instances = append(instances, instance)
	}

	return instances
}

// DeleteInstance 删除实例
func (im *InstanceManager) DeleteInstance(name string) error {
	im.mu.Lock()
	defer im.mu.Unlock()

	if _, exists := im.instances[name]; !exists {
		return fmt.Errorf("instance '%s' not found", name)
	}

	delete(im.instances, name)
	return nil
}

// generateInstanceID 生成实例 ID
func generateInstanceID() string {
	return fmt.Sprintf("mihomo-%d", time.Now().UnixNano())
}
```

#### 步骤2：修改配置支持多实例

修改配置文件结构，支持多实例配置：

```toml
# 主配置
[api]
address = "http://127.0.0.1:9090"
secret = ""

# 实例1配置
[[instances]]
name = "primary"
enabled = true
executable = "/usr/local/bin/mihomo"
config_file = "/etc/mihomo/primary.yaml"

[instances.daemon]
enabled = true
log_file = "/var/log/mihomo/primary.log"

[instances.api]
external_controller = "127.0.0.1:9091"

# 实例2配置
[[instances]]
name = "secondary"
enabled = true
executable = "/usr/local/bin/mihomo"
config_file = "/etc/mihomo/secondary.yaml"

[instances.daemon]
enabled = true
log_file = "/var/log/mihomo/secondary.log"

[instances.api]
external_controller = "127.0.0.1:9092"
```

#### 步骤3：实现 CLI 命令

添加新的 CLI 命令来管理多实例：

```bash
# 列出所有实例
mihomo-cli instance list

# 启动指定实例
mihomo-cli instance start primary

# 停止指定实例
mihomo-cli instance stop secondary

# 查看实例状态
mihomo-cli instance status primary

# 删除实例
mihomo-cli instance delete primary
```

### 5.4 测试计划

1. **单元测试**
   - 实例创建和管理
   - 实例隔离
   - 状态查询

2. **集成测试**
   - 多实例同时运行
   - 实例间独立操作
   - 资源隔离

3. **场景测试**
   - 启动多个实例
   - 停止部分实例
   - 实例崩溃恢复

---

## 六、系统集成

### 6.1 功能需求

#### Windows Service
- 安装为 Windows 服务
- 服务启动/停止
- 服务状态查询
- 自动启动配置

#### systemd (Linux)
- 创建 systemd service 文件
- 服务启用/禁用
- 服务启动/停止/重启
- 日志集成（journald）

#### launchd (macOS)
- 创建 launchd plist 文件
- 服务加载/卸载
- 服务启动/停止
- 登录项管理

### 6.2 实现步骤

#### 步骤1：创建服务管理器接口

**文件**: `internal/service/service_manager.go`

```go
package service

import "context"

// ServiceManager 服务管理器接口
type ServiceManager interface {
	// Install 安装服务
	Install(ctx context.Context, config *ServiceConfig) error

	// Uninstall 卸载服务
	Uninstall(ctx context.Context) error

	// Start 启动服务
	Start(ctx context.Context) error

	// Stop 停止服务
	Stop(ctx context.Context) error

	// Restart 重启服务
	Restart(ctx context.Context) error

	// Status 获取服务状态
	Status(ctx context.Context) (*ServiceStatus, error)

	// Enable 启用服务（开机自启）
	Enable(ctx context.Context) error

	// Disable 禁用服务
	Disable(ctx context.Context) error
}

// ServiceConfig 服务配置
type ServiceConfig struct {
	Name        string
	DisplayName string
	Description string
	Executable  string
	Arguments   []string
	WorkingDir  string
	AutoStart   bool
}

// ServiceStatus 服务状态
type ServiceStatus struct {
	Running  bool
	Enabled  bool
	PID      int
	Uptime   time.Duration
}
```

#### 步骤2：实现 Windows Service

**文件**: `internal/service/service_windows.go`

```go
//go:build windows

package service

import (
	"context"
	"fmt"
	"os/exec"
	"path/filepath"
	"syscall"
	"time"
)

// WindowsServiceManager Windows 服务管理器
type WindowsServiceManager struct {
	name string
}

// NewWindowsServiceManager 创建 Windows 服务管理器
func NewWindowsServiceManager(name string) *WindowsServiceManager {
	return &WindowsServiceManager{name: name}
}

// Install 安装服务
func (wsm *WindowsServiceManager) Install(ctx context.Context, config *ServiceConfig) error {
	// 使用 sc.exe 命令安装服务
	args := []string{
		"create", config.Name,
		"binPath=", fmt.Sprintf("\"%s\" %s", config.Executable, strings.Join(config.Arguments, " ")),
		"DisplayName=", config.DisplayName,
		"Description=", config.Description,
		"start=", "auto",
	}

	cmd := exec.CommandContext(ctx, "sc.exe", args...)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to create service: %w", err)
	}

	return nil
}

// Uninstall 卸载服务
func (wsm *WindowsServiceManager) Uninstall(ctx context.Context) error {
	// 先停止服务
	_ = wsm.Stop(ctx)

	// 删除服务
	cmd := exec.CommandContext(ctx, "sc.exe", "delete", wsm.name)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to delete service: %w", err)
	}

	return nil
}

// Start 启动服务
func (wsm *WindowsServiceManager) Start(ctx context.Context) error {
	cmd := exec.CommandContext(ctx, "sc.exe", "start", wsm.name)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to start service: %w", err)
	}
	return nil
}

// Stop 停止服务
func (wsm *WindowsServiceManager) Stop(ctx context.Context) error {
	cmd := exec.CommandContext(ctx, "sc.exe", "stop", wsm.name)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to stop service: %w", err)
	}
	return nil
}

// Status 获取服务状态
func (wsm *WindowsServiceManager) Status(ctx context.Context) (*ServiceStatus, error) {
	cmd := exec.CommandContext(ctx, "sc.exe", "query", wsm.name)
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to query service: %w", err)
	}

	// 解析输出获取状态
	status := &ServiceStatus{
		Running: false,
		Enabled: false,
	}

	if strings.Contains(string(output), "RUNNING") {
		status.Running = true
	}

	return status, nil
}

// Enable 启用服务
func (wsm *WindowsServiceManager) Enable(ctx context.Context) error {
	cmd := exec.CommandContext(ctx, "sc.exe", "config", wsm.name, "start=", "auto")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to enable service: %w", err)
	}
	return nil
}

// Disable 禁用服务
func (wsm *WindowsServiceManager) Disable(ctx context.Context) error {
	cmd := exec.CommandContext(ctx, "sc.exe", "config", wsm.name, "start=", "demand")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to disable service: %w", err)
	}
	return nil
}
```

#### 步骤3：实现 systemd Service

**文件**: `internal/service/service_linux.go`

```go
//go:build linux

package service

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"text/template"
)

// SystemdServiceManager systemd 服务管理器
type SystemdServiceManager struct {
	name string
}

// NewSystemdServiceManager 创建 systemd 服务管理器
func NewSystemdServiceManager(name string) *SystemdServiceManager {
	return &SystemdServiceManager{name: name}
}

// Install 安装服务
func (ssm *SystemdServiceManager) Install(ctx context.Context, config *ServiceConfig) error {
	// 生成 systemd service 文件
	serviceFile := fmt.Sprintf("/etc/systemd/system/%s.service", ssm.name)

	if err := ssm.generateServiceFile(serviceFile, config); err != nil {
		return fmt.Errorf("failed to generate service file: %w", err)
	}

	// 重新加载 systemd
	cmd := exec.CommandContext(ctx, "systemctl", "daemon-reload")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to reload systemd: %w", err)
	}

	return nil
}

// generateServiceFile 生成 systemd service 文件
func (ssm *SystemdServiceManager) generateServiceFile(path string, config *ServiceConfig) error {
	const serviceTemplate = `[Unit]
Description={{.Description}}
After=network.target

[Service]
Type=simple
ExecStart={{.Executable}} {{range .Arguments}}{{.}} {{end}}
WorkingDirectory={{.WorkingDir}}
Restart=on-failure
RestartSec=5s

[Install]
WantedBy=multi-user.target
`

	tmpl, err := template.New("service").Parse(serviceTemplate)
	if err != nil {
		return err
	}

	// 确保目录存在
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}

	// 创建文件
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()

	// 执行模板
	return tmpl.Execute(file, config)
}

// Uninstall 卸载服务
func (ssm *SystemdServiceManager) Uninstall(ctx context.Context) error {
	// 先停止并禁用服务
	_ = ssm.Stop(ctx)
	_ = ssm.Disable(ctx)

	// 删除服务文件
	serviceFile := fmt.Sprintf("/etc/systemd/system/%s.service", ssm.name)
	if err := os.Remove(serviceFile); err != nil {
		return fmt.Errorf("failed to remove service file: %w", err)
	}

	// 重新加载 systemd
	cmd := exec.CommandContext(ctx, "systemctl", "daemon-reload")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to reload systemd: %w", err)
	}

	return nil
}

// Start 启动服务
func (ssm *SystemdServiceManager) Start(ctx context.Context) error {
	cmd := exec.CommandContext(ctx, "systemctl", "start", ssm.name)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to start service: %w", err)
	}
	return nil
}

// Stop 停止服务
func (ssm *SystemdServiceManager) Stop(ctx context.Context) error {
	cmd := exec.CommandContext(ctx, "systemctl", "stop", ssm.name)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to stop service: %w", err)
	}
	return nil
}

// Status 获取服务状态
func (ssm *SystemdServiceManager) Status(ctx context.Context) (*ServiceStatus, error) {
	cmd := exec.CommandContext(ctx, "systemctl", "status", ssm.name)
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to query service: %w", err)
	}

	status := &ServiceStatus{
		Running: strings.Contains(string(output), "active (running)"),
		Enabled: strings.Contains(string(output), "enabled"),
	}

	return status, nil
}

// Enable 启用服务
func (ssm *SystemdServiceManager) Enable(ctx context.Context) error {
	cmd := exec.CommandContext(ctx, "systemctl", "enable", ssm.name)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to enable service: %w", err)
	}
	return nil
}

// Disable 禁用服务
func (ssm *SystemdServiceManager) Disable(ctx context.Context) error {
	cmd := exec.CommandContext(ctx, "systemctl", "disable", ssm.name)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to disable service: %w", err)
	}
	return nil
}
```

#### 步骤4：实现 launchd Service

**文件**: `internal/service/service_darwin.go`

```go
//go:build darwin

package service

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

// LaunchdServiceManager launchd 服务管理器
type LaunchdServiceManager struct {
	label     string
	plistPath string
}

// NewLaunchdServiceManager 创建 launchd 服务管理器
func NewLaunchdServiceManager(label, plistPath string) *LaunchdServiceManager {
	return &LaunchdServiceManager{
		label:     label,
		plistPath: plistPath,
	}
}

// Install 安装服务
func (lsm *LaunchdServiceManager) Install(ctx context.Context, config *ServiceConfig) error {
	// 生成 launchd plist 文件
	if err := lsm.generatePlistFile(config); err != nil {
		return fmt.Errorf("failed to generate plist file: %w", err)
	}

	// 加载服务
	cmd := exec.CommandContext(ctx, "launchctl", "load", lsm.plistPath)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to load service: %w", err)
	}

	return nil
}

// generatePlistFile 生成 launchd plist 文件
func (lsm *LaunchdServiceManager) generatePlistFile(config *ServiceConfig) error {
	plistContent := fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>Label</key>
    <string>%s</string>
    <key>ProgramArguments</key>
    <array>
        <string>%s</string>
        %s
    </array>
    <key>RunAtLoad</key>
    <true/>
    <key>KeepAlive</key>
    <true/>
    <key>WorkingDirectory</key>
    <string>%s</string>
</dict>
</plist>`,
		lsm.label,
		config.Executable,
		lsm.argumentsToXML(config.Arguments),
		config.WorkingDir,
	)

	// 确保目录存在
	if err := os.MkdirAll(filepath.Dir(lsm.plistPath), 0755); err != nil {
		return err
	}

	return os.WriteFile(lsm.plistPath, []byte(plistContent), 0644)
}

// argumentsToXML 将参数转换为 XML 格式
func (lsm *LaunchdServiceManager) argumentsToXML(args []string) string {
	var xml string
	for _, arg := range args {
		xml += fmt.Sprintf(`<string>%s</string>\n        `, arg)
	}
	return xml
}

// Uninstall 卸载服务
func (lsm *LaunchdServiceManager) Uninstall(ctx context.Context) error {
	// 先停止服务
	_ = lsm.Stop(ctx)

	// 卸载服务
	cmd := exec.CommandContext(ctx, "launchctl", "unload", lsm.plistPath)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to unload service: %w", err)
	}

	// 删除 plist 文件
	if err := os.Remove(lsm.plistPath); err != nil {
		return fmt.Errorf("failed to remove plist file: %w", err)
	}

	return nil
}

// Start 启动服务
func (lsm *LaunchdServiceManager) Start(ctx context.Context) error {
	cmd := exec.CommandContext(ctx, "launchctl", "start", lsm.label)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to start service: %w", err)
	}
	return nil
}

// Stop 停止服务
func (lsm *LaunchdServiceManager) Stop(ctx context.Context) error {
	cmd := exec.CommandContext(ctx, "launchctl", "stop", lsm.label)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to stop service: %w", err)
	}
	return nil
}

// Status 获取服务状态
func (lsm *LaunchdServiceManager) Status(ctx context.Context) (*ServiceStatus, error) {
	cmd := exec.CommandContext(ctx, "launchctl", "list", lsm.label)
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to query service: %w", err)
	}

	status := &ServiceStatus{
		Running: len(output) > 0,
		Enabled: true, // launchd 服务默认启用
	}

	return status, nil
}

// Enable 启用服务（launchd 不需要显式启用）
func (lsm *LaunchdServiceManager) Enable(ctx context.Context) error {
	return nil
}

// Disable 禁用服务（launchd 不需要显式禁用）
func (lsm *LaunchdServiceManager) Disable(ctx context.Context) error {
	return nil
}
```

#### 步骤5：实现 CLI 命令

修改 `cmd/service.go` 来支持服务管理：

```go
var serviceInstallCmd = &cobra.Command{
	Use:   "install",
	Short: "安装为系统服务",
	RunE: func(cmd *cobra.Command, args []string) error {
		// 获取服务管理器
		sm := service.GetServiceManager("mihomo")

		// 配置服务
		config := &service.ServiceConfig{
			Name:        "mihomo",
			DisplayName: "Mihomo Proxy Service",
			Description: "Mihomo proxy core service",
			Executable:  "/usr/local/bin/mihomo-cli",
			Arguments:   []string{"daemon", "start"},
			WorkingDir:  "/var/lib/mihomo",
			AutoStart:   true,
		}

		// 安装服务
		return sm.Install(context.Background(), config)
	},
}
```

### 6.3 测试计划

1. **Windows Service 测试**
   - 服务安装和卸载
   - 服务启动和停止
   - 服务状态查询
   - 自动启动配置

2. **systemd 测试**
   - 服务文件生成
   - 服务启用和禁用
   - 日志集成
   - 权限管理

3. **launchd 测试**
   - plist 文件生成
   - 服务加载和卸载
   - 登录项管理
   - 权限管理

---

## 七、实施时间表

### 第一阶段：日志轮转（1-2 周）
- 实现日志轮转管理器
- 集成到守护进程管理器
- 测试和文档

### 第二阶段：自动重启（1 周）
- 实现重启管理器
- 集成到守护进程管理器
- 测试和文档

### 第三阶段：健康监控（1 周）
- 实现健康监控器
- 集成到守护进程管理器
- 测试和文档

### 第四阶段：多实例支持（2 周）
- 实现实例管理器
- 修改配置支持
- 实现 CLI 命令
- 测试和文档

### 第五阶段：系统集成（2 周）
- 实现服务管理器
- 实现平台特定支持
- 实现 CLI 命令
- 测试和文档

### 总计：7-8 周

---

## 八、依赖管理

需要添加以下依赖：

```go
// go.mod
require (
	gopkg.in/natefinch/lumberjack.v2 v2.2.1  // 日志轮转
)
```

---

## 九、风险评估

| 风险 | 影响 | 概率 | 缓解措施 |
|------|------|------|----------|
| 日志轮转失败 | 中 | 低 | 错误处理和回退机制 |
| 自动重启循环 | 高 | 中 | 重启次数限制和延迟 |
| 健康检查误报 | 中 | 中 | 多次检查和阈值配置 |
| 多实例资源冲突 | 高 | 低 | 端口和 PID 隔离 |
| 系统服务权限问题 | 高 | 中 | 权限检查和提示 |

---

## 十、总结

### 10.1 预期效果

- ✅ 完善的日志管理
- ✅ 自动故障恢复
- ✅ 实时健康监控
- ✅ 多实例支持
- ✅ 系统级集成

### 10.2 后续优化

- 性能优化
- 安全加固
- 监控告警
- 自动化运维

---

**文档结束**
