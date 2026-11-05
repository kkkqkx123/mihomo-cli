# deploy 目录说明

> **本 deploy 目录仅针对 Linux 平台（Linux-only）。** 服务安装、systemd 单元、开机自启等逻辑仅在 Linux 上可用；非 Linux 平台仅提供构建与手动启动指引。

本目录集中存放 Mihomo CLI 的构建、安装与服务部署资产。所有路径、模板与编排逻辑均以此目录为基准，产物输出到仓库根的 `build/` 目录。

## 平台范围

**deploy 仅针对 Linux。** 构建脚本可在任意平台交叉编译，但服务安装、开机自启、systemd 单元渲染与二进制安装逻辑仅在 Linux 上执行。在 macOS / Windows 上运行 `install.sh` 时，脚本仅完成构建与配置初始化，并打印手动启动指引，不会写入任何服务单元。

## 文件清单

| 文件 | 作用 |
|---|---|
| `build.sh` | 统一构建 Mihomo 内核与 mihomo-cli，输出到 `build/` |
| `install.sh` | 编排：构建（可选）、配置初始化、服务安装（可选）、开机自启（可选） |
| `mihomo-core.service` | systemd 单元模板，安装时按实际路径渲染占位符 |
| `config.toml.example` | CLI TOML 配置模板，安装时复制到用户配置目录 |
| `mihomo-config.yaml.example` | Mihomo 内核配置模板，安装时复制到内核配置目录 |

## 构建逻辑（build.sh）

- 默认构建全部目标（`all`），可指定 `mihomo` 或 `cli`。
- 通过 ldflags 在构建期注入版本信息：内核注入 `constant.Version` / `constant.BuildTime`，CLI 注入 `main.version` / `main.commit`（注意注入目标是 `main` 包，`cmd` 包内同名变量在运行时会被覆写）。
- 输出固定为 `build/mihomo` 与 `build/mihomo-cli`（Windows 宿主或交叉到 Windows 时追加 `.exe`）。
- 环境变量：`MIHOMO_TAGS`（内核构建标签，默认 `with_gvisor`）、`GOOS`/`GOARCH`（交叉编译目标，设置后跳过本机运行验证）。
- 交叉编译时无法执行产物做版本验证，因此跳过验证步骤。

## 安装逻辑（install.sh）

### 选项

| 选项 | 含义 |
|---|---|
| `--build` | 执行构建（默认：二进制缺失时构建，存在则跳过） |
| `--no-build` / `--skip-build` | 跳过构建 |
| `--service` | 安装系统服务（Linux systemd） |
| `--no-service` | 不安装服务（默认） |
| `--enable-autostart` / `--autostart` | 开机自启（要求同时 `--service`） |
| `--system` | 系统级服务（需 root；默认用户级服务） |
| `--force` / `-f` | 覆盖已存在的配置文件 |

约束：`--enable-autostart` 必须与 `--service` 同时使用；`--system` 仅在 `--service` 下有效。

### 路径约定

- CLI 连接配置（`config.yaml`）：由 `mihomo-cli config init` 生成，位于 `<UserConfigDir>/mihomo-cli/`。
- TOML 配置（`config.toml`）：从模板复制，位于 `<UserConfigDir>/mihomo-cli/config.toml`。
- Mihomo 内核配置：从模板复制，用户级为 `~/.config/mihomo/config.yaml`，系统级为 `/etc/mihomo/config.yaml`。
- 用户级服务单元：`~/.config/systemd/user/mihomo-core.service`；系统级：`/etc/systemd/system/mihomo-core.service`。
- 二进制安装目录：用户级 `~/.local/bin`，系统级 `/usr/local/bin`。

`UserConfigDir` 遵循 `XDG_CONFIG_HOME`（若设置），否则为 `$HOME/.config`。

### 初始化策略

- 配置已存在时默认跳过，仅 `--force` 覆盖，保证重复执行安全。
- CLI 配置由二进制自带命令生成，保证权限与校验一致；TOML 与内核配置由模板复制，权限设为 `600`。
- 二进制仅在 `--service` 时安装到系统目录；不安装服务时仅使用 `build/` 产物。

### 服务渲染

- 仅 Linux 且已安装 `systemctl` 时执行。
- 将 `mihomo-core.service` 模板中的 `@BIN@`、`@MIHOMO_HOME@`、`@TARGET@` 替换为实际路径与目标（用户级 `default.target`，系统级 `multi-user.target`），写入单元文件并 `daemon-reload`。
- 服务直接运行内核：`mihomo -d <config-dir> -f <config-dir>/config.yaml`，`Type=simple`，`Restart=on-failure`。

### 开机自启

- 用户级服务启用 `systemctl --user enable`；若 `loginctl enable-linger` 可执行则启用 linger（失败时仅登录后启动，并提示管理员命令）。
- 系统级服务直接 `systemctl enable`，无需 linger。

### 非 Linux 行为

在 macOS / Windows 上，脚本完成构建与配置初始化后打印手动启动命令（`$CLI --toml ... start`）并退出，不安装服务、不写入 systemd 单元。

## 服务模型

- 服务单元运行的是 Mihomo 内核本身，而非 `mihomo-cli`。`mihomo-cli` 是无状态 API 客户端，不参与服务进程。
- 生命周期操作统一通过 `systemctl`（用户级加 `--user`），禁止用 `mihomo-cli start/stop` 管理服务实例。
- 配置变更后通过 `systemctl reload` / `restart` 生效，不要修改单元文件存放代理设置。

## 典型用法

```bash
# 构建全部
./deploy/build.sh

# 安装（用户级服务 + 开机自启）
./deploy/install.sh --service --enable-autostart

# 系统级服务（需 root）
sudo ./deploy/install.sh --system --service --enable-autostart

# 仅构建与配置初始化，不安装服务
./deploy/install.sh --no-service
```

服务管理（用户级）：

```bash
systemctl --user start mihomo-core.service
systemctl --user stop mihomo-core.service
systemctl --user restart mihomo-core.service
systemctl --user status mihomo-core.service
journalctl --user -u mihomo-core.service -f
```
