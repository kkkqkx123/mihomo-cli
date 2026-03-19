package system

import (
	"fmt"
	"strings"
)

// CleanupReport 清理报告
type CleanupReport struct {
	TotalFound   int          `json:"total_found"`
	TotalRemoved int          `json:"total_removed"`
	Removed      []RouteEntry `json:"removed"`
	Failed       []RouteEntry `json:"failed"`
	Skipped      []RouteEntry `json:"skipped"`
	LastError    error        `json:"last_error,omitempty"`
}

// FixReport 修复报告
type FixReport struct {
	Success       bool              `json:"success"`
	Message       string            `json:"message"`
	CleanupReport *CleanupReport    `json:"cleanup_report,omitempty"`
	Diagnosis     *NetworkDiagnosis `json:"diagnosis,omitempty"`
	Errors        []error           `json:"errors,omitempty"`
}

// CleanupMihomoRoutes 清理 Mihomo 添加的路由
func (rm *RouteManager) CleanupMihomoRoutes() error {
	routes, err := rm.ListRoutes()
	if err != nil {
		return err
	}

	var lastErr error
	for _, route := range routes {
		// 检查是否是 Mihomo 添加的路由
		if isMihomoRoute(route) {
			if err := rm.DeleteRoute(route); err != nil {
				lastErr = err
			}
		}
	}

	return lastErr
}

// CleanupMihomoResidualRoutes 清理 Mihomo 残留路由并返回详细报告
func (rm *RouteManager) CleanupMihomoResidualRoutes() (*CleanupReport, error) {
	residualRoutes, err := rm.CheckMihomoResidualRoutes()
	if err != nil {
		return nil, err
	}

	report := &CleanupReport{
		TotalFound:   len(residualRoutes),
		TotalRemoved: 0,
		Failed:       []RouteEntry{},
		Skipped:      []RouteEntry{},
	}

	for _, residual := range residualRoutes {
		route := residual.Route

		// 检查是否需要跳过（例如度量值不是 0 的默认路由）
		if shouldSkipRoute(route) {
			report.Skipped = append(report.Skipped, route)
			continue
		}

		// 尝试删除路由
		if err := rm.DeleteRoute(route); err != nil {
			report.Failed = append(report.Failed, route)
			report.LastError = err
		} else {
			report.TotalRemoved++
			report.Removed = append(report.Removed, route)
		}
	}

	return report, nil
}

// shouldSkipRoute 判断是否应该跳过删除某些路由
func shouldSkipRoute(route RouteEntry) bool {
	// 不要删除直连路由（On-link）
	if strings.Contains(route.Gateway, "On-link") {
		return true
	}

	// 不要删除不是 Mihomo 路由的配置
	if !isMihomoGateway(route.Gateway) && !strings.HasPrefix(route.Interface, "198.18.") {
		return true
	}

	// 对于非默认路由，如果度量值较高，可能是系统自动添加的，跳过
	if route.Destination != "0.0.0.0" && route.Metric > 100 {
		return true
	}

	return false
}

// FixRouteIssues 自动修复路由问题（一键修复）
// 用于用户遇到问题时快速修复
func (rm *RouteManager) FixRouteIssues() (*FixReport, error) {
	report := &FixReport{}

	// 1. 清理残留路由
	cleanupReport, err := rm.CleanupMihomoResidualRoutes()
	if err != nil {
		report.Errors = append(report.Errors, err)
	} else {
		report.CleanupReport = cleanupReport
	}

	// 2. 检查是否还有问题
	diagnosis, err := rm.DiagnoseNetworkRouting()
	if err != nil {
		report.Errors = append(report.Errors, err)
	} else {
		report.Diagnosis = diagnosis
	}

	// 3. 综合判断修复结果
	report.Success = len(report.Errors) == 0 && diagnosis.Health == "Healthy"
	if !report.Success {
		switch diagnosis.Health {
		case "Critical":
			report.Message = "Critical issues remain, manual intervention required"
		case "Warning":
			report.Message = "Minor issues remain, check details"
		}
	} else {
		report.Message = "All route issues fixed successfully"
	}

	return report, nil
}

// CheckResidual 检查是否有残留路由
func (rm *RouteManager) CheckResidual() (*Problem, error) {
	routes, err := rm.ListRoutes()
	if err != nil {
		return nil, err
	}

	var mihomoRoutes []RouteEntry
	for _, route := range routes {
		if isMihomoRoute(route) {
			mihomoRoutes = append(mihomoRoutes, route)
		}
	}

	if len(mihomoRoutes) > 0 {
		routeStrs := make([]string, len(mihomoRoutes))
		for i, route := range mihomoRoutes {
			routeStrs[i] = fmt.Sprintf("%s via %s", route.Destination, route.Gateway)
		}

		return &Problem{
			Type:        ProblemConfigResidual,
			Severity:    SeverityHigh,
			Description: "Routes added by Mihomo still exist",
			Details: map[string]interface{}{
				"routes": routeStrs,
			},
			Solutions: []Solution{
				{
					Description: "Remove routes",
					Command:     "mihomo-cli system cleanup --route",
					Auto:        true,
				},
				{
					Description: "Restart Mihomo to cleanup",
					Command:     "mihomo-cli restart",
					Auto:        true,
				},
				{
					Description: "Restart system to cleanup",
					Command:     "restart computer",
					Auto:        false,
				},
			},
		}, nil
	}

	return nil, nil
}

// Cleanup 清理路由表
func (rm *RouteManager) Cleanup() error {
	return rm.CleanupMihomoRoutes()
}
