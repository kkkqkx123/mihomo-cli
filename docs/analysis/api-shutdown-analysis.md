# Mihomo API 层面关闭操作分析

## 1. 问题背景

### 1.1 当前问题

在 Windows 平台上，通过进程信号（SIGINT/SIGTERM）优雅关闭 Mihomo 内核存在根本性限制：

1. **Windows 没有跨进程信号机制**：不像 POSIX 系统有 `kill(pid, SIGTERM)` 这样的跨进程信号
2. **GenerateConsoleCtrlEvent 的限制**：只能向共享同一控制台的进程发送事件
3. **进程无控制台**：当 Mihomo 进程的 stdout/stderr 被重定向时，它没有控制台，无法接收控制台事件

### 1.2 当前实现

当前 `mihomo-cli stop` 的实现流程：

```
1. 尝试发送 GenerateConsoleCtrlEvent (失败)
2. 回退到 proc.Kill() 强制终止
3. 进程被强制终止，可能残留系统配置
```

## 2. Mihomo 内核现有机制分析

### 2.1 信号处理

Mihomo 内核 (`main.go:189-202`) 监听以下信号：

```go
termSign := make(chan os.Signal, 1)
hupSign := make(chan os.Signal, 1)
signal.Notify(termSign, syscall.SIGINT, syscall.SIGTERM)
signal.Notify(hupSign, syscall.SIGHUP)
for {
    select {
    case <-termSign:
        return  // 触发 defer executor.Shutdown()
    case <-hupSign:
        // 重新加载配置
    }
}
```

### 2.2 Shutdown 函数

`executor.Shutdown()` (`hub/executor/executor.go:535-541`) 执行清理：

```go
func Shutdown() {
    listener.Cleanup()           // 清理监听器
    tproxy.CleanupTProxyIPTables() // 清理 TProxy iptables 规则
    resolver.StoreFakePoolState()   // 存储 fake-ip 池状态
    log.Warnln("Mihomo shutting down")
}
```

### 2.3 现有 API 端点

Mihomo 已有的 API 端点：

| 端点 | 方法 | 功能 |
|------|------|------|
| `/configs` | PATCH | 重载配置 |
| `/restart` | POST | 重启内核 |
| `/proxies/{name}` | PUT | 切换代理 |
| `/connections` | DELETE | 关闭所有连接 |

**注意**：没有 `/shutdown` 或 `/stop` 端点。

### 2.4 Restart 端点分析

`/restart` 端点 (`hub/route/restart.go:24-43`) 的实现：

```go
func restart(w http.ResponseWriter, r *http.Request) {
    execPath, err := os.Executable()
    // ...
    render.JSON(w, r, render.M{"status": "ok"})
    if f, ok := w.(http.Flusher); ok {
        f.Flush()
    }
    go restartExecutable(execPath)  // 异步重启
}

func restartExecutable(execPath string) {
    executor.Shutdown()  // 执行清理
    // ... 启动新进程
    os.Exit(0)
}
```

关键点：
- 调用 `executor.Shutdown()` 执行清理
- 然后启动新进程并退出当前进程

## 3. API 关闭方案设计

### 3.1 方案一：添加 /shutdown 端点（推荐）

**实现方式**：在 Mihomo 内核添加 `/shutdown` API 端点

**修改文件**：`mihomo-1.19.21/hub/route/shutdown.go`

```go
package route

import (
    "os"
    
    "github.com/metacubex/mihomo/hub/executor"
    "github.com/metacubex/chi"
    "github.com/metacubex/chi/render"
    "github.com/metacubex/http"
)

func shutdownRouter() http.Handler {
    r := chi.NewRouter()
    r.Post("/", shutdown)
    return r
}

func shutdown(w http.ResponseWriter, r *http.Request) {
    render.JSON(w, r, render.M{"status": "ok"})
    if f, ok := w.(http.Flusher); ok {
        f.Flush()
    }
    go func() {
        executor.Shutdown()
        os.Exit(0)
    }()
}
```

**注册路由**：在 `hub/route/server.go` 的 `router()` 函数中添加：

```go
r.Mount("/shutdown", shutdownRouter())
```

**优点**：
- 真正的优雅关闭，执行所有清理操作
- 跨平台一致的行为
- 不依赖进程信号机制
- 可以通过 API 密钥验证权限

**缺点**：
- 需要修改 Mihomo 内核代码
- 如果 API 服务异常，无法通过 API 关闭

### 3.2 方案二：利用现有 /restart 端点

**实现方式**：调用 `/restart` 后立即终止新进程

**流程**：
1. 调用 `POST /restart`
2. 等待响应返回
3. 查找新启动的进程
4. 终止新进程

**优点**：
- 不需要修改 Mihomo 内核
- 利用现有的清理机制

**缺点**：
- 实现复杂，需要追踪新进程
- 可能存在竞态条件
- 不够优雅

### 3.3 方案三：混合方案（推荐）

**实现方式**：优先使用 API 关闭，失败时回退到进程终止

**流程**：
```
1. 尝试调用 POST /shutdown (如果存在)
   ├─ 成功：等待进程退出
   └─ 失败：继续下一步

2. 尝试发送进程信号 (Unix: SIGTERM, Windows: GenerateConsoleCtrlEvent)
   ├─ 成功：等待进程退出
   └─ 失败：继续下一步

3. 强制终止进程 (proc.Kill())
```

**优点**：
- 兼容现有 Mihomo 版本
- 优先使用优雅关闭
- 有可靠的回退机制

## 4. mihomo-cli 实现建议

### 4.1 API 客户端扩展

在 `internal/api/` 添加 shutdown 方法：

```go
// internal/api/shutdown.go
package api

import "context"

// Shutdown 通过 API 关闭 Mihomo 内核
func (c *Client) Shutdown(ctx context.Context) error {
    var result struct {
        Status string `json:"status"`
    }
    return c.Post(ctx, "/shutdown", nil, nil, &result)
}
```

### 4.2 StopProcessByPID 修改

修改 `internal/mihomo/manager.go` 的 `StopProcessByPID` 函数：

```go
func StopProcessByPID(pid int) error {
    // 方案一：尝试通过 API 关闭
    if err := tryAPIShutdown(); err == nil {
        // 等待进程退出
        if waitForProcessExit(pid, 10*time.Second) {
            return nil
        }
    }
    
    // 方案二：尝试发送进程信号
    if err := SendGracefulSignal(proc); err == nil {
        if waitForProcessExit(pid, 10*time.Second) {
            return nil
        }
    }
    
    // 方案三：强制终止
    return forceKillProcess(proc, pid)
}
```

### 4.3 配置选项

在 `config.toml` 添加配置：

```toml
[mihomo]
# 关闭方式优先级：api > signal > kill
shutdown_method = "api"  # "api", "signal", "kill", "auto"
```

## 5. 实施建议

### 5.1 短期方案（无需修改内核）

1. 保持当前的进程终止实现
2. 在文档中说明 Windows 平台的限制
3. 建议用户在需要优雅关闭时手动调用 Mihomo API

### 5.2 中期方案（修改内核）

1. Fork Mihomo 内核，添加 `/shutdown` 端点
2. 修改 mihomo-cli 优先使用 API 关闭
3. 保留进程终止作为回退方案

### 5.3 长期方案（贡献上游）

1. 向 Mihomo/Mihomo 项目提交 PR，添加 `/shutdown` 端点
2. 等待上游合并后，mihomo-cli 可以依赖官方版本

## 6. 风险评估

### 6.1 API 关闭的风险

| 风险 | 影响 | 缓解措施 |
|------|------|----------|
| API 服务异常 | 无法关闭 | 回退到进程终止 |
| 网络问题 | 请求失败 | 设置超时，回退到进程终止 |
| 权限问题 | 请求被拒绝 | 检查 API 密钥配置 |

### 6.2 进程终止的风险

| 风险 | 影响 | 缓解措施 |
|------|------|----------|
| TUN 模式残留 | 网络异常 | 启动时检查并清理 |
| TProxy 规则残留 | 路由异常 | 启动时检查并清理 |
| 注册表残留 | 系统代理异常 | 启动时检查并清理 |

## 7. 结论

**推荐方案**：混合方案（方案三）

1. **优先使用 API 关闭**：如果 Mihomo 内核支持 `/shutdown` 端点
2. **回退到进程信号**：如果 API 不可用
3. **最终回退到强制终止**：如果信号发送失败

**实施步骤**：

1. **Phase 1**：修改 mihomo-cli，添加 API 关闭支持（检测 `/shutdown` 端点是否存在）
2. **Phase 2**：Fork Mihomo 内核，添加 `/shutdown` 端点
3. **Phase 3**：测试并验证跨平台行为
4. **Phase 4**：考虑向上游贡献代码

**预期效果**：

- Windows 平台：通过 API 实现真正的优雅关闭
- Unix 平台：保持现有的 SIGTERM 行为
- 兼容性：支持旧版本 Mihomo（回退到进程终止）
