package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"runtime"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/r0vx/puppeteer-real-browser-go/pkg/browser"
)

// StabilityTest 稳定性测试
type StabilityTest struct {
	pool           *browser.BrowserPool
	duration       time.Duration
	concurrency    int
	requestDelay   time.Duration
	
	// 统计
	startTime      time.Time
	totalRequests  atomic.Int64
	successCount   atomic.Int64
	failureCount   atomic.Int64
	
	// 内存基线
	baselineMemory uint64
	
	// 停止信号
	stopChan       chan struct{}
}

func NewStabilityTest(poolSize, concurrency int, duration, requestDelay time.Duration) *StabilityTest {
	return &StabilityTest{
		duration:     duration,
		concurrency:  concurrency,
		requestDelay: requestDelay,
		stopChan:     make(chan struct{}),
	}
}

func (st *StabilityTest) Run() error {
	fmt.Println("╔════════════════════════════════════════════════════════════════╗")
	fmt.Println("║              🧪 浏览器池稳定性测试                              ║")
	fmt.Println("╚════════════════════════════════════════════════════════════════╝")
	
	ctx := context.Background()
	
	// 创建浏览器池
	fmt.Println("\n📦 创建浏览器池...")
	poolSize := 10
	st.pool = browser.NewBrowserPool(poolSize, &browser.ConnectOptions{
		Headless: true,
		Args: []string{
			"--disable-gpu",
			"--no-sandbox",
			"--disable-dev-shm-usage",
		},
	})
	defer st.pool.Close()
	
	// 预热
	fmt.Printf("🔥 预热浏览器池 (目标: %d 个实例)...\n", poolSize/2)
	if err := st.pool.Warmup(ctx, poolSize/2); err != nil {
		log.Printf("⚠️  预热警告: %v", err)
	}
	
	// 记录基线内存
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	st.baselineMemory = m.Alloc
	
	fmt.Printf("\n📊 测试配置:\n")
	fmt.Printf("   - 池大小: %d\n", poolSize)
	fmt.Printf("   - 并发数: %d\n", st.concurrency)
	fmt.Printf("   - 测试时长: %s\n", st.duration)
	fmt.Printf("   - 请求间隔: %s\n", st.requestDelay)
	fmt.Printf("   - 基线内存: %.2f MB\n", float64(st.baselineMemory)/1024/1024)
	
	fmt.Println("\n⏳ 测试开始...")
	fmt.Println("─────────────────────────────────────────────────────────────────")
	
	st.startTime = time.Now()
	
	// 启动监控协程
	go st.monitor()
	
	// 启动工作协程
	var wg sync.WaitGroup
	for i := 0; i < st.concurrency; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			st.worker(ctx, workerID)
		}(i)
	}
	
	// 等待测试完成或中断
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	
	select {
	case <-time.After(st.duration):
		fmt.Println("\n⏰ 测试时间到达")
	case <-sigChan:
		fmt.Println("\n🛑 收到中断信号")
	}
	
	// 停止所有工作
	close(st.stopChan)
	
	fmt.Println("\n⏳ 等待所有工作协程完成...")
	wg.Wait()
	
	// 显示最终报告
	st.printReport()
	
	return nil
}

func (st *StabilityTest) worker(ctx context.Context, workerID int) {
	urls := []string{
		"https://example.com",
		"https://httpbin.org/html",
		"https://www.google.com",
	}
	
	for {
		select {
		case <-st.stopChan:
			return
		default:
		}
		
		st.totalRequests.Add(1)
		requestID := st.totalRequests.Load()
		
		url := urls[int(requestID)%len(urls)]
		
		err := st.pool.WithPooledBrowser(ctx, func(instance *browser.BrowserInstance) error {
			page := instance.Page()
			return page.Navigate(url)
		})
		
		if err != nil {
			st.failureCount.Add(1)
			log.Printf("❌ Worker %d 请求失败 #%d: %v", workerID, requestID, err)
		} else {
			st.successCount.Add(1)
		}
		
		// 请求间隔
		time.Sleep(st.requestDelay)
	}
}

func (st *StabilityTest) monitor() {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()
	
	lastTotal := int64(0)
	
	for {
		select {
		case <-ticker.C:
			elapsed := time.Since(st.startTime)
			total := st.totalRequests.Load()
			success := st.successCount.Load()
			failed := st.failureCount.Load()
			
			// 计算速率
			deltaTotal := total - lastTotal
			
			lastTotal = total
			
			// 获取内存统计
			var m runtime.MemStats
			runtime.ReadMemStats(&m)
			
			currentMemory := float64(m.Alloc) / 1024 / 1024
			memoryGrowth := float64(m.Alloc-st.baselineMemory) / 1024 / 1024
			
			// 获取池状态
			poolStats := st.pool.Stats()
			
			// 打印状态
			fmt.Printf("[%s] 📊 请求: %d (✅ %d | ❌ %d) | 速率: %.1f/s | 内存: %.1fMB (+%.1fMB) | Goroutine: %d | 池: %d/%d\n",
				formatDuration(elapsed),
				total,
				success,
				failed,
				float64(deltaTotal)/10.0,
				currentMemory,
				memoryGrowth,
				runtime.NumGoroutine(),
				poolStats.Available,
				poolStats.MaxSize,
			)
			
		case <-st.stopChan:
			return
		}
	}
}

func (st *StabilityTest) printReport() {
	elapsed := time.Since(st.startTime)
	total := st.totalRequests.Load()
	success := st.successCount.Load()
	failed := st.failureCount.Load()
	
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	
	finalMemory := float64(m.Alloc) / 1024 / 1024
	memoryGrowth := float64(m.Alloc-st.baselineMemory) / 1024 / 1024
	memoryGrowthPercent := (float64(m.Alloc-st.baselineMemory) / float64(st.baselineMemory)) * 100
	
	fmt.Println("\n╔════════════════════════════════════════════════════════════════╗")
	fmt.Println("║                    📊 测试报告                                  ║")
	fmt.Println("╚════════════════════════════════════════════════════════════════╝")
	
	fmt.Printf("\n⏱️  运行时长: %s\n", formatDuration(elapsed))
	
	fmt.Println("\n📈 请求统计:")
	fmt.Printf("   - 总请求数: %d\n", total)
	fmt.Printf("   - 成功: %s%d%s (%.2f%%)\n", 
		"\033[32m", success, "\033[0m",
		float64(success)/float64(total)*100)
	fmt.Printf("   - 失败: %s%d%s (%.2f%%)\n", 
		"\033[31m", failed, "\033[0m",
		float64(failed)/float64(total)*100)
	fmt.Printf("   - 平均速率: %.2f req/s\n", float64(total)/elapsed.Seconds())
	
	fmt.Println("\n💾 内存统计:")
	fmt.Printf("   - 基线内存: %.2f MB\n", float64(st.baselineMemory)/1024/1024)
	fmt.Printf("   - 最终内存: %.2f MB\n", finalMemory)
	fmt.Printf("   - 内存增长: %s%.2f MB%s (%s%.1f%%%s)\n",
		colorByMemoryGrowth(memoryGrowthPercent),
		memoryGrowth,
		"\033[0m",
		colorByMemoryGrowth(memoryGrowthPercent),
		memoryGrowthPercent,
		"\033[0m")
	fmt.Printf("   - GC 次数: %d\n", m.NumGC)
	
	fmt.Println("\n🔧 Go 运行时:")
	fmt.Printf("   - Goroutine: %d\n", runtime.NumGoroutine())
	
	poolStats := st.pool.Stats()
	fmt.Println("\n📦 浏览器池:")
	fmt.Printf("   - 可用实例: %d/%d\n", poolStats.Available, poolStats.MaxSize)
	fmt.Printf("   - 总创建数: %d\n", poolStats.Created)
	
	// 评估结果
	fmt.Println("\n🎯 稳定性评估:")
	
	successRate := float64(success) / float64(total) * 100
	
	if successRate >= 95 && memoryGrowthPercent < 20 && runtime.NumGoroutine() < 100 {
		fmt.Println("   ✅ 优秀 - 系统稳定，可用于生产环境")
	} else if successRate >= 80 && memoryGrowthPercent < 50 && runtime.NumGoroutine() < 200 {
		fmt.Println("   🟡 良好 - 系统基本稳定，建议进一步优化")
	} else {
		fmt.Println("   ❌ 需要改进 - 发现稳定性问题，不建议用于生产环境")
	}
	
	if memoryGrowthPercent > 30 {
		fmt.Println("   ⚠️  警告: 内存增长较大，可能存在内存泄漏")
	}
	
	if runtime.NumGoroutine() > 150 {
		fmt.Println("   ⚠️  警告: Goroutine 数量较多，可能存在泄漏")
	}
	
	fmt.Println()
}

func colorByMemoryGrowth(percent float64) string {
	if percent < 10 {
		return "\033[32m" // 绿色
	} else if percent < 30 {
		return "\033[33m" // 黄色
	}
	return "\033[31m" // 红色
}

func formatDuration(d time.Duration) string {
	hours := int(d.Hours())
	minutes := int(d.Minutes()) % 60
	seconds := int(d.Seconds()) % 60
	
	if hours > 0 {
		return fmt.Sprintf("%dh%dm%ds", hours, minutes, seconds)
	} else if minutes > 0 {
		return fmt.Sprintf("%dm%ds", minutes, seconds)
	}
	return fmt.Sprintf("%ds", seconds)
}

func main() {
	// 命令行参数
	duration := flag.Duration("duration", 5*time.Minute, "测试持续时间")
	concurrency := flag.Int("concurrency", 5, "并发工作协程数")
	requestDelay := flag.Duration("delay", 2*time.Second, "请求间隔时间")
	
	flag.Parse()
	
	test := NewStabilityTest(10, *concurrency, *duration, *requestDelay)
	
	if err := test.Run(); err != nil {
		log.Fatal(err)
	}
}

