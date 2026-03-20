//go:build darwin

package mihomo

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"syscall"

	pkgerrors "github.com/kkkqkx123/mihomo-cli/pkg/errors"
)

// darwinProcessChecker macOS 平台进程检查器
type darwinProcessChecker struct{}

// newProcessChecker 创建进程检查器（macOS 平台）
func newProcessChecker() ProcessChecker {
	return &darwinProcessChecker{}
}

// IsProcessRunning 检查进程是否正在运行
func (d *darwinProcessChecker) IsProcessRunning(pid int) bool {
	// 在 macOS 上，使用 os.FindProcess 和信号 0 检查进程是否存在
	// 信号 0 不会实际发送信号，只是检查进程是否存在
	proc, err := os.FindProcess(pid)
	if err != nil {
		return false
	}

	// 发送信号 0 检查进程是否存在
	err = proc.Signal(syscall.Signal(0))
	return err == nil
}

// GetProcessExecutable 获取进程的可执行文件路径
func (d *darwinProcessChecker) GetProcessExecutable(pid int) (string, error) {
	// 在 macOS 上，使用 ps 命令获取可执行文件路径
	// ps -p <pid> -o comm= 只返回命令名
	// ps -p <pid> -o args= 返回完整命令行
	cmd := exec.Command("ps", "-p", fmt.Sprintf("%d", pid), "-o", "comm=")
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return "", pkgerrors.ErrService("failed to get process executable: "+stderr.String(), err)
	}

	execPath := strings.TrimSpace(stdout.String())
	if execPath == "" {
		return "", pkgerrors.ErrService("empty process executable path", nil)
	}

	return execPath, nil
}

// getProcessResourceUsage 获取进程资源使用情况 (macOS 实现)
func getProcessResourceUsage(pid int) (cpu, memory float64, err error) {
	// 使用 ps 命令获取 CPU 和内存使用情况
	cmd := exec.Command("ps", "-p", fmt.Sprintf("%d", pid), "-o", "%cpu,%mem,rss=")
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return 0, 0, pkgerrors.ErrService("failed to get process resource usage: "+stderr.String(), err)
	}

	// 解析输出
	// 格式: %CPU %MEM RSS
	// 示例:  0.0  0.1 1234
	lines := strings.Split(strings.TrimSpace(stdout.String()), "\n")
	if len(lines) < 1 {
		return 0, 0, pkgerrors.ErrService("invalid ps output", nil)
	}

	fields := strings.Fields(lines[len(lines)-1])
	if len(fields) < 3 {
		return 0, 0, pkgerrors.ErrService("invalid ps output format", nil)
	}

	// 解析 CPU 使用率
	cpu, _ = strconv.ParseFloat(fields[0], 64)

	// 解析内存使用 (RSS, 单位: KB)
	rssKB, _ := strconv.ParseFloat(fields[2], 64)
	memory = rssKB / 1024.0 // 转换为 MB

	return cpu, memory, nil
}
