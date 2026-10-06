package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/r0vx/puppeteer-real-browser-go/pkg/browser"
)

func main() {
	fmt.Println("🎯 Custom CDP 模式使用演示")
	fmt.Println("=" + string(make([]byte, 60)))
	fmt.Println()
	fmt.Println("默认 CustomCDP 通道 - 完全避免 Runtime.Enable")
	fmt.Println()

	ctx := context.Background()

	// 使用 Custom CDP 模式（最强反检测）
	opts := &browser.ConnectOptions{
		Headless: false,
		Args: []string{
			"--disable-session-crashed-bubble",
			"--disable-infobars",
			"--no-first-run",
			"--no-default-browser-check",
		},
	}

	fmt.Println("🚀 启动浏览器（Custom CDP模式）...")
	instance, err := browser.Connect(ctx, opts)
	if err != nil {
		log.Fatalf("❌ 启动失败: %v", err)
	}
	defer instance.Close()

	page := instance.Page()

	// 示例1：基于坐标点击
	fmt.Println("\n📍 示例1：基于坐标点击")
	fmt.Println("-" + string(make([]byte, 40)))

	if err := page.Navigate("https://www.google.com"); err != nil {
		log.Fatalf("❌ 导航失败: %v", err)
	}
	time.Sleep(2 * time.Second)

	// 直接点击坐标（不需要Runtime.Enable）
	fmt.Println("   点击搜索框 (x: 400, y: 300)")
	page.Click(400, 300)

	// 示例2：使用辅助函数点击选择器
	fmt.Println("\n🎯 示例2：使用选择器点击（通过辅助函数）")
	fmt.Println("-" + string(make([]byte, 40)))

	// 使用辅助函数点击选择器
	fmt.Println("   点击搜索框 (使用选择器)")
	if err := browser.ClickSelector(page, "textarea[name='q']"); err != nil {
		fmt.Printf("   ⚠️  点击失败: %v\n", err)
	} else {
		fmt.Println("   ✅ 点击成功")
	}

	// 示例3：输入文本
	fmt.Println("\n⌨️  示例3：输入文本")
	fmt.Println("-" + string(make([]byte, 40)))

	searchText := "puppeteer anti-detection"
	fmt.Printf("   输入搜索词: %s\n", searchText)
	if err := browser.TypeText(page, "textarea[name='q']", searchText); err != nil {
		fmt.Printf("   ⚠️  输入失败: %v\n", err)
	} else {
		fmt.Println("   ✅ 输入成功")
	}

	time.Sleep(2 * time.Second)

	// 示例4：获取元素文本
	fmt.Println("\n📄 示例4：获取元素文本")
	fmt.Println("-" + string(make([]byte, 40)))

	title, err := page.GetTitle()
	if err != nil {
		fmt.Printf("   ⚠️  获取标题失败: %v\n", err)
	} else {
		fmt.Printf("   ✅ 页面标题: %s\n", title)
	}

	// 示例5：检查元素是否可见
	fmt.Println("\n👁️  示例5：检查元素可见性")
	fmt.Println("-" + string(make([]byte, 40)))

	visible, err := browser.IsElementVisible(page, "textarea[name='q']")
	if err != nil {
		fmt.Printf("   ⚠️  检查失败: %v\n", err)
	} else {
		fmt.Printf("   ✅ 搜索框可见: %v\n", visible)
	}

	// 示例6：访问反检测测试网站
	fmt.Println("\n🔍 示例6：访问反检测测试网站")
	fmt.Println("-" + string(make([]byte, 40)))

	testURL := "https://abrahamjuliot.github.io/creepjs/"
	fmt.Printf("   访问: %s\n", testURL)

	if err := page.Navigate(testURL); err != nil {
		log.Printf("⚠️  导航失败: %v", err)
	} else {
		fmt.Println("   ✅ 页面加载成功")
		time.Sleep(5 * time.Second)

		// 检查反检测效果
		checkAntiDetection(page)
	}

	fmt.Println("\n⏳ 保持浏览器打开30秒供您检查...")
	fmt.Println()
	fmt.Println("💡 手动检查要点：")
	fmt.Println("   1. 打开DevTools (F12)")
	fmt.Println("   2. 检查 Console 是否有 'Runtime.enable' 消息")
	fmt.Println("   3. 在Console输入: navigator.webdriver")
	fmt.Println("      应该返回: undefined")
	fmt.Println("   4. 查看Trust Score或检测结果")
	fmt.Println()

	time.Sleep(30 * time.Second)

	fmt.Println("✅ 演示完成！")
}

func checkAntiDetection(page browser.Page) {
	fmt.Println("\n   🔍 反检测检查:")

	// 检查 navigator.webdriver
	webdriver, err := page.Evaluate("navigator.webdriver")
	if err == nil {
		if webdriver == nil {
			fmt.Println("      ✅ navigator.webdriver = undefined (已隐藏)")
		} else {
			fmt.Printf("      ⚠️  navigator.webdriver = %v (暴露了！)\n", webdriver)
		}
	}

	// 检查 User-Agent
	ua, err := page.Evaluate("navigator.userAgent")
	if err == nil {
		if uaStr, ok := ua.(string); ok && len(uaStr) > 0 {
			fmt.Printf("      ✅ User-Agent: %s...\n", uaStr[:min(50, len(uaStr))])
		}
	}

	// 检查 Plugins
	pluginCount, err := page.Evaluate("navigator.plugins.length")
	if err == nil {
		fmt.Printf("      ✅ Plugins Count: %v\n", pluginCount)
	}

	// 检查 Languages
	langs, err := page.Evaluate("navigator.languages")
	if err == nil {
		fmt.Printf("      ✅ Languages: %v\n", langs)
	}

	// 检查 Chrome对象
	hasChrome, err := page.Evaluate("typeof window.chrome !== 'undefined'")
	if err == nil && hasChrome == true {
		fmt.Println("      ✅ window.chrome 对象存在")
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
