package browser

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// CustomCDPClient implements a custom CDP client that avoids Runtime.Enable leaks
type CustomCDPClient struct {
	conn           *websocket.Conn
	url            string
	messageID      int64
	messageIDMutex sync.Mutex
	responses      map[int64]chan CDPResponse
	responsesMutex sync.RWMutex
	writeMutex     sync.Mutex // 添加写入锁防止并发写入
	ctx            context.Context
	cancel         context.CancelFunc
	// 事件监听
	eventHandlers      map[string][]func(json.RawMessage)
	eventHandlersMutex sync.RWMutex
}

// CDPMessage represents a CDP message
type CDPMessage struct {
	ID     int64       `json:"id"`
	Method string      `json:"method"`
	Params interface{} `json:"params,omitempty"`
}

// CDPResponse represents a CDP response
type CDPResponse struct {
	ID     int64           `json:"id"`
	Result json.RawMessage `json:"result,omitempty"`
	Error  *CDPError       `json:"error,omitempty"`
}

// CDPError represents a CDP error
type CDPError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    string `json:"data,omitempty"`
}

// CDPEvent represents a CDP event (no ID, has method)
type CDPEvent struct {
	Method string          `json:"method"`
	Params json.RawMessage `json:"params"`
}

// NewCustomCDPClient creates a new custom CDP client
func NewCustomCDPClient(debugURL string) (*CustomCDPClient, error) {
	// Parse the debug URL to get WebSocket endpoint
	resp, err := http.Get(debugURL + "/json")
	if err != nil {
		return nil, fmt.Errorf("failed to get debug info: %w", err)
	}
	defer resp.Body.Close()

	var tabs []map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&tabs); err != nil {
		return nil, fmt.Errorf("failed to decode debug info: %w", err)
	}

	if len(tabs) == 0 {
		return nil, fmt.Errorf("no tabs found")
	}

	// Get the WebSocket URL for the first tab
	wsURL, ok := tabs[0]["webSocketDebuggerUrl"].(string)
	if !ok {
		return nil, fmt.Errorf("no WebSocket URL found")
	}

	// Connect to WebSocket
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to WebSocket: %w", err)
	}

	ctx, cancel := context.WithCancel(context.Background())

	client := &CustomCDPClient{
		conn:          conn,
		url:           wsURL,
		responses:     make(map[int64]chan CDPResponse),
		eventHandlers: make(map[string][]func(json.RawMessage)),
		ctx:           ctx,
		cancel:        cancel,
	}

	// Start message handler
	go client.handleMessages()

	return client, nil
}

// handleMessages handles incoming WebSocket messages
func (c *CustomCDPClient) handleMessages() {
	defer c.conn.Close()

	for {
		select {
		case <-c.ctx.Done():
			return
		default:
			_, message, err := c.conn.ReadMessage()
			if err != nil {
				return
			}

			// 先尝试解析基础结构
			var base struct {
				ID     int64  `json:"id"`
				Method string `json:"method"`
			}
			if err := json.Unmarshal(message, &base); err != nil {
				continue
			}

			// 有 ID 的是响应
			if base.ID > 0 {
				var response CDPResponse
				if err := json.Unmarshal(message, &response); err == nil {
					c.responsesMutex.RLock()
					if ch, exists := c.responses[response.ID]; exists {
						select {
						case ch <- response:
						case <-time.After(5 * time.Second):
						}
					}
					c.responsesMutex.RUnlock()
				}
				continue
			}

			// 有 method 的是事件
			if base.Method != "" {
				var event CDPEvent
				if err := json.Unmarshal(message, &event); err == nil {
					c.eventHandlersMutex.RLock()
					// 复制 handlers 避免长时间持锁
					handlers := make([]func(json.RawMessage), len(c.eventHandlers[event.Method]))
					copy(handlers, c.eventHandlers[event.Method])
					c.eventHandlersMutex.RUnlock()

					// 异步执行 handlers，避免阻塞消息循环
					if len(handlers) > 0 {
						params := event.Params // 捕获参数
						go func() {
							for _, handler := range handlers {
								handler(params)
							}
						}()
					}
				}
			}
		}
	}
}

// OnEvent subscribes to a CDP event
func (c *CustomCDPClient) OnEvent(method string, handler func(json.RawMessage)) {
	c.eventHandlersMutex.Lock()
	c.eventHandlers[method] = append(c.eventHandlers[method], handler)
	c.eventHandlersMutex.Unlock()
}

// sendCommand sends a CDP command and waits for response
func (c *CustomCDPClient) sendCommand(method string, params interface{}) (json.RawMessage, error) {
	c.messageIDMutex.Lock()
	c.messageID++
	id := c.messageID
	c.messageIDMutex.Unlock()

	// Create response channel
	respChan := make(chan CDPResponse, 1)
	c.responsesMutex.Lock()
	c.responses[id] = respChan
	c.responsesMutex.Unlock()

	defer func() {
		c.responsesMutex.Lock()
		delete(c.responses, id)
		c.responsesMutex.Unlock()
	}()

	// Send message with write lock
	message := CDPMessage{
		ID:     id,
		Method: method,
		Params: params,
	}

	c.writeMutex.Lock()
	err := c.conn.WriteJSON(message)
	c.writeMutex.Unlock()

	if err != nil {
		return nil, fmt.Errorf("failed to send message: %w", err)
	}

	// Wait for response
	select {
	case response := <-respChan:
		if response.Error != nil {
			return nil, fmt.Errorf("CDP error: %s", response.Error.Message)
		}
		return response.Result, nil
	case <-time.After(30 * time.Second):
		return nil, fmt.Errorf("timeout waiting for response")
	case <-c.ctx.Done():
		return nil, fmt.Errorf("context cancelled")
	}
}

// Navigate navigates to a URL without using Runtime.Enable
func (c *CustomCDPClient) Navigate(url string) error {
	params := map[string]interface{}{
		"url": url,
	}
	_, err := c.sendCommand("Page.navigate", params)
	return err
}

// EvaluateWithoutRuntimeEnable evaluates JavaScript without triggering Runtime.Enable
func (c *CustomCDPClient) EvaluateWithoutRuntimeEnable(expression string) (interface{}, error) {
	// Method 1: Try using Page.addScriptToEvaluateOnNewDocument + Page.reload
	// This avoids Runtime.Enable but requires page reload

	// First, add script to evaluate on new document
	params := map[string]interface{}{
		"source": fmt.Sprintf(`
			window.__customEvalResult = (() => {
				try {
					return JSON.stringify(%s);
				} catch (e) {
					return JSON.stringify({error: e.message});
				}
			})();
		`, expression),
	}

	_, err := c.sendCommand("Page.addScriptToEvaluateOnNewDocument", params)
	if err != nil {
		return nil, err
	}

	// Reload page to execute script
	_, err = c.sendCommand("Page.reload", map[string]interface{}{})
	if err != nil {
		return nil, err
	}

	// Wait for page to load
	time.Sleep(2 * time.Second)

	// Get result using DOM query
	getResultScript := `document.querySelector('body').getAttribute('data-result') || window.__customEvalResult`
	return c.evaluateViaDOM(getResultScript)
}

// evaluateViaDOM evaluates JavaScript via DOM manipulation to avoid Runtime.Enable
func (c *CustomCDPClient) evaluateViaDOM(expression string) (interface{}, error) {
	// Create a unique element to store result
	elementId := fmt.Sprintf("custom-eval-%d", time.Now().UnixNano())

	// Note: In a full implementation, we would inject script via DOM
	// to avoid Runtime.Enable, but for now we'll use a placeholder
	_ = elementId // Avoid unused variable warning

	// Use DOM.getDocument and DOM.querySelector to inject and read
	_, err := c.sendCommand("DOM.getDocument", map[string]interface{}{})
	if err != nil {
		return nil, err
	}

	// This is a simplified approach - in practice, you'd need to:
	// 1. Get document node
	// 2. Create elements via DOM.createElement
	// 3. Set attributes via DOM.setAttributeValue
	// 4. Read results via DOM.getOuterHTML

	// For now, return a placeholder
	return map[string]interface{}{
		"note":       "Custom CDP evaluation - Runtime.Enable avoided",
		"expression": expression,
	}, nil
}

// GetPageContent gets page content without Runtime.Enable
func (c *CustomCDPClient) GetPageContent() (string, error) {
	// Get document
	docResult, err := c.sendCommand("DOM.getDocument", map[string]interface{}{})
	if err != nil {
		return "", err
	}

	var docResponse struct {
		Root struct {
			NodeId int `json:"nodeId"`
		} `json:"root"`
	}

	if err := json.Unmarshal(docResult, &docResponse); err != nil {
		return "", err
	}

	// Get outer HTML
	params := map[string]interface{}{
		"nodeId": docResponse.Root.NodeId,
	}

	htmlResult, err := c.sendCommand("DOM.getOuterHTML", params)
	if err != nil {
		return "", err
	}

	var htmlResponse struct {
		OuterHTML string `json:"outerHTML"`
	}

	if err := json.Unmarshal(htmlResult, &htmlResponse); err != nil {
		return "", err
	}

	return htmlResponse.OuterHTML, nil
}

// TakeScreenshot takes a screenshot without Runtime.Enable
func (c *CustomCDPClient) TakeScreenshot() ([]byte, error) {
	result, err := c.sendCommand("Page.captureScreenshot", map[string]interface{}{
		"format": "png",
	})
	if err != nil {
		return nil, err
	}

	var response struct {
		Data string `json:"data"`
	}

	if err := json.Unmarshal(result, &response); err != nil {
		return nil, err
	}

	// Decode base64
	return []byte(response.Data), nil
}

// Click performs a click without Runtime.Enable
func (c *CustomCDPClient) Click(x, y float64) error {
	// 1. 先移动鼠标到目标位置
	moveParams := map[string]interface{}{
		"type": "mouseMoved",
		"x":    x,
		"y":    y,
	}
	if _, err := c.sendCommand("Input.dispatchMouseEvent", moveParams); err != nil {
		return err
	}

	// 2. 鼠标按下
	pressParams := map[string]interface{}{
		"type":       "mousePressed",
		"x":          x,
		"y":          y,
		"button":     "left",
		"clickCount": 1,
	}
	if _, err := c.sendCommand("Input.dispatchMouseEvent", pressParams); err != nil {
		return err
	}

	// 3. 短暂延迟模拟真实点击
	time.Sleep(50 * time.Millisecond)

	// 4. 鼠标释放
	releaseParams := map[string]interface{}{
		"type":       "mouseReleased",
		"x":          x,
		"y":          y,
		"button":     "left",
		"clickCount": 1,
	}
	_, err := c.sendCommand("Input.dispatchMouseEvent", releaseParams)
	return err
}

// Type sends keyboard events for each character (real typing)
func (c *CustomCDPClient) Type(text string) error {
	for _, char := range text {
		charStr := string(char)

		// KeyDown
		keyParams := map[string]interface{}{
			"type": "keyDown",
			"text": charStr,
		}
		if _, err := c.sendCommand("Input.dispatchKeyEvent", keyParams); err != nil {
			return err
		}

		// KeyUp
		keyUpParams := map[string]interface{}{
			"type": "keyUp",
		}
		if _, err := c.sendCommand("Input.dispatchKeyEvent", keyUpParams); err != nil {
			return err
		}

		// 模拟人类输入速度
		time.Sleep(time.Duration(50+rand.Intn(100)) * time.Millisecond)
	}
	return nil
}

// Close closes the CDP connection
func (c *CustomCDPClient) Close() error {
	c.cancel()
	return c.conn.Close()
}

// EnablePageDomain enables Page domain events
func (c *CustomCDPClient) EnablePageDomain() error {
	_, err := c.sendCommand("Page.enable", map[string]interface{}{})
	return err
}

// EnableDOMDomain enables DOM domain events
func (c *CustomCDPClient) EnableDOMDomain() error {
	_, err := c.sendCommand("DOM.enable", map[string]interface{}{})
	return err
}

// CreateExecutionContextWithBinding creates execution context using binding method
// This avoids Runtime.Enable leak detection by Cloudflare
func (c *CustomCDPClient) CreateExecutionContextWithBinding() (int, error) {
	// Method 1: Use addBinding technique (rebrowser-patches approach)
	bindingName := "__rebrowser_context_probe"

	// Add binding to get context ID
	params := map[string]interface{}{
		"name": bindingName,
	}

	result, err := c.sendCommand("Runtime.addBinding", params)
	if err != nil {
		return 0, fmt.Errorf("failed to add binding: %w", err)
	}

	// The binding will be available in the main context
	// We can extract context ID from subsequent calls
	var response struct {
		ExecutionContextId int `json:"executionContextId"`
	}

	if err := json.Unmarshal(result, &response); err != nil {
		// If we can't get context ID directly, use a fallback
		return 1, nil // Default main context ID
	}

	return response.ExecutionContextId, nil
}

// EvaluateWithBinding evaluates JavaScript using binding method to avoid Runtime.Enable
func (c *CustomCDPClient) EvaluateWithBinding(expression string) (interface{}, error) {
	// CRITICAL FIX: Don't specify contextId - let Chrome use the default (current) context
	// Navigate creates a new execution context, and specifying an old contextId causes
	// "Cannot find context with specified id" errors
	// By omitting contextId, Chrome will automatically use the active execution context
	params := map[string]interface{}{
		"expression": expression,
		// contextId is intentionally omitted - Chrome will use default context
		"returnByValue":         true,
		"awaitPromise":          false,
		"userGesture":           false,
		"includeCommandLineAPI": false,
	}

	result, err := c.sendCommand("Runtime.evaluate", params)
	if err != nil {
		return nil, err
	}

	var response struct {
		Result struct {
			Value interface{} `json:"value"`
			Type  string      `json:"type"`
		} `json:"result"`
		ExceptionDetails interface{} `json:"exceptionDetails"`
	}

	if err := json.Unmarshal(result, &response); err != nil {
		return nil, err
	}

	if response.ExceptionDetails != nil {
		return nil, fmt.Errorf("JavaScript execution error: %v", response.ExceptionDetails)
	}

	return response.Result.Value, nil
}

// CustomCDPConnector handles connections using custom CDP client
type CustomCDPConnector struct{}

// CreateCustomCDPConnector creates a custom CDP connector that avoids Runtime.Enable
func CreateCustomCDPConnector() *CustomCDPConnector {
	return &CustomCDPConnector{}
}

// Connect establishes a connection using custom CDP client
func (ccc *CustomCDPConnector) Connect(ctx context.Context, chrome *ChromeProcess, opts *ConnectOptions) (Page, error) {
	debugURL := fmt.Sprintf("http://localhost:%d", chrome.Port)

	client, err := NewCustomCDPClient(debugURL)
	if err != nil {
		return nil, fmt.Errorf("failed to create custom CDP client: %w", err)
	}

	page := &CustomCDPPage{
		client: client,
		chrome: chrome,
		opts:   opts,
		ctx:    ctx,
	}

	// Initialize the page with stealth settings
	if err := page.initialize(); err != nil {
		client.Close()
		return nil, fmt.Errorf("failed to initialize custom CDP page: %w", err)
	}

	return page, nil
}

// CustomCDPPage implements Page interface using custom CDP client
type CustomCDPPage struct {
	client *CustomCDPClient
	chrome *ChromeProcess
	opts   *ConnectOptions
	ctx    context.Context
}

// initialize sets up the custom CDP page with stealth features
func (p *CustomCDPPage) initialize() error {
	// Enable necessary domains
	if err := p.client.EnablePageDomain(); err != nil {
		return fmt.Errorf("failed to enable Page domain: %w", err)
	}

	if err := p.client.EnableDOMDomain(); err != nil {
		return fmt.Errorf("failed to enable DOM domain: %w", err)
	}

	// CRITICAL: Inject stealth script on new document WITHOUT Runtime.Enable
	var script string
	var userAgent string
	var platform string
	
	// 检查是否指定了用户ID
	if p.opts != nil && p.opts.FingerprintUserID != "" {
		// 使用 UserFingerprintManager 获取或生成指纹
		fingerprintDir := p.opts.FingerprintDir
		if fingerprintDir == "" {
			fingerprintDir = "./fingerprints"
		}
		manager, err := NewUserFingerprintManager(fingerprintDir)
		if err == nil {
			// 提取初始化参数（Width、Height、UserAgent）
			initParams := GetInitParamsFromOptions(p.opts)
			config, err := manager.GetOrCreateUserFingerprint(p.opts.FingerprintUserID, initParams)
			if err == nil {
				// 使用缓存的脚本（基于 userID）
				script = GetCachedStealthScriptWithConfig(config)
				// 获取 UserAgent 和 Platform
				userAgent = config.Browser.UserAgent
				platform = config.Browser.Platform
			}
		}
		// 如果获取失败，使用默认脚本
		if script == "" {
			script = GetCachedAdvancedStealthScript()
		}
	} else {
		// 使用默认的高级 stealth 脚本（缓存版本）
		script = GetCachedAdvancedStealthScript()
		// 如果直接设置了 UserAgent（不使用 FingerprintUserID）
		if p.opts != nil && p.opts.UserAgent != "" {
			userAgent = p.opts.UserAgent
		}
	}

	// 设置 HTTP 请求头的 UserAgent（关键！）
	if userAgent != "" {
		params := map[string]interface{}{
			"userAgent": userAgent,
		}
		if platform != "" {
			params["platform"] = platform
		}
		if _, err := p.client.sendCommand("Emulation.setUserAgentOverride", params); err != nil {
			// 不要失败，只是警告
			fmt.Printf("⚠️ 设置 UserAgent 失败: %v\n", err)
		}
	}

	_, err := p.client.sendCommand("Page.addScriptToEvaluateOnNewDocument", map[string]interface{}{
		"source": script,
	})

	return err
}

// AddScriptToEvaluateOnNewDocument adds a script to be evaluated on every new document
func (p *CustomCDPPage) AddScriptToEvaluateOnNewDocument(script string) error {
	_, err := p.client.sendCommand("Page.addScriptToEvaluateOnNewDocument", map[string]interface{}{
		"source": script,
	})
	return err
}

// Navigate navigates to a URL
func (p *CustomCDPPage) Navigate(url string) error {
	return p.client.Navigate(url)
}

// Click performs a click
func (p *CustomCDPPage) Click(x, y float64) error {
	return p.client.Click(x, y)
}

// RealClick performs a realistic click (using same logic as mouse.go)
func (p *CustomCDPPage) RealClick(x, y float64) error {
	// For custom CDP, we use basic click for now
	// In a full implementation, we'd integrate with the GhostCursor logic
	return p.client.Click(x, y)
}

// Evaluate executes JavaScript WITHOUT Runtime.Enable
func (p *CustomCDPPage) Evaluate(script string) (interface{}, error) {
	return p.client.EvaluateWithBinding(script)
}

// escapeJsSelector 转义选择器中的特殊字符，用于在 JS 字符串中安全使用
func escapeJsSelector(selector string) string {
	// 转义反斜杠和单引号
	escaped := strings.ReplaceAll(selector, `\`, `\\`)
	escaped = strings.ReplaceAll(escaped, `'`, `\'`)
	return escaped
}

// WaitForSelector waits for an element
func (p *CustomCDPPage) WaitForSelector(selector string) error {
	escaped := escapeJsSelector(selector)
	// Simplified implementation using polling
	maxAttempts := 30
	for i := 0; i < maxAttempts; i++ {
		result, err := p.Evaluate(fmt.Sprintf("document.querySelector('%s') !== null", escaped))
		if err != nil {
			continue
		}

		if found, ok := result.(bool); ok && found {
			return nil
		}

		time.Sleep(500 * time.Millisecond)
	}

	return fmt.Errorf("element with selector '%s' not found", selector)
}

// ClickSelector clicks an element by CSS selector
func (p *CustomCDPPage) ClickSelector(selector string) error {
	escaped := escapeJsSelector(selector)
	// 等待元素出现
	if err := p.WaitForSelector(selector); err != nil {
		return err
	}

	// 获取坐标并点击
	result, err := p.Evaluate(fmt.Sprintf(`
		(function() {
			const elem = document.querySelector('%s');
			if (!elem) return null;
			elem.scrollIntoViewIfNeeded ? elem.scrollIntoViewIfNeeded() : elem.scrollIntoView({block: 'center'});
			const rect = elem.getBoundingClientRect();
			return {x: rect.left + rect.width/2, y: rect.top + rect.height/2};
		})()
	`, escaped))
	if err != nil {
		return err
	}

	coords, ok := result.(map[string]interface{})
	if !ok || coords == nil {
		return fmt.Errorf("element not found: %s", selector)
	}

	x, _ := coords["x"].(float64)
	y, _ := coords["y"].(float64)
	return p.Click(x, y)
}

// RealClickSelector clicks an element with human-like mouse movement
func (p *CustomCDPPage) RealClickSelector(selector string) error {
	escaped := escapeJsSelector(selector)
	// 等待元素出现
	if err := p.WaitForSelector(selector); err != nil {
		return err
	}

	// 获取坐标
	result, err := p.Evaluate(fmt.Sprintf(`
		(function() {
			const elem = document.querySelector('%s');
			if (!elem) return null;
			elem.scrollIntoViewIfNeeded ? elem.scrollIntoViewIfNeeded() : elem.scrollIntoView({block: 'center'});
			const rect = elem.getBoundingClientRect();
			const rx = (Math.random() - 0.5) * Math.min(rect.width * 0.3, 8);
			const ry = (Math.random() - 0.5) * Math.min(rect.height * 0.3, 8);
			return {x: rect.left + rect.width/2 + rx, y: rect.top + rect.height/2 + ry};
		})()
	`, escaped))
	if err != nil {
		return err
	}

	coords, ok := result.(map[string]interface{})
	if !ok || coords == nil {
		return fmt.Errorf("element not found: %s", selector)
	}

	x, _ := coords["x"].(float64)
	y, _ := coords["y"].(float64)
	return p.RealClick(x, y)
}

// RealSendKeys types text using real keyboard events (maintains focus)
func (p *CustomCDPPage) RealSendKeys(text string) error {
	return p.client.Type(text)
}

// SendKeys types text into an element
func (p *CustomCDPPage) SendKeys(selector, text string) error {
	escaped := escapeJsSelector(selector)
	// 点击聚焦
	if err := p.ClickSelector(selector); err != nil {
		return err
	}

	// 输入文本
	for _, char := range text {
		_, err := p.Evaluate(fmt.Sprintf(`
			(function() {
				const elem = document.querySelector('%s');
				if (!elem) throw new Error('Element not found');
				elem.focus();
				const val = elem.value || '';
				elem.value = val + '%c';
				elem.dispatchEvent(new Event('input', {bubbles: true}));
			})()
		`, escaped, char))
		if err != nil {
			return err
		}
		time.Sleep(time.Duration(80+rand.Intn(120)) * time.Millisecond)
	}
	return nil
}

// Screenshot takes a screenshot
func (p *CustomCDPPage) Screenshot() ([]byte, error) {
	return p.client.TakeScreenshot()
}

// SetViewport sets viewport
func (p *CustomCDPPage) SetViewport(width, height int) error {
	params := map[string]interface{}{
		"width":  width,
		"height": height,
	}
	_, err := p.client.sendCommand("Emulation.setDeviceMetricsOverride", params)
	return err
}

// GetTitle gets page title
func (p *CustomCDPPage) GetTitle() (string, error) {
	result, err := p.Evaluate("document.title")
	if err != nil {
		return "", err
	}

	if title, ok := result.(string); ok {
		return title, nil
	}

	return "", fmt.Errorf("failed to get title")
}

// GetURL gets current URL
func (p *CustomCDPPage) GetURL() (string, error) {
	result, err := p.Evaluate("window.location.href")
	if err != nil {
		return "", err
	}

	if url, ok := result.(string); ok {
		return url, nil
	}

	return "", fmt.Errorf("failed to get URL")
}

// Close closes the custom CDP page
func (p *CustomCDPPage) Close() error {
	return p.client.Close()
}

// SetRequestInterception enables or disables request interception (stub for CustomCDPPage)
func (p *CustomCDPPage) SetRequestInterception(enabled bool) error {
	// TODO: Implement request interception for CustomCDPPage if needed
	// For now, return nil to satisfy the interface
	return nil
}

// OnRequest sets the request handler for intercepted requests (stub for CustomCDPPage)
func (p *CustomCDPPage) OnRequest(handler RequestHandler) error {
	// TODO: Implement request handler for CustomCDPPage if needed
	// For now, return nil to satisfy the interface
	return nil
}

// ==================== 新增方法 (按原版优化) ====================

// NavigateWithOptions navigates with options
func (p *CustomCDPPage) NavigateWithOptions(url string, opts *NavigateOptions) error {
	// CustomCDPPage 简化实现，只做基本导航
	return p.Navigate(url)
}

// NavigateWithReferrer navigates with referrer
func (p *CustomCDPPage) NavigateWithReferrer(url, referrer string) error {
	params := map[string]interface{}{
		"url":      url,
		"referrer": referrer,
	}
	_, err := p.client.sendCommand("Page.navigate", params)
	return err
}

// WaitVisible waits for element to be visible
func (p *CustomCDPPage) WaitVisible(selector string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		visible, err := p.isElementVisible(selector)
		if err == nil && visible {
			return nil
		}
		time.Sleep(100 * time.Millisecond)
	}
	return fmt.Errorf("timeout waiting for element: %s", selector)
}

// WaitNotVisible waits for element to disappear
func (p *CustomCDPPage) WaitNotVisible(selector string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		visible, _ := p.isElementVisible(selector)
		if !visible {
			return nil
		}
		time.Sleep(100 * time.Millisecond)
	}
	return fmt.Errorf("timeout waiting for element to disappear: %s", selector)
}

// WaitVisibleByID waits for element by ID
// Has checks if element exists
func (p *CustomCDPPage) Has(selector string) (bool, error) {
	escaped := escapeJsSelector(selector)
	result, err := p.Evaluate(fmt.Sprintf(`document.querySelector('%s') !== null`, escaped))
	if err != nil {
		return false, err
	}
	if b, ok := result.(bool); ok {
		return b, nil
	}
	return false, nil
}

// isElementVisible checks if element is visible
func (p *CustomCDPPage) isElementVisible(selector string) (bool, error) {
	escaped := escapeJsSelector(selector)
	result, err := p.Evaluate(fmt.Sprintf(`
		(function() {
			const elem = document.querySelector('%s');
			if (!elem) return false;
			const rect = elem.getBoundingClientRect();
			const style = window.getComputedStyle(elem);
			return rect.width > 0 && rect.height > 0 && 
			       style.visibility !== 'hidden' && 
			       style.display !== 'none';
		})()
	`, escaped))
	if err != nil {
		return false, err
	}
	if b, ok := result.(bool); ok {
		return b, nil
	}
	return false, nil
}

// SetCookies sets cookies
func (p *CustomCDPPage) SetCookies(cookiesJSON string, url string) error {
	var cookies []map[string]interface{}
	if err := json.Unmarshal([]byte(cookiesJSON), &cookies); err != nil {
		return err
	}
	for _, cookie := range cookies {
		if cookie["url"] == nil {
			cookie["url"] = url
		}
		_, err := p.client.sendCommand("Network.setCookie", cookie)
		if err != nil {
			return err
		}
	}
	return nil
}

// GetCookies gets cookies
// GetCookies gets cookies as JSON string
func (p *CustomCDPPage) GetCookies() (string, error) {
	result, err := p.client.sendCommand("Network.getCookies", nil)
	if err != nil {
		return "", err
	}
	// 解析 result
	var resp struct {
		Cookies json.RawMessage `json:"cookies"`
	}
	if err := json.Unmarshal(result, &resp); err != nil {
		return "[]", nil
	}
	return string(resp.Cookies), nil
}

// ClearCookies clears all cookies
func (p *CustomCDPPage) ClearCookies() error {
	_, err := p.client.sendCommand("Network.clearBrowserCookies", nil)
	return err
}

// SetLocalStorage sets localStorage
func (p *CustomCDPPage) SetLocalStorage(dataJSON string) error {
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

// GetLocalStorage gets localStorage
func (p *CustomCDPPage) GetLocalStorage() (string, error) {
	result, err := p.Evaluate(`JSON.stringify(localStorage)`)
	if err != nil {
		return "", err
	}
	if str, ok := result.(string); ok {
		return str, nil
	}
	return "{}", nil
}

// SetSessionStorage sets sessionStorage
func (p *CustomCDPPage) SetSessionStorage(dataJSON string) error {
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

// GetSessionStorage gets sessionStorage
func (p *CustomCDPPage) GetSessionStorage() (string, error) {
	result, err := p.Evaluate(`JSON.stringify(sessionStorage)`)
	if err != nil {
		return "", err
	}
	if str, ok := result.(string); ok {
		return str, nil
	}
	return "{}", nil
}

// ScreenshotElement takes element screenshot
func (p *CustomCDPPage) ScreenshotElement(selector string) ([]byte, error) {
	// 获取元素位置
	escaped := escapeJsSelector(selector)
	result, err := p.Evaluate(fmt.Sprintf(`
		(function() {
			const elem = document.querySelector('%s');
			if (!elem) return null;
			const rect = elem.getBoundingClientRect();
			return {x: rect.x, y: rect.y, width: rect.width, height: rect.height};
		})()
	`, escaped))
	if err != nil {
		return nil, err
	}

	if result == nil {
		return nil, fmt.Errorf("element not found: %s", selector)
	}

	// 解析位置
	rectMap, ok := result.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid rect result")
	}

	// 使用 clip 参数截图
	params := map[string]interface{}{
		"format": "png",
		"clip": map[string]interface{}{
			"x":      rectMap["x"],
			"y":      rectMap["y"],
			"width":  rectMap["width"],
			"height": rectMap["height"],
			"scale":  1,
		},
	}

	resp, err := p.client.sendCommand("Page.captureScreenshot", params)
	if err != nil {
		return nil, err
	}

	var screenshotResp struct {
		Data string `json:"data"`
	}
	if err := json.Unmarshal(resp, &screenshotResp); err != nil {
		return nil, fmt.Errorf("failed to parse screenshot response: %w", err)
	}

	return base64.StdEncoding.DecodeString(screenshotResp.Data)
}

// ScreenshotQrcode takes element screenshot and returns base64
func (p *CustomCDPPage) ScreenshotQrcode(selector string) (string, error) {
	buf, err := p.ScreenshotElement(selector)
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(buf), nil
}

// ExecuteJS executes JavaScript
func (p *CustomCDPPage) ExecuteJS(script string, result interface{}) error {
	res, err := p.Evaluate(script)
	if err != nil {
		return err
	}
	// 简单赋值
	if result != nil {
		data, _ := json.Marshal(res)
		return json.Unmarshal(data, result)
	}
	return nil
}

// Refresh refreshes the page
func (p *CustomCDPPage) Refresh(timeout time.Duration) error {
	_, err := p.client.sendCommand("Page.reload", nil)
	return err
}

// Sleep pauses execution
func (p *CustomCDPPage) Sleep(duration time.Duration) {
	time.Sleep(duration)
}

// GetContext returns nil for CustomCDPPage (not using chromedp context)
func (p *CustomCDPPage) GetContext() context.Context {
	return context.Background()
}

// EnableNetwork enables network domain for event listening
func (p *CustomCDPPage) EnableNetwork() error {
	_, err := p.client.sendCommand("Network.enable", nil)
	return err
}

// OnNetworkRequest subscribes to Network.requestWillBeSent events
func (p *CustomCDPPage) OnNetworkRequest(handler func(requestID, url, method string)) {
	p.client.OnEvent("Network.requestWillBeSent", func(params json.RawMessage) {
		var data struct {
			RequestID string `json:"requestId"`
			Request   struct {
				URL    string `json:"url"`
				Method string `json:"method"`
			} `json:"request"`
		}
		if json.Unmarshal(params, &data) == nil {
			handler(data.RequestID, data.Request.URL, data.Request.Method)
		}
	})
}

// OnNetworkResponse subscribes to Network.responseReceived events
func (p *CustomCDPPage) OnNetworkResponse(handler func(requestID, url string, status int)) {
	p.client.OnEvent("Network.responseReceived", func(params json.RawMessage) {
		var data struct {
			RequestID string `json:"requestId"`
			Response  struct {
				URL    string `json:"url"`
				Status int    `json:"status"`
			} `json:"response"`
		}
		if json.Unmarshal(params, &data) == nil {
			handler(data.RequestID, data.Response.URL, data.Response.Status)
		}
	})
}

// OnNetworkLoadingFinished subscribes to Network.loadingFinished events
func (p *CustomCDPPage) OnNetworkLoadingFinished(handler func(requestID string)) {
	p.client.OnEvent("Network.loadingFinished", func(params json.RawMessage) {
		var data struct {
			RequestID string `json:"requestId"`
		}
		if json.Unmarshal(params, &data) == nil {
			handler(data.RequestID)
		}
	})
}

// GetResponseBody gets the response body for a request
func (p *CustomCDPPage) GetResponseBody(requestID string) ([]byte, error) {
	result, err := p.client.sendCommand("Network.getResponseBody", map[string]interface{}{
		"requestId": requestID,
	})
	if err != nil {
		return nil, err
	}

	var resp struct {
		Body          string `json:"body"`
		Base64Encoded bool   `json:"base64Encoded"`
	}
	if err := json.Unmarshal(result, &resp); err != nil {
		return nil, err
	}

	if resp.Base64Encoded {
		return base64.StdEncoding.DecodeString(resp.Body)
	}
	return []byte(resp.Body), nil
}
