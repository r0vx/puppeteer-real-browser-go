package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/HNRow/puppeteer-real-browser-go/pkg/browser"
)

func main() {
	fmt.Println("🔍 Detailed Cloudflare Debug Test")
	fmt.Println("==================================")

	ctx := context.Background()

	opts := &browser.ConnectOptions{
		Headless:     false,
		UseCustomCDP: false,
		Turnstile:    true,
		Args: []string{
			"--start-maximized",
		},
	}

	fmt.Println("🚀 Starting browser...")
	instance, err := browser.Connect(ctx, opts)
	if err != nil {
		log.Fatalf("❌ Failed to connect: %v", err)
	}
	defer instance.Close()

	page := instance.Page()

	testURL := "https://irys.xyz/faucet"
	fmt.Printf("🎯 Testing: %s\n", testURL)

	if err := page.Navigate(testURL); err != nil {
		log.Fatalf("❌ Navigation failed: %v", err)
	}

	fmt.Println("⏳ Waiting 3 seconds for initial load...")
	time.Sleep(3 * time.Second)

	// 详细监控60秒
	for i := 0; i < 30; i++ {
		fmt.Printf("\n🔄 Check #%d (after %d seconds):\n", i+1, (i+1)*2)

		title, _ := page.GetTitle()
		url, _ := page.GetURL()

		fmt.Printf("   📄 Title: %s\n", title)
		fmt.Printf("   🌐 URL: %s\n", url)

		// 详细的页面状态检查
		pageInfo, err := page.Evaluate(`
			(() => {
				const body = document.body.innerText;
				const bodyLower = body.toLowerCase();
				
				// 检查各种状态
				const checks = {
					hasVerifyHuman: bodyLower.includes('verify you are human') || bodyLower.includes('verify that you are human'),
					hasWaiting: bodyLower.includes('请稍候') || bodyLower.includes('please wait') || bodyLower.includes('checking'),
					hasChallenge: bodyLower.includes('challenge') || bodyLower.includes('captcha'),
					hasCloudflare: bodyLower.includes('cloudflare'),
					hasJustMoment: bodyLower.includes('just a moment'),
					hasSuccessKeywords: bodyLower.includes('demo') || bodyLower.includes('success') || bodyLower.includes('welcome'),
					
					// URL状态
					urlHasRay: window.location.href.includes('ray'),
					urlHasChallenge: window.location.href.includes('challenge'),
					urlHasToken: window.location.href.includes('__cf_chl_rt_tk'),
					
					// 页面元素检查
					hasIframe: document.querySelectorAll('iframe').length > 0,
					hasSpinner: document.querySelectorAll('[class*="spinner"], [class*="loading"]').length > 0,
					
					// 完整信息
					fullTitle: document.title,
					fullURL: window.location.href,
					bodyPreview: body.substring(0, 200),
					bodyLength: body.length
				};
				
				return checks;
			})()
		`)

		if err != nil {
			fmt.Printf("   ⚠️  Evaluation error: %v\n", err)
		} else if info, ok := pageInfo.(map[string]interface{}); ok {
			fmt.Printf("   🔍 Page Analysis:\n")
			fmt.Printf("      - Verify Human: %v\n", info["hasVerifyHuman"])
			fmt.Printf("      - Waiting: %v\n", info["hasWaiting"])
			fmt.Printf("      - Challenge: %v\n", info["hasChallenge"])
			fmt.Printf("      - Just a Moment: %v\n", info["hasJustMoment"])
			fmt.Printf("      - Success Keywords: %v\n", info["hasSuccessKeywords"])
			fmt.Printf("      - URL has Token: %v\n", info["urlHasToken"])
			fmt.Printf("      - Has iFrame: %v\n", info["hasIframe"])
			fmt.Printf("      - Body Length: %v\n", info["bodyLength"])
			fmt.Printf("      - Body Preview: %v\n", info["bodyPreview"])

			// 判断状态
			isStillInChallenge := info["hasVerifyHuman"].(bool) || info["hasWaiting"].(bool) || info["hasChallenge"].(bool)
			hasToken := info["urlHasToken"].(bool)

			if !isStillInChallenge && hasToken {
				fmt.Printf("   ✅ STATUS: Appears to have bypassed Cloudflare!\n")
			} else if hasToken {
				fmt.Printf("   ⏳ STATUS: Token received, but still in challenge state\n")
			} else {
				fmt.Printf("   ❌ STATUS: Still being challenged\n")
			}
		}

		time.Sleep(2 * time.Second)
	}

	fmt.Println("\n📸 Taking final screenshot...")
	screenshot, err := page.Screenshot()
	if err == nil {
		fmt.Printf("✅ Screenshot saved: %d bytes\n", len(screenshot))
	}

	fmt.Println("\n💭 请告诉我你在浏览器中看到了什么:")
	fmt.Println("   1. 是否还显示'验证你是真人'的界面?")
	fmt.Println("   2. 是否卡在'Just a moment...'页面?")
	fmt.Println("   3. 是否已经进入到demo页面?")
	fmt.Println("   4. 其他什么情况?")

	fmt.Println("\n⏳ 保持浏览器打开60秒供检查...")
	time.Sleep(60 * time.Second)

	fmt.Println("✅ Debug test completed!")
}
