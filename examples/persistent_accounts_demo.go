package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/HNRow/puppeteer-real-browser-go/pkg/browser"
)

func main() {
	fmt.Println("💾 Persistent Account Management Demo")
	fmt.Println("=====================================")

	ctx := context.Background()

	// 基础配置
	baseOptions := &browser.ConnectOptions{
		Headless:     false,
		UseCustomCDP: true, // 最大隐蔽性
		Turnstile:    true, // 自动解验证码
		Args: []string{
			"--start-maximized",
			"--disable-blink-features=AutomationControlled",
			"--exclude-switches=enable-automation",
		},
		PersistProfile: true, // 启用持久化
	}

	// 创建账号管理器
	manager := browser.NewAccountManager(baseOptions)
	defer manager.CloseAll()

	// 演示1: 创建多个持久化账号
	fmt.Println("\n👥 Demo 1: Creating Persistent Accounts")
	accounts := []struct {
		name    string
		website string
		proxy   *browser.ProxyConfig
	}{
		{"alice_ecommerce", "https://httpbin.org/cookies/set/user/alice", nil},
		//{"bob_social", "https://httpbin.org/cookies/set/user/bob", nil},
		//{"charlie_work", "https://httpbin.org/cookies/set/user/charlie", nil},
	}

	for _, acc := range accounts {
		// 为每个账号创建独立的配置
		accountOptions := &browser.ConnectOptions{
			Proxy: acc.proxy,
		}

		account, err := manager.CreateAccount(ctx, acc.name, accountOptions)
		if err != nil {
			log.Printf("Failed to create account %s: %v", acc.name, err)
			continue
		}

		fmt.Printf("  ✅ Created account: %s\n", acc.name)

		// 导航到对应网站并设置 Cookie
		page := account.Instance.Page()
		if err := page.Navigate(acc.website); err != nil {
			log.Printf("Failed to navigate for %s: %v", acc.name, err)
			continue
		}

		// 设置页面标题以便识别
		page.Evaluate(fmt.Sprintf(`document.title = '%s - Persistent Account'`, acc.name))
		fmt.Printf("    🌐 %s: Set cookies and navigated to %s\n", acc.name, acc.website)

		time.Sleep(1 * time.Second)
	}

	// 演示2: 验证数据持久化
	fmt.Println("\n💾 Demo 2: Testing Data Persistence")

	for _, acc := range accounts {
		account, exists := manager.GetAccount(acc.name)
		if !exists {
			continue
		}

		// 创建新页面检查之前设置的 Cookie
		page := account.Instance.Page()
		if err := page.Navigate("https://httpbin.org/cookies"); err != nil {
			continue
		}

		// 检查 Cookie 是否持久化
		result, err := page.Evaluate(`
			return {
				url: window.location.href,
				title: document.title,
				cookies: document.cookie,
				storage: localStorage.length,
				hasUserCookie: document.cookie.includes('user=')
			};
		`)

		if err == nil {
			fmt.Printf("  🔍 %s persistence check: %v\n", acc.name, result)
		}
	}

	// 演示3: 关闭并重新创建账号（模拟重启应用）
	fmt.Println("\n🔄 Demo 3: Simulating Application Restart")
	fmt.Println("  📝 Closing all accounts...")

	// 记录当前状态
	accountNames := manager.ListAccounts()

	// 关闭所有账号
	manager.CloseAll()
	fmt.Printf("  ✅ Closed %d accounts\n", len(accountNames))

	// 等待一会儿
	time.Sleep(2 * time.Second)

	// 重新创建账号（模拟应用重启后的恢复）
	fmt.Println("  🔄 Recreating accounts with persistent data...")

	for _, accName := range accountNames {
		account, err := manager.CreateAccount(ctx, accName, nil)
		if err != nil {
			log.Printf("Failed to recreate account %s: %v", accName, err)
			continue
		}

		fmt.Printf("    ♻️  Recreated: %s\n", accName)

		// 验证之前的数据是否还在
		page := account.Instance.Page()
		if err := page.Navigate("https://httpbin.org/cookies"); err != nil {
			continue
		}

		time.Sleep(2 * time.Second)

		result, err := page.Evaluate(`
			return {
				hasPreviousCookies: document.cookie.includes('user='),
				cookieCount: document.cookie.split(';').length,
				title: document.title
			};
		`)

		if err == nil {
			fmt.Printf("      🔍 Persistence verification: %v\n", result)
		}

		// 设置新的页面标题标识
		page.Evaluate(fmt.Sprintf(`document.title = '%s - Restored Account'`, accName))
	}

	// 演示4: 账号管理功能
	fmt.Println("\n⚙️  Demo 4: Account Management Features")

	fmt.Printf("  📊 Total accounts: %d\n", manager.GetAccountCount())
	fmt.Printf("  📋 Account list: %v\n", manager.ListAccounts())

	// 为每个账号显示用户数据目录信息
	for _, accName := range manager.ListAccounts() {
		account, exists := manager.GetAccount(accName)
		if exists {
			page := account.Instance.Page()

			// 获取用户数据信息
			if err := page.Navigate("https://httpbin.org/user-agent"); err == nil {
				page.Evaluate(fmt.Sprintf(`
					console.log('Account: %s');
					console.log('Profile Name: %s');
					console.log('User Data Directory: Profile-specific');
					document.title = '%s - Account Info';
				`, accName, account.ProfileName, accName))
			}
		}
	}

	// 演示5: 指纹浏览器集成
	fmt.Println("\n🎭 Demo 5: Fingerprint Integration")

	// 为每个账号应用不同的指纹
	fingerprints := map[string]string{
		"alice_ecommerce": "US Windows User",
		"bob_social":      "UK macOS User",
		"charlie_work":    "DE Linux User",
	}

	for accName, fingerprint := range fingerprints {
		account, exists := manager.GetAccount(accName)
		if !exists {
			continue
		}

		page := account.Instance.Page()

		// 应用指纹脚本
		fingerprintScript := fmt.Sprintf(`
			// 模拟不同的指纹信息
			Object.defineProperty(navigator, 'language', {
				get: () => '%s'
			});
			
			console.log('🎭 Applied fingerprint: %s');
			document.title = '%s - %s';
		`, getLanguageForFingerprint(fingerprint), fingerprint, accName, fingerprint)

		page.Evaluate(fingerprintScript)
		fmt.Printf("  🎭 %s: Applied %s fingerprint\n", accName, fingerprint)
	}

	// 使用说明
	fmt.Println("\n💡 Key Benefits Demonstrated:")
	fmt.Println("  ✅ Each account has its own Chrome process and user data directory")
	fmt.Println("  ✅ Cookies, localStorage, and browsing history are isolated per account")
	fmt.Println("  ✅ Data persists between application restarts")
	fmt.Println("  ✅ Each account can have different fingerprints and proxies")
	fmt.Println("  ✅ Independent extension support per account")

	fmt.Println("\n🔍 Manual Verification:")
	fmt.Println("  1. Check multiple browser windows - each is a separate account")
	fmt.Println("  2. Look at browser titles to identify accounts")
	fmt.Println("  3. Check DevTools > Application > Storage for isolated data")
	fmt.Println("  4. Restart this program to see data persistence")

	fmt.Printf("\n📁 User Data Directories are stored in: ~/.puppeteer-real-browser-go/profiles/\n")
	fmt.Printf("  - alice_ecommerce/\n")
	fmt.Printf("  - bob_social/\n")
	fmt.Printf("  - charlie_work/\n")

	fmt.Println("\n⏳ Keeping all accounts open for 60 seconds for inspection...")
	time.Sleep(600 * time.Second)

	fmt.Println("✅ Persistent Account Management Demo completed!")
}

// getLanguageForFingerprint returns appropriate language for fingerprint
func getLanguageForFingerprint(fingerprint string) string {
	switch {
	case contains(fingerprint, "US"):
		return "en-US"
	case contains(fingerprint, "UK"):
		return "en-GB"
	case contains(fingerprint, "DE"):
		return "de-DE"
	default:
		return "en-US"
	}
}

// contains checks if string contains substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || (len(s) > len(substr) &&
		(s[:len(substr)] == substr || s[len(s)-len(substr):] == substr ||
			indexOf(s, substr) >= 0)))
}

// indexOf finds the index of substring in string
func indexOf(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}
