// Package turnstile 提供 Cloudflare Turnstile 自动点击。
// 实现位于 browser 包（ConnectOptions.Turnstile 需要在 Connect 时自动启用，放在这里会循环导入），
// 本包保留原有 API 供手动使用。
package turnstile

import (
	"context"

	"github.com/r0vx/puppeteer-real-browser-go/pkg/browser"
)

// Solver handles Cloudflare Turnstile captcha solving
type Solver = browser.TurnstileAutoSolver

// NewSolver creates a new Turnstile solver
func NewSolver(page browser.Page, ctx context.Context) *Solver {
	return browser.NewTurnstileAutoSolver(page, ctx)
}
