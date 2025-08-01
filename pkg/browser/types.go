package browser

import (
	"context"
	"os/exec"
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
	browser Browser
	page    Page
	chrome  *ChromeProcess
	ctx     context.Context
	cancel  context.CancelFunc
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
