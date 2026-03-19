//go:build linux

package system

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
)

// listTUNDevices 枚举 Linux TUN 设备
// Linux 上 TUN 设备可以通过 /sys/class/net/ 目录检测
// TUN 设备的 type 为 512 (IFF_TUN)，TAP 设备为 65536 (IFF_TAP)
func (tm *TUNManager) listTUNDevices() ([]TUNState, error) {
	var devices []TUNState

	// 遍历 /sys/class/net/ 目录
	netDir := "/sys/class/net"
	entries, err := os.ReadDir(netDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read %s: %w", netDir, err)
	}

	for _, entry := range entries {
		name := entry.Name()

		// 跳过符号链接和非目录
		if !entry.IsDir() && entry.Type() != os.ModeSymlink {
			continue
		}

		// 检查是否是 TUN/TAP 设备
		isTUN, err := isTUNTAPDevice(name)
		if err != nil || !isTUN {
			continue
		}

		// 获取设备详细信息
		device, err := tm.getTUNDeviceInfo(name)
		if err != nil {
			// 记录错误但继续处理其他设备
			device = TUNState{Name: name, Enabled: true}
		}

		devices = append(devices, device)
	}

	return devices, nil
}

// isTUNTAPDevice 检查是否是 TUN/TAP 设备
// 注意：此函数同时检测 TUN (type=512) 和 TAP (type=65536) 设备
func isTUNTAPDevice(name string) (bool, error) {
	// 方法1: 检查 tun_flags 文件存在
	// TUN/TAP 设备会有 tun_flags 文件
	tunFlagsPath := fmt.Sprintf("/sys/class/net/%s/tun_flags", name)
	if _, err := os.Stat(tunFlagsPath); err == nil {
		return true, nil
	}

	// 方法2: 检查设备类型
	// TUN 设备类型为 512 (IFF_TUN = 0x0002 << 8)
	// TAP 设备类型为 65536 (IFF_TAP = 0x0004 << 8)
	typePath := fmt.Sprintf("/sys/class/net/%s/type", name)
	data, err := os.ReadFile(typePath)
	if err != nil {
		return false, err
	}

	typeStr := strings.TrimSpace(string(data))
	devType, err := strconv.Atoi(typeStr)
	if err != nil {
		return false, err
	}

	// TUN = 512, TAP = 65536
	// 注意: 某些虚拟设备也可能使用这些类型
	if devType == 512 || devType == 65536 {
		return true, nil
	}

	// 方法3: 检查设备是否在 /dev/net/tun 控制下
	// 通过检查 uevent 文件
	ueventPath := fmt.Sprintf("/sys/class/net/%s/uevent", name)
	if data, err := os.ReadFile(ueventPath); err == nil {
		uevent := string(data)
		// TUN 设备通常有 DEVTYPE=tun 或类似的标记
		if strings.Contains(uevent, "DEVTYPE=tun") ||
			strings.Contains(uevent, "DEVTYPE=tap") {
			return true, nil
		}
	}

	// 方法4: 检查设备名称模式
	// Mihomo/Clash 通常使用 tun0, tun1, clash0 等名称
	if isMihomoTUN(name) {
		return true, nil
	}

	return false, nil
}

// getTUNDeviceInfo 获取 TUN 设备详细信息
func (tm *TUNManager) getTUNDeviceInfo(name string) (TUNState, error) {
	device := TUNState{
		Name:    name,
		Enabled: true,
	}

	// 获取操作状态
	operstatePath := fmt.Sprintf("/sys/class/net/%s/operstate", name)
	if data, err := os.ReadFile(operstatePath); err == nil {
		state := strings.TrimSpace(string(data))
		device.Enabled = state == "up" || state == "unknown"
	}

	// 获取 MTU
	mtuPath := fmt.Sprintf("/sys/class/net/%s/mtu", name)
	if data, err := os.ReadFile(mtuPath); err == nil {
		mtu, err := strconv.Atoi(strings.TrimSpace(string(data)))
		if err == nil {
			device.MTU = mtu
		}
	}

	// 获取 IP 地址（使用 ip 命令）
	ip, err := getInterfaceIP(name)
	if err == nil {
		device.IPAddress = ip
	}

	return device, nil
}

// getInterfaceIP 获取接口的 IP 地址
func getInterfaceIP(name string) (string, error) {
	// 使用 ip addr show <name>
	cmd := exec.Command("ip", "-o", "-4", "addr", "show", name)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("ip addr show failed: %w, stderr: %s", err, stderr.String())
	}

	// 输出格式: 2: tun0    inet 10.0.0.1/24 scope global tun0\       valid_lft forever preferred_lft forever
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// 解析 inet 行
		fields := strings.Fields(line)
		for i, field := range fields {
			if field == "inet" && i+1 < len(fields) {
				// 返回 IP 地址（包含 CIDR）
				return fields[i+1], nil
			}
		}
	}

	return "", fmt.Errorf("IP address not found for interface %s", name)
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

// removeTUN 删除 Linux TUN 设备
// 使用 ip link delete 命令
func (tm *TUNManager) removeTUN(name string) error {
	// 方法1: 使用 ip link delete
	cmd := exec.Command("ip", "link", "delete", name)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		// 方法2: 尝试使用 tunctl（如果可用）
		cmd2 := exec.Command("tunctl", "-d", name)
		if err2 := cmd2.Run(); err2 == nil {
			return nil
		}

		return fmt.Errorf("failed to delete TUN interface %s: %w, stderr: %s", name, err, stderr.String())
	}

	return nil
}


