package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/HNRow/puppeteer-real-browser-go/pkg/browser"
)

func main() {
	fmt.Println("🧩 Pre-installed Extensions Demo")
	fmt.Println("=================================")

	ctx := context.Background()

	// 使用你现有的插件路径 - 从其他浏览器拷贝的插件目录
	extensionPaths := []string{
		"./path/Extensions/kfjglmgfjedhhcddpfgfogkahmenikan/1.0_0",     // 从其他浏览器拷贝的插件目录1
		"./path/Extensions/mcohilncbfahbmgdjkbpemcciiolgcge/3.66.10_0", // 从其他浏览器拷贝的插件目录2
	}

	// 验证插件是否存在
	fmt.Println("📂 Checking extension packages...")
	for i, path := range extensionPaths {
		if _, err := os.Stat(path); os.IsNotExist(err) {
			fmt.Printf("  ❌ Extension %d not found: %s\n", i+1, path)
		} else {
			fmt.Printf("  ✅ Extension %d ready: %s\n", i+1, path)
		}
	}

	// 基础配置 - 所有账号都会预装这些插件
	baseOptions := &browser.ConnectOptions{
		Headless:       false,
		UseCustomCDP:   false, // 使用标准模式以便插件正常工作
		Turnstile:      true,
		Extensions:     extensionPaths, // 🔑 关键：这些插件会自动预装
		PersistProfile: true,           // 启用持久化
		Args: []string{
			"--start-maximized",
			"--disable-blink-features=AutomationControlled",
			"--enable-extensions", // 明确启用插件
		},
	}

	// 创建账号管理器
	fmt.Println("\n👥 Creating Account Manager with Pre-installed Extensions...")
	manager := browser.NewAccountManager(baseOptions)
	defer manager.CloseAll()

	// 演示1: 创建多个账号，每个都会自动预装插件
	fmt.Println("\n🔧 Demo 1: Automatic Extension Pre-installation")
	accounts := []struct {
		name        string
		description string
	}{
		{"shopping_account", "E-commerce account with your extensions"},
		{"work_account", "Work account with your extensions"},
		{"personal_account", "Personal account with your extensions"},
	}

	for _, acc := range accounts {
		fmt.Printf("\n📦 Creating account: %s\n", acc.name)
		fmt.Printf("  📋 Description: %s\n", acc.description)

		// 创建账号 - 插件会自动预装
		account, err := manager.CreateAccount(ctx, acc.name, nil)
		if err != nil {
			log.Printf("Failed to create account %s: %v", acc.name, err)
			continue
		}

		fmt.Printf("  ✅ Account created with pre-installed extensions\n")

		// 导航到插件管理页面验证
		page := account.Instance.Page()
		fmt.Printf("  🔍 Verifying extensions installation...\n")

		if err := page.Navigate("chrome://extensions/"); err != nil {
			log.Printf("Cannot navigate to extensions page: %v", err)
			continue
		}

		time.Sleep(3 * time.Second)

		// 检查预装的插件
		result, err := page.Evaluate(`
			// 等待页面加载
			await new Promise(resolve => setTimeout(resolve, 2000));
			
			const extensions = Array.from(document.querySelectorAll('extensions-item')).map(item => {
				const name = item.shadowRoot?.querySelector('#name')?.textContent || 'Unknown';
				const id = item.getAttribute('id') || 'Unknown';
				const enabled = item.shadowRoot?.querySelector('cr-toggle')?.checked || false;
				return { name, id, enabled };
			});
			
			return {
				extensionsFound: extensions.length > 0,
				extensions: extensions,
				totalCount: extensions.length,
				pageLoaded: true
			};
		`)

		if err != nil {
			log.Printf("Failed to check extensions: %v", err)
		} else {
			fmt.Printf("  📊 Extension verification: %v\n", result)
		}

		// 设置页面标题便于识别
		page.Evaluate(fmt.Sprintf(`document.title = '%s - Extensions Pre-installed'`, acc.name))
	}

	// 演示2: 验证插件功能
	fmt.Println("\n🧪 Demo 2: Testing Extension Functionality")

	for _, acc := range accounts {
		account, exists := manager.GetAccount(acc.name)
		if !exists {
			continue
		}

		fmt.Printf("\n🔍 Testing extensions for: %s\n", acc.name)

		// 创建新页面测试插件功能
		page := account.Instance.Page()
		if err := page.Navigate("https://httpbin.org/headers"); err != nil {
			continue
		}

		time.Sleep(2 * time.Second)

		// 测试插件是否工作
		extensionTest, err := page.Evaluate(`
			return {
				chromeExtensionsAPI: typeof chrome !== 'undefined' && typeof chrome.runtime !== 'undefined',
				windowChrome: typeof window.chrome !== 'undefined',
				extensionsDetected: document.querySelectorAll('*[id*="extension"], *[class*="extension"]').length,
				pageTitle: document.title,
				userAgent: navigator.userAgent
			};
		`)

		if err == nil {
			fmt.Printf("  🔬 Extension functionality test: %v\n", extensionTest)
		}

		// 设置页面标识
		page.Evaluate(fmt.Sprintf(`document.title = '%s - Extension Test'`, acc.name))
	}

	// 演示3: 持久化验证 - 关闭后重新打开，插件还在
	fmt.Println("\n💾 Demo 3: Persistence Verification")
	fmt.Println("  📝 Closing all accounts...")

	accountNames := manager.ListAccounts()
	manager.CloseAll()
	fmt.Printf("  ✅ Closed %d accounts\n", len(accountNames))

	time.Sleep(3 * time.Second)

	fmt.Println("  🔄 Recreating accounts - extensions should persist...")
	for _, accName := range accountNames {
		account, err := manager.CreateAccount(ctx, accName, nil)
		if err != nil {
			continue
		}

		fmt.Printf("  ♻️  Account restored: %s\n", accName)

		// 验证插件是否还在
		page := account.Instance.Page()
		if err := page.Navigate("chrome://extensions/"); err == nil {
			time.Sleep(2 * time.Second)

			result, err := page.Evaluate(`
				const extensions = document.querySelectorAll('extensions-item');
				return {
					extensionCount: extensions.length,
					persistent: extensions.length > 0
				};
			`)

			if err == nil {
				fmt.Printf("    🔍 Persistence check: %v\n", result)
			}

			page.Evaluate(fmt.Sprintf(`document.title = '%s - Extensions Persisted'`, accName))
		}
	}

	// 使用说明
	fmt.Println("\n💡 Key Features Demonstrated:")
	fmt.Println("  ✅ 从其他浏览器拷贝的插件自动预装到所有账号")
	fmt.Println("  ✅ 无需手动安装 - 绕过 'installation disabled' 错误")
	fmt.Println("  ✅ 插件在浏览器重启后持久化保存")
	fmt.Println("  ✅ 每个账号拥有独立的插件数据")
	fmt.Println("  ✅ 支持 .crx 文件和已解压的插件目录")
	fmt.Println("  ✅ 兼容任何 Chrome 插件")

	fmt.Println("\n🔧 Technical Implementation:")
	fmt.Println("  • Extensions are extracted and installed to: ~/.puppeteer-real-browser-go/profiles/{account}/Default/Extensions/")
	fmt.Println("  • Extension preferences are pre-configured")
	fmt.Println("  • Chrome launches with extensions already 'installed' and enabled")
	fmt.Println("  • No security warnings or manual approval needed")

	fmt.Println("\n🎯 Your Extensions (从其他浏览器拷贝):")
	for i, path := range extensionPaths {
		fmt.Printf("  %d. %s\n", i+1, path)
	}

	fmt.Println("\n🔍 Manual Verification:")
	fmt.Println("  1. Check multiple browser windows - each should have your extensions installed")
	fmt.Println("  2. Go to chrome://extensions/ in any account")
	fmt.Println("  3. Extensions should be enabled by default")
	fmt.Println("  4. No 'Developer mode' required")
	fmt.Println("  5. Extensions work immediately without setup")

	fmt.Printf("\n📁 Extension Files Location:\n")
	for _, accName := range manager.ListAccounts() {
		fmt.Printf("  %s: ~/.puppeteer-real-browser-go/profiles/%s/Default/Extensions/\n", accName, accName)
	}

	fmt.Printf("\n📦 Your Source Extensions (从其他浏览器拷贝):\n")
	fmt.Printf("  • 插件1: ./path/Extensions/kfjglmgfjedhhcddpfgfogkahmenikan\n")
	fmt.Printf("  • 插件2: ./path/Extensions/mcohilncbfahbmgdjkbpemcciiolgcge\n")

	fmt.Println("\n⏳ Keeping all accounts open for 90 seconds for testing...")
	time.Sleep(90 * time.Second)

	fmt.Println("✅ Pre-installed Extensions Demo completed!")
}
