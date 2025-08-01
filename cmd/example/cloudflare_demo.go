package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/HNRow/puppeteer-real-browser-go/pkg/browser"
)

func main() {
	fmt.Println("🛡️ Testing Cloudflare Detection Fix")
	fmt.Println("==================================")

	ctx := context.Background()

	// Use our fixed implementation with Runtime.Enable bypass
	opts := &browser.ConnectOptions{
		Headless:     false, // Keep visible to see Cloudflare page
		UseCustomCDP: false, // Use standard chromedp with Runtime bypass
		Turnstile:    true,  // Enable Turnstile solving
		Args: []string{
			"--start-maximized",
		},
	}

	fmt.Println("🚀 Starting browser with Runtime.Enable bypass...")
	instance, err := browser.Connect(ctx, opts)
	if err != nil {
		log.Fatalf("❌ Failed to connect: %v", err)
	}
	defer instance.Close()

	page := instance.Page()

	// Test with a simple site first
	fmt.Println("📂 Testing basic navigation...")
	if err := page.Navigate("https://google.com"); err != nil {
		log.Fatalf("❌ Navigation failed: %v", err)
	}

	time.Sleep(2 * time.Second)

	// Get title to verify basic functionality
	title, err := page.GetTitle()
	if err != nil {
		log.Printf("⚠️  Could not get title: %v", err)
	} else {
		fmt.Printf("✅ Basic test passed - Title: %s\n", title)
	}

	// Now test JavaScript evaluation (this was causing Runtime.Enable)
	fmt.Println("🔍 Testing JavaScript evaluation (critical test)...")
	result, err := page.Evaluate("navigator.webdriver")
	if err != nil {
		log.Printf("⚠️  JavaScript evaluation failed: %v", err)
	} else {
		fmt.Printf("✅ navigator.webdriver result: %v\n", result)
	}

	// Test mouse movement
	fmt.Println("🖱️  Testing realistic mouse movement...")
	if err := page.RealClick(100, 100); err != nil {
		log.Printf("⚠️  RealClick failed: %v", err)
	} else {
		fmt.Printf("✅ RealClick executed\n")
	}

	fmt.Println("\n🎯 Now testing Cloudflare protected site...")
	fmt.Println("⏳ This is the critical test - if it passes without 'Verify you are human', we succeeded!")

	// Navigate to a Cloudflare-protected site
	// Replace with actual Cloudflare-protected URL
	testURL := "https://nopecha.com/demo/cloudflare"
	fmt.Printf("📂 Navigating to: %s\n", testURL)

	if err := page.Navigate(testURL); err != nil {
		log.Printf("❌ Cloudflare test navigation failed: %v", err)
	} else {
		fmt.Println("✅ Navigation to Cloudflare site succeeded")
	}

	fmt.Println("\n💡 Key Success Indicators:")
	fmt.Println("  - Page loads without 'Verify you are human' message")
	fmt.Println("  - No Runtime.enable commands in DevTools console")
	fmt.Println("  - navigator.webdriver returns undefined (not an object)")

	fmt.Println("\n⏳ Keeping browser open for 30 seconds to observe results...")
	time.Sleep(30 * time.Second)

	fmt.Println("🏁 Test completed!")
}
