# API 客户端模块

## 概述

本模块提供了 Mihomo RESTful API 的 Go 语言客户端封装，实现了统一的请求处理、认证机制和错误处理。

## 核心功能

### ✅ 已实现功能

1. **客户端基础结构** (`client.go`)
   - `Client` 结构体封装 API 配置
   - `NewClient()` 构造函数，支持选项模式配置
   - 可配置的请求超时
   - 兼容性构造函数 `NewClientWithTimeout()`

2. **认证机制** (`http.go`)
   - Bearer Token 认证
   - 自动添加 `Authorization` 头到所有请求
   - 支持 Secret 配置

3. **通用请求方法** (`http.go` & `client.go`)
   - `GET` - 获取资源
   - `POST` - 创建资源
   - `PUT` - 更新资源
   - `PATCH` - 部分更新
   - `DELETE` - 删除资源

4. **错误处理** (`errors.go`)
   - `APIError` 类型，包含错误码、消息和原因
   - 错误码映射到退出码
   - HTTP 状态码到错误码的映射
   - 错误类型检查辅助函数

## 文件结构

```
internal/api/
├── client.go   # 客户端基础结构和通用方法
├── http.go     # HTTP 客户端实现和认证机制
├── errors.go   # 错误类型和处理
├── example.go  # 使用示例
└── README.md   # 本文档
```

## 使用方法

### 创建客户端

```go
import (
    "context"
    "time"
    "your-project/internal/api"
)

// 基本用法
client := api.NewClient(
    "http://127.0.0.1:9090",
    "your-secret-here",
)

// 自定义超时
client := api.NewClient(
    "http://127.0.0.1:9090",
    "your-secret-here",
    api.WithTimeout(15*time.Second),
)

// 兼容旧接口
client := api.NewClientWithTimeout("http://127.0.0.1:9090", "secret", 15)
```

### 发送请求

#### GET 请求

```go
// 获取版本信息
var version map[string]interface{}
err := client.Get(ctx, "/version", nil, &version)
if err != nil {
    // 处理错误
}

// 带查询参数
queryParams := map[string]string{
    "name": "google.com",
    "type": "A",
}
var dnsResult map[string]interface{}
err := client.Get(ctx, "/dns/query", queryParams, &dnsResult)
```

#### POST 请求

```go
data := map[string]interface{}{
    "path": "/path/to/config.yaml",
}
err := client.Post(ctx, "/configs", nil, data, nil)
```

#### PUT 请求

```go
data := map[string]interface{}{
    "name": "proxy-name",
}
err := client.Put(ctx, "/proxies/Proxy", nil, data, nil)
```

#### PATCH 请求

```go
data := map[string]interface{}{
    "mode":      "rule",
    "log-level": "info",
}
err := client.Patch(ctx, "/configs", nil, data, nil)
```

#### DELETE 请求

```go
err := client.Delete(ctx, "/connections/conn-id", nil, nil)
```

### 错误处理

```go
err := client.Get(ctx, "/proxies/nonexistent", nil, &result)
if err != nil {
    switch {
    case api.IsAPIConnectionError(err):
        fmt.Println("Failed to connect to API server")
    case api.IsAPIAuthError(err):
        fmt.Println("Authentication failed. Check your secret.")
    case api.IsTimeoutError(err):
        fmt.Println("Request timeout. Try increasing timeout.")
    case api.IsNotFoundError(err):
        fmt.Println("Resource not found")
    default:
        if apiErr, ok := err.(*api.APIError); ok {
            fmt.Printf("API Error [%d]: %s\n", apiErr.Code, apiErr.Message)
            if apiErr.Cause != nil {
                fmt.Printf("Caused by: %v\n", apiErr.Cause)
            }
        } else {
            fmt.Printf("Error: %v\n", err)
        }
    }
}
```

## 错误码

| 错误码 | 常量 | 说明 |
|--------|------|------|
| 0 | `ErrSuccess` | 成功 |
| 1 | `ErrGeneral` | 通用错误 |
| 2 | `ErrAPIConnection` | API 连接错误 |
| 3 | `ErrAPIAuth` | API 认证错误 |
| 4 | `ErrInvalidArgs` | 参数无效 |
| 5 | `ErrNotFound` | 资源不存在 |
| 6 | `ErrPermission` | 权限不足 |
| 7 | `ErrFileOperation` | 文件操作错误 |
| 8 | `ErrYAMLParse` | YAML 解析错误 |
| 9 | `ErrTimeout` | 请求超时 |
| 10 | `ErrAPIError` | API 返回错误 |

## HTTP 状态码映射

| HTTP 状态码 | 错误码 |
|-------------|--------|
| 400 | `ErrInvalidArgs` |
| 401 | `ErrAPIAuth` |
| 403 | `ErrPermission` |
| 404 | `ErrNotFound` |
| 408, 504 | `ErrTimeout` |
| 500, 503 | `ErrAPIError` |
| 其他 | `ErrGeneral` |

## 验收标准

### ✅ 已满足的验收标准

1. **所有请求自动添加 Authorization 头**
   - 在 `http.go` 的 `addAuthHeader()` 方法中实现
   - 所有请求方法（GET、POST、PUT、PATCH、DELETE）都会自动添加

2. **请求超时可配置**
   - 通过 `WithTimeout()` 选项配置
   - 默认超时 10 秒
   - 支持运行时修改：`SetTimeout()`

3. **错误响应统一处理**
   - 定义了 `APIError` 类型
   - 实现了错误码到退出码的映射
   - 提供了错误类型检查辅助函数
   - 自动解析 API 错误响应

## 设计特点

1. **选项模式配置**
   - 使用函数式选项模式，便于扩展配置

2. **统一的错误处理**
   - 所有错误都包装为 `APIError`
   - 提供清晰的错误信息和退出码

3. **Context 支持**
   - 所有请求方法都接受 `context.Context`
   - 支持请求超时和取消

4. **类型安全**
   - 使用泛型思想（通过 `interface{}` 实现）
   - 支持自定义响应类型

5. **易用性**
   - 简洁的 API 设计
   - 提供丰富的使用示例

## 下一步

基于这个 API 客户端，可以实现：
- 模式管理 API (`mode.go`)
- 代理管理 API (`proxy.go`)
- 配置管理 API (`config.go`)
- 订阅管理 API (`provider.go`)
- 规则管理 API (`rule.go`)
- 连接管理 API (`connection.go`)
- 缓存管理 API (`cache.go`)
- DNS 查询 API (`dns.go`)
- 系统管理 API (`system.go`)
- 版本查询 API (`version.go`)
- 监控 API (`monitor.go`)

## 相关文档

- [Mihomo API 文档](../../docs/spec/mihono-api.md)
- [设计文档](../../docs/spec/design.md)
- [需求规格](../../docs/spec/spec.md)

## 示例代码

完整的使用示例请参考 `example.go` 文件。