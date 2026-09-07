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

// delayTestClient 延迟测试所需的 API 客户端能力
type delayTestClient interface {
	TestDelay(ctx context.Context, name string, testURL string, timeout int) (uint16, error)
	GetProxy(ctx context.Context, name string) (*types.ProxyInfo, error)
	GroupDelay(ctx context.Context, groupName string, testURL string, timeout int) (map[string]uint16, error)
	ListProviders(ctx context.Context) (map[string]*types.ProviderInfo, error)
	TestProviderProxy(ctx context.Context, providerName, proxyName string, testURL string, timeout int) (uint16, error)
}

// ProviderProxyMapping provider 节点映射：节点名 -> provider 名
type ProviderProxyMapping map[string]string

// DelayTester 延迟测试器
type DelayTester struct {
	client     delayTestClient
	testURL    string
	timeout    int
	concurrent int
	progress   ProgressCallback
	providers  ProviderProxyMapping // provider 节点映射缓存
}

// NewDelayTester 创建新的延迟测试器
func NewDelayTester(client delayTestClient) *DelayTester {
	return &DelayTester{
		client:     client,
		testURL:    "https://www.google.com/generate_204",
		timeout:    5000,
		concurrent: 10,
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

// classifyTestError 根据 API 错误类型/HTTP 状态码分类失败原因
func classifyTestError(err error) (status, detail string) {
	if err == nil {
		return "", ""
	}

	apiErr, ok := err.(*api.APIError)
	if !ok {
		return "测试失败", err.Error()
	}

	switch {
	case api.IsAPIConnectionError(err):
		return "连接失败", apiErr.Message
	case api.IsTimeoutError(err):
		return "超时", apiErr.Message
	case apiErr.StatusCode == 400:
		return "参数错误", apiErr.Message
	case apiErr.StatusCode == 503:
		return "节点不可用", apiErr.Message
	default:
		return "测试失败", apiErr.Message
	}
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
		result.Status, result.Detail = classifyTestError(err)
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

// TestProviderProxy 测试 provider 中的单个节点延迟
func (t *DelayTester) TestProviderProxy(ctx context.Context, providerName, proxyName string) types.DelayResult {
	start := time.Now()
	result := types.DelayResult{
		Name: proxyName,
	}

	delay, err := t.client.TestProviderProxy(ctx, providerName, proxyName, t.testURL, t.timeout)
	result.Time = time.Since(start).Milliseconds()

	if err != nil {
		result.Error = err
		result.Status, result.Detail = classifyTestError(err)
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

// TestGroupNative 使用内核原生批量测速测试代理组
func (t *DelayTester) TestGroupNative(ctx context.Context, groupName string) ([]types.DelayResult, error) {
	dm, err := t.client.GroupDelay(ctx, groupName, t.testURL, t.timeout)
	if err != nil {
		return nil, pkgerrors.ErrAPI("failed to test group delay for "+groupName, err)
	}

	results := make([]types.DelayResult, 0, len(dm))
	for name, delay := range dm {
		result := types.DelayResult{
			Name:  name,
			Delay: delay,
			Time:  0, // 内核原生测速不返回客户端耗时
		}
		if delay == 0 {
			result.Status = "未知"
		} else if delay < 100 {
			result.Status = "优秀"
		} else if delay < 300 {
			result.Status = "良好"
		} else {
			result.Status = "较差"
		}
		results = append(results, result)
	}

	return results, nil
}

// TestGroup 测试代理组中所有节点的延迟
// 优先使用内核原生批量测速，如果失败则回退到逐个测试
func (t *DelayTester) TestGroup(ctx context.Context, groupName string) ([]types.DelayResult, error) {
	// 先尝试内核原生批量测速
	results, err := t.TestGroupNative(ctx, groupName)
	if err == nil {
		return results, nil
	}

	// 回退：获取代理组信息，逐个测试
	proxy, err := t.client.GetProxy(ctx, groupName)
	if err != nil {
		return nil, pkgerrors.ErrAPI("failed to get proxy group "+groupName, err)
	}

	if len(proxy.All) == 0 {
		return []types.DelayResult{}, nil
	}

	return t.TestNodes(ctx, proxy.All)
}

// BuildProviderMapping 构建 provider 节点映射
// 从所有 provider 中收集节点名到 provider 名的映射
func (t *DelayTester) BuildProviderMapping(ctx context.Context) error {
	providers, err := t.client.ListProviders(ctx)
	if err != nil {
		return err
	}

	t.providers = make(ProviderProxyMapping)
	for providerName, provider := range providers {
		for _, p := range provider.Proxies {
			t.providers[p.Name] = providerName
		}
	}
	return nil
}

// GetProviderName 获取节点所属的 provider 名称
// 如果节点不是 provider 节点，返回空字符串
func (t *DelayTester) GetProviderName(proxyName string) string {
	if t.providers == nil {
		return ""
	}
	return t.providers[proxyName]
}

// IsProviderProxy 判断节点是否为 provider 节点
func (t *DelayTester) IsProviderProxy(proxyName string) bool {
	return t.GetProviderName(proxyName) != ""
}

// TestNodes 测试多个节点的延迟
// 自动区分普通节点和 provider 节点，使用对应的测试方式
func (t *DelayTester) TestNodes(ctx context.Context, nodeNames []string) ([]types.DelayResult, error) {
	if len(nodeNames) == 0 {
		return []types.DelayResult{}, nil
	}

	// 按类型分组：普通节点和 provider 节点
	var normalNodes []string
	type providerNode struct {
		provider string
		name     string
	}
	var providerNodes []providerNode

	for _, name := range nodeNames {
		if providerName := t.GetProviderName(name); providerName != "" {
			providerNodes = append(providerNodes, providerNode{provider: providerName, name: name})
		} else {
			normalNodes = append(normalNodes, name)
		}
	}

	results := make([]types.DelayResult, len(nodeNames))
	resultIndex := 0
	var wg sync.WaitGroup
	var mu sync.Mutex

	sem := make(chan struct{}, t.concurrent)
	current := 0
	total := len(nodeNames)

	// 测试普通节点
	for _, name := range normalNodes {
		wg.Add(1)
		go func(n string) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			r := t.TestSingle(ctx, n)
			mu.Lock()
			results[resultIndex] = r
			resultIndex++
			current++
			mu.Unlock()

			if t.progress != nil {
				t.progress(current, total, n)
			}
		}(name)
	}

	// 测试 provider 节点
	for _, pn := range providerNodes {
		wg.Add(1)
		go func(provider, name string) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			r := t.TestProviderProxy(ctx, provider, name)
			mu.Lock()
			results[resultIndex] = r
			resultIndex++
			current++
			mu.Unlock()

			if t.progress != nil {
				t.progress(current, total, name)
			}
		}(pn.provider, pn.name)
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
