# 代理管理命令 (proxy)

代理管理命令用于管理 Mihomo 的代理节点，包括列出、切换、测试和自动选择节点。

## 命令列表

### proxy list - 列出代理节点

列出所有代理组的节点列表。支持多种过滤选项。

**语法：**
```bash
mihomo-cli proxy list [group]
```

**参数：**
- `group` (可选) - 指定代理组名称，只显示该代理组的节点

**选项：**
- `--type` - 按类型过滤（如 Vmess, Selector, URLTest 等）
- `--status` - 按状态过滤（alive/dead）
- `--exclude` - 排除名称匹配正则表达式的节点
- `--exclude-logical` - 排除逻辑节点（DIRECT, REJECT 等）
- `--groups-only` - 只显示代理组
- `--nodes-only` - 只显示节点（排除代理组）

**示例：**
```bash
# 列出所有代理节点
mihomo-cli proxy list

# 列出指定代理组的节点
mihomo-cli proxy list Proxy

# 按类型过滤
mihomo-cli proxy list --type Vmess

# 排除逻辑节点
mihomo-cli proxy list --exclude-logical

# 只显示存活的节点
mihomo-cli proxy list --status alive

# JSON 格式输出
mihomo-cli proxy list -o json
```

### proxy switch - 切换代理节点

切换指定代理组的选中节点。

**语法：**
```bash
mihomo-cli proxy switch <group> <node>
```

**参数：**
- `group` - 代理组名称
- `node` - 节点名称

**示例：**
```bash
mihomo-cli proxy switch Proxy Node1
```

### proxy test - 测试节点延迟

测试指定代理组或节点的延迟。

**语法：**
```bash
mihomo-cli proxy test <group> [node]
```

**参数：**
- `group` - 代理组名称
- `node` (可选) - 节点名称。如果只指定代理组，测试该组内所有节点

**选项：**
- `--url` - 测试 URL（可选，默认使用配置中的 URL）
- `--timeout` - 超时时间（毫秒，默认 5000）
- `--concurrent` - 并发测试数（默认 10）
- `--progress` - 显示进度条

**示例：**
```bash
# 测试代理组中所有节点
mihomo-cli proxy test Proxy

# 测试单个节点
mihomo-cli proxy test Proxy Node1

# 自定义测试参数
mihomo-cli proxy test Proxy --url https://www.google.com/generate_204 --timeout 5000 --progress
```

### proxy auto - 自动选择最快节点

测试代理组中所有节点的延迟，并自动切换到延迟最低的节点。

**语法：**
```bash
mihomo-cli proxy auto <group>
```

**参数：**
- `group` - 代理组名称

**选项：**
- `--url` - 测试 URL（可选，默认使用配置中的 URL）
- `--timeout` - 超时时间（毫秒，默认 5000）
- `--concurrent` - 并发测试数（默认 10）
- `--progress` - 显示进度条

**示例：**
```bash
mihomo-cli proxy auto Proxy
mihomo-cli proxy auto Proxy --url https://www.google.com/generate_204 --timeout 5000 --progress
```

### proxy unfix - 取消固定代理

取消代理组中固定的代理，恢复自动选择模式。

**语法：**
```bash
mihomo-cli proxy unfix <group>
```

**参数：**
- `group` - 代理组名称

**示例：**
```bash
mihomo-cli proxy unfix Proxy
```

### proxy current - 获取当前使用的节点

获取指定代理组当前使用的节点信息。

**语法：**
```bash
mihomo-cli proxy current <group>
```

**参数：**
- `group` - 代理组名称

**示例：**
```bash
mihomo-cli proxy current Proxy
```

## 配置文件

代理测试的相关配置可以在配置文件中设置：

```toml
[proxy]
test_url = "https://www.google.com/generate_204"  # 测试 URL
timeout = 10000                                    # 超时时间（毫秒）
concurrent = 10                                    # 并发测试数
```

## 注意事项

1. 测试延迟时，建议使用稳定的测试 URL
2. 并发测试数不宜过大，以免影响测试准确性
3. 自动选择功能会自动切换到延迟最低的节点
4. 取消固定代理后，代理组会恢复到自动选择模式
