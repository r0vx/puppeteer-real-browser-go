package browser

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// turnstileTargetsJS 返回需要点击的 Turnstile 复选框坐标（容器左侧 30px、垂直居中，与原版 checkTurnstile 一致）。
// 优先用 cf-turnstile-response 输入框的父容器，已拿到 token 的跳过；
// 没有输入框时按尺寸启发式找容器（约 300x65、无内外边距、无子元素：控件本体在封闭的 shadow root 里）
const turnstileTargetsJS = `(() => {
	const at = r => ({x: r.x + 30, y: r.y + r.height / 2});
	const inputs = document.querySelectorAll('[name="cf-turnstile-response"]');
	if (inputs.length > 0) {
		const points = [];
		for (const input of inputs) {
			if (input.value) continue;
			if (input.parentElement) points.push(at(input.parentElement.getBoundingClientRect()));
		}
		return points;
	}
	const points = [];
	for (const div of document.querySelectorAll('div')) {
		const r = div.getBoundingClientRect();
		const s = getComputedStyle(div);
		if (r.width > 290 && r.width <= 310 && r.height >= 60 && r.height <= 80 &&
			s.margin === '0px' && s.padding === '0px' && !div.querySelector('*')) {
			points.push(at(r));
		}
	}
	return points;
})()`

// TurnstileAutoSolver 后台每秒检查一次页面上的 Cloudflare Turnstile，点击尚未通过的复选框。
// ConnectOptions.Turnstile 为 true 时由 Connect 自动启动
type TurnstileAutoSolver struct {
	page Page
	ctx  context.Context

	mu     sync.Mutex
	cancel context.CancelFunc
	done   chan struct{}
}

// NewTurnstileAutoSolver creates a solver for page; ctx 结束时后台循环随之退出
func NewTurnstileAutoSolver(page Page, ctx context.Context) *TurnstileAutoSolver {
	return &TurnstileAutoSolver{page: page, ctx: ctx}
}

// Start 启动后台循环，重复调用无副作用
func (s *TurnstileAutoSolver) Start() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.cancel != nil {
		return nil
	}
	ctx, cancel := context.WithCancel(s.ctx)
	s.cancel, s.done = cancel, make(chan struct{})
	go s.run(ctx, s.done)
	return nil
}

// Stop 停止后台循环并等待它退出
func (s *TurnstileAutoSolver) Stop() error {
	s.mu.Lock()
	cancel, done := s.cancel, s.done
	s.cancel, s.done = nil, nil
	s.mu.Unlock()
	if cancel != nil {
		cancel()
		<-done
	}
	return nil
}

// IsRunning reports whether the background loop is running
func (s *TurnstileAutoSolver) IsRunning() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.cancel != nil
}

// run 每秒检查并点击一次，直到 ctx 结束
func (s *TurnstileAutoSolver) run(ctx context.Context, done chan struct{}) {
	defer close(done)
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	for {
		s.clickPending()
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

// clickPending 点击所有尚未通过的 Turnstile 复选框（经 CDP 输入事件，isTrusted 为 true）
func (s *TurnstileAutoSolver) clickPending() {
	result, err := s.page.Evaluate(turnstileTargetsJS)
	if err != nil {
		return // 页面切换中等情况下偶发失败，下一轮重试
	}
	points, _ := result.([]interface{})
	for _, pt := range points {
		m, ok := pt.(map[string]interface{})
		if !ok {
			continue
		}
		x, okX := m["x"].(float64)
		y, okY := m["y"].(float64)
		if okX && okY {
			s.page.Click(x, y)
		}
	}
}

// WaitForSolution 等到页面上有 Turnstile 响应框拿到 token；页面没有 Turnstile 时等到超时返回错误
func (s *TurnstileAutoSolver) WaitForSolution(timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(s.ctx, timeout)
	defer cancel()

	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	for {
		solved, err := s.page.Evaluate(`[...document.querySelectorAll('[name="cf-turnstile-response"]')].some(e => e.value)`)
		if err == nil && solved == true {
			return nil
		}
		select {
		case <-ctx.Done():
			return fmt.Errorf("timeout waiting for Turnstile solution")
		case <-ticker.C:
		}
	}
}
