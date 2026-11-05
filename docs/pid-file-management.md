# PID 文件管理文档

## 概述

Mihomo CLI 使用 PID 文件实现跨进程状态管理。由于 CLI 是无状态的（每次执行都是新进程），需要通过持久化的 PID 文件来跟踪 Mihomo 内核进程。

PID 文件**不是**运行状态的唯一真相来源：`status` 命令以 API 探测与本机进程扫描为准，PID 文件主要用于托管实例识别、启停定位与重复启动防护。

## 为什么需要 PID 文件

### 核心原因

CLI 设计为无状态工具，每次命令执行都是独立的进程，无法在内存中保持状态。PID 文件提供了以下关键功能：

1. **防止重复启动**：同一配置文件只能启动一个实例
2. **进程定位**：停止和查询时能找到正确的进程
3. **状态查询**：判断进程是否运行
4. **异常处理**：清理崩溃、系统重启等导致的残留文件

### 使用场景

- ✅ `start` 命令：检测重复启动
- ✅ `stop` 命令：定位要停止的进程
- ✅ `status` 命令：识别 CLI 托管实例（进程扫描的补充信息）
- ✅ `ps` 命令：识别 CLI 托管实例
- ✅ `cleanup` 命令：清理残留文件
- ✅ 健康检查超时：停止已启动的进程
- ❌ 信号处理：CLI 不注册全局信号处理器；`Ctrl+C` 只终止当前 CLI 进程，不会停止已守护化的内核或删除 PID 文件，残留由 `cleanup` 处理

## 存储路径

### 默认路径

基础目录为**平台规范目录** `os.UserConfigDir()/mihomo-cli`（由 `config.GetBaseDir()` 统一解析）：

- **Windows**: `%AppData%\Roaming\mihomo-cli\`
- **macOS**: `~/Library/Application Support/mihomo-cli/`
- **Linux**: `$XDG_CONFIG_HOME/mihomo-cli`（未设置时为 `~/.config/mihomo-cli/`）

```
<UserConfigDir>/mihomo-cli/
├── mihomo.pid             # 默认配置（未指定配置文件）
├── mihomo-{identity}.pid  # 指定配置文件的实例（PID 元数据 v1）
├── state-{identity}.json   # 运行状态文件
├── lock-{identity}         # 并发启动锁文件
├── backups/               # 配置备份目录
├── data/                  # 数据目录（审计日志、快照等）
└── history/               # 命令历史目录
```

### 路径生成规则

**基础目录**：`os.UserConfigDir()/mihomo-cli`（平台规范，无旧目录回退）。

**PID 文件名**：
- 未指定配置文件：`mihomo.pid`
- 指定配置文件：`mihomo-{identity}.pid`

### Identity 生成规则

identity 由 `internal/config/identity.go` 的 `config.IdentityOf()` **唯一**生成：
配置文件的规范化绝对路径（`filepath.Abs` + `filepath.Clean`）取 SHA-256 前 12 位
十六进制（48 bit）。同一配置文件无论以相对/绝对路径传入，结果一致；
空路径返回 `"default"`。

```go
func IdentityOf(configFile string) string {
    if configFile == "" {
        return DefaultIdentity // "default"
    }
    abs := identityAbsPath(configFile)
    sum := sha256.Sum256([]byte(abs))
    return hex.EncodeToString(sum[:])[:12]
}
```

同一 identity 同时用于 **PID 元数据文件、状态文件与锁文件**的命名。
此前三处各实现一套互不兼容的 hash（截断文件名 / SHA256[:16] / SHA256[:8]），
已全部收敛到 `config.IdentityOf` 一处。

### 示例

| 配置文件路径 | PID 文件路径 |
|-------------|-------------|
| (未指定) | `<UserConfigDir>/mihomo-cli/mihomo.pid` |
| `C:/Users/dev/proxy/mihomo-config.yaml` | `<UserConfigDir>/mihomo-cli/mihomo-<identity>.pid`（identity 由上述算法推导） |

## PID 文件生命周期

### 创建

**时机**：Mihomo 内核启动成功后（各平台守护进程管理器 `StartAsDaemon`）

**位置**：`internal/mihomo/daemon_common.go`（`DaemonManagerCommon.SavePID`），
内部组合元数据后由 `InstanceRegistry.Save`（`internal/mihomo/instance.go`）
原子写入（先写 `.tmp` 再 rename）：

```go
// SavePID 保存 PID 元数据（v1 JSON，含配置文件/可执行文件/API 地址/启动时间）
func (d *DaemonManagerCommon) SavePID(pid int) error {
    meta := InstanceMeta{
        PID:        pid,
        ConfigFile: d.base.GetConfigFile(),
        ExecPath:   d.base.GetExecutablePath(),
        APIAddr:    d.base.GetAPIAddress(),
        StartedAt:  time.Now().Format(time.RFC3339),
    }
    return d.pid.Save(meta)
}
```

**内容**：v1 JSON 元数据（不是纯数字 PID），使 `ps`/`status` 可直接读取
配置文件、可执行文件与 API 端口信息：

```json
{
  "version": 1,
  "pid": 18296,
  "config_file": "C:/Users/dev/proxy/mihomo-config.yaml",
  "exec_path": "C:/Users/dev/proxy/mihomo.exe",
  "api_addr": "127.0.0.1:9090",
  "started_at": "2026-09-05T10:00:00+08:00"
}
```

### 读取

**方法**：`ProcessManager.GetPIDFromPIDFile()` / `DaemonLauncher.GetRunningPID()`，
内部统一经 `InstanceRegistry.Load()`（`internal/mihomo/instance.go`）读取。

**验证流程**：
1. 读取 PID 文件内容并解析（必须为 v1 JSON，版本不符视为损坏文件）
2. 检查进程是否真实存在（使用各平台进程检测 API）
3. 如果进程不存在，返回错误

```go
meta, err := NewInstanceRegistry(pm.pidFile).Load()
if err != nil {
    return 0, err
}

// 关键：验证进程是否真实存在
if !IsProcessRunning(meta.PID) {
    return 0, pkgerrors.ErrService("process ... is not running", nil)
}

return meta.PID, nil
```

### 删除

**触发时机**：

1. **正常停止**：`stop` 命令执行成功后
   ```go
   os.Remove(pm.pidFile) // 实际由 DaemonLauncher.Stop / StopDaemon 调用 CleanupPID
   ```

2. **健康检查超时**：启动超时清理
   ```go
   if pid, err := pm.GetPIDFromPIDFile(); err == nil {
       // ProcessHandler.Start 调用 launcher.Stop(true)
       // 由 DaemonLauncher.Stop / StopDaemon 清理 PID 文件
   }
   ```

3. **手动清理**：`cleanup` 命令

4. **停止所有进程**：`stop --all` 会扫描实例并在进程退出后清理对应 PID 文件

**不会自动删除的情况**：
- 进程崩溃、被外部杀死、系统重启：PID 文件残留，需通过 `cleanup` 或 `stop --all` 清理
- CLI 进程被 `Ctrl+C` 终止：不会触发内核停止或 PID 文件删除

## 格式版本（锁定 v1）

| 文件 | 格式 | 说明 |
|---|---|---|
| PID 文件 | JSON 元数据 v1 + identity 文件名（`mihomo-<12位hex>.pid`） | 版本号锁定为 1；`InstanceMeta.Version` 固定写入 1 |
| 状态文件 | `state-<identity>.json` | 内容格式不变，命名统一走 `config.IdentityOf` |
| 锁文件 | `lock-<identity>` | 命名统一走 `config.IdentityOf` |

读取规则（`internal/mihomo/instance.go`）：

1. 内容可解析为 JSON 且 `version=1`、`pid>0` → 直接使用；
2. 否则视为损坏文件：跳过 + 提示（不按错误 PID 执行 kill）。

**开发阶段决策**：不做 v2 版本升级，不兼容任何旧数据格式（纯数字 PID、
旧目录、旧 hash 命名一律不读、不迁移、不回退）。解析失败即"跳过 + 提示"，
绝不按错误 PID 执行 kill（防止 PID 复用导致误杀）。

## 各命令使用情况

### start 命令

**用途**：
1. 检测重复启动（`launcher.GetRunningPID()`：先读状态文件，再读 PID 文件）
2. 通过进程锁避免并发启动
3. 启动成功后保存新进程的 PID 元数据

**流程**：
```
1. 读取状态 / PID 文件
2. 如果存在且进程运行 → 拒绝启动
3. 获取进程锁
4. 启动进程
5. SavePID 保存元数据
6. 启动生命周期监控
```

**相关代码**：
- `process_handler.go:Start()` - 业务流程与健康检查
- `daemon_launcher.go:Start()` - 创建守护进程管理器并启动
- `daemon_common.go:SavePID()` - 写入 PID 元数据

### stop 命令

**用途**：
1. 默认定位当前配置对应的实例（`launcher.GetRunningPID()`）
2. 通过 API 优雅关闭，失败时强制终止
3. 清理 PID 文件与状态文件

**流程**：
```
1. 读取状态 / PID 文件
2. 如果不存在或进程不运行 → 返回错误
3. 通过 PID 停止进程
4. 清理 PID 文件与状态文件
```

传入 PID 参数时，先校验该进程正在运行；`--all` 则扫描所有托管与外部实例并清理对应 PID 文件。

**相关代码**：
- `process_handler.go:Stop()` - 业务流程
- `daemon_launcher.go:Stop()` / `StopDaemon()` - 实际停止与 PID 文件清理

### status 命令

**用途**：
1. 查询内核运行状态与 API 可达性
2. 识别本机内核进程及其管理方式

**流程**：
```
1. 探测 API（GetVersion）
2. 解析 API 地址，仅对本机地址执行进程补充
3. 通过监听端口查找 PID，失败时静默扫描进程
4. 根据 PID 文件元数据判断是否为 CLI 托管实例
5. 输出状态、PID、管理方式、进程数等
```

**注意**：`status` 不把 PID 文件当作真相来源；PID 文件仅用于在进程扫描阶段识别 CLI 托管实例。

**相关代码**：
- `status.go:QueryStatus()` - 核心查询
- `discovery.go` / `scanner.go` - 托管与外部实例扫描

### cleanup 命令

**用途**：
1. 清理所有残留的 PID 元数据文件

**流程**：
```
1. 扫描 <UserConfigDir>/mihomo-cli/ 目录
2. 遍历所有 .pid 文件
3. 解析 v1 元数据；损坏文件直接删除
4. 验证进程是否存在，删除进程不存在的 PID 文件
```

**相关代码**：
- `scanner.go:CleanupPIDFiles()` - 清理残留文件

### ps 命令

**用途**：
1. 列出所有 Mihomo 进程（托管 + 外部）

托管实例来自 PID 目录元数据，外部实例来自全进程表枚举；PID 文件用于识别托管实例与读取元数据。

**相关代码**：
- `discovery.go:ScanMihomoProcesses()` - 两源合并扫描

## 异常处理

### 残留文件产生原因

1. **进程崩溃**：进程异常退出，无自动清理
2. **系统重启**：系统重启，PID 文件保留但进程已退出
3. **进程被外部杀死**：任务管理器、第三方工具
4. **启动超时**：健康检查超时，进程已启动但未响应，随后由 `launcher.Stop(true)` 清理
5. **文件系统错误**：写入 PID 失败，但进程已启动

### 清理策略

**手动清理**：
```bash
mihomo-cli cleanup
```

**批量清理**：
```bash
mihomo-cli stop --all
```

**自动清理**：
- `cleanup` 命令会自动扫描并清理所有无效的 PID 文件
- 验证进程是否存在后再删除

### 安全机制

**双重验证**：
1. PID 文件存在
2. 进程真实运行

```go
meta, err := NewInstanceRegistry(pm.pidFile).Load()
if err != nil {
    return 0, err
}

// 关键：验证进程是否真实存在
if !IsProcessRunning(meta.PID) {
    return 0, pkgerrors.ErrService("process ... is not running", nil)
}

return meta.PID, nil
```

**防误杀**：解析失败或版本不符的文件一律跳过并提示，绝不按错误 PID 执行 kill，避免 PID 复用导致误杀。

## Windows 进程检测

### 实现方式

使用 Windows API 检查进程是否存在：

```go
handle, err := windows.OpenProcess(
    windows.PROCESS_QUERY_INFORMATION,
    false,
    uint32(pid),
)
if err != nil {
    // 权限不足（ERROR_ACCESS_DENIED）时保守认为进程仍在运行
    if errno, ok := err.(windows.Errno); ok && errno == windows.ERROR_ACCESS_DENIED {
        return true
    }
    return false
}
_ = windows.CloseHandle(handle)
return true
```

### 为什么不使用 `os.Process.Signal`

- Windows 上 `Signal` 行为与 Unix 不同
- 需要使用 Windows API 确保可靠性

## 最佳实践

### 开发者

1. **始终验证进程**：读取 PID 后必须验证进程是否运行
2. **清理残留文件**：开发测试后运行 `mihomo-cli cleanup`
3. **使用配置文件**：不同配置使用不同 PID 文件，避免冲突

### 用户

1. **系统重启后清理**：系统重启后建议运行 `mihomo-cli cleanup`
2. **异常处理后清理**：如果进程崩溃或被杀死，运行清理命令
3. **检查残留**：如果 `stop` 或 `status` 命令异常，先清理再重试

## 文件权限

**目录权限**：`0755`（rwxr-xr-x）
- 所有者：读写执行
- 组和他人：读执行

**文件权限**：
- PID 文件：`0644`（rw-r--r--）
- 状态文件：`0600`（rw-------）
- 锁文件：`0644`（rw-r--r--）

## 故障排查

### 问题：显示"未运行"但进程存在

**可能原因**：
1. PID 文件损坏
2. PID 文件不存在
3. 配置文件路径变化导致 PID 文件不匹配
4. `status` 以 API / 进程扫描为准，PID 文件缺失不影响进程识别

**解决方法**：
```bash
# 清理残留文件
mihomo-cli cleanup

# 重新启动
mihomo-cli start
```

### 问题：显示"已在运行"但实际未运行

**可能原因**：
PID 文件残留（进程被外部杀死）

**解决方法**：
```bash
# 清理残留文件
mihomo-cli cleanup

# 重新启动
mihomo-cli start
```

### 问题：无法停止进程

**可能原因**：
1. 进程权限问题
2. 进程已停止但 PID 文件未删除

**解决方法**：
```bash
# 方法1：使用 stop --all
mihomo-cli stop --all

# 方法2：清理后重试
mihomo-cli cleanup
mihomo-cli stop

# 方法3：强制停止
mihomo-cli stop -F
```

## 相关文件

- `internal/config/identity.go` - `IdentityOf` 唯一身份算法（PID/状态/锁文件命名）
- `internal/config/paths.go` / `config/path_resolver.go` - 路径计算（平台规范目录）
- `internal/config/executable_resolver.go` - 内核可执行文件统一解析（启动/服务安装共用）
- `internal/mihomo/instance.go` - `InstanceMeta` / `InstanceRegistry`（PID 元数据 v1 读写，原子写）
- `internal/mihomo/daemon_common.go` - `DaemonManagerCommon.SavePID`（启动后写入完整元数据）
- `internal/mihomo/daemon_launcher.go` - `DaemonLauncher` 读取/清理 PID 文件
- `internal/mihomo/daemon_linux.go` / `daemon_darwin.go` / `daemon_windows.go` - 各平台守护进程管理器
- `internal/mihomo/manager.go` - `ProcessManager.GetPIDFromPIDFile`
- `internal/mihomo/lifecycle.go` - 生命周期阶段与状态追踪
- `internal/mihomo/state.go` - 状态文件读写
- `internal/mihomo/lock.go` - 进程锁文件
- `internal/mihomo/discovery.go` - 两源进程发现（managed + external 合并、验证分级）
- `internal/mihomo/scanner.go` - 实例扫描（`ps`）、残留文件清理（`cleanup`）
- `internal/mihomo/status.go` - `status` 查询（API + 进程扫描，PID 文件仅作托管识别）
- `internal/mihomo/process_windows.go` - Windows 进程检测
- `internal/mihomo/process_list_*.go` - 全进程枚举（windows/linux/darwin）
- `cmd/start.go` / `cmd/cleanup.go` / `cmd/ps.go` - start+stop+status / cleanup / ps 命令

## 参考资料

- 无状态 CLI 设计原则
- Windows 进程管理 API
- Unix PID 文件最佳实践