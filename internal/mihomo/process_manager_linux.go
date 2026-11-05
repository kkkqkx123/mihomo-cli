//go:build linux

package mihomo

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// systemdUnitNamePattern 限制单元名格式，避免把不可信的 cgroup 组件直接拼进 systemctl 参数。
var systemdUnitNamePattern = regexp.MustCompile(`^[A-Za-z0-9_.@:-]+\.service$`)

// detectProcessManager 通过 cgroup 判断进程是否由 systemd 管理。
// cgroup 是进程→单元映射的权威来源；找到单元名后，再用 systemctl 的 MainPID 做交叉校验，
// 避免 PID 复用或陈旧 cgroup 造成误报。systemctl 不可用时信任 cgroup，不丢失信息。
func detectProcessManager(pid int) (string, string) {
	data, err := os.ReadFile(filepath.Join("/proc", strconv.Itoa(pid), "cgroup"))
	if err != nil {
		return "", ""
	}
	content := string(data)

	unit := lastSystemdService(content)
	if unit == "" || !systemdUnitNamePattern.MatchString(unit) {
		return "", ""
	}

	preferUser := strings.Contains(content, "user.slice") || strings.Contains(content, "user@")

	// 交叉校验：systemctl 明确归属到另一个进程时，判定 cgroup 信息陈旧。
	mainPID, verified := systemdUnitMainPID(unit, preferUser)
	if !verified {
		mainPID, verified = systemdUnitMainPID(unit, !preferUser)
	}
	if verified && mainPID != pid {
		return "", ""
	}

	return StatusManagerSystemd, unit
}

// systemdUnitMainPID 查询 systemctl 报告的单元 MainPID。返回 (pid, true) 表示调用成功并得到正整数。
func systemdUnitMainPID(unit string, user bool) (int, bool) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	args := []string{"show", unit, "-p", "MainPID", "--value"}
	if user {
		args = append([]string{"--user"}, args...)
	}
	cmd := exec.CommandContext(ctx, "systemctl", args...)
	out, err := cmd.Output()
	if err != nil {
		return 0, false
	}
	value := strings.TrimSpace(string(out))
	pid, err := strconv.Atoi(value)
	if err != nil || pid <= 0 {
		return 0, false
	}
	return pid, true
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
			if strings.HasPrefix(component, "user@") {
				continue
			}
			unit = component
		}
	}
	return unit
}
