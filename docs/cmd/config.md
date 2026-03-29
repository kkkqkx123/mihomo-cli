# 配置管理命令 (config, mihomo, backup)

配置管理命令用于管理 CLI 配置、Mihomo 配置和配置备份。

## CLI 配置管理 (config)

管理 CLI 工具的本地配置，包括初始化、查看和设置配置项。

### config init - 初始化配置文件

生成默认配置文件。

**语法：**
```bash
mihomo-cli config init
```

**选项：**
- `-f, --force` - 强制覆盖已存在的配置文件

**生成的文件：**
- `config.toml` - 统一配置文件（包含 CLI 和 Mihomo 内核管理配置）
- `secrets.json` - 敏感数据存储（权限 600）

**示例：**
```bash
# 初始化配置
mihomo-cli config init

# 强制覆盖
mihomo-cli config init --force
```

### config show - 显示当前配置

显示当前的 CLI 配置信息。

**语法：**
```bash
mihomo-cli config show
```

**显示内容：**
- CLI 客户端配置（API 地址、超时时间、代理测试 URL 等）
- Mihomo 内核配置（可执行文件、配置文件、日志级别等）
- 敏感数据（API 密钥、订阅数量等，密钥会脱敏显示）

**示例：**
```bash
mihomo-cli config show
```

### config set - 设置配置项

设置指定的配置项。

**语法：**
```bash
mihomo-cli config set <key> <value>
```

**支持的配置项：**
- `api.address` - API 地址
- `api.timeout` - API 超时时间（秒）
- `proxy.test_url` - 代理测试 URL
- `proxy.timeout` - 代理超时时间（毫秒）
- `proxy.concurrent` - 并发测试数
- `output.mode` - 输出模式（console/file）
- `output.file` - 输出文件路径
- `mihomo.enabled` - Mihomo 启用状态
- `mihomo.executable` - Mihomo 可执行文件路径
- `mihomo.config_file` - Mihomo 配置文件路径
- `mihomo.auto_generate_secret` - 自动生成密钥
- `mihomo.health_check_timeout` - 健康检查超时（秒）
- `mihomo.api.external_controller` - 外部控制器地址
- `mihomo.log.level` - 日志级别

**示例：**
```bash
# 设置 API 地址
mihomo-cli config set api.address http://127.0.0.1:9090

# 设置超时时间
mihomo-cli config set api.timeout 10

# 设置代理测试 URL
mihomo-cli config set proxy.test_url https://www.google.com/generate_204
```

## Mihomo 配置管理 (mihomo)

管理 Mihomo 服务的运行时配置，包括热更新、重载和编辑配置文件。

### mihomo patch - 热更新 Mihomo 配置

通过 API 热更新 Mihomo 运行时配置，无需重启服务。

**语法：**
```bash
mihomo-cli mihomo patch <key> <value>
```

**参数：**
- `key` - 配置键
- `value` - 配置值

**选项：**
- `-f, --file` - 从 YAML/JSON 文件读取配置更新

**支持的热更新配置项：**
- `mode` - 运行模式
- `allow-lan` - 允许局域网连接
- `log-level` - 日志级别
- `ipv6` - IPv6 支持
- `sniffing` - 流量嗅探
- `tcp-concurrent` - TCP 并发

**示例：**
```bash
# 热更新模式
mihomo-cli mihomo patch mode rule

# 热更新日志级别
mihomo-cli mihomo patch log-level debug

# 从文件更新
mihomo-cli mihomo patch --file config-patch.yaml
```

### mihomo reload - 重载 Mihomo 配置文件

重新加载完整的 Mihomo 配置文件。

**语法：**
```bash
mihomo-cli mihomo reload
```

**选项：**
- `-p, --path` - 配置文件路径
- `-f, --force` - 强制重载，忽略部分错误

**示例：**
```bash
# 重载当前配置
mihomo-cli mihomo reload

# 重载指定配置文件
mihomo-cli mihomo reload --path /path/to/config.yaml

# 强制重载
mihomo-cli mihomo reload --force
```

### mihomo edit - 编辑 Mihomo 配置文件

编辑 Mihomo 配置文件并自动重载。

**语法：**
```bash
mihomo-cli mihomo edit <key> <value>
```

**参数：**
- `key` - 配置键
- `value` - 配置值

**选项：**
- `--no-reload` - 仅修改文件，不触发重载
- `-m, --mihomo-config` - Mihomo 配置文件路径

**示例：**
```bash
# 编辑配置并重载
mihomo-cli mihomo edit mode rule

# 编辑配置但不重载
mihomo-cli mihomo edit tun.enable true --no-reload
```

## 配置备份管理 (backup)

管理 Mihomo 配置文件的备份，包括创建、查看、恢复和删除备份。

### backup create - 创建配置备份

手动创建 Mihomo 配置文件的备份。

**语法：**
```bash
mihomo-cli backup create
```

**选项：**
- `-p, --path` - Mihomo 配置文件路径
- `-n, --note` - 备注信息

**示例：**
```bash
# 创建备份
mihomo-cli backup create

# 创建带备注的备份
mihomo-cli backup create -n "before-update"

# 指定配置文件路径
mihomo-cli backup create -p /path/to/config.yaml -n "manual-backup"
```

### backup list - 列出所有备份

列出 Mihomo 配置文件的所有备份。

**语法：**
```bash
mihomo-cli backup list
```

**选项：**
- `-p, --path` - Mihomo 配置文件路径

**示例：**
```bash
mihomo-cli backup list
mihomo-cli backup list -p /path/to/config.yaml
```

### backup restore - 恢复配置备份

从指定的备份文件恢复 Mihomo 配置。恢复前会自动备份当前配置。

**语法：**
```bash
mihomo-cli backup restore <备份文件|序号>
```

**参数：**
- 备份文件路径或备份序号

**选项：**
- `-p, --path` - Mihomo 配置文件路径
- `--no-reload` - 恢复后不自动重载配置

**示例：**
```bash
# 使用序号恢复
mihomo-cli backup restore 1

# 使用文件路径恢复
mihomo-cli backup restore /path/to/backup.yaml

# 恢复但不重载
mihomo-cli backup restore 1 --no-reload
```

### backup delete - 删除配置备份

删除指定的 Mihomo 配置备份文件。

**语法：**
```bash
mihomo-cli backup delete [备份文件|序号]
```

**参数：**
- 备份文件路径或备份序号（可选）

**选项：**
- `-p, --path` - Mihomo 配置文件路径
- `--all` - 删除所有备份
- `-k, --keep` - 保留最近 N 个备份，删除其余
- `--older-than` - 删除超过指定天数的备份

**示例：**
```bash
# 删除单个备份（序号）
mihomo-cli backup delete 1

# 删除单个备份（文件路径）
mihomo-cli backup delete /path/to/backup.yaml

# 删除所有备份
mihomo-cli backup delete --all

# 保留最近 5 个备份
mihomo-cli backup delete --keep 5

# 删除超过 30 天的备份
mihomo-cli backup delete --older-than 30
```

### backup prune - 清理旧备份

按策略清理旧的 Mihomo 配置备份文件。

**语法：**
```bash
mihomo-cli backup prune
```

**选项：**
- `-p, --path` - Mihomo 配置文件路径
- `-k, --keep` - 保留最近 N 个备份（默认 10）
- `--older-than` - 删除超过指定天数的备份
- `--dry-run` - 仅显示将被删除的备份，不实际删除

**示例：**
```bash
# 清理旧备份
mihomo-cli backup prune

# 保留最近 5 个备份
mihomo-cli backup prune --keep 5

# 删除超过 30 天的备份
mihomo-cli backup prune --older-than 30

# 预览将要删除的备份
mihomo-cli backup prune --dry-run
```

## 注意事项

1. 配置文件默认存储在 `~/.config/.mihomo-cli/` 目录
2. 敏感数据存储在 `secrets.json` 文件中，权限设置为 600
3. 热更新配置不会修改配置文件，只影响运行时配置
4. 重载配置会先自动创建备份
5. 备份文件存储在统一的备份目录中
6. 删除和清理操作不可逆，请谨慎操作
