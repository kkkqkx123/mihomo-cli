package config

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

// TestResolver_ExplicitRelativeToBaseDir 显式相对路径以 baseDir 为基准
func TestResolver_ExplicitRelativeToBaseDir(t *testing.T) {
	dir := t.TempDir()
	exe := filepath.Join(dir, "mihomo.exe")
	if err := os.WriteFile(exe, []byte("x"), 0755); err != nil {
		t.Fatal(err)
	}

	r := NewExecutableResolver(dir)
	got, source, err := r.Resolve("mihomo.exe", nil)
	if err != nil {
		t.Fatalf("Resolve failed: %v", err)
	}
	if got != filepath.Clean(exe) {
		t.Errorf("got %q, want %q", got, exe)
	}
	if source != ResolveSourceExplicit {
		t.Errorf("source = %q, want %q", source, ResolveSourceExplicit)
	}
}

// TestResolver_ExplicitAbsolute 显式绝对路径直接命中
func TestResolver_ExplicitAbsolute(t *testing.T) {
	dir := t.TempDir()
	exe := filepath.Join(dir, "mihomo")
	if err := os.WriteFile(exe, []byte("x"), 0755); err != nil {
		t.Fatal(err)
	}

	r := NewExecutableResolver(dir)
	got, _, err := r.Resolve(exe, nil)
	if err != nil {
		t.Fatalf("Resolve failed: %v", err)
	}
	if got != filepath.Clean(exe) {
		t.Errorf("got %q, want %q", got, exe)
	}
}

// TestResolver_ExplicitMissing 显式路径不存在必须报错（不回退搜索）
func TestResolver_ExplicitMissing(t *testing.T) {
	r := NewExecutableResolver(t.TempDir())
	if _, _, err := r.Resolve("does-not-exist-mihomo", nil); err == nil {
		t.Error("Resolve with missing explicit path should fail")
	}
}

// TestResolver_ExplicitDotExeAutoComplete Windows 上 .exe 自动补全
func TestResolver_ExplicitDotExeAutoComplete(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("windows-only behavior")
	}

	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "mihomo.exe"), []byte("x"), 0755); err != nil {
		t.Fatal(err)
	}

	r := NewExecutableResolver(dir)
	got, _, err := r.Resolve("mihomo", nil)
	if err != nil {
		t.Fatalf("Resolve failed: %v", err)
	}
	if got != filepath.Join(dir, "mihomo.exe") {
		t.Errorf("got %q, want %q", got, filepath.Join(dir, "mihomo.exe"))
	}
}

// TestResolver_SearchDisabled 禁止搜索且无显式路径 → 报错
func TestResolver_SearchDisabled(t *testing.T) {
	search := false
	r := NewExecutableResolver(t.TempDir())
	if _, _, err := r.Resolve("", &search); err == nil {
		t.Error("Resolve with search disabled and no explicit path should fail")
	}
}

// TestResolver_EmptyExplicit_SearchChain 空配置走 PATH → 平台候选链
// 无 mihomo 可执行文件时返回聚合错误而非 panic
func TestResolver_EmptyExplicit_SearchChain(t *testing.T) {
	r := NewExecutableResolver(t.TempDir())
	_, source, err := r.Resolve("", nil)
	if err != nil {
		// 搜索失败：应返回聚合错误（含候选目录提示）
		if len(err.Error()) < 20 {
			t.Errorf("aggregate error too short: %v", err)
		}
		return
	}
	// 本机若恰好存在 mihomo（PATH/候选目录），必须走搜索链 source
	if source != ResolveSourcePath && source != ResolveSourcePlatform {
		t.Errorf("source = %q, want path/platform", source)
	}
}
