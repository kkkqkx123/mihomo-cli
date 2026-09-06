//go:build !windows && !linux && !darwin

package mihomo

import (
	"runtime"

	pkgerrors "github.com/kkkqkx123/mihomo-cli/pkg/errors"
)

// listProcessPIDs 枚举系统全部进程 PID（不支持的平台：返回空列表 + 明确错误）
func listProcessPIDs() ([]int, error) {
	return nil, pkgerrors.ErrService(
		"process enumeration not supported on "+runtime.GOOS, nil)
}
