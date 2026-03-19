# 命令分类分析

本文档整理了项目中所有命令的实现方式，按照调用类型进行分类，并分析了 Mihomo 进程存在性验证的问题。

## 一、调用 Mihomo 内核 API 的命令

这些命令通过 `internal/api` 包中的客户端调用 Mihomo 内核的 RESTful API。

| 命令 | 文件位置 | 主要 API 调用 |
|------|----------|---------------|
| `mode get/set` | `cmd/mode.go` | `GetMode()`, `SetMode()` |
| `proxy list/switch/test/auto/unfix/current` | `cmd/proxy.go` | `ListProxies()`, `SwitchProxy()`, `GetProxy()` |
| `cache clear fakeip/dns` | `cmd/cache.go` | `FlushFakeIP()`, `FlushDNS()` |
| `conn list/close/close-all` | `cmd/conn.go` | `GetConnections()`, `CloseConnection()`, `CloseAllConnections()` |
| `dns query/config` | `cmd/dns.go` | `QueryDNS()`, `GetDNSConfig()` |
| `rule list/provider/disable/enable` | `cmd/rule.go` | `GetRules()`, `ListRuleProviders()`, `DisableRules()`, `EnableRules()` |
| `monitor traffic/memory` | `cmd/monitor.go` | `GetTraffic()`, `GetMemory()` |
| `logs view/stats/search/export` | `cmd/logs.go` | `StreamLogs()` |
| `version kernel` | `cmd/version.go` | `GetVersion()` |
| `geoip update` | `cmd/geoip.go` | `UpdateGeo()` |
| `sub update` | `cmd/sub.go` | `ListProviders()`, `UpdateProvider()` |
| `mihomo patch/reload` | `cmd/mihomo.go` | `PatchConfig()`, `ReloadConfig()` |
| `backup restore` (部分) | `cmd/backup.go` | `ReloadConfig()` (可选，用于重载配置) |

### 错误处理方式

这些命令都使用 `errors.WrapAPIError()` 包装 API 错误，会自动将 API 错误转换为 CLI 错误：

```go
// 示例：cmd/mode.go
modeInfo, err := client.GetMode(cmd.Context())
if err != nil {
    return errors.WrapAPIError("获取模式失败", err)
}
```

## 二、直接调用系统 API 的命令

这些命令直接调用操作系统 API 或系统服务。

| 命令 | 文件位置 | 系统调用类型 |
|------|----------|-------------|
| `service start/stop/install/uninstall/status` | `cmd/service.go` | Windows 服务管理 API (通过 `internal/service` 包) |
| `sysproxy get/set` | `cmd/sysproxy.go` | Windows 注册表 API (通过 `internal/sysproxy` 包) |
| `start` | `cmd/start.go` | 进程启动、信号处理 (通过 `internal/mihomo` 包) |
| `stop` | `cmd/start.go` | 进程终止 (通过 `internal/mihomo` 包) |
| `status` | `cmd/start.go` | 进程状态查询 (通过 `internal/mihomo` 包) |
| `ps` | `cmd/ps.go` | 进程扫描 (通过 `internal/mihomo.ScanMihomoProcesses()`) |
| `cleanup` | `cmd/cleanup.go` | PID 文件清理 (通过 `internal/mihomo.CleanupPIDFiles()`) |
| `diagnose route/network` | `cmd/diagnose.go` | 路由表诊断 (通过 `internal/system` 包) |
| `system status/cleanup/validate/fix/snapshot` | `cmd/system.go` | 系统配置管理 (通过 `internal/system` 包) |
| `recovery detect/execute/status` | `cmd/recovery.go` | 系统恢复 (通过 `internal/recovery` 包) |
| `operation query/clear/prune` | `cmd/operation.go` | 操作记录管理 (通过 `internal/operation` 包) |

### 特点

- 部分命令需要管理员权限（如 `sysproxy set`、`system cleanup`、`recovery execute`）
- 通过 `internal/util.IsAdmin()` 检查权限
- 错误处理使用 `pkgerrors.ErrService()` 包装

## 三、本地文件操作命令

这些命令只操作本地配置文件，不涉及 API 调用或系统 API。

| 命令 | 文件位置 | 操作类型 |
|------|----------|----------|
| `config init/show/set` | `cmd/config.go` | CLI 配置文件管理 (通过 `internal/config` 包) |
| `backup create/list/delete/prune` | `cmd/backup.go` | 配置备份管理 (通过 `internal/config.BackupHandler`) |
| `history` | `cmd/history.go` | 命令历史记录 (通过 `internal/history` 包) |
| `version` | `cmd/version.go` | 显示 CLI 版本 (直接输出构建信息) |
| `geoip status` | `cmd/geoip.go` | 检查 GeoIP 文件状态 (通过 `os.Stat()` 检查文件) |

### 特点

- 不依赖 Mihomo 进程运行状态
- 错误处理使用 `pkgerrors.ErrConfig()` 包装

## 四、Mihomo 进程存在性验证问题分析

### 当前状态

1. **没有统一的进程存在性验证**：调用 Mihomo API 的命令在执行时，如果 Mihomo 未运行，会收到连接错误
2. **错误信息不够友好**：用户看到的是 "API 连接失败"，而不是 "Mihomo 进程未运行"
3. **错误处理分散**：每个命令都通过 `errors.WrapAPIError()` 处理，但无法区分"进程未运行"和"网络问题"

### 错误处理流程

```
命令调用 API
    ↓
API 请求失败 (连接被拒绝/超时)
    ↓
api.HTTPClient 返回 *api.APIError (Code: ErrAPIConnection)
    ↓
errors.WrapAPIError() 转换为 *errors.CLIError (Code: ExitNetwork)
    ↓
用户看到: "API 连接失败"
```

### 问题示例

当 Mihomo 未运行时，用户执行 `mihomo-cli mode get` 会看到：

```
错误: 获取模式失败: API 连接失败: dial tcp 127.0.0.1:9090: connect: connection refused
```

这个错误信息没有明确告诉用户：
- Mihomo 进程是否在运行
- 如何解决这个问题

## 五、改进建议

### 推荐方案：改进错误信息

在 `internal/errors/api.go` 中改进 `WrapAPIError` 函数，检测连接错误时返回更友好的提示：

```go
// WrapAPIError 包装 API 错误，自动转换为 CLI 错误
func WrapAPIError(message string, err error) *errors.CLIError {
    if err == nil {
        return nil
    }

    // 如果已经是 CLI 错误，直接返回
    if cliErr := errors.GetCLIError(err); cliErr != nil {
        return errors.WrapError(message, cliErr)
    }

    // 检测是否是连接错误，提供更友好的提示
    if apiErr, ok := err.(*api.APIError); ok && api.IsAPIConnectionError(apiErr) {
        return errors.ErrService(
            message+": Mihomo 进程未运行或 API 地址配置错误\n"+
                "  提示: 请先启动 Mihomo: mihomo-cli start\n"+
                "  或检查 API 地址配置: mihomo-cli config show",
            err,
        )
    }

    // 如果是 API 错误，转换后包装
    if apiErr, ok := err.(*api.APIError); ok {
        cliErr := APIErrorToCLIError(apiErr)
        if cliErr != nil {
            return errors.WrapError(message, cliErr)
        }
    }

    // 其他错误，使用默认包装
    return errors.WrapError(message, err)
}
```

### 方案优点

1. **改动最小**：只需修改一处错误处理逻辑
2. **向后兼容**：不影响现有命令结构
3. **统一处理**：所有 API 命令自动获得改进的错误信息
4. **用户友好**：提供明确的错误原因和解决建议

### 改进后的错误信息示例

```
错误: 获取模式失败: Mihomo 进程未运行或 API 地址配置错误
  提示: 请先启动 Mihomo: mihomo-cli start
  或检查 API 地址配置: mihomo-cli config show
```

## 六、其他可选方案

### 方案二：在 API 客户端层添加预检查

在 `internal/api/client.go` 中添加健康检查方法：

```go
// IsProcessRunning 检查 Mihomo 进程是否运行
func (c *Client) IsProcessRunning(ctx context.Context) bool {
    ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
    defer cancel()
    
    _, err := c.GetVersion(ctx)
    return err == nil
}
```

然后在需要的地方调用：

```go
if !client.IsProcessRunning(cmd.Context()) {
    return errors.ErrService("Mihomo 进程未运行，请先启动: mihomo-cli start", nil)
}
```

**缺点**：需要在每个命令中添加检查，增加代码重复。

### 方案三：使用命令中间件

在 `cmd/root.go` 中添加命令中间件，自动为需要 API 的命令添加预检查：

```go
// 标记需要 Mihomo 运行的命令
cmd.Annotations = map[string]string{"requiresMihomo": "true"}

// 在 PersistentPreRunE 中检查
func preRun(cmd *cobra.Command, args []string) error {
    if cmd.Annotations["requiresMihomo"] == "true" {
        // 检查 Mihomo 是否运行
    }
    // ...
}
```

**缺点**：需要修改所有需要 API 的命令，改动较大。

## 七、总结

| 分类 | 命令数量 | 特点 |
|------|----------|------|
| 调用 Mihomo API | 14 个命令组 | 依赖 Mihomo 进程运行，错误处理统一 |
| 调用系统 API | 11 个命令组 | 部分需要管理员权限，平台相关 |
| 本地文件操作 | 5 个命令组 | 不依赖外部状态，最稳定 |

推荐采用**方案一**改进错误信息，以最小的改动获得最佳的用户体验提升。
