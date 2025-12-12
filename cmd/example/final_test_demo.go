package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/HNRow/puppeteer-real-browser-go/pkg/browser"
)

func main() {
	fmt.Println("🛡️ Final Cloudflare Bypass Test")
	fmt.Println("=================================")
	fmt.Println("✨ Using all the improvements we've made")

	ctx := context.Background()

	// Use our most optimized configuration
	opts := &browser.ConnectOptions{
		Headless:     false, // Keep visible to see what's happening
		UseCustomCDP: false, // Use standard chromedp with our Runtime.Enable bypass
		Turnstile:    true,  // Enable Turnstile solving
		Args: []string{
			"--start-maximized",
			"--disable-blink-features=AutomationControlled",
			"--exclude-switches=enable-automation",
			//"--disable-web-security", // Additional bypass flag
		},
	}

	fmt.Println("🚀 Starting browser with all anti-detection improvements...")
	instance, err := browser.Connect(ctx, opts)
	if err != nil {
		log.Fatalf("❌ Failed to connect: %v", err)
	}
	defer instance.Close()

	page := instance.Page()

	// Test your specific problematic URL
	fmt.Println("📝 Please enter the URL that's giving you trouble:")
	fmt.Println("   (Or I'll use a default Cloudflare test site)")

	// For now, let's use a known Cloudflare-protected site
	testURL := "https://nopecha.com/demo/cloudflare"

	fmt.Printf("🎯 Testing URL: %s\n", testURL)
	fmt.Println("📂 Navigating...")

	if err := page.Navigate(testURL); err != nil {
		log.Fatalf("❌ Navigation failed: %v", err)
	}

	// Advanced waiting and verification logic
	fmt.Println("⏳ Advanced verification process starting...")

	success := waitForCloudflareBypass(page, 60*time.Second)

	if success {
		fmt.Println("🎉 SUCCESS: Cloudflare bypass completed!")

		// Additional verification
		finalTitle, _ := page.GetTitle()
		fmt.Printf("📄 Final page title: %s\n", finalTitle)

		// Take a screenshot for verification
		fmt.Println("📸 Taking screenshot for verification...")
		screenshot, err := page.Screenshot()
		if err == nil {
			fmt.Printf("✅ Screenshot captured: %d bytes\n", len(screenshot))
		}

	} else {
		fmt.Println("❌ TIMEOUT: Could not bypass Cloudflare within timeout period")

		// Debug information
		currentTitle, _ := page.GetTitle()
		fmt.Printf("📄 Current title: %s\n", currentTitle)

		debugInfo, _ := page.Evaluate(`
			(() => {
				return {
					url: window.location.href,
					title: document.title,
					bodyText: document.body.innerText.substring(0, 500),
					hasChallenge: document.body.innerText.toLowerCase().includes('challenge') || 
								 document.body.innerText.toLowerCase().includes('verify') ||
								 document.body.innerText.toLowerCase().includes('captcha')
				};
			})()
		`)
		fmt.Printf("🔍 Debug info: %v\n", debugInfo)
	}

	fmt.Println("\n💡 Tips:")
	fmt.Println("   - If you see a challenge page but it doesn't auto-complete,")
	fmt.Println("     the site might have additional protections")
	fmt.Println("   - Try different sites to test effectiveness")
	fmt.Println("   - Some sites have rate limiting or IP-based blocks")

	fmt.Println("\n⏳ Keeping browser open for 30 seconds for manual inspection...")
	time.Sleep(30000 * time.Second)

	fmt.Println("✅ Test completed!")
}

// waitForCloudflareBypass implements intelligent waiting for Cloudflare bypass
func waitForCloudflareBypass(page browser.Page, timeout time.Duration) bool {
	startTime := time.Now()
	checkInterval := 2 * time.Second

	fmt.Println("🔄 Monitoring Cloudflare bypass progress...")

	for time.Since(startTime) < timeout {
		// Check current state
		result, err := page.Evaluate(`
			(() => {
				const body = document.body.innerText.toLowerCase();
				const url = window.location.href;
				
				// Multiple language support for Cloudflare messages
				const challengeKeywords = [
					'verify you are human',
					'verify that you are human', 
					'checking your browser',
					'正在验证您是否是真人',
					'正在检查您的浏览器',
					'verifique que usted es humano',
					'vérifiez que vous êtes humain',
					'challenge',
					'captcha',
					'ray id'
				];
				
				const hasChallenge = challengeKeywords.some(keyword => body.includes(keyword));
				const isCloudflareUrl = url.includes('ray') || url.includes('challenge');
				
				// 使用与smart_cloudflare_demo.go相同的成功检测逻辑
				const hasWaiting = body.includes('请稍候') || body.includes('please wait') || body.includes('checking');
				
				// 修正：成功的判断应该只检查没有挑战和没有等待状态
				// URL可能仍然包含challenge参数，这是正常的
				const success = !hasChallenge && !hasWaiting;
				
				return {
					hasChallenge: hasChallenge,
					isChallengePage: isCloudflareUrl,
					hasWaiting: hasWaiting,
					title: document.title,
					url: url,
					contentLength: body.length,
					success: success
				};
			})()
		`)

		if err != nil {
			fmt.Printf("⚠️  Evaluation error: %v\n", err)
			time.Sleep(checkInterval)
			continue
		}

		if state, ok := result.(map[string]interface{}); ok {
			fmt.Printf("📊 Status: Challenge=%v, URL=%v, Success=%v, Title=%v\n",
				state["isChallengePage"],
				state["url"],
				state["success"],
				state["title"])

			// Check if we've successfully bypassed
			if state["success"].(bool) {
				fmt.Println("✅ Cloudflare bypass detected!")
				return true
			}

			// If still in challenge state, continue waiting
			if state["hasChallenge"].(bool) || state["isChallengePage"].(bool) {
				fmt.Println("⏳ Still processing challenge, waiting...")
			}
		}

		time.Sleep(checkInterval)
	}

	return false
}
