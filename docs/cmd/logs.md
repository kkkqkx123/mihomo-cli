# 日志管理命令 (logs)

日志管理命令用于查看、统计、搜索和导出 Mihomo 内核的日志。

## 命令列表

### logs view - 实时查看 Mihomo 日志

实时查看 Mihomo 内核的日志输出，支持按级别、关键词、正则表达式过滤。

**语法：**
```bash
mihomo-cli logs view
```

**选项：**
- `-f, --follow` - 持续跟踪日志输出（默认开启）
- `--level` - 日志级别过滤（silent/error/warning/info/debug）
- `--keyword` - 包含关键词过滤（可多次使用，AND 逻辑）
- `--exclude` - 排除关键词过滤（可多次使用）
- `--regex` - 正则表达式过滤

**示例：**
```bash
# 实时查看所有日志
mihomo-cli logs view

# 只查看错误日志
mihomo-cli logs view --level error

# 查看包含关键词的日志
mihomo-cli logs view --keyword "proxy"

# 排除某些关键词
mihomo-cli logs view --exclude "keepalive"

# 使用正则表达式过滤
mihomo-cli logs view --regex "error"
```

### logs stats - 统计日志信息

统计 Mihomo 日志信息，包括日志总数、错误率、常见错误等。

**语法：**
```bash
mihomo-cli logs stats
```

**选项：**
- `--duration` - 收集日志的时间范围（如 30s, 5m, 1h，默认 1m）
- `-o, --output` - 输出格式（table/json）

**示例：**
```bash
# 统计最近 1 分钟的日志
mihomo-cli logs stats

# 统计最近 5 分钟的日志
mihomo-cli logs stats --duration 5m

# JSON 格式输出
mihomo-cli logs stats -o json
```

### logs search - 搜索日志

在收集的日志中搜索关键词或正则表达式。

**语法：**
```bash
mihomo-cli logs search
```

**选项：**
- `--keyword` - 搜索关键词（可多次使用，AND 逻辑）
- `--regex` - 正则表达式搜索
- `--level` - 日志级别过滤
- `--duration` - 收集日志的时间范围（如 30s, 5m, 1h，默认 1m）
- `-o, --output` - 输出格式（table/json）

**示例：**
```bash
# 搜索包含关键词的日志
mihomo-cli logs search --keyword "proxy"

# 使用正则表达式搜索
mihomo-cli logs search --regex "error"

# 组合条件搜索
mihomo-cli logs search --keyword "error" --level error

# 搜索最近 10 分钟的日志
mihomo-cli logs search --keyword "warning" --duration 10m
```

### logs export - 导出日志到文件

将收集的日志导出为 JSON、TXT 或 CSV 文件。

**语法：**
```bash
mihomo-cli logs export -o <文件>
```

**参数：**
- `-o, --output` - 输出文件路径（必需）

**选项：**
- `--format` - 导出格式（json/txt/csv，默认 json）
- `--level` - 日志级别过滤
- `--duration` - 收集日志的时间范围（如 30s, 5m, 1h，默认 1m）

**示例：**
```bash
# 导出为 JSON 文件
mihomo-cli logs export -o logs.json --format json

# 导出为 CSV 文件
mihomo-cli logs export -o logs.csv --format csv

# 只导出错误日志
mihomo-cli logs export -o errors.json --level error

# 导出最近 1 小时的日志
mihomo-cli logs export -o recent.json --duration 1h
```

## 日志级别说明

| 级别 | 说明 |
|------|------|
| silent | 静默模式，不显示日志 |
| error | 错误信息 |
| warning | 警告信息 |
| info | 一般信息 |
| debug | 调试信息 |

## 时间格式说明

时间范围支持以下格式：
- `30s` - 30 秒
- `5m` - 5 分钟
- `1h` - 1 小时

## 注意事项

1. 日志查看使用 WebSocket 连接，如果连接失败会自动降级为 HTTP 轮询
2. 日志收集有最大数量限制，超出限制会丢弃最早的日志
3. 导出日志时，文件路径可以是相对路径或绝对路径
4. 搜索和统计功能会先收集日志，然后再进行处理
