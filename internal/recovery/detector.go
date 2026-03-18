package recovery

import (
	"fmt"
	"sync"

	"github.com/kkkqkx123/mihomo-cli/internal/system"
)

// ProblemChecker 问题检查器接口
type ProblemChecker interface {
	Name() string
	Check() (*system.Problem, error)
}

// ProblemDetector 问题检测器
type ProblemDetector struct {
	checkers []ProblemChecker
	mu       sync.RWMutex
}

// NewProblemDetector 创建问题检测器
func NewProblemDetector() *ProblemDetector {
	return &ProblemDetector{
		checkers: []ProblemChecker{},
	}
}

// RegisterChecker 注册检查器
func (pd *ProblemDetector) RegisterChecker(checker ProblemChecker) {
	pd.mu.Lock()
	defer pd.mu.Unlock()
	pd.checkers = append(pd.checkers, checker)
}

// CheckAll 执行所有检查
func (pd *ProblemDetector) CheckAll() ([]*system.Problem, error) {
	pd.mu.RLock()
	checkers := make([]ProblemChecker, len(pd.checkers))
	copy(checkers, pd.checkers)
	pd.mu.RUnlock()

	var problems []*system.Problem
	var errors []error

	for _, checker := range checkers {
		problem, err := checker.Check()
		if err != nil {
			errors = append(errors, fmt.Errorf("checker %s failed: %w", checker.Name(), err))
			continue
		}
		if problem != nil {
			problems = append(problems, problem)
		}
	}

	if len(errors) > 0 {
		return problems, fmt.Errorf("some checkers failed: %v", errors)
	}

	return problems, nil
}

// CheckByType 按类型检查
func (pd *ProblemDetector) CheckByType(problemType system.ProblemType) ([]*system.Problem, error) {
	pd.mu.RLock()
	checkers := make([]ProblemChecker, len(pd.checkers))
	copy(checkers, pd.checkers)
	pd.mu.RUnlock()

	var problems []*system.Problem
	for _, checker := range checkers {
		problem, err := checker.Check()
		if err != nil {
			continue
		}
		if problem != nil && problem.Type == problemType {
			problems = append(problems, problem)
		}
	}

	return problems, nil
}

// GetCheckerNames 获取所有检查器名称
func (pd *ProblemDetector) GetCheckerNames() []string {
	pd.mu.RLock()
	defer pd.mu.RUnlock()

	names := make([]string, len(pd.checkers))
	for i, checker := range pd.checkers {
		names[i] = checker.Name()
	}
	return names
}

// SysProxyChecker 系统代理检查器
type SysProxyChecker struct {
	manager *system.SystemConfigManager
}

// NewSysProxyChecker 创建系统代理检查器
func NewSysProxyChecker(manager *system.SystemConfigManager) *SysProxyChecker {
	return &SysProxyChecker{
		manager: manager,
	}
}

// Name 返回检查器名称
func (spc *SysProxyChecker) Name() string {
	return "sysproxy"
}

// Check 执行检查
func (spc *SysProxyChecker) Check() (*system.Problem, error) {
	return spc.manager.GetSysProxyManager().CheckResidual()
}

// TUNChecker TUN 设备检查器
type TUNChecker struct {
	manager *system.SystemConfigManager
}

// NewTUNChecker 创建 TUN 检查器
func NewTUNChecker(manager *system.SystemConfigManager) *TUNChecker {
	return &TUNChecker{
		manager: manager,
	}
}

// Name 返回检查器名称
func (tc *TUNChecker) Name() string {
	return "tun"
}

// Check 执行检查
func (tc *TUNChecker) Check() (*system.Problem, error) {
	return tc.manager.GetTUNManager().CheckResidual()
}

// RouteChecker 路由表检查器
type RouteChecker struct {
	manager *system.SystemConfigManager
}

// NewRouteChecker 创建路由表检查器
func NewRouteChecker(manager *system.SystemConfigManager) *RouteChecker {
	return &RouteChecker{
		manager: manager,
	}
}

// Name 返回检查器名称
func (rc *RouteChecker) Name() string {
	return "route"
}

// Check 执行检查
func (rc *RouteChecker) Check() (*system.Problem, error) {
	return rc.manager.GetRouteManager().CheckResidual()
}

// ConfigChecker 配置检查器
type ConfigChecker struct {
	configFile string
}

// NewConfigChecker 创建配置检查器
func NewConfigChecker(configFile string) *ConfigChecker {
	return &ConfigChecker{
		configFile: configFile,
	}
}

// Name 返回检查器名称
func (cc *ConfigChecker) Name() string {
	return "config"
}

// Check 执行检查
func (cc *ConfigChecker) Check() (*system.Problem, error) {
	// TODO: 实现配置检查
	// 检查配置文件是否存在
	// 检查配置文件格式是否正确
	// 检查配置文件是否与当前状态一致
	return nil, nil
}

// ProcessChecker 进程检查器
type ProcessChecker struct {
	pid int
}

// NewProcessChecker 创建进程检查器
func NewProcessChecker(pid int) *ProcessChecker {
	return &ProcessChecker{
		pid: pid,
	}
}

// Name 返回检查器名称
func (pc *ProcessChecker) Name() string {
	return "process"
}

// Check 执行检查
func (pc *ProcessChecker) Check() (*system.Problem, error) {
	// TODO: 实现进程检查
	// 检查进程是否在运行
	// 检查进程资源使用情况
	// 检查进程健康状态
	return nil, nil
}
