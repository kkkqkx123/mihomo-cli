# Mihomo 订阅格式处理机制

## 概述

Mihomo 内核提供了完整的订阅格式处理机制，支持多种代理订阅格式和规则订阅格式。本文档详细分析了订阅格式处理的核心实现、支持的格式类型、处理流程以及如何使用 Mihomo API 完成订阅获取。

## 核心架构

Mihomo 采用分层架构处理订阅格式：

```
配置文件 (config.yaml)
    ↓
config.ParseConfig()
    ↓
parseProxies() / parseRuleProviders()
    ↓
provider.ParseProxyProvider() / RP.ParseRuleProvider()
    ↓
ProxySetProvider / RuleSetProvider
    ↓
resource.Fetcher
    ↓
Vehicle (FileVehicle / HTTPVehicle)
    ↓
Parser (NewProxiesParser / rulesParse)
    ↓
convert.ConvertsV2Ray() [V2Ray 格式转换]
    ↓
adapter.ParseProxy() / rules.ParseRule()
    ↓
最终代理/规则对象
```

## 支持的订阅格式

### 代理订阅格式

#### 1. YAML 格式

标准的 YAML 配置格式：

```yaml
proxies:
  - name: "proxy-name"
    type: ss
    server: example.com
    port: 8388
    cipher: aes-256-gcm
    password: "password"
```

#### 2. V2Ray 分享链接格式

支持多种协议的分享链接：

| 协议 | 链接格式 | 说明 |
|------|---------|------|
| VMess | `vmess://` | 支持 V2RayN 和 Xray VMessAEAD 风格 |
| VLESS | `vless://` | VLESS 协议 |
| Shadowsocks | `ss://` | Shadowsocks 协议 |
| ShadowsocksR | `ssr://` | ShadowsocksR 协议 |
| Trojan | `trojan://` | Trojan 协议 |
| Hysteria | `hysteria://` | Hysteria 协议 |
| Hysteria2 | `hysteria2://` 或 `hy2://` | Hysteria2 协议 |
| TUIC | `tuic://` | TUIC 协议 |
| SOCKS | `socks://`, `socks5://`, `socks5h://` | SOCKS 代理 |
| HTTP | `http://`, `https://` | HTTP 代理 |
| AnyTLS | `anytls://` | AnyTLS 协议 |

**示例：**
```
vmess://eyJhZGUiOiIiLCJhaWQiOiIwIiwiYWxwbiI6IiIsImhvc3QiOiIiLCJpZCI6IjEyMzQ1Njc4LTkwYWItY2RlZi0xMjM0LTU2Nzg5MGFiY2RlZiIsIm5ldCI6IndzIiwicGF0aCI6Ii8iLCJwb3J0IjoiNDQzIiwicHMiOiJUZXN0Iiwic2N5IjoiYXV0byIsInNuaiI6IiIsInRscyI6IiIsInR5cGUiOiIiLCJ2IjoiMiJ9
```

#### 3. Base64 编码格式

自动检测并解码 Base64 编码的内容，支持：
- `base64.StdEncoding`
- `base64.URLEncoding`
- `base64.RawStdEncoding`
- `base64.RawURLEncoding`

### 规则订阅格式

#### 1. YAML 格式 (YamlRule)

```yaml
payload:
  - domain1.com
  - domain2.com
```

#### 2. 文本格式 (TextRule)

```
domain1.com
domain2.com
```

#### 3. MRS 格式 (MrsRule)

二进制格式，仅支持 domain 和 ipcidr 行为，可与 YAML/Text 格式互相转换。

## 核心代码位置

### 配置解析层

| 文件路径 | 函数/结构 | 说明 |
|---------|----------|------|
| `config/config.go:855` | `parseProxies()` | 解析代理和提供者 |
| `config/config.go:989` | `parseRuleProviders()` | 解析规则提供者 |

### 代理提供者实现

| 文件路径 | 说明 |
|---------|------|
| `adapter/provider/parser.go` | 提供者解析器 |
| `adapter/provider/provider.go` | 提供者核心实现 |
| `adapter/provider/subscription_info.go` | 订阅信息处理 |

### 数据获取层

| 文件路径 | 说明 |
|---------|------|
| `component/resource/vehicle.go` | 文件和 HTTP 数据获取 |
| `component/resource/fetcher.go` | 自动更新和缓存管理 |

### 格式转换层

| 文件路径 | 说明 |
|---------|------|
| `common/convert/converter.go` | V2Ray 等格式转换 |
| `common/convert/base64.go` | Base64 编解码 |

### 健康检查

| 文件路径 | 说明 |
|---------|------|
| `adapter/provider/healthcheck.go` | 节点健康检查 |

## 核心数据结构

### Vehicle 类型

```go
type VehicleType int

const (
    File VehicleType = iota      // 本地文件
    HTTP                         // HTTP 远程订阅
    Compatible                   // 兼容模式
    Inline                       // 内联定义
)
```

### Provider 类型

```go
type ProviderType int

const (
    Proxy ProviderType = iota    // 代理提供者
    Rule                         // 规则提供者
)
```

### 规则行为

```go
type RuleBehavior int

const (
    Domain RuleBehavior = iota   // 域名规则
    IPCIDR                       // IP CIDR 规则
    Classical                    // 经典规则
)
```

### 规则格式

```go
type RuleFormat int

const (
    YamlRule RuleFormat = iota   // YAML 格式
    TextRule                     // 文本格式
    MrsRule                      // MRS 二进制格式
)
```

### ProxyProvider 配置结构

```go
type proxyProviderSchema struct {
    Type          string           // "file", "http", "inline"
    Path          string           // 文件路径
    URL           string           // HTTP 订阅链接
    Proxy         string           // 下载订阅时使用的代理
    Interval      int              // 更新间隔（秒）
    Filter        string           // 节点名称过滤器（正则）
    ExcludeFilter string           // 排除过滤器（正则）
    ExcludeType   string           // 排除的节点类型
    DialerProxy   string           // 拨号代理
    SizeLimit     int64            // 文件大小限制
    Payload       []map[string]any // 内联代理配置
    HealthCheck   healthCheckSchema
    Override      overrideSchema
    Header        map[string][]string // HTTP 请求头
}
```

## 订阅链接加载和更新机制

### 自动更新流程

#### 1. 初始化阶段 (Fetcher.Initial())

- 检查本地缓存文件是否存在
- 如果存在，先使用本地缓存
- 启动后台更新循环

#### 2. 自动更新循环 (Fetcher.pullLoop())

```go
// 核心逻辑
for {
    time.Sleep(interval)  // 等待配置的间隔时间
    Update()              // 执行更新
}
```

#### 3. 文件监视 (仅 FileVehicle)

- 使用 `fswatch` 监视文件变化
- 文件修改时自动触发更新

#### 4. HTTP 更新机制

- 支持 ETag 缓存（避免重复下载）
- 支持断点续传和大小限制
- 支持代理下载订阅
- 提取 `subscription-userinfo` 响应头

#### 5. 退避重试机制 (slowdown.Backoff)

- 更新失败时使用指数退避
- 最小退避时间 10 秒
- 最大退避时间为配置的 interval

### 配置示例

```yaml
proxy-providers:
  provider1:
    type: http
    url: "https://example.com/subscription"
    interval: 3600        # 每小时更新一次
    path: ./provider1.yaml
    proxy: DIRECT         # 使用 DIRECT 代理下载
    header:
      User-Agent:
        - "Clash/v1.18.0"
    size-limit: 10240     # 限制 10KB
    health-check:
      enable: true
      interval: 600
      url: https://cp.cloudflare.com/generate_204
```

## 订阅信息处理

### 订阅信息结构

```go
type SubscriptionInfo struct {
    Upload   int64  // 上传流量
    Download int64  // 下载流量
    Total    int64  // 总流量
    Expire   int64  // 过期时间戳
}
```

### 订阅信息提取

- 从 HTTP 响应头 `subscription-userinfo` 提取
- 格式：`upload=123;download=456;total=1000;expire=1234567890`
- 自动解析并存储到缓存文件

## 格式转换和验证逻辑

### V2Ray 格式转换核心代码

```go
// common/convert/converter.go:17
func ConvertsV2Ray(buf []byte) ([]map[string]any, error) {
    data := DecodeBase64(buf)  // 自动解码 Base64
    arr := strings.Split(string(data), "\n")
    
    for _, line := range arr {
        scheme, body, found := strings.Cut(line, "://")
        switch strings.ToLower(scheme) {
        case "vmess":
            // 解析 VMess 格式
        case "vless":
            // 解析 VLESS 格式
        case "ss":
            // 解析 Shadowsocks 格式
        case "trojan":
            // 解析 Trojan 格式
        // ... 其他协议
        }
    }
}
```

### 代理解析器

```go
// adapter/provider/provider.go:267
func NewProxiesParser(...) (resource.Parser[[]C.Proxy], error) {
    return func(buf []byte) ([]C.Proxy, error) {
        schema := &ProxySchema{}
        
        // 尝试解析 YAML
        if err := yaml.Unmarshal(buf, schema); err != nil {
            // 失败则尝试 V2Ray 格式转换
            proxies, err1 := convert.ConvertsV2Ray(buf)
            if err1 != nil {
                return nil, fmt.Errorf("%w, %w", err, err1)
            }
            schema.Proxies = proxies
        }
        
        // 应用过滤器、排除器、覆写配置
        // 解析每个代理配置
    }, nil
}
```

### 过滤器机制

- 支持正则表达式过滤节点名称
- 支持排除特定类型节点
- 支持名称正则替换
- 支持添加前缀/后缀

## 健康检查机制

### 健康检查配置

```yaml
health-check:
  enable: true
  url: https://cp.cloudflare.com/generate_204
  interval: 600
  timeout: 5000
  lazy: true
  expected-status: 204
```

### 健康检查实现

```go
// adapter/provider/healthcheck.go:58
func (hc *HealthCheck) check() {
    b := &errgroup.Group{}
    b.SetLimit(10)  // 并发限制
    
    // 执行默认健康检查
    hc.execute(b, hc.url, id, option)
    
    // 执行额外的健康检查（来自代理组）
    for url, option := range hc.extra {
        hc.execute(b, url, id, option)
    }
    
    b.Wait()
}
```

### 特性

- 支持多个测试 URL
- 支持期望状态码验证
- 支持正则表达式过滤
- 支持懒加载模式
- 并发限制（最多 10 个同时检查）

## 覆写配置

### Override 结构

```go
type overrideSchema struct {
    TFO            *bool
    MPTcp          *bool
    UDP            *bool
    UDPOverTCP     *bool
    Up             *string
    Down           *string
    DialerProxy    *string
    SkipCertVerify *bool
    Interface      *string
    RoutingMark    *int
    IPVersion      *string
    AdditionalPrefix *string
    AdditionalSuffix *string
    ProxyName      []overrideProxyNameSchema  // 正则替换节点名称
}
```

### 配置示例

```yaml
override:
  skip-cert-verify: true
  udp: true
  additional-prefix: "[provider1]"
  proxy-name:
    - pattern: "test"
      target: "TEST"
```

## 使用 Mihomo 内核完成订阅获取

### 方案概述

使用 Mihomo 内核的 RESTful API 来完成订阅获取，可以实现以下功能：

1. **代理提供者管理**
   - 获取所有代理提供者信息
   - 获取指定代理提供者的详细信息
   - 更新代理提供者的代理列表
   - 对代理提供者中的代理进行健康检查

2. **规则提供者管理**
   - 获取所有规则提供者信息
   - 更新规则提供者的规则列表

### 前置条件

1. 配置 Mihomo 内核的 external-controller
2. 在配置文件中定义 proxy-providers 或 rule-providers
3. 启动 Mihomo 内核

### 配置示例

```yaml
# Mihomo 配置文件
external-controller: 127.0.0.1:9090
secret: your-secret

proxy-providers:
  my-provider:
    type: http
    url: "https://example.com/subscription"
    interval: 3600
    path: ./my-provider.yaml
    health-check:
      enable: true
      interval: 600
      url: https://cp.cloudflare.com/generate_204

rule-providers:
  my-rules:
    type: http
    url: "https://example.com/rules.yaml"
    interval: 86400
    path: ./my-rules.yaml
```

### API 使用示例

#### 1. 获取所有代理提供者

```bash
curl -H "Authorization: Bearer your-secret" \
  http://127.0.0.1:9090/providers/proxies
```

**响应示例：**
```json
{
  "providers": {
    "my-provider": {
      "name": "my-provider",
      "type": "file",
      "vehicleType": "HTTP",
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
  }
}
```

#### 2. 获取指定代理提供者

```bash
curl -H "Authorization: Bearer your-secret" \
  http://127.0.0.1:9090/providers/proxies/my-provider
```

#### 3. 更新代理提供者

```bash
curl -X PUT \
  -H "Authorization: Bearer your-secret" \
  http://127.0.0.1:9090/providers/proxies/my-provider
```

**说明：** 此操作会触发 Mihomo 内核从配置的 URL 下载最新的订阅内容，并更新代理列表。

#### 4. 代理提供者健康检查

```bash
curl -X GET \
  -H "Authorization: Bearer your-secret" \
  http://127.0.0.1:9090/providers/proxies/my-provider/healthcheck
```

**说明：** 此操作会对代理提供者中的所有代理进行延迟测试。

#### 5. 获取所有规则提供者

```bash
curl -H "Authorization: Bearer your-secret" \
  http://127.0.0.1:9090/providers/rules
```

#### 6. 更新规则提供者

```bash
curl -X PUT \
  -H "Authorization: Bearer your-secret" \
  http://127.0.0.1:9090/providers/rules/my-rules
```

### 实现步骤

#### 步骤 1：配置 Mihomo 内核

1. 创建配置文件 `config.yaml`，定义代理提供者
2. 启动 Mihomo 内核

```bash
./mihomo -f config.yaml
```

#### 步骤 2：通过 API 获取订阅信息

1. 获取代理提供者列表
2. 查看提供者中的代理
3. 根据需要触发更新

#### 步骤 3：集成到应用中

**Go 示例：**

```go
package main

import (
    "bytes"
    "encoding/json"
    "fmt"
    "io"
    "net/http"
)

type MihomoClient struct {
    BaseURL string
    Secret  string
    Client  *http.Client
}

func NewMihomoClient(baseURL, secret string) *MihomoClient {
    return &MihomoClient{
        BaseURL: baseURL,
        Secret:  secret,
        Client:  &http.Client{},
    }
}

func (c *MihomoClient) doRequest(method, path string, body io.Reader) (*http.Response, error) {
    req, err := http.NewRequest(method, c.BaseURL+path, body)
    if err != nil {
        return nil, err
    }
    
    req.Header.Set("Authorization", "Bearer "+c.Secret)
    req.Header.Set("Content-Type", "application/json")
    
    return c.Client.Do(req)
}

// GetProviders 获取所有代理提供者
func (c *MihomoClient) GetProviders() (map[string]interface{}, error) {
    resp, err := c.doRequest("GET", "/providers/proxies", nil)
    if err != nil {
        return nil, err
    }
    defer resp.Body.Close()
    
    var result map[string]interface{}
    if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
        return nil, err
    }
    
    return result, nil
}

// UpdateProvider 更新代理提供者
func (c *MihomoClient) UpdateProvider(providerName string) error {
    resp, err := c.doRequest("PUT", "/providers/proxies/"+providerName, nil)
    if err != nil {
        return err
    }
    defer resp.Body.Close()
    
    if resp.StatusCode != http.StatusNoContent {
        return fmt.Errorf("update failed with status: %d", resp.StatusCode)
    }
    
    return nil
}

// HealthCheckProvider 对代理提供者进行健康检查
func (c *MihomoClient) HealthCheckProvider(providerName string) error {
    resp, err := c.doRequest("GET", "/providers/proxies/"+providerName+"/healthcheck", nil)
    if err != nil {
        return err
    }
    defer resp.Body.Close()
    
    if resp.StatusCode != http.StatusNoContent {
        return fmt.Errorf("health check failed with status: %d", resp.StatusCode)
    }
    
    return nil
}

func main() {
    client := NewMihomoClient("http://127.0.0.1:9090", "your-secret")
    
    // 获取所有提供者
    providers, err := client.GetProviders()
    if err != nil {
        fmt.Printf("Error getting providers: %v\n", err)
        return
    }
    
    fmt.Printf("Providers: %+v\n", providers)
    
    // 更新提供者
    if err := client.UpdateProvider("my-provider"); err != nil {
        fmt.Printf("Error updating provider: %v\n", err)
        return
    }
    
    fmt.Println("Provider updated successfully")
    
    // 健康检查
    if err := client.HealthCheckProvider("my-provider"); err != nil {
        fmt.Printf("Error health checking provider: %v\n", err)
        return
    }
    
    fmt.Println("Health check completed")
}
```

**Python 示例：**

```python
import requests

class MihomoClient:
    def __init__(self, base_url, secret):
        self.base_url = base_url
        self.secret = secret
        self.headers = {
            "Authorization": f"Bearer {secret}",
            "Content-Type": "application/json"
        }
    
    def get_providers(self):
        """获取所有代理提供者"""
        response = requests.get(
            f"{self.base_url}/providers/proxies",
            headers=self.headers
        )
        response.raise_for_status()
        return response.json()
    
    def update_provider(self, provider_name):
        """更新代理提供者"""
        response = requests.put(
            f"{self.base_url}/providers/proxies/{provider_name}",
            headers=self.headers
        )
        response.raise_for_status()
        return response
    
    def health_check_provider(self, provider_name):
        """对代理提供者进行健康检查"""
        response = requests.get(
            f"{self.base_url}/providers/proxies/{provider_name}/healthcheck",
            headers=self.headers
        )
        response.raise_for_status()
        return response

# 使用示例
client = MihomoClient("http://127.0.0.1:9090", "your-secret")

# 获取所有提供者
providers = client.get_providers()
print(f"Providers: {providers}")

# 更新提供者
client.update_provider("my-provider")
print("Provider updated successfully")

# 健康检查
client.health_check_provider("my-provider")
print("Health check completed")
```

### 最佳实践

1. **定时更新**：根据订阅提供者的更新频率，设置合理的定时任务来更新代理列表
2. **错误处理**：妥善处理网络错误、API 错误等情况
3. **缓存机制**：在应用层实现缓存，避免频繁调用 API
4. **并发控制**：避免同时发起多个更新请求
5. **监控告警**：监控订阅更新状态，及时发现问题

### 注意事项

1. **认证**：所有 API 请求都需要通过 `Authorization: Bearer <secret>` 进行认证
2. **URL 编码**：提供者名称等参数需要使用 URL 编码
3. **异步更新**：更新操作是异步的，需要轮询或通过其他方式确认更新完成
4. **性能影响**：频繁的健康检查可能会影响性能，建议合理设置间隔
5. **网络环境**：确保 Mihomo 内核能够访问订阅 URL

## 总结

Mihomo 内核提供了强大而灵活的订阅格式处理机制，支持多种订阅格式和协议。通过 RESTful API，可以方便地集成订阅获取功能到各种应用中。合理使用这些功能，可以实现自动化的订阅管理、健康检查和节点选择。

## 相关资源

- **API 文档**: `docs/api.md`
- **官方文档**: https://wiki.metacubex.one/api/
- **配置文件**: `config.yaml`
- **核心代码**: `adapter/provider/`, `component/resource/`, `common/convert/`