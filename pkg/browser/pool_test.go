package browser

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"
)

// TestBrowserPool 测试浏览器池基本功能
func TestBrowserPool(t *testing.T) {
	if testing.Short() {
		t.Skip("跳过集成测试")
	}

	ctx := context.Background()
	opts := &ConnectOptions{
		Headless: true,
		Args:     []string{"--disable-gpu", "--no-sandbox"},
	}

	pool := NewBrowserPool(3, opts)
	defer pool.Close()

	// 测试获取实例
	t.Run("Acquire", func(t *testing.T) {
		instance, err := pool.Acquire(ctx)
		if err != nil {
			t.Fatalf("获取实例失败: %v", err)
		}

		if instance == nil {
			t.Fatal("实例为 nil")
		}

		pool.Release(instance)
	})

	// 测试预热
	t.Run("Warmup", func(t *testing.T) {
		err := pool.Warmup(ctx, 2)
		if err != nil {
			t.Fatalf("预热失败: %v", err)
		}

		stats := pool.Stats()
		if stats.Available < 2 {
			t.Errorf("预热后可用实例数不足: got %d, want >= 2", stats.Available)
		}
	})

	// 测试统计信息
	t.Run("Stats", func(t *testing.T) {
		stats := pool.Stats()
		t.Logf("池统计: 可用=%d, 已创建=%d, 最大=%d", 
			stats.Available, stats.Created, stats.MaxSize)

		if stats.MaxSize != 3 {
			t.Errorf("最大容量错误: got %d, want 3", stats.MaxSize)
		}
	})

	// 测试函数式 API
	t.Run("WithPooledBrowser", func(t *testing.T) {
		err := pool.WithPooledBrowser(ctx, func(instance *BrowserInstance) error {
			page := instance.Page()
			return page.Navigate("https://example.com")
		})

		if err != nil {
			t.Fatalf("函数式 API 失败: %v", err)
		}
	})
}

// TestBrowserPoolConcurrent 测试并发安全性
func TestBrowserPoolConcurrent(t *testing.T) {
	if testing.Short() {
		t.Skip("跳过集成测试")
	}

	ctx := context.Background()
	opts := &ConnectOptions{
		Headless: true,
		Args:     []string{"--disable-gpu", "--no-sandbox"},
	}

	pool := NewBrowserPool(5, opts)
	defer pool.Close()

	pool.Warmup(ctx, 3)

	var wg sync.WaitGroup
	successCount := 0
	var mu sync.Mutex

	// 并发访问
	concurrency := 10
	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()

			err := pool.WithPooledBrowser(ctx, func(instance *BrowserInstance) error {
				page := instance.Page()
				return page.Navigate("https://example.com")
			})

			if err == nil {
				mu.Lock()
				successCount++
				mu.Unlock()
			}
		}(i)
	}

	wg.Wait()

	t.Logf("并发测试: %d/%d 成功", successCount, concurrency)

	if successCount < concurrency/2 {
		t.Errorf("成功率过低: %d/%d", successCount, concurrency)
	}
}

// TestWaitWithContext 测试智能等待
func TestWaitWithContext(t *testing.T) {
	ctx := context.Background()

	// 测试正常等待
	t.Run("Normal", func(t *testing.T) {
		start := time.Now()
		err := WaitWithContext(ctx, 100*time.Millisecond)
		elapsed := time.Since(start)

		if err != nil {
			t.Errorf("等待失败: %v", err)
		}

		if elapsed < 100*time.Millisecond {
			t.Errorf("等待时间不足: %v", elapsed)
		}
	})

	// 测试取消
	t.Run("Cancel", func(t *testing.T) {
		ctx, cancel := context.WithCancel(ctx)
		cancel() // 立即取消

		err := WaitWithContext(ctx, 1*time.Second)
		if err == nil {
			t.Error("应该返回取消错误")
		}
	})
}

// TestWaitForCondition 测试条件等待
func TestWaitForCondition(t *testing.T) {
	ctx := context.Background()

	t.Run("ImmediateSuccess", func(t *testing.T) {
		condition := func() (bool, error) {
			return true, nil
		}

		opts := &WaitOptions{
			Timeout:  5 * time.Second,
			Interval: 100 * time.Millisecond,
		}

		start := time.Now()
		err := WaitForCondition(ctx, condition, opts)
		elapsed := time.Since(start)

		if err != nil {
			t.Errorf("等待失败: %v", err)
		}

		// 应该立即返回
		if elapsed > 500*time.Millisecond {
			t.Errorf("等待时间过长: %v", elapsed)
		}
	})

	t.Run("DelayedSuccess", func(t *testing.T) {
		counter := 0
		condition := func() (bool, error) {
			counter++
			return counter >= 3, nil
		}

		opts := &WaitOptions{
			Timeout:  5 * time.Second,
			Interval: 100 * time.Millisecond,
		}

		err := WaitForCondition(ctx, condition, opts)
		if err != nil {
			t.Errorf("等待失败: %v", err)
		}

		if counter < 3 {
			t.Errorf("检查次数不足: got %d, want >= 3", counter)
		}
	})

	t.Run("Timeout", func(t *testing.T) {
		condition := func() (bool, error) {
			return false, nil // 永不满足
		}

		opts := &WaitOptions{
			Timeout:  500 * time.Millisecond,
			Interval: 100 * time.Millisecond,
		}

		err := WaitForCondition(ctx, condition, opts)
		if err == nil {
			t.Error("应该超时")
		}
	})
}

// TestRetryWithBackoff 测试重试机制
func TestRetryWithBackoff(t *testing.T) {
	ctx := context.Background()

	t.Run("Success", func(t *testing.T) {
		attempts := 0
		fn := func() error {
			attempts++
			if attempts >= 3 {
				return nil
			}
			return fmt.Errorf("临时错误")
		}

		err := RetryWithBackoff(ctx, 5, 10*time.Millisecond, fn)
		if err != nil {
			t.Errorf("重试失败: %v", err)
		}

		if attempts != 3 {
			t.Errorf("重试次数错误: got %d, want 3", attempts)
		}
	})

	t.Run("MaxAttemptsExceeded", func(t *testing.T) {
		attempts := 0
		fn := func() error {
			attempts++
			return fmt.Errorf("持续错误")
		}

		err := RetryWithBackoff(ctx, 3, 10*time.Millisecond, fn)
		if err == nil {
			t.Error("应该返回错误")
		}

		if attempts != 3 {
			t.Errorf("重试次数错误: got %d, want 3", attempts)
		}
	})
}

// BenchmarkBrowserPoolAcquire 基准测试池获取
func BenchmarkBrowserPoolAcquire(b *testing.B) {
	ctx := context.Background()
	opts := &ConnectOptions{
		Headless: true,
		Args:     []string{"--disable-gpu", "--no-sandbox"},
	}

	pool := NewBrowserPool(10, opts)
	defer pool.Close()

	// 预热
	pool.Warmup(ctx, 5)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		instance, _ := pool.Acquire(ctx)
		pool.Release(instance)
	}
}

// BenchmarkWaitWithContext 基准测试智能等待
func BenchmarkWaitWithContext(b *testing.B) {
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		WaitWithContext(ctx, 10*time.Millisecond)
	}
}
