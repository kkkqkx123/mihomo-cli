# Mihomo CLI - 项目上下文文档

## 项目概述

Mihomo CLI 是一个非交互式的 Mihomo 代理核心管理工具，专为 Windows 环境设计。该工具通过命令行界面提供对 Mihomo RESTful API 的完整管理能力，实现了无状态、可脚本化的代理管理解决方案。

### 项目目标

- 提供纯命令行界面的 Mihomo 管理工具，无需图形界面
- 支持自动化脚本和批量操作
- 与 Mihomo 核心进程分离，提高稳定性
- 遵循 Windows 服务管理规范

### 核心特性

- **无状态设计**：CLI 不保存运行时状态，所有状态查询实时请求 API
- **配置持久化**：API 地址和 Secret 保存到本地配置文件，避免重复输入
- **查询与修改分离**：明确的查询（get/list/show）和修改（set/switch/update）操作
- **输出格式可控**：支持表格和 JSON 两种输出格式，便于人工阅读和脚本解析

---

## 技术栈

### 编程语言

- **Go 1.26.1**：主开发语言

### 核心依赖

- **github.com/spf13/cobra v1.10.2**：命令行框架，用于构建 CLI 结构
- **github.com/spf13/viper v1.21.0**：配置管理，支持配置文件和环境变量
- **github.com/fatih/color v1.18.0**：彩色终端输出
- **github.com/olekukonko/tablewriter v0.0.5**：表格格式化输出

### 项目架构

- 模块化设计，使用标准 Go 项目结构
- 基于 Cobra 的命令树架构
- 分层设计：cmd（命令层）→ internal（业务逻辑层）→ pkg（公共类型层）

---

## 项目结构

```
mihomo-go/
├── cmd/                   # 命令定义入口
│   ├── root.go            # 根命令（全局 flags 和初始化）
│   ├── mode.go            # 模式管理（mode get/set）
│   ├── proxy.go           # 代理管理（list/switch/test/auto/unfix）
│   └── config.go          # CLI 配置管理（init/show/set）
├── internal/              # 内部业务逻辑实现
│   ├── api/               # Mihomo RESTful API 客户端封装
│   │   ├── client.go      # API 客户端主实现
│   │   ├── http.go        # HTTP 请求处理
│   │   ├── mode.go        # 模式相关 API
│   │   ├── proxy.go       # 代理相关 API
│   │   └── errors.go      # API 错误定义
│   ├── config/            # CLI 工具配置管理
│   │   ├── config.go      # 配置结构定义
│   │   └── loader.go      # 配置文件加载器
│   ├── proxy/             # 代理业务逻辑
│   │   ├── formatter.go   # 输出格式化
│   │   ├── selector.go    # 节点自动选择
│   │   └── tester.go      # 延迟测试
│   ├── output/            # 输出格式化
│   │   └── output.go      # 通用输出处理器
│   ├── service/           # Windows 服务管理（计划中）
│   ├── monitor/           # 监控功能（计划中）
│   ├── sysproxy/          # 系统代理管理（计划中）
│   └── util/              # 工具函数（计划中）
├── pkg/types/             # 公共类型定义
│   ├── mode.go            # 模式相关类型
│   └── proxy.go           # 代理相关类型
├── docs/                  # 项目文档
│   ├── spec/              # 需求规格说明
│   │   ├── spec.md        # 完整需求规格
│   │   ├── design.md      # 设计文档
│   │   ├── mihono-api.md  # Mihomo API 文档
│   │   └── tasks.md       # 任务清单
│   ├── architecture.md    # 架构设计文档
│   ├── 更换节点与批量测速.txt
│   ├── 内核处理订阅链接.txt
│   └── powershell使用.txt
├── mihomo-1.19.21/        # Mihomo 核心参考实现
├── main.go                # 程序入口
├── go.mod                 # Go 模块定义
└── README.md              # 项目说明（当前为空）
```

---

## 构建与运行

### 环境要求

- Go 1.26.1 或更高版本
- Windows 操作系统（主要目标平台）

### 运行

```bash
# 初始化配置
.\mihomo-cli.exe config init

# 查询当前模式
.\mihomo-cli.exe mode get

# 列出代理
.\mihomo-cli.exe proxy list

# 切换代理节点
.\mihomo-cli.exe proxy switch Proxy Node1

# 测试延迟
.\mihomo-cli.exe proxy test Proxy

# 自动选择最快节点
.\mihomo-cli.exe proxy auto Proxy

# JSON 格式输出
.\mihomo-cli.exe proxy list -o json
```

---

## 核心设计原则

### 1. 无状态（Stateless）

- CLI 不保存运行时状态
- 所有状态查询实时请求 API
- 避免状态同步问题

### 2. 配置持久化

- API 地址和 Secret 保存到本地配置文件
- 避免每次输入敏感信息
- 配置文件存储在用户目录，设置适当权限

### 3. 查询与修改分离

- 查询操作（Query）：get, list, show
- 修改操作（Mutation）：set, switch, update
- 明确的语义和退出码

### 4. 输出格式可控

- 支持表格和 JSON 两种输出
- 便于人工阅读和脚本解析
- 统一的输出格式规范

### 5. 权限管理

- 服务管理功能需要管理员权限
- 系统代理修改需要管理员权限
- 提供清晰的权限错误提示

## 文档资源

### Mihomo 参考

- `mihomo-1.19.21/`：Mihomo 核心参考实现（版本 1.19.21）
