#!/usr/bin/env bash
#
# install.sh — 构建、初始化配置、可选安装系统服务与开机自启
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
#   构建产物位于 ./build/；配置分别落到 CLI 用户目录与
#   ~/.config/mihomo/config.yaml；服务单元由 deploy/mihomo-cli.service 渲染。

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

user_base() {
	if [ -n "${XDG_CONFIG_HOME:-}" ]; then printf '%s' "$XDG_CONFIG_HOME"
	else printf '%s' "$HOME/.config"; fi
}

CLI_DIR="$(user_base)/mihomo-cli"
TOML_DEST="$CLI_DIR/config.toml"
MIHOMO_DEST="$HOME/.config/mihomo/config.yaml"

mkdir -p "$CLI_DIR" "$(dirname "$MIHOMO_DEST")"

echo "==> 初始化 CLI 配置 ..."
if [ "$FORCE" = 1 ]; then "$CLI" config init --force >/dev/null
elif ! "$CLI" config show >/dev/null 2>&1; then "$CLI" config init >/dev/null
else echo "    已存在，跳过（用 --force 覆盖）"; fi

copy_example() {
	src="$1"; dest="$2"; label="$3"
	if [ -f "$dest" ] && [ "$FORCE" = 0 ]; then
		echo "    $label 已存在，跳过（用 --force 覆盖）: $dest"
		return 0
	fi
	cp "$src" "$dest"
	chmod 600 "$dest"
	echo "    已写入: $dest"
}

echo "==> 初始化 TOML 配置 ..."
copy_example "$ROOT/deploy/config.toml.example" "$TOML_DEST" "TOML 配置"
echo "==> 初始化 Mihomo 配置 ..."
copy_example "$ROOT/deploy/mihomo-config.yaml.example" "$MIHOMO_DEST" "Mihomo 配置"

echo "    CLI 二进制: $CLI"
echo "    TOML: $TOML_DEST"
echo "    Mihomo: $MIHOMO_DEST"

if [ "$DO_SERVICE" = 0 ]; then
	echo "==> 完成（未安装服务）。手动启动: $CLI --toml \"$TOML_DEST\" start"
	exit 0
fi

OS="$(uname -s 2>/dev/null || echo unknown)"
if [ "$OS" != "Linux" ]; then
	echo "==> 非 Linux 系统，不自动安装服务。"
	echo "    Windows（管理员）: $CLI service install"
	echo "    macOS: 参考 launchd 手册将 \`$CLI --toml \"$TOML_DEST\" start\` 注册为服务"
	exit 0
fi

if ! command -v systemctl >/dev/null 2>&1; then
	echo "错误: 未找到 systemctl，无法安装服务" >&2
	exit 1
fi

if [ "$SCOPE" = "system" ]; then
	BIN_DIR="/usr/local/bin"
	UNIT_DEST="/etc/systemd/system/mihomo-cli.service"
	TARGET="multi-user.target"
	SUDO="sudo"
	if [ "$(id -u)" -ne 0 ] && ! command -v sudo >/dev/null 2>&1; then
		echo "错误: --system 需要 root 或 sudo" >&2
		exit 1
	fi
else
	BIN_DIR="$HOME/.local/bin"
	UNIT_DEST="$HOME/.config/systemd/user/mihomo-cli.service"
	TARGET="default.target"
	SUDO=""
	mkdir -p "$BIN_DIR" "$(dirname "$UNIT_DEST")"
fi

echo "==> 安装二进制到 $BIN_DIR ..."
if [ -n "$SUDO" ]; then $SUDO mkdir -p "$BIN_DIR"; fi
for bin in mihomo-cli mihomo; do
	if [ -f "$BUILD/$bin" ]; then
		if [ -n "$SUDO" ]; then $SUDO install -m 755 "$BUILD/$bin" "$BIN_DIR/$bin"
		else install -m 755 "$BUILD/$bin" "$BIN_DIR/$bin"; fi
		echo "    已安装: $BIN_DIR/$bin"
	fi
done

echo "==> 渲染服务单元 ..."
sed -e "s|@BIN@|$BIN_DIR|g" -e "s|@TOML@|$TOML_DEST|g" -e "s|@TARGET@|$TARGET|g" \
	"$ROOT/deploy/mihomo-cli.service" > "/tmp/mihomo-cli.service.$$"
if [ -n "$SUDO" ]; then
	$SUDO install -m 644 "/tmp/mihomo-cli.service.$$" "$UNIT_DEST"
	$SUDO systemctl daemon-reload
else
	install -m 644 "/tmp/mihomo-cli.service.$$" "$UNIT_DEST"
	systemctl --user daemon-reload
fi
rm -f "/tmp/mihomo-cli.service.$$"
echo "    已写入: $UNIT_DEST"

if [ "$DO_AUTOSTART" = 1 ]; then
	echo "==> 启用开机自启 ..."
	if [ -n "$SUDO" ]; then $SUDO systemctl enable mihomo-cli.service
	else systemctl --user enable mihomo-cli.service; fi
	echo "    已启用开机自启"
else
	echo "    未启用开机自启（加 --enable-autostart 启用）"
fi

echo "==> 完成。常用命令:"
if [ -n "$SUDO" ]; then
	echo "    启动: sudo systemctl start mihomo-cli.service"
	echo "    状态: sudo systemctl status mihomo-cli.service"
else
	echo "    启动: systemctl --user start mihomo-cli.service"
	echo "    状态: systemctl --user status mihomo-cli.service"
	echo "    注意: $BIN_DIR 需在 PATH 中"
fi
