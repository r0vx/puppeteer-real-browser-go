package browser

import (
	"context"
	"fmt"

	"github.com/chromedp/chromedp"
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

// CreateBrowserContext creates a new browser context (like puppeteer browser.createBrowserContext())
// This creates an independent browser context that can have multiple pages
func (bi *BrowserInstance) CreateBrowserContext(opts *BrowserContextOptions) (*BrowserContext, error) {
	if bi.chrome == nil {
		return nil, fmt.Errorf("chrome process not available")
	}

	// Create allocator context for connecting to existing Chrome instance
	// This connects to the same Chrome process but creates a new context
	allocCtx, allocCancel := chromedp.NewRemoteAllocator(context.Background(), fmt.Sprintf("http://localhost:%d", bi.chrome.Port))

	// Create the browser context
	browserCtx := &BrowserContext{
		allocCtx:    allocCtx,
		allocCancel: allocCancel,
		chrome:      bi.chrome,
		opts:        nil, // Don't assume page type - will be set when needed
	}

	// Try to get options from the page if it's a CDPPage
	if cdpPage, ok := bi.page.(*CDPPage); ok {
		browserCtx.opts = cdpPage.opts
	}

	return browserCtx, nil
}

// BrowserContextOptions represents options for creating browser context
type BrowserContextOptions struct {
	IgnoreHTTPSErrors bool
	ProxyServer       string
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
