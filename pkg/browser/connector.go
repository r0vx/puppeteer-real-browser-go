package browser

import (
	"context"
	"encoding/base64"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/chromedp/cdproto/dom"
	"github.com/chromedp/cdproto/fetch"
	"github.com/chromedp/cdproto/network"
	"github.com/chromedp/cdproto/page"
	"github.com/chromedp/chromedp"
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

	// Simply create a new context - this will create a new tab
	// Note: Chrome will have the default blank tab + this new tab (2 tabs total)
	// This is the standard chromedp behavior and is acceptable
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
	ctx                   context.Context
	cancel                context.CancelFunc
	allocCtx              context.Context
	allocCancel           context.CancelFunc
	chrome                *ChromeProcess
	opts                  *ConnectOptions
	initialized           bool
	requestHandler        RequestHandler
	interceptEnabled      bool
	targetHandler         *TargetHandler
	requestListenerCancel context.CancelFunc // 用于取消请求监听器
	requestListenerMu     sync.Mutex         // 保护监听器操作
}

// GetContext 返回 chromedp 上下文（用于直接调用 chromedp 方法）
func (p *CDPPage) GetContext() context.Context {
	return p.ctx
}

// initialize sets up the page with Runtime.Enable bypass (rebrowser-patches style)
func (p *CDPPage) initialize() error {
	// CRITICAL: Completely avoid Runtime.Enable to prevent Cloudflare detection
	// This is the core issue that was causing detection

	err := chromedp.Run(p.ctx,
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

	if err != nil {
		return err
	}

	// Start target handler to manage new pages/tabs (like original Node.js targetcreated event)
	p.targetHandler = NewTargetHandler(p.allocCtx, p.opts)
	if err := p.targetHandler.Start(p.ctx); err != nil {
		return fmt.Errorf("failed to start target handler: %w", err)
	}

	return nil
}

// Navigate navigates to the specified URL
func (p *CDPPage) Navigate(url string) error {
	// Navigate will invalidate the current execution context
	// We need to create a fresh chromedp context after navigation
	// But we can't change p.ctx without breaking other things
	// So we just navigate and accept that execution context might change
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
			"note":   "Evaluation attempted without Runtime.Enable",
			"script": script,
		}, nil
	}

	// For now, return a success indicator since the actual evaluation
	// happens on next page load/navigation
	return map[string]interface{}{
		"note":   "Script queued for next navigation - Runtime.Enable avoided",
		"script": script,
	}, nil
}

// WaitForSelector waits for an element to appear
func (p *CDPPage) WaitForSelector(selector string) error {
	return chromedp.Run(p.ctx, chromedp.WaitVisible(selector))
}

// ClickSelector clicks an element by CSS selector using chromedp native method
// This is more reliable than coordinate-based clicking for standard elements
func (p *CDPPage) ClickSelector(selector string) error {
	return chromedp.Run(p.ctx,
		chromedp.WaitVisible(selector),
		chromedp.Click(selector, chromedp.NodeVisible),
	)
}

// RealClickSelector clicks an element with human-like mouse movement
// Uses Bezier curve trajectory to move to element before clicking
func (p *CDPPage) RealClickSelector(selector string) error {
	// 先等待元素可见
	if err := chromedp.Run(p.ctx, chromedp.WaitVisible(selector)); err != nil {
		return fmt.Errorf("element not visible: %w", err)
	}

	// 获取元素坐标
	var x, y float64
	err := chromedp.Run(p.ctx, chromedp.Evaluate(fmt.Sprintf(`
		(function() {
			const elem = document.querySelector('%s');
			if (!elem) return null;
			
			elem.scrollIntoViewIfNeeded ? elem.scrollIntoViewIfNeeded() : elem.scrollIntoView({block: 'center'});
			
			const rect = elem.getBoundingClientRect();
			// 添加随机偏移更像人类
			const rx = (Math.random() - 0.5) * Math.min(rect.width * 0.3, 8);
			const ry = (Math.random() - 0.5) * Math.min(rect.height * 0.3, 8);
			
			return {
				x: rect.left + rect.width / 2 + rx,
				y: rect.top + rect.height / 2 + ry
			};
		})()
	`, selector), &map[string]float64{"x": 0, "y": 0}))
	if err != nil {
		return fmt.Errorf("failed to get element coords: %w", err)
	}

	// 从 Evaluate 结果中提取坐标
	var coordResult map[string]interface{}
	err = chromedp.Run(p.ctx, chromedp.Evaluate(fmt.Sprintf(`
		(function() {
			const elem = document.querySelector('%s');
			if (!elem) return null;
			const rect = elem.getBoundingClientRect();
			const rx = (Math.random() - 0.5) * Math.min(rect.width * 0.3, 8);
			const ry = (Math.random() - 0.5) * Math.min(rect.height * 0.3, 8);
			return {
				x: rect.left + rect.width / 2 + rx,
				y: rect.top + rect.height / 2 + ry
			};
		})()
	`, selector), &coordResult))
	if err != nil {
		return fmt.Errorf("failed to get element coords: %w", err)
	}

	if coordResult == nil {
		return fmt.Errorf("element not found: %s", selector)
	}

	x, _ = coordResult["x"].(float64)
	y, _ = coordResult["y"].(float64)

	// 使用拟人化点击
	return p.RealClick(x, y)
}

// SendKeys types text into an element using chromedp native method
func (p *CDPPage) SendKeys(selector, text string) error {
	return chromedp.Run(p.ctx,
		chromedp.WaitVisible(selector),
		chromedp.SendKeys(selector, text),
	)
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
	// 取消请求监听器
	p.requestListenerMu.Lock()
	if p.requestListenerCancel != nil {
		p.requestListenerCancel()
		p.requestListenerCancel = nil
	}
	p.requestListenerMu.Unlock()

	// Stop target handler
	if p.targetHandler != nil {
		p.targetHandler.Stop()
	}

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
		// Enable fetch with auth request handling for proxy authentication
		if err := fetch.Enable().WithHandleAuthRequests(true).Do(ctx); err != nil {
			return fmt.Errorf("failed to enable Fetch with auth: %w", err)
		}

		// Listen for auth required events (proxy authentication challenges)
		chromedp.ListenTarget(ctx, func(ev interface{}) {
			if authEv, ok := ev.(*fetch.EventAuthRequired); ok {
				go p.handleProxyAuth(ctx, authEv)
			}
		})

		return nil
	})
}

// handleProxyAuth handles proxy authentication challenges
func (p *CDPPage) handleProxyAuth(ctx context.Context, ev *fetch.EventAuthRequired) {
	if p.opts.Proxy == nil || p.opts.Proxy.Username == "" {
		// No credentials, cancel the auth
		fetch.ContinueWithAuth(ev.RequestID, &fetch.AuthChallengeResponse{
			Response: fetch.AuthChallengeResponseResponseCancelAuth,
		}).Do(ctx)
		return
	}

	// Provide credentials for proxy authentication
	fetch.ContinueWithAuth(ev.RequestID, &fetch.AuthChallengeResponse{
		Response: fetch.AuthChallengeResponseResponseProvideCredentials,
		Username: p.opts.Proxy.Username,
		Password: p.opts.Proxy.Password,
	}).Do(ctx)
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

// SetRequestInterception enables or disables request interception
func (p *CDPPage) SetRequestInterception(enabled bool) error {
	p.requestListenerMu.Lock()
	defer p.requestListenerMu.Unlock()

	// 先取消旧的监听器（如果存在）
	if p.requestListenerCancel != nil {
		p.requestListenerCancel()
		p.requestListenerCancel = nil
	}

	p.interceptEnabled = enabled

	if enabled {
		// 创建专用 context 用于监听器
		listenerCtx, cancel := context.WithCancel(p.ctx)
		p.requestListenerCancel = cancel

		// Enable both Network and Fetch domains for comprehensive request interception
		return chromedp.Run(p.ctx, chromedp.ActionFunc(func(ctx context.Context) error {
			// Enable Network domain first
			if err := network.Enable().Do(ctx); err != nil {
				return fmt.Errorf("failed to enable Network domain: %w", err)
			}

			// Enable fetch domain with request patterns - intercept everything
			patterns := []*fetch.RequestPattern{{
				URLPattern: "*",
			}}
			if err := fetch.Enable().WithHandleAuthRequests(false).WithPatterns(patterns).Do(ctx); err != nil {
				return fmt.Errorf("failed to enable Fetch domain: %w", err)
			}

			// Set up request interception listener（使用专用 context）
			chromedp.ListenTarget(listenerCtx, func(ev interface{}) {
				switch e := ev.(type) {
				case *fetch.EventRequestPaused:
					// Handle in a goroutine to avoid blocking
					go func() {
						// 检查 listenerCtx 是否已取消
						select {
						case <-listenerCtx.Done():
							return
						default:
						}

						if p.requestHandler != nil {
							// Create InterceptedRequest
							req := &InterceptedRequest{
								URL:          e.Request.URL,
								Method:       e.Request.Method,
								Headers:      make(map[string]string),
								ResourceType: string(e.ResourceType),
								RequestID:    string(e.RequestID),
							}

							// Convert headers
							for name, value := range e.Request.Headers {
								if str, ok := value.(string); ok {
									req.Headers[name] = str
								}
							}

							// Set page context for request operations
							req.setPageContext(p)

							// Call handler
							if err := p.requestHandler(req); err != nil {
								// If handler fails, continue the request
								fetch.ContinueRequest(e.RequestID).Do(ctx)
							}
						} else {
							// No handler, continue request
							fetch.ContinueRequest(e.RequestID).Do(ctx)
						}
					}()
				}
			})

			return nil
		}))
	} else {
		// Disable fetch domain
		return chromedp.Run(p.ctx, chromedp.ActionFunc(func(ctx context.Context) error {
			return fetch.Disable().Do(ctx)
		}))
	}
}

// OnRequest sets the request handler for intercepted requests
func (p *CDPPage) OnRequest(handler RequestHandler) error {
	p.requestHandler = handler
	return nil
}

// continueRequest continues an intercepted request
func (p *CDPPage) continueRequest(requestID string) error {
	return chromedp.Run(p.ctx, chromedp.ActionFunc(func(ctx context.Context) error {
		return fetch.ContinueRequest(fetch.RequestID(requestID)).Do(ctx)
	}))
}

// respondToRequest responds to an intercepted request with custom response
func (p *CDPPage) respondToRequest(requestID string, response *RequestResponse) error {
	return chromedp.Run(p.ctx, chromedp.ActionFunc(func(ctx context.Context) error {
		// Convert headers to the format expected by CDP
		headers := make([]*fetch.HeaderEntry, 0, len(response.Headers))
		for name, value := range response.Headers {
			headers = append(headers, &fetch.HeaderEntry{
				Name:  name,
				Value: value,
			})
		}

		// Ensure we have content-type header
		hasContentType := false
		for _, header := range headers {
			if strings.ToLower(header.Name) == "content-type" {
				hasContentType = true
				break
			}
		}
		if !hasContentType {
			headers = append(headers, &fetch.HeaderEntry{
				Name:  "content-type",
				Value: "text/html; charset=utf-8",
			})
		}

		// Use FulfillRequest with proper base64 encoding for body
		cmd := fetch.FulfillRequest(fetch.RequestID(requestID), int64(response.Status))

		if len(headers) > 0 {
			cmd = cmd.WithResponseHeaders(headers)
		}

		if response.Body != "" {
			// Fetch.FulfillRequest expects body as base64 encoded string
			bodyBase64 := base64.StdEncoding.EncodeToString([]byte(response.Body))
			cmd = cmd.WithBody(bodyBase64)
		}

		err := cmd.Do(ctx)
		if err != nil {
			// Log error but don't return it immediately, try to continue the request instead
			fmt.Printf("Failed to fulfill request: %v, continuing request instead\n", err)
			return fetch.ContinueRequest(fetch.RequestID(requestID)).Do(ctx)
		}

		return nil
	}))
}

// abortRequest aborts an intercepted request
func (p *CDPPage) abortRequest(requestID string) error {
	return chromedp.Run(p.ctx, chromedp.ActionFunc(func(ctx context.Context) error {
		return fetch.FailRequest(fetch.RequestID(requestID), network.ErrorReasonAborted).Do(ctx)
	}))
}
