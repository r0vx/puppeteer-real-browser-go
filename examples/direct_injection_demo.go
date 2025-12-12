package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/chromedp/chromedp"
)

func main() {
	fmt.Println("🔧 直接Chrome API注入测试")
	fmt.Println("========================")

	// 创建Chrome上下文
	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.Flag("headless", false),
		chromedp.Flag("disable-blink-features", "AutomationControlled"),
	)

	allocCtx, cancel := chromedp.NewExecAllocator(context.Background(), opts...)
	defer cancel()

	ctx, cancel := chromedp.NewContext(allocCtx)
	defer cancel()

	// 启动Chrome并导航到测试页面
	err := chromedp.Run(ctx,
		chromedp.Navigate("https://httpbin.org/get"),
		chromedp.WaitReady("body"),
	)
	if err != nil {
		log.Fatalf("启动失败: %v", err)
	}

	fmt.Println("✅ Chrome启动成功")

	// 直接注入Chrome API
	fmt.Println("\n💉 直接注入Chrome扩展API...")
	
	apiScript := `
		// 创建Chrome对象
		if (!window.chrome) window.chrome = {};
		
		// Runtime API
		window.chrome.runtime = {
			id: 'injected-test-extension-direct',
			injected: true,
			sendMessage: function(message, callback) {
				console.log('Extension runtime message:', message);
				if (callback) setTimeout(() => callback({received: true}), 10);
			},
			onMessage: {
				addListener: function(callback) {
					window.addEventListener('chrome-extension-message', function(event) {
						callback(event.detail.message, event.detail.sender, event.detail.sendResponse);
					});
				}
			}
		};
		
		// Storage API
		window.chrome.storage = {
			local: {
				get: function(keys, callback) {
					const stored = JSON.parse(localStorage.getItem('chrome-extension-storage') || '{}');
					console.log('Storage get:', stored);
					if (callback) callback(stored);
				},
				set: function(items, callback) {
					const stored = JSON.parse(localStorage.getItem('chrome-extension-storage') || '{}');
					Object.assign(stored, items);
					localStorage.setItem('chrome-extension-storage', JSON.stringify(stored));
					console.log('Storage set:', items);
					if (callback) callback();
				}
			}
		};
		
		// Tabs API
		window.chrome.tabs = {
			query: function(queryInfo, callback) {
				const tabs = [{
					id: 1,
					url: location.href,
					title: document.title,
					active: true,
					windowId: 1
				}];
				console.log('Tabs query:', tabs);
				if (callback) callback(tabs);
			}
		};
		
		console.log('✅ Chrome API注入完成:', window.chrome);
		true; // 返回成功标志
	`

	var injectionResult bool
	err = chromedp.Run(ctx, chromedp.Evaluate(apiScript, &injectionResult))
	if err != nil {
		fmt.Printf("❌ API注入失败: %v\n", err)
		return
	}

	if injectionResult {
		fmt.Println("✅ Chrome扩展API直接注入成功")
	} else {
		fmt.Println("⚠️  Chrome扩展API注入状态未知")
	}

	// 验证注入结果
	fmt.Println("\n🧪 验证API注入...")
	time.Sleep(1 * time.Second)

	var result map[string]interface{}
	err = chromedp.Run(ctx,
		chromedp.Evaluate(`({
			hasChrome: !!window.chrome,
			hasRuntime: !!(window.chrome && window.chrome.runtime),
			hasStorage: !!(window.chrome && window.chrome.storage),
			hasTabs: !!(window.chrome && window.chrome.tabs),
			extensionId: window.chrome && window.chrome.runtime && window.chrome.runtime.id,
			injected: window.chrome && window.chrome.runtime && window.chrome.runtime.injected
		})`, &result),
	)

	if err != nil {
		fmt.Printf("❌ 验证失败: %v\n", err)
		return
	}

	fmt.Println("📊 API验证结果:")
	allSuccessful := true
	for key, value := range result {
		status := "❌"
		if v, ok := value.(bool); ok && v {
			status = "✅"
		} else if value != nil && value != false && value != "" {
			status = "✅"
		} else {
			allSuccessful = false
		}
		fmt.Printf("  %s %s: %v\n", status, key, value)
	}

	if allSuccessful {
		fmt.Println("\n🎉 所有API注入成功！")
	} else {
		fmt.Println("\n⚠️  部分API需要进一步调试")
	}

	// 功能测试
	fmt.Println("\n🧪 功能测试:")

	// 测试Storage
	fmt.Println("  💾 测试Storage API...")
	err = chromedp.Run(ctx,
		chromedp.Evaluate(`
			window.chrome.storage.local.set({
				testData: 'Direct injection test',
				timestamp: Date.now()
			}, function() {
				console.log('Storage set complete');
			});
		`, nil),
	)
	if err == nil {
		fmt.Println("     ✅ Storage写入测试成功")
	}

	// 测试Tabs
	fmt.Println("  🗂️  测试Tabs API...")
	var tabsCount int
	err = chromedp.Run(ctx,
		chromedp.Evaluate(`
			new Promise((resolve) => {
				window.chrome.tabs.query({active: true}, function(tabs) {
					resolve(tabs.length);
				});
			})
		`, &tabsCount),
	)
	if err == nil && tabsCount > 0 {
		fmt.Printf("     ✅ Tabs查询成功，返回 %d 个标签页\n", tabsCount)
	}

	// 在控制台显示结果
	chromedp.Run(ctx, chromedp.Evaluate(`
		console.log('%c🎯 Chrome扩展API直接注入测试完成', 'color: green; font-size: 16px; font-weight: bold;');
		console.log('Chrome对象:', window.chrome);
		console.log('Runtime API:', window.chrome.runtime);
		console.log('Storage API:', window.chrome.storage);
		console.log('Tabs API:', window.chrome.tabs);
	`, nil))

	fmt.Println("\n⏳ 保持浏览器打开30秒...")
	fmt.Println("请在浏览器控制台中检查注入的API对象")
	time.Sleep(30 * time.Second)

	fmt.Println("\n✅ 直接注入测试完成")
}