# DNS 管理命令 (dns)

DNS 管理命令用于执行 DNS 查询和查看 DNS 配置。

## 命令列表

### dns query - 执行 DNS 查询

执行 DNS 查询，支持多种记录类型。

**语法：**
```bash
mihomo-cli dns query <domain>
```

**参数：**
- `domain` - 要查询的域名

**选项：**
- `--type` - DNS 记录类型，支持：
  - `A` - IPv4 地址记录（默认）
  - `AAAA` - IPv6 地址记录
  - `CNAME` - 别名记录
  - `MX` - 邮件交换记录
  - `TXT` - 文本记录
  - `NS` - 名称服务器记录
  - `SRV` - 服务记录

**示例：**
```bash
# 查询 A 记录
mihomo-cli dns query example.com

# 查询 AAAA 记录
mihomo-cli dns query example.com --type AAAA

# 查询 MX 记录
mihomo-cli dns query example.com --type MX

# 查询 TXT 记录
mihomo-cli dns query example.com --type TXT -o json
```

### dns config - 显示 DNS 配置

显示当前 DNS 配置信息。

**语法：**
```bash
mihomo-cli dns config
```

**示例：**
```bash
mihomo-cli dns config
mihomo-cli dns config -o json
```

## DNS 记录类型说明

| 类型 | 说明 |
|------|------|
| A | 将域名映射到 IPv4 地址 |
| AAAA | 将域名映射到 IPv6 地址 |
| CNAME | 将域名映射到另一个域名（别名） |
| MX | 指定邮件服务器 |
| TXT | 存储文本信息，常用于 SPF、DKIM 等 |
| NS | 指定域名的名称服务器 |
| SRV | 指定提供特定服务的服务器 |

## 注意事项

1. DNS 查询使用 Mihomo 配置的 DNS 服务器
2. 查询结果会显示解析的 IP 地址和 TTL（生存时间）
3. 如果查询失败，可能是 DNS 服务器配置问题或网络问题
4. DNS 配置信息包括服务器地址、端口、超时时间等
