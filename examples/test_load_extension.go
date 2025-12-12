package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/HNRow/puppeteer-real-browser-go/pkg/browser"
)

func main() {
	fmt.Println("🔧 测试--load-extension标志")
	fmt.Println("=========================")

	ctx := context.Background()

	// 使用唯一的用户名
	profileName := "load_ext_test_" + fmt.Sprintf("%d", time.Now().Unix())
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

	fmt.Println("✅ 浏览器启动成功")

	// 导航到扩展页面
	page := instance.Page()
	if err := page.Navigate("chrome://extensions/"); err != nil {
		log.Printf("导航失败: %v", err)
	} else {
		fmt.Println("📋 扩展管理页面已打开")
	}

	fmt.Println("\n💡 请检查:")
	fmt.Println("  1. chrome://extensions/ 页面是否显示插件")
	fmt.Println("  2. 浏览器工具栏是否有插件图标")

	fmt.Println("\n⏳ 保持浏览器开启20秒...")
	time.Sleep(20 * time.Second)

	instance.Close()
	fmt.Println("\n✅ 测试完成")
}