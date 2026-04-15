# Daemon 模块分析报告

## 概述

Daemon（守护进程）模块是 mihomo-cli 项目的核心组件之一，负责将 Mihomo 内核作为独立的后台进程运行，实现 CLI 工具与 Mihomo 内核的解耦。该模块提供了跨平台的守护进程管理能力，包括进程启动、停止、状态监控、PID 管理等功能。

## 架构设计

### 核心设计理念

Daemon 模块采用**分层架构**和**策略模式**设计：

```
应用层 (ProcessHandler, LifecycleManager)
    ↓
协调层 (DaemonLauncher, ProcessManager)
    ↓
平台抽象层 (DaemonManager 接口)
    ↓
平台实现层 (Windows/Linux/Darwin DaemonManager)
    ↓
通用层 (DaemonManagerCommon, DaemonManagerBase)
```

### 模块文件结构

```
internal/mihomo/
├── daemon.go              # 核心接口和基础类型定义
├── daemon_common.go       # 跨平台通用功能实现
├── daemon_windows.go      # Windows 平台实现
├── daemon_linux.go        # Linux 平台实现
├── daemon_darwin.go       # macOS 平台实现
├── daemon_launcher.go     # 守护进程启动器（统一入口）
├── manager.go             # 进程管理器（旧版实现）
├── process_handler.go     # 进程处理器（新版实现）
├── lifecycle.go           # 生命周期管理器
└── state.go               # 状态管理器
```

## 核心组件详解

### 1. DaemonManager 接口

**文件**: `daemon.go`

定义了守护进程管理器的统一接口，所有平台实现都必须遵循此接口：

```go
type DaemonManager interface {
    StartAsDaemon(ctx context.Context, cfg interface{}) error  // 启动守护进程
    StopDaemon(pid int) error                                   // 停止守护进程
    IsDaemonRunning(pid int) bool                               // 检查运行状态
    GetDaemonPID() (int, error)                                 // 获取进程 PID
    RedirectIO(cmd *exec.Cmd, logFile string) error            // 重定向 I/O
    CreateProcessGroup(cmd *exec.Cmd) error                     // 创建进程组
}
```

**关键配置类型**:

- **DaemonConfig**: 守护进程配置结构
  - `Enabled`: 是否启用
  - `WorkDir`: 工作目录
  - `LogFile`: 日志文件路径
  - `LogLevel`: 日志级别
  - `LogMaxSize`: 日志文件大小限制
  - `LogMaxBackups`: 日志备份数量
  - `LogMaxAge`: 日志保留天数
  - `AutoRestart`: 自动重启配置
  - `HealthCheck`: 健康检查配置

### 2. DaemonLauncher（守护进程启动器）

**文件**: `daemon_launcher.go`

这是启动和停止守护进程的**统一入口点**，负责：

#### 核心功能

1. **路径解析**
   - 使用 `PathResolver` 将所有相对路径转换为绝对路径
   - 管理可执行文件、配置文件、PID 文件、状态文件的路径

2. **状态管理**
   - 通过 `StateManager` 持久化进程状态（PID、API 地址、密钥等）
   - 状态保存在 JSON 格式的状态文件中

3. **启动流程** (`Start()` 方法)
   ```
   检查是否已运行 → 准备密钥 → 获取绝对路径 → 构建配置 
   → 创建 DaemonManager → 启动进程 → 获取 PID → 更新状态
   ```

4. **停止流程** (`Stop()` 方法)
   ```
   获取 PID → 优先 API 优雅关闭 → 失败则强制 kill → 清理状态和 PID 文件
   ```

5. **健康检查** (`WaitForHealthy()` 方法)
   - 轮询检查进程是否存活
   - 尝试连接 Mihomo API 验证服务可用性
   - 支持超时控制

#### 关键方法

| 方法 | 功能 | 返回值 |
|------|------|--------|
| `Start()` | 启动守护进程 | error |
| `Stop(force bool)` | 停止守护进程（force 参数控制是否强制） | error |
| `GetRunningPID()` | 获取运行中的 PID（从状态文件或 PID 文件） | (int, error) |
| `GetStatus()` | 获取运行状态 | (isRunning, pid, apiAddr, secret) |
| `WaitForHealthy(timeout)` | 等待进程健康检查 | error |

### 3. 跨平台实现

#### Windows 平台 (`daemon_windows.go`)

**进程创建标志**:
```go
cmd.SysProcAttr = &windows.SysProcAttr{
    CreationFlags: windows.CREATE_NEW_PROCESS_GROUP | windows.DETACHED_PROCESS,
}
```

- `CREATE_NEW_PROCESS_GROUP`: 创建新进程组，防止接收 Ctrl+C 信号
- `DETACHED_PROCESS`: 创建独立控制台进程，不继承父进程控制台

**I/O 重定向**:
- 日志文件模式：Stdout 和 Stderr 重定向到指定日志文件
- 无日志模式：重定向到 Windows 的 `NUL` 设备
- Stdin 始终重定向到 `NUL`

**停止策略**:
1. 优先通过 Mihomo API 调用 `Shutdown()` 优雅关闭
2. 等待最多 10 秒让进程正常退出
3. 超时或 API 失败则使用 `ForceKill()` 强制终止

#### Linux 平台 (`daemon_linux.go`)

**进程创建**:
- 使用 `setsid()` 创建新会话
- 使用 `setpgid()` 设置进程组
- 完全脱离父进程的会话和控制终端

**I/O 重定向**:
- 类似 Windows，使用 `/dev/null` 作为默认重定向目标

**停止策略**:
- 与 Windows 类似，先尝试 API 优雅关闭
- 失败后使用 SIGTERM/SIGKILL 强制终止

#### macOS 平台 (`daemon_darwin.go`)

实现与 Linux 类似，但增加了：
- launchd 集成支持（可选）
- 适合 macOS 的特定路径处理

### 4. 通用功能层

#### DaemonManagerCommon (`daemon_common.go`)

封装所有平台共享的功能：

1. **PID 文件管理** (`PIDFileManager`)
   - `Save(pid)`: 将 PID 写入文件
   - `Read()`: 从文件读取 PID
   - `Cleanup()`: 删除 PID 文件
   - 自动创建目录，确保文件路径存在

2. **进程状态检查**
   - `IsDaemonRunning(pid)`: 检查进程是否运行
   - 使用 `os.FindProcess()` 和信号检测

3. **强制终止** (`ForceKillDaemon()`)
   - 使用 `os.FindProcess()` 查找进程
   - 调用 `proc.Kill()` 发送 SIGKILL
   - 等待进程完全退出
   - 清理 PID 文件

#### DaemonManagerBase (`daemon.go`)

提供配置访问的 getter 方法：
- `GetConfig()`, `GetPIDFile()`, `GetSecret()`
- `GetAPIAddress()`, `GetExecutablePath()`, `GetConfigFile()`
- `GetWorkDir()`: 获取工作目录

### 5. 状态管理 (`state.go`)

**ProcessState 结构**:
```go
type ProcessState struct {
    PID             int            // 进程 ID
    APIAddress      string         // API 地址
    Secret          string         // API 密钥
    ConfigFile      string         // 配置文件路径
    StartedAt       time.Time      // 启动时间
    LastHealthCheck time.Time      // 最后健康检查时间
    Stage           LifecycleStage // 生命周期阶段
    ConfigHash      string         // 配置文件哈希
}
```

**生命周期阶段**:
```
pre-start → starting → running → pre-stop → stopping → stopped
                                      ↓
                                   failed
```

**StateManager 功能**:
- JSON 格式持久化状态到文件
- 线程安全（使用 `sync.RWMutex`）
- 支持原子更新（`Update()` 方法）
- 状态过期检测（`IsStale()` 方法）

### 6. 生命周期管理 (`lifecycle.go`)

**LifecycleManager** 提供了完整的进程生命周期控制：

#### 核心流程

**启动流程**:
```
1. PreStart 阶段
   - 执行 PreStart 钩子
   - 验证可执行文件和配置文件
   - 检查端口占用

2. Starting 阶段
   - 获取进程锁（防止并发启动）
   - 调用 ProcessManager.Start()
   - 更新状态为 Starting

3. Running 阶段
   - 读取 PID、API 地址、密钥
   - 更新状态为 Running
   - 启动进程监控器
   - 执行 PostStart 钩子
```

**停止流程**:
```
1. PreStop 阶段
   - 执行 PreStop 钩子
   - 更新状态为 PreStop

2. Stopping 阶段
   - 停止监控器
   - 调用 daemonManager.StopDaemon()
   - 更新状态为 Stopping

3. Stopped 阶段
   - 清除状态
   - 执行 PostStop 钩子
   - 更新状态为 Stopped
```

**生命周期钩子**:
```go
type LifecycleHook interface {
    OnPreStart(ctx context.Context, cfg *config.TomlConfig) error
    OnPostStart(ctx context.Context, pid int) error
    OnPreStop(ctx context.Context, pid int) error
    OnPostStop(ctx context.Context) error
    OnFailure(ctx context.Context, stage LifecycleStage, err error)
}
```

### 7. 进程处理器 (`process_handler.go`)

**ProcessHandler** 是面向用户的高级接口，整合了所有底层组件：

#### 核心功能

1. **启动 Mihomo** (`Start()` 方法)
   - 检查是否启用自动启动
   - 验证可执行文件存在性
   - 创建 DaemonLauncher
   - 检测高风险配置（TUN/TProxy）
   - 备份系统配置
   - 启动守护进程
   - 执行健康检查（基础 + 增强）
   - 返回启动结果（API 地址、密钥、PID）

2. **停止 Mihomo** (`Stop()` 方法)
   - 支持多种停止模式：
     - `--all`: 停止所有 Mihomo 进程
     - 指定 PID: 停止特定进程
     - 默认: 停止当前配置的进程
   - 支持强制停止 (`-F` 标志)
   - 停止后检查和清理系统配置

3. **状态查询** (`Status()` 方法)
   - 创建 DaemonLauncher
   - 获取运行状态
   - 返回进程信息（是否运行、PID、API 地址）

## 配置体系

### TOML 配置结构

**文件**: `internal/config/toml_config.go`

```toml
[daemon]
enabled = true                  # 是否启用守护进程模式
work_dir = ""                   # 工作目录（可选）
log_file = ""                   # 日志文件路径
log_level = "info"              # 日志级别
log_max_size = "100M"           # 日志文件最大大小
log_max_backups = 10            # 保留备份数量
log_max_age = 30                # 最大保留天数

[daemon.auto_restart]
enabled = false                 # 自动重启（规划中）
max_restarts = 5                # 最大重启次数
restart_delay = "5s"            # 重启延迟

[daemon.health_check]
enabled = false                 # 健康检查（规划中）
interval = "30s"                # 检查间隔
timeout = "10s"                 # 检查超时
```

### 默认配置

当配置文件不存在时，系统使用以下默认值：

```go
Daemon: &DaemonConfig{
    Enabled:       true,
    WorkDir:       "",
    LogFile:       "",
    LogLevel:      "info",
    LogMaxSize:    "100M",
    LogMaxBackups: 10,
    LogMaxAge:     30,
    AutoRestart: AutoRestartConfig{
        Enabled:      false,
        MaxRestarts:  5,
        RestartDelay: "5s",
    },
    HealthCheck: HealthCheckConfig{
        Enabled:  false,
        Interval: "30s",
        Timeout:  "10s",
    },
}
```

## 使用方式

### 基本使用流程

#### 1. 配置文件准备

创建 `config.toml`:

```toml
[api]
address = "http://127.0.0.1:9090"
secret = ""

[mihomo]
enabled = true
executable = "./mihomo.exe"
config_file = "./mihomo-config.yaml"
auto_generate_secret = true
health_check_timeout = 5

[mihomo.api]
external_controller = "127.0.0.1:9090"

[mihomo.log]
level = "info"

[daemon]
enabled = true
log_file = "./logs/mihomo-daemon.log"
log_level = "info"
```

#### 2. 启动守护进程

通过 CLI 命令启动（具体命令取决于 cmd 层的实现）：

```bash
mihomo-cli start
```

底层调用链：
```
cmd 层 → ProcessHandler.Start() 
       → DaemonLauncher.Start() 
       → GetDaemonManager() 
       → WindowsDaemonManager.StartAsDaemon()
```

#### 3. 查询状态

```bash
mihomo-cli status
```

调用链：
```
cmd 层 → ProcessHandler.Status() 
       → DaemonLauncher.GetStatus() 
       → 从 StateManager 和 PID 文件读取状态
```

#### 4. 停止守护进程

```bash
mihomo-cli stop
# 或强制停止
mihomo-cli stop -F
```

调用链：
```
cmd 层 → ProcessHandler.Stop() 
       → DaemonLauncher.Stop() 
       → 通过 API 关闭或 ForceKill()
```

### 编程接口使用

#### 示例 1: 直接启动守护进程

```go
import (
    "github.com/kkkqkx123/mihomo-cli/internal/config"
    "github.com/kkkqkx123/mihomo-cli/internal/mihomo"
)

// 加载配置
cfg, err := config.LoadTomlConfig("config.toml")
if err != nil {
    // 处理错误
}

// 创建进程处理器
handler := mihomo.NewProcessHandler("config.toml")

// 启动
result, err := handler.Start(cfg)
if err != nil {
    // 处理错误
}

fmt.Printf("Mihomo 已启动，PID: %d, API: %s\n", result.PID, result.APIAddress)
```

#### 示例 2: 使用生命周期管理器

```go
// 创建生命周期管理器
lm, err := mihomo.NewLifecycleManager(cfg)
if err != nil {
    // 处理错误
}

// 注册自定义钩子
lm.RegisterHook(&MyCustomHook{})

// 启动（包含所有生命周期阶段）
ctx := context.Background()
err = lm.Start(ctx, cfg)

// 停止
state := lm.GetState()
err = lm.Stop(ctx, state.PID)
```

#### 示例 3: 手动管理守护进程

```go
// 创建启动器
launcher, err := mihomo.NewDaemonLauncher(cfg)
if err != nil {
    // 处理错误
}

// 启动
err = launcher.Start()

// 等待健康检查
err = launcher.WaitForHealthy(10 * time.Second)

// 获取状态
isRunning, pid, apiAddr, secret := launcher.GetStatus()

// 停止
err = launcher.Stop(false)  // false 表示优雅关闭
```

## 关键特性

### 1. 跨平台兼容性

- **Windows**: 使用 `CREATE_NEW_PROCESS_GROUP` 和 `DETACHED_PROCESS`
- **Linux**: 使用 `setsid()` 和 `setpgid()` 系统调用
- **macOS**: 类似 Linux，支持 launchd 集成

### 2. 进程独立性

守护进程完全独立于 CLI 进程：
- 不共享控制台
- 不接收 Ctrl+C 信号
- CLI 退出不影响守护进程
- 独立的进程组和会话

### 3. 状态持久化

通过两个文件实现状态持久化：
- **PID 文件**: 存储进程 ID，供外部工具查询
- **状态文件**: 存储完整状态信息（JSON 格式），包括 API 地址、密钥等

### 4. 优雅关闭

两层关闭策略：
1. **优雅关闭**: 通过 Mihomo API 调用 `Shutdown()`，等待进程正常退出
2. **强制关闭**: 使用操作系统信号强制终止进程

### 5. 健康检查

启动后自动执行健康检查：
- **基础检查**: 连接 API 并调用 `GetMode()`
- **增强检查**: 验证配置文件中的各项组件是否正常
- **超时控制**: 可配置超时时间，避免无限等待

### 6. 日志管理

- 支持文件日志输出
- I/O 完全重定向，不依赖控制台
- 日志轮转配置（大小、备份、保留时间）
- 跨平台的空设备处理（NUL vs /dev/null）

## 实现状态

### 已实现功能 ✅

- [x] 跨平台守护进程启动（Windows/Linux/macOS）
- [x] PID 文件管理
- [x] 状态持久化和恢复
- [x] 优雅关闭和强制关闭
- [x] 健康检查（基础 + 增强）
- [x] 日志文件输出
- [x] 生命周期钩子系统
- [x] 进程锁（防止并发启动）
- [x] 路径解析和绝对路径转换
- [x] 配置验证
- [x] 系统配置备份和清理

### 规划中功能 📋

- [ ] 自动重启（`AutoRestart` 配置已定义但未实现）
- [ ] 定期健康检查（`HealthCheck` 配置已定义但未实现）
- [ ] 日志轮转实现（配置已定义，实际需要外部工具或额外实现）
- [ ] 多实例支持
- [ ] 监控告警集成

## 注意事项和最佳实践

### 1. 路径配置

- 所有路径建议使用绝对路径
- 如果使用相对路径，系统会自动转换为绝对路径
- 日志文件目录必须有写权限

### 2. 密钥管理

- 推荐启用 `auto_generate_secret: true`
- 自动生成的密钥会保存在状态文件中
- 不要在配置文件中硬编码密钥

### 3. 日志管理

- 生产环境建议配置日志文件
- 定期清理旧日志文件
- 注意磁盘空间，避免日志文件过大

### 4. 错误处理

- 启动失败时会自动清理状态
- 健康检查失败会停止守护进程
- 建议查看日志文件排查问题

### 5. 系统配置

- 使用 TUN/TProxy 时会自动备份系统配置
- 停止后会检查残留配置并尝试清理
- 需要管理员权限时会有明确提示

## 常见问题

### Q1: 守护进程启动后立即退出？

**可能原因**:
- 配置文件错误
- 端口被占用
- 可执行文件权限问题

**排查方法**:
1. 查看日志文件内容
2. 检查 Mihomo 配置文件
3. 验证端口占用情况

### Q2: 无法停止守护进程？

**解决方案**:
1. 尝试优雅关闭：`launcher.Stop(false)`
2. 强制关闭：`launcher.Stop(true)` 或 `mihomo-cli stop -F`
3. 手动删除 PID 文件和状态文件
4. 使用系统工具（任务管理器/kill 命令）终止进程

### Q3: 如何查看守护进程日志？

**Windows**:
```bash
type C:\Users\YourName\.local\share\mihomo\mihomo-daemon.log
```

**Linux/macOS**:
```bash
tail -f /var/log/mihomo/mihomo-daemon.log
```

### Q4: 状态文件在哪里？

状态文件路径由 `PathResolver` 计算，通常位于：
- 项目数据目录下
- 文件名格式：`state-{config_hash}.json`
- 可通过 `config.GetStateFilePath()` 获取具体路径

### Q5: 如何在脚本中使用？

```bash
#!/bin/bash

# 启动
mihomo-cli start

# 等待就绪
sleep 5

# 查询状态
mihomo-cli status -o json

# 执行其他操作...

# 停止
mihomo-cli stop
```

## 相关文档

- [守护进程模式使用指南](./plan/daemon-mode-usage.md)
- [守护进程实现方案](./plan/daemon-mode-implementation.md)
- [守护进程增强计划](./plan/daemon-enhancement-plan.md)
- [配置文件示例](./plan/daemon-config-example.toml)

---

**文档版本**: 1.0  
**分析日期**: 2026-04-12  
**适用版本**: mihomo-cli 当前版本
