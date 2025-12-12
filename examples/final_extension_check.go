package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/HNRow/puppeteer-real-browser-go/pkg/browser"
)

func main() {
	fmt.Println("🔧 Final Extension Test")
	fmt.Println("=======================")

	ctx := context.Background()

	// 绝对路径扩展
	extensionPaths := []string{
		"/Users/rowei/Develop/go/puppeteer-real-browser-go/examples/path/Extensions/kfjglmgfjedhhcddpfgfogkahmenikan/1.0_0",
		"/Users/rowei/Develop/go/puppeteer-real-browser-go/examples/path/Extensions/mcohilncbfahbmgdjkbpemcciiolgcge/3.66.10_0",
	}

	fmt.Printf("📦 使用绝对路径加载扩展:\n")
	for i, path := range extensionPaths {
		fmt.Printf("  %d. %s\n", i+1, path)
	}

	// 最基本的配置，强制加载扩展
	opts := &browser.ConnectOptions{
		Headless:   false,
		Extensions: extensionPaths,
		Args: []string{
			"--enable-extensions",
			"--disable-extensions-file-access-check",
			"--disable-web-security",
			"--allow-running-insecure-content",
		},
	}

	fmt.Println("\n🚀 启动浏览器...")
	instance, err := browser.Connect(ctx, opts)
	if err != nil {
		log.Fatalf("浏览器启动失败: %v", err)
	}
	defer instance.Close()

	fmt.Println("✅ 浏览器启动成功")

	// 直接导航到扩展页面
	page := instance.Page()
	fmt.Println("🔍 打开扩展页面...")
	if err := page.Navigate("chrome://extensions/"); err != nil {
		fmt.Printf("❌ 无法访问扩展页面: %v\n", err)
	} else {
		fmt.Println("✅ 扩展页面已打开")
	}

	fmt.Println("\n🔍 请检查浏览器窗口:")
	fmt.Println("  - 是否看到 Discord Token Login 扩展?")
	fmt.Println("  - 是否看到 OKX Wallet 扩展?")
	fmt.Println("  - 如果没有，说明 --load-extension 参数有问题")

	fmt.Println("\n⏳ 保持浏览器打开 10 秒...")
	time.Sleep(10 * time.Second)

	fmt.Println("✅ 测试完成")
}