# 缓存管理命令 (cache)

缓存管理命令用于管理 Mihomo 的缓存，包括 FakeIP 和 DNS 缓存。

## 命令列表

### cache clear fakeip - 清空 FakeIP 池

清空 FakeIP 地址池。

**语法：**
```bash
mihomo-cli cache clear fakeip
```

**示例：**
```bash
mihomo-cli cache clear fakeip
```

### cache clear dns - 清空 DNS 缓存

清空 DNS 缓存。

**语法：**
```bash
mihomo-cli cache clear dns
```

**示例：**
```bash
mihomo-cli cache clear dns
```

## 缓存说明

### FakeIP 缓存
- FakeIP 是 Mihomo 的一种优化机制，用于加速 DNS 解析
- 当遇到网络问题时，清空 FakeIP 可以帮助恢复正常解析
- 建议在切换网络环境后清空 FakeIP

### DNS 缓存
- DNS 缓存存储了域名解析的结果
- 清空 DNS 缓存会强制重新解析域名
- 当域名解析出现问题时，可以尝试清空 DNS 缓存

## 注意事项

1. 清空缓存会暂时影响性能，因为需要重新建立缓存
2. 缓存清空操作是瞬时的，不需要重启 Mihomo
3. 建议在遇到网络问题时先尝试清空缓存
