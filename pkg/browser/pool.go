package browser

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"
)

// BrowserPool 管理浏览器实例池，复用实例以提升性能
// 相比每次创建新实例，可以节省 70-80% 的启动时间
type BrowserPool struct {
	instances chan *BrowserInstance
	opts      *ConnectOptions
	mu        sync.RWMutex
	maxSize   int
	created   int32 // 原子计数器，跟踪创建的实例数
	closed    bool
}

// NewBrowserPool 创建新的浏览器实例池
// maxSize: 池中最多保留的实例数量
// opts: 浏览器连接选项
func NewBrowserPool(maxSize int, opts *ConnectOptions) *BrowserPool {
	if maxSize <= 0 {
		maxSize = 5 // 默认池大小
	}

	return &BrowserPool{
		instances: make(chan *BrowserInstance, maxSize),
		opts:      opts,
		maxSize:   maxSize,
		closed:    false,
	}
}

// Acquire 从池中获取一个浏览器实例
// 如果池中没有可用实例，会创建一个新的
func (p *BrowserPool) Acquire(ctx context.Context) (*BrowserInstance, error) {
	p.mu.RLock()
	if p.closed {
		p.mu.RUnlock()
		return nil, fmt.Errorf("pool is closed")
	}
	p.mu.RUnlock()

	// 尝试从池中获取实例（非阻塞）
	select {
	case instance := <-p.instances:
		// 验证实例是否仍然有效
		if instance.Chrome() != nil && instance.Chrome().IsRunning() {
			return instance, nil
		}
		// 实例已失效，关闭并创建新的
		instance.Close()
	default:
		// 池中没有可用实例
	}

	// 创建新实例
	instance, err := Connect(ctx, p.opts)
	if err != nil {
		return nil, fmt.Errorf("failed to create browser instance: %w", err)
	}

	atomic.AddInt32(&p.created, 1)
	return instance, nil
}

// Release 将实例归还到池中
// 如果池已满或实例失效，会关闭实例
func (p *BrowserPool) Release(instance *BrowserInstance) error {
	if instance == nil {
		return fmt.Errorf("cannot release nil instance")
	}

	p.mu.RLock()
	closed := p.closed
	p.mu.RUnlock()

	if closed {
		return instance.Close()
	}

	// 检查实例是否仍然有效
	if instance.Chrome() == nil || !instance.Chrome().IsRunning() {
		return instance.Close()
	}

	// 尝试放回池中（非阻塞）
	select {
	case p.instances <- instance:
		// 成功放回池中
		return nil
	default:
		// 池已满，关闭实例
		return instance.Close()
	}
}

// Close 关闭池和所有实例
func (p *BrowserPool) Close() error {
	p.mu.Lock()
	if p.closed {
		p.mu.Unlock()
		return nil
	}
	p.closed = true
	p.mu.Unlock()

	// 关闭通道
	close(p.instances)

	// 关闭池中所有实例
	var errs []error
	for instance := range p.instances {
		if err := instance.Close(); err != nil {
			errs = append(errs, err)
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("errors closing instances: %v", errs)
	}

	return nil
}

// Size 返回池中当前可用实例数量
func (p *BrowserPool) Size() int {
	return len(p.instances)
}

// Created 返回已创建的实例总数
func (p *BrowserPool) Created() int {
	return int(atomic.LoadInt32(&p.created))
}

// MaxSize 返回池的最大容量
func (p *BrowserPool) MaxSize() int {
	return p.maxSize
}

// Warmup 预热池，创建指定数量的实例
func (p *BrowserPool) Warmup(ctx context.Context, count int) error {
	if count > p.maxSize {
		count = p.maxSize
	}

	var wg sync.WaitGroup
	errChan := make(chan error, count)

	for i := 0; i < count; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			instance, err := Connect(ctx, p.opts)
			if err != nil {
				errChan <- err
				return
			}

			atomic.AddInt32(&p.created, 1)

			// 尝试放入池中
			select {
			case p.instances <- instance:
				// 成功
			default:
				// 池已满，不应该发生
				instance.Close()
			}
		}()
	}

	wg.Wait()
	close(errChan)

	// 收集错误
	var errs []error
	for err := range errChan {
		errs = append(errs, err)
	}

	if len(errs) > 0 {
		return fmt.Errorf("warmup errors: %v", errs)
	}

	return nil
}

// PoolStats 池的统计信息
type PoolStats struct {
	Available int   // 可用实例数
	Created   int   // 已创建实例总数
	MaxSize   int   // 最大容量
	Closed    bool  // 是否已关闭
}

// Stats 获取池的统计信息
func (p *BrowserPool) Stats() PoolStats {
	p.mu.RLock()
	defer p.mu.RUnlock()

	return PoolStats{
		Available: len(p.instances),
		Created:   int(atomic.LoadInt32(&p.created)),
		MaxSize:   p.maxSize,
		Closed:    p.closed,
	}
}

// WithPooledBrowser 提供便捷的函数式 API
// 自动获取和释放实例
func (p *BrowserPool) WithPooledBrowser(ctx context.Context, fn func(*BrowserInstance) error) error {
	instance, err := p.Acquire(ctx)
	if err != nil {
		return err
	}
	defer p.Release(instance)

	return fn(instance)
}

// WithPooledBrowserTimeout 带超时的函数式 API
func (p *BrowserPool) WithPooledBrowserTimeout(ctx context.Context, timeout time.Duration, fn func(*BrowserInstance) error) error {
	instance, err := p.Acquire(ctx)
	if err != nil {
		return err
	}
	defer p.Release(instance)

	// 创建带超时的 context
	ctxWithTimeout, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	// 创建错误通道
	errChan := make(chan error, 1)

	// 在 goroutine 中执行函数
	go func() {
		errChan <- fn(instance)
	}()

	// 等待完成或超时
	select {
	case err := <-errChan:
		return err
	case <-ctxWithTimeout.Done():
		return fmt.Errorf("operation timeout after %v", timeout)
	}
}
