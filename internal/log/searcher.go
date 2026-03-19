package log

import (
	"regexp"
	"strings"

	"github.com/kkkqkx123/mihomo-cli/pkg/types"
)

// SearchQuery 搜索查询条件
type SearchQuery struct {
	Keywords []string         // 关键词（AND 逻辑）
	Regex    *regexp.Regexp   // 正则表达式
	Level    string           // 日志级别过滤
}

// SearchResult 搜索结果
type SearchResult struct {
	Total   int               `json:"total"`
	Matches []*types.LogInfo  `json:"matches"`
}

// NewSearchQuery 创建新的搜索查询
func NewSearchQuery() *SearchQuery {
	return &SearchQuery{}
}

// WithKeywords 设置关键词搜索（AND 逻辑）
func (sq *SearchQuery) WithKeywords(keywords ...string) *SearchQuery {
	sq.Keywords = keywords
	return sq
}

// WithRegex 设置正则表达式搜索
func (sq *SearchQuery) WithRegex(pattern string) (*SearchQuery, error) {
	re, err := regexp.Compile(pattern)
	if err != nil {
		return nil, err
	}
	sq.Regex = re
	return sq, nil
}

// WithLevel 设置级别过滤
func (sq *SearchQuery) WithLevel(level string) *SearchQuery {
	sq.Level = level
	return sq
}

// LogSearcher 日志搜索器
type LogSearcher struct{}

// NewLogSearcher 创建新的日志搜索器
func NewLogSearcher() *LogSearcher {
	return &LogSearcher{}
}

// Search 执行搜索
func (ls *LogSearcher) Search(logs []*types.LogInfo, query *SearchQuery) *SearchResult {
	result := &SearchResult{
		Matches: make([]*types.LogInfo, 0),
	}

	// 先应用级别过滤
	filteredLogs := logs
	if query.Level != "" {
		filter := NewLogFilter().WithLevel(query.Level)
		filteredLogs = FilterLogs(logs, filter)
	}

	// 搜索匹配的日志
	for _, log := range filteredLogs {
		if ls.matchLog(log, query) {
			result.Matches = append(result.Matches, log)
		}
	}

	result.Total = len(result.Matches)
	return result
}

// matchLog 判断日志是否匹配搜索条件
func (ls *LogSearcher) matchLog(log *types.LogInfo, query *SearchQuery) bool {
	payload := strings.ToLower(log.Payload)

	// 关键词搜索（AND 逻辑）
	if len(query.Keywords) > 0 {
		for _, keyword := range query.Keywords {
			if !strings.Contains(payload, strings.ToLower(keyword)) {
				return false
			}
		}
	}

	// 正则表达式搜索
	if query.Regex != nil {
		if !query.Regex.MatchString(log.Payload) {
			return false
		}
	}

	return true
}

// SearchKeywords 按关键词搜索日志（便捷方法）
func SearchKeywords(logs []*types.LogInfo, keywords ...string) []*types.LogInfo {
	query := NewSearchQuery().WithKeywords(keywords...)
	searcher := NewLogSearcher()
	result := searcher.Search(logs, query)
	return result.Matches
}

// SearchRegex 按正则表达式搜索日志（便捷方法）
func SearchRegex(logs []*types.LogInfo, pattern string) ([]*types.LogInfo, error) {
	query, err := NewSearchQuery().WithRegex(pattern)
	if err != nil {
		return nil, err
	}
	searcher := NewLogSearcher()
	result := searcher.Search(logs, query)
	return result.Matches, nil
}