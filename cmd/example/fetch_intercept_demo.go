//go:build ignore
// +build ignore

package main

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/r0vx/puppeteer-real-browser-go/pkg/browser"
)

func main() {
	fmt.Println("🎯 Fetch 请求拦截测试")
	fmt.Println("=====================================")
	fmt.Println("目标: 拦截 wlog.gifshow.com 请求")
	fmt.Println("修改: kpn=1 → kpn=2")
	fmt.Println()

	ctx := context.Background()

	opts := &browser.ConnectOptions{
		Headless:     false,
		UseCustomCDP: true,
		Args: []string{
			"--window-size=1280,800",
		},
	}

	fmt.Println("🚀 启动浏览器...")
	instance, err := browser.Connect(ctx, opts)
	if err != nil {
		log.Fatalf("❌ 连接失败: %v", err)
	}
	defer instance.Close()

	page := instance.Page()

	// 类型断言获取 CustomCDPPage
	customPage, ok := page.(*browser.CustomCDPPage)
	if !ok {
		log.Fatal("❌ 需要 UseCustomCDP: true")
	}

	// 1. 启用网络监听（用于对比）
	if err := customPage.EnableNetwork(); err != nil {
		log.Fatalf("❌ 启用网络监听失败: %v", err)
	}

	// 2. 启用 Fetch 拦截
	// 拦截包含 wlog.gifshow.com 的请求
	patterns := []string{
		"*wlog.gifshow.com*",
		"*log/collect*",
	}
	if err := customPage.EnableFetch(patterns); err != nil {
		log.Fatalf("❌ 启用 Fetch 拦截失败: %v", err)
	}
	fmt.Println("✅ Fetch 拦截已启用!")
	fmt.Printf("   拦截模式: %v\n", patterns)

	// 3. 设置拦截处理器
	customPage.OnRequestPaused(func(requestID, url, method, postData string, headers map[string]string) {
		fmt.Println("\n" + strings.Repeat("=", 70))
		fmt.Println("🚫 请求被拦截!")
		fmt.Println(strings.Repeat("=", 70))
		fmt.Printf("RequestID: %s\n", requestID)
		fmt.Printf("Method: %s\n", method)
		fmt.Printf("原始 URL: %s\n", url)

		// 检查是否需要修改
		if strings.Contains(url, "kpn=1") {
			// 修改 URL: kpn=1 → kpn=2
			newURL := strings.Replace(url, "kpn=1", "kpn=2", 1)
			fmt.Printf("✏️  修改后 URL: %s\n", newURL)
			fmt.Println("📤 继续请求（使用修改后的 URL）...")

			if err := customPage.ContinueRequest(requestID, newURL); err != nil {
				fmt.Printf("⚠️ 继续请求失败: %v\n", err)
				// 失败时继续原请求
				customPage.ContinueRequest(requestID, "")
			}
		} else {
			fmt.Println("📤 继续请求（未修改）...")
			if err := customPage.ContinueRequest(requestID, ""); err != nil {
				fmt.Printf("⚠️ 继续请求失败: %v\n", err)
			}
		}
		fmt.Println(strings.Repeat("=", 70))
	})

	// 4. 监听网络响应（验证修改是否生效）
	customPage.OnNetworkResponse(func(requestID, url string, status int) {
		if strings.Contains(url, "wlog.gifshow.com") || strings.Contains(url, "log/collect") {
			fmt.Printf("\n📥 响应: [%d] %s\n", status, url)
		}
	})

	// 5. 导航到目标页面
	fmt.Println("\n📂 导航到快手充值页面...")
	if err := page.Navigate("https://pay.ssl.kuaishou.com/pay?source=NewReco"); err != nil {
		log.Fatalf("❌ 导航失败: %v", err)
	}

	fmt.Println("\n⏳ 等待请求... (60秒)")
	fmt.Println("💡 提示: 观察是否有 wlog.gifshow.com 请求被拦截并修改")
	time.Sleep(60 * time.Second)

	// 6. 禁用 Fetch
	if err := customPage.DisableFetch(); err != nil {
		fmt.Printf("⚠️ 禁用 Fetch 失败: %v\n", err)
	}

	fmt.Println("\n✅ 测试完成!")
}

