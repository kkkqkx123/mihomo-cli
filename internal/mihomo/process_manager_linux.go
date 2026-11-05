//go:build linux

package mihomo

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// detectProcessManager 通过 cgroup 判断进程是否由 systemd 管理。
// 返回最后一个 .service 组件，避免把 user@1000.service 当成具体服务。
func detectProcessManager(pid int) (string, string) {
	data, err := os.ReadFile(filepath.Join("/proc", strconv.Itoa(pid), "cgroup"))
	if err != nil {
		return "", ""
	}

	unit := lastSystemdService(string(data))
	if unit == "" {
		return "", ""
	}
	return StatusManagerSystemd, unit
}

// lastSystemdService 从 cgroup 内容中提取最末级 systemd 服务单元。
func lastSystemdService(content string) string {
	unit := ""
	for _, line := range strings.Split(content, "\n") {
		parts := strings.SplitN(line, ":", 3)
		if len(parts) != 3 {
			continue
		}
		for _, component := range strings.Split(parts[2], "/") {
			if !strings.HasSuffix(component, ".service") {
				continue
			}
			// user@<uid>.service 是用户管理器本身，不是具体应用服务。
			if strings.HasPrefix(component, "user@") {
				continue
			}
			unit = component
		}
	}
	return unit
}
