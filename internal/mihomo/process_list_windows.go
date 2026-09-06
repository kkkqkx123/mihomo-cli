//go:build windows

package mihomo

import (
	"unsafe"

	"golang.org/x/sys/windows"

	pkgerrors "github.com/kkkqkx123/mihomo-cli/pkg/errors"
)

// listProcessPIDs 枚举系统全部进程 PID（CreateToolhelp32Snapshot + Process32First/Next）。
// 只读遍历，不产生副作用；失败返回错误（调用方降级 external 扫描）。
func listProcessPIDs() ([]int, error) {
	snapshot, err := windows.CreateToolhelp32Snapshot(windows.TH32CS_SNAPPROCESS, 0)
	if err != nil {
		return nil, pkgerrors.ErrService("failed to create process snapshot", err)
	}
	defer windows.CloseHandle(snapshot)

	var pids []int
	var entry windows.ProcessEntry32
	entry.Size = uint32(unsafe.Sizeof(entry))

	err = windows.Process32First(snapshot, &entry)
	if err != nil {
		return nil, pkgerrors.ErrService("failed to enumerate processes", err)
	}

	for {
		pids = append(pids, int(entry.ProcessID))
		if err := windows.Process32Next(snapshot, &entry); err != nil {
			break
		}
	}

	return pids, nil
}
