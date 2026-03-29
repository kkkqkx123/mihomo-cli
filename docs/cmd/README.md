# Mihomo CLI 命令文档

本文档提供了 Mihomo CLI 所有命令的详细说明。

## 文档索引

### 1. [代理管理命令 (proxy)](./proxy.md)
管理 Mihomo 的代理节点，包括列出、切换、测试和自动选择节点。

**主要命令：**
- `proxy list` - 列出代理节点
- `proxy switch` - 切换代理节点
- `proxy test` - 测试节点延迟
- `proxy auto` - 自动选择最快节点
- `proxy unfix` - 取消固定代理
- `proxy current` - 获取当前使用的节点

### 2. [规则管理命令 (rule)](./rule.md)
管理 Mihomo 的路由规则，包括列出、禁用和启用规则。

**主要命令：**
- `rule list` - 列出所有规则
- `rule provider` - 列出规则提供者
- `rule disable` - 禁用指定规则
- `rule enable` - 启用指定规则

### 3. [缓存管理命令 (cache)](./cache.md)
管理 Mihomo 的缓存，包括 FakeIP 和 DNS 缓存。

**主要命令：**
- `cache clear fakeip` - 清空 FakeIP 池
- `cache clear dns` - 清空 DNS 缓存

### 4. [连接管理命令 (conn)](./conn.md)
管理活跃连接，包括列出、关闭指定连接和关闭所有连接。

**主要命令：**
- `conn list` - 列出活跃连接
- `conn close` - 关闭指定连接
- `conn close-all` - 关闭所有连接

### 5. [DNS 管理命令 (dns)](./dns.md)
执行 DNS 查询和查看 DNS 配置。

**主要命令：**
- `dns query` - 执行 DNS 查询
- `dns config` - 显示 DNS 配置

### 6. [日志管理命令 (logs)](./logs.md)
查看、统计、搜索和导出 Mihomo 内核的日志。

**主要命令：**
- `logs view` - 实时查看 Mihomo 日志
- `logs stats` - 统计日志信息
- `logs search` - 搜索日志
- `logs export` - 导出日志到文件

### 7. [监控管理命令 (monitor)](./monitor.md)
监控 Mihomo 的流量和内存使用情况。

**主要命令：**
- `monitor traffic` - 获取流量统计
- `monitor memory` - 获取内存使用

### 8. [服务管理命令 (service)](./service.md)
管理 Mihomo 系统服务的安装、启动、停止和卸载。

**主要命令：**
- `service install` - 安装服务
- `service uninstall` - 卸载服务
- `service start` - 启动服务
- `service stop` - 停止服务
- `service status` - 查询服务状态

### 9. [配置管理命令 (config, mihomo, backup)](./config.md)
管理 CLI 配置、Mihomo 配置和配置备份。

**主要命令：**
- `config init` - 初始化配置文件
- `config show` - 显示当前配置
- `config set` - 设置配置项
- `mihomo patch` - 热更新 Mihomo 配置
- `mihomo reload` - 重载 Mihomo 配置文件
- `mihomo edit` - 编辑 Mihomo 配置文件
- `backup create` - 创建配置备份
- `backup list` - 列出所有备份
- `backup restore` - 恢复配置备份
- `backup delete` - 删除配置备份
- `backup prune` - 清理旧备份

### 10. [系统管理命令 (system, sysproxy, diagnose, recovery, operation)](./system.md)
管理系统配置，包括系统代理、TUN 设备、路由表等。

**主要命令：**
- `system status` - 查询系统配置状态
- `system cleanup` - 清理系统配置
- `system validate` - 验证系统配置
- `system fix` - 修复系统配置问题
- `system snapshot create/list/restore/delete` - 配置快照管理
- `sysproxy get/set` - 系统代理管理
- `diagnose route/network` - 系统诊断
- `recovery detect/execute/status` - 自动恢复管理
- `operation query/clear/prune` - 操作记录管理

### 11. [进程管理命令 (start, stop, status, ps, cleanup, mode)](./process.md)
管理 Mihomo 内核进程和运行模式。

**主要命令：**
- `start` - 启动 Mihomo 内核
- `stop` - 停止 Mihomo 内核
- `status` - 查询 Mihomo 内核状态
- `ps` - 列出所有 Mihomo 进程
- `cleanup` - 清理残留的 PID 文件
- `mode get` - 查询当前运行模式
- `mode set` - 设置运行模式

### 12. [其他命令 (version, sub, geoip, history)](./other.md)
版本查询、订阅管理、GeoIP 数据库管理和历史记录管理。

**主要命令：**
- `version` - 显示版本信息
- `version kernel` - 显示 Mihomo 内核版本
- `sub update` - 更新代理订阅
- `geoip update` - 更新 GeoIP 数据库
- `geoip status` - 查询 GeoIP 数据库状态
- `history` - 查看命令历史记录
- `history clear` - 清除历史记录

## 全局选项

所有命令都支持以下全局选项：

- `-c, --config` - 配置文件路径（默认：`~/.config/.mihomo-cli/config.toml`）
- `-o, --output` - 输出格式（table/json）
- `--api` - API 地址（覆盖配置文件）
- `--secret` - API 密钥（覆盖配置文件）
- `-t, --timeout` - 请求超时时间（秒）

## 快速开始

### 初始化配置

```bash
mihomo-cli config init
```

### 启动 Mihomo 内核

```bash
mihomo-cli start
```

### 查看运行状态

```bash
mihomo-cli status
```

### 切换运行模式

```bash
mihomo-cli mode set rule
```

### 列出代理节点

```bash
mihomo-cli proxy list
```

### 测试节点延迟

```bash
mihomo-cli proxy test Proxy
```

### 自动选择最快节点

```bash
mihomo-cli proxy auto Proxy
```

### 停止 Mihomo 内核

```bash
mihomo-cli stop
```

## 获取帮助

查看所有可用命令：
```bash
mihomo-cli --help
```

查看特定命令的帮助：
```bash
mihomo-cli <command> --help
```

例如：
```bash
mihomo-cli proxy --help
mihomo-cli proxy list --help
```

## 注意事项

1. 某些命令需要管理员权限（如系统代理管理、系统配置清理等）
2. 服务管理命令仅支持 Windows 系统
3. 系统代理管理命令仅支持 Windows 系统
4. 配置文件默认存储在 `~/.config/.mihomo-cli/` 目录
5. 历史记录存储在 `~/.config/.mihomo-cli/history/` 目录
6. 备份文件存储在 `~/.config/.mihomo-cli/backups/` 目录

## 配置文件示例

```toml
[api]
address = "http://127.0.0.1:9090"
secret = ""
timeout = 10

[proxy]
test_url = "https://www.google.com/generate_204"
timeout = 10000
concurrent = 10

[output]
mode = "console"
file = ""
append = false

[mihomo]
enabled = true
executable = "mihomo"
config_file = "config.yaml"
auto_generate_secret = true
health_check_timeout = 30

[mihomo.api]
external_controller = "127.0.0.1:9090"

[mihomo.log]
level = "info"
```
