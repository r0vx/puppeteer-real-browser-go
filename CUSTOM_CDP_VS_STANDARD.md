# 🔄 Custom CDP vs Standard 模式对比

## 📊 核心差异

### Runtime.Enable 处理方式

```
┌─────────────────────────────────────────────────────────┐
│  UseCustomCDP: false (标准模式 - 推荐)                   │
└─────────────────────────────────────────────────────────┘

使用 chromedp 标准库
    ↓
通过 Page.addScriptToEvaluateOnNewDocument 注入脚本
    ↓
避免在页面加载时调用 Runtime.Enable
    ↓
✅ Runtime.Enable 被延迟或最小化使用
✅ API 完整，使用方便
✅ 足以应对大多数检测

═══════════════════════════════════════════════════════════

┌─────────────────────────────────────────────────────────┐
│  UseCustomCDP: true (自定义模式 - 极端隐身)              │
└─────────────────────────────────────────────────────────┘

完全自定义 CDP 客户端
    ↓
直接使用 WebSocket 通信
    ↓
仅启用 Page.enable 和 DOM.enable
    ↓
✅ Runtime.Enable 完全不使用
⚠️  需要手动实现某些功能
✅ 最强反检测能力
```

---

## 🎯 功能对比表

| 功能 | UseCustomCDP: false | UseCustomCDP: true | 说明 |
|------|--------------------|--------------------|------|
| **基础导航** | ✅ `page.Navigate(url)` | ✅ `page.Navigate(url)` | 完全相同 |
| **坐标点击** | ✅ `page.Click(x, y)` | ✅ `page.Click(x, y)` | 完全相同 |
| **真实鼠标** | ✅ `page.RealClick(x, y)` | ✅ `page.RealClick(x, y)` | 完全相同 |
| **执行JS** | ✅ `page.Evaluate(js)` | ✅ `page.Evaluate(js)` | 实现方式不同 |
| **截图** | ✅ `page.Screenshot()` | ✅ `page.Screenshot()` | 完全相同 |
| **获取标题** | ✅ `page.GetTitle()` | ✅ `page.GetTitle()` | 完全相同 |
| **选择器点击** | 🟡 需要辅助函数 | 🟡 需要辅助函数 | 都需要 |
| **等待选择器** | ✅ `page.WaitForSelector(sel)` | ✅ `page.WaitForSelector(sel)` | 实现不同 |

### 新增辅助函数（两种模式都可用）

| 辅助函数 | 功能 | 使用示例 |
|---------|------|---------|
| `ClickSelector(page, sel)` | 点击选择器 | `browser.ClickSelector(page, "#btn")` |
| `TypeText(page, sel, text)` | 输入文本 | `browser.TypeText(page, "input", "hello")` |
| `GetElementText(page, sel)` | 获取文本 | `text, _ := browser.GetElementText(page, ".msg")` |
| `GetElementCoords(page, sel)` | 获取坐标 | `coords, _ := browser.GetElementCoords(page, "#el")` |
| `IsElementVisible(page, sel)` | 检查可见 | `visible, _ := browser.IsElementVisible(page, "#popup")` |
| `SelectOption(page, sel, val)` | 选择下拉 | `browser.SelectOption(page, "select", "value1")` |
| `CheckCheckbox(page, sel, checked)` | 勾选框 | `browser.CheckCheckbox(page, "#agree", true)` |

---

## 💻 代码示例对比

### 示例：登录表单

#### UseCustomCDP: false（标准模式）

```go
opts := &browser.ConnectOptions{
    Headless: false,
    UseCustomCDP: false,  // 标准模式
}

instance, _ := browser.Connect(ctx, opts)
page := instance.Page()

page.Navigate("https://example.com/login")

// 使用辅助函数
browser.TypeText(page, "input[name='username']", "myuser")
browser.TypeText(page, "input[name='password']", "mypass")
browser.ClickSelector(page, "button[type='submit']")
```

#### UseCustomCDP: true（自定义模式）

```go
opts := &browser.ConnectOptions{
    Headless: false,
    UseCustomCDP: true,   // 自定义模式
}

instance, _ := browser.Connect(ctx, opts)
page := instance.Page()

page.Navigate("https://example.com/login")

// 完全相同的代码！辅助函数两种模式都支持
browser.TypeText(page, "input[name='username']", "myuser")
browser.TypeText(page, "input[name='password']", "mypass")
browser.ClickSelector(page, "button[type='submit']")
```

**结论**：使用辅助函数后，**两种模式的代码完全一样**！

---

## 🔍 检测规避对比

### Cloudflare 检测点

| 检测点 | 标准模式 | Custom CDP | 说明 |
|--------|---------|------------|------|
| **Runtime.Enable** | 🟡 延迟触发 | ✅ 完全避免 | Custom更好 |
| **Navigator.webdriver** | ✅ 已隐藏 | ✅ 已隐藏 | 相同 |
| **MouseEvent** | ✅ 已修复 | ✅ 已修复 | 相同 |
| **Chrome对象** | ✅ 已添加 | ✅ 已添加 | 相同 |
| **真实鼠标轨迹** | ✅ 支持 | ✅ 支持 | 相同 |
| **指纹伪造** | ✅ 支持 | ✅ 支持 | 相同 |

### 测试结果

| 网站 | 标准模式 | Custom CDP |
|------|---------|------------|
| 普通网站 | ✅ 通过 | ✅ 通过 |
| Cloudflare Basic | ✅ 通过 | ✅ 通过 |
| Cloudflare Turnstile | ✅ 通过 | ✅ 通过 |
| PerimeterX | ✅ 通过 | ✅ 通过 |
| DataDome | 🟡 可能通过 | ✅ 更容易通过 |

---

## 🎯 选择建议

### 使用标准模式（UseCustomCDP: false）如果：

- ✅ 访问一般网站
- ✅ 需要更方便的API
- ✅ 不想处理坐标转换
- ✅ Cloudflare基础检测
- ✅ 99%的使用场景

```go
opts := &browser.ConnectOptions{
    Headless: false,
    // UseCustomCDP 默认就是 false
}
```

### 使用自定义模式（UseCustomCDP: true）如果：

- ✅ 面对极强的反自动化检测
- ✅ Cloudflare Turnstile持续失败
- ✅ DataDome等高级检测
- ✅ 需要绝对避免Runtime.Enable
- ✅ 愿意使用辅助函数

```go
opts := &browser.ConnectOptions{
    UseCustomCDP: true,
}
```

---

## 📝 迁移指南

如果你想从标准模式切换到 Custom CDP 模式：

### 代码改动很小！

```go
// 之前（标准模式）
opts := &browser.ConnectOptions{
    Headless: false,
}
instance, _ := browser.Connect(ctx, opts)
page := instance.Page()

// 使用辅助函数操作
browser.ClickSelector(page, "#btn")
browser.TypeText(page, "input", "text")

// ═══════════════════════════════════

// 之后（Custom CDP模式）
opts := &browser.ConnectOptions{
    Headless: false,
    UseCustomCDP: true,  // 只需加这一行！
}
instance, _ := browser.Connect(ctx, opts)
page := instance.Page()

// 代码完全不变！
browser.ClickSelector(page, "#btn")
browser.TypeText(page, "input", "text")
```

**只需要改一行配置！**

---

## 🚀 最佳实践

```go
// 推荐：根据目标网站动态选择模式
func getConnectOptions(targetURL string) *browser.ConnectOptions {
    // 检测目标网站的反爬级别
    useCustomCDP := false
    
    if isHighSecuritySite(targetURL) {
        // 高安全网站使用 Custom CDP
        useCustomCDP = true
    }
    
    return &browser.ConnectOptions{
        Headless:     false,
        UseCustomCDP: useCustomCDP,
        Turnstile:    true,
    }
}

func isHighSecuritySite(url string) bool {
    // Cloudflare、DataDome等高级检测网站
    highSecurityDomains := []string{
        "cloudflare.com",
        "turnstile",
        "datadome",
        "perimeter-x",
    }
    
    for _, domain := range highSecurityDomains {
        if strings.Contains(url, domain) {
            return true
        }
    }
    return false
}
```

---

## 🎬 完整示例


