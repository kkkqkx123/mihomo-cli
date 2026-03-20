# Windows 系统代理使用指南

本文档详细说明如何在 Windows 系统中使用 mihomo-cli 工具管理系统代理，使浏览器和其他应用程序能够通过 Mihomo 代理访问网络。

## 重要说明

**系统代理操作与 Mihomo 内核的关系：**

- **系统代理操作独立于 Mihomo 内核**：`sysproxy` 命令直接调用 Windows API（通过修改注册表），**不需要** Mihomo 内核运行即可执行
- **两者配合使用**：虽然操作独立，但实际使用时需要配合：
  1. 启动 Mihomo 内核（监听代理端口，如 7890）
  2. 启用系统代理（告诉系统使用 127.0.0.1:7890 作为代理）
  3. 浏览器流量才会经过 Mihomo 代理服务器

**操作独立性示例：**
```bash
# 即使 Mihomo 未运行，也可以查询/修改系统代理
.\mihomo-cli.exe sysproxy get      # ✓ 可以执行
.\mihomo-cli.exe sysproxy set on   # ✓ 可以执行（需要管理员权限）

# 但只有 Mihomo 运行时，浏览器流量才会真正经过代理
```

## 一、问题背景

### 1.1 为什么需要系统代理？

Mihomo 内核启动后，会在本地监听代理端口（默认 7890），但仅启动内核是不够的：

| 应用类型 | 代理配置方式 | 说明 |
|---------|------------|------|
| **命令行工具** (curl, wget) | 环境变量 (HTTP_PROXY) | 设置环境变量后生效 |
| **浏览器** (Chrome, Edge, Firefox) | **Windows 系统代理设置** | 需要修改注册表 |
| **其他应用** | 系统代理或内置代理设置 | 各不相同 |

**关键点**：浏览器默认读取 Windows 系统代理设置，**必须启用系统代理才能让浏览器流量经过 Mihomo**。

### 1.2 常见误解

**错误理解：**
```
启动 Mihomo → 浏览器自动代理 ❌
```

**正确流程：**
```
启动 Mihomo → 启用系统代理 → 浏览器流量经过 Mihomo ✓
```

## 二、工作原理

### 2.1 系统代理机制

Windows 系统代理通过修改注册表实现：

```bash
注册表路径：HKEY_CURRENT_USER\Software\Microsoft\Windows\CurrentVersion\Internet Settings

关键字段：
- ProxyEnable: 1=启用，0=禁用
- ProxyServer: 代理服务器地址（如 "127.0.0.1:7890"）
- ProxyOverride: 绕过代理的地址列表
```

### 2.2 流量路由流程

```
浏览器访问 google.com
    ↓
读取 Windows 系统代理设置
    ↓
发现代理服务器: 127.0.0.1:7890
    ↓
发送请求到 127.0.0.1:7890
    ↓
Mihomo 接收请求
    ↓
根据规则匹配:
  DOMAIN-SUFFIX,google.com,PROXY ✓
    ↓
通过 PROXY 代理组转发到真实服务器
    ↓
返回响应给浏览器
```

### 2.3 绕过列表

绕过列表中的地址不经过代理，直接访问：

```
绕过列表默认值：
localhost;127.*;10.*;172.16.*;172.31.*;192.168.*

含义：
- localhost: 本地主机
- 127.*: 回环地址
- 10.*: A 类私有网络
- 172.16.*-172.31.*: B 类私有网络
- 192.168.*: C 类私有网络
```

**为什么要绕过？**
- 内网地址不应该经过代理
- 减少不必要的代理开销
- 避免代理服务器故障影响内网访问

## 三、使用方法

### 3.1 权限要求

**重要：启用/禁用系统代理需要管理员权限**

```powershell
# 检查是否有管理员权限
# 方法 1：查看命令提示符标题
# 方法 2：运行 whoami /groups | find "S-1-16-12288"

# 以管理员身份运行 PowerShell 或 CMD
# 右键点击 PowerShell → "以管理员身份运行"
```

### 3.2 查询系统代理状态

```bash
.\mihomo-cli.exe sysproxy get
```

**输出示例（已禁用）：**
```
系统代理状态:
  状态: 已禁用
```

**输出示例（已启用）：**
```
系统代理状态:
  状态: 已启用
  代理服务器: 127.0.0.1:7890
  绕过列表: localhost;127.*;10.*;172.16.*;172.31.*;192.168.*
```

### 3.3 启用系统代理

```bash
# 基础用法（使用默认配置）
.\mihomo-cli.exe sysproxy set on

# 指定代理服务器
.\mihomo-cli.exe sysproxy set on --server 127.0.0.1:7890

# 自定义绕过列表
.\mihomo-cli.exe sysproxy set on --server 127.0.0.1:7890 --bypass "localhost;127.*;10.*;192.168.*"
```

**输出示例：**
```
系统代理已启用
代理服务器: 127.0.0.1:7890
绕过列表: localhost;127.*;10.*;172.16.*;172.31.*;192.168.*
```

### 3.4 禁用系统代理

```bash
.\mihomo-cli.exe sysproxy set off
```

**输出示例：**
```
系统代理已禁用
```

## 四、配置说明

### 4.1 命令参数

| 参数 | 说明 | 默认值 |
|------|------|--------|
| `on/off` | 启用或禁用系统代理 | - |
| `--server` | 代理服务器地址 | `127.0.0.1:7890` |
| `--bypass` | 绕过代理的地址列表 | `localhost;127.*;10.*;172.16.*;172.31.*;192.168.*` |

### 4.2 代理服务器格式

支持以下格式：

```
# HTTP 代理
127.0.0.1:7890

# SOCKS5 代理（需要在地址前加 socks5://）
socks5://127.0.0.1:7891

# HTTPS 代理
https://127.0.0.1:7890
```

**Mihomo 默认配置：**
```yaml
mixed-port: 7890  # HTTP + SOCKS5 混合端口
```

### 4.3 绕过列表格式

使用分号 `;` 分隔多个地址模式：

```
# 基础格式
localhost;127.*;10.*;172.16.*;172.31.*;192.168.*

# 添加自定义地址
localhost;127.*;10.*;192.168.*;*.local;*.intranet.company.com

# 注意事项：
# 1. 使用分号分隔
# 2. 支持 * 通配符
# 3. 不支持逗号（逗号在不同语言中含义不同）
```

## 五、完整使用流程

### 5.1 启动流程

```bash
# 1. 检查 Mihomo 内核状态
.\mihomo-cli.exe status

# 2. 启动 Mihomo 内核（如果未启动）
.\mihomo-cli.exe start

# 3. 验证内核运行状态
.\mihomo-cli.exe status

# 4. 以管理员身份运行，启用系统代理
.\mihomo-cli.exe sysproxy set on

# 5. 验证系统代理状态
.\mihomo-cli.exe sysproxy get

# 6. 测试代理功能（在浏览器中访问 google.com）
```

### 5.2 停止流程

```bash
# 1. 禁用系统代理
.\mihomo-cli.exe sysproxy set off

# 2. 停止 Mihomo 内核
.\mihomo-cli.exe stop

# 3. 验证内核已停止
.\mihomo-cli.exe status
```

### 5.3 一键脚本

创建启动脚本 `start-with-proxy.bat`：

```batch
@echo off
echo =====================================
echo   启动 Mihomo 并启用系统代理
echo =====================================
echo.

echo [1/3] 检查 Mihomo 状态...
.\mihomo-cli.exe status
echo.

echo [2/3] 启动 Mihomo 内核...
.\mihomo-cli.exe start
echo.

echo [3/3] 启用系统代理（需要管理员权限）...
.\mihomo-cli.exe sysproxy set on
echo.

echo =====================================
echo   启动完成！
echo =====================================
echo.
echo 系统代理状态：
.\mihomo-cli.exe sysproxy get
echo.
pause
```

创建停止脚本 `stop-with-proxy.bat`：

```batch
@echo off
echo =====================================
echo   禁用系统代理并停止 Mihomo
echo =====================================
echo.

echo [1/2] 禁用系统代理...
.\mihomo-cli.exe sysproxy set off
echo.

echo [2/2] 停止 Mihomo 内核...
.\mihomo-cli.exe stop
echo.

echo =====================================
echo   停止完成！
echo =====================================
echo.
pause
```

## 六、验证系统代理

### 6.1 通过浏览器验证

1. **打开浏览器设置**
   - Chrome: 设置 → 系统 → 打开代理设置
   - Edge: 设置 → 系统 → 打开代理设置

2. **查看代理状态**
   - 应该看到 "使用代理服务器" 已启用
   - 地址：127.0.0.1，端口：7890

3. **测试访问**
   - 访问 `https://www.google.com` → 应该通过代理
   - 访问 `http://192.168.1.1` → 应该直连（绕过列表）

### 6.2 通过命令行验证

```bash
# 使用 curl 测试（需要设置环境变量）
set HTTP_PROXY=http://127.0.0.1:7890
set HTTPS_PROXY=http://127.0.0.1:7890

curl -I https://www.google.com

# 查看响应头，确认请求经过代理
```

### 6.3 查看代理日志

在 Mihomo 配置文件中设置日志级别为 debug：

```yaml
log-level: debug
```

查看日志输出，确认流量经过代理。

## 七、故障排查

### 7.1 权限错误

**错误信息：**
```
Error: this operation requires administrator privileges
```

**解决方法：**
1. 以管理员身份运行 PowerShell 或 CMD
2. 右键点击 PowerShell → "以管理员身份运行"

### 7.2 系统代理未生效

**问题：** 启用系统代理后，浏览器仍然直连

**排查步骤：**

1. **检查系统代理状态**
   ```bash
   .\mihomo-cli.exe sysproxy get
   ```

2. **检查 Mihomo 内核是否运行**
   ```bash
   .\mihomo-cli.exe status
   ```

3. **检查端口是否监听**
   ```bash
   netstat -ano | findstr :7890
   ```

4. **检查浏览器代理设置**
   - 打开浏览器代理设置页面
   - 确认代理服务器地址正确

5. **清除浏览器缓存**
   - 清除 DNS 缓存
   - 清除浏览器缓存

### 7.3 部分网站无法访问

**可能原因：**

1. **规则配置问题**
   ```bash
   # 查看规则列表
   .\mihomo-cli.exe rule list

   # 检查是否有匹配的规则
   ```

2. **DNS 问题**
   - 检查 DNS 配置
   - 尝试切换 DNS 服务器

3. **代理节点问题**
   ```bash
   # 测试代理节点延迟
   .\mihomo-cli.exe proxy test PROXY

   # 切换到其他节点
   .\mihomo-cli.exe proxy switch PROXY 节点名称
   ```

### 7.4 绕过列表不生效

**问题：** 内网地址仍然经过代理

**解决方法：**

1. **检查绕过列表格式**
   ```bash
   .\mihomo-cli.exe sysproxy get
   ```

2. **确认格式正确**
   - 使用分号 `;` 分隔
   - 不要使用逗号 `,`

3. **重启浏览器**
   - 浏览器可能缓存了旧的代理设置

### 7.5 系统代理自动禁用

**可能原因：**

1. **安全软件干扰**
   - 某些杀毒软件会禁用系统代理
   - 添加例外或关闭相关功能

2. **组策略限制**
   - 企业环境可能有组策略限制
   - 联系 IT 管理员

3. **其他软件冲突**
   - VPN 软件
   - 其他代理工具

## 八、高级配置

### 8.1 分段代理

为不同协议设置不同的代理：

```bash
# HTTP 和 HTTPS 使用不同端口
.\mihomo-cli.exe sysproxy set on --server "http=127.0.0.1:7890;https=127.0.0.1:7890"

# SOCKS5 代理
.\mihomo-cli.exe sysproxy set on --server "socks=127.0.0.1:7891"
```

### 8.2 条件启用

根据网络环境决定是否启用系统代理：

```batch
@echo off
:: 检查网络环境
ping -n 1 8.8.8.8 >nul 2>&1

if %errorlevel% equ 0 (
    echo 检测到互联网连接，启用系统代理
    .\mihomo-cli.exe sysproxy set on
) else (
    echo 未检测到互联网连接，禁用系统代理
    .\mihomo-cli.exe sysproxy set off
)
```

### 8.3 定时更新

结合 Windows 任务计划程序，定期更新代理：

```batch
:: update-proxy.bat
@echo off
echo 更新代理订阅...
.\mihomo-cli.exe sub update
echo 更新完成
```

创建任务计划程序，每小时运行一次。

## 九、注意事项

### 9.1 安全性

1. **不要在生产环境使用默认配置**
   - 修改默认端口
   - 设置强密码

2. **限制访问范围**
   - 不要使用 `allow-lan: true`（除非必要）
   - 绑定到特定地址

3. **定期更新**
   - 更新 Mihomo 内核
   - 更新代理节点

### 9.2 性能优化

1. **合理设置绕过列表**
   - 避免不必要的代理流量
   - 减少代理服务器负载

2. **选择合适的代理模式**
   - Rule 模式：根据规则分流（推荐）
   - Global 模式：全部流量走代理
   - Direct 模式：全部流量直连

3. **启用连接复用**
   ```yaml
   tcp-concurrent: true
   ```

### 9.3 兼容性

1. **某些应用不使用系统代理**
   - 需要在应用内部单独配置
   - 例如：某些游戏、特定软件

2. **IPv6 支持**
   ```yaml
   ipv6: true  # 如果需要 IPv6 支持
   ```

3. **代理服务器地址**
   - 使用 `127.0.0.1` 而不是 `localhost`
   - 某些应用可能无法解析 `localhost`

## 十、总结

### 10.1 关键要点

1. **系统代理 ≠ Mihomo 内核**
   - 内核运行 ≠ 代理生效
   - 必须启用系统代理

2. **浏览器需要系统代理**
   - 浏览器读取 Windows 系统代理设置
   - 不受环境变量影响

3. **管理员权限**
   - 启用/禁用系统代理需要管理员权限

4. **绕过列表很重要**
   - 内网地址不应该经过代理
   - 提高性能和稳定性

### 10.2 常用命令速查

```bash
# 查询系统代理状态
.\mihomo-cli.exe sysproxy get

# 启用系统代理
.\mihomo-cli.exe sysproxy set on

# 禁用系统代理
.\mihomo-cli.exe sysproxy set off

# 查询 Mihomo 状态
.\mihomo-cli.exe status

# 启动 Mihomo
.\mihomo-cli.exe start

# 停止 Mihomo
.\mihomo-cli.exe stop
```

### 10.3 推荐工作流程

```bash
# 日常使用
1. .\mihomo-cli.exe start          # 启动内核
2. .\mihomo-cli.exe sysproxy set on # 启用系统代理（管理员）
3. 正常使用浏览器...

# 关闭时
1. .\mihomo-cli.exe sysproxy set off # 禁用系统代理（管理员）
2. .\mihomo-cli.exe stop            # 停止内核
```

## 十一、相关资源

- **Mihomo 官方文档**: https://wiki.metacubex.one/
- **Mihomo API 文档**: `docs/spec/mihono-api.md`
- **配置文件示例**: `test-mihomo-config.yaml`
- **订阅导入指南**: `docs/subscription-import-guide.md`

---

**文档版本**: 1.0
**最后更新**: 2026-03-18
**适用版本**: mihomo-cli v1.0+