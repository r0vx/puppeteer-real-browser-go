package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"runtime"
	"sync"
	"sync/atomic"
	"time"

	"github.com/r0vx/puppeteer-real-browser-go/pkg/browser"
)

// ResourceMonitor 监控系统资源使用
type ResourceMonitor struct {
	pool            *browser.BrowserPool
	stats           *MonitorStats
	stopChan        chan struct{}
	interval        time.Duration
	startTime       time.Time
	mu              sync.RWMutex
}

// MonitorStats 监控统计数据
type MonitorStats struct {
	// 时间戳
	Timestamp time.Time `json:"timestamp"`
	
	// 运行时间
	Uptime time.Duration `json:"uptime"`
	
	// Go 运行时统计
	NumGoroutine   int     `json:"num_goroutine"`
	MemoryAllocMB  float64 `json:"memory_alloc_mb"`
	MemorySysMB    float64 `json:"memory_sys_mb"`
	MemoryUsagePC  float64 `json:"memory_usage_percent"`
	NumGC          uint32  `json:"num_gc"`
	
	// 浏览器池统计
	PoolAvailable  int `json:"pool_available"`
	PoolCreated    int `json:"pool_created"`
	PoolMaxSize    int `json:"pool_max_size"`
	PoolUsagePC    float64 `json:"pool_usage_percent"`
	
	// Chrome 进程统计
	ChromeProcesses int     `json:"chrome_processes"`
	
	// 请求统计（如果有）
	TotalRequests   int64   `json:"total_requests"`
	SuccessRequests int64   `json:"success_requests"`
	FailedRequests  int64   `json:"failed_requests"`
	SuccessRate     float64 `json:"success_rate"`
}

// RequestTracker 请求追踪器
type RequestTracker struct {
	total   atomic.Int64
	success atomic.Int64
	failed  atomic.Int64
}

var globalTracker = &RequestTracker{}

func NewResourceMonitor(pool *browser.BrowserPool, interval time.Duration) *ResourceMonitor {
	return &ResourceMonitor{
		pool:      pool,
		stats:     &MonitorStats{},
		stopChan:  make(chan struct{}),
		interval:  interval,
		startTime: time.Now(),
	}
}

// Start 开始监控
func (rm *ResourceMonitor) Start() {
	go rm.monitorLoop()
}

// Stop 停止监控
func (rm *ResourceMonitor) Stop() {
	close(rm.stopChan)
}

// monitorLoop 监控循环
func (rm *ResourceMonitor) monitorLoop() {
	ticker := time.NewTicker(rm.interval)
	defer ticker.Stop()
	
	for {
		select {
		case <-ticker.C:
			rm.collectStats()
			rm.displayStats()
		case <-rm.stopChan:
			return
		}
	}
}

// collectStats 收集统计信息
func (rm *ResourceMonitor) collectStats() {
	rm.mu.Lock()
	defer rm.mu.Unlock()
	
	// 时间统计
	rm.stats.Timestamp = time.Now()
	rm.stats.Uptime = time.Since(rm.startTime)
	
	// Go 运行时统计
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	
	rm.stats.NumGoroutine = runtime.NumGoroutine()
	rm.stats.MemoryAllocMB = float64(m.Alloc) / 1024 / 1024
	rm.stats.MemorySysMB = float64(m.Sys) / 1024 / 1024
	rm.stats.MemoryUsagePC = (float64(m.Alloc) / float64(m.Sys)) * 100
	rm.stats.NumGC = m.NumGC
	
	// 浏览器池统计
	if rm.pool != nil {
		poolStats := rm.pool.Stats()
		rm.stats.PoolAvailable = poolStats.Available
		rm.stats.PoolCreated = poolStats.Created
		rm.stats.PoolMaxSize = poolStats.MaxSize
		
		if poolStats.MaxSize > 0 {
			rm.stats.PoolUsagePC = float64(poolStats.Created-poolStats.Available) / float64(poolStats.MaxSize) * 100
		}
	}
	
	// Chrome 进程数量
	rm.stats.ChromeProcesses = rm.countChromeProcesses()
	
	// 请求统计
	rm.stats.TotalRequests = globalTracker.total.Load()
	rm.stats.SuccessRequests = globalTracker.success.Load()
	rm.stats.FailedRequests = globalTracker.failed.Load()
	
	if rm.stats.TotalRequests > 0 {
		rm.stats.SuccessRate = float64(rm.stats.SuccessRequests) / float64(rm.stats.TotalRequests) * 100
	}
}

// countChromeProcesses 统计 Chrome 进程数量
func (rm *ResourceMonitor) countChromeProcesses() int {
	// 简单实现：只统计池中的实例
	if rm.pool != nil {
		return rm.pool.Stats().Created
	}
	return 0
}

// displayStats 显示统计信息
func (rm *ResourceMonitor) displayStats() {
	rm.mu.RLock()
	defer rm.mu.RUnlock()
	
	// 清屏（在终端中）
	fmt.Print("\033[2J\033[H")
	
	fmt.Println("╔════════════════════════════════════════════════════════════════╗")
	fmt.Println("║          🔍 Puppeteer Real Browser - 资源监控面板              ║")
	fmt.Println("╚════════════════════════════════════════════════════════════════╝")
	
	fmt.Printf("\n⏰ 时间: %s\n", rm.stats.Timestamp.Format("2006-01-02 15:04:05"))
	fmt.Printf("⏱️  运行时长: %s\n", formatDuration(rm.stats.Uptime))
	
	fmt.Println("\n" + "─" + "─" + "─" + " Go 运行时 " + "─" + "─" + "─" + "─" + "─" + "─" + "─" + "─" + "─" + "─" + "─" + "─" + "─" + "─" + "─" + "─" + "─" + "─" + "─" + "─" + "─" + "─" + "─" + "─" + "─" + "─" + "─" + "─" + "─" + "─" + "─" + "─" + "─")
	
	fmt.Printf("🔢 Goroutine 数量: %s%d%s\n", 
		colorByGoroutines(rm.stats.NumGoroutine), 
		rm.stats.NumGoroutine, 
		"\033[0m")
	
	fmt.Printf("💾 内存分配: %s%.2f MB%s / %.2f MB (%.1f%%)\n",
		colorByMemory(rm.stats.MemoryUsagePC),
		rm.stats.MemoryAllocMB,
		"\033[0m",
		rm.stats.MemorySysMB,
		rm.stats.MemoryUsagePC)
	
	fmt.Printf("🗑️  GC 次数: %d\n", rm.stats.NumGC)
	
	fmt.Println("\n" + "─" + "─" + "─" + " 浏览器池 " + "─" + "─" + "─" + "─" + "─" + "─" + "─" + "─" + "─" + "─" + "─" + "─" + "─" + "─" + "─" + "─" + "─" + "─" + "─" + "─" + "─" + "─" + "─" + "─" + "─" + "─" + "─" + "─" + "─" + "─" + "─" + "─" + "─")
	
	fmt.Printf("📦 池容量: %d (最大: %d)\n", 
		rm.stats.PoolCreated, 
		rm.stats.PoolMaxSize)
	
	fmt.Printf("✅ 可用实例: %s%d%s\n",
		colorByAvailable(rm.stats.PoolAvailable, rm.stats.PoolMaxSize),
		rm.stats.PoolAvailable,
		"\033[0m")
	
	fmt.Printf("📊 使用率: %s%.1f%%%s\n",
		colorByUsage(rm.stats.PoolUsagePC),
		rm.stats.PoolUsagePC,
		"\033[0m")
	
	fmt.Printf("🌐 Chrome 进程: %d\n", rm.stats.ChromeProcesses)
	
	if rm.stats.TotalRequests > 0 {
		fmt.Println("\n" + "─" + "─" + "─" + " 请求统计 " + "─" + "─" + "─" + "─" + "─" + "─" + "─" + "─" + "─" + "─" + "─" + "─" + "─" + "─" + "─" + "─" + "─" + "─" + "─" + "─" + "─" + "─" + "─" + "─" + "─" + "─" + "─" + "─" + "─" + "─" + "─" + "─" + "─")
		
		fmt.Printf("📨 总请求: %d\n", rm.stats.TotalRequests)
		fmt.Printf("✅ 成功: %s%d%s\n", 
			"\033[32m", 
			rm.stats.SuccessRequests, 
			"\033[0m")
		fmt.Printf("❌ 失败: %s%d%s\n", 
			"\033[31m", 
			rm.stats.FailedRequests, 
			"\033[0m")
		fmt.Printf("📈 成功率: %s%.2f%%%s\n",
			colorBySuccessRate(rm.stats.SuccessRate),
			rm.stats.SuccessRate,
			"\033[0m")
	}
	
	fmt.Println("\n" + "─" + "─" + "─" + "─" + "─" + "─" + "─" + "─" + "─" + "─" + "─" + "─" + "─" + "─" + "─" + "─" + "─" + "─" + "─" + "─" + "─" + "─" + "─" + "─" + "─" + "─" + "─" + "─" + "─" + "─" + "─" + "─" + "─" + "─" + "─" + "─" + "─" + "─" + "─" + "─" + "─" + "─" + "─" + "─" + "─" + "─" + "─" + "─" + "─" + "─")
	fmt.Println("\n💡 提示: 按 Ctrl+C 停止监控")
}

// GetCurrentStats 获取当前统计（用于导出）
func (rm *ResourceMonitor) GetCurrentStats() *MonitorStats {
	rm.mu.RLock()
	defer rm.mu.RUnlock()
	
	// 返回副本
	stats := *rm.stats
	return &stats
}

// ExportToJSON 导出统计到 JSON 文件
func (rm *ResourceMonitor) ExportToJSON(filename string) error {
	stats := rm.GetCurrentStats()
	
	data, err := json.MarshalIndent(stats, "", "  ")
	if err != nil {
		return err
	}
	
	return os.WriteFile(filename, data, 0644)
}

// 颜色工具函数
func colorByGoroutines(count int) string {
	if count < 100 {
		return "\033[32m" // 绿色
	} else if count < 500 {
		return "\033[33m" // 黄色
	}
	return "\033[31m" // 红色
}

func colorByMemory(usage float64) string {
	if usage < 70 {
		return "\033[32m" // 绿色
	} else if usage < 85 {
		return "\033[33m" // 黄色
	}
	return "\033[31m" // 红色
}

func colorByAvailable(available, maxSize int) string {
	if available > maxSize/2 {
		return "\033[32m" // 绿色
	} else if available > 0 {
		return "\033[33m" // 黄色
	}
	return "\033[31m" // 红色
}

func colorByUsage(usage float64) string {
	if usage < 50 {
		return "\033[32m" // 绿色
	} else if usage < 80 {
		return "\033[33m" // 黄色
	}
	return "\033[31m" // 红色
}

func colorBySuccessRate(rate float64) string {
	if rate >= 95 {
		return "\033[32m" // 绿色
	} else if rate >= 80 {
		return "\033[33m" // 黄色
	}
	return "\033[31m" // 红色
}

func formatDuration(d time.Duration) string {
	hours := int(d.Hours())
	minutes := int(d.Minutes()) % 60
	seconds := int(d.Seconds()) % 60
	
	if hours > 0 {
		return fmt.Sprintf("%dh %dm %ds", hours, minutes, seconds)
	} else if minutes > 0 {
		return fmt.Sprintf("%dm %ds", minutes, seconds)
	}
	return fmt.Sprintf("%ds", seconds)
}

// TrackedRequest 包装请求并追踪统计
func TrackedRequest(pool *browser.BrowserPool, ctx context.Context, fn func(*browser.BrowserInstance) error) error {
	globalTracker.total.Add(1)
	
	err := pool.WithPooledBrowser(ctx, fn)
	
	if err != nil {
		globalTracker.failed.Add(1)
	} else {
		globalTracker.success.Add(1)
	}
	
	return err
}

// 示例程序
func main() {
	fmt.Println("🚀 启动资源监控系统...")
	
	ctx := context.Background()
	
	// 创建浏览器池
	pool := browser.NewBrowserPool(10, &browser.ConnectOptions{
		Headless: true,
		Args: []string{
			"--disable-gpu",
			"--no-sandbox",
		},
	})
	defer pool.Close()
	
	// 预热池
	fmt.Println("🔥 预热浏览器池...")
	if err := pool.Warmup(ctx, 5); err != nil {
		fmt.Printf("⚠️  预热警告: %v\n", err)
	}
	
	// 创建监控器
	monitor := NewResourceMonitor(pool, 2*time.Second)
	monitor.Start()
	defer monitor.Stop()
	
	fmt.Println("✅ 监控已启动，开始模拟请求...\n")
	time.Sleep(2 * time.Second)
	
	// 模拟并发请求
	var wg sync.WaitGroup
	
	// 持续运行
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()
			
			err := TrackedRequest(pool, ctx, func(instance *browser.BrowserInstance) error {
				page := instance.Page()
				
				// 随机访问不同网站
				urls := []string{
					"https://example.com",
					"https://httpbin.org/delay/1",
					"https://www.google.com",
				}
				url := urls[index%len(urls)]
				
				return page.Navigate(url)
			})
			
			if err != nil {
				// 错误已被 TrackedRequest 记录
			}
			
			// 模拟间隔
			time.Sleep(time.Duration(500+index*10) * time.Millisecond)
		}(i)
		
		// 控制并发数
		if i%10 == 9 {
			time.Sleep(5 * time.Second)
		}
	}
	
	fmt.Println("\n⏳ 等待所有请求完成...")
	wg.Wait()
	
	// 等待一段时间查看最终状态
	fmt.Println("\n✅ 所有请求完成，继续监控 30 秒...")
	time.Sleep(30 * time.Second)
	
	// 导出统计
	filename := fmt.Sprintf("monitor-stats-%s.json", time.Now().Format("20060102-150405"))
	if err := monitor.ExportToJSON(filename); err != nil {
		fmt.Printf("⚠️  导出失败: %v\n", err)
	} else {
		fmt.Printf("\n📊 统计数据已导出到: %s\n", filename)
	}
	
	fmt.Println("\n🎉 监控完成！")
}

