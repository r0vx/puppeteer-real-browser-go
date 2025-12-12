package browser

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// WaitOptions 等待选项配置
type WaitOptions struct {
	Timeout         time.Duration // 超时时间
	Interval        time.Duration // 检查间隔
	ExponentialBack bool          // 是否使用指数退避
}

// DefaultWaitOptions 返回默认等待选项
func DefaultWaitOptions() *WaitOptions {
	return &WaitOptions{
		Timeout:         30 * time.Second,
		Interval:        500 * time.Millisecond,
		ExponentialBack: false,
	}
}

// WaitCondition 等待条件函数类型
type WaitCondition func() (bool, error)

// WaitWithContext 使用 context 的智能等待函数
// 替代硬编码的 time.Sleep，支持可中断和超时控制
func WaitWithContext(ctx context.Context, duration time.Duration) error {
	select {
	case <-time.After(duration):
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// WaitForCondition 等待条件满足（可中断）
// 比 utils.WaitWithTimeout 更灵活，支持 context 取消
func WaitForCondition(ctx context.Context, condition WaitCondition, opts *WaitOptions) error {
	if opts == nil {
		opts = DefaultWaitOptions()
	}

	// 创建带超时的 context
	ctx, cancel := context.WithTimeout(ctx, opts.Timeout)
	defer cancel()

	interval := opts.Interval
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	// 立即检查一次
	if ok, err := condition(); err != nil {
		return err
	} else if ok {
		return nil
	}

	// 指数退避参数
	backoffMultiplier := 1.5
	maxInterval := 5 * time.Second

	for {
		select {
		case <-ticker.C:
			ok, err := condition()
			if err != nil {
				return fmt.Errorf("condition check failed: %w", err)
			}
			if ok {
				return nil
			}

			// 指数退避
			if opts.ExponentialBack {
				interval = time.Duration(float64(interval) * backoffMultiplier)
				if interval > maxInterval {
					interval = maxInterval
				}
				ticker.Reset(interval)
			}

		case <-ctx.Done():
			return fmt.Errorf("wait timeout: %w", ctx.Err())
		}
	}
}

// SmartSleep 智能延迟函数
// 根据 context 状态决定是否继续等待
func SmartSleep(ctx context.Context, duration time.Duration) error {
	timer := time.NewTimer(duration)
	defer timer.Stop()

	select {
	case <-timer.C:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// RetryWithBackoff 使用指数退避的重试机制
func RetryWithBackoff(ctx context.Context, maxAttempts int, initialDelay time.Duration, fn func() error) error {
	delay := initialDelay
	maxDelay := 30 * time.Second

	for attempt := 1; attempt <= maxAttempts; attempt++ {
		err := fn()
		if err == nil {
			return nil
		}

		// 最后一次尝试失败
		if attempt == maxAttempts {
			return fmt.Errorf("all %d attempts failed: %w", maxAttempts, err)
		}

		// 等待后重试
		select {
		case <-time.After(delay):
			// 指数增长延迟
			delay = time.Duration(float64(delay) * 1.5)
			if delay > maxDelay {
				delay = maxDelay
			}
		case <-ctx.Done():
			return fmt.Errorf("retry cancelled: %w", ctx.Err())
		}
	}

	return fmt.Errorf("retry exhausted")
}

// WaitGroupWithTimeout 包装 sync.WaitGroup，增加超时控制
type WaitGroupWithTimeout struct {
	wg sync.WaitGroup
}

// Add 添加等待计数
func (w *WaitGroupWithTimeout) Add(delta int) {
	w.wg.Add(delta)
}

// Done 减少等待计数
func (w *WaitGroupWithTimeout) Done() {
	w.wg.Done()
}

// WaitWithTimeout 等待所有任务完成或超时
func (w *WaitGroupWithTimeout) WaitWithTimeout(timeout time.Duration) error {
	done := make(chan struct{})

	go func() {
		w.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		return nil
	case <-time.After(timeout):
		return fmt.Errorf("wait group timeout after %v", timeout)
	}
}

// WaitWithContext 等待所有任务完成或 context 取消
func (w *WaitGroupWithTimeout) WaitWithContext(ctx context.Context) error {
	done := make(chan struct{})

	go func() {
		w.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}
