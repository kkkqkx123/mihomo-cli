# GeoIP 数据库管理指南

## 概述

GeoIP（地理位置数据库）用于 Mihomo 根据 IP 地址地理位置进行流量分流。mihomo-cli 提供 GeoIP 数据库的下载更新和状态查询功能。

## 命令参考

### 更新 GeoIP 数据库

从配置的数据源下载最新的 GeoIP 数据库文件：

```bash
mihomo-cli geoip update
```

**输出示例：**
```
✓ GeoIP 数据库更新成功
```

### 查询 GeoIP 数据库状态

检查本地 GeoIP 数据库文件的存在性和信息：

```bash
mihomo-cli geoip status
```

**输出示例（已安装）：**
```
GeoIP 数据库状态: ✓ 已安装

文件路径: C:\Users\<用户名>\.config\mihomo\Country.mmdb
文件名: Country.mmdb
文件大小: 4.52 MB
最后更新: 2026-05-30 12:00:00
存储目录: C:\Users\<用户名>\.config\mihomo
```

**输出示例（未安装）：**
```
GeoIP 数据库状态: ✗ 未安装

预期存储目录: C:\Users\<用户名>\.config\mihomo

支持的文件名（按优先级）:
  - Country.mmdb
  - geoip.db
  - geoip.metadb
  - GeoIP.dat

提示: 使用 'mihomo-cli geoip update' 命令下载 GeoIP 数据库
```

### JSON 输出

```bash
mihomo-cli geoip status -o json
```

## 文件存储位置

GeoIP 数据库文件存放在 Mihomo 配置目录：

- **Windows**: `C:\Users\<用户名>\.config\mihomo\`
- **Linux/macOS**: `~/.config/mihomo/`

## 支持的文件格式（按查找优先级）

1. `Country.mmdb` - MaxMind GeoLite2 格式（推荐）
2. `geoip.db` - 轻量级数据库格式
3. `geoip.metadb` - MetaDB 格式
4. `GeoIP.dat` - 传统 GeoIP 格式

## 注意事项

- GeoIP 数据库需要定期更新以获取最新的 IP 地理位置信息
- `geoip update` 命令通过 Mihomo API 触发更新，需要内核运行
- `geoip status` 命令直接检查本地文件系统，不需要内核运行
- 数据库文件可通过 Mihomo 配置文件中的 `geodata-loader` 选项切换加载器（`memconservative` 或 `standard`）

## 相关文档

- [GeoIP 工作机制分析](../../docs/mihomo-core/geoip-mechanism.md)