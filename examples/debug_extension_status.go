package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/HNRow/puppeteer-real-browser-go/pkg/browser"
)

func main() {
	fmt.Println("🔍 Chrome扩展状态深度调试")
	fmt.Println("========================")

	ctx := context.Background()

	// 使用最简单的配置测试
	opts := &browser.ConnectOptions{
		Headless:                  false,
		AutoLoadDefaultExtensions: true,
		PersistProfile:            true,
		ProfileName:               "debug_extensions",
		Args: []string{
			"--start-maximized",
			"--enable-extensions",
		},
	}

	fmt.Println("🚀 启动Chrome...")
	instance, err := browser.Connect(ctx, opts)
	if err != nil {
		log.Fatalf("❌ Chrome启动失败: %v", err)
	}
	defer instance.Close()

	fmt.Println("✅ Chrome启动成功")
	
	// 等待Chrome完全加载
	time.Sleep(3 * time.Second)

	page := instance.Page()
	
	// 导航到扩展页面
	fmt.Println("📋 导航到chrome://extensions/...")
	if err := page.Navigate("chrome://extensions/"); err != nil {
		log.Printf("❌ 导航失败: %v", err)
		return
	}

	// 等待页面加载
	time.Sleep(2 * time.Second)

	// 先启用开发者模式
	fmt.Println("🔧 启用开发者模式...")
	_, err = page.Evaluate(`
		(() => {
			// 查找开发者模式切换开关
			const devModeToggle = document.querySelector('extensions-manager') && 
				document.querySelector('extensions-manager').shadowRoot &&
				document.querySelector('extensions-manager').shadowRoot.querySelector('#devMode');
			
			if (devModeToggle && !devModeToggle.checked) {
				devModeToggle.click();
				return "开发者模式已启用";
			} else if (devModeToggle && devModeToggle.checked) {
				return "开发者模式已经启用";
			} else {
				return "无法找到开发者模式切换开关";
			}
		})()
	`)
	
	if err != nil {
		fmt.Printf("❌ 启用开发者模式失败: %v\n", err)
	}

	// 再次等待
	time.Sleep(2 * time.Second)

	// 执行全面的扩展检查
	fmt.Println("🔍 执行全面扩展检查...")
	result, err := page.Evaluate(`
		(() => {
			const info = {
				// 基本页面信息
				url: location.href,
				title: document.title,
				
				// DOM结构检查
				hasExtensionsManager: !!document.querySelector('extensions-manager'),
				
				// 扩展项目检查
				extensionItems: [],
				extensionItemsCount: 0,
				
				// 开发者模式状态
				devModeEnabled: false,
				
				// 详细的DOM内容
				bodyText: document.body ? document.body.innerText.slice(0, 1000) : '',
				
				// Shadow DOM 检查
				shadowRootAccess: false,
				
				// 错误信息
				errors: []
			};
			
			try {
				// 检查extensions-manager
				const manager = document.querySelector('extensions-manager');
				if (manager && manager.shadowRoot) {
					info.shadowRootAccess = true;
					
					// 检查开发者模式
					const devMode = manager.shadowRoot.querySelector('#devMode');
					if (devMode) {
						info.devModeEnabled = devMode.checked;
					}
					
					// 查找扩展项目 - 多种选择器
					const selectors = [
						'extensions-item',
						'extensions-item-list extensions-item', 
						'#items-list extensions-item',
						'#extensions-list extensions-item'
					];
					
					let items = null;
					for (const selector of selectors) {
						items = manager.shadowRoot.querySelectorAll(selector);
						if (items.length > 0) {
							info.extensionItemsCount = items.length;
							break;
						}
					}
					
					// 如果找到扩展项目，获取详细信息
					if (items && items.length > 0) {
						info.extensionItems = Array.from(items).map(item => {
							const name = item.shadowRoot ? 
								(item.shadowRoot.querySelector('#name') ? 
									item.shadowRoot.querySelector('#name').textContent.trim() : 'unknown name') 
								: 'no shadow root';
							const enabled = item.shadowRoot ? 
								(item.shadowRoot.querySelector('#enableToggle') ? 
									item.shadowRoot.querySelector('#enableToggle').checked : false)
								: false;
							
							return {
								name: name,
								enabled: enabled,
								id: item.id || 'unknown id',
								data: item.data || null
							};
						});
					}
					
					// 检查是否有"加载已解压的扩展程序"按钮
					const loadUnpackedBtn = manager.shadowRoot.querySelector('[id*="load"]') || 
										  manager.shadowRoot.querySelector('[class*="load"]');
					info.hasLoadUnpackedButton = !!loadUnpackedBtn;
				} else {
					info.errors.push('无法访问extensions-manager的shadowRoot');
				}
				
				// 直接检查页面中是否有扩展相关文本
				const pageText = document.body.innerText.toLowerCase();
				info.hasExtensionText = pageText.includes('extension') || 
									   pageText.includes('扩展') ||
									   pageText.includes('discord') ||
									   pageText.includes('okx');
									   
			} catch (error) {
				info.errors.push('JavaScript执行错误: ' + error.message);
			}
			
			return info;
		})()
	`)

	if err != nil {
		fmt.Printf("❌ 扩展检查JavaScript执行失败: %v\n", err)
	} else {
		// 格式化输出结果
		fmt.Printf("📊 扩展检查结果:\n")
		fmt.Printf("  🔗 URL: %v\n", getField(result, "url"))
		fmt.Printf("  📋 标题: %v\n", getField(result, "title"))
		fmt.Printf("  🏗️  ExtensionsManager存在: %v\n", getField(result, "hasExtensionsManager"))
		fmt.Printf("  🔓 ShadowRoot访问: %v\n", getField(result, "shadowRootAccess"))
		fmt.Printf("  🔧 开发者模式: %v\n", getField(result, "devModeEnabled"))
		fmt.Printf("  📦 扩展项目数量: %v\n", getField(result, "extensionItemsCount"))
		fmt.Printf("  🔍 包含扩展文本: %v\n", getField(result, "hasExtensionText"))
		
		if extensionItems := getField(result, "extensionItems"); extensionItems != nil {
			fmt.Printf("  🎯 扩展详情: %v\n", extensionItems)
		}
		
		if errors := getField(result, "errors"); errors != nil {
			fmt.Printf("  ❌ 错误: %v\n", errors)
		}
		
		// 显示部分页面内容用于调试
		if bodyText := getField(result, "bodyText"); bodyText != nil {
			fmt.Printf("  📄 页面内容预览: %v\n", bodyText)
		}
	}

	fmt.Println("\n💡 人工验证指南:")
	fmt.Println("  1. 查看扩展管理页面是否显示任何扩展")
	fmt.Println("  2. 确认开发者模式已启用")
	fmt.Println("  3. 检查浏览器工具栏是否有扩展图标")
	fmt.Println("  4. 在控制台查看是否有扩展加载错误")

	fmt.Println("\n⏳ 保持浏览器开启60秒供手动检查...")
	time.Sleep(60 * time.Second)

	fmt.Println("✅ 调试完成")
}

// 辅助函数：安全获取map字段
func getField(data interface{}, key string) interface{} {
	if m, ok := data.(map[string]interface{}); ok {
		return m[key]
	}
	return nil
}