package mihomo

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// MonitorCallback 监控回调接口
type MonitorCallback interface {
	OnProcessExit(pid int)
	OnHealthCheckFailed(pid int, err error)
	OnResourceUsage(pid int, cpu, memory float64)
}

// MonitorFunc 监控回调函数类型
type MonitorFunc func(pid int, event string, data interface{})

// ProcessMonitor 进程监控器
type ProcessMonitor struct {
	pid       int
	interval  time.Duration
	callbacks []MonitorCallback
	funcs     []MonitorFunc
	stopChan  chan struct{}
	wg        sync.WaitGroup
	mu        sync.RWMutex
	running   bool
}

// NewProcessMonitor 创建进程监控器
func NewProcessMonitor(pid int, interval time.Duration) *ProcessMonitor {
	if interval <= 0 {
		interval = 5 * time.Second
	}

	return &ProcessMonitor{
		pid:      pid,
		interval: interval,
		stopChan: make(chan struct{}),
	}
}

// Start 开始监控
func (pm *ProcessMonitor) Start() error {
	pm.mu.Lock()
	if pm.running {
		pm.mu.Unlock()
		return fmt.Errorf("monitor is already running")
	}
	pm.running = true
	pm.mu.Unlock()

	pm.wg.Add(1)
	go pm.monitorLoop()

	return nil
}

// Stop 停止监控
func (pm *ProcessMonitor) Stop() {
	pm.mu.Lock()
	if !pm.running {
		pm.mu.Unlock()
		return
	}
	pm.running = false
	pm.mu.Unlock()

	close(pm.stopChan)
	pm.wg.Wait()
}

// RegisterCallback 注册回调
func (pm *ProcessMonitor) RegisterCallback(callback MonitorCallback) {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	pm.callbacks = append(pm.callbacks, callback)
}

// RegisterFunc 注册回调函数
func (pm *ProcessMonitor) RegisterFunc(fn MonitorFunc) {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	pm.funcs = append(pm.funcs, fn)
}

// monitorLoop 监控循环
func (pm *ProcessMonitor) monitorLoop() {
	defer pm.wg.Done()

	ticker := time.NewTicker(pm.interval)
	defer ticker.Stop()

	for {
		select {
		case <-pm.stopChan:
			return
		case <-ticker.C:
			pm.checkProcess()
		}
	}
}

// checkProcess 检查进程状态
func (pm *ProcessMonitor) checkProcess() {
	// 检查进程是否还在运行
	if !IsProcessRunning(pm.pid) {
		pm.notifyCallbacks("exit", nil)
		pm.notifyFuncs(pm.pid, "exit", nil)
		return
	}

	// 获取资源使用情况
	cpu, memory, err := getProcessResourceUsage(pm.pid)
	if err == nil {
		pm.notifyCallbacks("resource", map[string]float64{
			"cpu":    cpu,
			"memory": memory,
		})
		pm.notifyFuncs(pm.pid, "resource", map[string]float64{
			"cpu":    cpu,
			"memory": memory,
		})
	}
}

// notifyCallbacks 通知回调
func (pm *ProcessMonitor) notifyCallbacks(event string, data interface{}) {
	pm.mu.RLock()
	callbacks := make([]MonitorCallback, len(pm.callbacks))
	copy(callbacks, pm.callbacks)
	pm.mu.RUnlock()

	for _, callback := range callbacks {
		switch event {
		case "exit":
			callback.OnProcessExit(pm.pid)
		case "health_failed":
			if err, ok := data.(error); ok {
				callback.OnHealthCheckFailed(pm.pid, err)
			}
		case "resource":
			if usage, ok := data.(map[string]float64); ok {
				callback.OnResourceUsage(pm.pid, usage["cpu"], usage["memory"])
			}
		}
	}
}

// notifyFuncs 通知回调函数
func (pm *ProcessMonitor) notifyFuncs(pid int, event string, data interface{}) {
	pm.mu.RLock()
	funcs := make([]MonitorFunc, len(pm.funcs))
	copy(funcs, pm.funcs)
	pm.mu.RUnlock()

	for _, fn := range funcs {
		fn(pid, event, data)
	}
}

// HealthCheckMonitor 健康检查监控器
type HealthCheckMonitor struct {
	pm         *ProcessMonitor
	checkFunc  func(ctx context.Context) error
	interval   time.Duration
	stopChan   chan struct{}
	wg         sync.WaitGroup
	mu         sync.RWMutex
	running    bool
}

// NewHealthCheckMonitor 创建健康检查监控器
func NewHealthCheckMonitor(pid int, checkFunc func(ctx context.Context) error, interval time.Duration) *HealthCheckMonitor {
	if interval <= 0 {
		interval = 10 * time.Second
	}

	return &HealthCheckMonitor{
		pm:        NewProcessMonitor(pid, interval),
		checkFunc: checkFunc,
		interval:  interval,
		stopChan:  make(chan struct{}),
	}
}

// Start 开始监控
func (hcm *HealthCheckMonitor) Start() error {
	hcm.mu.Lock()
	if hcm.running {
		hcm.mu.Unlock()
		return fmt.Errorf("health check monitor is already running")
	}
	hcm.running = true
	hcm.mu.Unlock()

	hcm.wg.Add(1)
	go hcm.healthCheckLoop()

	return hcm.pm.Start()
}

// Stop 停止监控
func (hcm *HealthCheckMonitor) Stop() {
	hcm.mu.Lock()
	if !hcm.running {
		hcm.mu.Unlock()
		return
	}
	hcm.running = false
	hcm.mu.Unlock()

	close(hcm.stopChan)
	hcm.pm.Stop()
	hcm.wg.Wait()
}

// healthCheckLoop 健康检查循环
func (hcm *HealthCheckMonitor) healthCheckLoop() {
	defer hcm.wg.Done()

	ticker := time.NewTicker(hcm.interval)
	defer ticker.Stop()

	for {
		select {
		case <-hcm.stopChan:
			return
		case <-ticker.C:
			hcm.performHealthCheck()
		}
	}
}

// performHealthCheck 执行健康检查
func (hcm *HealthCheckMonitor) performHealthCheck() {
	if hcm.checkFunc == nil {
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	if err := hcm.checkFunc(ctx); err != nil {
		hcm.pm.notifyCallbacks("health_failed", err)
		hcm.pm.notifyFuncs(hcm.pm.pid, "health_failed", err)
	}
}

// RegisterCallback 注册回调
func (hcm *HealthCheckMonitor) RegisterCallback(callback MonitorCallback) {
	hcm.pm.RegisterCallback(callback)
}

// RegisterFunc 注册回调函数
func (hcm *HealthCheckMonitor) RegisterFunc(fn MonitorFunc) {
	hcm.pm.RegisterFunc(fn)
}
