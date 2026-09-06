package config

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	pkgerrors "github.com/kkkqkx123/mihomo-cli/pkg/errors"
)

// ResolveSource 可执行文件解析命中方式
type ResolveSource string

const (
	// ResolveSourceExplicit 命中 config.toml 显式路径
	ResolveSourceExplicit ResolveSource = "explicit"
	// ResolveSourcePath 命中 PATH 搜索（exec.LookPath）
	ResolveSourcePath ResolveSource = "path"
	// ResolveSourcePlatform 命中平台默认候选目录
	ResolveSourcePlatform ResolveSource = "platform"
)

// defaultExecutableName 平台默认内核可执行文件名
func defaultExecutableName() string {
	if runtime.GOOS == "windows" {
		return "mihomo.exe"
	}
	return "mihomo"
}

// ExecutableResolver 内核可执行文件解析器（全项目唯一查找规则）。
//
// 查找顺序（命中即停）：
//  1. 显式路径（config.toml [mihomo] executable）：
//     相对路径以 baseDir（config.toml 所在目录）为基准；Windows 上自动补全 .exe；
//     文件不存在则报错（不回退搜索，便于定位配置错误）。
//  2. 配置为空且允许搜索：exec.LookPath("mihomo")（Windows 上 Go 按 PATHEXT 自动补 .exe）。
//  3. 仍未命中：平台默认候选目录探测（CLI 同目录 → 进程 CWD → /usr/local/bin、/usr/bin）。
//  4. 全部失败：返回聚合错误（列出已尝试的路径与原因）。
type ExecutableResolver struct {
	baseDir string // 相对路径基准目录；空则取进程 CWD
}

// NewExecutableResolver 创建可执行文件解析器。
// baseDir 为 config.toml 所在目录；传入空串时相对路径以进程 CWD 为基准。
func NewExecutableResolver(baseDir string) *ExecutableResolver {
	if baseDir == "" {
		if cwd, err := os.Getwd(); err == nil {
			baseDir = cwd
		}
	}
	return &ExecutableResolver{baseDir: baseDir}
}

// Resolve 解析内核可执行文件绝对路径。
//
// explicit：config.toml [mihomo] executable 显式值（可空）。
// search：executable_search；nil 视为 true（默认允许搜索）。
// 返回规范化绝对路径与命中方式，便于报错诊断。
func (r *ExecutableResolver) Resolve(explicit string, search *bool) (string, ResolveSource, error) {
	allowSearch := search == nil || *search
	attempts := []string{}

	// 1. 显式路径
	if explicit != "" {
		path, err := r.resolveExplicit(explicit)
		if err != nil {
			return "", "", pkgerrors.ErrConfig(
				"mihomo executable not found: "+explicit+
					" (resolved as "+path+"), check [mihomo] executable in config.toml", err)
		}
		return path, ResolveSourceExplicit, nil
	}

	// 2. PATH 搜索
	if allowSearch {
		if exePath, err := exec.LookPath("mihomo"); err == nil {
			if absPath, absErr := filepath.Abs(exePath); absErr == nil {
				return filepath.Clean(absPath), ResolveSourcePath, nil
			}
			return filepath.Clean(exePath), ResolveSourcePath, nil
		}
		attempts = append(attempts, "PATH search for 'mihomo'")

		// 3. 平台默认候选目录
		if path, ok := r.probePlatformCandidates(); ok {
			return path, ResolveSourcePlatform, nil
		}
		attempts = append(attempts, r.platformCandidatePaths()...)
	}

	// 4. 聚合错误
	return "", "", pkgerrors.ErrConfig(r.aggregateError(attempts, allowSearch), nil)
}

// resolveExplicit 解析显式配置的路径（相对 baseDir，Windows 自动补全 .exe）
func (r *ExecutableResolver) resolveExplicit(explicit string) (string, error) {
	p := explicit
	if !filepath.IsAbs(p) {
		p = filepath.Join(r.baseDir, p)
	}
	p = filepath.Clean(p)

	candidates := []string{p}
	// Windows：用户写 mihomo 而文件为 mihomo.exe 时自动补全
	if runtime.GOOS == "windows" && filepath.Ext(p) == "" {
		candidates = append(candidates, p+".exe")
	}

	for _, cand := range candidates {
		if fi, err := os.Stat(cand); err == nil && !fi.IsDir() {
			return filepath.Clean(cand), nil
		}
	}

	// 返回最后一个尝试路径用于诊断
	return p, pkgerrors.ErrConfig("file does not exist", nil)
}

// probePlatformCandidates 探测平台默认候选目录，命中返回绝对路径
func (r *ExecutableResolver) probePlatformCandidates() (string, bool) {
	for _, dir := range r.platformCandidatePaths() {
		cand := filepath.Join(dir, defaultExecutableName())
		if fi, err := os.Stat(cand); err == nil && !fi.IsDir() {
			return filepath.Clean(cand), true
		}
	}
	return "", false
}

// platformCandidatePaths 平台默认候选目录列表（按优先级排列）
func (r *ExecutableResolver) platformCandidatePaths() []string {
	dirs := []string{}

	if exe, err := os.Executable(); err == nil {
		dirs = append(dirs, filepath.Dir(exe))
	}
	if cwd, err := os.Getwd(); err == nil {
		dirs = append(dirs, cwd)
	}
	if runtime.GOOS != "windows" {
		dirs = append(dirs, "/usr/local/bin", "/usr/bin")
	}

	return dirs
}

// aggregateError 生成聚合错误信息，列出所有已尝试的路径
func (r *ExecutableResolver) aggregateError(attempts []string, allowSearch bool) string {
	var sb strings.Builder
	sb.WriteString("mihomo executable not found. ")
	if !allowSearch {
		sb.WriteString("executable_search is disabled and [mihomo] executable is empty; ")
	}
	sb.WriteString("tried:")
	for _, a := range attempts {
		sb.WriteString("\n  - " + a)
	}
	sb.WriteString("\nhint: set [mihomo] executable in config.toml, or place mihomo on PATH.")
	return fmt.Sprintf("%s", sb.String())
}
