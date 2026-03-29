# 服务管理命令 (service)

服务管理命令用于管理 Mihomo 系统服务的安装、启动、停止和卸载。

**注意：** 此命令仅支持 Windows 系统。

## 命令列表

### service install - 安装服务

将 Mihomo 安装为系统服务。

**语法：**
```bash
mihomo-cli service install
```

**示例：**
```bash
mihomo-cli service install
```

### service uninstall - 卸载服务

卸载 Mihomo 系统服务。

**语法：**
```bash
mihomo-cli service uninstall
```

**示例：**
```bash
mihomo-cli service uninstall
```

### service start - 启动服务

启动 Mihomo 系统服务。

**语法：**
```bash
mihomo-cli service start
```

**选项：**
- `-a, --async` - 异步模式，立即返回不等待

**示例：**
```bash
# 同步启动
mihomo-cli service start

# 异步启动
mihomo-cli service start --async
```

### service stop - 停止服务

停止 Mihomo 系统服务。

**语法：**
```bash
mihomo-cli service stop
```

**选项：**
- `-a, --async` - 异步模式，立即返回不等待

**示例：**
```bash
# 同步停止
mihomo-cli service stop

# 异步停止
mihomo-cli service stop --async
```

### service status - 查询服务状态

查询 Mihomo 系统服务的运行状态。

**语法：**
```bash
mihomo-cli service status
```

**示例：**
```bash
mihomo-cli service status
```

## 服务状态说明

服务状态可能为以下几种：
- **运行中** - 服务正在运行
- **已停止** - 服务已停止
- **未安装** - 服务未安装

## 启动模式说明

### 同步模式（默认）
- 命令会等待服务启动或停止完成
- 可以确认操作是否成功
- 适合需要确认操作结果的场景

### 异步模式
- 命令会立即返回，不等待操作完成
- 需要使用 `service status` 查询实际状态
- 适合批量操作或自动化脚本

## 注意事项

1. 安装和卸载服务需要管理员权限
2. 服务安装后会自动配置为开机自启动
3. 卸载服务前建议先停止服务
4. 异步模式下，建议使用 `service status` 确认操作结果
5. 服务名称和显示名称会在安装时显示
