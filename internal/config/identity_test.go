package config

import (
	"path/filepath"
	"testing"
)

func TestIdentityOf_Empty(t *testing.T) {
	if got := IdentityOf(""); got != DefaultIdentity {
		t.Errorf("IdentityOf(\"\") = %q, want %q", got, DefaultIdentity)
	}
}

func TestIdentityOf_StableAcrossRelativeAndAbsolute(t *testing.T) {
	cwd, err := filepath.Abs(".")
	if err != nil {
		t.Fatal(err)
	}

	abs := filepath.Join(cwd, "sub", "mihomo-config.yaml")
	rel, err := filepath.Rel(cwd, abs)
	if err != nil {
		t.Fatal(err)
	}

	// 同一配置文件的不同路径写法必须得到同一身份
	if got, want := IdentityOf(abs), IdentityOf(rel); got != want {
		t.Errorf("IdentityOf mismatch: abs=%q rel=%q", got, want)
	}
}

func TestIdentityOf_NormalizesDotSegments(t *testing.T) {
	abs := filepath.Join(t.TempDir(), "mihomo-config.yaml")
	dirty := filepath.Join(filepath.Dir(abs), "..", filepath.Base(filepath.Dir(abs)), filepath.Base(abs))

	if got, want := IdentityOf(abs), IdentityOf(dirty); got != want {
		t.Errorf("IdentityOf with dot segments mismatch: %q vs %q", got, want)
	}
}

func TestIdentityOf_DifferentConfigsDiffer(t *testing.T) {
	dir := t.TempDir()
	a := filepath.Join(dir, "a.yaml")
	b := filepath.Join(dir, "b.yaml")

	if got, want := IdentityOf(a), IdentityOf(b); got == want {
		t.Errorf("IdentityOf(%q) == IdentityOf(%q) = %q, want different", a, b, got)
	}
}

func TestIdentityOf_Format(t *testing.T) {
	id := IdentityOf(filepath.Join(t.TempDir(), "cfg.yaml"))
	if len(id) != 12 {
		t.Errorf("IdentityOf length = %d, want 12", len(id))
	}
	for _, c := range id {
		if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f')) {
			t.Errorf("IdentityOf %q contains non-hex char %q", id, c)
		}
	}
}
