#!/usr/bin/env bash
#
# build.sh — 统一构建 mihomo 内核与 mihomo-cli
#
# 说明：
#   - 两个目标均通过 ldflags 在构建期注入版本信息：
#       * mihomo:     constant.Version / constant.BuildTime
#         （修复内核以普通 go build 构建时，始终回显 constant/version.go
#           源码默认值 "1.10.0" 的问题，与 mihomo/Makefile 行为一致）
#       * mihomo-cli: main.version / main.commit（main 包变量，
#         经 cmd.Execute 传入；直接注入 cmd 包变量会被运行时覆写）
#   - 输出到 ./build/，内核固定为 build/mihomo(.exe)，
#     与 config.toml 中 [mihomo] executable = "./build/mihomo" 保持一致。
#
# 环境变量：
#   MIHOMO_TAGS   内核构建标签（默认 with_gvisor，同 mihomo/Makefile）
#   GOOS/GOARCH   交叉编译目标（设置后跳过本机运行验证步骤）

set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
OUT="$ROOT/build"
mkdir -p "$OUT"

usage() {
	cat <<'EOF'
用法: ./scripts/build.sh [mihomo|cli|all]   （缺省 all）

  mihomo   仅构建 Mihomo 内核 -> build/mihomo
  cli      仅构建 mihomo-cli  -> build/mihomo-cli
  all      构建全部

环境变量:
  MIHOMO_TAGS   内核构建标签 (默认 with_gvisor)
  GOOS/GOARCH   交叉编译目标 (设置后跳过本机运行验证)
EOF
}

# Windows 宿主（Git Bash / MSYS / Cygwin）或交叉到 windows 时补 .exe 后缀
EXE=""
case "$(uname -s 2>/dev/null || echo unknown)" in
	MINGW* | MSYS* | CYGWIN*) EXE=".exe" ;;
esac
[ "${GOOS:-}" = "windows" ] && EXE=".exe"

# 交叉编译时无法在本机执行产物做版本验证
RUN_CHECK=true
if [ -n "${GOOS:-}" ] || [ -n "${GOARCH:-}" ]; then
	RUN_CHECK=false
fi

# git_or <fallback> <仓库目录> <git 参数...>
git_or() {
	local fb="$1" dir="$2"
	shift 2
	local out
	out="$(git -C "$dir" "$@" 2>/dev/null)" || out=""
	if [ -z "$out" ]; then echo "$fb"; else echo "$out"; fi
}

build_mihomo() {
	echo "==> 构建 mihomo 内核 (mihomo/) ..."
	local ver btime tags
	ver="$(git_or dev "$ROOT/mihomo" describe --tags --always --dirty)"
	btime="$(date -u +%Y-%m-%dT%H:%M:%SZ)"
	tags="${MIHOMO_TAGS:-with_gvisor}"

	(
		cd "$ROOT/mihomo"
		CGO_ENABLED=0 go build -tags "$tags" -trimpath \
			-ldflags "-X github.com/metacubex/mihomo/constant.Version=$ver -X github.com/metacubex/mihomo/constant.BuildTime=$btime -s -w" \
			-o "$OUT/mihomo$EXE" .
	)
	echo "    注入版本: $ver (build: $btime, tags: $tags)"

	if $RUN_CHECK; then
		echo "    验证: $("$OUT/mihomo$EXE" -v 2>/dev/null | head -n 1)"
	fi
}

build_cli() {
	echo "==> 构建 mihomo-cli (根目录) ..."
	local ver commit
	ver="$(git_or dev "$ROOT" describe --tags --always --dirty)"
	commit="$(git_or unknown "$ROOT" rev-parse --short HEAD)"

	# 注意：注入目标是 main 包的 version/commit（见 main.go）。
	# main() 会把它们传给 cmd.Execute()，cmd 包内同名变量在运行时被覆写，
	# 直接 -X cmd 包的变量不会生效。
	(
		cd "$ROOT"
		go build -trimpath \
			-ldflags "-X main.version=$ver -X main.commit=$commit -s -w" \
			-o "$OUT/mihomo-cli$EXE" .
	)
	echo "    注入版本: $ver (commit: $commit)"

	if $RUN_CHECK; then
		echo "    验证: $("$OUT/mihomo-cli$EXE" version 2>/dev/null | head -n 1)"
	fi
}

target="${1:-all}"
case "$target" in
	mihomo) build_mihomo ;;
	cli)    build_cli ;;
	all)    build_mihomo; build_cli ;;
	-h|--help|help) usage; exit 0 ;;
	*) usage; exit 1 ;;
esac

echo "==> 完成，产物目录: $OUT"
