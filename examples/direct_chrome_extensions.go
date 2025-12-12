package main

import (
	"context"
	"fmt"
	"log"
	"path/filepath"
	"time"

	"github.com/chromedp/chromedp"
)

func main() {
	fmt.Println("🔧 直接使用Chrome加载插件")
	fmt.Println("========================")

	// 获取插件绝对路径
	ext1Path, _ := filepath.Abs("examples/path/Extensions/kfjglmgfjedhhcddpfgfogkahmenikan/1.0_0")
	ext2Path, _ := filepath.Abs("examples/path/Extensions/mcohilncbfahbmgdjkbpemcciiolgcge/3.66.10_0")

	fmt.Printf("📦 插件1路径: %s\n", ext1Path)
	fmt.Printf("📦 插件2路径: %s\n", ext2Path)

	// Chrome启动参数
	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		// 基础设置
		chromedp.Flag("headless", false),
		chromedp.Flag("no-sandbox", true),
		chromedp.Flag("disable-dev-shm-usage", true),
		
		// 关键：启用扩展
		chromedp.Flag("enable-extensions", true),
		chromedp.Flag("disable-extensions-file-access-check", true),
		chromedp.Flag("disable-web-security", true),
		chromedp.Flag("allow-running-insecure-content", true),
		
		// 加载扩展 - 使用逗号分隔多个扩展
		chromedp.Flag("load-extension", ext1Path+","+ext2Path),
		
		// 禁用一些可能干扰的功能
		chromedp.Flag("disable-features", "VizDisplayCompositor"),
		chromedp.Flag("disable-extensions-http-throttling", true),
		
		// 用户数据目录
		chromedp.UserDataDir("/tmp/chrome-with-extensions"),
	)

	allocCtx, cancel := chromedp.NewExecAllocator(context.Background(), opts...)
	defer cancel()

	ctx, cancel := chromedp.NewContext(allocCtx, chromedp.WithDebugf(log.Printf))
	defer cancel()

	fmt.Println("\n🚀 启动Chrome并加载插件...")
	
	// 启动Chrome并导航
	err := chromedp.Run(ctx,
		chromedp.Navigate("chrome://extensions/"),
		chromedp.WaitReady("body"),
	)
	if err != nil {
		log.Fatalf("Chrome启动失败: %v", err)
	}

	fmt.Println("✅ Chrome启动成功，已导航到扩展页面")

	// 等待页面加载
	time.Sleep(3 * time.Second)

	// 检查扩展是否加载
	fmt.Println("\n🔍 检查已加载的扩展...")
	var extensionInfo string
	err = chromedp.Run(ctx,
		chromedp.Evaluate(`
			// 等待页面完全加载
			new Promise((resolve) => {
				setTimeout(() => {
					try {
						// 尝试获取扩展信息
						const extensionItems = document.querySelectorAll('extensions-item');
						let extensionCount = extensionItems.length;
						
						let results = [];
						extensionItems.forEach((item, index) => {
							try {
								const name = item.shadowRoot?.querySelector('#name')?.textContent || 'Unknown';
								const id = item.getAttribute('id') || 'Unknown';
								const enabled = item.shadowRoot?.querySelector('cr-toggle')?.checked || false;
								results.push({name, id, enabled, index});
							} catch (e) {
								results.push({error: e.message, index});
							}
						});
						
						resolve(JSON.stringify({
							extensionCount: extensionCount,
							extensions: results,
							pageTitle: document.title,
							url: location.href
						}));
					} catch (e) {
						resolve(JSON.stringify({
							error: e.message,
							pageTitle: document.title,
							url: location.href
						}));
					}
				}, 2000);
			})
		`, &extensionInfo),
	)

	if err != nil {
		fmt.Printf("❌ 检查扩展失败: %v\n", err)
	} else {
		fmt.Printf("📊 扩展检查结果:\n%s\n", extensionInfo)
	}

	// 打开一个测试页面
	fmt.Println("\n🌐 打开测试页面验证插件功能...")
	err = chromedp.Run(ctx,
		chromedp.Navigate("https://httpbin.org/get"),
		chromedp.WaitReady("body"),
	)
	
	if err == nil {
		// 检查页面中是否有插件注入的内容
		time.Sleep(2 * time.Second)
		
		var pageInfo string
		err = chromedp.Run(ctx,
			chromedp.Evaluate(`
				JSON.stringify({
					title: document.title,
					url: location.href,
					hasChrome: !!window.chrome,
					hasOkxWallet: !!(window.okxwallet || window.ethereum?.isOkxWallet),
					hasEthereum: !!window.ethereum,
					windowProps: Object.keys(window).filter(key => 
						key.includes('okx') || 
						key.includes('ethereum') || 
						key.includes('wallet') || 
						key.includes('discord')
					)
				})
			`, &pageInfo),
		)
		
		if err == nil {
			fmt.Printf("🔗 页面插件状态:\n%s\n", pageInfo)
		}
	}

	fmt.Println("\n💡 手动验证:")
	fmt.Println("  1. 查看浏览器右上角是否有插件图标")
	fmt.Println("  2. 在扩展页面应该能看到2个插件:")
	fmt.Println("     - Discord Token Login")
	fmt.Println("     - OKX Wallet")
	fmt.Println("  3. 访问 discord.com 测试Discord插件")
	fmt.Println("  4. 访问 app.uniswap.org 测试OKX钱包")

	fmt.Println("\n⏳ 保持浏览器开启20秒供手动测试...")
	time.Sleep(20 * time.Second)

	fmt.Println("✅ 扩展测试完成")
}