package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/HNRow/puppeteer-real-browser-go/pkg/browser"
)

func main() {
	fmt.Println("🆕 全新用户扩展测试")
	fmt.Println("=================")

	ctx := context.Background()

	// 使用一个全新的用户名
	opts := &browser.ConnectOptions{
		Headless:                  false,
		AutoLoadDefaultExtensions: true,
		PersistProfile:            true,
		ProfileName:               "fresh_user_" + fmt.Sprintf("%d", time.Now().Unix()), // 确保唯一
	}

	fmt.Printf("👤 新用户: %s\n", opts.ProfileName)

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
		return
	}

	fmt.Println("📋 扩展管理页面已打开")
	fmt.Println("\n💡 请手动检查:")
	fmt.Println("  1. chrome://extensions/ 页面是否显示插件")
	fmt.Println("  2. 浏览器工具栏是否有插件图标")
	fmt.Println("  3. 如果看到插件，说明修复成功!")

	fmt.Println("\n⏳ 保持浏览器开启30秒供检查...")
	time.Sleep(30 * time.Second)

	fmt.Println("✅ 测试完成")
}