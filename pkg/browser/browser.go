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

	// 与原版一致：开启 Turnstile 时在页面生命周期内后台自动点击
	if opts.Turnstile {
		instance.turnstile = NewTurnstileAutoSolver(page, browserCtx)
		instance.turnstile.Start()
	}

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
	// 先停 Turnstile 循环，避免它在页面关闭过程中继续发命令
	if bi.turnstile != nil {
		bi.turnstile.Stop()
	}
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

	// 沿用实例的选项；CustomCDP 实例记下主页面，NewPage 经它在同一浏览器连接上开新标签页
	switch page := bi.page.(type) {
	case *CDPPage:
		browserCtx.opts = page.opts
	case *CustomCDPPage:
		browserCtx.opts, browserCtx.custom = page.opts, page
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
	// 默认 CustomCDP：所有目标在运行前下发身份；只有显式要求时才用旧的 chromedp 通道
	if opts.UseChromedp {
		return NewCDPConnector().Connect(ctx, chrome, opts)
	}
	return CreateCustomCDPConnector().Connect(ctx, chrome, opts)
}

// Connect is a convenience function to create and connect a browser
func Connect(ctx context.Context, opts *ConnectOptions) (*BrowserInstance, error) {
	browser := NewRealBrowser()
	return browser.Connect(ctx, opts)
}
