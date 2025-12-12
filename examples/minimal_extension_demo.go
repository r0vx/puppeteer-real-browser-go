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
	fmt.Println("🧪 极简扩展加载测试")
	fmt.Println("==================")

	ctx := context.Background()

	// 获取扩展目录的绝对路径
	extensions := []string{
		"examples/path/Extensions/kfjglmgfjedhhcddpfgfogkahmenikan/1.0_0",
		"examples/path/Extensions/mcohilncbfahbmgdjkbpemcciiolgcge/3.66.10_0",
	}

	var absolutePaths []string
	for _, ext := range extensions {
		if absPath, err := filepath.Abs(ext); err == nil {
			absolutePaths = append(absolutePaths, absPath)
			fmt.Printf("📂 扩展路径: %s\n", absPath)
		}
	}

	// 最简配置 - 只使用必要的标志
	opts := &browser.ConnectOptions{
		Headless:       false,
		PersistProfile: true,
		ProfileName:    "minimal_test",
		Extensions:     absolutePaths, // 直接指定扩展路径
		Args: []string{
			"--no-first-run",
			"--start-maximized",
			"--enable-extensions",
		},
	}

	fmt.Println("🚀 使用最简配置启动Chrome...")
	instance, err := browser.Connect(ctx, opts)
	if err != nil {
		log.Fatalf("❌ Chrome启动失败: %v", err)
	}
	defer instance.Close()

	fmt.Println("✅ Chrome启动成功")

	// 等待Chrome加载
	time.Sleep(5 * time.Second)

	page := instance.Page()

	// 导航到扩展页面
	fmt.Println("📋 导航到chrome://extensions/...")
	if err := page.Navigate("chrome://extensions/"); err != nil {
		log.Printf("❌ 导航失败: %v", err)
		return
	}

	time.Sleep(3 * time.Second)

	// 启用开发者模式
	fmt.Println("🔧 尝试启用开发者模式...")
	devModeResult, err := page.Evaluate(`
		(() => {
			try {
				const manager = document.querySelector('extensions-manager');
				if (manager && manager.shadowRoot) {
					const devMode = manager.shadowRoot.querySelector('#devMode');
					if (devMode) {
						if (!devMode.checked) {
							devMode.click();
							return { success: true, message: "开发者模式已启用" };
						} else {
							return { success: true, message: "开发者模式已经启用" };
						}
					}
					return { success: false, message: "无法找到开发者模式开关" };
				}
				return { success: false, message: "无法访问extensions-manager" };
			} catch (error) {
				return { success: false, message: "错误: " + error.message };
			}
		})()
	`)

	if err != nil {
		fmt.Printf("❌ 开发者模式设置失败: %v\n", err)
	} else {
		fmt.Printf("🔧 开发者模式结果: %v\n", devModeResult)
	}

	time.Sleep(2 * time.Second)

	// 检查扩展
	fmt.Println("🔍 检查扩展状态...")
	result, err := page.Evaluate(`
		(() => {
			try {
				const manager = document.querySelector('extensions-manager');
				if (!manager || !manager.shadowRoot) {
					return { error: "无法访问extensions-manager" };
				}

				const shadowRoot = manager.shadowRoot;
				
				// 尝试多种选择器查找扩展
				const selectors = [
					'extensions-item',
					'#items-list extensions-item',
					'extensions-item-list extensions-item',
					'[slot="main"] extensions-item'
				];

				let extensions = [];
				let itemsFound = 0;

				for (const selector of selectors) {
					const items = shadowRoot.querySelectorAll(selector);
					if (items.length > 0) {
						itemsFound = items.length;
						extensions = Array.from(items).map(item => {
							const name = item.shadowRoot ? 
								(item.shadowRoot.querySelector('#name') ? 
									item.shadowRoot.querySelector('#name').textContent.trim() : 'Unknown Name') 
								: 'No Shadow Root';
							return {
								name: name,
								id: item.id || 'unknown',
								enabled: item.shadowRoot ? 
									(item.shadowRoot.querySelector('#enableToggle') ? 
										item.shadowRoot.querySelector('#enableToggle').checked : false)
									: false
							};
						});
						break;
					}
				}

				// 检查开发者模式状态
				const devMode = shadowRoot.querySelector('#devMode');
				const isDevModeEnabled = devMode ? devMode.checked : false;

				// 检查页面内容
				const pageContent = document.body.innerText;
				
				return {
					extensionCount: itemsFound,
					extensions: extensions,
					devModeEnabled: isDevModeEnabled,
					hasExtensionText: pageContent.includes('扩展') || 
									pageContent.includes('extension') ||
									pageContent.includes('Discord') ||
									pageContent.includes('OKX'),
					pageTitle: document.title,
					url: location.href
				};
			} catch (error) {
				return { error: error.message };
			}
		})()
	`)

	if err != nil {
		fmt.Printf("❌ 扩展检查失败: %v\n", err)
	} else {
		fmt.Printf("📊 扩展检查结果:\n")
		if resultMap, ok := result.(map[string]interface{}); ok {
			for key, value := range resultMap {
				fmt.Printf("  %s: %v\n", key, value)
			}
		} else {
			fmt.Printf("  原始结果: %v\n", result)
		}
	}

	fmt.Println("\n💡 手动验证:")
	fmt.Println("  1. 检查chrome://extensions/页面")
	fmt.Println("  2. 确认开发者模式是否启用")
	fmt.Println("  3. 查看是否有任何扩展显示")

	fmt.Println("\n⏳ 保持浏览器开启30秒供检查...")
	time.Sleep(30 * time.Second)

	fmt.Println("✅ 测试完成")
}
