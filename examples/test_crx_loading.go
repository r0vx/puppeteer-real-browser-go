package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/HNRow/puppeteer-real-browser-go/pkg/browser"
)

func main() {
	fmt.Println("🚀 测试CRX文件自动加载")
	fmt.Println("=====================")

	ctx := context.Background()

	// 使用唯一的用户名
	profileName := "crx_test_" + fmt.Sprintf("%d", time.Now().Unix())
	opts := &browser.ConnectOptions{
		Headless:                  false,
		AutoLoadDefaultExtensions: true,  // 现在将加载.crx文件
		PersistProfile:            true,
		ProfileName:               profileName,
	}

	fmt.Printf("👤 测试用户: %s\n", profileName)
	fmt.Println("📦 自动加载CRX扩展:")
	fmt.Println("  • Discord Token Login (1.0_0.crx)")
	fmt.Println("  • OKX Wallet (3.66.10_0.crx)")

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
	time.Sleep(3 * time.Second)

	fmt.Println("\n💡 请检查:")
	fmt.Println("  1. chrome://extensions/ 页面是否显示插件")
	fmt.Println("  2. 浏览器工具栏是否有插件图标")
	fmt.Println("  3. 插件是否能正常工作")

	// 测试Discord插件
	fmt.Println("\n🎮 测试Discord插件功能...")
	discordContext, err := instance.CreateBrowserContext(nil)
	if err == nil {
		discordPage, err := discordContext.NewPage()
		if err == nil {
			discordPage.Navigate("https://discord.com/login")
			fmt.Println("  ✅ Discord测试页面已打开")
			time.Sleep(2 * time.Second)
		}
	}

	// 测试OKX钱包插件
	fmt.Println("\n💰 测试OKX钱包功能...")
	walletContext, err := instance.CreateBrowserContext(nil)
	if err == nil {
		walletPage, err := walletContext.NewPage()
		if err == nil {
			walletPage.Navigate("https://app.uniswap.org/")
			fmt.Println("  ✅ Uniswap测试页面已打开")
			time.Sleep(2 * time.Second)
		}
	}

	fmt.Println("\n🎉 CRX文件加载优势:")
	fmt.Println("  ✅ 使用打包好的扩展文件")
	fmt.Println("  ✅ 避免文件权限问题")
	fmt.Println("  ✅ 更接近正式安装方式")
	fmt.Println("  ✅ 支持扩展签名验证")

	fmt.Println("\n⏳ 保持浏览器开启30秒供测试...")
	time.Sleep(30 * time.Second)

	fmt.Println("✅ CRX扩展加载测试完成！")
}