//go:build linux

package mihomo

import (
	"os"
	"strconv"
)

// listProcessPIDs 枚举系统全部进程 PID（遍历 /proc/[0-9]+ 目录）。
func listProcessPIDs() ([]int, error) {
	entries, err := os.ReadDir("/proc")
	if err != nil {
		return nil, err
	}

	var pids []int
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		pid, err := strconv.Atoi(entry.Name())
		if err != nil {
			continue // 非数字目录（非进程）
		}
		pids = append(pids, pid)
	}

	return pids, nil
}
