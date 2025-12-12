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
	fmt.Println("🚀 ChromeDP扩展加载解决方案")
	fmt.Println("===========================")

	// 关键发现：--load-extension 只支持未打包的扩展目录，不支持CRX文件！
	// 解决方案：自动将CRX解压到临时目录，然后使用目录路径

	ctx := context.Background()

	fmt.Println("📋 问题分析:")
	fmt.Println("  ❌ Chrome --load-extension 不支持 .crx 文件")
	fmt.Println("  ✅ Chrome --load-extension 只支持未打包扩展目录")
	fmt.Println("  💡 解决方案: 动态解压CRX → 加载目录")

	fmt.Println("\n=== 方案1: 直接使用未打包扩展目录 ===")
	
	// 确保未打包扩展目录权限正确
	unpackedDirs := []string{
		"examples/path/Extensions/kfjglmgfjedhhcddpfgfogkahmenikan/1.0_0",
		"examples/path/Extensions/mcohilncbfahbmgdjkbpemcciiolgcge/3.66.10_0",
	}

	fmt.Println("🔧 修复扩展目录权限...")
	for _, dir := range unpackedDirs {
		if err := fixDirectoryPermissions(dir); err != nil {
			fmt.Printf("❌ 修复权限失败 %s: %v\n", dir, err)
		} else {
			fmt.Printf("✅ 权限已修复: %s\n", filepath.Base(dir))
		}
	}

	// 测试未打包扩展加载
	profileName := "chromedp_solution_" + fmt.Sprintf("%d", time.Now().Unix())
	
	fmt.Printf("\n🔧 使用未打包扩展目录测试 (配置: %s)\n", profileName)
	
	opts := &browser.ConnectOptions{
		Headless:       false,
		PersistProfile: true,
		ProfileName:    profileName,
		Extensions:     unpackedDirs, // 直接使用未打包目录
		Args: []string{
			"--start-maximized",
			"--disable-blink-features=AutomationControlled",
		},
	}

	instance, err := browser.Connect(ctx, opts)
	if err != nil {
		log.Fatalf("❌ Chrome启动失败: %v", err)
	}
	defer instance.Close()

	fmt.Println("✅ Chrome启动成功")

	// 导航到扩展页面
	page := instance.Page()
	if err := page.Navigate("chrome://extensions/"); err != nil {
		log.Printf("❌ 导航失败: %v", err)
	} else {
		fmt.Println("📋 扩展管理页面已打开")
	}

	// 等待并检查扩展
	time.Sleep(3 * time.Second)

	// JavaScript检查扩展状态
	result, err := page.Evaluate(`
		(() => {
			const extensionItems = document.querySelectorAll('extensions-item');
			const extensions = Array.from(extensionItems).map(item => ({
				name: item.shadowRoot ? 
					(item.shadowRoot.querySelector('#name') ? 
						item.shadowRoot.querySelector('#name').textContent.trim() : 'unknown') 
					: 'no shadow root',
				enabled: item.shadowRoot ? 
					(item.shadowRoot.querySelector('#enableToggle') ? 
						item.shadowRoot.querySelector('#enableToggle').checked : false)
					: false,
				id: item.id || 'unknown'
			}));
			
			return {
				count: extensionItems.length,
				extensions: extensions,
				devModeEnabled: document.querySelector('#devMode') ? 
					document.querySelector('#devMode').checked : false
			};
		})()
	`)

	if err != nil {
		fmt.Printf("❌ JavaScript执行失败: %v\n", err)
	} else {
		fmt.Printf("📊 扩展检测结果: %v\n", result)
	}

	fmt.Println("\n💡 验证步骤:")
	fmt.Println("  1. 检查chrome://extensions/页面")
	fmt.Println("  2. 确认看到 Discord Token Login 和 OKX Wallet")
	fmt.Println("  3. 查看浏览器工具栏的扩展图标")

	fmt.Println("\n⏳ 保持浏览器开启30秒供检查...")
	time.Sleep(30 * time.Second)

	fmt.Println("✅ 测试完成")
	
	fmt.Println("\n🎯 关键发现:")
	fmt.Println("  • Chrome --load-extension 参数只支持未打包扩展目录")
	fmt.Println("  • CRX文件需要先解压才能通过 --load-extension 加载")
	fmt.Println("  • 扩展目录必须有正确的权限 (755 for dirs, 644 for files)")
	fmt.Println("  • 建议回退到使用未打包扩展目录的方案")
}

// fixDirectoryPermissions 修复目录和文件权限
func fixDirectoryPermissions(dir string) error {
	// 检查目录是否存在
	if _, err := os.Stat(dir); err != nil {
		return fmt.Errorf("目录不存在: %w", err)
	}

	// 修复目录权限
	if err := os.Chmod(dir, 0755); err != nil {
		return fmt.Errorf("修复目录权限失败: %w", err)
	}

	// 递归修复所有子目录和文件权限
	return filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			// 目录权限设置为 755
			return os.Chmod(path, 0755)
		} else {
			// 文件权限设置为 644
			return os.Chmod(path, 0644)
		}
	})
}