package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/HNRow/puppeteer-real-browser-go/pkg/browser"
)

func main() {
	fmt.Println("🔄 扩展持久性测试")
	fmt.Println("================")

	ctx := context.Background()

	// 使用唯一的用户名
	profileName := "persistence_test_" + fmt.Sprintf("%d", time.Now().Unix())
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

	// 等待5秒让Chrome稳定
	fmt.Println("⏳ 等待5秒让Chrome稳定...")
	time.Sleep(5 * time.Second)

	// 检查扩展目录
	userDataDir := "/Users/rowei/.puppeteer-real-browser-go/profiles/" + profileName
	extensionsDir := userDataDir + "/Default/Extensions"

	fmt.Println("\n🔍 检查扩展目录...")
	entries, err := os.ReadDir(extensionsDir)
	if err != nil {
		fmt.Printf("❌ 无法读取Extensions目录: %v\n", err)
	} else {
		fmt.Printf("✅ 发现 %d 个扩展:\n", len(entries))
		for _, entry := range entries {
			fmt.Printf("  - %s\n", entry.Name())
		}
	}

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

	// 再次检查扩展目录
	fmt.Println("\n🔍 再次检查扩展目录...")
	entries2, err := os.ReadDir(extensionsDir)
	if err != nil {
		fmt.Printf("❌ 无法读取Extensions目录: %v\n", err)
	} else {
		fmt.Printf("✅ 现在有 %d 个扩展:\n", len(entries2))
		for _, entry := range entries2 {
			fmt.Printf("  - %s\n", entry.Name())
		}
	}

	instance.Close()
	fmt.Println("\n✅ 持久性测试完成")
}
