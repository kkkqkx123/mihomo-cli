# 实现任务规划：Mihomo CLI 管理工具

## 任务概览

| 统计项     | 数量           |
| ---------- | -------------- |
| 主任务数   | 19             |
| 子任务数   | 63             |
| 覆盖需求数 | 123 条验收标准 |

---

## 任务 1：项目初始化与基础框架搭建

**描述：** 创建项目基础结构，初始化 Go 模块，配置核心依赖。

**输入：** 无

**输出：** 可编译的项目骨架

**验收标准：**

- 项目目录结构符合设计文档
- go.mod 包含所有核心依赖
- main.go 可正常编译运行

### 子任务

#### 1.1 创建项目目录结构

创建 cmd/、internal/、pkg/ 目录结构，按照设计文档组织代码。

#### 1.2 初始化 Go 模块

执行 `go mod init github.com/user/mihomo-cli`，添加核心依赖：

- github.com/spf13/cobra
- github.com/spf13/viper
- github.com/fatih/color
- golang.org/x/sys/windows
- gopkg.in/yaml.v3
- github.com/olekukonko/tablewriter

#### 1.3 创建程序入口

创建 main.go，初始化 Cobra 根命令，设置版本信息。

---

## 任务 2：CLI 配置管理实现

**描述：** 实现 CLI 工具的本地配置管理功能，包括配置加载、保存和验证。

**输入：** 配置文件路径参数

**输出：** 可用的配置管理模块

**验收标准：**

- 支持从默认路径和指定路径加载配置
- 配置项可通过命令行标志覆盖
- 配置文件不存在时使用默认值

### 子任务

#### 2.1 定义配置结构

在 internal/config/config.go 中定义 CLIConfig 结构体，包含 API 地址、Secret、超时等字段。

#### 2.2 实现配置加载器

在 internal/config/loader.go 中使用 Viper 实现配置加载，支持 YAML 格式，支持环境变量覆盖。

#### 2.3 实现配置验证

验证 API 地址格式、超时值范围等，无效配置返回明确错误。

#### 2.4 实现 config 命令

在 cmd/config.go 中实现 config init/show/set 子命令。

---

## 任务 3：API 客户端封装实现

**描述：** 封装 Mihomo RESTful API 客户端，提供统一的请求处理和认证机制。

**输入：** API 地址、Secret

**输出：** 可用的 API 客户端模块

**验收标准：**

- 所有请求自动添加 Authorization 头
- 请求超时可配置
- 错误响应统一处理

### 子任务

#### 3.1 创建客户端基础结构

在 internal/api/client.go 中创建 Client 结构体，实现 NewClient 构造函数，配置 HTTP 客户端超时。

#### 3.2 实现认证机制

在 internal/api/auth.go 中实现 Bearer Token 认证，自动添加 Authorization 头到请求。

#### 3.3 实现通用请求方法

封装 GET、POST、PUT、PATCH、DELETE 方法，统一处理响应和错误。

#### 3.4 实现错误处理

定义 APIError 类型，解析 Mihomo API 返回的错误信息，设置适当的退出码。

---

## 任务 4：模式管理功能实现

**描述：** 实现运行模式的查询和切换功能。

**输入：** 模式命令参数

**输出：** mode get/set 命令可用

**验收标准：**

- mode get 返回当前运行模式
- mode set 可切换到 rule/global/direct 模式
- 无效模式返回错误提示

### 子任务

#### 4.1 实现模式 API

在 internal/api/mode.go 中实现 GetMode 和 SetMode 方法，调用 GET/PATCH /configs 端点。

#### 4.2 实现 mode 命令

在 cmd/mode.go 中实现 mode get/set 子命令，绑定 API 调用。

#### 4.3 实现模式验证

验证模式值是否为 rule/global/direct，无效值返回错误并列出有效选项。

---

## 任务 5：代理管理功能实现

**描述：** 实现代理节点的列表、切换、测速和自动选择功能。

**输入：** 代理命令参数

**输出：** proxy list/switch/test/auto 命令可用

**验收标准：**

- proxy list 显示所有代理组和节点
- proxy switch 可切换指定代理组的节点
- proxy test 可测试节点延迟
- proxy auto 可自动选择最快节点

### 子任务

#### 5.1 实现代理 API

在 internal/api/proxy.go 中实现：

- ListProxies：GET /proxies
- GetProxy：GET /proxies/{name}
- SwitchProxy：PUT /proxies/{group}
- TestDelay：GET /proxies/{name}/delay

#### 5.2 实现代理列表格式化

在 internal/proxy/formatter.go 中实现表格和 JSON 输出格式化，显示节点名称、类型、延迟等信息。

#### 5.3 实现延迟测试器

在 internal/proxy/tester.go 中实现并发延迟测试，支持自定义测试 URL 和超时。

#### 5.4 实现自动选择逻辑

在 internal/proxy/selector.go 中实现自动选择最快节点逻辑，测试所有节点后切换到延迟最低的节点。

#### 5.5 实现 proxy 命令

在 cmd/proxy.go 中实现 proxy list/switch/test/auto 子命令。

---

## 任务 6：Windows 服务管理实现

**描述：** 实现 Mihomo Windows 服务的安装、卸载、启动、停止和状态查询功能。

**输入：** 服务命令参数

**输出：** service start/stop/install/uninstall/status 命令可用

**验收标准：**

- 无管理员权限时返回明确错误
- 服务操作成功显示确认信息
- 服务不存在时返回适当错误

### 子任务

#### 6.1 实现权限检查

在 internal/util/admin.go 中实现 IsAdmin 函数，检查当前进程是否以管理员权限运行。

#### 6.2 实现服务管理器

在 internal/service/manager.go 中使用 golang.org/x/sys/windows/svc/mgr 实现服务管理器，封装服务打开、创建、删除操作。

#### 6.3 实现服务安装

在 internal/service/install.go 中实现服务安装，设置服务名称、显示名称、启动类型。

#### 6.4 实现服务控制

在 internal/service/control.go 中实现服务启动、停止、查询状态功能。

#### 6.5 实现 service 命令

在 cmd/service.go 中实现 service start/stop/install/uninstall/status 子命令。

---

## 任务 7：系统代理管理实现

**描述：** 实现 Windows 系统代理的查询和设置功能。

**输入：** 系统代理命令参数

**输出：** sysproxy get/set 命令可用

**验收标准：**

- sysproxy get 显示当前系统代理状态
- sysproxy set on/off 可开启/关闭系统代理
- 无管理员权限时返回明确错误

### 子任务

#### 7.1 实现注册表操作

在 internal/sysproxy/windows.go 中使用 golang.org/x/sys/windows/registry 实现注册表读写，操作 Internet Settings 键。

#### 7.2 实现系统代理设置

实现 EnableProxy 和 DisableProxy 函数，设置 ProxyEnable、ProxyServer、ProxyOverride 值。

#### 7.3 实现 sysproxy 命令

在 cmd/sysproxy.go 中实现 sysproxy get/set 子命令。

---

## 任务 8：配置热更新功能实现

**描述：** 实现 Mihomo 配置的热更新和重载功能。

**输入：** 配置更新命令参数

**输出：** config patch/reload/edit 命令可用

**验收标准：**

- config patch 可热更新运行时配置
- config reload 可重载完整配置文件
- config edit 可编辑配置文件并自动重载

### 子任务

#### 8.1 实现配置 API

在 internal/api/config.go 中实现：

- GetConfig：GET /configs
- PatchConfig：PATCH /configs
- ReloadConfig：PUT /configs
- UpdateGeo：POST /configs/geo

#### 8.2 实现配置文件编辑器

在 internal/config/editor.go 中实现：

- 读取现有 YAML 配置文件
- 合并用户指定的配置项
- 备份原配置文件
- 写入新配置文件

#### 8.3 实现配置键值映射

定义支持的配置键及其类型映射，验证配置键是否支持热更新。

#### 8.4 更新 config 命令

在 cmd/config.go 中添加 patch/reload/edit 子命令。

---

## 任务 9：订阅管理功能实现

**描述：** 实现代理订阅的更新功能。

**输入：** 订阅命令参数

**输出：** sub update 命令可用

**验收标准：**

- sub update 可触发订阅更新
- 更新成功显示确认信息
- 更新失败显示错误原因

### 子任务

#### 9.1 实现订阅 API

在 internal/api/provider.go 中实现：

- ListProviders：GET /providers/proxies
- UpdateProvider：PUT /providers/proxies/{name}

#### 9.2 实现 sub 命令

在 cmd/sub.go 中实现 sub update 子命令。

---

## 任务 10：输出处理模块实现

**描述：** 实现统一的输出处理模块，支持表格和 JSON 格式。

**输入：** 输出数据和格式参数

**输出：** 格式化的输出内容

**验收标准：**

- 默认输出表格格式
- -o json 输出 JSON 格式
- 表格支持彩色显示

### 子任务

#### 10.1 实现表格输出

在 internal/output/table.go 中使用 tablewriter 实现表格输出，支持表头、对齐、边框。

#### 10.2 实现 JSON 输出

在 internal/output/json.go 中实现 JSON 格式化输出，支持美化输出。

#### 10.3 实现彩色输出

在 internal/output/color.go 中使用 fatih/color 实现彩色输出，定义成功、错误、警告、信息颜色。

#### 10.4 实现输出接口

定义统一的 Output 接口，根据 -o 参数选择输出格式。

---

## 任务 11：错误处理与退出码实现

**描述：** 实现统一的错误处理机制和退出码设置。

**输入：** 错误信息

**输出：** 格式化的错误输出和退出码

**验收标准：**

- 所有错误有明确的错误信息
- 不同类型错误设置不同退出码
- 错误信息使用彩色显示

### 子任务

#### 11.1 定义错误类型

在 internal/util/error.go 中定义 CLIError 类型和错误码常量。

#### 11.2 实现错误包装

实现错误包装函数，添加上下文信息，保留原始错误。

#### 11.3 实现错误输出

实现错误输出函数，使用彩色显示错误信息，设置退出码。

#### 11.4 集成到命令

在所有命令的 RunE 函数中统一使用错误处理机制。

---

## 任务 12：根命令与全局标志实现

**描述：** 实现根命令和全局标志，整合所有子命令。

**输入：** 命令行参数

**输出：** 可用的 CLI 程序

**验收标准：**

- --help 显示完整帮助信息
- --version 显示版本信息
- 全局标志可覆盖配置文件

### 子任务

#### 12.1 实现根命令

在 cmd/root.go 中创建根命令，设置名称、描述、版本。

#### 12.2 添加全局标志

添加 --output/-o、--config/-c、--api、--secret、--timeout/-t 等全局标志。

#### 12.3 注册子命令

将所有子命令注册到根命令：mode、proxy、service、config、sysproxy、sub。

#### 12.4 实现配置初始化

在 PersistentPreRun 中初始化配置，处理配置文件加载和标志覆盖。

#### 12.5 实现版本命令

添加 version 子命令，显示版本和构建信息。

---

## 任务 13：规则管理功能实现

**描述：** 实现规则的列表查询和禁用/启用功能。

**输入：** 规则命令参数

**输出：** rule list/disable/enable 命令可用

**验收标准：**

- rule list 显示所有规则及统计信息
- rule disable 可禁用指定规则
- rule enable 可启用指定规则
- 规则索引无效时返回错误提示

### 子任务

#### 13.1 实现规则 API

在 internal/api/rule.go 中实现：

- GetRules：GET /rules
- DisableRules：PATCH /rules/disable

#### 13.2 实现规则格式化

在 internal/rule/formatter.go 中实现表格和 JSON 输出格式化，显示规则索引、类型、匹配内容、代理、命中次数等信息。

#### 13.3 实现规则索引验证

验证规则索引是否在有效范围内，无效索引返回错误。

#### 13.4 实现 rule 命令

在 cmd/rule.go 中实现 rule list/disable/enable 子命令。

---

## 任务 14：连接管理功能实现

**描述：** 实现活跃连接的列表查询和关闭功能。

**输入：** 连接命令参数

**输出：** conn list/close/close-all 命令可用

**验收标准：**

- conn list 显示所有活跃连接
- conn close 可关闭指定连接
- conn close-all 可关闭所有连接
- 连接 ID 不存在时返回错误提示

### 子任务

#### 14.1 实现连接 API

在 internal/api/connection.go 中实现：

- GetConnections：GET /connections
- CloseConnection：DELETE /connections/{id}
- CloseAllConnections：DELETE /connections

#### 14.2 实现连接格式化

在 internal/connection/formatter.go 中实现表格和 JSON 输出格式化，显示连接 ID、源地址、目标地址、流量、代理、规则等信息。

#### 14.3 实现 conn 命令

在 cmd/conn.go 中实现 conn list/close/close-all 子命令。

---

## 任务 15：缓存管理功能实现

**描述：** 实现 FakeIP 池和 DNS 缓存的清空功能。

**输入：** 缓存命令参数

**输出：** cache clear fakeip/dns 命令可用

**验收标准：**

- cache clear fakeip 可清空 FakeIP 池
- cache clear dns 可清空 DNS 缓存
- FakeIP 未启用时返回错误提示

### 子任务

#### 15.1 实现缓存 API

在 internal/api/cache.go 中实现：

- FlushFakeIP：POST /cache/fakeip/flush
- FlushDNS：POST /cache/dns/flush

#### 15.2 实现 cache 命令

在 cmd/cache.go 中实现 cache clear fakeip/dns 子命令。

---

## 任务 16：Geo 数据库管理功能实现

**描述：** 实现 GeoIP 和 GeoSite 数据库的更新功能。

**输入：** Geo 命令参数

**输出：** geo update 命令可用

**验收标准：**

- geo update 可下载并更新 Geo 数据库
- 更新成功显示确认信息
- 更新失败显示错误原因

### 子任务

#### 16.1 实现 Geo API

在 internal/api/system.go 中实现 UpdateGeo 方法，调用 POST /configs/geo 端点。

#### 16.2 实现 geo 命令

在 cmd/geo.go 中实现 geo update 子命令。

---

## 任务 17：DNS 查询功能实现

**描述：** 实现 DNS 查询功能，支持多种记录类型。

**输入：** DNS 查询参数

**输出：** dns query 命令可用

**验收标准：**

- dns query 可执行 DNS 查询
- 支持 A、AAAA、CNAME 等记录类型
- 查询成功显示解析结果

### 子任务

#### 17.1 实现 DNS API

在 internal/api/dns.go 中实现 Query 方法，调用 GET /dns/query 端点。

#### 17.2 实现 DNS 响应格式化

在 internal/dns/formatter.go 中实现表格和 JSON 输出格式化，显示查询结果。

#### 17.3 实现 dns 命令

在 cmd/dns.go 中实现 dns query 子命令，支持 --type 参数指定记录类型。

---

## 任务 18：监控管理功能实现

**描述：** 实现实时流量和内存监控功能，支持 Watch 模式。

**输入：** 监控命令参数

**输出：** monitor traffic/memory 命令可用

**验收标准：**

- monitor traffic 显示流量统计
- monitor memory 显示内存使用
- --watch 参数支持实时刷新

### 子任务

#### 18.1 实现监控 API

在 internal/api/monitor.go 中实现：

- GetTraffic：GET /traffic
- GetMemory：GET /memory

#### 18.2 实现 WebSocket 客户端

在 internal/monitor/stream.go 中实现 WebSocket 客户端，支持实时数据推送。

#### 18.3 实现 Watch 模式

实现持续刷新功能，使用定时器定期获取数据并显示。

#### 18.4 实现 monitor 命令

在 cmd/monitor.go 中实现 monitor traffic/memory 子命令，支持 --watch 参数。

---

## 任务 19：系统管理功能实现

**描述：** 实现 Mihomo 服务的重启和核心升级功能。

**输入：** 系统管理命令参数

**输出：** sys restart/upgrade 命令可用

**验收标准：**

- sys restart 可重启 Mihomo 服务
- sys upgrade 可升级核心程序
- 支持 channel 和 force 参数

### 子任务

#### 19.1 实现系统 API

在 internal/api/system.go 中实现：

- Restart：POST /restart
- Upgrade：POST /upgrade

#### 19.2 实现 sys 命令

在 cmd/sys.go 中实现 sys restart/upgrade 子命令，支持 --channel 和 --force 参数。

---

## 任务 20：版本查询功能实现

**描述：** 实现版本信息查询功能。

**输入：** 无

**输出：** version 命令可用

**验收标准：**

- version 显示 Mihomo 版本信息
- 显示项目名称和版本号

### 子任务

#### 20.1 实现版本 API

在 internal/api/version.go 中实现 GetVersion 方法，调用 GET /version 端点。

#### 20.2 实现 version 命令

在 cmd/version.go 中实现 version 子命令。

---

## 任务依赖关系

```
任务 1 (项目初始化)
    │
    ├──▶ 任务 2 (CLI 配置管理)
    │        │
    │        └──▶ 任务 3 (API 客户端)
    │                   │
    │                   ├──▶ 任务 4 (模式管理)
    │                   ├──▶ 任务 5 (代理管理)
    │                   ├──▶ 任务 8 (配置热更新)
    │                   ├──▶ 任务 9 (订阅管理)
    │                   ├──▶ 任务 13 (规则管理)
    │                   ├──▶ 任务 14 (连接管理)
    │                   ├──▶ 任务 15 (缓存管理)
    │                   ├──▶ 任务 16 (Geo 管理)
    │                   ├──▶ 任务 17 (DNS 查询)
    │                   ├──▶ 任务 18 (监控管理)
    │                   ├──▶ 任务 19 (系统管理)
    │                   └──▶ 任务 20 (版本查询)
    │
    ├──▶ 任务 6 (服务管理)
    │
    ├──▶ 任务 7 (系统代理)
    │
    ├──▶ 任务 10 (输出处理)
    │
    ├──▶ 任务 11 (错误处理)
    │
    └──▶ 任务 12 (根命令整合)
             │
             └──▶ 所有其他任务
```

---

## 优先级排序

| 优先级 | 任务    | 原因                     |
| ------ | ------- | ------------------------ |
| P0     | 任务 1  | 基础框架，其他任务依赖   |
| P0     | 任务 2  | 配置管理，API 客户端依赖 |
| P0     | 任务 3  | API 客户端，核心功能依赖 |
| P0     | 任务 10 | 输出处理，所有命令依赖   |
| P0     | 任务 11 | 错误处理，所有命令依赖   |
| P1     | 任务 4  | 核心功能：模式管理       |
| P1     | 任务 5  | 核心功能：代理管理       |
| P1     | 任务 8  | 核心功能：配置管理       |
| P2     | 任务 6  | 辅助功能：服务管理       |
| P2     | 任务 7  | 辅助功能：系统代理       |
| P2     | 任务 9  | 辅助功能：订阅管理       |
| P3     | 任务 12 | 整合：根命令             |
| P3     | 任务 13 | 高级功能：规则管理       |
| P3     | 任务 14 | 高级功能：连接管理       |
| P3     | 任务 15 | 高级功能：缓存管理       |
| P3     | 任务 16 | 高级功能：Geo 管理       |
| P3     | 任务 17 | 高级功能：DNS 查询       |
| P3     | 任务 18 | 高级功能：监控管理       |
| P3     | 任务 19 | 高级功能：系统管理       |
| P3     | 任务 20 | 基础信息：版本查询       |

---

## 执行建议

1. **按优先级顺序执行**：先完成 P0 任务搭建基础框架
2. **增量测试**：每完成一个任务进行编译和单元测试
3. **集成验证**：完成 P1 任务后进行集成测试
4. **文档更新**：实现过程中如有设计变更，同步更新设计文档
5. **高级功能分期实现**：P3 级高级功能可在基础功能稳定后逐步实现
6. **WebSocket 支持**：监控管理功能需要额外实现 WebSocket 客户端，建议在完成基础 API 封装后进行
