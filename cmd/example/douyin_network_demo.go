//go:build ignore
// +build ignore

package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/chromedp/cdproto/network"
	"github.com/chromedp/chromedp"
	"github.com/r0vx/puppeteer-real-browser-go/pkg/browser"
)

// EvenItem 事件项
type EvenItem struct {
	RequestID network.RequestID
	Name      string
	Method    string
	URL       string
}

// ChanResponse 响应通道数据
type ChanResponse struct {
	Name    string
	Method  string
	URL     string
	Message string
	Data    []byte
}

func main() {
	fmt.Println("🎯 抖音充值页面 - 监听指定 API 响应")
	fmt.Println("=====================================")
	fmt.Println()

	ctx := context.Background()

	opts := &browser.ConnectOptions{
		Headless:    false,
		UseChromedp: true, // 本示例直接调用 chromedp（GetContext + chromedp.ListenTarget），需要旧通道
		Turnstile:   false,
		Args: []string{
			"--window-size=1920,1080",
		},
	}

	fmt.Println("🚀 启动浏览器...")
	instance, err := browser.Connect(ctx, opts)
	if err != nil {
		log.Fatalf("❌ 连接失败: %v", err)
	}
	defer instance.Close()

	page := instance.Page()

	// 获取 chromedp 上下文
	type contextGetter interface {
		GetContext() context.Context
	}
	cdpPage, ok := page.(contextGetter)
	if !ok {
		log.Fatal("❌ 无法获取 chromedp 上下文")
	}
	chromedpCtx := cdpPage.GetContext()

	// 响应通道
	chanResponse := make(chan ChanResponse, 10)

	// 启动网络监听
	listenNetwork(chromedpCtx, chanResponse)

	// 启用 Network 域
	if err := chromedp.Run(chromedpCtx, network.Enable()); err != nil {
		log.Fatalf("❌ 启用 Network 失败: %v", err)
	}
	fmt.Println("✅ 网络监听已启用!")

	// 导航到页面
	fmt.Println("\n📂 导航到抖音充值页面...")
	if err := page.Navigate("https://www.douyin.com/user/self"); err != nil {
		log.Fatalf("❌ 导航失败: %v", err)
	}

	// 等待目标 API 响应
	fmt.Println("⏳ 等待目标 API 响应...")
	fmt.Println("💡 提示: 需要登录后操作充值才会触发")

	for {
		select {
		case resp := <-chanResponse:
			if resp.Name == "error" {
				fmt.Printf("❌ 错误: %s\n", resp.Message)
				continue
			}

			fmt.Println("\n" + strings.Repeat("=", 70))
			fmt.Printf("🎉 捕获到 API: %s\n", resp.Name)
			fmt.Println(strings.Repeat("=", 70))
			fmt.Printf("Method: %s\n", resp.Method)
			fmt.Printf("URL: %s\n", resp.URL)
			fmt.Printf("Body: %d bytes\n", len(resp.Data))
			fmt.Println(strings.Repeat("-", 70))
			fmt.Println("📦 响应 JSON:")
			fmt.Println(strings.Repeat("-", 70))
			fmt.Println(string(resp.Data))
			fmt.Println(strings.Repeat("=", 70))

			// 保存响应到文件
			filename := fmt.Sprintf("%s_response.json", resp.Name)
			os.WriteFile(filename, resp.Data, 0644)
			fmt.Printf("💾 响应已保存到: %s\n", filename)

		case <-time.After(120 * time.Second):
			fmt.Println("\n⏰ 等待超时（120秒）")
			goto END
		}
	}

END:
	// 保存截图
	if screenshot, err := page.Screenshot(); err == nil {
		os.WriteFile("douyin_api_test.png", screenshot, 0644)
		fmt.Println("📸 已保存截图: douyin_api_test.png")
	}

	fmt.Println("✅ 测试结束!")
}

// listenNetwork 监听网络事件
func listenNetwork(ctx context.Context, chanResponse chan ChanResponse) {
	evenItems := make([]EvenItem, 0)

	chromedp.ListenTarget(ctx, func(event interface{}) {
		switch ev := event.(type) {
		case *network.EventRequestWillBeSent:
			// 排除 OPTIONS 和 HEAD 请求
			if ev.Request.Method == "OPTIONS" || ev.Request.Method == "HEAD" {
				return
			}

			// 检查 URL 是否匹配目标 API
			var name string
			if strings.Contains(ev.Request.URL, "https://ttwid.bytedance.com/ttwid/union/register/") {
				name = "diamond_buy"
				fmt.Printf("\n🎯 捕获请求: [%s] %s\n", ev.Request.Method, ev.Request.URL)
			} else if strings.Contains(ev.Request.URL, "recharge_external_user_info_cache") {
				name = "user_info"
				fmt.Printf("\n🎯 捕获请求: [%s] %s\n", ev.Request.Method, ev.Request.URL)
			}

			if name != "" {
				item := EvenItem{
					RequestID: ev.RequestID,
					Name:      name,
					Method:    ev.Request.Method,
					URL:       ev.Request.URL,
				}
				evenItems = append(evenItems, item)
			}

		case *network.EventLoadingFinished:
			// 查找匹配的请求
			idx := -1
			for i, item := range evenItems {
				if item.RequestID == ev.RequestID {
					idx = i
					break
				}
			}

			if idx < 0 {
				return
			}

			// 获取事件项并从列表中删除
			item := evenItems[idx]
			evenItems = append(evenItems[:idx], evenItems[idx+1:]...)

			// 异步获取响应体
			go handleResponse(ctx, item, chanResponse)
		}
	})
}

// handleResponse 处理响应
func handleResponse(ctx context.Context, item EvenItem, chanResponse chan ChanResponse) {
	body, err := getResponseBody(ctx, item.RequestID)
	if err != nil {
		chanResponse <- ChanResponse{
			Name:    "error",
			Message: fmt.Sprintf("获取响应体失败: %v", err),
		}
		return
	}

	fmt.Printf("   📦 响应体: %d bytes\n", len(body))

	chanResponse <- ChanResponse{
		Name:   item.Name,
		Method: item.Method,
		URL:    item.URL,
		Data:   body,
	}
}

// getResponseBody 获取响应体
func getResponseBody(ctx context.Context, requestID network.RequestID) ([]byte, error) {
	var body []byte
	err := chromedp.Run(ctx, chromedp.ActionFunc(func(ctx context.Context) error {
		result, err := network.GetResponseBody(requestID).Do(ctx)
		if err != nil {
			return err
		}
		body = result
		return nil
	}))
	return body, err
}
