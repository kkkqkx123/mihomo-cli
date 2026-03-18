package cmd

import (
	"fmt"
	"runtime"

	"github.com/spf13/cobra"

	"github.com/kkkqkx123/mihomo-cli/internal/service"
)

var asyncMode bool

var serviceCmd = &cobra.Command{
	Use:   "service",
	Short: "服务管理",
	Long:  "管理 Mihomo 系统服务的安装、卸载、启动、停止和状态查询。",
	RunE: func(cmd *cobra.Command, args []string) error {
		factory := service.NewServiceFactory()
		sm, err := factory.CreateServiceManager()
		if err != nil {
			return err
		}

		if !sm.IsSupported() {
			return fmt.Errorf("service command not supported on %s", runtime.GOOS)
		}

		return cmd.Help()
	},
}

var serviceStartCmd = &cobra.Command{
	Use:   "start",
	Short: "启动服务",
	Long:  "启动 Mihomo 系统服务。",
	RunE:  runServiceStart,
}

var serviceStopCmd = &cobra.Command{
	Use:   "stop",
	Short: "停止服务",
	Long:  "停止 Mihomo 系统服务。",
	RunE:  runServiceStop,
}

var serviceInstallCmd = &cobra.Command{
	Use:   "install",
	Short: "安装服务",
	Long:  "将 Mihomo 安装为系统服务。",
	RunE:  runServiceInstall,
}

var serviceUninstallCmd = &cobra.Command{
	Use:   "uninstall",
	Short: "卸载服务",
	Long:  "卸载 Mihomo 系统服务。",
	RunE:  runServiceUninstall,
}

var serviceStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "查询服务状态",
	Long:  "查询 Mihomo 系统服务的运行状态。",
	RunE:  runServiceStatus,
}

func init() {
	serviceCmd.AddCommand(serviceStartCmd)
	serviceCmd.AddCommand(serviceStopCmd)
	serviceCmd.AddCommand(serviceInstallCmd)
	serviceCmd.AddCommand(serviceUninstallCmd)
	serviceCmd.AddCommand(serviceStatusCmd)

	// 添加异步标志
	serviceStartCmd.Flags().BoolVarP(&asyncMode, "async", "a", false, "异步模式，立即返回不等待")
	serviceStopCmd.Flags().BoolVarP(&asyncMode, "async", "a", false, "异步模式，立即返回不等待")
}

// NewServiceCmd 创建 service 命令
func NewServiceCmd() *cobra.Command {
	return serviceCmd
}

// getServiceManager 获取服务管理器
func getServiceManager() (service.ServiceManager, error) {
	factory := service.NewServiceFactory()
	return factory.CreateServiceManager()
}

func runServiceStart(cmd *cobra.Command, args []string) error {
	sm, err := getServiceManager()
	if err != nil {
		return err
	}

	if !sm.IsSupported() {
		return service.ErrPlatformNotSupported
	}

	err = sm.Start(asyncMode)
	if err != nil {
		return err
	}

	if asyncMode {
		fmt.Printf("服务 %s 已启动（异步模式）\n", sm.GetServiceName())
		fmt.Println("使用 'mihomo-cli service status' 查询运行状态")
	} else {
		fmt.Printf("服务 %s 已成功启动\n", sm.GetServiceName())
	}
	return nil
}

func runServiceStop(cmd *cobra.Command, args []string) error {
	sm, err := getServiceManager()
	if err != nil {
		return err
	}

	if !sm.IsSupported() {
		return service.ErrPlatformNotSupported
	}

	err = sm.Stop(asyncMode)
	if err != nil {
		return err
	}

	if asyncMode {
		fmt.Printf("服务 %s 已停止（异步模式）\n", sm.GetServiceName())
		fmt.Println("使用 'mihomo-cli service status' 查询运行状态")
	} else {
		fmt.Printf("服务 %s 已成功停止\n", sm.GetServiceName())
	}
	return nil
}

func runServiceInstall(cmd *cobra.Command, args []string) error {
	sm, err := getServiceManager()
	if err != nil {
		return err
	}

	if !sm.IsSupported() {
		return service.ErrPlatformNotSupported
	}

	err = sm.Install()
	if err != nil {
		return err
	}

	fmt.Printf("服务 %s 已成功安装\n", sm.GetServiceName())
	fmt.Printf("显示名称: %s\n", sm.GetDisplayName())
	fmt.Printf("可执行文件: %s\n", sm.GetExePath())
	return nil
}

func runServiceUninstall(cmd *cobra.Command, args []string) error {
	sm, err := getServiceManager()
	if err != nil {
		return err
	}

	if !sm.IsSupported() {
		return service.ErrPlatformNotSupported
	}

	err = sm.Uninstall()
	if err != nil {
		return err
	}

	fmt.Printf("服务 %s 已成功卸载\n", sm.GetServiceName())
	return nil
}

func runServiceStatus(cmd *cobra.Command, args []string) error {
	sm, err := getServiceManager()
	if err != nil {
		return err
	}

	if !sm.IsSupported() {
		return service.ErrPlatformNotSupported
	}

	status, err := sm.Status()
	if err != nil {
		return err
	}

	fmt.Printf("服务名称: %s\n", sm.GetServiceName())
	fmt.Printf("显示名称: %s\n", sm.GetDisplayName())

	switch status {
	case service.StatusRunning:
		fmt.Println("状态: 运行中")
	case service.StatusStopped:
		fmt.Println("状态: 已停止")
	case service.StatusNotInstalled:
		fmt.Println("状态: 未安装")
	default:
		fmt.Println("状态: 未知")
	}

	return nil
}
