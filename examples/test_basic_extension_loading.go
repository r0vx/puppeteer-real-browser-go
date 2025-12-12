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
	fmt.Println("🧪 基本扩展加载测试")
	fmt.Println("==================")

	ctx := context.Background()

	// 获取扩展路径
	ext1, _ := filepath.Abs("examples/path/Extensions/kfjglmgfjedhhcddpfgfogkahmenikan/1.0_0")
	ext2, _ := filepath.Abs("examples/path/Extensions/mcohilncbfahbmgdjkbpemcciiolgcge/3.66.10_0")

	fmt.Printf("📂 Discord扩展: %s\n", ext1)
	fmt.Printf("📂 OKX扩展: %s\n", ext2)

	// 直接使用Extensions参数，避免AutoLoadDefaultExtensions的复杂逻辑
	opts := &browser.ConnectOptions{
		Headless:       false,
		PersistProfile: true,
		ProfileName:    "direct_extensions",
		Extensions:     []string{ext1, ext2}, // 直接指定扩展
		Args: []string{
			"--no-first-run",
			"--start-maximized",
		},
	}

	fmt.Println("🚀 启动Chrome并直接加载扩展...")
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

	// 检查扩展状态
	fmt.Println("🔍 检查扩展...")
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

				// 等待
				return new Promise(resolve => {
					setTimeout(() => {
						// 检查扩展
						const items = manager.shadowRoot.querySelectorAll('extensions-item');
						const extensions = Array.from(items).map(item => {
							const name = item.shadowRoot ? 
								(item.shadowRoot.querySelector('#name') ? 
									item.shadowRoot.querySelector('#name').textContent.trim() : 'Unknown Name') 
								: 'No Shadow Root';
							const enabled = item.shadowRoot ? 
								(item.shadowRoot.querySelector('#enableToggle') ? 
									item.shadowRoot.querySelector('#enableToggle').checked : false)
								: false;
							
							return { name, enabled, id: item.id || 'unknown' };
						});

						resolve({
							success: true,
							extensionCount: items.length,
							extensions: extensions,
							devModeEnabled: devMode ? devMode.checked : false,
							timestamp: new Date().toISOString()
						});
					}, 2000);
				});
			} catch (error) {
				return { success: false, error: error.message };
			}
		})()
	`)

	if err != nil {
		fmt.Printf("❌ 扩展检查失败: %v\n", err)
	} else {
		fmt.Printf("📊 扩展检查结果:\n")
		if resultMap, ok := result.(map[string]interface{}); ok {
			if success, ok := resultMap["success"].(bool); ok && success {
				fmt.Printf("  ✅ 检查成功\n")
				fmt.Printf("  🔧 开发者模式: %v\n", resultMap["devModeEnabled"])
				fmt.Printf("  📦 扩展数量: %v\n", resultMap["extensionCount"])
				
				if extensions, ok := resultMap["extensions"].([]interface{}); ok && len(extensions) > 0 {
					fmt.Println("  🎯 找到的扩展:")
					for i, ext := range extensions {
						if extMap, ok := ext.(map[string]interface{}); ok {
							fmt.Printf("    %d. 名称: %v, 启用: %v, ID: %v\n", 
								i+1, extMap["name"], extMap["enabled"], extMap["id"])
						}
					}
				} else {
					fmt.Println("  ❌ 没有找到任何扩展")
				}
			} else {
				fmt.Printf("  ❌ 检查失败: %v\n", resultMap["error"])
			}
		}
	}

	fmt.Println("\n💡 手动验证:")
	fmt.Println("  1. 检查chrome://extensions/页面是否显示扩展")
	fmt.Println("  2. 如果有扩展显示，说明直接使用Extensions参数有效")
	fmt.Println("  3. 如果没有，说明问题在更深层次")

	fmt.Println("\n⏳ 保持浏览器开启30秒供手动检查...")
	time.Sleep(30 * time.Second)

	fmt.Println("✅ 测试完成")
}