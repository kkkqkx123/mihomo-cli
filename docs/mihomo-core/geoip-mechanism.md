# GeoIP 获取与加载机制分析

## 概述

mihomo 项目中的 GeoIP 功能用于根据 IP 地址的地理位置进行流量路由和 DNS 过滤。系统支持两种数据格式和加载模式，并提供自动下载和更新机制。

## 是否需要下载外部服务

**是的**，mihomo 需要从外部服务下载 GeoIP 数据库文件。首次运行时，如果检测到本地不存在有效的 GeoIP 数据库，系统会自动从 GitHub Releases 下载。

## 下载源

### 默认下载地址

| 数据类型 | 下载地址 |
|---------|---------|
| Geodata 模式 | `https://github.com/MetaCubeX/meta-rules-dat/releases/download/latest/geoip.dat` |
| MMDB 模式 | `https://github.com/MetaCubeX/meta-rules-dat/releases/download/latest/geoip.metadb` |
| GeoSite | `https://github.com/MetaCubeX/meta-rules-dat/releases/download/latest/geosite.dat` |
| ASN 数据库 | `https://github.com/MetaCubeX/meta-rules-dat/releases/download/latest/GeoLite2-ASN.mmdb` |

### 下载特性

- **超时时间**: 90 秒
- **验证机制**: 下载后自动验证文件完整性
- **重试策略**: 失败后自动删除损坏文件并重新下载

## 存储位置

### 目录

- **Windows**: `%USERPROFILE%\.config\mihomo\`
- **Linux/Mac**: `~/.config/mihomo/`

### 支持的文件名

系统会按以下优先级查找 GeoIP 数据库文件：

| 文件名 | 格式 | 说明 |
|-------|------|------|
| `Country.mmdb` | MaxMind MMDB | 标准 MaxMind 格式 |
| `geoip.db` | MMDB | 另一种 MMDB 格式命名 |
| `geoip.metadb` | Meta MMDB | Meta 专用的优化格式 |
| `GeoIP.dat` | Protobuf | Geodata 模式专用 |

## 数据加载模式

### 1. Geodata 模式

**特点**：
- 使用 Protobuf 格式的 `GeoIP.dat` 文件
- 支持按需加载，节省内存
- 适合内存受限的环境

**加载器类型**：
- `standard`: 完整加载到内存
- `memconservative`: 按需加载，内存优化

### 2. MMDB 模式

**特点**：
- 使用 MaxMind MMDB 格式
- 支持 mmap 内存映射，高效查询
- 支持实时 IP 查询

### 模式切换

通过配置文件控制：

```yaml
geodata-mode: true          # 启用 Geodata 模式
geodata-loader: "standard"  # 或 "memconservative"
```

## 自动更新机制

### 配置项

```yaml
geo-auto-update: false        # 是否启用自动更新（默认关闭）
geo-update-interval: 24       # 更新间隔，单位小时
```

### 更新策略

1. **定时更新**: 根据配置的时间间隔定期更新
2. **手动触发**: 通过 RESTful API 手动触发更新
3. **哈希验证**: 下载前检查文件哈希，避免重复下载
4. **原子替换**: 更新完成后原子性替换文件
5. **缓存清理**: 更新后清除内存缓存并重新加载数据库

### API 接口

- **路由**: `POST /geo`
- **功能**: 立即触发所有 GeoIP 数据库更新

## 配置说明

### 自定义下载源

```yaml
geox-url:
  geoip: "https://fastly.jsdelivr.net/gh/MetaCubeX/meta-rules-dat@release/geoip.dat"
  geosite: "https://fastly.jsdelivr.net/gh/MetaCubeX/meta-rules-dat@release/geosite.dat"
  mmdb: "https://fastly.jsdelivr.net/gh/MetaCubeX/meta-rules-dat@release/geoip.metadb"
  asn: "https://fastly.jsdelivr.net/gh/MetaCubeX/meta-rules-dat@release/GeoLite2-ASN.mmdb"
```

### 完整配置示例

```yaml
# 启用 Geodata 模式
geodata-mode: true
geodata-loader: "memconservative"
geosite-matcher: "succinct"

# 自定义下载源
geox-url:
  geoip: "https://fastly.jsdelivr.net/gh/MetaCubeX/meta-rules-dat@release/geoip.dat"
  geosite: "https://fastly.jsdelivr.net/gh/MetaCubeX/meta-rules-dat@release/geosite.dat"
  mmdb: "https://fastly.jsdelivr.net/gh/MetaCubeX/meta-rules-dat@release/geoip.metadb"

# 自动更新配置
geo-auto-update: false
geo-update-interval: 24
```

## 使用场景

### 1. 路由规则

在规则配置中使用 GEOIP 规则根据国家代码路由流量：

```yaml
rules:
  - GEOIP,CN,DIRECT          # 国内流量直连
  - GEOIP,HK,Proxy           # 香港流量走代理
  - GEOIP,US,US-Proxy        # 美国流量走美国代理
  - MATCH,Proxy              # 其他流量走默认代理
```

### 2. DNS 过滤

检查 DNS 返回的 IP 是否属于指定国家，决定是否使用 Fallback DNS：

```yaml
dns:
  enable: true
  listen: 0.0.0.0:1053
  enhanced-mode: fake-ip
  nameserver:
    - 8.8.8.8
    - https://doh.pub/dns-query
  fallback:
    - https://1.1.1.1/dns-query
  fallback-filter:
    geoip: true              # 启用 GeoIP 检查
    geoip-code: CN           # 当 IP 属于 CN 时，不使用 fallback
    ipcidr:
      - 240.0.0.0/4          # 保留 IP 段不使用 fallback
```

## 工作流程

### 初始化流程

```
程序启动
  ↓
解析配置文件
  ↓
检查 GeoIP 文件是否存在
  ↓
  是 → 验证文件完整性
       ↓
    无效 → 删除并重新下载
       ↓
  有效 → 加载到内存
  ↓
否 → 从默认源下载
  ↓
  验证完整性
  ↓
  加载到内存
```

### 查询流程

```
收到 IP 地址
  ↓
判断加载模式
  ↓
Geodata 模式 → 使用 protobuf 匹配器
MMDB 模式   → 使用 MMDB 查询器
  ↓
返回国家代码
  ↓
根据规则执行相应操作
```

## 关键代码文件

| 功能模块 | 文件路径 | 说明 |
|---------|---------|------|
| 路径管理 | `constant/path.go` | 定义 GeoIP 文件存储路径和查找逻辑 |
| 初始化和下载 | `component/geodata/init.go` | 实现 GeoIP 初始化和自动下载逻辑 |
| MMDB 加载器 | `component/mmdb/mmdb.go` | MMDB 格式数据库加载和查询 |
| 标准加载器 | `component/geodata/standard/standard.go` | Geodata 标准加载器实现 |
| 保守加载器 | `component/geodata/memconservative/memc.go` | 内存优化的 Geodata 加载器 |
| 更新机制 | `component/updater/update_geo.go` | 自动更新和手动更新实现 |
| 配置解析 | `config/config.go` | GeoIP 相关配置项定义和解析 |
| GEOIP 规则 | `rules/common/geoip.go` | GEOIP 路由规则匹配实现 |
| API 路由 | `hub/route/configs.go` | RESTful API 路由定义 |

## 注意事项

1. **网络要求**: 首次运行需要访问 GitHub Releases 或配置的自定义下载源
2. **磁盘空间**: GeoIP 数据库文件大小约 10-20MB
3. **内存占用**: 根据加载模式不同，内存占用有所差异
4. **更新频率**: 建议根据实际需求配置更新间隔，避免频繁更新浪费流量
5. **数据源可靠性**: 如果默认下载源访问不稳定，建议配置本地镜像或其他可靠的数据源