//go:build linux

package mihomo

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"

	pkgerrors "github.com/kkkqkx123/mihomo-cli/pkg/errors"
)

// linuxProcessChecker Linux 平台进程检查器
type linuxProcessChecker struct{}

// newProcessChecker 创建进程检查器（Linux 平台）
func newProcessChecker() ProcessChecker {
	return &linuxProcessChecker{}
}

// IsProcessRunning 检查进程是否正在运行
func (l *linuxProcessChecker) IsProcessRunning(pid int) bool {
	// 在 Linux 系统上，通过 /proc/<pid> 目录检查进程是否存在
	procPath := filepath.Join("/proc", strconv.Itoa(pid))

	// 尝试读取 /proc/<pid>/stat 文件
	statPath := filepath.Join(procPath, "stat")
	data, err := os.ReadFile(statPath)
	if err != nil {
		return false
	}

	// 如果文件存在且不为空，进程正在运行
	return len(data) > 0
}

// GetProcessExecutable 获取进程的可执行文件路径
func (l *linuxProcessChecker) GetProcessExecutable(pid int) (string, error) {
	// 在 Linux 系统上，通过 /proc/<pid>/exe 符号链接获取可执行文件路径
	procPath := filepath.Join("/proc", strconv.Itoa(pid), "exe")

	// 读取符号链接
	execPath, err := os.Readlink(procPath)
	if err != nil {
		// 如果符号链接读取失败，尝试从 cmdline 获取
		cmdlinePath := filepath.Join("/proc", strconv.Itoa(pid), "cmdline")
		data, err := os.ReadFile(cmdlinePath)
		if err != nil {
			return "", pkgerrors.ErrService("failed to get process executable", err)
		}

		// cmdline 以 null 分隔参数，取第一个参数
		parts := strings.Split(string(data), "\x00")
		if len(parts) > 0 && parts[0] != "" {
			return parts[0], nil
		}

		return "", pkgerrors.ErrService("failed to get process executable", nil)
	}

	return execPath, nil
}

// getProcessResourceUsage 获取进程资源使用情况 (Linux 实现)
func getProcessResourceUsage(pid int) (cpu, memory float64, err error) {
	// 读取 /proc/[pid]/stat 文件
	statPath := filepath.Join("/proc", strconv.Itoa(pid), "stat")
	data, err := os.ReadFile(statPath)
	if err != nil {
		return 0, 0, pkgerrors.ErrService("failed to read process stat", err)
	}

	// 解析 stat 文件内容
	// 格式: pid (comm) state ppid pgrp session tty_nr tpgid flags ...
	// 我们需要的是第 14-17 字段 (utime, stime, cutime, cstime)
	fields := strings.Fields(string(data))
	if len(fields) < 24 {
		return 0, 0, pkgerrors.ErrService("invalid stat format", nil)
	}

	// 注意：comm 字段可能包含空格和括号，需要特殊处理
	// 找到第一个 '(' 和最后一个 ')' 的位置
	start := strings.Index(string(data), "(")
	end := strings.LastIndex(string(data), ")")
	if start == -1 || end == -1 {
		return 0, 0, pkgerrors.ErrService("invalid stat format", nil)
	}

	// 重新分割字段，跳过 comm 字段
	afterComm := strings.Fields(string(data)[end+2:])
	if len(afterComm) < 20 {
		return 0, 0, pkgerrors.ErrService("invalid stat format", nil)
	}

	// utime 和 stime (单位: jiffies, 通常 100 Hz)
	utime, _ := strconv.ParseFloat(afterComm[10], 64)
	stime, _ := strconv.ParseFloat(afterComm[11], 64)
	cpu = (utime + stime) / 100.0 // 转换为秒

	// 读取 /proc/[pid]/status 获取内存信息
	statusPath := filepath.Join("/proc", strconv.Itoa(pid), "status")
	statusData, err := os.ReadFile(statusPath)
	if err == nil {
		lines := strings.Split(string(statusData), "\n")
		for _, line := range lines {
			if strings.HasPrefix(line, "VmRSS:") {
				// VmRSS: 实际物理内存使用 (kB)
				fields := strings.Fields(line)
				if len(fields) >= 2 {
					memKB, _ := strconv.ParseFloat(fields[1], 64)
					memory = memKB / 1024.0 // 转换为 MB
				}
				break
			}
		}
	}

	return cpu, memory, nil
}
