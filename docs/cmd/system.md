# 系统管理命令 (system, sysproxy, diagnose, recovery, operation)

系统管理命令用于管理系统配置，包括系统代理、TUN 设备、路由表等。

## 系统配置管理 (system)

管理系统配置，包括系统代理、TUN 设备、路由表等。

### system status - 查询系统配置状态

查询当前系统配置状态，包括系统代理、TUN 设备、路由表等。

**语法：**
```bash
mihomo-cli system status
```

**示例：**
```bash
mihomo-cli system status
```

### system cleanup - 清理系统配置

清理系统配置残留，包括系统代理、TUN 设备、路由表等。

**语法：**
```bash
mihomo-cli system cleanup
```

**选项：**
- `--sysproxy` - 清理系统代理（默认 true）
- `--tun` - 清理 TUN 设备（默认 true）
- `--route` - 清理路由表（默认 true）

**示例：**
```bash
# 清理所有配置
mihomo-cli system cleanup

# 只清理路由表
mihomo-cli system cleanup --route
```

### system validate - 验证系统配置

验证系统配置是否正常，检测是否有残留配置。

**语法：**
```bash
mihomo-cli system validate
```

**示例：**
```bash
mihomo-cli system validate
```

### system fix - 修复系统配置问题

自动检测并修复系统配置问题，包括路由残留、冲突等。

**语法：**
```bash
mihomo-cli system fix
```

**示例：**
```bash
mihomo-cli system fix
```

### system snapshot create - 创建配置快照

创建当前系统配置的快照。

**语法：**
```bash
mihomo-cli system snapshot create
```

**选项：**
- `-n, --note` - 快照备注

**示例：**
```bash
mihomo-cli system snapshot create
mihomo-cli system snapshot create -n "before-update"
```

### system snapshot list - 列出所有快照

列出所有系统配置快照。

**语法：**
```bash
mihomo-cli system snapshot list
```

**示例：**
```bash
mihomo-cli system snapshot list
```

### system snapshot restore - 恢复配置快照

恢复指定的系统配置快照。

**语法：**
```bash
mihomo-cli system snapshot restore <snapshot-id>
```

**参数：**
- `snapshot-id` - 快照 ID

**示例：**
```bash
mihomo-cli system snapshot restore abc123
```

### system snapshot delete - 删除配置快照

删除指定的系统配置快照。

**语法：**
```bash
mihomo-cli system snapshot delete <snapshot-id>
```

**参数：**
- `snapshot-id` - 快照 ID

**示例：**
```bash
mihomo-cli system snapshot delete abc123
```

## 系统代理管理 (sysproxy)

管理系统代理设置。

**注意：** 此命令仅支持 Windows 系统。

### sysproxy get - 查询系统代理状态

查询当前系统代理的状态。

**语法：**
```bash
mihomo-cli sysproxy get
```

**示例：**
```bash
mihomo-cli sysproxy get
```

### sysproxy set - 设置系统代理

开启或关闭系统代理。

**语法：**
```bash
mihomo-cli sysproxy set <on|off>
```

**参数：**
- `on` - 启用系统代理
- `off` - 禁用系统代理

**选项：**
- `--server` - 代理服务器地址（默认 127.0.0.1:7890）
- `--bypass` - 绕过代理的地址列表（默认 localhost;127.*;10.*;172.16.*;172.31.*;192.168.*）

**示例：**
```bash
# 启用系统代理
mihomo-cli sysproxy set on

# 禁用系统代理
mihomo-cli sysproxy set off

# 自定义代理服务器
mihomo-cli sysproxy set on --server 127.0.0.1:7890
```

## 系统诊断 (diagnose)

诊断系统问题，包括路由、网络等。

### diagnose route - 诊断路由问题

诊断路由表问题，包括残留路由和冲突。

**语法：**
```bash
mihomo-cli diagnose route
```

**选项：**
- `-f, --fix` - 自动修复问题
- `-o, --output` - 输出格式（table/json）

**示例：**
```bash
mihomo-cli diagnose route
mihomo-cli diagnose route --fix
```

### diagnose network - 诊断网络问题

诊断网络问题，包括路由和连通性。

**语法：**
```bash
mihomo-cli diagnose network
```

**选项：**
- `-f, --fix` - 自动修复问题
- `-o, --output` - 输出格式（table/json）

**示例：**
```bash
mihomo-cli diagnose network
mihomo-cli diagnose network --fix
```

## 自动恢复管理 (recovery)

管理系统自动恢复功能，包括问题检测和自动修复。

### recovery detect - 检测问题

检测系统配置问题。

**语法：**
```bash
mihomo-cli recovery detect
```

**示例：**
```bash
mihomo-cli recovery detect
```

### recovery execute - 执行恢复

执行系统配置恢复。

**语法：**
```bash
mihomo-cli recovery execute
```

**选项：**
- `-a, --auto` - 仅自动恢复可自动处理的问题
- `-c, --component` - 指定组件（sysproxy, tun, route）
- `-F, --force` - 强制执行高风险操作（跳过确认）
- `-p, --problem` - 指定要修复的问题类型

**支持的问题类型：**
- `config-residual` - 配置残留
- `process-abnormal` - 进程异常
- `config-inconsistent` - 配置不一致
- `port-conflict` - 端口冲突
- `permission-denied` - 权限不足

**示例：**
```bash
# 自动恢复
mihomo-cli recovery execute --auto

# 强制执行所有修复
mihomo-cli recovery execute --force

# 修复指定类型的问题
mihomo-cli recovery execute --problem config-residual
```

### recovery status - 查询恢复状态

查询自动恢复的状态和配置。

**语法：**
```bash
mihomo-cli recovery status
```

**示例：**
```bash
mihomo-cli recovery status
```

## 操作记录管理 (operation)

管理系统配置操作记录，包括查询、清理等。

### operation query - 查询操作记录

查询系统配置操作记录。

**语法：**
```bash
mihomo-cli operation query
```

**选项：**
- `-c, --component` - 过滤组件（sysproxy, tun, route, snapshot）
- `-l, --limit` - 限制返回数量（默认 20）
- `--since` - 起始时间（格式：2006-01-02）

**示例：**
```bash
# 查询所有操作记录
mihomo-cli operation query

# 按组件过滤
mihomo-cli operation query --component sysproxy

# 限制返回数量
mihomo-cli operation query --limit 10

# 按时间过滤
mihomo-cli operation query --since 2024-01-01
```

### operation clear - 清空操作记录

清空所有操作记录。

**语法：**
```bash
mihomo-cli operation clear
```

**示例：**
```bash
mihomo-cli operation clear
```

### operation prune - 清理旧操作记录

清理指定时间之前的操作记录。

**语法：**
```bash
mihomo-cli operation prune
```

**选项：**
- `--before` - 清理此时间之前的记录（格式：2006-01-02）

**示例：**
```bash
mihomo-cli operation prune --before 2024-01-01
```

## 注意事项

1. 系统代理管理仅支持 Windows 系统
2. 清理系统配置需要管理员权限
3. 恢复配置快照需要管理员权限
4. 执行恢复操作需要管理员权限
5. 自动恢复功能的高风险操作默认跳过，需要使用 `--force` 参数强制执行
6. 操作记录会记录所有系统配置的修改操作
