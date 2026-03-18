package proxy

import (
	"testing"

	"github.com/kkkqkx123/mihomo-cli/pkg/types"
)

func TestFilterProxies(t *testing.T) {
	// 创建测试数据
	proxies := map[string]*types.ProxyInfo{
		"Node1": {
			Name:  "Node1",
			Type:  "Vmess",
			Alive: true,
		},
		"Node2": {
			Name:  "Node2",
			Type:  "Vmess",
			Alive: false,
		},
		"Node3": {
			Name:  "Node3",
			Type:  "Shadowsocks",
			Alive: true,
		},
		"DIRECT": {
			Name:  "DIRECT",
			Type:  "Direct",
			Alive: true,
		},
		"REJECT": {
			Name:  "REJECT",
			Type:  "Reject",
			Alive: true,
		},
		"ProxyGroup": {
			Name:  "ProxyGroup",
			Type:  "Selector",
			Alive: true,
			All:   []string{"Node1", "Node2"},
		},
	}

	tests := []struct {
		name     string
		opts     FilterOptions
		expected int // 期望的结果数量
	}{
		{
			name:     "无过滤",
			opts:     FilterOptions{},
			expected: 6,
		},
		{
			name: "按类型过滤 Vmess",
			opts: FilterOptions{
				Type: "Vmess",
			},
			expected: 2,
		},
		{
			name: "按状态过滤 alive",
			opts: FilterOptions{
				Status: "alive",
			},
			expected: 5,
		},
		{
			name: "排除逻辑节点",
			opts: FilterOptions{
				ExcludeLogical: true,
			},
			expected: 4,
		},
		{
			name: "只显示代理组",
			opts: FilterOptions{
				GroupsOnly: true,
			},
			expected: 1,
		},
		{
			name: "只显示节点",
			opts: FilterOptions{
				NodesOnly: true,
			},
			expected: 5,
		},
		{
			name: "组合过滤：Vmess 且 alive",
			opts: FilterOptions{
				Type:   "Vmess",
				Status: "alive",
			},
			expected: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FilterProxies(proxies, tt.opts)
			if len(result) != tt.expected {
				t.Errorf("期望 %d 个结果，实际得到 %d 个", tt.expected, len(result))
			}
		})
	}
}

func TestFilterProxiesWithExcludeRegex(t *testing.T) {
	proxies := map[string]*types.ProxyInfo{
		"Node1": {
			Name:  "Node1",
			Type:  "Vmess",
			Alive: true,
		},
		"过滤掉12条线路": {
			Name:  "过滤掉12条线路",
			Type:  "Vmess",
			Alive: true,
		},
		"剩余流量：96.31 GB": {
			Name:  "剩余流量：96.31 GB",
			Type:  "Vmess",
			Alive: true,
		},
		"套餐到期：长期有效": {
			Name:  "套餐到期：长期有效",
			Type:  "Vmess",
			Alive: true,
		},
	}

	tests := []struct {
		name     string
		opts     FilterOptions
		expected int
	}{
		{
			name: "排除包含'过滤掉'的节点",
			opts: FilterOptions{
				ExcludeRegex: "过滤掉",
			},
			expected: 3,
		},
		{
			name: "排除包含'剩余流量'或'套餐到期'的节点",
			opts: FilterOptions{
				ExcludeRegex: "剩余流量|套餐到期",
			},
			expected: 2,
		},
		{
			name: "排除所有特殊节点",
			opts: FilterOptions{
				ExcludeRegex: "过滤掉|剩余流量|套餐到期",
			},
			expected: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FilterProxies(proxies, tt.opts)
			if len(result) != tt.expected {
				t.Errorf("期望 %d 个结果，实际得到 %d 个", tt.expected, len(result))
			}
		})
	}
}