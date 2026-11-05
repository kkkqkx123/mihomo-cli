# 配置加载技术细节

本文档整理 Mihomo CLI 的配置加载机制，包括 CLI YAML 配置、Viper 运行时配置、TOML 配置、路径解析与命令行覆盖规则。

## 配置体系概览

项目存在两类配置：

1. **CLI 配置（YAML）**：`<UserConfigDir>/mihomo-cli/config.yaml`
   - 由 Viper 管理。
   - 存储 API 连接、代理测试、输出等 CLI 运行参数。
   - 是大多数命令的默认配置来源。

2. **TOML 配置**：`<UserConfigDir>/mihomo-cli/config.toml` 或 `--toml` 指定路径
   - 存储 Mihomo 内核管理相关配置（可执行文件、配置文件、守护进程参数）。
   - 新部署默认置空 `executable` 与 `config_file`，由内核使用默认路径。
   - 主要用于 `start`、`stop`、`reload` 等内核管理命令。

两类配置分离，避免把 TOML 当作 YAML 解析，也避免把内核配置与 CLI 连接配置混在同一文件。

## 路径解析

### 基础目录

`internal/config/paths.go`：

- `GetBaseDir()` 返回 `os.UserConfigDir()/mihomo-cli`。
- 平台差异：
  - Linux：`$XDG_CONFIG_HOME/mihomo-cli` 或 `~/.config/mihomo-cli`
  - macOS：`~/Library/Application Support/mihomo-cli`
  - Windows：`%AppData%\Roaming\mihomo-cli`
- `Paths.ConfigFile` 固定为 `<baseDir>/config.yaml`。

### PathResolver

`internal/config/path_resolver.go` 提供统一路径解析：

- `GetConfigFile()`：CLI YAML 路径。
- `GetPIDDir()` / `GetBackupDir()` / `GetHistoryDir()` / `GetDataDir()`：运行时数据目录。
- `GetPIDFilePath(configFile)`：基于 Mihomo 配置文件路径生成唯一 PID 文件名；空配置文件使用 `mihomo.pid`。
- `GetStateFilePath(configFile)`：基于配置文件路径生成状态文件名。

### TOML 路径

`FindTomlConfigPath(customPath)`：

1. 如果传入 `--toml`，解析为绝对路径并要求文件存在。
2. 否则使用用户目录下的 `config.toml`，不存在则报错。
3. 不探测当前工作目录，避免隐式配置。

## CLI 配置加载

### 初始化顺序

`cmd/root.go` 的 `PersistentPreRunE`：

1. `initConfig(cmd)`：
   - 创建 `ConfigManager`。
   - 调用 `InitViperConfig(cfgFile)`。
   - 如果 `--api`、`--secret`、`--timeout` 显式传参，覆盖 Viper 值。
2. `config.LoadFromViper()` 读取并校验运行时配置。
3. 配置类命令 (`mihomo-cli config ...`) 豁免配置校验失败，便于修复坏配置。
4. 根据输出配置初始化输出流。

### Viper 初始化

`ConfigManager.InitViperConfig(cfgFile)`：

1. 设置默认值：
   - `api.address = http://127.0.0.1:9090`
   - `api.secret = ""`
   - `api.timeout = 10`
   - `proxy.*` 默认值
   - `output.*` 默认值
2. 配置文件：
   - 指定 `--config` 时使用该文件。
   - 否则固定 `viper.SetConfigFile(<baseDir>/config.yaml)`，类型为 YAML。
   - 这样避免同目录 `config.toml` 被误认为 YAML。
3. 启用环境变量：
   - 前缀 `MIHOMO`。
   - 点号映射为下划线，例如 `MIHOMO_API_ADDRESS`。
4. 调用 `viper.ReadInConfig()`，文件不存在时静默使用默认值。

### 配置优先级

从高到低：

1. 命令行参数：`--api`、`--secret`、`--timeout`、`--config`、`--toml`。
2. 环境变量：`MIHOMO_*`。
3. CLI YAML 配置文件。
4. Viper 默认值。

注意：`--api` / `--secret` 只在显式传参时覆盖，避免把默认空值写回配置。

### 配置校验

`CLIConfig.Validate()` 校验：

- API 地址必须以 `http://` 或 `https://` 开头，且包含 host。
- 端口范围 1-65535。
- 超时范围 1-300 秒。
- 代理超时、并发数、批大小、最大节点数范围。
- 测试 URL 必须为 HTTP/HTTPS。

配置文件不存在时，`LoadFromViper` 仍可返回默认配置（Viper 默认值），不会因文件缺失而失败。

## TOML 配置加载

### 加载函数

`LoadTomlConfig(path)`：

1. `path` 为空直接报错。
2. 文件必须存在。
3. 使用 `toml.DecodeFile` 解析。
4. 调用 `Validate()`。

### MihomoConfig 关键字段

- `enabled`
- `executable`
- `executable_search`
- `config_file`
- `health_check_timeout`

新部署默认：

- `executable = ""`：运行时从 PATH 查找 `mihomo`。
- `config_file = ""`：内核使用默认 `~/.config/mihomo/config.yaml`。

### DaemonConfig

包含守护进程、日志、自动重启、健康检查等配置。当前服务化部署主要使用 systemd，但 TOML 仍保留这些字段以兼容 CLI 启动模式。

## 配置命令

### config init

- 生成 `<baseDir>/config.yaml`。
- 默认 API 地址 `http://127.0.0.1:9090`。
- 不生成 `secrets.json`；敏感数据直接存储在 CLI YAML 的 `api.secret`。

### config show

- 显示生效配置、文件路径、脱敏后的密钥。
- 支持 `-o json`。

### config set

- 直接编辑 CLI YAML 文件，避免把命令行覆盖持久化。
- 支持 `api.*`、`proxy.*`、`output.*` 等键。

## Mihomo 配置加载与重载

### mihomo edit

1. 通过 `FindConfigPath(mihomoConfigPath, tomlFile)` 确定 Mihomo 配置文件：
   - 显式 `--mihomo-config` 优先。
   - 否则尝试 TOML 中的 `mihomo.config_file`。
   - 最后回退到 `~/.config/mihomo/config.yaml`。
2. 编辑文件并自动备份。
3. 默认触发 API 重载；`--no-reload` 可跳过。

### mihomo reload

1. `--path` 指定时：
   - 必须为绝对路径且文件存在。
   - 校验配置语法，`--force` 可忽略校验错误。
   - 重载前创建备份。
2. 未指定路径时：
   - 优先使用 TOML 中的 `mihomo.config_file`。
   - 否则把空路径传给 Mihomo API，由内核重载自己启动时使用的当前配置文件。
3. 调用 `PUT /configs`，`force=true` 时追加查询参数。
4. Mihomo 服务端在路径为空时使用 `C.Path.Config()`。

### mihomo patch

- 通过 `PATCH /configs` 热更新运行时配置。
- 支持单键值或 `--file` 文件输入。
- 只影响运行时，不修改配置文件。

## 与 status 查询的关系

- `status` 只使用 CLI YAML / `--api` / `--secret`，不读取 TOML。
- `status` 不把 TOML 的 `mihomo.config_file` 作为 API 地址来源。
- 因此即使 TOML 默认置空，`status` 仍可正常工作。

## 错误处理

- 配置文件不存在时，CLI 使用默认值，不视为致命错误。
- TOML 不存在时，需要 `--toml` 显式指定或使用默认用户路径。
- 配置校验失败时，配置类命令可豁免，以便修复坏配置。
- `mihomo reload` 在无法确定本地文件时跳过备份，但仍然尝试 API 重载。

## 相关文件

- `cmd/root.go`
- `cmd/config.go`
- `cmd/mihomo.go`
- `internal/config/config.go`
- `internal/config/config_manager.go`
- `internal/config/loader.go`
- `internal/config/paths.go`
- `internal/config/path_resolver.go`
- `internal/config/toml_config.go`
- `internal/api/config.go`
