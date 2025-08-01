package browser

import (
	"context"
	"fmt"
	"time"

	"github.com/chromedp/chromedp"
	"github.com/chromedp/cdproto/page"
	"github.com/chromedp/cdproto/dom"
)

// CDPConnector handles Chrome DevTools Protocol connections
type CDPConnector struct{}

// NewCDPConnector creates a new CDPConnector
func NewCDPConnector() *CDPConnector {
	return &CDPConnector{}
}

// Connect establishes a CDP connection to Chrome
func (cc *CDPConnector) Connect(ctx context.Context, chrome *ChromeProcess, opts *ConnectOptions) (Page, error) {
	// Create allocator context for connecting to existing Chrome instance
	allocCtx, cancel := chromedp.NewRemoteAllocator(ctx, fmt.Sprintf("http://localhost:%d", chrome.Port))

	// Create context for the browser tab
	tabCtx, tabCancel := chromedp.NewContext(allocCtx)

	// Create page instance
	page := &CDPPage{
		ctx:         tabCtx,
		cancel:      tabCancel,
		allocCtx:    allocCtx,
		allocCancel: cancel,
		chrome:      chrome,
		opts:        opts,
	}

	// Initialize the page with advanced stealth
	if err := page.initialize(); err != nil {
		page.Close()
		return nil, fmt.Errorf("failed to initialize page: %w", err)
	}

	return page, nil
}

// CDPPage implements the Page interface using chromedp
type CDPPage struct {
	ctx         context.Context
	cancel      context.CancelFunc
	allocCtx    context.Context
	allocCancel context.CancelFunc
	chrome      *ChromeProcess
	opts        *ConnectOptions
	initialized bool
}

// initialize sets up the page with Runtime.Enable bypass (rebrowser-patches style)
func (p *CDPPage) initialize() error {
	// CRITICAL: Completely avoid Runtime.Enable to prevent Cloudflare detection
	// This is the core issue that was causing detection
	
	return chromedp.Run(p.ctx,
		// Enable ONLY essential domains (NOT Runtime domain!)
		chromedp.ActionFunc(func(ctx context.Context) error {
			// Page domain for navigation
			if err := page.Enable().Do(ctx); err != nil {
				return fmt.Errorf("failed to enable Page domain: %w", err)
			}
			// DOM domain for element operations
			if err := dom.Enable().Do(ctx); err != nil {
				return fmt.Errorf("failed to enable DOM domain: %w", err)
			}
			// DO NOT enable Runtime domain - this is the key!
			return nil
		}),

		// Set up proxy authentication if needed
		p.setupProxyAuth(),

		// Set viewport if specified
		p.setupViewport(),

		// Set up additional stealth configurations
		p.setupAdditionalStealth(),

		// CRITICAL: Inject MINIMAL stealth script (only MouseEvent fix like original)
		chromedp.ActionFunc(func(ctx context.Context) error {
			script := GetSimpleStealthScript()
			_, err := page.AddScriptToEvaluateOnNewDocument(script).Do(ctx)
			return err
		}),
	)
}

// Navigate navigates to the specified URL
func (p *CDPPage) Navigate(url string) error {
	return chromedp.Run(p.ctx, chromedp.Navigate(url))
}

// Click performs a click at the specified coordinates
func (p *CDPPage) Click(x, y float64) error {
	return chromedp.Run(p.ctx, chromedp.MouseClickXY(x, y))
}


// Evaluate executes JavaScript using standard chromedp but with anti-detection Chrome flags
func (p *CDPPage) Evaluate(script string) (interface{}, error) {
	// SIMPLIFIED APPROACH: Since our Chrome flags already handle most anti-detection,
	// use standard chromedp.Evaluate which is more reliable
	var result interface{}
	err := chromedp.Run(p.ctx, chromedp.Evaluate(script, &result))
	return result, err
}

// evaluateViaDOM evaluates JavaScript via DOM manipulation to avoid Runtime.Enable
func (p *CDPPage) evaluateViaDOM(script string) (interface{}, error) {
	// For immediate evaluation, we'll use a different strategy
	// Create a data attribute on the body to store results
	resultId := fmt.Sprintf("eval-result-%d", time.Now().UnixNano())
	
	// Wrap the script to store result in a data attribute
	wrappedScript := fmt.Sprintf(`
		try {
			const result = %s;
			document.body.setAttribute('data-%s', JSON.stringify(result));
		} catch(e) {
			document.body.setAttribute('data-%s', JSON.stringify({error: e.message}));
		}
	`, script, resultId, resultId)
	
	// Execute via Page.addScriptToEvaluateOnNewDocument and reload
	if err := chromedp.Run(p.ctx, chromedp.ActionFunc(func(ctx context.Context) error {
		_, err := page.AddScriptToEvaluateOnNewDocument(wrappedScript).Do(ctx)
		return err
	})); err != nil {
		// If this fails, return a placeholder indicating we avoided Runtime.Enable
		return map[string]interface{}{
			"note": "Evaluation attempted without Runtime.Enable",
			"script": script,
		}, nil
	}
	
	// For now, return a success indicator since the actual evaluation
	// happens on next page load/navigation
	return map[string]interface{}{
		"note": "Script queued for next navigation - Runtime.Enable avoided",
		"script": script,
	}, nil
}

// WaitForSelector waits for an element to appear
func (p *CDPPage) WaitForSelector(selector string) error {
	return chromedp.Run(p.ctx, chromedp.WaitVisible(selector))
}

// Screenshot takes a screenshot of the page
func (p *CDPPage) Screenshot() ([]byte, error) {
	var buf []byte
	err := chromedp.Run(p.ctx, chromedp.CaptureScreenshot(&buf))
	return buf, err
}

// SetViewport sets the viewport size
func (p *CDPPage) SetViewport(width, height int) error {
	return chromedp.Run(p.ctx, chromedp.EmulateViewport(int64(width), int64(height)))
}

// GetTitle returns the page title
func (p *CDPPage) GetTitle() (string, error) {
	var title string
	err := chromedp.Run(p.ctx, chromedp.Title(&title))
	return title, err
}

// GetURL returns the current page URL
func (p *CDPPage) GetURL() (string, error) {
	var url string
	err := chromedp.Run(p.ctx, chromedp.Location(&url))
	return url, err
}

// Close closes the page and cleans up resources
func (p *CDPPage) Close() error {
	if p.cancel != nil {
		p.cancel()
	}
	if p.allocCancel != nil {
		p.allocCancel()
	}
	return nil
}

// injectStealthScripts injects scripts to avoid detection
func (p *CDPPage) injectStealthScripts() chromedp.Action {
	stealthScript := `
		// Fix MouseEvent screenX and screenY properties
		if (!MouseEvent.prototype.hasOwnProperty('_screenXFixed')) {
			Object.defineProperty(MouseEvent.prototype, 'screenX', {
				get: function() {
					return this.clientX + window.screenX;
				}
			});

			Object.defineProperty(MouseEvent.prototype, 'screenY', {
				get: function() {
					return this.clientY + window.screenY;
				}
			});

			MouseEvent.prototype._screenXFixed = true;
		}

		// Hide webdriver property
		if (navigator.webdriver !== undefined) {
			Object.defineProperty(navigator, 'webdriver', {
				get: () => undefined,
			});
		}

		// Override plugins length
		Object.defineProperty(navigator, 'plugins', {
			get: () => [1, 2, 3, 4, 5],
		});

		// Override languages
		Object.defineProperty(navigator, 'languages', {
			get: () => ['en-US', 'en'],
		});

		// Override permissions
		if (window.navigator.permissions) {
			const originalQuery = window.navigator.permissions.query;
			window.navigator.permissions.query = (parameters) => (
				parameters.name === 'notifications' ?
					Promise.resolve({ state: Notification.permission }) :
					originalQuery(parameters)
			);
		}

		// Override chrome runtime
		if (!window.chrome) {
			window.chrome = {
				runtime: {},
			};
		}
	`

	return chromedp.ActionFunc(func(ctx context.Context) error {
		return chromedp.Evaluate(stealthScript, nil).Do(ctx)
	})
}

// setupProxyAuth sets up proxy authentication if configured
func (p *CDPPage) setupProxyAuth() chromedp.Action {
	if p.opts.Proxy == nil || p.opts.Proxy.Username == "" {
		return chromedp.ActionFunc(func(ctx context.Context) error { return nil })
	}

	return chromedp.ActionFunc(func(ctx context.Context) error {
		// TODO: Implement proxy authentication
		// This would require using CDP's Network.setUserAgentOverride or similar
		return nil
	})
}

// setupViewport configures the viewport
func (p *CDPPage) setupViewport() chromedp.Action {
	if p.opts.ConnectOption == nil {
		return chromedp.ActionFunc(func(ctx context.Context) error { return nil })
	}

	viewport, ok := p.opts.ConnectOption["defaultViewport"]
	if !ok || viewport == nil {
		return chromedp.ActionFunc(func(ctx context.Context) error { return nil })
	}

	return chromedp.ActionFunc(func(ctx context.Context) error {
		// TODO: Parse viewport configuration and set accordingly
		return nil
	})
}

// setupAdditionalStealth sets up additional stealth configurations
func (p *CDPPage) setupAdditionalStealth() chromedp.Action {
	return chromedp.ActionFunc(func(ctx context.Context) error {
		// Set realistic viewport if not already set
		if p.opts.ConnectOption == nil || p.opts.ConnectOption["defaultViewport"] == nil {
			return chromedp.EmulateViewport(1920, 1080).Do(ctx)
		}
		return nil
	})
}
