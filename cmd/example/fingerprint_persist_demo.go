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
	fmt.Println("🔐 指纹持久化测试")
	fmt.Println("=====================================")
	fmt.Println()

	// 模拟用户ID
	userID := "douyin_12345"
	fingerprintDir := "./fingerprints"

	// 使用简化的方式启动浏览器
	fmt.Println("🚀 使用 FingerprintUserID 启动浏览器...")
	fmt.Printf("   UserID: %s\n", userID)
	fmt.Printf("   FingerprintDir: %s\n", fingerprintDir)

	ctx := context.Background()
	opts := &browser.ConnectOptions{
		Headless:          false,
		FingerprintUserID: userID,         // 只需指定 UserID
		FingerprintDir:    fingerprintDir, // 可选，默认 ./fingerprints
		// 初始化参数 - 首次创建指纹时使用，后续加载不会覆盖
		Width:     1920,
		Height:    1080,
		UserAgent: "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) Custom UA",
		Args:      []string{"--window-size=1920,1080"},
	}

	instance, err := browser.Connect(ctx, opts)
	if err != nil {
		log.Fatalf("❌ 连接失败: %v", err)
	}
	defer instance.Close()

	page := instance.Page()

	// 导航到测试页面
	if err := page.Navigate("about:blank"); err != nil {
		log.Printf("⚠️ 导航失败: %v", err)
	}
	time.Sleep(500 * time.Millisecond)

	// 验证指纹是否应用
	fmt.Println("\n📊 验证浏览器指纹...")

	// 检查 WebGL
	webglResult, _ := page.Evaluate(`
		(function() {
			const canvas = document.createElement('canvas');
			const gl = canvas.getContext('webgl');
			if (!gl) return 'WebGL not supported';
			const debugInfo = gl.getExtension('WEBGL_debug_renderer_info');
			if (!debugInfo) return 'Debug info not available';
			return {
				vendor: gl.getParameter(37445),
				renderer: gl.getParameter(37446)
			};
		})()
	`)
	fmt.Printf("   WebGL: %v\n", webglResult)

	// 检查 Navigator
	navResult, _ := page.Evaluate(`
		(function() {
			return {
				language: navigator.language,
				languages: navigator.languages,
				platform: navigator.platform,
				hardwareConcurrency: navigator.hardwareConcurrency
			};
		})()
	`)
	fmt.Printf("   Navigator: %v\n", navResult)

	// 检查配置文件是否已创建
	configFile := fmt.Sprintf("%s/%s.json", fingerprintDir, userID)
	if _, err := os.Stat(configFile); err == nil {
		fmt.Printf("\n💾 配置已保存到: %s\n", configFile)
	}

	fmt.Println("\n⏳ 保持浏览器运行 10 秒...")
	time.Sleep(10 * time.Second)

	// 显示不同用户的指纹差异
	fmt.Println("\n📊 测试不同用户的指纹差异...")
	manager, _ := browser.NewUserFingerprintManager(fingerprintDir)
	users := []string{"douyin_001", "douyin_002", "douyin_003"}
	for _, uid := range users {
		cfg, _ := manager.GetUserFingerprint(uid)
		renderer := cfg.WebGL.Renderer
		if len(renderer) > 30 {
			renderer = renderer[:30] + "..."
		}
		fmt.Printf("   [%s] WebGL: %s, Screen: %dx%d\n",
			uid, renderer, cfg.Screen.Width, cfg.Screen.Height)
	}

	fmt.Println("\n✅ 测试完成！")
	fmt.Printf("📁 所有指纹配置保存在: %s/\n", fingerprintDir)
}

