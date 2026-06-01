# DoH 服务器验证测试

本文档提供使用 PowerShell 测试 DoH（DNS over HTTPS）服务器连通性的方法。

## 测试命令

### 国内 DoH 服务器

```powershell
$env:HTTP_PROXY=""; $env:HTTPS_PROXY=""; curl -UseBasicParsing -Uri "https://doh.pub/dns-query?dns=AAABAAABAAAAAAAAA3d3dwNjb20AAAEAAQ" -Headers @{"accept"="application/dns-json"} -TimeoutSec 10
```

### 国外 DoH 服务器

```powershell
# Google
$env:HTTP_PROXY=""; $env:HTTPS_PROXY=""; curl -UseBasicParsing -Uri "https://8.8.8.8/dns-query?dns=AAABAAABAAAAAAAAAWd3d3cuZ29vZ2xlLmNvbQAAAAA==" -Headers @{"accept"="application/dns-message"} -TimeoutSec 10

# Cloudflare
$env:HTTP_PROXY=""; $env:HTTPS_PROXY=""; curl -UseBasicParsing -Uri "https://1.1.1.1/dns-query?dns=AAABAAABAAAAAAAAAWd3d3cuZ29vZ2xlLmNvbQAAAAA==" -Headers @{"accept"="application/dns-message"} -TimeoutSec 10
```

## 说明

- `$env:HTTP_PROXY=""` 和 `$env:HTTPS_PROXY=""` 用于清除代理环境变量，确保直连测试
- `-UseBasicParsing` 在 PowerShell 5.x 中使用以避免依赖 IE 组件
- `-TimeoutSec 10` 设置 10 秒超时
- `accept` header 根据 DoH 提供商要求设置：`application/dns-json`（doh.pub）或 `application/dns-message`（Google/Cloudflare）
- `dns=` 后的 Base64 编码内容是 `www.com` 的 DNS 查询
