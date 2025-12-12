package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/HNRow/puppeteer-real-browser-go/pkg/browser"
)

func main() {
	fmt.Println("🧪 简单扩展测试")
	fmt.Println("================")

	ctx := context.Background()

	// 扩展路径
	extensionPaths := []string{
		"../path/Extensions/kfjglmgfjedhhcddpfgfogkahmenikan/1.0_0",     // Discord Token Login
		"../path/Extensions/mcohilncbfahbmgdjkbpemcciiolgcge/3.66.10_0", // OKX Wallet
	}

	fmt.Println("📂 加载的扩展:")
	for i, path := range extensionPaths {
		fmt.Printf("  %d. %s\n", i+1, path)
	}

	// 配置选项 - 添加开发者模式相关标志
	options := &browser.ConnectOptions{
		Headless:       false,
		UseCustomCDP:   false,
		Turnstile:      false, // 暂时关闭，简化测试
		Extensions:     extensionPaths,
		PersistProfile: false, // 不使用持久化，简化测试
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

	fmt.Println("✅ 浏览器启动成功！")

	// 创建页面
	page := instance.Page()

	// 导航到扩展页面
	fmt.Println("🔍 导航到扩展管理页面...")
	if err := page.Navigate("chrome://extensions/"); err != nil {
		log.Fatalf("无法导航到扩展页面: %v", err)
	}

	fmt.Println("⏳ 等待页面加载...")
	time.Sleep(5 * time.Second)

	// 检查页面上的扩展
	fmt.Println("🔍 检查扩展...")
	result, err := page.Evaluate(`
		const items = document.querySelectorAll('extensions-item');
		const extensions = Array.from(items).map(item => {
			const shadow = item.shadowRoot;
			if (!shadow) return null;
			
			const name = shadow.querySelector('#name')?.textContent || 'Unknown';
			const id = item.getAttribute('id') || '';
			const toggle = shadow.querySelector('cr-toggle');
			const enabled = toggle ? toggle.checked : false;
			
			return { name: name.trim(), id, enabled };
		}).filter(ext => ext !== null);
		
		return {
			totalFound: items.length,
			extensions: extensions,
			developerMode: document.querySelector('#developerMode')?.checked || false
		};
	`)

	if err != nil {
		fmt.Printf("❌ 检查扩展失败: %v\n", err)
	} else {
		fmt.Printf("📊 扩展检查结果: %v\n", result)
	}

	fmt.Println("\n💡 手动检查:")
	fmt.Println("  1. 浏览器窗口应该已经打开")
	fmt.Println("  2. 当前显示 chrome://extensions/ 页面")
	fmt.Println("  3. 检查是否有扩展显示")

	fmt.Println("\n⏳ 保持浏览器打开 30 秒供手动检查...")
	time.Sleep(30 * time.Second)

	fmt.Println("✅ 测试完成!")
}