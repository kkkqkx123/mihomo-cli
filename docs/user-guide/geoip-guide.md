# mihomo-go

Mihomo CLI 是一个非交互式的 Mihomo 代理核心管理工具，通过命令行界面提供对 Mihomo RESTful API 的完整管理能力。

## 功能特性

### GeoIP 管理（新增）

支持 GeoIP 地理位置数据库的下载和状态查询：

```bash
# 更新 GeoIP 数据库
mihomo-cli geoip update

# 检查 GeoIP 数据库状态
mihomo-cli geoip status

# JSON 格式输出
mihomo-cli geoip status -o json
```

详细文档请参见：[GeoIP 命令使用指南](docs/geoip-command.md)

## 构建

```bash
go build -o mihomo-cli.exe main.go
```

## 使用

```bash
# 查看帮助
mihomo-cli --help

# 初始化配置
mihomo-cli config init

# 查询当前模式
mihomo-cli mode get

# 列出代理节点
mihomo-cli proxy list

# 切换代理节点
mihomo-cli proxy switch <proxy-name>
```

## 完整文档

请查看 [docs](docs/) 目录下的详细文档。

