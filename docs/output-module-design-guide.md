# 输出模块设计指南

## 问题背景

在测试 `internal/proxy/formatter_test.go` 时发现输出捕获问题，根本原因是**包初始化时缓存 Writer**，而不是在每次使用时动态获取。

## 问题诊断

### 错误示例

```go
// ❌ 错误代码：在包初始化时缓存 Writer
var defaultColor = NewColor()  // 只执行一次，缓存当时的 os.Stdout

func NewColor() *Color {
    return NewColorWithWriter(GetGlobalStdout())  // 这里获取的 writer 被永久缓存
}

// 全局函数使用缓存的 writer
func Warning(format string, a ...interface{}) {
    defaultColor.Warning(format, a...)  // 使用缓存的旧 writer
}
```

**问题表现**：
- 测试时重定向 `os.Stdout` 无法捕获输出
- `defaultColor` 内部缓存的 writer 仍然是旧的 `os.Stdout`
- 导致 `output.SetGlobalStdout(w)` 调用无效

### 测试结果

```
=== RUN   TestFormatAutoSelectResult_EmptyNode
⚠ 代理组 'Proxy' 中没有可用的节点
    formatter_test.go:317: Actual output: ""  // 输出为空！
--- FAIL: TestFormatAutoSelectResult_EmptyNode (0.00s)
```

## 正确实现原则

### 原则 1：延迟获取 Writer（Lazy Evaluation）

每次使用时动态获取，而不是在初始化时缓存。

```go
// ✅ 正确的做法
func Warning(format string, a ...interface{}) {
    // 每次都调用 GetGlobalStdout() 获取最新的 writer
    fmt.Fprintf(GetGlobalStdout(), "%s\n", defaultColor.warning.Sprintf("⚠ "+format, a...))
}
```

### 原则 2：依赖注入优先

提供 `WithWriter` 变体函数，让调用者显式传递 writer。

```go
// ✅ 推荐：使用依赖注入
func FWarning(w io.Writer, format string, a ...interface{}) {
    c := NewColorWithWriter(w)  // 每次创建新实例，使用传入的 writer
    c.Warning(format, a...)
}
```

### 原则 3：全局函数 = 延迟获取 + 格式化

全局函数应该只负责：
1. 动态获取当前 writer
2. 应用格式化（颜色、前缀等）
3. 调用底层输出函数

```go
// ✅ 当前修复后的实现（正确）
func Warning(format string, a ...interface{}) {
    // 1. 动态获取 writer
    // 2. 应用颜色格式化
    // 3. 输出
    fmt.Fprintf(GetGlobalStdout(), "%s\n", defaultColor.warning.Sprintf("⚠ "+format, a...))
}
```

### 原则 4：避免全局可变状态

不要在包级别存储可变的状态（如缓存的 writer），应该：
- 存储**获取函数**而不是**值**
- 或者每次都重新创建临时对象

```go
// ✅ 推荐：存储获取函数
type Color struct {
    getWriter func() io.Writer  // 函数，每次调用都获取最新值
    success   *color.Color
    // ...
}

// 或者

// ✅ 推荐：每次都创建新实例（开销很小）
func FWarning(w io.Writer, format string, a ...interface{}) {
    c := NewColorWithWriter(w)  // 临时创建，用完即弃
    c.Warning(format, a...)
}
```

## 代码对比

### 错误实现

```go
package output

// Color 颜色输出管理器
type Color struct {
    writer  io.Writer  // ❌ 缓存 writer
    success *color.Color
    error   *color.Color
    warning *color.Color
    info    *color.Color
}

// NewColor 创建新的颜色管理器（使用默认 stdout）
func NewColor() *Color {
    return NewColorWithWriter(GetGlobalStdout())
}

// NewColorWithWriter 使用指定 Writer 创建颜色管理器
func NewColorWithWriter(w io.Writer) *Color {
    return &Color{
        writer:  w,  // ❌ 缓存传入的 writer
        success: color.New(color.FgGreen),
        error:   color.New(color.FgRed),
        warning: color.New(color.FgYellow),
        info:    color.New(color.FgCyan),
    }
}

// Warning 打印警告信息
func (c *Color) Warning(format string, a ...interface{}) {
    fmt.Fprintf(c.writer, "%s\n", c.warning.Sprintf("⚠ "+format, a...))
}

// 全局颜色管理器实例
var defaultColor = NewColor()  // ❌ 包初始化时创建，缓存 os.Stdout

// Warning 打印警告信息（使用默认 stdout）
func Warning(format string, a ...interface{}) {
    defaultColor.Warning(format, a...)  // ❌ 使用缓存的 writer
}
```

### 正确实现

```go
package output

// Color 颜色输出管理器
type Color struct {
    // ✅ 只存储颜色配置（不可变）
    success *color.Color
    error   *color.Color
    warning *color.Color
    info    *color.Color
}

// NewColor 创建新的颜色管理器（不绑定 writer）
func NewColor() *Color {
    return &Color{
        success: color.New(color.FgGreen),
        error:   color.New(color.FgRed),
        warning: color.New(color.FgYellow),
        info:    color.New(color.FgCyan),
    }
}

// 全局颜色配置（只读，存储颜色但不存储 writer）
var globalColor = NewColor()

// Warning 打印警告信息（使用当前全局 stdout）
func Warning(format string, a ...interface{}) {
    // ✅ 每次调用都获取最新的 writer
    w := GetGlobalStdout()
    fmt.Fprintf(w, "%s\n", globalColor.warning.Sprintf("⚠ "+format, a...))
}

// FWarning 使用指定 writer 打印警告信息（依赖注入）
func FWarning(w io.Writer, format string, a ...interface{}) {
    // ✅ 临时创建，不缓存
    c := NewColor()
    c.warning.SetOutput(w)  // color.Color 支持 SetOutput
    c.Warning(format, a...)
}
```

## 测试最佳实践

### 测试辅助函数

```go
package proxy

import (
    "bytes"
    "io"
    "os"
    "testing"
    "github.com/kkkqkx123/mihomo-cli/internal/output"
)

// captureOutput 捕获函数的 stdout 输出
func captureOutput(f func() error) (string, error) {
    oldStdout := os.Stdout
    oldGlobalStdout := output.GetGlobalStdout()
    
    r, w, _ := os.Pipe()
    os.Stdout = w
    output.SetGlobalStdout(w)  // ✅ 更新全局 writer

    err := f()

    w.Close()
    os.Stdout = oldStdout
    output.SetGlobalStdout(oldGlobalStdout)  // ✅ 恢复

    var buf bytes.Buffer
    io.Copy(&buf, r)
    return buf.String(), err
}
```

### 测试用例示例

```go
func TestFormatWarning(t *testing.T) {
    output, err := captureOutput(func() error {
        output.Warning("测试警告: %s", "test")
        return nil
    })

    if err != nil {
        t.Fatalf("输出失败: %v", err)
    }

    expected := "⚠ 测试警告: test"
    if !strings.Contains(output, expected) {
        t.Errorf("期望包含 %q，实际输出: %q", expected, output)
    }
}
```

## 总结

**正确的输出模块设计**应遵循：

1. ✅ **不要**在包初始化时缓存 writer
2. ✅ **每次**使用时动态获取 `GetGlobalStdout()`
3. ✅ 提供 `WithWriter` 变体支持依赖注入
4. ✅ 全局函数 = 延迟获取 + 格式化 + 输出
5. ✅ 测试时只需重定向 `os.Stdout` 并调用 `SetGlobalStdout`

遵循这些原则，输出模块就能**可测试、可重定向、线程安全**。

## 实际修复记录

### 修复前

```go
// Warning 打印警告信息（使用默认 stdout）
func Warning(format string, a ...interface{}) {
    defaultColor.Warning(format, a...)  // ❌ 使用缓存的 writer
}

// Info 打印信息（使用默认 stdout）
func Info(format string, a ...interface{}) {
    defaultColor.Info(format, a...)  // ❌ 使用缓存的 writer
}
```

### 修复后

```go
// Warning 打印警告信息（使用默认 stdout）
func Warning(format string, a ...interface{}) {
    fmt.Fprintf(GetGlobalStdout(), "%s\n", defaultColor.warning.Sprintf("⚠ "+format, a...))
}

// Info 打印信息（使用默认 stdout）
func Info(format string, a ...interface{}) {
    fmt.Fprintf(GetGlobalStdout(), "%s\n", defaultColor.info.Sprintf("ℹ "+format, a...))
}
```

### 测试结果

修复后测试通过：

```bash
$ go test ./internal/proxy/... -run "TestFormatAutoSelectResult_EmptyNode|TestFormatGroupList_NoGroups" -v
=== RUN   TestFormatAutoSelectResult_EmptyNode
--- PASS: TestFormatAutoSelectResult_EmptyNode (0.00s)
=== RUN   TestFormatGroupList_NoGroups
--- PASS: TestFormatGroupList_NoGroups (0.00s)
PASS
```

## 相关文件

- `internal/output/color.go` - 颜色输出管理
- `internal/output/writer.go` - 全局 writer 管理
- `internal/output/output.go` - 通用输出接口
- `internal/proxy/formatter_test.go` - 测试用例（包含 captureOutput 实现）
