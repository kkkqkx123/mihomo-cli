package log

import (
	"context"
	"sync"
	"time"

	"github.com/kkkqkx123/mihomo-cli/internal/api"
	"github.com/kkkqkx123/mihomo-cli/pkg/types"
)

const (
	// DefaultMaxLogs 默认最大缓存日志数量
	DefaultMaxLogs = 10000
)

// LogCollector 日志收集器，用于临时缓存日志
type LogCollector struct {
	logs    []*types.LogInfo
	mu      sync.RWMutex
	maxLogs int
}

// NewLogCollector 创建新的日志收集器
func NewLogCollector(maxLogs int) *LogCollector {
	if maxLogs <= 0 {
		maxLogs = DefaultMaxLogs
	}
	return &LogCollector{
		logs:    make([]*types.LogInfo, 0, maxLogs),
		maxLogs: maxLogs,
	}
}

// Add 添加单条日志
func (lc *LogCollector) Add(log *types.LogInfo) {
	lc.mu.Lock()
	defer lc.mu.Unlock()

	// 如果达到最大缓存数量，移除最旧的日志
	if len(lc.logs) >= lc.maxLogs {
		lc.logs = lc.logs[1:]
	}

	lc.logs = append(lc.logs, log)
}

// GetLogs 获取所有日志
func (lc *LogCollector) GetLogs() []*types.LogInfo {
	lc.mu.RLock()
	defer lc.mu.RUnlock()

	// 返回日志的副本
	logs := make([]*types.LogInfo, len(lc.logs))
	copy(logs, lc.logs)
	return logs
}

// Clear 清空日志缓存
func (lc *LogCollector) Clear() {
	lc.mu.Lock()
	defer lc.mu.Unlock()

	lc.logs = make([]*types.LogInfo, 0, lc.maxLogs)
}

// Count 获取当前缓存的日志数量
func (lc *LogCollector) Count() int {
	lc.mu.RLock()
	defer lc.mu.RUnlock()
	return len(lc.logs)
}

// CollectWithDuration 从日志流中收集指定时间范围的日志
func (lc *LogCollector) CollectWithDuration(ctx context.Context, client *api.Client, duration time.Duration) error {
	ctx, cancel := context.WithTimeout(ctx, duration)
	defer cancel()

	// 获取日志流
	stream, err := client.StreamLogs(ctx)
	if err != nil {
		return err
	}
	defer stream.Close()

	// 读取日志消息
	for {
		select {
		case <-ctx.Done():
			// 超时或被取消，正常结束收集
			return nil
		case logMsg, ok := <-stream.Messages():
			if !ok {
				// 检查是否有错误
				if err := stream.Err(); err != nil {
					return err
				}
				return nil
			}
			lc.Add(logMsg)
		}
	}
}
