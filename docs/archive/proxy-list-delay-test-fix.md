# Proxy List 测速功能修复与改进方案

## 问题总结

### 1. 测速全部显示超时的原因

**根本原因**：代码中尝试访问 `proxy.LogicalTypes[proxy.Type]`，但 `ProxyInfo` 结构体中没有 `LogicalTypes` 字段。`LogicalTypes` 是 `proxy` 包的包级变量。

**已修复**：修改 `shouldIncludeProxyForTest` 函数，使用本地定义的 `logicalTypes` map 来判断逻辑节点。

### 2. 节点缺失问题

**现象**：如"日本 JP=HY2"节点实际可用，但 list 无法显示。

**可能原因**：

1. **名称包含特殊字符**：节点名称中的 `=` 等特殊字符可能在正则过滤时被错误匹配
2. **Provider 节点未加载**：某些节点可能来自外部 Provider，在初始列表中没有被完全加载
3. **默认过滤规则**：可能存在某些默认的 exclude 规则过滤了这些节点

**解决方案**：

- 检查配置文件中的 `exclude` 参数
- 检查是否使用了 `--exclude` 或 `--exclude-logical` 标志
- 验证节点是否属于外部 Provider，可能需要刷新 Provider

### 3. 测速逻辑设计问题

**当前问题**：

- 没有批次限制，大型订阅会占用过多连接
- 缺少默认的超时时间设置
- 测速时没有进度反馈（除非使用 `--progress`）
- 测速时阻塞等待，用户体验差

**已修复**：

- ✅ 添加 `--batch-size` 参数（默认 100）
- ✅ 添加 `--max-nodes` 参数（默认 500）
- ✅ 添加 `--wait` 参数控制是否等待测速完成
- ✅ 设置默认超时时间为 10 秒
- ✅ 批次间添加 500ms 延迟，避免连接过多

## 改进方案

### 方案 1：优化测速逻辑（推荐）

#### 1.1 添加批次限制

```go
// 在 cmd/proxy.go 中添加
var (
    testBatchSize int  // 每批次测试的节点数
)

// 在 newProxyListCmd 中添加
cmd.Flags().IntVar(&testBatchSize, "batch-size", 100, "每批次测试的节点数（默认 100）")
```

#### 1.2 实现分批测速

```go
// 分批测试节点
func testNodesInBatches(tester *proxy.DelayTester, nodeNames []string, batchSize int) []types.DelayResult {
    var allResults []types.DelayResult

    for i := 0; i < len(nodeNames); i += batchSize {
        end := i + batchSize
        if end > len(nodeNames) {
            end = len(nodeNames)
        }

        batch := nodeNames[i:end]
        results, _ := tester.TestNodes(context.Background(), batch)
        allResults = append(allResults, results...)

        // 批次间短暂延迟，避免连接过多
        if end < len(nodeNames) {
            time.Sleep(500 * time.Millisecond)
        }
    }

    return allResults
}
```

#### 1.3 默认超时时间

在配置文件中添加默认超时时间：

```toml
[proxy]
test_url = "https://www.google.com/generate_204"
timeout = 10000  # 默认 10 秒
concurrent = 10  # 并发数
batch_size = 100  # 每批次节点数
```

### 方案 2：异步测速（可选）

#### 2.1 立即返回 + 后台测速

```bash
# 立即返回，不等待测速完成
mihomo-cli proxy list --test-delay --async

# 等待测速完成（默认行为）
mihomo-cli proxy list --test-delay --wait
```

#### 2.2 实现方式

```go
if async {
    // 启动后台 goroutine 测速
    go func() {
        tester.TestNodes(ctx, nodeNames)
    }()

    // 立即返回结果
    return proxy.FormatProxyList(proxies, groupFilter, outputFmt, filterOpts)
} else {
    // 等待测速完成
    results, _ := tester.TestNodes(ctx, nodeNames)
    // 更新延迟数据
    // ...
}
```

### 方案 3：智能测速策略

#### 3.1 基于历史记录的测速

```go
// 只测试没有历史记录或历史记录过时的节点
func shouldTestNode(proxy *types.ProxyInfo) bool {
    if len(proxy.History) == 0 {
        return true
    }

    // 检查最后测速时间
    lastTest := proxy.History[len(proxy.History)-1]
    return time.Since(lastTest.Time) > 5*time.Minute
}
```

#### 3.2 渐进式测速

```bash
# 第一次：快速测试（只测前 50 个节点）
mihomo-cli proxy list --test-delay --quick

# 第二次：完整测试（测试所有节点）
mihomo-cli proxy list --test-delay --full
```

## 实施建议

### 第一阶段：修复当前问题

1. ✅ 修复 `LogicalTypes` 引用错误
2. ✅ 排除逻辑节点测速
3. ⏳ 检查节点缺失问题

### 第二阶段：优化测速性能

1. 添加批次限制（默认 100 个节点/批次）
2. 添加默认超时时间（10 秒）
3. 优化并发控制

### 第三阶段：增强功能

1. 异步测速选项
2. 智能测速策略
3. 测速历史记录

## 配置示例

### 推荐配置（config.toml）

```toml
[proxy]
# 测速 URL（推荐使用 Google 204）
test_url = "https://www.google.com/generate_204"

# 超时时间（毫秒）
timeout = 10000

# 并发数（建议不超过 50）
concurrent = 20

# 每批次节点数（大型订阅建议 100-200）
batch_size = 100
```

### 使用示例

```bash
# 基本用法：列出节点（不测速，立即返回）
mihomo-cli proxy list

# 测速：等待测速完成（默认行为）
mihomo-cli proxy list --test-delay

# 测速 + 进度条 + 20 并发
mihomo-cli proxy list --test-delay --progress --concurrent 20

# 测速 + 批次限制（每批 100 个节点）
mihomo-cli proxy list --test-delay --batch-size 100

# 测速 + 限制最大节点数（最多测试 500 个）
mihomo-cli proxy list --test-delay --max-nodes 500

# 异步测速：立即返回，后台测速（不显示结果）
mihomo-cli proxy list --test-delay --wait=false

# 排除逻辑节点
mihomo-cli proxy list --test-delay --exclude-logical

# 按延迟排序
mihomo-cli proxy list --test-delay --sort delay

# 只测试特定类型
mihomo-cli proxy list --test-delay --type Vmess

# 自定义超时时间（15 秒）
mihomo-cli proxy list --test-delay --timeout 15000
```

## 节点缺失排查步骤

1. **检查节点名称**：

   ```bash
   # 使用 JSON 格式查看完整列表
   mihomo-cli proxy list -o json
   ```

2. **检查过滤规则**：

   ```bash
   # 不使用任何过滤
   mihomo-cli proxy list --exclude-logical=false
   ```

3. **刷新 Provider**：

   ```bash
   # 刷新外部 Provider
   mihomo-cli provider update
   ```

4. **检查 API 响应**：
   ```bash
   # 直接调用 API 查看原始数据
   curl http://127.0.0.1:9090/proxies
   ```

## 注意事项

1. **测速超时时间**：建议设置为 10 秒，过短可能导致误判
2. **并发数**：建议不超过 50，避免占用过多系统资源
3. **批次大小**：大型订阅（500+ 节点）建议分批测试，默认每批 100 个节点
4. **最大节点数**：默认最多测试 500 个节点，避免测速时间过长
5. **逻辑节点**：DIRECT、REJECT 等逻辑节点无法测速，会自动排除
6. **异步测速**：使用 `--wait=false` 可以立即返回，但测速结果不会显示在当前列表中
7. **批次延迟**：每批次之间会自动延迟 500ms，避免网络拥塞
