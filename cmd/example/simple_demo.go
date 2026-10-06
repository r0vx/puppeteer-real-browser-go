package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/r0vx/puppeteer-real-browser-go/pkg/browser"
)

func main() {
	fmt.Println("🔍 Simple Anti-Detection Test")
	fmt.Println("=============================")

	ctx := context.Background()

	// Test with our fixed implementation
	opts := &browser.ConnectOptions{
		Headless:  false,
		Turnstile: false,
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

	fmt.Println("📂 Navigating to Google...")
	if err := page.Navigate("https://pay.ssl.kuaishou.com/pay?source=NewReco"); err != nil {
		log.Fatalf("❌ Navigation failed: %v", err)
	}

	// Wait for page to load
	time.Sleep(3 * time.Second)

	fmt.Println("📋 Getting page title...")
	title, err := page.GetTitle()
	if err != nil {
		log.Printf("⚠️  Could not get title: %v", err)
	} else {
		fmt.Printf("✅ Page title: %s\n", title)
	}

	fmt.Println("🖱️  Testing realistic mouse click...")
	if err := page.RealClick(200, 300); err != nil {
		log.Printf("⚠️  RealClick failed: %v", err)
	} else {
		fmt.Printf("✅ RealClick executed successfully\n")
	}

	fmt.Println("📸 Taking screenshot...")
	screenshot, err := page.Screenshot()
	if err != nil {
		log.Printf("⚠️  Screenshot failed: %v", err)
	} else {
		fmt.Printf("✅ Screenshot taken: %d bytes\n", len(screenshot))
	}

	fmt.Println("\n💡 Key Points:")
	fmt.Println("  - Browser should be running without automation flags")
	fmt.Println("  - Check DevTools Console: NO Runtime.enable should appear")
	fmt.Println("  - Mouse movements should be smooth and human-like")
	fmt.Println("  - navigator.webdriver should be hidden by stealth scripts")

	fmt.Println("\n🎯 Manual Check:")
	fmt.Println("  1. Open DevTools (F12)")
	fmt.Println("  2. Go to Console tab")
	fmt.Println("  3. Type: navigator.webdriver")
	fmt.Println("  4. Should return: undefined (not true)")

	fmt.Println("\n⏳ Keeping browser open for 15 seconds for manual inspection...")
	time.Sleep(15 * time.Second)

	fmt.Println("✅ Test completed!")
}
