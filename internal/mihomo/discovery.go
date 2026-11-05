package mihomo

import (
	"github.com/kkkqkx123/mihomo-cli/internal/config"
	"github.com/kkkqkx123/mihomo-cli/internal/output"
	pkgerrors "github.com/kkkqkx123/mihomo-cli/pkg/errors"
)

// ScanMihomoProcesses 扫描所有 Mihomo 实例（managed + external 两源合并）。
// managed 实例在前（含元数据：配置/端口/可执行路径），external 实例在后；
// external 枚举失败仅降级为空，不影响 managed 结果。
func ScanMihomoProcesses() ([]ProcessInfo, error) {
	managed, err := scanManagedProcesses()
	if err != nil {
		return nil, err
	}

	external, err := scanExternalProcesses(managed)
	if err != nil {
		output.Warning("external process scan failed: " + err.Error())
		external = nil
	}

	return append(managed, external...), nil
}

// scanManagedProcesses 扫描本 CLI 托管的实例：
// InstanceRegistry 元数据 + 进程存活/可执行路径一致性校验（confirmed），
// 元数据缺失时以可执行文件名匹配兜底（likely）。
func scanManagedProcesses() ([]ProcessInfo, error) {
	pathResolver, err := config.NewPathResolver()
	if err != nil {
		return nil, pkgerrors.ErrConfig("failed to create path resolver", err)
	}

	instances, err := ListInstanceFiles(pathResolver.GetPIDDir())
	if err != nil {
		return nil, pkgerrors.ErrService("failed to list instance files", err)
	}

	processes := []ProcessInfo{}
	for _, meta := range instances {
		pid := meta.PID

		// 验证进程是否真实存在（防止 PID 复用误判）
		if !IsProcessRunning(pid) {
			continue
		}

		execPath, err := GetProcessExecutable(pid)
		if err != nil {
			continue
		}

		// 分级验证：元数据 ExecPath 与实时路径一致 → confirmed；否则名称匹配 → likely
		verified := VerifyLikely
		if meta.ExecPath != "" && sameExecutablePath(execPath, meta.ExecPath) {
			verified = VerifyConfirmed
		}
		if verified != VerifyConfirmed && !isMihomoExecutablePath(execPath) {
			continue
		}

		// API 端口：优先取元数据 external-controller；缺失时从配置文件解析
		apiPort := ""
		if meta.APIAddr != "" {
			apiPort = apiPortFromAddr(meta.APIAddr)
		} else if meta.ConfigFile != "" {
			if port, err := extractAPIPortFromConfig(meta.ConfigFile); err == nil {
				apiPort = port
			}
		}

		cmdline, _ := GetProcessCommandLine(pid)

		processes = append(processes, ProcessInfo{
			PID:        pid,
			ExecPath:   execPath,
			APIPort:    apiPort,
			StartTime:  meta.StartedAt,
			CmdLine:    cmdline,
			IsVerified: true,
			Source:     SourceManaged,
			Verified:   verified,
			Manager:    StatusManagerCLI,
			ConfigFile: meta.ConfigFile,
		})
	}

	return processes, nil
}

// scanExternalProcesses 枚举全进程表，按可执行文件名过滤出未托管的 mihomo 实例。
// 只做精确文件名匹配，避免把命令行中包含 mihomo 的 shell/CLI 进程误判为内核。
func scanExternalProcesses(managed []ProcessInfo) ([]ProcessInfo, error) {
	pids, err := listProcessPIDs()
	if err != nil {
		return nil, err
	}

	managedPIDs := map[int]bool{}
	for _, p := range managed {
		managedPIDs[p.PID] = true
	}

	var out []ProcessInfo
	for _, pid := range pids {
		if managedPIDs[pid] {
			continue
		}

		execPath, err := GetProcessExecutable(pid)
		if err != nil {
			continue
		}

		if !isMihomoExecutablePath(execPath) {
			continue
		}

		cmdline, _ := GetProcessCommandLine(pid)
		manager, unit := detectProcessManager(pid)

		apiPort := ""
		cfgFile := extractConfigFileFromCommandLine(cmdline)
		if cfgFile != "" {
			if port, err := extractAPIPortFromConfig(cfgFile); err == nil {
				apiPort = port
			}
		}

		out = append(out, ProcessInfo{
			PID:        pid,
			ExecPath:   execPath,
			APIPort:    apiPort,
			CmdLine:    cmdline,
			IsVerified: true,
			Source:     SourceExternal,
			Verified:   VerifyLikely,
			Manager:    manager,
			Unit:       unit,
			ConfigFile: cfgFile,
		})
	}

	return out, nil
}
