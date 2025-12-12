package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/HNRow/puppeteer-real-browser-go/pkg/browser"
)

func main() {
	fmt.Println("🚀 自动加载默认扩展演示")
	fmt.Println("=====================")

	ctx := context.Background()

	// 简单配置 - 启用自动加载默认扩展
	opts := &browser.ConnectOptions{
		Headless:                  false,
		AutoLoadDefaultExtensions: true,                     // 🔑 关键设置：自动加载默认扩展
		PersistProfile:            true,                     // 启用持久化配置文件
		ProfileName:               "auto_with_extensionscs", // 配置文件名
		Args: []string{
			"--start-maximized",
			"--enable-extensions",
			//"--auto-open-devtools=false",
			//"--exclude-switches=enable-automation",
		},
	}

	fmt.Println("📦 自动加载以下扩展 (使用未打包扩展目录):")
	fmt.Println("  • Discord Token Login (未打包目录)")
	fmt.Println("  • OKX Wallet (未打包目录)")

	fmt.Println("\n🔧 启动浏览器...")
	instance, err := browser.Connect(ctx, opts)
	if err != nil {
		log.Fatalf("浏览器连接失败: %v", err)
	}
	defer instance.Close()

	fmt.Println("✅ 浏览器启动成功")

	// 直接导航到扩展页面查看结果
	page := instance.Page()
	if err := page.Navigate("chrome://extensions/"); err != nil {
		log.Printf("导航失败: %v", err)
	}

	fmt.Println("📋 扩展管理页面已打开")
	fmt.Println("\n💡 自动验证:")
	fmt.Println("  1. 扩展已自动加载，无需手动操作")
	fmt.Println("  2. 在扩展管理页面应该能看到2个插件")
	fmt.Println("  3. 浏览器工具栏会显示插件图标")

	// 等待几秒让用户查看
	time.Sleep(5 * time.Second)

	// 测试Discord插件页面
	fmt.Println("\n🎮 测试Discord插件功能...")
	discordContext, err := instance.CreateBrowserContext(nil)
	if err == nil {
		discordPage, err := discordContext.NewPage()
		if err == nil {
			discordPage.Navigate("https://discord.com/login")
			fmt.Println("  ✅ Discord测试页面已打开")
		}
	}

	// 测试OKX钱包插件页面
	fmt.Println("\n💰 测试OKX钱包功能...")
	walletContext, err := instance.CreateBrowserContext(nil)
	if err == nil {
		walletPage, err := walletContext.NewPage()
		if err == nil {
			walletPage.Navigate("https://app.uniswap.org/")
			fmt.Println("  ✅ Uniswap测试页面已打开")
		}
	}

	fmt.Println("\n🎉 未打包扩展加载特点:")
	fmt.Println("  ✅ 使用未打包的扩展目录")
	fmt.Println("  ✅ Chrome --load-extension 原生支持")
	fmt.Println("  ✅ 开发模式扩展加载方式")
	fmt.Println("  ✅ 文件权限正确设置")
	fmt.Println("  ✅ 每次启动自动可用")

	fmt.Println("\n⏳ 保持浏览器开启30秒供测试...")
	time.Sleep(30 * time.Second)

	fmt.Println("✅ 自动加载扩展演示完成！")
}
