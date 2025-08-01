package main

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/HNRow/puppeteer-real-browser-go/pkg/browser"
	"github.com/HNRow/puppeteer-real-browser-go/pkg/page"
	"github.com/HNRow/puppeteer-real-browser-go/pkg/turnstile"
)

func main() {
	fmt.Println("☁️ Cloudflare 绕过测试")
	fmt.Println("======================")

	ctx := context.Background()

	// 经过验证的最佳配置 - 基于原版JavaScript版本的配置
	opts := &browser.ConnectOptions{
		Headless:     false,
		Turnstile:    true,
		UseCustomCDP: false, // 暂时使用标准CDP，避免上下文问题
		Args: []string{
			"--start-maximized",
			"--disable-blink-features=AutomationControlled",
			"--exclude-switches=enable-automation",
			"--disable-extensions",
			"--no-sandbox",
			"--disable-dev-shm-usage",
			"--disable-setuid-sandbox",
			"--disable-gpu-sandbox",
			"--disable-software-rasterizer",
			"--disable-background-timer-throttling",
			"--disable-backgrounding-occluded-windows",
			"--disable-renderer-backgrounding",
			"--disable-features=TranslateUI,BlinkGenPropertyTrees",
			"--disable-ipc-flooding-protection",
			"--disable-hang-monitor",
			"--disable-prompt-on-repost",
			"--disable-client-side-phishing-detection",
			"--disable-component-extensions-with-background-pages",
			"--disable-default-apps",
			"--disable-sync",
			"--disable-translate",
			"--hide-scrollbars",
			"--mute-audio",
			"--no-first-run",
			"--safebrowsing-disable-auto-update",
			"--ignore-certificate-errors",
			"--ignore-ssl-errors",
			"--ignore-certificate-errors-spki-list",
			"--disable-features=VizDisplayCompositor",
			"--user-agent=Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36",
		},
		CustomConfig: map[string]interface{}{
			"ignoreDefaultFlags": true,
		},
	}

	fmt.Println("🚀 启动增强反检测模式...")
	instance, err := browser.Connect(ctx, opts)
	if err != nil {
		log.Fatalf("❌ 浏览器启动失败: %v", err)
	}
	defer instance.Close()

	browserPage := instance.Page()

	// 创建页面控制器
	controller := page.NewController(browserPage, ctx, true)
	if err := controller.Initialize(); err != nil {
		log.Fatalf("❌ 页面控制器初始化失败: %v", err)
	}
	defer controller.Stop()

	// 创建Turnstile解决器
	if opts.Turnstile {
		solver := turnstile.NewSolver(browserPage, ctx)
		if err := solver.Start(); err != nil {
			log.Printf("⚠️ Turnstile解决器启动失败: %v", err)
		} else {
			defer solver.Stop()
			fmt.Println("✅ Turnstile自动解决已启用")
		}
	}

	fmt.Println("✅ 浏览器配置完成")

	// 步骤 1: 验证反检测功能
	fmt.Println("\n🛡️ 步骤 1: 验证反检测功能")
	testAntiDetection(browserPage)

	// 步骤 2: 预热浏览器
	fmt.Println("\n🔥 步骤 2: 预热浏览器")
	warmupBrowser(browserPage)

	// 步骤 3: 测试受保护的网站
	fmt.Println("\n🎯 步骤 3: 测试受保护的网站")

	// 你可以替换为任何受 Cloudflare 保护的网站
	testSites := []string{
		"https://irys.xyz/faucet",
		// "https://your-target-site.com",
	}

	for _, site := range testSites {
		fmt.Printf("\n🌐 测试网站: %s\n", site)
		success := testProtectedSite(browserPage, site)

		if success {
			fmt.Printf("🎉 成功绕过 %s 的 Cloudflare 保护!\n", site)
		} else {
			fmt.Printf("⚠️ %s 可能仍被保护\n", site)
		}

		time.Sleep(5 * time.Second)
	}

	fmt.Println("\n✅ Cloudflare 测试完成!")
	fmt.Println("浏览器将保持打开 30 秒供手动验证...")
	time.Sleep(30 * time.Second)
}

func testAntiDetection(page browser.Page) {
	script := `({
		webdriver: navigator.webdriver,
		userAgent: navigator.userAgent,
		plugins: navigator.plugins.length,
		languages: navigator.languages.length,
		hardwareConcurrency: navigator.hardwareConcurrency,
		chrome: typeof window.chrome !== 'undefined',
		mouseEventTest: (() => {
			const event = new MouseEvent('click', { clientX: 100, clientY: 200 });
			return {
				clientX: event.clientX,
				screenX: event.screenX,
				fixed: event.screenX === event.clientX + (window.screenX || 0)
			};
		})()
	})`

	result, err := page.Evaluate(script)
	if err != nil {
		fmt.Printf("❌ 反检测测试失败: %v\n", err)
		return
	}

	fmt.Printf("📊 反检测状态: %+v\n", result)
}

func warmupBrowser(page browser.Page) {
	fmt.Println("🌐 预热访问 Google...")
	err := page.Navigate("https://www.google.com")
	if err != nil {
		fmt.Printf("⚠️ 预热失败: %v\n", err)
		return
	}

	time.Sleep(3 * time.Second)
	title, _ := page.GetTitle()
	fmt.Printf("✅ 预热完成: %s\n", title)
	time.Sleep(2 * time.Second)
}

func testProtectedSite(page browser.Page, url string) bool {
	fmt.Printf("🔄 访问: %s\n", url)

	err := page.Navigate(url)
	if err != nil {
		fmt.Printf("❌ 导航失败: %v\n", err)
		return false
	}

	// 等待页面加载
	fmt.Println("⏳ 等待页面加载...")
	time.Sleep(8 * time.Second)

	// 检查页面状态
	title, _ := page.GetTitle()
	fmt.Printf("📄 页面标题: %s\n", title)

	// 分析是否成功
	if isCloudflareBlocked(title) {
		fmt.Println("🛡️ 检测到 Cloudflare 挑战页面")
		return false
	}

	if title != "" && len(title) > 3 {
		fmt.Println("✅ 成功访问目标页面!")
		return true
	}

	fmt.Println("❓ 页面状态未知")
	return false
}

func isCloudflareBlocked(title string) bool {
	indicators := []string{
		"just a moment",
		"checking your browser",
		"cloudflare",
		"please wait",
		"verifying you are human",
		"security check",
		"ddos protection",
	}

	titleLower := strings.ToLower(title)
	for _, indicator := range indicators {
		if strings.Contains(titleLower, indicator) {
			return true
		}
	}
	return false
}
