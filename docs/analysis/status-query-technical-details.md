# Status 查询技术细节

本文档整理 `mihomo-cli status` 的当前实现、数据流、状态判定与平台差异，便于后续维护与扩展。

## 入口与调用链

- 命令入口：`cmd/start.go` 中 `statusCmd` / `runStatus`。
- 核心查询：`internal/mihomo/status.go` 中 `QueryStatus(apiAddress, secret)`。
- 管理方式识别：`internal/mihomo/process_manager_linux.go` / `process_manager_other.go` 中 `detectProcessManager(pid)`。
- 进程扫描：`internal/mihomo/discovery.go`、`scanner.go`、`portcheck.go`、`process_linux.go`。

调用顺序：

1. `PersistentPreRunE` (`cmd/root.go`) 初始化 Viper，读取 CLI 配置与 `--api` / `--secret` 覆盖。
2. `runStatus` 调用 `mihomo.QueryStatus(viper.GetString("api.address"), viper.GetString("api.secret"))`。
3. `QueryStatus` 先探测 API，再补充本机进程信息。
4. `runStatus` 根据 `outputFmt` 输出文本或 JSON。

## 配置来源

`status` 不依赖 TOML，也不读取本地 PID/state 文件作为真相来源。API 地址与密钥完全来自：

- CLI 配置文件：`<UserConfigDir>/mihomo-cli/config.yaml` 中 `api.address` / `api.secret`。
- 全局参数：`--api`、`--secret`（显式传参时覆盖配置文件）。
- 环境变量：`MIHOMO_API_ADDRESS`、`MIHOMO_API_SECRET`（Viper 自动环境变量）。

因此 `status` 是无状态 API 查询命令；内核是否由 systemd、CLI 或外部进程启动，不影响查询本身。

## 查询流程

`QueryStatus` 的执行步骤：

1. 校验 `apiAddress` 非空。
2. 创建 API 客户端，超时 2 秒。
3. 调用 `client.GetVersion(ctx)`：
   - 成功：`APIReachable = true`、`Running = true`。
   - 失败：记录 `APIError`，不立即判定未运行。
4. 解析 API 地址，得到 `host` 与 `port`。
5. 判断 API 是否指向本机（`isLocalAPIHost`）。
6. 仅对本机 API 地址执行进程补充：
   - 先通过监听端口查找 PID（`findStatusPID`）。
   - 端口查找失败或进程身份不匹配时，执行一次静默进程扫描。
   - 只有一个内核进程时采用其 PID；多个进程时只记录数量，不猜测 PID。
7. 如果找到 PID：
   - 写入 `PID`。
   - 调用 `detectProcessManager(pid)` 获取管理方式与单元名。
   - 如果未识别为 systemd，则根据 CLI 实例元数据判断是否为 `cli`，否则为 `external`。
8. 根据 API 可达性、进程存在性、是否本机地址确定最终 `State`。

## 状态判定

`State` 取值：

- `running`：API 可达。
- `unreachable`：API 不可达，但检测到内核进程；或远程 API 不可达（无法用本机进程确认）。
- `stopped`：本机 API 不可达，且未检测到内核进程。

`Running` 字段表示是否检测到内核进程或可达 API。`APIReachable` 单独表示 API 是否可达。

退出码约定：`0` 表示状态查询完成；具体运行状态由 JSON 字段表达，不通过退出码区分。

## PID 与进程识别

### 端口到 PID

`findStatusPID`：

1. 空 host/port 直接返回 0。
2. IPv6 地址跳过端口查找，交由进程扫描。
3. 调用 `FindProcessByPort(host, port)`。
4. 用 `VerifyMihomoProcess(pid)` 校验可执行文件名精确为 `mihomo` 或 `mihomo.exe`。

`FindProcessByPort` 通过 `/proc/net/tcp` 查找监听 socket 的 inode，再遍历 `/proc/*/fd` 匹配 `socket:[inode]`。Linux 上已修正 inode 字符串匹配格式。

### 进程扫描兜底

`scanProcessesForStatus` 合并：

- CLI 托管实例：来自 PID 目录元数据，校验进程存活与可执行路径。
- 外部实例：全进程表枚举，仅精确匹配 `mihomo` / `mihomo.exe`。

不再使用“命令行包含 mihomo”的宽松匹配，避免把 `mihomo-cli`、bash 等误判为内核。

### 单进程兜底

端口查找失败时：

- 如果扫描到恰好一个内核进程，采用其 PID。
- 如果扫描到多个进程，只记录 `ProcessCount`，不猜测 PID。
- 如果没有进程，`PID` 为空。

## 管理方式识别

`detectProcessManager(pid)` 的策略：

1. 读取 `/proc/<pid>/cgroup`。
2. 提取最后一个 `.service` 组件作为 systemd 单元名。
3. 单元名必须匹配 `^[A-Za-z0-9_.@:-]+\.service$`。
4. 通过 cgroup 路径推断优先管理器：
   - 包含 `user.slice` 或 `user@` 时优先 `--user`。
   - 否则优先系统管理器。
5. 调用 `systemctl [--user] show <unit> -p MainPID --value` 交叉校验。
6. 如果 systemctl 明确返回另一个 PID，则判定 cgroup 信息陈旧，降级为非 systemd。
7. 如果 systemctl 不可用或调用失败，则信任 cgroup，不丢失信息。
8. 非 Linux 平台返回空，不报告管理方式。

CLI 托管实例在 `ProcessInfo` 中标记为 `cli`；端口识别到的进程如果不是 systemd，再根据 CLI 实例元数据判断 `cli` 或 `external`。

## 输出

文本输出：

- 运行中：`✓ 状态: 运行中`
- API 不可达且检测到进程：`⚠ 状态: 运行中（API 不可达）`
- API 不可达且未检测到进程：`⚠ 状态: API 不可达`
- 未运行：`状态: 未运行`
- 可选字段：`PID`、`管理`、`进程数`、`API 地址`

JSON 输出通过 `output.PrintJSON(result)` 输出纯 JSON，字段包括：

- `state`
- `running`
- `api_reachable`
- `pid`
- `api`
- `manager`
- `unit`
- `process_count`
- `api_error`

进度信息不写入 stdout。

## 平台与环境差异

- Linux：支持 cgroup 与 systemctl 交叉校验；`/proc/<pid>/exe` 可能带 ` (deleted)` 后缀，已规范化。
- 非 Linux：`detectProcessManager` 返回空，管理方式不报告。
- 容器环境：cgroup 往往没有 `.service` 后缀，分类退化为 `cli` 或 `external`，不会编造管理方式。
- 远程 API：不执行本机进程补充，避免把本机 mihomo 错误关联到远程地址。

## 相关文件

- `cmd/start.go`
- `internal/mihomo/status.go`
- `internal/mihomo/process_manager_linux.go`
- `internal/mihomo/process_manager_other.go`
- `internal/mihomo/discovery.go`
- `internal/mihomo/scanner.go`
- `internal/mihomo/portcheck.go`
- `internal/mihomo/process_linux.go`
