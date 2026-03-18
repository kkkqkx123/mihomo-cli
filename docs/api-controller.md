# Mihomo API 控制端口开启说明

## 概述

Mihomo 提供 RESTful API 用于控制和监控代理核心，支持多种通信协议，包括 HTTP、HTTPS、Unix Domain Socket 和 Windows 命名管道。

## 启动流程

### 1. 命令行参数和配置文件解析

在程序启动时，通过以下方式获取控制器配置：

**命令行参数：**

- `-ext-ctl`: 覆盖外部控制器地址（对应环境变量 `CLASH_OVERRIDE_EXTERNAL_CONTROLLER`）
- `-ext-ctl-unix`: 覆盖 Unix Socket 地址（对应环境变量 `CLASH_OVERRIDE_EXTERNAL_CONTROLLER_UNIX`）
- `-ext-ctl-pipe`: 覆盖命名管道地址（对应环境变量 `CLASH_OVERRIDE_EXTERNAL_CONTROLLER_PIPE`）
- `-secret`: 覆盖 API 密钥（对应环境变量 `CLASH_OVERRIDE_SECRET`）

### 2. Hub 层处理

配置通过 `hub.Parse()` 传递，在 `hub/applyRoute()` 中创建服务器：

```go
func applyRoute(cfg *config.Config) {
    route.ReCreateServer(&route.Config{
        Addr:           cfg.Controller.ExternalController,
        TLSAddr:        cfg.Controller.ExternalControllerTLS,
        UnixAddr:       cfg.Controller.ExternalControllerUnix,
        PipeAddr:       cfg.Controller.ExternalControllerPipe,
        Secret:         cfg.Controller.Secret,
        Certificate:    cfg.TLS.Certificate,
        PrivateKey:     cfg.TLS.PrivateKey,
        ClientAuthType: cfg.TLS.ClientAuthType,
        ClientAuthCert: cfg.TLS.ClientAuthCert,
        EchKey:         cfg.TLS.EchKey,
        DohServer:      cfg.Controller.ExternalDohServer,
        IsDebug:        cfg.General.LogLevel == log.DEBUG,
        Cors: route.Cors{
            AllowOrigins:        cfg.Controller.Cors.AllowOrigins,
            AllowPrivateNetwork: cfg.Controller.Cors.AllowPrivateNetwork,
        },
    })
}
```

### 3. 创建 API 服务器

`ReCreateServer` 根据配置启动多个并发服务器：

```go
func ReCreateServer(cfg *Config) {
    go start(cfg)      // HTTP 服务器
    go startTLS(cfg)   // HTTPS 服务器
    go startUnix(cfg)  // Unix Socket 服务器
    if inbound.SupportNamedPipe {
        go startPipe(cfg)  // Windows 命名管道服务器
    }
}
```

### 4. 服务器监听实现

#### HTTP 服务器

```go
func start(cfg *Config) {
    if len(cfg.Addr) > 0 {
        l, err := inbound.Listen("tcp", cfg.Addr)
        server := &http.Server{
            Handler: router(cfg.IsDebug, cfg.Secret, cfg.DohServer, cfg.Cors),
        }
        server.Serve(l)
    }
}
```

#### HTTPS 服务器

```go
func startTLS(cfg *Config) {
    l, err := inbound.Listen("tcp", cfg.TLSAddr)
    tlsConfig := &tls.Config{
        GetCertificate: func(*tls.ClientHelloInfo) (*tls.Certificate, error) {
            return certLoader()
        },
    }
    server := &http.Server{
        Handler: router(cfg.IsDebug, cfg.Secret, cfg.DohServer, cfg.Cors),
    }
    server.Serve(tls.NewListener(l, tlsConfig))
}
```

#### Unix Socket 服务器

```go
func startUnix(cfg *Config) {
    addr := C.Path.Resolve(cfg.UnixAddr)
    _ = syscall.Unlink(addr)  // Windows 需要先删除 socket 文件
    l, err := inbound.Listen("unix", addr)
    server.Serve(l)
}
```

#### Windows 命名管道服务器

```go
func startPipe(cfg *Config) {
    l, err := inbound.ListenNamedPipe(cfg.PipeAddr)
    server.Serve(l)
}
```

### 5. 路由注册

使用 chi 框架注册 API 路由，支持以下接口：

**基础接口：**

- `GET /` - 欢迎信息
- `GET /logs` - 日志流
- `GET /traffic` - 流量统计
- `GET /memory` - 内存使用
- `GET /version` - 版本信息

**配置管理：**

- `GET /configs` - 获取配置
- `PUT /configs` - 重载配置

**代理管理：**

- `GET /proxies` - 获取所有代理
- `GET /proxies/{name}` - 获取指定代理
- `PUT /proxies/{name}` - 切换代理

**其他接口：**

- `GET /rules` - 规则列表
- `GET /connections` - 连接列表
- `DELETE /connections` - 关闭所有连接
- `GET /providers/proxies` - 代理提供者
- `GET /providers/rules` - 规则提供者
- `GET /dns` - DNS 配置

## 认证机制

### Bearer Token 认证

```
Authorization: Bearer <secret>
```

### WebSocket Token 认证

```
ws://127.0.0.1:9090/logs?token=<secret>
```

## 使用示例

### 通过命令行参数启动

```bash
# 指定 API 地址和密钥
./mihomo -f config.yaml -ext-ctl "127.0.0.1:9090" -secret "my-secret"

# 使用环境变量
export CLASH_OVERRIDE_EXTERNAL_CONTROLLER="127.0.0.1:9090"
export CLASH_OVERRIDE_SECRET="my-secret"
./mihomo -f config.yaml
```

### API 调用示例

```bash
# 获取版本信息
curl http://127.0.0.1:9090/version

# 获取所有代理（需要认证）
curl -H "Authorization: Bearer your-secret-key" http://127.0.0.1:9090/proxies

# 切换代理
curl -X PUT -H "Authorization: Bearer your-secret-key" \
     -H "Content-Type: application/json" \
     -d '{"name":"ss-node"}' \
     http://127.0.0.1:9090/proxies/Proxy

# 获取配置
curl -H "Authorization: Bearer your-secret-key" http://127.0.0.1:9090/configs
```

## 关键特性

- **多协议支持**：HTTP、HTTPS、Unix Socket、Windows 命名管道
- **认证机制**：支持 Bearer Token 和 WebSocket token 认证
- **CORS 支持**：可配置跨域访问
- **并发启动**：所有服务器类型可同时运行
- **热重载**：支持 SIGHUP 信号重新加载配置
- **安全认证**：支持双向 TLS 认证（mTLS）
- **WebSocket 支持**：日志和流量统计支持 WebSocket 推送

## 安全建议

1. **使用强密钥**：设置足够复杂的 secret 密钥
2. **限制访问**：使用 `allow-lan` 和 `bind-address` 限制访问范围
3. **启用 HTTPS**：在生产环境中使用 HTTPS 而非 HTTP
4. **配置 CORS**：仅允许受信任的域名跨域访问
5. **使用 mTLS**：在需要高安全性的场景启用双向 TLS 认证
6. **定期更新**：保持 Mihomo 版本更新以获取安全修复

## 故障排查

### API 无法访问

- 检查端口是否被占用：`netstat -ano | findstr 9090`
- 检查防火墙设置
- 查看日志：`tail -f /path/to/mihomo.log`

### 认证失败

- 确认 secret 配置正确
- 检查 Authorization header 格式
- 验证 WebSocket token 参数

### Unix Socket 问题

- 确保路径有写权限
- 检查 socket 文件是否存在（需要先删除）
- 验证文件路径格式

### Windows 命名管道问题

- 确认管道名称格式：`\\.\pipe\mihomo`
- 检查管道是否已被占用
- 验证管理员权限（某些情况需要）
