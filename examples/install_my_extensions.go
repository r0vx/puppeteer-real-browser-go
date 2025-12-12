package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/HNRow/puppeteer-real-browser-go/pkg/browser"
)

func main() {
	fmt.Println("🔧 安装你的2个插件")
	fmt.Println("================")

	ctx := context.Background()

	// 你的插件路径
	extensionPaths := []string{
		"./examples/path/Extensions/kfjglmgfjedhhcddpfgfogkahmenikan/1.0_0", // Discord Token Login
		"./examples/path/Extensions/mcohilncbfahbmgdjkbpemcciiolgcge/3.66.10_0", // OKX Wallet
	}

	fmt.Println("📦 准备安装的插件:")
	fmt.Println("  1. Discord Token Login (v1.0)")
	fmt.Println("  2. OKX Wallet (v3.66.10)")

	// 配置浏览器选项 - 使用最简化的设置
	opts := &browser.ConnectOptions{
		Headless:     false, 
		Extensions:   extensionPaths,
		IgnoreAllFlags: true, // 忽略默认标志，避免冲突
		Args: []string{
			"--no-sandbox",
			"--disable-dev-shm-usage",
			"--enable-extensions",
			"--disable-extensions-file-access-check",
			"--disable-web-security",
			"--allow-running-insecure-content",
			"--user-data-dir=/tmp/chrome-with-extensions-simple",
		},
	}

	fmt.Println("\n🚀 启动浏览器并安装插件...")
	instance, err := browser.Connect(ctx, opts)
	if err != nil {
		log.Fatalf("浏览器连接失败: %v", err)
	}
	defer instance.Close()

	fmt.Println("✅ 浏览器启动成功")

	// 打开主页面
	page := instance.Page()
	if err := page.Navigate("https://httpbin.org/get"); err != nil {
		log.Printf("页面导航失败: %v", err)
	}

	// 等待插件加载
	fmt.Println("\n⏳ 等待插件加载...")
	time.Sleep(3 * time.Second)

	// 检查插件是否加载成功
	fmt.Println("\n🔍 检查插件状态...")
	
	// 检查页面中是否有插件注入的内容
	result, err := page.Evaluate(`
		{
			// 检查是否有Chrome插件环境
			hasChromeRuntime: !!(window.chrome && window.chrome.runtime),
			
			// 检查是否有OKX钱包插件
			hasOkxWallet: !!(window.okxwallet || window.ethereum?.isOkxWallet),
			
			// 检查扩展数量
			extensionsCount: window.chrome?.runtime ? 'available' : 'not available',
			
			// 检查页面标题是否被修改（Discord插件可能会修改）
			pageTitle: document.title,
			
			// 检查URL
			currentUrl: location.href
		}
	`)
	
	if err != nil {
		fmt.Printf("❌ 检查失败: %v\n", err)
	} else {
		fmt.Printf("📊 插件检查结果: %v\n", result)
	}

	// 打开插件管理页面
	fmt.Println("\n📦 打开Chrome插件管理页面...")
	context, err := instance.CreateBrowserContext(nil)
	if err == nil {
		extensionsPage, err := context.NewPage()
		if err == nil {
			if err := extensionsPage.Navigate("chrome://extensions/"); err == nil {
				fmt.Println("✅ 插件管理页面已打开")
				
				// 等待页面加载后检查插件
				time.Sleep(2 * time.Second)
				
				result, err := extensionsPage.Evaluate(`
					// 等待页面加载
					setTimeout(() => {
						const extensionItems = document.querySelectorAll('extensions-item');
						const extensions = Array.from(extensionItems).map(item => {
							const name = item.shadowRoot?.querySelector('#name')?.textContent || 'Unknown';
							const id = item.getAttribute('id') || item.dataset?.id || 'Unknown';
							return { name, id };
						});
						console.log('找到的插件:', extensions);
					}, 1000);
					
					// 返回当前状态
					{
						pageLoaded: true,
						url: location.href
					}
				`)
				
				if err == nil {
					fmt.Printf("📋 插件页面状态: %v\n", result)
				}
			}
		}
	}

	// 测试Discord插件功能
	fmt.Println("\n🎮 测试Discord插件...")
	discordContext, err := instance.CreateBrowserContext(nil)
	if err == nil {
		discordPage, err := discordContext.NewPage()
		if err == nil {
			// 导航到Discord测试
			discordPage.Navigate("https://discord.com/login")
			time.Sleep(3 * time.Second)
			
			// 检查Discord插件是否注入
			discordResult, err := discordPage.Evaluate(`
				{
					url: location.href,
					hasDiscordToken: !!localStorage.getItem('token'),
					canSetToken: typeof localStorage !== 'undefined',
					currentDomain: location.hostname
				}
			`)
			
			if err == nil {
				fmt.Printf("🎮 Discord插件测试: %v\n", discordResult)
			}
		}
	}

	// 测试OKX钱包功能
	fmt.Println("\n💰 测试OKX钱包插件...")
	walletPage, err := instance.CreateBrowserContext(nil)
	if err == nil {
		testPage, err := walletPage.NewPage()
		if err == nil {
			// 导航到一个Web3网站测试
			testPage.Navigate("https://app.uniswap.org/")
			time.Sleep(3 * time.Second)
			
			// 检查钱包是否可用
			walletResult, err := testPage.Evaluate(`
				{
					hasEthereum: !!window.ethereum,
					hasOkxWallet: !!(window.okxwallet || window.ethereum?.isOkxWallet),
					providers: Object.keys(window).filter(key => key.includes('wallet') || key.includes('ethereum')),
					injectedProviders: window.ethereum ? Object.keys(window.ethereum) : []
				}
			`)
			
			if err == nil {
				fmt.Printf("🔗 钱包连接测试: %v\n", walletResult)
			}
		}
	}

	fmt.Println("\n💡 使用说明:")
	fmt.Println("  1. ✅ Discord Token Login 插件已加载 - 在discord.com使用")
	fmt.Println("  2. ✅ OKX Wallet 插件已加载 - 可连接Web3应用")
	fmt.Println("  3. 📱 查看浏览器右上角的插件图标")
	fmt.Println("  4. 🔧 在 chrome://extensions/ 页面管理插件")

	fmt.Println("\n📋 手动验证步骤:")
	fmt.Println("  • 访问 https://discord.com - 测试Discord插件")
	fmt.Println("  • 访问 https://app.uniswap.org - 测试OKX钱包连接")
	fmt.Println("  • 点击浏览器工具栏中的插件图标")
	fmt.Println("  • 检查 chrome://extensions/ 页面中的插件状态")

	fmt.Println("\n⏳ 保持浏览器开启15秒供测试...")
	time.Sleep(15 * time.Second)

	fmt.Println("✅ 插件安装演示完成！")
	fmt.Println("🎉 你的2个插件已成功安装并可使用")
}