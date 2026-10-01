package browser

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/chromedp/cdproto/cdp"
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
		cursor:      NewGhostCursor(),
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
	ctx               context.Context
	cancel            context.CancelFunc
	allocCtx          context.Context
	allocCancel       context.CancelFunc
	chrome            *ChromeProcess
	opts              *ConnectOptions
	identity          *Identity // 下发给本页的身份（SetViewport 要保留它的屏幕与 DPR）
	initialized       bool
	requestHandler    RequestHandler
	interceptEnabled  bool
	targetHandler     *TargetHandler
	fetchListening    bool         // Fetch 事件分发器是否已注册（只注册一次，避免重复放行同一请求）
	requestListenerMu sync.Mutex   // 保护 requestHandler / interceptEnabled / fetchListening
	cursor            *GhostCursor // 持久化拟人光标，避免每次点击瞬移
}

// GetContext 返回 chromedp 上下文（用于直接调用 chromedp 方法）
func (p *CDPPage) GetContext() context.Context {
	return p.ctx
}

// initialize sets up the page with Runtime.Enable bypass (rebrowser-patches style)
func (p *CDPPage) initialize() error {
	// CRITICAL: Completely avoid Runtime.Enable to prevent Cloudflare detection
	// This is the core issue that was causing detection

	// 必须在 chromedp 建出自己的标签页之前读取：此时启动标签页是唯一的页面（见 browserHostInfo）
	host, err := browserHostInfo(p.chrome)
	if err != nil {
		return err
	}

	err = chromedp.Run(p.ctx,
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

		// 身份：由真实浏览器信息派生并下发到主页面（chromedp 路径不接管 iframe / Worker / 弹窗，见子项目 D）
		chromedp.ActionFunc(func(ctx context.Context) error {
			// CustomCDP 实例上开的页面：target manager 已在它运行前下发了账号身份。本会话必须下发同一份，
			// 不能另算（opts 不同会得到默认身份），也不能不下发（本会话渲染端的空模拟状态会盖掉 navigator.platform）
			p.chrome.hostMu.Lock()
			id := p.chrome.managedIdentity
			p.chrome.hostMu.Unlock()
			if id == nil {
				var err error
				if id, err = identityForOptions(p.opts, host); err != nil {
					return err
				}
			}
			call := func(method string, params any) (json.RawMessage, error) {
				return nil, cdp.Execute(ctx, method, params, nil)
			}
			if err := applyIdentity(call, id, kindPage); err != nil {
				return err
			}
			p.identity = id
			if id.Screen.Width > 0 {
				return setChromedpWindowBounds(ctx, windowBounds(id))
			}
			return nil
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
	WaitLoad             WaitUntil = "load"             // 等待 load 事件（默认）
	WaitDOMContentLoaded WaitUntil = "domcontentloaded" // 等待 DOMContentLoaded
	WaitNetworkIdle0     WaitUntil = "networkidle0"     // 500ms 内无网络请求
	WaitNetworkIdle2     WaitUntil = "networkidle2"     // 500ms 内 ≤2 个网络请求
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

// lifecycleEventName 把等待策略映射为 Page.lifecycleEvent 名称。与 puppeteer 同源：
// networkAlmostIdle = 500ms 内不超过 2 个连接，networkIdle = 500ms 内没有连接
func lifecycleEventName(w WaitUntil) string {
	switch w {
	case WaitDOMContentLoaded:
		return "DOMContentLoaded"
	case WaitNetworkIdle0:
		return "networkIdle"
	case WaitNetworkIdle2:
		return "networkAlmostIdle"
	default:
		return "load"
	}
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
	return chromedp.Run(ctx, chromedp.ActionFunc(func(ctx context.Context) error {
		return navigateAndWait(ctx, url, opts.Referrer, lifecycleEventName(opts.WaitUntil))
	}))
}

// NavigateWithReferrer navigates with a referrer header and waits for load
func (p *CDPPage) NavigateWithReferrer(url, referrer string) error {
	return p.NavigateWithOptions(url, &NavigateOptions{Referrer: referrer})
}

// navigateAndWait 导航并等待本次导航（按 loaderId 区分，不会被上一个文档的事件误触发）的指定生命周期事件
func navigateAndWait(ctx context.Context, url, referrer, event string) error {
	if err := page.SetLifecycleEventsEnabled(true).Do(ctx); err != nil {
		return fmt.Errorf("enable lifecycle events: %w", err)
	}

	lctx, cancel := context.WithCancel(ctx)
	defer cancel()
	// 监听必须先于导航注册；回调在 chromedp 事件循环里同步执行，只做非阻塞投递
	events := make(chan *page.EventLifecycleEvent, 256)
	chromedp.ListenTarget(lctx, func(ev interface{}) {
		if e, ok := ev.(*page.EventLifecycleEvent); ok {
			select {
			case events <- e:
			default:
			}
		}
	})

	_, loaderID, errorText, _, err := page.Navigate(url).WithReferrer(referrer).Do(ctx)
	if err != nil {
		return err
	}
	if errorText != "" {
		return fmt.Errorf("page load error %s", errorText)
	}
	if loaderID == "" {
		return nil // 同文档导航（如锚点）不产生新文档，也没有生命周期事件
	}
	for {
		select {
		case e := <-events:
			if e.LoaderID == loaderID && e.Name == event {
				return nil
			}
		case <-ctx.Done():
			return ctx.Err()
		}
	}
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

	// 获取元素坐标（先滚动到视口内，否则 getBoundingClientRect 返回的
	// 视口相对坐标会指向屏幕外，导致折叠线下的元素点空）
	var coordResult map[string]interface{}
	err := chromedp.Run(p.ctx, chromedp.Evaluate(fmt.Sprintf(`
		(function() {
			const elem = document.querySelector('%s');
			if (!elem) return null;
			elem.scrollIntoViewIfNeeded ? elem.scrollIntoViewIfNeeded() : elem.scrollIntoView({block: 'center'});
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

	x, _ := coordResult["x"].(float64)
	y, _ := coordResult["y"].(float64)

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
	return chromedp.Run(p.ctx, chromedp.ActionFunc(func(ctx context.Context) error {
		if err := cdp.Execute(ctx, "Emulation.setDeviceMetricsOverride", viewportOverride(p.identity, width, height), nil); err != nil {
			return fmt.Errorf("Emulation.setDeviceMetricsOverride: %w", err)
		}
		if p.identity == nil || p.identity.Screen.Width == 0 {
			return nil // 身份不调整几何（有界面且不设指纹）时不动用户可见的窗口
		}
		return setChromedpWindowBounds(ctx, viewportWindowBounds(width, height))
	}))
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
	// Fetch 分发器绑定在 p.ctx 上，随下面的 cancel 一起移除

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

// hasProxyAuth reports whether a proxy with credentials is configured
func (p *CDPPage) hasProxyAuth() bool {
	return p.opts != nil && p.opts.Proxy != nil && p.opts.Proxy.Username != ""
}

// setupProxyAuth sets up proxy authentication if configured.
// handleAuthRequests 只对被 Fetch 拦截的请求生效，所以要拦截全部请求；
// 暂停的请求由 listenFetch 注册的分发器放行（未开启用户拦截时）
func (p *CDPPage) setupProxyAuth() chromedp.Action {
	return chromedp.ActionFunc(func(ctx context.Context) error {
		if !p.hasProxyAuth() {
			return nil
		}
		p.listenFetch(ctx)
		if err := enableFetchAll(ctx, true); err != nil {
			return fmt.Errorf("failed to enable Fetch with auth: %w", err)
		}
		return nil
	})
}

// enableFetchAll 拦截全部请求；handleAuth 为 true 时同时接管认证质询
func enableFetchAll(ctx context.Context, handleAuth bool) error {
	return fetch.Enable().
		WithHandleAuthRequests(handleAuth).
		WithPatterns([]*fetch.RequestPattern{{URLPattern: "*"}}).
		Do(ctx)
}

// proxyAuthResponse 只对代理发起的质询提供代理凭据；目标站自己的 HTTP 认证交给浏览器默认处理，
// 否则任何返回 401 的网站都能拿到代理账号密码
func proxyAuthResponse(proxy *ProxyConfig, challenge *fetch.AuthChallenge) *fetch.AuthChallengeResponse {
	if proxy == nil || proxy.Username == "" || challenge == nil || challenge.Source != fetch.AuthChallengeSourceProxy {
		return &fetch.AuthChallengeResponse{Response: fetch.AuthChallengeResponseResponseDefault}
	}
	return &fetch.AuthChallengeResponse{
		Response: fetch.AuthChallengeResponseResponseProvideCredentials,
		Username: proxy.Username,
		Password: proxy.Password,
	}
}

// listenFetch 注册唯一的 Fetch 事件分发器。ctx 须带 chromedp 执行器且与页面同生命周期
func (p *CDPPage) listenFetch(ctx context.Context) {
	p.requestListenerMu.Lock()
	defer p.requestListenerMu.Unlock()
	if p.fetchListening {
		return
	}
	p.fetchListening = true

	chromedp.ListenTarget(ctx, func(ev interface{}) {
		// 监听回调在 chromedp 事件循环里同步执行，发命令必须另起 goroutine
		switch e := ev.(type) {
		case *fetch.EventRequestPaused:
			go p.handleRequestPaused(ctx, e)
		case *fetch.EventAuthRequired:
			go fetch.ContinueWithAuth(e.RequestID, proxyAuthResponse(p.opts.Proxy, e.AuthChallenge)).Do(ctx)
		}
	})
}

// handleRequestPaused 开启拦截且设置了处理器时交给处理器，否则直接放行（代理认证模式下拦截全部请求）
func (p *CDPPage) handleRequestPaused(ctx context.Context, e *fetch.EventRequestPaused) {
	p.requestListenerMu.Lock()
	handler := p.requestHandler
	intercept := p.interceptEnabled
	p.requestListenerMu.Unlock()

	if !intercept || handler == nil {
		fetch.ContinueRequest(e.RequestID).Do(ctx)
		return
	}

	req := &InterceptedRequest{
		URL:          e.Request.URL,
		Method:       e.Request.Method,
		Headers:      make(map[string]string),
		ResourceType: string(e.ResourceType),
		RequestID:    string(e.RequestID),
	}
	for name, value := range e.Request.Headers {
		if str, ok := value.(string); ok {
			req.Headers[name] = str
		}
	}
	req.setPageContext(p)

	// 处理器出错时放行，避免请求挂起
	if err := handler(req); err != nil {
		fetch.ContinueRequest(e.RequestID).Do(ctx)
	}
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

// SetRequestInterception enables or disables request interception
func (p *CDPPage) SetRequestInterception(enabled bool) error {
	p.requestListenerMu.Lock()
	p.interceptEnabled = enabled
	p.requestListenerMu.Unlock()

	return chromedp.Run(p.ctx, chromedp.ActionFunc(func(ctx context.Context) error {
		if !enabled {
			// 代理认证依赖 Fetch 拦截：保持开启，分发器会直接放行
			if p.hasProxyAuth() {
				return nil
			}
			return fetch.Disable().Do(ctx)
		}

		if err := network.Enable().Do(ctx); err != nil {
			return fmt.Errorf("failed to enable Network domain: %w", err)
		}
		p.listenFetch(ctx)
		// 重新 enable 会覆盖之前的配置，必须保留代理认证
		if err := enableFetchAll(ctx, p.hasProxyAuth()); err != nil {
			return fmt.Errorf("failed to enable Fetch domain: %w", err)
		}
		return nil
	}))
}

// OnRequest sets the request handler for intercepted requests
func (p *CDPPage) OnRequest(handler RequestHandler) error {
	p.requestListenerMu.Lock()
	p.requestHandler = handler
	p.requestListenerMu.Unlock()
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

// ContinueRequestWithBody continues an intercepted request with a modified URL
// and/or POST data (parity with CustomCDPPage). Requires request interception to
// be enabled (SetRequestInterception + OnRequest); call this from the handler.
//
// headers, when non-nil, REPLACES the entire request header set — pass nil to keep
// the original headers. Chrome recomputes Content-Length for an overridden body.
func (p *CDPPage) ContinueRequestWithBody(requestID, url, postData string, headers map[string]string) error {
	return chromedp.Run(p.ctx, chromedp.ActionFunc(func(ctx context.Context) error {
		cmd := fetch.ContinueRequest(fetch.RequestID(requestID))
		if url != "" {
			cmd = cmd.WithURL(url)
		}
		if postData != "" {
			cmd = cmd.WithPostData(base64.StdEncoding.EncodeToString([]byte(postData)))
		}
		if len(headers) > 0 {
			entries := make([]*fetch.HeaderEntry, 0, len(headers))
			for name, value := range headers {
				entries = append(entries, &fetch.HeaderEntry{Name: name, Value: value})
			}
			cmd = cmd.WithHeaders(entries)
		}
		return cmd.Do(ctx)
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

// setChromedpWindowBounds chromedp 路径：调整本页所在窗口的位置与外框
func setChromedpWindowBounds(ctx context.Context, bounds map[string]any) error {
	c := chromedp.FromContext(ctx)
	bctx := cdp.WithExecutor(ctx, c.Browser)
	browser := func(m string, params any) (json.RawMessage, error) {
		var raw json.RawMessage
		err := cdp.Execute(bctx, m, params, &raw)
		return raw, err
	}
	return setTargetWindowBounds(browser, string(c.Target.TargetID), bounds)
}
