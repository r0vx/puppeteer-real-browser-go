# CRX扩展自动加载指南

## 概述

现在支持使用打包好的`.crx`文件自动加载Chrome扩展，这比未打包的扩展目录更稳定可靠。

## 可用扩展

在`examples/path/crx/`目录下有以下打包好的扩展：

1. **Discord Token Login** (`1.0_0.crx`)
   - 用于Discord自动登录和token管理
   - 文件大小：~16KB

2. **OKX Wallet** (`3.66.10_0.crx`)
   - OKX官方Web3钱包扩展
   - 文件大小：~47MB
   - 支持多链钱包功能

## 使用方法

### 基本用法

```go
opts := &browser.ConnectOptions{
    Headless:                  false,
    AutoLoadDefaultExtensions: true,  // 启用自动加载扩展
    PersistProfile:            true,  // 启用持久化配置
    ProfileName:               "my_extensions_profile",
}

instance, err := browser.Connect(ctx, opts)
```

### 运行演示

```bash
go run examples/auto_load_extensions.go
```

## CRX文件的优势

- ✅ **稳定性更好**: 使用打包文件，避免文件权限问题
- ✅ **更接近真实使用**: 模拟正式扩展安装方式
- ✅ **支持签名验证**: Chrome可以验证扩展完整性
- ✅ **部署简单**: 单个文件包含完整扩展
- ✅ **性能更好**: 减少文件系统访问

## 工作原理

1. 配置中设置`AutoLoadDefaultExtensions: true`
2. 系统自动读取`examples/path/crx/`目录下的`.crx`文件
3. 通过`--load-extension`标志将CRX文件传递给Chrome
4. Chrome自动加载和注册扩展
5. 扩展在`chrome://extensions/`页面可见

## Chrome启动参数

自动加载时会添加以下关键参数：

```
--enable-extensions
--load-extension=/path/to/1.0_0.crx,/path/to/3.66.10_0.crx
--enable-extension-activity-logging
--disable-extensions-file-access-check
--disable-extensions-http-throttling
--disable-extensions-install-verification
```

## 自定义扩展

如果你有自己的`.crx`文件，可以：

1. 将`.crx`文件放在`examples/path/crx/`目录下
2. 修改`internal/config/config.go`中的`GetDefaultExtensionPaths()`函数
3. 添加新的.crx文件路径到返回的数组中

```go
func GetDefaultExtensionPaths() []string {
    return []string{
        "examples/path/crx/1.0_0.crx",      // Discord Token Login
        "examples/path/crx/3.66.10_0.crx",  // OKX Wallet
        "examples/path/crx/your_extension.crx", // 你的扩展
    }
}
```

## 故障排除

### 扩展未显示
1. 检查`.crx`文件是否存在且有读取权限
2. 确认文件不是损坏的
3. 查看Chrome控制台是否有错误信息

### 权限问题
```bash
chmod 644 examples/path/crx/*.crx
```

### 测试特定扩展
可以临时只加载某个扩展进行测试：

```go
// 只测试Discord扩展
opts.Extensions = []string{"examples/path/crx/1.0_0.crx"}
opts.AutoLoadDefaultExtensions = false
```

## 注意事项

- CRX文件需要有效的扩展签名才能被Chrome接受
- 一些企业环境可能限制加载外部扩展
- 扩展加载需要Chrome启动时设置适当的标志
- 建议使用持久化配置文件以保持扩展状态