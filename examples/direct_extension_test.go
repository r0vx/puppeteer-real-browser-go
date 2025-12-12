package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/HNRow/puppeteer-real-browser-go/pkg/browser"
)

func main() {
	fmt.Println("🔍 直接扩展加载测试")
	fmt.Println("==================")

	ctx := context.Background()

	// 直接指定扩展路径 - 使用 --load-extension 方式
	extensionPaths := []string{
		"./path/Extensions/kfjglmgfjedhhcddpfgfogkahmenikan/1.0_0",
		"./path/Extensions/mcohilncbfahbmgdjkbpemcciiolgcge/3.66.10_0",
	}

	fmt.Println("📂 扩展路径:")
	for i, path := range extensionPaths {
		fmt.Printf("  %d. %s\n", i+1, path)
	}

	// 配置选项 - 关键是使用正确的标志
	options := &browser.ConnectOptions{
		Headless:       false,
		UseCustomCDP:   false, // 使用标准模式
		Turnstile:      true,
		Extensions:     extensionPaths, // 这会触发 --load-extension 和 --enable-extensions
		PersistProfile: false,          // 暂时不使用持久化，简化测试
		Args: []string{
			"--start-maximized",
			"--disable-blink-features=AutomationControlled",
			// 不要添加 --enable-extensions，会由 GetExtensionFlags 自动添加
		},
	}

	fmt.Println("\n🚀 创建浏览器实例...")
	instance, err := browser.Connect(ctx, options)
	if err != nil {
		log.Fatalf("创建浏览器实例失败: %v", err)
	}
	defer instance.Close()

	fmt.Println("✅ 浏览器启动成功")

	// 创建页面并导航到扩展管理页面
	page := instance.Page()
	fmt.Println("🔍 导航到扩展管理页面...")
	
	if err := page.Navigate("chrome://extensions/"); err != nil {
		log.Fatalf("无法导航到扩展页面: %v", err)
	}

	fmt.Println("⏳ 等待页面加载...")
	time.Sleep(5 * time.Second)

	// 检查扩展是否加载
	result, err := page.Evaluate(`
		// 等待页面完全加载
		setTimeout(() => {
			console.log('开始检查扩展...');
			
			// 获取所有扩展项
			const extensionItems = document.querySelectorAll('extensions-item');
			console.log('找到扩展项数量:', extensionItems.length);
			
			const extensions = Array.from(extensionItems).map(item => {
				const nameElement = item.shadowRoot?.querySelector('#name');
				const toggleElement = item.shadowRoot?.querySelector('cr-toggle');
				
				return {
					name: nameElement ? nameElement.textContent : 'Unknown',
					id: item.getAttribute('id') || 'Unknown',
					enabled: toggleElement ? toggleElement.checked : false
				};
			});
			
			console.log('扩展详情:', extensions);
			
			return {
				found: extensions.length > 0,
				count: extensions.length,
				extensions: extensions,
				pageTitle: document.title,
				developerMode: document.querySelector('#developerMode')?.checked || false
			};
		}, 2000);
		
		return { status: 'waiting' };
	`)

	if err != nil {
		log.Printf("检查扩展失败: %v", err)
	} else {
		fmt.Printf("📊 初始检查结果: %v\n", result)
	}

	// 等待更长时间让扩展完全加载
	fmt.Println("⏳ 等待扩展加载完成...")
	time.Sleep(8 * time.Second)

	// 再次检查
	finalResult, err := page.Evaluate(`
		const extensionItems = document.querySelectorAll('extensions-item');
		const extensions = Array.from(extensionItems).map(item => {
			const nameElement = item.shadowRoot?.querySelector('#name');
			const toggleElement = item.shadowRoot?.querySelector('cr-toggle');
			const idText = item.getAttribute('id') || '';
			
			return {
				name: nameElement ? nameElement.textContent.trim() : 'Unknown',
				id: idText,
				enabled: toggleElement ? toggleElement.checked : false,
				visible: item.offsetWidth > 0 && item.offsetHeight > 0
			};
		});
		
		return {
			totalExtensions: extensions.length,
			enabledExtensions: extensions.filter(ext => ext.enabled).length,
			extensions: extensions,
			developerMode: document.querySelector('#developerMode')?.checked || false,
			pageReady: true
		};
	`)

	if err != nil {
		log.Printf("最终检查失败: %v", err)
	} else {
		fmt.Printf("🎯 最终结果: %v\n", finalResult)
	}

	// 显示用法说明
	fmt.Println("\n💡 检查说明:")
	fmt.Println("  1. 查看浏览器窗口是否已打开")
	fmt.Println("  2. 当前应该显示 chrome://extensions/ 页面") 
	fmt.Println("  3. 如果看到扩展，说明加载成功")
	fmt.Println("  4. 如果没看到扩展，可能需要启用开发者模式")

	fmt.Println("\n⏳ 保持浏览器打开 30 秒供手动检查...")
	time.Sleep(30 * time.Second)

	fmt.Println("✅ 直接扩展加载测试完成!")
}