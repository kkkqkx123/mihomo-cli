package mihomo

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/kkkqkx123/mihomo-cli/internal/config"
	pkgerrors "github.com/kkkqkx123/mihomo-cli/pkg/errors"
)

// LifecycleStage 生命周期阶段
type LifecycleStage string

const (
	StagePreStart LifecycleStage = "pre-start"
	StageStarting LifecycleStage = "starting"
	StageRunning  LifecycleStage = "running"
	StagePreStop  LifecycleStage = "pre-stop"
	StageStopping LifecycleStage = "stopping"
	StageStopped  LifecycleStage = "stopped"
	StageFailed   LifecycleStage = "failed"
)

// ProcessState 进程状态
type ProcessState struct {
	PID             int            `json:"pid"`
	APIAddress      string         `json:"api_address"`
	Secret          string         `json:"secret"`
	ConfigFile      string         `json:"config_file"`
	StartedAt       time.Time      `json:"started_at"`
	LastHealthCheck time.Time      `json:"last_health_check"`
	Stage           LifecycleStage `json:"stage"`
	ConfigHash      string         `json:"config_hash"`
}

// StateManager 状态管理器
type StateManager struct {
	stateFile    string
	mu           sync.RWMutex
	state        *ProcessState
	pathResolver *config.PathResolver
}

// NewStateManager 创建状态管理器
func NewStateManager(configFile string) (*StateManager, error) {
	// 创建路径解析器
	pathResolver, err := config.NewPathResolver()
	if err != nil {
		return nil, err
	}

	return NewStateManagerWithResolver(configFile, pathResolver)
}

// NewStateManagerWithResolver 使用指定的路径解析器创建状态管理器
func NewStateManagerWithResolver(configFile string, pathResolver *config.PathResolver) (*StateManager, error) {
	// 获取状态文件路径
	stateFile := pathResolver.GetStateFilePath(configFile)

	sm := &StateManager{
		stateFile:    stateFile,
		pathResolver: pathResolver,
	}

	// 尝试加载现有状态
	_ = sm.Load()

	return sm, nil
}

// getStateFilePath 获取状态文件路径
func getStateFilePath(configFile string) (string, error) {
	baseDir, err := config.GetBaseDir()
	if err != nil {
		return "", err
	}

	// 根据配置文件生成唯一的状态文件名
	hash := generateConfigHash(configFile)
	return filepath.Join(baseDir, fmt.Sprintf("state-%s.json", hash)), nil
}

// Save 保存状态到文件
func (sm *StateManager) Save() error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	if sm.state == nil {
		return nil
	}

	// 序列化状态
	data, err := json.MarshalIndent(sm.state, "", "  ")
	if err != nil {
		return pkgerrors.ErrService("failed to marshal process state", err)
	}

	// 确保目录存在
	stateDir := filepath.Dir(sm.stateFile)
	if err := os.MkdirAll(stateDir, 0755); err != nil {
		return pkgerrors.ErrService("failed to create state directory", err)
	}

	// 写入文件
	if err := os.WriteFile(sm.stateFile, data, 0644); err != nil {
		return pkgerrors.ErrService("failed to write state file", err)
	}

	return nil
}

// Load 从文件加载状态
func (sm *StateManager) Load() error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	// 读取文件
	data, err := os.ReadFile(sm.stateFile)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return pkgerrors.ErrService("failed to read state file", err)
	}

	// 反序列化
	var state ProcessState
	if err := json.Unmarshal(data, &state); err != nil {
		return pkgerrors.ErrService("failed to unmarshal process state", err)
	}

	sm.state = &state
	return nil
}

// Update 更新状态
func (sm *StateManager) Update(updateFunc func(*ProcessState)) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	if sm.state == nil {
		sm.state = &ProcessState{}
	}

	updateFunc(sm.state)

	// 直接调用 save（不获取锁，因为已经持有锁）
	return sm.save()
}

// save 内部保存方法（调用者必须持有锁）
func (sm *StateManager) save() error {
	if sm.state == nil {
		return nil
	}

	// 序列化状态
	data, err := json.MarshalIndent(sm.state, "", "  ")
	if err != nil {
		return pkgerrors.ErrService("failed to marshal process state", err)
	}

	// 确保目录存在
	stateDir := filepath.Dir(sm.stateFile)
	if err := os.MkdirAll(stateDir, 0755); err != nil {
		return pkgerrors.ErrService("failed to create state directory", err)
	}

	// 写入文件
	if err := os.WriteFile(sm.stateFile, data, 0644); err != nil {
		return pkgerrors.ErrService("failed to write state file", err)
	}

	return nil
}

// Get 获取当前状态
func (sm *StateManager) Get() *ProcessState {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	if sm.state == nil {
		return nil
	}

	// 返回副本
	state := *sm.state
	return &state
}

// Clear 清除状态
func (sm *StateManager) Clear() error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	sm.state = nil

	// 删除状态文件
	if err := os.Remove(sm.stateFile); err != nil && !os.IsNotExist(err) {
		return pkgerrors.ErrService("failed to remove state file", err)
	}

	return nil
}

// IsStale 检查状态是否过期
func (sm *StateManager) IsStale() bool {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	if sm.state == nil {
		return true
	}

	// 检查进程是否还在运行
	if !IsProcessRunning(sm.state.PID) {
		return true
	}

	// 检查最后健康检查时间
	if !sm.state.LastHealthCheck.IsZero() {
		// 如果超过 5 分钟没有健康检查，认为状态过期
		if time.Since(sm.state.LastHealthCheck) > 5*time.Minute {
			return true
		}
	}

	return false
}

// SetStage 设置生命周期阶段
func (sm *StateManager) SetStage(stage LifecycleStage) error {
	return sm.Update(func(state *ProcessState) {
		state.Stage = stage
	})
}

// SetPID 设置进程 ID
func (sm *StateManager) SetPID(pid int) error {
	return sm.Update(func(state *ProcessState) {
		state.PID = pid
	})
}

// SetAPIAddress 设置 API 地址
func (sm *StateManager) SetAPIAddress(addr string) error {
	return sm.Update(func(state *ProcessState) {
		state.APIAddress = addr
	})
}

// SetSecret 设置密钥
func (sm *StateManager) SetSecret(secret string) error {
	return sm.Update(func(state *ProcessState) {
		state.Secret = secret
	})
}

// SetConfigFile 设置配置文件
func (sm *StateManager) SetConfigFile(configFile string) error {
	return sm.Update(func(state *ProcessState) {
		state.ConfigFile = configFile
	})
}

// SetStartedAt 设置启动时间
func (sm *StateManager) SetStartedAt(t time.Time) error {
	return sm.Update(func(state *ProcessState) {
		state.StartedAt = t
	})
}

// UpdateHealthCheck 更新健康检查时间
func (sm *StateManager) UpdateHealthCheck() error {
	return sm.Update(func(state *ProcessState) {
		state.LastHealthCheck = time.Now()
	})
}

// generateConfigHash 根据配置文件路径生成唯一 hash
func generateConfigHash(configFile string) string {
	if configFile == "" {
		return "default"
	}

	// 使用文件路径的绝对路径作为 hash
	absPath, err := filepath.Abs(configFile)
	if err != nil {
		absPath = configFile
	}

	// 使用 SHA256 计算路径的 hash
	h := sha256.New()
	h.Write([]byte(absPath))

	// 取前16个字符，提供足够的唯一性同时保持可读性
	hash := hex.EncodeToString(h.Sum(nil))[:16]
	return hash
}
