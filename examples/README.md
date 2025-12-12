# 浏览器上下文和指纹浏览器示例

这个目录包含了展示如何使用浏览器上下文和指纹浏览器功能的完整示例。

## 📁 示例文件

### 1. `multi_account_demo.go` - 多账号管理演示
展示如何使用浏览器上下文进行多账号管理，包括：
- ✅ 创建独立的浏览器上下文
- 🔐 并行登录多个账号  
- 🍪 Session 和 Cookie 隔离
- 🎯 不同账号执行不同任务

### 2. `fingerprint_browser_demo.go` - 指纹浏览器演示
展示完整的指纹浏览器实现，包括：
- 🎭 随机生成浏览器指纹
- 💾 指纹配置的保存和加载
- 🌐 使用不同指纹并行浏览
- 🔍 指纹检测和验证
- 🛒 电商场景应用

### 3. `extension_demo.go` - Chrome 插件支持演示
展示 Chrome 插件集成功能，包括：
- 🧩 加载和管理 Chrome 插件
- 📦 创建示例插件（广告拦截器、密码管理器）
- 🌐 插件在多上下文中的工作
- 🎭 插件与指纹浏览器的结合
- 🔍 插件功能验证

## 🚀 运行示例

### 运行多账号管理演示
```bash
cd examples
go run multi_account_demo.go
```

### 运行指纹浏览器演示
```bash
cd examples  
go run fingerprint_browser_demo.go
```

## 💡 关键特性演示

### 多账号管理 (`multi_account_demo.go`)

#### 1. 账号管理器初始化
```go
manager, err := NewAccountManager(5) // 最多5个并发账号
```

#### 2. 添加账号和创建上下文
```go
account := &Account{
    Name:     "Alice",
    Email:    "alice@example.com",
    Password: "password123",
}
manager.AddAccount(account) // 自动创建独立的浏览器上下文
```

#### 3. 并行登录
```go
manager.LoginAllAccounts() // 所有账号并行登录
```

#### 4. Session 隔离验证
```go
manager.TestSessionIsolation() // 验证 Cookie 隔离
```

### 指纹浏览器 (`fingerprint_browser_demo.go`)

#### 1. 随机指纹生成
```go
fingerprint := GenerateRandomFingerprint("user_usa_windows")
// 包含: UserAgent, Platform, Language, Timezone, Screen, WebGL 等
```

#### 2. 指纹应用
```go
manager.BrowseWithProfile("user_usa_windows", "https://example.com")
// 自动应用指纹设置到页面
```

#### 3. 指纹检测测试
```go
manager.TestFingerprintDetection()
// 验证指纹是否成功应用
```

## 🎯 实际应用场景

### 多账号电商运营
```go
// 创建不同地区的买家账号
accounts := []string{"buyer_usa", "buyer_uk", "buyer_de"}
for _, account := range accounts {
    manager.AddAccount(&Account{Name: account})
    // 每个账号有独立的 Cookie 和 Session
}
```

### 社交媒体管理
```go
// 管理多个品牌账号
socialAccounts := []string{"brand_main", "brand_support", "personal"}
for _, account := range socialAccounts {
    // 独立的浏览器上下文，避免互相影响
    ctx := browserInstance.CreateBrowserContext(nil)
}
```

### 数据收集和竞品分析
```go
// 不同指纹收集不同地区数据
regions := []string{"us_west", "eu_central", "asia_pacific"}
for _, region := range regions {
    // 每个地区使用不同的指纹和代理
    manager.BrowseWithProfile(region, targetURL)
}
```

## 🔧 技术亮点

### 1. 真实的浏览器指纹
- 🌐 **User Agent**: 匹配操作系统的真实浏览器标识
- 📱 **屏幕分辨率**: 根据平台生成合理的分辨率
- 🌍 **时区和语言**: 地理位置一致性
- 🎨 **Canvas/WebGL**: 硬件指纹随机化

### 2. 完整的隔离机制
- 🍪 **Cookie 隔离**: 每个上下文独立的 Cookie 存储
- 💾 **LocalStorage 隔离**: 独立的本地存储
- 🔐 **Session 隔离**: 登录状态互不影响
- 🌐 **代理支持**: 每个上下文可配置独立代理

### 3. 高级反检测
- 🛡️ **Runtime.Enable 绕过**: 避免 Cloudflare 检测
- 🖱️ **人类行为模拟**: 真实的鼠标轨迹和打字节奏
- ⚡ **动态指纹**: 可随时切换指纹配置
- 🎭 **批量管理**: 支持大量并发实例

## 📊 性能特点

- **内存效率**: 多个上下文共享同一 Chrome 进程
- **启动速度**: 复用已启动的浏览器实例
- **并发能力**: 支持大量并发上下文和页面
- **资源管理**: 自动清理和资源回收

## 🚨 注意事项

1. **合规使用**: 请确保符合目标网站的使用条款
2. **资源管理**: 及时关闭不需要的上下文和页面
3. **代理配置**: 使用高质量的代理服务器
4. **指纹更新**: 定期更新指纹库以保持有效性

## 💡 扩展建议

1. **数据库存储**: 将指纹配置存储到数据库
2. **Web 界面**: 开发 Web 管理界面
3. **API 接口**: 提供 REST API 进行远程管理
4. **监控告警**: 添加异常检测和告警机制
5. **负载均衡**: 实现多机器分布式部署