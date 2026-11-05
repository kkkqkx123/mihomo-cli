# 进程管理命令 (start, stop, status, ps, cleanup, mode)

进程管理命令用于管理 Mihomo 内核进程和运行模式。

## 进程管理

### start - 启动 Mihomo 内核

启动 Mihomo 内核并自动生成随机密钥。

**语法：**
```bash
mihomo-cli start
```

**启动流程：**
1. 读取 config.toml 配置文件（优先当前目录）
2. 自动生成 SHA256 随机密钥
3. 启动 Mihomo 内核进程
4. 等待并验证内核启动成功
5. 输出 API 地址和密钥信息

**示例：**
```bash
mihomo-cli start
```

### stop - 停止 Mihomo 内核

停止正在运行的 Mihomo 内核进程。

**语法：**
```bash
mihomo-cli stop [pid]
```

**参数：**
- `pid` (可选) - 进程 ID

**选项：**
- `-a, --all` - 停止所有 Mihomo 进程
- `-c, --config` - 指定配置文件路径
- `-F, --force` - 强制关闭进程（不通过 API）

**示例：**
```bash
# 停止默认配置的实例（通过 API）
mihomo-cli stop

# 停止指定 PID 的实例（通过 API）
mihomo-cli stop 12345

# 强制关闭默认配置的实例
mihomo-cli stop -F

# 强制关闭指定 PID 的实例
mihomo-cli stop -F 12345

# 停止所有 Mihomo 进程
mihomo-cli stop --all
```

### status - 查询 Mihomo 内核状态

通过 API 实时查询内核状态，不需要 TOML 配置。

- API 地址和密钥使用 `[api]`、`--api`、`--secret` 指定。
- API 可达时报告运行中；API 不可达时通过进程扫描区分“未运行”和“运行中但 API 不可达”。
- 能确定时显示 PID、管理方式（systemd 单元 / mihomo-cli / 外部进程）。
- 无法确定 PID 时不输出 PID。

**语法：**
```bash
mihomo-cli status [--api URL] [--secret SECRET]
```

**示例：**
```bash
mihomo-cli status
mihomo-cli status --api http://127.0.0.1:9090
mihomo-cli status -o json
```

**JSON 字段：**

- `state`：`running`、`unreachable`、`stopped`
- `running`：是否检测到内核进程或可达 API
- `api_reachable`：API 是否可达
- `pid`：可确定时的内核 PID
- `api`：完整 API 地址
- `manager`：`systemd`、`cli`、`external`
- `unit`：systemd 单元名
- `process_count`：无法确定 PID 时检测到的内核进程数
- `api_error`：API 探测失败原因

退出码 0 表示状态查询完成；运行状态通过 JSON 字段判断。

### ps - 列出所有 Mihomo 进程

列出所有正在运行的 Mihomo 进程及其详细信息。

- 来源分为 `托管`（本 CLI 启动）、`systemd`（由 systemd 单元管理）与 `外部`。
- systemd 管理方式通过 `/proc/<pid>/cgroup` 识别，并用 systemctl 的 MainPID 交叉校验；systemctl 不可用时信任 cgroup。
- 进程可执行文件名精确匹配 `mihomo` / `mihomo.exe`，不会把 `mihomo-cli`、shell 等误判为内核。
- 能从进程命令行解析到配置文件时，API 端口会从该配置的 `external-controller` 中提取；无法解析时显示“未知”。

**语法：**
```bash
mihomo-cli ps
```

**示例：**
```bash
mihomo-cli ps
```

### cleanup - 清理残留的 PID 文件

清理所有残留的 PID 文件（进程已退出但 PID 文件仍存在）。

**语法：**
```bash
mihomo-cli cleanup
```

**示例：**
```bash
mihomo-cli cleanup
```

## 运行模式管理

### mode get - 查询当前运行模式

查询当前 Mihomo 的运行模式。

**语法：**
```bash
mihomo-cli mode get
```

**示例：**
```bash
mihomo-cli mode get
mihomo-cli mode get -o json
```

### mode set - 设置运行模式

设置 Mihomo 的运行模式。

**语法：**
```bash
mihomo-cli mode set <mode>
```

**参数：**
- `mode` - 运行模式，可选值：
  - `rule` - 规则模式：根据规则文件决定流量走向
  - `global` - 全局模式：所有流量通过代理
  - `direct` - 直连模式：所有流量不经过代理

**示例：**
```bash
# 切换到规则模式
mihomo-cli mode set rule

# 切换到全局模式
mihomo-cli mode set global

# 切换到直连模式
mihomo-cli mode set direct
```

## 进程信息说明

进程列表中包含以下信息：
- PID - 进程 ID
- 状态 - 进程状态（已验证/未知）
- 可执行文件 - 进程的可执行文件路径
- API 端口 - API 监听端口

## 运行模式说明

| 模式 | 说明 |
|------|------|
| rule | 根据配置的规则文件决定流量的走向 |
| global | 所有流量都通过代理 |
| direct | 所有流量都不经过代理，直连访问 |

## 注意事项

1. 启动内核时会进行健康检查，确保内核成功启动
2. 如果启动失败或超时，会自动停止内核
3. 停止操作会等待进程完全退出后才返回
4. 默认通过 API 优雅关闭，如果 API 不可用，需要使用 `-F` 强制关闭
5. PID 文件用于跟踪进程状态，进程退出后应清理残留文件
6. 切换运行模式会立即生效，不需要重启内核
