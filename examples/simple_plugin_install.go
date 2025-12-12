package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/HNRow/puppeteer-real-browser-go/pkg/browser"
)

func main() {
	fmt.Println("🔧 简单插件安装演示")
	fmt.Println("==================")

	ctx := context.Background()

	// 1. 创建测试插件
	pluginDir := "test_plugin"
	if err := createTestPlugin(pluginDir); err != nil {
		log.Fatalf("创建测试插件失败: %v", err)
	}
	defer os.RemoveAll(pluginDir)

	// 2. 配置浏览器启动选项
	opts := &browser.ConnectOptions{
		Headless:   false, // 插件需要界面
		Extensions: []string{pluginDir}, // 指定插件目录
		Args: []string{
			"--enable-extensions",
			"--disable-extensions-file-access-check",
			"--load-extension=" + pluginDir, // 直接加载插件
		},
	}

	fmt.Println("🚀 启动带插件的浏览器...")
	instance, err := browser.Connect(ctx, opts)
	if err != nil {
		log.Fatalf("浏览器连接失败: %v", err)
	}
	defer instance.Close()

	fmt.Println("✅ 浏览器启动成功")

	// 3. 测试插件
	page := instance.Page()
	
	// 导航到测试页面
	if err := page.Navigate("https://httpbin.org/get"); err != nil {
		log.Fatalf("页面导航失败: %v", err)
	}

	// 检查插件是否注入成功
	time.Sleep(2 * time.Second)
	
	result, err := page.Evaluate(`
		// 检查插件是否注入了内容
		{
			hasPlugin: window.TestPlugin !== undefined,
			pluginMessage: window.TestPlugin ? window.TestPlugin.message : null,
			chromeRuntime: !!window.chrome?.runtime
		}
	`)
	
	if err != nil {
		fmt.Printf("检查插件失败: %v\n", err)
	} else {
		fmt.Printf("📊 插件状态: %v\n", result)
	}

	// 4. 查看插件管理页面
	fmt.Println("\n📦 打开插件管理页面...")
	context, err := instance.CreateBrowserContext(nil)
	if err == nil {
		pluginPage, err := context.NewPage()
		if err == nil {
			pluginPage.Navigate("chrome://extensions/")
			time.Sleep(1 * time.Second)
			fmt.Println("✅ 请在浏览器中查看 chrome://extensions/ 页面")
		}
	}

	fmt.Println("\n💡 说明:")
	fmt.Println("  1. 插件已自动安装并启用")
	fmt.Println("  2. 查看 chrome://extensions/ 确认插件加载")
	fmt.Println("  3. 插件会在页面中注入 TestPlugin 对象")

	fmt.Println("\n⏳ 保持浏览器开启10秒供测试...")
	time.Sleep(10 * time.Second)

	fmt.Println("✅ 插件安装演示完成")
}

// createTestPlugin 创建一个简单的测试插件
func createTestPlugin(dir string) error {
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	// manifest.json
	manifest := `{
	"manifest_version": 3,
	"name": "测试插件",
	"version": "1.0",
	"description": "一个简单的测试插件",
	"permissions": ["activeTab"],
	"content_scripts": [{
		"matches": ["<all_urls>"],
		"js": ["content.js"]
	}],
	"action": {
		"default_popup": "popup.html"
	}
}`

	if err := os.WriteFile(filepath.Join(dir, "manifest.json"), []byte(manifest), 0644); err != nil {
		return err
	}

	// content.js
	content := `
console.log('🔌 测试插件已加载');

// 在页面中注入插件对象
window.TestPlugin = {
	name: '测试插件',
	version: '1.0',
	message: '插件注入成功！',
	loaded: Date.now()
};

// 修改页面标题
document.title = '📦 ' + document.title + ' (已安装插件)';

console.log('✅ TestPlugin 注入完成:', window.TestPlugin);
`

	if err := os.WriteFile(filepath.Join(dir, "content.js"), []byte(content), 0644); err != nil {
		return err
	}

	// popup.html
	popup := `<!DOCTYPE html>
<html>
<head>
	<style>
		body { width: 200px; padding: 15px; font-family: Arial; }
		h3 { color: #333; margin-top: 0; }
		.info { background: #e8f5e8; padding: 10px; border-radius: 5px; }
	</style>
</head>
<body>
	<h3>🔌 测试插件</h3>
	<div class="info">
		<strong>状态:</strong> ✅ 运行中<br>
		<strong>版本:</strong> 1.0<br>
		<strong>功能:</strong> 页面注入测试
	</div>
	<p><small>这是一个测试插件的弹窗</small></p>
</body>
</html>`

	if err := os.WriteFile(filepath.Join(dir, "popup.html"), []byte(popup), 0644); err != nil {
		return err
	}

	fmt.Println("✅ 测试插件创建完成")
	return nil
}