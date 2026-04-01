package mihomo

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/kkkqkx123/mihomo-cli/internal/config"
	pkgerrors "github.com/kkkqkx123/mihomo-cli/pkg/errors"
)

// ProcessLock 进程锁（使用系统级文件锁）
type ProcessLock struct {
	lockFile string
	lock     FileLock
}

// FileLock 文件锁接口（跨平台抽象）
type FileLock interface {
	Lock() error
	Unlock() error
}

// NewProcessLock 创建进程锁
func NewProcessLock(configFile string) (*ProcessLock, error) {
	// 获取锁文件路径
	lockFile, err := getLockFilePath(configFile)
	if err != nil {
		return nil, err
	}

	return &ProcessLock{
		lockFile: lockFile,
	}, nil
}

// getLockFilePath 获取锁文件路径
func getLockFilePath(configFile string) (string, error) {
	baseDir, err := config.GetBaseDir()
	if err != nil {
		return "", err
	}

	// 根据配置文件生成唯一的锁文件名
	hash := generateConfigHash(configFile)
	return filepath.Join(baseDir, fmt.Sprintf("lock-%s", hash)), nil
}

// Acquire 获取锁（使用系统级文件锁）
func (pl *ProcessLock) Acquire() error {
	// 确保目录存在
	lockDir := filepath.Dir(pl.lockFile)
	if err := os.MkdirAll(lockDir, 0755); err != nil {
		return pkgerrors.ErrService("failed to create lock directory", err)
	}

	// 创建或打开锁文件
	file, err := os.OpenFile(pl.lockFile, os.O_CREATE|os.O_RDWR, 0644)
	if err != nil {
		return pkgerrors.ErrService("failed to open lock file", err)
	}

	// 创建平台特定的文件锁
	lock := newSystemFileLock(file)

	// 尝试获取锁
	if err := lock.Lock(); err != nil {
		file.Close()
		return pl.handleLockError(err)
	}

	// 写入当前进程 PID
	pid := os.Getpid()
	if _, err := fmt.Fprintf(file, "%d", pid); err != nil {
		lock.Unlock()
		file.Close()
		return pkgerrors.ErrService("failed to write pid to lock file", err)
	}

	pl.lock = lock
	return nil
}

// handleLockError 处理锁获取失败的情况
func (pl *ProcessLock) handleLockError(err error) error {
	// 检查是否是陈旧的锁
	pid, readErr := pl.readLockFilePID()
	if readErr == nil && pid > 0 {
		if !IsProcessRunning(pid) {
			// 进程已退出，清理陈旧的锁文件
			os.Remove(pl.lockFile)
			// 重试获取锁
			return pl.Acquire()
		}
		return pkgerrors.ErrService(fmt.Sprintf("process is already running (PID: %d)", pid), nil)
	}
	return pkgerrors.ErrService("failed to acquire lock", err)
}

// readLockFilePID 从锁文件读取 PID
func (pl *ProcessLock) readLockFilePID() (int, error) {
	data, err := os.ReadFile(pl.lockFile)
	if err != nil {
		return 0, err
	}

	var pid int
	_, err = fmt.Sscanf(string(data), "%d", &pid)
	return pid, err
}

// Release 释放锁
func (pl *ProcessLock) Release() error {
	if pl.lock != nil {
		if err := pl.lock.Unlock(); err != nil {
			// 记录错误但继续清理
		}
		pl.lock = nil
	}

	// 删除锁文件
	if err := os.Remove(pl.lockFile); err != nil && !os.IsNotExist(err) {
		return pkgerrors.ErrService("failed to remove lock file", err)
	}

	return nil
}

// IsLocked 检查是否已锁定
func (pl *ProcessLock) IsLocked() bool {
	// 尝试获取非阻塞锁来检查
	file, err := os.OpenFile(pl.lockFile, os.O_CREATE|os.O_RDWR, 0644)
	if err != nil {
		return false
	}
	defer file.Close()

	lock := newSystemFileLock(file)
	if err := lock.Lock(); err != nil {
		return true // 已被锁定
	}

	// 获取成功，立即释放
	lock.Unlock()
	return false
}

// GetLockInfo 获取锁信息
func (pl *ProcessLock) GetLockInfo() (int, error) {
	return pl.readLockFilePID()
}

// TryAcquire 尝试获取锁（带超时）
func (pl *ProcessLock) TryAcquire(timeout time.Duration) error {
	deadline := time.Now().Add(timeout)

	for {
		err := pl.Acquire()
		if err == nil {
			return nil
		}

		// 检查是否是"已锁定"错误
		if !isAlreadyLockedError(err) {
			return err
		}

		// 检查是否超时
		if time.Now().After(deadline) {
			return pkgerrors.ErrService("failed to acquire lock: timeout", nil)
		}

		// 等待一段时间后重试
		time.Sleep(100 * time.Millisecond)
	}
}

// isAlreadyLockedError 检查是否是"已锁定"错误
func isAlreadyLockedError(err error) bool {
	if err == nil {
		return false
	}
	errStr := err.Error()
	return len(errStr) > 20 && errStr[:20] == "process is already"
}

// WithLock 使用锁执行函数
func (pl *ProcessLock) WithLock(fn func() error) error {
	if err := pl.Acquire(); err != nil {
		return err
	}
	defer func() {
		_ = pl.Release()
	}()

	return fn()
}