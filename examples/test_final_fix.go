package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/HNRow/puppeteer-real-browser-go/pkg/browser"
)

func main() {
	fmt.Println("🔧 测试权限修复后的扩展加载")
	fmt.Println("============================")

	ctx := context.Background()

	// 使用唯一的用户名
	profileName := "final_fix_test_" + fmt.Sprintf("%d", time.Now().Unix())
	opts := &browser.ConnectOptions{
		Headless:                  false,
		AutoLoadDefaultExtensions: true,
		PersistProfile:            true,
		ProfileName:               profileName,
	}

	fmt.Printf("👤 测试用户: %s\n", profileName)

	fmt.Println("\n🔧 启动浏览器...")
	instance, err := browser.Connect(ctx, opts)
	if err != nil {
		log.Fatalf("浏览器连接失败: %v", err)
	}
	defer instance.Close()

	fmt.Println("✅ 浏览器启动成功")

	// 导航到扩展页面
	page := instance.Page()
	if err := page.Navigate("chrome://extensions/"); err != nil {
		log.Printf("导航失败: %v", err)
	} else {
		fmt.Println("📋 扩展管理页面已打开")
	}

	// 等待页面加载
	time.Sleep(2 * time.Second)

	// 尝试获取页面信息
	result, err := page.Evaluate(`
		// 检查扩展页面内容
		const extensionCards = document.querySelectorAll('extensions-item');
		const extensionCount = extensionCards.length;
		
		let extensions = [];
		extensionCards.forEach(card => {
			const nameElement = card.shadowRoot.querySelector('#name');
			if (nameElement) {
				extensions.push(nameElement.textContent.trim());
			}
		});
		
		return {
			extensionCount: extensionCount,
			extensions: extensions,
			url: location.href,
			title: document.title
		};
	`)
	
	if err != nil {
		fmt.Printf("❌ 无法评估页面: %v\n", err)
	} else {
		fmt.Printf("📊 页面信息: %v\n", result)
	}

	fmt.Println("\n💡 请手动检查:")
	fmt.Println("  1. chrome://extensions/ 页面是否显示插件")
	fmt.Println("  2. 浏览器工具栏是否有插件图标")
	fmt.Println("  3. 如果看到插件，说明权限修复成功!")

	fmt.Println("\n⏳ 保持浏览器开启30秒供检查...")
	time.Sleep(30 * time.Second)

	fmt.Println("✅ 测试完成")
}