package main

import (
	"context"
	"fmt"
	"log"
	"path/filepath"
	"strings"
	"time"

	"github.com/HNRow/puppeteer-real-browser-go/pkg/browser"
)

func main() {
	fmt.Println("🔬 真实指纹测试 - 验证实际效果")
	fmt.Println("==================================")

	ctx := context.Background()

	// 创建指纹管理器
	fingerprintManager, err := browser.NewUserFingerprintManager("./real_test_fingerprints")
	if err != nil {
		log.Fatalf("❌ 创建指纹管理器失败: %v", err)
	}

	// 测试2个用户
	testUsers := []string{"real_user_001", "real_user_002"}

	fmt.Println("🔧 为每个用户启动独立浏览器实例...")

	for i, userID := range testUsers {
		fmt.Printf("\n👤 测试用户: %s\n", userID)
		fmt.Println("=" + strings.Repeat("=", len(userID)+10))

		// 获取用户指纹配置
		config, err := fingerprintManager.GetUserFingerprint(userID)
		if err != nil {
			log.Printf("❌ 获取用户指纹失败: %v", err)
			continue
		}

		fmt.Printf("📊 预期指纹配置:\n")
		fmt.Printf("   JA4: %s\n", config.TLSConfig.JA4)
		fmt.Printf("   Akamai: %s\n", config.HTTP2Config.AKAMAI)
		fmt.Printf("   Audio采样率: %d Hz\n", config.Audio.SampleRate)
		fmt.Printf("   WebGL渲染器: %s\n", config.WebGL.Renderer)

		// 获取扩展路径
		ext1, _ := filepath.Abs("examples/path/Extensions/kfjglmgfjedhhcddpfgfogkahmenikan/1.0_0")
		ext2, _ := filepath.Abs("examples/path/Extensions/mcohilncbfahbmgdjkbpemcciiolgcge/3.66.10_0")

		// 创建指纹注入器
		injector := browser.NewFingerprintInjector(config)
		injectionScript := injector.GenerateInjectionScript()

		// 配置Chrome启动参数
		chromeFlags := config.GetChromeFlags()
		
		opts := &browser.ConnectOptions{
			Headless:       false,
			PersistProfile: true,
			ProfileName:    fmt.Sprintf("real_test_%s", userID),
			Extensions:     []string{ext1, ext2},
			Args: append([]string{
				"--start-maximized",
				"--no-first-run",
				"--disable-blink-features=AutomationControlled",
				"--exclude-switches=enable-automation",
			}, chromeFlags...),
		}

		fmt.Printf("🚀 启动用户 %s 的Chrome...\n", userID)
		instance, err := browser.Connect(ctx, opts)
		if err != nil {
			log.Printf("❌ Chrome启动失败: %v", err)
			continue
		}

		go func(userID string, instance interface{}, injectionScript string) {
			defer func() {
				fmt.Printf("🔚 用户 %s 浏览器测试完成\n", userID)
			}()

			time.Sleep(3 * time.Second)

			page := instance.Page()
			if page == nil {
				fmt.Printf("❌ 用户 %s 无法获取页面对象\n", userID)
				return
			}

			fmt.Printf("💉 用户 %s 注入指纹脚本...\n", userID)
			
			// 在导航前注入脚本
			err := page.EvaluateOnNewDocument(injectionScript)
			if err != nil {
				fmt.Printf("❌ 用户 %s 脚本注入失败: %v\n", userID, err)
			} else {
				fmt.Printf("✅ 用户 %s 脚本注入成功\n", userID)
			}

			fmt.Printf("🌐 用户 %s 导航到指纹检测网站...\n", userID)
			err = page.Navigate("https://iplark.com/fingerprint")
			if err != nil {
				fmt.Printf("❌ 用户 %s 导航失败: %v\n", userID, err)
				return
			}

			// 等待页面加载
			time.Sleep(15 * time.Second)

			fmt.Printf("📊 收集用户 %s 的实际指纹...\n", userID)
			result, err := page.Evaluate(`
				(() => {
					const fingerprint = {};
					
					// 基本信息
					fingerprint.userAgent = navigator.userAgent;
					fingerprint.language = navigator.language;
					fingerprint.platform = navigator.platform;
					fingerprint.hardwareConcurrency = navigator.hardwareConcurrency;
					fingerprint.webdriver = navigator.webdriver;
					
					// 屏幕信息
					fingerprint.screen = {
						width: screen.width,
						height: screen.height,
						devicePixelRatio: window.devicePixelRatio
					};
					
					// WebGL信息
					try {
						const canvas = document.createElement('canvas');
						const gl = canvas.getContext('webgl') || canvas.getContext('experimental-webgl');
						if (gl) {
							fingerprint.webgl = {
								vendor: gl.getParameter(gl.VENDOR),
								renderer: gl.getParameter(gl.RENDERER),
								version: gl.getParameter(gl.VERSION),
								maxTextureSize: gl.getParameter(gl.MAX_TEXTURE_SIZE)
							};
						} else {
							fingerprint.webgl = { error: "WebGL不可用" };
						}
					} catch (e) {
						fingerprint.webgl = { error: e.message };
					}
					
					// 音频指纹
					try {
						const AudioContext = window.AudioContext || window.webkitAudioContext;
						if (AudioContext) {
							const audioCtx = new AudioContext();
							fingerprint.audio = {
								sampleRate: audioCtx.sampleRate,
								maxChannelCount: audioCtx.destination.maxChannelCount
							};
							audioCtx.close();
						}
					} catch (e) {
						fingerprint.audio = { error: e.message };
					}
					
					// Canvas指纹
					try {
						const canvas = document.createElement('canvas');
						const ctx = canvas.getContext('2d');
						ctx.textBaseline = 'top';
						ctx.font = '14px Arial';
						ctx.fillText('Fingerprint test ' + Date.now(), 2, 2);
						fingerprint.canvasHash = canvas.toDataURL().substring(0, 100);
					} catch (e) {
						fingerprint.canvasHash = { error: e.message };
					}
					
					// 时区信息
					fingerprint.timezone = {
						offset: new Date().getTimezoneOffset(),
						timezone: Intl.DateTimeFormat().resolvedOptions().timeZone
					};
					
					return fingerprint;
				})()
			`)

			if err != nil {
				fmt.Printf("❌ 用户 %s 指纹收集失败: %v\n", userID, err)
				return
			}

			fmt.Printf("📋 用户 %s 实际指纹结果:\n", userID)
			if data, ok := result.(map[string]interface{}); ok {
				fmt.Printf("   🌐 UserAgent: %v\n", data["userAgent"])
				fmt.Printf("   🗣️  语言: %v\n", data["language"]) 
				fmt.Printf("   🖥️  平台: %v\n", data["platform"])
				fmt.Printf("   🔧 CPU核心: %v\n", data["hardwareConcurrency"])
				fmt.Printf("   🤖 WebDriver: %v\n", data["webdriver"])
				
				if screen, ok := data["screen"].(map[string]interface{}); ok {
					fmt.Printf("   📱 屏幕: %.0fx%.0f (DPR: %v)\n", 
						screen["width"], screen["height"], screen["devicePixelRatio"])
				}
				
				if webgl, ok := data["webgl"].(map[string]interface{}); ok {
					if webgl["error"] != nil {
						fmt.Printf("   🎨 WebGL: ❌ %v\n", webgl["error"])
					} else {
						fmt.Printf("   🎨 WebGL厂商: %v\n", webgl["vendor"])
						fmt.Printf("   🎨 WebGL渲染器: %v\n", webgl["renderer"])
						fmt.Printf("   🎨 最大纹理: %v\n", webgl["maxTextureSize"])
					}
				}
				
				if audio, ok := data["audio"].(map[string]interface{}); ok {
					if audio["error"] != nil {
						fmt.Printf("   🎵 Audio: ❌ %v\n", audio["error"])
					} else {
						fmt.Printf("   🎵 音频采样率: %v Hz\n", audio["sampleRate"])
						fmt.Printf("   🎵 最大通道数: %v\n", audio["maxChannelCount"])
					}
				}
				
				if canvasHash, ok := data["canvasHash"].(string); ok {
					fmt.Printf("   🎨 Canvas哈希: %s...\n", canvasHash)
				}
				
				if timezone, ok := data["timezone"].(map[string]interface{}); ok {
					fmt.Printf("   ⏰ 时区: %v (偏移: %v)\n", timezone["timezone"], timezone["offset"])
				}
			}

			// 尝试检测网络层指纹 (这些无法通过JS修改)
			fmt.Printf("⚠️  用户 %s 网络层指纹说明:\n", userID)
			fmt.Println("   🔐 JA4指纹: 由浏览器TLS握手决定，JavaScript无法修改")
			fmt.Println("   🌐 HTTP2指纹: 由浏览器HTTP2实现决定，JavaScript无法修改")
			fmt.Println("   📡 这些指纹需要在浏览器内核或网络代理层面修改")

		}(userID, instance, injectionScript)

		// 延迟启动下一个浏览器
		if i < len(testUsers)-1 {
			time.Sleep(5 * time.Second)
		}
	}

	fmt.Println("\n🔍 问题分析")
	fmt.Println("============")
	fmt.Println("❗ JavaScript能修改的指纹:")
	fmt.Println("   ✅ Navigator对象 (userAgent, language, platform等)")
	fmt.Println("   ✅ Screen对象 (width, height, devicePixelRatio等)")
	fmt.Println("   ✅ WebGL上下文信息 (vendor, renderer等)")
	fmt.Println("   ✅ AudioContext属性 (sampleRate, channelCount等)")
	fmt.Println("   ✅ Canvas指纹 (通过噪音)")
	fmt.Println("   ✅ 时区信息")
	
	fmt.Println("\n❌ JavaScript无法修改的指纹:")
	fmt.Println("   🔐 JA4/JA3指纹 (TLS握手层)")
	fmt.Println("   🌐 HTTP2指纹/Akamai指纹 (HTTP2协议层)")
	fmt.Println("   📡 TCP指纹")
	fmt.Println("   🔒 证书指纹")

	fmt.Println("\n💡 解决方案建议")
	fmt.Println("================")
	fmt.Println("🔧 方案1: 网络代理")
	fmt.Println("   - 使用HTTP/HTTPS代理修改网络层指纹")
	fmt.Println("   - 在代理层实现TLS指纹伪装")
	fmt.Println("   - 修改HTTP2头部和设置")
	
	fmt.Println("\n🔧 方案2: 浏览器定制")
	fmt.Println("   - 编译定制版Chromium")
	fmt.Println("   - 修改TLS和HTTP2实现")
	fmt.Println("   - 成本高但效果最好")
	
	fmt.Println("\n🔧 方案3: 混合方案")
	fmt.Println("   - JavaScript修改浏览器层指纹")
	fmt.Println("   - 网络代理修改协议层指纹")
	fmt.Println("   - 达到最佳指纹隔离效果")

	fmt.Println("\n⏳ 保持浏览器开启60秒供检查...")
	time.Sleep(60 * time.Second)

	fmt.Println("✅ 真实指纹测试完成")
	fmt.Println("\n📊 结论: 当前系统可以修改JavaScript层指纹，")
	fmt.Println("但JA4、HTTP2等网络层指纹需要额外的网络层解决方案。")
}