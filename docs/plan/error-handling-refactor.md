# 错误处理和工具包重构计划

## 文档信息

- **创建日期**: 2026-03-17
- **状态**: 待评审
- **相关文件**: 
  - `internal/util/error.go`
  - `internal/api/errors.go`
  - `internal/util/admin.go`

---

## 一、错误创建方式分析

### 1.1 当前问题总结

#### 存在多套独立的错误系统

**系统 A: `internal/util/error.go` - CLI 错误系统** ⚠️ 完全未使用
- 定义了 `CLIError` 结构体（包含 Code、Message、Cause）
- 预定义了 8 种退出码常量
- 提供了便捷函数：`ErrInvalidArg`、`ErrNetwork`、`ErrAPI`、`ErrConfig` 等
- 包含错误打印和退出处理函数

**系统 B: `internal/api/errors.go` - API 错误系统** ✅ 部分使用
- 定义了 `APIError` 结构体
- 预定义了 10 种 API 错误码
- 专门用于 HTTP API 调用错误处理
- 实现了 HTTP 状态码到错误码的映射

**系统 C: 标准库错误** ⚠️ 过度使用
- 大量使用 `fmt.Errorf` 创建错误
- 测试文件中使用 `errors.New`

#### 使用情况统计数据

| 位置 | `fmt.Errorf` | `errors.New` | `util.*Error` | `APIError` |
|------|-------------|--------------|---------------|-----------|
| `internal/` | 143 处 | 11 处 | 0 处 ⚠️ | 部分使用 |
| `cmd/` | 38 处 | 0 处 | 0 处 ⚠️ | 0 处 |

**关键发现：**
- ❌ **完全未使用** `util` 包的错误创建函数（搜索结果为空）
- ❌ `cmd/` 层完全使用 `fmt.Errorf`，没有使用任何统一的错误类型
- ⚠️ `api/` 包内部部分使用 `APIError`，但也混用了 `fmt.Errorf`

#### 具体问题示例

**混用示例（`api/http.go`）：**
```go
// 使用了 APIError
return nil, NewConnectionError(err)
return nil, NewTimeoutError(err)

// 但也混用了 fmt.Errorf
return "", fmt.Errorf("invalid base URL: %w", err)  // 第45行
```

**完全未使用统一错误的示例（`cmd/mihomo.go`）：**
```go
return fmt.Errorf("读取配置文件失败: %w", err)
return fmt.Errorf("不支持的配置键")
return fmt.Errorf("配置键 %s 不支持热更新，请使用 mihomo edit 命令", key)
```

---

### 1.2 建议的错误处理架构

```
┌─────────────────────────────────────┐
│   cmd/ 层（命令入口）                 │
│   使用 CLIError (errors)             │
│   - 提供正确的退出码                  │
│   - 统一错误输出格式                  │
└──────────────┬──────────────────────┘
               │
┌──────────────▼──────────────────────┐
│   internal/ 业务逻辑层                │
│   使用 CLIError (errors)             │
│   - 配置错误 → ErrConfig             │
│   - 参数错误 → ErrInvalidArg         │
│   - 服务错误 → ErrService            │
└──────────────┬──────────────────────┘
               │
┌──────────────▼──────────────────────┐
│   internal/api/ API 调用层           │
│   使用 APIError (api/errors.go)     │
│   - 专门处理 HTTP 错误               │
│   - HTTP 状态码映射                  │
└──────────────┬──────────────────────┘
               │
       转换为 CLIError
```

#### 具体建议

1. **保留 `APIError` 系统**
   - `internal/api/` 层继续使用 `APIError`
   - 适合处理 HTTP 响应和状态码映射
   - 在返回给上层时，转换为 `CLIError`

2. **大力推广 `errors` 包的使用**
   - `cmd/` 层必须使用 `CLIError`
   - `internal/` 各模块（`config/`、`proxy/`、`service/`、`mihomo/`、`sysproxy/`）应使用 `CLIError`
   - 确保所有错误都有正确的退出码

3. **建立转换规则**
   ```go
   // API 错误转换为 CLI 错误
   func APIErrorToCLIError(apiErr *api.APIError) *errors.CLIError {
       switch apiErr.Code {
       case api.ErrAPIConnection:
           return errors.ErrNetwork(apiErr.Message, apiErr.Cause)
       case api.ErrAPIAuth:
           return errors.ErrAuth(apiErr.Message, apiErr.Cause)
       case api.ErrTimeout:
           return errors.ErrTimeout(apiErr.Message, apiErr.Cause)
       default:
           return errors.ErrAPI(apiErr.Message, apiErr.Cause)
       }
   }
   ```

4. **停止使用裸 `fmt.Errorf`**
   - 将所有 `fmt.Errorf` 替换为对应的 `errors.*Error` 函数
   - 或者使用 `errors.WrapError` 包装现有错误

---

### 1.3 修复优先级建议

**高优先级：**
1. 在 `cmd/` 层统一使用 `CLIError`（38 处）
2. 在 `internal/config/` 层统一使用 `CLIError`（约 30 处）
3. 在 `internal/mihomo/` 层统一使用 `CLIError`（约 20 处）

**中优先级：**
4. 在 `internal/service/` 层统一使用 `CLIError`（约 15 处）
5. 在 `internal/proxy/` 层统一使用 `CLIError`（约 10 处）
6. 在 `internal/api/` 层消除混用，统一使用 `APIError`（约 10 处）

**低优先级：**
7. 在 `internal/sysproxy/` 层统一使用 `CLIError`（约 4 处）
8. 在 `internal/output/` 层统一使用 `CLIError`（约 1 处）

---

## 二、util 包职责划分分析

### 2.1 当前 `util` 包内容

```
internal/util/
├── error.go       - CLI 错误类型和处理（未使用）
└── admin.go       - Windows 管理员权限检查（部分使用）
```

---

### 2.2 发现的职责分散问题

#### 问题 1：路径和目录操作（分散在多处）

**重复模式：**
```go
// 在 mihomo/manager.go, mihomo/scanner.go, config/loader.go, config/backup.go 中重复
home, err := os.UserHomeDir()
configDir := filepath.Join(home, ".mihomo-cli", "config.yaml")
backupDir := filepath.Join(home, ".mihomo-cli", "backups")
pidDir := filepath.Join(home, ".mihomo-cli")
```

**建议创建：`internal/paths/paths.go`**
```go
package paths

// GetUserHomeDir 获取用户主目录（带缓存）
func GetUserHomeDir() (string, error)

// GetMihomoDir 获取 Mihomo CLI 主目录
func GetMihomoDir() (string, error)

// GetConfigDir 获取配置目录
func GetConfigDir() (string, error)

// GetDefaultConfigPath 获取默认配置文件路径
func GetDefaultConfigPath() (string, error)

// GetBackupDir 获取备份目录
func GetBackupDir() (string, error)

// GetPIDDir 获取 PID 文件目录
func GetPIDDir() (string, error)

// GenerateBackupFilename 生成备份文件名
func GenerateBackupFilename(basePath string, timestamp time.Time) string

// GeneratePIDFilename 生成 PID 文件名
func GeneratePIDFilename(configFile string) string
```

---

#### 问题 2：文件名和路径处理（分散在各模块）

**重复模式：**
```go
// 在 mihomo/manager.go, config/backup.go, config/editor.go 中重复
baseName := filepath.Base(configPath)
ext := filepath.Ext(baseName)
nameWithoutExt := strings.TrimSuffix(baseName, ext)
```

**建议创建：`internal/paths/names.go`**
```go
package paths

// ParseFilePath 解析文件路径的各个部分
type FilePathInfo struct {
    Dir      string // 目录
    BaseName string // 基础文件名
    Name     string // 文件名（不含扩展名）
    Ext      string // 扩展名
}

// ParseFilePath 解析文件路径
func ParseFilePath(path string) FilePathInfo

// BuildFilePath 构建文件路径
func BuildFilePath(dir, name, ext string) string

// ValidateConfigPath 验证配置文件路径
func ValidateConfigPath(path string) error
```

---

#### 问题 3：字符串处理工具（分散在各处）

**重复模式：**
```go
// 在 mihomo/manager.go, config/backup.go, config/keys.go 中重复
strings.HasPrefix(a.Address, "http://")
strings.HasSuffix(entry.Name(), ".pid")
strings.ToLower(filepath.Base(execPath))
strings.SplitN(name, ".", 2)
```

**建议创建：`internal/strings/strings.go`**
```go
package strings

// HasPrefixIgnoreCase 检查字符串是否有指定前缀（不区分大小写）
func HasPrefixIgnoreCase(s, prefix string) bool

// HasSuffixIgnoreCase 检查字符串是否有指定后缀（不区分大小写）
func HasSuffixIgnoreCase(s, suffix string) bool

// ContainsIgnoreCase 检查字符串是否包含子串（不区分大小写）
func ContainsIgnoreCase(s, substr string) bool

// SplitByLastN 按分隔符从右边分割
func SplitByLastN(s string, sep string, n int) []string

// Truncate 截断字符串到指定长度
func Truncate(s string, maxLen int) string

// SafeSubString 安全截取子字符串
func SafeSubString(s string, start, end int) string
```

---

#### 问题 4：数值转换和验证（分散在各处）

**重复模式：**
```go
// 在 mihomo/manager.go, config/keys.go, api/proxy.go 中重复
portNum, err := strconv.Atoi(port)
val, err := strconv.Atoi(value)
queryParams["timeout"] = strconv.Itoa(timeout)
```

**建议创建：`internal/convert/convert.go`**
```go
package convert

// StringToInt 字符串转整数（带错误处理）
func StringToInt(s string) (int, error)

// StringToIntDefault 字符串转整数（带默认值）
func StringToIntDefault(s string, defaultValue int) int

// StringToBool 字符串转布尔值
func StringToBool(s string) (bool, error)

// BoolToString 布尔值转字符串
func BoolToString(b bool) string

// IntToString 整数转字符串
func IntToString(i int) string

// ValidatePort 验证端口号
func ValidatePort(port int) error

// ValidateRange 验证数值范围
func ValidateRange(value, min, max int) error
```

---

#### 问题 5：系统检查功能（分散在 util 和 mihomo 包）

**现状：**
```go
// internal/util/admin.go - 已有
func IsAdmin() bool

// internal/mihomo/scanner.go - 重复逻辑
func isProcessRunningWindows(pid int) bool
func GetAllMihomoPIDs() ([]int, error)

// internal/mihomo/manager.go - 重复逻辑
func IsProcessRunning(pid int) bool
func ValidateProcess(pid int, force bool) error
```

**建议创建：`internal/system/admin.go`**
```go
package system

// IsAdmin 检查是否有管理员权限
func IsAdmin() bool

// IsElevated 检查是否提升权限（Windows）
func IsElevated() bool

// RequireAdmin 要求管理员权限，否则返回错误
func RequireAdmin() error
```

**建议创建：`internal/system/process.go`**
```go
package system

// IsProcessRunning 检查进程是否运行
func IsProcessRunning(pid int) bool

// GetProcessExecutable 获取进程可执行文件路径
func GetProcessExecutable(pid int) (string, error)

// FindProcessesByName 按名称查找进程
func FindProcessesByName(name string) ([]int, error)

// GetProcessOwner 获取进程所有者
func GetProcessOwner(pid int) (string, error)

// KillProcess 终止进程
func KillProcess(pid int) error
```

---

#### 问题 6：输出和日志系统（util 和 output 包重复）

**现状：**
```go
// internal/util/error.go
func PrintError(err error)
func PrintErrorAndExit(err error)

// internal/output/color.go
func Error(format string, a ...interface{})
func Success(format string, a ...interface{})
func Warning(format string, a ...interface{})
func Info(format string, a ...interface{})

// internal/output/output.go
func PrintError(msg string) error
func PrintSuccess(msg string)
func PrintWarning(msg string)
func PrintInfo(msg string)
```

**建议创建：`internal/logger/logger.go`**
```go
package logger

// LogLevel 日志级别
type LogLevel int

const (
    LevelDebug LogLevel = iota
    LevelInfo
    LevelWarning
    LevelError
    LevelFatal
)

// Logger 日志接口
type Logger interface {
    Debug(format string, a ...interface{})
    Info(format string, a ...interface{})
    Warning(format string, a ...interface{})
    Error(format string, a ...interface{})
    Fatal(format string, a ...interface{})
}

// SetGlobalLogger 设置全局日志器
func SetGlobalLogger(logger Logger)

// 全局日志函数
func Debug(format string, a ...interface{})
func Info(format string, a ...interface{})
func Warning(format string, a ...interface{})
func Error(format string, a ...interface{})
func Fatal(format string, a ...interface{})
```

**建议创建：`internal/logger/color.go`**
```go
package logger

// ColorLogger 彩色终端日志器
type ColorLogger struct {
    level LogLevel
    // ...
}

// NewColorLogger 创建彩色日志器
func NewColorLogger(level LogLevel) *ColorLogger
```

---

### 2.3 推荐的包结构重组方案

```
internal/
├── api/                    # API 客户端（保持不变）
│   ├── client.go
│   ├── errors.go           # API 错误类型
│   └── ...
├── config/                 # 配置管理（保持不变）
│   ├── config.go
│   ├── loader.go
│   └── ...
├── logger/                 # 日志系统（新增）
│   ├── logger.go           # 日志接口和全局函数
│   ├── color.go            # 彩色终端实现
│   └── json.go             # JSON 格式实现
├── output/                 # 输出格式化（重构）
│   ├── table.go            # 表格输出
│   └── json.go             # JSON 输出
├── paths/                  # 路径操作（新增）
│   ├── paths.go            # 目录和路径获取
│   └── names.go            # 文件名处理
├── strings/                # 字符串工具（新增）
│   └── strings.go          # 字符串处理工具
├── convert/                # 类型转换（新增）
│   └── convert.go          # 数值转换和验证
├── system/                 # 系统操作（新增）
│   ├── admin.go            # 权限检查
│   └── process.go          # 进程管理
├── errors/                 # 错误处理（重构）
│   ├── errors.go           # CLI 错误类型
│   ├── api.go              # API 错误转换
│   └── handler.go          # 错误处理器
├── proxy/                  # 代理管理（保持不变）
├── service/                # 服务管理（保持不变）
├── mihomo/                 # Mihomo 进程管理（保持不变）
└── sysproxy/               # 系统代理（保持不变）
```

---

## 三、迁移计划

### 阶段 1：创建新包（不影响现有代码）

1. **创建 `internal/paths/` 包**
   - `paths.go` - 目录和路径获取
   - `names.go` - 文件名处理

2. **创建 `internal/strings/` 包**
   - `strings.go` - 字符串处理工具

3. **创建 `internal/convert/` 包**
   - `convert.go` - 数值转换和验证

4. **创建 `internal/system/` 包**
   - `admin.go` - 权限检查
   - `process.go` - 进程管理

5. **创建 `internal/logger/` 包**
   - `logger.go` - 日志接口和全局函数
   - `color.go` - 彩色终端实现

---

### 阶段 2：迁移错误处理

1. **将 `util/error.go` 移动到 `errors/`**
   - 创建 `internal/errors/errors.go`
   - 保持所有现有的错误类型和函数

2. **创建错误转换层**
   - `internal/errors/api.go` - API 错误转换
   - `internal/errors/handler.go` - 错误处理器

3. **更新 API 层错误处理**
   - 在 `api/` 包中添加 APIError 到 CLIError 的转换
   - 消除 `api/` 包中的 `fmt.Errorf` 混用

---

### 阶段 3：逐模块迁移

#### 3.1 迁移 `config/` 模块
- 使用 `paths` 包处理路径操作
- 使用 `errors` 包处理错误
- 使用 `convert` 包处理类型转换

#### 3.2 迁移 `mihomo/` 模块
- 使用 `paths` 包处理路径操作
- 使用 `system` 包处理系统操作
- 使用 `errors` 包处理错误
- 使用 `strings` 包处理字符串操作

#### 3.3 迁移 `service/` 模块
- 使用 `system` 包处理系统操作
- 使用 `errors` 包处理错误

#### 3.4 迁移 `sysproxy/` 模块
- 使用 `system` 包处理系统操作
- 使用 `errors` 包处理错误

#### 3.5 迁移 `proxy/` 模块
- 使用 `errors` 包处理错误
- 使用 `strings` 包处理字符串操作

#### 3.6 迁移 `cmd/` 层
- 使用 `errors` 包处理所有错误
- 使用 `logger` 包替代 `output` 包

---

### 阶段 4：清理和优化

1. **删除废弃的包**
   - 删除 `internal/util/` 包
   - 评估是否合并 `output/` 到 `logger/`

2. **更新导入路径**
   - 全局搜索替换导入路径

3. **运行测试**
   - 确保所有测试通过
   - 添加新包的测试

4. **更新文档**
   - 更新 API 文档
   - 更新贡献指南

---

## 四、核心原则

1. **单一职责**：每个包只负责一类功能
2. **避免循环依赖**：底层包不应依赖上层包
3. **可测试性**：所有工具函数都应该易于测试
4. **可复用性**：避免在多处重复相同的逻辑
5. **清晰的命名**：包名和函数名应该清晰表达其用途

---

## 五、预期收益

### 代码质量提升
- 消除重复代码
- 统一错误处理方式
- 提高代码可维护性

### 开发效率提升
- 减少查找代码的时间
- 更容易定位问题
- 新功能开发更快速

### 系统稳定性提升
- 统一的错误退出码
- 更好的错误信息
- 更容易调试

---

## 六、风险评估

### 低风险
- 创建新包（不影响现有代码）
- 迁移单个模块

### 中风险
- 迁移多个模块
- 更新导入路径

### 高风险
- 删除 `util` 包前确保所有引用已更新
- 合并 `output` 到 `logger`

---

## 七、后续跟进

- [x] 审核此计划
- [x] 开始实施
- [ ] 确定实施时间表
- [ ] 分配任务
- [ ] 代码审查
- [ ] 测试验证
- [ ] 文档更新

---

## 八、实施状态（2026-03-17）

### 已完成的工作

#### 1. 创建错误处理框架

**pkg/errors/errors.go** ✅
- 从 `internal/util/error.go` 迁移所有错误类型和函数
- 定义了 8 种退出码常量
- 提供了 CLIError 结构体和便捷函数

**internal/errors/errors.go** ✅
- 实现错误格式化函数
- 实现错误类型检查函数
- 实现错误组合函数

**internal/errors/api.go** ✅
- 实现 APIError 到 CLIError 的转换
- 实现 API 错误包装函数
- 实现 API 错误类型检查函数

**internal/errors/handler.go** ✅
- 实现 CLIHandler 错误处理器
- 实现命令行错误处理函数
- 实现错误建议生成函数
- 实现 panic 恢复机制

#### 2. 迁移核心模块的错误处理

**internal/api/http.go** ✅
- 将 `fmt.Errorf` 替换为 `NewConnectionError`
- 保持错误处理的一致性

**internal/config/config.go** ✅
- 将所有 `fmt.Errorf` 替换为 `ErrConfig`
- 添加 `pkg/errors` 导入

**internal/config/loader.go** ✅
- 将所有 `fmt.Errorf` 替换为 `ErrConfig`
- 保持配置加载逻辑不变

**cmd/mode.go** ✅
- 将 `fmt.Errorf` 替换为 `WrapAPIError` 和 `ErrInvalidArg`
- 添加错误处理别名导入
- 测试编译成功

#### 3. 验证

**编译测试** ✅
- 项目成功编译，无错误
- 程序正常运行，帮助信息显示正常

### 待完成的工作

#### 高优先级

1. **继续迁移其他 cmd 文件**
   - cmd/proxy.go
   - cmd/config.go
   - cmd/mihomo.go
   - cmd/service.go
   - cmd/sysproxy.go
   - cmd/sub.go
   - cmd/start.go
   - cmd/stop.go
   - cmd/ps.go
   - cmd/cleanup.go
   - cmd/backup.go

2. **迁移 internal 其他模块**
   - internal/mihomo/ (约 20 处)
   - internal/service/ (约 15 处)
   - internal/proxy/ (约 10 处)
   - internal/sysproxy/ (约 4 处)
   - internal/output/ (约 1 处)

#### 中优先级

3. **更新所有导入路径**
   - 确保所有文件正确导入新的错误包
   - 移除对 `internal/util/error.go` 的引用

4. **运行完整测试**
   - 运行所有单元测试
   - 运行集成测试
   - 修复测试失败

#### 低优先级

5. **删除废弃文件**
   - 删除 `internal/util/error.go`
   - 确认没有其他文件引用

### 技术细节

#### 导入别名规范

为了避免包名冲突，使用以下导入规范：

```go
import (
    internalerrors "github.com/kkkqkx123/mihomo-cli/internal/errors"
    pkgerrors      "github.com/kkkqkx123/mihomo-cli/pkg/errors"
)
```

或者在文件中使用：
```go
import (
    "github.com/kkkqkx123/mihomo-cli/internal/errors"
    pkgerrors "github.com/kkkqkx123/mihomo-cli/pkg/errors"
)
```

#### 错误创建模式

1. **包装错误**：
```go
return errors.WrapError("操作失败", err)
```

2. **包装 API 错误**：
```go
return errors.WrapAPIError("API 调用失败", err)
```

3. **创建特定类型错误**：
```go
return pkgerrors.ErrConfig("配置错误", nil)
return pkgerrors.ErrInvalidArg("参数无效", nil)
return pkgerrors.ErrNetwork("网络错误", err)
```

4. **创建带退出码的错误**：
```go
return pkgerrors.NewError(pkgerrors.ExitConfig, "配置错误", nil)
```

### 下一步建议

1. **逐文件迁移**：按照优先级逐个文件迁移错误处理
2. **测试驱动**：每次迁移后立即测试，确保功能正常
3. **代码审查**：迁移完成后进行代码审查
4. **文档更新**：更新开发文档和贡献指南

### 注意事项

- 迁移过程中保持向后兼容
- 测试是关键，确保所有功能正常
- 使用编译器检查错误，不要手动检查
- 遵循 Go 语言的最佳实践