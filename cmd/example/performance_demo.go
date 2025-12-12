package main

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/r0vx/puppeteer-real-browser-go/pkg/browser"
)

func main() {
	fmt.Println("🧪 性能优化测试")
	fmt.Println("========================================")

	ctx := context.Background()

	// 测试 1: 启动速度对比
	fmt.Println("\n📊 测试 1: 浏览器启动速度对比")
	fmt.Println("----------------------------------------")
	testStartupPerformance(ctx)

	// 测试 2: 池的基本功能
	fmt.Println("\n📊 测试 2: 浏览器池功能测试")
	fmt.Println("----------------------------------------")
	testPoolFunctionality(ctx)

	// 测试 3: 并发性能
	fmt.Println("\n📊 测试 3: 并发性能测试")
	fmt.Println("----------------------------------------")
	testConcurrentPerformance(ctx)

	fmt.Println("\n✅ 所有测试完成！")
}

// testStartupPerformance 测试启动性能
func testStartupPerformance(ctx context.Context) {
	opts := &browser.ConnectOptions{
		Headless: true,
		Args: []string{
			"--disable-gpu",
			"--no-sandbox",
		},
	}

	// 方式 1: 直接启动（无池）
	fmt.Println("   方式 1: 直接启动浏览器...")
	start := time.Now()
	instance1, err := browser.Connect(ctx, opts)
	if err != nil {
		log.Printf("启动失败: %v", err)
		return
	}
	directTime := time.Since(start)
	fmt.Printf("   ⏱️  耗时: %v\n", directTime)
	instance1.Close()

	// 方式 2: 使用池
	fmt.Println("\n   方式 2: 使用浏览器池...")
	pool := browser.NewBrowserPool(5, opts)
	defer pool.Close()

	// 预热
	fmt.Println("   预热池（创建 3 个实例）...")
	warmupStart := time.Now()
	if err := pool.Warmup(ctx, 3); err != nil {
		log.Printf("预热失败: %v", err)
		return
	}
	fmt.Printf("   ⏱️  预热耗时: %v\n", time.Since(warmupStart))

	// 从池获取
	fmt.Println("\n   从池中获取实例...")
	start = time.Now()
	instance2, err := pool.Acquire(ctx)
	if err != nil {
		log.Printf("获取失败: %v", err)
		return
	}
	poolTime := time.Since(start)
	fmt.Printf("   ⏱️  耗时: %v\n", poolTime)
	pool.Release(instance2)

	// 计算提升
	if poolTime > 0 && directTime > poolTime {
		improvement := float64(directTime-poolTime) / float64(directTime) * 100
		fmt.Printf("\n   🎯 性能提升: %.1f%%\n", improvement)
	}
}

// testPoolFunctionality 测试池功能
func testPoolFunctionality(ctx context.Context) {
	opts := &browser.ConnectOptions{
		Headless: true,
		Args: []string{
			"--disable-gpu",
			"--no-sandbox",
		},
	}

	pool := browser.NewBrowserPool(5, opts)
	defer pool.Close()

	// 预热
	fmt.Println("   预热池...")
	if err := pool.Warmup(ctx, 3); err != nil {
		log.Printf("预热失败: %v", err)
		return
	}

	// 显示统计
	stats := pool.Stats()
	fmt.Printf("   📊 池状态:\n")
	fmt.Printf("      - 可用实例: %d\n", stats.Available)
	fmt.Printf("      - 已创建总数: %d\n", stats.Created)
	fmt.Printf("      - 最大容量: %d\n", stats.MaxSize)

	// 测试函数式 API
	fmt.Println("\n   测试函数式 API...")
	start := time.Now()
	err := pool.WithPooledBrowser(ctx, func(instance *browser.BrowserInstance) error {
		page := instance.Page()
		return page.Navigate("https://example.com")
	})
	elapsed := time.Since(start)

	if err != nil {
		fmt.Printf("   ❌ 访问失败: %v\n", err)
	} else {
		fmt.Printf("   ✅ 访问成功，耗时: %v\n", elapsed)
	}

	// 再次显示统计
	stats = pool.Stats()
	fmt.Printf("\n   📊 使用后状态:\n")
	fmt.Printf("      - 可用实例: %d\n", stats.Available)
	fmt.Printf("      - 已创建总数: %d\n", stats.Created)
}

// testConcurrentPerformance 测试并发性能
func testConcurrentPerformance(ctx context.Context) {
	opts := &browser.ConnectOptions{
		Headless: true,
		Args: []string{
			"--disable-gpu",
			"--no-sandbox",
		},
	}

	pool := browser.NewBrowserPool(10, opts)
	defer pool.Close()

	// 预热
	fmt.Println("   预热池...")
	pool.Warmup(ctx, 5)

	concurrency := 20
	fmt.Printf("\n   并发执行 %d 次访问...\n", concurrency)

	start := time.Now()
	var wg sync.WaitGroup
	successCount := 0
	var mu sync.Mutex

	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()

			err := pool.WithPooledBrowser(ctx, func(instance *browser.BrowserInstance) error {
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
	totalTime := time.Since(start)

	fmt.Printf("\n   📊 并发测试结果:\n")
	fmt.Printf("      - 成功: %d/%d\n", successCount, concurrency)
	fmt.Printf("      - 总耗时: %v\n", totalTime)
	fmt.Printf("      - 平均每次: %v\n", totalTime/time.Duration(concurrency))
	if totalTime.Seconds() > 0 {
		fmt.Printf("      - 吞吐量: %.1f 次/秒\n", float64(successCount)/totalTime.Seconds())
	}
}
