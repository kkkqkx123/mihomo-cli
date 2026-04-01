# 守护进程模式使用指南

## 概述

守护进程模式允许 Mihomo 内核作为独立的后台进程运行，与 CLI 客户端完全分离。关闭终端不会影响内核运行，提供更稳定的服务体验。

## 主要特性

- ✅ **独立运行**：内核作为独立进程组运行，不受终端影响
- ✅ **日志管理**：支持日志文件输出和轮转
- ✅ **跨平台**：支持 Windows、Linux、macOS
- ✅ **向后兼容**：保留传统启动模式
- ✅ **灵活配置**：通过配置文件控制所有行为

## 快速开始

### 1. 启用守护进程模式

在配置文件 `config.toml` 中添加以下配置：

```toml
[daemon]
enabled = true
log_file = "/path/to/mihomo-daemon.log"
log_level = "info"
```

### 2. 启动 Mihomo

```bash
mihomo-cli start
```

### 3. 验证运行状态

```bash
mihomo-cli status
```

### 4. 停止 Mihomo

```bash
mihomo-cli stop
```

## 配置说明

### 基础配置

```toml
[daemon]
# 启用守护进程模式
enabled = true

# 工作目录（可选）
work_dir = "/var/lib/mihomo"

# 日志文件路径
log_file = "/var/log/mihomo/mihomo-daemon.log"

# 日志级别
log_level = "info"
```

### 日志轮转配置

```toml
[daemon]
# 日志文件最大大小
log_max_size = "100M"

# 保留的日志文件备份数量
log_max_backups = 10

# 日志文件最大保留天数
log_max_age = 30
```

### 自动重启配置

```toml
[daemon.auto_restart]
# 启用自动重启
enabled = true

# 最大重启次数
max_restarts = 5

# 重启延迟
restart_delay = "5s"
```

### 健康检查配置

```toml
[daemon.health_check]
# 启用健康检查
enabled = true

# 健康检查间隔
interval = "30s"

# 健康检查超时
timeout = "10s"
```

## 完整配置示例

```toml
[api]
address = "http://127.0.0.1:9090"
secret = ""

[mihomo]
enabled = true
executable = "/usr/local/bin/mihomo"
config_file = "/etc/mihomo/config.yaml"
auto_generate_secret = true
health_check_timeout = 5

[mihomo.api]
external_controller = "127.0.0.1:9090"

[mihomo.log]
level = "info"

[daemon]
enabled = true
work_dir = "/var/lib/mihomo"
log_file = "/var/log/mihomo/mihomo-daemon.log"
log_level = "info"
log_max_size = "100M"
log_max_backups = 10
log_max_age = 30

[daemon.auto_restart]
enabled = true
max_restarts = 5
restart_delay = "5s"

[daemon.health_check]
enabled = true
interval = "30s"
timeout = "10s"
```

## 平台特定说明

### Windows

在 Windows 上，守护进程模式使用 `CREATE_NEW_PROCESS_GROUP` 标志创建独立进程组：

```toml
[daemon]
enabled = true
log_file = "C:/Users/YourName/AppData/Local/mihomo/mihomo-daemon.log"
```

### Linux

在 Linux 上，守护进程模式使用 `setsid()` 和 `setpgid()` 系统调用：

```toml
[daemon]
enabled = true
log_file = "/var/log/mihomo/mihomo-daemon.log"
work_dir = "/var/lib/mihomo"
```

### macOS

在 macOS 上，守护进程模式与 Linux 类似，同时支持 launchd 集成：

```toml
[daemon]
enabled = true
log_file = "/Users/YourName/Library/Logs/mihomo/mihomo-daemon.log"
```

## 模式对比

### 守护进程模式 vs 传统模式

| 特性 | 守护进程模式 | 传统模式 |
|------|------------|---------|
| 终端关闭影响 | 不影响 | 可能停止 |
| 日志输出 | 文件 | 终端/缓冲区 |
| 进程组 | 独立 | 依赖父进程 |
| 适用场景 | 生产环境 | 开发调试 |
| 配置复杂度 | 中等 | 简单 |

### 切换模式

要切换模式，只需修改配置文件中的 `daemon.enabled` 选项：

```toml
# 启用守护进程模式
[daemon]
enabled = true

# 禁用守护进程模式（传统模式）
[daemon]
enabled = false
```

或完全删除 `[daemon]` 配置节。

## 故障排查

### 问题1：守护进程无法启动

**症状**：执行 `mihomo-cli start` 后立即退出

**解决方案**：
1. 检查日志文件路径是否存在和可写
2. 检查工作目录是否存在
3. 查看 `mihomo-cli status` 确认进程状态

### 问题2：日志文件未生成

**症状**：配置了日志文件但没有内容

**解决方案**：
1. 确认日志目录存在且有写权限
2. 检查磁盘空间是否充足
3. 验证日志级别设置

### 问题3：进程意外停止

**症状**：守护进程运行一段时间后自动停止

**解决方案**：
1. 启用健康检查和自动重启
2. 查看日志文件中的错误信息
3. 检查系统资源限制

### 问题4：无法停止守护进程

**症状**：`mihomo-cli stop` 命令失败

**解决方案**：
1. 使用强制关闭：`mihomo-cli stop -F`
2. 检查进程是否真的在运行
3. 手动清理 PID 文件

## 日志管理

### 查看日志

```bash
# 查看完整日志
cat /var/log/mihomo/mihomo-daemon.log

# 实时查看日志
tail -f /var/log/mihomo/mihomo-daemon.log

# 查看最后100行
tail -n 100 /var/log/mihomo/mihomo-daemon.log

# 搜索错误
grep ERROR /var/log/mihomo/mihomo-daemon.log
```

### 日志轮转

当日志文件达到 `log_max_size` 指定的大小时，会自动创建备份文件：

```
mihomo-daemon.log          # 当前日志
mihomo-daemon.log.1        # 备份1
mihomo-daemon.log.2        # 备份2
...
```

备份文件数量由 `log_max_backups` 控制，超过保留天数的文件会自动删除。

## 性能优化

### 减少日志输出

对于生产环境，可以降低日志级别：

```toml
[daemon]
log_level = "warning"
```

### 调整健康检查频率

减少健康检查频率可以降低系统负载：

```toml
[daemon.health_check]
interval = "1m"  # 每分钟检查一次
```

### 禁用自动重启

如果不需要自动重启功能，可以禁用：

```toml
[daemon.auto_restart]
enabled = false
```

## 安全建议

1. **日志文件权限**：确保日志文件只有授权用户可读
2. **工作目录**：使用受限的工作目录
3. **密钥管理**：不要在配置文件中硬编码密钥
4. **日志清理**：定期清理旧日志文件

## 最佳实践

1. **生产环境**：始终使用守护进程模式
2. **开发调试**：使用传统模式，方便查看实时输出
3. **日志管理**：配置合理的日志轮转策略
4. **监控告警**：集成监控系统监控守护进程状态
5. **定期检查**：定期检查日志文件和进程状态

## 迁移指南

### 从传统模式迁移到守护进程模式

1. **备份配置**：备份当前的配置文件
2. **添加守护进程配置**：按照示例配置添加 `[daemon]` 节
3. **测试启动**：使用 `mihomo-cli start` 测试启动
4. **验证功能**：验证所有功能正常工作
5. **监控日志**：查看日志文件确认运行状态

### 回滚到传统模式

如果遇到问题，可以随时回滚：

1. **停止守护进程**：`mihomo-cli stop`
2. **修改配置**：设置 `daemon.enabled = false`
3. **重新启动**：`mihomo-cli start`

## 常见问题

### Q: 守护进程模式和传统模式可以同时使用吗？

A: 不可以，只能选择一种模式。通过配置文件中的 `daemon.enabled` 控制。

### Q: 守护进程模式的性能开销大吗？

A: 性能开销很小，主要是日志文件 I/O 和健康检查的开销。

### Q: 可以在运行时切换模式吗？

A: 不可以，需要停止进程后修改配置再重新启动。

### Q: 守护进程模式支持多实例吗？

A: 当前版本不支持，每个配置只能运行一个实例。多实例支持将在未来版本中实现。

### Q: 日志文件可以放在哪里？

A: 可以放在任何有写权限的目录。建议放在标准的日志目录中。

## 参考资源

- [守护进程模式实现方案](./daemon-mode-implementation.md)
- [配置文件示例](./daemon-config-example.toml)
- [命令参考](../cmd/process.md)

---

**文档版本**: 1.0
**最后更新**: 2026-04-01
