# 守护进程模式实现方案

## 文档信息

- **创建日期**: 2026-04-01
- **版本**: 1.0
- **状态**: 分析阶段
- **目标**: 实现 CLI 客户端与 Mihomo 内核的完全分离

---

## 一、问题背景

### 1.1 当前问题

在当前的实现中，Mihomo 内核与 CLI 程序存在以下耦合问题：

1. **信号处理绑定**：CLI 程序监听 `SIGINT` 和 `SIGTERM` 信号，终端关闭时会主动停止内核
2. **进程依赖关系**：内核作为子进程启动，属于 CLI 进程的进程组
3. **输出管道绑定**：子进程的 stdout/stderr 重定向到父进程缓冲区
4. **终端关闭影响**：终端关闭时，`SIGHUP` 信号会被转发到整个进程组

### 1.2 问题代码位置

- **信号监听**: `cmd/start.go:88-91`
- **信号处理逻辑**: `cmd/start.go:104-174`
- **进程启动**: `internal/mihomo/manager.go:80-86`
- **输出重定向**: `internal/mihomo/manager.go:83-86`

### 1.3 影响范围

- 用户关闭终端时，Mihomo 内核会被意外停止
- 无法实现真正的后台运行
- 影响用户体验和系统稳定性

---

## 二、守护进程模式设计

### 2.1 守护进程特性

守护进程应该具备以下特性：

1. **独立进程组**：创建新的进程组，脱离父进程控制
2. **输出重定向**：将 stdin/stdout/stderr 重定向到 `/dev/null` 或日志文件
3. **会话分离**：脱离控制终端，避免 SIGHUP 信号影响
4. **进程持久化**：父进程退出后仍能继续运行
5. **生命周期管理**：通过 PID 文件和 API 进行管理

### 2.2 架构设计

```
┌─────────────────┐
│   CLI Client    │
│  (临时进程)     │
└────────┬────────┘
         │ 1. 启动命令
         ▼
┌─────────────────┐
│  Daemon Manager │
│  (启动器)       │
└────────┬────────┘
         │ 2. 创建守护进程
         ▼
┌─────────────────┐
│  Mihomo Kernel  │
│  (独立进程组)   │
│  - 重定向 I/O   │
│  - 独立会话     │
└────────┬────────┘
         │ 3. 返回 PID
         ▼
┌─────────────────┐
│  CLI Client     │
│  (退出)         │
└─────────────────┘
```

### 2.3 核心组件

#### 2.3.1 DaemonManager（守护进程管理器）

```go
type DaemonManager struct {
    config    *config.TomlConfig
    pidFile   string
    logFile   string
    workDir   string
}
```

**职责**：
- 创建守护进程
- 管理进程生命周期
- 处理日志输出
- 维护 PID 文件

#### 2.3.2 ProcessGroupManager（进程组管理器）

```go
type ProcessGroupManager interface {
    CreateProcessGroup() error
    DetachFromTerminal() error
    RedirectIO(logFile string) error
}
```

**职责**：
- 创建独立进程组
- 脱离控制终端
- 重定向标准输入输出

---

## 三、多平台实现方案

### 3.1 平台差异分析

| 特性 | Windows | Linux | macOS |
|------|---------|-------|-------|
| 进程组创建 | `CREATE_NEW_PROCESS_GROUP` | `setsid()` / `setpgid()` | `setsid()` / `setpgid()` |
| 终端分离 | 不适用（无终端概念） | `setsid()` | `setsid()` |
| I/O 重定向 | 文件句柄 | `/dev/null` | `/dev/null` |
| 信号处理 | Windows 信号机制 | POSIX 信号 | POSIX 信号 |
| 服务集成 | Windows Service | systemd | launchd |

### 3.2 Windows 平台实现

#### 3.2.1 进程创建

```go
// process_daemon_windows.go

type WindowsDaemonManager struct {
    DaemonManager
}

// StartAsDaemon 以守护进程方式启动
func (wdm *WindowsDaemonManager) StartAsDaemon() error {
    // 使用 CREATE_NEW_PROCESS_GROUP 标志创建新进程组
    cmd := exec.Command(wdm.config.Mihomo.Executable, "-f", configFile)

    // 设置进程创建标志
    cmd.SysProcAttr = &syscall.SysProcAttr{
        CreationFlags: syscall.CREATE_NEW_PROCESS_GROUP,
        HideWindow:    true, // 隐藏窗口
    }

    // 重定向输出到日志文件
    if wdm.logFile != "" {
        logFH, err := os.OpenFile(wdm.logFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
        if err != nil {
            return err
        }
        defer logFH.Close()

        cmd.Stdout = logFH
        cmd.Stderr = logFH
    } else {
        // 重定向到 NUL（Windows 的 /dev/null）
        nul, err := os.OpenFile("NUL", os.O_RDWR, 0)
        if err != nil {
            return err
        }
        defer nul.Close()

        cmd.Stdout = nul
        cmd.Stderr = nul
    }

    // 重定向 stdin 到 NUL
    nul, _ := os.OpenFile("NUL", os.O_RDONLY, 0)
    cmd.Stdin = nul

    // 启动进程
    if err := cmd.Start(); err != nil {
        return err
    }

    // 保存 PID
    return wdm.SavePID(cmd.Process.Pid)
}
```

#### 3.2.2 进程组隔离

```go
// Windows 使用 CREATE_NEW_PROCESS_GROUP 标志
// 这样创建的进程不会收到父进程的 Ctrl+C 信号

// 如果需要更严格的隔离，可以考虑使用 Windows Job Objects
type WindowsJobObjectManager struct {
    jobHandle syscall.Handle
}

func NewWindowsJobObjectManager() *WindowsJobObjectManager {
    return &WindowsJobObjectManager{}
}

func (j *WindowsJobObjectManager) CreateJobObject() error {
    // 创建 Job Object
    // 设置 JOB_OBJECT_LIMIT_KILL_ON_JOB_CLOSE
    // 这样当最后一个句柄关闭时，所有关联进程都会被终止
    return nil
}
```

### 3.3 Linux 平台实现

#### 3.3.1 进程创建

```go
// process_daemon_linux.go

type LinuxDaemonManager struct {
    DaemonManager
}

// StartAsDaemon 以守护进程方式启动
func (ldm *LinuxDaemonManager) StartAsDaemon() error {
    // 使用 syscall.SysProcAttr 设置进程组
    cmd := exec.Command(ldm.config.Mihomo.Executable, "-f", configFile)

    cmd.SysProcAttr = &syscall.SysProcAttr{
        Setsid: true, // 创建新会话
        Setpgid: true, // 创建新进程组
    }

    // 设置工作目录
    cmd.Dir = ldm.workDir

    // 重定向 I/O
    if err := ldm.RedirectIO(cmd); err != nil {
        return err
    }

    // 启动进程
    if err := cmd.Start(); err != nil {
        return err
    }

    // 保存 PID
    return ldm.SavePID(cmd.Process.Pid)
}

// RedirectIO 重定向标准输入输出
func (ldm *LinuxDaemonManager) RedirectIO(cmd *exec.Cmd) error {
    // 重定向 stdin 到 /dev/null
    if ldm.logFile != "" {
        // 重定向到日志文件
        logFH, err := os.OpenFile(ldm.logFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
        if err != nil {
            return err
        }
        cmd.Stdout = logFH
        cmd.Stderr = logFH
    } else {
        // 重定向到 /dev/null
        devNull, err := os.OpenFile("/dev/null", os.O_RDWR, 0)
        if err != nil {
            return err
        }
        cmd.Stdout = devNull
        cmd.Stderr = devNull
    }

    // 重定向 stdin 到 /dev/null
    devNull, _ := os.OpenFile("/dev/null", os.O_RDONLY, 0)
    cmd.Stdin = devNull

    return nil
}
```

#### 3.3.2 传统守护进程方式（可选）

对于需要更严格的守护进程行为，可以使用传统的 double-fork 方法：

```go
// StartAsTraditionalDaemon 使用传统的 double-fork 方法
func (ldm *LinuxDaemonManager) StartAsTraditionalDaemon() error {
    // 第一个 fork
    cmd := exec.Command(ldm.config.Mihomo.Executable, "-f", configFile)
    cmd.SysProcAttr = &syscall.SysProcAttr{
        Setpgid: true,
    }

    if err := cmd.Start(); err != nil {
        return err
    }

    // 等待第一个子进程完成 double-fork
    go func() {
        cmd.Wait()
    }()

    // 保存 PID（注意：这里需要从守护进程中获取实际的 PID）
    // 可以通过进程间通信（IPC）或共享文件获取

    return nil
}
```

### 3.4 macOS 平台实现

#### 3.4.1 进程创建

macOS 的实现与 Linux 类似，因为它们都是 Unix-like 系统：

```go
// process_daemon_darwin.go

type DarwinDaemonManager struct {
    DaemonManager
}

// StartAsDaemon 以守护进程方式启动
func (ddm *DarwinDaemonManager) StartAsDaemon() error {
    // macOS 与 Linux 类似
    cmd := exec.Command(ddm.config.Mihomo.Executable, "-f", configFile)

    cmd.SysProcAttr = &syscall.SysProcAttr{
        Setsid: true,
        Setpgid: true,
    }

    cmd.Dir = ddm.workDir

    if err := ddm.RedirectIO(cmd); err != nil {
        return err
    }

    if err := cmd.Start(); err != nil {
        return err
    }

    return ddm.SavePID(cmd.Process.Pid)
}

// RedirectIO 重定向标准输入输出
func (ddm *DarwinDaemonManager) RedirectIO(cmd *exec.Cmd) error {
    // 与 Linux 实现相同
    return nil
}
```

#### 3.4.2 launchd 集成（可选）

macOS 原生支持通过 launchd 管理守护进程：

```go
// LaunchdManager launchd 守护进程管理器
type LaunchdManager struct {
    plistPath string
    label     string
}

// CreatePlist 创建 launchd plist 文件
func (lm *LaunchdManager) CreatePlist(config *config.TomlConfig) error {
    plistContent := fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>Label</key>
    <string>%s</string>
    <key>ProgramArguments</key>
    <array>
        <string>%s</string>
        <string>-f</string>
        <string>%s</string>
    </array>
    <key>RunAtLoad</key>
    <true/>
    <key>KeepAlive</key>
    <true/>
    <key>StandardOutPath</key>
    <string>%s</string>
    <key>StandardErrorPath</key>
    <string>%s</string>
    <key>WorkingDirectory</key>
    <string>%s</string>
</dict>
</plist>`,
        lm.label,
        config.Mihomo.Executable,
        config.Mihomo.ConfigFile,
        lm.GetLogPath("stdout.log"),
        lm.GetLogPath("stderr.log"),
        filepath.Dir(config.Mihomo.Executable),
    )

    return os.WriteFile(lm.plistPath, []byte(plistContent), 0644)
}

// Load 加载守护进程
func (lm *LaunchdManager) Load() error {
    cmd := exec.Command("launchctl", "load", lm.plistPath)
    return cmd.Run()
}

// Unload 卸载守护进程
func (lm *LaunchdManager) Unload() error {
    cmd := exec.Command("launchctl", "unload", lm.plistPath)
    return cmd.Run()
}
```

---

## 四、实现步骤

### 4.1 第一阶段：基础守护进程功能

#### 目标
实现基本的守护进程启动功能，确保 CLI 退出后内核继续运行。

#### 任务

1. **创建 DaemonManager 接口**
   - 定义统一的守护进程管理接口
   - 实现平台特定的管理器

2. **修改启动逻辑**
   - 移除 `cmd/start.go` 中的信号监听
   - 使用 DaemonManager 启动内核
   - 保留健康检查机制

3. **实现 I/O 重定向**
   - 将 stdout/stderr 重定向到日志文件
   - 将 stdin 重定向到 `/dev/null`（或 NUL）

4. **测试验证**
   - 验证终端关闭后内核继续运行
   - 验证日志输出正常
   - 验证 PID 文件正确

#### 代码结构

```
internal/mihomo/
├── daemon.go                    # 守护进程接口定义
├── daemon_manager.go            # 守护进程管理器（平台无关）
├── daemon_windows.go            # Windows 实现
├── daemon_linux.go              # Linux 实现
├── daemon_darwin.go             # macOS 实现
└── daemon_other.go              # 其他平台实现
```

### 4.2 第二阶段：日志管理

#### 目标
实现完善的日志管理，支持日志轮转和级别控制。

#### 任务

1. **日志文件管理**
   - 支持自定义日志路径
   - 实现日志文件轮转
   - 支持日志级别过滤

2. **日志格式化**
   - 统一日志格式
   - 添加时间戳和进程信息
   - 支持结构化日志

3. **日志查询**
   - 提供 `mihomo-cli logs` 命令
   - 支持实时日志查看（tail -f）
   - 支持日志搜索和过滤

#### 配置示例

```toml
[mihomo]
enabled = true
executable = "/path/to/mihomo"
config_file = "/path/to/config.yaml"

[daemon]
enabled = true
log_file = "/var/log/mihomo/mihomo.log"
log_level = "info"
log_max_size = "100M"
log_max_backups = 10
log_max_age = 30
```

### 4.3 第三阶段：系统集成

#### 目标
与系统原生服务管理集成，提供更好的用户体验。

#### 任务

1. **Windows Service 集成**
   - 创建 Windows 服务
   - 支持服务安装/卸载
   - 支持服务启动/停止

2. **systemd 集成（Linux）**
   - 创建 systemd service 文件
   - 支持服务安装/卸载
   - 支持服务启动/停止/重启

3. **launchd 集成（macOS）**
   - 创建 launchd plist 文件
   - 支持服务加载/卸载
   - 支持服务启动/停止

#### 命令示例

```bash
# 安装为系统服务
mihomo-cli service install

# 启动服务
mihomo-cli service start

# 停止服务
mihomo-cli service stop

# 查看服务状态
mihomo-cli service status

# 卸载服务
mihomo-cli service uninstall
```

### 4.4 第四阶段：增强功能

#### 目标
提供更多高级功能，提升用户体验。

#### 任务

1. **自动重启**
   - 进程崩溃时自动重启
   - 可配置重启策略
   - 记录崩溃日志

2. **健康监控**
   - 定期健康检查
   - 异常告警
   - 自动恢复

3. **资源限制**
   - CPU 限制
   - 内存限制
   - 网络限制

4. **多实例管理**
   - 支持运行多个实例
   - 实例隔离
   - 统一管理

---

## 五、配置设计

### 5.1 配置文件结构

```toml
[mihomo]
enabled = true
executable = "/path/to/mihomo"
config_file = "/path/to/config.yaml"
auto_generate_secret = true

[daemon]
enabled = true
work_dir = "/var/lib/mihomo"
log_file = "/var/log/mihomo/mihomo.log"
log_level = "info"
log_max_size = "100M"
log_max_backups = 10
log_max_age = 30

[daemon.auto_restart]
enabled = true
max_restarts = 5
restart_delay = "5s"

[daemon.health_check]
enabled = true
interval = "30s"
timeout = "10s"

[api]
external_controller = "127.0.0.1:9090"
secret = ""
```

### 5.2 环境变量支持

```bash
# 启用守护进程模式
export MIHOMO_DAEMON_ENABLED=true

# 设置日志路径
export MIHOMO_DAEMON_LOG_FILE=/var/log/mihomo/mihomo.log

# 设置日志级别
export MIHOMO_DAEMON_LOG_LEVEL=info
```

---

## 六、兼容性考虑

### 6.1 向后兼容

- 保留现有的启动方式
- 通过配置文件选择模式
- 默认使用守护进程模式

### 6.2 平台兼容

- 所有平台统一接口
- 平台特定功能可选
- 优雅降级处理

### 6.3 版本兼容

- 支持旧版本配置文件
- 提供迁移工具
- 详细升级文档

---

## 七、测试计划

### 7.1 单元测试

- 守护进程管理器测试
- 进程组管理测试
- I/O 重定向测试
- PID 文件管理测试

### 7.2 集成测试

- 启动/停止流程测试
- 健康检查测试
- 日志输出测试
- 信号处理测试

### 7.3 平台测试

- Windows 测试
- Linux 测试（多个发行版）
- macOS 测试

### 7.4 场景测试

- 正常启动测试
- 终端关闭测试
- 进程崩溃测试
- 资源限制测试

---

## 八、风险评估

### 8.1 技术风险

| 风险 | 影响 | 概率 | 缓解措施 |
|------|------|------|----------|
| 平台兼容性问题 | 高 | 中 | 充分的平台测试 |
| 日志文件权限问题 | 中 | 中 | 提前检查权限 |
| 进程组隔离失败 | 高 | 低 | 多重隔离机制 |
| PID 文件竞争 | 中 | 低 | 文件锁机制 |

### 8.2 运维风险

| 风险 | 影响 | 概率 | 缓解措施 |
|------|------|------|----------|
| 日志文件过大 | 中 | 中 | 实现日志轮转 |
| 进程僵尸化 | 高 | 低 | 定期清理 |
| 资源泄露 | 高 | 低 | 资源监控 |
| 配置错误 | 中 | 中 | 配置验证 |

---

## 九、实施时间表

### 第一阶段（1-2 周）
- 守护进程基础功能
- 平台特定实现
- 基本测试

### 第二阶段（1 周）
- 日志管理
- 日志轮转
- 日志查询

### 第三阶段（2 周）
- 系统服务集成
- 服务管理命令
- 文档更新

### 第四阶段（1-2 周）
- 自动重启
- 健康监控
- 资源限制

### 总计：5-7 周

---

## 十、参考资源

### 10.1 文档

- [Go exec 包文档](https://pkg.go.dev/os/exec)
- [Linux 守护进程编写指南](https://www.freedesktop.org/software/systemd/man/daemon.html)
- [Windows 服务开发](https://docs.microsoft.com/en-us/windows/win32/services/services)
- [macOS launchd 文档](https://developer.apple.com/library/archive/documentation/MacOSX/Conceptual/BPSystemStartup/Chapters/CreatingLaunchdJobs.html)

### 10.2 代码示例

- [Prometheus 守护进程实现](https://github.com/prometheus/prometheus)
- [Nginx 守护进程实现](https://nginx.org/en/docs/ctrl.html)
- [systemd 服务示例](https://www.freedesktop.org/software/systemd/man/systemd.service.html)

### 10.3 最佳实践

- [Linux 系统编程](https://man7.org/tlpi/)
- [Windows 系统编程](https://docs.microsoft.com/en-us/windows/win32/api/)
- [Go 并发编程](https://go.dev/doc/effective_go#concurrency)

---

## 十一、总结

### 11.1 核心目标

实现 CLI 客户端与 Mihomo 内核的完全分离，确保内核作为独立的守护进程运行。

### 11.2 关键技术

- 进程组隔离
- I/O 重定向
- 终端分离
- 日志管理
- 系统服务集成

### 11.3 预期效果

- ✅ 终端关闭后内核继续运行
- ✅ 完善的日志管理
- ✅ 统一的进程管理
- ✅ 跨平台支持
- ✅ 系统服务集成

### 11.4 后续优化

- 性能优化
- 安全加固
- 监控告警
- 自动化运维

---

**文档结束**
