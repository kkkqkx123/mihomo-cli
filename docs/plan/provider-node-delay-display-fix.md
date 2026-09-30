# provider 节点延迟展示修复方案

## 一、问题现象

连续执行以下两条命令，第二条仍显示"未测试"：

```bash
mihomo-cli proxy list --test-delay PROXY   # 子节点均已显示延迟，当前节点 245ms
mihomo-cli proxy current PROXY             # 延迟：未测试
```

追加 `--test-delay` 后实测输出"状态：超时"，而同一节点在第一条命令里刚刚测出 245ms。

## 二、根因分析

mihomo 内核把节点分两处存放：

- 静态节点与代理组注册在 `/proxies`；
- `proxy-providers` 产生的节点只存在于 `/providers/proxies/{provider}/{name}`。

对运行中的内核实测（配置中 PROXY 组的 46 个节点全部来自 `my-subscription`）：

| 请求 | 结果 |
|------|------|
| `GET /proxies/🇯🇵 日本W10 \| IEPL` | 404 |
| `GET /providers/proxies/my-subscription/🇯🇵 日本W10 \| IEPL` | history 含 245ms 等记录（由 `list --test-delay` 写入） |
| `GET /proxies/PROXY` | `history: []`、`alive: true` |
| `GET /proxies/AUTO`（URLTest 组） | `history: []` |
| `GET /providers/proxies` | 只有 `default`（静态节点与组）与 `my-subscription` 两个 provider |

两个结论：

1. provider 节点在 `/proxies` 下必然 404，其延迟历史只能从 provider 接口读取；
2. 代理组自身从不产生延迟记录（`getGroupDelay` 调用的是内层 adapter 的 URLTest，不写外层 history），所以组行、组的 `alive` 都不能代表当前节点的延迟状态。

内核为每个节点输出 `provider-name` 字段：真实 provider 节点非空（`mihomo/adapter/provider/provider.go:449` 统一注入），`default` 桶内的静态节点与组为空。该字段可作为节点归属的唯一判定依据。

## 三、缺陷清单

1. `cmd/proxy.go:706` —— `proxy current` 无 `--test-delay` 时用 `client.GetProxy(proxyGroup.Now)` 读当前节点，provider 节点打到 404 后被 `if err == nil` 静默吞掉，`Delay` 保持 0，而 `Alive` 取自组（true），于是显示"未测试"。这是本问题的直接原因。
2. `cmd/proxy.go:697` —— `proxy current --test-delay` 分支用 `TestSingle` 打 `/proxies/{now}/delay`，同样 404，`Alive` 被置 false，显示"状态：超时"。`cmd/proxy.go:455` 的 `runProxyTest` 已有 provider 分支，`current` 未复用。
3. `cmd/proxy.go:706` —— 用 `proxyGroup.Delay == 0` 决定是否去查节点历史，而组 history 恒为空，条件恒真；语义上"当前节点信息"本就应始终取当前节点的数据。
4. `internal/api/proxy.go:140` —— `GetProviderProxy` 全项目无调用点，且不像 `ListProxies`/`GetProxy` 那样从 history 回填 `Delay`，即使被调用也拿不到延迟。
5. `internal/proxy/formatter.go:111,164` —— 组内子节点从 `allProxies` 查条目，provider 节点不在 `/proxies` 中，所以 `proxy list PROXY`（不带 `--test-delay`）子节点全显示 `-`，尽管内核里有完整 history；同文件 JSON 分支在组过滤模式下只输出组本身，子节点数据无论如何都到不了 `-o json` 的结果里。
6. `internal/proxy/formatter.go:96,149` —— 组行延迟取组自身数据（恒为 0），`list --test-delay PROXY` 首行恒显示"未测试"。
7. `pkg/types/proxy.go:13` —— `Provider` 字段 tag 与内核 `provider-name` 不符，且全项目无引用，节点归属本可由内核字段直接判定。
8. `cmd/proxy.go` 的 `runProxyTest`/`runProxyAuto`（验证阶段发现）—— 无配置文件时 `viper.GetString("proxy.test_url")` 为空，`SetTestURL("")` 覆盖了 `NewDelayTester` 的内置默认 URL，请求不带 `url` 参数，内核立即返回 503，`proxy test` 对任何节点都报"节点不可用"，耗时 0ms。`runProxyList`/`runProxyCurrent` 用的是"为空则不覆盖"的写法，两处不一致。

## 四、修改方案

统一规则，先于一切分支：**`ProxyInfo.ProviderName`（内核 `provider-name`）非空即 provider 节点，数据与测速走 provider 接口；为空即普通节点，走 `/proxies` 接口。**

### pkg/types

- `proxy.go`：把无引用的 `Provider` 字段改为 `ProviderName`，tag 修正为 `provider-name,omitempty`。
- `provider.go`：`ProviderInfo.Proxies` 直接以 `[]*ProxyInfo` 解码，删除字段大量重复的 `ProviderProxyInfo`；同步修改 `internal/api/provider_test.go` 中的构造。

### internal/api

- 抽出 `fillDelayFromHistory`，由 `ListProxies`、`GetProxy` 复用，消除重复的回填片段。
- `provider.go`：`ListProviders` 对每个 provider 节点回填 `Delay`；新增 `ListProviderProxyMap`，一次请求返回按节点名索引的全部 provider 节点（含 `ProviderName`），供 list 补全子节点、current 定位当前节点使用。
- 删除死代码 `GetProviderProxy`，其职责由索引接口取代。

### internal/proxy

- `tester.go` 的 `BuildProviderMapping` 改用节点自带的 `ProviderName` 建映射、空值跳过，不再依赖 provider 的 map key，使 `list --test-delay`、`test`、`auto`、`current` 对同一节点走同一条路径。本配置下 key 与字段值相同，行为不变；静态节点则回到规范的 `/proxies/{name}/delay`。
- `formatter.go` 的组过滤 JSON 分支输出组本身与组内子节点，与不过滤时的扁平结构一致，使 `-o json` 也能拿到子节点的 history 与 delay。

### cmd/proxy.go — runProxyList

- 存在代理组时拉取 provider 索引，失败直接返回错误，不再 Warning 后继续输出错误数据。
- 组过滤模式：`group.All` 中缺失于 `proxies` 的节点用索引补入，修复子节点显示 `-` 与 JSON 缺失子节点。
- 测速结果注入之后，每个组的 `Delay/Alive` 由其 `Now` 节点的解析结果赋值（`proxies` 优先、索引兜底），作为定义而非回退；异步测速的提前返回路径同样先赋值再格式化。

### cmd/proxy.go — runProxyCurrent

- 以 provider 索引取当前节点：命中则用索引数据并取 `ProviderName`；未命中则查 `/proxies/{now}`，仍失败返回错误，不再静默显示"未测试"。
- `--test-delay` 分支按 `ProviderName` 是否为空选择 `TestProviderProxy` 或 `TestSingle`，与 `runProxyTest` 一致。
- 读取分支直接用节点数据赋 `Delay/Alive`，删除恒真的 `proxyGroup.Delay == 0` 条件。

### cmd/proxy.go — runProxyTest / runProxyAuto

- 测试 URL 统一为"命令行参数 > 配置文件 > 内置默认值"：配置为空时不覆盖 `DelayTester` 的默认 URL，消除无配置文件时测速必现 503 的缺陷（`runProxyList`、`runProxyCurrent` 本就是这个写法）。

不改动：版本号、模块划分；`proxy auto` 仅统一 URL 写法，验证时不执行其切换行为。

## 五、验证步骤

1. `go test ./...`，覆盖新增的 `ListProviderProxyMap` history 回填与 `ProviderName` 用例。
2. `go build -o build/mihomo-cli .` 单次构建。
3. 对运行中的内核手工核对：
   - `proxy current PROXY` 显示真实延迟而非"未测试"；
   - `proxy current PROXY --test-delay` 显示实测延迟而非"超时"；
   - `proxy list PROXY` 子节点为延迟值而非 `-`，组行为当前节点延迟；
   - `proxy list --test-delay PROXY` 首行不再"未测试"；
   - `proxy list -o json PROXY` 子节点含 history 与 delay；
   - `proxy test PROXY <节点>` 确认 `BuildProviderMapping` 改动未影响既有路径。

## 六、验证结果

- `go test ./...` 全部通过，`go build -o build/mihomo-cli .` 单次构建成功，改动文件均通过 gofmt。
- 对运行中的内核实测（复现原始两条命令）：

| 命令 | 修改前 | 修改后 |
|------|--------|--------|
| `proxy list --test-delay PROXY` 首行 | 未测试 | 250ms（当前节点实测值） |
| `proxy current PROXY` | 未测试 | 250ms |
| `proxy current PROXY --test-delay` | 状态：超时 | 275ms |
| `proxy list PROXY` 子节点 | `-` | 各节点历史延迟（如 187ms） |
| `proxy list PROXY -o json` | 只有组本身 | 47 条（组 + 46 子节点，含 history/delay/provider-name） |
| `proxy list`（不带参数）组行 | 未测试 | AUTO 63ms、PROXY 312ms |
| `proxy test PROXY 香港W01` | 节点不可用（0ms，503） | 169ms |
| `proxy list PROXY --wait=false --test-delay` | 提前返回 | 正常输出历史延迟后再返回 |
