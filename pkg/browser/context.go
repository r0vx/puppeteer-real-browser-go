package browser

import (
	"context"
	"fmt"

	"github.com/chromedp/chromedp"
)

// NewPage creates a new page in this browser context (like puppeteer context.newPage())
// This creates a new tab/context within the browser context
func (bc *BrowserContext) NewPage() (Page, error) {
	// Create a new chromedp context within this browser context
	// This is equivalent to creating a new tab in the same browser context
	tabCtx, tabCancel := chromedp.NewContext(bc.allocCtx)

	// Create page instance using the new context
	opts := bc.opts
	if opts == nil {
		// Create default options if not available
		opts = &ConnectOptions{
			Headless:     false,
			Turnstile:    true,
			UseCustomCDP: false,
		}
	}
	
	page := &CDPPage{
		ctx:         tabCtx,
		cancel:      tabCancel,
		allocCtx:    bc.allocCtx,
		allocCancel: bc.allocCancel, // Don't close allocator, only tab
		chrome:      bc.chrome,
		opts:        opts,
	}

	// Initialize the page with advanced stealth
	if err := page.initialize(); err != nil {
		page.Close()
		return nil, fmt.Errorf("failed to initialize page: %w", err)
	}

	return page, nil
}

// Close closes the browser context and all its pages
func (bc *BrowserContext) Close() error {
	if bc.allocCancel != nil {
		bc.allocCancel()
	}
	return nil
}

// Pages returns all pages in this context (simplified implementation)
func (bc *BrowserContext) Pages() ([]Page, error) {
	// This is a simplified implementation
	// In a full implementation, we would track all pages created in this context
	return []Page{}, nil
}

// CreateBrowserInstance creates a BrowserInstance from this context with a new page
// This is equivalent to Node.js context.newPage() but returns a BrowserInstance
func (bc *BrowserContext) CreateBrowserInstance() (*BrowserInstance, error) {
	// Create a new page in this context
	page, err := bc.NewPage()
	if err != nil {
		return nil, fmt.Errorf("failed to create page: %w", err)
	}

	// Create context for the browser instance (NOT tied to the browser context)
	ctx, cancel := context.WithCancel(context.Background())

	// Create BrowserInstance that wraps this page only
	// DO NOT close the browser context when this instance closes
	// 🔧 FIX: Don't pass chrome reference to prevent killing global browser
	instance := &BrowserInstance{
		browser: nil, // We don't have a Browser interface here
		page:    page,
		chrome:  nil, // Don't pass chrome - prevents killing global browser process
		ctx:     ctx,
		cancel:  cancel, // Simple cancel, don't touch the browser context
	}

	return instance, nil
}