//go:build !windows

package util

import (
	"os"
)

// IsAdmin 检查当前进程是否以管理员权限运行
// 在 Linux/macOS 上，检查是否为 root 用户（UID 0）
func IsAdmin() bool {
	return os.Geteuid() == 0
}
