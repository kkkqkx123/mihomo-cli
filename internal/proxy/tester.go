package proxy

import (
	"context"
	"sync"
	"time"

	"github.com/kkkqkx123/mihomo-cli/internal/api"
	"github.com/kkkqkx123/mihomo-cli/pkg/types"

	pkgerrors "github.com/kkkqkx123/mihomo-cli/pkg/errors"
)

// ProgressCallback 进度回调函数类型
type ProgressCallback func(current, total int, nodeName string)

// DelayTester 延迟测试器
type DelayTester struct {
	client    *api.Client
	testURL   string
	timeout   int
	concurrent int
	progress  ProgressCallback
}

// NewDelayTester 创建新的延迟测试器
func NewDelayTester(client *api.Client) *DelayTester {
	return &DelayTester{
		client:     client,
		testURL:    "", // 使用默认测试 URL
		timeout:    5000, // 默认 5 秒超时
		concurrent: 10,  // 默认并发 10
	}
}

// SetProgress 设置进度回调
func (t *DelayTester) SetProgress(progress ProgressCallback) {
	t.progress = progress
}

// SetTestURL 设置测试 URL
func (t *DelayTester) SetTestURL(url string) {
	t.testURL = url
}

// SetTimeout 设置超时时间（毫秒）
func (t *DelayTester) SetTimeout(timeout int) {
	t.timeout = timeout
}

// SetConcurrent 设置并发数
func (t *DelayTester) SetConcurrent(concurrent int) {
	t.concurrent = concurrent
}

// TestSingle 测试单个代理的延迟
func (t *DelayTester) TestSingle(ctx context.Context, proxyName string) types.DelayResult {
	start := time.Now()
	result := types.DelayResult{
		Name: proxyName,
	}

	delay, err := t.client.TestDelay(ctx, proxyName, t.testURL, t.timeout)
	result.Time = time.Since(start).Milliseconds()

	if err != nil {
		result.Error = err
		result.Status = "超时"
	} else if delay == 0 {
		result.Status = "未知"
	} else {
		result.Delay = delay
		if delay < 100 {
			result.Status = "优秀"
		} else if delay < 300 {
			result.Status = "良好"
		} else {
			result.Status = "较差"
		}
	}

	return result
}

// TestGroup 测试代理组中所有节点的延迟
func (t *DelayTester) TestGroup(ctx context.Context, groupName string) ([]types.DelayResult, error) {
	// 获取代理组信息
	proxy, err := t.client.GetProxy(ctx, groupName)
	if err != nil {
		return nil, pkgerrors.ErrAPI("failed to get proxy group "+groupName, err)
	}

	// 如果没有节点，返回空结果
	if len(proxy.All) == 0 {
		return []types.DelayResult{}, nil
	}

	// 并发测试所有节点
	return t.TestNodes(ctx, proxy.All)
}

// TestNodes 测试多个节点的延迟
func (t *DelayTester) TestNodes(ctx context.Context, nodeNames []string) ([]types.DelayResult, error) {
	if len(nodeNames) == 0 {
		return []types.DelayResult{}, nil
	}

	results := make([]types.DelayResult, len(nodeNames))
	var wg sync.WaitGroup

	// 使用信号量控制并发数
	sem := make(chan struct{}, t.concurrent)

	for i, nodeName := range nodeNames {
		wg.Add(1)
		go func(index int, name string) {
			defer wg.Done()

			// 获取信号量
			sem <- struct{}{}
			defer func() { <-sem }()

			results[index] = t.TestSingle(ctx, name)

			// 调用进度回调
			if t.progress != nil {
				t.progress(index+1, len(nodeNames), name)
			}
		}(i, nodeName)
	}

	wg.Wait()
	return results, nil
}

// TestAll 测试所有代理组的延迟
func (t *DelayTester) TestAll(ctx context.Context, groupNames []string) (map[string][]types.DelayResult, error) {
	results := make(map[string][]types.DelayResult)
	var mu sync.Mutex
	var wg sync.WaitGroup

	for _, groupName := range groupNames {
		wg.Add(1)
		go func(name string) {
			defer wg.Done()

			groupResults, err := t.TestGroup(ctx, name)
			if err == nil {
				mu.Lock()
				results[name] = groupResults
				mu.Unlock()
			}
		}(groupName)
	}

	wg.Wait()
	return results, nil
}