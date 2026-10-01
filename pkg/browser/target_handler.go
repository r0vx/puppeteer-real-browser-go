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
	allocCtx         context.Context
	opts             *ConnectOptions
	stealthScript    string
	mu               sync.RWMutex
	targets          map[target.ID]*targetInfo // 存储 cancel 函数以防止泄漏
	stopChan         chan struct{}
	isRunning        bool
	activeGoroutines sync.WaitGroup // 追踪活动的 goroutine
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
	// chromedp 把 target 事件循环绑定在首次 Run 的 ctx 上：先用 targetCtx 附加，
	// 否则 30s 超时后该标签页再收不到事件（Fetch 暂停的请求无人放行）
	if err := chromedp.Run(targetCtx); err != nil {
		cancel()
		return
	}
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
		if err := th.setupProxyAuthForTarget(targetCtx, timeoutCtx); err != nil {
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

// setupProxyAuthForTarget sets up proxy authentication for a specific target.
// 监听器绑定 targetCtx（标签页生命周期），开启拦截用 setupCtx（带超时）；
// 接管认证需拦截全部请求，暂停的请求在这里放行
func (th *TargetHandler) setupProxyAuthForTarget(targetCtx, setupCtx context.Context) error {
	run := func(action chromedp.ActionFunc) { chromedp.Run(targetCtx, action) }
	chromedp.ListenTarget(targetCtx, func(ev interface{}) {
		switch e := ev.(type) {
		case *fetch.EventRequestPaused:
			go run(func(ctx context.Context) error { return fetch.ContinueRequest(e.RequestID).Do(ctx) })
		case *fetch.EventAuthRequired:
			go run(func(ctx context.Context) error {
				return fetch.ContinueWithAuth(e.RequestID, proxyAuthResponse(th.opts.Proxy, e.AuthChallenge)).Do(ctx)
			})
		}
	})

	return chromedp.Run(setupCtx, chromedp.ActionFunc(func(ctx context.Context) error {
		if err := enableFetchAll(ctx, true); err != nil {
			return fmt.Errorf("failed to enable Fetch with auth: %w", err)
		}
		return nil
	}))
}
