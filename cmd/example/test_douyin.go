package main

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/r0vx/puppeteer-real-browser-go/pkg/browser"
)

func main() {
	fmt.Println("🔍 测试抖音充值页面反检测")
	fmt.Println(strings.Repeat("=", 60))

	ctx := context.Background()

	// 测试 1: 不使用反检测
	fmt.Println("\n📊 测试 1: 原生访问（预期被检测）")
	fmt.Println(strings.Repeat("-", 60))
	testWithoutStealth(ctx)

	// 等待一下
	fmt.Println("\n⏳ 等待 5 秒后进行第二个测试...\n")
	time.Sleep(5 * time.Second)

	// 测试 2: 使用本项目的反检测
	fmt.Println("\n📊 测试 2: 启用反检测（预期通过）")
	fmt.Println(strings.Repeat("-", 60))
	testWithStealth(ctx)

	fmt.Println("\n✅ 测试完成！")
}

func testWithoutStealth(ctx context.Context) {
	fmt.Println("  [配置] 不使用反检测...")

	opts := &browser.ConnectOptions{
		Headless:     false, // 可视化观察
		UseChromedp:  true,  // 对照组：旧 chromedp 通道（只给主页面下发身份）
	}

	instance, err := browser.Connect(ctx, opts)
	if err != nil {
		fmt.Printf("  ❌ 连接失败: %v\n", err)
		return
	}
	defer instance.Close()

	page := instance.Page()

	// 访问抖音充值页面
	fmt.Println("  [导航] 访问 https://www.douyin.com/pay ...")
	if err := page.Navigate("https://www.douyin.com/pay"); err != nil {
		fmt.Printf("  ❌ 导航失败: %v\n", err)
		return
	}

	// 等待页面加载
	time.Sleep(3 * time.Second)

	// 检测是否被识别为自动化
	fmt.Println("  [检测] 检查浏览器指纹...")
	result, err := page.Evaluate(`
		(function() {
			return {
				webdriver: navigator.webdriver,
				chrome: !!window.chrome,
				plugins: navigator.plugins.length,
				languages: navigator.languages.join(','),
				// 检测字节跳动的反爬虫系统
				byted_acrawler: typeof window.byted_acrawler !== 'undefined',
				slardar: typeof window.__SLARDAR__ !== 'undefined',
				tea: typeof window.__TEA__ !== 'undefined',
				// 检测自动化特征
				automation: {
					webdriver_prop: 'webdriver' in navigator,
					chrome_runtime: !!window.chrome?.runtime,
				}
			};
		})()
	`)

	if err != nil {
		fmt.Printf("  ⚠️  检测脚本执行失败: %v\n", err)
	} else {
		fmt.Printf("  📋 检测结果:\n")
		fmt.Printf("      %+v\n", result)
	}

	// 获取页面标题
	title, _ := page.Evaluate("document.title")
	fmt.Printf("  📄 页面标题: %v\n", title)

	// 等待观察
	fmt.Println("  ⏳ 保持 10 秒观察页面反应...")
	fmt.Println("     (请查看浏览器窗口，是否有验证码或警告)")
	time.Sleep(10 * time.Second)
}

func testWithStealth(ctx context.Context) {
	fmt.Println("  [配置] 启用完整反检测...")

	opts := &browser.ConnectOptions{
		Headless:     false, // 可视化观察
		Turnstile:    true,  // 启用验证码自动解决
		Args: []string{
			"--start-maximized",
			"--disable-blink-features=AutomationControlled",
			"--exclude-switches=enable-automation",
		},
	}

	instance, err := browser.Connect(ctx, opts)
	if err != nil {
		fmt.Printf("  ❌ 连接失败: %v\n", err)
		return
	}
	defer instance.Close()

	page := instance.Page()

	// 访问抖音充值页面
	fmt.Println("  [导航] 访问 https://www.douyin.com/pay ...")
	if err := page.Navigate("https://www.douyin.com/pay"); err != nil {
		fmt.Printf("  ❌ 导航失败: %v\n", err)
		return
	}

	// 等待页面加载
	time.Sleep(3 * time.Second)

	// 检测是否成功隐藏
	fmt.Println("  [检测] 检查浏览器指纹...")
	result, err := page.Evaluate(`
		(function() {
			return {
				webdriver: navigator.webdriver,
				chrome: !!window.chrome,
				plugins: navigator.plugins.length,
				languages: navigator.languages.join(','),
				byted_acrawler: typeof window.byted_acrawler !== 'undefined',
				slardar: typeof window.__SLARDAR__ !== 'undefined',
				tea: typeof window.__TEA__ !== 'undefined',
				automation: {
					webdriver_prop: 'webdriver' in navigator,
					chrome_runtime: !!window.chrome?.runtime,
				}
			};
		})()
	`)

	if err != nil {
		fmt.Printf("  ⚠️  检测脚本执行失败: %v\n", err)
	} else {
		fmt.Printf("  📋 检测结果:\n")
		fmt.Printf("      %+v\n", result)
	}

	// 获取页面标题
	title, _ := page.Evaluate("document.title")
	fmt.Printf("  📄 页面标题: %v\n", title)

	// 等待观察
	fmt.Println("  ⏳ 保持 15 秒观察页面反应...")
	fmt.Println("     (请查看浏览器窗口，对比两次测试的差异)")
	time.Sleep(15 * time.Second)

	fmt.Println("  ✅ 测试完成")
}


