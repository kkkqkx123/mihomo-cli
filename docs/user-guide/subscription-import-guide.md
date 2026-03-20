# Mihomo 订阅导入操作指南

本文档详细说明如何使用 mihomo-cli 工具导入代理订阅并持久化保存配置。

## 一、环境准备

### 1. 构建 CLI 工具

```bash
go build -o mihomo-cli.exe .
```

### 2. 准备 Mihomo 配置文件

创建 Mihomo 配置文件（例如 `test-mihomo-config.yaml`），包含以下关键配置：

```yaml
# 基础配置
mixed-port: 7890
allow-lan: true
mode: rule
log-level: info
external-controller: 127.0.0.1:9090

# DNS 配置
dns:
  enable: true
  ipv6: false
  enhanced-mode: fake-ip
  fake-ip-range: 198.18.0.1/16
  nameserver:
    - 223.5.5.5
    - 119.29.29.29

# 代理提供者配置
proxy-providers:
  my-subscription:
    type: http
    url: "https://your-subscription-url"
    interval: 3600
    path: ./profiles/my-subscription.yaml
    proxy: DIRECT
    header:
      User-Agent:
        - "Clash/v1.18.0"
    health-check:
      enable: true
      interval: 600
      url: https://cp.cloudflare.com/generate_204

# 代理组
proxy-groups:
  - name: PROXY
    type: select
    use:
      - my-subscription

  - name: AUTO
    type: url-test
    use:
      - my-subscription
    url: http://www.gstatic.com/generate_204
    interval: 300

# 规则
rules:
  - MATCH,PROXY
```

### 3. 配置 config.toml

编辑 `config.toml` 文件，指定 Mihomo 可执行文件和配置文件路径：

```toml
[api]
address = "http://127.0.0.1:9090"
secret = ""
timeout = 10

[mihomo]
enabled = true
executable = "E:\\server\\mihomo\\mihomo-windows-amd64.exe"
config_file = "E:\\project\\mihomo-go\\test-mihomo-config.yaml"
auto_generate_secret = true

[mihomo.api]
external_controller = "127.0.0.1:9090"

[mihomo.log]
level = "info"
```

## 二、启动 Mihomo 内核

### 方法一：使用 CLI 工具启动

```bash
./mihomo-cli.exe start
```

输出示例：

```
=====================================
  Mihomo 内核已启动
=====================================
API 地址: http://127.0.0.1:9090
密钥: dc544eb8b12ddc483a45ad726b2cae7e632f82f864004c5a1f8b474d60ebc098

提示: 内核已在后台运行
使用以下命令管理：
  mihomo-cli status  - 查询运行状态
  mihomo-cli stop    - 停止内核
=====================================
```

### 方法二：直接启动 Mihomo

```bash
E:\server\mihomo\mihomo-windows-amd64.exe -f E:\project\mihomo-go\test-mihomo-config.yaml
```

## 三、配置 CLI 工具

### 1. 初始化配置

```bash
./mihomo-cli.exe config init --force
```

### 2. 设置 API 密钥

使用启动时生成的密钥：

```bash
./mihomo-cli.exe config set api.secret 2b9075640e8205fb2e7b267f94d459927668c460c5b8d3e01695dbec47b410a1
```

### 3. 验证配置

```bash
./mihomo-cli.exe config show
```

## 四、导入订阅

### 1. 更新订阅

```bash
./mihomo-cli.exe sub update
```

输出示例：

```
找到 2 个代理提供者，开始更新...

正在更新 default (Compatible)...
  ✓ 更新成功
正在更新 my-subscription (HTTP)...
  ✓ 更新成功

更新完成: 成功 2 个，失败 0 个
```

### 2. 查看导入的代理节点

```bash
./mihomo-cli.exe proxy list
```

输出示例：

```
┌─────────────────────────┬────────────┬─────────────────────┬────────┬──────┬──────┐
│          名称           │    类型    │        当前         │ 节点数 │ 延迟 │ 状态 │
├─────────────────────────┼────────────┼─────────────────────┼────────┼──────┼──────┤
│ 剩余流量：96.31 GB      │ Vmess      │ -                   │ -      │ -    │ ✓    │
│ 套餐到期：长期有效      │ Vmess      │ -                   │ -      │ -    │ ✓    │
│ 日本-优化               │ Vmess      │ -                   │ -      │ -    │ ✓    │
│ 新加坡-优化-Gemini-GPT  │ Vmess      │ -                   │ -      │ -    │ ✓    │
│ 香港-优化-Gemini        │ Vmess      │ -                   │ -      │ -    │ ✓    │
│ ...                     │ ...        │ ...                 │ ...    │ ...  │ ...  │
└─────────────────────────┴────────────┴─────────────────────┴────────┴──────┴──────┘
```

## 五、验证持久化保存

### 1. 检查订阅文件

订阅文件保存在 Mihomo 配置目录下：

**Windows:**

```
C:\Users\<用户名>\.config\mihomo\profiles\my-subscription.yaml
```

**Linux/macOS:**

```
~/.config/mihomo/profiles/my-subscription.yaml
```

### 2. 查看订阅文件内容

```bash
cat C:\Users\33530\.config\mihomo\profiles\my-subscription.yaml
```

文件内容示例：

```yaml
proxies:
  - { name: '剩余流量：96.31 GB', type: vmess, server: planb.mojcn.com, port: 16617, uuid: ff3b259a-e082-408f-b862-e5f6836f0da3, alterId: 0, cipher: auto, udp: true, network: ws, ws-opts: { path: /, headers: { Host: 4591f7a1d707444c7f68a38baba091ce.mobgslb.tbcache.com } } }
  - { name: 套餐到期：长期有效, type: vmess, server: planb.mojcn.com, port: 16617, uuid: ff3b259a-e082-408f-b862-e5f6836f0da3, alterId: 0, cipher: auto, udp: true, network: ws, ws-opts: { path: /, headers: { Host: 4591f7a1d707444c7f68a38baba091ce.mobgslb.tbcache.com } } }
  - { name: 日本-优化, type: vmess, server: planb.mojcn.com, port: 16617, ... }
  ...
```

### 3. 检查缓存数据库

Mihomo 会将订阅信息缓存到数据库文件：

**Windows:**

```
C:\Users\<用户名>\.config\mihomo\cache.db
```

**Linux/macOS:**

```
~/.config/mihomo/cache.db
```

## 六、订阅处理流程详解

### 1. 订阅获取流程

```
HTTPVehicle.Read()
    ↓
发送 HTTP GET 请求到订阅 URL
    ↓
提取 subscription-userinfo 响应头（流量信息）
    ↓
读取响应内容（Base64 编码的订阅数据）
    ↓
返回原始字节数据
```

### 2. Base64 解码流程

```
DecodeBase64(buf)
    ↓
尝试 RawStdEncoding 解码
    ↓
失败则尝试 StdEncoding 解码
    ↓
失败则返回原始内容
    ↓
得到第一次解码后的内容（vmess:// 链接列表）
```

### 3. V2Ray 格式转换流程

```
ConvertsV2Ray(buf)
    ↓
按行分割内容
    ↓
对每行：
    1. 提取协议类型（vmess://, vless://, ss:// 等）
    2. 提取 Base64 编码的配置部分
    3. 第二次 Base64 解码得到 JSON 配置
    4. 解析 JSON 并转换为 mihomo 格式
    ↓
返回代理配置列表
```

### 4. 代理解析流程

```
ParseProxy(mapping)
    ↓
根据 type 字段选择解析器
    ↓
解码配置参数
    ↓
创建对应的代理适配器（Vmess, Vless, Shadowsocks 等）
    ↓
返回代理对象
```

### 5. 持久化保存流程

```
Fetcher.loadBuf()
    ↓
调用 parser 解析订阅数据
    ↓
调用 vehicle.Write() 保存到文件
    ↓
更新内存中的代理列表
    ↓
触发 onUpdate 回调
```

## 七、常见问题排查

### 1. 订阅更新失败

**问题：** 无法连接到订阅 URL

**解决：**

- 检查网络连接
- 检查订阅 URL 是否正确
- 检查是否需要代理访问订阅 URL（在 proxy-providers 中设置 `proxy` 字段）

### 2. 配置文件测试失败

**问题：** GeoIP 数据库下载失败

**解决：**

- 简化 DNS 配置，移除 `fallback-filter` 中的 `geoip` 字段
- 或手动下载 GeoIP 数据库文件

### 3. API 连接失败

**问题：** CLI 无法连接到 Mihomo API

**解决：**

- 确认 Mihomo 内核已启动
- 检查 API 地址和端口是否正确
- 检查 API 密钥是否正确

### 4. 订阅格式不支持

**问题：** 订阅内容无法解析

**解决：**

- 确认订阅格式为支持的类型（YAML、V2Ray 分享链接、Base64 编码）
- 检查订阅内容是否完整

## 八、高级配置

### 1. 使用过滤器

```yaml
proxy-providers:
  my-subscription:
    type: http
    url: "https://your-subscription-url"
    filter: "日本|香港" # 只保留包含"日本"或"香港"的节点
    exclude-filter: "过期" # 排除包含"过期"的节点
    exclude-type: "ssr" # 排除 SSR 类型节点
```

### 2. 使用覆写配置

```yaml
proxy-providers:
  my-subscription:
    type: http
    url: "https://your-subscription-url"
    override:
      skip-cert-verify: true
      udp: true
      additional-prefix: "[MyProvider] "
```

### 3. 自定义请求头

```yaml
proxy-providers:
  my-subscription:
    type: http
    url: "https://your-subscription-url"
    header:
      User-Agent:
        - "Clash/v1.18.0"
      Accept:
        - "text/html,application/xhtml+xml,application/xml;q=0.9"
```

### 4. 设置文件大小限制

```yaml
proxy-providers:
  my-subscription:
    type: http
    url: "https://your-subscription-url"
    size-limit: 10240 # 限制 10KB
```

## 九、总结

通过 mihomo-cli 工具导入订阅的完整流程：

1. **准备配置文件**：创建包含 proxy-providers 的 Mihomo 配置文件
2. **启动 Mihomo 内核**：使用 CLI 工具或直接启动 Mihomo
3. **配置 CLI 工具**：初始化配置并设置 API 密钥
4. **导入订阅**：使用 `sub update` 命令更新订阅
5. **验证结果**：查看代理列表和持久化文件

订阅数据会自动持久化保存到：

- **订阅文件**：`~/.config/mihomo/profiles/<provider-name>.yaml`
- **缓存数据库**：`~/.config/mihomo/cache.db`

Mihomo 会自动处理：

- Base64 解码（支持多次解码）
- V2Ray 格式转换
- 多协议支持（vmess, vless, ss, ssr, trojan, hysteria 等）
- 订阅信息提取（流量、过期时间）
- 自动更新（按配置的 interval 定期更新）
- 健康检查（定期检查节点可用性）
