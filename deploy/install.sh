#!/usr/bin/env bash
#
# install.sh — 构建、初始化配置、可选安装 mihomo 内核系统服务与开机自启
#
# 用法:
#   ./deploy/install.sh [选项]
#
# 选项:
#   --build              执行构建（默认：二进制缺失时构建，存在则跳过）
#   --no-build           跳过构建
#   --service            安装系统服务（Linux systemd；macOS/Windows 仅打印指引）
#   --no-service         不安装服务（默认）
#   --enable-autostart   开机自启（要求同时 --service）
#   --system             系统级服务（需 root；默认用户级服务）
#   --force              覆盖已存在的配置文件
#   -h, --help           显示帮助
#
# 效果:
#   构建产物位于 ./build/；服务单元 deploy/mihomo-core.service 直接运行 mihomo 内核。
#   用户级服务配置目录为 ~/.config/mihomo；系统级服务配置目录为 /etc/mihomo。
#   CLI 配置始终位于当前用户的 <UserConfigDir>/mihomo-cli。

set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
BUILD="$ROOT/build"
CLI="$BUILD/mihomo-cli"

DO_BUILD="auto"
DO_SERVICE=0
DO_AUTOSTART=0
SCOPE="user"
FORCE=0

usage() {
	sed -n '2,/^$/p' "$0" | sed 's/^# \?//'
}

while [ $# -gt 0 ]; do
	case "$1" in
		--build) DO_BUILD=1; shift ;;
		--no-build|--skip-build) DO_BUILD=0; shift ;;
		--service) DO_SERVICE=1; shift ;;
		--no-service) DO_SERVICE=0; shift ;;
		--enable-autostart|--autostart) DO_AUTOSTART=1; shift ;;
		--system) SCOPE="system"; shift ;;
		--force|-f) FORCE=1; shift ;;
		-h|--help|help) usage; exit 0 ;;
		*) echo "未知参数: $1（用 --help 查看）" >&2; exit 1 ;;
	esac
done

if [ "$DO_AUTOSTART" = 1 ] && [ "$DO_SERVICE" = 0 ]; then
	echo "错误: --enable-autostart 要求同时 --service" >&2
	exit 1
fi
if [ "$SCOPE" = "system" ] && [ "$DO_SERVICE" = 0 ]; then
	echo "错误: --system 仅在 --service 下有效" >&2
	exit 1
fi

if [ "$DO_BUILD" = "auto" ]; then
	if [ -x "$CLI" ]; then DO_BUILD=0; else DO_BUILD=1; fi
fi

if [ "$DO_BUILD" = 1 ]; then
	echo "==> 构建 ..."
	bash "$ROOT/deploy/build.sh" all
fi

if [ ! -x "$CLI" ]; then
	echo "错误: 找不到可执行文件 $CLI（先用 --build 构建）" >&2
	exit 1
fi

OS="$(uname -s 2>/dev/null || echo unknown)"

user_base() {
	if [ -n "${XDG_CONFIG_HOME:-}" ]; then printf '%s' "$XDG_CONFIG_HOME"
	else printf '%s' "$HOME/.config"; fi
}

CLI_DIR="$(user_base)/mihomo-cli"
TOML_DEST="$CLI_DIR/config.toml"

SUDO=""
if [ "$SCOPE" = "system" ]; then
	BIN_DIR="/usr/local/bin"
	MIHOMO_HOME="/etc/mihomo"
	UNIT_DEST="/etc/systemd/system/mihomo-core.service"
	TARGET="multi-user.target"
	SUDO="sudo"
	if [ "$(id -u)" -ne 0 ]; then
		if ! command -v sudo >/dev/null 2>&1; then
			echo "错误: --system 需要 root 或 sudo" >&2
			exit 1
		fi
	else
		SUDO=""
	fi
else
	BIN_DIR="$HOME/.local/bin"
	MIHOMO_HOME="$HOME/.config/mihomo"
	UNIT_DEST="$HOME/.config/systemd/user/mihomo-core.service"
	TARGET="default.target"
fi

run_root() {
	if [ -n "$SUDO" ]; then $SUDO "$@"; else "$@"; fi
}

copy_cli_example() {
	src="$1"; dest="$2"; label="$3"
	if [ -f "$dest" ] && [ "$FORCE" = 0 ]; then
		echo "    $label 已存在，跳过（用 --force 覆盖）: $dest"
		return 0
	fi
	mkdir -p "$(dirname "$dest")"
	cp "$src" "$dest"
	chmod 600 "$dest"
	echo "    已写入: $dest"
}

copy_core_example() {
	src="$1"; dest="$2"; label="$3"
	if run_root test -f "$dest" && [ "$FORCE" = 0 ]; then
		echo "    $label 已存在，跳过（用 --force 覆盖）: $dest"
		return 0
	fi
	run_root mkdir -p "$(dirname "$dest")"
	run_root cp "$src" "$dest"
	run_root chmod 600 "$dest"
	echo "    已写入: $dest"
}

mkdir -p "$CLI_DIR"
run_root mkdir -p "$MIHOMO_HOME"

echo "==> 初始化 CLI 配置 ..."
if [ "$FORCE" = 1 ]; then "$CLI" config init --force >/dev/null
elif [ ! -f "$CLI_DIR/config.yaml" ]; then "$CLI" config init >/dev/null
else echo "    已存在，跳过（用 --force 覆盖）"; fi

echo "==> 初始化 TOML 配置 ..."
copy_cli_example "$ROOT/deploy/config.toml.example" "$TOML_DEST" "TOML 配置"
echo "==> 初始化 Mihomo 配置 ..."
copy_core_example "$ROOT/deploy/mihomo-config.yaml.example" "$MIHOMO_HOME/config.yaml" "Mihomo 配置"

echo "    CLI 二进制: $CLI"
echo "    TOML: $TOML_DEST"
echo "    Mihomo: $MIHOMO_HOME/config.yaml"

if [ "$DO_SERVICE" = 1 ]; then
	echo "==> 安装二进制到 $BIN_DIR ..."
	run_root mkdir -p "$BIN_DIR"
	for bin in mihomo mihomo-cli; do
		if [ -f "$BUILD/$bin" ]; then
			run_root install -m 755 "$BUILD/$bin" "$BIN_DIR/$bin"
			echo "    已安装: $BIN_DIR/$bin"
		fi
	done
fi

if [ "$OS" != "Linux" ]; then
	if [ "$DO_SERVICE" = 1 ]; then
		echo "==> 非 Linux 系统，不自动安装服务。"
		echo "    Windows（管理员）: 将 mihomo.exe 注册为服务需要实现 svc.Run 的包装器"
		echo "    macOS: 参考 launchd 手册将以下命令注册为服务"
		echo "           $BIN_DIR/mihomo -d \"$MIHOMO_HOME\" -f \"$MIHOMO_HOME/config.yaml\""
	else
		echo "==> 完成（未安装服务）。手动启动:"
		echo "    $CLI --toml \"$TOML_DEST\" start"
	fi
	exit 0
fi

if [ "$DO_SERVICE" = 0 ]; then
	echo "==> 完成（未安装服务）。手动启动:"
	echo "    $CLI --toml \"$TOML_DEST\" start"
	exit 0
fi

if ! command -v systemctl >/dev/null 2>&1; then
	echo "错误: 未找到 systemctl，无法安装服务" >&2
	exit 1
fi

echo "==> 渲染服务单元 ..."
run_root mkdir -p "$(dirname "$UNIT_DEST")"
unit_tmp="$(mktemp)"
sed -e "s|@BIN@|$BIN_DIR|g" -e "s|@MIHOMO_HOME@|$MIHOMO_HOME|g" -e "s|@TARGET@|$TARGET|g" \
	"$ROOT/deploy/mihomo-core.service" > "$unit_tmp"
run_root install -m 644 "$unit_tmp" "$UNIT_DEST"
rm -f "$unit_tmp"

if [ -n "$SUDO" ]; then
	$SUDO systemctl daemon-reload
else
	systemctl --user daemon-reload
fi
echo "    已写入: $UNIT_DEST"

if [ "$DO_AUTOSTART" = 1 ]; then
	echo "==> 启用开机自启 ..."
	linger_ok=0
	if [ -n "$SUDO" ]; then
		$SUDO systemctl enable mihomo-core.service
		linger_ok=1
	else
		systemctl --user enable mihomo-core.service
		if command -v loginctl >/dev/null 2>&1; then
			if loginctl enable-linger "$(id -un)" >/dev/null 2>&1; then
				linger_ok=1
				echo "    已启用 linger（用户服务随系统启动）"
			else
				echo "    已启用用户级自启（登录后启动）" >&2
				echo "    开机自启需管理员执行: sudo loginctl enable-linger $(id -un)" >&2
			fi
		fi
	fi
	if [ "$linger_ok" = 1 ]; then
		echo "    已启用开机自启"
	fi
else
	echo "    未启用开机自启（加 --enable-autostart 启用）"
fi

echo "==> 完成。常用命令:"
if [ -n "$SUDO" ]; then
	echo "    启动: sudo systemctl start mihomo-core.service"
	echo "    状态: sudo systemctl status mihomo-core.service"
	echo "    日志: sudo journalctl -u mihomo-core.service -f"
else
	echo "    启动: systemctl --user start mihomo-core.service"
	echo "    状态: systemctl --user status mihomo-core.service"
	echo "    日志: journalctl --user -u mihomo-core.service -f"
	echo "    注意: $BIN_DIR 需在 PATH 中"
fi
