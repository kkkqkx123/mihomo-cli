# proxy test 延迟测试结果展示改造设计

## 一、现状分析

### 1.1 当前调用链

```
cmd/proxy.go (proxy test / proxy auto / proxy list --test-delay)
  ↓
internal/proxy/tester.go TestSingle
  ↓
internal/api/proxy.go TestDelay → GET /proxies/{name}/delay
  ↓
internal/proxy/formatter.go FormatTestResults
```

### 1.2 当前展示字段

`internal/proxy/formatter.go:229` 表格输出 4 列：

| 列名 | 来源 | 当前表现 |
|------|------|----------|
| 节点名称 | `result.Name` | 正常显示 |
| 延迟 | `result.Delay` | 成功为 `xxms`；失败/未知显示 `-` |
| 耗时 | `result.Time` | 显示 `xxms`（含超时等待时间） |
| 状态 | `result.Status` | 成功：`优秀/良好/较差`；失败：固定为 `超时`（红色）；未知：`未知`（黄色） |

### 1.3 当前错误映射

`internal/proxy/tester.go:66-80`：

```go
if err != nil {
    result.Error = err
    result.Status = "超时"          // 任何 err 都是“超时”
} else if delay == 0 {
    result.Status = "未知"
} else {
    result.Status = "优秀/良好/较差" // 基于 delay 阈值
}
```

配合 `internal/api/proxy.go:91-93`：

```go
if err != nil {
    return 0, NewAPIError(ErrTimeout, "测试延迟失败", err) // 旧实现统一为 ErrTimeout
    // 当前代码已改为 ErrAPIError，但仍把原始错误包在 Cause 里
}
```

### 1.4 内核真实返回

`mihomo-core/hub/route/proxies.go:106-148`：

- `timeout` 参数缺失/非法 → HTTP 400
- `expected` 参数非法 → HTTP 400
- 测试超时 → HTTP 504（Gateway Timeout）
- 节点测试失败或 `delay == 0` → HTTP 503（Service Unavailable）
- 成功 → HTTP 200，返回 `{"delay": <ms>}`

## 二、存在的问题

| 问题 | 说明 | 影响 |
|------|------|------|
| 状态字段语义单一 | 所有失败都显示“超时” | 用户无法区分节点不可用、参数错误、连接失败 |
| 缺少错误详情 | 表格只有 4 列，没有地方展示原始错误信息 | 排查困难 |
| 状态与 HTTP 码不对应 | 400/503/504 都被归为“超时” | 误导用户判断 |
| 耗时列含义模糊 | 失败时耗时接近 timeout，但用户可能误以为是实际测速时间 | 体验不佳 |
| 缺少汇总 | 大量节点测试后没有成功/失败统计 | 需要用户自己数 |
| 当前 TestDelay 错误包装 | 使用 `NewAPIError(ErrAPIError, ..., err)` 后再包一层 | 丢失了原始 HTTP 状态码，导致上层无法精确分类 |

## 三、改造目标

1. **精确状态**：根据内核返回的真实原因显示 `超时 / 节点不可用 / 参数错误 / 连接失败 / 测试失败`。
2. **展示详情**：在表格或 JSON 中保留可读的失败原因。
3. **保留性能**：不增加额外 API 调用，只改变展示层和错误分类逻辑。
4. **向后兼容**：JSON 输出保留现有字段，新增字段使用 `omitempty`。
5. **可测试**：为错误分类和格式化输出补充单元测试。

## 四、设计草案

### 4.1 错误分类

在 `internal/proxy/tester.go` 的 `TestSingle` 中引入分类：

| 内核返回 | 检测方式 | 状态 | 详情示例 |
|----------|----------|------|----------|
| HTTP 200 + delay > 0 | `err == nil && delay > 0` | `优秀/良好/较差` | - |
| HTTP 200 + delay == 0 | `err == nil && delay == 0` | `未知` | - |
| HTTP 504 | `api.IsTimeoutError(err)` | `超时` | 请求超时 |
| HTTP 400 | HTTP 状态码 400 | `参数错误` | 无效的超时参数 |
| HTTP 503 | HTTP 状态码 503 | `节点不可用` | 节点测试失败 |
| 连接被拒绝/断连 | `api.IsAPIConnectionError(err)` | `连接失败` | Mihomo 未运行 |
| 其他 | 兜底 | `测试失败` | <原始消息> |

### 4.2 API 层调整

`internal/api/proxy.go TestDelay` 不再强制套 `NewAPIError`：

```go
var result types.DelayResponse
err := c.Get(ctx, "/proxies/"+encodedName+"/delay", queryParams, &result)
if err != nil {
    // 直接透传原始 API 错误，保留 HTTP 状态码
    return 0, err
}
return result.Delay, nil
```

这样上层可以拿到原始 `*api.APIError`，通过 `apiErr.StatusCode` 精确分类。

### 4.3 数据结构扩展

`pkg/types/proxy.go` 的 `DelayResult`：

```go
type DelayResult struct {
    Name    string
    Delay   uint16
    Error   error
    Status  string
    Time    int64
    Detail  string `json:"detail,omitempty"` // 失败原因简短描述
}
```

### 4.4 表格展示

`internal/proxy/formatter.go formatTestResultsTable` 改为 5 列：

| 节点名称 | 延迟 | 耗时 | 状态 | 详情 |
|----------|------|------|------|------|
| Node-A | 45ms | 120ms | 优秀 | - |
| Node-B | - | 5000ms | 超时 | 请求超时 |
| Node-C | - | 20ms | 参数错误 | 无效的超时参数 |
| Node-D | - | 15ms | 节点不可用 | 节点测试失败 |

- 成功时“详情”列留空（不显示 `-`，保持简洁）。
- 失败时“状态”按分类着色：参数错误黄色、节点不可用/超时/连接失败红色。

### 4.5 JSON 输出

```json
[
  {
    "Name": "Node-A",
    "Delay": 45,
    "Status": "优秀",
    "Time": 120
  },
  {
    "Name": "Node-B",
    "Delay": 0,
    "Status": "超时",
    "Time": 5000,
    "detail": "请求超时"
  }
]
```

### 4.6 测试覆盖

- `TestTestSingle_ClassifyErrors`：mock 不同 HTTP 状态码，验证 `Status` 和 `Detail`。
- `TestFormatTestResultsTable_WithDetails`：验证“详情”列输出。
- 现有成功/失败用例保持兼容。

## 五、待拍板开放点

1. **是否保留“耗时”列？**
   - 方案 A：保留，失败时显示总耗时。
   - 方案 B：失败时改为显示内核返回的 HTTP 状态码（如 `504`）。
   - 推荐方案 A，因为耗时有助于判断是真的超时还是快速失败。

2. **状态文案：**
   - 失败分类是否使用更中性的 `失败` + 详情列？
   - 推荐保留分类状态，更直观。

3. **是否需要测试汇总？**
   - 推荐在表格后追加一行：`测试完成：成功 8 / 失败 2 / 总计 10`。

4. **是否同时改造 `proxy list --test-delay` 的展示？**
   - 当前 `proxy list` 只展示节点列表中的延迟数字，不展示测试详情。
   - 推荐在 `proxy list --test-delay` 失败时，在节点后增加失败标记，不展开完整表格。

## 六、实施顺序（下一阶段）

1. 修改 `internal/api/proxy.go TestDelay` 透传原始错误。
2. 扩展 `pkg/types/proxy.go DelayResult` 增加 `Detail` 字段。
3. 修改 `internal/proxy/tester.go TestSingle` 实现错误分类。
4. 修改 `internal/proxy/formatter.go` 增加详情列和汇总。
5. 补充 `internal/proxy/tester_test.go` 和 `formatter_test.go` 单元测试。
6. 运行全量测试和构建验证。
7. 生成最终 patch。

## 七、实施记录

- 已按本设计完成改造，`TestSingle` 在分类基础上通过 `Detail` 字段保留了原始 API 错误消息。
- 关键文件变更：
  - `internal/api/proxy.go`：透传原始错误。
  - `pkg/types/proxy.go`：`DelayResult` 增加 `Detail` 字段。
  - `internal/proxy/tester.go`：新增 `classifyTestError`，实现 `超时/连接失败/参数错误/节点不可用/测试失败` 分类。
  - `internal/proxy/formatter.go`：表格增加“详情”列和测试汇总行。
  - `internal/proxy/tester_test.go` / `formatter_test.go`：补充分类和详情展示单元测试。
- `go build ./...` 与 `go test ./...` 均通过。
