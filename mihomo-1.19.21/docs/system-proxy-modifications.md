# Mihomo内核系统代理配置修改分析

本文档详细分析mihomo内核在开启系统代理时对系统配置的修改，包括不同代理模式和不同操作系统平台的具体实现。

## 目录

- [概述](#概述)
- [系统代理模式](#系统代理模式)
  - [标准系统代理](#标准系统代理)
  - [TUN模式](#tun模式)
  - [TProxy透明代理](#tproxy透明代理)
- [Windows平台配置修改](#windows平台配置修改)
- [Linux平台配置修改](#linux平台配置修改)
- [路由表修改详解](#路由表修改详解)
- [iptables规则详解](#iptables规则详解)
- [配置恢复机制](#配置恢复机制)

---

## 概述

Mihomo内核支持多种系统代理模式，每种模式对系统配置的修改方式不同：

| 代理模式 | 适用平台 | 配置修改类型 | 权限要求 |
|---------|---------|-------------|---------|
| 标准系统代理 | Windows/Linux | 注册表/环境变量 | 管理员权限 |
| TUN模式 | Windows/Linux/macOS | 虚拟网卡/路由表 | 管理员权限 |
| TProxy透明代理 | Linux | iptables/路由表 | root权限 |

---

## 系统代理模式

### 标准系统代理

标准系统代理通过修改系统代理设置实现流量转发，是最基础的代理模式。

**实现位置：**
- `internal/sysproxy/windows.go` - Windows实现
- `internal/sysproxy/linux.go` - Linux实现

**核心接口：**
```go
type SysProxy interface {
    GetStatus() (*ProxySettings, error)
    Enable(server, bypassList string) error
    Disable() error
    IsSupported() bool
}
```

### TUN模式

TUN模式通过创建虚拟网卡实现全局透明代理，支持更精细的流量控制。

**实现位置：**
- `listener/sing_tun/server.go` - 核心实现
- `listener/inbound/tun.go` - 入站配置

**关键配置参数：**
- `auto-route`: 自动配置路由表
- `auto-redirect`: 自动重定向流量
- `strict-route`: 严格路由模式
- `iproute2-table-index`: 路由表索引
- `iproute2-rule-index`: 路由规则索引

### TProxy透明代理

TProxy模式通过iptables实现透明代理，仅支持Linux平台。

**实现位置：**
- `listener/tproxy/tproxy_iptables.go` - iptables规则管理
- `listener/tproxy/tproxy.go` - 监听器实现

**关键常量：**
```go
const (
    PROXY_FWMARK      = "0x2d0"    // 防火墙标记
    PROXY_ROUTE_TABLE = "0x2d0"    // 路由表ID
)
```

---

## Windows平台配置修改

### 1. 注册表修改（标准系统代理）

**注册表路径：**
```
HKEY_CURRENT_USER\SOFTWARE\Microsoft\Windows\CurrentVersion\Internet Settings
```

**修改的注册表项：**

| 注册表项 | 类型 | 说明 | 示例值 |
|---------|------|------|--------|
| `ProxyEnable` | DWORD | 代理启用标志 | 1（启用）/ 0（禁用） |
| `ProxyServer` | SZ | 代理服务器地址 | `127.0.0.1:7890` |
| `ProxyOverride` | SZ | 绕过代理列表 | `localhost;127.*;10.*;172.16.*;172.31.*;192.168.*` |

**代码实现：**
```go
// 启用系统代理
func (sp *windowsSysProxy) Enable(server, bypassList string) error {
    wr, err := NewWindowsRegistry()
    if err != nil {
        return err
    }
    defer wr.Close()

    settings := &ProxySettings{
        Enabled:    true,
        Server:     server,
        BypassList: bypassList,
    }

    return wr.SetSettings(settings)
}
```

**影响范围：**
- Internet Explorer/Edge浏览器
- 使用WinHTTP/WinINet的应用程序
- 部分遵循系统代理设置的应用

### 2. TUN模式配置修改

**虚拟网卡创建：**
- 创建名为 `Meta`（默认）的TUN虚拟网卡
- 配置IPv4/IPv6地址（默认：`172.19.0.1/30`, `fdfe:dcba:9876::1/126`）
- 设置MTU（默认：9000）

**路由表修改：**
当启用 `auto-route` 时，会修改Windows路由表：

1. **添加路由规则：**
   - 将默认路由指向TUN网卡
   - 排除本地网络地址

2. **路由表索引：**
   - 默认使用 `tun.DefaultIPRoute2TableIndex`
   - 可通过 `iproute2-table-index` 配置

**Windows版本兼容性：**
```go
if features.WindowsMajorVersion < 10 {
    // Windows 7/8 需要强制绑定接口
    EnforceBindInterface = true
}
```

---

## Linux平台配置修改

### 1. 环境变量修改（标准系统代理）

**配置文件路径：**
- 优先：`/etc/environment.d/proxy.conf`（systemd环境）
- 备用：`/etc/environment`

**修改的环境变量：**

| 环境变量 | 说明 | 示例值 |
|---------|------|--------|
| `HTTP_PROXY` | HTTP代理地址 | `http://127.0.0.1:7890` |
| `HTTPS_PROXY` | HTTPS代理地址 | `http://127.0.0.1:7890` |
| `http_proxy` | HTTP代理（小写） | `http://127.0.0.1:7890` |
| `https_proxy` | HTTPS代理（小写） | `http://127.0.0.1:7890` |
| `NO_PROXY` | 绕过代理列表 | `localhost,127.0.0.1,10.*` |
| `no_proxy` | 绕过代理（小写） | `localhost,127.0.0.1,10.*` |

**代码实现：**
```go
func (sp *linuxSysProxy) Enable(server, bypassList string) error {
    content := fmt.Sprintf(
        "HTTP_PROXY=%s\n"+
            "HTTPS_PROXY=%s\n"+
            "http_proxy=%s\n"+
            "https_proxy=%s\n",
        server, server, server, server,
    )

    if bypassList != "" {
        content += fmt.Sprintf("NO_PROXY=%s\nno_proxy=%s\n", bypassList, bypassList)
    }

    // 尝试写入 systemd environment.d 目录
    if err := writeProxyConfig(ProxyEnvFile, content); err == nil {
        return nil
    }

    // 回退到 /etc/environment
    return writeProxyConfig(ProxyEnvFileFallback, content)
}
```

**影响范围：**
- 支持环境变量代理的Shell应用
- systemd用户会话
- 部分遵循环境变量的应用

### 2. TUN模式配置修改

**虚拟网卡创建：**
- 创建名为 `tun`（默认）的TUN虚拟网卡
- 配置IPv4/IPv6地址
- 设置MTU

**路由表修改：**
```go
tunOptions := tun.Options{
    Name:                    tunName,
    MTU:                     tunMTU,
    Inet4Address:            options.Inet4Address,
    Inet6Address:            options.Inet6Address,
    AutoRoute:               options.AutoRoute,
    IPRoute2TableIndex:      tableIndex,
    IPRoute2RuleIndex:       ruleIndex,
    StrictRoute:             options.StrictRoute,
    // ...
}
```

**自动重定向（auto-redirect）：**
```go
if options.AutoRedirect {
    l.autoRedirect, err = tun.NewAutoRedirect(tun.AutoRedirectOptions{
        TunOptions:             &tunOptions,
        Context:                ctx,
        Handler:                handler.TypeMutation(C.REDIR),
        Logger:                 log.SingLogger,
        NetworkMonitor:         l.networkUpdateMonitor,
        TableName:              "mihomo",
        // ...
    })
}
```

---

## 路由表修改详解

### TProxy模式路由表修改

**路由规则添加：**
```bash
# 添加策略路由规则
ip -f inet rule add fwmark 0x2d0 lookup 0x2d0

# 添加路由表条目
ip -f inet route add local default dev <interface> table 0x2d0
```

**路由表说明：**
- **fwmark**: `0x2d0` (720) - 防火墙标记，用于标识需要代理的流量
- **route table**: `0x2d0` (720) - 自定义路由表ID
- **local default**: 将所有流量路由到本地TProxy端口

### TUN模式路由表修改

**路由表索引配置：**
```go
tableIndex := options.IPRoute2TableIndex
if tableIndex == 0 {
    tableIndex = tun.DefaultIPRoute2TableIndex
}

ruleIndex := options.IPRoute2RuleIndex
if ruleIndex == 0 {
    ruleIndex = tun.DefaultIPRoute2RuleIndex
}
```

**路由地址控制：**
- `route-address`: 指定需要代理的IP地址范围
- `route-exclude-address`: 排除不需要代理的IP地址
- `route-address-set`: 使用规则集动态控制路由地址

---

## iptables规则详解

### TProxy模式iptables规则

#### 1. 路由策略规则（mangle表）

**mihomo_divert链：**
```bash
# 创建divert链
iptables -t mangle -N mihomo_divert
iptables -t mangle -F mihomo_divert

# 设置防火墙标记
iptables -t mangle -A mihomo_divert -j MARK --set-mark 0x2d0
iptables -t mangle -A mihomo_divert -j ACCEPT
```

**mihomo_prerouting链：**
```bash
# 创建prerouting链
iptables -t mangle -N mihomo_prerouting
iptables -t mangle -F mihomo_prerouting

# 排除Docker网络
iptables -t mangle -A mihomo_prerouting -s 172.17.0.0/16 -j RETURN

# DNS流量处理（如果启用DNS重定向）
iptables -t mangle -A mihomo_prerouting -p udp --dport 53 -j ACCEPT
iptables -t mangle -A mihomo_prerouting -p tcp --dport 53 -j ACCEPT

# 排除本地地址
iptables -t mangle -A mihomo_prerouting -m addrtype --dst-type LOCAL -j RETURN

# 排除私有网络地址
iptables -t mangle -A mihomo_prerouting -d 10.0.0.0/8 -j RETURN
iptables -t mangle -A mihomo_prerouting -d 172.16.0.0/12 -j RETURN
iptables -t mangle -A mihomo_prerouting -d 192.168.0.0/16 -j RETURN
# ... 更多私有地址段

# 已建立连接的socket重定向
iptables -t mangle -A mihomo_prerouting -p tcp -m socket -j mihomo_divert
iptables -t mangle -A mihomo_prerouting -p udp -m socket -j mihomo_divert

# TProxy重定向
iptables -t mangle -A mihomo_prerouting -p tcp -j TPROXY --on-port <port> --tproxy-mark 0x2d0/0x2d0
iptables -t mangle -A mihomo_prerouting -p udp -j TPROXY --on-port <port> --tproxy-mark 0x2d0/0x2d0

# 应用到PREROUTING链
iptables -t mangle -A PREROUTING -j mihomo_prerouting
```

**mihomo_output链：**
```bash
# 创建output链
iptables -t mangle -N mihomo_output
iptables -t mangle -F mihomo_output

# 排除已标记的流量
iptables -t mangle -A mihomo_output -m mark --mark <mark> -j RETURN

# 排除DNS和NTP流量（如果启用DNS重定向）
iptables -t mangle -A mihomo_output -p udp -m multiport --dports 53,123,137 -j ACCEPT
iptables -t mangle -A mihomo_output -p tcp --dport 53 -j ACCEPT

# 排除本地和广播地址
iptables -t mangle -A mihomo_output -m addrtype --dst-type LOCAL -j RETURN
iptables -t mangle -A mihomo_output -m addrtype --dst-type BROADCAST -j RETURN

# 标记出站流量
iptables -t mangle -A mihomo_output -p tcp -j MARK --set-mark 0x2d0
iptables -t mangle -A mihomo_output -p udp -j MARK --set-mark 0x2d0

# 应用到OUTPUT链
iptables -t mangle -I OUTPUT -o <interface> -j mihomo_output
```

#### 2. NAT规则（nat表）

**DNS重定向：**
```bash
# PREROUTING DNS重定向
iptables -t nat -I PREROUTING ! -s 172.17.0.0/16 ! -d 127.0.0.0/8 -p tcp --dport 53 -j REDIRECT --to <dns-port>
iptables -t nat -I PREROUTING ! -s 172.17.0.0/16 ! -d 127.0.0.0/8 -p udp --dport 53 -j REDIRECT --to <dns-port>

# OUTPUT DNS重定向
iptables -t nat -N mihomo_dns_output
iptables -t nat -F mihomo_dns_output
iptables -t nat -A mihomo_dns_output -p udp -j REDIRECT --to-ports <dns-port>
iptables -t nat -A mihomo_dns_output -p tcp -j REDIRECT --to-ports <dns-port>
iptables -t nat -I OUTPUT -p tcp --dport 53 -j mihomo_dns_output
iptables -t nat -I OUTPUT -p udp --dport 53 -j mihomo_dns_output
```

**NAT伪装：**
```bash
# POSTROUTING NAT伪装
iptables -t nat -A POSTROUTING -o <interface> -m addrtype ! --src-type LOCAL -j MASQUERADE
```

#### 3. 转发规则（filter表）

```bash
# 启用IP转发
sysctl -w net.ipv4.ip_forward=1

# FORWARD链规则
iptables -t filter -A FORWARD -o <interface> -m conntrack --ctstate RELATED,ESTABLISHED -j ACCEPT
iptables -t filter -A FORWARD -o <interface> -j ACCEPT
iptables -t filter -A FORWARD -i <interface> ! -o <interface> -j ACCEPT
iptables -t filter -A FORWARD -i <interface> -o <interface> -j ACCEPT
```

#### 4. 排除的私有网络地址

TProxy模式会自动排除以下私有网络地址：

| 地址范围 | 说明 |
|---------|------|
| `0.0.0.0/8` | 本网络地址 |
| `10.0.0.0/8` | 私有网络A类 |
| `100.64.0.0/10` | 运营商级NAT |
| `127.0.0.0/8` | 本地回环 |
| `169.254.0.0/16` | 链路本地 |
| `172.16.0.0/12` | 私有网络B类 |
| `192.0.0.0/24` | IETF协议分配 |
| `192.0.2.0/24` | 测试网络1 |
| `192.88.99.0/24` | IPv6到IPv4中继 |
| `192.168.0.0/16` | 私有网络C类 |
| `198.51.100.0/24` | 测试网络2 |
| `203.0.113.0/24` | 测试网络3 |
| `224.0.0.0/4` | 组播地址 |
| `240.0.0.0/4` | 保留地址 |
| `255.255.255.255/32` | 广播地址 |

---

## 配置恢复机制

### TProxy模式清理

**清理函数：** `CleanupTProxyIPTables()`

**清理步骤：**

1. **删除路由规则：**
```bash
ip -f inet rule del fwmark 0x2d0 lookup 0x2d0
ip -f inet route del local default dev <interface> table 0x2d0
```

2. **删除iptables规则：**
```bash
# 删除PREROUTING规则
iptables -t mangle -D PREROUTING -j mihomo_prerouting
iptables -t nat -D PREROUTING <dns-redirect-rules>

# 删除OUTPUT规则
iptables -t mangle -D OUTPUT -o <interface> -j mihomo_output
iptables -t nat -D OUTPUT <dns-output-rules>

# 删除POSTROUTING规则
iptables -t nat -D POSTROUTING -o <interface> -m addrtype ! --src-type LOCAL -j MASQUERADE

# 删除FORWARD规则
iptables -t filter -D FORWARD <forward-rules>

# 清空并删除自定义链
iptables -t mangle -F mihomo_prerouting
iptables -t mangle -X mihomo_prerouting
iptables -t mangle -F mihomo_divert
iptables -t mangle -X mihomo_divert
iptables -t mangle -F mihomo_output
iptables -t mangle -X mihomo_output
iptables -t nat -F mihomo_dns_output
iptables -t nat -X mihomo_dns_output
```

### TUN模式清理

**清理函数：** `Listener.Close()`

**清理步骤：**

1. **关闭TUN栈：**
```go
return common.Close(
    l.ruleUpdateCallbackCloser,
    l.tunStack,
    l.tunIf,
    l.autoRedirect,
    l.defaultInterfaceMonitor,
    l.networkUpdateMonitor,
    l.packageManager,
)
```

2. **恢复路由标记：**
```go
if l.autoRedirectOutputMark != 0 {
    dialer.DefaultRoutingMark.CompareAndSwap(l.autoRedirectOutputMark, 0)
}
```

3. **恢复接口查找器：**
```go
if l.cDialerInterfaceFinder != nil {
    dialer.DefaultInterfaceFinder.CompareAndSwap(l.cDialerInterfaceFinder, nil)
}
```

### 标准系统代理恢复

**Windows：**
```go
func (sp *windowsSysProxy) Disable() error {
    wr, err := NewWindowsRegistry()
    if err != nil {
        return err
    }
    defer wr.Close()

    settings := &ProxySettings{
        Enabled: false,
    }

    return wr.SetSettings(settings)
}
```

**Linux：**
```go
func (sp *linuxSysProxy) Disable() error {
    // 删除 systemd environment.d 配置
    if err := removeProxyConfig(ProxyEnvFile); err != nil {
        return err
    }

    // 从 /etc/environment 中移除代理配置
    if err := removeFromEtcEnvironment(); err != nil {
        return pkgerrors.ErrService("failed to remove proxy from /etc/environment", err)
    }

    return nil
}
```

---

## 总结

Mihomo内核通过多种方式实现系统代理，每种方式对系统配置的修改程度不同：

1. **标准系统代理**：修改注册表或环境变量，影响范围有限，易于恢复
2. **TUN模式**：创建虚拟网卡并修改路由表，实现全局透明代理
3. **TProxy模式**：通过iptables和路由策略实现透明代理，仅支持Linux

所有模式都提供了完善的配置恢复机制，确保在程序退出时能够正确清理系统配置。
