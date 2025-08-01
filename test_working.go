package main

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/HNRow/puppeteer-real-browser-go/pkg/browser"
)

func main() {
	fmt.Println("🚀 工作测试 - 修复版本")
	fmt.Println("======================")

	ctx := context.Background()

	// 稳定的配置
	opts := &browser.ConnectOptions{
		Headless:  false,
		Turnstile: true,
		Args: []string{
			"--start-maximized",
			"--disable-blink-features=AutomationControlled",
			"--exclude-switches=enable-automation",
			"--no-sandbox",
			"--disable-dev-shm-usage",
		},
	}

	fmt.Println("📱 启动浏览器...")
	instance, err := browser.Connect(ctx, opts)
	if err != nil {
		log.Fatalf("❌ 连接失败: %v", err)
	}
	defer instance.Close()

	page := instance.Page()
	fmt.Println("✅ 浏览器启动成功!")

	// 测试反检测功能
	fmt.Println("\n🛡️ 测试反检测功能")
	testAntiDetection(page)

	// 测试 Cloudflare 绕过
	fmt.Println("\n☁️ 测试 Cloudflare 绕过")
	testCloudflareBypass(page)

	fmt.Println("\n⏱️ 浏览器将保持打开 15 秒...")
	time.Sleep(15 * time.Second)

	fmt.Println("✅ 测试完成!")
}

func testAntiDetection(page browser.Page) {
	script := `({
		webdriver: navigator.webdriver,
		userAgent: navigator.userAgent.includes('HeadlessChrome'),
		plugins: navigator.plugins.length,
		languages: navigator.languages.length,
		hardwareConcurrency: navigator.hardwareConcurrency,
		chrome: typeof window.chrome !== 'undefined'
	})`

	result, err := page.Evaluate(script)
	if err != nil {
		fmt.Printf("❌ 反检测测试失败: %v\n", err)
		return
	}

	fmt.Printf("📊 反检测状态: %+v\n", result)
}

func testCloudflareBypass(page browser.Page) {
	// 先预热
	fmt.Println("🔥 预热浏览器...")
	err := page.Navigate("https://www.google.com")
	if err != nil {
		fmt.Printf("⚠️ 预热失败: %v\n", err)
	} else {
		fmt.Println("✅ 预热完成")
		time.Sleep(2 * time.Second)
	}

	// 测试 Cloudflare 保护站点
	fmt.Println("🎯 访问 Irys.xyz...")
	err = page.Navigate("https://irys.xyz/faucet")
	if err != nil {
		fmt.Printf("❌ 导航失败: %v\n", err)
		return
	}

	// 等待并检查结果
	for i := 1; i <= 8; i++ {
		fmt.Printf("⏳ 检查页面状态... (%d/8)\n", i)
		time.Sleep(2 * time.Second)
		
		title, err := page.GetTitle()
		if err != nil {
			continue
		}
		
		fmt.Printf("📄 当前标题: %s\n", title)
		
		if isSuccess(title) {
			fmt.Println("🎉 成功绕过 Cloudflare 保护!")
			return
		}
		
		if isBlocked(title) {
			fmt.Println("🚫 被 Cloudflare 阻止")
			return
		}
	}
	
	fmt.Println("❓ 页面状态未确定")
}

func isSuccess(title string) bool {
	successIndicators := []string{
		"irys",
		"faucet",
		"testnet",
	}
	
	titleLower := strings.ToLower(title)
	for _, indicator := range successIndicators {
		if strings.Contains(titleLower, indicator) {
			return true
		}
	}
	return false
}

func isBlocked(title string) bool {
	blockIndicators := []string{
		"just a moment",
		"checking your browser",
		"cloudflare",
		"please wait",
		"verifying you are human",
		"security check",
	}
	
	titleLower := strings.ToLower(title)
	for _, indicator := range blockIndicators {
		if strings.Contains(titleLower, indicator) {
			return true
		}
	}
	return false
}
