# 规则管理命令 (rule)

规则管理命令用于管理 Mihomo 的路由规则，包括列出、禁用和启用规则。

## 命令列表

### rule list - 列出所有规则

列出所有规则及其统计信息。

**语法：**
```bash
mihomo-cli rule list
```

**选项：**
- `--type` - 过滤规则类型（如 DOMAIN, IP-CIDR 等）

**示例：**
```bash
# 列出所有规则
mihomo-cli rule list

# 按类型过滤
mihomo-cli rule list --type DOMAIN

# JSON 格式输出
mihomo-cli rule list -o json
```

### rule provider - 列出规则提供者

列出所有规则提供者及其信息。

**语法：**
```bash
mihomo-cli rule provider
```

**示例：**
```bash
mihomo-cli rule provider
mihomo-cli rule provider -o json
```

### rule disable - 禁用指定规则

禁用一个或多个规则。使用规则索引指定要禁用的规则。

**语法：**
```bash
mihomo-cli rule disable <index> [index...]
```

**参数：**
- `index` - 规则索引，支持以下格式：
  - 单个索引：`0`
  - 多个索引：`0 1 2`
  - 范围索引：`0-5`

**示例：**
```bash
# 禁用单个规则
mihomo-cli rule disable 0

# 禁用多个规则
mihomo-cli rule disable 0 1 2

# 禁用范围规则
mihomo-cli rule disable 0-5
```

### rule enable - 启用指定规则

启用一个或多个规则。使用规则索引指定要启用的规则。

**语法：**
```bash
mihomo-cli rule enable <index> [index...]
```

**参数：**
- `index` - 规则索引，支持以下格式：
  - 单个索引：`0`
  - 多个索引：`0 1 2`
  - 范围索引：`0-5`

**示例：**
```bash
# 启用单个规则
mihomo-cli rule enable 0

# 启用多个规则
mihomo-cli rule enable 0 1 2

# 启用范围规则
mihomo-cli rule enable 0-5
```

## 规则索引说明

规则索引从 0 开始，按规则在配置文件中的顺序排列。可以使用 `rule list` 命令查看所有规则及其索引。

## 注意事项

1. 禁用规则会使其在路由匹配中失效
2. 启用规则会使其重新参与路由匹配
3. 规则索引是基于当前规则列表的，添加或删除规则后索引可能会变化
4. 范围索引 `0-5` 表示从索引 0 到索引 5（包含）的所有规则
