# 技术设计文档：Mihomo CLI 管理工具

## 1. 架构概览

### 1.1 系统架构图

```
┌─────────────────────────────────────────────────────────────────┐
│                        mihomo-cli                               │
├─────────────────────────────────────────────────────────────────┤
│                         cmd/ (CLI 入口)                         │
│  ┌─────────┐ ┌─────────┐ ┌─────────┐ ┌─────────┐ ┌─────────┐   │
│  │ root.go │ │ mode.go │ │proxy.go │ │service.go│ │config.go│   │
│  └────┬────┘ └────┬────┘ └────┬────┘ └────┬────┘ └────┬────┘   │
└───────┼──────────┼──────────┼──────────┼──────────┼─────────────┘
        │          │          │          │          │
        └──────────┴──────────┴──────────┴──────────┘
                              │
┌─────────────────────────────┴───────────────────────────────────┐
│                    internal/ (业务逻辑层)                        │
├─────────────────────────────────────────────────────────────────┤
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐          │
│  │   api/       │  │   service/   │  │   config/    │          │
│  │  (API客户端)  │  │ (Win服务管理)│  │  (配置管理)   │          │
│  └──────────────┘  └──────────────┘  └──────────────┘          │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐          │
│  │   proxy/     │  │   sysproxy/  │  │    util/     │          │
│  │  (代理管理)   │  │ (系统代理)    │  │  (工具函数)   │          │
│  └──────────────┘  └──────────────┘  └──────────────┘          │
└─────────────────────────────────────────────────────────────────┘
                              │
┌─────────────────────────────┴───────────────────────────────────┐
│                       外部依赖                                   │
├─────────────────────────────────────────────────────────────────┤
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────────────────┐ │
│  │   Cobra     │  │   Viper     │  │  golang.org/x/sys       │ │
│  │  (CLI框架)   │  │  (配置管理)  │  │     (Windows服务)       │ │
│  └─────────────┘  └─────────────┘  └─────────────────────────┘ │
│  ┌─────────────┐  ┌─────────────┐                               │
│  │  fatih/color│  │  Mihomo API │                               │
│  │  (彩色输出)  │  │  (RESTful)  │                               │
│  └─────────────┘  └─────────────┘                               │
└─────────────────────────────────────────────────────────────────┘
```

### 1.2 设计原则

| 原则 | 说明 |
|------|------|
| 无状态设计 | CLI 不保存运行时状态，所有状态查询实时请求 API |
| 配置持久化 | API 地址和 Secret 保存在本地配置文件 |
| 查询与修改分离 | get/list/show (只读) vs set/switch/update (写操作) |
| 输出格式可控 | 支持 table 和 json 格式，方便脚本调用 |
| 错误处理统一 | 统一的错误处理机制，清晰的错误信息和退出码 |

### 1.3 WebSocket 支持设计

Mihomo API 部分端点支持 WebSocket 连接，用于实时数据推送。CLI 工具支持通过 WebSocket 接收实时数据。

**支持的 WebSocket 端点：**
- `/logs` - 实时日志推送
- `/traffic` - 实时流量统计推送
- `/memory` - 实时内存使用推送
- `/connections` - 实时连接列表推送

**WebSocket 客户端设计：**

```go
// WebSocketClient WebSocket 客户端
type WebSocketClient struct {
    baseURL string
    secret  string
    timeout time.Duration
}

// StreamTraffic 流量统计流式接收
func (c *WebSocketClient) StreamTraffic(ctx context.Context, callback func(*TrafficInfo)) error

// StreamMemory 内存使用流式接收
func (c *WebSocketClient) StreamMemory(ctx context.Context, callback func(*MemoryInfo)) error

// StreamConnections 连接列表流式接收
func (c *WebSocketClient) StreamConnections(ctx context.Context, callback func(*ConnectionInfo)) error
```

**连接流程：**

```
WebSocket 连接流程:
┌─────────────┐     ┌─────────────┐     ┌─────────────┐
│ 创建客户端  │ ──▶ │ 建立 WS 连接│ ──▶ │ 接收实时数据│
│             │     │ (带 token)  │     │             │
└─────────────┘     └─────────────┘     └─────────────┘
                                              │
                    ┌─────────────┐           │
                    │ 回调处理数据│ ◀─────────┘
                    └─────────────┘
```

**Watch 模式：**

当用户使用 `--watch` 参数时，CLI 将使用 WebSocket 连接持续接收并显示实时数据：

```bash
# 实时监控流量
mihomo-cli monitor traffic --watch

# 实时监控内存
mihomo-cli monitor memory --watch
```

---

## 2. 项目结构

```
mihomo-cli/
├── cmd/                        # 命令定义入口
│   ├── root.go                 # 根命令和全局标志
│   ├── mode.go                 # 模式管理命令
│   ├── proxy.go                # 代理管理命令
│   ├── service.go              # 服务管理命令
│   ├── config.go               # 配置管理命令
│   ├── sysproxy.go             # 系统代理命令
│   ├── sub.go                  # 订阅管理命令
│   ├── rule.go                 # 规则管理命令
│   ├── conn.go                 # 连接管理命令
│   ├── cache.go                # 缓存管理命令
│   ├── dns.go                  # DNS 查询命令
│   ├── geo.go                  # Geo 管理命令
│   ├── monitor.go              # 监控命令
│   ├── version.go              # 版本查询命令
│   └── sys.go                  # 系统管理命令
├── internal/                   # 内部业务逻辑
│   ├── api/                    # Mihomo API 客户端
│   │   ├── client.go           # HTTP 客户端封装
│   │   ├── auth.go             # 认证处理
│   │   ├── mode.go             # 模式 API
│   │   ├── proxy.go            # 代理 API
│   │   ├── config.go           # 配置 API
│   │   ├── provider.go         # 订阅 API
│   │   ├── rule.go             # 规则 API
│   │   ├── connection.go       # 连接 API
│   │   ├── cache.go            # 缓存 API
│   │   ├── dns.go              # DNS API
│   │   ├── system.go           # 系统 API
│   │   ├── version.go          # 版本 API
│   │   └── monitor.go          # 监控 API
│   ├── service/                # Windows 服务管理
│   │   ├── manager.go          # 服务管理器
│   │   ├── install.go          # 服务安装
│   │   └── control.go          # 服务控制
│   ├── config/                 # CLI 配置管理
│   │   ├── config.go           # 配置结构定义
│   │   ├── loader.go           # 配置加载
│   │   └── editor.go           # 配置文件编辑
│   ├── proxy/                  # 代理管理逻辑
│   │   ├── selector.go         # 节点选择
│   │   ├── tester.go           # 延迟测试
│   │   └── formatter.go        # 输出格式化
│   ├── sysproxy/               # 系统代理管理
│   │   └── windows.go          # Windows 注册表操作
│   ├── monitor/                # 监控模块
│   │   └── stream.go           # WebSocket 流处理
│   ├── output/                 # 输出处理
│   │   ├── table.go            # 表格输出
│   │   ├── json.go             # JSON 输出
│   │   └── color.go            # 彩色输出
│   └── util/                   # 工具函数
│       ├── error.go            # 错误处理
│       ├── validate.go         # 参数验证
│       └── admin.go            # 管理员权限检查
├── pkg/                        # 可复用包
│   └── types/                  # 类型定义
│       ├── mode.go             # 模式类型
│       ├── proxy.go            # 代理类型
│       ├── config.go           # 配置类型
│       ├── rule.go             # 规则类型
│       ├── connection.go       # 连接类型
│       ├── dns.go              # DNS 类型
│       ├── monitor.go          # 监控类型
│       └── system.go           # 系统类型
├── go.mod
├── go.sum
└── main.go                     # 程序入口
```

---

## 3. 核心模块设计

### 3.1 API 客户端模块 (internal/api)

#### 3.1.1 客户端结构

```go
// Client Mihomo API 客户端
type Client struct {
    baseURL    string           // API 基础地址
    secret     string           // API 密钥
    httpClient *http.Client     // HTTP 客户端
    timeout    time.Duration    // 请求超时
}

// ClientOption 客户端配置选项
type ClientOption func(*Client)
```

#### 3.1.2 API 接口定义

```go
// ModeAPI 模式管理接口
type ModeAPI interface {
    GetMode(ctx context.Context) (*ModeInfo, error)
    SetMode(ctx context.Context, mode TunnelMode) error
}

// ProxyAPI 代理管理接口
type ProxyAPI interface {
    ListProxies(ctx context.Context) (map[string]*ProxyInfo, error)
    GetProxy(ctx context.Context, name string) (*ProxyInfo, error)
    SwitchProxy(ctx context.Context, group, proxy string) error
    TestDelay(ctx context.Context, name string, timeout int) (uint16, error)
}

// ConfigAPI 配置管理接口
type ConfigAPI interface {
    GetConfig(ctx context.Context) (*ConfigInfo, error)
    PatchConfig(ctx context.Context, patch map[string]any) error
    ReloadConfig(ctx context.Context, path string, force bool) error
    UpdateGeo(ctx context.Context) error
}

// ProviderAPI 订阅管理接口
type ProviderAPI interface {
    ListProviders(ctx context.Context) (map[string]*ProviderInfo, error)
    UpdateProvider(ctx context.Context, name string) error
}

// RuleAPI 规则管理接口
type RuleAPI interface {
    GetRules(ctx context.Context) ([]RuleInfo, error)
    DisableRules(ctx context.Context, ruleIndexes map[int]bool) error
}

// ConnectionAPI 连接管理接口
type ConnectionAPI interface {
    GetConnections(ctx context.Context) (*ConnectionInfo, error)
    CloseConnection(ctx context.Context, id string) error
    CloseAllConnections(ctx context.Context) error
}

// CacheAPI 缓存管理接口
type CacheAPI interface {
    FlushFakeIP(ctx context.Context) error
    FlushDNS(ctx context.Context) error
}

// DNSAPI DNS 查询接口
type DNSAPI interface {
    Query(ctx context.Context, name string, recordType string) (*DNSResponse, error)
}

// SystemAPI 系统管理接口
type SystemAPI interface {
    Restart(ctx context.Context) error
    Upgrade(ctx context.Context, channel string, force bool) error
    UpdateGeo(ctx context.Context) error
}

// VersionAPI 版本查询接口
type VersionAPI interface {
    GetVersion(ctx context.Context) (*VersionInfo, error)
}

// MonitorAPI 监控接口
type MonitorAPI interface {
    GetTraffic(ctx context.Context) (*TrafficInfo, error)
    GetMemory(ctx context.Context) (*MemoryInfo, error)
}
```

#### 3.1.3 认证机制

```
请求流程:
┌──────────┐     ┌──────────────────┐     ┌─────────────┐
│ CLI 请求 │ ──▶ │ 添加 Authorization│ ──▶ │ Mihomo API  │
└──────────┘     │   Bearer {secret} │     └─────────────┘
                 └──────────────────┘
```

### 3.2 配置管理模块 (internal/config)

#### 3.2.1 CLI 配置结构

```go
// CLIConfig CLI 工具配置
type CLIConfig struct {
    API APIConfig `mapstructure:"api"`
}

// APIConfig API 连接配置
type APIConfig struct {
    Address string `mapstructure:"address"` // API 地址，如 http://127.0.0.1:9090
    Secret  string `mapstructure:"secret"`  // API 密钥
    Timeout int    `mapstructure:"timeout"` // 请求超时（秒）
}
```

#### 3.2.2 配置文件编辑流程

```
配置编辑流程:
┌─────────────┐     ┌─────────────┐     ┌─────────────┐
│ 读取原配置  │ ──▶ │ 解析 YAML   │ ──▶ │ 合并新配置  │
└─────────────┘     └─────────────┘     └─────────────┘
                                              │
┌─────────────┐     ┌─────────────┐           │
│ 调用重载API │ ◀── │ 写入新配置  │ ◀─────────┘
└─────────────┘     └─────────────┘
       │
       ▼
┌─────────────┐
│  备份原文件  │ (可选)
└─────────────┘
```

#### 3.2.3 规则禁用功能设计

```go
// DisableRulesRequest 禁用规则请求
type DisableRulesRequest struct {
    RuleIndexes map[int]bool `json:"-"` // key: 规则索引, value: true=禁用, false=启用
}

// 禁用规则流程:
// 1. 用户执行 `mihomo-cli rule disable <index>`
// 2. 解析规则索引
// 3. 构造请求体 {index: true}
// 4. 调用 PATCH /rules/disable
// 5. 返回操作结果

// 启用规则流程:
// 1. 用户执行 `mihomo-cli rule enable <index>`
// 2. 解析规则索引
// 3. 构造请求体 {index: false}
// 4. 调用 PATCH /rules/disable
// 5. 返回操作结果
```

### 3.3 服务管理模块 (internal/service)

#### 3.3.1 Windows 服务管理

```go
// ServiceManager Windows 服务管理器
type ServiceManager struct {
    serviceName string
    displayName string
    exePath     string
}

// ServiceStatus 服务状态
type ServiceStatus string

const (
    StatusRunning ServiceStatus = "running"
    StatusStopped ServiceStatus = "stopped"
    StatusNotInstalled ServiceStatus = "not-installed"
)
```

#### 3.3.2 服务管理流程

```
服务安装流程:
┌─────────────┐     ┌─────────────┐     ┌─────────────┐
│ 检查管理员  │ ──▶ │ 检查服务存在│ ──▶ │ 创建服务   │
│   权限     │     │   (报错)    │     │             │
└─────────────┘     └─────────────┘     └─────────────┘

服务控制流程:
┌─────────────┐     ┌─────────────┐     ┌─────────────┐
│ 打开服务管理│ ──▶ │ 打开目标服务│ ──▶ │ 执行操作   │
│    器      │     │             │     │(start/stop) │
└─────────────┘     └─────────────┘     └─────────────┘
```

### 3.4 代理管理模块 (internal/proxy)

#### 3.4.1 延迟测试策略

```go
// DelayTester 延迟测试器
type DelayTester struct {
    client    *api.Client
    testURL   string        // 测试 URL
    timeout   time.Duration // 超时时间
    concurrent int          // 并发数
}

// TestResult 测试结果
type TestResult struct {
    Name   string
    Delay  uint16
    Error  error
}
```

#### 3.4.2 自动选择最快节点流程

```
自动选择流程:
┌─────────────┐     ┌─────────────┐     ┌─────────────┐
│ 获取代理组  │ ──▶ │ 并发测试延迟│ ──▶ │ 排序结果   │
│  节点列表  │     │             │     │             │
└─────────────┘     └─────────────┘     └─────────────┘
                                              │
                    ┌─────────────┐           │
                    │ 切换到最快  │ ◀─────────┘
                    │   节点     │
                    └─────────────┘
```

### 3.5 系统代理模块 (internal/sysproxy)

#### 3.5.1 Windows 注册表操作

```go
// WindowsRegistry Windows 注册表操作
type WindowsRegistry struct {
    key registry.Key
}

// SystemProxySettings 系统代理设置
type SystemProxySettings struct {
    Enabled    bool
    Server     string
    BypassList string
}
```

#### 3.5.2 系统代理设置流程

```
设置系统代理:
┌─────────────┐     ┌─────────────┐     ┌─────────────┐
│ 检查管理员  │ ──▶ │ 打开注册表  │ ──▶ │ 写入代理值  │
│   权限     │     │   Key      │     │             │
└─────────────┘     └─────────────┘     └─────────────┘
       │                   │                   │
       ▼                   ▼                   ▼
   HKCU\Software\Microsoft\Windows\CurrentVersion\Internet Settings
```

---

## 4. 命令设计

### 4.1 命令树结构

```
mihomo-cli
├── mode                    # 模式管理
│   ├── get                 # 查询当前模式
│   └── set <mode>          # 设置模式 (rule/global/direct)
├── proxy                   # 代理管理
│   ├── list [group]        # 列出代理节点
│   ├── switch <group> <node>  # 切换节点
│   ├── test <group> [node]    # 测试延迟
│   ├── auto <group>        # 自动选择最快
│   └── unfix <group>       # 取消固定代理（恢复自动选择）
├── service                 # 服务管理
│   ├── start               # 启动服务
│   ├── stop                # 停止服务
│   ├── install             # 安装服务
│   ├── uninstall           # 卸载服务
│   └── status              # 查询状态
├── config                  # 配置管理
│   ├── init                # 初始化 CLI 配置
│   ├── show                # 显示当前配置
│   ├── set <key> <value>   # 设置配置项
│   ├── patch <key> <value> # 热更新配置
│   ├── reload [--path] [--force]  # 重载配置
│   └── edit <key> <value> [--no-reload]  # 编辑配置文件
├── sysproxy                # 系统代理
│   ├── get                 # 查询状态
│   └── set <on|off>        # 开启/关闭
├── sub                     # 订阅管理
│   └── update              # 更新订阅
├── rule                    # 规则管理
│   ├── list                # 列出所有规则
│   ├── disable <index>     # 禁用规则
│   └── enable <index>      # 启用规则
├── conn                    # 连接管理
│   ├── list                # 列出活跃连接
│   ├── close <id>          # 关闭指定连接
│   └── close-all           # 关闭所有连接
├── cache                   # 缓存管理
│   ├── clear fakeip        # 清空 FakeIP 池
│   └── clear dns           # 清空 DNS 缓存
├── geo                     # Geo 数据库管理
│   └── update              # 更新 Geo 数据库
├── dns                     # DNS 查询
│   └── query <domain> [--type]  # 执行 DNS 查询
├── monitor                 # 实时监控
│   ├── traffic [--watch]   # 流量统计
│   └── memory [--watch]    # 内存使用
├── version                 # 版本查询
└── sys                     # 系统管理
    ├── restart             # 重启服务
    └── upgrade [--channel] [--force]  # 升级核心
```

### 4.2 全局标志

| 标志 | 简写 | 说明 | 默认值 |
|------|------|------|--------|
| --output | -o | 输出格式 (table/json) | table |
| --config | -c | CLI 配置文件路径 | ~/.mihomo-cli/config.yaml |
| --api | | API 地址 (覆盖配置文件) | |
| --secret | | API 密钥 (覆盖配置文件) | |
| --timeout | -t | 请求超时 (秒) | 10 |
| --help | -h | 显示帮助 | |
| --version | -v | 显示版本 | |

### 4.3 命令实现示例

```go
// mode.go - 模式管理命令
func NewModeCmd() *cobra.Command {
    cmd := &cobra.Command{
        Use:   "mode",
        Short: "管理 Mihomo 运行模式",
    }
    cmd.AddCommand(
        newModeGetCmd(),
        newModeSetCmd(),
    )
    return cmd
}

func newModeGetCmd() *cobra.Command {
    return &cobra.Command{
        Use:   "get",
        Short: "查询当前运行模式",
        RunE:  runModeGet,
    }
}

func runModeGet(cmd *cobra.Command, args []string) error {
    client := api.NewClientFromFlags()
    mode, err := client.GetMode(cmd.Context())
    if err != nil {
        return err
    }
    return output.Print(mode)
}
```

---

## 5. 数据模型

### 5.1 API 响应类型

```go
// ModeInfo 模式信息
type ModeInfo struct {
    Mode string `json:"mode"`
}

// ProxyInfo 代理信息
type ProxyInfo struct {
    Name    string            `json:"name"`
    Type    string            `json:"type"`
    Now     string            `json:"now"`
    All     []string          `json:"all"`
    Delay   uint16            `json:"delay"`
    Alive   bool              `json:"alive"`
    Extra   map[string]any    `json:"extra"`
}

// ConfigInfo 配置信息
type ConfigInfo struct {
    Port        int    `json:"port"`
    SocksPort   int    `json:"socks-port"`
    MixedPort   int    `json:"mixed-port"`
    AllowLan    bool   `json:"allow-lan"`
    BindAddress string `json:"bind-address"`
    Mode        string `json:"mode"`
    LogLevel    string `json:"log-level"`
    IPv6        bool   `json:"ipv6"`
}

// ProviderInfo 订阅提供者信息
type ProviderInfo struct {
    Name     string   `json:"name"`
    Type     string   `json:"type"`
    Proxies  []string `json:"proxies"`
    Updated  int64    `json:"updatedAt"`
}

// RuleInfo 规则信息
type RuleInfo struct {
    Index    int                    `json:"index"`
    Type     string                 `json:"type"`
    Payload  string                 `json:"payload"`
    Proxy    string                 `json:"proxy"`
    Size     int                    `json:"size"`
    Disabled bool                   `json:"disabled"`
    HitCount int                    `json:"hitCount"`
    HitAt    string                 `json:"hitAt"`
    MissCount int                   `json:"missCount"`
    MissAt   string                 `json:"missAt"`
}

// ConnectionInfo 连接信息
type ConnectionInfo struct {
    DownloadTotal int64                `json:"downloadTotal"`
    UploadTotal   int64                `json:"uploadTotal"`
    Connections   []ConnectionDetail   `json:"connections"`
}

// ConnectionDetail 连接详情
type ConnectionDetail struct {
    ID       string          `json:"id"`
    Metadata ConnectionMeta  `json:"metadata"`
    Upload   int64           `json:"upload"`
    Download int64           `json:"download"`
    Start    string          `json:"start"`
    Chains   []string        `json:"chains"`
    Rule     string          `json:"rule"`
    RulePayload string       `json:"rulePayload"`
}

// ConnectionMeta 连接元数据
type ConnectionMeta struct {
    Net             string `json:"net"`
    Type            string `json:"type"`
    SourceIP        string `json:"sourceIP"`
    DestinationIP   string `json:"destinationIP"`
    SourcePort      string `json:"sourcePort"`
    DestinationPort string `json:"destinationPort"`
    Host            string `json:"host"`
    DnsMode         string `json:"dnsMode"`
    ProcessPath     string `json:"processPath"`
    SpecialProxy    string `json:"specialProxy"`
}

// TrafficInfo 流量信息
type TrafficInfo struct {
    Up        int64 `json:"up"`
    Down      int64 `json:"down"`
    UpTotal   int64 `json:"upTotal"`
    DownTotal int64 `json:"downTotal"`
}

// MemoryInfo 内存信息
type MemoryInfo struct {
    Inuse   int64 `json:"inuse"`
    Oslimit int64 `json:"oslimit"`
}

// VersionInfo 版本信息
type VersionInfo struct {
    Meta    string `json:"meta"`
    Version string `json:"version"`
}

// DNSResponse DNS 响应
type DNSResponse struct {
    Status     int              `json:"Status"`
    Question   []DNSQuestion    `json:"Question"`
    TC         bool             `json:"TC"`
    RD         bool             `json:"RD"`
    RA         bool             `json:"RA"`
    AD         bool             `json:"AD"`
    CD         bool             `json:"CD"`
    Answer     []DNSAnswer      `json:"Answer"`
    Authority  []DNSAuthority   `json:"Authority"`
    Additional []DNSAdditional  `json:"Additional"`
}

// DNSQuestion DNS 查询问题
type DNSQuestion struct {
    Name  string `json:"name"`
    Type  int    `json:"type"`
    Class int    `json:"class"`
}

// DNSAnswer DNS 响应答案
type DNSAnswer struct {
    Name string `json:"name"`
    Type int    `json:"type"`
    TTL  int    `json:"TTL"`
    Data string `json:"data"`
}

// DNSAuthority DNS 权威记录
type DNSAuthority struct {
    Name string `json:"name"`
    Type int    `json:"type"`
    TTL  int    `json:"TTL"`
    Data string `json:"data"`
}

// DNSAdditional DNS 附加记录
type DNSAdditional struct {
    Name string `json:"name"`
    Type int    `json:"type"`
    TTL  int    `json:"TTL"`
    Data string `json:"data"`
}
```

### 5.2 输出格式类型

```go
// OutputFormat 输出格式
type OutputFormat string

const (
    FormatTable OutputFormat = "table"
    FormatJSON  OutputFormat = "json"
)

// TableOutput 表格输出接口
type TableOutput interface {
    Headers() []string
    Rows() [][]string
}
```

---

## 6. 错误处理设计

### 6.1 错误类型

```go
// CLIError CLI 错误类型
type CLIError struct {
    Code    ErrorCode
    Message string
    Cause   error
}

// ErrorCode 错误码
type ErrorCode int

const (
    ErrSuccess        ErrorCode = 0
    ErrGeneral        ErrorCode = 1
    ErrAPIConnection  ErrorCode = 2
    ErrAPIAuth        ErrorCode = 3
    ErrInvalidArgs    ErrorCode = 4
    ErrNotFound       ErrorCode = 5
    ErrPermission     ErrorCode = 6
    ErrFileOperation  ErrorCode = 7
    ErrYAMLParse      ErrorCode = 8
)
```

### 6.2 错误处理流程

```
错误处理流程:
┌─────────────┐     ┌─────────────┐     ┌─────────────┐
│  捕获错误   │ ──▶ │  包装错误   │ ──▶ │  输出错误   │
│             │     │  (添加上下文)│     │  (彩色显示) │
└─────────────┘     └─────────────┘     └─────────────┘
                                              │
                    ┌─────────────┐           │
                    │  设置退出码  │ ◀─────────┘
                    └─────────────┘
```

---

## 7. 技术选型

### 7.1 核心依赖

| 依赖 | 版本 | 用途 |
|------|------|------|
| github.com/spf13/cobra | v1.8+ | CLI 框架 |
| github.com/spf13/viper | v1.18+ | 配置管理 |
| github.com/fatih/color | v1.16+ | 彩色输出 |
| golang.org/x/sys | latest | Windows 服务管理 |
| gopkg.in/yaml.v3 | v3+ | YAML 解析 |
| github.com/olekukonko/tablewriter | v0.0.5 | 表格输出 |

### 7.2 API 端点映射

| CLI 命令 | HTTP 方法 | API 端点 |
|----------|-----------|----------|
| mode get | GET | /configs |
| mode set | PATCH | /configs |
| proxy list | GET | /proxies |
| proxy switch | PUT | /proxies/{group} |
| proxy test | GET | /proxies/{name}/delay |
| proxy unfix | DELETE | /proxies/{group} |
| config patch | PATCH | /configs |
| config reload | PUT | /configs |
| sub update | PUT | /providers/proxies/{name} |
| rule list | GET | /rules |
| rule disable/enable | PATCH | /rules/disable |
| conn list | GET | /connections |
| conn close | DELETE | /connections/{id} |
| conn close-all | DELETE | /connections |
| cache clear fakeip | POST | /cache/fakeip/flush |
| cache clear dns | POST | /cache/dns/flush |
| geo update | POST | /configs/geo |
| dns query | GET | /dns/query |
| monitor traffic | GET | /traffic |
| monitor memory | GET | /memory |
| version | GET | /version |
| sys restart | POST | /restart |
| sys upgrade | POST | /upgrade |

---

## 8. 安全设计

### 8.1 敏感信息处理

- Secret 存储在用户目录下，设置文件权限为 0600
- 输出配置时对 Secret 进行脱敏（显示为 `****`）
- 不在日志中输出 Secret

### 8.2 路径安全

- 配置文件路径必须是绝对路径
- 检查路径是否在安全范围内，防止路径遍历攻击

### 8.3 权限检查

- 服务管理命令需要管理员权限
- 系统代理设置需要管理员权限
- 提供权限检查函数，无权限时给出明确提示

---

## 9. 性能考虑

### 9.1 并发控制

- 延迟测试使用 goroutine 并发执行
- 使用 sync.WaitGroup 等待所有测试完成
- 可配置并发数限制

### 9.2 超时控制

- 所有 API 请求设置超时时间
- 使用 context.Context 传递超时设置
- 默认超时 10 秒，可通过参数调整

---

## 10. 测试策略

### 10.1 单元测试

- API 客户端使用 mock 服务器测试
- 配置解析使用测试数据验证
- 错误处理覆盖所有错误码

### 10.2 集成测试

- 使用真实的 Mihomo 实例测试
- 测试完整的命令执行流程
- 验证输出格式正确性

### 10.3 测试覆盖率目标

- 核心模块覆盖率 > 80%
- 错误处理覆盖率 100%
