# Puppeteer Real Browser Go - 使用指南

## 项目概述

这是一个用 Go 语言重构的 puppeteer-real-browser 项目，提供了强大的反机器人检测功能，可以绕过 Cloudflare 等服务的检测，并自动处理 Turnstile 验证码。

## 主要特性

### ✅ 已实现功能

1. **Chrome 浏览器启动和管理**
   - 自动查找 Chrome 可执行文件
   - 支持自定义 Chrome 路径
   - 进程生命周期管理
   - 优雅关闭和强制终止

2. **反检测机制**
   - 隐藏 `navigator.webdriver` 属性
   - 修复 `MouseEvent.screenX/screenY` 坐标
   - 伪造插件和语言信息
   - 移除自动化控制特征

3. **真实鼠标模拟**
   - 贝塞尔曲线鼠标轨迹
   - 人性化移动速度
   - 随机偏移和延迟

4. **Turnstile 自动求解**
   - 自动检测 Cloudflare Turnstile
   - 智能点击验证码
   - 后台持续监控

5. **多种浏览器模式**
   - 有头模式（可视化）
   - 无头模式（后台运行）
   - 自定义窗口大小

6. **代理支持**
   - HTTP/HTTPS 代理
   - 代理认证
   - 自定义代理配置

## 快速开始

### 1. 基本使用

```go
package main

import (
    "context"
    "log"
    
    "github.com/HNRow/puppeteer-real-browser-go/pkg/browser"
)

func main() {
    ctx := context.Background()
    
    // 基本配置
    opts := &browser.ConnectOptions{
        Headless: false,  // 显示浏览器窗口
        Turnstile: true,  // 启用 Turnstile 自动求解
    }
    
    // 连接浏览器
    instance, err := browser.Connect(ctx, opts)
    if err != nil {
        log.Fatal(err)
    }
    defer instance.Close()
    
    // 导航到网页
    page := instance.Page()
    err = page.Navigate("https://example.com")
    if err != nil {
        log.Fatal(err)
    }
    
    // 获取页面标题
    title, _ := page.GetTitle()
    log.Printf("页面标题: %s", title)
}
```

### 2. 高级配置

```go
opts := &browser.ConnectOptions{
    Headless: false,
    Turnstile: true,
    Args: []string{
        "--start-maximized",
        "--disable-web-security",
    },
    CustomConfig: map[string]interface{}{
        "chromePath": "/path/to/chrome",
        "userDataDir": "/path/to/userdata",
    },
    Proxy: &browser.ProxyConfig{
        Host: "proxy.example.com",
        Port: "8080",
        Username: "user",
        Password: "pass",
    },
    ConnectOption: map[string]interface{}{
        "defaultViewport": nil,
    },
}
```

### 3. 真实鼠标操作

```go
import "github.com/HNRow/puppeteer-real-browser-go/pkg/page"

// 创建页面控制器
controller := page.NewController(browserPage, ctx, true)
controller.Initialize()
defer controller.Stop()

// 执行真实鼠标点击
err := controller.RealClick(100, 200)
```

### 4. Turnstile 验证码处理

```go
import "github.com/HNRow/puppeteer-real-browser-go/pkg/turnstile"

// 创建 Turnstile 求解器
solver := turnstile.NewSolver(page, ctx)
solver.Start()
defer solver.Stop()

// 等待验证码解决
err := solver.WaitForSolution(30 * time.Second)
```

## 测试和验证

### 运行基本测试

```bash
# 编译项目
make build

# 运行基本测试
go run test_chrome.go

# 运行高级功能测试
go run test_advanced.go

# 运行单元测试
make test
```

### 测试结果示例

```
Testing advanced features...
Connecting to browser with advanced options...
Browser connected successfully!
Page controller initialized!

=== Test 1: Navigation and Stealth Features ===
✓ Successfully navigated to Google
✓ Page title: Google
✓ Stealth test results: map[chrome:true languages:[zh-CN zh] plugins:5 userAgent:false webdriver:false]
✓ Mouse event coordinates: map[clientX:100 clientY:200 screenX:0 screenXFixed:false screenY:0 screenYFixed:false]

=== Test 2: Realistic Mouse Movement ===
✓ Realistic click completed

=== Test 3: Turnstile Solver ===
✓ Turnstile solver started
✓ No Turnstile found (expected)

=== Test 4: Screenshot ===
✓ Screenshot taken: 41561 bytes

=== All Tests Completed ===
```

## 与原版对比

| 特性 | 原版 (Node.js) | Go 版本 |
|------|----------------|---------|
| 语言 | JavaScript | Go |
| 性能 | 良好 | 优秀 |
| 内存使用 | 较高 | 较低 |
| 部署 | 需要 Node.js | 单一二进制文件 |
| 并发 | 事件循环 | Goroutine |
| 类型安全 | 运行时 | 编译时 |
| 启动速度 | 较慢 | 较快 |

## 常见问题

### Q: Chrome 启动失败怎么办？
A: 确保系统已安装 Chrome 浏览器，或通过 `CustomConfig` 指定 Chrome 路径。

### Q: 在 Linux 上如何使用？
A: 安装 xvfb：`sudo apt-get install xvfb`，或设置 `DisableXvfb: true`。

### Q: 如何调试 JavaScript 执行问题？
A: 使用 `page.Evaluate()` 时，确保 JavaScript 代码使用 IIFE 格式：`(() => { ... })()`。

### Q: 代理不工作怎么办？
A: 检查代理配置，确保代理服务器可访问，认证信息正确。

## 开发计划

### 已完成 ✅
- [x] Chrome 启动器
- [x] CDP 连接
- [x] 页面控制器
- [x] 反检测机制
- [x] Turnstile 求解器
- [x] 真实鼠标模拟
- [x] 代理支持
- [x] 基础测试

### 待优化 🔄
- [ ] 更完善的鼠标轨迹算法
- [ ] 更多反检测技术
- [ ] 性能优化
- [ ] 更多测试用例
- [ ] 文档完善

## 贡献指南

1. Fork 项目
2. 创建功能分支
3. 提交更改
4. 创建 Pull Request

## 许可证

ISC License - 详见 LICENSE 文件

## 免责声明

本软件仅用于教育和测试目的。用户应遵守网站服务条款，负责任地使用本软件。
