//go:build darwin

package system

import (
	"bytes"
	"fmt"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
)

// listTUNDevices 枚举 macOS TUN 设备
// macOS 上 TUN 设备通常命名为 utun0, utun1, ...
// 这些设备由内核管理，通常由 VPN 或隧道软件创建
func (tm *TUNManager) listTUNDevices() ([]TUNState, error) {
	var devices []TUNState

	// 方法1: 使用 ifconfig 遍历 utun 设备
	utunDevices, err := tm.listUTUNDevices()
	if err == nil {
		devices = append(devices, utunDevices...)
	}

	// 方法2: 使用 netstat -in 作为补充
	netstatDevices, err := tm.listTUNDevicesFromNetstat()
	if err == nil {
		// 合并结果，去重
		for _, dev := range netstatDevices {
			found := false
			for _, existing := range devices {
				if existing.Name == dev.Name {
					found = true
					break
				}
			}
			if !found {
				devices = append(devices, dev)
			}
		}
	}

	return devices, nil
}

// listUTUNDevices 遍历 utun 设备
func (tm *TUNManager) listUTUNDevices() ([]TUNState, error) {
	var devices []TUNState

	// macOS utun 设备命名规则: utun0, utun1, ...
	// 通常最多 256 个
	for i := 0; i < 256; i++ {
		name := fmt.Sprintf("utun%d", i)

		// 使用 ifconfig 检查设备是否存在
		cmd := exec.Command("ifconfig", name)
		var stderr bytes.Buffer
		cmd.Stderr = &stderr
		output, err := cmd.Output()
		if err != nil {
			// 设备不存在，继续检查下一个
			continue
		}

		// 解析 ifconfig 输出
		device, err := parseIfconfigOutput(name, output)
		if err != nil {
			device = TUNState{Name: name, Enabled: true}
		}

		devices = append(devices, device)
	}

	return devices, nil
}

// parseIfconfigOutput 解析 ifconfig 输出
// macOS ifconfig 输出格式示例:
// utun0: flags=8051<UP,POINTOPOINT,RUNNING,MULTICAST> mtu 1500
// 	inet 10.0.0.1 --> 10.0.0.2 netmask 0xffffffff
// 	status: active
func parseIfconfigOutput(name string, output []byte) (TUNState, error) {
	device := TUNState{
		Name:    name,
		Enabled: false,
	}

	outputStr := string(output)
	lines := strings.Split(outputStr, "\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)

		// 解析 flags 行
		// flags=8051<UP,POINTOPOINT,RUNNING,MULTICAST> mtu 1500
		if strings.Contains(line, "flags=") {
			// 检查是否包含 UP 标志
			if strings.Contains(line, "<UP,") || strings.Contains(line, ",UP,") || strings.Contains(line, ",UP>") {
				device.Enabled = true
			}

			// 提取 MTU
			mtuPattern := regexp.MustCompile(`mtu\s+(\d+)`)
			if matches := mtuPattern.FindStringSubmatch(line); len(matches) > 1 {
				mtu, err := strconv.Atoi(matches[1])
				if err == nil {
					device.MTU = mtu
				}
			}
		}

		// 解析 inet 行
		// inet 10.0.0.1 --> 10.0.0.2 netmask 0xffffffff
		// 或 inet 10.0.0.1 netmask 0xffffff00
		if strings.HasPrefix(line, "inet ") {
			// 提取 IP 地址
			inetPattern := regexp.MustCompile(`inet\s+(\d+\.\d+\.\d+\.\d+)`)
			if matches := inetPattern.FindStringSubmatch(line); len(matches) > 1 {
				device.IPAddress = matches[1]
			}
		}

		// 检查状态
		if strings.HasPrefix(line, "status: ") {
			status := strings.TrimPrefix(line, "status: ")
			if status == "active" {
				device.Enabled = true
			}
		}
	}

	return device, nil
}

// listTUNDevicesFromNetstat 使用 netstat 获取 TUN 设备列表
func (tm *TUNManager) listTUNDevicesFromNetstat() ([]TUNState, error) {
	// netstat -in 显示所有网络接口
	cmd := exec.Command("netstat", "-in")
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("netstat -in failed: %w, stderr: %s", err, stderr.String())
	}

	return parseNetstatOutput(output)
}

// parseNetstatOutput 解析 netstat -in 输出
// 输出格式示例:
// Name  Mtu   Network       Address         Ipkts Ierrs    Ibytes    Opkts Oerrs    Obytes Coll
// lo0   16384 <Link#1>                      12345     0  12345678    12345     0  12345678   0
// utun0 1500  <Link#6>                      12345     0  12345678    12345     0  12345678   0
// utun0 1500  10.0.0        10.0.0.1        12345     0  12345678    12345     0  12345678   0
func parseNetstatOutput(output []byte) ([]TUNState, error) {
	var devices []TUNState
	deviceMap := make(map[string]*TUNState)

	lines := strings.Split(string(output), "\n")
	inData := false

	for _, line := range lines {
		line = strings.TrimSpace(line)

		// 检测标题行
		if strings.HasPrefix(line, "Name") && strings.Contains(line, "Mtu") {
			inData = true
			continue
		}

		if !inData || line == "" {
			continue
		}

		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}

		name := fields[0]

		// 检查是否是 TUN 设备
		if !isMihomoTUN(name) && !strings.HasPrefix(name, "utun") {
			continue
		}

		// 获取或创建设备记录
		device, exists := deviceMap[name]
		if !exists {
			device = &TUNState{
				Name:    name,
				Enabled: true,
			}
			deviceMap[name] = device

			// 解析 MTU
			if len(fields) > 1 {
				mtu, err := strconv.Atoi(fields[1])
				if err == nil {
					device.MTU = mtu
				}
			}
		}

		// 解析 IP 地址（如果有）
		// 格式: Name Mtu Network Address ...
		if len(fields) >= 4 && strings.Contains(fields[2], ".") {
			// fields[3] 是 Address
			ipPattern := regexp.MustCompile(`^(\d+\.\d+\.\d+\.\d+)$`)
			if matches := ipPattern.FindStringSubmatch(fields[3]); len(matches) > 1 {
				device.IPAddress = matches[1]
			}
		}
	}

	// 转换为切片
	for _, device := range deviceMap {
		devices = append(devices, *device)
	}

	return devices, nil
}

// removeTUN 删除 macOS TUN 设备
// 注意: macOS 的 utun 设备由内核管理，通常无法手动删除
// 需要关闭创建它的进程或重启系统
func (tm *TUNManager) removeTUN(name string) error {
	// 尝试使用 ifconfig 禁用接口
	cmd := exec.Command("ifconfig", name, "down")
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to bring down TUN interface %s: %w, stderr: %s", name, err, stderr.String())
	}

	// macOS utun 设备通常无法手动删除
	// 返回提示信息
	return fmt.Errorf("macOS TUN device %s has been disabled but cannot be fully removed without system restart", name)
}

// checkUTUNAvailability 检查 utun 设备是否可用
func checkUTUNAvailability() bool {
	// 检查是否有可用的 utun 设备
	cmd := exec.Command("ifconfig", "utun0")
	if err := cmd.Run(); err == nil {
		return true
	}

	// 检查系统是否支持 utun
	// macOS 10.6+ 支持 utun
	return true
}

// getUTUNDeviceCount 获取当前 utun 设备数量
func getUTUNDeviceCount() int {
	count := 0
	for i := 0; i < 256; i++ {
		name := fmt.Sprintf("utun%d", i)
		cmd := exec.Command("ifconfig", name)
		if err := cmd.Run(); err == nil {
			count++
		}
	}
	return count
}

// findAvailableUTUN 查找可用的 utun 设备编号
func findAvailableUTUN() int {
	for i := 0; i < 256; i++ {
		name := fmt.Sprintf("utun%d", i)
		cmd := exec.Command("ifconfig", name)
		if err := cmd.Run(); err != nil {
			return i
		}
	}
	return -1
}
