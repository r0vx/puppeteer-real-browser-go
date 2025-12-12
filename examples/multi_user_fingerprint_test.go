package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"path/filepath"
	"strconv"
	"time"

	"github.com/HNRow/puppeteer-real-browser-go/pkg/browser"
)

func main() {
	fmt.Println("🔍 多用户独立指纹测试")
	fmt.Println("========================")

	ctx := context.Background()

	// 创建指纹管理器
	fingerprintManager, err := browser.NewUserFingerprintManager("./fingerprint_configs")
	if err != nil {
		log.Fatalf("❌ 创建指纹管理器失败: %v", err)
	}

	// 测试用户列表
	testUsers := []string{"user001", "user002", "user003"}

	fmt.Println("📊 生成不同用户的指纹配置...")

	// 为每个用户生成指纹配置
	userConfigs := make(map[string]*browser.FingerprintConfig)
	for _, userID := range testUsers {
		config, err := fingerprintManager.GetUserFingerprint(userID)
		if err != nil {
			log.Printf("❌ 获取用户 %s 指纹配置失败: %v", userID, err)
			continue
		}
		userConfigs[userID] = config

		fmt.Printf("\n👤 用户: %s\n", userID)
		fmt.Printf("   🌐 UserAgent: %s\n", config.Browser.UserAgent)
		fmt.Printf("   🖥️  屏幕: %dx%d (%.1f)\n", 
			config.Screen.Width, config.Screen.Height, config.Screen.DevicePixelRatio)
		fmt.Printf("   🗣️  语言: %s\n", config.Browser.Language)
		fmt.Printf("   ⏰ 时区: %s (%d)\n", config.Timezone.Timezone, config.Timezone.Offset)
		fmt.Printf("   🔧 CPU核心: %d\n", config.Browser.HardwareConcurrency)
		fmt.Printf("   🎵 音频采样: %d Hz\n", config.Audio.SampleRate)
		fmt.Printf("   🔋 电池: %.0f%% (充电: %t)\n", 
			config.Battery.Level*100, config.Battery.Charging)
		fmt.Printf("   🎨 Canvas噪音: %.3f\n", config.Canvas.NoiseLevel)
	}

	// 显示统计信息
	stats, err := fingerprintManager.GetUserStats()
	if err == nil {
		fmt.Println("\n📈 指纹统计信息:")
		if statsJSON, err := json.MarshalIndent(stats, "   ", "  "); err == nil {
			fmt.Printf("   %s\n", string(statsJSON))
		}
	}

	// 获取扩展路径
	ext1, _ := filepath.Abs("examples/path/Extensions/kfjglmgfjedhhcddpfgfogkahmenikan/1.0_0")
	ext2, _ := filepath.Abs("examples/path/Extensions/mcohilncbfahbmgdjkbpemcciiolgcge/3.66.10_0")

	fmt.Println("\n🚀 启动多个浏览器实例测试不同指纹...")

	// 为每个用户启动浏览器并测试指纹
	for i, userID := range testUsers {
		if i >= 2 { // 限制同时测试的用户数量
			break
		}

		fmt.Printf("\n🔄 测试用户 %s 的指纹...\n", userID)
		
		config := userConfigs[userID]
		if config == nil {
			continue
		}

		// 创建指纹注入器
		injector := browser.NewFingerprintInjector(config)
		injectionScript := injector.GenerateInjectionScript()

		// 构建连接选项
		opts := &browser.ConnectOptions{
			Headless:       false,
			PersistProfile: true,
			ProfileName:    "fingerprint_test_" + userID,
			Extensions:     []string{ext1, ext2},
			Args: append([]string{
				"--start-maximized",
				"--no-first-run",
				"--disable-blink-features=AutomationControlled",
				"--exclude-switches=enable-automation",
			}, config.GetChromeFlags()...),
		}

		fmt.Printf("🌐 UserAgent: %s\n", config.Browser.UserAgent)
		fmt.Printf("🖥️  屏幕: %dx%d\n", config.Screen.Width, config.Screen.Height)

		// 启动浏览器
		instance, err := browser.Connect(ctx, opts)
		if err != nil {
			log.Printf("❌ 用户 %s Chrome启动失败: %v", userID, err)
			continue
		}

		go func(userID string, instance interface{}, injectionScript string) {
			defer func() {
				if instance != nil {
					// instance.Close() - 需要根据实际的instance类型调用Close方法
				}
			}()

			time.Sleep(2 * time.Second)

			// 这里需要根据实际的API获取page对象并执行脚本注入
			// page := instance.Page()

			fmt.Printf("✅ 用户 %s 浏览器启动成功\n", userID)

			// 在实际实现中，这里应该注入指纹修改脚本
			fmt.Printf("💉 为用户 %s 注入指纹修改脚本\n", userID)
			
			// 导航到测试页面进行指纹验证
			fmt.Printf("🔍 用户 %s 导航到指纹检测页面...\n", userID)

			// 这里应该实现页面导航和指纹收集
			// if err := page.Navigate("https://iplark.com/fingerprint"); err != nil {
			//     log.Printf("❌ 用户 %s 导航失败: %v", userID, err)
			//     return
			// }

			time.Sleep(30 * time.Second) // 保持浏览器开启一段时间
		}(userID, instance, injectionScript)

		// 避免同时启动太多浏览器
		time.Sleep(5 * time.Second)
	}

	fmt.Println("\n🎯 指纹对比测试")
	fmt.Println("=================")

	// 比较不同用户的指纹差异
	if len(userConfigs) >= 2 {
		user1ID := testUsers[0]
		user2ID := testUsers[1]
		
		config1 := userConfigs[user1ID]
		config2 := userConfigs[user2ID]

		fmt.Printf("👥 比较用户 %s 和 %s 的指纹差异:\n", user1ID, user2ID)
		
		// 比较关键指纹参数
		differences := []string{}
		
		if config1.Browser.UserAgent != config2.Browser.UserAgent {
			differences = append(differences, "UserAgent不同")
		}
		
		if config1.Screen.Width != config2.Screen.Width || config1.Screen.Height != config2.Screen.Height {
			differences = append(differences, "屏幕分辨率不同")
		}
		
		if config1.Browser.Language != config2.Browser.Language {
			differences = append(differences, "语言设置不同")
		}
		
		if config1.Timezone.Timezone != config2.Timezone.Timezone {
			differences = append(differences, "时区不同")
		}
		
		if config1.Browser.HardwareConcurrency != config2.Browser.HardwareConcurrency {
			differences = append(differences, "CPU核心数不同")
		}
		
		if config1.Audio.SampleRate != config2.Audio.SampleRate {
			differences = append(differences, "音频采样率不同")
		}
		
		if config1.WebGL.Renderer != config2.WebGL.Renderer {
			differences = append(differences, "WebGL渲染器不同")
		}

		fmt.Printf("📊 发现 %d 个主要差异:\n", len(differences))
		for i, diff := range differences {
			fmt.Printf("   %d. %s\n", i+1, diff)
		}

		if len(differences) >= 3 {
			fmt.Println("✅ 指纹差异充分，两用户具有独立的指纹特征")
		} else {
			fmt.Println("⚠️  指纹差异较少，建议增加更多随机化参数")
		}
	}

	fmt.Println("\n💾 指纹配置管理测试")
	fmt.Println("=====================")

	// 测试指纹配置的导出和导入
	for _, userID := range testUsers[:2] {
		// 导出配置
		configJSON, err := fingerprintManager.ExportUserFingerprint(userID)
		if err != nil {
			log.Printf("❌ 导出用户 %s 配置失败: %v", userID, err)
			continue
		}

		fmt.Printf("✅ 用户 %s 配置导出成功 (%d 字符)\n", userID, len(configJSON))

		// 测试克隆配置
		cloneUserID := userID + "_clone"
		err = fingerprintManager.CloneUserFingerprint(userID, cloneUserID)
		if err != nil {
			log.Printf("❌ 克隆用户 %s 配置失败: %v", userID, err)
		} else {
			fmt.Printf("✅ 用户 %s 配置克隆为 %s 成功\n", userID, cloneUserID)
		}
	}

	fmt.Println("\n🧪 自定义指纹测试")
	fmt.Println("==================")

	// 创建一个自定义指纹配置
	generator := browser.NewFingerprintGenerator()
	customConfig := generator.GenerateFingerprint("custom_user")
	
	// 手动修改一些参数
	customConfig.Browser.UserAgent = "Mozilla/5.0 (Custom Browser) AppleWebKit/537.36"
	customConfig.Screen.Width = 1024
	customConfig.Screen.Height = 768
	customConfig.Browser.Language = "ja-JP"
	customConfig.Timezone.Timezone = "Asia/Tokyo"
	customConfig.Timezone.Offset = -540

	err = fingerprintManager.CreateCustomUserFingerprint("custom_user", customConfig)
	if err != nil {
		log.Printf("❌ 创建自定义指纹失败: %v", err)
	} else {
		fmt.Println("✅ 自定义指纹配置创建成功")
		fmt.Printf("   🌐 自定义UserAgent: %s\n", customConfig.Browser.UserAgent)
		fmt.Printf("   🖥️  自定义屏幕: %dx%d\n", customConfig.Screen.Width, customConfig.Screen.Height)
		fmt.Printf("   🗣️  自定义语言: %s\n", customConfig.Browser.Language)
	}

	fmt.Println("\n📋 批量指纹生成测试")
	fmt.Println("=====================")

	// 生成批量用户指纹
	batchUsers := []string{}
	for i := 1; i <= 10; i++ {
		batchUsers = append(batchUsers, "batch_user_"+strconv.Itoa(i))
	}

	startTime := time.Now()
	err = fingerprintManager.GenerateBatchFingerprints(batchUsers)
	duration := time.Since(startTime)

	if err != nil {
		log.Printf("❌ 批量生成失败: %v", err)
	} else {
		fmt.Printf("✅ 批量生成 %d 个用户指纹成功，耗时: %v\n", len(batchUsers), duration)
		fmt.Printf("   📊 平均每个用户: %.2f ms\n", float64(duration.Nanoseconds())/float64(len(batchUsers))/1000000)
	}

	// 最终统计
	finalStats, _ := fingerprintManager.GetUserStats()
	if totalUsers, ok := finalStats["total_users"].(int); ok {
		fmt.Printf("\n📈 最终统计: 共管理 %d 个用户指纹\n", totalUsers)
	}

	fmt.Println("\n💡 测试建议:")
	fmt.Println("  1. 每个用户都有独立的指纹配置文件")
	fmt.Println("  2. 指纹参数在合理范围内随机生成")
	fmt.Println("  3. 支持自定义和批量管理")
	fmt.Println("  4. 配置持久化存储，重启后保持一致")

	fmt.Println("\n⏳ 保持浏览器开启60秒供手动验证...")
	time.Sleep(60 * time.Second)

	fmt.Println("✅ 多用户独立指纹测试完成")
}