package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/HNRow/puppeteer-real-browser-go/pkg/browser"
)

func main() {
	fmt.Println("🛡️ Smart Cloudflare Bypass Test")
	fmt.Println("================================")

	ctx := context.Background()

	opts := &browser.ConnectOptions{
		Headless:     false,
		UseCustomCDP: false,
		Turnstile:    true,
		Args: []string{
			"--start-maximized",
		},
	}

	fmt.Println("🚀 Starting browser with enhanced anti-detection...")
	instance, err := browser.Connect(ctx, opts)
	if err != nil {
		log.Fatalf("❌ Failed to connect: %v", err)
	}
	defer instance.Close()

	page := instance.Page()

	// Test the same Cloudflare site
	testURL := "https://nopecha.com/demo/cloudflare"
	fmt.Printf("📂 Navigating to: %s\n", testURL)

	if err := page.Navigate(testURL); err != nil {
		log.Fatalf("❌ Navigation failed: %v", err)
	}

	fmt.Println("⏳ Waiting for initial page load...")
	time.Sleep(3 * time.Second)

	// Check initial state
	title, _ := page.GetTitle()
	fmt.Printf("📄 Initial title: %s\n", title)

	// Smart waiting logic - like original Node.js version
	maxWaitTime := 30 * time.Second
	checkInterval := 2 * time.Second
	startTime := time.Now()

	fmt.Println("🔄 Smart waiting for Cloudflare verification...")

	for time.Since(startTime) < maxWaitTime {
		// Check current page state
		currentTitle, err := page.GetTitle()
		if err != nil {
			fmt.Printf("⚠️  Could not get title: %v\n", err)
			continue
		}

		// Check page content
		bodyCheck, err := page.Evaluate(`
			(() => {
				const body = document.body.innerText.toLowerCase();
				return {
					hasVerifyHuman: body.includes('verify you are human') || body.includes('verify that you are human') || body.includes('正在验证您是否是真人'),
					hasCloudflare: body.includes('cloudflare'),
					hasChallenge: body.includes('challenge') || body.includes('captcha') || body.includes('验证'),
					hasWaiting: body.includes('请稍候') || body.includes('please wait') || body.includes('checking'),
					currentURL: window.location.href,
					title: document.title,
					bodyPreview: body.substring(0, 300)
				};
			})()
		`)

		if err != nil {
			fmt.Printf("⚠️  Could not evaluate page: %v\n", err)
			time.Sleep(checkInterval)
			continue
		}

		if result, ok := bodyCheck.(map[string]interface{}); ok {
			fmt.Printf("📊 Status check - Title: %s\n", currentTitle)
			fmt.Printf("   - Verify human: %v\n", result["hasVerifyHuman"])
			fmt.Printf("   - Cloudflare: %v\n", result["hasCloudflare"])
			fmt.Printf("   - Challenge: %v\n", result["hasChallenge"])
			fmt.Printf("   - Waiting: %v\n", result["hasWaiting"])

			// If no longer in challenge state, we might have passed
			if !result["hasVerifyHuman"].(bool) && !result["hasChallenge"].(bool) && !result["hasWaiting"].(bool) {
				fmt.Println("🎉 Cloudflare verification appears to have completed!")
				break
			}

			// If stuck in verification, wait longer
			if result["hasVerifyHuman"].(bool) || result["hasChallenge"].(bool) {
				fmt.Println("⏳ Still in verification state, waiting...")
			}
		}

		time.Sleep(checkInterval)
	}

	// Final check
	fmt.Println("\n🔍 Final verification status:")
	finalTitle, _ := page.GetTitle()
	fmt.Printf("📄 Final title: %s\n", finalTitle)

	// Check if we successfully bypassed
	finalCheck, err := page.Evaluate(`
		(() => {
			const body = document.body.innerText.toLowerCase();
			const url = window.location.href;
			
			return {
				hasVerifyHuman: body.includes('verify you are human') || body.includes('正在验证您是否是真人'),
				hasSuccessContent: body.includes('demo') || body.includes('test') || body.includes('api'),
				url: url,
				urlChanged: !url.includes('ray'),
				title: document.title,
				contentLength: body.length
			};
		})()
	`)

	if err != nil {
		fmt.Printf("⚠️  Could not perform final check: %v\n", err)
	} else if result, ok := finalCheck.(map[string]interface{}); ok {
		fmt.Printf("📊 Final results:\n")
		fmt.Printf("   - Still has verify human: %v\n", result["hasVerifyHuman"])
		fmt.Printf("   - Has success content: %v\n", result["hasSuccessContent"])
		fmt.Printf("   - URL changed from challenge: %v\n", result["urlChanged"])
		fmt.Printf("   - Content length: %v\n", result["contentLength"])

		// Determine success
		if !result["hasVerifyHuman"].(bool) && (result["hasSuccessContent"].(bool) || result["urlChanged"].(bool)) {
			fmt.Println("🎉 SUCCESS: Appears to have bypassed Cloudflare verification!")
		} else {
			fmt.Println("❌ BLOCKED: Still stuck in Cloudflare verification")
		}
	}

	// Test navigator.webdriver one more time
	webdriverTest, err := page.Evaluate("navigator.webdriver")
	if err != nil {
		fmt.Printf("⚠️  Could not test webdriver: %v\n", err)
	} else {
		fmt.Printf("🔍 navigator.webdriver: %v\n", webdriverTest)
	}

	fmt.Println("\n⏳ Keeping browser open for inspection...")
	time.Sleep(20 * time.Second)

	fmt.Println("✅ Test completed!")
}