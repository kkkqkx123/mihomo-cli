# Linux 平台 start 及进程管理功能适配分析与改进方案

## 一、问题总览

当前 `start` 命令在 Linux 上完全无法启动内核，直接报错：

```
fork/exec build/mihomo: operation not permitted
```

经过深入分析代码和环境测试，发现以下问题：

---

## 二、关键问题分析

### 问题 1（P0 致命）：`Setsid + Setpgid` 组合导致 EPERM

**文件**：`internal/mihomo/daemon_linux.go:127-131`

```go
cmd.SysProcAttr = &syscall.SysProcAttr{
    Setsid:  true, // 创建新会话
    Setpgid: true, // 创建新进程组
}
```

**根因**：
- `setsid()` 系统调用要求调用者**不能**是进程组 leader（即 `PID != PGID`）
- 当从 shell 启动 mihomo-cli 时，shell 会为子进程创建新的进程组（`PGID = PID`），使 mihomo-cli 成为进程组 leader
- mihomo-cli fork 子进程时，子进程继承父进程的 `PGID`（此时 `PGID != 子进程 PID`，所以子进程**不是**进程组 leader）
- 但在 Go 的 `exec.Command().Start()` 内部，Go 运行时可能在 fork 后、exec 前调用了 `setpgid(0, 0)` 使子进程成为进程组 leader，导致后续 `setsid()` 失败

**验证结果**：

| 组合 | 结果 |
|------|------|
| `nil` | ✅ OK |
| `Setsid: true` only | ✅ OK |
| `Setpgid: true` only | ✅ OK |
| **`Setsid: true` + `Setpgid: true`** | ❌ **operation not permitted** |

**正确做法**：
- `Setsid: true` 已经隐含创建新进程组（新会话的第一个进程自动成为会话 leader 和进程组 leader），不需要额外设置 `Setpgid: true`
- 或者只用 `Setpgid: true` 创建新进程组（足以让子进程脱离父进程的进程组，不受终端关闭影响）

**影响范围**：`daemon_linux.go` 和 `daemon_darwin.go` 均有此问题。

---

### 问题 2（P1 严重）：Darwin `RedirectIO` 在子进程启动前关闭文件句柄

**文件**：`internal/mihomo/daemon_darwin.go:174-215`

```go
func (ddm *DarwinDaemonManager) RedirectIO(cmd *exec.Cmd, logFile string) error {
    // ...
    devNull, err := os.OpenFile("/dev/null", os.O_RDWR, 0)
    defer devNull.Close()  // ← 在 cmd.Start() 前就关闭了！
    cmd.Stdout = devNull
    cmd.Stderr = devNull
    // ...
}
```

**根因**：
- `defer devNull.Close()` 在函数返回时执行
- 但 `cmd.Start()` 在 `RedirectIO` 返回**之后**才调用
- 文件句柄在 `cmd.Start()` 前就被关闭，子进程继承的是无效的文件描述符

**对比 Linux 版本**：Linux 的 `RedirectIO` 正确地返回 `[]io.Closer`，在 `cmd.Start()` 之后才关闭。

---

### 问题 3（P2 中等）：`ExecutableResolver.resolveExplicit` 可能返回相对路径

**文件**：`internal/config/executable_resolver.go:100-121`

```go
func (r *ExecutableResolver) resolveExplicit(explicit string) (string, error) {
    p := explicit
    if !filepath.IsAbs(p) {
        p = filepath.Join(r.baseDir, p) // filepath.Join(".", "./build/mihomo") = "build/mihomo"
    }
    p = filepath.Clean(p)              // "build/mihomo" (仍是相对路径)
    // ...
    return filepath.Clean(p), nil
}
```

**根因**：
- `filepath.Join(".", "./build/mihomo")` 返回 `"build/mihomo"`（不是绝对路径）
- `filepath.Clean` 不会将相对路径转为绝对路径
- 结果：`exec.Command("build/mihomo", ...)` 使用相对路径，在某些场景下可能导致找不到文件

**影响**：
- `DaemonLauncher.GetWorkDir()` 返回 `filepath.Dir("build/mihomo")` = `"build"`（相对路径）
- `cmd.Dir = "build"` 可能导致子进程的工作目录不正确

---

### 问题 4（P2 中等）：启动前未清理过期的 State 文件

**文件**：`internal/mihomo/process_handler.go:94-98`

```go
// 启动前检查并清理残留配置
if err := ph.checkAndCleanupBeforeStart(cfg); err != nil {
    return nil, ...
}
```

`checkAndCleanupBeforeStart` 只调用 `system.NewSystemConfigManager().ValidateState()` 检查系统级残留（路由表、TUN 接口），**不清理 CLI 自身的过期 State 文件**。

当上次启动失败时，`state-*.json` 中 `stage: "failed"` 和 `pid: 0` 的残留文件不会被清理，可能导致后续启动逻辑异常。

---

### 问题 5（P3 低）：`ProcessLock` 在 `LifecycleManager.Start` 中获取但 `ProcessHandler.Start` 中也可能获取

**文件**：
- `lifecycle.go:84`：`lm.lock.Acquire()`
- `process_handler.go:122`：`lm.Start(ctx, cfg)` 会获取锁

`ProcessHandler.Start` 调用 `LifecycleManager.Start` 获取锁，但 `DaemonLauncher.Start` 内部也有自己的 PID 文件检查（`GetRunningPID`）。两层锁机制并存可能导致：
- 锁竞争：两个 CLI 实例同时启动时，一个获取了 `ProcessLock`，另一个在 `DaemonLauncher.Start` 的 PID 检查处被拒绝
- 锁粒度不一致：`ProcessLock` 是文件锁（`flock`），PID 检查是基于 PID 文件

---

### 问题 6（P3 低）：`DaemonLauncher.GetWorkDir()` 对相对路径处理不当

**文件**：`internal/mihomo/daemon_launcher.go:72-78`

```go
func (dl *DaemonLauncher) GetWorkDir() string {
    execPath := dl.GetExecutablePath()
    if execPath == "" {
        return ""
    }
    return filepath.Dir(execPath)
}
```

如果 `execPath` 是相对路径（如 `"build/mihomo"`），`GetWorkDir()` 返回 `"build"`（也是相对路径），导致 `cmd.Dir = "build"` 可能不正确。

---

## 三、改进方案

### 方案 1：修复 `CreateProcessGroup`（P0 致命）

**修改文件**：`internal/mihomo/daemon_linux.go`、`internal/mihomo/daemon_darwin.go`

**Linux 版本**：

```go
func (ldm *LinuxDaemonManager) CreateProcessGroup(cmd *exec.Cmd) error {
    // 使用 Setpgid 创建新进程组，使子进程脱离父进程的进程组
    // 不使用 Setsid，因为在某些受限环境中 Setsid+Setpgid 组合会失败
    // Setpgid 足以让子进程不受终端关闭影响
    cmd.SysProcAttr = &syscall.SysProcAttr{
        Setpgid: true,
    }
    return nil
}
```

**Darwin 版本**：同理修改 `daemon_darwin.go` 的 `CreateProcessGroup`。

**理由**：
- `Setpgid: true` 足够让子进程成为新进程组 leader，脱离父进程组
- `Setsid` 创建新会话（脱离控制终端），但对 daemon 场景非必须——mihomo 内核本身不依赖控制终端
- 如果确实需要脱离控制终端，可以在 `Setpgid` 成功后再单独调用 `setsid()`，但需要先确保 `setpgid` 已生效

**备选方案**：只用 `Setsid: true`（去掉 `Setpgid`）。`setsid()` 隐含创建新会话和新进程组。但需确保调用时进程不是进程组 leader——Go 的 `exec.Command` 在 fork 后、exec 前会自动调用 `setpgid(0, 0)`（如果设置了 `Setpgid`），所以只用 `Setsid` 也可能失败。**推荐只用 `Setpgid`**。

---

### 方案 2：修复 Darwin `RedirectIO` 资源泄漏（P1 严重）

**修改文件**：`internal/mihomo/daemon_darwin.go`

将 Darwin 的 `RedirectIO` 签名和实现改为与 Linux 一致：

```go
func (ddm *DarwinDaemonManager) RedirectIO(cmd *exec.Cmd, logFile string) ([]io.Closer, error) {
    var closers []io.Closer

    if logFile != "" {
        logDir := filepath.Dir(logFile)
        if err := os.MkdirAll(logDir, 0755); err != nil {
            return nil, pkgerrors.ErrConfig("failed to create log directory", err)
        }

        logFH, err := os.OpenFile(logFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
        if err != nil {
            return nil, pkgerrors.ErrConfig("failed to open log file", err)
        }

        cmd.Stdout = logFH
        cmd.Stderr = logFH
        closers = append(closers, logFH)
    } else {
        devNull, err := os.OpenFile("/dev/null", os.O_RDWR, 0)
        if err != nil {
            return nil, pkgerrors.ErrConfig("failed to open /dev/null", err)
        }
        cmd.Stdout = devNull
        cmd.Stderr = devNull
        closers = append(closers, devNull)
    }

    devNull, err := os.OpenFile("/dev/null", os.O_RDONLY, 0)
    if err != nil {
        return nil, pkgerrors.ErrConfig("failed to open /dev/null for stdin", err)
    }
    cmd.Stdin = devNull
    closers = append(closers, devNull)

    return closers, nil
}
```

同时更新 `DaemonManager` 接口签名以匹配 Linux 版本。

---

### 方案 3：确保 `resolveExplicit` 返回绝对路径（P2 中等）

**修改文件**：`internal/config/executable_resolver.go`

```go
func (r *ExecutableResolver) resolveExplicit(explicit string) (string, error) {
    p := explicit
    if !filepath.IsAbs(p) {
        p = filepath.Join(r.baseDir, p)
    }
    // 确保返回绝对路径
    if !filepath.IsAbs(p) {
        if abs, err := filepath.Abs(p); err == nil {
            p = abs
        }
    }
    p = filepath.Clean(p)
    // ... 其余不变
}
```

---

### 方案 4：启动前清理过期 State 文件（P2 中等）

**修改文件**：`internal/mihomo/process_handler.go`

在 `checkAndCleanupBeforeStart` 中增加清理过期 State 文件的逻辑：

```go
func (ph *ProcessHandler) checkAndCleanupBeforeStart(cfg *config.TomlConfig) error {
    // ... 现有系统配置检查 ...

    // 清理过期的 State 文件（stage=failed 或 pid=0）
    pathResolver, err := config.NewPathResolver()
    if err == nil {
        stateFile := pathResolver.GetStateFilePath(cfg.Mihomo.ConfigFile)
        if stateFile != "" {
            var state ProcessState
            data, err := os.ReadFile(stateFile)
            if err == nil {
                if json.Unmarshal(data, &state) == nil {
                    if state.Stage == StageFailed || state.PID == 0 {
                        os.Remove(stateFile)
                        output.Info("Cleaned up stale state file")
                    }
                }
            }
        }
    }

    return nil
}
```

---

### 方案 5：统一锁机制（P3 低）

**建议**：保留 `ProcessLock`（flock）作为主要的并发控制机制，移除 `DaemonLauncher.Start` 中的冗余 PID 检查（或将其降级为日志警告）。

---

### 方案 6：`GetWorkDir` 强制返回绝对路径（P2 中等）

**修改文件**：`internal/mihomo/daemon_launcher.go`

```go
func (dl *DaemonLauncher) GetWorkDir() string {
    execPath := dl.GetExecutablePath()
    if execPath == "" {
        return ""
    }
    dir := filepath.Dir(execPath)
    if !filepath.IsAbs(dir) {
        if abs, err := filepath.Abs(dir); err == nil {
            return abs
        }
    }
    return dir
}
```

---

## 四、修改优先级

| 优先级 | 问题 | 影响 | 修改文件 |
|--------|------|------|----------|
| **P0** | `Setsid + Setpgid` 组合 | start 完全无法使用 | `daemon_linux.go`, `daemon_darwin.go` |
| **P1** | Darwin RedirectIO 资源泄漏 | macOS 上子进程启动后 I/O 异常 | `daemon_darwin.go` |
| **P2** | resolveExplicit 相对路径 | 路径解析不一致 | `executable_resolver.go` |
| **P2** | 启动前未清理过期 State | 残留状态影响后续启动 | `process_handler.go` |
| **P2** | GetWorkDir 相对路径 | cmd.Dir 不正确 | `daemon_launcher.go` |
| **P3** | 锁机制冗余 | 并发启动时行为不确定 | `lifecycle.go` / `daemon_launcher.go` |

---

## 五、测试验证

修改后需要验证：

1. **Linux start/stop/status**：`Setsid+Setpgid` 修复后，start 应能正常拉起内核
2. **Darwin start/stop/status**：同样验证 + 检查 I/O 重定向正常
3. **相对路径场景**：`config.toml` 中 `executable = "./build/mihomo"` 应正确解析为绝对路径
4. **并发启动**：两个 CLI 实例同时 `start` 应只允许一个成功
5. **异常退出恢复**：上次启动失败后再次 `start` 应清理过期 State
