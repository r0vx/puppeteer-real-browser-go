package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/HNRow/puppeteer-real-browser-go/pkg/browser"
)

func main() {
	fmt.Println("🔧 Simple Extension Test")
	fmt.Println("========================")

	ctx := context.Background()

	// 正确的扩展路径
	extensionPaths := []string{
		"./examples/path/Extensions/kfjglmgfjedhhcddpfgfogkahmenikan/1.0_0",
		"./examples/path/Extensions/mcohilncbfahbmgdjkbpemcciiolgcge/3.66.10_0",
	}

	fmt.Printf("🔍 将加载的扩展路径:\n")
	for i, path := range extensionPaths {
		fmt.Printf("  %d. %s\n", i+1, path)
	}

	// 浏览器配置
	opts := &browser.ConnectOptions{
		Headless:     false,
		UseCustomCDP: false,
		Extensions:   extensionPaths,
		Args: []string{
			"--start-maximized",
			"--enable-extensions",
		},
	}

	fmt.Println("\n🚀 启动浏览器...")
	instance, err := browser.Connect(ctx, opts)
	if err != nil {
		log.Fatalf("浏览器连接失败: %v", err)
	}
	defer instance.Close()

	fmt.Println("✅ 浏览器启动成功")

	// 获取页面
	page := instance.Page()

	// 简单的扩展检查
	fmt.Println("🔍 检查扩展状态...")
	
	// 先导航到普通页面
	if err := page.Navigate("chrome://version/"); err != nil {
		log.Printf("无法访问chrome://version/: %v", err)
	} else {
		fmt.Println("✅ 访问chrome://version/成功")
		time.Sleep(2 * time.Second)
		
		// 获取命令行参数
		result, err := page.Evaluate(`
			const commandLine = document.querySelector('#command_line');
			return commandLine ? commandLine.textContent : 'Command line not found';
		`)
		
		if err != nil {
			fmt.Printf("❌ 获取命令行失败: %v\n", err)
		} else {
			fmt.Printf("📝 Chrome命令行参数: %s\n", result)
		}
	}

	// 导航到扩展页面
	fmt.Println("\n🔍 检查扩展页面...")
	if err := page.Navigate("chrome://extensions/"); err != nil {
		log.Printf("无法访问扩展页面: %v", err)
	} else {
		fmt.Println("✅ 访问chrome://extensions/成功")
		time.Sleep(3 * time.Second)
		
		// 简单检查页面内容
		result, err := page.Evaluate(`
			return {
				pageTitle: document.title,
				hasExtensionsManager: !!document.querySelector('extensions-manager'),
				hasExtensionsItems: document.querySelectorAll('extensions-item').length,
				pageText: document.body.innerText.substring(0, 200)
			};
		`)
		
		if err != nil {
			fmt.Printf("❌ 页面检查失败: %v\n", err)
		} else {
			fmt.Printf("📊 扩展页面信息: %+v\n", result)
		}
	}

	fmt.Println("\n💡 手动验证:")
	fmt.Println("  1. 查看浏览器窗口的chrome://extensions/页面")
	fmt.Println("  2. 检查是否有扩展出现")
	fmt.Println("  3. 如果没有扩展，检查命令行参数中是否有--load-extension")

	fmt.Println("\n⏳ 保持浏览器打开 10 秒...")
	time.Sleep(10 * time.Second)

	fmt.Println("✅ 测试完成")
}