package log

import (
	"sort"
	"strings"

	"github.com/kkkqkx123/mihomo-cli/pkg/types"
)

// LogStatistics 日志统计信息
type LogStatistics struct {
	TotalCount int                        `json:"total_count"`
	LevelCount map[string]int             `json:"level_count"`
	ErrorRate  float64                    `json:"error_rate"`
	TopErrors  []ErrorFrequency           `json:"top_errors"`
}

// ErrorFrequency 错误频率统计
type ErrorFrequency struct {
	Message string `json:"message"`
	Count   int    `json:"count"`
}

// CalculateStatistics 计算日志统计信息
func CalculateStatistics(logs []*types.LogInfo) *LogStatistics {
	stats := &LogStatistics{
		TotalCount: len(logs),
		LevelCount: make(map[string]int),
		TopErrors:  make([]ErrorFrequency, 0),
	}

	if stats.TotalCount == 0 {
		return stats
	}

	// 统计各级别日志数量
	errorCount := 0
	errorMessages := make(map[string]int)

	for _, log := range logs {
		level := strings.ToLower(log.LogType)
		stats.LevelCount[level]++

		// 统计错误日志
		if level == "error" {
			errorCount++
			// 统计错误消息频率
			normalizedMsg := strings.TrimSpace(log.Payload)
			errorMessages[normalizedMsg]++
		}
	}

	// 计算错误率
	stats.ErrorRate = float64(errorCount) / float64(stats.TotalCount) * 100

	// 统计 Top 10 错误
	if len(errorMessages) > 0 {
		// 转换为切片并排序
		errors := make([]ErrorFrequency, 0, len(errorMessages))
		for msg, count := range errorMessages {
			errors = append(errors, ErrorFrequency{
				Message: msg,
				Count:   count,
			})
		}

		// 按出现次数降序排序
		sort.Slice(errors, func(i, j int) bool {
			return errors[i].Count > errors[j].Count
		})

		// 取 Top 10
		if len(errors) > 10 {
			errors = errors[:10]
		}
		stats.TopErrors = errors
	}

	return stats
}

// GetLevelCountMap 获取标准化的级别统计映射
func (ls *LogStatistics) GetLevelCountMap() map[string]int {
	// 确保所有级别都有值
	result := make(map[string]int)
	for _, level := range []string{"silent", "error", "warning", "info", "debug"} {
		result[level] = ls.LevelCount[level]
	}
	return result
}