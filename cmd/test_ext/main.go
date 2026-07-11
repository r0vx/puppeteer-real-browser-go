package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/r0vx/puppeteer-real-browser-go/pkg/browser"
)

func main() {
	fmt.Println("🧪 扩展加载测试")
	fmt.Println("===============")

	ctx := context.Background()

	// 扩展路径
	extensionPaths := []string{
		"../examples/path/Extensions/kfjglmgfjedhhcddpfgfogkahmenikan/1.0_0",     // Discord Token Login
		"../examples/path/Extensions/mcohilncbfahbmgdjkbpemcciiolgcge/3.66.10_0", // OKX Wallet
	}

	fmt.Println("📂 扩展路径:")
	for i, path := range extensionPaths {
		fmt.Printf("  %d. %s\n", i+1, path)
	}

	// 配置选项
	options := &browser.ConnectOptions{
		Headless:       false,
		UseCustomCDP:   false,
		Turnstile:      false,
		Extensions:     extensionPaths,
		PersistProfile: false,
		Args: []string{
			"--start-maximized",
			"--disable-blink-features=AutomationControlled",
		},
	}

	fmt.Println("\n🚀 启动浏览器...")
	instance, err := browser.Connect(ctx, options)
	if err != nil {
		log.Fatalf("启动失败: %v", err)
	}
	defer instance.Close()

	fmt.Println("✅ 浏览器启动成功")

	page := instance.Page()

	fmt.Println("🔍 导航到扩展页面...")
	if err := page.Navigate("chrome://extensions/"); err != nil {
		log.Fatalf("导航失败: %v", err)
	}

	fmt.Println("⏳ 等待页面加载...")
	time.Sleep(5 * time.Second)

	// 检查扩展
	result, err := page.Evaluate(`
		const items = document.querySelectorAll('extensions-item');
		console.log('找到扩展项:', items.length);
		
		const extensions = Array.from(items).map(item => {
			const shadow = item.shadowRoot;
			if (!shadow) return null;
			
			const nameEl = shadow.querySelector('#name');
			const name = nameEl ? nameEl.textContent.trim() : 'Unknown';
			const id = item.getAttribute('id') || '';
			const toggle = shadow.querySelector('cr-toggle');
			const enabled = toggle ? toggle.checked : false;
			
			return { name, id, enabled };
		}).filter(ext => ext !== null);
		
		return {
			totalFound: items.length,
			extensions: extensions,
			developerMode: document.querySelector('#developerMode')?.checked || false
		};
	`)

	if err != nil {
		fmt.Printf("❌ 检查失败: %v\n", err)
	} else {
		fmt.Printf("📊 结果: %v\n", result)
	}

	fmt.Println("\n💡 请手动检查:")
	fmt.Println("  1. chrome://extensions/ 页面是否打开")
	fmt.Println("  2. 是否看到插件列表")
	fmt.Println("  3. 开发者模式是否启用")

	fmt.Println("\n⏳ 保持打开30秒...")
	time.Sleep(30 * time.Second)

	fmt.Println("✅ 测试完成")
}