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
	fmt.Println("🔧 最终插件加载测试")
	fmt.Println("==================")

	// 获取插件绝对路径
	ext1Path, _ := filepath.Abs("examples/path/Extensions/kfjglmgfjedhhcddpfgfogkahmenikan/1.0_0")
	ext2Path, _ := filepath.Abs("examples/path/Extensions/mcohilncbfahbmgdjkbpemcciiolgcge/3.66.10_0")

	fmt.Printf("📦 插件1: %s\n", ext1Path)
	fmt.Printf("📦 插件2: %s\n", ext2Path)

	// 更精确的Chrome参数配置
	opts := []chromedp.ExecAllocatorOption{
		// 基础设置
		chromedp.NoFirstRun,
		chromedp.NoDefaultBrowserCheck,
		chromedp.Flag("headless", false),
		
		// 启用扩展的关键参数
		chromedp.Flag("enable-extensions", true),
		chromedp.Flag("disable-extensions-file-access-check", true),
		chromedp.Flag("disable-web-security", true),
		chromedp.Flag("allow-running-insecure-content", true),
		
		// 加载我们的插件
		chromedp.Flag("load-extension", ext1Path),
		chromedp.Flag("load-extension", ext2Path),
		
		// 开发者模式相关
		chromedp.Flag("enable-logging", true),
		chromedp.Flag("enable-extension-activity-logging", true),
		chromedp.Flag("disable-extensions-http-throttling", true),
		
		// 安全相关
		chromedp.Flag("no-sandbox", true),
		chromedp.Flag("disable-dev-shm-usage", true),
		chromedp.Flag("disable-gpu", true),
		
		// 数据目录
		chromedp.UserDataDir("/tmp/chrome-extensions-test"),
	}

	allocCtx, cancel := chromedp.NewExecAllocator(context.Background(), opts...)
	defer cancel()

	// 创建上下文，启用调试
	ctx, cancel := chromedp.NewContext(allocCtx)
	defer cancel()

	fmt.Println("\n🚀 启动Chrome...")

	// 首先导航到扩展管理页面
	err := chromedp.Run(ctx,
		chromedp.Navigate("chrome://extensions/"),
		chromedp.Sleep(3*time.Second),
	)

	if err != nil {
		log.Fatalf("❌ Chrome启动失败: %v", err)
	}

	fmt.Println("✅ Chrome启动成功")

	// 启用开发者模式
	fmt.Println("\n🔧 启用开发者模式...")
	err = chromedp.Run(ctx,
		chromedp.WaitVisible("body", chromedp.ByQuery),
		chromedp.Sleep(1*time.Second),
		// 尝试点击开发者模式开关
		chromedp.Evaluate(`
			const toggle = document.querySelector('extensions-manager')?.shadowRoot
				?.querySelector('extensions-toolbar')?.shadowRoot
				?.querySelector('#devMode');
			if (toggle && !toggle.checked) {
				toggle.click();
				console.log('开发者模式已启用');
			}
		`, nil),
		chromedp.Sleep(2*time.Second),
	)

	if err != nil {
		fmt.Printf("⚠️  开发者模式设置失败: %v\n", err)
	}

	// 检查已加载的扩展
	fmt.Println("\n🔍 检查已加载的扩展...")
	var extensionCount int
	err = chromedp.Run(ctx,
		chromedp.Evaluate(`
			// 等待页面加载完成
			new Promise((resolve) => {
				setTimeout(() => {
					try {
						const manager = document.querySelector('extensions-manager');
						const items = manager?.shadowRoot?.querySelectorAll('extensions-item') || [];
						console.log('找到的扩展数量:', items.length);
						
						items.forEach((item, index) => {
							const name = item?.shadowRoot?.querySelector('#name')?.textContent || 'Unknown';
							const id = item?.getAttribute('id') || 'Unknown';
							console.log('扩展', index + 1, ':', name, 'ID:', id);
						});
						
						resolve(items.length);
					} catch (e) {
						console.error('检查扩展时出错:', e);
						resolve(0);
					}
				}, 3000);
			})
		`, &extensionCount),
	)

	if err != nil {
		fmt.Printf("❌ 扩展检查失败: %v\n", err)
	} else {
		fmt.Printf("📊 检测到 %d 个扩展\n", extensionCount)
	}

	// 打开新页面测试扩展功能
	fmt.Println("\n🌐 测试扩展功能...")
	
	// 测试页面1 - httpbin
	err = chromedp.Run(ctx,
		chromedp.Navigate("https://httpbin.org/get"),
		chromedp.WaitReady("body"),
		chromedp.Sleep(3*time.Second),
	)

	if err == nil {
		var testResult map[string]interface{}
		chromedp.Run(ctx,
			chromedp.Evaluate(`
				({
					url: location.href,
					title: document.title,
					hasChrome: !!window.chrome,
					hasOkxWallet: !!(window.okxwallet || window.ethereum?.isOkxWallet),
					hasEthereum: !!window.ethereum,
					chromeRuntime: !!window.chrome?.runtime,
					chromeRuntimeId: window.chrome?.runtime?.id,
					// 检查页面是否被扩展修改
					titleModified: document.title.includes('📦') || document.title !== document.title,
					// 检查全局变量
					globalVars: Object.keys(window).filter(key => 
						key.toLowerCase().includes('discord') || 
						key.toLowerCase().includes('okx') ||
						key.toLowerCase().includes('wallet')
					)
				})
			`, &testResult),
		)
		
		fmt.Printf("🔗 页面测试结果: %+v\n", testResult)
	}

	// 手动验证说明
	fmt.Println("\n💡 手动验证步骤:")
	fmt.Println("  1. 查看Chrome扩展管理页面 (chrome://extensions/)")
	fmt.Println("  2. 确认开发者模式已启用")
	fmt.Println("  3. 查找以下扩展:")
	fmt.Println("     - Discord Token Login")
	fmt.Println("     - OKX Wallet")
	fmt.Println("  4. 检查浏览器工具栏是否有扩展图标")
	fmt.Println("  5. 测试功能:")
	fmt.Println("     - 访问 discord.com 测试Discord扩展")
	fmt.Println("     - 访问 app.uniswap.org 测试OKX钱包")

	fmt.Println("\n📋 故障排除:")
	fmt.Println("  - 如果扩展未显示，检查manifest.json是否有效")
	fmt.Println("  - 确认扩展路径正确且可访问")
	fmt.Println("  - 查看Chrome控制台是否有错误信息")

	fmt.Println("\n⏳ 保持浏览器开启30秒供测试...")
	time.Sleep(30 * time.Second)

	fmt.Println("✅ 插件加载测试完成")
}