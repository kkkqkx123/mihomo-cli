# build.ps1 — 在 Windows 上构建 mihomo 内核与 mihomo-cli（与 build.sh 等价）
#
# 用法:
#   .\scripts\build.ps1               # 构建全部
#   .\scripts\build.ps1 -Target cli   # 仅 mihomo-cli
#   .\scripts\build.ps1 -Target mihomo
#
# 说明:
#   - 通过 ldflags 注入版本信息（mihomo: constant.Version/BuildTime；
#     mihomo-cli: main.version/main.commit，main 包变量经 cmd.Execute 传入），
#     修复内核恒显示 1.10.0 的问题。
#   - 产物输出到 .\build\，内核固定为 build\mihomo.exe，
#     与 config.toml 中 [mihomo] executable = "./build/mihomo" 匹配。
#
# 环境变量:
#   MIHOMO_TAGS  内核构建标签（默认 with_gvisor，同 mihomo/Makefile）

param(
    [ValidateSet("all", "cli", "mihomo")]
    [string]$Target = "all"
)

$ErrorActionPreference = "Stop"

$Root = Split-Path -Parent $PSScriptRoot
$Out = Join-Path $Root "build"
New-Item -ItemType Directory -Force -Path $Out | Out-Null

function Invoke-Git([string]$Dir, [string]$Fallback, [string[]]$GitArgs) {
    $v = & git -C $Dir @GitArgs 2>$null
    if ($LASTEXITCODE -ne 0 -or -not $v) { return $Fallback }
    return ($v | Select-Object -First 1)
}

function Get-BuildTime {
    (Get-Date).ToUniversalTime().ToString("yyyy-MM-ddTHH:mm:ssZ")
}

function Build-Mihomo {
    Write-Host "==> 构建 mihomo 内核 (mihomo\) ..."
    $Dir = Join-Path $Root "mihomo"
    $Ver = Invoke-Git $Dir "dev" @("describe", "--tags", "--always", "--dirty")
    $BTime = Get-BuildTime
    $Tags = if ($env:MIHOMO_TAGS) { $env:MIHOMO_TAGS } else { "with_gvisor" }

    Push-Location $Dir
    try {
        $env:CGO_ENABLED = "0"
        $Ld = "-X github.com/metacubex/mihomo/constant.Version=$Ver -X github.com/metacubex/mihomo/constant.BuildTime=$BTime -s -w"
        & go build -tags $Tags -trimpath -ldflags $Ld -o (Join-Path $Out "mihomo.exe") .
        if ($LASTEXITCODE -ne 0) { throw "mihomo 构建失败 (exit $LASTEXITCODE)" }
    } finally { Pop-Location }

    Write-Host "    注入版本: $Ver (build: $BTime, tags: $Tags)"
    Write-Host "    验证: $(( & (Join-Path $Out "mihomo.exe") -v 2>$null | Select-Object -First 1 ))"
}

function Build-Cli {
    Write-Host "==> 构建 mihomo-cli (根目录) ..."
    $Ver = Invoke-Git $Root "dev" @("describe", "--tags", "--always", "--dirty")
    $Commit = Invoke-Git $Root "unknown" @("rev-parse", "--short", "HEAD")

    # 注意：注入目标是 main 包的 version/commit（见 main.go）。
    # main() 会把它们传给 cmd.Execute()，cmd 包内同名变量在运行时被覆写，
    # 直接 -X cmd 包的变量不会生效。
    Push-Location $Root
    try {
        $Ld = "-X main.version=$Ver -X main.commit=$Commit -s -w"
        & go build -trimpath -ldflags $Ld -o (Join-Path $Out "mihomo-cli.exe") .
        if ($LASTEXITCODE -ne 0) { throw "mihomo-cli 构建失败 (exit $LASTEXITCODE)" }
    } finally { Pop-Location }

    Write-Host "    注入版本: $Ver (commit: $Commit)"
    Write-Host "    验证: $(( & (Join-Path $Out "mihomo-cli.exe") version 2>$null | Select-Object -First 1 ))"
}

switch ($Target) {
    "mihomo" { Build-Mihomo }
    "cli"    { Build-Cli }
    "all"    { Build-Mihomo; Build-Cli }
}

Write-Host "==> 完成，产物目录: $Out"
