package config

import (
	"crypto/sha256"
	"encoding/hex"
	"path/filepath"
)

// DefaultIdentity 空配置（未显式指定 mihomo 配置文件）时使用的身份标识。
// 用于 state-default.json / lock-default 等默认文件命名。
const DefaultIdentity = "default"

// IdentityOf 计算配置文件的唯一身份标识。
//
// 输入为配置文件路径：先归一化为绝对路径（filepath.Abs + filepath.Clean），
// 再取路径 SHA-256 的前 12 位十六进制（48 bit）。同一配置文件无论以何种
// 相对/绝对写法传入，得到的身份都一致。
//
// 空路径返回 DefaultIdentity，保证与默认 PID/状态/锁文件名（mihomo.pid、
// state-default.json、lock-default）保持一致。
func IdentityOf(configFile string) string {
	if configFile == "" {
		return DefaultIdentity
	}

	abs := identityAbsPath(configFile)

	sum := sha256.Sum256([]byte(abs))
	return hex.EncodeToString(sum[:])[:12]
}

// identityAbsPath 归一化为绝对路径
func identityAbsPath(path string) string {
	abs, err := filepath.Abs(path)
	if err != nil {
		abs = path
	}
	return filepath.Clean(abs)
}
