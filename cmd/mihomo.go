package cmd

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"gopkg.in/yaml.v3"

	"github.com/kkkqkx123/mihomo-cli/internal/api"
	"github.com/kkkqkx123/mihomo-cli/internal/config"
	"github.com/kkkqkx123/mihomo-cli/internal/errors"
	"github.com/kkkqkx123/mihomo-cli/internal/output"
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
	mihomoCmd.AddCommand(newMihomoRestartCmd())
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
			return pkgerrors.ErrConfig("读取配置文件失败", err)
		}

		if err := yaml.Unmarshal(data, &patchData); err != nil {
			return pkgerrors.ErrConfig("解析配置文件失败", err)
		}
	} else if len(args) == 2 {
		// 从命令行参数读取
		key := args[0]
		valueStr := args[1]

		// 检查配置键是否支持
		if !config.IsConfigKeySupported(key) {
			return pkgerrors.ErrInvalidArg(fmt.Sprintf("不支持的配置键: %s，使用 --help 查看支持的配置键", key), nil)
		}

		// 检查是否支持热更新
		if !config.IsHotUpdateSupported(key) {
			return pkgerrors.ErrInvalidArg(fmt.Sprintf("配置键 %s 不支持热更新，请使用 mihomo edit 命令", key), nil)
		}

		// 解析配置值
		value, err := config.ParseConfigValue(key, valueStr)
		if err != nil {
			return pkgerrors.ErrConfig("解析配置值失败", err)
		}

		patchData = map[string]interface{}{key: value}
	} else {
		return pkgerrors.ErrInvalidArg("请指定配置键值对或使用 --file 参数", nil)
	}

	// 执行热更新
	if err := client.PatchConfig(ctx, patchData); err != nil {
		return errors.WrapAPIError("热更新配置失败", err)
	}

	output.Success("配置已热更新")
	for k, v := range patchData {
		output.Printf("  %s = %v\n", k, v)
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
	var actualConfigPath string
	if configPath != "" {
		// 检查是否为绝对路径
		if !filepath.IsAbs(configPath) {
			return pkgerrors.ErrInvalidArg("配置文件路径必须是绝对路径: "+configPath, nil)
		}

		// 检查文件是否存在
		if _, err := os.Stat(configPath); os.IsNotExist(err) {
			return pkgerrors.ErrConfig("配置文件不存在: "+configPath, nil)
		}

		actualConfigPath = configPath

		// 验证配置文件语法和有效性
		validator := config.NewConfigValidator(configPath)
		if err := validator.ValidateConfigSyntax(); err != nil {
			output.Warning("配置校验失败: %v", err)
			if !force {
				return pkgerrors.ErrConfig("配置校验失败，使用 --force 强制执行", err)
			}
			output.Warning("强制重载，忽略校验错误...")
		}

		// 检查是否有高风险配置
		if err := validator.ValidateAndWarn(); err != nil {
			output.Warning("高风险配置警告: %v", err)
		}
	} else {
		// 如果没有指定配置文件路径，尝试从配置中获取
		tomlConfigPath := config.FindTomlConfigPath("")
		tomlCfg, err := config.LoadTomlConfig(tomlConfigPath)
		if err != nil {
			return pkgerrors.ErrConfig("加载配置失败", err)
		}
		actualConfigPath = tomlCfg.Mihomo.ConfigFile
	}

	// 在重载前备份当前配置
	if actualConfigPath != "" {
		output.Println("重载前备份当前配置...")
		backupHandler := config.NewBackupHandler(actualConfigPath)
		backupInfo, err := backupHandler.CreateBackup("", "pre-reload")
		if err != nil {
			output.Warning("备份失败: %v", err)
			output.Warning("继续重载，不创建备份...")
		} else {
			output.Success("备份已创建: %s", backupInfo.Path)
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
		return errors.WrapAPIError("重载配置失败", err)
	}

	output.Success("配置已重载")
	if configPath != "" {
		output.Printf("  配置文件: %s\n", configPath)
	}
	if force {
		output.Println("  模式: 强制重载")
	}

	return nil
}

// newMihomoRestartCmd 创建 mihomo restart 命令
func newMihomoRestartCmd() *cobra.Command {
	var wait bool
	var waitTimeout int

	cmd := &cobra.Command{
		Use:   "restart",
		Short: "通过 API 重启 Mihomo 内核",
		Long: `通过 Mihomo RESTful API 触发核心进程重启。
注意：此操作仅重启 Mihomo 内核进程，不会重启 CLI 本身。`,
		Example: `  mihomo-cli mihomo restart
  mihomo-cli mihomo restart --no-wait
  mihomo-cli mihomo restart --wait-timeout 60`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runMihomoRestart(cmd.Context(), wait, waitTimeout)
		},
	}

	cmd.Flags().BoolVar(&wait, "wait", true, "重启后等待并验证核心重新可用")
	cmd.Flags().IntVar(&waitTimeout, "wait-timeout", 30, "等待核心重新可用的最大秒数")

	return cmd
}

// runMihomoRestart 执行重启命令
func runMihomoRestart(ctx context.Context, wait bool, waitTimeout int) error {
	// 创建 API 客户端
	client := api.NewClientWithTimeout(
		viper.GetString("api.address"),
		viper.GetString("api.secret"),
		viper.GetInt("api.timeout"),
	)

	// 记录重启前版本信息，用于后续对比
	var originalVersion string
	if info, err := client.GetVersion(ctx); err == nil {
		originalVersion = info.Version
	}

	// 执行重启
	if err := client.Restart(ctx); err != nil {
		return errors.WrapAPIError("failed to restart Mihomo", err)
	}

	if !wait {
		output.Success("Mihomo 重启指令已发送")
		return nil
	}

	output.Info("Mihomo 重启指令已发送，等待核心恢复...")

	// 轮询等待核心重新可用
	pollCtx, cancel := context.WithTimeout(ctx, time.Duration(waitTimeout)*time.Second)
	defer cancel()

	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-pollCtx.Done():
			return errors.WrapAPIError(
				"Mihomo 重启后未能在规定时间内恢复",
				api.NewAPIError(api.ErrTimeout, "等待超时", pollCtx.Err()),
			)
		case <-ticker.C:
			info, err := client.GetVersion(pollCtx)
			if err != nil {
				// 核心可能仍在启动中，继续等待
				output.Info("  核心尚未就绪，继续等待...")
				continue
			}

			output.Success("Mihomo 已重启并恢复可用")
			if info.Version != "" {
				output.Printf("  版本: %s\n", info.Version)
			}
			if originalVersion != "" && info.Version != originalVersion {
				output.Printf("  版本变化: %s -> %s\n", originalVersion, info.Version)
			}
			return nil
		}
	}
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
		output.Warning("配置键 %s 不在已知配置键列表中", key)
	}

	// 解析配置值
	value, err := config.ParseConfigValue(key, valueStr)
	if err != nil {
		return pkgerrors.ErrConfig("failed to parse config value", err)
	}

	// 创建编辑器
	editor := config.NewEditor(configPath)

	// 设置备份目录为统一的备份目录
	pathResolver, err := config.NewPathResolver()
	if err != nil {
		return pkgerrors.ErrConfig("failed to create path resolver", err)
	}
	backupDir := pathResolver.GetBackupDir()
	editor.SetBackupDir(backupDir)

	// 生成备份备注：记录修改的键值对
	note := fmt.Sprintf("edit-%s", key)

	// 编辑配置文件（带备注备份）
	backupPath, err := editor.EditWithNote(key, value, false, note)
	if err != nil {
		return pkgerrors.ErrConfig("failed to edit config file", err)
	}

	output.Success("配置文件已更新")
	output.Printf("  配置文件: %s\n", configPath)
	output.Printf("  %s = %v\n", key, value)
	if backupPath != "" {
		output.Printf("  备份文件: %s\n", backupPath)
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
			output.Warning("配置文件已修改，但未生效，请手动重启服务")
			return pkgerrors.ErrAPI(fmt.Sprintf("重载配置失败: %v", err), err)
		}

		output.Success("配置已重载生效")
	}

	return nil
}
