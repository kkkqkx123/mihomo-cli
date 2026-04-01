测试国内DoH服务器：
$env:HTTP_PROXY=""; $env:HTTPS_PROXY=""; curl -UseBasicParsing -Uri "https://doh.pub/dns-query?dns=AAABAAABAAAAAAAAA3d3dwNjb20AAAEAAQ" -Headers @{"accept"="application/dns-json"} -TimeoutSec 10

国外：
$env:HTTP_PROXY=""; $env:HTTPS_PROXY=""; curl -UseBasicParsing -Uri "https://8.8.8.8/dns-query?dns=AAABAAABAAAAAAAAAWd3d3cuZ29vZ2xlLmNvbQAAAAA==" -Headers @{"accept"="application/dns-message"} -TimeoutSec 10

$env:HTTP_PROXY=""; $env:HTTPS_PROXY=""; curl -UseBasicParsing -Uri "https://1.1.1.1/dns-query?dns=AAABAAABAAAAAAAAAWd3d3cuZ29vZ2xlLmNvbQAAAAA==" -Headers @{"accept"="application/dns-message"} -TimeoutSec 10
