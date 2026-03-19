# 日志管理分析报告

## 一、`internal/log/formatter.go` 使用情况分析

### 1. 使用位置

该文件中的函数**仅在一个地方被使用**：`cmd/logs.go`

| 函数 | 位置 | 用途 |
|------|------|------|
| `FormatLogHeader()` | `cmd/logs.go:61` | 打印日志流开始提示 |
| `PrintLogMessage()` | `cmd/logs.go:86` | 打印每条格式化的日志消息 |
| `FormatLogMessage()` | 未直接调用 | 被 `PrintLogMessage` 内部调用 |

### 2. 功能说明

```go
// 格式化单条日志消息，根据类型添加颜色和前缀
func FormatLogMessage(log *types.LogInfo) string

// 打印格式化后的日志消息到 stdout
func PrintLogMessage(log *types.LogInfo)

// 打印日志流开始提示信息
func FormatLogHeader()
```

### 3. 支持的日志类型

| 类型 | 前缀 | 颜色 | 颜色常量 |
|------|------|------|----------|
| info | [INFO] | 绿色 | `color.FgGreen` |
| warning/warn | [WARN] | 黄色 | `color.FgYellow` |
| error | [ERROR] | 红色 | `color.FgRed` |
| debug | [DEBUG] | 青色 | `color.FgCyan` |
| silent | [SILENT] | 暗灰色 | `color.FgHiBlack` |
| 其他 | [TYPE] | 绿色(默认) | `color.FgGreen` |

### 4. 数据流

```
Mihomo 内核 → WebSocket API → internal/api/websocket.go → types.LogInfo → internal/log/formatter.go → stdout
```

---

## 二、项目日志管理架构

项目采用**双层日志管理架构**：

### 架构图

```
┌─────────────────────────────────────────────────────────────┐
│                    mihomo-cli 日志系统                        │
├─────────────────────────────────────────────────────────────┤
│                                                             │
│  ┌─────────────────────┐    ┌─────────────────────────┐    │
│  │   internal/log      │    │    internal/output      │    │
│  │   (外部日志流)       │    │    (内部状态输出)        │    │
│  ├─────────────────────┤    ├─────────────────────────┤    │
│  │ • FormatLogMessage  │    │ • Info()                │    │
│  │ • PrintLogMessage   │    │ • Success()             │    │
│  │ • FormatLogHeader   │    │ • Warning()             │    │
│  │                     │    │ • Error()               │    │
│  │ 数据源: WebSocket    │    │ 数据源: CLI 内部逻辑     │    │
│  │ 用途: logs 命令      │    │ 用途: 所有命令状态反馈   │    │
│  └─────────────────────┘    └─────────────────────────┘    │
│                                                             │
└─────────────────────────────────────────────────────────────┘
```

### 1. 外部日志流（`internal/log`）

- **用途**: 接收并展示 Mihomo 内核的实时日志
- **数据来源**: WebSocket 连接 Mihomo API (`/logs` 端点)
- **数据结构**: `types.LogInfo` (`pkg/types/base.go`)
- **使用场景**: `mihomo-cli logs` 命令
- **特点**: 
  - 实时流式处理
  - 支持多种日志级别
  - 颜色化输出

### 2. 内部状态输出（`internal/output`）

- **用途**: CLI 工具自身的状态消息输出
- **主要函数**:
  - `output.Info()` - 信息提示 (ℹ 前缀，青色)
  - `output.Success()` - 成功消息 (✓ 前缀，绿色)
  - `output.Warning()` - 警告消息 (⚠ 前缀，黄色)
  - `output.Error()` - 错误消息 (✗ 前缀，红色)
- **特点**:
  - 统一的视觉风格
  - 支持自定义 Writer
  - 颜色可配置

---

## 三、日志记录位置统计

### 按模块分类

| 模块 | 文件数 | 主要用途 |
|------|--------|----------|
| cmd | 12 | 命令执行状态反馈 |
| internal/mihomo | 6 | 进程管理、健康检查、扫描 |
| internal/config | 2 | 配置验证、系统检查 |
| internal/proxy | 1 | 代理信息格式化 |
| internal/connection | 1 | 连接信息格式化 |
| internal/dns | 1 | DNS 查询信息格式化 |
| internal/rule | 1 | 规则信息格式化 |
| internal/recovery | 1 | 恢复管理 |
| internal/sysproxy | 2 | 系统代理设置 |

### 详细文件列表

#### cmd 目录（命令层）

| 文件 | 日志类型 | 说明 |
|------|----------|------|
| logs.go | Info | 日志流状态提示 |
| service.go | Success, Info, Warning | 服务管理操作 |
| config.go | Success, Warning | 配置验证和操作 |
| backup.go | Success, Warning | 备份恢复操作 |
| cache.go | Success | 缓存操作 |
| geoip.go | Success | GeoIP 更新 |
| recovery.go | Success | 恢复操作 |
| sub.go | Warning, Error, Success | 订阅管理 |
| sysproxy.go | Success, Info | 系统代理设置 |
| mode.go | Success | 模式切换 |
| ps.go | Warning | 进程状态 |

#### internal/mihomo 目录（核心业务）

| 文件 | 日志类型 | 说明 |
|------|----------|------|
| manager.go | Success, Warning, Error | Mihomo 进程管理 |
| lifecycle.go | Warning, Error | 进程生命周期 |
| health_checker.go | Success, Error | 健康检查 |
| process_handler.go | Info, Warning, Success | 进程处理 |
| scanner.go | Error, Success | 进程扫描 |
| scanner_unix.go | Error, Success | Unix 进程扫描 |

#### internal/config 目录（配置处理）

| 文件 | 日志类型 | 说明 |
|------|----------|------|
| validator.go | Warning | 配置验证 |
| system_checker.go | Success | 系统检查 |

#### 格式化模块

| 文件 | 日志类型 | 说明 |
|------|----------|------|
| internal/proxy/formatter.go | Success, Warning, Info | 代理信息 |
| internal/connection/formatter.go | Info, Success | 连接信息 |
| internal/dns/formatter.go | Info, Warning | DNS 信息 |
| internal/rule/formatter.go | Info, Success, Warning | 规则信息 |

---

## 四、需要补充日志记录的位置

### 1. 缺少日志的关键位置

#### 1.1 API 请求层 (`internal/api`)

**当前状态**: API 层完全没有日志记录，所有请求都是静默执行的。

| 文件 | 函数/位置 | 行号 | 建议添加的日志 | 优先级 |
|------|-----------|------|----------------|--------|
| http.go | `doRequest()` | 70 | `output.Info("API 请求: %s %s", method, fullURL)` | 高 |
| http.go | `doRequest()` 超时 | 112 | `output.Warning("API 请求超时: %s", fullURL)` | 高 |
| http.go | `doRequest()` 连接失败 | 115 | `output.Error("API 连接失败: %s - %v", fullURL, err)` | 高 |
| http.go | `handleResponse()` 错误状态 | 126 | `output.Warning("API 响应错误: %s (状态码: %d)", endpoint, resp.StatusCode)` | 中 |
| websocket.go | `connectWebSocket()` | 51 | `output.Info("建立 WebSocket 连接: %s", wsURL)` | 中 |
| websocket.go | `readMessages()` 读取失败 | 112 | `output.Error("WebSocket 读取失败: %v", err)` | 中 |

**代码示例** (`internal/api/http.go:70`):
```go
func (c *HTTPClient) doRequest(ctx context.Context, method, baseURL, endpoint string, ...) (*http.Response, error) {
    // 构建完整 URL
    fullURL, err := c.buildURL(baseURL, endpoint, queryParams)
    if err != nil {
        return nil, NewConnectionError(err)
    }

    // 添加日志
    output.Info("API 请求: %s %s", method, fullURL)

    // ... 后续代码
}
```

#### 1.2 配置加载层 (`internal/config`)

**当前状态**: 配置加载过程没有日志，用户无法了解配置加载进度。

| 文件 | 函数/位置 | 行号 | 建议添加的日志 | 优先级 |
|------|-----------|------|----------------|--------|
| loader.go | `Load()` 开始 | 24 | `output.Info("加载配置文件: %s", configPath)` | 中 |
| loader.go | `Load()` 验证 | 43 | `output.Info("验证配置...")` | 低 |
| loader.go | `Save()` 开始 | 74 | `output.Info("保存配置到: %s", configPath)` | 中 |
| loader.go | `Save()` 完成 | 103 | `output.Success("配置保存成功")` | 低 |

**代码示例** (`internal/config/loader.go:24`):
```go
func (l *Loader) Load(configPath string) (*CLIConfig, error) {
    output.Info("加载配置文件: %s", configPath)

    l.v.SetConfigFile(configPath)
    if err := l.v.ReadInConfig(); err != nil {
        if _, ok := err.(viper.ConfigFileNotFoundError); ok {
            return nil, errors.ErrConfig("config file not found", nil)
        }
        return nil, errors.ErrConfig("failed to read config file", err)
    }

    // ... 后续代码
}
```

#### 1.3 命令层 (`cmd`)

**当前状态**: 部分命令缺少操作开始提示。

| 文件 | 函数/位置 | 行号 | 建议添加的日志 | 优先级 |
|------|-----------|------|----------------|--------|
| start.go | `runStart()` 开始 | 67 | `output.Info("查找配置文件...")` | 中 |
| start.go | `runStart()` 加载配置 | 72 | `output.Info("加载配置: %s", configPath)` | 中 |
| start.go | `runStart()` 启动进程 | 88 | `output.Info("启动 Mihomo 进程...")` | 高 |
| stop.go | `runStop()` 开始 | 152 | `output.Info("查找 Mihomo 进程...")` | 中 |
| monitor.go | `runTrafficWatch()` | 87 | `output.Info("开始流量监控...")` | 低 |
| monitor.go | `runMemoryWatch()` | 193 | `output.Info("开始内存监控...")` | 低 |

**代码示例** (`cmd/start.go:67`):
```go
func runStart(cmd *cobra.Command, args []string) error {
    output.Info("查找配置文件...")
    configPath := config.FindTomlConfigPath(cfgFile)

    output.Info("加载配置: %s", configPath)
    cfg, err := config.LoadTomlConfig(configPath)
    if err != nil {
        return pkgerrors.ErrConfig("failed to load config", err)
    }

    output.Info("启动 Mihomo 进程...")
    // ... 后续代码
}
```

#### 1.4 进程管理层 (`internal/mihomo`)

**当前状态**: 部分关键操作缺少日志。

| 文件 | 函数/位置 | 建议添加的日志 | 优先级 |
|------|-----------|----------------|--------|
| manager.go | `Start()` 启动前 | `output.Info("准备启动 Mihomo 进程 (配置: %s)", configPath)` | 高 |
| manager.go | `WritePIDFile()` | `output.Info("写入 PID 文件: %d -> %s", pid, pidFilePath)` | 中 |
| lifecycle.go | `Stop()` 发送信号 | `output.Info("发送停止信号到进程: PID=%d", pid)` | 中 |

### 2. 日志级别使用问题

#### 2.1 当前问题分析

| 文件 | 位置 | 当前级别 | 问题 | 建议级别 |
|------|------|----------|------|----------|
| cmd/ps.go:32 | 未找到进程 | Warning | 未找到进程是正常情况 | Info |
| internal/config/validator.go:67 | 配置项缺失 | Warning | 非关键配置缺失不应警告 | Info |
| internal/mihomo/manager.go:104 | 配置文件不存在 | Warning | 正常情况 | Info |

#### 2.2 日志级别使用规范

```
┌─────────────────────────────────────────────────────────────┐
│                      日志级别使用规范                         │
├─────────────────────────────────────────────────────────────┤
│                                                             │
│  Error   - 操作失败，影响功能正常使用                          │
│           例: API 连接失败、配置解析错误、进程启动失败           │
│                                                             │
│  Warning - 潜在问题，但不影响主要功能                          │
│           例: 配置项使用默认值、重试操作、降级处理               │
│                                                             │
│  Info    - 正常操作流程信息                                   │
│           例: 开始执行操作、操作进度、条件分支选择               │
│                                                             │
│  Success - 操作成功确认                                       │
│           例: 操作完成、验证通过、进程启动成功                   │
│                                                             │
└─────────────────────────────────────────────────────────────┘
```

### 2. 日志级别使用建议

#### 当前问题

部分位置日志级别使用不当：

| 位置 | 当前 | 建议 | 原因 |
|------|------|------|------|
| 配置验证警告 | Warning | Info | 非关键配置项缺失不应作为警告 |
| 进程扫描未找到 | Warning | Info | 未找到进程是正常情况 |
| 重试操作 | 无日志 | Warning | 重试应该记录 |

#### 建议的日志级别使用规范

```
Error   - 操作失败，影响功能正常使用
Warning - 潜在问题，但不影响主要功能
Info    - 正常操作流程信息
Success - 操作成功确认
```

### 3. 建议添加的日志模式

#### 3.1 操作开始/结束模式

```go
// 长时间操作应记录开始和结束
output.Info("开始执行 %s...", operationName)
// ... 执行操作
output.Success("%s 完成", operationName)
```

#### 3.2 重试操作模式

```go
for attempt := 1; attempt <= maxRetries; attempt++ {
    if attempt > 1 {
        output.Warning("重试 %s (第 %d/%d 次)", operation, attempt, maxRetries)
    }
    // ... 执行操作
}
```

#### 3.3 条件分支模式

```go
if condition {
    output.Info("检测到 %s，执行 %s", conditionDesc, action)
} else {
    output.Info("未检测到 %s，跳过 %s", conditionDesc, action)
}
```

#### 3.4 API 请求模式

```go
func (c *Client) doRequest(ctx context.Context, method, endpoint string, ...) error {
    output.Info("API 请求: %s %s", method, endpoint)
    
    resp, err := c.executeRequest(ctx, method, endpoint, body)
    if err != nil {
        output.Error("API 请求失败: %s - %v", endpoint, err)
        return err
    }
    
    if resp.StatusCode >= 400 {
        output.Warning("API 响应异常: %s (状态码: %d)", endpoint, resp.StatusCode)
    }
    
    return parseResponse(resp, result)
}
```

---

## 五、改进建议

### 1. 短期改进（优先级排序）

| 优先级 | 改进项 | 涉及文件 | 工作量 |
|--------|--------|----------|--------|
| 高 | 为 API 请求添加日志 | `internal/api/http.go` | 小 |
| 高 | 为 start 命令添加进度日志 | `cmd/start.go` | 小 |
| 中 | 为配置加载添加日志 | `internal/config/loader.go` | 小 |
| 中 | 修正日志级别使用 | 多个文件 | 中 |
| 低 | 为监控命令添加开始日志 | `cmd/monitor.go` | 小 |

### 2. 长期改进

| 改进项 | 说明 | 收益 |
|--------|------|------|
| 结构化日志 | 引入 zap/zerolog 等库 | 便于日志分析和过滤 |
| 日志级别配置 | 支持通过配置控制日志级别 | 灵活控制输出详细程度 |
| 日志输出目标 | 支持输出到文件 | 便于问题排查和审计 |
| 日志轮转 | 支持日志文件轮转 | 防止日志文件过大 |

### 3. 实施建议

#### 第一阶段：补充关键日志

1. 在 `internal/api/http.go` 中添加请求日志
2. 在 `cmd/start.go` 中添加操作进度日志
3. 修正不当的日志级别使用

#### 第二阶段：完善日志覆盖

1. 为配置加载添加日志
2. 为进程管理添加详细日志
3. 统一日志格式和风格

#### 第三阶段：增强日志能力

1. 评估是否需要结构化日志
2. 添加日志级别配置支持
3. 添加日志文件输出支持

---

## 六、总结

### 关键发现

1. **`internal/log/formatter.go`** 是专用模块，仅用于 `logs` 命令展示 Mihomo 内核日志
2. **`internal/output`** 是主要日志输出模块，提供统一的颜色化输出
3. **API 层缺少日志**：`internal/api` 目录完全没有日志记录
4. **部分命令缺少进度日志**：start、stop 等命令缺少操作开始提示

### 改进优先级

```
高优先级: API 请求日志 → start 命令进度日志
中优先级: 配置加载日志 → 日志级别修正
低优先级: 监控命令日志 → 结构化日志
```

### 预期效果

补充日志后，用户将能够：
- 清晰了解 API 请求状态
- 掌握命令执行进度
- 快速定位问题原因
- 获得更好的使用体验
