package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
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

// effectiveCLIConfigPath 返回本次运行生效的 CLI 配置路径
// 显式 --config 相对路径以调用时工作目录为基准，此处归一化为绝对路径展示与读写，避免相对值在日志中含糊。
func effectiveCLIConfigPath(cfgManager *config.ConfigManager) string {
	if cfgFile != "" {
		if abs, err := filepath.Abs(cfgFile); err == nil {
			return abs
		}
		return cfgFile
	}
	return cfgManager.GetDefaultCLIConfigPath()
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

	configPath := effectiveCLIConfigPath(cfgManager)

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
		Long:  `显示当前生效的 CLI 配置与配置文件路径，密钥脱敏展示。`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runConfigShow()
		},
	}

	return cmd
}

// runConfigShow 执行配置显示
func runConfigShow() error {
	cfgManager, err := config.NewConfigManager()
	if err != nil {
		return pkgerrors.ErrConfig("failed to create config manager", err)
	}
	configPath := effectiveCLIConfigPath(cfgManager)

	// 展示运行时配置，包含文件、环境变量与命令行覆盖后的结果
	cfg, err := cfgManager.LoadCLIConfigFromViper()
	if err != nil {
		return pkgerrors.ErrConfig("加载配置失败", err)
	}

	if outputFmt == "json" {
		masked := *cfg
		if masked.API.Secret != "" {
			masked.API.Secret = "****"
		}
		return output.PrintJSON(map[string]interface{}{
			"config_file": configPath,
			"config":      masked,
		})
	}

	output.Println(output.CyanString("当前配置："))
	output.PrintKeyValue("配置文件", configPath)
	output.PrintKeyValue("API 地址", cfg.API.Address)
	output.PrintKeyValue("超时时间", fmt.Sprintf("%d 秒", cfg.API.Timeout))
	if cfg.API.Secret != "" {
		output.PrintKeyValue("API 密钥", "****")
	} else {
		output.PrintKeyValue("API 密钥", "(未设置)")
	}
	if cfg.Proxy.TestURL != "" {
		output.PrintKeyValue("测速地址", cfg.Proxy.TestURL)
	} else {
		output.PrintKeyValue("测速地址", "(未设置)")
	}
	output.PrintKeyValue("测速超时", fmt.Sprintf("%d 毫秒", cfg.Proxy.Timeout))
	output.PrintKeyValue("测速并发", fmt.Sprintf("%d", cfg.Proxy.Concurrent))
	output.PrintKeyValue("测速批大小", fmt.Sprintf("%d", cfg.Proxy.BatchSize))
	output.PrintKeyValue("最大节点数", fmt.Sprintf("%d", cfg.Proxy.MaxNodes))
	if cfg.Output.File != "" {
		output.PrintKeyValue("输出文件", cfg.Output.File)
	} else {
		output.PrintKeyValue("输出文件", "(未设置)")
	}
	output.PrintKeyValue("输出模式", cfg.Output.Mode)
	output.PrintKeyValue("追加写入", fmt.Sprintf("%v", cfg.Output.Append))

	return nil
}

// newConfigSetCmd 创建 config set 命令
func newConfigSetCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "set <key> <value>",
		Short: "设置配置项",
		Long: `设置指定的配置项。支持的配置项：api.address, api.secret, api.timeout, proxy.test_url, proxy.timeout, proxy.concurrent, proxy.batch_size, proxy.max_nodes, output.file, output.mode, output.append`,
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
	configPath := effectiveCLIConfigPath(cfgManager)

	// 直接编辑配置文件，避免把临时命令行覆盖持久化
	var cfg *config.CLIConfig
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		cfg = config.GetDefaultConfig()
	} else {
		cfg, err = cfgManager.LoadCLIConfig(configPath)
		if err != nil {
			return pkgerrors.ErrConfig("加载配置失败", err)
		}
	}

	// 解析并设置配置项
	normalized := strings.ToLower(strings.TrimSpace(key))
	switch normalized {
	case "api.address", "address":
		cfg.API.Address = strings.TrimSpace(value)
	case "api.secret", "secret":
		cfg.API.Secret = value
	case "api.timeout", "timeout":
		timeout, err := strconv.Atoi(strings.TrimSpace(value))
		if err != nil {
			return pkgerrors.ErrInvalidArg("invalid timeout value: "+value, nil)
		}
		cfg.API.Timeout = timeout
	case "proxy.test_url", "proxy.test-url", "test_url":
		cfg.Proxy.TestURL = strings.TrimSpace(value)
	case "proxy.timeout":
		timeout, err := strconv.Atoi(strings.TrimSpace(value))
		if err != nil {
			return pkgerrors.ErrInvalidArg("invalid proxy.timeout value: "+value, nil)
		}
		cfg.Proxy.Timeout = timeout
	case "proxy.concurrent":
		concurrent, err := strconv.Atoi(strings.TrimSpace(value))
		if err != nil {
			return pkgerrors.ErrInvalidArg("invalid proxy.concurrent value: "+value, nil)
		}
		cfg.Proxy.Concurrent = concurrent
	case "proxy.batch_size", "proxy.batch-size":
		batchSize, err := strconv.Atoi(strings.TrimSpace(value))
		if err != nil {
			return pkgerrors.ErrInvalidArg("invalid proxy.batch_size value: "+value, nil)
		}
		cfg.Proxy.BatchSize = batchSize
	case "proxy.max_nodes", "proxy.max-nodes":
		maxNodes, err := strconv.Atoi(strings.TrimSpace(value))
		if err != nil {
			return pkgerrors.ErrInvalidArg("invalid proxy.max_nodes value: "+value, nil)
		}
		cfg.Proxy.MaxNodes = maxNodes
	case "output.file":
		cfg.Output.File = strings.TrimSpace(value)
	case "output.mode":
		cfg.Output.Mode = strings.ToLower(strings.TrimSpace(value))
	case "output.append":
		appendMode, err := strconv.ParseBool(strings.TrimSpace(value))
		if err != nil {
			return pkgerrors.ErrInvalidArg("invalid output.append value: "+value, nil)
		}
		cfg.Output.Append = appendMode
	default:
		output.Warning("invalid config key: %s", key)
		output.Warning("supported keys:")
		output.Warning("  api.address  - API address")
		output.Warning("  api.secret   - API secret")
		output.Warning("  api.timeout  - timeout (seconds)")
		output.Warning("  proxy.test_url   - test URL")
		output.Warning("  proxy.timeout    - test timeout (milliseconds)")
		output.Warning("  proxy.concurrent - concurrent tests")
		output.Warning("  proxy.batch_size - nodes per batch")
		output.Warning("  proxy.max_nodes  - max nodes to test")
		output.Warning("  output.file   - output file path")
		output.Warning("  output.mode   - console/file/both")
		output.Warning("  output.append - true/false")
		return pkgerrors.ErrInvalidArg("invalid config key: "+key, nil)
	}

	// 保存配置，保存时做完整校验
	if err := cfgManager.SaveCLIConfig(cfg, configPath); err != nil {
		return pkgerrors.ErrConfig("failed to save config", err)
	}

	// 脱敏显示 Secret
	displayValue := value
	if strings.Contains(normalized, "secret") {
		displayValue = "****"
	}

	output.Success("配置已更新: %s = %s", key, displayValue)

	return nil
}
