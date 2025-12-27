package browser

import (
	"context"
	"fmt"
	"os/exec"
	"sync/atomic"
	"time"
)

// Browser represents a browser instance interface
type Browser interface {
	Connect(ctx context.Context, opts *ConnectOptions) (*BrowserInstance, error)
	Close() error
}

// ConnectOptions contains configuration for browser connection
type ConnectOptions struct {
	// Headless mode setting (false, true, "new", "shell")
	Headless interface{} `json:"headless"`

	// Additional Chrome flags
	Args []string `json:"args"`

	// Custom Chrome launcher configuration
	CustomConfig map[string]interface{} `json:"customConfig"`

	// Proxy configuration
	Proxy *ProxyConfig `json:"proxy"`

	// Enable Turnstile auto-solving
	Turnstile bool `json:"turnstile"`

	// Puppeteer connect options
	ConnectOption map[string]interface{} `json:"connectOption"`

	// Disable Xvfb on Linux
	DisableXvfb bool `json:"disableXvfb"`

	// Ignore all default flags
	IgnoreAllFlags bool `json:"ignoreAllFlags"`

	// Plugin configurations (for future extensibility)
	Plugins []interface{} `json:"plugins"`

	// Use custom CDP client to avoid Runtime.Enable leaks (experimental)
	UseCustomCDP bool `json:"useCustomCDP"`

	// Chrome extensions support
	Extensions []string `json:"extensions"` // Paths to extension directories

	// Auto-load default extensions
	AutoLoadDefaultExtensions bool `json:"autoLoadDefaultExtensions"` // Automatically load default extensions

	// Profile/Account management
	ProfileName    string `json:"profileName"`    // Unique profile name for this account
	PersistProfile bool   `json:"persistProfile"` // Whether to persist user data or .crx files
}

// ProxyConfig contains proxy server configuration
type ProxyConfig struct {
	Host     string `json:"host"`
	Port     string `json:"port"`
	Username string `json:"username"`
	Password string `json:"password"`
}

// BrowserInstance represents a connected browser instance
type BrowserInstance struct {
	browser      Browser
	page         Page
	chrome       *ChromeProcess
	ctx          context.Context
	cancel       context.CancelFunc
	lastUsed     time.Time    // 最后使用时间（用于健康检查优化）
	healthStatus atomic.Value // 缓存的健康状态
}

// BrowserContext represents a browser context (like puppeteer browserContext)
type BrowserContext struct {
	allocCtx    context.Context
	allocCancel context.CancelFunc
	chrome      *ChromeProcess
	opts        *ConnectOptions
}

// ChromeProcess represents a Chrome process
type ChromeProcess struct {
	Cmd   *exec.Cmd
	Port  int
	PID   int
	Flags []string
}

// Page represents a browser page interface
type Page interface {
	Navigate(url string) error
	Click(x, y float64) error
	RealClick(x, y float64) error
	Evaluate(script string) (interface{}, error)
	WaitForSelector(selector string) error
	Screenshot() ([]byte, error)
	Close() error
	SetViewport(width, height int) error
	GetTitle() (string, error)
	GetURL() (string, error)
	SetRequestInterception(enabled bool) error
	OnRequest(handler RequestHandler) error
}

// PageWithSelector extends Page with selector-based methods
// Use type assertion to access these methods: page.(*CDPPage).ClickSelector(...)
type PageWithSelector interface {
	Page

	// 选择器操作
	ClickSelector(selector string) error     // 原生 chromedp 点击
	RealClickSelector(selector string) error // 拟人化贝塞尔曲线点击
	SendKeys(selector, text string) error    // 原生 chromedp 输入
	RealSendKeys(text string) error          // 真实键盘输入（需先获取焦点）

	// 等待方法
	WaitVisible(selector string, timeout time.Duration) error    // 等待元素可见
	WaitNotVisible(selector string, timeout time.Duration) error // 等待元素消失
	Has(selector string) (bool, error)                           // 元素存在检查

	// Cookie/Storage 管理
	SetCookies(cookiesJSON string, url string) error // 设置 cookies
	GetCookies() (string, error)                     // 获取 cookies (JSON 格式)
	ClearCookies() error                             // 清除 cookies
	SetLocalStorage(dataJSON string) error           // 设置 localStorage
	GetLocalStorage() (string, error)                // 获取 localStorage
	SetSessionStorage(dataJSON string) error         // 设置 sessionStorage
	GetSessionStorage() (string, error)              // 获取 sessionStorage

	// 截图
	ScreenshotElement(selector string) ([]byte, error) // 元素截图
	ScreenshotQrcode(selector string) (string, error)  // 二维码截图(base64)

	// 导航
	NavigateWithOptions(url string, opts *NavigateOptions) error // 带选项导航
	NavigateWithReferrer(url, referrer string) error             // 带 Referrer 导航
	Refresh(timeout time.Duration) error                         // 刷新页面

	// 便捷方法
	ExecuteJS(script string, result interface{}) error // 执行 JS
	Sleep(duration time.Duration)                      // 等待
	GetContext() context.Context                       // 获取 chromedp 上下文

}

// RequestHandler is a function type for handling intercepted requests
type RequestHandler func(req *InterceptedRequest) error

// InterceptedRequest represents an intercepted network request
type InterceptedRequest struct {
	URL          string
	Method       string
	Headers      map[string]string
	ResourceType string
	RequestID    string
	page         Page
}

// setPageContext sets the page context for request operations (internal method)
func (req *InterceptedRequest) setPageContext(page Page) {
	req.page = page
}

// InterceptedRequest methods for responding to requests
func (req *InterceptedRequest) Continue() error {
	if cdpPage, ok := req.page.(*CDPPage); ok {
		return cdpPage.continueRequest(req.RequestID)
	}
	return fmt.Errorf("unsupported page type for Continue")
}

func (req *InterceptedRequest) Respond(response *RequestResponse) error {
	if cdpPage, ok := req.page.(*CDPPage); ok {
		return cdpPage.respondToRequest(req.RequestID, response)
	}
	return fmt.Errorf("unsupported page type for Respond")
}

func (req *InterceptedRequest) Abort() error {
	if cdpPage, ok := req.page.(*CDPPage); ok {
		return cdpPage.abortRequest(req.RequestID)
	}
	return fmt.Errorf("unsupported page type for Abort")
}

// RequestResponse represents a custom response for intercepted requests
type RequestResponse struct {
	Status      int               `json:"status"`
	Headers     map[string]string `json:"headers"`
	Body        string            `json:"body"`
	ContentType string            `json:"contentType"`
}

// TurnstileSolver handles Cloudflare Turnstile captcha solving
type TurnstileSolver interface {
	Start() error
	Stop() error
	IsRunning() bool
}

// XvfbSession represents a virtual display session on Linux
type XvfbSession interface {
	Start() error
	Stop() error
	IsRunning() bool
	GetDisplay() string
}
