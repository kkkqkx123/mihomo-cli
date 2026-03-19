package log

import (
	"regexp"
	"strings"

	"github.com/kkkqkx123/mihomo-cli/pkg/types"
)

// LogTypePriority 日志级别优先级
var LogTypePriority = map[string]int{
	"silent":  0,
	"error":   1,
	"warning": 2,
	"warn":    2,
	"info":    3,
	"debug":   4,
}

// LogFilter 日志过滤器
type LogFilter struct {
	Level     string   // 日志级别（silent/error/warning/info/debug）
	Keywords  []string // 包含的关键词（AND 逻辑）
	Exclude   []string // 排除的关键词
	Regex     *regexp.Regexp // 正则表达式
}

// NewLogFilter 创建新的日志过滤器
func NewLogFilter() *LogFilter {
	return &LogFilter{}
}

// WithLevel 设置日志级别过滤
func (lf *LogFilter) WithLevel(level string) *LogFilter {
	lf.Level = strings.ToLower(level)
	return lf
}

// WithKeywords 设置关键词过滤（AND 逻辑）
func (lf *LogFilter) WithKeywords(keywords ...string) *LogFilter {
	lf.Keywords = keywords
	return lf
}

// WithExclude 设置排除关键词
func (lf *LogFilter) WithExclude(exclude ...string) *LogFilter {
	lf.Exclude = exclude
	return lf
}

// WithRegex 设置正则表达式过滤
func (lf *LogFilter) WithRegex(pattern string) error {
	re, err := regexp.Compile(pattern)
	if err != nil {
		return err
	}
	lf.Regex = re
	return nil
}

// Match 判断单条日志是否匹配过滤条件
func (lf *LogFilter) Match(log *types.LogInfo) bool {
	// 级别过滤
	if lf.Level != "" {
		logLevel := strings.ToLower(log.LogType)
		filterLevel := strings.ToLower(lf.Level)

		// 检查优先级，只有当日志级别大于等于过滤级别时才匹配
		if LogTypePriority[logLevel] < LogTypePriority[filterLevel] {
			return false
		}
	}

	// 关键词过滤（AND 逻辑）
	if len(lf.Keywords) > 0 {
		payload := strings.ToLower(log.Payload)
		for _, keyword := range lf.Keywords {
			if !strings.Contains(payload, strings.ToLower(keyword)) {
				return false
			}
		}
	}

	// 排除关键词
	if len(lf.Exclude) > 0 {
		payload := strings.ToLower(log.Payload)
		for _, keyword := range lf.Exclude {
			if strings.Contains(payload, strings.ToLower(keyword)) {
				return false
			}
		}
	}

	// 正则表达式过滤
	if lf.Regex != nil {
		if !lf.Regex.MatchString(log.Payload) {
			return false
		}
	}

	return true
}

// FilterLogs 批量过滤日志
func FilterLogs(logs []*types.LogInfo, filter *LogFilter) []*types.LogInfo {
	if filter == nil {
		return logs
	}

	filtered := make([]*types.LogInfo, 0, len(logs))
	for _, log := range logs {
		if filter.Match(log) {
			filtered = append(filtered, log)
		}
	}

	return filtered
}

// FilterLogsByLevel 按级别过滤日志
func FilterLogsByLevel(logs []*types.LogInfo, level string) []*types.LogInfo {
	filter := NewLogFilter().WithLevel(level)
	return FilterLogs(logs, filter)
}

// FilterLogsByKeywords 按关键词过滤日志（AND 逻辑）
func FilterLogsByKeywords(logs []*types.LogInfo, keywords ...string) []*types.LogInfo {
	filter := NewLogFilter().WithKeywords(keywords...)
	return FilterLogs(logs, filter)
}

// FilterLogsExclude 排除包含特定关键词的日志
func FilterLogsExclude(logs []*types.LogInfo, exclude ...string) []*types.LogInfo {
	filter := NewLogFilter().WithExclude(exclude...)
	return FilterLogs(logs, filter)
}