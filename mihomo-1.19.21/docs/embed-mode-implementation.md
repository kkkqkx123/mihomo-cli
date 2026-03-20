# Mihomo Embed 模式实现分析

## 概述

Embed 模式是 Mihomo 为嵌入式环境（如 Android 平台）设计的特殊运行模式，通过限制特定的 API 操作来确保系统稳定性。本文档详细分析了 Embed 模式的实现机制、使用场景和相关配置。

## 目录

- [核心机制](#核心机制)
- [Go Embed 使用](#go-embed-使用)
- [API 限制](#api-限制)
- [设计理念](#设计理念)
- [关键文件](#关键文件)
- [配置选项](#配置选项)
- [使用场景](#使用场景)

---

## 核心机制

### 1. embedMode 变量

**位置**: `hub/route/server.go:40`

```go
var (
    uiPath = ""

    httpServer *http.Server
    tlsServer  *http.Server
    unixServer *http.Server
    pipeServer *http.Server

    embedMode = false  // 默认为 false
)
```

### 2. SetEmbedMode 函数

**位置**: `hub/route/server.go:43-44`

```go
func SetEmbedMode(embed bool) {
    embedMode = embed
}
```

### 3. Android 平台自动启用

**位置**: `hub/route/patch_android.go`

```go
//go:build android && cmfa

package route

func init() {
    SetEmbedMode(true) // set embed mode default
}
```

**关键点**：
- 使用 Go 编译标签 `android && cmfa`
- 在包初始化时自动设置 embed 模式
- 仅在特定 Android 构建条件下启用

---

## Go Embed 使用

### 1. CA 证书嵌入

**位置**: `component/ca/config.go:5,22`

```go
package ca

import (
    _ "embed"  // 导入 embed 包，但不使用其导出函数
    // ...
)

//go:embed ca-certificates.crt
var _CaCertificates []byte
```

**关键点**：
- 使用 Go 1.16+ 的 `//go:embed` 指令
- 将 `ca-certificates.crt` 文件内容嵌入到 `_CaCertificates` 变量
- 编译时文件内容被直接编译到二进制中

### 2. 证书加载逻辑

**位置**: `component/ca/config.go:24-60`

```go
var DisableEmbedCa, _ = strconv.ParseBool(os.Getenv("DISABLE_EMBED_CA"))
var DisableSystemCa, _ = strconv.ParseBool(os.Getenv("DISABLE_SYSTEM_CA"))

func initializeCertPool() {
    var err error
    if DisableSystemCa {
        globalCertPool = x509.NewCertPool()
    } else {
        globalCertPool, err = x509.SystemCertPool()
        if err != nil {
            globalCertPool = x509.NewCertPool()
        }
    }
    if !DisableEmbedCa {
        globalCertPool.AppendCertsFromPEM(_CaCertificates)
    }
}
```

**关键点**：
- 支持通过环境变量控制 embed 证书的使用
- `DISABLE_EMBED_CA`: 禁用 embed 证书
- `DISABLE_SYSTEM_CA`: 禁用系统证书
- 默认优先使用 embed 证书

### 3. Zero Trust 证书池

**位置**: `component/ca/config.go:127-135`

```go
var zeroTrustCertPool = once.OnceValue(func() *x509.CertPool {
    if len(_CaCertificates) != 0 { // always using embed cert first
        zeroTrustCertPool := x509.NewCertPool()
        if zeroTrustCertPool.AppendCertsFromPEM(_CaCertificates) {
            return zeroTrustCertPool
        }
    }
    return nil // fallback to system pool
})
```

**关键点**：
- Zero Trust 模式始终优先使用 embed 证书
- 使用 `once.OnceValue` 确保只初始化一次
- 如果 embed 证书不可用，回退到系统证书池

---

## API 限制

### 1. 配置管理 API

**位置**: `hub/route/configs.go:12-18`

```go
func configRouter() http.Handler {
    r := chi.NewRouter()
    r.Get("/", getConfigs)
    if !embedMode { // disallow update/patch configs in embed mode
        r.Put("/", updateConfigs)
        r.Post("/geo", updateGeoDatabases)
        r.Patch("/", patchConfigs)
    }
    return r
}
```

**禁用的端点**：
- `PUT /configs` - 重载配置文件
- `PATCH /configs` - 部分更新配置
- `POST /configs/geo` - 更新 Geo 数据库

### 2. 规则管理 API

**位置**: `hub/route/rules.go:11-15`

```go
func ruleRouter() http.Handler {
    r := chi.NewRouter()
    r.Get("/", getRules)
    if !embedMode { // disallow update/patch rules in embed mode
        r.Patch("/disable", disableRules)
    }
    return r
}
```

**禁用的端点**：
- `PATCH /rules/disable` - 禁用/启用规则

### 3. 系统管理 API

**位置**: `hub/route/server.go:135-138`

```go
r.Mount("/upgrade", upgradeRouter())
if !embedMode { // disallow restart/shutdown in embed mode
    r.Mount("/restart", restartRouter())
    r.Mount("/shutdown", shutdownRouter())
}
```

**禁用的端点**：
- `POST /restart` - 重启服务
- `POST /shutdown` - 关闭服务

### 4. 升级管理 API

**位置**: `hub/route/upgrade.go:11-18`

```go
func upgradeRouter() http.Handler {
    r := chi.NewRouter()
    r.Post("/ui", updateUI)
    if !embedMode { // disallow upgrade core/geo in embed mode
        r.Post("/", upgradeCore)
        r.Post("/geo", updateGeoDatabases)
    }
    return r
}
```

**禁用的端点**：
- `POST /upgrade` - 升级核心程序
- `POST /upgrade/geo` - 更新 Geo 数据库

**注意**：`POST /upgrade/ui` 在 embed 模式下仍然可用。

---

## 设计理念

### 1. 稳定性优先

Embed 模式的设计目标是确保嵌入式环境的稳定性，通过禁用可能影响系统稳定性的操作：
- 配置热更新可能导致配置错误
- 规则动态修改可能影响路由逻辑
- 服务重启/关闭可能导致服务中断
- 核心升级可能引入不兼容的变更

### 2. 最小权限原则

仅保留必要的更新能力（如 UI 更新），限制对核心配置的修改权限。

### 3. 编译时确定

Embed 模式在编译时确定，运行时无法动态切换，确保行为的一致性和可预测性。

---

## 关键文件

### 核心实现文件

| 文件路径 | 说明 |
|---------|------|
| `hub/route/server.go` | embedMode 变量和 SetEmbedMode 函数 |
| `hub/route/configs.go` | 配置 API 限制 |
| `hub/route/rules.go` | 规则 API 限制 |
| `hub/route/upgrade.go` | 升级 API 限制 |
| `hub/route/patch_android.go` | Android 默认 embed 模式 |

### CA 证书嵌入

| 文件路径 | 说明 |
|---------|------|
| `component/ca/config.go` | embed 包导入和证书初始化 |
| `component/ca/ca-certificates.crt` | 内嵌证书文件 |

---

## 配置选项

### 环境变量控制

| 环境变量 | 说明 | 默认值 |
|---------|------|--------|
| `DISABLE_EMBED_CA` | 禁用嵌入的 CA 证书 | `false` |
| `DISABLE_SYSTEM_CA` | 禁用系统 CA 证书 | `false` |

### 编译时配置

**Android 平台自动启用**：

```bash
# 构建命令（在 Makefile 中）
android-arm64:
	GOARCH=arm64 GOOS=android $(GOBUILD) -o $(BINDIR)/$(NAME)-$@
```

**编译标签**：
- `android && cmfa` - Android + Clash Meta for Android
- 启用条件编译的 `patch_android.go`

---

## 使用场景

### 1. Android 平台

**适用场景**：
- Clash Meta for Android (cmfa)
- 需要稳定运行的移动应用环境

**特点**：
- 配置由应用管理，用户无法直接修改
- 升级通过应用商店进行，无需核心升级 API
- UI 更新仍然支持，保持用户体验

### 2. 嵌入式设备

**适用场景**：
- 路由器固件
- IoT 设备
- 容器化部署

**特点**：
- 配置固定，无需热更新
- 升级通过固件更新进行
- 确保核心功能稳定运行

### 3. 对比普通模式

| 特性 | 普通模式 | Embed 模式 |
|------|----------|-----------|
| 配置热更新 | ✓ | ✗ |
| 规则动态修改 | ✓ | ✗ |
| 服务重启/关闭 | ✓ | ✗ |
| 核心升级 | ✓ | ✗ |
| Geo 数据库更新 | ✓ | ✗ |
| UI 更新 | ✓ | ✓ |
| 嵌入证书 | 可选 | 优先使用 |
| 使用场景 | 桌面/服务器 | Android/嵌入式设备 |

---

## 代码示例

### Router 配置中的 embed 模式检查

**位置**: `hub/route/server.go:125-142`

```go
func router(isDebug bool, secret string, dohServer string, cors Cors) *chi.Mux {
    r := chi.NewRouter()
    cors.Apply(r)

    // ... 其他路由配置 ...

    r.Group(func(r chi.Router) {
        if secret != "" {
            r.Use(authentication(secret))
        }
        r.Get("/", hello)
        r.Get("/logs", getLogs)
        r.Get("/traffic", traffic)
        r.Get("/memory", memory)
        r.Get("/version", version)
        r.Mount("/configs", configRouter())
        r.Mount("/proxies", proxyRouter())
        r.Mount("/group", groupRouter())
        r.Mount("/rules", ruleRouter())
        r.Mount("/connections", connectionRouter())
        r.Mount("/providers/proxies", proxyProviderRouter())
        r.Mount("/providers/rules", ruleProviderRouter())
        r.Mount("/cache", cacheRouter())
        r.Mount("/dns", dnsRouter())

        // embed 模式下禁用重启和关闭
        if !embedMode {
            r.Mount("/restart", restartRouter())
            r.Mount("/shutdown", shutdownRouter())
        }

        r.Mount("/upgrade", upgradeRouter())
        addExternalRouters(r)
    })

    // ... UI 路由配置 ...

    return r
}
```

---

## 总结

Mihomo 的 embed 模式实现体现了良好的工程实践：

1. **简洁性**：使用单一变量控制，逻辑清晰
2. **安全性**：通过 API 限制防止意外修改
3. **灵活性**：支持环境变量控制和编译标签
4. **可维护性**：代码结构清晰，易于理解和扩展

这种实现方式特别适合需要稳定运行的嵌入式场景，如 Android 平台，确保了核心功能的稳定性，同时保留了必要的更新能力（如 UI 更新）。

---

## 相关资源

- [Mihomo API 文档](https://wiki.metacubex.one/api/)
- [Go Embed 文档](https://pkg.go.dev/embed)
- [项目配置示例](./config.yaml)