package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/r0vx/puppeteer-real-browser-go/pkg/browser"
	"github.com/r0vx/puppeteer-real-browser-go/pkg/page"
	"github.com/r0vx/puppeteer-real-browser-go/pkg/turnstile"
)

func main() {
	// Create context
	ctx := context.Background()

	// Configure browser options
	opts := &browser.ConnectOptions{
		Headless:  false, // Set to true for headless mode
		Turnstile: true,  // Enable automatic Turnstile solving
		Args: []string{
			"--start-maximized",
			"--disable-web-security", // For testing purposes
		},
		ConnectOption: map[string]interface{}{
			"defaultViewport": nil, // Use full browser window
		},
	}

	// Connect to browser
	fmt.Println("Starting browser...")
	instance, err := browser.Connect(ctx, opts)
	if err != nil {
		log.Fatalf("Failed to connect to browser: %v", err)
	}
	defer instance.Close()

	// Get the page
	browserPage := instance.Page()

	// Create page controller for enhanced functionality
	controller := page.NewController(browserPage, ctx, opts.Turnstile)
	if err := controller.Initialize(); err != nil {
		log.Fatalf("Failed to initialize page controller: %v", err)
	}
	defer controller.Stop()

	// Example 1: Basic navigation
	fmt.Println("Navigating to Google...")
	if err := browserPage.Navigate("https://www.google.com"); err != nil {
		log.Fatalf("Failed to navigate: %v", err)
	}

	// Wait a bit for the page to load
	time.Sleep(2 * time.Second)

	// Get page title
	title, err := browserPage.GetTitle()
	if err != nil {
		log.Printf("Failed to get title: %v", err)
	} else {
		fmt.Printf("Page title: %s\n", title)
	}

	// Example 2: Test Turnstile solving (if available)
	fmt.Println("Testing Turnstile functionality...")
	testTurnstile(ctx, browserPage)

	// Example 3: Test realistic clicking
	fmt.Println("Testing realistic mouse movement...")
	testRealisticClick(controller)

	// Example 4: Test stealth features
	fmt.Println("Testing stealth features...")
	testStealthFeatures(browserPage)

	// Keep browser open for a while to observe
	fmt.Println("Browser will stay open for 30 seconds for observation...")
	time.Sleep(30 * time.Second)

	fmt.Println("Example completed successfully!")
}

// testTurnstile demonstrates Turnstile captcha solving
func testTurnstile(ctx context.Context, page browser.Page) {
	// Navigate to a page that might have Turnstile (for testing)
	// Note: This is just an example - replace with actual test URL
	fmt.Println("  - Navigating to Turnstile test page...")

	// Create Turnstile solver
	solver := turnstile.NewSolver(page, ctx)

	// Start the solver
	if err := solver.Start(); err != nil {
		log.Printf("Failed to start Turnstile solver: %v", err)
		return
	}
	defer solver.Stop()

	// Wait for potential solution
	fmt.Println("  - Waiting for Turnstile solution...")
	if err := solver.WaitForSolution(10 * time.Second); err != nil {
		fmt.Printf("  - No Turnstile found or timeout: %v\n", err)
	} else {
		fmt.Println("  - Turnstile solved successfully!")
	}
}

// testRealisticClick demonstrates realistic mouse movement
func testRealisticClick(controller *page.Controller) {
	fmt.Println("  - Performing realistic click...")

	// Click at coordinates (100, 100) with realistic mouse movement
	if err := controller.RealClick(100, 100); err != nil {
		log.Printf("Failed to perform realistic click: %v", err)
		return
	}

	fmt.Println("  - Realistic click completed")
}

// testStealthFeatures tests anti-detection features
func testStealthFeatures(page browser.Page) {
	fmt.Println("  - Testing stealth features...")

	// Test webdriver property hiding
	script := `({
		webdriver: navigator.webdriver,
		plugins: navigator.plugins.length,
		languages: navigator.languages,
		chrome: typeof window.chrome !== 'undefined'
	})`

	result, err := page.Evaluate(script)
	if err != nil {
		log.Printf("Failed to evaluate stealth script: %v", err)
		return
	}

	fmt.Printf("  - Stealth test results: %+v\n", result)

	// Test MouseEvent coordinates
	mouseScript := `(() => {
		const event = new MouseEvent('click', {
			clientX: 100,
			clientY: 200
		});
		return {
			clientX: event.clientX,
			clientY: event.clientY,
			screenX: event.screenX,
			screenY: event.screenY
		};
	})()`

	mouseResult, err := page.Evaluate(mouseScript)
	if err != nil {
		log.Printf("Failed to evaluate mouse script: %v", err)
		return
	}

	fmt.Printf("  - Mouse event test: %+v\n", mouseResult)
}

// Additional example functions for specific use cases

// ExampleWithProxy demonstrates using a proxy
func ExampleWithProxy() {
	ctx := context.Background()

	opts := &browser.ConnectOptions{
		Headless: false,
		Proxy: &browser.ProxyConfig{
			Host:     "proxy.example.com",
			Port:     "8080",
			Username: "user",
			Password: "pass",
		},
	}

	instance, err := browser.Connect(ctx, opts)
	if err != nil {
		log.Fatalf("Failed to connect with proxy: %v", err)
	}
	defer instance.Close()

	// Use the browser with proxy...
}

// ExampleHeadless demonstrates headless mode
func ExampleHeadless() {
	ctx := context.Background()

	opts := &browser.ConnectOptions{
		Headless: true, // or "new" for new headless mode
	}

	instance, err := browser.Connect(ctx, opts)
	if err != nil {
		log.Fatalf("Failed to connect in headless mode: %v", err)
	}
	defer instance.Close()

	// Take a screenshot
	screenshot, err := instance.Page().Screenshot()
	if err != nil {
		log.Printf("Failed to take screenshot: %v", err)
		return
	}

	fmt.Printf("Screenshot taken: %d bytes\n", len(screenshot))
}

// ExampleCustomChrome demonstrates using custom Chrome path
func ExampleCustomChrome() {
	ctx := context.Background()

	opts := &browser.ConnectOptions{
		CustomConfig: map[string]interface{}{
			"chromePath":  "/path/to/custom/chrome",
			"userDataDir": "/path/to/custom/userdata",
		},
	}

	instance, err := browser.Connect(ctx, opts)
	if err != nil {
		log.Fatalf("Failed to connect with custom Chrome: %v", err)
	}
	defer instance.Close()

	// Use custom Chrome instance...
}
