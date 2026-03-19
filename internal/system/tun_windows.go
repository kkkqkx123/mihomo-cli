//go:build windows

package system

import (
	"bytes"
	"fmt"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
)

// listTUNDevices 枚举 Windows TUN/TAP 设备
// Windows 上 TUN 设备通常由 Wintun 或 TAP-Windows 驱动创建
// 使用 netsh 和 PowerShell 查询网络适配器
func (tm *TUNManager) listTUNDevices() ([]TUNState, error) {
	var devices []TUNState

	// 方法1: 使用 PowerShell 查询 TUN/TAP 适配器
	psDevices, err := tm.listTUNDevicesPowerShell()
	if err == nil && len(psDevices) > 0 {
		devices = append(devices, psDevices...)
	}

	// 方法2: 使用 netsh 作为备选
	if len(devices) == 0 {
		netshDevices, err := tm.listTUNDevicesNetsh()
		if err == nil {
			devices = append(devices, netshDevices...)
		}
	}

	return devices, nil
}

// listTUNDevicesPowerShell 使用 PowerShell 查询 TUN 设备
func (tm *TUNManager) listTUNDevicesPowerShell() ([]TUNState, error) {
	// PowerShell 查询 Wintun/TAP 适配器
	// Wintun 适配器的 InterfaceDescription 通常包含 "Wintun"
	// TAP 适配器的 InterfaceDescription 通常包含 "TAP" 或 "Tun"
	psScript := `Get-NetAdapter | Where-Object { $_.InterfaceDescription -match 'Wintun|TAP|Tun|WireGuard' -or $_.Name -match 'utun|tun|clash|mihomo|wintun' } | Select-Object Name, InterfaceDescription, Status, MacAddress | ConvertTo-Json`

	cmd := exec.Command("powershell", "-NoProfile", "-NonInteractive", "-Command", psScript)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("powershell query failed: %w, stderr: %s", err, stderr.String())
	}

	return parsePowerShellTUNOutput(output)
}

// parsePowerShellTUNOutput 解析 PowerShell 输出
func parsePowerShellTUNOutput(output []byte) ([]TUNState, error) {
	var devices []TUNState
	outputStr := strings.TrimSpace(string(output))
	if outputStr == "" || outputStr == "null" {
		return devices, nil
	}

	// PowerShell JSON 输出可能是数组或单个对象
	// 简单解析，不使用 encoding/json 以避免额外依赖
	// 格式示例:
	// [
	//   {
	//     "Name": "Wintun",
	//     "InterfaceDescription": "Wintun Userspace Tunnel",
	//     "Status": "Up",
	//     "MacAddress": "00-00-00-00-00-00"
	//   }
	// ]

	// 使用正则提取设备信息
	namePattern := regexp.MustCompile(`"Name"\s*:\s*"([^"]+)"`)
	descPattern := regexp.MustCompile(`"InterfaceDescription"\s*:\s*"([^"]+)"`)
	statusPattern := regexp.MustCompile(`"Status"\s*:\s*"([^"]+)"`)

	// 分割多个设备
	blocks := regexp.MustCompile(`\}\s*,\s*\{`).Split(outputStr, -1)

	for _, block := range blocks {
		nameMatch := namePattern.FindStringSubmatch(block)
		descMatch := descPattern.FindStringSubmatch(block)
		statusMatch := statusPattern.FindStringSubmatch(block)

		if len(nameMatch) > 1 {
			device := TUNState{
				Name:    nameMatch[1],
				Enabled: len(statusMatch) > 1 && statusMatch[1] == "Up",
			}

			// 尝试获取 IP 地址
			ip, err := getInterfaceIP(nameMatch[1])
			if err == nil {
				device.IPAddress = ip
			}

			// 尝试获取 MTU
			mtu, err := getInterfaceMTU(nameMatch[1])
			if err == nil {
				device.MTU = mtu
			}

			// 记录接口描述（用于调试）
			if len(descMatch) > 1 {
				_ = descMatch[1] // InterfaceDescription
			}

			devices = append(devices, device)
		}
	}

	return devices, nil
}

// listTUNDevicesNetsh 使用 netsh 查询 TUN 设备（备选方法）
func (tm *TUNManager) listTUNDevicesNetsh() ([]TUNState, error) {
	cmd := exec.Command("netsh", "interface", "show", "interface")
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("netsh query failed: %w, stderr: %s", err, stderr.String())
	}

	return parseNetshTUNOutput(output)
}

// parseNetshTUNOutput 解析 netsh interface show interface 输出
// 输出格式示例:
// Admin State    State          Type             Interface Name
// Enabled        Connected      Dedicated        Wintun
// Enabled        Connected      Dedicated        Ethernet
func parseNetshTUNOutput(output []byte) ([]TUNState, error) {
	var devices []TUNState
	lines := strings.Split(string(output), "\n")

	// 跳过标题行，找到数据开始位置
	inData := false
	for _, line := range lines {
		line = strings.TrimSpace(line)

		// 检测标题行
		if strings.Contains(line, "Admin State") && strings.Contains(line, "Interface Name") {
			inData = true
			continue
		}

		// 跳过分隔线
		if strings.HasPrefix(line, "---") {
			continue
		}

		if !inData || line == "" {
			continue
		}

		// 解析数据行
		// 格式: Admin State    State    Type    Interface Name
		fields := strings.Fields(line)
		if len(fields) >= 4 {
			name := strings.Join(fields[3:], " ") // Interface Name 可能有空格

			// 检查是否是 TUN 相关接口
			if isMihomoTUN(name) || strings.Contains(strings.ToLower(name), "wintun") ||
				strings.Contains(strings.ToLower(name), "tap") || strings.Contains(strings.ToLower(name), "tun") {
				device := TUNState{
					Name:    name,
					Enabled: fields[1] == "Connected",
				}

				// 尝试获取 IP 地址
				ip, err := getInterfaceIP(name)
				if err == nil {
					device.IPAddress = ip
				}

				// 尝试获取 MTU
				mtu, err := getInterfaceMTU(name)
				if err == nil {
					device.MTU = mtu
				}

				devices = append(devices, device)
			}
		}
	}

	return devices, nil
}

// getInterfaceIP 获取接口的 IP 地址
func getInterfaceIP(ifaceName string) (string, error) {
	// 使用 netsh interface ip show config name="接口名"
	cmd := exec.Command("netsh", "interface", "ip", "show", "config", fmt.Sprintf("name=\"%s\"", ifaceName))
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}

	// 解析输出，查找 IP 地址
	// 格式: "IP Address:                           192.168.1.100"
	ipPattern := regexp.MustCompile(`IP Address\s*:\s*(\d+\.\d+\.\d+\.\d+)`)
	matches := ipPattern.FindStringSubmatch(string(output))
	if len(matches) > 1 {
		return matches[1], nil
	}

	return "", fmt.Errorf("IP address not found")
}

// getInterfaceMTU 获取接口的 MTU
func getInterfaceMTU(ifaceName string) (int, error) {
	// 使用 netsh interface ipv4 show subinterface
	cmd := exec.Command("netsh", "interface", "ipv4", "show", "subinterface")
	output, err := cmd.Output()
	if err != nil {
		return 0, err
	}

	// 解析输出
	// 格式: MTU  MediaSenseState   Bytes In  Bytes Out  Interface
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if strings.Contains(line, ifaceName) {
			fields := strings.Fields(line)
			if len(fields) >= 5 {
				mtu, err := strconv.Atoi(fields[0])
				if err == nil {
					return mtu, nil
				}
			}
		}
	}

	return 0, fmt.Errorf("MTU not found")
}

// removeTUN 删除 Windows TUN 设备
// Windows 上 TUN 设备通常由驱动管理，需要禁用接口或卸载驱动
func (tm *TUNManager) removeTUN(name string) error {
	// 方法1: 禁用接口
	cmd := exec.Command("netsh", "interface", "set", "interface", name, "admin=disable")
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to disable TUN interface %s: %w, stderr: %s", name, err, stderr.String())
	}

	// 注意: 完全删除 TUN 设备需要卸载驱动程序
	// 这通常需要管理员权限和更复杂的操作
	// 这里只禁用接口，实际删除由驱动程序管理

	return nil
}
