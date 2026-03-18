# 代理节点选择和启动规则代理验证指南

本文档记录了完整的代理节点选择、切换和验证流程，便于后续复现和参考。

## 环境准备

### 1. 配置文件

确保 `config.toml` 配置正确：

```toml
[api]
address = "http://127.0.0.1:9090"
secret = ""
timeout = 10

[mihomo]
enabled = true
executable = "E:\\server\\mihomo\\mihomo-windows-amd64.exe"
config_file = "E:\\project\\mihomo-go\\test-mihomo-config.yaml"
auto_generate_secret = true
```

确保 `test-mihomo-config.yaml` 配置正确：

```yaml
mixed-port: 7890
allow-lan: true
mode: rule
external-controller: 127.0.0.1:9090

proxy-providers:
  my-subscription:
    type: http
    url: "https://your-subscription-url"
    interval: 3600
    path: ./profiles/my-subscription.yaml

proxy-groups:
  - name: PROXY
    type: select
    use:
      - my-subscription

  - name: AUTO
    type: url-test
    use:
      - my-subscription

rules:
  - MATCH,PROXY
```

## 操作流程

### 步骤 1：启动 Mihomo 内核

```bash
.\mihomo-cli.exe start
```

**输出示例：**
```
=====================================
  Mihomo 内核已启动
=====================================
API 地址: http://127.0.0.1:9090
密钥: f08d7c4e3cef7cc4b2378034a5f5b895f7e291853b3e03f49d8fee41ed480170

提示: 内核已在后台运行
使用以下命令管理：
  mihomo-cli status  - 查询运行状态
  mihomo-cli stop    - 停止内核
=====================================
```

**说明：**
- 内核会在后台运行
- 自动生成随机密钥用于 API 认证
- 记录密钥用于后续 API 调用

### 步骤 2：配置 API 密钥

使用启动时生成的密钥配置 CLI 工具：

```bash
.\mihomo-cli.exe config set api.secret f08d7c4e3cef7cc4b2378034a5f5b895f7e291853b3e03f49d8fee41ed480170
```

**输出示例：**
```
配置已更新: api.secret = ****
```

### 步骤 3：更新订阅获取代理节点

```bash
.\mihomo-cli.exe sub update
```

**输出示例：**
```
找到 2 个代理提供者，开始更新...

正在更新 default (Compatible)...
  ✓ 更新成功
正在更新 my-subscription (HTTP)...
  ✓ 更新成功

更新完成: 成功 2 个，失败 0 个
```

**说明：**
- 更新所有代理提供者
- 代理节点会持久化保存到 `~/.config/mihomo/profiles/` 目录
- 支持多种订阅格式（YAML、V2Ray 链接、Base64 编码）

### 步骤 4：列出可用的代理节点

```bash
.\mihomo-cli.exe proxy list
```

**输出示例：**
```
┌─────────────────────────┬────────────┬────────────────────┬────────┬──────┬──────┐
│          名称           │    类型    │        当前        │ 节点数 │ 延迟 │ 状态 │
├─────────────────────────┼────────────┼────────────────────┼────────┼──────┼──────┤
│ 美国LA-优化3-GPT        │ Vmess      │ -                  │ -      │ -    │ ✓    │
│ AUTO                    │ URLTest    │ 香港-优化-Gemini   │ 30     │ -    │ ✓    │
│ DIRECT                  │ Direct     │ -                  │ -      │ -    │ ✓    │
│ PROXY                   │ Selector   │ 剩余流量：96.31 GB │ 30     │ -    │ ✓    │
│ 香港-优化-Gemini        │ Vmess      │ -                  │ -      │ -    │ ✓    │
│ ...
└─────────────────────────┴────────────┴────────────────────┴────────┴──────┴──────┘
```

**重要代理组说明：**
- **PROXY**：Selector 类型，手动选择节点组
- **AUTO**：URLTest 类型，自动测速选择最快节点
- **DIRECT**：直连模式
- **REJECT**：拒绝连接

### 步骤 5：切换代理节点

**方式一：手动切换**
```bash
.\mihomo-cli.exe proxy switch PROXY "香港-优化-Gemini"
```

**输出示例：**
```
✓ 代理切换成功
  代理组: PROXY
  节点: 香港-优化-Gemini
```

**方式二：自动选择最快节点**
```bash
.\mihomo-cli.exe proxy auto PROXY
```

**说明：**
- 自动测试所有节点延迟
- 选择延迟最低的节点并切换
- 支持自定义测试 URL 和超时时间

**方式三：测试延迟后手动选择**
```bash
# 测试所有节点延迟
.\mihomo-cli.exe proxy test PROXY

# 测试单个节点延迟
.\mihomo-cli.exe proxy test PROXY "香港-优化-Gemini"

# 自定义测试参数
.\mihomo-cli.exe proxy test PROXY --url https://www.google.com/generate_204 --timeout 10000
```

### 步骤 6：设置代理环境变量

**Windows PowerShell：**
```powershell
$env:HTTP_PROXY="http://127.0.0.1:7890"
$env:HTTPS_PROXY="http://127.0.0.1:7890"
```

**验证环境变量：**
```powershell
Write-Host "HTTP_PROXY=$env:HTTP_PROXY"
Write-Host "HTTPS_PROXY=$env:HTTPS_PROXY"
```

**永久设置（推荐）：**
```powershell
# 设置用户级环境变量
[System.Environment]::SetEnvironmentVariable('HTTP_PROXY', 'http://127.0.0.1:7890', 'User')
[System.Environment]::SetEnvironmentVariable('HTTPS_PROXY', 'http://127.0.0.1:7890', 'User')
```

### 步骤 7：验证代理是否成功

**测试访问谷歌：**
```powershell
curl -I --connect-timeout 10 https://www.google.com
```

**成功输出示例：**
```
HTTP/1.1 200 Connection established

HTTP/1.1 200 OK
Content-Type: text/html; charset=ISO-8859-1
...
```

**失败输出示例（不使用代理）：**
```
curl: (28) Connection timed out after 5012 milliseconds
```

**其他验证方式：**
```bash
# 查询当前 IP
curl https://api.ipify.org

# 测试延迟
.\mihomo-cli.exe proxy test PROXY "香港-优化-Gemini"

# 查看当前模式
.\mihomo-cli.exe mode get
```

## 代理模式说明

### Rule 模式（推荐）

当前配置使用规则模式，所有流量会根据规则匹配：

```yaml
mode: rule
rules:
  - MATCH,PROXY
```

**工作原理：**
1. 流量到达 Mihomo
2. 根据规则列表进行匹配
3. 匹配到规则后，流量转发到对应的代理组
4. 当前配置中，所有流量都匹配到 PROXY 组

### Global 模式

所有流量都通过代理：

```bash
.\mihomo-cli.exe mode set global
```

### Direct 模式

所有流量直连，不走代理：

```bash
.\mihomo-cli.exe mode set direct
```

## 常用命令汇总

### 内核管理

```bash
# 启动内核
.\mihomo-cli.exe start

# 查询状态
.\mihomo-cli.exe status

# 停止内核
.\mihomo-cli.exe stop

# 停止所有内核
.\mihomo-cli.exe stop --all
```

### 订阅管理

```bash
# 更新订阅
.\mihomo-cli.exe sub update

# 查看订阅列表
.\mihomo-cli.exe sub list
```

### 代理管理

```bash
# 列出所有代理
.\mihomo-cli.exe proxy list

# 列出指定代理组
.\mihomo-cli.exe proxy list PROXY

# 切换代理
.\mihomo-cli.exe proxy switch PROXY "节点名称"

# 测试代理组延迟
.\mihomo-cli.exe proxy test PROXY

# 测试单个节点延迟
.\mihomo-cli.exe proxy test PROXY "节点名称"

# 自动选择最快节点
.\mihomo-cli.exe proxy auto PROXY

# 取消固定代理
.\mihomo-cli.exe proxy unfix PROXY
```

### 模式管理

```bash
# 获取当前模式
.\mihomo-cli.exe mode get

# 设置模式
.\mihomo-cli.exe mode set rule
.\mihomo-cli.exe mode set global
.\mihomo-cli.exe mode set direct
```

### 配置管理

```bash
# 初始化配置
.\mihomo-cli.exe config init

# 查看配置
.\mihomo-cli.exe config show

# 设置配置
.\mihomo-cli.exe config set api.secret "密钥"
.\mihomo-cli.exe config set api.address "http://127.0.0.1:9090"
```

## 故障排查

### 1. API 连接失败

**问题：** 无法连接到 Mihomo API

**解决：**
- 确认 Mihomo 内核已启动：`.\mihomo-cli.exe status`
- 检查 API 地址是否正确
- 检查 API 密钥是否正确
- 检查防火墙设置

### 2. 订阅更新失败

**问题：** 无法更新订阅

**解决：**
- 检查网络连接
- 检查订阅 URL 是否正确
- 检查是否需要代理访问订阅 URL
- 查看日志了解详细错误信息

### 3. 代理切换失败

**问题：** 无法切换代理节点

**解决：**
- 确认节点名称正确
- 确认节点状态为可用（✓）
- 测试节点延迟确认节点可用
- 检查代理组配置

### 4. 代理验证失败

**问题：** 无法访问谷歌

**解决：**
- 确认代理环境变量已设置
- 确认代理节点已切换
- 测试节点延迟
- 检查代理模式是否正确
- 尝试切换其他节点

## 最佳实践

### 1. 自动化启动流程

创建 PowerShell 脚本 `start-proxy.ps1`：

```powershell
# 启动 Mihomo
.\mihomo-cli.exe start

# 等待内核启动
Start-Sleep -Seconds 5

# 更新订阅
.\mihomo-cli.exe sub update

# 自动选择最快节点
.\mihomo-cli.exe proxy auto PROXY

# 设置代理环境变量
$env:HTTP_PROXY="http://127.0.0.1:7890"
$env:HTTPS_PROXY="http://127.0.0.1:7890"

Write-Host "代理启动完成！"
```

### 2. 定期自动测速

使用 Windows 任务计划程序创建定时任务：
1. 触发器：每天 0:00
2. 操作：运行 PowerShell 脚本
3. 脚本内容：`.\mihomo-cli.exe proxy auto PROXY`

### 3. 节点选择策略

**推荐节点：**
- 延迟低于 200ms 的节点
- 状态为 ✓（可用）
- 流量充足的节点（如 "剩余流量：96.31 GB"）

**避免节点：**
- 延迟超过 500ms 的节点
- 状态为 ✗（不可用）
- 流量即将耗尽的节点

## 附录

### 代理端口说明

- **mixed-port: 7890**：HTTP + SOCKS 混合端口
- **port: 7890**：HTTP 端口（配置未单独设置）
- **socks-port**：SOCKS 端口（配置未单独设置）

### 配置文件位置

**Windows：**
```
C:\Users\<用户名>\.config\mihomo\
├── profiles\           # 订阅文件
│   └── my-subscription.yaml
├── cache.db            # 缓存数据库
└── config.yaml         # 运行时配置
```

**订阅持久化：**
- 订阅数据会自动保存到 `profiles/` 目录
- 支持自动更新（按配置的 interval 定期更新）
- 支持健康检查（定期检查节点可用性）

### 参考文档

- 订阅导入指南：`docs/subscription-import-guide.md`
- 更换节点与批量测速：`docs/更换节点与批量测速.txt`
- Mihomo API 文档：`docs/spec/mihono-api.md`