package mihomo

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"

	pkgerrors "github.com/kkkqkx123/mihomo-cli/pkg/errors"
)

// MetaVersion PID 元数据版本。当前格式固定为 1（JSON 元数据）。
// 开发阶段不兼容任何旧数据格式，版本号锁定，不做向后兼容读取。
const MetaVersion = 1

// InstanceMeta 单实例元数据（v1，JSON 形式写入 .pid 文件）。
// 记录配置文件、可执行文件与 API 地址，使进程发现（ps/status）
// 无需通过文件名反推配置。
type InstanceMeta struct {
	Version    int    `json:"version"`
	PID        int    `json:"pid"`
	ConfigFile string `json:"config_file,omitempty"` // mihomo 配置文件绝对路径；空表示未指定
	ExecPath   string `json:"exec_path,omitempty"`   // 内核可执行文件绝对路径
	APIAddr    string `json:"api_addr,omitempty"`    // external-controller 地址，如 127.0.0.1:9090
	StartedAt  string `json:"started_at,omitempty"`  // RFC3339
}

// InstanceRegistry 单个 PID 元数据文件的读写封装（跨平台通用）。
// 持有 pidFile 路径，提供原子写、读取、删除与存在性检查。
type InstanceRegistry struct {
	pidFile string
}

// NewInstanceRegistry 创建 PID 元数据注册器
func NewInstanceRegistry(pidFile string) *InstanceRegistry {
	return &InstanceRegistry{pidFile: pidFile}
}

// Save 将元数据原子写入 PID 文件（先写 .tmp 再 rename）。
// pidFile 为空时静默跳过（与旧 PIDFileManager.Save 行为一致）。
func (r *InstanceRegistry) Save(meta InstanceMeta) error {
	if r.pidFile == "" {
		return nil
	}

	meta.Version = MetaVersion
	if meta.PID <= 0 {
		return pkgerrors.ErrConfig("invalid PID in instance meta", nil)
	}

	return WriteInstanceFile(r.pidFile, meta)
}

// Load 读取 PID 元数据（v1 JSON 格式）。
func (r *InstanceRegistry) Load() (*InstanceMeta, error) {
	if r.pidFile == "" {
		return nil, pkgerrors.ErrConfig("PID file not configured", nil)
	}

	return ReadInstanceFile(r.pidFile)
}

// Remove 删除 PID 文件（不存在时静默成功）
func (r *InstanceRegistry) Remove() error {
	if r.pidFile == "" {
		return nil
	}
	if err := os.Remove(r.pidFile); err != nil && !os.IsNotExist(err) {
		return pkgerrors.ErrConfig("failed to remove PID file", err)
	}
	return nil
}

// Exists 检查 PID 文件是否存在
func (r *InstanceRegistry) Exists() bool {
	if r.pidFile == "" {
		return false
	}
	_, err := os.Stat(r.pidFile)
	return err == nil
}

// WriteInstanceFile 将元数据原子写入指定 PID 文件
func WriteInstanceFile(pidFile string, meta InstanceMeta) error {
	// 确保目录存在
	pidDir := filepath.Dir(pidFile)
	if err := os.MkdirAll(pidDir, 0755); err != nil {
		return pkgerrors.ErrConfig("failed to create PID directory", err)
	}

	data, err := json.Marshal(meta)
	if err != nil {
		return pkgerrors.ErrConfig("failed to marshal instance meta", err)
	}

	// 先写临时文件再 rename，避免写入中断留下半截内容
	tmpFile := pidFile + ".tmp"
	if err := os.WriteFile(tmpFile, data, 0644); err != nil {
		return pkgerrors.ErrConfig("failed to write PID file", err)
	}
	if err := os.Rename(tmpFile, pidFile); err != nil {
		os.Remove(tmpFile)
		return pkgerrors.ErrConfig("failed to rename PID file", err)
	}

	return nil
}

// ReadInstanceFile 读取 PID 文件（v1 JSON 元数据格式）：
//  1. 内容可解析为 JSON 且 version=1、PID>0 → 返回元数据；
//  2. 否则报错（由调用方决定跳过或终止）。
//
// 开发阶段不兼容任何旧数据格式（如纯数字 PID），解析失败一律视为损坏文件。
func ReadInstanceFile(pidFile string) (*InstanceMeta, error) {
	data, err := os.ReadFile(pidFile)
	if err != nil {
		return nil, pkgerrors.ErrConfig("failed to read PID file", err)
	}

	var meta InstanceMeta
	if err := json.Unmarshal(data, &meta); err != nil {
		return nil, pkgerrors.ErrConfig("invalid PID file format: "+pidFile, nil)
	}
	if meta.Version != MetaVersion || meta.PID <= 0 {
		return nil, pkgerrors.ErrConfig("invalid PID file format: "+pidFile, nil)
	}

	return &meta, nil
}

// ListInstanceFiles 遍历目录下所有 *.pid 文件并读取元数据。
// 损坏/不可读的文件被跳过（不中断整体扫描）。
func ListInstanceFiles(pidDir string) ([]InstanceMeta, error) {
	instances := []InstanceMeta{}

	entries, err := os.ReadDir(pidDir)
	if err != nil {
		if os.IsNotExist(err) {
			return instances, nil
		}
		return nil, pkgerrors.ErrConfig("failed to read pid directory", err)
	}

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".pid") {
			continue
		}

		meta, err := ReadInstanceFile(filepath.Join(pidDir, entry.Name()))
		if err != nil {
			// 文件损坏或格式未知，跳过（不误判、不误杀）
			continue
		}

		instances = append(instances, *meta)
	}

	return instances, nil
}
