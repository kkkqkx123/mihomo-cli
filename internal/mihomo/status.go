package mihomo

import (
	"context"
	"net"
	"net/url"
	"strings"
	"time"

	"github.com/kkkqkx123/mihomo-cli/internal/api"
	"github.com/kkkqkx123/mihomo-cli/internal/config"
	pkgerrors "github.com/kkkqkx123/mihomo-cli/pkg/errors"
)

// KernelState 内核状态。
type KernelState string

const (
	KernelStateRunning     KernelState = "running"
	KernelStateUnreachable KernelState = "unreachable"
	KernelStateStopped     KernelState = "stopped"
)

const (
	StatusManagerSystemd  = "systemd"
	StatusManagerCLI      = "cli"
	StatusManagerExternal = "external"
)

// StatusResult 内核状态查询结果。
type StatusResult struct {
	State        KernelState `json:"state"`
	Running      bool        `json:"running"`
	APIReachable bool        `json:"api_reachable"`
	PID          int         `json:"pid,omitempty"`
	APIURL       string      `json:"api"`
	Manager      string      `json:"manager,omitempty"`
	Unit         string      `json:"unit,omitempty"`
	ProcessCount int         `json:"process_count,omitempty"`
	APIError     string      `json:"api_error,omitempty"`
}

// QueryStatus 通过 API 与进程信息查询内核状态。
// API 地址和密钥由调用方提供，不依赖 TOML。
func QueryStatus(apiAddress, secret string) (*StatusResult, error) {
	if apiAddress == "" {
		return nil, pkgerrors.ErrConfig("API address is required", nil)
	}

	result := &StatusResult{APIURL: apiAddress}

	client := api.NewClient(apiAddress, secret, api.WithTimeout(2*time.Second))
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	if _, err := client.GetVersion(ctx); err != nil {
		result.APIError = err.Error()
	} else {
		result.APIReachable = true
		result.Running = true
	}

	pid := 0
	host, port := splitAPIHostPort(apiAddress)
	localAPI := isLocalAPIHost(host)
	if localAPI {
		pid = findStatusPID(host, port)
		if pid == 0 {
			processes := scanProcessesForStatus()
			result.ProcessCount = len(processes)
			if len(processes) == 1 {
				pid = processes[0].PID
			}
			if len(processes) > 0 {
				result.Running = true
			}
		}
	}

	if pid > 0 {
		result.PID = pid
		result.Manager, result.Unit = detectProcessManager(pid)
		if result.Manager == "" {
			if isCLIManagedPID(pid) {
				result.Manager = StatusManagerCLI
			} else {
				result.Manager = StatusManagerExternal
			}
		}
	}

	switch {
	case result.APIReachable:
		result.State = KernelStateRunning
	case result.Running:
		result.State = KernelStateUnreachable
	case !localAPI:
		// 远程 API 无法通过本机进程判断内核是否仍在运行。
		result.State = KernelStateUnreachable
	default:
		result.State = KernelStateStopped
	}

	return result, nil
}

// findStatusPID 通过 API 监听端口查找内核 PID，并校验进程身份。
func findStatusPID(host, port string) int {
	if host == "" || port == "" {
		return 0
	}

	// 端口查找目前基于 /proc/net/tcp（IPv4）；IPv6 地址交给进程扫描。
	if ip := net.ParseIP(host); ip != nil && ip.To4() == nil {
		return 0
	}

	pid := FindProcessByPort(host, port)
	if pid <= 0 {
		return 0
	}
	if ok, err := VerifyMihomoProcess(pid); err != nil || !ok {
		return 0
	}
	return pid
}

// isLocalAPIHost 判断 API 地址是否指向本机，只有本机地址才做进程补充。
func isLocalAPIHost(host string) bool {
	if host == "" || strings.EqualFold(host, "localhost") {
		return true
	}
	ip := net.ParseIP(host)
	if ip == nil {
		return false
	}
	return ip.IsLoopback() || ip.IsUnspecified()
}

// scanProcessesForStatus 静默扫描进程，仅用于状态补充，不向 stdout 输出警告。
func scanProcessesForStatus() []ProcessInfo {
	managed, _ := scanManagedProcesses()
	external, _ := scanExternalProcesses(managed)
	return append(managed, external...)
}

// isCLIManagedPID 判断 PID 是否命中 CLI 的实例元数据。
func isCLIManagedPID(pid int) bool {
	pathResolver, err := config.NewPathResolver()
	if err != nil {
		return false
	}
	instances, err := ListInstanceFiles(pathResolver.GetPIDDir())
	if err != nil {
		return false
	}

	execPath, execErr := GetProcessExecutable(pid)
	for _, meta := range instances {
		if meta.PID != pid {
			continue
		}
		if meta.ExecPath == "" || execErr != nil || sameExecutablePath(execPath, meta.ExecPath) {
			return true
		}
	}
	return false
}

// splitAPIHostPort 从 API URL 中提取 host 与 port。
func splitAPIHostPort(apiAddress string) (string, string) {
	parsed, err := url.Parse(apiAddress)
	if err != nil || parsed.Host == "" {
		return "", ""
	}

	host := parsed.Hostname()
	port := parsed.Port()
	if port == "" {
		switch parsed.Scheme {
		case "http":
			port = "80"
		case "https":
			port = "443"
		}
	}
	return host, port
}
