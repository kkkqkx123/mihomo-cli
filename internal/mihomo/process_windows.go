//go:build windows

package mihomo

import (
	"os/exec"
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
	modkernel32                    = syscall.NewLazyDLL("kernel32.dll")
	procOpenProcess                = modkernel32.NewProc("OpenProcess")
	procQueryFullProcessImageName  = modkernel32.NewProc("QueryFullProcessImageNameW")
	procCloseHandle                = modkernel32.NewProc("CloseHandle")
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
	// 如果进程不存在，OpenProcess 会返回 0
	handle, _, _ := procOpenProcess.Call(
		uintptr(PROCESS_QUERY_INFORMATION),
		0,
		uintptr(pid),
	)
	if handle == 0 {
		return false
	}
	// 关闭句柄
	_, _, _ = procCloseHandle.Call(handle)
	return true
}

// GetProcessExecutable 获取进程的可执行文件路径
func (w *windowsProcessChecker) GetProcessExecutable(pid int) (string, error) {
	// 打开进程
	handle, _, err := procOpenProcess.Call(
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

// SetSysProcAttr 设置进程的系统属性（隐藏窗口）
func (w *windowsProcessChecker) SetSysProcAttr(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{
		HideWindow: true,
	}
}
