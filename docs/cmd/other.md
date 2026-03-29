# 其他命令 (version, sub, geoip, history)

其他命令包括版本查询、订阅管理、GeoIP 数据库管理和历史记录管理。

## 版本查询 (version)

### version - 显示版本信息

显示 mihomo-cli 的版本和构建信息。

**语法：**
```bash
mihomo-cli version
```

**显示内容：**
- mihomo-cli 版本号
- Git 提交哈希
- 构建日期
- Go 版本
- 操作系统（GOOS）
- 架构（GOARCH）

**示例：**
```bash
mihomo-cli version
```

### version kernel - 显示 Mihomo 内核版本

显示正在运行的 Mihomo 内核版本信息。

**语法：**
```bash
mihomo-cli version kernel
```

**显示内容：**
- Mihomo Kernel 版本
- Premium 版本标识
- Home 目录
- 配置文件路径

**示例：**
```bash
mihomo-cli version kernel
```

## 订阅管理 (sub)

### sub update - 更新代理订阅

触发 Mihomo 更新所有代理提供者的订阅配置。

**语法：**
```bash
mihomo-cli sub update
```

**示例：**
```bash
mihomo-cli sub update
```

## GeoIP 数据库管理 (geoip)

### geoip update - 更新 GeoIP 数据库

从配置的数据源更新 GeoIP 地理位置数据库文件。

**语法：**
```bash
mihomo-cli geoip update
```

**示例：**
```bash
mihomo-cli geoip update
mihomo-cli geoip update -o json
```

### geoip status - 查询 GeoIP 数据库状态

检查 GeoIP 数据库文件的存在性、大小和最后修改时间。

**语法：**
```bash
mihomo-cli geoip status
```

**显示内容：**
- 数据库状态（已安装/未安装）
- 文件路径
- 文件名
- 文件大小
- 最后更新时间
- 存储目录
- 支持的文件名列表

**示例：**
```bash
mihomo-cli geoip status
mihomo-cli geoip status -o json
```

**支持的文件名（按优先级）：**
- Country.mmdb
- geoip.db
- geoip.metadb
- GeoIP.dat

## 历史记录管理 (history)

### history - 查看命令历史记录

查看所有执行过的命令历史记录。

**语法：**
```bash
mihomo-cli history
```

**选项：**
- `-l, --limit` - 显示记录数量（默认 50）

**显示内容：**
- 时间
- 命令
- 状态（✓ 成功 / ✗ 失败）

**示例：**
```bash
# 查看最近 50 条记录
mihomo-cli history

# 查看最近 20 条记录
mihomo-cli history --limit 20

# JSON 格式输出
mihomo-cli history -o json
```

### history clear - 清除历史记录

清除所有命令历史记录。

**语法：**
```bash
mihomo-cli history clear
```

**示例：**
```bash
mihomo-cli history clear
```

## 注意事项

1. 版本信息是在构建时注入的
2. 订阅更新会更新所有代理提供者的配置
3. GeoIP 数据库用于地理位置路由规则
4. 更新 GeoIP 数据库需要 Mihomo 配置中设置数据源
5. 历史记录存储在 `~/.config/.mihomo-cli/history/commands.jsonl`
6. 清除历史记录需要确认
