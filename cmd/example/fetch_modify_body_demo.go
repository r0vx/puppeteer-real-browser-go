//go:build ignore
// +build ignore

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/r0vx/puppeteer-real-browser-go/pkg/browser"
)

func main() {
	fmt.Println("🎯 Fetch POST 请求拦截 + 修改 + 获取响应")
	fmt.Println("=====================================")
	fmt.Println("目标: 拦截 cashier 请求")
	fmt.Println("修改: userId 666666 → 7777777")
	fmt.Println("获取: 响应内容")
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

	customPage, ok := page.(*browser.CustomCDPPage)
	if !ok {
		log.Fatal("❌ 需要 UseCustomCDP: true")
	}

	// 启用网络监听
	if err := customPage.EnableNetwork(); err != nil {
		log.Fatalf("❌ 启用网络监听失败: %v", err)
	}

	// 启用 Fetch 拦截 - 拦截 cashier 请求
	patterns := []string{
		"*cashier*",
		"*deposit*",
	}
	if err := customPage.EnableFetch(patterns); err != nil {
		log.Fatalf("❌ 启用 Fetch 拦截失败: %v", err)
	}
	fmt.Println("✅ Fetch 拦截已启用!")
	fmt.Printf("   拦截模式: %v\n", patterns)

	// 设置拦截处理器
	customPage.OnRequestPaused(func(requestID, url, method, postData string, headers map[string]string) {
		fmt.Println("\n" + strings.Repeat("=", 70))
		fmt.Println("🚫 请求被拦截!")
		fmt.Println(strings.Repeat("=", 70))
		fmt.Printf("RequestID: %s\n", requestID)
		fmt.Printf("Method: %s\n", method)
		fmt.Printf("URL: %s\n", url)
		if postData != "" {
			fmt.Printf("原始 PostData: %s\n", postData)
		}

		// 检查是否是 POST 请求且包含 userId
		if method == "POST" && strings.Contains(postData, "userId") {
			// 解析 JSON（UseNumber 保留原始数字精度，避免大整数被 float64 破坏）
			dec := json.NewDecoder(strings.NewReader(postData))
			dec.UseNumber()
			var body map[string]interface{}
			if err := dec.Decode(&body); err != nil {
				fmt.Printf("⚠️ 解析 JSON 失败: %v\n", err)
				customPage.ContinueRequest(requestID, "")
				return
			}

			// 修改 userId（保留原始类型：字符串仍为字符串，数字仍为数字）
			oldUserID := body["userId"]
			if _, isStr := oldUserID.(string); isStr {
				body["userId"] = "7777777"
			} else {
				body["userId"] = json.Number("7777777")
			}
			fmt.Printf("✏️  修改 userId: %v → %v\n", oldUserID, body["userId"])

			// 序列化回 JSON
			newPostData, err := json.Marshal(body)
			if err != nil {
				fmt.Printf("⚠️ 序列化 JSON 失败: %v\n", err)
				customPage.ContinueRequest(requestID, "")
				return
			}

			fmt.Printf("📝 新 PostData: %s\n", string(newPostData))
			fmt.Println("📤 继续请求（使用修改后的请求体）...")

			// 使用修改后的请求体继续请求
			if err := customPage.ContinueRequestWithBody(requestID, "", string(newPostData), nil); err != nil {
				fmt.Printf("⚠️ 继续请求失败: %v\n", err)
				customPage.ContinueRequest(requestID, "")
			}
		} else {
			fmt.Println("📤 继续请求（未修改）...")
			customPage.ContinueRequest(requestID, "")
		}
		fmt.Println(strings.Repeat("=", 70))
	})

	// 监听网络请求（记录 Network 层的 requestID 映射）
	var networkRequestsMu sync.Mutex
	networkRequests := make(map[string]string) // Network requestID -> URL

	customPage.OnNetworkRequest(func(requestID, url, method string) {
		if strings.Contains(url, "cashier") || strings.Contains(url, "deposit") {
			networkRequestsMu.Lock()
			networkRequests[requestID] = url
			networkRequestsMu.Unlock()
		}
	})

	// 监听响应完成 - 获取响应内容
	customPage.OnNetworkLoadingFinished(func(requestID string) {
		networkRequestsMu.Lock()
		url, exists := networkRequests[requestID]
		if exists {
			delete(networkRequests, requestID)
		}
		networkRequestsMu.Unlock()

		if exists {
			// 获取响应内容
			body, err := customPage.GetResponseBody(requestID)
			if err != nil {
				fmt.Printf("\n⚠️ 获取响应失败 [%s]: %v\n", requestID, err)
				return
			}

			fmt.Println("\n" + strings.Repeat("=", 70))
			fmt.Println("📥 响应内容")
			fmt.Println(strings.Repeat("=", 70))
			fmt.Printf("URL: %s\n", url)
			fmt.Printf("RequestID: %s\n", requestID)
			fmt.Println(strings.Repeat("-", 70))

			// 尝试格式化 JSON
			var prettyJSON map[string]interface{}
			if json.Unmarshal(body, &prettyJSON) == nil {
				formatted, _ := json.MarshalIndent(prettyJSON, "", "  ")
				fmt.Println(string(formatted))
			} else {
				fmt.Println(string(body))
			}
			fmt.Println(strings.Repeat("=", 70))
		}
	})

	// 导航到目标页面
	fmt.Println("\n📂 导航到快手充值页面...")
	if err := page.Navigate("https://pay.ssl.kuaishou.com/pay?source=NewReco"); err != nil {
		log.Fatalf("❌ 导航失败: %v", err)
	}

	fmt.Println("\n⏳ 等待用户操作... (120秒)")
	fmt.Println("💡 提示: 点击充值按钮触发 cashier 请求")
	fmt.Println("📱 观察 userId 是否被修改为 7777777")
	fmt.Println("📥 查看响应内容")
	time.Sleep(120 * time.Second)

	if err := customPage.DisableFetch(); err != nil {
		fmt.Printf("⚠️ 禁用 Fetch 失败: %v\n", err)
	}

	fmt.Println("\n✅ 测试完成!")
}
