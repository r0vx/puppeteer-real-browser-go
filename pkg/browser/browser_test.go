package browser

import (
	"context"
	"testing"
	"time"
)

func TestConnect(t *testing.T) {
	ctx := context.Background()

	opts := &ConnectOptions{
		Headless: true, // Use headless for testing
		Args: []string{
			"--no-sandbox",
			"--disable-dev-shm-usage",
		},
	}

	instance, err := Connect(ctx, opts)
	if err != nil {
		t.Fatalf("Failed to connect to browser: %v", err)
	}
	defer instance.Close()

	if instance == nil {
		t.Fatal("Browser instance is nil")
	}

	if instance.Page() == nil {
		t.Fatal("Page is nil")
	}
}

func TestBrowserNavigation(t *testing.T) {
	ctx := context.Background()

	opts := &ConnectOptions{
		Headless: true,
		Args: []string{
			"--no-sandbox",
			"--disable-dev-shm-usage",
		},
	}

	instance, err := Connect(ctx, opts)
	if err != nil {
		t.Fatalf("Failed to connect to browser: %v", err)
	}
	defer instance.Close()

	page := instance.Page()

	// Test navigation
	err = page.Navigate("https://www.google.com")
	if err != nil {
		t.Fatalf("Failed to navigate: %v", err)
	}

	// Wait for page to load
	time.Sleep(2 * time.Second)

	// Test getting title
	title, err := page.GetTitle()
	if err != nil {
		t.Fatalf("Failed to get title: %v", err)
	}

	if title == "" {
		t.Fatal("Title is empty")
	}

	t.Logf("Page title: %s", title)
}

func TestBrowserScreenshot(t *testing.T) {
	ctx := context.Background()

	opts := &ConnectOptions{
		Headless: true,
		Args: []string{
			"--no-sandbox",
			"--disable-dev-shm-usage",
		},
	}

	instance, err := Connect(ctx, opts)
	if err != nil {
		t.Fatalf("Failed to connect to browser: %v", err)
	}
	defer instance.Close()

	page := instance.Page()

	// Navigate to a simple page
	err = page.Navigate("data:text/html,<html><body><h1>Test Page</h1></body></html>")
	if err != nil {
		t.Fatalf("Failed to navigate: %v", err)
	}

	// Take screenshot
	screenshot, err := page.Screenshot()
	if err != nil {
		t.Fatalf("Failed to take screenshot: %v", err)
	}

	if len(screenshot) == 0 {
		t.Fatal("Screenshot is empty")
	}

	t.Logf("Screenshot size: %d bytes", len(screenshot))
}

func TestStealthFeatures(t *testing.T) {
	ctx := context.Background()

	opts := &ConnectOptions{
		Headless: true,
		Args: []string{
			"--no-sandbox",
			"--disable-dev-shm-usage",
		},
	}

	instance, err := Connect(ctx, opts)
	if err != nil {
		t.Fatalf("Failed to connect to browser: %v", err)
	}
	defer instance.Close()

	page := instance.Page()

	// Navigate to a test page
	err = page.Navigate("data:text/html,<html><body><h1>Stealth Test</h1></body></html>")
	if err != nil {
		t.Fatalf("Failed to navigate: %v", err)
	}

	// Test webdriver property
	result, err := page.Evaluate("navigator.webdriver")
	if err != nil {
		t.Fatalf("Failed to evaluate webdriver property: %v", err)
	}

	// webdriver should be undefined (hidden)
	if result != nil {
		t.Logf("Warning: webdriver property is not hidden: %v", result)
	}

	// Test MouseEvent coordinates
	mouseResult, err := page.Evaluate(`(() => {
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
	})()`)
	if err != nil {
		t.Fatalf("Failed to evaluate mouse event: %v", err)
	}

	t.Logf("Mouse event coordinates: %+v", mouseResult)
}

func TestChromeProcessManagement(t *testing.T) {
	ctx := context.Background()

	opts := &ConnectOptions{
		Headless: true,
		Args: []string{
			"--no-sandbox",
			"--disable-dev-shm-usage",
		},
	}

	instance, err := Connect(ctx, opts)
	if err != nil {
		t.Fatalf("Failed to connect to browser: %v", err)
	}

	chrome := instance.Chrome()
	if chrome == nil {
		t.Fatal("Chrome process is nil")
	}

	if chrome.PID == 0 {
		t.Fatal("Chrome PID is 0")
	}

	if chrome.Port == 0 {
		t.Fatal("Chrome port is 0")
	}

	// Check if process is running
	if !chrome.IsRunning() {
		t.Fatal("Chrome process is not running")
	}

	t.Logf("Chrome PID: %d, Port: %d", chrome.PID, chrome.Port)

	// Close the browser
	err = instance.Close()
	if err != nil {
		t.Fatalf("Failed to close browser: %v", err)
	}

	// Wait a bit for process to terminate
	time.Sleep(1 * time.Second)

	// Check if process is stopped
	if chrome.IsRunning() {
		t.Fatal("Chrome process is still running after close")
	}
}

func TestConnectOptions(t *testing.T) {
	ctx := context.Background()

	// Test with various options
	testCases := []struct {
		name string
		opts *ConnectOptions
	}{
		{
			name: "Default options",
			opts: &ConnectOptions{
				Headless: true,
			},
		},
		{
			name: "With custom args",
			opts: &ConnectOptions{
				Headless: true,
				Args: []string{
					"--no-sandbox",
					"--disable-dev-shm-usage",
					"--window-size=1280,720",
				},
			},
		},
		{
			name: "With ignore all flags",
			opts: &ConnectOptions{
				Headless:       true,
				IgnoreAllFlags: true,
				Args: []string{
					"--no-sandbox",
					"--disable-dev-shm-usage",
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			instance, err := Connect(ctx, tc.opts)
			if err != nil {
				t.Fatalf("Failed to connect with %s: %v", tc.name, err)
			}
			defer instance.Close()

			// Basic functionality test
			err = instance.Page().Navigate("data:text/html,<html><body><h1>Test</h1></body></html>")
			if err != nil {
				t.Fatalf("Failed to navigate with %s: %v", tc.name, err)
			}
		})
	}
}
