package cmd

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"gopkg.in/yaml.v3"

	"github.com/kkkqkx123/mihomo-cli/internal/api"
	"github.com/kkkqkx123/mihomo-cli/internal/config"
	"github.com/kkkqkx123/mihomo-cli/internal/errors"
	pkgerrors "github.com/kkkqkx123/mihomo-cli/pkg/errors"
)

var mihomoCmd = &cobra.Command{
	Use:   "mihomo",
	Short: "管理 Mihomo 配置",
	Long:  `管理 Mihomo 服务的运行时配置，包括热更新、重载和编辑配置文件。`,
}

func init() {
	rootCmd.AddCommand(mihomoCmd)
	mihomoCmd.AddCommand(newMihomoPatchCmd())
	mihomoCmd.AddCommand(newMihomoReloadCmd())
	mihomoCmd.AddCommand(newMihomoEditCmd())
}

// newMihomoPatchCmd 创建 mihomo patch 命令
func newMihomoPatchCmd() *cobra.Command {
	var configFile string

	cmd := &cobra.Command{
		Use:   "patch <key> <value>",
		Short: "热更新 Mihomo 配置",
		Long: `通过 API 热更新 Mihomo 运行时配置，无需重启服务。
支持的热更新配置项：mode, allow-lan, log-level, ipv6, sniffing, tcp-concurrent 等。`,
		Example: `  mihomo-cli mihomo patch mode rule
  mihomo-cli mihomo patch allow-lan true
  mihomo-cli mihomo patch log-level debug
  mihomo-cli mihomo patch --file config-patch.yaml`,
		Args: cobra.MaximumNArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runMihomoPatch(cmd.Context(), args, configFile)
		},
	}

	cmd.Flags().StringVarP(&configFile, "file", "f", "", "从 YAML/JSON 文件读取配置更新")

	return cmd
}

// runMihomoPatch 执行配置热更新
func runMihomoPatch(ctx context.Context, args []string, configFile string) error {
	// 创建 API 客户端
	client := api.NewClientWithTimeout(
		viper.GetString("api.address"),
		viper.GetString("api.secret"),
		viper.GetInt("api.timeout"),
	)

	var patchData map[string]interface{}

	if configFile != "" {
		// 从文件读取配置
		data, err := os.ReadFile(configFile)
		if err != nil {
			return pkgerrors.ErrConfig("failed to read config file", err)
		}

		if err := yaml.Unmarshal(data, &patchData); err != nil {
			return pkgerrors.ErrConfig("failed to parse config file", err)
		}
	} else if len(args) == 2 {
		// 从命令行参数读取
		key := args[0]
		valueStr := args[1]

		// 检查配置键是否支持
		if !config.IsConfigKeySupported(key) {
			color.Yellow("unsupported config key: %s", key)
			color.Yellow("use --help to see supported config keys")
			return pkgerrors.ErrInvalidArg("unsupported config key: "+key, nil)
		}

		// 检查是否支持热更新
		if !config.IsHotUpdateSupported(key) {
			return pkgerrors.ErrInvalidArg("config key "+key+" does not support hot update, use 'mihomo edit' command instead", nil)
		}

		// 解析配置值
		value, err := config.ParseConfigValue(key, valueStr)
		if err != nil {
			return pkgerrors.ErrConfig("failed to parse config value", err)
		}

		patchData = map[string]interface{}{key: value}
	} else {
		return pkgerrors.ErrInvalidArg("please specify config key-value pair or use --file parameter", nil)
	}

	// 执行热更新
	if err := client.PatchConfig(ctx, patchData); err != nil {
		return errors.WrapAPIError("failed to hot update config", err)
	}

	color.Green("✓ 配置已热更新")
	for k, v := range patchData {
		fmt.Printf("  %s = %v\n", k, v)
	}

	return nil
}

// newMihomoReloadCmd 创建 mihomo reload 命令
func newMihomoReloadCmd() *cobra.Command {
	var configPath string
	var force bool

	cmd := &cobra.Command{
		Use:   "reload",
		Short: "重载 Mihomo 配置文件",
		Long: `重新加载完整的 Mihomo 配置文件。
如果不指定路径，则重载当前配置文件。`,
		Example: `  mihomo-cli mihomo reload
  mihomo-cli mihomo reload --path /path/to/config.yaml
  mihomo-cli mihomo reload --force`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runMihomoReload(cmd.Context(), configPath, force)
		},
	}

	cmd.Flags().StringVarP(&configPath, "path", "p", "", "配置文件路径")
	cmd.Flags().BoolVarP(&force, "force", "f", false, "强制重载，忽略部分错误")

	return cmd
}

// runMihomoReload 执行配置重载
func runMihomoReload(ctx context.Context, configPath string, force bool) error {
	// 验证路径
	if configPath != "" {
		// 检查是否为绝对路径
		if !filepath.IsAbs(configPath) {
			return pkgerrors.ErrInvalidArg("config file path must be absolute: "+configPath, nil)
		}

		// 检查文件是否存在
		if _, err := os.Stat(configPath); os.IsNotExist(err) {
			return pkgerrors.ErrConfig("config file does not exist: "+configPath, nil)
		}
	}

	// 创建 API 客户端
	client := api.NewClientWithTimeout(
		viper.GetString("api.address"),
		viper.GetString("api.secret"),
		viper.GetInt("api.timeout"),
	)

	// 执行重载
	if err := client.ReloadConfig(ctx, configPath, force); err != nil {
		return errors.WrapAPIError("failed to reload config", err)
	}

	color.Green("✓ 配置已重载")
	if configPath != "" {
		fmt.Printf("  配置文件: %s\n", configPath)
	}
	if force {
		fmt.Println("  模式: 强制重载")
	}

	return nil
}

// newMihomoEditCmd 创建 mihomo edit 命令
func newMihomoEditCmd() *cobra.Command {
	var noReload bool
	var mihomoConfigPath string

	cmd := &cobra.Command{
		Use:   "edit <key> <value>",
		Short: "编辑 Mihomo 配置文件",
		Long: `编辑 Mihomo 配置文件并自动重载。
修改配置文件后会自动调用 reload 命令使配置生效。`,
		Example: `  mihomo-cli mihomo edit mode rule
  mihomo-cli mihomo edit allow-lan true
  mihomo-cli mihomo edit tun.enable true --no-reload`,
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runMihomoEdit(cmd.Context(), args[0], args[1], mihomoConfigPath, noReload)
		},
	}

	cmd.Flags().BoolVar(&noReload, "no-reload", false, "仅修改文件，不触发重载")
	cmd.Flags().StringVarP(&mihomoConfigPath, "mihomo-config", "m", "", "Mihomo 配置文件路径")

	return cmd
}

// runMihomoEdit 执行配置文件编辑
func runMihomoEdit(ctx context.Context, key, valueStr, mihomoConfigPath string, noReload bool) error {
	// 确定配置文件路径
	configPath, err := config.FindConfigPath(mihomoConfigPath)
	if err != nil {
		return err
	}

	// 检查配置键是否支持
	if !config.IsConfigKeySupported(key) {
		color.Yellow("警告: 配置键 %s 不在已知配置键列表中", key)
	}

	// 解析配置值
	value, err := config.ParseConfigValue(key, valueStr)
	if err != nil {
		return pkgerrors.ErrConfig("failed to parse config value", err)
	}

	// 创建编辑器
	editor := config.NewEditor(configPath)

	// 设置备份目录为统一的备份目录
	backupDir, err := config.GetBackupDir()
	if err != nil {
		return pkgerrors.ErrConfig("failed to get backup directory", err)
	}
	editor.SetBackupDir(backupDir)

	// 生成备份备注：记录修改的键值对
	note := fmt.Sprintf("edit-%s", key)

	// 编辑配置文件（带备注备份）
	backupPath, err := editor.EditWithNote(key, value, false, note)
	if err != nil {
		return pkgerrors.ErrConfig("failed to edit config file", err)
	}

	color.Green("✓ 配置文件已更新")
	fmt.Printf("  配置文件: %s\n", configPath)
	fmt.Printf("  %s = %v\n", key, value)
	if backupPath != "" {
		fmt.Printf("  备份文件: %s\n", backupPath)
	}

	// 如果需要重载
	if !noReload {
		// 创建 API 客户端
		client := api.NewClientWithTimeout(
			viper.GetString("api.address"),
			viper.GetString("api.secret"),
			viper.GetInt("api.timeout"),
		)

		// 重载配置
		if err := client.ReloadConfig(ctx, configPath, false); err != nil {
			color.Yellow("警告: 重载配置失败: %v", err)
			color.Yellow("配置文件已修改，但未生效，请手动重启服务")
			return nil
		}

		color.Green("✓ 配置已重载生效")
	}

	return nil
}
