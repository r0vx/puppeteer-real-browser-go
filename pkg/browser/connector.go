package browser

import (
	"context"
	"encoding/base64"
	"encoding/json"
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

// WaitUntil 定义导航等待策略
type WaitUntil string

const (
	WaitLoad            WaitUntil = "load"            // 等待 load 事件（默认）
	WaitDOMContentLoaded WaitUntil = "domcontentloaded" // 等待 DOMContentLoaded
	WaitNetworkIdle0    WaitUntil = "networkidle0"    // 500ms 内无网络请求
	WaitNetworkIdle2    WaitUntil = "networkidle2"    // 500ms 内 ≤2 个网络请求
)

// NavigateOptions 导航选项
type NavigateOptions struct {
	WaitUntil WaitUntil     // 等待策略
	Timeout   time.Duration // 超时时间
	Referrer  string        // Referrer
}

// Navigate navigates to the specified URL (waits for load event)
func (p *CDPPage) Navigate(url string) error {
	return chromedp.Run(p.ctx, chromedp.Navigate(url))
}

// NavigateWithOptions navigates with custom options (like puppeteer page.goto)
func (p *CDPPage) NavigateWithOptions(url string, opts *NavigateOptions) error {
	if opts == nil {
		return p.Navigate(url)
	}

	ctx := p.ctx
	if opts.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(p.ctx, opts.Timeout)
		defer cancel()
	}

	// 构建导航 action
	var actions []chromedp.Action

	// 如果有 Referrer，先设置
	if opts.Referrer != "" {
		actions = append(actions, chromedp.ActionFunc(func(ctx context.Context) error {
			return page.SetDocumentContent("", fmt.Sprintf(`<script>Object.defineProperty(document, 'referrer', {get: () => '%s'})</script>`, opts.Referrer)).Do(ctx)
		}))
	}

	// 根据 WaitUntil 策略选择等待方式
	switch opts.WaitUntil {
	case WaitDOMContentLoaded:
		// 只等待 DOM 解析完成
		actions = append(actions, chromedp.ActionFunc(func(ctx context.Context) error {
			_, _, _, _, err := page.Navigate(url).Do(ctx)
			return err
		}))
		actions = append(actions, chromedp.WaitReady("body", chromedp.ByQuery))

	case WaitNetworkIdle0, WaitNetworkIdle2:
		// 等待网络空闲
		actions = append(actions, chromedp.Navigate(url))
		actions = append(actions, p.waitNetworkIdle(opts.WaitUntil == WaitNetworkIdle0))

	default: // WaitLoad 或默认
		actions = append(actions, chromedp.Navigate(url))
	}

	return chromedp.Run(ctx, actions...)
}

// NavigateWithReferrer navigates with a referrer header
func (p *CDPPage) NavigateWithReferrer(url, referrer string) error {
	return chromedp.Run(p.ctx, chromedp.ActionFunc(func(ctx context.Context) error {
		_, _, _, _, err := page.Navigate(url).WithReferrer(referrer).Do(ctx)
		return err
	}))
}

// waitNetworkIdle waits for network to become idle
func (p *CDPPage) waitNetworkIdle(strict bool) chromedp.Action {
	return chromedp.ActionFunc(func(ctx context.Context) error {
		maxPending := 2
		if strict {
			maxPending = 0
		}

		pendingRequests := 0
		idleStart := time.Time{}
		done := make(chan struct{})
		timeout := time.After(30 * time.Second)

		chromedp.ListenTarget(ctx, func(ev interface{}) {
			switch ev.(type) {
			case *network.EventRequestWillBeSent:
				pendingRequests++
				idleStart = time.Time{}
			case *network.EventLoadingFinished, *network.EventLoadingFailed:
				pendingRequests--
				if pendingRequests < 0 {
					pendingRequests = 0
				}
				if pendingRequests <= maxPending {
					if idleStart.IsZero() {
						idleStart = time.Now()
					}
				}
			}
		})

		// 检查网络空闲
		ticker := time.NewTicker(100 * time.Millisecond)
		defer ticker.Stop()

		for {
			select {
			case <-timeout:
				return nil // 超时也继续
			case <-done:
				return nil
			case <-ticker.C:
				if !idleStart.IsZero() && time.Since(idleStart) >= 500*time.Millisecond {
					return nil
				}
			case <-ctx.Done():
				return ctx.Err()
			}
		}
	})
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

// ==================== Cookie/Storage 管理 ====================

// SetCookies sets cookies for the page
func (p *CDPPage) SetCookies(cookiesJSON string, url string) error {
	return chromedp.Run(p.ctx, chromedp.ActionFunc(func(ctx context.Context) error {
		// 解析 cookies JSON
		var cookies []*network.CookieParam
		if err := json.Unmarshal([]byte(cookiesJSON), &cookies); err != nil {
			return fmt.Errorf("failed to parse cookies: %w", err)
		}

		// 设置每个 cookie 的 URL
		for _, cookie := range cookies {
			if cookie.URL == "" {
				cookie.URL = url
			}
		}

		return network.SetCookies(cookies).Do(ctx)
	}))
}

// GetCookies gets all cookies for the current page
// GetCookies gets all cookies as JSON string
func (p *CDPPage) GetCookies() (string, error) {
	var cookies []*network.Cookie
	err := chromedp.Run(p.ctx, chromedp.ActionFunc(func(ctx context.Context) error {
		var err error
		cookies, err = network.GetCookies().Do(ctx)
		return err
	}))
	if err != nil {
		return "", err
	}
	data, err := json.Marshal(cookies)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// ClearCookies clears all cookies
func (p *CDPPage) ClearCookies() error {
	return chromedp.Run(p.ctx, chromedp.ActionFunc(func(ctx context.Context) error {
		return network.ClearBrowserCookies().Do(ctx)
	}))
}

// SetLocalStorage sets localStorage data
func (p *CDPPage) SetLocalStorage(dataJSON string) error {
	script := fmt.Sprintf(`
		(function() {
			const data = %s;
			for (const [key, value] of Object.entries(data)) {
				localStorage.setItem(key, typeof value === 'string' ? value : JSON.stringify(value));
			}
		})()
	`, dataJSON)
	_, err := p.Evaluate(script)
	return err
}

// GetLocalStorage gets all localStorage data as JSON
func (p *CDPPage) GetLocalStorage() (string, error) {
	result, err := p.Evaluate(`JSON.stringify(localStorage)`)
	if err != nil {
		return "", err
	}
	if str, ok := result.(string); ok {
		return str, nil
	}
	return "{}", nil
}

// SetSessionStorage sets sessionStorage data
func (p *CDPPage) SetSessionStorage(dataJSON string) error {
	script := fmt.Sprintf(`
		(function() {
			const data = %s;
			for (const [key, value] of Object.entries(data)) {
				sessionStorage.setItem(key, typeof value === 'string' ? value : JSON.stringify(value));
			}
		})()
	`, dataJSON)
	_, err := p.Evaluate(script)
	return err
}

// GetSessionStorage gets all sessionStorage data as JSON
func (p *CDPPage) GetSessionStorage() (string, error) {
	result, err := p.Evaluate(`JSON.stringify(sessionStorage)`)
	if err != nil {
		return "", err
	}
	if str, ok := result.(string); ok {
		return str, nil
	}
	return "{}", nil
}

// ==================== 等待方法 ====================

// WaitVisible waits for element to be visible with timeout
func (p *CDPPage) WaitVisible(selector string, timeout time.Duration) error {
	escaped := escapeSelector(selector)
	deadline := time.Now().Add(timeout)

	for time.Now().Before(deadline) {
		result, err := p.Evaluate(fmt.Sprintf(`
			(function() {
				const elem = document.querySelector('%s');
				if (!elem) return false;
				const rect = elem.getBoundingClientRect();
				const style = window.getComputedStyle(elem);
				
				// display:contents 特殊处理（元素存在但宽高为0）
				if (style.display === 'contents') {
					return style.visibility !== 'hidden' && style.opacity !== '0';
				}
				
				return (rect.width > 0 || rect.height > 0) && 
				       style.visibility !== 'hidden' && 
				       style.display !== 'none' &&
				       style.opacity !== '0';
			})()
		`, escaped))
		if err == nil {
			if visible, ok := result.(bool); ok && visible {
				return nil
			}
		}
		time.Sleep(100 * time.Millisecond)
	}
	return fmt.Errorf("timeout waiting for element visible: %s", selector)
}

// WaitNotVisible waits for element to disappear with timeout
func (p *CDPPage) WaitNotVisible(selector string, timeout time.Duration) error {
	escaped := escapeSelector(selector)
	deadline := time.Now().Add(timeout)

	for time.Now().Before(deadline) {
		result, err := p.Evaluate(fmt.Sprintf(`
			(function() {
				const elem = document.querySelector('%s');
				if (!elem) return true; // 不存在即为不可见
				const rect = elem.getBoundingClientRect();
				const style = window.getComputedStyle(elem);
				// 宽高为0或隐藏
				return rect.width === 0 || rect.height === 0 || 
				       style.visibility === 'hidden' || 
				       style.display === 'none' ||
				       style.opacity === '0';
			})()
		`, escaped))
		if err == nil {
			if notVisible, ok := result.(bool); ok && notVisible {
				return nil
			}
		}
		time.Sleep(100 * time.Millisecond)
	}
	return fmt.Errorf("timeout waiting for element to disappear: %s", selector)
}

// WaitVisibleByID waits for element by ID to be visible
// Has checks if element exists in the DOM
func (p *CDPPage) Has(selector string) (bool, error) {
	result, err := p.Evaluate(fmt.Sprintf(`document.querySelector('%s') !== null`, escapeSelector(selector)))
	if err != nil {
		return false, err
	}
	if b, ok := result.(bool); ok {
		return b, nil
	}
	return false, nil
}

// ==================== 便捷方法 ====================

// ExecuteJS executes JavaScript and returns result (alias for Evaluate)
func (p *CDPPage) ExecuteJS(script string, result interface{}) error {
	return chromedp.Run(p.ctx, chromedp.Evaluate(script, result))
}

// Refresh refreshes the current page
func (p *CDPPage) Refresh(timeout time.Duration) error {
	ctx := p.ctx
	if timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(p.ctx, timeout)
		defer cancel()
	}
	return chromedp.Run(ctx, chromedp.Reload())
}

// Sleep pauses execution for the specified duration
func (p *CDPPage) Sleep(duration time.Duration) {
	time.Sleep(duration)
}

// ==================== 元素截图 ====================

// ScreenshotElement takes a screenshot of a specific element
func (p *CDPPage) ScreenshotElement(selector string) ([]byte, error) {
	// 先检查元素是否存在
	has, err := p.Has(selector)
	if err != nil {
		return nil, err
	}
	if !has {
		return nil, fmt.Errorf("element not found: %s", selector)
	}

	// 直接截图，不使用 WaitVisible
	var buf []byte
	ctx, cancel := context.WithTimeout(p.ctx, 5*time.Second)
	defer cancel()

	err = chromedp.Run(ctx,
		chromedp.Screenshot(selector, &buf, chromedp.NodeVisible),
	)
	return buf, err
}

// ScreenshotQrcode takes a screenshot of an element and returns base64 string
func (p *CDPPage) ScreenshotQrcode(selector string) (string, error) {
	buf, err := p.ScreenshotElement(selector)
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(buf), nil
}

// ==================== 网络监听 ====================

// ==================== 辅助函数 ====================

// escapeSelector escapes special characters in selector for JavaScript
func escapeSelector(selector string) string {
	escaped := strings.ReplaceAll(selector, `\`, `\\`)
	escaped = strings.ReplaceAll(escaped, `'`, `\'`)
	return escaped
}
