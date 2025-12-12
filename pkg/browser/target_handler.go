package browser

import (
	"context"
	"fmt"
	"sync"

	"github.com/chromedp/cdproto/fetch"
	"github.com/chromedp/cdproto/page"
	"github.com/chromedp/cdproto/target"
	"github.com/chromedp/chromedp"
)

// TargetHandler manages new page/tab events and applies stealth scripts
type TargetHandler struct {
	allocCtx    context.Context
	opts        *ConnectOptions
	stealthScript string
	mu          sync.RWMutex
	targets     map[target.ID]context.Context
	stopChan    chan struct{}
	isRunning   bool
}

// NewTargetHandler creates a new target handler
func NewTargetHandler(allocCtx context.Context, opts *ConnectOptions) *TargetHandler {
	return &TargetHandler{
		allocCtx:      allocCtx,
		opts:          opts,
		stealthScript: GetSimpleStealthScript(),
		targets:       make(map[target.ID]context.Context),
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
			// Handle new target (page/tab) created
			go th.handleTargetCreated(ctx, e)
		case *target.EventTargetDestroyed:
			// Clean up destroyed target
			th.handleTargetDestroyed(e)
		}
	})

	return nil
}

// Stop stops the target handler
func (th *TargetHandler) Stop() {
	th.mu.Lock()
	defer th.mu.Unlock()
	
	if !th.isRunning {
		return
	}
	
	th.isRunning = false
	close(th.stopChan)
}

// handleTargetCreated handles a new target (page/tab) being created
func (th *TargetHandler) handleTargetCreated(ctx context.Context, ev *target.EventTargetCreated) {
	// Only handle page targets
	if ev.TargetInfo.Type != "page" {
		return
	}

	targetID := ev.TargetInfo.TargetID
	
	// Create a new context for this target
	targetCtx, cancel := chromedp.NewContext(th.allocCtx, chromedp.WithTargetID(targetID))
	
	th.mu.Lock()
	th.targets[targetID] = targetCtx
	th.mu.Unlock()

	// Inject stealth scripts into the new page
	err := th.injectStealthToTarget(targetCtx)
	if err != nil {
		// Log error but don't fail - the target might have been destroyed
		fmt.Printf("Warning: failed to inject stealth to new target %s: %v\n", targetID, err)
		cancel()
		return
	}

	// Set up proxy authentication if needed
	if th.opts.Proxy != nil && th.opts.Proxy.Username != "" {
		th.setupProxyAuthForTarget(targetCtx)
	}
}

// handleTargetDestroyed cleans up a destroyed target
func (th *TargetHandler) handleTargetDestroyed(ev *target.EventTargetDestroyed) {
	th.mu.Lock()
	defer th.mu.Unlock()
	
	delete(th.targets, ev.TargetID)
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
