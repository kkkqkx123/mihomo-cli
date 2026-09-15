# 系统崩溃后恢复启动指南

## 问题描述

系统崩溃后，`mihomo-cli start` 可能会失败，错误信息类似：

```
Automatic cleanup failed: cleanup completed with 1 errors: [sysproxy cleanup failed: failed to remove proxy from /etc/environment: open /etc/environment: permission denied]
✗ 启动 Mihomo 失败: failed to cleanup residual configuration
```

这是因为系统代理配置存储在 `/etc/environment` 中，需要 root 权限才能修改。

## 解决方案

### 方案1：使用 sudo 清理后启动（推荐）

```bash
# 清理系统代理配置
sudo mihomo-cli sysproxy set off

# 正常启动
mihomo-cli start
```

### 方案2：使用系统清理命令

```bash
# 清理所有残留配置
sudo mihomo-cli system cleanup

# 正常启动
mihomo-cli start
```

### 方案3：强制启动（跳过清理）

```bash
# 强制启动，跳过清理检查
mihomo-cli start --force
```

**注意**：强制启动会跳过清理检查，可能导致系统代理配置仍然残留。建议后续手动清理。

### 方案4：手动编辑配置文件

如果上述命令都无法使用，可以手动编辑配置文件：

```bash
# 编辑 /etc/environment
sudo nano /etc/environment

# 删除以下行（如果存在）：
# HTTP_PROXY=...
# HTTPS_PROXY=...
# http_proxy=...
# https_proxy=...
# NO_PROXY=...
# no_proxy=...

# 保存并退出

# 启动 mihomo
mihomo-cli start
```

## 详细步骤

### 步骤1：诊断问题

首先确认问题原因：

```bash
# 检查系统代理状态
mihomo-cli sysproxy get

# 检查 /etc/environment 是否包含代理配置
cat /etc/environment | grep -i proxy
```

### 步骤2：选择解决方案

根据你的权限情况选择方案：

**有 sudo 权限**：
```bash
sudo mihomo-cli sysproxy set off
mihomo-cli start
```

**无 sudo 权限**：
```bash
mihomo-cli start --force
```

### 步骤3：验证启动成功

```bash
# 检查运行状态
mihomo-cli status

# 应该显示：
# 状态: 运行中
# PID: xxxxx
# API 地址: http://...
```

## 常见问题

### Q1: 为什么会出现这个问题？

系统崩溃时，Mihomo 可能正在运行并设置了系统代理。崩溃后，系统代理配置仍然残留在 `/etc/environment` 中。下次启动时，Mihomo 会检测到这个残留配置并尝试清理，但由于权限不足而失败。

### Q2: 强制启动有什么风险？

强制启动会跳过清理检查，主要风险：
- 系统代理配置可能仍然指向已停止的 Mihomo
- 其他程序可能无法正常访问网络

建议在强制启动后，有权限时手动清理。

### Q3: 如何避免这个问题？

1. **正常停止**：使用 `mihomo-cli stop` 正常停止，会自动清理
2. **配置自动清理**：在 config.toml 中配置自动清理选项
3. **使用 systemd**：通过 systemd 管理 Mihomo，可以配置停止时自动清理

### Q4: 清理后需要重启系统吗？

不需要。清理操作会立即生效，但当前终端会话可能不会立即更新环境变量。建议：
- 开启新的终端会话
- 或者运行 `source /etc/environment` 更新当前会话

## 技术细节

### 系统代理配置位置

- **主要配置**：`/etc/environment.d/proxy.conf`（systemd 环境）
- **备用配置**：`/etc/environment`（通用环境）

### 配置内容示例

```bash
HTTP_PROXY=http://127.0.0.1:7890
HTTPS_PROXY=http://127.0.0.1:7890
http_proxy=http://127.0.0.1:7890
https_proxy=http://127.0.0.1:7890
NO_PROXY=localhost,127.0.0.1
no_proxy=localhost,127.0.0.1
```

### 权限要求

- 修改 `/etc/environment` 需要 root 权限
- 使用 `sudo` 或以 root 用户运行

## 相关命令

```bash
# 查看系统代理状态
mihomo-cli sysproxy get

# 启用系统代理
mihomo-cli sysproxy set on

# 禁用系统代理（需要 sudo）
sudo mihomo-cli sysproxy set off

# 系统清理（需要 sudo）
sudo mihomo-cli system cleanup

# 强制启动
mihomo-cli start --force
```

## 获取帮助

如果以上方法都无法解决问题：

1. 查看详细日志：`mihomo-cli logs`
2. 检查系统状态：`mihomo-cli system status`
3. 提交 Issue：https://github.com/kkkqkx123/mihomo-cli/issues
