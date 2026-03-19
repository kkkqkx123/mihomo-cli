# 当前项目实现情况对比分析

## 概述

本文档对照 `mihomo-1.19.21\docs\system-proxy-modifications.md` 和 `docs\fault\unrecoverable-operations-analysis.md`，分析当前项目对于主要问题来源的处理情况，并提出额外的备份建议。

---

## 一、已实现的功能 ✅

### 1. 系统代理管理（sysproxy）

**实现文件：** `internal/sysproxy/sysproxy_windows.go`

**已实现功能：**
- ✅ Windows注册表操作
- ✅ 系统代理启用/禁用
- ✅ 错误处理包含恢复建议
- ✅ `InternetSetOption` 刷新机制（兼容旧版应用）

**已解决问题：**
- ✅ 场景1.1.1：注册表设置残留（通过错误提示指导用户清理）
- ✅ 场景1.1.2：权限不足导致部分配置修改失败（通过详细错误信息）
- ✅ 场景1.2.1：注册表操作失败（提供多种恢复方案）

---

### 2. 进程管理优化

**实现文件：** `internal/mihomo/process_handler.go`

**已实现功能：**
- ✅ 启动前配置检查（TUN/TProxy风险警告）
- ✅ 增强的健康检查（API + 配置状态）
- ✅ 停止后系统配置状态检查
- ✅ 进程退出时捕获错误输出

**已解决问题：**
- ✅ 场景2.1.1：启动成功但健康检查超时（已处理）
- ✅ 场景2.1.3：进程异常退出（已捕获错误输出）
- ✅ 缺少系统配置状态检查（已实现 `SystemChecker.CheckAfterStop()`）

---

### 3. 系统配置管理

**实现文件：** `internal/system/manager.go`

**已实现功能：**
- ✅ 系统配置状态查询（`GetConfigState()`）
- ✅ 统一清理接口（`CleanupAll()`）
- ✅ 配置快照管理（`SnapshotManager`）
- ✅ 审计日志记录（`AuditLogger`）
- ✅ 残留检查（`ValidateState()`）

**已解决问题：**
- ✅ 场景2.2.3：`--all` 参数停止所有进程时的清理不完整
- ✅ 场景3.1.1：重载过程中进程崩溃导致配置不一致（可通过快照恢复）
- ✅ 场景3.1.2：重载后网络配置切换失败

---

### 4. 路由表管理

**实现文件：** `internal/system/route.go`

**已实现功能：**
- ✅ 路由表查询（`ListRoutes()`）
- ✅ 残留路由检测（`CheckMihomoResidualRoutes()`）
- ✅ 默认路由冲突检查（`CheckDefaultRouteConflicts()`）
- ✅ 网络路由诊断（`DiagnoseNetworkRouting()`）
- ✅ 自动修复路由问题（`FixRouteIssues()`）
- ✅ 启动前检查（`CheckBeforeStart()`）
- ✅ 停止后检查（`CheckAfterStop()`）

**已解决问题：**
- ✅ 场景2.1.4：TUN/TProxy配置启动后未正常退出（残留路由检测和清理）
- ✅ 场景5.2：TUN模式 + 系统代理同时启用导致网络混乱（路由冲突检查）

---

### 5. 配置验证

**实现文件：** `internal/config/validator.go`

**已实现功能：**
- ✅ 配置文件语法验证（`ValidateConfigSyntax()`）
- ✅ TUN/TProxy模式检测（`ValidateAndWarn()`）
- ✅ 高风险配置警告（`warnTunEnabled()`, `warnTProxyEnabled()`）

**已解决问题：**
- ✅ 场景4.1.1：编辑后重载失败（配置验证可提前发现问题）
- ✅ 场景4.1.2：编辑后使用 `--no-reload` 参数（验证可发现潜在问题）
- ✅ 缺少配置验证（已实现）

---

### 6. 备份管理

**实现文件：** `internal/config/backup_handler.go`

**已实现功能：**
- ✅ 配置文件备份（`CreateBackup()`）
- ✅ 备份列表查询（`ListBackups()`）
- ✅ 备份恢复（`RestoreBackup()`）
- ✅ 备份清理（`DeleteBackup()`, `PruneBackups()`）
- ✅ 恢复前自动备份（防止误操作）

**已解决问题：**
- ✅ 场景3.1.1：重载过程中进程崩溃（可通过备份恢复）
- ✅ 场景3.2.1：热更新TUN配置后进程崩溃（可通过备份回滚）

---

### 7. 快照管理

**实现文件：** `internal/system/snapshot.go`

**已实现功能：**
- ✅ 配置快照创建（`CreateSnapshot()`）
- ✅ 快照列表查询（`ListSnapshots()`）
- ✅ 快照删除（`DeleteSnapshot()`）
- ✅ 快照清理（`PruneSnapshots()`）
- ✅ 按时间排序（最新的在前）

**已解决问题：**
- ✅ 场景3.1.1：重载过程中进程崩溃（可通过快照恢复）
- ✅ 场景3.1.2：重载后网络配置切换失败

---

### 8. 错误处理

**实现文件：** 多个文件

**已实现功能：**
- ✅ 所有错误信息都包含恢复建议
- ✅ 提供具体的清理命令
- ✅ 操作失败时指导用户如何恢复

**已解决问题：**
- ✅ 缺少错误处理提示（已实现）

---

## 二、未完全解决的问题 ⚠️

### 1. 缺少优雅关闭机制

**问题来源：** `docs\fault\unrecoverable-operations-analysis.md` - 根本原因6.1

**当前状态：**
- ❌ 仍使用 `proc.Kill()` 强制终止进程
- ❌ 没有通过API发送优雅关闭请求
- ❌ 没有设置超时等待进程自然退出

**影响：**
- 场景2.1.2：启动后立即收到中断信号，进程被强制终止
- 场景2.1.3：进程异常退出
- 场景2.1.4：TUN/TProxy配置启动后未正常退出

**代码位置：**
```go
// internal/mihomo/manager.go
if err := proc.Kill(); err != nil {
    return pkgerrors.ErrService("failed to stop process", err)
}
```

**建议改进：**
```go
// 1. 优先通过API发送优雅关闭请求
if err := client.Shutdown(ctx); err == nil {
    // 2. 等待进程自然退出（超时10秒）
    if waitForProcessExit(pid, 10*time.Second) {
        return nil
    }
}

// 3. 超时后使用Kill强制终止
if err := proc.Kill(); err != nil {
    return pkgerrors.ErrService("failed to stop process", err)
}
```

---

### 2. 信号处理不完整

**问题来源：** `docs\fault\unrecoverable-operations-analysis.md` - 根本原因6.2

**当前状态：**
- ❌ 信号处理逻辑直接调用 `StopProcessByPID()`
- ❌ 没有先尝试优雅关闭
- ❌ 没有等待进程自然退出

**影响：**
- 场景2.1.2：启动后立即收到中断信号，进程被强制终止

**代码位置：**
```go
// cmd/start.go:runStart()
case sig := <-sigChan:
    // 收到中断信号
    fmt.Printf("\n收到信号 %v，正在停止内核...\n", sig)

    // 通过 PID 文件停止进程
    pm := mihomo.NewProcessManager(cfg)
    if pid, err := pm.GetPIDFromPIDFile(); err == nil {
        if err := mihomo.StopProcessByPID(pid); err != nil {
            fmt.Printf("停止内核失败: %v\n", err)
            return err
        }
    }
```

**建议改进：**
```go
case sig := <-sigChan:
    output.Printf("\n收到信号 %v，正在优雅停止内核...\n", sig)

    // 1. 优先通过API发送优雅关闭请求
    if client != nil {
        if err := client.Shutdown(ctx); err == nil {
            // 2. 等待进程自然退出
            if err := waitForProcessExit(pid, 10*time.Second); err == nil {
                output.Println("内核已优雅退出")
                return nil
            }
        }
    }

    // 3. 超时后强制终止
    output.Warning("优雅关闭超时，正在强制终止...")
    if err := mihomo.StopProcessByPID(pid); err != nil {
        return err
    }

    // 4. 检查系统配置是否清理
    checker := config.NewSystemChecker()
    if err := checker.CheckAfterStop(); err != nil {
        output.Warning("系统配置未完全清理: " + err.Error())
    }
```

---

### 3. 停止后配置清理验证不完整

**问题来源：** `docs\fault\unrecoverable-operations-analysis.md` - 场景2.2.1

**当前状态：**
- ✅ 已实现 `SystemChecker.CheckAfterStop()`
- ⚠️ 但在某些场景下可能不完整（如TUN虚拟网卡）

**影响：**
- Windows：TUN虚拟网卡可能残留
- Linux：iptables规则可能残留

**建议改进：**
```go
// 在 StopProcessByPID() 后添加
func (pm *ProcessManager) CleanupAfterStop() error {
    // 1. 等待进程完全退出
    if err := waitForProcessExit(pid, 5*time.Second); err != nil {
        return err
    }

    // 2. 检查系统配置状态
    checker := config.NewSystemChecker()
    if err := checker.CheckAfterStop(); err != nil {
        // 如果发现残留配置，尝试自动清理
        scm, _ := system.NewSystemConfigManager()
        if err := scm.CleanupAll(); err != nil {
            return fmt.Errorf("failed to cleanup system config: %w", err)
        }
    }

    return nil
}
```

---

## 三、需要额外备份的操作 📋

### 1. 路由表备份并持久化

**优先级：** 高

**原因：**
- TUN/TProxy模式会修改路由表
- 进程异常退出时路由表可能残留
- 残留路由可能导致网络问题

**建议实现：**

```go
// internal/system/route.go

// BackupRoutes 备份路由表
func (rm *RouteManager) BackupRoutes() (*RouteBackup, error) {
    routes, err := rm.ListRoutes()
    if err != nil {
        return nil, err
    }

    backup := &RouteBackup{
        Timestamp: time.Now(),
        Routes:    routes,
    }

    // 持久化到文件
    dataDir, err := config.GetDataDir()
    if err != nil {
        return nil, err
    }

    backupFile := filepath.Join(dataDir, "routes-backup.json")
    data, err := json.MarshalIndent(backup, "", "  ")
    if err != nil {
        return nil, err
    }

    if err := os.WriteFile(backupFile, data, 0644); err != nil {
        return nil, err
    }

    return backup, nil
}

// RestoreRoutes 恢复路由表
func (rm *RouteManager) RestoreRoutes() error {
    dataDir, err := config.GetDataDir()
    if err != nil {
        return err
    }

    backupFile := filepath.Join(dataDir, "routes-backup.json")
    data, err := os.ReadFile(backupFile)
    if err != nil {
        return err
    }

    var backup RouteBackup
    if err := json.Unmarshal(data, &backup); err != nil {
        return err
    }

    // 恢复路由表
    // 注意：这需要谨慎处理，避免删除系统路由
    for _, route := range backup.Routes {
        if isMihomoRoute(route) {
            // 只恢复Mihomo添加的路由
            if err := rm.DeleteRoute(route); err != nil {
                // 记录错误但继续
            }
        }
    }

    return nil
}
```

**使用场景：**
- 启动TUN/TProxy前备份当前路由表
- 停止Mihomo后检查是否有残留路由
- 如果有残留，对比备份文件进行清理

---

### 2. 注册表备份（Windows）

**优先级：** 中

**原因：**
- 系统代理设置存储在注册表
- 进程异常退出时可能残留
- 残留设置可能导致网络问题

**建议实现：**

```go
// internal/sysproxy/sysproxy_windows.go

// BackupRegistrySettings 备份注册表设置
func (sp *windowsSysProxy) BackupRegistrySettings() (*ProxySettings, error) {
    wr, err := NewWindowsRegistry()
    if err != nil {
        return nil, err
    }
    defer wr.Close()

    return wr.GetSettings()
}

// RestoreRegistrySettings 恢复注册表设置
func (sp *windowsSysProxy) RestoreRegistrySettings(settings *ProxySettings) error {
    wr, err := NewWindowsRegistry()
    if err != nil {
        return err
    }
    defer wr.Close()

    return wr.SetSettings(settings)
}
```

**使用场景：**
- 启用系统代理前备份当前设置
- 禁用系统代理时恢复原始设置
- 异常退出后检查并恢复

---

### 3. iptables规则备份（Linux）

**优先级：** 高（Linux平台）

**原因：**
- TProxy模式会修改iptables规则
- 进程异常退出时规则可能残留
- 残留规则可能导致网络问题

**建议实现：**

```go
// internal/sysproxy/sysproxy_linux.go

// BackupIPTablesRules 备份iptables规则
func (sp *linuxSysProxy) BackupIPTablesRules() ([]string, error) {
    // 备份 mangle 表规则
    cmd := exec.Command("iptables-save", "-t", "mangle")
    output, err := cmd.CombinedOutput()
    if err != nil {
        return nil, err
    }

    var rules []string
    scanner := bufio.NewScanner(strings.NewReader(string(output)))
    for scanner.Scan() {
        line := scanner.Text()
        if strings.Contains(line, "mihomo") {
            rules = append(rules, line)
        }
    }

    return rules, nil
}

// RestoreIPTablesRules 恢复iptables规则
func (sp *linuxSysProxy) RestoreIPTablesRules() error {
    // 删除所有Mihomo相关的规则
    cmd := exec.Command("iptables", "-t", "mangle", "-F", "mihomo_prerouting")
    cmd.Run()

    cmd = exec.Command("iptables", "-t", "mangle", "-F", "mihomo_output")
    cmd.Run()

    cmd = exec.Command("iptables", "-t", "mangle", "-F", "mihomo_divert")
    cmd.Run()

    return nil
}
```

---

### 4. DNS配置备份

**优先级：** 中

**原因：**
- Mihomo可能会修改DNS设置
- 启用DNS重定向时会影响系统DNS
- 异常退出时DNS设置可能残留

**建议实现：**

```go
// internal/system/dns.go

// BackupDNSConfig 备份DNS配置
func (scm *SystemConfigManager) BackupDNSConfig() (*DNSConfig, error) {
    // 获取当前DNS服务器配置
    dnsServers, err := getDNSServers()
    if err != nil {
        return nil, err
    }

    return &DNSConfig{
        Timestamp: time.Now(),
        Servers:   dnsServers,
    }, nil
}

// RestoreDNSConfig 恢复DNS配置
func (scm *SystemConfigManager) RestoreDNSConfig(config *DNSConfig) error {
    // 恢复DNS服务器配置
    return setDNSServers(config.Servers)
}
```

---

### 5. TUN接口状态备份

**优先级：** 高

**原因：**
- TUN模式创建虚拟网卡
- 进程异常退出时网卡可能残留
- 残留网卡可能导致路由问题

**建议实现：**

```go
// internal/system/tun.go

// BackupTUNState 备份TUN接口状态
func (tm *TUNManager) BackupTUNState() (*TUNState, error) {
    state := &TUNState{
        Timestamp: time.Now(),
        Exists:    false,
    }

    // 检查TUN接口是否存在
    iface, err := getTUNInterface()
    if err != nil {
        // 接口不存在，记录为不存在
        return state, nil
    }

    state.Exists = true
    state.Interface = iface.Name
    state.IPAddress = iface.IPAddress
    state.Netmask = iface.Netmask
    state.MTU = iface.MTU

    return state, nil
}

// RestoreTUNState 恢复TUN接口状态
func (tm *TUNManager) RestoreTUNState(state *TUNState) error {
    // 如果备份时接口不存在，现在存在，则删除
    if !state.Exists {
        return tm.Cleanup()
    }

    // 如果备份时接口存在，现在不存在，则不处理
    // （Mihomo启动时会创建新接口）

    return nil
}
```

---

### 6. 配置文件操作前备份

**优先级：** 中

**原因：**
- 配置重载可能失败
- 配置编辑可能引入错误
- 需要能够回滚到之前的状态

**当前状态：** ✅ 已实现（`internal/config/backup_handler.go`）

**建议改进：**
- 在重载前自动创建备份
- 在编辑前自动创建备份
- 提供配置文件差异比较

---

### 7. PID文件备份

**优先级：** 低

**原因：**
- PID文件用于追踪进程状态
- 文件损坏可能导致无法正确管理进程
- 需要能够恢复PID文件

**建议实现：**

```go
// internal/mihomo/manager.go

// BackupPIDFile 备份PID文件
func (pm *ProcessManager) BackupPIDFile() error {
    pid, err := pm.GetPIDFromPIDFile()
    if err != nil {
        return err
    }

    dataDir, err := config.GetDataDir()
    if err != nil {
        return err
    }

    backupFile := filepath.Join(dataDir, "pid-backup.txt")
    return os.WriteFile(backupFile, []byte(fmt.Sprintf("%d", pid)), 0644)
}

// RestorePIDFile 恢复PID文件
func (pm *ProcessManager) RestorePIDFile() error {
    dataDir, err := config.GetDataDir()
    if err != nil {
        return err
    }

    backupFile := filepath.Join(dataDir, "pid-backup.txt")
    data, err := os.ReadFile(backupFile)
    if err != nil {
        return err
    }

    return os.WriteFile(pm.pidFile, data, 0644)
}
```

---

## 四、建议的改进优先级

### 高优先级（立即实施）

1. **实现优雅关闭机制**
   - 优先通过API发送优雅关闭请求
   - 设置超时时间（10秒）
   - 超时后再使用Kill强制终止

2. **改进信号处理**
   - 捕获信号后先尝试优雅关闭
   - 等待进程自然退出
   - 超时后再强制终止

3. **实现路由表备份**
   - 启动TUN/TProxy前备份
   - 停止后检查残留
   - 提供自动恢复功能

4. **实现TUN接口状态备份**
   - 记录TUN接口是否存在
   - 停止后检查残留
   - 提供自动清理功能

### 中优先级（1-2周内实施）

1. **实现注册表备份（Windows）**
   - 启用系统代理前备份
   - 禁用时恢复原始设置

2. **实现iptables规则备份（Linux）**
   - 启动TProxy前备份
   - 停止后检查残留

3. **实现DNS配置备份**
   - 记录DNS服务器配置
   - 恢复原始DNS设置

4. **改进停止后配置清理验证**
   - 确保所有配置都已清理
   - 发现残留时自动清理

### 低优先级（1个月内实施）

1. **实现PID文件备份**
   - 防止PID文件损坏
   - 提供恢复功能

2. **实现配置文件差异比较**
   - 帮助用户理解配置变更
   - 提供配置审查功能

---

## 五、总结

### 已解决的主要问题

✅ 缺少系统配置状态检查
✅ 缺少配置验证
✅ 错误处理不完善
✅ 缺少配置管理功能（备份、快照、审计）
✅ 缺少路由管理功能
✅ 缺少残留检测功能

### 仍需解决的问题

❌ 缺少优雅关闭机制
❌ 信号处理不完整
❌ 停止后配置清理验证不完整

### 需要额外备份的操作

1. 路由表备份（高优先级）
2. TUN接口状态备份（高优先级）
3. 注册表备份（中优先级）
4. iptables规则备份（中优先级）
5. DNS配置备份（中优先级）
6. 配置文件操作前备份（已实现）
7. PID文件备份（低优先级）

---

## 六、参考文档

- `mihomo-1.19.21\docs\system-proxy-modifications.md` - Mihomo内核系统代理配置修改分析
- `docs\fault\unrecoverable-operations-analysis.md` - Mihomo CLI不可恢复问题分析
- `internal/sysproxy/sysproxy_windows.go` - Windows系统代理实现
- `internal/mihomo/process_handler.go` - 进程管理处理器
- `internal/system/manager.go` - 系统配置管理器
- `internal/system/route.go` - 路由表管理器
- `internal/config/validator.go` - 配置验证器
- `internal/config/backup_handler.go` - 备份处理器
- `internal/system/snapshot.go` - 快照管理器