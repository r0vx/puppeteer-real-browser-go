//go:build ignore
// +build ignore

package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/r0vx/puppeteer-real-browser-go/pkg/browser"
)

func main() {
	fmt.Println("🔬 指纹保护测试")
	fmt.Println("=====================================")
	fmt.Println("测试项: Canvas, WebGL, Audio, Font, Battery")
	fmt.Println()

	ctx := context.Background()

	opts := &browser.ConnectOptions{
		Headless:     false,
		Args:         []string{"--window-size=1280,800"},
	}

	fmt.Println("🚀 启动浏览器...")
	instance, err := browser.Connect(ctx, opts)
	if err != nil {
		log.Fatalf("❌ 连接失败: %v", err)
	}
	defer instance.Close()

	page := instance.Page()

	// 先导航到一个页面，触发 stealth 脚本注入
	fmt.Println("📂 导航到测试页面...")
	if err := page.Navigate("about:blank"); err != nil {
		log.Printf("⚠️ 导航失败: %v", err)
	}
	time.Sleep(500 * time.Millisecond)

	// 测试 1: Canvas 指纹
	fmt.Println("\n📊 测试 1: Canvas 指纹")
	fmt.Println("-" + string(make([]byte, 40)))

	canvasScript := `
	(function() {
		const canvas = document.createElement('canvas');
		canvas.width = 200;
		canvas.height = 50;
		const ctx = canvas.getContext('2d');
		
		// 绘制测试图案
		ctx.fillStyle = 'rgb(255,0,0)';
		ctx.fillRect(0, 0, 100, 50);
		ctx.fillStyle = 'rgb(0,255,0)';
		ctx.fillRect(100, 0, 100, 50);
		ctx.font = '18px Arial';
		ctx.fillStyle = 'blue';
		ctx.fillText('Fingerprint Test', 10, 30);
		
		return canvas.toDataURL().substring(0, 100) + '...';
	})()
	`

	for i := 1; i <= 3; i++ {
		result, err := page.Evaluate(canvasScript)
		if err != nil {
			fmt.Printf("   ❌ 执行失败: %v\n", err)
		} else {
			fmt.Printf("   第 %d 次: %v\n", i, result)
		}
		time.Sleep(100 * time.Millisecond)
	}

	// 测试 1.5: 验证 stealth 脚本注入
	fmt.Println("\n📊 测试 1.5: Stealth 脚本注入验证")
	fmt.Println("-" + string(make([]byte, 40)))

	stealthCheck := `
	(function() {
		return {
			stealthInjected: window.__stealthInjected === true,
			webdriverHidden: navigator.webdriver === undefined || navigator.webdriver === false
		};
	})()
	`
	stealthResult, stealthErr := page.Evaluate(stealthCheck)
	if stealthErr != nil {
		fmt.Printf("   ❌ 执行失败: %v\n", stealthErr)
	} else {
		fmt.Printf("   Stealth 状态: %v\n", stealthResult)
	}

	// 测试 2: WebGL 指纹
	fmt.Println("\n📊 测试 2: WebGL 指纹")
	fmt.Println("-" + string(make([]byte, 40)))

	webglScript := `
	(function() {
		const canvas = document.createElement('canvas');
		const gl = canvas.getContext('webgl') || canvas.getContext('experimental-webgl');
		if (!gl) return 'WebGL not supported';
		
		const debugInfo = gl.getExtension('WEBGL_debug_renderer_info');
		if (!debugInfo) return 'Debug info not available';
		
		// 直接读取参数值
		const vendor = gl.getParameter(37445);
		const renderer = gl.getParameter(37446);
		
		return {
			vendor: vendor,
			renderer: renderer,
			getParameterType: typeof gl.getParameter
		};
	})()
	`

	webglResult, webglErr := page.Evaluate(webglScript)
	if webglErr != nil {
		fmt.Printf("   ❌ 执行失败: %v\n", webglErr)
	} else {
		fmt.Printf("   WebGL 信息: %v\n", webglResult)
	}

	// 测试 3: Audio 指纹
	fmt.Println("\n📊 测试 3: Audio 指纹")
	fmt.Println("-" + string(make([]byte, 40)))

	audioScript := `
	(function() {
		try {
			const AudioContext = window.AudioContext || window.webkitAudioContext;
			if (!AudioContext) return 'AudioContext not supported';
			
			const ctx = new AudioContext();
			const analyser = ctx.createAnalyser();
			const oscillator = ctx.createOscillator();
			
			return {
				sampleRate: ctx.sampleRate,
				analyserFftSize: analyser.fftSize,
				oscillatorFreq: oscillator.frequency.value.toFixed(4)
			};
		} catch(e) {
			return 'Error: ' + e.message;
		}
	})()
	`

	for i := 1; i <= 3; i++ {
		result, err := page.Evaluate(audioScript)
		if err != nil {
			fmt.Printf("   ❌ 执行失败: %v\n", err)
		} else {
			fmt.Printf("   第 %d 次: %v\n", i, result)
		}
		time.Sleep(100 * time.Millisecond)
	}

	// 测试 4: navigator 属性
	fmt.Println("\n📊 测试 4: Navigator 属性")
	fmt.Println("-" + string(make([]byte, 40)))

	navScript := `
	(function() {
		return {
			webdriver: navigator.webdriver,
			plugins: navigator.plugins.length,
			languages: navigator.languages,
			hardwareConcurrency: navigator.hardwareConcurrency,
			deviceMemory: navigator.deviceMemory,
			vendor: navigator.vendor,
			maxTouchPoints: navigator.maxTouchPoints
		};
	})()
	`

	navResult, navErr := page.Evaluate(navScript)
	if navErr != nil {
		fmt.Printf("   ❌ 执行失败: %v\n", navErr)
	} else {
		fmt.Printf("   Navigator: %v\n", navResult)
	}

	// 测试 5: Battery API
	fmt.Println("\n📊 测试 5: Battery API")
	fmt.Println("-" + string(make([]byte, 40)))

	batteryScript := `
	(function() {
		if (!navigator.getBattery) return 'Battery API not available';
		return navigator.getBattery().then(b => ({
			charging: b.charging,
			level: b.level,
			chargingTime: b.chargingTime,
			dischargingTime: b.dischargingTime
		}));
	})()
	`

	batteryResult, batteryErr := page.Evaluate(batteryScript)
	if batteryErr != nil {
		fmt.Printf("   ❌ 执行失败: %v\n", batteryErr)
	} else {
		fmt.Printf("   Battery: %v\n", batteryResult)
	}

	// 测试 6: 窗口尺寸
	fmt.Println("\n📊 测试 6: 窗口尺寸")
	fmt.Println("-" + string(make([]byte, 40)))

	windowScript := `
	(function() {
		return {
			innerWidth: window.innerWidth,
			innerHeight: window.innerHeight,
			outerWidth: window.outerWidth,
			outerHeight: window.outerHeight,
			screenX: window.screenX,
			screenY: window.screenY
		};
	})()
	`

	windowResult, windowErr := page.Evaluate(windowScript)
	if windowErr != nil {
		fmt.Printf("   ❌ 执行失败: %v\n", windowErr)
	} else {
		fmt.Printf("   Window: %v\n", windowResult)
	}

	// 导航到指纹检测网站
	fmt.Println("\n🌐 导航到指纹检测网站...")
	if err := page.Navigate("https://browserleaks.com/canvas"); err != nil {
		log.Printf("⚠️ 导航失败: %v", err)
	}

	fmt.Println("⏳ 等待页面加载 (5秒)...")
	time.Sleep(5 * time.Second)

	// 截图
	if screenshot, err := page.Screenshot(); err == nil {
		os.WriteFile("fingerprint_test.png", screenshot, 0644)
		fmt.Println("📸 已保存截图: fingerprint_test.png")
	}

	fmt.Println("\n💡 请在浏览器中查看 Canvas 指纹检测结果")
	fmt.Println("⏳ 保持运行 30 秒供查看...")
	time.Sleep(30 * time.Second)

	fmt.Println("\n✅ 测试完成!")
}

