package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/kkkqkx123/mihomo-cli/internal/config"
	"github.com/kkkqkx123/mihomo-cli/internal/output"
	pkgerrors "github.com/kkkqkx123/mihomo-cli/pkg/errors"
)

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "管理 CLI 配置",
	Long:  `管理 CLI 工具的本地配置，包括初始化、查看和设置配置项。`,
}

func init() {
	rootCmd.AddCommand(configCmd)
	configCmd.AddCommand(newConfigInitCmd())
	configCmd.AddCommand(newConfigShowCmd())
	configCmd.AddCommand(newConfigSetCmd())
}

// newConfigInitCmd 创建 config init 命令
func newConfigInitCmd() *cobra.Command {
	var force bool

	cmd := &cobra.Command{
		Use:   "init",
		Short: "初始化配置文件",
		Long:  `创建默认配置文件，如果配置文件已存在，需要使用 --force 参数覆盖。`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runConfigInit(force)
		},
	}

	cmd.Flags().BoolVarP(&force, "force", "f", false, "强制覆盖已存在的配置文件")

	return cmd
}

// runConfigInit 执行配置初始化
func runConfigInit(force bool) error {
	// 创建配置管理器
	cfgManager, err := config.NewConfigManager()
	if err != nil {
		return pkgerrors.ErrConfig("failed to create config manager", err)
	}

	// 获取默认配置路径
	configPath := cfgManager.GetDefaultCLIConfigPath()

	// 检查配置文件是否存在
	if _, err := os.Stat(configPath); err == nil {
		if !force {
			output.Warning("配置文件已存在: %s", configPath)
			output.Warning("如需覆盖，请使用 --force 参数")
			return nil
		}
		output.Warning("配置文件已存在，将被覆盖: %s", configPath)
	}

	// 创建默认配置
	cfg := config.GetDefaultConfig()

	// 保存配置
	if err := cfgManager.SaveCLIConfig(cfg, configPath); err != nil {
		return pkgerrors.ErrConfig("failed to save config", err)
	}

	output.Success("配置文件创建成功: %s", configPath)
	output.PrintKeyValue("API 地址", cfg.API.Address)
	output.PrintKeyValue("超时时间", fmt.Sprintf("%d 秒", cfg.API.Timeout))

	return nil
}

// newConfigShowCmd 创建 config show 命令
func newConfigShowCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "show",
		Short: "显示当前配置",
		Long:  `显示当前的 CLI 配置信息。`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runConfigShow()
		},
	}

	return cmd
}

// runConfigShow 执行配置显示
func runConfigShow() error {
	// 创建配置管理器
	cfgManager, err := config.NewConfigManager()
	if err != nil {
		return pkgerrors.ErrConfig("failed to create config manager", err)
	}

	// 获取默认配置路径
	configPath := cfgManager.GetDefaultCLIConfigPath()

	// 加载配置文件
	cfg, err := cfgManager.LoadCLIConfig(configPath)
	if err != nil {
		return pkgerrors.ErrConfig("failed to load config", err)
	}

	output.Println(output.CyanString("当前配置："))
	output.PrintKeyValue("API 地址", cfg.API.Address)
	output.PrintKeyValue("超时时间", fmt.Sprintf("%d 秒", cfg.API.Timeout))

	// Secret 脱敏显示
	if cfg.API.Secret != "" {
		output.PrintKeyValue("API 密钥", "****")
	} else {
		output.PrintKeyValue("API 密钥", "(未设置)")
	}

	return nil
}

// newConfigSetCmd 创建 config set 命令
func newConfigSetCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "set <key> <value>",
		Short: "设置配置项",
		Long:  `设置指定的配置项。支持的配置项：api.address, api.secret, api.timeout`,
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runConfigSet(args[0], args[1])
		},
	}

	return cmd
}

// runConfigSet 执行配置设置
func runConfigSet(key, value string) error {
	// 创建配置管理器
	cfgManager, err := config.NewConfigManager()
	if err != nil {
		return pkgerrors.ErrConfig("failed to create config manager", err)
	}

	// 获取当前配置
	cfg, err := cfgManager.LoadCLIConfigFromViper()
	if err != nil {
		// 如果加载失败，使用默认配置
		cfg = config.GetDefaultConfig()
	}

	// 解析并设置配置项
	switch key {
	case "api.address", "address":
		cfg.API.Address = value
	case "api.secret", "secret":
		cfg.API.Secret = value
	case "api.timeout", "timeout":
		var timeout int
		if _, err := fmt.Sscanf(value, "%d", &timeout); err != nil {
			return pkgerrors.ErrInvalidArg("invalid timeout value: "+value, nil)
		}
		cfg.API.Timeout = timeout
	default:
		output.Warning("invalid config key: %s", key)
		output.Warning("supported keys:")
		output.Warning("  api.address  - API address")
		output.Warning("  api.secret   - API secret")
		output.Warning("  api.timeout  - timeout (seconds)")
		return pkgerrors.ErrInvalidArg("invalid config key: "+key, nil)
	}

	// 保存配置
	configPath := cfgManager.GetDefaultCLIConfigPath()
	if err := cfgManager.SaveCLIConfig(cfg, configPath); err != nil {
		return pkgerrors.ErrConfig("failed to save config", err)
	}

	// 脱敏显示 Secret
	displayValue := value
	if strings.Contains(key, "secret") {
		displayValue = "****"
	}

	output.Success("配置已更新: %s = %s", key, displayValue)

	return nil
}
