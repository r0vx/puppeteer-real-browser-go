package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/HNRow/puppeteer-real-browser-go/pkg/browser"
)

func main() {
	fmt.Println("🔧 Developer Mode Extension Test")
	fmt.Println("================================")

	ctx := context.Background()

	// 扩展路径
	extensionPaths := []string{
		"./examples/path/Extensions/kfjglmgfjedhhcddpfgfogkahmenikan/1.0_0",
		"./examples/path/Extensions/mcohilncbfahbmgdjkbpemcciiolgcge/3.66.10_0",
	}

	fmt.Printf("📦 准备加载扩展:\n")
	for i, path := range extensionPaths {
		fmt.Printf("  %d. %s\n", i+1, path)
	}

	// 浏览器配置
	opts := &browser.ConnectOptions{
		Headless:   false,
		Extensions: extensionPaths,
		Args: []string{
			"--enable-extensions",
			"--disable-extensions-file-access-check",
		},
	}

	fmt.Println("\n🚀 启动浏览器...")
	instance, err := browser.Connect(ctx, opts)
	if err != nil {
		log.Fatalf("浏览器启动失败: %v", err)
	}
	defer instance.Close()

	fmt.Println("✅ 浏览器启动成功")

	// 获取页面
	page := instance.Page()

	// 第一步：导航到扩展页面
	fmt.Println("🔍 第一步：打开扩展页面...")
	if err := page.Navigate("chrome://extensions/"); err != nil {
		log.Printf("❌ 无法访问扩展页面: %v", err)
		return
	}

	fmt.Println("✅ 扩展页面已打开")
	time.Sleep(2 * time.Second)

	// 第二步：自动开启开发者模式
	fmt.Println("🔧 第二步：自动开启开发者模式...")

	// 检查并开启开发者模式
	err = page.Evaluate(`
		// 等待页面加载
		setTimeout(() => {
			// 查找开发者模式开关
			const toggle = document.querySelector('#developerMode');
			if (toggle) {
				console.log('找到开发者模式开关');
				if (!toggle.checked) {
					console.log('开发者模式未开启，正在开启...');
					toggle.click();
				} else {
					console.log('开发者模式已开启');
				}
			} else {
				console.log('未找到开发者模式开关');
			}
		}, 1000);
	`)

	if err != nil {
		fmt.Printf("❌ 开启开发者模式失败: %v\n", err)
	} else {
		fmt.Println("✅ 开发者模式脚本已执行")
	}

	// 等待开发者模式生效
	time.Sleep(3 * time.Second)

	// 第三步：检查扩展是否出现
	fmt.Println("🔍 第三步：检查扩展是否出现...")

	// 简单的扩展检查
	result, err := page.Evaluate(`
		const items = document.querySelectorAll('extensions-item');
		const count = items.length;
		
		let extensionNames = [];
		items.forEach(item => {
			const shadow = item.shadowRoot;
			if (shadow) {
				const name = shadow.querySelector('#name');
				if (name) {
					extensionNames.push(name.textContent.trim());
				}
			}
		});
		
		return {
			extensionCount: count,
			extensionNames: extensionNames
		};
	`)

	if err != nil {
		fmt.Printf("❌ 检查扩展失败: %v\n", err)
	} else {
		fmt.Printf("📊 扩展检查结果: %+v\n", result)
	}

	fmt.Println("\n" + "="*50)
	fmt.Println("🔍 请手动验证:")
	fmt.Println("=" * 50)
	fmt.Println("1. 开发者模式是否已开启？（右上角开关）")
	fmt.Println("2. 现在是否看到了Discord Token Login扩展？")
	fmt.Println("3. 现在是否看到了OKX Wallet扩展？")
	fmt.Println("4. 如果还是没有，尝试点击'加载已解压的扩展程序'")
	fmt.Println("=" * 50)

	fmt.Println("\n⏳ 保持浏览器打开 10 秒供验证...")
	time.Sleep(10 * time.Second)

	fmt.Println("✅ 测试完成")
}
