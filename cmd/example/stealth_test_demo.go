//go:build ignore
// +build ignore

package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/r0vx/puppeteer-real-browser-go/pkg/browser"
)

// 测试每个 stealth 项对页面的影响
// 运行方式: go run stealth_test_demo.go

func main() {
	fmt.Println("🧪 Stealth 项逐一测试")
	fmt.Println("=====================================")
	fmt.Println()

	// 测试项列表（从 GetAdvancedStealthScript 提取）
	stealthItems := []struct {
		Name   string
		Script string
	}{
		{
			Name: "1. MouseEvent fix (基础)",
			Script: `
				Object.defineProperty(MouseEvent.prototype, 'screenX', {
					get: function() { return this.clientX + (window.screenX || 0); },
					configurable: true
				});
				Object.defineProperty(MouseEvent.prototype, 'screenY', {
					get: function() { return this.clientY + (window.screenY || 0); },
					configurable: true
				});
			`,
		},
		{
			Name: "2. navigator.webdriver 隐藏",
			Script: `
				Object.defineProperty(navigator, 'webdriver', {
					get: () => undefined,
					configurable: true
				});
			`,
		},
		{
			Name: "3. navigator.plugins 伪造",
			Script: `
				Object.defineProperty(navigator, 'plugins', {
					get: () => {
						const plugins = [
							{ name: 'Chrome PDF Plugin', filename: 'internal-pdf-viewer', description: 'Portable Document Format', length: 1 },
							{ name: 'Chrome PDF Viewer', filename: 'mhjfbmdgcfjbbpaeojofohoefgiehjai', description: '', length: 1 },
							{ name: 'Native Client', filename: 'internal-nacl-plugin', description: '', length: 2 }
						];
						plugins.refresh = () => {};
						return plugins;
					},
					configurable: true
				});
			`,
		},
		{
			Name: "4. navigator.languages 固定",
			Script: `
				Object.defineProperty(navigator, 'languages', {
					get: () => ['en-US', 'en'],
					configurable: true
				});
			`,
		},
		{
			Name: "5. permissions.query 拦截",
			Script: `
				if (navigator.permissions && navigator.permissions.query) {
					const originalQuery = navigator.permissions.query.bind(navigator.permissions);
					navigator.permissions.query = (parameters) => {
						if (parameters.name === 'notifications') {
							return Promise.resolve({ state: Notification.permission, onchange: null });
						}
						return originalQuery(parameters);
					};
				}
			`,
		},
		{
			Name: "6. window.chrome 伪造",
			Script: `
				if (!window.chrome) { window.chrome = {}; }
				if (!window.chrome.runtime) {
					window.chrome.runtime = { onConnect: undefined, onMessage: undefined };
				}
			`,
		},
		{
			Name: "7. 删除自动化痕迹",
			Script: `
				delete window.__nightmare;
				delete window._phantom;
				delete window.callPhantom;
				delete window.__webdriver_script_fn;
				delete window.__webdriver_evaluate;
				delete window.__selenium_unwrapped;
				delete window.webdriver;
				delete window.domAutomation;
				delete window.domAutomationController;
			`,
		},
		{
			Name: "8. console 过滤 ⚠️",
			Script: `
				const originalConsole = { debug: console.debug, log: console.log, warn: console.warn, error: console.error };
				const filterLogs = (method, args) => {
					const message = args.join(' ');
					if (message.includes('DevTools') || message.includes('puppeteer')) return;
					return originalConsole[method].apply(console, args);
				};
				console.debug = (...args) => filterLogs('debug', args);
				console.log = (...args) => filterLogs('log', args);
				console.warn = (...args) => filterLogs('warn', args);
				console.error = (...args) => filterLogs('error', args);
			`,
		},
		{
			Name: "9. Error.prepareStackTrace 修改",
			Script: `
				const originalPrepareStackTrace = Error.prepareStackTrace;
				Error.prepareStackTrace = function(error, stack) {
					if (originalPrepareStackTrace) {
						const result = originalPrepareStackTrace(error, stack);
						if (typeof result === 'string') {
							return result.replace(/chrome-extension:\/\/[^\/]+/g, 'chrome-extension://redacted');
						}
						return result;
					}
					return stack;
				};
			`,
		},
		{
			Name: "10. performance.now 偏移 ⚠️",
			Script: `
				const originalPerformanceNow = performance.now;
				let timeOffset = Math.random() * 10 - 5;
				performance.now = function() {
					return originalPerformanceNow.call(performance) + timeOffset;
				};
			`,
		},
		{
			Name: "11. document.createElement 拦截 ⚠️⚠️",
			Script: `
				const originalCreateElement = document.createElement;
				document.createElement = function(tagName) {
					const element = originalCreateElement.call(document, tagName);
					if (tagName.toLowerCase() === 'iframe') {
						const originalSetAttribute = element.setAttribute;
						element.setAttribute = function(name, value) {
							if (name === 'src' && typeof value === 'string') {
								if (value.includes('devtools') || value.includes('chrome-extension')) {
									return;
								}
							}
							return originalSetAttribute.call(element, name, value);
						};
					}
					return element;
				};
			`,
		},
		{
			Name: "12. HeadlessChrome 替换",
			Script: `
				if (navigator.userAgent.includes('HeadlessChrome')) {
					Object.defineProperty(navigator, 'userAgent', {
						get: () => navigator.userAgent.replace('HeadlessChrome', 'Chrome'),
						configurable: true
					});
				}
			`,
		},
		{
			Name: "13-15. navigator 属性伪造",
			Script: `
				Object.defineProperty(navigator, 'vendor', { get: () => 'Google Inc.', configurable: true });
				Object.defineProperty(navigator, 'product', { get: () => 'Gecko', configurable: true });
				Object.defineProperty(navigator, 'hardwareConcurrency', { get: () => 4, configurable: true });
				Object.defineProperty(navigator, 'deviceMemory', { get: () => 8, configurable: true });
			`,
		},
		{
			Name: "16. Notification.permission",
			Script: `
				if (Notification.permission === 'default') {
					Object.defineProperty(Notification, 'permission', {
						get: () => 'denied',
						configurable: true
					});
				}
			`,
		},
		{
			Name: "17. navigator.connection",
			Script: `
				if (navigator.connection) {
					Object.defineProperty(navigator.connection, 'rtt', {
						get: () => 50 + Math.random() * 50,
						configurable: true
					});
				}
			`,
		},
		{
			Name: "18. Function.prototype.toString ⚠️",
			Script: `
				const originalToString = Function.prototype.toString;
				Function.prototype.toString = function() {
					const result = originalToString.call(this);
					if (result.includes('native code') && (result.includes('puppeteer') || result.includes('chromedp'))) {
						return 'function () { [native code] }';
					}
					return result;
				};
			`,
		},
		{
			Name: "19. window 尺寸修复",
			Script: `
				if (window.outerHeight === 0 || window.outerWidth === 0) {
					Object.defineProperty(window, 'outerHeight', { get: () => window.innerHeight + 120, configurable: true });
					Object.defineProperty(window, 'outerWidth', { get: () => window.innerWidth + 16, configurable: true });
				}
			`,
		},
		{
			Name: "20. screen 属性",
			Script: `
				Object.defineProperty(navigator, 'maxTouchPoints', { get: () => 0, configurable: true });
				Object.defineProperty(navigator, 'onLine', { get: () => true, configurable: true });
				if (window.screen) {
					Object.defineProperty(window.screen, 'availLeft', { get: () => 0, configurable: true });
					Object.defineProperty(window.screen, 'availTop', { get: () => 0, configurable: true });
				}
			`,
		},
	}

	// 选择要测试的项（修改这里）
	// -1 = 全部, -2 = 前半(0-9), -3 = 后半(10-17)
	// -4 = 0-4, -5 = 5-9, -6 = 5-6, -7 = 7-9
	testIndex := 8 // ⬅️ 测 8 = Error.prepareStackTrace

	ctx := context.Background()

	opts := &browser.ConnectOptions{
		Headless: false,
		Args:     []string{"--window-size=1280,720"},
	}

	fmt.Println("🚀 启动浏览器...")
	instance, err := browser.Connect(ctx, opts)
	if err != nil {
		log.Fatalf("❌ 连接失败: %v", err)
	}
	defer instance.Close()

	page := instance.Page()

	// 注入测试脚本（使用 addScriptToEvaluateOnNewDocument 方式，与实际一致）
	var testScript string
	if testIndex >= 0 && testIndex < len(stealthItems) {
		item := stealthItems[testIndex]
		fmt.Printf("🧪 测试项: %s\n", item.Name)
		testScript = fmt.Sprintf(`(() => { 'use strict'; %s })();`, item.Script)
	} else if testIndex == -2 {
		fmt.Println("🧪 测试前半部分 (0-9)")
		for i := 0; i <= 9 && i < len(stealthItems); i++ {
			fmt.Printf("  - %s\n", stealthItems[i].Name)
			testScript += fmt.Sprintf(`(() => { 'use strict'; %s })();`, stealthItems[i].Script)
		}
	} else if testIndex == -3 {
		fmt.Println("🧪 测试后半部分 (10-17)")
		for i := 10; i < len(stealthItems); i++ {
			fmt.Printf("  - %s\n", stealthItems[i].Name)
			testScript += fmt.Sprintf(`(() => { 'use strict'; %s })();`, stealthItems[i].Script)
		}
	} else if testIndex == -4 {
		fmt.Println("🧪 测试 0-4")
		for i := 0; i <= 4 && i < len(stealthItems); i++ {
			fmt.Printf("  - %s\n", stealthItems[i].Name)
			testScript += fmt.Sprintf(`(() => { 'use strict'; %s })();`, stealthItems[i].Script)
		}
	} else if testIndex == -5 {
		fmt.Println("🧪 测试 5-9")
		for i := 5; i <= 9 && i < len(stealthItems); i++ {
			fmt.Printf("  - %s\n", stealthItems[i].Name)
			testScript += fmt.Sprintf(`(() => { 'use strict'; %s })();`, stealthItems[i].Script)
		}
	} else {
		fmt.Println("🧪 测试所有项")
		for _, item := range stealthItems {
			testScript += fmt.Sprintf(`(() => { 'use strict'; %s })();`, item.Script)
		}
	}

	// 使用 CustomCDPPage 的 addScriptToEvaluateOnNewDocument
	customPage, ok := page.(*browser.CustomCDPPage)
	if ok {
		fmt.Println("📜 使用 addScriptToEvaluateOnNewDocument 注入...")
		if err := customPage.AddScriptToEvaluateOnNewDocument(testScript); err != nil {
			log.Printf("⚠️ 脚本注入失败: %v", err)
		}
	} else {
		fmt.Println("⚠️ 回退到 Evaluate 注入...")
		if _, err := page.Evaluate(testScript); err != nil {
			log.Printf("⚠️ 脚本注入失败: %v", err)
		}
	}

	// 导航到抖音
	fmt.Println("\n📂 导航到抖音...")
	if err := page.Navigate("https://www.douyin.com/user/self"); err != nil {
		log.Fatalf("❌ 导航失败: %v", err)
	}

	fmt.Println("\n👀 观察二维码是否显示...")
	fmt.Println("⏳ 等待 60 秒...")
	time.Sleep(60 * time.Second)

	fmt.Println("\n✅ 测试完成!")
}
