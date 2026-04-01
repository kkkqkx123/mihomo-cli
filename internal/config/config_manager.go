package config

import (
	"github.com/spf13/viper"
)

// ConfigManager 统一的配置管理器
type ConfigManager struct {
	pathResolver *PathResolver
	loader       *Loader
}

// NewConfigManager 创建配置管理器
func NewConfigManager() (*ConfigManager, error) {
	pathResolver, err := NewPathResolver()
	if err != nil {
		return nil, err
	}

	return &ConfigManager{
		pathResolver: pathResolver,
		loader:       NewLoader(),
	}, nil
}

// GetPathResolver 获取路径解析器
func (cm *ConfigManager) GetPathResolver() *PathResolver {
	return cm.pathResolver
}

// LoadTomlConfig 加载 TOML 配置文件
func (cm *ConfigManager) LoadTomlConfig(configPath string) (*TomlConfig, error) {
	return LoadTomlConfig(configPath)
}

// FindTomlConfigPath 查找 TOML 配置文件路径
func (cm *ConfigManager) FindTomlConfigPath(configFile string) string {
	return FindTomlConfigPath(configFile)
}

// LoadCLIConfig 加载 CLI 配置
func (cm *ConfigManager) LoadCLIConfig(configPath string) (*CLIConfig, error) {
	return cm.loader.Load(configPath)
}

// LoadCLIConfigFromViper 从全局 Viper 实例加载 CLI 配置
func (cm *ConfigManager) LoadCLIConfigFromViper() (*CLIConfig, error) {
	return LoadFromViper()
}

// SaveCLIConfig 保存 CLI 配置
func (cm *ConfigManager) SaveCLIConfig(cfg *CLIConfig, configPath string) error {
	return cm.loader.Save(cfg, configPath)
}

// GetDefaultCLIConfigPath 获取默认 CLI 配置文件路径
func (cm *ConfigManager) GetDefaultCLIConfigPath() string {
	return cm.pathResolver.GetConfigFile()
}

// EnsureBaseDirs 确保所有基础目录存在
func (cm *ConfigManager) EnsureBaseDirs() error {
	return cm.pathResolver.EnsureBaseDirs()
}

// InitViperConfig 初始化 Viper 配置
func (cm *ConfigManager) InitViperConfig(cfgFile string) {
	// 设置默认值
	viper.SetDefault("api.address", "http://127.0.0.1:9090")
	viper.SetDefault("api.secret", "")
	viper.SetDefault("api.timeout", 10)

	if cfgFile != "" {
		// 使用指定的配置文件
		viper.SetConfigFile(cfgFile)
	} else {
		// 使用默认配置文件路径
		viper.AddConfigPath(cm.pathResolver.GetBaseDir())
		viper.SetConfigType("yaml")
		viper.SetConfigName("config")
	}

	// 支持环境变量
	viper.SetEnvPrefix("MIHOMO")
	viper.AutomaticEnv()

	// 读取配置文件（如果存在）
	_ = viper.ReadInConfig()
}
