package main

import (
	"context"
	"fmt"
	"log"
	"path/filepath"
	"time"

	"github.com/HNRow/puppeteer-real-browser-go/pkg/browser"
)

func main() {
	fmt.Println("🧪 简单扩展测试")
	fmt.Println("==============")

	ctx := context.Background()

	// 获取简单测试扩展的绝对路径
	simpleExt, _ := filepath.Abs("examples/simple_test_extension")
	fmt.Printf("📂 测试扩展: %s\n", simpleExt)

	// 最简配置测试
	opts := &browser.ConnectOptions{
		Headless:       false,
		PersistProfile: true,
		ProfileName:    "simple_extension_test",
		Extensions:     []string{simpleExt},
		Args: []string{
			"--start-maximized",
		},
	}

	fmt.Println("🚀 启动Chrome...")
	instance, err := browser.Connect(ctx, opts)
	if err != nil {
		log.Fatalf("❌ Chrome启动失败: %v", err)
	}
	defer instance.Close()

	fmt.Println("✅ Chrome启动成功")
	time.Sleep(3 * time.Second)

	page := instance.Page()

	// 导航到扩展页面
	fmt.Println("📋 导航到chrome://extensions/...")
	if err := page.Navigate("chrome://extensions/"); err != nil {
		log.Printf("❌ 导航失败: %v", err)
		return
	}

	time.Sleep(3 * time.Second)

	// 启用开发者模式并检查扩展
	result, err := page.Evaluate(`
		(() => {
			try {
				const manager = document.querySelector('extensions-manager');
				if (!manager || !manager.shadowRoot) {
					return { error: "无法访问extensions-manager" };
				}

				// 启用开发者模式
				const devMode = manager.shadowRoot.querySelector('#devMode');
				if (devMode && !devMode.checked) {
					devMode.click();
				}

				// 等待一下让页面更新
				setTimeout(() => {}, 1000);

				// 检查扩展
				const items = manager.shadowRoot.querySelectorAll('extensions-item');
				const extensions = Array.from(items).map(item => {
					return {
						name: item.shadowRoot ? 
							(item.shadowRoot.querySelector('#name') ? 
								item.shadowRoot.querySelector('#name').textContent.trim() : 'Unknown') 
							: 'No Shadow Root',
						enabled: item.shadowRoot ? 
							(item.shadowRoot.querySelector('#enableToggle') ? 
								item.shadowRoot.querySelector('#enableToggle').checked : false)
							: false
					};
				});

				return {
					success: true,
					extensionCount: items.length,
					extensions: extensions,
					devModeEnabled: devMode ? devMode.checked : false
				};
			} catch (error) {
				return { error: error.message };
			}
		})()
	`)

	if err != nil {
		fmt.Printf("❌ 检查失败: %v\n", err)
	} else {
		fmt.Printf("📊 结果: %v\n", result)
	}

	fmt.Println("\n💡 手动检查:")
	fmt.Println("  1. 查看chrome://extensions/页面")
	fmt.Println("  2. 应该能看到'Simple Test Extension'")
	fmt.Println("  3. 开发者模式应该已启用")

	fmt.Println("\n⏳ 保持浏览器开启30秒供检查...")
	time.Sleep(30 * time.Second)

	fmt.Println("✅ 测试完成")
}