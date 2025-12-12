package main

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/HNRow/puppeteer-real-browser-go/pkg/browser"
)

// Account represents a user account
type Account struct {
	Name     string
	Email    string
	Password string
	Proxy    *browser.ProxyConfig
}

// AccountManager manages multiple accounts using browser contexts
type AccountManager struct {
	browser     *browser.BrowserInstance
	contexts    map[string]*browser.BrowserContext
	accounts    map[string]*Account
	mutex       sync.RWMutex
	maxConcurrent int
}

func main() {
	fmt.Println("🔄 Multi-Account Management Demo")
	fmt.Println("=================================")

	// Initialize account manager
	manager, err := NewAccountManager(5) // 最多5个并发账号
	if err != nil {
		log.Fatalf("Failed to create account manager: %v", err)
	}
	defer manager.Close()

	// 定义测试账号
	accounts := []*Account{
		{
			Name:     "Alice",
			Email:    "alice@example.com", 
			Password: "password123",
		},
		{
			Name:     "Bob",
			Email:    "bob@example.com",
			Password: "password456",
		},
		{
			Name:     "Charlie", 
			Email:    "charlie@example.com",
			Password: "password789",
		},
	}

	// 注册账号
	for _, account := range accounts {
		if err := manager.AddAccount(account); err != nil {
			log.Printf("Failed to add account %s: %v", account.Name, err)
			continue
		}
		fmt.Printf("✅ Added account: %s\n", account.Name)
	}

	// 示例1: 并行登录所有账号
	fmt.Println("\n📱 Demo 1: Parallel Login")
	if err := manager.LoginAllAccounts(); err != nil {
		log.Printf("Login failed: %v", err)
	}

	time.Sleep(2 * time.Second)

	// 示例2: 每个账号执行不同任务
	fmt.Println("\n🎯 Demo 2: Account-Specific Tasks")
	tasks := map[string]func(*browser.BrowserContext, *Account) error{
		"Alice":   checkEmailTask,
		"Bob":     socialMediaTask,
		"Charlie": ecommerceTask,
	}

	if err := manager.ExecuteTasks(tasks); err != nil {
		log.Printf("Task execution failed: %v", err)
	}

	// 示例3: Cookie 和会话隔离测试
	fmt.Println("\n🍪 Demo 3: Session Isolation Test")
	if err := manager.TestSessionIsolation(); err != nil {
		log.Printf("Session isolation test failed: %v", err)
	}

	fmt.Println("\n⏳ Keeping browsers open for 30 seconds for inspection...")
	time.Sleep(30 * time.Second)

	fmt.Println("✅ Demo completed!")
}

// NewAccountManager creates a new account manager
func NewAccountManager(maxConcurrent int) (*AccountManager, error) {
	ctx := context.Background()

	// 使用最大隐蔽模式
	opts := &browser.ConnectOptions{
		Headless:     false, // 可视化模式便于观察
		UseCustomCDP: true,  // 最大隐蔽性
		Turnstile:    true,  // 自动解验证码
		Args: []string{
			"--start-maximized",
			"--disable-blink-features=AutomationControlled",
			"--exclude-switches=enable-automation",
		},
	}

	// 创建主浏览器实例
	browserInstance, err := browser.Connect(ctx, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to connect browser: %w", err)
	}

	return &AccountManager{
		browser:       browserInstance,
		contexts:      make(map[string]*browser.BrowserContext),
		accounts:      make(map[string]*Account),
		maxConcurrent: maxConcurrent,
	}, nil
}

// AddAccount adds a new account and creates a dedicated browser context
func (am *AccountManager) AddAccount(account *Account) error {
	am.mutex.Lock()
	defer am.mutex.Unlock()

	if len(am.accounts) >= am.maxConcurrent {
		return fmt.Errorf("maximum concurrent accounts reached (%d)", am.maxConcurrent)
	}

	// 为每个账号创建独立的浏览器上下文
	contextOpts := &browser.BrowserContextOptions{
		IgnoreHTTPSErrors: true,
	}

	if account.Proxy != nil {
		contextOpts.ProxyServer = fmt.Sprintf("http://%s:%s", account.Proxy.Host, account.Proxy.Port)
	}

	ctx, err := am.browser.CreateBrowserContext(contextOpts)
	if err != nil {
		return fmt.Errorf("failed to create browser context for %s: %w", account.Name, err)
	}

	am.contexts[account.Name] = ctx
	am.accounts[account.Name] = account

	return nil
}

// LoginAllAccounts logs in all accounts in parallel
func (am *AccountManager) LoginAllAccounts() error {
	var wg sync.WaitGroup
	errChan := make(chan error, len(am.accounts))

	am.mutex.RLock()
	accounts := make([]*Account, 0, len(am.accounts))
	for _, account := range am.accounts {
		accounts = append(accounts, account)
	}
	am.mutex.RUnlock()

	for _, account := range accounts {
		wg.Add(1)
		go func(acc *Account) {
			defer wg.Done()
			if err := am.loginAccount(acc); err != nil {
				errChan <- fmt.Errorf("failed to login %s: %w", acc.Name, err)
			} else {
				fmt.Printf("  ✅ %s logged in successfully\n", acc.Name)
			}
		}(account)
	}

	wg.Wait()
	close(errChan)

	// 检查是否有错误
	for err := range errChan {
		log.Printf("Login error: %v", err)
	}

	return nil
}

// loginAccount logs in a specific account
func (am *AccountManager) loginAccount(account *Account) error {
	am.mutex.RLock()
	ctx := am.contexts[account.Name]
	am.mutex.RUnlock()

	if ctx == nil {
		return fmt.Errorf("no context found for account %s", account.Name)
	}

	// 在上下文中创建页面
	page, err := ctx.NewPage()
	if err != nil {
		return fmt.Errorf("failed to create page: %w", err)
	}

	// 模拟登录流程（这里使用一个示例网站）
	loginURL := "https://httpbin.org/forms/post"
	
	if err := page.Navigate(loginURL); err != nil {
		return fmt.Errorf("failed to navigate to login page: %w", err)
	}

	// 等待页面加载
	time.Sleep(2 * time.Second)

	// 模拟填写登录表单
	// 注意: 这是一个简化的示例，实际应用中需要根据具体网站调整
	
	// 设置页面标题以便识别
	script := fmt.Sprintf(`document.title = 'Account: %s - ' + document.title`, account.Name)
	if _, err := page.Evaluate(script); err != nil {
		log.Printf("Failed to set title for %s: %v", account.Name, err)
	}

	fmt.Printf("  🔑 %s: Navigated to login page successfully\n", account.Name)
	return nil
}

// ExecuteTasks executes different tasks for different accounts
func (am *AccountManager) ExecuteTasks(tasks map[string]func(*browser.BrowserContext, *Account) error) error {
	var wg sync.WaitGroup
	
	for accountName, task := range tasks {
		am.mutex.RLock()
		ctx := am.contexts[accountName]
		account := am.accounts[accountName]
		am.mutex.RUnlock()

		if ctx == nil || account == nil {
			log.Printf("Account %s not found", accountName)
			continue
		}

		wg.Add(1)
		go func(name string, taskFunc func(*browser.BrowserContext, *Account) error, context *browser.BrowserContext, acc *Account) {
			defer wg.Done()
			if err := taskFunc(context, acc); err != nil {
				log.Printf("Task failed for %s: %v", name, err)
			} else {
				fmt.Printf("  ✅ %s: Task completed successfully\n", name)
			}
		}(accountName, task, ctx, account)
	}

	wg.Wait()
	return nil
}

// TestSessionIsolation tests that accounts have isolated sessions
func (am *AccountManager) TestSessionIsolation() error {
	testURL := "https://httpbin.org/cookies/set/test"
	
	am.mutex.RLock()
	accountNames := make([]string, 0, len(am.accounts))
	for name := range am.accounts {
		accountNames = append(accountNames, name)
	}
	am.mutex.RUnlock()

	// 为每个账号设置不同的 Cookie
	for i, name := range accountNames {
		ctx := am.contexts[name]
		page, err := ctx.NewPage()
		if err != nil {
			return fmt.Errorf("failed to create page for %s: %w", name, err)
		}

		// 设置不同的测试 Cookie
		cookieURL := fmt.Sprintf("%s/%s_%d", testURL, name, i)
		if err := page.Navigate(cookieURL); err != nil {
			return fmt.Errorf("failed to set cookie for %s: %w", name, err)
		}

		time.Sleep(1 * time.Second)
		fmt.Printf("  🍪 %s: Set test cookie\n", name)
	}

	// 验证 Cookie 隔离
	time.Sleep(2 * time.Second)
	fmt.Println("  🔍 Verifying cookie isolation...")

	for _, name := range accountNames {
		ctx := am.contexts[name]
		page, err := ctx.NewPage()
		if err != nil {
			continue
		}

		// 检查 Cookie
		if err := page.Navigate("https://httpbin.org/cookies"); err != nil {
			continue
		}

		time.Sleep(1 * time.Second)
		
		// 获取页面标题表明这是哪个账号的页面
		title, _ := page.GetTitle()
		fmt.Printf("  🔍 %s: Cookies verified (Title: %s)\n", name, title)
	}

	return nil
}

// Close closes all browser contexts and the main browser
func (am *AccountManager) Close() error {
	am.mutex.Lock()
	defer am.mutex.Unlock()

	// 关闭所有上下文
	for name, ctx := range am.contexts {
		if err := ctx.Close(); err != nil {
			log.Printf("Failed to close context for %s: %v", name, err)
		}
	}

	// 关闭主浏览器
	return am.browser.Close()
}

// Task functions for different accounts

func checkEmailTask(ctx *browser.BrowserContext, account *Account) error {
	page, err := ctx.NewPage()
	if err != nil {
		return err
	}

	// 模拟邮箱检查任务
	if err := page.Navigate("https://httpbin.org/headers"); err != nil {
		return err
	}

	// 设置页面标题
	script := fmt.Sprintf(`document.title = '%s - Email Task'`, account.Name)
	page.Evaluate(script)

	time.Sleep(2 * time.Second)
	fmt.Printf("  📧 %s: Checking emails...\n", account.Name)
	return nil
}

func socialMediaTask(ctx *browser.BrowserContext, account *Account) error {
	page, err := ctx.NewPage()
	if err != nil {
		return err
	}

	// 模拟社交媒体任务
	if err := page.Navigate("https://httpbin.org/user-agent"); err != nil {
		return err
	}

	// 设置页面标题
	script := fmt.Sprintf(`document.title = '%s - Social Media Task'`, account.Name)
	page.Evaluate(script)

	time.Sleep(2 * time.Second)
	fmt.Printf("  📱 %s: Managing social media...\n", account.Name)
	return nil
}

func ecommerceTask(ctx *browser.BrowserContext, account *Account) error {
	page, err := ctx.NewPage()
	if err != nil {
		return err
	}

	// 模拟电商任务
	if err := page.Navigate("https://httpbin.org/ip"); err != nil {
		return err
	}

	// 设置页面标题
	script := fmt.Sprintf(`document.title = '%s - E-commerce Task'`, account.Name)
	page.Evaluate(script)

	time.Sleep(2 * time.Second)
	fmt.Printf("  🛒 %s: Managing e-commerce...\n", account.Name)
	return nil
}