package mihomo

import (
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// FindProcessByPort 查找占用指定端口的进程 PID（通过 /proc/net/tcp）
func FindProcessByPort(host, port string) int {
	data, err := os.ReadFile("/proc/net/tcp")
	if err != nil {
		return 0
	}

	var portNum int
	fmt.Sscanf(port, "%d", &portNum)
	if portNum <= 0 {
		return 0
	}
	targetPort := fmt.Sprintf("%04X", portNum)

	targetAddr := ""
	if host != "" && host != "0.0.0.0" && host != "*" {
		if ip := net.ParseIP(host); ip != nil {
			ip4 := ip.To4()
			if ip4 != nil {
				targetAddr = fmt.Sprintf("%02X%02X%02X%02X", ip4[3], ip4[2], ip4[1], ip4[0])
			}
		}
	}

	lines := strings.Split(string(data), "\n")
	for _, line := range lines[1:] {
		fields := strings.Fields(line)
		if len(fields) < 10 {
			continue
		}
		localAddr := fields[1]
		parts := strings.Split(localAddr, ":")
		if len(parts) != 2 {
			continue
		}
		if parts[1] != targetPort {
			continue
		}
		if targetAddr != "" && parts[0] != targetAddr {
			continue
		}
		inode := fields[9]
		if inode == "0" {
			continue
		}
		if pid := findPIDByInode(inode); pid > 0 {
			return pid
		}
	}
	return 0
}

// findPIDByInode 通过 socket inode 查找 PID
func findPIDByInode(inode string) int {
	entries, err := os.ReadDir("/proc")
	if err != nil {
		return 0
	}
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		pid := entry.Name()
		if _, err := strconv.Atoi(pid); err != nil {
			continue
		}
		fdDir := filepath.Join("/proc", pid, "fd")
		fds, err := os.ReadDir(fdDir)
		if err != nil {
			continue
		}
		for _, fd := range fds {
			link, err := os.Readlink(filepath.Join(fdDir, fd.Name()))
			if err != nil {
				continue
			}
			if link == "socket:"+inode {
				pidNum, _ := strconv.Atoi(pid)
				return pidNum
			}
		}
	}
	return 0
}
