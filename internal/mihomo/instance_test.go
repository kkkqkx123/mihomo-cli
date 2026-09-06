package mihomo

import (
	"os"
	"path/filepath"
	"testing"
)

func TestInstanceRegistry_RoundTrip(t *testing.T) {
	dir := t.TempDir()
	pidFile := filepath.Join(dir, "mihomo-1a2b3c4d5e6f.pid")

	reg := NewInstanceRegistry(pidFile)
	meta := InstanceMeta{
		PID:        12345,
		ConfigFile: filepath.Join(dir, "mihomo-config.yaml"),
		ExecPath:   filepath.Join(dir, "mihomo.exe"),
		APIAddr:    "127.0.0.1:9090",
		StartedAt:  "2026-09-05T10:00:00+08:00",
	}

	if err := reg.Save(meta); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	got, err := reg.Load()
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	if got.Version != MetaVersion {
		t.Errorf("Version = %d, want %d", got.Version, MetaVersion)
	}
	if got.PID != 12345 {
		t.Errorf("PID = %d, want 12345", got.PID)
	}
	if got.ConfigFile != meta.ConfigFile {
		t.Errorf("ConfigFile = %q, want %q", got.ConfigFile, meta.ConfigFile)
	}
	if got.ExecPath != meta.ExecPath {
		t.Errorf("ExecPath = %q, want %q", got.ExecPath, meta.ExecPath)
	}
	if got.APIAddr != meta.APIAddr {
		t.Errorf("APIAddr = %q, want %q", got.APIAddr, meta.APIAddr)
	}
}

func TestInstanceRegistry_SaveForcesVersion(t *testing.T) {
	dir := t.TempDir()
	reg := NewInstanceRegistry(filepath.Join(dir, "mihomo.pid"))

	if err := reg.Save(InstanceMeta{PID: 42}); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	got, err := reg.Load()
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	if got.Version != MetaVersion {
		t.Errorf("Version = %d, want %d", got.Version, MetaVersion)
	}
}

func TestInstanceRegistry_SaveInvalidPID(t *testing.T) {
	reg := NewInstanceRegistry(filepath.Join(t.TempDir(), "mihomo.pid"))
	if err := reg.Save(InstanceMeta{PID: 0}); err == nil {
		t.Error("Save with PID=0 should fail")
	}
}

func TestInstanceRegistry_EmptyPIDFile(t *testing.T) {
	reg := NewInstanceRegistry("")
	if err := reg.Save(InstanceMeta{PID: 1}); err != nil {
		t.Errorf("Save with empty pidFile should no-op, got %v", err)
	}
	if _, err := reg.Load(); err == nil {
		t.Error("Load with empty pidFile should fail")
	}
	if reg.Exists() {
		t.Error("Exists() with empty pidFile should be false")
	}
}

func TestReadInstanceFile_PlainPIDRejected(t *testing.T) {
	dir := t.TempDir()
	pidFile := filepath.Join(dir, "mihomo-1a2b3c4d5e6f.pid")
	// 纯数字 PID 属于旧格式，开发阶段不兼容，应视为损坏文件
	if err := os.WriteFile(pidFile, []byte("9876\n"), 0644); err != nil {
		t.Fatal(err)
	}

	if _, err := ReadInstanceFile(pidFile); err == nil {
		t.Error("ReadInstanceFile on plain-number PID file should fail (legacy format not supported)")
	}
}

func TestReadInstanceFile_Corrupted(t *testing.T) {
	dir := t.TempDir()
	pidFile := filepath.Join(dir, "mihomo-broken.pid")
	if err := os.WriteFile(pidFile, []byte("not-a-pid-at-all"), 0644); err != nil {
		t.Fatal(err)
	}

	if _, err := ReadInstanceFile(pidFile); err == nil {
		t.Error("ReadInstanceFile on corrupted file should fail")
	}

	// 半截 JSON（写入中断）也应报错而非返回错误 PID
	truncFile := filepath.Join(dir, "mihomo-truncated.pid")
	if err := os.WriteFile(truncFile, []byte(`{"version":1,"pid":123`), 0644); err != nil {
		t.Fatal(err)
	}
	if _, err := ReadInstanceFile(truncFile); err == nil {
		t.Error("ReadInstanceFile on truncated JSON should fail")
	}

	// 版本号不匹配（version != 1）也应报错
	wrongVerFile := filepath.Join(dir, "mihomo-wrongver.pid")
	if err := os.WriteFile(wrongVerFile, []byte(`{"version":2,"pid":123}`), 0644); err != nil {
		t.Fatal(err)
	}
	if _, err := ReadInstanceFile(wrongVerFile); err == nil {
		t.Error("ReadInstanceFile on wrong-version file should fail")
	}
}

func TestListInstanceFiles(t *testing.T) {
	dir := t.TempDir()

	// v1 元数据文件
	reg := NewInstanceRegistry(filepath.Join(dir, "mihomo-1a2b3c4d5e6f.pid"))
	if err := reg.Save(InstanceMeta{PID: 111, ConfigFile: filepath.Join(dir, "a.yaml"), ExecPath: "a"}); err != nil {
		t.Fatal(err)
	}

	// 纯数字旧格式文件（开发阶段不兼容，应被跳过）
	_ = os.WriteFile(filepath.Join(dir, "mihomo-oldname.pid"), []byte("222"), 0644)

	// 损坏文件（应被跳过）
	_ = os.WriteFile(filepath.Join(dir, "mihomo-corrupt.pid"), []byte("garbage"), 0644)

	// 非 .pid 文件（应被忽略）
	_ = os.WriteFile(filepath.Join(dir, "state-1a2b3c4d5e6f.json"), []byte("{}"), 0644)

	// 子目录（应被忽略）
	if err := os.MkdirAll(filepath.Join(dir, "mihomo-dir.pid"), 0755); err != nil {
		t.Fatal(err)
	}

	instances, err := ListInstanceFiles(dir)
	if err != nil {
		t.Fatalf("ListInstanceFiles failed: %v", err)
	}

	if len(instances) != 1 {
		t.Fatalf("got %d instances, want 1: %+v", len(instances), instances)
	}

	if instances[0].PID != 111 {
		t.Errorf("unexpected PID: %d, want 111", instances[0].PID)
	}
}

func TestListInstanceFiles_MissingDir(t *testing.T) {
	instances, err := ListInstanceFiles(filepath.Join(t.TempDir(), "does-not-exist"))
	if err != nil {
		t.Fatalf("missing dir should return empty list, got err %v", err)
	}
	if len(instances) != 0 {
		t.Errorf("got %d instances, want 0", len(instances))
	}
}

func TestInstanceRegistry_RemoveAndExists(t *testing.T) {
	dir := t.TempDir()
	pidFile := filepath.Join(dir, "mihomo-1a2b3c4d5e6f.pid")
	reg := NewInstanceRegistry(pidFile)

	if reg.Exists() {
		t.Error("Exists() should be false before Save")
	}
	if err := reg.Save(InstanceMeta{PID: 1}); err != nil {
		t.Fatal(err)
	}
	if !reg.Exists() {
		t.Error("Exists() should be true after Save")
	}
	if err := reg.Remove(); err != nil {
		t.Fatalf("Remove failed: %v", err)
	}
	if reg.Exists() {
		t.Error("Exists() should be false after Remove")
	}
	// 重复删除应静默成功
	if err := reg.Remove(); err != nil {
		t.Errorf("second Remove should no-op, got %v", err)
	}
}
