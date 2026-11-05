# .mihomo-cli 目录说明

该目录仅为用户目录结构的演示，不会被程序自动加载。

实际生效路径为各平台用户配置目录下的 `mihomo-cli`：

* Windows：`%AppData%/mihomo-cli/config.toml`
* macOS：`~/Library/Application Support/mihomo-cli/config.toml`
* Linux：`~/.config/mihomo-cli/config.toml`（遵循 `XDG_CONFIG_HOME`）

演示文件与实际文件对应关系：

* `.mihomo-cli/config.toml` 对应用户目录下的 `config.toml`

使用方式：

* 将演示文件复制到上述用户目录后直接运行命令。
* 或使用显式参数指向任意位置，例如 `mihomo-cli --toml /path/to/config.toml start`。
* `mihomo` 内核自身的 `config.yaml` 仍需用 `--path` 或 `--mihomo-config` 显式指定，或放在 `~/.config/mihomo/config.yaml`。

该目录下不放置可被隐式拾取的配置文件，切换工作目录不会改变命令行为。
