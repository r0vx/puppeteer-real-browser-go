package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/HNRow/puppeteer-real-browser-go/pkg/browser"
)

func main() {
	fmt.Println("🧪 绝对路径扩展测试")
	fmt.Println("==================")

	ctx := context.Background()

	// 获取当前工作目录
	wd, err := os.Getwd()
	if err != nil {
		log.Fatalf("无法获取工作目录: %v", err)
	}

	// 构建绝对路径
	ext1Path := filepath.Join(wd, "../examples/path/Extensions/kfjglmgfjedhhcddpfgfogkahmenikan/1.0_0")
	ext2Path := filepath.Join(wd, "../examples/path/Extensions/mcohilncbfahbmgdjkbpemcciiolgcge/3.66.10_0")

	extensionPaths := []string{ext1Path, ext2Path}

	fmt.Println("📂 使用绝对路径:")
	for i, path := range extensionPaths {
		fmt.Printf("  %d. %s\n", i+1, path)
		
		// 验证路径是否存在
		if _, err := os.Stat(path); err != nil {
			fmt.Printf("     ❌ 路径不存在: %v\n", err)
		} else {
			fmt.Printf("     ✅ 路径有效\n")
		}
		
		// 检查manifest.json
		manifestPath := filepath.Join(path, "manifest.json")
		if _, err := os.Stat(manifestPath); err != nil {
			fmt.Printf("     ❌ manifest.json不存在\n")
		} else {
			fmt.Printf("     ✅ manifest.json存在\n")
		}
	}

	// 配置选项 - 使用最简单的配置
	options := &browser.ConnectOptions{
		Headless:       false,
		UseCustomCDP:   false,
		Turnstile:      false,
		Extensions:     extensionPaths,
		PersistProfile: false,
		IgnoreAllFlags: false, // 使用默认标志，但会应用我们的冲突解决逻辑
		Args: []string{
			"--start-maximized",
		},
	}

	fmt.Println("\n🚀 启动浏览器...")
	instance, err := browser.Connect(ctx, options)
	if err != nil {
		log.Fatalf("启动失败: %v", err)
	}
	defer instance.Close()

	fmt.Println("✅ 浏览器启动成功")

	page := instance.Page()

	// 首先导航到一个简单页面测试
	fmt.Println("🔍 测试基本页面导航...")
	if err := page.Navigate("https://httpbin.org/headers"); err != nil {
		log.Printf("导航到测试页面失败: %v", err)
	} else {
		fmt.Println("✅ 基本导航正常")
	}

	time.Sleep(2 * time.Second)

	// 导航到扩展页面
	fmt.Println("🔍 导航到扩展管理页面...")
	if err := page.Navigate("chrome://extensions/"); err != nil {
		log.Printf("导航到扩展页面失败: %v", err)
	} else {
		fmt.Println("✅ 成功导航到chrome://extensions/")
	}

	fmt.Println("⏳ 等待页面加载...")
	time.Sleep(5 * time.Second)

	// 简单的检查 - 不使用复杂的JavaScript
	fmt.Println("🔍 检查页面内容...")
	title, err := page.Evaluate(`document.title`)
	if err != nil {
		fmt.Printf("❌ 无法获取页面标题: %v\n", err)
	} else {
		fmt.Printf("📄 页面标题: %v\n", title)
	}

	// 检查页面是否包含扩展相关元素
	hasExtensionsItems, err := page.Evaluate(`document.querySelectorAll('extensions-item').length > 0`)
	if err != nil {
		fmt.Printf("❌ 无法检查扩展项: %v\n", err)
	} else {
		fmt.Printf("🔍 页面是否有扩展项: %v\n", hasExtensionsItems)
	}

	itemCount, err := page.Evaluate(`document.querySelectorAll('extensions-item').length`)
	if err != nil {
		fmt.Printf("❌ 无法获取扩展数量: %v\n", err)
	} else {
		fmt.Printf("📊 发现扩展项数量: %v\n", itemCount)
	}

	fmt.Println("\n💡 手动检查步骤:")
	fmt.Println("  1. 浏览器窗口应该显示chrome://extensions/页面")
	fmt.Println("  2. 查看是否有任何扩展显示")
	fmt.Println("  3. 如果没有扩展，尝试:")
	fmt.Println("     - 启用右上角的'开发者模式'开关")
	fmt.Println("     - 点击'加载已解压的扩展程序'")
	fmt.Println("     - 手动选择扩展目录")

	fmt.Println("\n📍 扩展目录位置:")
	for _, path := range extensionPaths {
		fmt.Printf("  - %s\n", path)
	}

	fmt.Println("\n⏳ 保持浏览器打开60秒供检查...")
	time.Sleep(60 * time.Second)

	fmt.Println("✅ 测试完成")
}