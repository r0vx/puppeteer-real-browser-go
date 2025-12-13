# 🎭 浏览器指纹伪造使用指南

## 📋 目录
- [快速开始](#快速开始)
- [基础用法](#基础用法)
- [高级用法](#高级用法)
- [最佳实践](#最佳实践)
- [常见问题](#常见问题)

---

## 🚀 快速开始

### 方法1：最简单的方式（推荐）

```go
package main

import (
    "context"
    "github.com/r0vx/puppeteer-real-browser-go/pkg/browser"
)

func main() {
    ctx := context.Background()
    
    // 1. 创建指纹管理器
    manager, _ := browser.NewUserFingerprintManager("./fingerprint_configs")
    
    // 2. 获取用户指纹（自动生成）
    fingerprint, _ := manager.GetUserFingerprint("user_001")
    
    // 3. 创建指纹注入器
    injector := browser.NewFingerprintInjector(fingerprint)
    
    // 4. 配置浏览器
    opts := &browser.ConnectOptions{
        Headless: false,
        Args:     fingerprint.GetChromeFlags(), // 自动应用指纹参数
    }
    
    // 5. 启动浏览器
    instance, _ := browser.Connect(ctx, opts)
    defer instance.Close()
    
    page := instance.Page()
    
    // 6. 注入指纹脚本（重要！）
    page.EvaluateOnNewDocument(injector.GenerateInjectionScript())
    
    // 7. 正常使用
    page.Navigate("https://browserleaks.com/canvas")
    
    // 完成！现在浏览器使用的是伪造的指纹
}
```

### 方法2：一行代码搞定

```go
package main

import (
    "context"
    "github.com/r0vx/puppeteer-real-browser-go/pkg/browser"
)

func main() {
    // 使用高级指纹管理器（自动处理所有细节）
    manager, _ := browser.NewAdvancedFingerprintManager("./fingerprint_configs")
    defer manager.Close()
    
    opts := &browser.ConnectOptions{Headless: false}
    
    // 一键启动带完整指纹的浏览器
    instance, _ := manager.LaunchBrowserWithFullFingerprint(
        context.Background(), 
        "user_001",  // 用户ID
        opts,
    )
    defer instance.Close()
    
    // 开始使用
    // ...
}
```

---

## 📖 基础用法

### 1. 创建指纹管理器

```go
// 指定配置文件目录
manager, err := browser.NewUserFingerprintManager("./fingerprint_configs")
if err != nil {
    log.Fatal(err)
}
```

### 2. 获取/生成指纹

```go
// 方式A：自动生成（基于用户ID的确定性生成）
fingerprint, err := manager.GetUserFingerprint("user_001")

// 方式B：加载自定义指纹
fingerprint, err := browser.LoadFingerprintConfig("./my_fingerprint.json")

// 方式C：从真实浏览器收集指纹
// 运行: go run cmd/fingerprint_collector/main.go
// 然后在浏览器中打开 http://localhost:8765
```

### 3. 查看指纹信息

```go
fmt.Printf("User-Agent: %s\n", fingerprint.Browser.UserAgent)
fmt.Printf("Platform: %s\n", fingerprint.Browser.Platform)
fmt.Printf("Screen: %dx%d\n", fingerprint.Screen.Width, fingerprint.Screen.Height)
fmt.Printf("WebGL: %s\n", fingerprint.WebGL.Renderer)
fmt.Printf("CPU Cores: %d\n", fingerprint.Browser.HardwareConcurrency)
```

### 4. 应用指纹到浏览器

```go
// 创建注入器
injector := browser.NewFingerprintInjector(fingerprint)

// 配置浏览器启动参数
opts := &browser.ConnectOptions{
    Headless: false,
    Args:     fingerprint.GetChromeFlags(), // 自动获取所需参数
}

// 启动浏览器
instance, _ := browser.Connect(ctx, opts)
page := instance.Page()

// 注入JavaScript指纹修改脚本
page.EvaluateOnNewDocument(injector.GenerateInjectionScript())

// 现在可以正常导航了
page.Navigate("https://example.com")
```

---

## 🎯 高级用法

### 多用户/多指纹管理

```go
func runMultipleUsers() {
    manager, _ := browser.NewUserFingerprintManager("./fingerprint_configs")
    
    users := []string{"user_001", "user_002", "user_003"}
    
    for _, userID := range users {
        // 每个用户独立的指纹
        fingerprint, _ := manager.GetUserFingerprint(userID)
        injector := browser.NewFingerprintInjector(fingerprint)
        
        opts := &browser.ConnectOptions{
            Headless:       false,
            ProfileName:    userID, // 每个用户独立的Profile
            PersistProfile: true,
            Args:           fingerprint.GetChromeFlags(),
        }
        
        instance, _ := browser.Connect(ctx, opts)
        page := instance.Page()
        page.EvaluateOnNewDocument(injector.GenerateInjectionScript())
        
        // 使用该用户身份访问
        page.Navigate("https://example.com")
        
        // ... 执行操作 ...
        
        instance.Close()
    }
}
```

### 动态切换指纹

```go
func switchFingerprints() {
    manager, _ := browser.NewUserFingerprintManager("./fingerprint_configs")
    
    // 使用用户1的指纹
    fp1, _ := manager.GetUserFingerprint("user_001")
    runWithFingerprint(fp1, "https://site1.com")
    
    // 切换到用户2的指纹
    fp2, _ := manager.GetUserFingerprint("user_002")
    runWithFingerprint(fp2, "https://site2.com")
}

func runWithFingerprint(fp *browser.FingerprintConfig, url string) {
    injector := browser.NewFingerprintInjector(fp)
    opts := &browser.ConnectOptions{
        Headless: false,
        Args:     fp.GetChromeFlags(),
    }
    
    instance, _ := browser.Connect(context.Background(), opts)
    defer instance.Close()
    
    page := instance.Page()
    page.EvaluateOnNewDocument(injector.GenerateInjectionScript())
    page.Navigate(url)
    
    // ... 操作 ...
}
```

### 自定义指纹

```go
func customFingerprint() {
    // 生成基础指纹
    manager, _ := browser.NewUserFingerprintManager("./fingerprint_configs")
    fp, _ := manager.GetUserFingerprint("user_001")
    
    // 自定义修改
    fp.Browser.UserAgent = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) ..."
    fp.Screen.Width = 2560
    fp.Screen.Height = 1440
    fp.Browser.HardwareConcurrency = 16
    
    // 保存自定义指纹
    manager.CreateCustomUserFingerprint("custom_user", fp)
    
    // 使用自定义指纹
    customFP, _ := manager.GetUserFingerprint("custom_user")
    // ... 使用 ...
}
```

### 克隆已有指纹

```go
// 克隆用户1的指纹给用户2使用
manager.CloneUserFingerprint("user_001", "user_002")

// 现在user_002有和user_001完全相同的指纹
fp, _ := manager.GetUserFingerprint("user_002")
```

---

## 💡 最佳实践

### 1. 指纹持久化

```go
// ✅ 推荐：为每个用户使用持久化Profile
opts := &browser.ConnectOptions{
    PersistProfile: true,
    ProfileName:    fmt.Sprintf("fp_%s", userID),
    // ...
}

// 好处：
// - Cookie会保存
// - 指纹一致性更好
// - 模拟真实用户行为
```

### 2. 指纹与Proxy配合

```go
func useWithProxy() {
    fingerprint, _ := manager.GetUserFingerprint("user_001")
    
    opts := &browser.ConnectOptions{
        Headless: false,
        Args: append(
            fingerprint.GetChromeFlags(),
            "--proxy-server=http://proxy.example.com:8080",
        ),
    }
    
    // 现在指纹+代理双重保护
    instance, _ := browser.Connect(ctx, opts)
    // ...
}
```

### 3. 测试指纹效果

```go
func testFingerprint(page browser.Page) {
    // 访问指纹检测网站
    testSites := []string{
        "https://browserleaks.com/canvas",      // Canvas指纹
        "https://abrahamjuliot.github.io/creepjs/", // 综合检测
        "https://pixelscan.net/",               // Bot检测
        "https://amiunique.org/",               // 唯一性检测
    }
    
    for _, site := range testSites {
        page.Navigate(site)
        time.Sleep(5 * time.Second)
        // 手动检查结果
    }
}
```

### 4. 错误处理

```go
func robustFingerprint() {
    manager, err := browser.NewUserFingerprintManager("./fingerprint_configs")
    if err != nil {
        log.Fatalf("指纹管理器创建失败: %v", err)
    }
    
    fingerprint, err := manager.GetUserFingerprint("user_001")
    if err != nil {
        log.Fatalf("获取指纹失败: %v", err)
    }
    
    injector := browser.NewFingerprintInjector(fingerprint)
    opts := &browser.ConnectOptions{
        Headless: false,
        Args:     fingerprint.GetChromeFlags(),
    }
    
    instance, err := browser.Connect(ctx, opts)
    if err != nil {
        log.Fatalf("浏览器启动失败: %v", err)
    }
    defer instance.Close()
    
    page := instance.Page()
    
    // 注入脚本（如果失败也继续）
    if err := page.EvaluateOnNewDocument(injector.GenerateInjectionScript()); err != nil {
        log.Printf("⚠️  指纹脚本注入失败（浏览器仍可用）: %v", err)
    }
    
    // 继续使用...
}
```

---

## 🔧 配置说明

### 指纹配置文件结构

```json
{
  "user_id": "user_001",
  "screen": {
    "width": 1920,
    "height": 1080,
    "device_pixel_ratio": 1.0
  },
  "browser": {
    "user_agent": "Mozilla/5.0...",
    "platform": "Win32",
    "hardware_concurrency": 8
  },
  "webgl": {
    "vendor": "Google Inc. (NVIDIA)",
    "renderer": "ANGLE (NVIDIA, NVIDIA GeForce RTX 3060...)"
  },
  "audio": {
    "sample_rate": 48000,
    "max_channel_count": 2
  },
  "canvas": {
    "noise_level": 0.005,
    "text_variance": 3
  }
  // ... 更多配置
}
```

### Chrome启动参数说明

```go
// GetChromeFlags() 会自动生成以下参数：
--user-agent=...           // 用户代理
--lang=...                 // 语言
--window-size=...          // 窗口大小
// ... 以及其他指纹相关参数
```

---

## ❓ 常见问题

### Q1: 为什么会出现"恢复页面"的提示？

**A**: 这是Chrome在检测到异常退出时的提示。解决方法：

```go
// 方法1：添加禁用参数（推荐）
args := fingerprintConfig.GetChromeFlags()
args = append(args,
    "--disable-session-crashed-bubble",  // 禁用崩溃提示
    "--disable-infobars",                // 禁用信息栏
    "--no-first-run",                    // 禁用首次运行
    "--no-default-browser-check",        // 禁用默认浏览器检查
)

opts := &browser.ConnectOptions{
    Args: args,
}
```

```bash
# 方法2：清理旧Profile
./clean_profiles.sh
```

### Q2: 指纹是否每次都不同？

**A**: 不是！使用相同的用户ID会生成**相同的指纹**，这是设计如此，目的是保持一致性。如果需要不同指纹，使用不同的用户ID。

```go
fp1, _ := manager.GetUserFingerprint("user_001") // 第1次
fp2, _ := manager.GetUserFingerprint("user_001") // 第2次
// fp1 == fp2 （相同）

fp3, _ := manager.GetUserFingerprint("user_002") // 不同用户
// fp3 != fp1 （不同）
```

### Q3: 需要重启浏览器才能应用新指纹吗？

**A**: 是的。指纹需要在浏览器启动时应用。如果要切换指纹，需要关闭当前浏览器实例并启动新实例。

### Q4: 指纹对所有网站都有效吗？

**A**: 是的，指纹修改是全局的，对所有访问的网站都有效。

### Q5: Canvas/WebGL指纹是否会暴露？

**A**: 如果正确使用本项目，指纹会被修改。可以访问 https://browserleaks.com/canvas 测试效果。

### Q6: 如何收集真实设备的指纹？

**A**: 运行指纹收集工具：

```bash
# 方法1：使用HTML页面
go run cmd/fingerprint_collector/main.go
# 然后打开 http://localhost:8765

# 方法2：直接打开HTML文件
open fingerprint_collector.html
# 点击下载JSON即可
```

### Q7: 指纹配置池有多大？

**A**: 运行统计工具查看：

```bash
go run cmd/fingerprint_stats/main.go
```

当前配置池：**34万亿**种组合！

### Q8: 指纹会被检测为机器人吗？

**A**: 如果使用得当，不会。本项目的指纹：
- ✅ 基于真实设备配置
- ✅ 使用加权随机（常见配置更容易被选中）
- ✅ 各项指纹参数相互一致
- ✅ 包含增强版Audio/WebGL修改

### Q9: 可以和代理一起使用吗？

**A**: 可以！推荐组合使用：

```go
opts := &browser.ConnectOptions{
    Args: append(
        fingerprint.GetChromeFlags(),
        "--proxy-server=socks5://127.0.0.1:1080",
    ),
}
```

---

## 🎬 完整示例

查看完整的示例代码：

```bash
# 基础示例
go run cmd/example/fingerprint_demo.go

# 简单测试
go run cmd/example/simple_demo.go

# 高级用法
go run cmd/example/main.go
```

---

## 📚 相关资源

- **指纹检测网站**:
  - https://browserleaks.com/
  - https://abrahamjuliot.github.io/creepjs/
  - https://pixelscan.net/
  - https://amiunique.org/

- **文档**:
  - [快速开始指南](./dom/quick_setup_guide.md)
  - [反检测修复说明](./dom/ANTI_DETECTION_FIXES.md)

---

## 🚀 总结

使用指纹的基本流程：

1. **创建管理器** → `NewUserFingerprintManager()`
2. **获取指纹** → `GetUserFingerprint(userID)`
3. **创建注入器** → `NewFingerprintInjector(fingerprint)`
4. **配置浏览器** → 使用 `GetChromeFlags()`
5. **启动浏览器** → `Connect()`
6. **注入脚本** → `EvaluateOnNewDocument()`
7. **正常使用** → `Navigate()` 等

就是这么简单！🎉

