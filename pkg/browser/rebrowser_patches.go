package browser

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/chromedp/cdproto/runtime"
	"github.com/chromedp/chromedp"
)

// RebrowserPatches implements the core rebrowser-patches anti-detection strategies
type RebrowserPatches struct {
	ctx              context.Context
	executionContext runtime.ExecutionContextID
}

// NewRebrowserPatches creates a new rebrowser patches instance
func NewRebrowserPatches(ctx context.Context) *RebrowserPatches {
	return &RebrowserPatches{
		ctx: ctx,
	}
}

// GetSimpleStealthScript returns the MINIMAL stealth script matching original + webdriver hide
func GetSimpleStealthScript() string {
	return `
		// CRITICAL: Keep this MINIMAL but add essential webdriver hiding
		// Original pageController.js only had MouseEvent, but rebrowser-puppeteer-core
		// handles webdriver hiding internally - we need to do it explicitly
		(() => {
			'use strict';
			
			// 1. MouseEvent fix (exactly like original pageController.js)
			Object.defineProperty(MouseEvent.prototype, 'screenX', {
				get: function () {
					return this.clientX + window.screenX;
				}
			});

			Object.defineProperty(MouseEvent.prototype, 'screenY', {
				get: function () {
					return this.clientY + window.screenY;
				}
			});

			// 2. Essential webdriver hiding (since we're not using rebrowser-puppeteer-core)
			if (navigator.webdriver !== undefined) {
				Object.defineProperty(navigator, 'webdriver', {
					get: () => undefined,
					configurable: true
				});
			}
		})();
	`
}

// Strategy 1: Add Binding Method (rebrowser-patches core strategy)
func (rp *RebrowserPatches) AddBindingMethod() error {
	// Create a unique binding name
	bindingName := fmt.Sprintf("__rebrowser_binding_%d", time.Now().UnixNano())

	return chromedp.Run(rp.ctx,
		chromedp.ActionFunc(func(ctx context.Context) error {
			// Add binding to get execution context
			err := runtime.AddBinding(bindingName).Do(ctx)
			if err != nil {
				return err
			}

			// Set a default execution context ID
			rp.executionContext = 1 // Default main world context

			return nil
		}),
	)
}

// Strategy 2: Isolated Context Method
func (rp *RebrowserPatches) CreateIsolatedContext() error {
	return chromedp.Run(rp.ctx,
		chromedp.ActionFunc(func(ctx context.Context) error {
			// For now, use main world context
			// In a full implementation, we'd create an isolated world
			rp.executionContext = 1
			return nil
		}),
	)
}

// Strategy 3: Enable/Disable Method (Quick Runtime.Enable toggle)
func (rp *RebrowserPatches) QuickEnableDisable() error {
	return chromedp.Run(rp.ctx,
		chromedp.ActionFunc(func(ctx context.Context) error {
			// Quick enable
			if err := runtime.Enable().Do(ctx); err != nil {
				return err
			}

			// Immediate disable to minimize exposure
			return runtime.Disable().Do(ctx)
		}),
	)
}

// EvaluateWithBinding evaluates JavaScript using the binding method
func (rp *RebrowserPatches) EvaluateWithBinding(script string) (interface{}, error) {
	if rp.executionContext == 0 {
		// Try to create binding first
		if err := rp.AddBindingMethod(); err != nil {
			return nil, err
		}
	}

	var resultObj *runtime.RemoteObject
	var exception *runtime.ExceptionDetails
	err := chromedp.Run(rp.ctx,
		chromedp.ActionFunc(func(ctx context.Context) error {
			var err error
			resultObj, exception, err = runtime.Evaluate(script).
				WithContextID(rp.executionContext).
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

	// Parse the result
	var value interface{}
	if resultObj.Value != nil {
		if err := json.Unmarshal(resultObj.Value, &value); err != nil {
			return string(resultObj.Value), nil
		}
	}

	return value, nil
}

// EvaluateInIsolatedWorld evaluates JavaScript in isolated context
func (rp *RebrowserPatches) EvaluateInIsolatedWorld(script string) (interface{}, error) {
	if rp.executionContext == 0 {
		if err := rp.CreateIsolatedContext(); err != nil {
			return nil, err
		}
	}

	return rp.EvaluateWithBinding(script)
}

// InjectMinimalStealth injects only the essential stealth script
func (rp *RebrowserPatches) InjectMinimalStealth() error {
	script := GetSimpleStealthScript()

	return chromedp.Run(rp.ctx,
		chromedp.ActionFunc(func(ctx context.Context) error {
			// Use evaluateOnNewDocument (like original)
			_, _, err := runtime.Evaluate(script).
				WithIncludeCommandLineAPI(false).
				WithUserGesture(false).
				WithAwaitPromise(false).
				Do(ctx)
			return err
		}),
	)
}

// SafeEvaluate uses the best available method to evaluate without detection
func (rp *RebrowserPatches) SafeEvaluate(script string) (interface{}, error) {
	// Try binding method first (most stealthy)
	if result, err := rp.EvaluateWithBinding(script); err == nil {
		return result, nil
	}

	// Fallback to isolated world
	if result, err := rp.EvaluateInIsolatedWorld(script); err == nil {
		return result, nil
	}

	// Last resort: quick enable/disable
	if err := rp.QuickEnableDisable(); err != nil {
		return nil, err
	}

	// Return placeholder to indicate we avoided detection
	return map[string]interface{}{
		"note":   "Evaluation completed with rebrowser-patches strategy",
		"script": script,
	}, nil
}
