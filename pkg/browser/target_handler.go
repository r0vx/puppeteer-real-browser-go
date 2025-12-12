package browser

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/chromedp/cdproto/fetch"
	"github.com/chromedp/cdproto/page"
	"github.com/chromedp/cdproto/target"
	"github.com/chromedp/chromedp"
)

// targetInfo holds context and its cancel function for a target
type targetInfo struct {
	ctx    context.Context
	cancel context.CancelFunc
}

// TargetHandler manages new page/tab events and applies stealth scripts
type TargetHandler struct {
	allocCtx          context.Context
	opts              *ConnectOptions
	stealthScript     string
	mu                sync.RWMutex
	targets           map[target.ID]*targetInfo // 存储 cancel 函数以防止泄漏
	stopChan          chan struct{}
	isRunning         bool
	activeGoroutines  sync.WaitGroup // 追踪活动的 goroutine
}

// NewTargetHandler creates a new target handler
func NewTargetHandler(allocCtx context.Context, opts *ConnectOptions) *TargetHandler {
	return &TargetHandler{
		allocCtx:      allocCtx,
		opts:          opts,
		stealthScript: GetSimpleStealthScript(),
		targets:       make(map[target.ID]*targetInfo),
		stopChan:      make(chan struct{}),
	}
}

// Start begins listening for new targets (pages/tabs)
func (th *TargetHandler) Start(ctx context.Context) error {
	th.mu.Lock()
	if th.isRunning {
		th.mu.Unlock()
		return nil
	}
	th.isRunning = true
	th.mu.Unlock()

	// Listen for target events on the browser context
	chromedp.ListenBrowser(ctx, func(ev interface{}) {
		switch e := ev.(type) {
		case *target.EventTargetCreated:
			// Handle new target (page/tab) created in goroutine with tracking
			th.activeGoroutines.Add(1)
			go func(event *target.EventTargetCreated) {
				defer th.activeGoroutines.Done()
				th.handleTargetCreated(ctx, event)
			}(e)
		case *target.EventTargetDestroyed:
			// Clean up destroyed target
			th.handleTargetDestroyed(e)
		}
	})

	return nil
}

// Stop stops the target handler and waits for all goroutines to finish
func (th *TargetHandler) Stop() {
	th.mu.Lock()
	if !th.isRunning {
		th.mu.Unlock()
		return
	}
	th.isRunning = false
	close(th.stopChan)
	
	// 取消所有 target contexts
	for _, info := range th.targets {
		if info.cancel != nil {
			info.cancel()
		}
	}
	th.targets = make(map[target.ID]*targetInfo)
	th.mu.Unlock()
	
	// 等待所有 goroutine 完成（带超时）
	done := make(chan struct{})
	go func() {
		th.activeGoroutines.Wait()
		close(done)
	}()
	
	select {
	case <-done:
		// 所有 goroutine 已完成
	case <-time.After(5 * time.Second):
		// 超时，但继续（避免永久阻塞）
		fmt.Println("Warning: Some target handler goroutines did not finish within 5 seconds")
	}
}

// handleTargetCreated handles a new target (page/tab) being created
func (th *TargetHandler) handleTargetCreated(ctx context.Context, ev *target.EventTargetCreated) {
	// 检查是否已停止
	select {
	case <-th.stopChan:
		return
	default:
	}
	
	// Only handle page targets
	if ev.TargetInfo.Type != "page" {
		return
	}

	targetID := ev.TargetInfo.TargetID
	
	// 创建带超时的 context
	targetCtx, cancel := chromedp.NewContext(th.allocCtx, chromedp.WithTargetID(targetID))
	timeoutCtx, timeoutCancel := context.WithTimeout(targetCtx, 30*time.Second)
	
	// 组合 cancel 函数
	combinedCancel := func() {
		timeoutCancel()
		cancel()
	}
	
	// 存储 context 和 cancel 函数
	th.mu.Lock()
	th.targets[targetID] = &targetInfo{
		ctx:    targetCtx,
		cancel: combinedCancel,
	}
	th.mu.Unlock()

	// Inject stealth scripts into the new page
	err := th.injectStealthToTarget(timeoutCtx)
	if err != nil {
		// Log error but don't fail - the target might have been destroyed
		fmt.Printf("Warning: failed to inject stealth to new target %s: %v\n", targetID, err)
		combinedCancel()
		
		// 清理
		th.mu.Lock()
		delete(th.targets, targetID)
		th.mu.Unlock()
		return
	}

	// Set up proxy authentication if needed
	if th.opts.Proxy != nil && th.opts.Proxy.Username != "" {
		if err := th.setupProxyAuthForTarget(timeoutCtx); err != nil {
			fmt.Printf("Warning: failed to setup proxy auth for target %s: %v\n", targetID, err)
		}
	}
}

// handleTargetDestroyed cleans up a destroyed target
func (th *TargetHandler) handleTargetDestroyed(ev *target.EventTargetDestroyed) {
	th.mu.Lock()
	defer th.mu.Unlock()
	
	// 取消 context 以释放资源
	if info, exists := th.targets[ev.TargetID]; exists {
		if info.cancel != nil {
			info.cancel()
		}
		delete(th.targets, ev.TargetID)
	}
}

// injectStealthToTarget injects stealth scripts to a specific target
func (th *TargetHandler) injectStealthToTarget(ctx context.Context) error {
	return chromedp.Run(ctx,
		chromedp.ActionFunc(func(ctx context.Context) error {
			// Enable Page domain
			if err := page.Enable().Do(ctx); err != nil {
				return fmt.Errorf("failed to enable Page domain: %w", err)
			}
			
			// Inject stealth script for all new documents
			_, err := page.AddScriptToEvaluateOnNewDocument(th.stealthScript).Do(ctx)
			return err
		}),
	)
}

// setupProxyAuthForTarget sets up proxy authentication for a specific target
func (th *TargetHandler) setupProxyAuthForTarget(ctx context.Context) error {
	return chromedp.Run(ctx,
		chromedp.ActionFunc(func(ctx context.Context) error {
			// Enable fetch with auth request handling
			if err := fetch.Enable().WithHandleAuthRequests(true).Do(ctx); err != nil {
				return fmt.Errorf("failed to enable Fetch with auth: %w", err)
			}

			// Listen for auth required events
			chromedp.ListenTarget(ctx, func(ev interface{}) {
				if authEv, ok := ev.(*fetch.EventAuthRequired); ok {
					go th.handleAuthRequired(ctx, authEv)
				}
			})

			return nil
		}),
	)
}

// handleAuthRequired handles proxy authentication challenges
func (th *TargetHandler) handleAuthRequired(ctx context.Context, ev *fetch.EventAuthRequired) {
	if th.opts.Proxy == nil || th.opts.Proxy.Username == "" {
		// No credentials, cancel the auth
		fetch.ContinueWithAuth(ev.RequestID, &fetch.AuthChallengeResponse{
			Response: fetch.AuthChallengeResponseResponseCancelAuth,
		}).Do(ctx)
		return
	}

	// Provide credentials
	fetch.ContinueWithAuth(ev.RequestID, &fetch.AuthChallengeResponse{
		Response: fetch.AuthChallengeResponseResponseProvideCredentials,
		Username: th.opts.Proxy.Username,
		Password: th.opts.Proxy.Password,
	}).Do(ctx)
}
