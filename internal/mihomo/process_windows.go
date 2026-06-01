//go:build windows

package mihomo

import (
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"

	pkgerrors "github.com/kkkqkx123/mihomo-cli/pkg/errors"
)

// Windows API 常量
const (
	MAX_PATH = 260
)

// PROCESS_MEMORY_COUNTERS 进程内存计数器
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

// Windows API 函数（psapi.dll 用于获取内存信息）
var (
	modpsapi                   = syscall.NewLazyDLL("psapi.dll")
	procGetProcessMemoryInfo   = modpsapi.NewProc("GetProcessMemoryInfo")
)

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

// GetProcessExecutable 获取进程的可执行文件路径
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

	// 获取可执行文件路径
	var path [MAX_PATH]uint16
	var size uint32 = MAX_PATH

	err = windows.QueryFullProcessImageName(
		handle,
		0,
		&path[0],
		&size,
	)
	if err != nil {
		return "", pkgerrors.ErrService("failed to query process image name", err)
	}

	return syscall.UTF16ToString(path[:]), nil
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
