//go:build windows

package util

import (
	"golang.org/x/sys/windows"
)

// IsAdmin 检查当前进程是否以管理员权限运行
func IsAdmin() bool {
	var sid *windows.SID

	// 创建管理员组 SID
	err := windows.AllocateAndInitializeSid(
		&windows.SECURITY_NT_AUTHORITY,
		2,
		windows.SECURITY_BUILTIN_DOMAIN_RID,
		windows.DOMAIN_ALIAS_RID_ADMINS,
		0, 0, 0, 0, 0, 0,
		&sid,
	)
	if err != nil {
		return false
	}
	defer windows.FreeSid(sid)

	// 检查当前进程令牌是否属于管理员组
	token := windows.Token(0)
	member, err := token.IsMember(sid)
	if err != nil {
		return false
	}

	return member
}
