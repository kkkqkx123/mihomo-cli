package mihomo

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/kkkqkx123/mihomo-cli/internal/config"
	pkgerrors "github.com/kkkqkx123/mihomo-cli/pkg/errors"
)

// ProcessLock 进程锁
type ProcessLock struct {
	lockFile string
	lock     *os.File
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

// Acquire 获取锁
func (pl *ProcessLock) Acquire() error {
	// 确保目录存在
	lockDir := filepath.Dir(pl.lockFile)
	if err := os.MkdirAll(lockDir, 0755); err != nil {
		return pkgerrors.ErrService("failed to create lock directory", err)
	}

	// 尝试创建锁文件
	f, err := os.OpenFile(pl.lockFile, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0644)
	if err != nil {
		if os.IsExist(err) {
			// 锁文件已存在，检查是否是陈旧的锁
			if err := pl.checkStaleLock(); err != nil {
				return err
			}
			// 陈旧的锁已清理，重试
			return pl.Acquire()
		}
		return pkgerrors.ErrService("failed to create lock file", err)
	}

	// 写入当前进程 PID
	pid := os.Getpid()
	if _, err := f.WriteString(strconv.Itoa(pid)); err != nil {
		f.Close()
		os.Remove(pl.lockFile)
		return pkgerrors.ErrService("failed to write pid to lock file", err)
	}

	pl.lock = f
	return nil
}

// Release 释放锁
func (pl *ProcessLock) Release() error {
	if pl.lock != nil {
		pl.lock.Close()
		pl.lock = nil
	}

	if err := os.Remove(pl.lockFile); err != nil && !os.IsNotExist(err) {
		return pkgerrors.ErrService("failed to remove lock file", err)
	}

	return nil
}

// IsLocked 检查是否已锁定
func (pl *ProcessLock) IsLocked() bool {
	if _, err := os.Stat(pl.lockFile); err == nil {
		return true
	}
	return false
}

// GetLockInfo 获取锁信息
func (pl *ProcessLock) GetLockInfo() (int, error) {
	// 读取锁文件
	data, err := os.ReadFile(pl.lockFile)
	if err != nil {
		if os.IsNotExist(err) {
			return 0, nil
		}
		return 0, pkgerrors.ErrService("failed to read lock file", err)
	}

	// 解析 PID
	pid, err := strconv.Atoi(string(data))
	if err != nil {
		return 0, pkgerrors.ErrService("invalid pid in lock file", err)
	}

	return pid, nil
}

// checkStaleLock 检查是否是陈旧的锁
func (pl *ProcessLock) checkStaleLock() error {
	// 读取锁文件中的 PID
	pid, err := pl.GetLockInfo()
	if err != nil {
		return err
	}

	// 检查进程是否还在运行
	if !IsProcessRunning(pid) {
		// 进程已退出，清理陈旧的锁
		if err := os.Remove(pl.lockFile); err != nil {
			return pkgerrors.ErrService("failed to remove stale lock file", err)
		}
		return nil
	}

	// 进程还在运行，返回错误
	return pkgerrors.ErrService(fmt.Sprintf("process is already running (PID: %d)", pid), nil)
}

// TryAcquire 尝试获取锁（带超时）
func (pl *ProcessLock) TryAcquire(timeout time.Duration) error {
	deadline := time.Now().Add(timeout)

	for {
		err := pl.Acquire()
		if err == nil {
			return nil
		}

		// 如果不是"已锁定"错误，直接返回
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
	defer pl.Release()

	return fn()
}
