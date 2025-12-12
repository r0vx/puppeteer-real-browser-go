package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/HNRow/puppeteer-real-browser-go/pkg/browser"
)

func main() {
	fmt.Println("🔍 Chrome扩展加载深度调试")
	fmt.Println("==========================")

	ctx := context.Background()

	// 创建一个全新的测试配置文件
	profileName := "debug_extension_" + fmt.Sprintf("%d", time.Now().Unix())
	
	fmt.Printf("👤 测试用户: %s\n", profileName)
	fmt.Println("🎯 调试目标: 分析扩展加载失败的真正原因")

	// 1. 首先检查CRX文件的有效性
	fmt.Println("\n=== 步骤1: 验证CRX文件 ===")
	crxFiles := []string{
		"examples/path/crx/1.0_0.crx",
		"examples/path/crx/3.66.10_0.crx",
	}
	
	for i, crxPath := range crxFiles {
		fmt.Printf("🔍 检查CRX文件 %d: %s\n", i+1, crxPath)
		
		// 检查文件是否存在
		if info, err := os.Stat(crxPath); err != nil {
			fmt.Printf("  ❌ 文件不存在: %v\n", err)
			continue
		} else {
			fmt.Printf("  ✅ 文件存在 (大小: %d bytes, 权限: %s)\n", info.Size(), info.Mode())
		}
		
		// 检查文件权限
		if info, _ := os.Stat(crxPath); info.Mode().Perm() != 0644 {
			fmt.Printf("  ⚠️  权限不正确: %s (应该是 -rw-r--r--)\n", info.Mode())
		}
		
		// 检查CRX文件头
		if err := validateCRXFile(crxPath); err != nil {
			fmt.Printf("  ❌ CRX文件格式错误: %v\n", err)
		} else {
			fmt.Printf("  ✅ CRX文件格式正确\n")
		}
	}

	// 2. 启动Chrome并获取详细信息
	fmt.Println("\n=== 步骤2: 启动Chrome ===")
	opts := &browser.ConnectOptions{
		Headless:                  false,
		AutoLoadDefaultExtensions: true,
		PersistProfile:            true,
		ProfileName:               profileName,
	}

	instance, err := browser.Connect(ctx, opts)
	if err != nil {
		log.Fatalf("❌ Chrome启动失败: %v", err)
	}
	defer instance.Close()

	fmt.Println("✅ Chrome启动成功")

	// 3. 检查用户数据目录结构
	fmt.Println("\n=== 步骤3: 检查用户数据目录 ===")
	userDataDir := fmt.Sprintf("/Users/rowei/.puppeteer-real-browser-go/profiles/%s", profileName)
	checkUserDataStructure(userDataDir)

	// 4. 导航到扩展页面并进行JavaScript调试
	fmt.Println("\n=== 步骤4: JavaScript扩展调试 ===")
	page := instance.Page()
	if err := page.Navigate("chrome://extensions/"); err != nil {
		log.Printf("❌ 导航失败: %v", err)
	} else {
		fmt.Println("✅ 已导航到 chrome://extensions/")
	}

	// 等待页面加载
	time.Sleep(3 * time.Second)

	// 执行深度JavaScript检查
	result, err := page.Evaluate(`
		// 深度检查扩展状态
		(() => {
			const info = {
				// 基本信息
				url: location.href,
				title: document.title,
				
				// 扩展相关DOM元素
				extensionItems: document.querySelectorAll('extensions-item').length,
				extensionManager: !!document.querySelector('extensions-manager'),
				extensionsList: !!document.querySelector('extensions-item-list'),
				
				// Chrome扩展API
				chromeExtensions: !!(window.chrome && window.chrome.management),
				
				// 页面内容分析
				pageText: document.body ? document.body.innerText.slice(0, 500) : '',
				
				// 开发者模式状态
				devModeToggle: !!document.querySelector('#devMode'),
				devModeEnabled: document.querySelector('#devMode') ? 
					document.querySelector('#devMode').checked : false,
				
				// 错误信息
				errorElements: document.querySelectorAll('.error, .warning').length,
				
				// 扩展卡片详细信息
				extensionCards: Array.from(document.querySelectorAll('extensions-item')).map(card => ({
					name: card.shadowRoot ? 
						(card.shadowRoot.querySelector('#name') ? 
							card.shadowRoot.querySelector('#name').textContent : 'unknown') 
						: 'no shadow root',
					enabled: card.shadowRoot ? 
						(card.shadowRoot.querySelector('#enableToggle') ? 
							card.shadowRoot.querySelector('#enableToggle').checked : false)
						: false
				}))
			};
			
			return info;
		})()
	`)

	if err != nil {
		fmt.Printf("❌ JavaScript执行失败: %v\n", err)
	} else {
		fmt.Printf("📊 扩展页面分析结果:\n")
		fmt.Printf("  🔗 URL: %v\n", result)
		// 这里应该打印更详细的结果，但由于返回类型复杂，先简化显示
	}

	// 5. 检查Chrome错误日志
	fmt.Println("\n=== 步骤5: Chrome错误检查 ===")
	
	// 尝试获取console错误
	logs, err := page.Evaluate(`
		// 获取可能的错误信息
		(() => {
			const errors = [];
			
			// 检查是否有JavaScript错误
			if (window.console && window.console.error) {
				errors.push('Console API available');
			}
			
			// 检查页面是否正确加载
			if (document.readyState !== 'complete') {
				errors.push('Page not fully loaded: ' + document.readyState);
			}
			
			return errors;
		})()
	`)
	
	if err != nil {
		fmt.Printf("❌ 无法获取错误日志: %v\n", err)
	} else {
		fmt.Printf("📝 错误检查结果: %v\n", logs)
	}

	fmt.Println("\n💡 手动验证指南:")
	fmt.Println("  1. 检查chrome://extensions/页面是否显示任何扩展")
	fmt.Println("  2. 打开开发者工具(F12)查看Console错误")
	fmt.Println("  3. 在Extensions页面启用'开发者模式'")
	fmt.Println("  4. 查看是否有'加载已解压的扩展程序'选项")

	fmt.Println("\n⏳ 保持浏览器开启60秒供手动检查...")
	time.Sleep(60 * time.Second)

	fmt.Println("✅ 调试完成")
}

// validateCRXFile 验证CRX文件的基本格式
func validateCRXFile(filePath string) error {
	file, err := os.Open(filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	// 读取CRX文件头
	header := make([]byte, 16)
	n, err := file.Read(header)
	if err != nil || n < 16 {
		return fmt.Errorf("无法读取CRX文件头")
	}

	// 检查CRX魔术字节
	if string(header[:4]) != "Cr24" {
		return fmt.Errorf("不是有效的CRX文件 (魔术字节: %s)", string(header[:4]))
	}

	// 检查版本
	version := uint32(header[4]) | uint32(header[5])<<8 | uint32(header[6])<<16 | uint32(header[7])<<24
	if version != 2 && version != 3 {
		return fmt.Errorf("不支持的CRX版本: %d", version)
	}

	fmt.Printf("    📋 CRX版本: %d\n", version)
	return nil
}

// checkUserDataStructure 检查用户数据目录结构
func checkUserDataStructure(userDataDir string) {
	fmt.Printf("📁 用户数据目录: %s\n", userDataDir)
	
	// 检查关键目录和文件
	paths := []string{
		filepath.Join(userDataDir, "Default"),
		filepath.Join(userDataDir, "Default", "Extensions"),
		filepath.Join(userDataDir, "Default", "Preferences"),
		filepath.Join(userDataDir, "Default", "Local State"),
	}
	
	for _, path := range paths {
		if info, err := os.Stat(path); err != nil {
			fmt.Printf("  ❌ %s: 不存在\n", filepath.Base(path))
		} else {
			if info.IsDir() {
				if entries, err := os.ReadDir(path); err == nil {
					fmt.Printf("  📁 %s: 存在 (%d 项)\n", filepath.Base(path), len(entries))
				} else {
					fmt.Printf("  📁 %s: 存在但无法读取\n", filepath.Base(path))
				}
			} else {
				fmt.Printf("  📄 %s: 存在 (%d bytes)\n", filepath.Base(path), info.Size())
			}
		}
	}
}