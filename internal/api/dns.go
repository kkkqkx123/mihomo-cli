package api

import (
	"context"
	"fmt"
	"strings"

	"github.com/kkkqkx123/mihomo-cli/pkg/types"
)

// QueryDNS 执行 DNS 查询
func (c *Client) QueryDNS(ctx context.Context, domain string, recordType string) (*types.DNSQueryResponse, error) {
	// 验证域名
	if domain == "" {
		return nil, NewAPIError(ErrInvalidArgs, "域名不能为空", nil)
	}

	// 验证记录类型
	if recordType == "" {
		recordType = "A" // 默认查询 A 记录
	}

	// 转换为大写
	recordType = strings.ToUpper(recordType)

	// 检查是否为支持的类型
	if _, ok := types.DNSType[recordType]; !ok {
		return nil, NewAPIError(ErrInvalidArgs, fmt.Sprintf("不支持的 DNS 记录类型: %s", recordType), nil)
	}

	// 准备查询参数
	queryParams := map[string]string{
		"name": domain,
		"type": recordType,
	}

	var result types.DNSQueryResponse
	err := c.Get(ctx, "/dns/query", queryParams, &result)
	if err != nil {
		return nil, NewAPIError(ErrAPIError, "DNS 查询失败", err)
	}

	return &result, nil
}
