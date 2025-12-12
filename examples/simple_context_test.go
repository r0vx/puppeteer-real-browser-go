package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/HNRow/puppeteer-real-browser-go/pkg/browser"
)

// SimpleContextTest demonstrates basic browser context usage
func main() {
	fmt.Println("🧪 Simple Browser Context Test")
	fmt.Println("==============================")

	ctx := context.Background()

	// 创建主浏览器实例
	opts := &browser.ConnectOptions{
		Headless:     false,
		UseCustomCDP: false, // 使用标准模式便于调试
		Args: []string{
			"--start-maximized",
		},
	}

	fmt.Println("🚀 Starting main browser...")
	mainBrowser, err := browser.Connect(ctx, opts)
	if err != nil {
		log.Fatalf("Failed to connect browser: %v", err)
	}
	defer mainBrowser.Close()

	// 测试1: 创建多个上下文
	fmt.Println("\n📋 Test 1: Creating Multiple Contexts")
	context1, err := mainBrowser.CreateBrowserContext(nil)
	if err != nil {
		log.Fatalf("Failed to create context 1: %v", err)
	}
	defer context1.Close()

	context2, err := mainBrowser.CreateBrowserContext(nil)
	if err != nil {
		log.Fatalf("Failed to create context 2: %v", err)
	}
	defer context2.Close()

	fmt.Println("  ✅ Created 2 browser contexts")

	// 测试2: 在每个上下文中创建页面
	fmt.Println("\n📄 Test 2: Creating Pages in Each Context")
	
	page1, err := context1.NewPage()
	if err != nil {
		log.Fatalf("Failed to create page in context 1: %v", err)
	}
	
	page2, err := context2.NewPage()
	if err != nil {
		log.Fatalf("Failed to create page in context 2: %v", err)
	}

	fmt.Println("  ✅ Created pages in both contexts")

	// 测试3: 导航到不同网站
	fmt.Println("\n🌐 Test 3: Navigating to Different Sites")
	
	if err := page1.Navigate("https://httpbin.org/cookies/set/context/page1"); err != nil {
		log.Printf("Failed to navigate page 1: %v", err)
	} else {
		// 设置页面标题便于识别
		page1.Evaluate(`document.title = 'Context 1 - Page 1'`)
		fmt.Println("  ✅ Page 1: Set cookie for context 1")
	}

	if err := page2.Navigate("https://httpbin.org/cookies/set/context/page2"); err != nil {
		log.Printf("Failed to navigate page 2: %v", err)
	} else {
		// 设置页面标题便于识别
		page2.Evaluate(`document.title = 'Context 2 - Page 2'`)
		fmt.Println("  ✅ Page 2: Set cookie for context 2")
	}

	time.Sleep(3 * time.Second)

	// 测试4: 验证 Cookie 隔离
	fmt.Println("\n🍪 Test 4: Verifying Cookie Isolation")
	
	// Context 1 检查 Cookie
	page1Check, err := context1.NewPage()
	if err == nil {
		if err := page1Check.Navigate("https://httpbin.org/cookies"); err == nil {
			page1Check.Evaluate(`document.title = 'Context 1 - Cookie Check'`)
			fmt.Println("  ✅ Context 1: Cookie check page loaded")
		}
	}

	// Context 2 检查 Cookie  
	page2Check, err := context2.NewPage()
	if err == nil {
		if err := page2Check.Navigate("https://httpbin.org/cookies"); err == nil {
			page2Check.Evaluate(`document.title = 'Context 2 - Cookie Check'`)
			fmt.Println("  ✅ Context 2: Cookie check page loaded")
		}
	}

	// 测试5: 在同一上下文创建多个页面
	fmt.Println("\n📑 Test 5: Multiple Pages in Same Context")
	
	page1_2, err := context1.NewPage()
	if err == nil {
		if err := page1_2.Navigate("https://httpbin.org/user-agent"); err == nil {
			page1_2.Evaluate(`document.title = 'Context 1 - Page 2 (Shared Cookies)'`)
			fmt.Println("  ✅ Context 1: Created second page (shares cookies)")
		}
	}

	// 显示结果
	fmt.Println("\n📊 Test Results:")
	fmt.Println("  🔍 Check the browser windows:")
	fmt.Println("    - Context 1 pages should share cookies with each other")
	fmt.Println("    - Context 2 pages should have different cookies")
	fmt.Println("    - Each context is completely isolated")
	fmt.Println("    - All contexts share the same Chrome process")

	fmt.Println("\n💡 Manual Verification:")
	fmt.Println("  1. Look at the browser window titles")
	fmt.Println("  2. Check DevTools > Application > Cookies")
	fmt.Println("  3. Verify each context has different cookies")

	fmt.Println("\n⏳ Keeping browsers open for 20 seconds for inspection...")
	time.Sleep(20 * time.Second)

	fmt.Println("✅ Simple context test completed!")
}