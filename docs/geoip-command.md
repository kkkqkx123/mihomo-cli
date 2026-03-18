# GeoIP 命令使用指南

## 概述

`geoip` 命令用于管理 Mihomo 代理核心的 GeoIP 地理位置数据库。GeoIP 数据库是 mihomo 根据 IP 地址地理位置进行流量路由和 DNS 过滤的基础数据。

## 功能特性

- **更新数据库**: 从配置的数据源下载或更新 GeoIP 数据库
- **状态检查**: 检查本地 GeoIP 数据库的状态、文件大小和最后更新时间
- **多格式支持**: 支持 Table 和 JSON 两种输出格式，便于脚本解析
- **文件自动检测**: 自动检测多种格式的 GeoIP 数据库文件

## 支持的 GeoIP 数据库格式

mihomo 支持以下多种 GeoIP 数据库格式（按优先级排序）：

1. **Country.mmdb** - MaxMind MMDB 标准格式
2. **geoip.db** - 另一种 MMDB 格式命名
3. **geoip.metadb** - Meta 专用的优化 MMDB 格式
4. **GeoIP.dat** - Protobuf 格式（Geodata 模式）

## 存储位置

GeoIP 数据库文件存储在以下目录：

- **Windows**: `%USERPROFILE%\.config\mihomo\`
- **Linux/Mac**: `~/.config/mihomo/`

## 命令结构

```bash
mihomo-cli geoip [子命令] [选项]
```

### 子命令

#### 1. 更新 GeoIP 数据库

```bash
mihomo-cli geoip update [选项]
```

**功能说明**: 从配置的数据源下载或更新 GeoIP 数据库文件。

**示例**:
```bash
# 更新 GeoIP 数据库
mihomo-cli geoip update

# 以 JSON 格式输出
mihomo-cli geoip update -o json
```

**输出示例**:
```
✓ GeoIP 数据库更新成功
```

**JSON 输出示例**:
```json
{
  "status": "success",
  "data": {
    "message": "GeoIP 数据库更新成功",
    "action": "update"
  }
}
```

**注意事项**:
- 此命令会调用 Mihomo RESTful API 的 `POST /configs/geo` 接口
- 更新过程可能需要一些时间，取决于网络状况
- 如果配置了自定义下载源，将使用配置中的地址

#### 2. 查询 GeoIP 状态

```bash
mihomo-cli geoip status [选项]
```

**功能说明**: 检查本地 GeoIP 数据库文件的存在性、大小和最后更新时间。

**示例**:
```bash
# 检查 GeoIP 数据库状态
mihomo-cli geoip status

# 以 JSON 格式输出
mihomo-cli geoip status -o json
```

**输出示例（已安装）**:
```
GeoIP 数据库状态: ✓ 已安装

文件路径: C:\Users\Username\.config\mihomo\geoip.metadb
文件名: geoip.metadb
文件大小: 18.45 MB
最后更新: 2024-01-15 10:30:45
存储目录: C:\Users\Username\.config\mihomo
```

**输出示例（未安装）**:
```
GeoIP 数据库状态: ✗ 未安装

预期存储目录: C:\Users\Username\.config\mihomo

支持的文件名（按优先级）:
  - Country.mmdb
  - geoip.db
  - geoip.metadb
  - GeoIP.dat

提示: 使用 'mihomo-cli geoip update' 命令下载 GeoIP 数据库
```

**JSON 输出示例（已安装）**:
```json
{
  "status": "success",
  "data": {
    "exists": true,
    "file_path": "C:\\Users\\Username\\.config\\mihomo\\geoip.metadb",
    "file_name": "geoip.metadb",
    "file_size": 19345678,
    "mod_time": "2024-01-15T10:30:45.123456789+08:00",
    "directory": "C:\\Users\\Username\\.config\\mihomo"
  }
}
```

**JSON 输出示例（未安装）**:
```json
{
  "status": "success",
  "data": {
    "exists": false,
    "directory": "C:\\Users\\Username\\.config\\mihomo"
  }
}
```

## 全局选项

所有 `geoip` 子命令都支持以下全局选项：

### -o, --output string

- **默认值**: `table`
- **可选值**: `table`, `json`
- **说明**: 指定输出格式，用于控制命令的输出形式

**示例**:
```bash
# Table 格式（默认）
mihomo-cli geoip status -o table

# JSON 格式（便于脚本解析）
mihomo-cli geoip status -o json
```

### -c, --config string

- **说明**: 指定配置文件路径
- **默认值**: `~/.mihomo-cli/config.yaml`

**示例**:
```bash
mihomo-cli geoip status -c ./config.yaml
```

### --api string

- **说明**: 覆盖配置文件中的 API 地址
- **格式**: `http://127.0.0.1:9090`

**示例**:
```bash
mihomo-cli geoip update --api http://192.168.1.100:9090
```

### --secret string

- **说明**: 覆盖配置文件中的 API 密钥

**示例**:
```bash
mihomo-cli geoip update --secret your-secret-token
```

### -t, --timeout int

- **默认值**: `10`
- **单位**: 秒
- **说明**: 请求超时时间（仅影响 API 调用的超时）

**示例**:
```bash
# 设置 30 秒超时
mihomo-cli geoip update -t 30
```

## 完整示例

### 示例 1: 检查 GeoIP 数据库状态并更新

```bash
# 1. 先检查当前状态
mihomo-cli geoip status

# 如果显示未安装，则执行更新
mihomo-cli geoip update

# 再次检查状态确认
mihomo-cli geoip status
```

### 示例 2: 脚本自动化检查

```bash
#!/bin/bash
# 检查 GeoIP 数据库是否存在，如果不存在则自动更新

STATUS=$(mihomo-cli geoip status -o json)
EXISTS=$(echo $STATUS | grep -o '"exists":[a-z]*' | cut -d':' -f2)

if [ "$EXISTS" = "false" ]; then
    echo "GeoIP 数据库不存在，开始更新..."
    mihomo-cli geoip update
else
    echo "GeoIP 数据库已存在"
fi
```

### 示例 3: PowerShell 自动化

```powershell
# 检查 GeoIP 数据库状态
$status = mihomo-cli geoip status -o json | ConvertFrom-Json

if (-not $status.data.exists) {
    Write-Host "GeoIP 数据库不存在，开始更新..." -ForegroundColor Yellow
    mihomo-cli geoip update
    if ($LASTEXITCODE -eq 0) {
        Write-Host "更新成功！" -ForegroundColor Green
    }
} else {
    Write-Host "GeoIP 数据库已安装" -ForegroundColor Green
    Write-Host "文件大小: $([math]::Round($status.data.file_size / 1MB, 2)) MB"
    Write-Host "最后更新: $($status.data.mod_time)"
}
```

## 故障排除

### 问题 1: 更新失败

**症状**: 执行 `mihomo-cli geoip update` 时返回错误

**可能原因**:
1. Mihomo 核心未运行或 API 地址配置错误
2. 网络连接问题，无法访问下载源
3. API 密钥不正确
4. 配置文件权限问题

**解决方案**:
1. 检查 Mihomo 核心是否正在运行
2. 验证 API 地址和密钥配置：
   ```bash
   mihomo-cli config show
   ```
3. 检查网络连接，尝试手动访问下载源
4. 查看 Mihomo 核心日志获取详细信息

### 问题 2: 状态检查找不到文件

**症状**: `mihomo-cli geoip status` 显示未安装，但文件实际存在

**可能原因**:
1. 文件不在预期的存储目录
2. 文件名不符合支持的格式
3. 文件权限问题

**解决方案**:
1. 确认文件存储路径：
   - Windows: `%USERPROFILE%\.config\mihomo\`
   - Linux/Mac: `~/.config/mihomo/`
2. 检查文件名是否为支持的格式之一
3. 检查文件权限，确保可读

### 问题 3: JSON 输出格式错误

**症状**: 脚本无法解析 JSON 输出

**可能原因**:
1. 输出中包含额外信息
2. 编码问题

**解决方案**:
1. 确保使用 `-o json` 参数
2. 检查标准错误输出，分离 stderr 和 stdout
3. 使用管道时避免混用 Table 和 JSON 格式

## 相关文档

- [GeoIP 获取与加载机制分析](geoip-mechanism.md)
- [Mihomo RESTful API 文档](spec/mihono-api.md)
- [代理节点操作指南](proxy-node-operations.md)
- [配置管理命令](config-command.md)

## 版本历史

- **v1.0.0**: 初始版本
  - 添加 `geoip update` 命令
  - 添加 `geoip status` 命令
  - 支持 Table 和 JSON 输出格式
  - 自动检测多种 GeoIP 数据库格式
