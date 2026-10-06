package main

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/r0vx/puppeteer-real-browser-go/pkg/browser"
)

func main() {
	fmt.Println("🔧 Chrome启动参数调试")
	fmt.Println("==================")

	ctx := context.Background()

	// 扩展路径
	extensionPaths := []string{
		"../examples/path/Extensions/kfjglmgfjedhhcddpfgfogkahmenikan/1.0_0",
		"../examples/path/Extensions/mcohilncbfahbmgdjkbpemcciiolgcge/3.66.10_0",
	}

	fmt.Println("📂 指定的扩展路径:")
	for i, path := range extensionPaths {
		fmt.Printf("  %d. %s\n", i+1, path)
	}

	// 配置选项
	options := &browser.ConnectOptions{
		Headless:       false,
		Extensions:     extensionPaths,
		PersistProfile: false,
		Args: []string{
			"--start-maximized",
			"--disable-blink-features=AutomationControlled",
		},
	}

	fmt.Println("\n🔧 生成Chrome启动参数...")

	// 创建Chrome启动器并获取启动标志
	launcher := browser.NewChromeLauncher()
	
	// 使用反射或者修改代码来获取标志...
	// 但是由于buildChromeFlags是私有方法，我们需要通过其他方式调试
	
	fmt.Println("\n🚀 尝试启动浏览器以查看实际效果...")
	instance, err := browser.Connect(ctx, options)
	if err != nil {
		log.Fatalf("启动失败: %v", err)
	}
	
	fmt.Println("✅ 浏览器启动成功!")
	fmt.Println("📋 请手动检查:")
	fmt.Println("  1. 浏览器是否打开")
	fmt.Println("  2. 在地址栏输入: chrome://extensions/")
	fmt.Println("  3. 检查扩展是否加载")
	fmt.Println("  4. 如果没有扩展，尝试启用开发者模式")
	
	// 让我们尝试获取用户数据目录位置
	// 这样可以手动检查扩展是否被复制
	
	// 保持浏览器运行
	fmt.Println("\n⏳ 浏览器将保持运行，请手动检查...")
	fmt.Println("按Ctrl+C结束程序")
	
	// 阻塞程序
	select {}
}