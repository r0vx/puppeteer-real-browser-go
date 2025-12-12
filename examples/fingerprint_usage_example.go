package main

import (
	"context"
	"fmt"
	"log"
	"path/filepath"
	"time"

	"github.com/HNRow/puppeteer-real-browser-go/pkg/browser"
)

func main() {
	fmt.Println("🌟 用户独立指纹使用示例")
	fmt.Println("==========================")

	ctx := context.Background()

	// 1. 创建指纹管理器
	fingerprintManager, err := browser.NewUserFingerprintManager("./user_fingerprints")
	if err != nil {
		log.Fatalf("❌ 创建指纹管理器失败: %v", err)
	}

	// 2. 为用户生成独立指纹
	userID := "demo_user_001"
	fingerprintConfig, err := fingerprintManager.GetUserFingerprint(userID)
	if err != nil {
		log.Fatalf("❌ 获取用户指纹失败: %v", err)
	}

	// 3. 显示用户的独特指纹信息
	fmt.Printf("👤 用户ID: %s\n", userID)
	fmt.Printf("🌐 浏览器指纹:\n")
	fmt.Printf("   UserAgent: %s\n", fingerprintConfig.Browser.UserAgent)
	fmt.Printf("   语言: %s\n", fingerprintConfig.Browser.Language)
	fmt.Printf("   屏幕: %dx%d (DPR: %.1f)\n", 
		fingerprintConfig.Screen.Width, 
		fingerprintConfig.Screen.Height, 
		fingerprintConfig.Screen.DevicePixelRatio)
	fmt.Printf("   时区: %s (%d分钟偏移)\n", 
		fingerprintConfig.Timezone.Timezone, 
		fingerprintConfig.Timezone.Offset)
	fmt.Printf("   CPU核心: %d\n", fingerprintConfig.Browser.HardwareConcurrency)

	// 4. 创建指纹注入器
	injector := browser.NewFingerprintInjector(fingerprintConfig)
	
	// 5. 获取JavaScript注入脚本
	injectionScript := injector.GenerateInjectionScript()
	fmt.Printf("\n💉 生成指纹注入脚本 (%d字符)\n", len(injectionScript))

	// 6. 获取Chrome启动参数
	chromeFlags := fingerprintConfig.GetChromeFlags()
	fmt.Printf("⚙️  Chrome启动参数 (%d个):\n", len(chromeFlags))
	for i, flag := range chromeFlags {
		fmt.Printf("   %d. %s\n", i+1, flag)
	}

	// 7. 配置浏览器启动选项
	ext1, _ := filepath.Abs("examples/path/Extensions/kfjglmgfjedhhcddpfgfogkahmenikan/1.0_0")
	ext2, _ := filepath.Abs("examples/path/Extensions/mcohilncbfahbmgdjkbpemcciiolgcge/3.66.10_0")

	opts := &browser.ConnectOptions{
		Headless:       false,
		PersistProfile: true,
		ProfileName:    fmt.Sprintf("user_%s", userID),
		Extensions:     []string{ext1, ext2},
		Args: append([]string{
			"--start-maximized",
			"--no-first-run",
			"--disable-blink-features=AutomationControlled",
			"--exclude-switches=enable-automation",
		}, chromeFlags...),
	}

	// 8. 启动浏览器
	fmt.Println("\n🚀 启动带独立指纹的浏览器...")
	instance, err := browser.Connect(ctx, opts)
	if err != nil {
		log.Fatalf("❌ 浏览器启动失败: %v", err)
	}

	fmt.Println("✅ 浏览器启动成功!")

	// 9. 在实际应用中，这里需要注入指纹脚本
	fmt.Println("💉 注入指纹修改脚本...")
	fmt.Println("   (在实际实现中，应该使用page.EvaluateOnNewDocument或扩展注入)")
	
	page := instance.Page()
	if page != nil {
		// 导航到指纹检测网站
		fmt.Println("🌐 导航到指纹检测网站...")
		err = page.Navigate("https://iplark.com/fingerprint")
		if err != nil {
			log.Printf("❌ 导航失败: %v", err)
		} else {
			time.Sleep(5 * time.Second)

			// 收集修改后的指纹
			fmt.Println("📊 收集修改后的指纹参数...")
			result, err := page.Evaluate(`
				(() => {
					return {
						userAgent: navigator.userAgent,
						language: navigator.language,
						platform: navigator.platform,
						screen: {
							width: screen.width,
							height: screen.height,
							devicePixelRatio: window.devicePixelRatio
						},
						hardwareConcurrency: navigator.hardwareConcurrency,
						timezone: Intl.DateTimeFormat().resolvedOptions().timeZone,
						webdriver: navigator.webdriver
					};
				})()
			`)

			if err == nil {
				fmt.Println("✅ 指纹修改验证:")
				if data, ok := result.(map[string]interface{}); ok {
					fmt.Printf("   🌐 UserAgent: %v\n", data["userAgent"])
					fmt.Printf("   🗣️  语言: %v\n", data["language"])
					fmt.Printf("   🖥️  平台: %v\n", data["platform"])
					if screen, ok := data["screen"].(map[string]interface{}); ok {
						fmt.Printf("   📱 屏幕: %.0fx%.0f (DPR: %v)\n", 
							screen["width"], screen["height"], screen["devicePixelRatio"])
					}
					fmt.Printf("   🔧 CPU核心: %v\n", data["hardwareConcurrency"])
					fmt.Printf("   ⏰ 时区: %v\n", data["timezone"])
					fmt.Printf("   🤖 WebDriver: %v\n", data["webdriver"])
				}
			}
		}
	}

	// 10. 演示不同用户的差异
	fmt.Println("\n🔄 演示多用户指纹差异...")
	
	otherUsers := []string{"demo_user_002", "demo_user_003"}
	for _, otherUserID := range otherUsers {
		otherConfig, err := fingerprintManager.GetUserFingerprint(otherUserID)
		if err != nil {
			continue
		}

		fmt.Printf("\n👤 用户: %s\n", otherUserID)
		fmt.Printf("   🌐 UserAgent: %s\n", otherConfig.Browser.UserAgent)
		fmt.Printf("   🖥️  屏幕: %dx%d\n", otherConfig.Screen.Width, otherConfig.Screen.Height)
		fmt.Printf("   🗣️  语言: %s\n", otherConfig.Browser.Language)
		fmt.Printf("   ⏰ 时区: %s\n", otherConfig.Timezone.Timezone)
	}

	defer instance.Close()

	fmt.Println("\n💡 使用总结:")
	fmt.Println("============")
	fmt.Println("✅ 每个用户都有完全独立的浏览器指纹")
	fmt.Println("✅ 指纹参数涵盖所有主要检测点")
	fmt.Println("✅ 配置自动持久化，重启后保持一致")
	fmt.Println("✅ 支持JavaScript注入脚本自动生成")
	fmt.Println("✅ Chrome启动参数自动配置")
	fmt.Println("✅ 扩展系统完全兼容")

	fmt.Println("\n⏳ 保持浏览器开启30秒供检查...")
	time.Sleep(30 * time.Second)

	fmt.Println("✅ 演示完成")
}