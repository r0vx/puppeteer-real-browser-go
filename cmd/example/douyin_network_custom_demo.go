//go:build ignore
// +build ignore

package main

import (
	"context"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/r0vx/puppeteer-real-browser-go/pkg/browser"
)

func main() {
	fmt.Println("🎯 抖音二维码 API 监听 (CustomCDP)")
	fmt.Println("=====================================")
	fmt.Println("监听: get_qrcode, check_qrconnect")
	fmt.Println("⚡ 使用默认的 CustomCDP 通道")
	fmt.Println()

	ctx := context.Background()

	opts := &browser.ConnectOptions{
		Headless:          false,
		FingerprintUserID: "douyin_qrcode_test",
		FingerprintDir:    "./fingerprints",
		Language:          "zh-CN",
		Languages:         []string{"zh-CN", "zh", "en"},
		UserAgent:         "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36",
		Width:             1920,
		Height:            1080,
		Timezone:          "Asia/Shanghai",
		TimezoneOffset:    480,
		Args: []string{
			"--window-size=1920,1080",
			"--start-maximized"},
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
		log.Fatal("❌ 需要 CustomCDP 通道（不要设 UseChromedp）才能使用此功能")
	}

	// 存储请求信息
	var requestsMu sync.Mutex
	requests := make(map[string]struct {
		URL    string
		Method string
	})

	// 监听网络请求
	customPage.OnNetworkRequest(func(requestID, url, method string) {
		if method == "OPTIONS" || method == "HEAD" {
			return
		}

		// 过滤 get_qrcode 和 check_qrconnect 请求
		if strings.Contains(url, "get_qrcode") || strings.Contains(url, "check_qrconnect") {
			// 提取 API 名称
			apiName := "unknown"
			if strings.Contains(url, "get_qrcode") {
				apiName = "get_qrcode"
			} else if strings.Contains(url, "check_qrconnect") {
				apiName = "check_qrconnect"
			}
			fmt.Printf("\n🎯 [%s] 捕获请求: [%s] \n", apiName, method)
			requestsMu.Lock()
			requests[requestID] = struct {
				URL    string
				Method string
			}{URL: url, Method: method}
			requestsMu.Unlock()
		}
	})

	// 监听加载完成
	customPage.OnNetworkLoadingFinished(func(requestID string) {
		requestsMu.Lock()
		req, exists := requests[requestID]
		delete(requests, requestID)
		requestsMu.Unlock()

		if exists {
			// 获取响应体
			body, err := customPage.GetResponseBody(requestID)
			if err != nil {
				fmt.Printf("⚠️ 获取响应失败: %v\n", err)
				return
			}

			// 提取 API 名称
			apiName := "unknown"
			if strings.Contains(req.URL, "get_qrcode") {
				apiName = "🔑 get_qrcode"
			} else if strings.Contains(req.URL, "check_qrconnect") {
				apiName = "🔄 check_qrconnect"
			}

			fmt.Println("\n" + strings.Repeat("=", 70))
			fmt.Printf("📦 %s 响应\n", apiName)
			fmt.Println(strings.Repeat("=", 70))
			fmt.Printf("URL: %s\n", req.URL)
			fmt.Printf("Method: %s\n", req.Method)
			fmt.Printf("Body: %d bytes\n", len(body))
			fmt.Println(strings.Repeat("-", 70))
			// 显示完整 JSON（格式化输出）
			fmt.Println(string(body))
			fmt.Println(strings.Repeat("=", 70))
		}
	})

	// 启用网络监听 - 必须在导航前启用!
	if err := customPage.EnableNetwork(); err != nil {
		log.Fatalf("❌ 启用网络监听失败: %v", err)
	}
	fmt.Println("✅ 网络监听已启用!")

	// 导航到抖音
	fmt.Println("\n📂 导航到抖音...")
	if err := page.Navigate("https://www.douyin.com/user/self"); err != nil {
		log.Fatalf("❌ 导航失败: %v", err)
	}

	fmt.Println("\n⏳ 等待二维码 API... (120秒)")
	fmt.Println("💡 提示: get_qrcode 获取二维码，check_qrconnect 检查扫码状态")
	fmt.Println("📱 请使用抖音 APP 扫描二维码登录")
	time.Sleep(2000 * time.Second)

	fmt.Println("\n✅ 测试完成!")
}
