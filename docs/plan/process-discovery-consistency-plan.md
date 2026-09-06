# Mihomo 内核定位与进程发现一致性改造方案

> 状态：草案（待评审）
> 适用范围：Windows / Linux / macOS 三个目标平台（不支持 BSD / Android，见 1.3）
> 关联文档：`docs/pid-file-management.md`、`docs/daemon-module-analysis.md`、`docs/plan/cross-platform-abstraction.md`
> 本文所有文件引用均基于当前 `main` 分支（HEAD 8b5f0eb）实际代码核对。

## 0. 执行状态与决策变更（2026-09-06 更新）

### 0.1 决策变更：PID 文件格式锁定 v1，取消 v2 升级与一切回退逻辑

开发阶段按以下约束执行（取代原 4.1.2 / 5.x 的 v2 设计）：

1. **版本锁定 1**：PID 元数据格式版本固定为 `MetaVersion = 1`（`internal/mihomo/instance.go`），
   JSON 元数据能力保留（config/exec/api 信息直接可读），但不再称 "v2"。
2. **取消一切回退逻辑**：删除旧格式识别与降级读取
   （`identity.go` 的 `IsLegacyPIDFileName`、`ReadInstanceFile` 的纯数字降级解析），
   解析失败一律视为损坏文件（跳过 + 提示，不误杀）。
3. **不兼容任何旧数据**：旧目录（`<UserConfigDir>/mihomo-cli`）、旧 hash 命名、纯数字 PID
   一律不读、不迁移、不回退；原 4.6 的旧目录迁移（`paths_migrate.go` / `config migrate` 命令）取消。
4. **目录平台化直接切换**：基础目录改为 `os.UserConfigDir()/mihomo-cli`，无旧目录回退。
5. **安全底线保留**：任何 PID 文件解析失败仍采取"跳过 + 提示"，绝不按错误 PID 执行 kill。

### 0.2 各 Phase 执行进度（全部完成）

| Phase | 内容 | 状态 |
|---|---|---|
| Phase 1 | 配置身份统一（`IdentityOf` 12 位 hex 单点实现）+ PID 元数据 v1 + scanner 读 InstanceRegistry + 命名统一（pid/state/lock） | ✅ 已实现 |
| Phase 2 | `ExecutableResolver`（显式路径→PATH→平台默认候选）+ 启动链收敛（manager 委托 DaemonLauncher、service 复用 resolver、删除 darwin LaunchdManager 双轨） | ✅ 已实现 |
| Phase 3 | `ProcessChecker` 契约扩展（`GetProcessCommandLine`）+ 平台修正（win 动态缓冲/wmic、linux EACCES/cmdline、darwin procargs2/ps args）+ 两源进程发现（`process_list_*.go` / `discovery.go`）+ 验证分级（confirmed/likely）+ ps 新列 + `stop --all --include-unmanaged` | ✅ 已实现 |
| Phase 4 | force_kill Unix 轮询化（消除 ECHILD 误报）+ 目录平台化 `os.UserConfigDir()`；迁移命令按 0.1 取消 | ✅ 已实现 |

---

## 1. 背景与目标

### 1.1 背景

对现有实现的审查（见 `docs/analysis` 相关分析）发现，CLI 在"找到 mihomo 内核可执行文件"与"发现/验证运行中的内核进程"两条链路上存在若干设计缺陷与跨平台不一致：

- 内核可执行文件只信任 `config.toml` 中的字符串，解析相对路径、默认名、校验点、以及另一条服务安装路径（`internal/service/factory.go` 的 `LookPath`+`Glob`）彼此不一致；
- 同一份 mihomo 配置被三套互不兼容的 hash 命名（PID 文件=截断文件名、lock/state=SHA256[:16]、scanner 反查=SHA256[:8]），导致 `ps` 的 API 端口反查实际失效；
- 进程发现只读"本工具写过的 PID 文件"，手动启动的实例完全不可见；验证只靠可执行名包含 "mihomo"；
- macOS 取进程可执行路径用 `ps -o comm=`（仅命令名）、Windows 用固定 260 字节缓冲，两者信息不完整且与 Linux 的"绝对路径"语义不一致；
- Unix 强杀路径对非子进程调用 `proc.Wait()`，会因 `ECHILD` 误报失败；
- 所有状态文件目录硬编码 `<UserConfigDir>/mihomo-cli`，不遵循平台目录规范；
- 启动/停止/状态存在 `ProcessHandler` / `DaemonLauncher` / `ProcessManager` 三层重叠实现。

### 1.2 目标

1. **配置身份单一化**：同一配置在 PID、锁、状态、扫描反查中只使用一种标识算法，一处实现；`ps` 能可靠展示每个实例对应的配置文件与 API 端口。
2. **可执行文件定位收敛**：全项目只有一套解析规则（配置优先 → PATH 搜索 → 平台默认候选），默认名与解析基准目录平台化，启动校验与服务安装共用同一解析器。
3. **进程发现两源合并**：既能识别"本工具托管的实例"，也能枚举"外部启动的 mihomo 实例"，输出统一结构并标注来源。
4. **进程信息三平台语义一致**：`Executable`=绝对路径、`CmdLine`=真实命令行、`IsVerified` 分级（confirmed / likely），Windows/Linux/macOS 能力对齐。
5. **平台实现缺陷修复**：macOS 完整路径、Windows 长路径、Unix 强杀误报、各平台 `IsProcessRunning` 权限语义一致。
6. **目录平台化**：状态目录遵循 `os.UserConfigDir()` 平台规范，并提供旧目录迁移。
7. **启动链路收敛**：明确 `ProcessHandler`/`DaemonLauncher`/`ProcessManager` 职责，消除重复实现；macOS launchd 与系统服务模块合并去重。

### 1.3 非目标

- 不支持 FreeBSD / OpenBSD / Android 等平台（若要支持需另补平台文件，见第 8 节说明，本期不做）；
- 不做内核二进制自动下载、版本匹配、内置内核（内核仍由用户提供并配置路径）；
- 不改动 API 客户端与业务命令（mode/proxy/config 等）的行为；
- 不做 Windows 服务/SysProxy 等既有权限模型的调整，仅复用其路径解析能力。

---

## 2. 现状问题清单与根因

| # | 问题 | 位置（当前代码） | 影响 | 根因 |
|---|---|---|---|---|
| P1 | 配置身份 hash 三套分裂：PID 文件=文件名截断 8 字符；lock/state=SHA256[:16]；scanner 反查=SHA256[:8] | `config/path_resolver.go:89-110`、`config/paths.go:105-129`、`internal/mihomo/state.go:287-304`、`internal/mihomo/scanner.go:24-51,162-202` | `ps` 的 APIPort 基本恒为"未知"；同一配置不同模块标识不互通；8 字符截断易碰撞 | 无统一 identity 模块，各文件自造算法；`RegisterConfigHash` 无调用者 |
| P2 | 进程发现盲区：只读 `*.pid` 文件 | `internal/mihomo/scanner.go:54-131` | 手动/第三方启动的 mihomo 完全不可见；违背"实时查询"定位 | 无全进程枚举能力，依赖自家 PID 文件 |
| P3 | 验证仅靠可执行名 contains "mihomo" | `scanner.go:102`、`VerifyMihomoProcess` `scanner.go:150-159` | 误报/误判；PID 复用时可能误伤 | 无元数据可比对（config 路径、启动参数） |
| P4 | `ProcessInfo.CmdLine` 被填成 `ExecPath`，未读真实命令行 | `scanner.go:125`、`process_info.go` | 无法区分同 exe 多实例/不同 `-f` 配置 | 无跨平台取命令行能力 |
| P5 | 可执行文件定位不一致：start 只信配置；服务安装另用 LookPath+Glob；无平台默认名；相对路径基于 CLI CWD | `internal/mihomo/process_handler.go:46-49`、`daemon_launcher.go:52-65`、`internal/service/factory.go:66-136`、`config/toml_config.go:179` | 同一环境 start 与 service 可能指向不同内核；换目录执行结果不同；Linux/macOS 默认名不可用 | 无统一 ExecutableResolver |
| P6 | macOS `ps -o comm=` 只返回命令名 | `internal/mihomo/process_darwin.go:40-60` | ExecPath 非绝对路径，与 win/linux 语义不一致 | 未使用 libproc `proc_pidpath` |
| P7 | Windows `QueryFullProcessImageName` 固定 260 缓冲，不处理 `ERROR_INSUFFICIENT_BUFFER` | `process_windows.go:72-99` | 长路径实例被 ps 静默跳过 | 未按返回码动态扩容 |
| P8 | Linux `IsProcessRunning` 对无权限读 `/proc/<pid>/stat` 一律判死 | `process_linux.go:23-36` | 高权限用户进程被误判不存在 | 未区分 EACCES 语义 |
| P9 | Unix 强杀对非子进程 `proc.Wait()` → ECHILD 误报失败 | `force_kill_platform_unix.go:14-45` | `stop -F` / 健康检查超时回收路径误报 | Wait 仅适用于 Start 的子进程 |
| P10 | 状态目录硬编码 `<UserConfigDir>/mihomo-cli` | `config/paths.go:22-51`、`path_resolver.go:18-27` | Linux 忽略 XDG_CONFIG_HOME；Windows 不用 AppData | 未用 `os.UserConfigDir()` |
| P11 | 三层启动实现重叠 + 重复代码 | `manager.go`、`daemon_launcher.go`、`process_handler.go`；`config/paths.go` 与 `path_resolver.go` 重复 | 行为漂移风险、维护成本 | 历史演进未收敛 |
| P12 | darwin `LaunchdManager` 未接线，与 setsid 守护语义并存 | `daemon_darwin.go:208-327` | 维护两套"常驻"方案 | 未做决策 |

---

## 3. 总体改造架构

### 3.1 设计原则

1. **单一事实源**：一个配置身份 → 一种算法 → 一处实现；一份解析规则 → 一个 Resolver。
2. **语义契约先行**：先定义跨平台接口语义（绝对路径、真实命令行、分级验证），再逐平台实现，禁止平台私有"捷径"污染上层。
3. **元数据优先**：PID 文件从"纯数字"升级为"结构化元数据"，让进程发现无需 hash 反推即可拿到 config/exec/端口信息。
4. **只读兼容**：旧格式文件（纯数字 PID、旧目录、旧 hash 命名）只做"可读 + 可清理"，不保证向后写入。
5. **每阶段可独立合入、可编译、可验证**。

### 3.2 改造后的模块图

```text
cmd (start/stop/status/ps/cleanup/service)
   │  仅依赖编排层
   ▼
内部编排层 internal/mihomo
   ProcessHandler ──► DaemonLauncher（唯一启动入口，替代 manager.go 的重复逻辑）
        │                     │
        ▼                     ▼
   ExecutableResolver   DaemonManager 接口（平台实现：windows/linux/darwin）
   （internal/config）         │ 写入 PID v2 元数据
        │                     ▼
        ▼               InstanceRegistry（PID 元数据读写）
   Identity（配置身份算法，internal/config/identity.go）
        ▲ 供 pid/lock/state 文件命名与扫描
        │
进程发现层 internal/mihomo
   ProcessDiscovery
     ├─ ManagedScanner：遍历 InstanceRegistry（PID 元数据）
     └─ ExternalScanner：全进程枚举（platform: process_list_*.go）
         └─ ProcessChecker 接口（IsProcessRunning / GetProcessExecutable /
            GetProcessCommandLine，语义三平台一致）
```

### 3.3 关键设计决策

| 决策 | 内容 | 理由 |
|---|---|---|
| D1 | 配置身份 = SHA256(规范化绝对路径)[:12]，收敛到 `internal/config/identity.go` 唯一实现 | 48bit 碰撞概率可忽略；与旧命名可区分（旧=截断文件名/SHA256[:16]）；单点实现杜绝再分裂 |
| D2 | PID 文件升级为 JSON 元数据（文件扩展名保持 `.pid`，内容可自描述），旧纯数字格式可读 | 进程发现不再需要"文件名→配置文件"反推；端口/配置/启动时间直接可得 |
| D3 | 新增 `ExecutableResolver`：`config.toml` 显式路径（相对 config.toml 所在目录）→ `exec.LookPath("mihomo")` → 平台默认候选目录；start 与服务安装共用 | 统一查找语义与默认名，消除 P5 |
| D4 | `ProcessChecker` 接口扩展 `GetProcessCommandLine`；`GetProcessExecutable` 三平台统一返回绝对路径 | 语义对齐（P4/P6），CmdLine 用于多实例区分与验证 |
| D5 | `IsVerified` 分级：`confirmed`（命中元数据或配置匹配）> `likely`（可执行名/命令行匹配）；停止默认只作用于 managed 实例 | 避免误杀外部实例（P2/P3） |
| D6 | 状态目录改用 `os.UserConfigDir()/mihomo-cli`，保留旧目录只读回退 + 一键迁移 | 平台规范（P10） |
| D7 | 启动链收敛：`ProcessHandler`（命令编排）→ `DaemonLauncher`（唯一启动/停止封装）；`ProcessManager` 退化为 DaemonLauncher 的薄包装供 lifecycle 使用；darwin LaunchdManager 移入系统服务模块 | 消除 P11/P12 |

---

## 4. 分模块详细设计

### 4.1 配置身份统一（D1/D2）—— 解决 P1、并为 P2/P3 打基础

#### 4.1.1 新增 `internal/config/identity.go`

```go
// IdentityOf 计算配置身份。identity 是规范化的绝对路径，保证同一配置文件
// 无论以何种相对路径写入都得到同一身份。
func IdentityOf(configFile string) string {
    abs := absPath(configFile)          // filepath.Abs + filepath.Clean
    sum := sha256.Sum256([]byte(abs))
    return hex.EncodeToString(sum[:])[:12] // 48bit
}

// 旧格式识别（仅用于迁移与清理，不用于写入）：
func IsLegacyPIDFileName(name string) bool { ... } // 匹配非 12 位 hex 的 mihomo-*.pid
```

约定：

- **算法与位置唯一**：删除 `config/paths.go` 与 `config/path_resolver.go` 中两份私有的 `generateConfigHash`、`internal/mihomo/state.go` 的 `generateConfigHash`（SHA256[:16]）、`internal/mihomo/scanner.go` 的 `computeConfigHash`，全部改为调用 `config.IdentityOf`。
- **文件命名规则**（唯一写入规则）：
  - PID 元数据：`mihomo-<identity>.pid`
  - 状态文件：`state-<identity>.json`
  - 锁文件：`lock-<identity>`
  - 空配置（`ConfigFile == ""`）沿用默认名 `mihomo.pid` / `state-default.json` / `lock-default`，但元数据中 `configFile` 记为 `<default>`。

#### 4.1.2 PID 文件升级为元数据（v2），新增 `InstanceRegistry`

位置：`internal/mihomo/instance.go`（新增），替代 `daemon_common.go` 中 `PIDFileManager` 的读写职责。

```go
// InstanceMeta PID 元数据 v2（写入 .pid 文件，JSON）
type InstanceMeta struct {
    Version    int    `json:"version"`              // = 2
    PID        int    `json:"pid"`
    ConfigFile string `json:"config_file,omitempty"` // 规范化绝对路径；"" 或 "<default>"
    ExecPath   string `json:"exec_path,omitempty"`   // 内核可执行文件绝对路径
    APIAddr    string `json:"api_addr,omitempty"`    // external-controller，如 127.0.0.1:9090
    StartedAt  string `json:"started_at,omitempty"`  // RFC3339
}

type InstanceRegistry struct {
    pathResolver *config.PathResolver
}
func (r *InstanceRegistry) Save(meta InstanceMeta) error      // 原子写（先写 .tmp 再 rename）
func (r *InstanceRegistry) Load(configFile string) (*InstanceMeta, error)
func (r *InstanceRegistry) List() ([]InstanceMeta, error)     // 遍历 *.pid，兼容旧纯数字格式
func (r *InstanceRegistry) Remove(configFile string) error
```

兼容读取规则（`List`/`Load` 内实现）：

1. 内容可 `json.Unmarshal` 且 `Version == 2` → 直接使用；
2. 否则按旧纯数字 PID 解析（`fmt.Sscanf`），得到一个 `InstanceMeta{PID: n, ConfigFile: ""}`（`configFile` 未知）；
3. 文件名不符合新命名但可解析出旧截断名（`mihomo-test-mih.pid` 形式）的，仅用于展示为 legacy 实例，不参与反查。

写入方收敛：`daemon_common.go` 的 `PIDFileManager`（`Save/Read/Cleanup`）改为委托 `InstanceRegistry`，启动成功后写入完整元数据（PID、config、exec、apiAddr、时间戳）。停止/清理时删除对应元数据文件。

#### 4.1.3 scanner 重写（反查不再依赖 hash）

`internal/mihomo/scanner.go` 中 `ScanMihomoProcesses` 改为：

1. `InstanceRegistry.List()` 得到 managed 候选（含 configFile/apiAddr/execPath）；
2. 对每个 PID 做 `IsProcessRunning` + `GetProcessExecutable` 一致性校验（见 4.4.3）；
3. `getConfigPathFromHash` / `configHashMapping` / `RegisterConfigHash` / `computeConfigHash` 全部删除（不再需要"文件名→配置"反推）；`extractAPIPortFromConfig` 保留，仅用于旧格式或元数据缺失时兜底（直接以元数据中的 configFile 解析）。
4. `CleanupPIDFiles` 改为 `InstanceRegistry.List()` + 进程存活检查（逻辑不变，源数据变化），并**额外清理与任何有效实例都不对应的 legacy 文件**（一次性迁移）。

#### 4.1.4 涉及文件清单

| 动作 | 文件 | 说明 |
|---|---|---|
| 新增 | `internal/config/identity.go` | 唯一身份算法 + legacy 识别辅助 |
| 新增 | `internal/mihomo/instance.go` | `InstanceMeta` / `InstanceRegistry` |
| 改 | `internal/mihomo/scanner.go` | 扫描改读 InstanceRegistry；删除 hash 反查链 |
| 改 | `internal/mihomo/daemon_common.go` | `PIDFileManager` 委托 InstanceRegistry |
| 改 | `internal/mihomo/daemon_launcher.go` | 启动成功后写完整元数据 |
| 改 | `internal/config/paths.go` / `path_resolver.go` | `GetPIDFilePath` 用 `config.IdentityOf`；删除私有 hash |
| 改 | `internal/mihomo/state.go` / `lock.go` | 命名改用 `config.IdentityOf`；删除本地 hash 实现 |
| 改 | `docs/pid-file-management.md` | 同步新格式说明 |

### 4.2 可执行文件解析器（D3）—— 解决 P5

#### 4.2.1 新增 `internal/config/executable_resolver.go`

```go
type ExecutableResolver struct {
    // baseDir：解析相对路径的基准目录。默认取 config.toml 所在目录；
    // 无配置文件时取进程 CWD。
    baseDir string
}

// Resolve 返回规范化绝对路径 + 证据链（命中方式），便于报错诊断。
func (r *ExecutableResolver) Resolve(cfg *config.TomlConfig) (string, ResolveSource, error)
```

查找顺序（命中即停，返回 `source`）：

1. `cfg.Mihomo.Executable` 非空：
   - 相对路径 → 相对 `r.baseDir`（config.toml 目录）解析为绝对路径；
   - 校验 `os.Stat` 存在（Windows 追加 `.exe` 尝试：若用户写 `mihomo` 而文件为 `mihomo.exe`，自动补全并提示）。
2. 配置为空 → `exec.LookPath("mihomo")`（Windows 上 Go 会按 PATHEXT 自动补 `.exe`）。
3. 仍未命中 → 平台默认候选目录探测（顺序）：
   - CLI 可执行文件同目录（`os.Executable()` 目录）；
   - 进程 CWD；
   - Linux/macOS 追加 `/usr/local/bin`、`/usr/bin`；
   - 名称取平台默认：Windows `mihomo.exe`，其余 `mihomo`。
4. 全部失败 → 返回聚合错误（列出已尝试的每个路径与原因），错误文案引导用户检查 `config.toml [mihomo] executable`。

新增配置项（`[mihomo]`，可选）：`executable_search = true|false`（默认 true，允许用户强制只使用显式路径）。

#### 4.2.2 复用点

- `internal/mihomo/process_handler.go Start`：删除第 46-49 行裸 `os.Stat`，改为 `ExecutableResolver.Resolve`（单一校验点）；校验通过后统一由 resolver 返回的绝对路径进入后续启动，**消除"os.Stat 相对 CWD 校验、filepath.Abs 再解析"的两步不一致**。
- `internal/service/factory.go findMihomoExecutable`：删除私有 LookPath+Glob 逻辑，改调 `ExecutableResolver`（语义：以"安装位置与服务安装时的 CWD"为 baseDir）。`internal/mihomo` 与 `internal/service` 两个包对 resolver 的依赖方向保持一致（resolver 属于 `internal/config`，无环依赖）。
- `DaemonLauncher` / `ProcessManager`：`GetExecutablePath` 改为基于 resolver 结果的缓存字段，不再各自 `filepath.Abs`。

#### 4.2.3 默认值平台化

- `GetDefaultTomlConfig`（`config/toml_config.go:179`）默认 `Executable` 置空 + `executable_search = true`，由 resolver 在运行时按平台选择默认名与候选目录；`config.toml` 注释示例改为同时给出三平台说明。旧配置中显式 `mihomo.exe` 仍按显式路径处理（行为不变）。

### 4.3 ProcessChecker 平台修正（D4/D5）—— 解决 P4/P6/P7/P8

#### 4.3.1 统一契约（`internal/mihomo/process.go`）

```go
type ProcessChecker interface {
    // IsProcessRunning：进程存在即 true；权限不足时保守返回 true（各平台语义一致）。
    IsProcessRunning(pid int) bool
    // GetProcessExecutable：返回规范化绝对路径（macOS 不再返回命令名）。
    GetProcessExecutable(pid int) (string, error)
    // GetProcessCommandLine：返回完整命令行（含参数，可含空格）；不可用时返回可读错误。
    GetProcessCommandLine(pid int) (string, error)
}
```

#### 4.3.2 Windows（`process_windows.go`）

- `GetProcessExecutable`：改用动态缓冲——先 `MAX_PATH` 调用；若返回 `ERROR_INSUFFICIENT_BUFFER`，按 `size` 返回值扩容重试（上限 32K，超限报错而非静默跳过）；成功路径统一去掉可能的 `\\?\` 前缀（`strings.TrimPrefix(path, `\\?\`)`），保证与配置/解析器产出的路径可做字符串级比对。
- `IsProcessRunning`：保持现状语义（Access Denied → true），并把该语义在注释与文档中明确为三平台约定（Linux/macOS 见下）。
- 新增 `GetProcessCommandLine`（务实降级实现）：优先 `wmic process where processid=<pid> get commandline`（已弃用但稳定、无额外依赖；解析输出取首行）；若调用失败/为空返回错误。**不引入 PEB/ReadProcessMemory 方案**（复杂度与权限要求不成比例，且多实例区分已由 InstanceMeta 承担）。

#### 4.3.3 Linux（`process_linux.go`）

- `IsProcessRunning`：读 `/proc/<pid>/stat` 失败时区分：`os.IsPermission` → 返回 true（保守存活，与 Windows 对齐）；`os.IsNotExist`/其他 → false。
- `GetProcessExecutable` 保持 `readlink /proc/<pid>/exe`，回退逻辑不变。
- 新增 `GetProcessCommandLine`：读 `/proc/<pid>/cmdline`，以 `\x00` 切分后以空格 join（首个元素为 argv[0]）；`/proc/<pid>/cmdline` 为空（内核线程/僵尸）返回错误。

#### 4.3.4 macOS（`process_darwin.go`）

- `GetProcessExecutable` 主实现改为 **libproc `proc_pidpath`**（cgo-free，走 `syscall` 加载 `/usr/lib/libproc.dylib`，`LazyDLL` + `NewProc`）：

```go
// 示意（无 cgo）：
var libproc          = syscall.NewLazyDLL("/usr/lib/libproc.dylib")
var procPidpath      = libproc.NewProc("proc_pidpath")

func (d *darwinProcessChecker) GetProcessExecutable(pid int) (string, error) {
    var buf [4096]byte // PATH_MAX
    n, _, err := procPidpath.Call(uintptr(pid), uintptr(unsafe.Pointer(&buf[0])), uintptr(len(buf)))
    if n <= 0 { ... 回退策略 ... }
    return strings.TrimRight(string(buf[:n]), "\x00"), nil
}
```

  回退策略：`proc_pidpath` 失败（权限等）→ 保留现有 `ps -p <pid> -o comm=`，但仅作为 `likely` 验证输入（见 4.4.3），不得作为 `Executable` 返回。
- `IsProcessRunning`：保持 Signal(0)，但补充 `ESRCH`→false、`EPERM`→true（保守存活）的显式区分，对齐三平台语义。
- 新增 `GetProcessCommandLine`：`ps -p <pid> -o args=`（完整 argv）。

### 4.4 进程发现两源合并与验证分级（D2/D5）—— 解决 P2/P3

#### 4.4.1 新增 `internal/mihomo/discovery.go`

```go
type Source string
const (
    SourceManaged  Source = "managed"  // 由本 CLI 启动并留有元数据
    SourceExternal Source = "external" // 全进程表枚举发现
)

type DiscoveredProcess struct {
    ProcessInfo                      // 保留原字段，修正 CmdLine 语义
    Source    Source   `json:"source"`
    ConfigFile string  `json:"config_file,omitempty"` // managed 必有；external 尽力而为
    Verified  VerifyLevel `json:"verified"`           // confirmed / likely
}

// ScanManaged：InstanceRegistry.List() + 进程存活/一致性校验。
// ScanExternal：平台全进程枚举 + 名称/命令行过滤（见 4.4.2）。
// ScanMihomoProcesses 返回两者合并（managed 在前，去重按 PID+execPath）。
```

`ProcessInfo` 字段调整（`process_info.go`）：

- `CmdLine`：改为真实命令行（4.3 提供的 `GetProcessCommandLine`）；无法获取时置空而非伪造。
- 新增 `StartTime` 已有字段补全来源；`APIPort` 仅由元数据/配置解析（不再依赖文件名反查）。

#### 4.4.2 平台全进程枚举（新增 `process_list_*.go`）

| 平台 | 实现 | 说明 |
|---|---|---|
| Windows | `CreateToolhelp32Snapshot(TH32CS_SNAPPROCESS)` + `Process32First/Next` | 标准、无额外依赖；返回 pid 列表 |
| Linux | 遍历 `/proc/[0-9]+/stat` | 读目录 + 数字名过滤 |
| macOS | `sysctl(KERN_PROCALL)` 经 `golang.org/x/sys/unix`（`SysctlRaw` + `KinfoProc`） | 返回 `p_pid` 列表 |

枚举后过滤条件（`ScanExternal`）：`GetProcessExecutable` 基名小写含 `mihomo` **或** `GetProcessCommandLine` 含 `mihomo`（覆盖改名场景，如 `clash-meta`/自改名副本 → 仅在命令行出现 `mihomo` 时命中）。过滤只读信息，不产生副作用。

#### 4.4.3 验证分级

| 级别 | 判定 | 用途 |
|---|---|---|
| `confirmed` | 命中 InstanceMeta 且 `GetProcessExecutable` 与元数据 `ExecPath` 一致（规范化后比对）；或 config 显式指定且进程 argv 中的 `-f` 参数一致 | `stop`（无参，默认只停当前配置实例）与 `stop --all` 默认集合；`status` 判运行 |
| `likely` | 仅名称/命令行匹配（external 或 legacy 元数据） | `ps` 展示为"未托管/未验证"；`stop` 需显式 `--pid` 才允许 |

`ps` 输出新增"来源"列（managed/external）与验证列（confirmed/likely），原"已验证"列语义更新为 confirmed 计数。`StopAllMihomoProcesses` 默认只处理 confirmed；`--all --include-unmanaged` 才扩展到 likely（明确提示风险）。

### 4.5 Unix 强杀路径修正（force_kill）—— 解决 P9

`internal/mihomo/force_kill_platform_unix.go`：删除 `os.FindProcess` + `proc.Wait()` 组合（Wait 仅对 `Start` 的子进程有效，对 `FindProcess` 得到的非子进程返回 `ECHILD`），改为与 Windows 实现同构的**轮询等待**：

```go
func forceKillPlatform(pid int, timeout time.Duration) error {
    if !IsProcessRunning(pid) {
        output.Info("Process %d already exited", pid)
        return nil
    }
    // 优先 SIGKILL（与现有分级终止一致）
    if err := syscall.Kill(pid, syscall.SIGKILL); err != nil && err != syscall.ESRCH {
        return pkgerrors.ErrService("failed to kill process", err)
    }
    deadline := time.Now().Add(timeout)
    for time.Now().Before(deadline) {
        if !IsProcessRunning(pid) { return nil }
        time.Sleep(100 * time.Millisecond)
    }
    return pkgerrors.ErrService(fmt.Sprintf("wait for process exit timeout after %v", timeout), nil)
}
```

注：`SIGKILL` 发送成功即视为完成终止动作，等待仅用于确认退出；进程已退出（ESRCH）不算错误。该函数同时服务于 `launcher.Stop(force)`（`daemon_launcher.go:187`）与健康检查超时回收（`process_handler.go:155`）两条路径。

### 4.6 目录平台化与迁移（D6）—— 解决 P10

#### 4.6.1 目录规则

`internal/config/paths.go` 与 `path_resolver.go` 统一为单一来源 `config.PathResolver`：

```go
// baseDir 计算规则（新增 resolveBaseDir）：
// 1. 环境变量 MIHOMO_CLI_CONFIG_DIR 显式覆盖（调试/多账号场景）；
// 2. 否则 os.UserConfigDir()（Go 标准实现，按平台解析）：
//    - Windows: %AppData%\Roaming   → %AppData%\Roaming\mihomo-cli
//    - macOS:   ~/Library/Application Support → ~/Library/Application Support\mihomo-cli
//    - Linux:   $XDG_CONFIG_HOME 或 ~/.config → $XDG_CONFIG_HOME\mihomo-cli
// 3. 若新目录为空而旧目录（<UserHomeDir>/.config/.mihomo-cli）存在 → 迁移提示 + 读旧目录兜底。
```

具体动作：

- `Paths`/`PathResolver` 合并：删除 `config/paths.go` 的包级重复函数（`GetBaseDir`/`GetPIDDir`/`GetPIDFilePath`/`GetPaths` 等），全部走 `PathResolver` 实例方法；`paths.go` 仅保留 `Paths` 结构与兼容薄封装（`GetPaths()` 若仍有调用者，返回与 resolver 一致的路径）。
- `GetConfigFile`（CLI 自身 `config.yaml`）与 `config.toml` 搜索路径同步使用新 baseDir（`FindTomlConfigPath` 第三候选，`toml_config.go:135-141`）。
- 新增 `internal/config/paths_migrate.go`：`DetectLegacyDir() (legacy string, hasData bool)` 与 `MigrateLegacyDir(dryRun bool) (moved int, err error)`，仅迁移本工具自身文件（`.pid/.json/.yaml/.toml/backups/history` 白名单），不做破坏性删除；`cleanup` 命令可在确认后清空旧目录。

#### 4.6.2 迁移策略

1. 首次运行任一命令时自动检测：新目录存在任何文件则直接使用新目录；
2. 新目录为空 + 旧目录有数据 → 打印提示并**只读**旧目录（保持现状可用），不静默拷贝；
3. 用户执行 `mihomo-cli config migrate`（新命令，随 Phase 4 实现）后完成搬迁与 PID 元数据刷新。

### 4.7 启动链路收敛与 launchd 处置（D7）—— 解决 P11/P12

#### 4.7.1 职责收敛（文件级）

| 层 | 归属 | 收敛后职责 |
|---|---|---|
| 命令编排 | `ProcessHandler`（`process_handler.go`） | 仅做：配置加载校验（含 ExecutableResolver 单点校验）、TUN/TProxy 备份、健康检查编排；内部只调用 DaemonLauncher |
| 启动封装 | `DaemonLauncher`（`daemon_launcher.go`） | 唯一拥有"启动/停止/GetRunningPID/GetStatus"实体逻辑；`Start` 内部复用 ExecutableResolver 结果 + InstanceRegistry 写元数据 |
| 生命周期 | `ProcessManager`（`manager.go`） | 删除其 `Start` 复制逻辑（第 37-75 行重复 DaemonLauncher.Start），改为持有 DaemonLauncher 并转发；保留 `GetPIDFromPIDFile` 供 `LifecycleManager`（`lifecycle.go:54`）与 signal 处理使用，实现改为 `InstanceRegistry.Load` |
| 平台守护 | `DaemonManager` 接口及 win/linux/darwin 实现 | 保持现状（daemonize/IO/信号语义已合理），仅替换 PID 写入为元数据 |

删除/停用：`daemon_darwin.go` 中 `LaunchdManager` 相关类型与方法（L208-319），其能力并入 `internal/service` 的"安装为系统服务"路径（launchd plist 生成逻辑移到 `internal/service/service_darwin.go`，作为 `--install-service` 的实现，由服务工厂按 D3 resolver 提供 exec 路径）。不接线、不双轨，`GetDaemonManager`（darwin 版）仅保留 setsid 守护实现。

#### 4.7.2 重复代码清理

- `config/paths.go` 与 `path_resolver.go` 合并（见 4.6.1）；
- `internal/mihomo/scanner.go` 与 `daemon_common.go` 中与 `InstanceRegistry` 重复的 PID 读写全部删除；
- 迁移验证以 `go build ./...` + `staticcheck`（若项目已配置）+ 单测为门槛。

---

## 5. 数据格式与兼容性

### 5.1 PID 文件 v2 JSON（示例）

```json
{
  "version": 2,
  "pid": 12345,
  "config_file": "C:/Users/dev/proxy/mihomo-config.yaml",
  "exec_path": "C:/Users/dev/proxy/mihomo.exe",
  "api_addr": "127.0.0.1:9090",
  "started_at": "2026-09-05T10:00:00+08:00"
}
```

### 5.2 兼容矩阵

| 场景 | 旧版本写入 | 新版本写入 | 说明 |
|---|---|---|---|
| 旧纯数字 `.pid`（截断文件名） | 读 ✅（降级为 legacy，无 configFile） | 不写 | `cleanup` 一次性清理 |
| 旧 hash 命名的 state/lock 文件 | 读 ✅（仅命名解析不同，内容格式不变） | 不写 | 迁移后删除 |
| 新 v2 `.pid` | 旧版不可读（安全跳过：纯数字解析失败则不误杀） | 写 ✅ | 向前兼容由"保守跳过"保证 |
| 旧目录（`<UserConfigDir>/mihomo-cli`） | — | 只读回退 + 迁移命令 | 见 4.6.2 |

**向后兼容承诺**：新版本读到旧格式文件时，任何解析失败都必须"跳过 + 提示"而非"当作错误中断"或"按错误 PID 执行 kill"——这是 P2/P3 之外另一个安全底线（防止 PID 复用误杀）。

---

## 6. 分阶段实施计划

每个阶段独立合入、独立可验证；阶段间无强耦合时允许并行，但建议按顺序执行以免合并冲突集中在 scanner/daemon 文件上。

### Phase 1 —— 配置身份统一 + PID 元数据 v2 + scanner 重写（解决 P1，为 P2/P3 奠基）

**改动文件**：新增 `internal/config/identity.go`、`internal/mihomo/instance.go`；改 `internal/config/paths.go`、`path_resolver.go`、`internal/mihomo/scanner.go`、`daemon_common.go`、`daemon_launcher.go`、`state.go`、`lock.go`、`manager.go`（读路径）；更新 `docs/pid-file-management.md`。

**实施要点**

1. `identity.go`：实现 `IdentityOf` + `IsLegacyPIDFileName`，配套单测（相对路径/绝对路径/`..` 规范化后同身份、空路径语义、长度/碰撞）。
2. 替换全部私有 hash（paths.go、path_resolver.go、state.go、lock.go、scanner.go），行为断言：同一 `configFile` 在新旧代码路径产出的 PID 文件名变化**只发生一次**（8 字符截断名 → 12 位 hex 名），旧文件不删除只降级。
3. `instance.go`：`Save/Load/List/Remove` + 旧格式降级读取；`Save` 原子写。
4. scanner 重写：删除 `configHashMapping`/`RegisterConfigHash`/`getConfigPathFromHash`/`computeConfigHash`；`extractAPIPortFromConfig` 保留作兜底。
5. daemon 写入点：`StartAsDaemon` 成功后写完整元数据（exec/config/api 由 DaemonManagerBase 提供）。

**验收**

- `go build ./...`、`go vet ./...`、`go test ./internal/config/... ./internal/mihomo/...`；
- 单测覆盖：identity 幂等性；InstanceRegistry 旧格式（纯数字）读取、损坏文件跳过；scanner 在"仅旧文件"与"仅新文件"两种 fixture 下输出正确；
- 手工：`start` 后 `ps` 能看到 API 端口与配置文件（此前为"未知"）；`stop`/`status` 行为与改造前一致。

### Phase 2 —— ExecutableResolver + 启动链收敛（解决 P5/P11）

**改动文件**：新增 `internal/config/executable_resolver.go`；改 `internal/mihomo/process_handler.go`、`daemon_launcher.go`、`manager.go`、`config/toml_config.go`（默认值）、`internal/service/factory.go`（复用 resolver）；删 `daemon_darwin.go` LaunchdManager（或移至 service，见 4.7.1）。

**实施要点**

1. resolver 单测（Go 标准 `t.TempDir()` 构建 fixture）：显式相对路径以 config 目录为基准、LookPath 命中、平台候选目录探测、`.exe` 自动补全、聚合错误信息。
2. `process_handler.go` 删除裸 `os.Stat`；校验与 exec 路径同一来源。
3. `manager.go` 的 `Start` 删除复制逻辑，改委托 `DaemonLauncher`；保留 `GetPIDFromPIDFile` 但走 `InstanceRegistry.Load`。
4. launchd 代码迁移到 `internal/service/service_darwin.go`（若 Phase 3/4 时间紧，可先删除 LaunchdManager 并记录 issue，绝不留"双轨死代码"）。

**验收**

- 三平台交叉编译：`GOOS=windows GOARCH=amd64`、`GOOS=linux GOARCH=amd64`、`GOOS=darwin GOARCH=arm64` 下 `go build ./...` 全通过；
- 手工：Linux 上不带 executable 配置（默认空）执行 `start`，能从 PATH 命中并给出 source 提示；配置 `./mihomo.exe` 在非 Windows 平台给出可读诊断。

### Phase 3 —— ProcessChecker 修正 + 两源进程发现（解决 P4/P6/P7/P8/P2/P3）

**改动文件**：改 `internal/mihomo/process.go`、`process_windows.go`、`process_linux.go`、`process_darwin.go`、`process_other.go`（契约注释同步）；新增 `internal/mihomo/process_list_windows.go`、`process_list_linux.go`、`process_list_darwin.go`、`process_list_other.go`（返回空列表 + 明确不支持）；新增 `internal/mihomo/discovery.go`；改 `scanner.go`（两源合并）、`process_info.go`、`cmd/ps.go`（新列）、`daemon_common.go`（`StopAllMihomoProcesses` 的 confirmed 语义）、`cmd/stop` 帮助文案（`--include-unmanaged`）。

**实施要点**

1. 按 4.3 契约逐平台实现，先写契约级单测（mock checker：running/exec/cmdline 三元组）再写平台实现。
2. macOS `proc_pidpath`（无 cgo）如遇符号加载问题，回退实现必须显式降级为 `likely`，不允许用 `comm=` 冒充绝对路径。
3. 全进程枚举是只读遍历，Windows 注意 `Process32Next` 循环终止条件；Linux 目录遍历用数字名正则过滤。
4. `ProcessInfo.CmdLine` 修正后检查所有消费方（`cmd/ps.go` 仅展示，`process_info.go` 序列化）无破坏。

**验收**

- 单测：mock + 平台真实进程冒烟（本机各平台各跑一次 `ps`，断言 Linux/macOS 的 ExecPath 为绝对路径且与 `/proc`/`proc_pidpath` 一致；Windows 长路径目录下放一个副本再 `ps`，断言不再被跳过）；
- `go test ./...` 全绿；`ps` 输出三平台字段对齐（Exec/CmdLine/端口/来源/验证级）。

### Phase 4 —— 强杀修正 + 目录平台化 + 迁移命令（解决 P9/P10）

**改动文件**：改 `internal/mihomo/force_kill_platform_unix.go`、`config/paths.go`、`path_resolver.go`（合并）、`config/toml_config.go`（FindTomlConfigPath 第三候选）；新增 `internal/config/paths_migrate.go`、`cmd/migrate.go`（`mihomo-cli config migrate`）；改 `cmd/cleanup.go`。

**实施要点**

1. force kill 轮询化（4.5 代码块）；单测用短 timeout + 自 fork 子进程验证"已退出不算错误"。
2. 目录切换遵循 4.6.2 三段策略；`config migrate` 默认 `--dry-run`，实际迁移需 `--apply`。
3. `cleanup` 增加旧目录残留清理分支（仅白名单文件）。

**验收**

- Linux/macOS 手工：`stop -F <pid>` 在进程已先被外部 kill 的场景返回成功而非 ECHILD 错误；
- Windows：`%AppData%\Roaming\mihomo-cli` 生效；Linux：设 `XDG_CONFIG_HOME=/tmp/xdg` 后状态文件落在其下；
- 迁移冒烟：旧目录 fixture → 新目录为空 → `config migrate --apply` 后 start/status 正常。

---

## 7. 测试与验证矩阵

### 7.1 自动化单测（新增清单）

| 模块 | 用例 |
|---|---|
| `identity` | 同配置多种相对写法同身份；不同配置不同身份；空配置默认名 |
| `InstanceRegistry` | v2 读写；旧纯数字读取；损坏/非 UTF8 跳过；原子写（中断无半文件） |
| `ExecutableResolver` | 显式/空/LookPath/候选目录/`.exe` 补全/聚合错误 |
| `ProcessChecker`（mock） | 三平台语义一致：权限不足→保守存活；CmdLine 空→可读错误 |
| `Discovery` | managed+external 合并去重；confirmed/likely 分级 |
| `force_kill` | 已退出进程不报错；超时返回；真实子进程 kill 冒烟 |
| `paths_migrate` | 目录选择优先级；dry-run 不落盘 |

### 7.2 三平台手工验证矩阵

| 用例 | Windows | Linux | macOS |
|---|---|---|---|
| start → ps 显示 API 端口与配置 | ✅ | ✅ | ✅ |
| start 后 ExecPath 与配置/进程一致（绝对路径） | ✅ QueryFullProcessImageName | ✅ /proc/pid/exe | ✅ proc_pidpath |
| 长路径/含空格目录实例可见 | ✅ 动态缓冲 | — | — |
| 外部手动启动 mihomo → ps 显示 external/likely | ✅ | ✅ | ✅ |
| `stop` 默认不杀 external；`stop -F <external pid>` 显式可杀 | ✅ | ✅ | ✅ |
| 权限受限进程（admin 起的进程用普通用户 CLI 查看） | ✅ 保守存活 | ✅ EACCES 保守存活 | ✅ EPERM 保守存活 |
| CmdLine 展示真实 `-f` 参数 | ✅ wmic（尽力） | ✅ /proc | ✅ ps args |
| XDG/AppData 目录规范 | ✅ AppData\Roaming | ✅ $XDG_CONFIG_HOME | ✅ Application Support |
| 旧目录/旧 pid 文件不误杀、可迁移 | ✅ | ✅ | ✅ |
| 交叉编译（含本机） | ✅ | ✅ | ✅ |

### 7.3 回归关注点

- `status`/`stop` 语义不变（默认操作"当前配置实例"，行为与 Phase 0 基线一致）；
- 并发启动互斥依赖 lock 命名变更，需回归"重复 start 报已在运行"；
- 服务安装（Windows Service）在 Phase 2 后指向与 `start` 相同的内核解析结果。

---

## 8. 风险与回滚

| 风险 | 影响 | 缓解 |
|---|---|---|
| PID 文件格式升级导致旧版 CLI 与管理中实例失联 | 旧版误判实例未运行 | 5.2 兼容矩阵"保守跳过"；新文件保持 `.pid` 扩展名；发布说明提示升级顺序 |
| 目录迁移后旧脚本硬编码路径失效 | 用户脚本破坏 | 迁移命令打印新旧路径映射；`MIHOMO_CLI_CONFIG_DIR` 逃生舱；`cleanup` 不自动删旧目录 |
| 全进程枚举在低权限环境部分受限 | external 扫描不完整 | 枚举失败仅降级 external 为空并提示，不影响 managed；不引入提权 |
| `proc_pidpath` 在 macOS 系统版本差异下不可用 | macOS ExecPath 降级 | 回退链完整（proc_pidpath → ps 降级标记 likely）；实现含符号加载探测单测 |
| 合并冲突集中在 scanner/daemon 文件 | 阶段推进慢 | Phase 1-4 顺序执行，每个 Phase 独立 PR；共享文件改动集中在 Phase 1/3 |

**回滚策略**：Phase 1-4 各自独立合入，任何阶段出现回归时 revert 该阶段即可；v2 元数据写入点在 `InstanceRegistry.Save` 单点，可一键切回纯数字写入（保留 `PIDFileManager` 兼容层 1 个发布周期后移除）。

