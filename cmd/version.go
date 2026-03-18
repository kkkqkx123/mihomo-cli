package cmd

import (
	"fmt"
	"runtime"

	"github.com/spf13/cobra"
)

var (
	// 构建时注入的版本信息
	version = "dev"
	commit  = "unknown"
	date    = "unknown"
)

// NewVersionCmd 创建版本命令
func NewVersionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "显示版本信息",
		Long:  `显示 mihomo-cli 的版本和构建信息。`,
		Run: func(cmd *cobra.Command, args []string) {
			printVersion()
		},
	}
}

// printVersion 打印版本信息
func printVersion() {
	fmt.Printf("mihomo-cli version %s\n", version)
	fmt.Println("Build Information:")
	fmt.Printf("  Git Commit:  %s\n", commit)
	fmt.Printf("  Build Date:  %s\n", date)
	fmt.Printf("  Go Version:  %s\n", runtime.Version())
	fmt.Printf("  GOOS:        %s\n", runtime.GOOS)
	fmt.Printf("  GOARCH:      %s\n", runtime.GOARCH)
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
