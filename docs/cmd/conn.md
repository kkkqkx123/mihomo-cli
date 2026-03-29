# 连接管理命令 (conn)

连接管理命令用于管理活跃连接，包括列出、关闭指定连接和关闭所有连接。

## 命令列表

### conn list - 列出活跃连接

列出当前所有活跃的连接信息。

**语法：**
```bash
mihomo-cli conn list
```

**示例：**
```bash
mihomo-cli conn list
mihomo-cli conn list -o json
```

### conn close - 关闭指定连接

关闭指定 ID 的连接。

**语法：**
```bash
mihomo-cli conn close <id>
```

**参数：**
- `id` - 连接 ID

**示例：**
```bash
mihomo-cli conn close abc123
```

### conn close-all - 关闭所有连接

关闭所有活跃的连接。

**语法：**
```bash
mihomo-cli conn close-all
```

**示例：**
```bash
mihomo-cli conn close-all
```

## 连接信息说明

连接列表中包含以下信息：
- 连接 ID - 唯一标识符
- 元数据 - 连接的元数据信息
- 上传/下载流量 - 该连接的流量统计
- 连接时间 - 连接建立的时间
- 使用的代理 - 该连接使用的代理节点

## 注意事项

1. 关闭连接会立即终止该连接的传输
2. 关闭所有连接会断开所有正在进行的网络请求
3. 建议在遇到连接问题时使用此功能排查
4. 关闭连接操作是瞬时的，不需要重启 Mihomo
