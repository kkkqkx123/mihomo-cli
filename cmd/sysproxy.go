package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

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
	Long:  "管理 Windows 系统代理设置。",
}

var sysproxyGetCmd = &cobra.Command{
	Use:   "get",
	Short: "查询系统代理状态",
	Long:  "查询当前 Windows 系统代理的状态。",
	RunE:  runSysproxyGet,
}

var sysproxySetCmd = &cobra.Command{
	Use:   "set <on|off>",
	Short: "设置系统代理",
	Long:  "开启或关闭 Windows 系统代理。",
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
	settings, err := sysproxy.GetProxyStatus()
	if err != nil {
		return err
	}

	fmt.Println("系统代理状态:")
	if settings.Enabled {
		fmt.Println("  状态: 已启用")
		fmt.Printf("  代理服务器: %s\n", settings.Server)
		if settings.BypassList != "" {
			fmt.Printf("  绕过列表: %s\n", settings.BypassList)
		}
	} else {
		fmt.Println("  状态: 已禁用")
	}

	return nil
}

func runSysproxySet(cmd *cobra.Command, args []string) error {
	// 检查管理员权限
	if !util.IsAdmin() {
		return pkgerrors.ErrService("this operation requires administrator privileges, please run as administrator", nil)
	}

	action := args[0]

	switch action {
	case "on":
		err := sysproxy.EnableProxy(proxyServer, bypassList)
		if err != nil {
			return err
		}
		fmt.Println("系统代理已启用")
		fmt.Printf("代理服务器: %s\n", proxyServer)
		if bypassList != "" {
			fmt.Printf("绕过列表: %s\n", bypassList)
		}

	case "off":
		err := sysproxy.DisableProxy()
		if err != nil {
			return err
		}
		fmt.Println("系统代理已禁用")

	default:
		return pkgerrors.ErrInvalidArg("invalid parameter: "+action+", please use 'on' or 'off'", nil)
	}

	return nil
}
