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
	fmt.Println("监听: web/get_qrcode/")
	fmt.Println("⚡ 使用 UseCustomCDP: true")
	fmt.Println()

	ctx := context.Background()

	opts := &browser.ConnectOptions{
		Headless:     false,
		UseCustomCDP: true, // 使用自定义 CDP
		Args:         []string{"--window-size=1280,720"},
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
		log.Fatal("❌ 需要 UseCustomCDP: true 才能使用此功能")
	}

	// 存储请求信息
	var requestsMu sync.Mutex
	requests := make(map[string]struct {
		URL    string
		Method string
	})

	// 监听网络请求
	customPage.OnNetworkRequest(func(requestID, url, method string) {
		// 过滤 web/get_qrcode/ 请求
		if strings.Contains(url, "web/get_qrcode") {
			fmt.Printf("\n🎯 捕获请求: [%s] %s\n", method, url)
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

			fmt.Println("\n" + strings.Repeat("=", 60))
			fmt.Printf("📦 二维码 API 响应\n")
			fmt.Println(strings.Repeat("=", 60))
			fmt.Printf("URL: %s\n", req.URL)
			fmt.Printf("Method: %s\n", req.Method)
			fmt.Printf("Body: %d bytes\n", len(body))
			fmt.Println(strings.Repeat("-", 60))
			// 只显示前 500 字符
			if len(body) > 500 {
				fmt.Printf("%s...(truncated)\n", string(body[:500]))
			} else {
				fmt.Println(string(body))
			}
			fmt.Println(strings.Repeat("=", 60))
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

	fmt.Println("\n⏳ 等待二维码 API... (60秒)")
	fmt.Println("💡 提示: 二维码会自动刷新，每次刷新都会触发 API")
	fmt.Println("⚠️ 注意: UseCustomCDP: true 可能丢失部分请求")
	time.Sleep(60 * time.Second)

	fmt.Println("\n✅ 测试完成!")
}
