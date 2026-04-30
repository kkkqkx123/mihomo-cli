# 启动前检查功能实现说明

## 修改概述

针对之前分析报告中提出的改进建议，本次修改添加了**启动前检查并清理残留配置**的功能，确保每次启动 Mihomo 前系统状态是干净的。

## 修改内容

### 1. 新增 `checkAndCleanupBeforeStart` 函数

**文件**: `internal/mihomo/process_handler.go`

**功能**: 
- 在启动 Mihomo 前检查系统配置状态
- 检测是否有上次异常退出留下的残留配置（TUN 设备、路由表、系统代理）
- 自动清理所有检测到的残留配置
- 确保系统状态干净后再启动新实例

**代码位置**: `process_handler.go:270-297`

```go
// checkAndCleanupBeforeStart 启动前检查并清理残留配置
func (ph *ProcessHandler) checkAndCleanupBeforeStart(_ *config.TomlConfig) error {
	scm, err := system.NewSystemConfigManager()
	if err != nil {
		return err
	}

	// 检查是否有上次异常退出留下的残留
	problems, err := scm.ValidateState()
	if err != nil {
		return err
	}

	if len(problems) > 0 {
		output.Warning("Detected %d residual configuration issues from abnormal exit", len(problems))
		for _, problem := range problems {
			output.Printf("  - %s (severity: %s)\n", problem.Description, problem.Severity)
		}

		// 尝试自动清理
		output.Info("Cleaning up residual configuration before start...")
		if err := scm.CleanupAll(); err != nil {
			output.Warning("Automatic cleanup failed: " + err.Error())
			output.Println("Manual cleanup may be required")
			return fmt.Errorf("failed to cleanup residual configuration: %w", err)
		}
		output.Success("Automatic cleanup completed, system is now clean")
	}

	return nil
}
```

### 2. 在启动流程中集成检查

**文件**: `internal/mihomo/process_handler.go`

**修改位置**: `Start` 方法中，备份系统配置之前

```go
// 启动前检查并清理残留配置（确保系统状态干净）
output.Info("Checking for residual configuration from abnormal exit...")
if err := ph.checkAndCleanupBeforeStart(cfg); err != nil {
	return nil, pkgerrors.ErrService("failed to cleanup residual configuration", err)
}

// 启动前备份系统配置
if hasTUN || hasTProxy {
	output.Info("Creating system configuration backup...")
	// ...
}
```

### 3. 更新生命周期钩子

**文件**: `internal/mihomo/lifecycle.go`

**修改**: 在 `OnPreStart` 钩子中添加检查日志

## 功能优势

### 1. 解决的问题

**场景 1: Mihomo 进程崩溃**
```
崩溃 → 系统配置残留 → 用户再次启动 → 自动清理残留 → 正常启动 ✅
```

**场景 2: 强制终止进程**
```
kill -9 → TUN 设备残留 → 用户再次启动 → 自动清理 TUN → 正常启动 ✅
```

**场景 3: 系统断电/重启**
```
断电 → 路由表残留 → 重启后启动 → 自动清理路由 → 正常启动 ✅
```

### 2. 清理的配置类型

- ✅ **TUN 设备残留**: 虚拟网卡未删除
- ✅ **路由表残留**: Mihomo 添加的路由未恢复
- ✅ **系统代理残留**: 代理设置未禁用
- ✅ **iptables 规则**: Linux 平台 TProxy 规则未清理

### 3. 用户体验提升

**之前**:
```
用户：启动 Mihomo
系统：启动失败（因为有残留配置）
用户：手动清理/重启系统
```

**现在**:
```
用户：启动 Mihomo
系统：检测到残留配置，自动清理
系统：清理完成，正常启动
用户：无需手动干预
```

## 执行流程

### 完整启动流程

```
1. 检查配置文件
   ↓
2. 检测 TUN/TProxy 配置
   ↓
3. 【新增】检查并清理残留配置 ← 本次修改
   ↓
4. 备份当前系统配置
   ↓
5. 启动 Mihomo 进程
   ↓
6. 健康检查
   ↓
7. 启动完成
```

### 清理流程

```
checkAndCleanupBeforeStart()
   ↓
ValidateState() - 检查所有配置
   ↓
发现残留配置？
   ├─ 是 → CleanupAll() - 自动清理
   │        ├─ 清理系统代理
   │        ├─ 清理 TUN 设备
   │        └─ 清理路由表
   │
   └─ 否 → 直接返回
```

## 日志输出示例

### 无残留配置
```
[INFO] Checking for residual configuration from abnormal exit...
[INFO] Checking for residual configuration from abnormal exit...
```

### 有残留配置并清理成功
```
[INFO] Checking for residual configuration from abnormal exit...
[WARNING] Detected 2 residual configuration issues from abnormal exit
  - TUN devices created by Mihomo still exist (severity: high)
  - Routes added by Mihomo still exist (severity: high)
[INFO] Cleaning up residual configuration before start...
[SUCCESS] Automatic cleanup completed, system is now clean
[INFO] Creating system configuration backup...
[SUCCESS] System configuration backup created
```

### 清理失败
```
[INFO] Checking for residual configuration from abnormal exit...
[WARNING] Detected 1 residual configuration issues from abnormal exit
  - TUN devices created by Mihomo still exist (severity: high)
[INFO] Cleaning up residual configuration before start...
[WARNING] Automatic cleanup failed: permission denied
[WARNING] Manual cleanup may be required
[ERROR] failed to cleanup residual configuration: permission denied
```

## 兼容性

- ✅ **向后兼容**: 不影响现有功能
- ✅ **跨平台支持**: Windows/Linux/macOS 均支持
- ✅ **性能影响**: 微小（仅启动时增加一次检查）
- ✅ **错误处理**: 清理失败会阻止启动，避免问题恶化

## 测试建议

### 测试场景 1: 正常启动
```bash
# 确保没有残留配置
mihomo-cli start
# 应该正常启动，无清理日志
```

### 测试场景 2: 模拟崩溃后启动
```bash
# 1. 启动 Mihomo（TUN 模式）
mihomo-cli start

# 2. 强制杀死进程
taskkill /F /PID <pid>  # Windows
kill -9 <pid>           # Linux/macOS

# 3. 再次启动
mihomo-cli start
# 应该看到自动清理日志
```

### 测试场景 3: 权限不足
```bash
# 以普通用户启动（TUN 模式需要管理员权限）
mihomo-cli start
# 应该看到清理失败提示
```

## 相关文件修改

1. `internal/mihomo/process_handler.go` - 核心实现
2. `internal/mihomo/lifecycle.go` - 生命周期钩子更新

## 依赖的现有功能

本次修改完全基于已有的系统配置管理功能：
- `system.SystemConfigManager` - 系统配置管理器
- `system.ValidateState()` - 状态验证
- `system.CleanupAll()` - 统一清理
- `system.CheckResidual()` - 残留检测

没有添加新的依赖，只是整合了现有功能。

## 总结

本次修改实现了**启动前自动检查和清理**功能，确保：
1. ✅ 每次启动前系统状态干净
2. ✅ 自动处理异常退出后的残留配置
3. ✅ 提升用户体验，减少手动干预
4. ✅ 防止配置残留导致启动失败

这是一个**防御性编程**的典型实践，通过主动检查和清理，避免问题积累和恶化。
