//go:build !linux

package mihomo

// detectProcessManager 非 Linux 平台不读取 cgroup。
func detectProcessManager(pid int) (string, string) {
	return "", ""
}
