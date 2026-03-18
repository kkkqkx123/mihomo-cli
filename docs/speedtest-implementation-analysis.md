# 测速功能实现分析与改进方案

## 一、当前实现的功能

### 1.1 已实现的核心功能

#### 延迟测试器 (`internal/proxy/tester.go`)

**功能清单：**
- ✅ `TestSingle` - 测试单个节点延迟
- ✅ `TestGroup` - 测试代理组中所有节点延迟
- ✅ `TestNodes` - 测试多个节点延迟（支持并发）
- ✅ `TestAll` - 测试所有代理组延迟

**并发控制：**
- 使用 goroutine + channel 实现并发控制
- 默认并发数：10
- 支持自定义并发数

**配置参数：**
- `testURL` - 测试目标 URL（可选）
- `timeout` - 超时时间（毫秒）
- `concurrent` - 并发数

#### API 客户端 (`internal/api/proxy.go`)

**功能清单：**
- ✅ `ListProxies` - 获取所有代理信息
- ✅ `GetProxy` - 获取指定代理详细信息
- ✅ `SwitchProxy` - 切换代理组中选中的代理
- ✅ `TestDelay` - 测试指定代理的延迟
- ✅ `UnfixProxy` - 取消代理组中固定的代理

**API 接口调用：**
- `GET /proxies` - 获取代理列表
- `GET /proxies/{name}` - 获取代理详情
- `PUT /proxies/{group}` - 切换代理
- `GET /proxies/{name}/delay` - 测试延迟
- `DELETE /proxies/{group}` - 取消固定代理

#### 节点选择器 (`internal/proxy/selector.go`)

**功能清单：**
- ✅ `SelectBestNode` - 选择延迟最低的节点
- ✅ `SelectAndSwitch` - 选择并切换到最快节点
- ✅ `SelectBestNodesByCount` - 选择前 N 个延迟最低的节点
- ✅ `SelectByThreshold` - 选择延迟低于阈值的节点

**算法流程：**
1. 测试代理组中所有节点的延迟
2. 筛选测试成功的节点
3. 按延迟排序
4. 返回最佳节点

#### 输出格式化 (`internal/proxy/formatter.go`)

**功能清单：**
- ✅ `FormatProxyList` - 格式化代理列表（表格/JSON）
- ✅ `FormatTestResults` - 格式化测试结果（表格/JSON）
- ✅ `FormatAutoSelectResult` - 格式化自动选择结果
- ✅ `FormatSwitchResult` - 格式化切换代理结果

**状态分类：**
- 优秀：< 100ms（绿色）
- 良好：100-300ms（黄色）
- 较差：> 300ms（红色）
- 超时：连接失败（红色）

#### 命令行接口 (`cmd/proxy.go`)

**命令清单：**
- ✅ `proxy list [group]` - 列出代理节点
- ✅ `proxy switch <group> <node>` - 切换代理节点
- ✅ `proxy test <group> [node]` - 测试节点延迟
- ✅ `proxy auto <group>` - 自动选择最快节点
- ✅ `proxy unfix <group>` - 取消固定代理

**命令行参数：**
- `--url` - 自定义测试 URL
- `--timeout` - 超时时间（毫秒）
- `--concurrent` - 并发测试数
- `--output` - 输出格式（table/json）

---

## 二、功能对比分析

### 2.1 参考文档中的功能 (docs/更换节点与批量测速.txt)

| 功能 | PowerShell 实现 | Go CLI 实现 | 状态 |
|------|----------------|-------------|------|
| 查看节点列表 | `GET /proxies` | `proxy list` | ✅ 已实现 |
| 单个节点测速 | `GET /proxies/{name}/delay` | `proxy test <group> <node>` | ✅ 已实现 |
| 批量测速 | 循环调用测速接口 | `proxy test <group>` | ✅ 已实现 |
| 并发测速 | `ForEach-Object -Parallel` | `--concurrent` 参数 | ✅ 已实现 |
| 切换节点 | `PUT /proxies/{group}` | `proxy switch <group> <node>` | ✅ 已实现 |
| 自动选优 | 测速+比较+切换 | `proxy auto <group>` | ✅ 已实现 |
| 进度显示 | `Write-Progress` | ❌ 未实现 | ⚠️ 待实现 |
| 结果排序 | `Sort-Object` | 自动排序 | ✅ 已实现 |
| 实时状态反馈 | `Write-Host` | 表格输出 | ✅ 已实现 |

### 2.2 功能完整性评估

**基础功能：✅ 100% 完成**
- 测速功能完整实现
- 节点选择算法完善
- 命令行接口齐全

**高级功能：⚠️ 60% 完成**
- 缺少实时进度显示
- 缺少测速过程中的状态反馈
- 缺少详细的历史记录

**用户体验：⚠️ 70% 完成**
- 输出格式友好（表格/JSON）
- 缺少实时进度条
- 缺少颜色高亮（部分实现）

---

## 三、改进建议

### 3.1 优先级：高

#### 改进 1：实时进度显示

**当前问题：**
- 测速 30 个节点时，用户看不到进度
- 不知道测速是否在进行中
- 无法预估剩余时间

**实现方案：**

```go
// 在 DelayTester 中添加进度回调
type DelayTester struct {
    client    *api.Client
    testURL   string
    timeout   int
    concurrent int
    progress  ProgressCallback  // 新增
}

type ProgressCallback func(current, total int, nodeName string)

func (t *DelayTester) TestGroup(ctx context.Context, groupName string, progress ProgressCallback) ([]types.DelayResult, error) {
    // ...
    for i, nodeName := range nodeNames {
        wg.Add(1)
        go func(index int, name string) {
            defer wg.Done()
            sem <- struct{}{}
            defer func() { <-sem }()

            results[index] = t.TestSingle(ctx, name)

            // 调用进度回调
            if progress != nil {
                progress(index+1, len(nodeNames), name)
            }
        }(i, nodeName)
    }
    // ...
}
```

**命令行集成：**

```go
// 在 runProxyTest 中使用进度条
func runProxyTest(cmd *cobra.Command, args []string) error {
    // ...

    // 创建进度条
    bar := progressbar.NewOptions(
        len(proxy.All),
        progressbar.OptionSetDescription("测速中"),
        progressbar.OptionShowCount(),
        progressbar.OptionShowIts(),
    )

    // 使用进度回调
    tester := proxy.NewDelayTester(client)
    results, err := tester.TestGroup(ctx, groupName, func(current, total int, nodeName string) {
        bar.Set(current)
    })

    // ...
}
```

#### 改进 2：测速过程中的状态反馈

**当前问题：**
- 测速超时或失败时，无法及时反馈
- 用户不知道哪个节点测速失败

**实现方案：**

```go
// 扩展 DelayResult 类型
type DelayResult struct {
    Name   string
    Delay  uint16
    Error  error
    Status string  // 新增：状态描述
    Time   int64   // 新增：测速耗时（毫秒）
}

// 在 TestSingle 中记录时间和状态
func (t *DelayTester) TestSingle(ctx context.Context, proxyName string) types.DelayResult {
    start := time.Now()
    result := types.DelayResult{
        Name: proxyName,
    }

    delay, err := t.client.TestDelay(ctx, proxyName, t.testURL, t.timeout)
    result.Time = time.Since(start).Milliseconds()

    if err != nil {
        result.Error = err
        result.Status = "超时"
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
```

### 3.2 优先级：中

#### 改进 3：支持多种测速模式

**实现方案：**

```go
type TestMode string

const (
    ModeStandard TestMode = "standard"  // 标准模式
    ModeFast     TestMode = "fast"      // 快速模式（降低超时）
    ModeAccuracy TestMode = "accuracy"  // 精确模式（多次测速取平均）
)

type DelayTester struct {
    // ...
    mode TestMode
}

func (t *DelayTester) SetMode(mode TestMode) {
    t.mode = mode

    switch mode {
    case ModeFast:
        t.timeout = 3000  // 快速模式降低超时
    case ModeAccuracy:
        t.timeout = 10000 // 精确模式增加超时
    default:
        t.timeout = 5000  // 标准模式
    }
}
```

#### 改进 4：历史记录和趋势分析

**实现方案：**

```go
type DelayHistory struct {
    Time     time.Time
    Delay    uint16
    NodeName string
}

type HistoryStorage interface {
    Save(groupName string, results []types.DelayResult) error
    Load(groupName string, limit int) ([]DelayHistory, error)
    GetTrend(groupName, nodeName string, hours int) ([]DelayHistory, error)
}
```

### 3.3 优先级：低

#### 改进 5：支持自定义测速策略

**实现方案：**

```go
type SelectionStrategy string

const (
    StrategyFastest      SelectionStrategy = "fastest"       // 最快节点
    StrategyMostStable   SelectionStrategy = "most_stable"   // 最稳定节点
    StrategyLowestJitter SelectionStrategy = "lowest_jitter" // 最低抖动
    StrategyBalanced     SelectionStrategy = "balanced"      // 平衡模式
)

type Selector struct {
    // ...
    strategy SelectionStrategy
}

func (s *Selector) SelectBestNode(ctx context.Context, groupName string) (string, uint16, error) {
    switch s.strategy {
    case StrategyFastest:
        return s.selectFastestNode(ctx, groupName)
    case StrategyMostStable:
        return s.selectMostStableNode(ctx, groupName)
    case StrategyLowestJitter:
        return s.selectLowestJitterNode(ctx, groupName)
    case StrategyBalanced:
        return s.selectBalancedNode(ctx, groupName)
    }
}
```

---

## 四、与参考文档的差异对比

### 4.1 PowerShell 实现

**优势：**
- ✅ 实时进度显示 (`Write-Progress`)
- ✅ 灵活的脚本扩展性
- ✅ 易于调试和定制

**劣势：**
- ❌ 性能较差（单线程）
- ❌ 需要手动管理并发
- ❌ 需要额外安装 PowerShell 7+

### 4.2 Go CLI 实现

**优势：**
- ✅ 性能优秀（原生并发）
- ✅ 代码结构清晰
- ✅ 跨平台支持好
- ✅ 可编译为独立可执行文件

**劣势：**
- ⚠️ 缺少实时进度显示
- ⚠️ 扩展性相对较低
- ⚠️ 需要重新编译才能修改

### 4.3 功能对比总结

| 功能项 | PowerShell | Go CLI | 评价 |
|--------|-----------|--------|------|
| 性能 | ⭐⭐⭐ | ⭐⭐⭐⭐⭐ | Go CLI 优势明显 |
| 易用性 | ⭐⭐⭐⭐ | ⭐⭐⭐⭐ | 相当 |
| 扩展性 | ⭐⭐⭐⭐⭐ | ⭐⭐⭐ | PowerShell 优势 |
| 跨平台 | ⭐⭐⭐⭐ | ⭐⭐⭐⭐⭐ | Go CLI 优势 |
| 进度显示 | ⭐⭐⭐⭐⭐ | ⭐⭐ | PowerShell 优势 |
| 并发控制 | ⭐⭐ | ⭐⭐⭐⭐⭐ | Go CLI 优势 |
| 部署便捷性 | ⭐⭐⭐ | ⭐⭐⭐⭐⭐ | Go CLI 优势 |

---

## 五、实现建议

### 5.1 短期目标（1-2 周）

1. **添加实时进度显示**
   - 使用 `github.com/schollz/progressbar/v3` 库
   - 在 `proxy test` 和 `proxy auto` 命令中集成
   - 显示当前进度、预计剩余时间

2. **改进状态反馈**
   - 扩展 `DelayResult` 类型
   - 记录测速耗时
   - 显示更详细的状态描述

3. **优化输出格式**
   - 改进表格样式
   - 添加更多颜色高亮
   - 支持更详细的 JSON 输出

### 5.2 中期目标（1 个月）

1. **支持多种测速模式**
   - 快速模式、标准模式、精确模式
   - 在命令行中添加 `--mode` 参数

2. **历史记录功能**
   - 实现历史记录存储
   - 支持查看历史趋势
   - 支持历史数据导出

3. **智能选择策略**
   - 最快节点、最稳定节点、最低抖动、平衡模式
   - 在命令行中添加 `--strategy` 参数

### 5.3 长期目标（2-3 个月）

1. **高级分析功能**
   - 节点稳定性分析
   - 延迟趋势预测
   - 智能推荐节点

2. **配置文件支持**
   - 支持保存测速配置
   - 支持定时自动测速
   - 支持测速结果通知

3. **插件系统**
   - 支持自定义测速策略
   - 支持自定义输出格式
   - 支持第三方集成

---

## 六、技术实现细节

### 6.1 依赖库推荐

```go
// 进度条
import "github.com/schollz/progressbar/v3"

// 表格美化
import "github.com/olekukonko/tablewriter"

// 颜色输出
import "github.com/fatih/color"

// JSON 序列化
import "encoding/json"
```

### 6.2 代码结构建议

```
internal/proxy/
├── tester.go          # 延迟测试器（核心）
├── selector.go        # 节点选择器（核心）
├── formatter.go       # 输出格式化
├── history.go         # 历史记录管理（新增）
├── strategy.go        # 选择策略（新增）
└── progress.go        # 进度显示（新增）
```

### 6.3 命令行参数建议

```bash
# 测速命令增强
mihomo-cli proxy test <group> [node] \
  --url <url> \              # 测试 URL
  --timeout <ms> \           # 超时时间
  --concurrent <n> \         # 并发数
  --mode <mode> \            # 测速模式
  --progress \               # 显示进度
  --sort <field> \           # 排序字段
  --filter <pattern> \       # 过滤节点
  --output <format>          # 输出格式

# 自动选择命令增强
mihomo-cli proxy auto <group> \
  --url <url> \              # 测试 URL
  --timeout <ms> \           # 超时时间
  --concurrent <n> \         # 并发数
  --strategy <strategy> \    # 选择策略
  --max-delay <ms> \         # 最大延迟
  --min-delay <ms> \         # 最小延迟
  --exclude <pattern> \      # 排除节点
  --progress \               # 显示进度
```

---

## 七、总结

### 7.1 当前状态

**已实现功能：**
- ✅ 完整的测速功能
- ✅ 并发测速支持
- ✅ 多种选择策略
- ✅ 友好的输出格式

**待改进功能：**
- ⚠️ 实时进度显示
- ⚠️ 测速状态反馈
- ⚠️ 历史记录管理
- ⚠️ 高级选择策略

### 7.2 对比结论

Go CLI 实现在性能、跨平台、部署便捷性方面优于 PowerShell 实现，但在实时进度显示和扩展性方面还有改进空间。

**推荐方案：**
1. 保留当前的 Go CLI 实现（性能优势）
2. 添加实时进度显示功能（提升用户体验）
3. 实现历史记录功能（增强实用性）
4. 提供多种选择策略（满足不同需求）

### 7.3 下一步行动

1. **立即开始**：实现实时进度显示
2. **本周完成**：改进状态反馈
3. **下周开始**：实现历史记录功能
4. **持续优化**：根据用户反馈调整功能

---

## 附录

### A. 参考文档

- `docs/更换节点与批量测速.txt` - PowerShell 实现参考
- `docs/subscription-import-guide.md` - 订阅导入指南
- `docs/spec/mihono-api.md` - Mihomo API 文档

### B. 相关代码文件

- `internal/proxy/tester.go` - 延迟测试器实现
- `internal/proxy/selector.go` - 节点选择器实现
- `internal/api/proxy.go` - API 客户端实现
- `cmd/proxy.go` - 命令行接口实现

### C. Mihomo API 接口

- `GET /proxies` - 获取代理列表
- `GET /proxies/{name}` - 获取代理详情
- `PUT /proxies/{group}` - 切换代理
- `GET /proxies/{name}/delay` - 测试延迟
- `DELETE /proxies/{group}` - 取消固定代理