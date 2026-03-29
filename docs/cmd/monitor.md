# 监控管理命令 (monitor)

监控管理命令用于监控 Mihomo 的流量和内存使用情况。

## 命令列表

### monitor traffic - 获取流量统计

获取实时流量统计信息，包括上传速度、下载速度、总上传流量和总下载流量。

**语法：**
```bash
mihomo-cli monitor traffic
```

**选项：**
- `-w, --watch` - 持续刷新显示实时流量
- `-i, --interval` - 刷新间隔（秒，仅用于 --watch 模式，默认 1）

**示例：**
```bash
# 单次查询流量统计
mihomo-cli monitor traffic

# 持续监控流量
mihomo-cli monitor traffic --watch

# 设置刷新间隔
mihomo-cli monitor traffic --watch --interval 2

# JSON 格式输出
mihomo-cli monitor traffic -o json
```

### monitor memory - 获取内存使用

获取当前内存使用情况。

**语法：**
```bash
mihomo-cli monitor memory
```

**选项：**
- `-w, --watch` - 持续刷新显示内存使用
- `-i, --interval` - 刷新间隔（秒，仅用于 --watch 模式，默认 1）

**示例：**
```bash
# 单次查询内存使用
mihomo-cli monitor memory

# 持续监控内存
mihomo-cli monitor memory --watch

# 设置刷新间隔
mihomo-cli monitor memory --watch --interval 2

# JSON 格式输出
mihomo-cli monitor memory -o json
```

## 监控模式说明

### 单次查询模式
- 默认模式，执行一次查询后退出
- 适合获取当前状态的快照
- 支持 JSON 格式输出

### Watch 模式
- 持续监控，定期刷新显示
- 使用 `-w` 或 `--watch` 参数启用
- 使用 `-i` 或 `--interval` 参数设置刷新间隔
- 按 Ctrl+C 停止监控
- 优先使用 WebSocket 连接，失败时自动降级为 HTTP 轮询

## 流量统计信息

流量统计包含以下信息：
- 上传速度 - 当前上传速率
- 下载速度 - 当前下载速率
- 总上传流量 - 累计上传流量
- 总下载流量 - 累计下载流量
- 累计流量 - 总流量（上传+下载）

## 内存使用信息

内存使用包含以下信息：
- 已使用内存 - 当前使用的内存量
- 内存使用率 - 内存使用百分比

## 注意事项

1. Watch 模式会持续运行，直到手动停止
2. WebSocket 连接失败时会自动降级为 HTTP 轮询模式
3. 流量统计是实时的，但可能会有轻微延迟
4. 内存使用信息反映的是 Mihomo 进程的内存占用
