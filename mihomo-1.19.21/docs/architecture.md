# Mihomo 项目架构设计文档

## 项目概述

Mihomo（原名 Clash.Meta）是一个功能强大的网络代理工具内核，使用 Go 语言编写。它是 Clash 的一个分支，提供了更多的协议支持和增强功能。

- **项目名称**: mihomo (Meta Kernel)
- **开发语言**: Go 1.20+
- **项目类型**: 网络代理工具内核
- **架构风格**: 分层架构，模块化设计
- **许可证**: GPL-3.0

## 整体架构设计

Mihomo 采用清晰的分层架构设计，从上到下依次为：

```
┌─────────────────────────────────────────────────────┐
│                    入口层 (Entry)                    │
│                   main.go                           │
└─────────────────────────────────────────────────────┘
                          ↓
┌─────────────────────────────────────────────────────┐
│                 核心控制器层 (Hub)                   │
│              hub/hub.go, hub/executor/              │
└─────────────────────────────────────────────────────┘
                          ↓
┌─────────────────────────────────────────────────────┐
│                 配置管理层 (Config)                  │
│                  config/                            │
└─────────────────────────────────────────────────────┘
                          ↓
┌──────────────┬──────────────┬──────────────┬────────┐
│  入站监听层   │  规则引擎层   │  出站适配层   │ DNS层  │
│   listener/  │    rules/    │   adapter/   │ dns/   │
└──────────────┴──────────────┴──────────────┴────────┘
                          ↓
┌─────────────────────────────────────────────────────┐
│                 流量隧道层 (Tunnel)                  │
│                   tunnel/                           │
└─────────────────────────────────────────────────────┘
                          ↓
┌─────────────────────────────────────────────────────┐
│                 传输层 (Transport)                  │
│                 transport/                          │
└─────────────────────────────────────────────────────┘
                          ↓
┌─────────────────────────────────────────────────────┐
│               核心组件层 (Component)                 │
│                  component/                         │
└─────────────────────────────────────────────────────┘
                          ↓
┌─────────────────────────────────────────────────────┐
│               基础工具层 (Common)                    │
│                   common/                           │
└─────────────────────────────────────────────────────┘
```

## 目录详细说明

### 1. adapter/ - 代理适配器层

**职责**: 提供代理协议的统一抽象接口和实现

**主要子目录**:
- `adapter.go` - 代理适配器核心接口定义
- `parser.go` - 代理配置解析器
- `inbound/` - 入站代理实现
- `outbound/` - 出站代理实现（20+ 种协议）
  - Shadowsocks, VMess, VLESS, Trojan, Snell, TUIC, Hysteria, WireGuard 等
- `outboundgroup/` - 代理组实现
  - Selector（选择器）
  - URLTest（自动测试）
  - LoadBalance（负载均衡）
  - Relay（中继）
  - Fallback（自动回退）
- `provider/` - 代理提供者
  - 文件提供者
  - HTTP 订阅提供者

**核心功能**:
- 统一的代理接口抽象
- 代理延迟测试和健康检查
- 代理状态管理和历史记录
- 支持多种代理协议和代理组策略

### 2. common/ - 通用工具库

**职责**: 提供项目通用的工具函数和数据结构

**主要子目录**:
- `atomic/` - 原子操作封装
- `arc/` - ARC（自适应替换缓存）算法实现
- `batch/` - 批处理工具
- `buf/` - 字节缓冲区操作
- `callback/` - 回调函数管理
- `cmd/` - 命令行工具
- `contextutils/` - 上下文工具
- `convert/` - 类型转换工具
- `deque/` - 双端队列实现
- `lru/` - LRU（最近最少使用）缓存
- `maphash/` - Map 哈希工具
- `murmur3/` - Murmur3 哈希算法
- `net/` - 网络工具
- `observable/` - 可观察对象模式
- `once/` - 一次性执行工具
- `orderedmap/` - 有序 Map 实现
- `picker/` - 选择器算法
- `pool/` - 内存池管理
- `queue/` - 队列实现
- `singledo/` - 单次执行工具
- `singleflight/` - 单飞模式（合并并发请求）
- `sockopt/` - Socket 选项操作
- `structure/` - 结构体工具
- `utils/` - 通用工具函数
- `xsync/` - 扩展同步工具
- `yaml/` - YAML 解析工具

**核心功能**:
- 高效的数据结构实现
- 内存管理和池化
- 并发控制工具
- 网络和 IO 辅助函数

### 3. component/ - 核心组件层

**职责**: 实现网络代理所需的核心功能组件

**主要子目录**:
- `auth/` - 认证组件（用户名密码、Bearer Token 等）
- `ca/` - 证书颁发机构管理
- `cidr/` - CIDR 网络地址处理
- `dhcp/` - DHCP 客户端实现
- `dialer/` - 网络拨号器（支持多种拨号策略）
- `ech/` - Encrypted Client Hello 支持
- `fakeip/` - Fake IP 池管理
- `generator/` - 配置生成器
- `geodata/` - GeoIP 和 GeoSite 数据加载
- `http/` - HTTP 客户端和服务端
- `iface/` - 网络接口管理
- `keepalive/` - TCP/UDP Keep-Alive
- `loopback/` - 回环设备管理
- `memory/` - 内存监控和管理
- `mmdb/` - MaxMind 数据库解析
- `mptcp/` - Multipath TCP 支持
- `nat/` - NAT（网络地址转换）处理
- `pool/` - 对象池管理
- `power/` - 电源管理（移动设备）
- `process/` - 进程识别和匹配
- `profile/` - 性能分析
- `proxydialer/` - 代理拨号器
- `resolver/` - DNS 解析器
- `resource/` - 资源管理
- `slowdown/` - 流量控制
- `sniffer/` - 协议嗅探器（识别流量类型）
- `tls/` - TLS 相关功能
- `trie/` - 前缀树实现（用于域名匹配）
- `updater/` - 配置和订阅更新器
- `wildcard/` - 通配符匹配

**核心功能**:
- 网络连接建立和管理
- 协议识别和嗅探
- DNS 解析和缓存
- 证书和 TLS 处理
- 资源和订阅更新

### 4. config/ - 配置管理层

**职责**: 解析和管理配置文件

**主要文件**:
- `config.go` - 配置结构定义和解析
- `initial.go` - 初始配置处理
- `utils.go` - 配置工具函数
- `utils_test.go` - 配置工具测试

**支持的配置项**:
- 入站配置（端口、认证等）
- 出站代理配置
- 代理组配置
- 规则配置
- DNS 配置
- TUN 配置
- 日志配置
- API 配置

**核心功能**:
- YAML 配置文件解析
- 配置验证
- 远程配置订阅
- 配置重载和热更新

### 5. constant/ - 常量定义层

**职责**: 定义项目中使用的常量、接口和类型

**主要文件**:
- `adapters.go` - 适配器接口定义
- `context.go` - 上下文相关常量
- `dns.go` - DNS 相关常量
- `listener.go` - 监听器类型常量
- `matcher.go` - 规则匹配器常量
- `metadata.go` - 元数据类型定义
- `path.go` - 路径常量
- `path_test.go` - 路径测试
- `rule.go` - 规则类型常量
- `tun.go` - TUN 模式常量
- `tunnel.go` - 隧道相关常量
- `version.go` - 版本信息

**主要子目录**:
- `features/` - 功能特性标志
- `provider/` - 提供者相关常量
- `sniffer/` - 嗅探器相关常量

**核心功能**:
- 接口定义和规范
- 类型系统
- 版本管理
- 功能开关

### 6. context/ - 上下文层

**职责**: 封装连接和 DNS 上下文信息

**主要文件**:
- `conn.go` - 连接上下文
- `dns.go` - DNS 上下文
- `packetconn.go` - 包连接上下文

**核心功能**:
- 请求元数据管理（源地址、目标地址、协议等）
- 上下文传递
- 连接状态跟踪

### 7. dns/ - DNS 服务层

**职责**: 实现 DNS 服务器和解析功能

**主要文件**:
- `server.go` - DNS 服务器
- `client.go` - DNS 客户端
- `resolver.go` - DNS 解析器
- `service.go` - DNS 服务接口
- `doh.go` - DNS over HTTPS 实现
- `dot.go` - DNS over TLS 实现
- `doq.go` - DNS over QUIC 实现
- `dialer.go` - DNS 拨号器
- `edns0_subnet.go` - EDNS0 子网支持
- `enhancer.go` - DNS 增强器
- `middleware.go` - DNS 中间件
- `policy.go` - DNS 策略
- `rcode.go` - DNS 返回码
- `system.go` - 系统 DNS 交互
- `system_posix.go` - POSIX 系统 DNS
- `system_windows.go` - Windows 系统 DNS
- `system_common.go` - 通用系统 DNS
- `util.go` - DNS 工具函数
- `dhcp.go` - DHCP DNS 配置
- `patch_android.go` - Android 补丁

**核心功能**:
- DoH/DoT/DoQ 支持
- Fake IP 模式
- DNS 缓存
- 分流解析（基于规则）
- DNS 劫持和重定向
- 多上游支持

### 8. hub/ - 核心控制器层

**职责**: 协调各模块，提供核心控制功能

**主要文件**:
- `hub.go` - 核心控制器
- `executor/` - 配置执行器目录
  - `executor.go` - 配置执行器实现

**主要子目录**:
- `route/` - RESTful API 路由

**核心功能**:
- 配置解析和验证
- 服务启动和关闭
- 配置热重载
- RESTful API 提供
- 信号处理（SIGHUP 重载配置，SIGINT/SIGTERM 退出）

### 9. listener/ - 入站监听层

**职责**: 实现各种入站连接监听器

**主要文件**:
- `listener.go` - 监听器核心接口
- `parse.go` - 监听器配置解析

**主要子目录**:
- `http/` - HTTP 代理监听器
- `socks/` - SOCKS5 代理监听器
- `mixed/` - 混合端口（HTTP + SOCKS5）监听器
- `redir/` - 透明代理（REDIRECT）监听器
- `tproxy/` - 透明代理（TPROXY）监听器
- `auth/` - 认证相关
- `config/` - 配置相关
- `inbound/` - 入站通用功能
- `inner/` - 内部连接处理
- `anytls/` - AnyTLS 协议支持
- `reality/` - Reality 协议支持
- `shadowsocks/` - Shadowsocks 入站
- `trojan/` - Trojan 入站
- `tuic/` - TUIC 入站
- `sing_tun/` - Sing TUN 模式
- `sing_vmess/` - Sing VMess 协议
- `sing_vless/` - Sing VLESS 协议
- `sing_shadowsocks/` - Sing Shadowsocks 协议
- `sing_hysteria2/` - Sing Hysteria2 协议
- `mieru/` - Mieru 协议支持
- `sudoku/` - Sudoku 协议支持
- `trusttunnel/` - Trust Tunnel 支持
- `tunnel/` - 隧道监听器

**核心功能**:
- 支持多种入站协议
- 用户认证
- 连接管理
- 透明代理支持
- TUN 模式支持

### 10. log/ - 日志系统层

**职责**: 提供日志记录功能

**主要文件**:
- `log.go` - 日志核心实现
- `level.go` - 日志级别定义
- `sing.go` - Sing 日志格式

**核心功能**:
- 分级日志（Debug, Info, Warning, Error, Fatal）
- 日志输出格式化
- 日志过滤
- 支持多种输出目标

### 11. rules/ - 规则引擎层

**职责**: 实现流量分流规则系统

**主要文件**:
- `parser.go` - 规则解析器

**主要子目录**:
- `common/` - 常见规则类型实现
  - DOMAIN, DOMAIN-SUFFIX, DOMAIN-KEYWORD
  - IP-CIDR, GEOIP, GEOSITE
  - PROCESS-NAME, PROCESS-PATH
  - SCRIPT 规则
- `logic/` - 逻辑规则
  - AND, OR, NOT 组合规则
- `logic_test/` - 逻辑规则测试
- `provider/` - 规则提供者
- `wrapper/` - 规则包装器

**核心功能**:
- 支持 30+ 种规则类型
- 规则优先级管理
- 逻辑规则组合
- 远程规则订阅
- 规则热更新

### 12. tunnel/ - 流量隧道层

**职责**: 实现流量转发的核心逻辑

**主要文件**:
- `tunnel.go` - 隧道核心实现

**核心功能**:
- 连接入站和出站
- 流量转发
- 连接管理
- 错误处理
- 性能优化（零拷贝、连接复用）

### 13. transport/ - 传输层

**职责**: 实现各种传输协议和传输方式

**支持的协议**:
- HTTP/HTTPS
- WebSocket
- gRPC
- QUIC
- h2
- h3
- 自定义传输协议

**核心功能**:
- 传输层抽象
- 多协议支持
- 传输加密
- 混淆和伪装

### 14. ntp/ - 时间同步层

**职责**: 提供 NTP 时间校准功能

**主要文件**:
- `time.go` - 时间管理
- `ntp/` - NTP 协议实现

**核心功能**:
- NTP 时间同步
- 时间校准
- 时钟偏差补偿

## 架构特点

### 1. 清晰的分层架构
- 从入口层到基础层，职责明确
- 每层专注于特定功能
- 层与层之间通过接口通信

### 2. 模块化设计
- 各模块职责明确，低耦合高内聚
- 易于扩展和维护
- 支持插件化开发

### 3. 丰富的协议支持
- 支持 20+ 种出站代理协议
- 支持 30+ 种规则类型
- 支持多种入站方式
- 支持多种传输协议

### 4. 高性能设计
- 基于 goroutine 并发
- 连接池复用
- 零拷贝优化
- LRU 缓存
- 内存池管理

### 5. 灵活的规则系统
- 支持多维度匹配（域名、IP、进程、协议）
- 支持逻辑规则组合（AND, OR, NOT）
- 支持远程规则订阅
- 规则热更新

### 6. 完善的 API
- RESTful API 接口
- 支持配置管理
- 支持代理切换
- 支持连接管理
- 支持日志查询

### 7. 可观测性
- 详细的日志系统
- 连接跟踪
- 延迟监控
- 性能分析工具

## 关键文件说明

### 入口和核心控制
- `main.go` - 程序入口，命令行参数解析，信号处理
- `hub/hub.go` - 核心控制器，协调各模块
- `hub/executor/executor.go` - 配置执行器，管理服务生命周期

### 配置和规则
- `config/config.go` - 配置结构定义和解析
- `rules/parser.go` - 规则解析器

### 代理和适配器
- `adapter/adapter.go` - 代理适配器接口定义
- `adapter/outbound/` - 出站代理实现
- `adapter/outboundgroup/` - 代理组实现

### 网络和传输
- `component/dialer/dialer.go` - 网络拨号器
- `transport/` - 传输层实现
- `tunnel/tunnel.go` - 流量隧道核心

### DNS 和解析
- `dns/server.go` - DNS 服务器
- `component/resolver/` - DNS 解析器
- `component/sniffer/` - 协议嗅探器

### 常量和接口
- `constant/adapters.go` - 适配器接口定义
- `constant/rule.go` - 规则类型定义
- `constant/version.go` - 版本信息

## 数据流

### 请求处理流程

```
1. 客户端请求 → listener/ (入站监听)
                  ↓
2. 连接处理 → context/ (上下文封装)
                  ↓
3. 规则匹配 → rules/ (规则引擎)
                  ↓
4. 代理选择 → adapter/outboundgroup/ (代理组)
                  ↓
5. 代理连接 → adapter/outbound/ (出站代理)
                  ↓
6. 流量转发 → tunnel/ (隧道层)
                  ↓
7. 传输层 → transport/ (传输协议)
                  ↓
8. 网络通信 → component/dialer/ (拨号器)
```

### 配置加载流程

```
1. 配置文件 → config/config.go (解析)
                  ↓
2. 配置验证 → hub/executor/ (验证)
                  ↓
3. 服务初始化 → hub/executor/ (初始化)
                  ↓
4. 服务启动 → hub/executor/ (启动)
                  ↓
5. 运行时管理 → hub/hub.go (管理)
```

## 扩展性

### 添加新的代理协议
1. 在 `adapter/outbound/` 下实现新的代理协议
2. 在 `constant/adapters.go` 中定义新的代理类型
3. 在 `config/` 中添加配置解析
4. 在 `rules/parser.go` 中添加规则支持

### 添加新的规则类型
1. 在 `rules/common/` 下实现新的规则类型
2. 在 `constant/rule.go` 中定义新的规则常量
3. 在 `rules/parser.go` 中添加解析逻辑

### 添加新的监听器
1. 在 `listener/` 下实现新的监听器
2. 在 `constant/listener.go` 中定义新的监听器类型
3. 在 `config/` 中添加配置解析

## 性能优化策略

1. **并发处理**: 使用 goroutine 并发处理连接
2. **连接复用**: 连接池和 Keep-Alive
3. **内存管理**: 内存池和零拷贝
4. **缓存策略**: LRU 缓存 DNS 和连接
5. **懒加载**: 按需加载资源
6. **批处理**: 批量处理请求

## 安全特性

1. **TLS 加密**: 支持 TLS 1.3 和自定义证书
2. **认证支持**: 用户名密码、Bearer Token
3. **协议混淆**: 支持多种混淆方式
4. **流量加密**: 端到端加密
5. **安全审计**: 连接日志和审计追踪

## 总结

Mihomo 采用分层架构设计，具有以下优势：

- **高内聚低耦合**: 各模块职责明确，易于维护和扩展
- **高性能**: 基于并发模型和内存优化
- **灵活性**: 支持多种协议和规则配置
- **可观测性**: 完善的日志和监控
- **安全性**: 多层安全保护机制

这种架构设计使得 Mihomo 能够高效、稳定地处理复杂的网络代理需求，同时保持良好的可维护性和可扩展性。