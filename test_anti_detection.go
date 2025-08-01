package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/HNRow/puppeteer-real-browser-go/pkg/browser"
)

func main() {
	fmt.Println("🔍 Testing Anti-Detection Fixes")
	fmt.Println("================================")

	ctx := context.Background()

	// Test with standard chromedp (should avoid Runtime.Enable now)
	fmt.Println("\n1. Testing Standard CDP with Fixes...")
	testStandardCDP(ctx)

	// Test with custom CDP client
	fmt.Println("\n2. Testing Custom CDP Client...")
	testCustomCDP(ctx)

	fmt.Println("\n✅ Anti-Detection Tests Completed!")
}

func testStandardCDP(ctx context.Context) {
	opts := &browser.ConnectOptions{
		Headless:     false,
		UseCustomCDP: false, // Use standard chromedp with our fixes
		Turnstile:    true,
		Args: []string{
			"--start-maximized",
		},
	}

	instance, err := browser.Connect(ctx, opts)
	if err != nil {
		log.Printf("❌ Failed to connect with standard CDP: %v", err)
		return
	}
	defer instance.Close()

	page := instance.Page()

	// Test anti-detection features
	testAntiDetectionFeatures(page, "Standard CDP")
}

func testCustomCDP(ctx context.Context) {
	opts := &browser.ConnectOptions{
		Headless:     false,
		UseCustomCDP: true, // Use custom CDP client
		Turnstile:    true,
		Args: []string{
			"--start-maximized",
		},
	}

	instance, err := browser.Connect(ctx, opts)
	if err != nil {
		log.Printf("❌ Failed to connect with custom CDP: %v", err)
		return
	}
	defer instance.Close()

	page := instance.Page()

	// Test anti-detection features
	testAntiDetectionFeatures(page, "Custom CDP")
}

func testAntiDetectionFeatures(page browser.Page, clientType string) {
	fmt.Printf("  Testing %s...\n", clientType)

	// Navigate to a test page
	if err := page.Navigate("https://google.com"); err != nil {
		log.Printf("  ❌ Navigation failed: %v", err)
		return
	}

	time.Sleep(3 * time.Second)

	// Test 1: Check webdriver property
	result, err := page.Evaluate("navigator.webdriver")
	if err != nil {
		log.Printf("  ⚠️  Could not evaluate webdriver property: %v", err)
	} else {
		fmt.Printf("  navigator.webdriver: %v\n", result)
		if result == nil {
			fmt.Printf("  ✅ webdriver property hidden successfully\n")
		} else {
			fmt.Printf("  ❌ webdriver property still visible!\n")
		}
	}

	// Test 2: Check plugins
	result, err = page.Evaluate("navigator.plugins.length")
	if err != nil {
		log.Printf("  ⚠️  Could not evaluate plugins: %v", err)
	} else {
		fmt.Printf("  navigator.plugins.length: %v\n", result)
	}

	// Test 3: Check languages
	result, err = page.Evaluate("navigator.languages")
	if err != nil {
		log.Printf("  ⚠️  Could not evaluate languages: %v", err)
	} else {
		fmt.Printf("  navigator.languages: %v\n", result)
	}

	// Test 4: Check MouseEvent screenX/screenY fix
	result, err = page.Evaluate(`
		const event = new MouseEvent('click', { clientX: 100, clientY: 200 });
		({ 
			clientX: event.clientX, 
			clientY: event.clientY, 
			screenX: event.screenX, 
			screenY: event.screenY 
		})
	`)
	if err != nil {
		log.Printf("  ⚠️  Could not test MouseEvent: %v", err)
	} else {
		fmt.Printf("  MouseEvent test: %v\n", result)
	}

	// Test 5: Check window dimensions
	result, err = page.Evaluate(`
		({ 
			innerWidth: window.innerWidth, 
			innerHeight: window.innerHeight,
			outerWidth: window.outerWidth, 
			outerHeight: window.outerHeight
		})
	`)
	if err != nil {
		log.Printf("  ⚠️  Could not test window dimensions: %v", err)
	} else {
		fmt.Printf("  Window dimensions: %v\n", result)
	}

	// Test 6: Test realistic mouse movement
	fmt.Printf("  Testing realistic mouse click...\n")
	if err := page.RealClick(100, 100); err != nil {
		log.Printf("  ⚠️  RealClick failed: %v", err)
	} else {
		fmt.Printf("  ✅ RealClick executed successfully\n")
	}

	// Test 7: Check if Runtime.enable was called (this would be in DevTools console)
	fmt.Printf("  💡 Check DevTools Console for Runtime.enable calls\n")
	fmt.Printf("  💡 If no Runtime.enable appears, our fix is working!\n")

	fmt.Printf("  %s tests completed.\n", clientType)
}