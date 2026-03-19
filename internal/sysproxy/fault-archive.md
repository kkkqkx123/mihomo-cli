# 系统代理模块缺陷分析报告

## 概述

这是一个设计清晰、跨平台处理得当的模块，使用了经典的工厂模式和接口抽象。代码结构良好，利用 build tags 实现了平台隔离。

本文档记录了已发现的问题及其修复状态。

---

## 已修复问题

### 1. ✅ Linux 下覆盖 `/etc/environment` 的逻辑错误（已修复）

**问题描述：**
在 `linux.go` 的 `Enable` 方法中，当回退写入 `/etc/environment` 时，代码直接使用 `os.WriteFile` 覆盖了整个文件。

**后果：**
`/etc/environment` 通常包含系统关键环境变量（如 `PATH`, `LANG`, `JAVA_HOME` 等）。这段代码会导致该文件原有的所有内容被清空，只保留代理变量。**这会导致系统命令无法找到、语言环境丢失等严重故障。**

**修复方案：**
新增 `addToEtcEnvironment()` 函数，该函数会：
1. 读取 `/etc/environment` 原有内容
2. 过滤掉旧的代理环境变量
3. 将新的代理变量追加到文件末尾

这样既保存了原有系统配置，又正确添加了代理设置。

**修复状态：** ✅ 已完成

---

## 兼容性增强（已实现）

### 2. ✅ Windows 代理设置通知机制（已实现）

**背景：**
Windows 下的代理设置通过注册表配置。对于现代应用程序（如 Chrome、终端），注册表修改通常会立即生效。

**增强功能：**
为了兼容某些长期运行的旧版应用或特定系统组件（如旧版 IE 插件、后台服务），添加了 `InternetSetOption` 通知机制。

**实现细节：**
- 调用 `wininet.dll` 的 `InternetSetOptionW` 函数
- 使用 `INTERNET_OPTION_SETTINGS_CHANGED` 和 `INTERNET_OPTION_REFRESH` 标志
- 在 `Enable` 和 `Disable` 成功后自动调用
- 该功能作为兼容性增强，即使失败也不会影响主要功能

**重要说明：**
- 对于大多数现代应用（Chrome、Firefox、终端等），注册表修改本身已足够
- `InternetSetOption` 主要用于通知依赖 WinINet 缓存旧设置的旧版应用
- 该机制不是必需的，但作为兼容性增强提供了更好的体验

**修复状态：** ✅ 已完成

---

## 用户提示（已添加）

### 3. ✅ Linux 环境变量生效限制提示（已添加）

**说明：**
CLI 工具无法直接修改当前 Shell 会话的环境变量。修改配置文件后，当前终端会话不会立即生效。

**用户提示：**
在 Linux 环境下成功启用代理后，会显示以下提示：
```
Warning: Proxy settings have been saved to configuration file.
Note: The current terminal session will not reflect these changes immediately.
To apply the proxy settings to your current session, run:
  source /etc/environment.d/proxy.conf
Or start a new terminal session.
```

**修复状态：** ✅ 已完成

---

## 其他审查意见

### Linux 实现 (`linux.go`)

- ✅ **优点**：优先使用 `/etc/environment.d/proxy.conf` 是正确的做法。这是 systemd 现代化的配置方式，不会污染主配置文件，且大部分现代 Linux 发行版都支持。
- ✅ **GetStatus 逻辑**：先检查 `os.Getenv` 再检查文件。这意味着如果用户手动 `export HTTP_PROXY=xxx`，CLI 会读取到这个值，但可能和持久化文件中的内容不一致。这在 CLI 工具中属于常见情况，当前逻辑是合理的。

### Windows 实现 (`windows.go`)

- ✅ **Disable 逻辑**：`Disable` 方法只是将 `ProxyEnable` 设为 0，保留了 `ProxyServer` 和 `ProxyOverride` 的值。这是合理的做法，方便下次启用时恢复设置。
- ✅ **错误处理**：错误信息包含详细的恢复建议，体验很好。

### 接口设计 (`interface.go`)

- ✅ **设计合理**：`ProxySettings` 结构体定义清晰，`SysProxy` 接口划分得当。

---

## 总结

| 问题 | 严重程度 | 状态 |
|------|----------|------|
| Linux 覆盖 `/etc/environment` Bug | 🔴 严重 | ✅ 已修复 |
| Windows InternetSetOption 通知机制 | 🟡 兼容性增强 | ✅ 已实现 |
| Linux 环境变量生效提示 | 🟢 用户体验改进 | ✅ 已添加 |

所有发现的问题均已修复，模块现在可以安全使用。