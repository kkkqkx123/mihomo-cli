# WSL 系统 DNS 失效问题分析与修复指南

## 一、现象

- 经 Windows 宿主机 mihomo 代理（`localhost:7890`）访问外网正常，`curl https://www.gstatic.com/generate_204` 返回 204。
- 去代理直连全部失败：`curl --noproxy` 卡死超时，`getent hosts` 报 `Temporary failure in name resolution`。
- 本地无 mihomo 进程，`mihomo-cli status` 显示未运行，符合预期（本机不跑核，只用宿主机代理）。

## 二、环境

- WSL2，`systemd=true`，`generateResolvConf=false`，`interop=false`。
- `systemd-resolved` 为 masked + inactive，`/run/systemd/resolve/` 不存在。
- `http_proxy/https_proxy=http://localhost:7890` 经 WSL localhost 转发指向宿主机 mihomo，实测可用，保留不动。

## 三、排查过程

排查思路是逐层隔离：先确认代理通道，再确认直连 IP 层，最后确认系统 DNS 层。

- 用默认代理环境请求外网成功，证明宿主机代理与 WSL 转发正常，问题不在代理。
- 用 `--noproxy` 请求数字 IP（`http://1.1.1.1/`、`http://20.205.243.166/` 加 Host 头）成功，证明 WSL 直连出网能力正常，问题收敛到域名解析。
- `getent` 与 `getaddrinfo` 对所有域名失败，但经 `ss` 可见 7890 仍有连接，说明是本地 NSS 解析坏了，而非对端服务坏了。
- 绕过 NSS 做原生 UDP DNS 查询：发往 `223.5.5.5` 与 `8.8.8.8` 的请求超时，发往网关与 `10.255.255.254` 的请求秒回且结果正确。证明上游 DNS 可达，只是 `resolv.conf` 指错了地方。
- 检查 `nsswitch.conf` 发现 `resolve [!UNAVAIL=return]` 排在 `files/dns` 之前，而提供该服务的 `systemd-resolved` 根本未运行。检查进程与网卡确认无 mihomo 残留 tun 口与进程，`10.255.255.254:53` 无属主监听属于 WSL 自身转发，并非劫持。

## 四、根本原因

共两个，均为 WSL 系统配置问题，与之前 mihomo 是否正常关闭无关：

1. `nsswitch.conf` 引用了不存在的解析服务。`hosts` 行首选 `resolve`，配合 `!UNAVAIL=return`，在 `systemd-resolved` 被 mask 后，每次解析直接返回不可用，不再尝试 `files` 与 `dns`。表现为所有域名 `Temporary failure`。
2. `resolv.conf` 指向了本环境不可达的公网 UDP DNS。`223.5.5.5` 与 `8.8.8.8` 的 53/UDP 在此 WSL 超时，即使修好上一条也依然解析失败。实测可达的是默认网关（Windows 宿主，约 5ms）与 `10.255.255.254`（WSL 转发）。

## 五、修复方法

只需改两个系统文件，不动代理，不起本地 mihomo 服务。

- 将 `nsswitch.conf` 的 `hosts` 行去掉 `resolve [!UNAVAIL=return]`，保留 `mymachines files myhostname dns`。不建议解开 `systemd-resolved`，在当前 WSL 配置下直接走标准解析更简单可靠。
- 将 `resolv.conf` 改为网关在前、`10.255.255.254` 在后。`wsl.conf` 已关闭自动生成，该文件重启后保持。
- 网关地址由 DHCP 分配，重启或网络变化后可能改变。若日后 DNS 又变慢或失败，先用查看默认路由的方式确认新网关并同步更新，`10.255.255.254` 可作为不受网关变化影响的兜底。

## 六、验证标准

按下述顺序验证，全部通过即算修复：

- `getent hosts` 对常用域名能返回地址，不再报 `Temporary failure`。
- 无代理请求数字 IP 与域名均成功，例如对 github、百度与 `1.1.1.1` 的普通 HTTP 请求返回 200/301。
- 无代理 HTTPS 请求成功，例如对百度的 HTTPS 请求返回 200。
- 原代理通道仍成功，对 `generate_204` 经默认代理返回 204。

本次修复后实测：`getent` 解析 github、百度、谷歌均成功；无代理 `curl` 对 github 返回 301、对百度与 `1.1.1.1` 返回 200、无代理 HTTPS 对百度返回 200；经代理请求 `generate_204` 返回 204。

## 七、注意事项

- 本 WSL 不运行 mihomo 核，`mihomo-core` 服务不存在，保持现状即可。日常代理继续使用宿主机 `localhost:7890`。
- 无代理验证必须显式绕过代理环境，例如给 curl 加 `--noproxy '*'`，否则会误走代理而掩盖 DNS 真实状态。
- 未取得 root 前无法读取 `iptables`，但从进程、网卡与分层测试看，无证据表明存在残留劫持规则。如日后再次出现直连 IP 也不通的情况，再以 root 身份检查防火墙与转发规则。

## 八、错误 DNS 的来源分析

结论先行：现有 `~/.config/mihomo/config.yaml` 不会改动系统 DNS，mihomo 核与 mihomo-cli 均无写系统 DNS 的代码路径。错误的系统 DNS 更可能是人工照抄 mihomo 上游所致，`nsswitch` 则是镜像自带的潜伏错配，本地服务阶段掩盖了它，停服后才暴露。

- 现有 mihomo 配置的 `dns` 段只是核的内部解析器，决定核如何解析代理节点与分流域名，不触及宿主系统的解析器。当前文件无 `tun` 节与 `dns.listen`，连包级别的 DNS 劫持都不具备条件，`ip link` 亦无 tun 口佐证。
- 核仅读取系统 DNS：`mihomo/dns/system_posix.go` 只读 `/etc/resolv.conf`，`component/resolver/hosts` 只读 `/etc/hosts`，无任何写回逻辑。mihomo-cli 亦然：`internal/systemdns` 在 Linux 仅读取并解析 `resolv.conf` 做诊断展示，`internal/sysproxy` 在 Linux 只写代理环境文件，不碰 DNS，`deploy` 脚本无 DNS 相关操作。
- 错误值高度疑似人工写入：故障时 `resolv.conf` 的两行恰为 mihomo 配置的上游首选，即 `nameserver` 首项与 `fallback` 首项的组合，而这两组公网 UDP 在本 WSL 实测全部超时。合理推测是此前按防污染教程或排障直觉把核的上游照抄进了系统文件。WSL 默认 DNS 应为网关或 WSL 转发，不会是该组合。
- `nsswitch` 的 `resolve` 条目来自发行版镜像对 `systemd-resolved` 的集成，而本 WSL 明确 mask 了该服务，属于镜像与现状脱节的潜伏问题。本地核运行期间，流量经 `mixed-port` 由核代为解析，系统 DNS 坏了也无感知；切到宿主机代理后，代理侧代为解析，同样无感知；直到需要直连验证时才集中暴露，容易误判为某次服务未正常关闭所致。
- 预防：不把公网 DNS 手写进 WSL 的 `resolv.conf`，以网关与 WSL 转发为准；本地服务启停前后各做一次无代理留痕，即 `getent` 加 `curl --noproxy` 请求数字 IP 与域名，代理正常不代表系统 DNS 正常。
