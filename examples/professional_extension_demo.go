package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/HNRow/puppeteer-real-browser-go/pkg/browser"
)

func main() {
	fmt.Println("🎯 Professional Extension Management Test")
	fmt.Println("========================================")

	ctx := context.Background()

	// 扩展路径
	extensionPaths := []string{
		"./examples/path/Extensions/kfjglmgfjedhhcddpfgfogkahmenikan/1.0_0",
		"./examples/path/Extensions/mcohilncbfahbmgdjkbpemcciiolgcge/3.66.10_0",
	}

	fmt.Printf("📦 目标扩展:\n")
	for i, path := range extensionPaths {
		fmt.Printf("  %d. %s\n", i+1, path)
	}

	// 方法1: 使用预配置Chrome偏好设置
	fmt.Println("\n🔧 方法1: 预配置Chrome偏好设置...")
	opts := &browser.ConnectOptions{
		Headless:       false,
		Extensions:     extensionPaths,
		PersistProfile: true, // 启用持久化配置
		Args: []string{
			"--start-maximized",
			// 扩展开发者模式相关参数
			"--enable-extensions",
			"--disable-extensions-file-access-check",
			"--allow-running-insecure-content",
			"--disable-web-security",
			// 反检测参数
			"--disable-blink-features=AutomationControlled",
			"--exclude-switches=enable-automation",
			"--disable-dev-shm-usage",
		},
	}

	// 创建浏览器实例
	instance, err := browser.Connect(ctx, opts)
	if err != nil {
		log.Fatalf("浏览器启动失败: %v", err)
	}
	defer instance.Close()

	fmt.Println("✅ 浏览器启动成功")

	// 方法2: 使用CDP管理器进行高级操作
	fmt.Println("\n🔧 方法2: 使用CDP扩展管理器...")

	page := instance.Page()
	cdpManager := browser.NewExtensionCDPManager(ctx)

	// 注入反检测脚本
	fmt.Println("  🛡️ 注入反检测脚本...")
	if err := cdpManager.InjectExtensionBypassScript(); err != nil {
		fmt.Printf("    ❌ 反检测脚本注入失败: %v\n", err)
	} else {
		fmt.Println("    ✅ 反检测脚本注入成功")
	}

	// 安装扩展
	fmt.Println("  📦 安装未打包扩展...")
	if err := cdpManager.InstallUnpackedExtensions(extensionPaths); err != nil {
		fmt.Printf("    ❌ CDP扩展安装失败: %v\n", err)
	} else {
		fmt.Println("    ✅ CDP扩展安装完成")
	}

	// 等待扩展加载
	time.Sleep(3 * time.Second)

	// 检查扩展状态
	fmt.Println("\n🔍 检查扩展安装状态...")
	extensions, err := cdpManager.GetLoadedExtensions()
	if err != nil {
		fmt.Printf("❌ 获取扩展列表失败: %v\n", err)
	} else {
		fmt.Printf("📊 已加载扩展: %d 个\n", len(extensions))
		for i, ext := range extensions {
			status := "❌ 已禁用"
			if ext["enabled"].(bool) {
				status = "✅ 已启用"
			}
			fmt.Printf("  %d. %s (v%s) - %s\n",
				i+1, ext["name"], ext["version"], status)
		}
	}

	// 导航到扩展页面进行最终验证
	fmt.Println("\n🔍 最终验证...")
	if err := page.Navigate("chrome://extensions/"); err != nil {
		fmt.Printf("❌ 无法访问扩展页面: %v\n", err)
	} else {
		fmt.Println("✅ 扩展页面已打开")
	}

	// 专业级验证
	fmt.Println("\n" + "="*60)
	fmt.Println("🎯 专业验证清单:")
	fmt.Println("=" * 60)
	fmt.Println("1. 开发者模式是否自动启用？")
	fmt.Println("2. 扩展是否出现在列表中？")
	fmt.Println("3. 扩展状态是否为'已启用'？")
	fmt.Println("4. 浏览器地址栏是否显示扩展图标？")
	fmt.Println("5. 扩展功能是否正常工作？")
	fmt.Println("=" * 60)

	fmt.Println("\n💡 如果以上都成功，说明专业方案有效！")
	fmt.Println("💡 如果部分成功，我们可以进一步优化配置")

	fmt.Println("\n⏳ 保持浏览器打开 10 秒供验证...")
	time.Sleep(10 * time.Second)

	fmt.Println("✅ 专业测试完成")
}
