//go:build windows

package mihomo

import (
	"bytes"
	"fmt"
	"os/exec"
	"strings"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"

	pkgerrors "github.com/kkkqkx123/mihomo-cli/pkg/errors"
)

// Windows API 函数（psapi.dll 用于获取内存信息）
var (
	modpsapi                 = syscall.NewLazyDLL("psapi.dll")
	procGetProcessMemoryInfo = modpsapi.NewProc("GetProcessMemoryInfo")
)

type PROCESS_MEMORY_COUNTERS struct {
	CB                         uint32
	PageFaultCount             uint32
	PeakWorkingSetSize         uintptr
	WorkingSetSize             uintptr
	QuotaPeakPagedPoolUsage    uintptr
	QuotaPagedPoolUsage        uintptr
	QuotaPeakNonPagedPoolUsage uintptr
	QuotaNonPagedPoolUsage     uintptr
	PagefileUsage              uintptr
	PeakPagefileUsage          uintptr
}

// FILETIME Windows 文件时间结构（与 windows.Filetime 兼容）
type FILETIME = windows.Filetime

// windowsProcessChecker Windows 平台进程检查器
type windowsProcessChecker struct{}

// newProcessChecker 创建进程检查器（Windows 平台）
func newProcessChecker() ProcessChecker {
	return &windowsProcessChecker{}
}

// IsProcessRunning 检查进程是否正在运行
func (w *windowsProcessChecker) IsProcessRunning(pid int) bool {
	// 使用 PROCESS_QUERY_INFORMATION 权限打开进程
	// 如果进程不存在，OpenProcess 会返回错误
	handle, err := windows.OpenProcess(
		windows.PROCESS_QUERY_INFORMATION,
		false,
		uint32(pid),
	)
	if err != nil {
		// 如果是权限不足（ERROR_ACCESS_DENIED），保守认为进程仍在运行
		if errno, ok := err.(windows.Errno); ok && errno == windows.ERROR_ACCESS_DENIED {
			return true
		}
		return false
	}
	// 关闭句柄
	_ = windows.CloseHandle(handle)
	return true
}

// GetProcessExecutable 获取进程的可执行文件绝对路径。
// 动态缓冲：先以 260 字节调用，若返回 ERROR_INSUFFICIENT_BUFFER 按返回值扩容重试
// （上限 32K）；成功路径去掉可能的 \\?\ 前缀，保证与配置/解析器产出的路径可字符串级比对。
func (w *windowsProcessChecker) GetProcessExecutable(pid int) (string, error) {
	// 打开进程
	handle, err := windows.OpenProcess(
		windows.PROCESS_QUERY_INFORMATION|windows.PROCESS_VM_READ,
		false,
		uint32(pid),
	)
	if err != nil {
		return "", pkgerrors.ErrService("failed to open process", err)
	}
	defer func() { _ = windows.CloseHandle(handle) }()

	buf := make([]uint16, 260)
	const maxBuf = 32 * 1024 // 32K 上限，超限报错而非静默跳过

	for {
		size := uint32(len(buf))
		err = windows.QueryFullProcessImageName(
			handle,
			0,
			&buf[0],
			&size,
		)
		if err == windows.ERROR_INSUFFICIENT_BUFFER {
			if len(buf)*2 > maxBuf {
				return "", pkgerrors.ErrService("process image path exceeds 32K buffer limit", nil)
			}
			buf = make([]uint16, len(buf)*2)
			continue
		}
		if err != nil {
			return "", pkgerrors.ErrService("failed to query process image name", err)
		}

		path := syscall.UTF16ToString(buf[:size])
		// 去掉 \\?\ 前缀（QueryFullProcessImageName 可能返回扩展路径形式）
		path = strings.TrimPrefix(path, `\\?\`)
		return path, nil
	}
}

// GetProcessCommandLine 获取进程完整命令行。
// 务实降级实现：wmic process where processid=<pid> get commandline /value
// （已弃用但稳定、无额外依赖）；调用失败/输出为空时返回可读错误。
func (w *windowsProcessChecker) GetProcessCommandLine(pid int) (string, error) {
	cmd := exec.Command("wmic", "process", "where", fmt.Sprintf("processid=%d", pid), "get", "commandline", "/value")
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return "", pkgerrors.ErrService("failed to get process command line: "+stderr.String(), err)
	}

	// /value 输出格式：CommandLine=xxx（可能多行，取 CommandLine= 的值）
	for _, line := range strings.Split(stdout.String(), "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "CommandLine=") {
			cmdline := strings.TrimPrefix(line, "CommandLine=")
			if cmdline != "" {
				return cmdline, nil
			}
		}
	}

	return "", pkgerrors.ErrService("empty process command line", nil)
}

// getProcessResourceUsage 获取进程资源使用情况 (Windows 实现)
// 注意：此函数返回的是进程累计 CPU 时间（秒），而非瞬时 CPU 使用率
// 如需计算 CPU 使用率，需要在两个时间点采样并计算差值
func getProcessResourceUsage(pid int) (cpu, memory float64, err error) {
	// 打开进程
	handle, err := windows.OpenProcess(
		windows.PROCESS_QUERY_INFORMATION|windows.PROCESS_VM_READ,
		false,
		uint32(pid),
	)
	if err != nil {
		return 0, 0, pkgerrors.ErrService("failed to open process", err)
	}
	defer func() { _ = windows.CloseHandle(handle) }()

	// 获取进程时间信息
	var creationTime, exitTime, kernelTime, userTime FILETIME
	err = windows.GetProcessTimes(
		handle,
		&creationTime,
		&exitTime,
		&kernelTime,
		&userTime,
	)
	if err != nil {
		return 0, 0, pkgerrors.ErrService("failed to get process times", err)
	}

	// 计算累计 CPU 时间（秒）
	// FILETIME 是 100 纳秒为单位，转换为秒需要除以 1e7
	kernelTimeValue := float64((uint64(kernelTime.HighDateTime)<<32)|uint64(kernelTime.LowDateTime)) / 1e7
	userTimeValue := float64((uint64(userTime.HighDateTime)<<32)|uint64(userTime.LowDateTime)) / 1e7
	cpu = kernelTimeValue + userTimeValue

	// 获取内存使用情况（工作集大小，单位：MB）
	var memCounters PROCESS_MEMORY_COUNTERS
	memCounters.CB = uint32(unsafe.Sizeof(memCounters))
	ret, _, err := procGetProcessMemoryInfo.Call(
		uintptr(handle),
		uintptr(unsafe.Pointer(&memCounters)),
		uintptr(memCounters.CB),
	)
	if ret != 0 {
		// WorkingSetSize 单位是字节，转换为 MB
		memory = float64(memCounters.WorkingSetSize) / 1024.0 / 1024.0
	}

	return cpu, memory, nil
}
