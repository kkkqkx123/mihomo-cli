# IP数据库切换分析报告

## 概述

本文档分析了 mihomo-go 项目中 IP 数据库的使用机制，并评估了切换不同 IP 数据库源的可行性和方法。

## 当前IP数据库使用情况

### 1. 支持的数据库格式

项目支持**两种数据格式**：

#### MMDB格式（默认）
- **技术实现**：使用 `maxminddb-golang` 库读取
- **特点**：支持 mmap 内存映射，查询效率高
- **支持类型**：
  - `typeMaxmind` - 标准 MaxMind 格式
  - `typeSing` - sing-geoip 格式
  - `typeMetaV0` - Meta 优化格式

#### Geodata格式
- **技术实现**：使用 Protobuf 格式
- **特点**：支持按需加载，内存占用更小
- **加载器**：
  - `standard` - 完整加载到内存
  - `memconservative` - 按需加载，内存优化

### 2. 支持的数据库文件

系统按以下优先级自动识别数据库文件：

| 优先级 | 文件名 | 格式 | 说明 |
|--------|--------|------|------|
| 1 | `Country.mmdb` | MaxMind MMDB | 标准 MaxMind 格式 |
| 2 | `geoip.db` | MMDB | 另一种 MMDB 格式命名 |
| 3 | `geoip.metadb` | Meta MMDB | Meta 专用的优化格式（默认） |
| 4 | `GeoIP.dat` | Protobuf | Geodata 模式专用 |

### 3. 默认下载源

所有数据库默认从 MetaCubeX GitHub 仓库下载：

```yaml
GeoXUrl:
  Mmdb:    "https://github.com/MetaCubeX/meta-rules-dat/releases/download/latest/geoip.metadb"
  ASN:     "https://github.com/MetaCubeX/meta-rules-dat/releases/download/latest/GeoLite2-ASN.mmdb"
  GeoIp:   "https://github.com/MetaCubeX/meta-rules-dat/releases/download/latest/geoip.dat"
  GeoSite: "https://github.com/MetaCubeX/meta-rules-dat/releases/download/latest/geosite.dat"
```

**代码位置**：`mihomo-1.19.21/config/config.go:568-573`

### 4. 存储位置

- **Windows**: `%USERPROFILE%\.config\mihomo\`
- **Linux/Mac**: `~/.config/mihomo/`

## 切换IP数据库的方法

### ✅ 结论：完全支持切换

项目设计灵活，支持多种方式切换 IP 数据库源。

### 方法一：配置文件切换下载源

在配置文件中添加 `geox-url` 配置：

```yaml
geox-url:
  geoip: "https://your-custom-source/geoip.dat"
  mmdb: "https://your-custom-source/geoip.metadb"
  asn: "https://your-custom-source/GeoLite2-ASN.mmdb"
  geosite: "https://your-custom-source/geosite.dat"
```

**优点**：
- 官方支持的方式
- 支持自动更新
- 配置清晰明确

### 方法二：切换数据格式模式

```yaml
# 启用 Geodata 模式（使用 GeoIP.dat）
geodata-mode: true

# 选择加载器
geodata-loader: "memconservative"  # 或 "standard"
```

**适用场景**：
- 内存受限环境
- 需要按需加载
- 使用 Protobuf 格式数据

### 方法三：手动替换数据库文件

直接将数据库文件放到配置目录，系统会自动识别。

**步骤**：
1. 下载所需的数据库文件
2. 放置到配置目录（`~/.config/mihomo/`）
3. 重启服务或调用更新API

**优点**：
- 完全离线操作
- 不依赖网络下载
- 可使用自定义数据库

### 方法四：使用国内镜像源

解决 GitHub 访问不稳定问题：

```yaml
geox-url:
  geoip: "https://fastly.jsdelivr.net/gh/MetaCubeX/meta-rules-dat@release/geoip.dat"
  mmdb: "https://fastly.jsdelivr.net/gh/MetaCubeX/meta-rules-dat@release/geoip.metadb"
  asn: "https://fastly.jsdelivr.net/gh/MetaCubeX/meta-rules-dat@release/GeoLite2-ASN.mmdb"
  geosite: "https://fastly.jsdelivr.net/gh/MetaCubeX/meta-rules-dat@release/geosite.dat"
```

**优点**：
- 国内访问速度快
- 稳定性高
- 无需修改其他配置

## 可用的替代IP数据库源

| 数据源 | 说明 | URL示例 | 优点 | 缺点 |
|--------|------|---------|------|------|
| MetaCubeX官方 | 默认源 | `github.com/MetaCubeX/meta-rules-dat` | 官方维护，更新及时 | 国内访问不稳定 |
| jsDelivr CDN | 国内加速 | `fastly.jsdelivr.net/gh/MetaCubeX/...` | 国内速度快，稳定 | 依赖CDN服务 |
| MaxMind官方 | GeoLite2 | 需注册账号获取 | 数据权威，更新频繁 | 需要注册账号 |
| IPinfo | 免费ASN数据库 | `ipinfo.io` | ASN数据准确 | 需要API密钥 |
| 自建服务器 | 完全可控 | 自定义URL | 完全自主控制 | 需要自行维护 |

## 自动更新配置

```yaml
# 启用自动更新
geo-auto-update: true

# 更新间隔（小时）
geo-update-interval: 24
```

**更新机制**：
- 定时检查更新
- 哈希验证避免重复下载
- 原子替换确保数据一致性
- 更新后自动重新加载数据库

## 技术实现细节

### 数据库加载流程

```
程序启动
  ↓
检查数据库文件是否存在
  ↓
  存在 → 验证文件完整性
         ↓
      有效 → 加载到内存
      无效 → 删除并重新下载
  ↓
  不存在 → 从配置的URL下载
           ↓
        验证完整性
           ↓
        加载到内存
```

### MMDB读取实现

**代码位置**：`mihomo-1.19.21/component/mmdb/mmdb.go`

```go
func IPInstance() IPReader {
    ipOnce.Do(func() {
        mmdbPath := C.Path.MMDB()
        mmdb, err := maxminddb.Open(mmdbPath)
        // 根据数据库类型设置不同的读取器
        switch mmdb.Metadata.DatabaseType {
        case "sing-geoip":
            ipReader.databaseType = typeSing
        case "Meta-geoip0":
            ipReader.databaseType = typeMetaV0
        default:
            ipReader.databaseType = typeMaxmind
        }
    })
    return ipReader
}
```

### Geodata加载实现

**代码位置**：`mihomo-1.19.21/component/geodata/init.go`

支持两种加载器：
- `standard`：完整加载，查询快速
- `memconservative`：按需加载，内存优化

## 实际应用场景

### 1. GEOIP路由规则

```yaml
rules:
  - GEOIP,CN,DIRECT          # 国内流量直连
  - GEOIP,HK,Proxy           # 香港流量走代理
  - GEOIP,US,US-Proxy        # 美国流量走美国代理
  - MATCH,Proxy              # 其他流量走默认代理
```

### 2. DNS过滤

```yaml
dns:
  enable: true
  fallback:
    - https://1.1.1.1/dns-query
  fallback-filter:
    geoip: true              # 启用 GeoIP 检查
    geoip-code: CN           # 当 IP 属于 CN 时，不使用 fallback
```

### 3. ASN查询

用于识别IP所属的自治系统，支持：
- GeoLite2-ASN 格式
- IPinfo ASN 格式

## 推荐配置方案

### 方案一：国内用户推荐

```yaml
# 使用国内镜像源
geox-url:
  geoip: "https://fastly.jsdelivr.net/gh/MetaCubeX/meta-rules-dat@release/geoip.dat"
  mmdb: "https://fastly.jsdelivr.net/gh/MetaCubeX/meta-rules-dat@release/geoip.metadb"
  asn: "https://fastly.jsdelivr.net/gh/MetaCubeX/meta-rules-dat@release/GeoLite2-ASN.mmdb"
  geosite: "https://fastly.jsdelivr.net/gh/MetaCubeX/meta-rules-dat@release/geosite.dat"

# 启用自动更新
geo-auto-update: true
geo-update-interval: 168  # 每周更新一次
```

### 方案二：内存受限环境

```yaml
# 使用 Geodata 模式
geodata-mode: true
geodata-loader: "memconservative"

# 使用国内镜像源
geox-url:
  geoip: "https://fastly.jsdelivr.net/gh/MetaCubeX/meta-rules-dat@release/geoip.dat"
  geosite: "https://fastly.jsdelivr.net/gh/MetaCubeX/meta-rules-dat@release/geosite.dat"
```

### 方案三：离线环境

1. 手动下载数据库文件
2. 放置到配置目录
3. 禁用自动更新：

```yaml
geo-auto-update: false
```

## 注意事项

1. **网络要求**：首次运行需要访问下载源或手动放置数据库文件
2. **磁盘空间**：数据库文件约 10-20MB
3. **内存占用**：根据加载模式不同，内存占用有差异
4. **更新频率**：建议根据实际需求配置，避免频繁更新
5. **数据源可靠性**：建议配置国内镜像或其他可靠源
6. **文件验证**：系统会自动验证文件完整性，损坏文件会被重新下载

## 总结

mihomo-go 项目的 IP 数据库系统设计非常灵活：

- ✅ 支持多种数据格式（MMDB、Geodata）
- ✅ 支持多种数据源（官方、镜像、自建）
- ✅ 支持灵活切换（配置文件、手动替换）
- ✅ 支持自动更新（可配置间隔）
- ✅ 支持国内镜像（解决访问问题）

**推荐做法**：国内用户使用 jsDelivr CDN 镜像源，配置自动更新，确保数据库及时更新且访问稳定。
