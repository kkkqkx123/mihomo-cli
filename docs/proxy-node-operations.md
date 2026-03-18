# 代理节点操作机制分析

本文档详细分析了 mihomo-cli 项目中代理节点操作的实现机制。

## 1. 整体架构

项目采用 **CLI 命令行工具** 架构，通过调用 **Mihomo REST API** 来管理代理节点。核心流程如下：

```
CLI命令 → API客户端 → HTTP请求 → Mihomo REST API → 代理节点操作
```

## 2. 核心组件

### 2.1 命令层

**文件位置**: `cmd/proxy.go`

提供以下子命令：

| 命令 | 功能 | 示例 |
|------|------|------|
| `proxy list` | 列出代理节点 | `mihomo-cli proxy list [group]` |
| `proxy switch` | 切换代理节点 | `mihomo-cli proxy switch <group> <node>` |
| `proxy test` | 测试节点延迟 | `mihomo-cli proxy test <group> [node]` |
| `proxy auto` | 自动选择最快节点 | `mihomo-cli proxy auto <group>` |
| `proxy unfix` | 取消固定代理 | `mihomo-cli proxy unfix <group>` |

### 2.2 API客户端层

**文件位置**: `internal/api/`

核心文件说明：

| 文件 | 功能 |
|------|------|
| `client.go` | API客户端封装，提供统一的客户端接口 |
| `http.go` | HTTP请求封装，处理认证、超时、错误等 |
| `proxy.go` | 代理相关API操作的具体实现 |

### 2.3 业务逻辑层

**文件位置**: `internal/proxy/`

核心组件：

| 组件 | 文件 | 功能 |
|------|------|------|
| DelayTester | `tester.go` | 延迟测试器，支持并发测速 |
| Selector | `selector.go` | 节点选择器，实现智能选择逻辑 |

## 3. API接口详解

### 3.1 切换代理节点

**方法**: `SwitchProxy`

**API端点**: `PUT /proxies/{group}`

**请求体**:
```json
{
  "name": "节点名称"
}
```

**代码实现** (`internal/api/proxy.go:36-50`):
```go
func (c *Client) SwitchProxy(ctx context.Context, group, proxy string) error {
    encodedGroup := url.PathEscape(group)
    request := types.SwitchRequest{
        Name: proxy,
    }
    err := c.Put(ctx, "/proxies/"+encodedGroup, nil, &request, nil)
    if err != nil {
        return fmt.Errorf("切换代理失败: %w", err)
    }
    return nil
}
```

### 3.2 测试节点延迟

**方法**: `TestDelay`

**API端点**: `GET /proxies/{name}/delay`

**查询参数**:
- `url`: 测试URL（可选）
- `timeout`: 超时时间（毫秒）

**代码实现** (`internal/api/proxy.go:53-72`):
```go
func (c *Client) TestDelay(ctx context.Context, name string, testURL string, timeout int) (uint16, error) {
    encodedName := url.PathEscape(name)
    queryParams := make(map[string]string)
    if testURL != "" {
        queryParams["url"] = testURL
    }
    if timeout > 0 {
        queryParams["timeout"] = strconv.Itoa(timeout)
    }
    var result types.DelayResponse
    err := c.Get(ctx, "/proxies/"+encodedName+"/delay", queryParams, &result)
    if err != nil {
        return 0, fmt.Errorf("测试延迟失败: %w", err)
    }
    return result.Delay, nil
}
```

### 3.3 获取代理信息

**方法**: `GetProxy`

**API端点**: `GET /proxies/{name}`

**代码实现** (`internal/api/proxy.go:23-33`):
```go
func (c *Client) GetProxy(ctx context.Context, name string) (*types.ProxyInfo, error) {
    encodedName := url.PathEscape(name)
    var result types.ProxyInfo
    err := c.Get(ctx, "/proxies/"+encodedName, nil, &result)
    if err != nil {
        return nil, fmt.Errorf("获取代理 %s 失败: %w", name, err)
    }
    return &result, nil
}
```

### 3.4 列出所有代理

**方法**: `ListProxies`

**API端点**: `GET /proxies`

**代码实现** (`internal/api/proxy.go:13-20`):
```go
func (c *Client) ListProxies(ctx context.Context) (map[string]*types.ProxyInfo, error) {
    var result types.ProxiesResponse
    err := c.Get(ctx, "/proxies", nil, &result)
    if err != nil {
        return nil, fmt.Errorf("获取代理列表失败: %w", err)
    }
    return result.Proxies, nil
}
```

### 3.5 取消固定代理

**方法**: `UnfixProxy`

**API端点**: `DELETE /proxies/{group}`

**代码实现** (`internal/api/proxy.go:75-85`):
```go
func (c *Client) UnfixProxy(ctx context.Context, group string) error {
    encodedGroup := url.PathEscape(group)
    err := c.Delete(ctx, "/proxies/"+encodedGroup, nil, nil)
    if err != nil {
        return fmt.Errorf("取消固定代理失败: %w", err)
    }
    return nil
}
```

## 4. 业务逻辑实现

### 4.1 延迟测试器 (DelayTester)

**文件位置**: `internal/proxy/tester.go`

**核心功能**:

1. **单节点测试** (`TestSingle`):
   - 测试单个代理节点的延迟
   - 根据延迟值判断状态（优秀/良好/较差/超时）

2. **批量测试** (`TestGroup`):
   - 获取代理组中所有节点
   - 并发测试所有节点延迟
   - 支持进度回调

3. **并发控制**:
   - 使用信号量机制控制并发数
   - 默认并发数为10
   - 可通过 `SetConcurrent()` 方法配置

**关键代码** (`internal/proxy/tester.go:103-134`):
```go
func (t *DelayTester) TestNodes(ctx context.Context, nodeNames []string) ([]types.DelayResult, error) {
    results := make([]types.DelayResult, len(nodeNames))
    var wg sync.WaitGroup
    sem := make(chan struct{}, t.concurrent) // 信号量控制并发

    for i, nodeName := range nodeNames {
        wg.Add(1)
        go func(index int, name string) {
            defer wg.Done()
            sem <- struct{}{}           // 获取信号量
            defer func() { <-sem }()    // 释放信号量
            results[index] = t.TestSingle(ctx, name)
            if t.progress != nil {
                t.progress(index+1, len(nodeNames), name)
            }
        }(i, nodeName)
    }

    wg.Wait()
    return results, nil
}
```

### 4.2 节点选择器 (Selector)

**文件位置**: `internal/proxy/selector.go`

**核心功能**:

1. **选择最佳节点** (`SelectBestNode`):
   - 测试所有节点延迟
   - 筛选有效节点
   - 按延迟排序，返回最低延迟节点

2. **选择并切换** (`SelectAndSwitch`):
   - 调用 `SelectBestNode` 选择最佳节点
   - 调用 `SwitchProxy` 切换到该节点

3. **按数量选择** (`SelectBestNodesByCount`):
   - 返回前N个延迟最低的节点

4. **按阈值选择** (`SelectByThreshold`):
   - 返回延迟低于指定阈值的所有节点

**关键代码** (`internal/proxy/selector.go:32-63`):
```go
func (s *Selector) SelectBestNode(ctx context.Context, groupName string) (string, uint16, error) {
    // 测试所有节点延迟
    results, err := s.tester.TestGroup(ctx, groupName)
    if err != nil {
        return "", 0, pkgerrors.ErrAPI("failed to test node delay", err)
    }

    // 筛选有效节点
    var validResults []DelayResultInfo
    for _, result := range results {
        if result.Error == nil && result.Delay > 0 {
            validResults = append(validResults, DelayResultInfo{
                Name:  result.Name,
                Delay: result.Delay,
            })
        }
    }

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
```

## 5. 数据结构

### 5.1 ProxyInfo

**文件位置**: `pkg/types/proxy.go`

```go
type ProxyInfo struct {
    Name     string         `json:"name"`           // 代理名称
    Type     string         `json:"type"`           // 代理类型 (ss, vmess, trojan等)
    UDP      bool           `json:"udp"`            // 是否支持UDP
    XUDP     bool           `json:"xudp"`           // 是否支持XUDP
    History  []DelayHistory `json:"history"`        // 延迟历史记录
    Alive    bool           `json:"alive"`          // 是否存活
    Now      string         `json:"now,omitempty"`  // 当前选中的节点（仅代理组）
    All      []string       `json:"all,omitempty"`  // 所有可用节点列表（仅代理组）
    Provider string         `json:"provider,omitempty"` // 提供者名称
    Delay    uint16         `json:"delay"`          // 当前延迟
}
```

### 5.2 DelayResult

```go
type DelayResult struct {
    Name   string  // 节点名称
    Delay  uint16  // 延迟值（毫秒）
    Error  error   // 错误信息
    Status string  // 状态描述：优秀/良好/较差/超时/未知
    Time   int64   // 测速耗时（毫秒）
}
```

### 5.3 SwitchRequest

```go
type SwitchRequest struct {
    Name string `json:"name"` // 要切换到的节点名称
}
```

## 6. HTTP通信机制

### 6.1 认证方式

使用 Bearer Token 认证：

```go
func (c *HTTPClient) addAuthHeader(req *http.Request) {
    if c.secret != "" {
        req.Header.Set("Authorization", "Bearer "+c.secret)
    }
}
```

### 6.2 请求构建

**文件位置**: `internal/api/http.go`

核心方法 `buildURL` 负责构建完整的API URL：

```go
func (c *HTTPClient) buildURL(baseURL, endpoint string, queryParams map[string]string) (string, error) {
    u, err := url.Parse(baseURL)
    if err != nil {
        return "", NewConnectionError(err)
    }
    u.Path = path.Join(u.Path, endpoint)
    if len(queryParams) > 0 {
        q := u.Query()
        for key, value := range queryParams {
            q.Set(key, value)
        }
        u.RawQuery = q.Encode()
    }
    return u.String(), nil
}
```

### 6.3 错误处理

- **超时错误**: 检测 `context.DeadlineExceeded` 或 `http: Client.Timeout exceeded`
- **连接错误**: 网络连接失败
- **API错误**: 服务端返回非2xx状态码

## 7. 完整操作流程示例

### 7.1 切换代理节点流程

```
用户执行: mihomo-cli proxy switch Proxy Node1
    ↓
命令解析: groupName="Proxy", nodeName="Node1"
    ↓
创建API客户端: api.NewClientWithTimeout(address, secret, timeout)
    ↓
调用切换API: client.SwitchProxy(ctx, "Proxy", "Node1")
    ↓
发送HTTP请求: PUT {baseURL}/proxies/Proxy
    - Headers: Authorization: Bearer {secret}
    - Body: {"name": "Node1"}
    ↓
Mihomo处理请求并切换节点
    ↓
返回结果并格式化输出
```

### 7.2 自动选择最快节点流程

```
用户执行: mihomo-cli proxy auto Proxy
    ↓
获取代理组信息: client.GetProxy(ctx, "Proxy")
    ↓
创建延迟测试器: proxy.NewDelayTester(client)
    ↓
并发测试所有节点延迟: tester.TestGroup(ctx, "Proxy")
    - 使用信号量控制并发数
    - 每个节点独立测试
    - 支持进度回调
    ↓
筛选有效节点并按延迟排序
    ↓
选择延迟最低的节点
    ↓
切换到最佳节点: client.SwitchProxy(ctx, "Proxy", bestNode)
    ↓
输出结果
```

## 8. 关键设计特点

### 8.1 分层架构

```
命令层 (cmd/)
    ↓ 调用
API层 (internal/api/)
    ↓ 调用
HTTP层 (internal/api/http.go)
    ↓ 请求
Mihomo REST API
```

职责清晰，便于维护和扩展。

### 8.2 并发测试

- 使用 Go 的 goroutine 实现并发测试
- 使用 channel 作为信号量控制并发数
- 使用 sync.WaitGroup 等待所有测试完成
- 使用 sync.Mutex 保护共享数据

### 8.3 可配置性

支持以下配置项：
- 测试URL（`--url`）
- 超时时间（`--timeout`）
- 并发数（`--concurrent`）
- 进度显示（`--progress`）

### 8.4 进度反馈

支持进度条显示测速进度，使用 `progressbar` 库实现：

```go
bar := progressbar.NewOptions(nodeCount,
    progressbar.OptionSetDescription("测速中"),
    progressbar.OptionShowCount(),
    progressbar.OptionShowIts(),
    progressbar.OptionClearOnFinish(),
)
tester.SetProgress(func(current, total int, nodeName string) {
    bar.Set(current)
})
```

### 8.5 错误处理

统一的错误包装机制：
- `WrapAPIError`: 包装API错误
- `NewConnectionError`: 连接错误
- `NewTimeoutError`: 超时错误
- `NewAPIError`: 通用API错误

## 9. 扩展建议

### 9.1 支持更多选择策略

当前支持：
- 最低延迟选择
- 按数量选择
- 按阈值选择

可扩展：
- 负载均衡选择
- 地理位置选择
- 自定义权重选择

### 9.2 增强测试功能

当前功能：
- 单节点测试
- 批量测试
- 并发测试

可增强：
- 持续监控模式
- 历史数据分析
- 异常告警机制

### 9.3 优化用户体验

当前支持：
- 进度条显示
- 多种输出格式（table/json）

可优化：
- 交互式选择界面
- 节点收藏功能
- 快捷切换历史节点
