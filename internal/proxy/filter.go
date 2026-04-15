package proxy

import (
	"regexp"
	"sort"
	"strings"

	"github.com/kkkqkx123/mihomo-cli/pkg/types"
)

// FilterOptions 过滤选项
type FilterOptions struct {
	Type           string // 按类型过滤
	Status         string // 按状态过滤
	ExcludeRegex   string // 排除名称匹配正则
	ExcludeLogical bool   // 排除逻辑节点
	GroupsOnly     bool   // 只显示代理组
	NodesOnly      bool   // 只显示节点
	SortBy         string // 排序方式（name/delay）
}

// FilterProxies 根据过滤条件过滤代理列表
func FilterProxies(proxies map[string]*types.ProxyInfo, opts FilterOptions) map[string]*types.ProxyInfo {
	result := make(map[string]*types.ProxyInfo)

	// 编译正则表达式（如果需要）
	var excludeRegex *regexp.Regexp
	var err error
	if opts.ExcludeRegex != "" {
		excludeRegex, err = regexp.Compile(opts.ExcludeRegex)
		if err != nil {
			// 正则表达式无效，返回原列表
			return proxies
		}
	}

	for name, proxy := range proxies {
		// 检查是否应该包含此代理
		if shouldIncludeProxy(name, proxy, opts, excludeRegex) {
			result[name] = proxy
		}
	}

	return result
}

// shouldIncludeProxy 判断是否应该包含此代理
func shouldIncludeProxy(name string, proxy *types.ProxyInfo, opts FilterOptions, excludeRegex *regexp.Regexp) bool {
	// 1. 按类型过滤
	if opts.Type != "" {
		if !strings.EqualFold(proxy.Type, opts.Type) {
			return false
		}
	}

	// 2. 按状态过滤
	if opts.Status != "" {
		switch strings.ToLower(opts.Status) {
		case "alive":
			if !proxy.Alive {
				return false
			}
		case "dead":
			if proxy.Alive {
				return false
			}
		}
	}

	// 3. 排除逻辑节点
	if opts.ExcludeLogical {
		if logicalTypes[proxy.Type] {
			return false
		}
	}

	// 4. 排除名称匹配正则
	if excludeRegex != nil {
		if excludeRegex.MatchString(name) {
			return false
		}
	}

	// 5. 只显示代理组
	if opts.GroupsOnly {
		if len(proxy.All) == 0 {
			return false
		}
	}

	// 6. 只显示节点
	if opts.NodesOnly {
		if len(proxy.All) > 0 {
			return false
		}
	}

	return true
}

// SortProxies 对代理进行排序
func SortProxies(proxies map[string]*types.ProxyInfo, sortBy string) []string {
	names := make([]string, 0, len(proxies))
	for name := range proxies {
		names = append(names, name)
	}

	switch sortBy {
	case "name":
		sort.Strings(names)
	case "delay":
		sort.Slice(names, func(i, j int) bool {
			delayI := proxies[names[i]].Delay
			delayJ := proxies[names[j]].Delay
			// 延迟为 0 的排在最后
			if delayI == 0 && delayJ == 0 {
				return names[i] < names[j]
			}
			if delayI == 0 {
				return false
			}
			if delayJ == 0 {
				return true
			}
			return delayI < delayJ
		})
	default:
		sort.Strings(names)
	}

	return names
}