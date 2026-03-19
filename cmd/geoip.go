package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/kkkqkx123/mihomo-cli/internal/api"
	"github.com/kkkqkx123/mihomo-cli/internal/errors"
	"github.com/kkkqkx123/mihomo-cli/internal/output"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// GeoInfo GeoIP 数据库信息
type GeoInfo struct {
	Exists    bool      `json:"exists"`
	FilePath  string    `json:"file_path"`
	FileSize  int64     `json:"file_size"`
	ModTime   time.Time `json:"mod_time"`
	FileName  string    `json:"file_name"`
	Directory string    `json:"directory"`
}

// NewGeoIPCmd 创建 GeoIP 管理命令
func NewGeoIPCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "geoip",
		Short: "管理 GeoIP 数据库",
		Long:  `管理 GeoIP 地理位置数据库，包括下载、更新和状态查询。`,
	}

	cmd.AddCommand(newGeoIPUpdateCmd())
	cmd.AddCommand(newGeoIPStatusCmd())

	return cmd
}

// newGeoIPUpdateCmd 创建更新 GeoIP 命令
func newGeoIPUpdateCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "update",
		Short: "更新 GeoIP 数据库",
		Long:  `从配置的数据源更新 GeoIP 地理位置数据库文件。`,
		Example: `  mihomo-cli geoip update
  mihomo-cli geoip update -o json`,
		RunE: runGeoIPUpdate,
	}
}

// runGeoIPUpdate 执行更新 GeoIP 命令
func runGeoIPUpdate(cmd *cobra.Command, args []string) error {
	// 创建 API 客户端
	client := api.NewClientWithTimeout(
		viper.GetString("api.address"),
		viper.GetString("api.secret"),
		viper.GetInt("api.timeout"),
	)

	// 更新 GeoIP 数据库
	err := client.UpdateGeo(cmd.Context())
	if err != nil {
		return errors.WrapAPIError("更新 GeoIP 数据库失败", err)
	}

	// 显示成功信息
	if outputFmt == "json" {
		output.Success("更新成功", map[string]interface{}{
			"message": "GeoIP 数据库更新成功",
			"action":  "update",
		})
	} else {
		output.Success("✓ GeoIP 数据库更新成功")
	}

	return nil
}

// newGeoIPStatusCmd 创建查询 GeoIP 状态命令
func newGeoIPStatusCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "查询 GeoIP 数据库状态",
		Long:  `检查 GeoIP 数据库文件的存在性、大小和最后修改时间。`,
		Example: `  mihomo-cli geoip status
  mihomo-cli geoip status -o json`,
		RunE: runGeoIPStatus,
	}
}

// runGeoIPStatus 执行查询 GeoIP 状态命令
func runGeoIPStatus(cmd *cobra.Command, args []string) error {
	// 获取 GeoIP 信息
	info, err := getGeoIPInfo()
	if err != nil {
		return err
	}

	// 根据输出格式显示结果
	if outputFmt == "json" {
		output.Success("查询成功", info)
		return nil
	}

	// 表格输出
	if info.Exists {
		fmt.Fprintf(output.GetGlobalStdout(), "GeoIP 数据库状态: ✓ 已安装\n\n")
		fmt.Fprintf(output.GetGlobalStdout(), "文件路径: %s\n", info.FilePath)
		fmt.Fprintf(output.GetGlobalStdout(), "文件名: %s\n", info.FileName)
		fmt.Fprintf(output.GetGlobalStdout(), "文件大小: %.2f MB\n", float64(info.FileSize)/1024/1024)
		fmt.Fprintf(output.GetGlobalStdout(), "最后更新: %s\n", info.ModTime.Format("2006-01-02 15:04:05"))
		fmt.Fprintf(output.GetGlobalStdout(), "存储目录: %s\n", info.Directory)
	} else {
		fmt.Fprintf(output.GetGlobalStdout(), "GeoIP 数据库状态: ✗ 未安装\n\n")
		fmt.Fprintf(output.GetGlobalStdout(), "预期存储目录: %s\n", info.Directory)
		fmt.Fprintf(output.GetGlobalStdout(), "\n")
		fmt.Fprintf(output.GetGlobalStdout(), "支持的文件名（按优先级）:\n")
		fmt.Fprintf(output.GetGlobalStdout(), "  - Country.mmdb\n")
		fmt.Fprintf(output.GetGlobalStdout(), "  - geoip.db\n")
		fmt.Fprintf(output.GetGlobalStdout(), "  - geoip.metadb\n")
		fmt.Fprintf(output.GetGlobalStdout(), "  - GeoIP.dat\n")
		fmt.Fprintf(output.GetGlobalStdout(), "\n")
		fmt.Fprintf(output.GetGlobalStdout(), "提示: 使用 'mihomo-cli geoip update' 命令下载 GeoIP 数据库\n")
	}

	return nil
}

// getGeoIPInfo 获取 GeoIP 数据库信息
func getGeoIPInfo() (*GeoInfo, error) {
	// 获取配置目录
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("无法获取用户目录: %w", err)
	}

	// mihomo 配置目录
	configDir := filepath.Join(homeDir, ".config", "mihomo")

	// 支持的文件名列表（按优先级排序）
	fileNames := []string{
		"Country.mmdb",
		"geoip.db",
		"geoip.metadb",
		"GeoIP.dat",
	}

	info := &GeoInfo{
		Directory: configDir,
	}

	// 查找存在的文件
	for _, fileName := range fileNames {
		filePath := filepath.Join(configDir, fileName)
		if fileInfo, err := os.Stat(filePath); err == nil {
			info.Exists = true
			info.FilePath = filePath
			info.FileName = fileName
			info.FileSize = fileInfo.Size()
			info.ModTime = fileInfo.ModTime()
			break
		}
	}

	return info, nil
}
