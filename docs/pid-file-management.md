# PID 文件管理文档

## 概述

Mihomo CLI 使用 PID 文件实现跨进程状态管理。由于 CLI 是无状态的（每次执行都是新进程），需要通过持久化的 PID 文件来跟踪 Mihomo 内核进程。

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
- ✅ `status` 命令：查询运行状态
- ✅ `cleanup` 命令：清理残留文件
- ✅ 健康检查超时：停止已启动的进程
- ✅ 信号处理：用户按 Ctrl+C 时停止进程

## 存储路径

### 默认路径

```
~/.config/.mihomo-cli/
├── mihomo.pid           # 默认配置（未指定配置文件）
├── mihomo-{hash}.pid    # 指定配置文件的实例
└── backups/             # 配置备份目录
```

### 路径生成规则

**基础目录**：
- 首选：`~/.config/.mihomo-cli/`（用户主目录）
- 备选：`%TEMP%/.mihomo-cli/`（无法获取用户目录时）

**PID 文件名**：
- 未指定配置文件：`mihomo.pid`
- 指定配置文件：`mihomo-{hash}.pid`

### Hash 生成规则

Hash 基于配置文件路径生成，确保同一配置文件使用同一 PID 文件：

```go
func generateConfigHash(configFile string) string {
    // 1. 获取配置文件的绝对路径
    absPath := filepath.Abs(configFile)

    // 2. 提取文件名（不含扩展名）
    filename := filepath.Base(absPath)
    nameWithoutExt := strings.TrimSuffix(filename, filepath.Ext(filename))

    // 3. 限制长度（最多 8 个字符）
    if len(nameWithoutExt) > 8 {
        nameWithoutExt = nameWithoutExt[:8]
    }

    // 4. 空名称使用默认值
    if nameWithoutExt == "" {
        nameWithoutExt = "default"
    }

    return nameWithoutExt
}
```

### 示例

| 配置文件路径 | PID 文件路径 |
|-------------|-------------|
| (未指定) | `~/.config/.mihomo-cli/mihomo.pid` |
| `config.yaml` | `~/.config/.mihomo-cli/mihomo-config.pid` |
| `scripts/mihomo-config.yaml` | `~/.config/.mihomo-cli/mihomo-mihomo-co.pid` |
| `E:\project\mihomo-go\test-config.yaml` | `~/.config/.mihomo-cli/mihomo-test-con.pid` |

## PID 文件生命周期

### 创建

**时机**：Mihomo 内核启动成功后

**位置**：`internal/mihomo/manager.go:Start()`

```go
// 1. 保存 PID 到文件
if err := pm.SavePID(pm.process.Pid); err != nil {
    fmt.Printf("Warning: failed to save pid file: %v\n", err)
}

// 2. 启动后台监控，进程退出时自动删除
go func() {
    err := pm.cmd.Wait()
    os.Remove(pm.pidFile)  // 进程退出时删除
}()
```

**内容**：纯文本，仅包含进程 ID

```
18296
```

### 读取

**方法**：`ProcessManager.GetPIDFromPIDFile()`

**验证流程**：
1. 读取 PID 文件内容
2. 解析为整数
3. 检查进程是否真实存在（使用 Windows API）
4. 如果进程不存在，返回错误

```go
func (pm *ProcessManager) GetPIDFromPIDFile() (int, error) {
    pid, err := pm.ReadPID()
    if err != nil {
        return 0, err
    }

    // 验证进程是否真的在运行
    if !IsProcessRunning(pid) {
        return 0, pkgerrors.ErrService("process ... is not running", nil)
    }

    return pid, nil
}
```

### 删除

**触发时机**：

1. **正常停止**：`stop` 命令执行成功后
   ```go
   os.Remove(pm.pidFile)
   ```

2. **进程退出**：进程监控 goroutine 检测到进程退出
   ```go
   go func() {
       pm.cmd.Wait()
       os.Remove(pm.pidFile)
   }()
   ```

3. **健康检查超时**：启动超时清理
   ```go
   if pid, err := pm.GetPIDFromPIDFile(); err == nil {
       StopProcessByPID(pid)
       // PID 文件会在进程退出时被监控 goroutine 删除
   }
   ```

4. **信号中断**：用户按 Ctrl+C
   ```go
   if pid, err := pm.GetPIDFromPIDFile(); err == nil {
       StopProcessByPID(pid)
   }
   ```

5. **手动清理**：`cleanup` 命令

## 各命令使用情况

### start 命令

**用途**：
1. 检测重复启动
2. 保存新进程的 PID

**流程**：
```
1. 读取 PID 文件
2. 如果存在且进程运行 → 拒绝启动
3. 启动进程
4. 保存新 PID
5. 启动后台监控（进程退出时删除 PID）
```

**相关代码**：
- `process_handler.go:Start()` - 重复启动检测
- `manager.go:Start()` - 保存 PID

### stop 命令

**用途**：
1. 定位要停止的进程
2. 删除 PID 文件

**流程**：
```
1. 读取 PID 文件
2. 如果不存在或进程不运行 → 返回错误
3. 通过 PID 停止进程
4. 删除 PID 文件
```

**相关代码**：
- `process_handler.go:Stop()` - 读取 PID 并停止进程

### status 命令

**用途**：
1. 查询进程运行状态

**流程**：
```
1. 读取 PID 文件
2. 如果不存在 → 返回"未运行"
3. 验证进程是否存在
4. 返回运行状态和 PID
```

**相关代码**：
- `process_handler.go:Status()` - 查询状态

### cleanup 命令

**用途**：
1. 清理所有残留的 PID 文件

**流程**：
```
1. 扫描 ~/.mihomo-cli/ 目录
2. 遍历所有 .pid 文件
3. 验证进程是否存在
4. 删除无效的 PID 文件：
   - 进程不存在的
   - 文件损坏的
```

**相关代码**：
- `scanner.go:CleanupPIDFiles()` - 清理残留文件

## 异常处理

### 残留文件产生原因

1. **进程崩溃**：进程异常退出，监控 goroutine 未执行
2. **系统重启**：系统重启，PID 文件保留但进程已退出
3. **进程被外部杀死**：任务管理器、第三方工具
4. **启动超时**：健康检查超时，进程已启动但未响应
5. **文件系统错误**：写入 PID 失败，但进程已启动

### 清理策略

**手动清理**：
```bash
mihomo-cli cleanup
```

**自动清理**：
- `cleanup` 命令会自动扫描并清理所有无效的 PID 文件
- 验证进程是否存在后再删除

### 安全机制

**双重验证**：
1. PID 文件存在
2. 进程真实运行

```go
func (pm *ProcessManager) GetPIDFromPIDFile() (int, error) {
    pid, err := pm.ReadPID()
    if err != nil {
        return 0, err
    }

    // 关键：验证进程是否真实存在
    if !IsProcessRunning(pid) {
        return 0, pkgerrors.ErrService("process ... is not running", nil)
    }

    return pid, nil
}
```

## Windows 进程检测

### 实现方式

使用 Windows API 检查进程是否存在：

```go
func isProcessRunningWindows(pid int) bool {
    // 使用 PROCESS_QUERY_INFORMATION 权限打开进程
    handle, _, _ := procOpenProcess.Call(
        uintptr(PROCESS_QUERY_INFORMATION),
        0,
        uintptr(pid),
    )
    if handle == 0 {
        return false
    }
    procCloseHandle.Call(handle)
    return true
}
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

**文件权限**：`0644`（rw-r--r--）
- 所有者：读写
- 组和他人：只读

## 故障排查

### 问题：显示"未运行"但进程存在

**可能原因**：
1. PID 文件损坏
2. PID 文件不存在
3. 配置文件路径变化导致 PID 文件不匹配

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

# 方法3：手动通过 PID 停止
mihomo-cli stop <PID>
```

## 相关文件

- `internal/mihomo/manager.go` - PID 文件路径生成、读写逻辑
- `internal/mihomo/process_handler.go` - 各命令的 PID 文件使用
- `internal/mihomo/scanner.go` - 残留文件清理、进程检测
- `cmd/start.go` - start 命令的 PID 文件使用
- `cmd/stop.go` - stop 命令的 PID 文件使用
- `cmd/status.go` - status 命令的 PID 文件使用
- `cmd/cleanup.go` - cleanup 命令实现

## 参考资料

- 无状态 CLI 设计原则
- Windows 进程管理 API
- Unix PID 文件最佳实践