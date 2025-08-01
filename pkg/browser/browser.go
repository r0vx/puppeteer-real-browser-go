package browser

import (
	"context"
	"fmt"
)

// RealBrowser implements the Browser interface
type RealBrowser struct {
	instance *BrowserInstance
}

// NewRealBrowser creates a new RealBrowser instance
func NewRealBrowser() *RealBrowser {
	return &RealBrowser{}
}

// Connect establishes a connection to Chrome browser
func (rb *RealBrowser) Connect(ctx context.Context, opts *ConnectOptions) (*BrowserInstance, error) {
	if opts == nil {
		opts = &ConnectOptions{
			Headless:  false,
			Turnstile: false,
		}
	}

	// Create context with cancellation
	browserCtx, cancel := context.WithCancel(ctx)

	// Initialize Chrome process
	chrome, err := rb.launchChrome(browserCtx, opts)
	if err != nil {
		cancel()
		return nil, fmt.Errorf("failed to launch Chrome: %w", err)
	}

	// Create browser instance
	instance := &BrowserInstance{
		browser: rb,
		chrome:  chrome,
		ctx:     browserCtx,
		cancel:  cancel,
	}

	// Connect to Chrome via CDP
	page, err := rb.connectToChrome(browserCtx, chrome, opts)
	if err != nil {
		cancel()
		chrome.Kill()
		return nil, fmt.Errorf("failed to connect to Chrome: %w", err)
	}

	instance.page = page
	rb.instance = instance

	return instance, nil
}

// Close closes the browser connection
func (rb *RealBrowser) Close() error {
	if rb.instance != nil {
		return rb.instance.Close()
	}
	return nil
}

// Close closes the browser instance
func (bi *BrowserInstance) Close() error {
	if bi.cancel != nil {
		bi.cancel()
	}

	if bi.page != nil {
		bi.page.Close()
	}

	if bi.chrome != nil {
		return bi.chrome.Kill()
	}

	return nil
}

// Page returns the current page
func (bi *BrowserInstance) Page() Page {
	return bi.page
}

// Browser returns the browser interface
func (bi *BrowserInstance) Browser() Browser {
	return bi.browser
}

// Chrome returns the Chrome process
func (bi *BrowserInstance) Chrome() *ChromeProcess {
	return bi.chrome
}

// launchChrome starts a Chrome process
func (rb *RealBrowser) launchChrome(ctx context.Context, opts *ConnectOptions) (*ChromeProcess, error) {
	launcher := NewChromeLauncher()
	return launcher.Launch(ctx, opts)
}

// connectToChrome establishes CDP connection to Chrome
func (rb *RealBrowser) connectToChrome(ctx context.Context, chrome *ChromeProcess, opts *ConnectOptions) (Page, error) {
	if opts.UseCustomCDP {
		// Use custom CDP client to avoid Runtime.Enable leaks
		connector := CreateCustomCDPConnector()
		return connector.Connect(ctx, chrome, opts)
	} else {
		// Use standard chromedp
		connector := NewCDPConnector()
		return connector.Connect(ctx, chrome, opts)
	}
}

// Connect is a convenience function to create and connect a browser
func Connect(ctx context.Context, opts *ConnectOptions) (*BrowserInstance, error) {
	browser := NewRealBrowser()
	return browser.Connect(ctx, opts)
}
