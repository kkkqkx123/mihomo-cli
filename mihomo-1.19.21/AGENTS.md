# Mihomo (Meta Kernel) - 项目上下文文档

## 项目概述

Mihomo（原名 Clash.Meta）是一个功能强大的网络代理工具内核，使用 Go 语言编写。它是 Clash 的一个分支，提供了更多的协议支持和增强功能。

### 主要特性

- **多协议支持**: VMess, VLESS, Shadowsocks, Trojan, Snell, TUIC, Hysteria, WireGuard 等
- **代理服务器**: 本地 HTTP/HTTPS/SOCKS 服务器，支持认证
- **DNS 服务器**: 内置 DNS 服务器，支持 DoH/DoT 上游和 Fake IP
- **规则引擎**: 基于域名、GEOIP、IPCIDR 或进程的规则系统
- **代理组**: 支持自动回退、负载均衡、基于延迟的自动节点选择
- **远程提供者**: 支持远程订阅节点列表
- **TUN 模式**: 支持虚拟网卡模式（使用 gvisor 或 system 栈）
- **RESTful API**: 完整的 HTTP API 控制器
- **透明代理**: 支持 Linux iptables/nftables 透明代理

## 技术栈

- **语言**: Go 1.20+
- **模块**: `github.com/metacubex/mihomo`
- **版本**: 1.10.0+
- **许可证**: GPL-3.0

## 项目结构

```
mihomo/
├── main.go              # 程序入口
├── adapter/             # 代理适配器
│   ├── adapter.go       # 代理实例封装
│   ├── outbound/        # 出站代理实现 (SS, VMess, Trojan 等)
│   ├── outboundgroup/   # 代理组实现 (Selector, URLTest, LoadBalance 等)
│   └── provider/        # 代理提供者 (文件/HTTP订阅)
├── common/              # 通用工具库
│   ├── atomic/          # 原子操作封装
│   ├── lru/             # LRU 缓存
│   ├── pool/            # 内存池
│   └── utils/           # 工具函数
├── component/           # 核心组件
│   ├── auth/            # 认证组件
│   ├── dialer/          # 拨号器
│   ├── fakeip/          # Fake IP 池
│   ├── geodata/         # GeoIP/GeoSite 数据处理
│   ├── resolver/        # DNS 解析器
│   ├── sniffer/         # 协议嗅探器
│   └── tls/             # TLS 相关
├── config/              # 配置解析
├── constant/            # 常量定义
├── context/             # 连接上下文
├── dns/                 # DNS 服务器实现
├── hub/                 # 核心控制器
│   ├── executor/        # 配置执行器
│   └── route/           # RESTful API 路由
├── listener/            # 入站监听器
│   ├── http/            # HTTP 代理
│   ├── socks/           # SOCKS 代理
│   ├── mixed/           # 混合端口
│   ├── redir/           # 透明代理 (REDIRECT)
│   ├── tproxy/          # 透明代理 (TPROXY)
│   └── sing_tun/        # TUN 模式
├── log/                 # 日志系统
├── rules/               # 规则系统
│   ├── common/          # 规则类型实现
│   ├── logic/           # 逻辑规则 (AND/OR/NOT)
│   └── provider/        # 规则提供者
├── tunnel/              # 流量隧道核心
└── transport/           # 传输层实现
```

## 构建与运行

### 环境要求

- Go 1.20 或更高版本

### 基本构建

```bash
# 克隆仓库
git clone https://github.com/MetaCubeX/mihomo.git
cd mihomo

# 下载依赖
go mod download

# 构建 (基础版本)
go build

# 构建 (带 gvisor TUN 支持)
go build -tags with_gvisor
```

### Makefile 构建

```bash
# 构建当前平台版本
make all

# 构建特定平台
make linux-amd64-v3
make windows-amd64-v3
make darwin-arm64

# 构建所有架构
make all-arch

# 清理构建产物
make clean
```

### 运行

```bash
# 使用默认配置运行
./mihomo

# 指定配置文件
./mihomo -f /path/to/config.yaml

# 使用 base64 编码的配置字符串
./mihomo -config <base64-encoded-config>

# 测试配置有效性
./mihomo -t

# 显示版本
./mihomo -v
```

### 命令行参数

| 参数 | 环境变量 | 说明 |
|------|----------|------|
| `-d` | `CLASH_HOME_DIR` | 配置目录 |
| `-f` | `CLASH_CONFIG_FILE` | 配置文件路径 |
| `-config` | `CLASH_CONFIG_STRING` | Base64 编码的配置字符串 |
| `-ext-ui` | `CLASH_OVERRIDE_EXTERNAL_UI_DIR` | 覆盖外部 UI 目录 |
| `-ext-ctl` | `CLASH_OVERRIDE_EXTERNAL_CONTROLLER` | 覆盖 API 控制器地址 |
| `-secret` | `CLASH_OVERRIDE_SECRET` | 覆盖 API 密钥 |
| `-m` | - | 启用 geodata 模式 |
| `-t` | - | 测试配置并退出 |
| `-v` | - | 显示版本 |

## 开发规范

### 代码风格

项目使用 `golangci-lint` 进行代码检查，配置如下：

```yaml
linters:
  enable:
    - gofumpt      # 格式化工具
    - staticcheck  # 静态分析
    - govet        # 标准分析工具
    - gci          # 导入排序
```

### 导入排序规则

1. 标准库
2. `github.com/metacubex/mihomo` 前缀
3. 第三方库

### 测试

```bash
# 运行所有测试
go test ./...

# 运行 lint
make lint

# 运行 vet
make vet
```

## 核心架构

### 1. 入口 (main.go)

程序入口负责：
- 解析命令行参数
- 初始化配置
- 启动 Hub（核心控制器）
- 处理信号（SIGHUP 重载配置，SIGINT/SIGTERM 退出）

### 2. Hub 层 (hub/)

核心协调器，负责：
- 配置解析与验证
- 启动/重载/关闭服务
- 初始化 RESTful API 路由

### 3. 配置层 (config/)

处理 YAML 配置解析，包括：
- 入站配置（端口、认证等）
- 出站代理配置
- 代理组配置
- 规则配置
- DNS 配置
- TUN 配置

### 4. 适配器层 (adapter/)

代理实例的抽象和封装：
- `Proxy`: 代理实例包装器，包含延迟历史、存活状态
- `outbound/`: 各种代理协议实现
- `outboundgroup/`: 代理组实现（选择器、自动测试、负载均衡）

### 5. 监听器层 (listener/)

入站连接处理：
- HTTP 代理
- SOCKS5 代理
- 混合端口（HTTP + SOCKS5）
- 透明代理（REDIRECT/TPROXY）
- TUN 虚拟网卡

### 6. 规则层 (rules/)

流量路由决策：
- 域名规则（DOMAIN, DOMAIN-SUFFIX, DOMAIN-KEYWORD）
- IP 规则（IP-CIDR, GEOIP）
- 进程规则（PROCESS-NAME, PROCESS-PATH）
- 逻辑规则（AND, OR, NOT）

### 7. DNS 层 (dns/)

DNS 解析系统：
- 支持 DoH/DoT/DoQ
- Fake IP 模式
- DNS 缓存
- 分流解析

### 8. 隧道层 (tunnel/)

核心流量转发逻辑，连接入站和出站。

## 配置文件结构

配置文件为 YAML 格式，主要包含以下部分：

```yaml
# 端口配置
mixed-port: 10801          # HTTP + SOCKS 混合端口
port: 7890                 # HTTP 端口
socks-port: 7891          # SOCKS 端口
redir-port: 7892          # 透明代理端口
tproxy-port: 7893         # TProxy 端口

# 通用配置
allow-lan: true
bind-address: "*"
mode: rule
log-level: debug
ipv6: true

# DNS 配置
dns:
  enable: true
  listen: 0.0.0.0:1053
  enhanced-mode: fake-ip
  nameserver:
    - 8.8.8.8
    - https://doh.pub/dns-query

# TUN 配置
tun:
  enable: true
  stack: system  # 或 gvisor
  device: utun0

# 代理配置
proxies:
  - name: "ss-node"
    type: ss
    server: server.com
    port: 8388
    cipher: aes-256-gcm
    password: "password"

# 代理组配置
proxy-groups:
  - name: "Proxy"
    type: select
    proxies:
      - ss-node
      - DIRECT

# 规则配置
rules:
  - DOMAIN-SUFFIX,google.com,Proxy
  - GEOIP,CN,DIRECT
  - MATCH,Proxy
```

## API 接口

Mihomo 提供 RESTful API 用于控制和监控：

- `GET /version` - 版本信息
- `GET /configs` - 获取配置
- `PUT /configs` - 重载配置
- `GET /proxies` - 获取所有代理
- `GET /proxies/{name}` - 获取指定代理
- `PUT /proxies/{name}` - 切换代理
- `GET /rules` - 获取规则列表
- `GET /connections` - 获取连接列表
- `DELETE /connections` - 关闭所有连接

API 文档参考: https://wiki.metacubex.one/api/

## 调试与日志

```bash
# 设置日志级别为 debug
log-level: debug

# 使用 API 获取调试信息
curl http://127.0.0.1:9093/debug/pprof/goroutine
```

## 相关资源

- **官方文档**: https://wiki.metacubex.one/
- **配置示例**: `./docs/config.yaml`
- **Web UI**: [metacubexd](https://github.com/MetaCubeX/metacubexd)
- **规则数据**: [meta-rules-dat](https://github.com/MetaCubeX/meta-rules-dat)

## 注意事项

1. 本项目为 GPL-3.0 许可证
2. 非 `MetaCubeX` 附属的下游项目不得在名称中包含 `mihomo`
3. 上游项目：Dreamacro/clash, SagerNet/sing-box
