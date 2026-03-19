# 其他平台实现改进分析

## 概述

本文档分析 Linux 和 Darwin/macOS 平台的系统代理和路由管理实现，对照 Windows 平台的改进措施，评估是否需要类似的改进。

---

## 一、Linux 平台分析

### 1. 系统代理实现 (`sysproxy_linux.go`)

#### 当前实现
- ✅ 支持环境变量代理设置
- ✅ 支持配置文件 (`/etc/environment.d/proxy.conf`, `/etc/environment`)
- ✅ 提供完整的 CRUD 操作
- ✅ 清晰的错误处理和用户提示

#### 改进建议

**需要添加的功能：**

1. **环境变量备份功能**
   - 原因：环境变量配置文件可能被意外修改
   - 建议：在启用代理前备份当前配置
   - 实现：添加 `BackupEnvSettings()` 和 `RestoreEnvSettings()` 方法

2. **环境变量残留检测**
   - 原因：进程异常退出时环境变量可能残留
   - 建议：在禁用代理后检查是否完全清理
   - 实现：添加 `CheckEnvResidual()` 方法

3. **配置文件差异比较**
   - 原因：帮助用户理解配置变更
   - 建议：提供配置差异比较功能
   - 实现：添加 `CompareEnvSettings()` 方法

#### 代码示例

```go
// EnvBackup 环境变量备份
type EnvBackup struct {
	ID        string    `json:"id"`
	Timestamp time.Time `json:"timestamp"`
	EnvFile   string    `json:"env_file"`
	Content   string    `json:"content"`
	Note      string    `json:"note,omitempty"`
}

// BackupEnvSettings 备份环境变量设置
func (sp *linuxSysProxy) BackupEnvSettings(note string) (*EnvBackup, error) {
	id := time.Now().Format("20060102-150405")

	// 优先备份 systemd environment.d 配置
	content := ""
	envFile := ProxyEnvFile
	if data, err := os.ReadFile(ProxyEnvFile); err == nil {
		content = string(data)
	} else {
		// 备份 /etc/environment
		envFile = ProxyEnvFileFallback
		if data, err := os.ReadFile(ProxyEnvFileFallback); err == nil {
			content = string(data)
		}
	}

	backup := &EnvBackup{
		ID:        id,
		Timestamp: time.Now(),
		EnvFile:   envFile,
		Content:   content,
		Note:      note,
	}

	return backup, nil
}

// RestoreEnvSettings 恢复环境变量设置
func (sp *linuxSysProxy) RestoreEnvSettings(backup *EnvBackup) error {
	return os.WriteFile(backup.EnvFile, []byte(backup.Content), 0644)
}
```

---

### 2. 路由管理实现 (`route_linux.go`)

#### 当前实现
- ✅ 支持 IPv4 和 IPv6 路由
- ✅ 提供完整的 CRUD 操作
- ✅ 使用 `ip` 命令（现代方式）
- ✅ 提供接口存在性和网关可达性检查

#### 改进建议

**需要添加的功能：**

1. **iptables 规则备份**
   - 原因：TProxy 模式会修改 iptables 规则
   - 建议：在启动 TProxy 前备份当前规则
   - 优先级：高

2. **iptables 规则恢复**
   - 原因：进程异常退出时规则可能残留
   - 建议：提供规则恢复功能
   - 优先级：高

3. **路由表备份功能**
   - 原因：TUN 模式会修改路由表
   - 建议：与 Windows 平台一致的备份功能
   - 优先级：中

#### 代码示例

```go
// IPTablesBackup iptables 规则备份
type IPTablesBackup struct {
	ID        string         `json:"id"`
	Timestamp time.Time      `json:"timestamp"`
	Rules     []IPTablesRule `json:"rules"`
	Note      string         `json:"note,omitempty"`
}

// BackupIPTablesRules 备份 iptables 规则
func (rm *RouteManager) BackupIPTablesRules(note string) (*IPTablesBackup, error) {
	id := time.Now().Format("20060102-150405")

	// 备份 mangle 表规则
	rules, err := rm.getIPTablesRules("mangle")
	if err != nil {
		return nil, err
	}

	backup := &IPTablesBackup{
		ID:        id,
		Timestamp: time.Now(),
		Rules:     rules,
		Note:      note,
	}

	return backup, nil
}

// getIPTablesRules 获取指定表的 iptables 规则
func (rm *RouteManager) getIPTablesRules(table string) ([]IPTablesRule, error) {
	cmd := exec.Command("iptables-save", "-t", table)
	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	var rules []IPTablesRule
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, ":") {
			continue
		}

		if strings.Contains(line, "mihomo") {
			rule, err := parseIPTablesRule(line, table)
			if err == nil {
				rules = append(rules, rule)
			}
		}
	}

	return rules, nil
}

// RestoreIPTablesRules 恢复 iptables 规则
func (rm *RouteManager) RestoreIPTablesRules(backup *IPTablesBackup) error {
	// 删除所有 Mihomo 相关的规则
	tables := []string{"mangle", "nat", "filter"}
	for _, table := range tables {
		// 清空自定义链
		chains := []string{"mihomo_prerouting", "mihomo_output", "mihomo_divert"}
		for _, chain := range chains {
			cmd := exec.Command("iptables", "-t", table, "-F", chain)
			cmd.Run()
		}
	}

	return nil
}
```

---

## 二、Darwin/macOS 平台分析

### 1. 系统代理实现 (`sysproxy_darwin.go`)

#### 当前实现
- ✅ 支持环境变量代理设置
- ✅ 支持配置文件 (`/etc/profile.d/proxy.sh`, `/etc/environment`)
- ✅ 提供完整的 CRUD 操作
- ✅ 清晰的错误处理和用户提示

#### 改进建议

**需要添加的功能：**

1. **macOS 系统代理设置备份**
   - 原因：macOS 有独立的系统代理设置（网络偏好设置）
   - 建议：通过 `networksetup` 命令备份系统代理设置
   - 优先级：高

2. **环境变量备份功能**
   - 原因：与 Linux 平台一致的需求
   - 建议：添加环境变量配置备份
   - 优先级：中

#### 代码示例

```go
// MacProxyBackup macOS 系统代理备份
type MacProxyBackup struct {
	ID        string                  `json:"id"`
	Timestamp time.Time               `json:"timestamp"`
	Services  map[string]ProxyService `json:"services"`
	Note      string                  `json:"note,omitempty"`
}

// ProxyService 网络服务代理配置
type ProxyService struct {
	HTTPEnabled   bool   `json:"http_enabled"`
	HTTPProxy     string `json:"http_proxy"`
	HTTPPort      int    `json:"http_port"`
	HTTPSEnabled  bool   `json:"https_enabled"`
	HTTPSProxy    string `json:"https_proxy"`
	HTTPSPort     int    `json:"https_port"`
	SOCKSEnabled  bool   `json:"socks_enabled"`
	SOCKSProxy    string `json:"socks_proxy"`
	SOCKSPort     int    `json:"socks_port"`
	AutoProxyURL  string `json:"auto_proxy_url"`
	ProxyAutoDisc bool   `json:"proxy_auto_disc"`
}

// BackupMacProxySettings 备份 macOS 系统代理设置
func (sp *darwinSysProxy) BackupMacProxySettings(note string) (*MacProxyBackup, error) {
	id := time.Now().Format("20060102-150405")

	// 获取所有网络服务
	services, err := sp.getNetworkServices()
	if err != nil {
		return nil, err
	}

	backup := &MacProxyBackup{
		ID:        id,
		Timestamp: time.Now(),
		Services:  services,
		Note:      note,
	}

	return backup, nil
}

// getNetworkServices 获取所有网络服务的代理配置
func (sp *darwinSysProxy) getNetworkServices() (map[string]ProxyService, error) {
	services := make(map[string]ProxyService)

	// 获取所有网络服务列表
	cmd := exec.Command("networksetup", "-listallnetworkservices")
	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || line == "*" {
			continue
		}

		// 获取每个服务的代理配置
		service, err := sp.getServiceProxyConfig(line)
		if err != nil {
			continue
		}
		services[line] = service
	}

	return services, nil
}

// getServiceProxyConfig 获取指定服务的代理配置
func (sp *darwinSysProxy) getServiceProxyConfig(service string) (ProxyService, error) {
	var proxy ProxyService

	// 获取 HTTP 代理状态
	cmd := exec.Command("networksetup", "-getwebproxy", service)
	output, _ := cmd.Output()
	if strings.Contains(string(output), "Enabled: Yes") {
		proxy.HTTPEnabled = true
		// 解析代理地址和端口
		// ...
	}

	// 获取 HTTPS 代理状态
	cmd = exec.Command("networksetup", "-getsecurewebproxy", service)
	output, _ = cmd.Output()
	if strings.Contains(string(output), "Enabled: Yes") {
		proxy.HTTPSEnabled = true
		// ...
	}

	// 获取 SOCKS 代理状态
	cmd = exec.Command("networksetup", "-getsocksfirewallproxy", service)
	output, _ = cmd.Output()
	if strings.Contains(string(output), "Enabled: Yes") {
		proxy.SOCKSEnabled = true
		// ...
	}

	return proxy, nil
}
```

---

### 2. 路由管理实现 (`route_darwin.go`)

#### 当前实现
- ✅ 支持 IPv4 和 IPv6 路由
- ✅ 提供完整的 CRUD 操作
- ✅ 使用 `netstat` 和 `route` 命令
- ✅ 提供接口存在性和网关可达性检查

#### 改进建议

**需要添加的功能：**

1. **路由表备份功能**
   - 原因：TUN 模式会修改路由表
   - 建议：与 Windows 平台一致的备份功能
   - 优先级：中

2. **pf 规则备份**
   - 原因：macOS 使用 pf (Packet Filter) 而不是 iptables
   - 建议：在启动 TProxy 前备份 pf 规则
   - 优先级：中

#### 代码示例

```go
// PfBackup pf 规则备份
type PfBackup struct {
	ID        string    `json:"id"`
	Timestamp time.Time `json:"timestamp"`
	Rules     string    `json:"rules"`
	Note      string    `json:"note,omitempty"`
}

// BackupPfRules 备份 pf 规则
func (rm *RouteManager) BackupPfRules(note string) (*PfBackup, error) {
	id := time.Now().Format("20060102-150405")

	// 获取当前 pf 规则
	cmd := exec.Command("pfctl", "-s", "rules")
	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	backup := &PfBackup{
		ID:        id,
		Timestamp: time.Now(),
		Rules:     string(output),
		Note:      note,
	}

	return backup, nil
}

// RestorePfRules 恢复 pf 规则
func (rm *RouteManager) RestorePfRules(backup *PfBackup) error {
	// 重载 pf 规则
	cmd := exec.Command("pfctl", "-f", "-")
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return err
	}

	go func() {
		defer stdin.Close()
		io.WriteString(stdin, backup.Rules)
	}()

	return cmd.Run()
}
```

---

## 三、跨平台改进建议

### 1. 统一备份接口

**建议：** 为所有平台提供统一的备份接口

```go
// SystemBackup 统一的系统备份接口
type SystemBackup interface {
	BackupSysProxy(note string) (interface{}, error)
	RestoreSysProxy(backup interface{}) error
	BackupRoutes(note string) (interface{}, error)
	RestoreRoutes(backup interface{}) error
	BackupFirewall(note string) (interface{}, error)
	RestoreFirewall(backup interface{}) error
}

// LinuxBackup Linux 平台备份实现
type LinuxBackup struct {
	routeManager *RouteManager
	sysProxy     SysProxy
}

func (lb *LinuxBackup) BackupSysProxy(note string) (*EnvBackup, error) {
	return lb.sysProxy.(*linuxSysProxy).BackupEnvSettings(note)
}

func (lb *LinuxBackup) BackupFirewall(note string) (*IPTablesBackup, error) {
	return lb.routeManager.BackupIPTablesRules(note)
}

// DarwinBackup macOS 平台备份实现
type DarwinBackup struct {
	routeManager *RouteManager
	sysProxy     SysProxy
}

func (db *DarwinBackup) BackupSysProxy(note string) (*MacProxyBackup, error) {
	return db.sysProxy.(*darwinSysProxy).BackupMacProxySettings(note)
}

func (db *DarwinBackup) BackupFirewall(note string) (*PfBackup, error) {
	return db.routeManager.BackupPfRules(note)
}
```

---

### 2. 改进 process_handler.go 的跨平台支持

**当前问题：** `backupSystemConfig()` 方法只支持 Windows 平台的注册表备份

**建议改进：**

```go
// backupSystemConfig 备份系统配置（跨平台）
func (ph *ProcessHandler) backupSystemConfig(cfg *config.TomlConfig, hasTUN, hasTProxy bool) error {
	dataDir, err := config.GetDataDir()
	if err != nil {
		return err
	}

	backupDir := filepath.Join(dataDir, "system-backups")
	if err := os.MkdirAll(backupDir, 0755); err != nil {
		return err
	}

	scm, err := config.NewSystemConfigManager()
	if err != nil {
		return err
	}

	// 备份路由表（所有平台）
	if hasTUN || hasTProxy {
		routeManager := scm.GetRouteManager()
		routeBackup, err := routeManager.BackupRoutes("pre-start backup")
		if err == nil {
			_, err = routeManager.SaveBackup(routeBackup)
			if err != nil {
				output.Warning("failed to save route backup: " + err.Error())
			}
		}
	}

	// 备份 TUN 接口状态（所有平台）
	if hasTUN {
		tunManager := scm.GetTUNManager()
		tunBackup, err := tunManager.BackupTUNState("pre-start backup")
		if err == nil {
			_, err = tunManager.SaveTUNBackup(tunBackup)
			if err != nil {
				output.Warning("failed to save TUN backup: " + err.Error())
			}
		}
	}

	// 平台特定备份
	spm := config.NewSystemProxyManager()
	if spm.IsSupported() {
		backup, err := ph.backupPlatformSpecific(spm)
		if err == nil {
			ph.savePlatformBackup(backupDir, backup)
		}
	}

	return nil
}

// backupPlatformSpecific 备份平台特定配置
func (ph *ProcessHandler) backupPlatformSpecific(spm config.SysProxy) (interface{}, error) {
	switch spm.(type) {
	case *sysproxy.WindowsSysProxy:
		return spm.(*sysproxy.WindowsSysProxy).BackupRegistrySettings("pre-start backup")
	case *sysproxy.LinuxSysProxy:
		return spm.(*sysproxy.LinuxSysProxy).BackupEnvSettings("pre-start backup")
	case *sysproxy.DarwinSysProxy:
		return spm.(*sysproxy.DarwinSysProxy).BackupMacProxySettings("pre-start backup")
	default:
		return nil, nil
	}
}

// savePlatformBackup 保存平台特定备份
func (ph *ProcessHandler) savePlatformBackup(backupDir string, backup interface{}) error {
	if backup == nil {
		return nil
	}

	backupData, err := json.MarshalIndent(backup, "", "  ")
	if err != nil {
		return err
	}

	// 根据备份类型生成文件名
	var filename string
	switch backup.(type) {
	case *sysproxy.RegistryBackup:
		backup := backup.(*sysproxy.RegistryBackup)
		filename = fmt.Sprintf("registry-backup-%s.json", backup.ID)
	case *sysproxy.EnvBackup:
		backup := backup.(*sysproxy.EnvBackup)
		filename = fmt.Sprintf("env-backup-%s.json", backup.ID)
	case *sysproxy.MacProxyBackup:
		backup := backup.(*sysproxy.MacProxyBackup)
		filename = fmt.Sprintf("macproxy-backup-%s.json", backup.ID)
	default:
		return fmt.Errorf("unknown backup type")
	}

	backupFile := filepath.Join(backupDir, filename)
	return os.WriteFile(backupFile, backupData, 0644)
}
```

---

## 四、改进优先级总结

### Linux 平台

| 功能 | 优先级 | 状态 |
|------|--------|------|
| iptables 规则备份 | 高 | 未实现 |
| iptables 规则恢复 | 高 | 未实现 |
| 环境变量备份 | 中 | 未实现 |
| 环境变量残留检测 | 中 | 未实现 |
| 路由表备份 | 中 | 已实现（通用） |

### Darwin/macOS 平台

| 功能 | 优先级 | 状态 |
|------|--------|------|
| 系统代理设置备份 | 高 | 未实现 |
| pf 规则备份 | 中 | 未实现 |
| pf 规则恢复 | 中 | 未实现 |
| 环境变量备份 | 中 | 未实现 |
| 路由表备份 | 中 | 已实现（通用） |

### 跨平台

| 功能 | 优先级 | 状态 |
|------|--------|------|
| 统一备份接口 | 中 | 未实现 |
| process_handler 跨平台支持 | 高 | 需要改进 |

---

## 五、实施建议

### 第一阶段（高优先级）

1. **Linux 平台：**
   - 实现 iptables 规则备份功能
   - 实现 iptables 规则恢复功能

2. **Darwin/macOS 平台：**
   - 实现系统代理设置备份功能（通过 networksetup）

3. **跨平台：**
   - 改进 `process_handler.go` 的跨平台备份支持

### 第二阶段（中优先级）

1. **Linux 平台：**
   - 实现环境变量备份功能
   - 实现环境变量残留检测

2. **Darwin/macOS 平台：**
   - 实现 pf 规则备份功能
   - 实现 pf 规则恢复功能
   - 实现环境变量备份功能

3. **跨平台：**
   - 实现统一备份接口

---

## 六、总结

### 已实现的跨平台功能

✅ 路由表管理（所有平台）
✅ TUN 接口管理（所有平台）
✅ 系统代理管理（Windows/Linux/Darwin）

### 需要改进的地方

❌ Linux iptables 规则备份（高优先级）
❌ Darwin/macOS 系统代理设置备份（高优先级）
❌ process_handler 跨平台备份支持（高优先级）
❌ Linux 环境变量备份（中优先级）
❌ Darwin/macOS pf 规则备份（中优先级）

### 关键差异

| 平台 | 系统代理 | 防火墙 | 路由管理 |
|------|----------|--------|----------|
| Windows | 注册表 | 不适用 | route print |
| Linux | 环境变量 | iptables | ip route |
| Darwin | 环境变量 + networksetup | pf | netstat/route |

各平台的系统代理和防火墙管理方式差异较大，需要为每个平台提供特定的备份和恢复功能。路由管理已经实现了跨平台支持，但需要确保备份功能在所有平台上都能正常工作。

---

## 七、参考文档

- `internal/sysproxy/sysproxy_linux.go` - Linux 系统代理实现
- `internal/sysproxy/sysproxy_darwin.go` - Darwin/macOS 系统代理实现
- `internal/system/route_linux.go` - Linux 路由管理实现
- `internal/system/route_darwin.go` - Darwin/macOS 路由管理实现
- `internal/system/types.go` - 共享类型定义
- `internal/mihomo/process_handler.go` - 进程处理器