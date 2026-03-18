使用 Go 语言编写一个非交互式的 Mihomo CLI 管理工具是一个非常棒的想法。Go 的跨平台能力、单二进制分发特性以及对系统调用的支持非常适合此类工具。

以下是针对 **Windows 环境** 设计的 **无交互 CLI 客户端** 项目架构方案。

### 1. 项目架构设计 (Project Structure)

采用标准的 Go CLI 项目结构，使用 `cobra` 作为命令行框架，`viper` 管理本地配置。

```text
mihomo-ctl/
├── cmd/                   # 命令定义入口
│   ├── root.go            # 根命令 (全局 flags: --host, --secret, --output)
│   ├── mode.go            # 模式查询/修改 (mode get / mode set)
│   ├── proxy.go           # 节点管理 (proxy list / proxy switch / proxy test)
│   ├── service.go         # 服务管理 (service start / stop / install)
│   └── config.go          # 本地配置管理 (config init / update)
├── internal/              # 内部业务逻辑
│   ├── api/               # Mihomo External API 客户端封装
│   │   ├── client.go      # HTTP 客户端初始化
│   │   ├── proxy.go       # 代理相关 API 实现
│   │   └── config.go      # 配置/模式 API 实现
│   ├── svc/               # Windows 服务管理逻辑
│   │   └── windows.go     # SCM (Service Control Manager) 调用
│   └── utils/             # 工具函数
│       ├── output.go      # 格式化输出 (Table/JSON)
│       └── admin.go       # 管理员权限检查
├── main.go                # 程序入口
├── go.mod
└── README.md
```

### 2. 核心依赖 (Dependencies)

```bash
go mod init mihomo-ctl
go get github.com/spf13/cobra      # 命令行框架
go get github.com/spf13/viper      # 配置文件管理
go get github.com/fatih/color      # 彩色输出
go get golang.org/x/sys/windows    # Windows 系统调用 (权限/服务)
```

### 3. 核心设计原则

1.  **无状态 (Stateless)**: CLI 不保存运行时状态，所有状态查询实时请求 API。
2.  **配置持久化**: API 地址和 Secret 保存在本地 `%APPDATA%/mihomo-ctl/config.yaml`，避免每次输入。
3.  **查询与修改分离**:
    - **查询 (Query)**: `get`, `list`, `show`。只读，输出结果，Exit Code 0。
    - **修改 (Mutation)**: `set`, `switch`, `update`。写操作，输出确认信息，失败时 Exit Code 1。
4.  **输出格式可控**: 支持 `-o json` 方便脚本调用，默认人类可读格式。

### 4. 关键代码实现

#### 4.1 全局配置与 API 客户端 (`internal/api/client.go`)

封装 HTTP 请求，统一处理 Secret 和错误。

```go
package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

type Client struct {
	BaseURL string
	Secret  string
	Client  *http.Client
}

func NewClient(baseURL, secret string) *Client {
	return &Client{
		BaseURL: baseURL,
		Secret:  secret,
		Client:  &http.Client{Timeout: 10 * time.Second},
	}
}

func (c *Client) request(method, path string, body interface{}, result interface{}) error {
	req, err := http.NewRequest(method, c.BaseURL+path, nil)
	if err != nil {
		return err
	}

	if c.Secret != "" {
		req.Header.Set("Authorization", "Bearer "+c.Secret)
	}

	if body != nil && method != "GET" {
		// 处理 Body (略，需根据实际实现补充)
	}

	resp, err := c.Client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		return fmt.Errorf("API error: %d", resp.StatusCode)
	}

	if result != nil {
		return json.NewDecoder(resp.Body).Decode(result)
	}
	return nil
}
```

#### 4.2 命令区分设计 (`cmd/mode.go`)

通过 `cobra` 的子命令自然区分查询和修改。

```go
package cmd

import (
	"fmt"
	"mihomo-ctl/internal/api"
	"github.com/spf13/cobra"
)

var modeCmd = &cobra.Command{
	Use:   "mode",
	Short: "管理 Mihomo 运行模式",
}

// 查询命令 (Query)
var modeGetCmd = &cobra.Command{
	Use:   "get",
	Short: "获取当前模式",
	Run: func(cmd *cobra.Command, args []string) {
		client := initClient() // 从配置初始化
		var res map[string]interface{}
		err := client.request("GET", "/configs", nil, &res)
		if err != nil {
			fmt.Println("Error:", err)
			return
		}
		fmt.Printf("Current Mode: %s\n", res["mode"])
	},
}

// 修改命令 (Mutation)
var modeSetCmd = &cobra.Command{
	Use:   "set [mode]",
	Short: "切换模式 (rule/global/direct)",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		targetMode := args[0]
		client := initClient()

		// 构造 PATCH 请求 Body
		body := fmt.Sprintf(`{"mode": "%s"}`, targetMode)

		err := client.request("PATCH", "/configs", body, nil)
		if err != nil {
			fmt.Printf("Failed to switch mode: %v\n", err)
			// 修改失败应退出非 0 状态
			panic(1)
		}
		fmt.Printf("Mode switched to: %s\n", targetMode)
	},
}

func init() {
	modeCmd.AddCommand(modeGetCmd)
	modeCmd.AddCommand(modeSetCmd)
	rootCmd.AddCommand(modeCmd)
}
```

#### 4.3 节点切换与测速 (`cmd/proxy.go`)

实现批量测速和指定节点切换。

```go
// 简化的切换逻辑
var proxySwitchCmd = &cobra.Command{
	Use:   "switch [group] [node]",
	Short: "切换代理组的节点",
	Args:  cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		group, node := args[0], args[1]
		client := initClient()

		// PUT /proxies/{group} Body: {"name": "{node}"}
		body := fmt.Sprintf(`{"name": "%s"}`, node)
		err := client.request("PUT", fmt.Sprintf("/proxies/%s", group), body, nil)

		if err != nil {
			fmt.Printf("Switch failed: %v\n", err)
		} else {
			fmt.Printf("Switched %s -> %s\n", group, node)
		}
	},
}

// 测速逻辑 (简化版)
var proxyTestCmd = &cobra.Command{
	Use:   "test [group]",
	Short: "批量测试组内节点延迟",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		group := args[0]
		client := initClient()

		// 1. 获取组内所有节点
		var proxies map[string]interface{}
		client.request("GET", fmt.Sprintf("/proxies/%s", group), nil, &proxies)

		// 2. 遍历测速 (实际代码需处理并发)
		// ... 调用 /proxies/{node}/delay 接口

		fmt.Println("Speed test completed.")
	},
}
```

#### 4.4 Windows 服务管理 (`cmd/service.go`)

区分 **服务注册** (调用 mihomo.exe) 和 **服务控制** (调用 Windows SCM)。

```go
import (
	"os/exec"
	"golang.org/x/sys/windows/svc/mgr"
)

var serviceCmd = &cobra.Command{
	Use:   "service",
	Short: "管理 Windows 服务",
}

// 启动/停止 使用 Go 原生 SCM
var serviceStartCmd = &cobra.Command{
	Use:   "start",
	Run: func(cmd *cobra.Command, args []string) {
		m, err := mgr.Connect()
		if err != nil { return }
		defer m.Disconnect()

		s, err := m.OpenService("mihomo")
		if err != nil { return }
		defer s.Close()

		err = s.Start()
		if err != nil {
			fmt.Println("Start failed:", err)
		} else {
			fmt.Println("Service started.")
		}
	},
}

// 安装/卸载 委托给 mihomo.exe 自身 (最安全)
var serviceInstallCmd = &cobra.Command{
	Use:   "install",
	Run: func(cmd *cobra.Command, args []string) {
		// 假设 mihomo.exe 在同级目录或 PATH 中
		cmd := exec.Command("mihomo.exe", "service", "install")
		// 需要管理员权限检查
		if !utils.IsAdmin() {
			fmt.Println("Error: Admin privileges required.")
			return
		}
		output, err := cmd.CombinedOutput()
		fmt.Println(string(output))
		if err != nil { panic(1) }
	},
}
```

#### 4.5 权限检查 (`internal/utils/admin.go`)

Windows 下修改服务或系统代理需要管理员权限。

```go
package utils

import (
	"os"
	"syscall"
)

func IsAdmin() bool {
	_, err := os.Open("\\\\.\\PHYSICALDRIVE0")
	return err == nil
}

// 或者通过检查 Token
func IsUserAdmin() bool {
	var sid *syscall.SID
	err := syscall.AllocateAndInitializeSid(
		&syscall.SECURITY_NT_AUTHORITY,
		2,
		syscall.SECURITY_BUILTIN_DOMAIN_RID,
		syscall.DOMAIN_ALIAS_RID_ADMINS,
		0, 0, 0, 0, 0, 0,
		&sid)
	if err != nil { return false }
	defer syscall.FreeSid(sid)

	token := syscall.Token(0)
	return token.IsMember(sid)
}
```

### 5. 命令使用示例

编译后生成 `mihomo-ctl.exe`，使用方式如下：

#### 5.1 初始化配置 (只需一次)

```powershell
# 保存 API 地址和 Secret 到本地配置
.\mihomo-ctl.exe config init --host 127.0.0.1:9090 --secret "my-secret"
```

#### 5.2 查询操作 (Query)

```powershell
# 查看当前模式
.\mihomo-ctl.exe mode get

# 列出所有代理组
.\mihomo-ctl.exe proxy list

# 查看实时流量 (JSON 格式，方便脚本解析)
.\mihomo-ctl.exe traffic --output json
```

#### 5.3 修改操作 (Mutation)

```powershell
# 切换模式
.\mihomo-ctl.exe mode set rule

# 切换节点 (组名：Proxy, 节点名：US-01)
.\mihomo-ctl.exe proxy switch Proxy US-01

# 自动选择最快节点 (内置逻辑)
.\mihomo-ctl.exe proxy auto-select --group Proxy

# 服务管理 (需要管理员权限)
.\mihomo-ctl.exe service install
.\mihomo-ctl.exe service restart
```

### 6. 高级功能设计建议

1.  **自动选择最快节点 (`proxy auto-select`)**:
    - 在 Go 内部实现并发测速逻辑（使用 `errgroup` 限制并发数）。
    - 比较延迟值，调用 `PUT /proxies/{group}` 接口完成切换。
    - 输出最终选择的节点名称。

2.  **系统代理开关 (`system-proxy`)**:
    - 由于 Mihomo API 不管理系统代理，CLI 需直接操作注册表。
    - 命令：`.\mihomo-ctl.exe system-proxy on` / `off`。
    - 实现：调用 `winreg` 包修改 `HKCU\Software\Microsoft\Windows\CurrentVersion\Internet Settings`。

3.  **订阅更新 (`subscribe`)**:
    - 命令：`.\mihomo-ctl.exe subscribe update`。
    - 实现：下载远程 YAML -> 覆盖本地 `config.yaml` -> 调用 `service restart`。

4.  **脚本友好性**:
    - 所有命令支持 `--quiet` 模式，只输出关键结果。
    - 所有命令支持 `--output json`，方便被其他脚本（如 PowerShell, Python）调用解析。
    - 错误码规范：成功返回 0，API 错误返回 1，权限错误返回 2。

### 7. 构建与分发

使用 `go build` 并加上版本信息：

```bash
go build -ldflags "-s -w -X main.version=1.0.0" -o mihomo-ctl.exe
```

### 8. 总结

这个架构的优势在于：

1.  **解耦**: CLI 工具与内核进程分离，内核崩溃不影响 CLI 使用。
2.  **安全**: 敏感信息（Secret）本地存储，不暴露在命令行历史中。
3.  **灵活**: 既支持人工操作（人类可读输出），也支持自动化（JSON 输出/Exit Code）。
4.  **合规**: 遵循 Windows 服务管理规范，不暴力杀进程，而是通过 SCM 管理。

通过这种方式，你实际上构建了一个 **Headless 版的 Clash Verge**，完全可以通过 PowerShell 脚本或任务计划程序实现高度自动化的代理管理。
