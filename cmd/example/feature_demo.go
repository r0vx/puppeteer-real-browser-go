//go:build ignore
// +build ignore

package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/r0vx/puppeteer-real-browser-go/pkg/browser"
)

func main() {
	fmt.Println("🧪 在抖音充值页面测试新增功能")
	fmt.Println("================================")

	ctx := context.Background()

	opts := &browser.ConnectOptions{
		Headless:     false,
		UseCustomCDP: false,
		Args: []string{
			"--window-size=1920,1080",
		},
	}

	fmt.Println("🚀 启动浏览器...")
	instance, err := browser.Connect(ctx, opts)
	if err != nil {
		log.Fatalf("❌ 连接失败: %v", err)
	}
	defer instance.Close()

	page := instance.Page()

	// 类型断言获取扩展方法
	selectorPage, ok := page.(browser.PageWithSelector)
	if !ok {
		log.Fatal("❌ 无法获取 PageWithSelector")
	}

	// ==================== 测试 Navigate ====================
	fmt.Println("\n📌 测试 1: Navigate 到抖音充值页面")
	if err := page.Navigate("https://www.douyin.com/pay"); err != nil {
		fmt.Printf("   ❌ Navigate 失败: %v\n", err)
	} else {
		fmt.Println("   ✅ Navigate 成功")
	}
	time.Sleep(3 * time.Second)

	// ==================== 测试 GetTitle ====================
	fmt.Println("\n📌 测试 2: GetTitle")
	title, err := page.GetTitle()
	if err != nil {
		fmt.Printf("   ❌ GetTitle 失败: %v\n", err)
	} else {
		fmt.Printf("   ✅ GetTitle 成功: %s\n", title)
	}

	// ==================== 测试 GetURL ====================
	fmt.Println("\n📌 测试 3: GetURL")
	url, err := page.GetURL()
	if err != nil {
		fmt.Printf("   ❌ GetURL 失败: %v\n", err)
	} else {
		fmt.Printf("   ✅ GetURL 成功: %s\n", url)
	}

	// ==================== 测试 WaitVisible (等待充值按钮) ====================
	fmt.Println("\n📌 测试 4: WaitVisible (等待页面元素)")
	
	// 先调试看看元素状态
	debugResult, _ := page.Evaluate(`
		(function() {
			const elem = document.querySelector('#root');
			if (!elem) return {exists: false};
			const rect = elem.getBoundingClientRect();
			const style = window.getComputedStyle(elem);
			return {
				exists: true,
				width: rect.width,
				height: rect.height,
				visibility: style.visibility,
				display: style.display,
				opacity: style.opacity
			};
		})()
	`)
	fmt.Printf("   调试 #root 状态: %+v\n", debugResult)
	
	// 抖音页面用 div 模拟按钮，等待 #root 容器
	if err := selectorPage.WaitVisible("#root", 10*time.Second); err != nil {
		fmt.Printf("   ❌ WaitVisible 失败: %v\n", err)
	} else {
		fmt.Println("   ✅ WaitVisible 成功: 找到 #root 元素")
	}

	// ==================== 测试 Has ====================
	fmt.Println("\n📌 测试 5: Has (检查元素是否存在)")
	has, err := selectorPage.Has("#root")
	if err != nil {
		fmt.Printf("   ❌ Has 失败: %v\n", err)
	} else {
		fmt.Printf("   ✅ Has 成功: #root 存在 = %v\n", has)
	}

	// ==================== 测试 Screenshot ====================
	fmt.Println("\n📌 测试 6: Screenshot (全页截图)")
	screenshot, err := page.Screenshot()
	if err != nil {
		fmt.Printf("   ❌ Screenshot 失败: %v\n", err)
	} else {
		fmt.Printf("   ✅ Screenshot 成功: %d bytes\n", len(screenshot))
	}

	// ==================== 测试 ScreenshotElement ====================
	fmt.Println("\n📌 测试 7: ScreenshotElement (元素截图)")
	// 截图充值区域
	screenshotSelector := "#root > div > div > div.recharge_pc_body-gYwEj0 > div > div.recharge_container_body-eKxEpe"
	elemScreenshot, err := selectorPage.ScreenshotElement(screenshotSelector)
	if err != nil {
		fmt.Printf("   ⚠️ ScreenshotElement: %v\n", err)
	} else {
		fmt.Printf("   ✅ ScreenshotElement 成功: %d bytes\n", len(elemScreenshot))
	}

	// ==================== 测试 GetCookies ====================
	fmt.Println("\n📌 测试 8: GetCookies")
	cookies, err := selectorPage.GetCookies()
	if err != nil {
		fmt.Printf("   ❌ GetCookies 失败: %v\n", err)
	} else {
		cookiePreview := cookies
		if len(cookiePreview) > 100 {
			cookiePreview = cookiePreview[:100] + "..."
		}
		fmt.Printf("   ✅ GetCookies 成功: %s\n", cookiePreview)
	}

	// ==================== 测试 SetLocalStorage / GetLocalStorage ====================
	fmt.Println("\n📌 测试 9: LocalStorage")
	if err := selectorPage.SetLocalStorage(`{"test_key": "test_value_123"}`); err != nil {
		fmt.Printf("   ❌ SetLocalStorage 失败: %v\n", err)
	} else {
		fmt.Println("   ✅ SetLocalStorage 成功")
	}

	localStorage, err := selectorPage.GetLocalStorage()
	if err != nil {
		fmt.Printf("   ❌ GetLocalStorage 失败: %v\n", err)
	} else {
		localStoragePreview := localStorage
		if len(localStoragePreview) > 200 {
			localStoragePreview = localStoragePreview[:200] + "..."
		}
		fmt.Printf("   ✅ GetLocalStorage 成功: %s\n", localStoragePreview)
	}

	// ==================== 测试 SetSessionStorage / GetSessionStorage ====================
	fmt.Println("\n📌 测试 10: SessionStorage")
	if err := selectorPage.SetSessionStorage(`{"session_test": "session_value"}`); err != nil {
		fmt.Printf("   ❌ SetSessionStorage 失败: %v\n", err)
	} else {
		fmt.Println("   ✅ SetSessionStorage 成功")
	}

	sessionStorage, err := selectorPage.GetSessionStorage()
	if err != nil {
		fmt.Printf("   ❌ GetSessionStorage 失败: %v\n", err)
	} else {
		sessionStoragePreview := sessionStorage
		if len(sessionStoragePreview) > 200 {
			sessionStoragePreview = sessionStoragePreview[:200] + "..."
		}
		fmt.Printf("   ✅ GetSessionStorage 成功: %s\n", sessionStoragePreview)
	}

	// ==================== 测试 ExecuteJS ====================
	fmt.Println("\n📌 测试 11: ExecuteJS")
	var jsResult interface{}
	if err := selectorPage.ExecuteJS("document.querySelectorAll('button').length", &jsResult); err != nil {
		fmt.Printf("   ❌ ExecuteJS 失败: %v\n", err)
	} else {
		fmt.Printf("   ✅ ExecuteJS 成功: 页面有 %v 个按钮\n", jsResult)
	}

	fmt.Println("\n================================")
	fmt.Println("🎉 测试完成!")
	fmt.Println("⏳ 浏览器保持 3 秒...")
	time.Sleep(3 * time.Second)
}
