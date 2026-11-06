# WSL 网络设置（更新后快照与说明）

更新时间：2026-11-06。Linux 侧已应用并验证通过；Windows 侧（`.wslconfig`）保持现状，按需检查。

## 一、最终设置

`/etc/resolv.conf`（`generateResolvConf=false`，手动维护，重启保持）：

```text
nameserver 10.255.255.254
nameserver 192.168.185.211
options timeout:2 attempts:2 edns0
```

`/etc/nsswitch.conf`（与 masked 的 `systemd-resolved` 保持一致，不启用解析服务）：

```text
hosts: mymachines files myhostname dns
```

`/etc/wsl.conf`（本次未改，确认保持）：

```text
[boot]
systemd=true
[automount]
enabled=false
[network]
generateResolvConf = false
[interop]
enabled = false
appendWindowsPath = false
```

Windows 侧 `%USERPROFILE%\.wslconfig`：本次未改。沿用 NAT 默认；如需调整仅建议按需显式启用隧道化 DNS，改后以 `wsl --shutdown` 重启生效。

## 二、关键说明

- 顺序含义：首选 `10.255.255.254`（WSL 固定虚拟 DNS，不随重启换网变化），次选当前网关（实测约 5ms 的快路，但系 DHCP 分配、会变）。此前网关在前，网关一变每次解析先等超时；对调后长期稳定性优先，速度损失可忽略。
- 选项含义：`timeout:2 attempts:2` 将单次故障切换封顶在约 4 秒内，避免默认 5 秒重试导致的长时间卡顿；`edns0` 支持大 DNS 响应。
- `generateResolvConf=false` 必须保留，否则 WSL 每次启动按 Windows DNS 重写本文件，可能引入宿主机代理或 VPN 的污染，前两次故障即与此相关。
- `nsswitch` 与 `resolved` 必须成对变更：当前双关（服务 mask、配置无 `resolve`）是一致态；日后若解开服务，必须同步加回条目。
- 隔离三项（`automount/interop/appendWindowsPath` 全关）与本模型一致：WSL 只做开发、代理在宿主机，不从 WSL 反调 Windows。

## 三、网关变化后的刷新方法

现象：重启或换网后 DNS 突然变慢但不断网。先看默认路由确认新网关，再同步更新 `resolv.conf` 第二行，首行与 `options` 不动。

## 四、本次验证结果

- `getent hosts github.com` 返回 `20.205.243.166`，百度与谷歌解析正常。
- 无代理 `curl`：github 返回 301，百度 HTTP/HTTPS 返回 200。
- 经宿主机代理请求 `generate_204` 返回 204，代理通道不受影响。

## 五、mirrored 模式备选方案（未启用，按需切换）

现状为 NAT，稳定够用，本节仅作预案。mirrored 与 Windows 共享 IP、回环与防火墙，无 `192.168.x` 网关，`localhost:7890` 变为直连而非转发，`10.255.255.254` 与网关行大概率消失，因此第四节之前的 `resolv.conf` 不可直接沿用。

- Windows 侧 `%USERPROFILE%\.wslconfig` 写法：`networkingMode` 设为 mirrored，`dnsTunneling` 设为 true 使 DNS 经 Windows 走，`firewall` 设为 true 使宿主防火墙同时管 WSL，`autoProxy` 设为 false 以保留显式 `http_proxy` 而不叠加自动代理。改后以 `wsl --shutdown` 重启生效。前提为 WSL 2.0 以上与支持的 Windows 11 版本。
- Linux 侧：`nsswitch` 保持无 `resolve` 不变；`wsl.conf` 保持即可；`resolv.conf` 必须重做，删掉网关行。若 `dnsTunneling` 为 true，先看隧道化后 WSL 实际可用的 Windows DNS 并手写；若为 false（例如宿主机 mihomo TUN 污染了 Windows DNS），则手写可靠公网 DNS，但必须重测 UDP 是否可达，不可沿用 NAT 下超时的旧结论。
- 切换后验证：`ip addr` 应见与宿主同网段地址，`localhost:7890` 直通，重做 `getent` 加 `curl --noproxy` 分层验证。
- 回滚：删掉 `networkingMode` 行再 `shutdown` 即回 NAT，并将 `resolv.conf` 恢复为第一节快照。无 IPv6 或局域网直达需求时不建议切换。
