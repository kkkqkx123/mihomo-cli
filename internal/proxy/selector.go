package proxy

import (
	"context"
	"fmt"
	"sort"

	"github.com/kkkqkx123/mihomo-cli/internal/api"
	pkgerrors "github.com/kkkqkx123/mihomo-cli/pkg/errors"
)

// Selector 节点选择器
type Selector struct {
	client *api.Client
	tester *DelayTester
}

// NewSelector 创建新的节点选择器
func NewSelector(client *api.Client) *Selector {
	return &Selector{
		client: client,
		tester: NewDelayTester(client),
	}
}

// SetTester 设置延迟测试器
func (s *Selector) SetTester(tester *DelayTester) {
	s.tester = tester
}

// SelectBestNode 选择延迟最低的节点
func (s *Selector) SelectBestNode(ctx context.Context, groupName string) (string, uint16, error) {
	// 测试代理组中所有节点的延迟
	results, err := s.tester.TestGroup(ctx, groupName)
	if err != nil {
		return "", 0, pkgerrors.ErrAPI("failed to test node delay", err)
	}

	// 筛选出测试成功的节点
	var validResults []DelayResultInfo
	for _, result := range results {
		if result.Error == nil && result.Delay > 0 {
			validResults = append(validResults, DelayResultInfo{
				Name:  result.Name,
				Delay: result.Delay,
			})
		}
	}

	// 如果没有可用的节点
	if len(validResults) == 0 {
		return "", 0, pkgerrors.ErrAPI("no available nodes", nil)
	}

	// 按延迟排序
	sort.Slice(validResults, func(i, j int) bool {
		return validResults[i].Delay < validResults[j].Delay
	})

	// 返回延迟最低的节点
	bestNode := validResults[0]
	return bestNode.Name, bestNode.Delay, nil
}

// SelectAndSwitch 选择并切换到最快节点
func (s *Selector) SelectAndSwitch(ctx context.Context, groupName string) (string, uint16, error) {
	// 选择最佳节点
	bestNode, delay, err := s.SelectBestNode(ctx, groupName)
	if err != nil {
		return "", 0, err
	}

	// 切换到最佳节点
	err = s.client.SwitchProxy(ctx, groupName, bestNode)
	if err != nil {
		return "", 0, pkgerrors.ErrAPI("failed to switch to node "+bestNode, err)
	}

	return bestNode, delay, nil
}

// DelayResultInfo 延迟结果信息（用于排序）
type DelayResultInfo struct {
	Name  string
	Delay uint16
}

// SelectBestNodesByCount 选择前 N 个延迟最低的节点
func (s *Selector) SelectBestNodesByCount(ctx context.Context, groupName string, count int) ([]string, error) {
	// 测试代理组中所有节点的延迟
	results, err := s.tester.TestGroup(ctx, groupName)
	if err != nil {
		return nil, pkgerrors.ErrAPI("failed to test node delay", err)
	}

	// 筛选出测试成功的节点
	var validResults []DelayResultInfo
	for _, result := range results {
		if result.Error == nil && result.Delay > 0 {
			validResults = append(validResults, DelayResultInfo{
				Name:  result.Name,
				Delay: result.Delay,
			})
		}
	}

	// 如果没有可用的节点
	if len(validResults) == 0 {
		return nil, pkgerrors.ErrAPI("no available nodes", nil)
	}

	// 按延迟排序
	sort.Slice(validResults, func(i, j int) bool {
		return validResults[i].Delay < validResults[j].Delay
	})

	// 返回前 N 个节点
	var bestNodes []string
	for i := 0; i < count && i < len(validResults); i++ {
		bestNodes = append(bestNodes, validResults[i].Name)
	}

	return bestNodes, nil
}

// SelectByThreshold 选择延迟低于阈值的节点
func (s *Selector) SelectByThreshold(ctx context.Context, groupName string, threshold uint16) ([]string, error) {
	// 测试代理组中所有节点的延迟
	results, err := s.tester.TestGroup(ctx, groupName)
	if err != nil {
		return nil, pkgerrors.ErrAPI("failed to test node delay", err)
	}

	// 筛选出延迟低于阈值的节点
	var goodNodes []string
	for _, result := range results {
		if result.Error == nil && result.Delay > 0 && result.Delay <= threshold {
			goodNodes = append(goodNodes, result.Name)
		}
	}

	// 如果没有符合条件的节点
	if len(goodNodes) == 0 {
		return nil, pkgerrors.ErrAPI("no nodes with delay below "+fmt.Sprintf("%d", threshold)+"ms", nil)
	}

	return goodNodes, nil
}