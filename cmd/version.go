package cmd

import (
	"runtime"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/kkkqkx123/mihomo-cli/internal/api"
	"github.com/kkkqkx123/mihomo-cli/internal/errors"
	"github.com/kkkqkx123/mihomo-cli/internal/output"
)

var (
	// 构建时注入的版本信息
	version = "dev"
	commit  = "unknown"
	date    = "unknown"
)

// NewVersionCmd 创建版本命令
func NewVersionCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "version",
		Short: "显示版本信息",
		Long:  `显示 mihomo-cli 的版本和构建信息。`,
		Run: func(cmd *cobra.Command, args []string) {
			printVersion()
		},
	}

	cmd.AddCommand(newVersionKernelCmd())

	return cmd
}

// newVersionKernelCmd 创建内核版本命令
func newVersionKernelCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "kernel",
		Short: "显示 Mihomo 内核版本",
		Long:  `显示正在运行的 Mihomo 内核版本信息。`,
		RunE:  runVersionKernel,
	}
}

// runVersionKernel 执行内核版本命令
func runVersionKernel(cmd *cobra.Command, args []string) error {
	// 创建 API 客户端
	client := api.NewClientWithTimeout(
		viper.GetString("api.address"),
		viper.GetString("api.secret"),
		viper.GetInt("api.timeout"),
	)

	// 获取内核版本信息
	versionInfo, err := client.GetVersion(cmd.Context())
	if err != nil {
		return errors.WrapAPIError("获取内核版本失败", err)
	}

	// 格式化输出
	output.Printf("Mihomo Kernel Version: %s\n", versionInfo.Version)
	if versionInfo.PreRelease {
		output.Printf("Premium: Yes\n")
	}
	if versionInfo.HomeDir != "" {
		output.Printf("Home Directory: %s\n", versionInfo.HomeDir)
	}
	if versionInfo.ConfigPath != "" {
		output.Printf("Config Path: %s\n", versionInfo.ConfigPath)
	}

	return nil
}

// printVersion 打印版本信息
func printVersion() {
	output.Printf("mihomo-cli version %s\n", version)
	output.Println("Build Information:")
	output.Printf("  Git Commit:  %s\n", commit)
	output.Printf("  Build Date:  %s\n", date)
	output.Printf("  Go Version:  %s\n", runtime.Version())
	output.Printf("  GOOS:        %s\n", runtime.GOOS)
	output.Printf("  GOARCH:      %s\n", runtime.GOARCH)
}

// GetVersion 获取版本号
func GetVersion() string {
	return version
}

// GetCommit 获取 Git 提交哈希
func GetCommit() string {
	return commit
}

// GetBuildDate 获取构建日期
func GetBuildDate() string {
	return date
}

// SetVersionInfo 设置版本信息（用于构建时注入）
func SetVersionInfo(v, c, d string) {
	version = v
	commit = c
	date = d
}
