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
	var lastError error

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
			// 检查错误信息，判断是否是"设备不存在"错误
			errStr := stderr.String()
			// macOS 的 ifconfig 在设备不存在时会返回特定的错误信息
			if strings.Contains(errStr, "does not exist") ||
				strings.Contains(errStr, "interface") ||
				cmd.ProcessState.ExitCode() == 1 {
				// 设备不存在，继续检查下一个
				continue
			}
			// 其他错误，记录但继续
			lastError = fmt.Errorf("ifconfig %s failed: %w", name, err)
			continue
		}

		// 解析 ifconfig 输出
		device, err := parseIfconfigOutput(name, output)
		if err != nil {
			// 解析失败，但仍添加设备列表
			device = TUNState{Name: name, Enabled: true}
		}

		devices = append(devices, device)
	}

	// 如果有错误但没有找到任何设备，返回错误
	if len(devices) == 0 && lastError != nil {
		return nil, lastError
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
			// 使用更通用的方法：检查 flags 值中是否包含 UP
			// 格式: flags=XXXXX<FLAG1,FLAG2,...>
			if flagsMatch := regexp.MustCompile(`flags=\d+<([^>]+)>`).FindStringSubmatch(line); len(flagsMatch) > 1 {
				flagsStr := flagsMatch[1]
				flags := strings.Split(flagsStr, ",")
				for _, flag := range flags {
					if strings.TrimSpace(flag) == "UP" {
						device.Enabled = true
						break
					}
				}
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
		// 灵活匹配 IP 地址字段
		for _, field := range fields {
			ipPattern := regexp.MustCompile(`^(\d{1,3}\.\d{1,3}\.\d{1,3}\.\d{1,3})$`)
			if matches := ipPattern.FindStringSubmatch(field); len(matches) > 1 {
				device.IPAddress = matches[1]
				break
			}
		}
	}

	// 转换为切片
	for _, device := range deviceMap {
		devices = append(devices, *device)
	}

	return devices, nil
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

// removeTUN 删除 macOS TUN 设备
// 注意: macOS 的 utun 设备由内核管理，通常无法手动删除
// 需要关闭创建它的进程或重启系统
// 此函数实际执行禁用操作，返回成功表示接口已禁用
func (tm *TUNManager) removeTUN(name string) error {
	// 尝试使用 ifconfig 禁用接口
	cmd := exec.Command("ifconfig", name, "down")
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to bring down TUN interface %s: %w, stderr: %s", name, err, stderr.String())
	}

	// macOS utun 设备已禁用，但无法完全删除
	// 返回成功，表示接口已禁用
	return nil
}
