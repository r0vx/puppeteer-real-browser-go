package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/HNRow/puppeteer-real-browser-go/pkg/browser"
)

func main() {
	fmt.Println("🧪 新用户扩展测试")
	fmt.Println("================")

	ctx := context.Background()

	// 创建新用户配置
	opts := &browser.ConnectOptions{
		Headless:                  false,
		AutoLoadDefaultExtensions: true, // 自动加载默认扩展
		PersistProfile:            true,
		ProfileName:               "new_user_test", // 全新的配置文件
	}

	fmt.Println("👤 新用户: new_user_test")
	fmt.Println("📦 应该自动加载:")
	fmt.Println("  • Discord Token Login")
	fmt.Println("  • OKX Wallet")

	fmt.Println("\n🔧 启动浏览器...")
	instance, err := browser.Connect(ctx, opts)
	if err != nil {
		log.Fatalf("浏览器连接失败: %v", err)
	}
	defer instance.Close()

	fmt.Println("✅ 浏览器启动成功")

	// 直接导航到扩展页面
	page := instance.Page()
	if err := page.Navigate("chrome://extensions/"); err != nil {
		log.Printf("导航失败: %v", err)
	} else {
		fmt.Println("📋 扩展管理页面已打开")
	}

	// 等待页面加载
	time.Sleep(3 * time.Second)

	// 测试扩展功能
	fmt.Println("\n🔍 测试扩展功能...")
	result, err := page.Evaluate(`
		// 检查扩展是否工作
		{
			hasChrome: !!window.chrome,
			hasRuntime: !!(window.chrome && window.chrome.runtime),
			extensionPageLoaded: location.href.includes('chrome://extensions'),
			timestamp: Date.now()
		}
	`)
	
	if err != nil {
		fmt.Printf("❌ 功能测试失败: %v\n", err)
	} else {
		fmt.Printf("📊 功能测试结果: %v\n", result)
	}

	// 测试Discord页面
	fmt.Println("\n🎮 测试Discord插件...")
	discordContext, err := instance.CreateBrowserContext(nil)
	if err == nil {
		discordPage, err := discordContext.NewPage()
		if err == nil {
			discordPage.Navigate("https://discord.com/login")
			time.Sleep(2 * time.Second)
			fmt.Println("  ✅ Discord测试页面已打开")
		}
	}

	// 测试Web3页面
	fmt.Println("\n💰 测试OKX钱包插件...")
	web3Context, err := instance.CreateBrowserContext(nil)
	if err == nil {
		web3Page, err := web3Context.NewPage()
		if err == nil {
			web3Page.Navigate("https://app.uniswap.org/")
			time.Sleep(2 * time.Second)
			fmt.Println("  ✅ Uniswap测试页面已打开")
		}
	}

	fmt.Println("\n💡 验证步骤:")
	fmt.Println("  1. 查看chrome://extensions/页面是否显示插件")
	fmt.Println("  2. 检查浏览器工具栏是否有插件图标")
	fmt.Println("  3. 在Discord页面测试插件功能")
	fmt.Println("  4. 在Uniswap页面测试钱包连接")

	fmt.Println("\n⏳ 保持浏览器开启20秒供测试...")
	time.Sleep(20 * time.Second)

	fmt.Println("✅ 新用户测试完成！")
}