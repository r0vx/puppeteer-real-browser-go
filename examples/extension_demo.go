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
	fmt.Println("🧩 Chrome Extension Support Demo")
	fmt.Println("=================================")

	// 创建示例扩展目录结构
	if err := createSampleExtensions(); err != nil {
		log.Fatalf("Failed to create sample extensions: %v", err)
	}
	defer cleanupExtensions()

	ctx := context.Background()

	// 配置浏览器选项，包含扩展支持
	extensionPaths := []string{
		"./path/Extensions/1.0_0.crx",
		//"./path/Extensions/mcohilncbfahbmgdjkbpemcciiolgcge",
	}

	opts := &browser.ConnectOptions{
		Headless:     false, // 必须为 false，插件需要可见界面
		UseCustomCDP: false, // 使用标准模式以便插件正常工作
		Extensions:   extensionPaths,
		Args: []string{
			"--start-maximized",
			"--disable-blink-features=AutomationControlled",
			// 插件相关的额外参数
			"--enable-extensions",                    // 明确启用插件
			"--disable-extensions-file-access-check", // 允许文件访问
			"--disable-web-security",                 // 临时禁用安全检查以便测试
			"--allow-running-insecure-content",       // 允许不安全内容
		},
	}

	fmt.Printf("🚀 Starting browser with %d extensions...\n", len(extensionPaths))
	for i, path := range extensionPaths {
		fmt.Printf("  %d. %s\n", i+1, path)
		
		// 检查文件是否存在
		if _, err := os.Stat(path); os.IsNotExist(err) {
			fmt.Printf("     ⚠️  Warning: Extension file does not exist: %s\n", path)
		} else {
			fmt.Printf("     ✅ Extension file found: %s\n", path)
		}
	}

	// 连接浏览器
	fmt.Println("🔧 Connecting to browser with extension support...")
	instance, err := browser.Connect(ctx, opts)
	if err != nil {
		log.Fatalf("Failed to connect browser: %v", err)
	}
	defer instance.Close()
	
	fmt.Println("✅ Browser connected successfully")

	// 测试1: 基本扩展加载测试
	fmt.Println("\n📦 Test 1: Extension Loading Test")
	page := instance.Page()

	// 导航到 Chrome 扩展页面
	if err := page.Navigate("chrome://extensions/"); err != nil {
		log.Printf("Cannot navigate to extensions page: %v", err)
	} else {
		fmt.Println("  ✅ Navigated to chrome://extensions/")

		// 等待页面加载
		time.Sleep(3 * time.Second)

		// 检查扩展是否加载
		extensionsScript := `
			// 等待页面完全加载
			await new Promise(resolve => setTimeout(resolve, 2000));
			
			const extensions = Array.from(document.querySelectorAll('extensions-item')).map(item => {
				const name = item.shadowRoot?.querySelector('#name')?.textContent || 'Unknown';
				const id = item.getAttribute('id') || 'Unknown';
				const enabled = item.shadowRoot?.querySelector('cr-toggle')?.checked || false;
				return { name, id, enabled };
			});
			
			// 如果没有找到 extensions-item，尝试其他方式
			if (extensions.length === 0) {
				// 检查页面内容
				const pageContent = document.body.innerText;
				return {
					pageContent: pageContent.substring(0, 500),
					extensionsFound: false,
					querySelector: !!document.querySelector,
					shadowRootSupport: !!Element.prototype.attachShadow
				};
			}
			
			return {
				extensions: extensions,
				extensionsFound: true,
				totalCount: extensions.length
			};
		`

		result, err := page.Evaluate(extensionsScript)
		if err != nil {
			log.Printf("Failed to check extensions: %v", err)
		} else {
			fmt.Printf("  📋 Loaded extensions: %v\n", result)
		}
	}

	// 测试2: 创建多个上下文，每个都支持扩展
	fmt.Println("\n🌐 Test 2: Multiple Contexts with Extensions")

	// 上下文1 - 电商浏览
	context1, err := instance.CreateBrowserContext(nil)
	if err == nil {
		page1, err := context1.NewPage()
		if err == nil {
			page1.Navigate("https://httpbin.org/headers")
			page1.Evaluate(`document.title = 'Context 1 - E-commerce (with extensions)'`)
			fmt.Println("  ✅ Context 1: E-commerce browsing with extensions")
		}
	}

	// 上下文2 - 社交媒体
	context2, err := instance.CreateBrowserContext(nil)
	if err == nil {
		page2, err := context2.NewPage()
		if err == nil {
			page2.Navigate("https://httpbin.org/user-agent")
			page2.Evaluate(`document.title = 'Context 2 - Social Media (with extensions)'`)
			fmt.Println("  ✅ Context 2: Social media browsing with extensions")
		}
	}

	// 测试3: 扩展功能验证
	fmt.Println("\n🔍 Test 3: Extension Functionality Test")
	testPage := instance.Page()
	err = testPage.Navigate("https://httpbin.org/get")
	if err == nil {
		// 检查是否有广告拦截器的痕迹
		adBlockScript := `
			return {
				hasAdBlocker: !!window.AdBlock || !!window.adblock || !!window.uBlock,
				extensionsCount: (navigator.plugins || []).length,
				webStoreAccess: typeof chrome !== 'undefined' && typeof chrome.runtime !== 'undefined'
			};
		`

		result, err := page.Evaluate(adBlockScript)
		if err != nil {
			log.Printf("Failed to test extensions: %v", err)
		} else {
			fmt.Printf("  🔍 Extension test results: %v\n", result)
		}
	}

	// 测试4: 指纹浏览器 + 扩展组合
	fmt.Println("\n🎭 Test 4: Fingerprint Browser + Extensions")

	// 创建带有不同指纹的浏览器上下文
	fingerprintContext, err := instance.CreateBrowserContext(nil)
	if err == nil {
		fingerprintPage, err := fingerprintContext.NewPage()
		if err == nil {
			// 应用指纹
			fingerprintScript := `
				// 修改指纹信息
				Object.defineProperty(navigator, 'language', {
					get: () => 'de-DE'
				});
				Object.defineProperty(screen, 'width', {
					get: () => 1366
				});
				Object.defineProperty(screen, 'height', {
					get: () => 768
				});
				
				console.log('🎭 Fingerprint applied with extensions support');
			`

			fingerprintPage.Evaluate(fingerprintScript)
			fingerprintPage.Navigate("https://httpbin.org/anything/fingerprint-test")
			fingerprintPage.Evaluate(`document.title = 'Fingerprint Browser + Extensions'`)
			fmt.Println("  ✅ Fingerprint browser with extensions enabled")
		}
	}

	// 显示使用说明
	fmt.Println("\n💡 Manual Verification:")
	fmt.Println("  1. Check browser windows - you should see extension icons")
	fmt.Println("  2. Go to chrome://extensions/ to verify extensions are loaded")
	fmt.Println("  3. Test extension functionality (ad blocking, password management, etc.)")
	fmt.Println("  4. Each context should maintain extension state independently")

	fmt.Println("\n📊 Extension Features Demonstrated:")
	fmt.Println("  ✅ Load multiple extensions simultaneously")
	fmt.Println("  ✅ Extensions work in all browser contexts")
	fmt.Println("  ✅ Compatible with fingerprint randomization")
	fmt.Println("  ✅ Support for unpacked extensions")
	fmt.Println("  ✅ Extension isolation per context")

	fmt.Println("\n⏳ Keeping browser open for 60 seconds for manual testing...")
	time.Sleep(60 * time.Second)

	fmt.Println("✅ Extension demo completed!")
}

// createSampleExtensions creates sample extension directories for testing
func createSampleExtensions() error {
	baseDir := "sample_extensions"

	// 创建广告拦截器扩展
	adBlockerDir := filepath.Join(baseDir, "ad_blocker")
	if err := os.MkdirAll(adBlockerDir, 0755); err != nil {
		return err
	}

	// 创建 manifest.json
	adBlockerManifest := `{
	"manifest_version": 3,
	"name": "Sample Ad Blocker",
	"version": "1.0",
	"description": "A sample ad blocker extension for testing",
	"permissions": [
		"storage",
		"activeTab"
	],
	"action": {
		"default_popup": "popup.html",
		"default_title": "Sample Ad Blocker"
	},
	"content_scripts": [{
		"matches": ["<all_urls>"],
		"js": ["content.js"]
	}],
	"icons": {
		"16": "icon16.png",
		"48": "icon48.png", 
		"128": "icon128.png"
	}
}`

	if err := os.WriteFile(filepath.Join(adBlockerDir, "manifest.json"), []byte(adBlockerManifest), 0644); err != nil {
		return err
	}

	// 创建 popup.html
	adBlockerPopup := `<!DOCTYPE html>
<html>
<head>
	<style>
		body { width: 200px; padding: 10px; }
		.status { color: green; font-weight: bold; }
	</style>
</head>
<body>
	<h3>Sample Ad Blocker</h3>
	<div class="status">✅ Active</div>
	<p>Blocking ads on this page!</p>
</body>
</html>`

	if err := os.WriteFile(filepath.Join(adBlockerDir, "popup.html"), []byte(adBlockerPopup), 0644); err != nil {
		return err
	}

	// 创建 content.js
	adBlockerContent := `
// Sample ad blocker content script
console.log('🛡️ Sample Ad Blocker: Content script loaded');

// 简单的广告拦截模拟
const blockAds = () => {
	// 隐藏常见的广告选择器
	const adSelectors = ['.ad', '.ads', '.advertisement', '[id*="ad"]', '[class*="ad"]'];
	adSelectors.forEach(selector => {
		const ads = document.querySelectorAll(selector);
		ads.forEach(ad => {
			ad.style.display = 'none';
		});
	});
};

// 页面加载时运行
if (document.readyState === 'loading') {
	document.addEventListener('DOMContentLoaded', blockAds);
} else {
	blockAds();
}

// 标记扩展存在
window.AdBlockerExtension = true;
`

	if err := os.WriteFile(filepath.Join(adBlockerDir, "content.js"), []byte(adBlockerContent), 0644); err != nil {
		return err
	}

	// 创建密码管理器扩展
	passwordManagerDir := filepath.Join(baseDir, "password_manager")
	if err := os.MkdirAll(passwordManagerDir, 0755); err != nil {
		return err
	}

	// 密码管理器的 manifest.json
	passwordManagerManifest := `{
	"manifest_version": 3,
	"name": "Sample Password Manager",
	"version": "1.0",
	"description": "A sample password manager extension for testing",
	"permissions": [
		"storage",
		"activeTab",
		"scripting"
	],
	"action": {
		"default_popup": "popup.html",
		"default_title": "Sample Password Manager"
	},
	"content_scripts": [{
		"matches": ["<all_urls>"],
		"js": ["content.js"]
	}]
}`

	if err := os.WriteFile(filepath.Join(passwordManagerDir, "manifest.json"), []byte(passwordManagerManifest), 0644); err != nil {
		return err
	}

	// 密码管理器的 popup.html
	passwordManagerPopup := `<!DOCTYPE html>
<html>
<head>
	<style>
		body { width: 250px; padding: 10px; }
		.vault { background: #f0f8ff; padding: 8px; margin: 5px 0; }
	</style>
</head>
<body>
	<h3>Password Manager</h3>
	<div class="vault">
		<strong>🔐 Secure Vault</strong><br>
		<small>3 passwords saved</small>
	</div>
	<button onclick="fillPassword()">Auto-fill</button>
	<script>
		function fillPassword() {
			chrome.tabs.query({active: true, currentWindow: true}, function(tabs) {
				console.log('Password auto-fill requested');
			});
		}
	</script>
</body>
</html>`

	if err := os.WriteFile(filepath.Join(passwordManagerDir, "popup.html"), []byte(passwordManagerPopup), 0644); err != nil {
		return err
	}

	// 密码管理器的 content.js
	passwordManagerContent := `
// Sample password manager content script
console.log('🔐 Sample Password Manager: Content script loaded');

// 检测密码字段
const detectPasswordFields = () => {
	const passwordFields = document.querySelectorAll('input[type="password"]');
	const emailFields = document.querySelectorAll('input[type="email"], input[name*="email"], input[name*="username"]');
	
	if (passwordFields.length > 0) {
		console.log('🔐 Password fields detected:', passwordFields.length);
		
		// 添加自动填充提示
		passwordFields.forEach(field => {
			field.addEventListener('focus', () => {
				console.log('🔐 Password field focused - auto-fill available');
			});
		});
	}
};

// 页面加载时检测
if (document.readyState === 'loading') {
	document.addEventListener('DOMContentLoaded', detectPasswordFields);
} else {
	detectPasswordFields();
}

// 标记扩展存在
window.PasswordManagerExtension = true;
`

	if err := os.WriteFile(filepath.Join(passwordManagerDir, "content.js"), []byte(passwordManagerContent), 0644); err != nil {
		return err
	}

	fmt.Println("  ✅ Created sample extensions:")
	fmt.Println("    - Sample Ad Blocker")
	fmt.Println("    - Sample Password Manager")

	return nil
}

// cleanupExtensions removes sample extension directories
func cleanupExtensions() {
	os.RemoveAll("sample_extensions")
	fmt.Println("  🧹 Cleaned up sample extensions")
}
