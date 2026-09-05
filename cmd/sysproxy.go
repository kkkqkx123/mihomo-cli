package cmd

import (
	"fmt"
	"runtime"

	"github.com/spf13/cobra"

	"github.com/kkkqkx123/mihomo-cli/internal/output"
	"github.com/kkkqkx123/mihomo-cli/internal/sysproxy"
	"github.com/kkkqkx123/mihomo-cli/internal/util"
	pkgerrors "github.com/kkkqkx123/mihomo-cli/pkg/errors"
)

var (
	proxyServer string
	bypassList  string
)

var sysproxyCmd = &cobra.Command{
	Use:   "sysproxy",
	Short: "系统代理管理",
	Long:  "管理系统代理设置。",
	RunE: func(cmd *cobra.Command, args []string) error {
		sp := sysproxy.NewSysProxy()

		if !sp.IsSupported() {
			return pkgerrors.ErrService(fmt.Sprintf("当前平台 %s 不支持系统代理管理", runtime.GOOS), sysproxy.ErrPlatformNotSupported)
		}

		return cmd.Help()
	},
}

var sysproxyGetCmd = &cobra.Command{
	Use:   "get",
	Short: "查询系统代理状态",
	Long:  "查询当前系统代理的状态。",
	RunE:  runSysproxyGet,
}

var sysproxySetCmd = &cobra.Command{
	Use:   "set <on|off>",
	Short: "设置系统代理",
	Long:  "开启或关闭系统代理。",
	Args:  cobra.ExactArgs(1),
	RunE:  runSysproxySet,
}

func init() {
	sysproxyCmd.AddCommand(sysproxyGetCmd)
	sysproxyCmd.AddCommand(sysproxySetCmd)

	// 添加标志
	sysproxySetCmd.Flags().StringVar(&proxyServer, "server", "127.0.0.1:7890", "代理服务器地址")
	sysproxySetCmd.Flags().StringVar(&bypassList, "bypass", "localhost;127.*;10.*;172.16.*;172.31.*;192.168.*", "绕过代理的地址列表")
}

// NewSysproxyCmd 创建 sysproxy 命令
func NewSysproxyCmd() *cobra.Command {
	return sysproxyCmd
}

func runSysproxyGet(cmd *cobra.Command, args []string) error {
	sp := sysproxy.NewSysProxy()

	if !sp.IsSupported() {
		return pkgerrors.ErrService("当前平台不支持系统代理管理", sysproxy.ErrPlatformNotSupported)
	}

	settings, err := sp.GetStatus()
	if err != nil {
		return pkgerrors.ErrService("获取系统代理状态失败", err)
	}

	output.Println("系统代理状态:")
	if settings.Enabled {
		output.Success("  状态: 已启用")
		output.PrintKeyValue("代理服务器", settings.Server)
		if settings.BypassList != "" {
			output.PrintKeyValue("绕过列表", settings.BypassList)
		}
	} else {
		output.Info("  状态: 已禁用")
	}

	return nil
}

func runSysproxySet(cmd *cobra.Command, args []string) error {
	sp := sysproxy.NewSysProxy()

	if !sp.IsSupported() {
		return pkgerrors.ErrService("当前平台不支持系统代理管理", sysproxy.ErrPlatformNotSupported)
	}

	// 检查管理员权限
	if !util.IsAdmin() {
		return pkgerrors.ErrService("该操作需要管理员权限，请以管理员身份运行", nil)
	}

	action := args[0]

	switch action {
	case "on":
		err := sp.Enable(proxyServer, bypassList)
		if err != nil {
			return pkgerrors.ErrService("启用系统代理失败", err)
		}
		output.Success("系统代理已启用")
		output.PrintKeyValue("代理服务器", proxyServer)
		if bypassList != "" {
			output.PrintKeyValue("绕过列表", bypassList)
		}

	case "off":
		err := sp.Disable()
		if err != nil {
			return pkgerrors.ErrService("禁用系统代理失败", err)
		}
		output.Success("系统代理已禁用")

	default:
		return pkgerrors.ErrInvalidArg("invalid parameter: "+action+", please use 'on' or 'off'", nil)
	}

	return nil
}
