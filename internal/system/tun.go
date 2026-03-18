package system

import (
	"fmt"
	"runtime"
)

// TUNManager TUN 网卡管理器
type TUNManager struct {
	audit *AuditLogger
}

// NewTUNManager 创建 TUN 管理器
func NewTUNManager(audit *AuditLogger) *TUNManager {
	return &TUNManager{
		audit: audit,
	}
}

// ListTUNDevices 列出所有 TUN 设备
func (tm *TUNManager) ListTUNDevices() ([]TUNState, error) {
	// 根据平台调用不同的实现
	switch runtime.GOOS {
	case "windows":
		return tm.listTUNDevicesWindows()
	case "linux":
		return tm.listTUNDevicesLinux()
	case "darwin":
		return tm.listTUNDevicesDarwin()
	default:
		return []TUNState{}, nil
	}
}

// CheckMihomoTUN 检查 Mihomo 创建的 TUN 设备
func (tm *TUNManager) CheckMihomoTUN() ([]TUNState, error) {
	devices, err := tm.ListTUNDevices()
	if err != nil {
		return nil, err
	}

	// 过滤出 Mihomo 相关的 TUN 设备
	var mihomoDevices []TUNState
	for _, dev := range devices {
		// Mihomo 通常使用 "utun" 或 "tun" 作为前缀
		if isMihomoTUN(dev.Name) {
			mihomoDevices = append(mihomoDevices, dev)
		}
	}

	return mihomoDevices, nil
}

// RemoveTUN 删除 TUN 设备
func (tm *TUNManager) RemoveTUN(name string) error {
	var err error

	switch runtime.GOOS {
	case "windows":
		err = tm.removeTUNWindows(name)
	case "linux":
		err = tm.removeTUNLinux(name)
	case "darwin":
		err = tm.removeTUNDarwin(name)
	default:
		err = fmt.Errorf("unsupported platform: %s", runtime.GOOS)
	}

	if tm.audit != nil {
		result := "success"
		if err != nil {
			result = "failed"
		}
		_ = tm.audit.Record("remove", "tun", name, result, err)
	}

	return err
}

// GetState 获取 TUN 状态
func (tm *TUNManager) GetState() (*TUNState, error) {
	devices, err := tm.CheckMihomoTUN()
	if err != nil {
		return nil, err
	}

	if len(devices) == 0 {
		return &TUNState{Enabled: false}, nil
	}

	// 返回第一个设备的状态
	dev := devices[0]
	return &TUNState{
		Name:      dev.Name,
		Enabled:   true,
		IPAddress: dev.IPAddress,
		MTU:       dev.MTU,
	}, nil
}

// CheckResidual 检查是否有残留 TUN 设备
func (tm *TUNManager) CheckResidual() (*Problem, error) {
	devices, err := tm.CheckMihomoTUN()
	if err != nil {
		return nil, err
	}

	if len(devices) > 0 {
		deviceNames := make([]string, len(devices))
		for i, dev := range devices {
			deviceNames[i] = dev.Name
		}

		return &Problem{
			Type:        ProblemConfigResidual,
			Severity:    SeverityHigh,
			Description: "TUN devices created by Mihomo still exist",
			Details: map[string]interface{}{
				"devices": deviceNames,
			},
			Solutions: []Solution{
				{
					Description: "Remove TUN devices",
					Command:     "mihomo-cli system cleanup --tun",
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

// Cleanup 清理 TUN 设备
func (tm *TUNManager) Cleanup() error {
	devices, err := tm.CheckMihomoTUN()
	if err != nil {
		return err
	}

	var lastErr error
	for _, dev := range devices {
		if err := tm.RemoveTUN(dev.Name); err != nil {
			lastErr = err
		}
	}

	return lastErr
}

// isMihomoTUN 检查是否是 Mihomo 创建的 TUN 设备
func isMihomoTUN(name string) bool {
	// Mihomo 通常使用以下前缀
	prefixes := []string{"utun", "tun", "clash", "mihomo"}
	for _, prefix := range prefixes {
		if len(name) >= len(prefix) && name[:len(prefix)] == prefix {
			return true
		}
	}
	return false
}

// 平台特定实现（stub）
func (tm *TUNManager) listTUNDevicesWindows() ([]TUNState, error) {
	// TODO: 实现 Windows TUN 设备枚举
	// 可以使用 netsh 或 WMI 查询
	return []TUNState{}, nil
}

func (tm *TUNManager) listTUNDevicesLinux() ([]TUNState, error) {
	// TODO: 实现 Linux TUN 设备枚举
	// 可以读取 /sys/class/net/ 目录
	return []TUNState{}, nil
}

func (tm *TUNManager) listTUNDevicesDarwin() ([]TUNState, error) {
	// TODO: 实现 macOS TUN 设备枚举
	// 可以使用 ifconfig 命令
	return []TUNState{}, nil
}

func (tm *TUNManager) removeTUNWindows(name string) error {
	// TODO: 实现 Windows TUN 设备删除
	// 可以使用 netsh 或设备管理器 API
	return fmt.Errorf("TUN device removal not implemented on Windows")
}

func (tm *TUNManager) removeTUNLinux(name string) error {
	// TODO: 实现 Linux TUN 设备删除
	// 可以使用 ip link delete 命令
	return fmt.Errorf("TUN device removal not implemented on Linux")
}

func (tm *TUNManager) removeTUNDarwin(name string) error {
	// TODO: 实现 macOS TUN 设备删除
	// 通常需要重启系统
	return fmt.Errorf("TUN device removal not implemented on macOS")
}
