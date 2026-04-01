//go:build windows

package mihomo

import (
	"os"
	"syscall"

	"golang.org/x/sys/windows"
)

// windowsFileLock Windows 平台文件锁实现
type windowsFileLock struct {
	file *os.File
	handle windows.Handle
}

// newSystemFileLock 创建系统特定的文件锁
func newSystemFileLock(file *os.File) FileLock {
	return &windowsFileLock{
		file:   file,
		handle: windows.Handle(file.Fd()),
	}
}

// Lock 获取文件锁（阻塞）
func (l *windowsFileLock) Lock() error {
	// Windows 使用 LockFileEx 进行文件锁定
	var overlapped windows.Overlapped

	return windows.LockFileEx(
		l.handle,
		windows.LOCKFILE_EXCLUSIVE_LOCK, // 独占锁
		0,                               // 保留
		0xFFFFFFFF,                      // 锁定整个文件（低32位）
		0xFFFFFFFF,                      // 锁定整个文件（高32位）
		&overlapped,
	)
}

// Unlock 释放文件锁
func (l *windowsFileLock) Unlock() error {
	var overlapped windows.Overlapped

	if err := windows.UnlockFileEx(
		l.handle,
		0,          // 保留
		0xFFFFFFFF, // 解锁整个文件（低32位）
		0xFFFFFFFF, // 解锁整个文件（高32位）
		&overlapped,
	); err != nil {
		return err
	}

	// 关闭文件
	return l.file.Close()
}

// syscallFileLock 使用 syscall 的备用实现（如果 golang.org/x/sys/windows 不可用）
type syscallFileLock struct {
	file   *os.File
	handle uintptr
}

// newSyscallFileLock 创建基于 syscall 的文件锁
func newSyscallFileLock(file *os.File) FileLock {
	return &syscallFileLock{
		file:   file,
		handle: file.Fd(),
	}
}

// Lock 获取文件锁
func (l *syscallFileLock) Lock() error {
	// 使用 syscall 的 LockFile
	kernel32 := syscall.NewLazyDLL("kernel32.dll")
	lockFile := kernel32.NewProc("LockFile")

	ret, _, err := lockFile.Call(
		l.handle,
		0, // 文件开始位置（低32位）
		0, // 文件开始位置（高32位）
		1, // 锁定大小（低32位）- 锁定1字节足够
		0, // 锁定大小（高32位）
	)

	if ret == 0 {
		return err
	}
	return nil
}

// Unlock 释放文件锁
func (l *syscallFileLock) Unlock() error {
	kernel32 := syscall.NewLazyDLL("kernel32.dll")
	unlockFile := kernel32.NewProc("UnlockFile")

	ret, _, err := unlockFile.Call(
		l.handle,
		0, // 文件开始位置（低32位）
		0, // 文件开始位置（高32位）
		1, // 解锁大小（低32位）
		0, // 解锁大小（高32位）
	)

	if ret == 0 {
		return err
	}

	return l.file.Close()
}
