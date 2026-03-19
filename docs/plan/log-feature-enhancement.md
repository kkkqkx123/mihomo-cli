# 日志功能增强设计文档

## 概述

本文档描述了对 mihomo-cli 日志功能的增强设计，包括日志过滤、统计、搜索和持久化等功能。

## 设计原则

1. **无状态设计**：CLI 工具不持久化日志数据，所有日志从 Mihomo 内核实时获取
2. **非阻塞设计**：所有操作不阻塞主流程，支持 Ctrl+C 优雅退出
3. **文件导向**：所有导出操作的最终结果都是文件，不涉及外部服务
4. **简单实用**：聚焦核心功能，避免过度设计

## 设计目标

1. **日志过滤**：支持按级别、关键词、正则表达式过滤实时日志
2. **日志统计**：提供日志数量、错误率、常见错误等基本统计
3. **日志搜索**：在收集的日志中搜索关键词或正则表达式
4. **日志导出**：将收集的日志导出为 JSON、TXT 或 CSV 文件

## 整体架构

### 模块结构

```
cmd/logs.go (命令层)
├── logs view      # 查看实时日志（带过滤）
├── logs stats     # 统计日志信息
├── logs search    # 搜索日志
└── logs export    # 导出日志到文件

internal/log/
├── formatter.go    # 格式化（已存在）
├── filter.go       # 过滤逻辑（新增）
├── collector.go    # 日志收集（新增）
├── statistics.go   # 统计分析（新增）
└── storage.go      # 文件导出（新增）
```

### 数据流

```
Mihomo 内核 (WebSocket /logs)
    ↓
API 客户端层 (internal/api/websocket.go)
    ↓
日志收集器 (LogCollector) - 内存缓存
    ↓
过滤器/搜索器/统计器/导出器
    ↓
终端输出 或 文件输出
```

### 核心设计

- **日志收集器**：临时缓存日志，支持指定时间范围收集
- **过滤器**：在日志输出前进行过滤，不修改原始数据
- **统计器**：对收集的日志进行统计分析
- **导出器**：将日志写入文件，支持多种格式

## 功能模块设计

### 1. 日志过滤

#### 核心功能

- **级别过滤**：支持指定日志级别（silent/error/warning/info/debug）
- **关键词过滤**：支持关键词匹配（AND 逻辑）
- **排除关键词**：排除包含特定关键词的日志
- **正则表达式**：使用正则表达式匹配

#### 过滤器结构

- `LogFilter`：包含过滤条件的结构体
- `Match()` 方法：判断单条日志是否匹配
- `FilterLogs()` 函数：批量过滤日志列表

#### 级别优先级

```
silent (0) < error (1) < warning (2) < info (3) < debug (4)
```

#### 命令行接口

```bash
# 查看错误日志
mihomo-cli logs --level error

# 查看包含关键词的日志
mihomo-cli logs --keyword "proxy"

# 排除特定日志
mihomo-cli logs --exclude "keepalive"

# 使用正则表达式
mihomo-cli logs --regex "error"
```

### 2. 日志统计

#### 核心功能

- **总体统计**：日志总数、各级别数量
- **错误率计算**：错误日志占总日志的百分比
- **常见错误统计**：统计出现频率最高的错误信息（Top 10）

#### 统计指标

- `TotalCount`：总日志数
- `LevelCount`：各级别日志数量
- `ErrorRate`：错误率（百分比）
- `TopErrors`：出现频率最高的错误（Top 10）

#### 命令行接口

```bash
# 统计最近 5 分钟的日志
mihomo-cli logs stats --duration 5m

# 统计错误日志
mihomo-cli logs stats --level error

# JSON 格式输出
mihomo-cli logs stats -o json
```

#### 输出示例

```
日志统计信息
=============
总日志数: 1523
错误率: 2.35%

按级别统计:
  ERROR: 35
  WARNING: 87
  INFO: 1023
  DEBUG: 378

常见错误 (Top 10):
  1. connection refused (12 次)
  2. timeout (8 次)
  3. invalid certificate (6 次)
```

### 3. 日志搜索

#### 核心功能

- **关键词搜索**：支持单个或多个关键词搜索
- **正则搜索**：使用正则表达式搜索
- **级别过滤**：可指定只搜索特定级别的日志
- **时间范围**：可指定搜索的时间范围

#### 搜索器结构

- `Searcher`：日志搜索器
- `SearchQuery`：搜索查询条件
- `SearchResult`：搜索结果
- `Search()` 方法：执行搜索

#### 搜索逻辑

1. 先按时间范围收集日志
2. 按级别过滤（可选）
3. 在过滤后的日志中搜索关键词或正则表达式

#### 命令行接口

```bash
# 搜索关键词
mihomo-cli logs search --keyword "proxy"

# 使用正则表达式
mihomo-cli logs search --regex "error"

# 搜索特定级别
mihomo-cli logs search --keyword "error" --level error

# 搜索最近 10 分钟的日志
mihomo-cli logs search --keyword "warning" --duration 10m
```

### 4. 日志导出

#### 核心功能

- **多格式导出**：支持 JSON、TXT、CSV 格式
- **级别过滤**：导出时按级别过滤
- **时间范围**：导出指定时间范围的日志

#### 导出器结构

- `LogExporter`：日志导出器
- `ExportFormat`：导出格式枚举（JSON/TXT/CSV）
- `ExportToFile()`：导出日志到文件

#### 导出格式

**JSON 格式**：
```json
[
  {
    "type": "error",
    "payload": "connection refused"
  }
]
```

**TXT 格式**：
```
[ERROR] connection refused
[INFO] proxy started
```

**CSV 格式**：
```csv
Level,Message
error,connection refused
info,proxy started
```

#### 命令行接口

```bash
# 导出为 JSON
mihomo-cli logs export -o logs.json --format json

# 导出为 CSV
mihomo-cli logs export -o logs.csv --format csv

# 只导出错误日志
mihomo-cli logs export -o errors.json --level error

# 导出最近 1 小时的日志
mihomo-cli logs export -o recent.json --duration 1h
```

## 数据结构

### LogInfo

保持现有的 `LogInfo` 结构，无需扩展：

```go
type LogInfo struct {
    LogType string `json:"type"`    // info, warning, error, debug, silent
    Payload string `json:"payload"`
}
```

### 日志收集器

- `LogCollector`：日志收集管理器，临时缓存日志
- `Add()`：添加单条日志
- `GetLogs()`：获取所有日志
- `CollectWithDuration()`：收集指定时间的日志

## 配置

保持现有的 CLI 工具日志配置，无需新增配置项。所有功能通过命令行参数控制。

## 实现优先级

### Phase 1 - 基础过滤（优先级：高）

- 实现日志过滤器模块 (`internal/log/filter.go`)
- 扩展 `logs` 命令支持过滤参数
- 实现日志收集器模块 (`internal/log/collector.go`)

**目标**：用户可以按级别和关键词过滤查看实时日志

### Phase 2 - 统计功能（优先级：中）

- 实现日志统计器模块 (`internal/log/statistics.go`)
- 添加 `logs stats` 命令

**目标**：用户可以查看日志统计信息

### Phase 3 - 搜索功能（优先级：中）

- 实现日志搜索器模块 (`internal/log/searcher.go`)
- 添加 `logs search` 命令

**目标**：用户可以快速定位特定的日志信息

### Phase 4 - 导出功能（优先级：高）

- 实现日志导出器模块 (`internal/log/storage.go`)
- 添加 `logs export` 命令

**目标**：用户可以将日志导出为文件

## 技术要点

### 内存管理

- 限制日志收集器的最大缓存数量（如 10,000 条）
- 使用切片缓存日志，避免频繁内存分配
- 收集完成后及时释放内存

### 错误处理

- 连接断开：显示错误信息并退出
- 解析错误：容错处理，记录解析失败的原始数据
- 文件写入错误：显示错误信息并退出

### 兼容性

- Mihomo 版本：兼容不同版本的 Mihomo API
- 操作系统：跨平台路径处理
- 编码格式：UTF-8 统一编码

## 使用场景

### 场景 1：问题排查

```bash
# 查看错误日志
mihomo-cli logs --level error

# 搜索特定错误
mihomo-cli logs search --keyword "connection refused"

# 导出错误日志
mihomo-cli logs export -o errors.json --level error
```

### 场景 2：日常运维

```bash
# 实时监控日志
mihomo-cli logs --follow

# 过滤无关日志
mihomo-cli logs --exclude "keepalive"

# 导出日志备份
mihomo-cli logs export -o backup.json
```

## 测试策略

### 单元测试

- 过滤器测试：测试各种过滤条件
- 统计器测试：验证统计结果的准确性
- 搜索器测试：测试搜索功能
- 导出器测试：测试文件写入功能

### 集成测试

- 端到端测试：测试完整的日志收集、过滤、导出流程
- API 测试：测试与 Mihomo API 的交互

## 限制

### 已知限制

1. **Mihomo 内核限制**：
   - 不支持从 Mihomo 内核获取历史日志
   - 日志在内存中，重启即清空
   - 不支持日志文件输出

2. **CLI 工具限制**：
   - 日志收集受限于缓存大小
   - 搜索功能依赖于内存中的日志
   - 长时间运行可能占用较多内存
   - 所有日志都是临时收集，命令执行完即释放

### 缓解措施

1. **限制缓存大小**：设置合理的日志缓存上限（如 10,000 条）
2. **用户提示**：在长时间收集时提示用户

## 参考资料

- Mihomo API 文档：`docs/spec/mihono-api.md`
- 当前日志实现：`cmd/logs.go`
- WebSocket 客户端：`internal/api/websocket.go`
- 输出模块：`internal/output/`