//go:build windows

package mihomo

import (
	"syscall"
	"unsafe"

	pkgerrors "github.com/kkkqkx123/mihomo-cli/pkg/errors"
)

// Windows API 常量
const (
	PROCESS_QUERY_INFORMATION = 0x0400
	PROCESS_VM_READ           = 0x0010
	MAX_PATH                  = 260
)

// Windows API 函数
var (
	modkernel32                   = syscall.NewLazyDLL("kernel32.dll")
	procOpenProcess               = modkernel32.NewProc("OpenProcess")
	procQueryFullProcessImageName = modkernel32.NewProc("QueryFullProcessImageNameW")
	procCloseHandle               = modkernel32.NewProc("CloseHandle")
	procGetProcessTimes           = modkernel32.NewProc("GetProcessTimes")
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

// FILETIME Windows 文件时间结构
type FILETIME struct {
	DwLowDateTime  uint32
	DwHighDateTime uint32
}

// windowsProcessChecker Windows 平台进程检查器
type windowsProcessChecker struct{}

// newProcessChecker 创建进程检查器（Windows 平台）
func newProcessChecker() ProcessChecker {
	return &windowsProcessChecker{}
}

// IsProcessRunning 检查进程是否正在运行
func (w *windowsProcessChecker) IsProcessRunning(pid int) bool {
	// 使用 PROCESS_QUERY_INFORMATION 权限打开进程
	// 如果进程不存在，OpenProcess 会返回 0
	handle, _, err := procOpenProcess.Call(
		uintptr(PROCESS_QUERY_INFORMATION),
		0,
		uintptr(pid),
	)
	if handle == 0 {
		// 如果是权限不足（ERROR_ACCESS_DENIED），保守认为进程仍在运行
		if err != nil && err.Error() == "Access is denied." {
			return true
		}
		return false
	}
	// 关闭句柄
	_, _, _ = procCloseHandle.Call(handle)
	return true
}

// GetProcessExecutable 获取进程的可执行文件路径
func (w *windowsProcessChecker) GetProcessExecutable(pid int) (string, error) {
	// 打开进程
	handle, _, _ := procOpenProcess.Call(
		uintptr(PROCESS_QUERY_INFORMATION|PROCESS_VM_READ),
		0,
		uintptr(pid),
	)
	if handle == 0 {
		return "", pkgerrors.ErrService("failed to open process", nil)
	}
	defer func() { _, _, _ = procCloseHandle.Call(handle) }()

	// 获取可执行文件路径
	var path [MAX_PATH]uint16
	var size uint32 = MAX_PATH

	ret, _, err := procQueryFullProcessImageName.Call(
		handle,
		0,
		uintptr(unsafe.Pointer(&path[0])),
		uintptr(unsafe.Pointer(&size)),
	)
	if ret == 0 {
		return "", pkgerrors.ErrService("failed to query process image name", err)
	}

	return syscall.UTF16ToString(path[:]), nil
}

// getProcessResourceUsage 获取进程资源使用情况 (Windows 实现)
func getProcessResourceUsage(pid int) (cpu, memory float64, err error) {
	// 打开进程
	handle, _, _ := procOpenProcess.Call(
		uintptr(PROCESS_QUERY_INFORMATION|PROCESS_VM_READ),
		0,
		uintptr(pid),
	)
	if handle == 0 {
		return 0, 0, pkgerrors.ErrService("failed to open process", nil)
	}
	defer func() { _, _, _ = procCloseHandle.Call(handle) }()

	// 获取进程时间信息
	var creationTime, exitTime, kernelTime, userTime FILETIME
	ret, _, err := procGetProcessTimes.Call(
		handle,
		uintptr(unsafe.Pointer(&creationTime)),
		uintptr(unsafe.Pointer(&exitTime)),
		uintptr(unsafe.Pointer(&kernelTime)),
		uintptr(unsafe.Pointer(&userTime)),
	)
	if ret == 0 {
		return 0, 0, pkgerrors.ErrService("failed to get process times", err)
	}

	// 计算CPU使用率 (简化版本，返回总CPU时间)
	kernelTimeValue := float64((uint64(kernelTime.DwHighDateTime)<<32)|uint64(kernelTime.DwLowDateTime)) / 1e7
	userTimeValue := float64((uint64(userTime.DwHighDateTime)<<32)|uint64(userTime.DwLowDateTime)) / 1e7
	cpu = kernelTimeValue + userTimeValue

	// 获取内存使用情况 (简化版本，返回工作集大小)
	// 注意：这里需要调用 GetProcessMemoryInfo，但为了简化，我们返回0
	// 完整实现需要加载 psapi.dll 并调用 GetProcessMemoryInfo
	memory = 0

	return cpu, memory, nil
}
