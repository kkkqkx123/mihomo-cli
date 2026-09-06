//go:build darwin

package mihomo

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"

	"golang.org/x/sys/unix"

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
	// 信号 0 不会实际发送信号，只是检查进程是否存在
	proc, err := os.FindProcess(pid)
	if err != nil {
		return false
	}

	err = proc.Signal(syscall.Signal(0))
	if err == nil {
		return true
	}
	// 权限不足（EPERM）时保守认为进程存活，与 Windows/Linux 语义对齐
	if errors.Is(err, syscall.EPERM) {
		return true
	}
	// ESRCH（进程不存在）及其它错误 → false
	return false
}

// GetProcessExecutable 获取进程的可执行文件绝对路径。
// 无 cgo 实现：读 kern.procargs2 sysctl 取 argv[0]（守护进程以绝对路径启动时即 exec 路径）；
// 仅当结果为绝对路径时返回，否则报错（调用方降级处理），绝不使用 ps comm= 冒充绝对路径。
func (d *darwinProcessChecker) GetProcessExecutable(pid int) (string, error) {
	if path, ok := procArgsExecPath(pid); ok && filepath.IsAbs(path) {
		return filepath.Clean(path), nil
	}
	return "", pkgerrors.ErrService("failed to get process executable path", nil)
}

// GetProcessCommandLine 获取进程完整命令行（ps -p <pid> -o args=）
func (d *darwinProcessChecker) GetProcessCommandLine(pid int) (string, error) {
	cmd := exec.Command("ps", "-p", fmt.Sprintf("%d", pid), "-o", "args=")
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return "", pkgerrors.ErrService("failed to get process command line: "+stderr.String(), err)
	}

	cmdline := strings.TrimSpace(stdout.String())
	if cmdline == "" {
		return "", pkgerrors.ErrService("empty process command line", nil)
	}
	return cmdline, nil
}

// procArgsExecPath 从 kern.procargs2 sysctl 读取进程 argv[0]（即 exec 路径）
func procArgsExecPath(pid int) (string, bool) {
	data, err := unix.SysctlRaw("kern.procargs2", pid)
	if err != nil || len(data) < 4 {
		return "", false
	}

	// 前 4 字节为 argc，其后是 NUL 分隔的 argv 字符串
	rest := data[4:]
	end := bytes.IndexByte(rest, 0)
	if end <= 0 {
		return "", false
	}
	return string(rest[:end]), true
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
