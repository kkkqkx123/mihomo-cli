# Mihomo RESTful API 文档

## 概述

Mihomo 提供完整的 RESTful API 接口，用于管理和监控代理服务。API 支持多种认证方式（Bearer Token 和 URL Token），并提供了 WebSocket 支持用于实时数据推送。

**API 地址配置**:

- HTTP 地址: 通过 `external-controller` 配置项设置
- HTTPS 地址: 通过 `external-controller-tls` 配置项设置
- Unix Socket: 通过 `external-controller-unix` 配置项设置
- Named Pipe: 通过 `external-controller-pipe` 配置项设置

**认证方式**:

- Bearer Token: `Authorization: Bearer <secret>`
- URL Token: `?token=<secret>` (仅用于 WebSocket 连接)
- Secret 通过 `secret` 配置项设置

**支持的方法**:

- GET
- POST
- PUT
- PATCH
- DELETE

**错误响应格式**:

```json
{
  "message": "错误描述"
}
```

## 基础 API

### 1. Hello

**端点**: `GET /`

**描述**: 检查 API 服务是否正常运行

**认证**: 需要

**响应**:

```json
{
  "hello": "mihomo"
}
```

---

### 2. 获取版本

**端点**: `GET /version`

**描述**: 获取 Mihomo 版本信息

**认证**: 需要

**响应**:

```json
{
  "meta": "mihomo",
  "version": "1.19.21"
}
```

---

### 3. 获取日志

**端点**: `GET /logs`

**描述**: 实时获取 Mihomo 日志，支持 WebSocket

**认证**: 需要

**查询参数**:

- `level` (可选): 日志级别，默认 `info`
  - 可选值: `silent`, `error`, `warning`, `info`, `debug`
- `format` (可选): 日志格式，默认 `default`
  - 可选值: `default`, `structured`

**WebSocket 升级**: 支持

**响应 (HTTP)**:

```json
{
  "type": "info",
  "payload": "日志内容"
}
```

**响应 (Structured 格式)**:

```json
{
  "time": "14:30:00",
  "level": "info",
  "message": "日志内容",
  "fields": []
}
```

---

### 4. 获取流量统计

**端点**: `GET /traffic`

**描述**: 获取实时流量统计信息，支持 WebSocket

**认证**: 需要

**WebSocket 升级**: 支持

**响应**:

```json
{
  "up": 1024,
  "down": 2048,
  "upTotal": 1024000,
  "downTotal": 2048000
}
```

**字段说明**:

- `up`: 当前上传速度（字节/秒）
- `down`: 当前下载速度（字节/秒）
- `upTotal`: 总上传流量（字节）
- `downTotal`: 总下载流量（字节）

---

### 5. 获取内存使用

**端点**: `GET /memory`

**描述**: 获取实时内存使用情况，支持 WebSocket

**认证**: 需要

**WebSocket 升级**: 支持

**响应**:

```json
{
  "inuse": 10485760,
  "oslimit": 0
}
```

**字段说明**:

- `inuse`: 当前内存使用量（字节）
- `oslimit`: 操作系统内存限制（字节，保留字段）

---

## 配置管理 API

### 6. 获取配置

**端点**: `GET /configs`

**描述**: 获取当前配置信息

**认证**: 需要

**响应**:

```json
{
  "port": 7890,
  "socks-port": 7891,
  "redir-port": 7892,
  "tproxy-port": 7893,
  "mixed-port": 7890,
  "allow-lan": true,
  "bind-address": "*",
  "mode": "rule",
  "log-level": "info",
  "ipv6": true,
  "sniffing": true,
  "tcp-concurrent": true,
  "find-process-mode": "off",
  "interface-name": "",
  "tun": {
    "enable": false,
    "device": "utun0",
    "stack": "system",
    "dns-hijack": ["any:53"],
    "auto-route": false,
    "auto-detect-interface": false,
    "mtu": 9000,
    "gso": false,
    "gso-max-size": 65536,
    "inet6-address": ["fdfe:dcba:9876::1/126"]
  },
  "tuic-server": {
    "enable": false,
    "listen": "0.0.0.0:10000",
    "token": ["password"],
    "certificate": "./server.crt",
    "private-key": "./server.key"
  },
  "skip-auth-prefixes": [],
  "lan-allowed-ips": ["0.0.0.0/0"],
  "lan-disallowed-ips": []
}
```

---

### 7. 更新配置

**端点**: `PUT /configs`

**描述**: 重载配置文件或使用新配置

**认证**: 需要

**注意**: 在 embed 模式下不可用

**查询参数**:

- `force` (可选): 是否强制更新，默认 `false`

**请求体**:

```json
{
  "path": "/path/to/config.yaml",
  "payload": "base64 encoded config or empty string"
}
```

**字段说明**:

- `path`: 配置文件路径，如果为空则使用默认配置文件
- `payload`: Base64 编码的配置内容，如果为空则从 path 读取

**响应**: 204 No Content

---

### 8. 部分更新配置

**端点**: `PATCH /configs`

**描述**: 部分更新配置项

**认证**: 需要

**注意**: 在 embed 模式下不可用

**请求体**:

```json
{
  "port": 7890,
  "socks-port": 7891,
  "redir-port": 7892,
  "tproxy-port": 7893,
  "mixed-port": 7890,
  "allow-lan": true,
  "bind-address": "*",
  "mode": "rule",
  "log-level": "info",
  "ipv6": true,
  "sniffing": true,
  "tcp-concurrent": true,
  "find-process-mode": "off",
  "interface-name": "eth0",
  "tun": {
    "enable": true,
    "device": "utun0",
    "stack": "system",
    "dns-hijack": ["any:53"],
    "auto-route": true,
    "auto-detect-interface": true,
    "mtu": 9000,
    "gso": false,
    "gso-max-size": 65536,
    "inet6-address": ["fdfe:dcba:9876::1/126"]
  },
  "tuic-server": {
    "enable": false,
    "listen": "0.0.0.0:10000",
    "token": ["password"]
  },
  "skip-auth-prefixes": ["127.0.0.1/32"],
  "lan-allowed-ips": ["0.0.0.0/0"],
  "lan-disallowed-ips": []
}
```

**响应**: 204 No Content

---

### 9. 更新 Geo 数据库

**端点**: `POST /configs/geo`

**描述**: 更新 GeoIP 和 GeoSite 数据库

**认证**: 需要

**注意**: 在 embed 模式下不可用

**响应**: 204 No Content

---

## 代理管理 API

### 10. 获取所有代理

**端点**: `GET /proxies`

**描述**: 获取所有代理信息（包括代理提供者中的代理）

**认证**: 需要

**响应**:

```json
{
  "proxies": {
    "proxy-name": {
      "name": "proxy-name",
      "type": "ss",
      "udp": true,
      "xudp": true,
      "history": [
        {
          "time": "2024-01-01T00:00:00.000Z",
          "delay": 100
        }
      ],
      "alive": true,
      "now": "selected-proxy-name",
      "all": ["proxy1", "proxy2", "proxy3"],
      "provider": "provider-name",
      "external-controller": "http://127.0.0.1:9090"
    }
  }
}
```

---

### 11. 获取指定代理

**端点**: `GET /proxies/{name}`

**描述**: 获取指定代理的详细信息

**认证**: 需要

**路径参数**:

- `name`: 代理名称（URL 编码）

**响应**:

```json
{
  "name": "proxy-name",
  "type": "ss",
  "udp": true,
  "xudp": true,
  "history": [
    {
      "time": "2024-01-01T00:00:00.000Z",
      "delay": 100
    }
  ],
  "alive": true,
  "now": "selected-proxy-name",
  "all": ["proxy1", "proxy2", "proxy3"],
  "provider": "provider-name"
}
```

---

### 12. 测试代理延迟

**端点**: `GET /proxies/{name}/delay`

**描述**: 测试指定代理的延迟

**认证**: 需要

**路径参数**:

- `name`: 代理名称（URL 编码）

**查询参数**:

- `url` (可选): 测试 URL，默认配置中的 URL
- `timeout` (可选): 超时时间（毫秒），默认 5000
- `expected` (可选): 期望的 HTTP 状态码范围，例如 `200,204` 或 `200-299`

**响应**:

```json
{
  "delay": 100
}
```

**错误响应**:

- 504 Gateway Timeout: 测试超时
- 503 Service Unavailable: 测试失败

---

### 13. 切换代理

**端点**: `PUT /proxies/{name}`

**描述**: 切换代理组中选中的代理

**认证**: 需要

**路径参数**:

- `name`: 代理组名称（URL 编码）

**请求体**:

```json
{
  "name": "target-proxy-name"
}
```

**字段说明**:

- `name`: 要切换到的目标代理名称

**响应**: 204 No Content

**错误响应**:

- 400 Bad Request: 代理不是 Selector 类型或目标代理不存在

---

### 14. 取消固定代理

**端点**: `DELETE /proxies/{name}`

**描述**: 取消代理组中固定的代理（恢复自动选择）

**认证**: 需要

**路径参数**:

- `name`: 代理组名称（URL 编码）

**响应**: 204 No Content

---

## 代理组管理 API

### 15. 获取所有代理组

**端点**: `GET /group`

**描述**: 获取所有代理组信息

**认证**: 需要

**响应**:

```json
{
  "proxies": [
    {
      "name": "group-name",
      "type": "select",
      "udp": true,
      "xudp": true,
      "history": [],
      "alive": true,
      "now": "selected-proxy",
      "all": ["proxy1", "proxy2", "proxy3"]
    }
  ]
}
```

---

### 16. 获取指定代理组

**端点**: `GET /group/{name}`

**描述**: 获取指定代理组的详细信息

**认证**: 需要

**路径参数**:

- `name`: 代理组名称（URL 编码）

**响应**:

```json
{
  "name": "group-name",
  "type": "select",
  "udp": true,
  "xudp": true,
  "history": [],
  "alive": true,
  "now": "selected-proxy",
  "all": ["proxy1", "proxy2", "proxy3"]
}
```

**错误响应**:

- 404 Not Found: 代理组不存在

---

### 17. 测试代理组延迟

**端点**: `GET /group/{name}/delay`

**描述**: 测试代理组中所有代理的延迟

**认证**: 需要

**路径参数**:

- `name`: 代理组名称（URL 编码）

**查询参数**:

- `url` (可选): 测试 URL，默认配置中的 URL
- `timeout` (可选): 超时时间（毫秒），默认 5000
- `expected` (可选): 期望的 HTTP 状态码范围

**响应**:

```json
{
  "proxy1": {
    "time": "2024-01-01T00:00:00.000Z",
    "delay": 100
  },
  "proxy2": {
    "time": "2024-01-01T00:00:00.000Z",
    "delay": 200
  }
}
```

**错误响应**:

- 404 Not Found: 代理组不存在
- 504 Gateway Timeout: 测试超时

---

## 规则管理 API

### 18. 获取所有规则

**端点**: `GET /rules`

**描述**: 获取所有规则信息

**认证**: 需要

**响应**:

```json
{
  "rules": [
    {
      "index": 0,
      "type": "DOMAIN-SUFFIX",
      "payload": "google.com",
      "proxy": "Proxy",
      "size": -1,
      "extra": {
        "disabled": false,
        "hitCount": 100,
        "hitAt": "2024-01-01T00:00:00.000Z",
        "missCount": 10,
        "missAt": "2024-01-01T00:00:00.000Z"
      }
    },
    {
      "index": 1,
      "type": "GEOIP",
      "payload": "CN",
      "proxy": "DIRECT",
      "size": 10000,
      "extra": {
        "disabled": false,
        "hitCount": 1000,
        "hitAt": "2024-01-01T00:00:00.000Z",
        "missCount": 50,
        "missAt": "2024-01-01T00:00:00.000Z"
      }
    }
  ]
}
```

**字段说明**:

- `index`: 规则索引
- `type`: 规则类型（DOMAIN, DOMAIN-SUFFIX, DOMAIN-KEYWORD, IP-CIDR, GEOIP, GEOSITE, PROCESS-NAME 等）
- `payload`: 规则匹配内容
- `proxy`: 使用的代理
- `size`: 规则包含的记录数（GEOIP 和 GEOSITE 规则）
- `extra`: 额外信息
  - `disabled`: 是否禁用
  - `hitCount`: 命中次数
  - `hitAt`: 最后命中时间
  - `missCount`: 未命中次数
  - `missAt`: 最后未命中时间

---

### 19. 禁用/启用规则

**端点**: `PATCH /rules/disable`

**描述**: 批量禁用或启用规则

**认证**: 需要

**注意**: 在 embed 模式下不可用

**请求体**:

```json
{
  "0": true,
  "1": false,
  "2": true
}
```

**字段说明**:

- Key: 规则索引（整数）
- Value: 是否禁用（true=禁用，false=启用）

**响应**: 204 No Content

---

## 连接管理 API

### 20. 获取所有连接

**端点**: `GET /connections`

**描述**: 获取当前所有活跃连接信息

**认证**: 需要

**查询参数**:

- `interval` (可选): WebSocket 推送间隔（毫秒），默认 1000，仅在 WebSocket 模式下生效

**WebSocket 升级**: 支持

**响应**:

```json
{
  "downloadTotal": 1024000,
  "uploadTotal": 512000,
  "connections": [
    {
      "id": "conn-id",
      "metadata": {
        "net": "tcp",
        "type": "HTTP",
        "sourceIP": "192.168.1.1",
        "destinationIP": "1.1.1.1",
        "sourcePort": "12345",
        "destinationPort": "443",
        "host": "example.com",
        "dnsMode": "normal",
        "processPath": "/usr/bin/curl",
        "specialProxy": "Proxy"
      },
      "upload": 1024,
      "download": 2048,
      "start": "2024-01-01T00:00:00.000Z",
      "chains": ["proxy1", "proxy2"],
      "rule": "DOMAIN-SUFFIX,google.com",
      "rulePayload": "google.com"
    }
  ]
}
```

**字段说明**:

- `downloadTotal`: 总下载流量（字节）
- `uploadTotal`: 总上传流量（字节）
- `connections`: 连接列表
  - `id`: 连接 ID
  - `metadata`: 连接元数据
    - `net`: 网络类型（tcp, udp）
    - `type`: 连接类型（HTTP, SOCKS5 等）
    - `sourceIP`: 源 IP 地址
    - `destinationIP`: 目标 IP 地址
    - `sourcePort`: 源端口
    - `destinationPort`: 目标端口
    - `host`: 目标主机名
    - `dnsMode`: DNS 解析模式
    - `processPath`: 进程路径
    - `specialProxy`: 使用的代理
  - `upload`: 上传流量（字节）
  - `download`: 下载流量（字节）
  - `start`: 连接开始时间
  - `chains`: 代理链
  - `rule`: 匹配的规则
  - `rulePayload`: 规则内容

---

### 21. 关闭指定连接

**端点**: `DELETE /connections/{id}`

**描述**: 关闭指定 ID 的连接

**认证**: 需要

**路径参数**:

- `id`: 连接 ID

**响应**: 204 No Content

---

### 22. 关闭所有连接

**端点**: `DELETE /connections`

**描述**: 关闭所有活跃连接

**认证**: 需要

**响应**: 204 No Content

---

## 代理提供者 API

### 23. 获取所有代理提供者

**端点**: `GET /providers/proxies`

**描述**: 获取所有代理提供者信息

**认证**: 需要

**响应**:

```json
{
  "providers": {
    "provider-name": {
      "name": "provider-name",
      "type": "file",
      "vehicleType": "File",
      "proxies": [],
      "updatedAt": "2024-01-01T00:00:00.000Z",
      "proxyTotal": 10
    }
  }
}
```

---

### 24. 获取指定代理提供者

**端点**: `GET /providers/proxies/{providerName}`

**描述**: 获取指定代理提供者的详细信息

**认证**: 需要

**路径参数**:

- `providerName`: 提供者名称（URL 编码）

**响应**:

```json
{
  "name": "provider-name",
  "type": "file",
  "vehicleType": "File",
  "proxies": [
    {
      "name": "proxy1",
      "type": "ss",
      "udp": true,
      "xudp": true,
      "history": [],
      "alive": true
    }
  ],
  "updatedAt": "2024-01-01T00:00:00.000Z",
  "proxyTotal": 10
}
```

---

### 25. 更新代理提供者

**端点**: `PUT /providers/proxies/{providerName}`

**描述**: 更新指定代理提供者的代理列表

**认证**: 需要

**路径参数**:

- `providerName`: 提供者名称（URL 编码）

**响应**: 204 No Content

**错误响应**:

- 404 Not Found: 提供者不存在
- 503 Service Unavailable: 更新失败

---

### 26. 代理提供者健康检查

**端点**: `GET /providers/proxies/{providerName}/healthcheck`

**描述**: 对代理提供者中的所有代理进行健康检查

**认证**: 需要

**路径参数**:

- `providerName`: 提供者名称（URL 编码）

**响应**: 204 No Content

---

### 27. 获取提供者中的代理

**端点**: `GET /providers/proxies/{providerName}/{name}`

**描述**: 获取代理提供者中指定代理的详细信息

**认证**: 需要

**路径参数**:

- `providerName`: 提供者名称（URL 编码）
- `name`: 代理名称（URL 编码）

**响应**:

```json
{
  "name": "proxy-name",
  "type": "ss",
  "udp": true,
  "xudp": true,
  "history": [],
  "alive": true
}
```

---

### 28. 测试提供者中的代理延迟

**端点**: `GET /providers/proxies/{providerName}/{name}/healthcheck`

**描述**: 测试代理提供者中指定代理的延迟

**认证**: 需要

**路径参数**:

- `providerName`: 提供者名称（URL 编码）
- `name`: 代理名称（URL 编码）

**响应**:

```json
{
  "delay": 100
}
```

---

## 规则提供者 API

### 29. 获取所有规则提供者

**端点**: `GET /providers/rules`

**描述**: 获取所有规则提供者信息

**认证**: 需要

**响应**:

```json
{
  "providers": {
    "provider-name": {
      "name": "provider-name",
      "type": "file",
      "vehicleType": "File",
      "updatedAt": "2024-01-01T00:00:00.000Z",
      "ruleCount": 100
    }
  }
}
```

---

### 30. 更新规则提供者

**端点**: `PUT /providers/rules/{name}`

**描述**: 更新指定规则提供者的规则列表

**认证**: 需要

**路径参数**:

- `name`: 提供者名称（URL 编码）

**响应**: 204 No Content

**错误响应**:

- 404 Not Found: 提供者不存在
- 503 Service Unavailable: 更新失败

---

## 缓存管理 API

### 31. 清空 FakeIP 池

**端点**: `POST /cache/fakeip/flush`

**描述**: 清空 FakeIP 地址池

**认证**: 需要

**响应**: 204 No Content

**错误响应**:

- 400 Bad Request: FakeIP 未启用或清空失败

---

### 32. 清空 DNS 缓存

**端点**: `POST /cache/dns/flush`

**描述**: 清空 DNS 缓存

**认证**: 需要

**响应**: 204 No Content

---

## DNS API

### 33. 查询 DNS

**端点**: `GET /dns/query`

**描述**: 执行 DNS 查询

**认证**: 需要

**查询参数**:

- `name`: 要查询的域名（必需）
- `type`: DNS 记录类型（可选），默认 `A`
  - 可选值: `A`, `AAAA`, `CNAME`, `MX`, `TXT`, `NS`, `SRV` 等

**响应**:

```json
{
  "Status": 0,
  "Question": [
    {
      "name": "example.com.",
      "type": 1,
      "class": 1
    }
  ],
  "TC": false,
  "RD": true,
  "RA": true,
  "AD": false,
  "CD": false,
  "Answer": [
    {
      "name": "example.com.",
      "type": 1,
      "TTL": 300,
      "data": "93.184.216.34"
    }
  ],
  "Authority": [],
  "Additional": []
}
```

**字段说明**:

- `Status`: DNS 响应码（0=成功）
- `Question`: 查询问题列表
- `TC`: 是否截断
- `RD`: 是否期望递归
- `RA`: 是否可用递归
- `AD`: 是否已验证
- `CD`: 是否禁用检查
- `Answer`: 响应答案列表
- `Authority`: 权威记录列表
- `Additional`: 附加记录列表

**错误响应**:

- 500 Internal Server Error: DNS 未启用或查询失败

---

### 34. DoH 服务器

**端点**: `GET /doh`, `POST /doh`

**描述**: DNS over HTTPS 服务器接口

**认证**: 需要（如果 DoH 路径需要认证）

**GET 方法查询参数**:

- `dns`: Base64 URL 编码的 DNS 查询数据（必需）

**POST 方法**:

- Content-Type: `application/dns-message`
- Body: DNS 查询数据（二进制格式，最大 65535 字节）

**响应**:

- Content-Type: `application/dns-message`
- Body: DNS 响应数据（二进制格式）

---

## 系统管理 API

### 35. 重启服务

**端点**: `POST /restart`

**描述**: 重启 Mihomo 服务

**认证**: 需要

**注意**: 在 embed 模式下不可用

**响应**:

```json
{
  "status": "ok"
}
```

---

### 36. 升级核心

**端点**: `POST /upgrade`

**描述**: 升级 Mihomo 核心程序

**认证**: 需要

**注意**: 在 embed 模式下不可用

**查询参数**:

- `channel` (可选): 更新通道（例如: `alpha`, `beta`, `stable`）
- `force` (可选): 是否强制更新，默认 `false`

**响应**:

```json
{
  "status": "ok"
}
```

**错误响应**:

- 500 Internal Server Error: 升级失败

---

### 37. 更新 Geo 数据库

**端点**: `POST /upgrade/geo`

**描述**: 更新 GeoIP 和 GeoSite 数据库

**认证**: 需要

**注意**: 在 embed 模式下不可用

**响应**:

```json
{
  "status": "ok"
}
```

---

### 38. 更新 UI

**端点**: `POST /upgrade/ui`

**描述**: 更新 Web UI 界面

**认证**: 需要

**响应**:

```json
{
  "status": "ok"
}
```

**错误响应**:

- 500 Internal Server Error: 更新失败

---

## 调试 API

### 39. 触发垃圾回收

**端点**: `PUT /debug/gc`

**描述**: 触发 Go 运行时垃圾回收

**认证**: 需要

**注意**: 仅在 Debug 模式下可用

**响应**: 204 No Content

---

### 40. 性能分析

**端点**: `GET /debug/pprof/*`

**描述**: pprof 性能分析工具接口

**认证**: 需要

**注意**: 仅在 Debug 模式下可用

**支持的端点**:

- `GET /debug/pprof/` - pprof 首页
- `GET /debug/pprof/goroutine` - Goroutine 堆栈
- `GET /debug/pprof/heap` - 堆内存分析
- `GET /debug/pprof/threadcreate` - 线程创建分析
- `GET /debug/pprof/block` - 阻塞分析
- `GET /debug/pprof/mutex` - 互斥锁分析
- `GET /debug/pprof/profile` - CPU profile
- `GET /debug/pprof/trace` - 执行追踪

---

## Web UI

### 41. Web UI 界面

**端点**: `GET /ui`, `GET /ui/*`

**描述**: 访问 Web UI 界面

**认证**: 需要（根据配置）

**注意**: 需要通过 `external-ui` 配置项设置 UI 路径

**响应**: 静态文件或重定向到 `/ui/`

---

## 外部路由

### 42. 外部自定义路由

**端点**: 动态注册

**描述**: 支持通过 `Register` 函数注册外部自定义路由

**认证**: 根据自定义路由配置

**使用方法**:

```go
route.Register(func(r chi.Router) {
    r.Get("/custom", customHandler)
})
```

---

## WebSocket 支持

以下 API 支持 WebSocket 连接，用于实时数据推送：

1. `GET /logs` - 实时日志推送
2. `GET /traffic` - 实时流量统计推送
3. `GET /memory` - 实时内存使用推送
4. `GET /connections` - 实时连接列表推送

**WebSocket 连接**:

- HTTP Upgrade: `Upgrade: websocket`
- 认证方式: 使用 URL 参数 `?token=<secret>`

**WebSocket 数据格式**:

- 所有 WebSocket 数据均为文本格式
- 每条消息为一个完整的 JSON 对象

---

## CORS 支持

API 支持 CORS（跨域资源共享）配置：

**配置项**:

```yaml
external-controller: 127.0.0.1:9090
external-ui: ./ui
secret: your-secret

external-controller-cors:
  allow-origins:
    - https://example.com
    - https://ui.example.com
  allow-private-network: true
```

**CORS 响应头**:

- `Access-Control-Allow-Origin`: 配置的允许源
- `Access-Control-Allow-Methods`: GET, POST, PUT, PATCH, DELETE
- `Access-Control-Allow-Headers`: Content-Type, Authorization
- `Access-Control-Max-Age`: 300

---

## 错误码说明

| HTTP 状态码 | 说明                           |
| ----------- | ------------------------------ |
| 200         | 请求成功                       |
| 204         | 请求成功，无返回内容           |
| 400         | 请求参数错误                   |
| 401         | 未授权（认证失败）             |
| 403         | 禁止访问                       |
| 404         | 资源不存在                     |
| 405         | 方法不允许                     |
| 426         | 需要升级（WebSocket 握手失败） |
| 500         | 服务器内部错误                 |
| 503         | 服务不可用                     |
| 504         | 请求超时                       |

---

## 使用示例

### 1. 获取所有代理

```bash
curl -H "Authorization: Bearer your-secret" \
  http://127.0.0.1:9090/proxies
```

### 2. 切换代理

```bash
curl -X PUT \
  -H "Authorization: Bearer your-secret" \
  -H "Content-Type: application/json" \
  -d '{"name": "proxy-name"}' \
  http://127.0.0.1:9090/proxies/Proxy
```

### 3. 测试代理延迟

```bash
curl -H "Authorization: Bearer your-secret" \
  "http://127.0.0.1:9090/proxies/proxy-name/delay?url=https://www.google.com/generate_204&timeout=5000"
```

### 4. 关闭所有连接

```bash
curl -X DELETE \
  -H "Authorization: Bearer your-secret" \
  http://127.0.0.1:9090/connections
```

### 5. DNS 查询

```bash
curl -H "Authorization: Bearer your-secret" \
  "http://127.0.0.1:9090/dns/query?name=example.com&type=A"
```

### 6. WebSocket 连接获取实时日志

```javascript
const ws = new WebSocket("ws://127.0.0.1:9090/logs?token=your-secret");
ws.onmessage = (event) => {
  const log = JSON.parse(event.data);
  console.log(log.type, log.payload);
};
```

---

## 注意事项

1. **认证**: 大部分 API 需要认证，请在请求头中添加 `Authorization: Bearer <secret>`
2. **URL 编码**: 代理名称、提供者名称等参数需要使用 URL 编码
3. **时间格式**: 所有时间字段使用 RFC3339 格式（例如: `2024-01-01T00:00:00.000Z`）
4. **Embed 模式**: 某些 API 在 embed 模式下不可用
5. **Debug 模式**: 调试 API 仅在 Debug 模式下可用
6. **WebSocket**: WebSocket 连接使用 URL 参数 `?token=<secret>` 进行认证
7. **CORS**: 默认启用 CORS，可通过配置限制访问源

---

## 相关资源

- **官方文档**: https://wiki.metacubex.one/api/
- **配置文件**: `config.yaml`
- **Web UI**: [metacubexd](https://github.com/MetaCubeX/metacubexd)
