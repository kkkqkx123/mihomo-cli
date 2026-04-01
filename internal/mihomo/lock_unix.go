//go:build linux || darwin

package mihomo

import (
	"os"
	"syscall"
)

// unixFileLock Unix 平台文件锁实现（使用 flock）
type unixFileLock struct {
	file *os.File
}

// newSystemFileLock 创建系统特定的文件锁
func newSystemFileLock(file *os.File) FileLock {
	return &unixFileLock{file: file}
}

// Lock 获取文件锁（阻塞）
func (l *unixFileLock) Lock() error {
	// 使用 flock 系统调用获取独占锁
	// LOCK_EX: 独占锁
	// LOCK_NB: 非阻塞（我们不使用，因为要阻塞等待）
	return syscall.Flock(int(l.file.Fd()), syscall.LOCK_EX)
}

// Unlock 释放文件锁
func (l *unixFileLock) Unlock() error {
	// 使用 flock 系统调用释放锁
	err := syscall.Flock(int(l.file.Fd()), syscall.LOCK_UN)
	if err != nil {
		return err
	}
	// 关闭文件
	return l.file.Close()
}

// TryLock 尝试获取文件锁（非阻塞）
func (l *unixFileLock) TryLock() error {
	// LOCK_EX | LOCK_NB: 独占锁，非阻塞
	return syscall.Flock(int(l.file.Fd()), syscall.LOCK_EX|syscall.LOCK_NB)
}
