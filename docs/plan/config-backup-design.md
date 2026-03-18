# 配置备份功能设计方案

## 一、现状分析

### 1.1 已实现的功能

当前项目在 `internal/config/editor.go` 中已实现基础的备份功能：

1. **编辑时自动备份** (`editor.go:47-69`)
   - `Editor.BackupConfig()` 方法在编辑配置文件时自动创建备份
   - 备份文件命名格式：`{原文件名}.backup.{时间戳}{扩展名}`
   - 例如：`config.yaml` → `config.backup.20260317-143025.yaml`
   - 备份位置：与原配置文件同目录

2. **使用场景** (`cmd/config.go:420-429`)
   - 仅在 `config edit` 命令执行时自动触发备份
   - 可通过 `--no-reload` 参数控制是否备份

### 1.2 未实现的功能

- ❌ 没有独立的 `config backup` 命令
- ❌ 没有备份列表查看功能
- ❌ 没有备份恢复功能
- ❌ 没有备份清理/删除功能
- ❌ 没有备份管理策略（如保留数量、过期清理）

---

## 二、设计方案

### 2.1 命令结构

新增 `config backup` 子命令组，提供完整的备份管理能力：

```
mihomo-cli config backup [子命令]
```

### 2.2 子命令设计

#### 2.2.1 创建备份命令

```bash
mihomo-cli config backup create [选项]
```

**功能：**
- 手动创建配置文件备份
- 支持指定配置文件路径
- 支持添加备份备注

**选项：**
| 选项 | 简写 | 说明 | 默认值 |
|------|------|------|--------|
| `--path` | `-p` | 指定要备份的配置文件路径 | 自动查找 |
| `--note` | `-n` | 添加备份备注 | 无 |
| `--compress` | `-c` | 压缩备份文件 | false |

**示例：**
```bash
# 创建备份（自动查找配置文件）
mihomo-cli config backup create

# 指定配置文件创建备份
mihomo-cli config backup create -p /path/to/config.yaml

# 创建带备注的备份
mihomo-cli config backup create -n "before-update"
```

#### 2.2.2 列出备份命令

```bash
mihomo-cli config backup list [选项]
```

**功能：**
- 列出所有可用的备份文件
- 显示备份时间、大小、备注等信息

**选项：**
| 选项 | 简写 | 说明 | 默认值 |
|------|------|------|--------|
| `--path` | `-p` | 指定配置文件路径 | 自动查找 |
| `--all` | `-a` | 列出所有配置文件的备份 | false |

**输出格式：**
```
备份列表 (config.yaml):
  序号  时间                  大小    备注
  1     2026-03-17 14:30:25   2.3KB   before-update
  2     2026-03-16 10:15:00   2.1KB   auto-backup
  3     2026-03-15 08:00:00   2.0KB   
```

#### 2.2.3 恢复备份命令

```bash
mihomo-cli config backup restore <备份文件|序号> [选项]
```

**功能：**
- 从指定备份文件恢复配置
- 恢复前自动备份当前配置

**选项：**
| 选项 | 简写 | 说明 | 默认值 |
|------|------|------|--------|
| `--force` | `-f` | 跳过确认直接恢复 | false |
| `--no-reload` | | 恢复后不自动重载配置 | false |

**示例：**
```bash
# 通过序号恢复
mihomo-cli config backup restore 1

# 通过文件路径恢复
mihomo-cli config backup restore /path/to/backup.yaml

# 强制恢复（跳过确认）
mihomo-cli config backup restore 1 -f
```

#### 2.2.4 删除备份命令

```bash
mihomo-cli config backup delete <备份文件|序号> [选项]
```

**功能：**
- 删除指定的备份文件

**选项：**
| 选项 | 简写 | 说明 | 默认值 |
|------|------|------|--------|
| `--all` | | 删除所有备份（需二次确认） | false |
| `--keep` | `-k` | 保留最近 N 个备份，删除其余 | 0 |
| `--older-than` | | 删除超过指定天数的备份 | 0 |

**示例：**
```bash
# 删除指定备份
mihomo-cli config backup delete 1

# 删除所有备份
mihomo-cli config backup delete --all

# 保留最近 5 个备份，删除其余
mihomo-cli config backup delete --keep 5

# 删除 30 天前的备份
mihomo-cli config backup delete --older-than 30
```

#### 2.2.5 清理备份命令

```bash
mihomo-cli config backup prune [选项]
```

**功能：**
- 按策略清理旧备份文件

**选项：**
| 选项 | 简写 | 说明 | 默认值 |
|------|------|------|--------|
| `--keep` | `-k` | 保留最近 N 个备份 | 10 |
| `--older-than` | | 删除超过指定天数的备份 | 0 |
| `--dry-run` | | 仅显示将被删除的备份，不实际删除 | false |

**示例：**
```bash
# 保留最近 10 个备份
mihomo-cli config backup prune

# 保留最近 5 个备份
mihomo-cli config backup prune --keep 5

# 删除 30 天前的备份
mihomo-cli config backup prune --older-than 30

# 预览清理结果
mihomo-cli config backup prune --dry-run
```

---

## 三、实现方案

### 3.1 文件结构

```
internal/config/
├── backup.go        # 备份管理核心逻辑
├── backup_test.go   # 单元测试
cmd/
├── config.go        # 添加 backup 子命令注册
```

### 3.2 核心数据结构

```go
// BackupInfo 备份文件信息
type BackupInfo struct {
    Path      string    // 备份文件路径
    Size      int64     // 文件大小（字节）
    CreatedAt time.Time // 创建时间
    Note      string    // 备份备注
    Checksum  string    // 文件校验和（MD5）
}

// BackupManager 备份管理器
type BackupManager struct {
    configPath string        // 配置文件路径
    backupDir  string        // 备份目录
    maxBackups int           // 最大保留数量
    maxAge     time.Duration // 最大保留时间
}
```

### 3.3 核心方法

```go
// NewBackupManager 创建备份管理器
func NewBackupManager(configPath string) *BackupManager

// CreateBackup 创建备份
func (bm *BackupManager) CreateBackup(note string) (*BackupInfo, error)

// ListBackups 列出所有备份
func (bm *BackupManager) ListBackups() ([]*BackupInfo, error)

// RestoreBackup 恢复备份
func (bm *BackupManager) RestoreBackup(backupPath string) error

// DeleteBackup 删除备份
func (bm *BackupManager) DeleteBackup(backupPath string) error

// PruneBackups 清理旧备份
func (bm *BackupManager) PruneBackups(keep int, olderThan time.Duration) ([]string, error)

// GetBackupByIndex 通过序号获取备份
func (bm *BackupManager) GetBackupByIndex(index int) (*BackupInfo, error)
```

### 3.4 备份目录结构（方案B）

采用统一备份目录方案，便于集中管理：

```
~/.mihomo-cli/
├── config.yaml           # CLI 配置文件
└── backups/              # 备份目录
    ├── config.20260317-143025.before-update.yaml
    ├── config.20260316-101500.auto-backup.yaml
    └── config.20260315-080000.yaml
```

**优点：**
- 统一管理，便于查看和清理
- 不污染配置文件所在目录
- 支持多配置文件的备份

### 3.5 备份文件命名规则

```
{配置文件名}.{时间戳}.{备注}.yaml
```

**示例：**
- `config.20260317-143025.before-update.yaml` - 带备注
- `config.20260317-143025.yaml` - 无备注

**时间戳格式：** `YYYYMMDD-HHMMSS`

### 3.6 元数据存储

将备注信息编码到文件名中，无需额外的元数据文件。解析文件名即可获取：
- 原配置文件名
- 备份时间
- 备份备注

---

## 四、实现步骤

### 4.1 第一阶段：核心功能

1. 创建 `internal/config/backup.go`
   - 实现 `BackupInfo` 结构体
   - 实现 `BackupManager` 结构体
   - 实现核心方法

2. 创建 `internal/config/backup_test.go`
   - 编写单元测试

### 4.2 第二阶段：命令集成

1. 修改 `cmd/config.go`
   - 添加 `backup` 子命令组
   - 实现 `create`、`list`、`restore`、`delete`、`prune` 子命令

### 4.3 第三阶段：测试与文档

1. 编写集成测试
2. 更新 README 文档

---

## 五、与现有功能的集成

### 5.1 与 `config edit` 的关系

- `config edit` 继续使用现有的自动备份功能
- 备份文件统一存储到 `~/.mihomo-cli/backups/` 目录
- 可通过 `config backup list` 查看所有备份（包括自动备份）

### 5.2 备份恢复后的处理

恢复备份后，自动调用 `config reload` 使配置生效，除非指定 `--no-reload` 参数。

---

## 六、安全考虑

1. **恢复前备份**：恢复操作前自动备份当前配置，防止误操作
2. **确认提示**：删除和恢复操作需要用户确认（除非使用 `--force`）
3. **校验和验证**：恢复前验证备份文件的完整性
4. **权限检查**：确保备份目录有正确的读写权限
