# 功能实现详细设计方案

本文档详细说明进度显示、状态反馈、历史记录、智能选择四个功能的实现方案。

---

## 一、进度显示功能

### 1.1 功能描述

在批量测速时，实时显示测速进度，包括：
- 当前进度百分比
- 已完成/总数
- 当前测速的节点名称
- 预计剩余时间

### 1.2 技术选型

**推荐库：** `github.com/schollz/progressbar/v3`

**优势：**
- 轻量级，无外部依赖
- 支持多种样式和主题
- 跨平台兼容性好
- 易于集成

**安装：**
```bash
go get github.com/schollz/progressbar/v3
```

### 1.3 实现方案

#### 1.3.1 创建进度回调接口

**文件：** `internal/proxy/progress.go`

```go
package proxy

// ProgressCallback 进度回调函数类型
type ProgressCallback func(current, total int, nodeName string, result DelayResult)

// ProgressReporter 进度报告器
type ProgressReporter struct {
    callback ProgressCallback
}

// NewProgressReporter 创建进度报告器
func NewProgressReporter(callback ProgressCallback) *ProgressReporter {
    return &ProgressReporter{
        callback: callback,
    }
}

// Report 报告进度
func (r *ProgressReporter) Report(current, total int, nodeName string, result DelayResult) {
    if r.callback != nil {
        r.callback(current, total, nodeName, result)
    }
}
```

#### 1.3.2 修改 DelayTester 支持进度回调

**文件：** `internal/proxy/tester.go`

```go
// DelayTester 延迟测试器
type DelayTester struct {
    client     *api.Client
    testURL    string
    timeout    int
    concurrent int
    reporter   *ProgressReporter  // 新增：进度报告器
}

// SetReporter 设置进度报告器
func (t *DelayTester) SetReporter(reporter *ProgressReporter) {
    t.reporter = reporter
}

// TestNodes 测试多个节点的延迟（支持进度回调）
func (t *DelayTester) TestNodes(ctx context.Context, nodeNames []string) ([]types.DelayResult, error) {
    if len(nodeNames) == 0 {
        return []types.DelayResult{}, nil
    }

    results := make([]types.DelayResult, len(nodeNames))
    var wg sync.WaitGroup

    // 使用信号量控制并发数
    sem := make(chan struct{}, t.concurrent)

    // 创建完成计数器（用于按顺序报告进度）
    completed := make(chan struct{}, len(nodeNames))
    go func() {
        count := 0
        for range completed {
            count++
            // 报告进度（注意：这里报告的是完成顺序，不是测试顺序）
            if t.reporter != nil {
                t.reporter.Report(count, len(nodeNames), nodeNames[count-1], results[count-1])
            }
        }
    }()

    for i, nodeName := range nodeNames {
        wg.Add(1)
        go func(index int, name string) {
            defer wg.Done()

            // 获取信号量
            sem <- struct{}{}
            defer func() { <-sem }()

            results[index] = t.TestSingle(ctx, name)

            // 通知完成
            completed <- struct{}{}
        }(i, nodeName)
    }

    wg.Wait()
    close(completed)
    return results, nil
}
```

#### 1.3.3 在命令行中集成进度条

**文件：** `cmd/proxy.go`

```go
import (
    "github.com/schollz/progressbar/v3"
)

// runProxyTest 执行测试延迟命令（带进度条）
func runProxyTest(cmd *cobra.Command, args []string) error {
    groupName := args[0]

    // 创建 API 客户端
    client := api.NewClientWithTimeout(
        viper.GetString("api.address"),
        viper.GetString("api.secret"),
        viper.GetInt("api.timeout"),
    )

    // 创建延迟测试器
    tester := proxy.NewDelayTester(client)
    if testURL != "" {
        tester.SetTestURL(testURL)
    }
    if testTimeout > 0 {
        tester.SetTimeout(testTimeout)
    }
    if concurrent > 0 {
        tester.SetConcurrent(concurrent)
    }

    // 获取代理组信息（用于获取节点总数）
    proxy, err := client.GetProxy(cmd.Context(), groupName)
    if err != nil {
        return fmt.Errorf("获取代理组失败: %w", err)
    }

    if len(proxy.All) == 0 {
        fmt.Println("代理组中没有节点")
        return nil
    }

    // 创建进度条
    bar := progressbar.NewOptions(
        len(proxy.All),
        progressbar.OptionSetDescription("测速中"),
        progressbar.OptionSetWriter(os.Stderr),  // 输出到 stderr，避免干扰表格输出
        progressbar.OptionShowCount(),
        progressbar.OptionShowIts(),
        progressbar.OptionSetTheme(progressbar.Theme{
            Suffix: "   ",
            SuffixColor: "[cyan]",
        }),
    )

    // 创建进度报告器
    reporter := proxy.NewProgressReporter(func(current, total int, nodeName string, result types.DelayResult) {
        bar.Set(current)
    })
    tester.SetReporter(reporter)

    var results []types.DelayResult

    // 如果指定了节点名称，测试单个节点
    if len(args) == 2 {
        nodeName := args[1]
        result := tester.TestSingle(cmd.Context(), nodeName)
        results = []types.DelayResult{result}
    } else {
        // 测试代理组中所有节点
        results, err = tester.TestGroup(cmd.Context(), groupName)
        if err != nil {
            return fmt.Errorf("测试延迟失败: %w", err)
        }
    }

    // 关闭进度条
    bar.Finish()

    // 换行，确保表格输出在新行
    fmt.Println()

    // 格式化输出结果
    return proxy.FormatTestResults(results, output)
}
```

### 1.4 使用示例

```bash
# 测速时显示进度条
.\mihomo-cli.exe proxy test PROXY

# 输出示例：
测速中 15/30 [=====================================>---]  83%  2.5s

┌─────────────────────────┬────────┬────────┐
│      节点名称           │  延迟  │  状态  │
├─────────────────────────┼────────┼────────┤
│ 香港-优化-Gemini        │  45ms  │ 优秀   │
│ 日本-优化               │  78ms  │ 优秀   │
│ ...
```

### 1.5 注意事项

1. **输出到 stderr**：进度条输出到 stderr，避免干扰表格输出（stdout）
2. **并发进度**：由于测速是并发的，进度显示的是完成顺序，不是测试顺序
3. **性能影响**：进度回调会轻微影响性能，但影响很小（< 1%）
4. **终端兼容性**：progressbar 库在 Windows、Linux、macOS 上都能正常工作

---

## 二、状态反馈功能

### 2.1 功能描述

在测速过程中和测速完成后，提供详细的状态反馈：
- 测速耗时
- 详细的状态描述
- 错误类型分类
- 实时状态更新

### 2.2 扩展类型定义

**文件：** `pkg/types/proxy.go`

```go
// DelayResult 延迟测试结果（扩展版）
type DelayResult struct {
    Name   string    // 节点名称
    Delay  uint16    // 延迟（毫秒）
    Error  error     // 错误信息
    Status string    // 状态描述
    Time   int64     // 测速耗时（毫秒）
    Type   string    // 节点类型（可选）
}

// TestStatus 测速状态
type TestStatus string

const (
    StatusUnknown    TestStatus = "未知"     // 未知状态
    StatusSuccess    TestStatus = "成功"     // 测速成功
    StatusTimeout    TestStatus = "超时"     // 连接超时
    StatusFailed     TestStatus = "失败"     // 连接失败
    StatusDnsError   TestStatus = "DNS错误"  // DNS 解析失败
    StatusNetworkError TestStatus = "网络错误" // 网络错误
)

// StatusInfo 状态信息
type StatusInfo struct {
    Status   TestStatus // 状态
    Message  string     // 详细消息
    Category string     // 错误分类
}
```

### 2.3 实现方案

#### 2.3.1 扩展 TestSingle 方法

**文件：** `internal/proxy/tester.go`

```go
import (
    "time"
    "strings"
    "errors"
)

// TestSingle 测试单个代理的延迟（带详细状态）
func (t *DelayTester) TestSingle(ctx context.Context, proxyName string) types.DelayResult {
    start := time.Now()
    result := types.DelayResult{
        Name: proxyName,
    }

    delay, err := t.client.TestDelay(ctx, proxyName, t.testURL, t.timeout)
    result.Time = time.Since(start).Milliseconds()

    // 分析错误类型
    statusInfo := t.analyzeError(err, delay)
    result.Status = string(statusInfo.Status)

    if err != nil {
        result.Error = err
    } else if delay == 0 {
        result.Status = string(types.StatusUnknown)
    } else {
        result.Delay = delay
    }

    return result
}

// analyzeError 分析错误类型
func (t *DelayTester) analyzeError(err error, delay uint16) types.StatusInfo {
    if err == nil {
        if delay > 0 {
            return types.StatusInfo{
                Status:   types.StatusSuccess,
                Message:  fmt.Sprintf("测速成功，延迟 %dms", delay),
                Category: "success",
            }
        }
        return types.StatusInfo{
            Status:   types.StatusUnknown,
            Message:  "测速成功但延迟为 0",
            Category: "warning",
        }
    }

    errMsg := err.Error()
    errStr := strings.ToLower(errMsg)

    // 分类错误
    switch {
    case strings.Contains(errStr, "timeout") || strings.Contains(errStr, "deadline exceeded"):
        return types.StatusInfo{
            Status:   types.StatusTimeout,
            Message:  fmt.Sprintf("连接超时（%dms）", t.timeout),
            Category: "timeout",
        }
    case strings.Contains(errStr, "dns") || strings.Contains(errStr, "no such host"):
        return types.StatusInfo{
            Status:   types.StatusDnsError,
            Message:  "DNS 解析失败",
            Category: "dns",
        }
    case strings.Contains(errStr, "connection refused") || strings.Contains(errStr, "connection reset"):
        return types.StatusInfo{
            Status:   types.StatusNetworkError,
            Message:  "连接被拒绝或重置",
            Category: "network",
        }
    default:
        return types.StatusInfo{
            Status:   types.StatusFailed,
            Message:  fmt.Sprintf("连接失败: %s", errMsg),
            Category: "error",
        }
    }
}
```

#### 2.3.2 改进输出格式化

**文件：** `internal/proxy/formatter.go`

```go
// formatTestResultsTable 以表格格式输出测试结果（增强版）
func formatTestResultsTable(results []types.DelayResult) error {
    table := tablewriter.NewTable(os.Stdout,
        tablewriter.WithHeader([]string{"节点名称", "延迟", "耗时", "状态", "分类"}),
        tablewriter.WithHeaderAutoFormat(tw.On),
        tablewriter.WithRowAlignment(tw.AlignLeft),
        tablewriter.WithBorders(tw.Border{Left: tw.Off, Right: tw.Off, Top: tw.Off, Bottom: tw.Off}),
    )

    for _, result := range results {
        var delayStr string
        var timeStr string
        var status string
        var category string

        if result.Error != nil {
            delayStr = "-"
            timeStr = fmt.Sprintf("%dms", result.Time)

            // 根据状态设置颜色
            switch result.Status {
            case string(types.StatusTimeout):
                status = color.YellowString(result.Status)
                category = color.YellowString("超时")
            case string(types.StatusDnsError):
                status = color.RedString(result.Status)
                category = color.RedString("DNS")
            case string(types.StatusNetworkError):
                status = color.RedString(result.Status)
                category = color.RedString("网络")
            default:
                status = color.RedString(result.Status)
                category = color.RedString("错误")
            }
        } else if result.Delay == 0 {
            delayStr = "-"
            timeStr = fmt.Sprintf("%dms", result.Time)
            status = color.YellowString(result.Status)
            category = color.YellowString("未知")
        } else {
            delayStr = fmt.Sprintf("%dms", result.Delay)
            timeStr = fmt.Sprintf("%dms", result.Time)

            // 根据延迟设置状态
            if result.Delay < 100 {
                status = color.GreenString("优秀")
                category = color.GreenString("快")
            } else if result.Delay < 300 {
                status = color.YellowString("良好")
                category = color.YellowString("中")
            } else {
                status = color.RedString("较差")
                category = color.RedString("慢")
            }
        }

        table.Append([]string{
            result.Name,
            delayStr,
            timeStr,
            status,
            category,
        })
    }

    table.Render()

    // 输出统计信息
    printStatistics(results)

    return nil
}

// printStatistics 打印统计信息
func printStatistics(results []types.DelayResult) {
    var successCount int
    var timeoutCount int
    var errorCount int
    var totalDelay uint64

    for _, result := range results {
        if result.Error == nil && result.Delay > 0 {
            successCount++
            totalDelay += uint64(result.Delay)
        } else if result.Error != nil {
            if strings.Contains(result.Status, "超时") {
                timeoutCount++
            } else {
                errorCount++
            }
        }
    }

    fmt.Println("\n统计信息:")
    fmt.Printf("  总计: %d 个节点\n", len(results))
    fmt.Printf("  成功: %s%d%s 个\n", color.GreenString(""), successCount, color.ResetString())
    fmt.Printf("  超时: %s%d%s 个\n", color.YellowString(""), timeoutCount, color.ResetString())
    fmt.Printf("  失败: %s%d%s 个\n", color.RedString(""), errorCount, color.ResetString())

    if successCount > 0 {
        avgDelay := totalDelay / uint64(successCount)
        fmt.Printf("  平均延迟: %dms\n", avgDelay)
    }

    successRate := float64(successCount) / float64(len(results)) * 100
    fmt.Printf("  成功率: %.1f%%\n", successRate)
}
```

### 2.4 使用示例

```bash
.\mihomo-cli.exe proxy test PROXY

# 输出示例：
测速中 30/30 [==========================================] 100%  5.2s

┌─────────────────────────┬────────┬────────┬────────┬────────┐
│      节点名称           │  延迟  │  耗时  │  状态  │  分类  │
├─────────────────────────┼────────┼────────┼────────┼────────┤
│ 香港-优化-Gemini        │  45ms  │  48ms  │ 优秀   │ 快     │
│ 日本-优化               │  78ms  │  82ms  │ 优秀   │ 快     │
│ 美国-优化               │  345ms │ 350ms  │ 较差   │ 慢     │
│ 新加坡-优化             │   -    │ 5000ms │ 超时   │ 超时   │
│ 台湾-优化               │   -    │  23ms  │ DNS错误│ DNS    │
└─────────────────────────┴────────┴────────┴────────┴────────┘

统计信息:
  总计: 30 个节点
  成功: 25 个
  超时: 3 个
  失败: 2 个
  平均延迟: 87ms
  成功率: 83.3%
```

### 2.5 注意事项

1. **性能优化**：测速耗时使用 `time.Since()` 计算，精度为毫秒
2. **错误分类**：根据错误信息字符串匹配进行分类，可能不够精确
3. **统计信息**：统计信息只在测试完成后显示，避免干扰进度条
4. **颜色输出**：使用 ANSI 颜色代码，确保终端兼容性

---

## 三、历史记录功能

### 3.1 功能描述

保存和查询测速历史记录，支持：
- 保存每次测速的结果
- 查看历史趋势
- 节点稳定性分析
- 历史数据导出

### 3.2 技术选型

**存储方案：** SQLite 数据库

**优势：**
- 轻量级，无需额外服务
- 支持复杂查询和聚合
- 事务支持，数据安全
- 跨平台兼容性好

**推荐库：** `github.com/mattn/go-sqlite3`

**安装：**
```bash
go get github.com/mattn/go-sqlite3
```

### 3.3 数据库设计

#### 3.3.1 表结构

```sql
-- 测速记录表
CREATE TABLE speed_test_records (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    group_name TEXT NOT NULL,        -- 代理组名称
    test_time INTEGER NOT NULL,      -- 测速时间（Unix 时间戳）
    total_nodes INTEGER NOT NULL,    -- 总节点数
    success_nodes INTEGER NOT NULL,  -- 成功节点数
    failed_nodes INTEGER NOT NULL,   -- 失败节点数
    timeout_nodes INTEGER NOT NULL,  -- 超时节点数
    avg_delay INTEGER,               -- 平均延迟
    max_delay INTEGER,               -- 最大延迟
    min_delay INTEGER,               -- 最小延迟
    test_duration INTEGER NOT NULL   -- 测速耗时（毫秒）
);

-- 节点测速结果表
CREATE TABLE node_test_results (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    record_id INTEGER NOT NULL,      -- 关联测速记录 ID
    node_name TEXT NOT NULL,         -- 节点名称
    delay INTEGER,                   -- 延迟（毫秒）
    test_time INTEGER NOT NULL,      -- 测耗时（毫秒）
    status TEXT NOT NULL,            -- 状态
    error_message TEXT,              -- 错误信息
    FOREIGN KEY (record_id) REFERENCES speed_test_records(id) ON DELETE CASCADE
);

-- 创建索引
CREATE INDEX idx_speed_test_records_group ON speed_test_records(group_name);
CREATE INDEX idx_speed_test_records_time ON speed_test_records(test_time);
CREATE INDEX idx_node_test_results_record ON node_test_results(record_id);
CREATE INDEX idx_node_test_results_node ON node_test_results(node_name);
```

### 3.4 实现方案

#### 3.4.1 创建历史记录管理器

**文件：** `internal/proxy/history.go`

```go
package proxy

import (
    "database/sql"
    "fmt"
    "os"
    "path/filepath"
    "sync"
    "time"

    _ "github.com/mattn/go-sqlite3"

    "github.com/kkkqkx123/mihomo-cli/pkg/types"
)

// HistoryManager 历史记录管理器
type HistoryManager struct {
    db   *sql.DB
    mu   sync.RWMutex
    path string
}

// NewHistoryManager 创建历史记录管理器
func NewHistoryManager(dbPath string) (*HistoryManager, error) {
    // 确保目录存在
    dir := filepath.Dir(dbPath)
    if err := os.MkdirAll(dir, 0755); err != nil {
        return nil, fmt.Errorf("创建目录失败: %w", err)
    }

    // 打开数据库
    db, err := sql.Open("sqlite3", dbPath)
    if err != nil {
        return nil, fmt.Errorf("打开数据库失败: %w", err)
    }

    // 设置连接池
    db.SetMaxOpenConns(1)
    db.SetMaxIdleConns(1)

    // 创建表
    if err := createTables(db); err != nil {
        db.Close()
        return nil, fmt.Errorf("创建表失败: %w", err)
    }

    return &HistoryManager{
        db:   db,
        path: dbPath,
    }, nil
}

// createTables 创建数据库表
func createTables(db *sql.DB) error {
    sqlStatements := []string{
        `CREATE TABLE IF NOT EXISTS speed_test_records (
            id INTEGER PRIMARY KEY AUTOINCREMENT,
            group_name TEXT NOT NULL,
            test_time INTEGER NOT NULL,
            total_nodes INTEGER NOT NULL,
            success_nodes INTEGER NOT NULL,
            failed_nodes INTEGER NOT NULL,
            timeout_nodes INTEGER NOT NULL,
            avg_delay INTEGER,
            max_delay INTEGER,
            min_delay INTEGER,
            test_duration INTEGER NOT NULL
        )`,
        `CREATE TABLE IF NOT EXISTS node_test_results (
            id INTEGER PRIMARY KEY AUTOINCREMENT,
            record_id INTEGER NOT NULL,
            node_name TEXT NOT NULL,
            delay INTEGER,
            test_time INTEGER NOT NULL,
            status TEXT NOT NULL,
            error_message TEXT,
            FOREIGN KEY (record_id) REFERENCES speed_test_records(id) ON DELETE CASCADE
        )`,
        `CREATE INDEX IF NOT EXISTS idx_speed_test_records_group ON speed_test_records(group_name)`,
        `CREATE INDEX IF NOT EXISTS idx_speed_test_records_time ON speed_test_records(test_time)`,
        `CREATE INDEX IF NOT EXISTS idx_node_test_results_record ON node_test_results(record_id)`,
        `CREATE INDEX IF NOT EXISTS idx_node_test_results_node ON node_test_results(node_name)`,
    }

    for _, stmt := range sqlStatements {
        if _, err := db.Exec(stmt); err != nil {
            return fmt.Errorf("执行 SQL 失败: %w", err)
        }
    }

    return nil
}

// SaveTestResult 保存测速结果
func (h *HistoryManager) SaveTestResult(groupName string, results []types.DelayResult, testDuration int64) (int64, error) {
    h.mu.Lock()
    defer h.mu.Unlock()

    // 开始事务
    tx, err := h.db.Begin()
    if err != nil {
        return 0, fmt.Errorf("开始事务失败: %w", err)
    }
    defer tx.Rollback()

    // 统计数据
    var successCount, failedCount, timeoutCount int
    var totalDelay uint64
    var maxDelay, minDelay uint16
    minDelay = 65535

    for _, result := range results {
        if result.Error == nil && result.Delay > 0 {
            successCount++
            totalDelay += uint64(result.Delay)
            if result.Delay > maxDelay {
                maxDelay = result.Delay
            }
            if result.Delay < minDelay {
                minDelay = result.Delay
            }
        } else if result.Error != nil {
            if result.Status == string(types.StatusTimeout) {
                timeoutCount++
            } else {
                failedCount++
            }
        }
    }

    var avgDelay int
    if successCount > 0 {
        avgDelay = int(totalDelay / uint64(successCount))
    }

    // 插入测速记录
    now := time.Now().Unix()
    result, err := tx.Exec(
        `INSERT INTO speed_test_records 
         (group_name, test_time, total_nodes, success_nodes, failed_nodes, timeout_nodes, 
          avg_delay, max_delay, min_delay, test_duration)
         VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
        groupName, now, len(results), successCount, failedCount, timeoutCount,
        avgDelay, maxDelay, minDelay, testDuration,
    )
    if err != nil {
        return 0, fmt.Errorf("插入测速记录失败: %w", err)
    }

    recordID, err := result.LastInsertId()
    if err != nil {
        return 0, fmt.Errorf("获取记录 ID 失败: %w", err)
    }

    // 插入节点结果
    stmt, err := tx.Prepare(
        `INSERT INTO node_test_results 
         (record_id, node_name, delay, test_time, status, error_message)
         VALUES (?, ?, ?, ?, ?, ?)`,
    )
    if err != nil {
        return 0, fmt.Errorf("准备语句失败: %w", err)
    }
    defer stmt.Close()

    for _, result := range results {
        var delay sql.NullInt16
        var errMsg sql.NullString

        if result.Delay > 0 {
            delay.Int16 = int16(result.Delay)
            delay.Valid = true
        }

        if result.Error != nil {
            errMsg.String = result.Error.Error()
            errMsg.Valid = true
        }

        _, err := stmt.Exec(
            recordID, result.Name, delay, result.Time, result.Status, errMsg,
        )
        if err != nil {
            return 0, fmt.Errorf("插入节点结果失败: %w", err)
        }
    }

    // 提交事务
    if err := tx.Commit(); err != nil {
        return 0, fmt.Errorf("提交事务失败: %w", err)
    }

    return recordID, nil
}

// GetHistory 获取历史记录
func (h *HistoryManager) GetHistory(groupName string, limit int, offset int) ([]TestRecord, error) {
    h.mu.RLock()
    defer h.mu.RUnlock()

    query := `
        SELECT id, group_name, test_time, total_nodes, success_nodes, failed_nodes, timeout_nodes,
               avg_delay, max_delay, min_delay, test_duration
        FROM speed_test_records
        WHERE group_name = ?
        ORDER BY test_time DESC
        LIMIT ? OFFSET ?
    `

    rows, err := h.db.Query(query, groupName, limit, offset)
    if err != nil {
        return nil, fmt.Errorf("查询历史记录失败: %w", err)
    }
    defer rows.Close()

    var records []TestRecord
    for rows.Next() {
        var record TestRecord
        var testTime int64

        err := rows.Scan(
            &record.ID, &record.GroupName, &testTime, &record.TotalNodes,
            &record.SuccessNodes, &record.FailedNodes, &record.TimeoutNodes,
            &record.AvgDelay, &record.MaxDelay, &record.MinDelay, &record.TestDuration,
        )
        if err != nil {
            return nil, fmt.Errorf("扫描记录失败: %w", err)
        }

        record.TestTime = time.Unix(testTime, 0)
        records = append(records, record)
    }

    return records, nil
}

// GetNodeHistory 获取节点历史
func (h *HistoryManager) GetNodeHistory(nodeName string, hours int, limit int) ([]NodeHistory, error) {
    h.mu.RLock()
    defer h.mu.RUnlock()

    since := time.Now().Add(-time.Duration(hours) * time.Hour).Unix()

    query := `
        SELECT r.test_time, n.delay, n.status
        FROM node_test_results n
        JOIN speed_test_records r ON n.record_id = r.id
        WHERE n.node_name = ? AND r.test_time >= ?
        ORDER BY r.test_time DESC
        LIMIT ?
    `

    rows, err := h.db.Query(query, nodeName, since, limit)
    if err != nil {
        return nil, fmt.Errorf("查询节点历史失败: %w", err)
    }
    defer rows.Close()

    var history []NodeHistory
    for rows.Next() {
        var h NodeHistory
        var testTime int64
        var delay sql.NullInt16

        err := rows.Scan(&testTime, &delay, &h.Status)
        if err != nil {
            return nil, fmt.Errorf("扫描历史失败: %w", err)
        }

        h.Time = time.Unix(testTime, 0)
        if delay.Valid {
            h.Delay = uint16(delay.Int16)
        }

        history = append(history, h)
    }

    return history, nil
}

// Close 关闭数据库连接
func (h *HistoryManager) Close() error {
    h.mu.Lock()
    defer h.mu.Unlock()

    return h.db.Close()
}

// TestRecord 测速记录
type TestRecord struct {
    ID            int64     // 记录 ID
    GroupName     string    // 代理组名称
    TestTime      time.Time // 测速时间
    TotalNodes    int       // 总节点数
    SuccessNodes  int       // 成功节点数
    FailedNodes   int       // 失败节点数
    TimeoutNodes  int       // 超时节点数
    AvgDelay      int       // 平均延迟
    MaxDelay      uint16    // 最大延迟
    MinDelay      uint16    // 最小延迟
    TestDuration  int64     // 测速耗时（毫秒）
}

// NodeHistory 节点历史
type NodeHistory struct {
    Time   time.Time // 测速时间
    Delay  uint16    // 延迟
    Status string    // 状态
}
```

#### 3.4.2 集成到测速流程

**文件：** `cmd/proxy.go`

```go
var (
    historyFile string  // 历史记录文件路径
    saveHistory bool    // 是否保存历史记录
)

func init() {
    // 在命令中添加历史记录相关参数
    proxyCmd := NewProxyCmd()
    proxyCmd.PersistentFlags().StringVar(&historyFile, "history-file", "", "历史记录文件路径")
    proxyCmd.PersistentFlags().BoolVar(&saveHistory, "save-history", false, "保存测速历史")
}

// runProxyTest 执行测试延迟命令（带历史记录）
func runProxyTest(cmd *cobra.Command, args []string) error {
    // ... 原有的测速代码 ...

    startTime := time.Now()

    // 执行测速
    results, err := tester.TestGroup(cmd.Context(), groupName)
    if err != nil {
        return fmt.Errorf("测试延迟失败: %w", err)
    }

    testDuration := time.Since(startTime).Milliseconds()

    // 保存历史记录
    if saveHistory && historyFile != "" {
        historyManager, err := proxy.NewHistoryManager(historyFile)
        if err != nil {
            fmt.Fprintf(os.Stderr, "警告：创建历史记录管理器失败: %v\n", err)
        } else {
            defer historyManager.Close()

            recordID, err := historyManager.SaveTestResult(groupName, results, testDuration)
            if err != nil {
                fmt.Fprintf(os.Stderr, "警告：保存历史记录失败: %v\n", err)
            } else {
                fmt.Fprintf(os.Stderr, "历史记录已保存 (ID: %d)\n", recordID)
            }
        }
    }

    // ... 原有的输出代码 ...
}
```

### 3.5 使用示例

```bash
# 保存历史记录
.\mihomo-cli.exe proxy test PROXY --save-history --history-file ~/.mihomo-cli/history.db

# 查看历史记录（新增命令）
.\mihomo-cli.exe proxy history PROXY --limit 10

# 查看节点历史（新增命令）
.\mihomo-cli.exe proxy node-history "香港-优化-Gemini" --hours 24 --limit 20
```

### 3.6 注意事项

1. **并发安全**：使用读写锁保护数据库访问
2. **事务处理**：使用事务确保数据一致性
3. **数据清理**：定期清理过期历史记录（可配置）
4. **性能优化**：合理设置连接池大小
5. **跨平台**：确保数据库路径在不同系统下正确

---

## 四、智能选择功能

### 4.1 功能描述

提供多种节点选择策略：
- **最快节点**：选择延迟最低的节点
- **最稳定节点**：选择历史成功率最高的节点
- **最低抖动**：选择延迟波动最小的节点
- **平衡模式**：综合延迟、稳定性、抖动等因素

### 4.2 实现方案

#### 4.2.1 定义选择策略

**文件：** `internal/proxy/strategy.go`

```go
package proxy

import (
    "fmt"
    "math"
    "sort"
    "time"

    "github.com/kkkqkx123/mihomo-cli/pkg/types"
)

// SelectionStrategy 选择策略
type SelectionStrategy string

const (
    StrategyFastest      SelectionStrategy = "fastest"       // 最快节点
    StrategyMostStable   SelectionStrategy = "most_stable"   // 最稳定节点
    StrategyLowestJitter SelectionStrategy = "lowest_jitter" // 最低抖动
    StrategyBalanced     SelectionStrategy = "balanced"      // 平衡模式
)

// StrategyConfig 策略配置
type StrategyConfig struct {
    Strategy        SelectionStrategy // 策略类型
    MinSuccessRate  float64           // 最低成功率（0-1）
    MaxDelay        uint16            // 最大延迟（毫秒）
    MaxJitter       uint16            // 最大抖动（毫秒）
    HistoryHours    int               // 历史数据时间范围（小时）
    StabilityWeight float64           // 稳定性权重（平衡模式）
    DelayWeight     float64           // 延迟权重（平衡模式）
    JitterWeight    float64           // 抖动权重（平衡模式）
}

// DefaultStrategyConfig 默认策略配置
func DefaultStrategyConfig() StrategyConfig {
    return StrategyConfig{
        Strategy:        StrategyFastest,
        MinSuccessRate:  0.8,  // 最低成功率 80%
        MaxDelay:        500,  // 最大延迟 500ms
        MaxJitter:       100,  // 最大抖动 100ms
        HistoryHours:    24,   // 24 小时历史数据
        StabilityWeight: 0.4,  // 稳定性权重 40%
        DelayWeight:     0.4,  // 延迟权重 40%
        JitterWeight:    0.2,  // 抖动权重 20%
    }
}

// NodeScore 节点评分
type NodeScore struct {
    Name         string  // 节点名称
    Delay        uint16  // 当前延迟
    SuccessRate  float64 // 成功率
    Jitter       uint16  // 抖动（标准差）
    Stability    float64 // 稳定性评分
    Score        float64 // 综合评分
}

// StrategySelector 策略选择器
type StrategySelector struct {
    tester        *DelayTester
    historyMgr    *HistoryManager
    config        StrategyConfig
}

// NewStrategySelector 创建策略选择器
func NewStrategySelector(tester *DelayTester, historyMgr *HistoryManager, config StrategyConfig) *StrategySelector {
    return &StrategySelector{
        tester:     tester,
        historyMgr: historyMgr,
        config:     config,
    }
}

// SelectBestNode 选择最佳节点
func (s *StrategySelector) SelectBestNode(ctx context.Context, groupName string) (string, *NodeScore, error) {
    // 获取代理组信息
    proxy, err := s.tester.client.GetProxy(ctx, groupName)
    if err != nil {
        return "", nil, fmt.Errorf("获取代理组失败: %w", err)
    }

    if len(proxy.All) == 0 {
        return "", nil, fmt.Errorf("代理组中没有节点")
    }

    // 测试所有节点延迟
    results := s.tester.TestNodes(ctx, proxy.All)

    // 计算节点评分
    scores := s.calculateNodeScores(proxy.All, results)

    // 根据策略选择节点
    var bestScore *NodeScore
    switch s.config.Strategy {
    case StrategyFastest:
        bestScore = s.selectFastest(scores)
    case StrategyMostStable:
        bestScore = s.selectMostStable(scores)
    case StrategyLowestJitter:
        bestScore = s.selectLowestJitter(scores)
    case StrategyBalanced:
        bestScore = s.selectBalanced(scores)
    default:
        bestScore = s.selectFastest(scores)
    }

    if bestScore == nil {
        return "", nil, fmt.Errorf("没有符合条件的节点")
    }

    return bestScore.Name, bestScore, nil
}

// calculateNodeScores 计算节点评分
func (s *StrategySelector) calculateNodeScores(nodeNames []string, results []types.DelayResult) []*NodeScore {
    nodeMap := make(map[string]*NodeScore)

    // 初始化节点评分
    for _, name := range nodeNames {
        nodeMap[name] = &NodeScore{
            Name: name,
        }
    }

    // 填充当前延迟
    for _, result := range results {
        if node, exists := nodeMap[result.Name]; exists {
            node.Delay = result.Delay
        }
    }

    // 计算历史统计
    if s.historyMgr != nil {
        for _, name := range nodeNames {
            history, err := s.historyMgr.GetNodeHistory(name, s.config.HistoryHours, 100)
            if err == nil && len(history) > 0 {
                calculateNodeStats(nodeMap[name], history)
            }
        }
    }

    // 计算综合评分
    for _, node := range nodeMap {
        calculateCompositeScore(node, s.config)
    }

    // 转换为切片
    var scores []*NodeScore
    for _, node := range nodeMap {
        scores = append(scores, node)
    }

    return scores
}

// calculateNodeStats 计算节点统计信息
func calculateNodeStats(node *NodeScore, history []NodeHistory) {
    var successCount, totalCount int
    var delays []float64

    for _, h := range history {
        totalCount++
        if h.Delay > 0 {
            successCount++
            delays = append(delays, float64(h.Delay))
        }
    }

    if totalCount > 0 {
        node.SuccessRate = float64(successCount) / float64(totalCount)
    }

    if len(delays) > 0 {
        // 计算平均延迟
        sum := 0.0
        for _, d := range delays {
            sum += d
        }
        avg := sum / float64(len(delays))

        // 计算标准差（抖动）
        variance := 0.0
        for _, d := range delays {
            variance += math.Pow(d-avg, 2)
        }
        variance /= float64(len(delays))
        node.Jitter = uint16(math.Sqrt(variance))

        // 稳定性评分（成功率越高，抖动越小，稳定性越高）
        node.Stability = (node.SuccessRate * 0.7) + (1.0 - math.Min(float64(node.Jitter)/200.0, 1.0)*0.3)
    }
}

// calculateCompositeScore 计算综合评分
func calculateCompositeScore(node *NodeScore, config StrategyConfig) {
    // 标准化各项指标（0-1）
    delayScore := 0.0
    if node.Delay > 0 {
        delayScore = 1.0 - math.Min(float64(node.Delay)/float64(config.MaxDelay), 1.0)
    }

    stabilityScore := node.Stability
    jitterScore := 0.0
    if node.Jitter > 0 {
        jitterScore = 1.0 - math.Min(float64(node.Jitter)/float64(config.MaxJitter), 1.0)
    }

    // 综合评分
    node.Score = (delayScore * config.DelayWeight) +
                 (stabilityScore * config.StabilityWeight) +
                 (jitterScore * config.JitterWeight)
}

// selectFastest 选择最快节点
func (s *StrategySelector) selectFastest(scores []*NodeScore) *NodeScore {
    var validScores []*NodeScore

    for _, score := range scores {
        // 过滤条件：延迟 > 0，成功率 >= 最低成功率
        if score.Delay > 0 && score.SuccessRate >= s.config.MinSuccessRate {
            validScores = append(validScores, score)
        }
    }

    if len(validScores) == 0 {
        return nil
    }

    // 按延迟排序
    sort.Slice(validScores, func(i, j int) bool {
        return validScores[i].Delay < validScores[j].Delay
    })

    return validScores[0]
}

// selectMostStable 选择最稳定节点
func (s *StrategySelector) selectMostStable(scores []*NodeScore) *NodeScore {
    var validScores []*NodeScore

    for _, score := range scores {
        // 过滤条件：有历史数据，成功率 >= 最低成功率
        if score.SuccessRate > 0 && score.SuccessRate >= s.config.MinSuccessRate {
            validScores = append(validScores, score)
        }
    }

    if len(validScores) == 0 {
        return nil
    }

    // 按稳定性评分排序
    sort.Slice(validScores, func(i, j int) bool {
        return validScores[i].Stability > validScores[j].Stability
    })

    return validScores[0]
}

// selectLowestJitter 选择最低抖动节点
func (s *StrategySelector) selectLowestJitter(scores []*NodeScore) *NodeScore {
    var validScores []*NodeScore

    for _, score := range scores {
        // 过滤条件：有抖动数据，成功率 >= 最低成功率
        if score.Jitter > 0 && score.SuccessRate >= s.config.MinSuccessRate {
            validScores = append(validScores, score)
        }
    }

    if len(validScores) == 0 {
        return nil
    }

    // 按抖动排序
    sort.Slice(validScores, func(i, j int) bool {
        return validScores[i].Jitter < validScores[j].Jitter
    })

    return validScores[0]
}

// selectBalanced 选择平衡模式节点
func (s *StrategySelector) selectBalanced(scores []*NodeScore) *NodeScore {
    var validScores []*NodeScore

    for _, score := range scores {
        // 过滤条件：有完整数据，满足各项要求
        if score.Delay > 0 &&
           score.SuccessRate >= s.config.MinSuccessRate &&
           score.Delay <= s.config.MaxDelay &&
           score.Jitter <= s.config.MaxJitter {
            validScores = append(validScores, score)
        }
    }

    if len(validScores) == 0 {
        return nil
    }

    // 按综合评分排序
    sort.Slice(validScores, func(i, j int) bool {
        return validScores[i].Score > validScores[j].Score
    })

    return validScores[0]
}
```

#### 4.2.2 扩展命令行接口

**文件：** `cmd/proxy.go`

```go
var (
    strategy      string  // 选择策略
    minSuccessRate float64 // 最低成功率
    maxDelay      int     // 最大延迟
    maxJitter     int     // 最大抖动
    historyHours  int     // 历史数据时间范围
)

func init() {
    autoCmd := newProxyAutoCmd()
    autoCmd.Flags().StringVar(&strategy, "strategy", "fastest", "选择策略 (fastest/most_stable/lowest_jitter/balanced)")
    autoCmd.Flags().Float64Var(&minSuccessRate, "min-success-rate", 0.8, "最低成功率 (0-1)")
    autoCmd.Flags().IntVar(&maxDelay, "max-delay", 500, "最大延迟（毫秒）")
    autoCmd.Flags().IntVar(&maxJitter, "max-jitter", 100, "最大抖动（毫秒）")
    autoCmd.Flags().IntVar(&historyHours, "history-hours", 24, "历史数据时间范围（小时）")
}

// runProxyAuto 执行自动选择命令（增强版）
func runProxyAuto(cmd *cobra.Command, args []string) error {
    groupName := args[0]

    // 创建 API 客户端
    client := api.NewClientWithTimeout(
        viper.GetString("api.address"),
        viper.GetString("api.secret"),
        viper.GetInt("api.timeout"),
    )

    // 创建延迟测试器
    tester := proxy.NewDelayTester(client)
    if testURL != "" {
        tester.SetTestURL(testURL)
    }
    if testTimeout > 0 {
        tester.SetTimeout(testTimeout)
    }
    if concurrent > 0 {
        tester.SetConcurrent(concurrent)
    }

    // 创建历史记录管理器
    var historyMgr *proxy.HistoryManager
    if historyFile != "" {
        var err error
        historyMgr, err = proxy.NewHistoryManager(historyFile)
        if err != nil {
            fmt.Fprintf(os.Stderr, "警告：创建历史记录管理器失败: %v\n", err)
        } else {
            defer historyMgr.Close()
        }
    }

    // 创建策略选择器
    config := proxy.StrategyConfig{
        Strategy:        proxy.SelectionStrategy(strategy),
        MinSuccessRate:  minSuccessRate,
        MaxDelay:        uint16(maxDelay),
        MaxJitter:       uint16(maxJitter),
        HistoryHours:    historyHours,
        StabilityWeight: 0.4,
        DelayWeight:     0.4,
        JitterWeight:    0.2,
    }

    selector := proxy.NewStrategySelector(tester, historyMgr, config)

    // 选择最佳节点
    bestNode, score, err := selector.SelectBestNode(cmd.Context(), groupName)
    if err != nil {
        return fmt.Errorf("自动选择失败: %w", err)
    }

    // 切换到最佳节点
    err = client.SwitchProxy(cmd.Context(), groupName, bestNode)
    if err != nil {
        return fmt.Errorf("切换到节点 %s 失败: %w", bestNode, err)
    }

    // 输出结果
    fmt.Printf("%s", color.GreenString("✓ 已自动切换到最佳节点\n"))
    fmt.Printf("  代理组: %s\n", groupName)
    fmt.Printf("  节点: %s\n", bestNode)
    fmt.Printf("  延迟: %dms\n", score.Delay)
    fmt.Printf("  成功率: %.1f%%\n", score.SuccessRate*100)
    fmt.Printf("  抖动: %dms\n", score.Jitter)
    fmt.Printf("  综合评分: %.2f\n", score.Score)

    return nil
}
```

### 4.3 使用示例

```bash
# 最快节点策略
.\mihomo-cli.exe proxy auto PROXY --strategy fastest

# 最稳定节点策略
.\mihomo-cli.exe proxy auto PROXY --strategy most_stable --history-hours 48

# 最低抖动策略
.\mihomo-cli.exe proxy auto PROXY --strategy lowest_jitter --max-jitter 50

# 平衡模式
.\mihomo-cli.exe proxy auto PROXY --strategy balanced \
  --min-success-rate 0.9 \
  --max-delay 300 \
  --max-jitter 50 \
  --history-hours 24
```

### 4.4 注意事项

1. **历史数据依赖**：部分策略需要历史数据，首次使用时可能效果不佳
2. **参数调优**：不同网络环境需要调整策略参数
3. **性能考虑**：历史数据查询可能影响性能，建议缓存结果
4. **错误处理**：当没有符合条件的节点时，需要友好的错误提示

---

## 五、集成方案总结

### 5.1 依赖库汇总

```go
require (
    github.com/mattn/go-sqlite3 v1.14.22  // SQLite 数据库
    github.com/schollz/progressbar/v3 v3.14.1  // 进度条
    // ... 其他现有依赖
)
```

### 5.2 新增文件清单

```
internal/proxy/
├── progress.go      # 进度回调接口
├── history.go       # 历史记录管理器
└── strategy.go      # 智能选择策略

pkg/types/
└── proxy.go         # 扩展类型定义（已存在，需修改）
```

### 5.3 文件修改清单

```
internal/proxy/
├── tester.go        # 添加进度报告器支持
└── formatter.go     # 改进输出格式化和统计信息

cmd/
└── proxy.go         # 集成新功能和命令行参数
```

### 5.4 配置文件更新

```toml
[proxy]
# 历史记录配置
history_enabled = true
history_file = "~/.mihomo-cli/history.db"
history_retention_days = 30

# 智能选择配置
default_strategy = "balanced"
min_success_rate = 0.8
max_delay = 500
max_jitter = 100
history_hours = 24

# 测速配置
show_progress = true
show_statistics = true
```

### 5.5 实现优先级

**高优先级（第一周）：**
1. ✅ 进度显示功能
2. ✅ 状态反馈功能

**中优先级（第二周）：**
3. ✅ 历史记录功能
4. ✅ 智能选择功能

**低优先级（后续优化）：**
5. 历史数据可视化
6. 策略性能优化
7. 更多选择策略

### 5.6 测试计划

**单元测试：**
- `tester_test.go` - 测试延迟测试器
- `history_test.go` - 测试历史记录管理
- `strategy_test.go` - 测试选择策略

**集成测试：**
- 测试完整的测速流程
- 测试历史记录保存和查询
- 测试各种选择策略

**性能测试：**
- 测试大量节点（100+）的测速性能
- 测试历史记录查询性能
- 测试并发测速性能

---

## 六、风险评估

### 6.1 技术风险

| 风险 | 影响 | 概率 | 缓解措施 |
|------|------|------|----------|
| SQLite 兼容性问题 | 中 | 低 | 使用纯 Go 实现的 SQLite 驱动 |
| 进度条终端兼容性 | 低 | 中 | 提供禁用选项，测试主流终端 |
| 历史数据查询性能 | 中 | 中 | 添加索引，限制查询范围 |
| 选择策略准确性 | 中 | 中 | 提供多种策略，允许用户调整参数 |

### 6.2 用户体验风险

| 风险 | 影响 | 概率 | 缓解措施 |
|------|------|------|----------|
| 功能过于复杂 | 高 | 中 | 提供默认配置，简化常用操作 |
| 历史数据占用空间 | 低 | 中 | 提供自动清理功能 |
| 性能下降 | 中 | 低 | 优化数据库查询，添加缓存 |

---

## 七、后续优化方向

### 7.1 功能增强

1. **历史数据可视化**：生成延迟趋势图表
2. **智能推荐**：基于用户习惯推荐节点
3. **定时测速**：支持定时自动测速和节点切换
4. **结果通知**：测速完成或节点切换时发送通知

### 7.2 性能优化

1. **数据库缓存**：缓存常用查询结果
2. **并发优化**：优化并发控制策略
3. **增量更新**：只更新变化的节点

### 7.3 用户体验优化

1. **交互式选择**：提供交互式节点选择界面
2. **配置向导**：提供配置向导帮助用户设置
3. **帮助文档**：完善命令行帮助文档

---

## 八、总结

本方案详细说明了四个核心功能的实现：

1. **进度显示**：使用进度条库实现实时进度反馈
2. **状态反馈**：扩展结果类型，提供详细的状态信息
3. **历史记录**：使用 SQLite 数据库存储和查询历史数据
4. **智能选择**：实现多种选择策略，满足不同需求

所有功能都考虑了：
- 性能优化
- 用户体验
- 可扩展性
- 跨平台兼容性

建议按照优先级逐步实现，并在每个阶段进行充分测试。