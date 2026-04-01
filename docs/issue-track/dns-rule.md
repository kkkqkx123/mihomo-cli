# Mihomo DNS 问题排查与解决手册

## 一、问题现象

### 典型症状

```bash
curl.exe -x http://127.0.0.1:7890 -v http://google.com
# 返回: HTTP/1.1 502 Bad Gateway
```

### 代理状态检查

```bash
# 端口监听正常
netstat -ano | findstr :7890
# 显示 LISTENING 和 ESTABLISHED 连接

# 代理连接正常
Test-NetConnection -ComputerName 127.0.0.1 -Port 7890
# TcpTestSucceeded : True
```

**关键判断**：代理服务运行正常，但无法代理请求 → **DNS 解析问题**

---

## 二、根本原因分析

### 1. **DNS 循环依赖死锁**

```
Mihomo 启动
    ↓
需要解析代理节点域名
    ↓
使用 DoH (1.1.1.1:443) 解析
    ↓
DoH 需要 TLS 连接到 1.1.1.1:443
    ↓
连接被规则拦截需要走代理
    ↓
代理需要解析节点域名 ← 死循环
```

### 2. **网络环境限制**

- 国外 IP 的 443 端口（DoH）被阻断
- 国外 IP 的 UDP 53 端口相对宽松
- 国内 DoH 服务可正常访问

### 3. **配置缺陷**

- fallback DNS 全是不可访问的国外 DoH
- nameserver 使用国外 DNS 导致污染
- 没有域名白名单区分国内外流量

---

## 三、诊断步骤

### Step 1: 确认代理服务状态

```powershell
# 检查端口监听
netstat -ano | findstr :7890

# 检查进程
tasklist | findstr <PID>

# 测试连接
Test-NetConnection -ComputerName 127.0.0.1 -Port 7890
```

### Step 2: 验证 DNS 可访问性

```powershell
# 测试国内 DoH
curl.exe -v https://doh.pub/dns-query

# 测试国外 DoH
curl.exe -v https://1.1.1.1/dns-query

# 测试国外 UDP DNS
Test-NetConnection -ComputerName 8.8.8.8 -Port 53

# 测试国外 TCP DNS
Test-NetConnection -ComputerName 8.8.8.8 -Port 853
```

### Step 3: 检查 DNS 配置

```powershell
# 查看 Mihomo DNS 配置
curl.exe -X GET http://127.0.0.1:9090/configs/dns

# 查看 DNS 缓存
curl.exe -X GET http://127.0.0.1:9090/dns/cache

# 测试 DNS 解析
nslookup google.com 127.0.0.1:1053
```

### Step 4: 查看详细日志

```yaml
# 临时修改配置
log-level: debug
```

```powershell
# 观察 DNS 查询日志
# 应该看到类似输出：
# [DEBUG] [DNS] Lookup for google.com using nameserver...
# [ERROR] [DNS] Failed to connect to 1.1.1.1:443
```

---

## 四、解决方案

### 方案 A：最小改动（快速修复）

```yaml
dns:
  enable: true
  enhanced-mode: fake-ip
  nameserver:
    - 223.5.5.5 # 阿里 DNS
    - 119.29.29.29 # 腾讯 DNS
    - https://doh.pub/dns-query # 国内 DoH
  fallback:
    - 8.8.8.8 # 国外 UDP DNS
    - 1.1.1.1 # 国外 UDP DNS
  # 移除所有 DoH fallback
```

### 方案 B：优化配置（推荐）

```yaml
dns:
  enable: true
  enhanced-mode: fake-ip
  fake-ip-range: 198.18.0.1/16

  # 分层 DNS 策略
  nameserver:
    - 223.5.5.5 # 国内 UDP
    - 119.29.29.29 # 国内 UDP
    - https://doh.pub/dns-query # 国内 DoH

  fallback:
    - 8.8.8.8 # 国外 UDP
    - 1.1.1.1 # 国外 UDP
    - tls://8.8.8.8:853 # DNS-over-TLS

  # 智能 fallback 策略
  fallback-filter:
    geoip: true
    geoip-code: CN
    domain:
      - "geosite:google"
      - "geosite:github"
      - "geosite:openai"
      - "geosite:cloudflare"
    ipcidr:
      - 240.0.0.0/4
      - 0.0.0.0/32

# 添加 DoH 域名直连规则
rules:
  - DOMAIN,doh.pub,DIRECT
  - DOMAIN,dns.alidns.com,DIRECT
  - DOMAIN,cloudflare-dns.com,DIRECT
  - DOMAIN-SUFFIX,google.com,PROXY
  - GEOIP,CN,DIRECT
  - MATCH,PROXY
```

### 方案 C：Hosts 预处理（最可靠）

```yaml
# 预先解析 DoH 服务器
hosts:
  "doh.pub": "1.12.12.12"
  "dns.alidns.com": "223.5.5.5"
  "cloudflare-dns.com": "1.1.1.1"
  "dns.google": "8.8.8.8"

dns:
  enable: true
  nameserver:
    - https://doh.pub/dns-query
    - https://dns.alidns.com/dns-query
  fallback:
    - https://cloudflare-dns.com/dns-query
    - https://dns.google/dns-query
```

### 方案 D：独立 DNS 服务（最佳实践）

```yaml
# Mihomo 配置
dns:
  enable: true
  nameserver:
    - 127.0.0.1:5353 # 指向本地 DNS 服务
```

配合 AdGuardHome 或 mosdns 独立运行。

---

## 五、配置对比表

| 配置项          | ❌ 错误配置               | ✅ 正确配置             |
| --------------- | ------------------------- | ----------------------- |
| nameserver      | 8.8.8.8                   | 223.5.5.5, 119.29.29.29 |
| fallback        | https://1.1.1.1/dns-query | 8.8.8.8, 1.1.1.1        |
| fallback-filter | 空                        | geoip + domain 白名单   |
| DoH 使用        | 国外 DoH 为主             | 国内 DoH + 国外 UDP     |
| 协议支持        | 仅 DoH                    | UDP + DoH + DoT         |

---

## 六、验证方法

### 1. 测试 DNS 解析

```powershell
# 通过代理测试
curl.exe -x http://127.0.0.1:7890 -v http://httpbin.org/ip

# 预期返回
{
  "origin": "代理服务器IP"
}
```

### 2. 测试不同域名

```powershell
# 国内域名（应直连）
curl.exe -x http://127.0.0.1:7890 -v http://www.baidu.com

# 国外域名（应代理）
curl.exe -x http://127.0.0.1:7890 -v http://google.com
```

### 3. 检查 DNS 查询路径

```powershell
# 启用 debug 日志后查看
# 应该看到 DNS 查询使用国内服务器
[DEBUG] [DNS] google.com -> using nameserver 223.5.5.5
[DEBUG] [DNS] google.com -> fallback to 8.8.8.8
```

---

## 七、常见错误码速查

| 错误码              | 含义         | 解决方法               |
| ------------------- | ------------ | ---------------------- |
| 502 Bad Gateway     | DNS 解析失败 | 检查 fallback DNS 配置 |
| 504 Gateway Timeout | DNS 超时     | 更换可用的 DNS 服务器  |
| 407 Proxy Auth      | 代理认证失败 | 检查代理配置           |
| Connection Refused  | 代理未运行   | 启动 Mihomo            |
| No route to host    | 网络不通     | 检查防火墙设置         |

---

## 八、快速排查清单

1. 代理端口是否监听？
   netstat -ano | findstr :7890

2. 代理进程是否运行？
   tasklist | findstr mihomo

3. 能否连接代理？
   Test-NetConnection 127.0.0.1 -Port 7890

4. DNS 服务器可访问？
   Test-NetConnection 8.8.8.8 -Port 53

5. 国内 DNS 可访问？
   curl https://doh.pub/dns-query

6. 配置文件 DNS 段正确？
   检查 nameserver 和 fallback

7. fallback-filter 配置？
   确认有域名白名单

8. 规则是否包含 DNS 域名？
   添加 DOMAIN,doh.pub,DIRECT

9. 日志级别是否足够？
   log-level: debug

10. 是否看到 DNS 查询日志？
    确认 DNS 查询路径

---

## 九、核心要点总结

1. **DNS 是代理的基础**：没有 DNS 就无法代理
2. **避免循环依赖**：DNS 查询不应依赖代理
3. **分层 DNS 策略**：国内用国内 DNS，国外用国外 DNS
4. **协议多样性**：UDP 53 比 DoH 更可靠
5. **冗余设计**：至少 2-3 个备用 DNS 服务器
6. **域名白名单**：明确哪些域名需要国外 DNS
7. **直连 DNS 服务器**：DNS 查询域名必须直连

---

## 十、相关资源

- Mihomo 文档: https://wiki.metacubex.one/
- DNS 测试工具: https://dnschecker.org/
- 国内 DoH 服务:
  - 腾讯: https://doh.pub/dns-query
  - 阿里: https://dns.alidns.com/dns-query
  - 360: https://doh.360.cn/dns-query

---

**记住这个原则**：DNS 解析必须独立于代理规则，否则就会陷入"先有鸡还是先有蛋"的死循环。
