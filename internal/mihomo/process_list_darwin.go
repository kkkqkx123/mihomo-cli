//go:build darwin

package mihomo

import (
	"golang.org/x/sys/unix"
)

// listProcessPIDs 枚举系统全部进程 PID（sysctl kern.proc.all）。
func listProcessPIDs() ([]int, error) {
	kinfos, err := unix.SysctlKinfoProcSlice("kern.proc.all")
	if err != nil {
		return nil, err
	}

	pids := make([]int, 0, len(kinfos))
	for _, k := range kinfos {
		pids = append(pids, int(k.Eproc.Ppid))
	}

	return pids, nil
}
