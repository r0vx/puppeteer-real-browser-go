package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/HNRow/puppeteer-real-browser-go/pkg/browser"
)

func main() {
	fmt.Println("🔍 对比手动vs自动加载扩展")
	fmt.Println("==========================")

	ctx := context.Background()

	fmt.Println("\n=== 测试1: 手动指定扩展路径 ===")
	
	// 测试1: 手动指定扩展路径 (模拟成功的方式)
	opts1 := &browser.ConnectOptions{
		Headless:       false,
		PersistProfile: true,
		ProfileName:    "manual_test_" + fmt.Sprintf("%d", time.Now().Unix()),
		Extensions: []string{
			"examples/path/Extensions/kfjglmgfjedhhcddpfgfogkahmenikan/1.0_0",
			"examples/path/Extensions/mcohilncbfahbmgdjkbpemcciiolgcge/3.66.10_0",
		},
	}

	fmt.Printf("👤 手动测试用户: %s\n", opts1.ProfileName)
	fmt.Println("📦 手动扩展路径:")
	for i, ext := range opts1.Extensions {
		fmt.Printf("  %d. %s\n", i+1, ext)
	}

	instance1, err := browser.Connect(ctx, opts1)
	if err != nil {
		log.Printf("手动加载失败: %v", err)
	} else {
		fmt.Println("✅ 手动加载Chrome启动成功")
		
		page1 := instance1.Page()
		if err := page1.Navigate("chrome://extensions/"); err != nil {
			log.Printf("导航失败: %v", err)
		} else {
			fmt.Println("📋 扩展管理页面已打开")
		}
		
		fmt.Println("⏳ 检查手动加载的扩展 (10秒)...")
		time.Sleep(10 * time.Second)
		
		instance1.Close()
	}

	fmt.Println("\n=== 测试2: 自动加载扩展 ===")
	
	// 测试2: 自动加载扩展 (当前方式)
	opts2 := &browser.ConnectOptions{
		Headless:                  false,
		AutoLoadDefaultExtensions: true,
		PersistProfile:            true,
		ProfileName:               "auto_test_" + fmt.Sprintf("%d", time.Now().Unix()),
	}

	fmt.Printf("👤 自动测试用户: %s\n", opts2.ProfileName)
	fmt.Println("🔄 使用AutoLoadDefaultExtensions: true")

	instance2, err := browser.Connect(ctx, opts2)
	if err != nil {
		log.Printf("自动加载失败: %v", err)
	} else {
		fmt.Println("✅ 自动加载Chrome启动成功")
		
		page2 := instance2.Page()
		if err := page2.Navigate("chrome://extensions/"); err != nil {
			log.Printf("导航失败: %v", err)
		} else {
			fmt.Println("📋 扩展管理页面已打开")
		}
		
		fmt.Println("⏳ 检查自动加载的扩展 (10秒)...")
		time.Sleep(10 * time.Second)
		
		instance2.Close()
	}

	fmt.Println("\n✅ 对比测试完成")
	fmt.Println("💡 请观察两次测试中chrome://extensions页面的差异")
}