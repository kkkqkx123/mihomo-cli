# 错误处理分析与重构方案

## 文档信息

- **创建日期**: 2026-03-18
- **项目**: mihomo-go
- **版本**: 1.0
- **状态**: 执行中

---

## 一、当前错误处理方式分析

### 1.1 多套错误系统并存

项目目前存在 **4 套** 错误处理系统：

| 系统 | 位置 | 用途 | 特点 |
|------|------|------|------|
| **CLI 错误系统** | `pkg/errors/errors.go` | 命令行错误，带退出码 | 包含 Code、Message、Cause 字段 |
| **API 错误系统** | `internal/api/errors.go` | API 调用错误，带 HTTP 状态码 | 包含 Code、Message、StatusCode、Cause 字段 |
| **错误工具系统** | `internal/errors/` | 错误转换、格式化、处理 | 提供错误包装、转换、处理函数 |
| **标准库错误** | `fmt.Errorf` | 临时错误处理 | 47% 代码仍在使用 |

### 1.2 各模块使用情况统计

| 模块 | 错误处理位置数 | 使用统一错误类型 | 使用 fmt.Errorf | 使用率 |
|------|---------------|----------------|----------------|--------|
| cmd/ | 156 | 109 (70%) | 47 (30%) | 70% |
| internal/api/ | 107 | 32 (30%) | 75 (70%) | 30% |
| internal/config/ | 122 | 24 (20%) | 98 (80%) | 20% |
| internal/proxy/ | 76 | 46 (60%) | 30 (40%) | 60% |
| internal/service/ | 58 | 52 (90%) | 6 (10%) | 90% |
| internal/sysproxy/ | 26 | 26 (100%) | 0 (0%) | 100% |
| **总计** | **545+** | **289 (53%)** | **256 (47%)** | **53%** |

**关键发现**：
- 整体统一错误类型使用率：**53%**
- 最高：`internal/sysproxy/` (100%)
- 最低：`internal/config/` (20%)
- 需要重点改进：`internal/api/` (30%)、`internal/config/` (20%)

### 1.3 错误包装方式混乱

代码中存在 **4 种** 不同的错误包装方式：

```go
// 方式 1: fmt.Errorf（47% 代码使用）
return fmt.Errorf("操作失败: %w", err)

// 方式 2: WrapError
return errors.WrapError("操作失败", err)

// 方式 3: WrapAPIError
return errors.WrapAPIError("API 调用失败", err)

// 方式 4: 特定错误类型
return pkgerrors.ErrConfig("配置错误", err)
```

### 1.4 未使用 Go 1.13+ 最佳实践

全项目 **未使用** `errors.Is` 和 `errors.As`，全部使用类型断言：

```go
// ❌ 当前做法
if apiErr, ok := err.(*api.APIError); ok {
    return apiErr.Code == api.ErrAPIConnection
}

// ✅ 推荐做法
if errors.Is(err, api.ErrAPIConnection) {
    return true
}
```

---

## 二、统一错误处理设计方案

### 2.1 架构设计原则

1. **清晰的错误层级**：不同层级使用不同的错误类型
2. **统一的错误创建**：禁止裸 `fmt.Errorf`
3. **遵循 Go 最佳实践**：使用 `errors.Is` 和 `errors.As`
4. **向后兼容**：保留现有错误系统，逐步迁移

### 2.2 错误处理层级

```
┌─────────────────────────────────────────┐
│  cmd/ 层（命令入口层）                    │
│  使用: errors.WrapAPIError()             │
│  作用: 统一错误包装，添加命令上下文         │
└─────────────────┬───────────────────────┘
                  │
┌─────────────────▼───────────────────────┐
│  internal/ 业务逻辑层                      │
│  使用: pkg/errors.Err*()                 │
│  作用: 创建语义化的业务错误                │
└─────────────────┬───────────────────────┘
                  │
┌─────────────────▼───────────────────────┐
│  internal/api/ API 调用层                │
│  使用: api.New*Error()                   │
│  作用: 创建 API 特定错误（含 HTTP 状态）   │
└─────────────────────────────────────────┘
```

### 2.3 统一规则

#### 规则 1: API 层（`internal/api/`）

**使用 API 错误系统**，创建包含 HTTP 信息的错误：

```go
// ✅ 正确
func (c *Client) GetMode() (string, error) {
    resp, err := c.http.Get("/configs")
    if err != nil {
        return "", api.NewConnectionError(err)
    }
    defer resp.Body.Close()
    
    if resp.StatusCode != http.StatusOK {
        return "", api.ParseErrorResponse(resp)
    }
    
    // ...
}

// ❌ 错误
func (c *Client) GetMode() (string, error) {
    return "", fmt.Errorf("获取模式失败: %w", err)
}
```

#### 规则 2: 业务逻辑层（`internal/` 中的其他包）

**使用 CLI 错误系统**，创建语义化的业务错误：

```go
// ✅ 正确
func (bm *BackupManager) CreateBackup() (*Backup, error) {
    if err := os.MkdirAll(bm.backupDir, 0755); err != nil {
        return nil, pkgerrors.ErrService("创建备份目录失败", err)
    }
    // ...
}

func (cfg *CLIConfig) Validate() error {
    if cfg.API.Address == "" {
        return pkgerrors.ErrConfig("API address is required", nil)
    }
    return nil
}

// ❌ 错误
func (bm *BackupManager) CreateBackup() (*Backup, error) {
    return nil, fmt.Errorf("创建备份目录失败: %w", err)
}
```

#### 规则 3: 命令层（`cmd/`）

**使用错误包装系统**，统一包装底层错误：

```go
// ✅ 正确
func (c *modeCmd) run(cmd *cobra.Command, args []string) error {
    mode, err := apiClient.GetMode()
    if err != nil {
        return errors.WrapAPIError("获取模式失败", err)
    }
    // ...
}

// ❌ 错误
func (c *modeCmd) run(cmd *cobra.Command, args []string) error {
    mode, err := apiClient.GetMode()
    if err != nil {
        return err  // 直接返回，未包装
    }
}
```

#### 规则 4: 错误检查

**使用 `errors.Is` 和 `errors.As`**：

```go
// ✅ 正确 - 检查错误类型
if errors.Is(err, api.ErrAPIConnection) {
    // 处理连接错误
}

// ✅ 正确 - 提取错误信息
var apiErr *api.APIError
if errors.As(err, &apiErr) {
    fmt.Printf("API 错误: %s (状态码: %d)\n", apiErr.Message, apiErr.StatusCode)
}

// ✅ 正确 - 获取 CLI 错误
if cliErr := errors.GetCLIError(err); cliErr != nil {
    return cliErr.Code
}

// ❌ 错误 - 使用类型断言
if apiErr, ok := err.(*api.APIError); ok {
    // ...
}
```

#### 规则 5: 错误消息

**提供清晰的上下文信息**：

```go
// ✅ 正确 - 包含足够上下文
return pkgerrors.ErrConfig("配置文件 "+path+" 不存在或无法读取", err)

// ❌ 错误 - 信息不足
return pkgerrors.ErrConfig("配置错误", err)

// ❌ 错误 - 直接返回底层错误
return err
```

### 2.4 错误类型映射表

| 场景 | 使用类型 | 示例函数 | 退出码 |
|------|---------|---------|--------|
| API 连接失败 | `api.APIError` | `api.NewConnectionError()` | 3 (ExitNetwork) |
| API 认证失败 | `api.APIError` | `api.NewAuthError()` | 8 (ExitAuth) |
| API 超时 | `api.APIError` | `api.NewTimeoutError()` | 7 (ExitTimeout) |
| 配置错误 | `*CLIError` | `pkgerrors.ErrConfig()` | 5 (ExitConfig) |
| 参数无效 | `*CLIError` | `pkgerrors.ErrInvalidArg()` | 2 (ExitInvalid) |
| 服务错误 | `*CLIError` | `pkgerrors.ErrService()` | 6 (ExitService) |
| 网络错误 | `*CLIError` | `pkgerrors.ErrNetwork()` | 3 (ExitNetwork) |
| 超时错误 | `*CLIError` | `pkgerrors.ErrTimeout()` | 7 (ExitTimeout) |

---

## 三、迁移计划

### 3.1 高优先级（影响大、改动小）

#### 阶段 1: `internal/config/` 重构
- **目标**: 统一使用 `pkg/errors`
- **改动数**: 约 98 处
- **文件列表**:
  - `backup.go`
  - `editor.go`
  - `keys.go`
  - `toml_config.go`

#### 阶段 2: `internal/api/` 重构
- **目标**: 统一使用 `APIError`
- **改动数**: 约 75 处
- **文件列表**:
  - `mode.go`
  - `config.go`
  - `proxy.go`
  - `rule.go`
  - `provider.go`

#### 阶段 3: `cmd/` 重构
- **目标**: 补充缺失的错误包装
- **改动数**: 约 47 处
- **文件列表**:
  - `root.go`
  - `service.go`
  - `sub.go`
  - `sysproxy.go`

### 3.2 中优先级

#### 阶段 4: `internal/mihomo/` 重构
- **目标**: 统一使用 `pkg/errors`
- **改动数**: 约 70% 处

#### 阶段 5: `internal/proxy/` 重构
- **目标**: 补充缺失的错误包装
- **改动数**: 约 30 处

### 3.3 低优先级

#### 阶段 6: 最佳实践迁移
- **目标**: 使用 `errors.Is` 和 `errors.As`
- **改动数**: 约 50 处类型断言

#### 阶段 7: 其他模块调整
- **目标**: 小范围调整和优化
- **改动数**: 约 20 处

---

## 四、代码审查清单

每次代码提交前，确保：

- [ ] ✅ 没有使用裸 `fmt.Errorf`（特殊情况除外）
- [ ] ✅ 使用了正确的错误类型（API 层用 `APIError`，业务层用 `CLIError`）
- [ ] ✅ 所有错误都被包装，提供了足够的上下文
- [ ] ✅ 使用 `errors.Is` 或 `errors.As` 进行错误检查
- [ ] ✅ 错误消息清晰，包含必要的上下文信息
- [ ] ✅ 退出码正确，符合错误类型
- [ ] ✅ 所有 `defer` 资源清理都有错误处理
- [ ] ✅ 所有错误路径都经过了测试

---

## 五、预期收益

### 5.1 代码质量提升
- 提高代码可维护性和可读性
- 统一错误处理逻辑，减少 bug
- 便于错误追踪和调试

### 5.2 开发效率提升
- 新开发者更容易理解错误处理逻辑
- 减少错误处理相关的代码审查时间
- 加快问题定位和修复速度

### 5.3 用户体验提升
- 统一的错误输出格式
- 更清晰的错误消息
- 更准确的错误分类和退出码

### 5.4 符合最佳实践
- 遵循 Go 语言错误处理最佳实践
- 与 Go 1.13+ 的错误处理机制保持一致
- 便于未来功能扩展和维护

---

## 六、进度跟踪

### 6.1 总体进度

- [ ] 阶段 1: `internal/config/` 重构 (0/4 文件)
- [ ] 阶段 2: `internal/api/` 重构 (0/5 文件)
- [ ] 阶段 3: `cmd/` 重构 (0/4 文件)
- [ ] 阶段 4: `internal/mihomo/` 重构 (0/3 文件)
- [ ] 阶段 5: `internal/proxy/` 重构 (0/3 文件)
- [ ] 阶段 6: 最佳实践迁移 (0/50 处)
- [ ] 阶段 7: 其他模块调整 (0/20 处)

### 6.2 详细进度

#### 阶段 1: `internal/config/`
- [ ] `backup.go`
- [ ] `editor.go`
- [ ] `keys.go`
- [ ] `toml_config.go`

#### 阶段 2: `internal/api/`
- [ ] `mode.go`
- [ ] `config.go`
- [ ] `proxy.go`
- [ ] `rule.go`
- [ ] `provider.go`

#### 阶段 3: `cmd/`
- [ ] `root.go`
- [ ] `service.go`
- [ ] `sub.go`
- [ ] `sysproxy.go`

#### 阶段 4: `internal/mihomo/`
- [ ] `manager.go`
- [ ] `process_handler.go`
- [ ] `scanner.go`

#### 阶段 5: `internal/proxy/`
- [ ] `filter.go`
- [ ] `formatter.go`
- [ ] `tester.go`

---

## 七、风险评估

### 7.1 技术风险
- **低风险**: 错误处理重构不会影响核心功能逻辑
- **中风险**: 需要确保所有错误路径都被测试覆盖
- **低风险**: 重构过程可以逐文件进行，便于回滚

### 7.2 兼容性风险
- **低风险**: 保持错误类型接口不变，向后兼容
- **低风险**: 退出码保持不变，不影响外部调用

### 7.3 测试风险
- **中风险**: 需要更新相关测试用例
- **低风险**: 现有测试框架可以继续使用

---

## 八、总结

### 8.1 当前问题
- 多套错误系统并存，使用混乱
- 整体统一错误类型使用率仅 53%
- 未遵循 Go 1.13+ 错误处理最佳实践

### 8.2 设计方案
- **分层架构**: API 层 → 业务逻辑层 → 命令层
- **统一规则**: 各层使用特定的错误类型和创建函数
- **最佳实践**: 使用 `errors.Is` 和 `errors.As`
- **渐进迁移**: 按优先级逐步重构各模块

### 8.3 预期收益
- 提高代码可维护性和可读性
- 统一错误处理逻辑，减少 bug
- 便于错误追踪和调试
- 符合 Go 语言最佳实践

---

## 九、参考资料

- [Go 官方错误处理文档](https://go.dev/doc/tutorial/errors)
- [Go 1.13 错误处理](https://go.dev/blog/go1.13-errors)
- [Effective Go - Errors](https://go.dev/doc/effective_go#errors)
- [Go 错误处理最佳实践](https://github.com/golang/go/wiki/Errors)

---

**文档维护**: 请在每次完成一个阶段后更新进度跟踪部分。