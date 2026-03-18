# Mihomo 内核代理配置文件解析与应用机制分析

本文档详细分析了 Mihomo 内核如何解析代理配置文件并应用，包括订阅链接的获取、Base64 解码、V2Ray 格式转换等核心流程。

## 一、整体架构流程

```
配置文件 (config.yaml)
    ↓
config.ParseConfig() [mihomo-1.19.21/config/config.go:603]
    ↓
parseProxies() [config.go:855]
    ↓
provider.ParseProxyProvider() [adapter/provider/parser.go:46]
    ↓
ProxySetProvider [adapter/provider/provider.go:119]
    ↓
resource.Fetcher [component/resource/fetcher.go:20]
    ↓
HTTPVehicle.Read() [component/resource/vehicle.go:122]
    ↓
NewProxiesParser() [adapter/provider/provider.go:342]
    ↓
convert.ConvertsV2Ray() [common/convert/converter.go:16]
    ↓
adapter.ParseProxy() [adapter/parser.go:11]
    ↓
最终代理对象
```

## 二、订阅链接获取流程

### 1. 配置解析入口

在 `config/config.go:855` 的 `parseProxies()` 函数中：

```go
// 解析 proxy-providers 配置
for name, mapping := range providersConfig {
    pd, err := provider.ParseProxyProvider(name, mapping)
    providersMap[name] = pd
}
```

### 2. ProxyProvider 解析

在 `adapter/provider/parser.go:46` 的 `ParseProxyProvider()` 函数中：

```go
func ParseProxyProvider(name string, mapping map[string]any) (P.ProxyProvider, error) {
    // 解析配置结构
    schema := &proxyProviderSchema{
        HealthCheck: healthCheckSchema{Lazy: true},
    }
    
    // 根据类型创建不同的 Vehicle
    switch schema.Type {
    case "http":
        vehicle = resource.NewHTTPVehicle(schema.URL, path, schema.Proxy, schema.Header, ...)
    case "file":
        vehicle = resource.NewFileVehicle(path)
    case "inline":
        return NewInlineProvider(name, schema.Payload, parser, hc)
    }
    
    // 创建解析器
    parser, err := NewProxiesParser(name, schema.Filter, schema.ExcludeFilter, ...)
    
    // 创建 ProxySetProvider
    return NewProxySetProvider(name, interval, schema.Payload, parser, vehicle, hc)
}
```

### 3. HTTP 订阅获取

在 `component/resource/vehicle.go:122` 的 `HTTPVehicle.Read()` 方法中：

```go
func (h *HTTPVehicle) Read(ctx context.Context, oldHash utils.HashType) (buf []byte, hash utils.HashType, err error) {
    // 发送 HTTP GET 请求
    resp, err := mihomoHttp.HttpRequest(ctx, h.url, http.MethodGet, header, nil, ...)
    
    // 提取 subscription-userinfo 响应头
    if subscriptionInfo := resp.Header.Get("subscription-userinfo"); subscriptionInfo != "" {
        cachefile.Cache().SetSubscriptionInfo(name, subscriptionInfo)
    }
    
    // 读取响应内容
    buf, err = io.ReadAll(reader)
    return buf, hash, nil
}
```

## 三、Base64 解码和 V2Ray 格式转换

### 1. 代理解析器

在 `adapter/provider/provider.go:342` 的 `NewProxiesParser()` 函数中：

```go
func NewProxiesParser(...) (resource.Parser[[]C.Proxy], error) {
    return func(buf []byte) ([]C.Proxy, error) {
        schema := &ProxySchema{}
        
        // 首先尝试 YAML 格式解析
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
        for _, mapping := range schema.Proxies {
            proxy, err := adapter.ParseProxy(mapping, adapter.WithProviderName(pdName))
            proxies = append(proxies, proxy)
        }
        
        return proxies, nil
    }, nil
}
```

### 2. V2Ray 格式转换核心

在 `common/convert/converter.go:16` 的 `ConvertsV2Ray()` 函数中：

```go
func ConvertsV2Ray(buf []byte) ([]map[string]any, error) {
    // 第一次 Base64 解码
    data := DecodeBase64(buf)
    
    // 按行分割
    arr := strings.Split(string(data), "\n")
    
    for _, line := range arr {
        scheme, body, found := strings.Cut(line, "://")
        
        switch strings.ToLower(scheme) {
        case "vmess":
            // 第二次 Base64 解码 vmess:// 后的内容
            dcBuf, err := tryDecodeBase64([]byte(body))
            if err != nil {
                // 尝试 Xray VMessAEAD 格式
                urlVMess, err := url.Parse(line)
                // ...
            }
            
            // 解析 JSON 配置
            values := make(map[string]any)
            jsonDc.Decode(&values)
            
            // 构建 mihomo 格式的代理配置
            vmess := make(map[string]any)
            vmess["name"] = values["ps"]
            vmess["type"] = "vmess"
            vmess["server"] = values["add"]
            vmess["port"] = values["port"]
            vmess["uuid"] = values["id"]
            // ... 其他字段
            
        case "vless", "trojan", "ss", "ssr", "hysteria", "hysteria2", "tuic":
            // 解析其他协议
        }
    }
    
    return proxies, nil
}
```

### 3. Base64 解码逻辑

在 `common/convert/base64.go:16` 中：

```go
// 自动检测并解码 Base64
func DecodeBase64(buf []byte) []byte {
    result, err := tryDecodeBase64(buf)
    if err != nil {
        return buf  // 如果不是 Base64，返回原始内容
    }
    return result
}

func tryDecodeBase64(buf []byte) ([]byte, error) {
    // 尝试 RawStdEncoding
    dBuf := make([]byte, encRaw.DecodedLen(len(buf)))
    n, err := encRaw.Decode(dBuf, buf)
    if err != nil {
        // 尝试 StdEncoding
        n, err = enc.Decode(dBuf, buf)
        if err != nil {
            return nil, err
        }
    }
    return dBuf[:n], nil
}
```

## 四、订阅链接处理示例

以 `https://msub.xn--m7r52rosihxm.com/api/v1/client/subscribe?token=...` 为例：

### 1. 第一次 Base64 解码

HTTPVehicle 下载订阅内容后，得到二次 Base64 编码的内容：
```
dm1lc3M6Ly9leUoySWpvaU1pSXNJbkJ6SWpvaVRYbFRaWE...  (Base64 编码)
```

`DecodeBase64()` 第一次解码后得到：
```
vmess://eyJ2IjoiMiIsInBzIjoiXHU1MjY5XHU0ZjU5XHU2ZDQxXHU5MWNmXHVmZjFhOTYuMzEgR0IiLC...  (多行)
vmess://eyJ2IjoiMiIsInBzIjoiXHU1OTU3XHU5OTEwXHU1MjMwXHU2NzFmXHVmZjFhXHU5NTdmXHU2NzFm...
```

### 2. 按行分割并解析每个 vmess:// 链接

对于每个 `vmess://` 链接：

1. 提取 `://` 后的 Base64 内容
2. 第二次 Base64 解码得到 JSON：
```json
{
  "v":"2",
  "ps":"剩余流量：96.31 GB",
  "add":"planb.mojcn.com",
  "port":"16617",
  "id":"ff3b259a-e082-408f-b862-e5f6836f0da3",
  "aid":"0",
  "net":"ws",
  "type":"none",
  "host":"4591f7a1d707444c7f68a38baba091ce.mobgslb.tbcache.com",
  "path":"\/",
  "tls":""
}
```

3. 转换为 mihomo 格式：
```yaml
name: "剩余流量：96.31 GB"
type: vmess
server: planb.mojcn.com
port: 16617
uuid: ff3b259a-e082-408f-b862-e5f6836f0da3
alterId: 0
network: ws
ws-opts:
  path: /
  headers:
    Host: 4591f7a1d707444c7f68a38baba091ce.mobgslb.tbcache.com
udp: true
xudp: true
```

## 五、代理解析和应用

### 1. 代理解析

在 `adapter/parser.go:11` 的 `ParseProxy()` 函数中：

```go
func ParseProxy(mapping map[string]any, options ...ProxyOption) (C.Proxy, error) {
    proxyType := mapping["type"].(string)
    
    switch proxyType {
    case "vmess":
        vmessOption := &outbound.VmessOption{BasicOption: basicOption}
        decoder.Decode(mapping, vmessOption)
        proxy, err = outbound.NewVmess(*vmessOption)
    case "ss", "trojan", "vless", "hysteria2":
        // 解析其他协议
    }
    
    return NewProxy(proxy), nil
}
```

### 2. 自动更新机制

在 `component/resource/fetcher.go:146` 的 `pullLoop()` 方法中：

```go
func (f *Fetcher[V]) pullLoop(forceUpdate bool) {
    timer := time.NewTimer(initialInterval)
    defer timer.Stop()
    
    for {
        select {
        case <-timer.C:
            f.updateWithLog()  // 定期更新
            timer.Reset(f.interval)
        case <-f.ctx.Done():
            return
        }
    }
}
```

### 3. 订阅信息提取

在 `adapter/provider/subscription_info.go:18` 中：

```go
func NewSubscriptionInfo(userinfo string) *SubscriptionInfo {
    // 解析格式: upload=123;download=456;total=1000;expire=1234567890
    for _, field := range strings.Split(userinfo, ";") {
        name, value, _ := strings.Cut(field, "=")
        switch name {
        case "upload":
            si.Upload = intValue
        case "download":
            si.Download = intValue
        case "total":
            si.Total = intValue
        case "expire":
            si.Expire = intValue
        }
    }
    return si
}
```

## 六、关键特性

1. **自动格式检测**：自动识别 YAML、V2Ray 分享链接、Base64 编码格式
2. **多协议支持**：支持 vmess、vless、ss、ssr、trojan、hysteria、hysteria2、tuic 等
3. **智能 Base64 解码**：自动检测并解码多种 Base64 编码格式
4. **订阅信息提取**：从 HTTP 响应头提取流量和过期信息
5. **自动更新**：支持定时更新和文件监视
6. **ETag 缓存**：避免重复下载未变更的订阅
7. **退避重试**：更新失败时使用指数退避策略
8. **过滤器机制**：支持正则表达式过滤和排除节点

## 七、配置示例

```yaml
proxy-providers:
  my-provider:
    type: http
    url: "https://example.com/subscription"
    interval: 3600
    path: ./my-provider.yaml
    proxy: DIRECT
    header:
      User-Agent:
        - "Clash/v1.18.0"
    health-check:
      enable: true
      interval: 600
      url: https://cp.cloudflare.com/generate_204
```

## 八、核心代码位置

| 功能模块 | 文件路径 | 关键函数/结构 |
|---------|---------|-------------|
| 配置解析 | `config/config.go` | `ParseConfig()`, `parseProxies()` |
| Provider 解析 | `adapter/provider/parser.go` | `ParseProxyProvider()` |
| Provider 实现 | `adapter/provider/provider.go` | `ProxySetProvider`, `NewProxiesParser()` |
| 数据获取 | `component/resource/vehicle.go` | `HTTPVehicle.Read()` |
| 自动更新 | `component/resource/fetcher.go` | `Fetcher.pullLoop()` |
| 格式转换 | `common/convert/converter.go` | `ConvertsV2Ray()` |
| Base64 解码 | `common/convert/base64.go` | `DecodeBase64()` |
| 代理解析 | `adapter/parser.go` | `ParseProxy()` |
| 订阅信息 | `adapter/provider/subscription_info.go` | `NewSubscriptionInfo()` |
| 健康检查 | `adapter/provider/healthcheck.go` | `HealthCheck.check()` |

## 九、错误处理和容错机制

1. **格式兼容性**：YAML 解析失败时自动尝试 V2Ray 格式转换
2. **Base64 容错**：支持多种 Base64 编码格式，解码失败返回原始内容
3. **网络重试**：使用指数退避策略处理网络错误
4. **本地缓存**：优先使用本地缓存文件，网络失败时降级使用缓存
5. **健康检查**：定期检查代理可用性，自动剔除不可用节点
