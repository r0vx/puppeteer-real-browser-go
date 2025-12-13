package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/r0vx/puppeteer-real-browser-go/pkg/browser"
)

func main() {
	fmt.Println("🎭 浏览器指纹伪造演示")
	fmt.Println("=" + string(make([]byte, 60)))
	fmt.Println()

	// 创建指纹管理器
	fingerprintManager, err := browser.NewUserFingerprintManager("./cmd/fingerprint_configs")
	if err != nil {
		log.Fatalf("❌ 创建指纹管理器失败: %v", err)
	}

	// 演示不同的使用场景
	fmt.Println("📋 选择测试场景：")
	fmt.Println("1. 使用单个用户指纹")
	fmt.Println("2. 使用多个不同用户指纹（模拟多设备）")
	fmt.Println("3. 对比有无指纹的差异")
	fmt.Println()

	// 场景1：单个用户指纹
	testSingleFingerprint(fingerprintManager)

	// 场景2：多用户指纹
	// testMultipleFingerprints(fingerprintManager)

	// 场景3：对比测试
	// testFingerprintComparison()
}

// testSingleFingerprint 使用单个用户指纹测试
func testSingleFingerprint(manager *browser.UserFingerprintManager) {
	fmt.Println("\n🔍 场景1：使用单个用户指纹")
	fmt.Println("-" + string(make([]byte, 60)))

	userID := "test_user_001"

	// 获取或生成用户指纹
	fingerprintConfig, err := manager.GetUserFingerprint(userID)
	if err != nil {
		log.Fatalf("❌ 获取指纹失败: %v", err)
	}

	fmt.Printf("✅ 已加载用户指纹: %s\n", userID)

	// 安全地截取字符串
	ua := fingerprintConfig.Browser.UserAgent
	if len(ua) > 60 {
		ua = ua[:60] + "..."
	}
	fmt.Printf("   📱 User-Agent: %s\n", ua)

	fmt.Printf("   🖥️  Platform: %s\n", fingerprintConfig.Browser.Platform)
	fmt.Printf("   📺 Screen: %dx%d (DPR: %.1f)\n",
		fingerprintConfig.Screen.Width,
		fingerprintConfig.Screen.Height,
		fingerprintConfig.Screen.DevicePixelRatio)

	renderer := fingerprintConfig.WebGL.Renderer
	if len(renderer) > 50 {
		renderer = renderer[:50] + "..."
	}
	fmt.Printf("   🎨 WebGL: %s\n", renderer)
	fmt.Println()

	// 创建指纹注入器
	injector := browser.NewFingerprintInjector(fingerprintConfig)

	// 生成JavaScript注入脚本
	injectionScript := injector.GenerateInjectionScript()

	// 配置浏览器选项
	ctx := context.Background()

	// 合并指纹参数和禁用恢复页面的参数
	args := fingerprintConfig.GetChromeFlags()
	args = append(args,
		"--disable-session-crashed-bubble", // 禁用崩溃提示
		"--disable-infobars",               // 禁用信息栏
		"--no-first-run",                   // 禁用首次运行提示
		"--no-default-browser-check",       // 禁用默认浏览器检查
		"--disable-popup-blocking",         // 禁用弹窗拦截
		"--disable-translate",              // 禁用翻译提示
		"--disable-features=TranslateUI",   // 禁用翻译UI
		"--disable-features=Translate",     // 禁用翻译功能
	)

	opts := &browser.ConnectOptions{
		Headless:       false,
		PersistProfile: true,
		ProfileName:    fmt.Sprintf("fingerprint_%s", userID),
		Args:           args,
	}

	fmt.Println("🚀 启动浏览器...")
	instance, err := browser.Connect(ctx, opts)
	if err != nil {
		log.Fatalf("❌ 浏览器启动失败: %v", err)
	}
	defer instance.Close()

	page := instance.Page()

	// 注入指纹脚本
	// 注意：在实际项目中，指纹脚本会在浏览器初始化时自动注入
	// 这里只是演示概念
	fmt.Println("💉 指纹脚本已准备好...")
	_ = injectionScript // 脚本会在浏览器启动时自动应用

	// 访问指纹检测网站
	fmt.Println("🌐 访问指纹检测网站...")
	testURLs := []string{
		"https://browserleaks.com/canvas",
		"https://abrahamjuliot.github.io/creepjs/",
		"https://pixelscan.net/",
	}

	for i, url := range testURLs {
		fmt.Printf("\n📍 测试 %d/%d: %s\n", i+1, len(testURLs), url)

		if err := page.Navigate(url); err != nil {
			log.Printf("⚠️  导航失败: %v", err)
			continue
		}

		// 等待页面加载
		time.Sleep(5 * time.Second)

		// 获取页面标题
		title, _ := page.GetTitle()
		fmt.Printf("   ✅ 页面标题: %s\n", title)

		// 检查指纹是否生效
		checkFingerprint(page, fingerprintConfig)

		if i < len(testURLs)-1 {
			fmt.Println("   ⏳ 等待5秒后继续...")
			time.Sleep(5 * time.Second)
		}
	}

	fmt.Println("\n⏳ 保持浏览器打开30秒供您检查...")
	fmt.Println("💡 您可以手动在控制台检查以下内容：")
	fmt.Println("   • navigator.userAgent")
	fmt.Println("   • navigator.webdriver")
	fmt.Println("   • screen.width + 'x' + screen.height")
	fmt.Println("   • navigator.hardwareConcurrency")
	time.Sleep(30 * time.Second)

	fmt.Println("\n✅ 测试完成！")
}

// testMultipleFingerprints 测试多个不同的指纹
func testMultipleFingerprints(manager *browser.UserFingerprintManager) {
	fmt.Println("\n🔍 场景2：使用多个不同用户指纹")
	fmt.Println("-" + string(make([]byte, 60)))

	userIDs := []string{"user_001", "user_002", "user_003"}

	for i, userID := range userIDs {
		fmt.Printf("\n📱 测试用户 %d/%d: %s\n", i+1, len(userIDs), userID)
		fmt.Println("-" + string(make([]byte, 40)))

		// 获取指纹
		fingerprintConfig, err := manager.GetUserFingerprint(userID)
		if err != nil {
			log.Printf("❌ 获取指纹失败: %v", err)
			continue
		}

		// 显示指纹信息
		ua := fingerprintConfig.Browser.UserAgent
		if len(ua) > 50 {
			ua = ua[:50] + "..."
		}
		fmt.Printf("   🌐 User-Agent: %s\n", ua)
		fmt.Printf("   💻 Platform: %s\n", fingerprintConfig.Browser.Platform)
		fmt.Printf("   📺 Screen: %dx%d\n",
			fingerprintConfig.Screen.Width,
			fingerprintConfig.Screen.Height)
		fmt.Printf("   ⚙️  CPU Cores: %d\n", fingerprintConfig.Browser.HardwareConcurrency)

		// 创建浏览器实例
		ctx := context.Background()

		// 合并指纹参数和禁用恢复页面的参数
		args := fingerprintConfig.GetChromeFlags()
		args = append(args,
			"--disable-session-crashed-bubble",
			"--disable-infobars",
			"--no-first-run",
			"--no-default-browser-check",
		)

		opts := &browser.ConnectOptions{
			Headless:       false,
			PersistProfile: true,
			ProfileName:    fmt.Sprintf("fp_%s", userID),
			Args:           args,
		}

		instance, err := browser.Connect(ctx, opts)
		if err != nil {
			log.Printf("❌ 浏览器启动失败: %v", err)
			continue
		}

		page := instance.Page()

		// 注入指纹（指纹会在浏览器启动时自动应用）
		injector := browser.NewFingerprintInjector(fingerprintConfig)
		_ = injector // 指纹通过启动参数应用

		// 访问测试页面
		fmt.Println("   🌐 访问测试页面...")
		page.Navigate("https://abrahamjuliot.github.io/creepjs/")
		time.Sleep(8 * time.Second)

		fmt.Println("   ✅ 完成")

		// 关闭浏览器
		instance.Close()

		if i < len(userIDs)-1 {
			time.Sleep(2 * time.Second)
		}
	}

	fmt.Println("\n✅ 多用户指纹测试完成！")
}

// testFingerprintComparison 对比有无指纹的差异
func testFingerprintComparison() {
	fmt.Println("\n🔍 场景3：对比有无指纹的差异")
	fmt.Println("-" + string(make([]byte, 60)))

	ctx := context.Background()

	// 测试1：无指纹（原始浏览器）
	fmt.Println("\n1️⃣  测试：无指纹保护")
	fmt.Println("-" + string(make([]byte, 40)))

	opts1 := &browser.ConnectOptions{
		Headless:    false,
		ProfileName: "no_fingerprint",
		Args: []string{
			"--disable-session-crashed-bubble",
			"--disable-infobars",
			"--no-first-run",
			"--no-default-browser-check",
		},
	}

	instance1, err := browser.Connect(ctx, opts1)
	if err != nil {
		log.Printf("❌ 启动失败: %v", err)
		return
	}

	page1 := instance1.Page()
	page1.Navigate("https://abrahamjuliot.github.io/creepjs/")

	fmt.Println("⏳ 等待10秒查看结果...")
	time.Sleep(10 * time.Second)

	instance1.Close()

	time.Sleep(2 * time.Second)

	// 测试2：有指纹保护
	fmt.Println("\n2️⃣  测试：启用指纹保护")
	fmt.Println("-" + string(make([]byte, 40)))

	manager, _ := browser.NewUserFingerprintManager("./cmd/fingerprint_configs")
	fingerprintConfig, _ := manager.GetUserFingerprint("comparison_user")

	args2 := fingerprintConfig.GetChromeFlags()
	args2 = append(args2,
		"--disable-session-crashed-bubble",
		"--disable-infobars",
		"--no-first-run",
		"--no-default-browser-check",
	)

	opts2 := &browser.ConnectOptions{
		Headless:    false,
		ProfileName: "with_fingerprint",
		Args:        args2,
	}

	instance2, err := browser.Connect(ctx, opts2)
	if err != nil {
		log.Printf("❌ 启动失败: %v", err)
		return
	}

	page2 := instance2.Page()

	// 注入指纹（指纹会在浏览器启动时自动应用）
	injector := browser.NewFingerprintInjector(fingerprintConfig)
	_ = injector // 指纹通过启动参数应用

	page2.Navigate("https://abrahamjuliot.github.io/creepjs/")

	fmt.Println("⏳ 等待10秒查看结果...")
	time.Sleep(10 * time.Second)

	instance2.Close()

	fmt.Println("\n✅ 对比测试完成！")
	fmt.Println("\n💡 观察要点：")
	fmt.Println("   • Canvas指纹是否不同")
	fmt.Println("   • WebGL指纹是否不同")
	fmt.Println("   • Audio指纹是否不同")
	fmt.Println("   • 总体Trust Score的变化")
}

// checkFingerprint 检查指纹是否正确应用
func checkFingerprint(page browser.Page, config *browser.FingerprintConfig) {
	fmt.Println("   🔍 检查指纹是否生效...")

	// 检查User-Agent
	ua, err := page.Evaluate(`navigator.userAgent`)
	if err == nil {
		if uaStr, ok := ua.(string); ok {
			// 安全地比较前30个字符
			compareLen := min(30, min(len(config.Browser.UserAgent), len(uaStr)))
			if compareLen > 0 {
				expectedUA := config.Browser.UserAgent[:compareLen]
				actualUA := uaStr[:compareLen]
				if expectedUA == actualUA {
					fmt.Println("      ✅ User-Agent 已修改")
				} else {
					fmt.Println("      ⚠️  User-Agent 未生效")
				}
			}
		}
	}

	// 检查webdriver
	webdriver, err := page.Evaluate(`navigator.webdriver`)
	if err == nil {
		if webdriver == nil || webdriver == false {
			fmt.Println("      ✅ navigator.webdriver 已隐藏")
		} else {
			fmt.Println("      ⚠️  navigator.webdriver 仍然暴露")
		}
	}

	// 检查屏幕分辨率
	screen, err := page.Evaluate(`[screen.width, screen.height]`)
	if err == nil {
		fmt.Printf("      ✅ Screen: %v\n", screen)
	}

	// 检查硬件并发数
	cores, err := page.Evaluate(`navigator.hardwareConcurrency`)
	if err == nil {
		fmt.Printf("      ✅ CPU Cores: %v\n", cores)
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
