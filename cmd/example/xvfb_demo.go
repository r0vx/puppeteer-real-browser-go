package main

import (
	"context"
	"fmt"
	"log"
	"runtime"
	"time"

	"github.com/HNRow/puppeteer-real-browser-go/pkg/browser"
)

func main() {
	fmt.Println("🖥️  Xvfb 虚拟显示演示")
	fmt.Println("==================")
	fmt.Println()

	// 检查平台
	fmt.Printf("当前平台: %s/%s\n", runtime.GOOS, runtime.GOARCH)
	
	if runtime.GOOS != "linux" {
		fmt.Println("⚠️  Xvfb 只在 Linux 平台上需要")
		fmt.Println("其他平台会自动使用系统显示")
		fmt.Println()
	}

	ctx := context.Background()

	// 场景 1: 默认配置（自动管理 Xvfb）
	fmt.Println("📝 场景 1: 默认配置 - 自动管理 Xvfb")
	fmt.Println("-------------------------------------")
	testDefaultConfig(ctx)

	// 场景 2: 禁用 Xvfb
	fmt.Println("\n📝 场景 2: 明确禁用 Xvfb")
	fmt.Println("-------------------------------------")
	testDisabledXvfb(ctx)

	// 场景 3: headless 模式（不需要 Xvfb）
	fmt.Println("\n📝 场景 3: Headless 模式（不需要 Xvfb）")
	fmt.Println("-------------------------------------")
	testHeadlessMode(ctx)

	fmt.Println("\n✅ 所有场景测试完成！")
}

// testDefaultConfig 测试默认配置（自动管理 Xvfb）
func testDefaultConfig(ctx context.Context) {
	opts := &browser.ConnectOptions{
		Headless:    false, // 可见模式
		DisableXvfb: false, // 不禁用 Xvfb（默认值）
		Args: []string{
			"--start-maximized",
		},
	}

	fmt.Println("🚀 启动浏览器（自动管理 Xvfb）...")
	
	if runtime.GOOS == "linux" {
		// 检查 Xvfb 是否安装
		if browser.IsXvfbInstalled() {
			fmt.Println("✅ Xvfb 已安装")
		} else {
			fmt.Println("⚠️  Xvfb 未安装")
			fmt.Println("安装命令:", browser.GetXvfbInstallCommand())
		}
	}

	instance, err := browser.Connect(ctx, opts)
	if err != nil {
		log.Printf("❌ 启动失败: %v", err)
		return
	}
	defer instance.Close()

	page := instance.Page()

	// 导航测试
	fmt.Println("📂 导航到 Google...")
	if err := page.Navigate("https://www.google.com"); err != nil {
		log.Printf("❌ 导航失败: %v", err)
		return
	}

	time.Sleep(2 * time.Second)

	// 获取标题
	title, err := page.GetTitle()
	if err != nil {
		log.Printf("⚠️  获取标题失败: %v", err)
	} else {
		fmt.Printf("✅ 页面标题: %s\n", title)
	}

	// 截图验证
	screenshot, err := page.Screenshot()
	if err != nil {
		log.Printf("⚠️  截图失败: %v", err)
	} else {
		fmt.Printf("✅ 截图成功: %d bytes\n", len(screenshot))
	}

	fmt.Println("⏳ 保持运行 3 秒...")
	time.Sleep(3 * time.Second)
}

// testDisabledXvfb 测试禁用 Xvfb
func testDisabledXvfb(ctx context.Context) {
	opts := &browser.ConnectOptions{
		Headless:    false, // 可见模式
		DisableXvfb: true,  // 明确禁用 Xvfb
		Args: []string{
			"--start-maximized",
		},
	}

	fmt.Println("🚀 启动浏览器（禁用 Xvfb）...")
	fmt.Println("ℹ️  如果没有图形界面，可能会失败")

	instance, err := browser.Connect(ctx, opts)
	if err != nil {
		log.Printf("❌ 启动失败（预期行为）: %v", err)
		return
	}
	defer instance.Close()

	page := instance.Page()

	fmt.Println("📂 导航到 Example.com...")
	if err := page.Navigate("https://example.com"); err != nil {
		log.Printf("❌ 导航失败: %v", err)
		return
	}

	time.Sleep(2 * time.Second)

	title, err := page.GetTitle()
	if err != nil {
		log.Printf("⚠️  获取标题失败: %v", err)
	} else {
		fmt.Printf("✅ 页面标题: %s\n", title)
	}

	fmt.Println("⏳ 保持运行 3 秒...")
	time.Sleep(3 * time.Second)
}

// testHeadlessMode 测试 headless 模式
func testHeadlessMode(ctx context.Context) {
	opts := &browser.ConnectOptions{
		Headless: true, // headless 模式（不需要 Xvfb）
		Args: []string{
			"--disable-gpu",
		},
	}

	fmt.Println("🚀 启动浏览器（Headless 模式）...")
	fmt.Println("ℹ️  Headless 模式不需要显示服务器")

	instance, err := browser.Connect(ctx, opts)
	if err != nil {
		log.Printf("❌ 启动失败: %v", err)
		return
	}
	defer instance.Close()

	page := instance.Page()

	fmt.Println("📂 导航到 GitHub...")
	if err := page.Navigate("https://github.com"); err != nil {
		log.Printf("❌ 导航失败: %v", err)
		return
	}

	time.Sleep(2 * time.Second)

	title, err := page.GetTitle()
	if err != nil {
		log.Printf("⚠️  获取标题失败: %v", err)
	} else {
		fmt.Printf("✅ 页面标题: %s\n", title)
	}

	// Headless 模式特别适合截图
	screenshot, err := page.Screenshot()
	if err != nil {
		log.Printf("⚠️  截图失败: %v", err)
	} else {
		fmt.Printf("✅ Headless 截图成功: %d bytes\n", len(screenshot))
	}

	fmt.Println("⏳ 保持运行 3 秒...")
	time.Sleep(3 * time.Second)
}

// PrintXvfbInfo 打印 Xvfb 相关信息（辅助函数）
func PrintXvfbInfo() {
	fmt.Println("\n📘 Xvfb 使用指南")
	fmt.Println("================")
	fmt.Println()
	fmt.Println("什么是 Xvfb?")
	fmt.Println("  Xvfb (X Virtual Frame Buffer) 是一个虚拟显示服务器")
	fmt.Println("  允许在没有物理显示器的 Linux 服务器上运行图形程序")
	fmt.Println()
	fmt.Println("何时需要 Xvfb?")
	fmt.Println("  ✅ Linux 服务器")
	fmt.Println("  ✅ headless: false（需要可见浏览器）")
	fmt.Println("  ✅ 没有 DISPLAY 环境变量")
	fmt.Println()
	fmt.Println("何时不需要 Xvfb?")
	fmt.Println("  ❌ macOS/Windows（使用系统显示）")
	fmt.Println("  ❌ headless: true（无头模式）")
	fmt.Println("  ❌ 已有图形界面（DISPLAY 已设置）")
	fmt.Println()
	
	if runtime.GOOS == "linux" {
		fmt.Println("安装 Xvfb:")
		fmt.Printf("  %s\n", browser.GetXvfbInstallCommand())
		fmt.Println()
		
		if browser.IsXvfbInstalled() {
			fmt.Println("✅ Xvfb 已安装在您的系统上")
		} else {
			fmt.Println("⚠️  Xvfb 未安装")
		}
	}
}

func init() {
	// 程序启动时打印 Xvfb 信息
	PrintXvfbInfo()
}

