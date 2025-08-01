package browser

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/chromedp/chromedp"
	"github.com/chromedp/cdproto/runtime"
)

// RuntimeBypass implements the core rebrowser-patches Runtime.Enable bypass
// This is the CRITICAL component for avoiding Cloudflare detection
type RuntimeBypass struct {
	ctx             context.Context
	executionCtxID  runtime.ExecutionContextID
	bindingName     string
	runtimeEnabled  bool
	mutex           sync.RWMutex
}

// NewRuntimeBypass creates a new Runtime.Enable bypass system
func NewRuntimeBypass(ctx context.Context) *RuntimeBypass {
	return &RuntimeBypass{
		ctx:         ctx,
		bindingName: fmt.Sprintf("__bypass_%d", time.Now().UnixNano()),
	}
}

// InitializeBypass sets up the bypass system without triggering Runtime.Enable
func (rb *RuntimeBypass) InitializeBypass() error {
	rb.mutex.Lock()
	defer rb.mutex.Unlock()

	// Strategy 1: Use AddBinding to get execution context WITHOUT Runtime.Enable
	// This is the key rebrowser-patches technique
	return chromedp.Run(rb.ctx,
		chromedp.ActionFunc(func(ctx context.Context) error {
			// Add binding to get execution context ID
			err := runtime.AddBinding(rb.bindingName).Do(ctx)
			if err != nil {
				return fmt.Errorf("failed to add binding: %w", err)
			}

			// Set default execution context (main world)
			rb.executionCtxID = 1

			return nil
		}),
	)
}

// QuickEnableDisable implements the rebrowser-patches quick enable/disable strategy
func (rb *RuntimeBypass) QuickEnableDisable() error {
	rb.mutex.Lock()
	defer rb.mutex.Unlock()

	return chromedp.Run(rb.ctx,
		chromedp.ActionFunc(func(ctx context.Context) error {
			// Quick enable to get context events
			if err := runtime.Enable().Do(ctx); err != nil {
				return err
			}

			// Immediate disable to minimize exposure
			if err := runtime.Disable().Do(ctx); err != nil {
				return err
			}

			rb.runtimeEnabled = false
			return nil
		}),
	)
}

// SafeEvaluate evaluates JavaScript without triggering Runtime.Enable detection
func (rb *RuntimeBypass) SafeEvaluate(script string) (interface{}, error) {
	rb.mutex.RLock()
	ctxID := rb.executionCtxID
	rb.mutex.RUnlock()

	if ctxID == 0 {
		// Initialize if not done yet
		if err := rb.InitializeBypass(); err != nil {
			return nil, err
		}
		ctxID = 1
	}

	var resultObj *runtime.RemoteObject
	var exception *runtime.ExceptionDetails

	err := chromedp.Run(rb.ctx,
		chromedp.ActionFunc(func(ctx context.Context) error {
			var err error
			// Use specific context ID to avoid triggering Runtime.Enable
			resultObj, exception, err = runtime.Evaluate(script).
				WithContextID(ctxID).
				WithReturnByValue(true).
				WithAwaitPromise(false).
				WithUserGesture(false).
				WithIncludeCommandLineAPI(false).
				Do(ctx)
			return err
		}),
	)

	if err != nil {
		return nil, err
	}

	if exception != nil {
		return nil, fmt.Errorf("evaluation error: %s", exception.Text)
	}

	// Parse the result value
	var value interface{}
	if resultObj.Value != nil {
		if err := json.Unmarshal(resultObj.Value, &value); err != nil {
			return string(resultObj.Value), nil
		}
	}

	return value, nil
}

// AvoidRuntimeEnable completely avoids using Runtime.Enable
// This is the core function that prevents Cloudflare detection
func (rb *RuntimeBypass) AvoidRuntimeEnable() error {
	rb.mutex.Lock()
	defer rb.mutex.Unlock()

	// Mark that we're explicitly avoiding Runtime.Enable
	rb.runtimeEnabled = false

	// Use binding-based approach instead
	return rb.InitializeBypass()
}

// IsRuntimeEnabled checks if Runtime domain is enabled
func (rb *RuntimeBypass) IsRuntimeEnabled() bool {
	rb.mutex.RLock()
	defer rb.mutex.RUnlock()
	return rb.runtimeEnabled
}

// GetExecutionContextID returns the current execution context ID
func (rb *RuntimeBypass) GetExecutionContextID() runtime.ExecutionContextID {
	rb.mutex.RLock()
	defer rb.mutex.RUnlock()
	return rb.executionCtxID
}

// EvaluateWithoutRuntimeEnable evaluates script using bypass techniques
func (rb *RuntimeBypass) EvaluateWithoutRuntimeEnable(script string) (interface{}, error) {
	// Ensure bypass is initialized
	if rb.executionCtxID == 0 {
		if err := rb.InitializeBypass(); err != nil {
			return nil, err
		}
	}

	// Use safe evaluation method
	return rb.SafeEvaluate(script)
}

// CleanupBinding removes the bypass binding
func (rb *RuntimeBypass) CleanupBinding() error {
	if rb.bindingName == "" {
		return nil
	}

	return chromedp.Run(rb.ctx,
		chromedp.ActionFunc(func(ctx context.Context) error {
			return runtime.RemoveBinding(rb.bindingName).Do(ctx)
		}),
	)
}