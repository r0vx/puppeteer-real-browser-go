package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/HNRow/puppeteer-real-browser-go/pkg/browser"
)

func main() {
	fmt.Println("🔍 Chrome企业策略检查")
	fmt.Println("====================")

	ctx := context.Background()

	opts := &browser.ConnectOptions{
		Headless:       false,
		PersistProfile: true,
		ProfileName:    "policy_check",
		Args: []string{
			"--start-maximized",
		},
	}

	fmt.Println("🚀 启动Chrome...")
	instance, err := browser.Connect(ctx, opts)
	if err != nil {
		log.Fatalf("❌ Chrome启动失败: %v", err)
	}
	defer instance.Close()

	fmt.Println("✅ Chrome启动成功")
	time.Sleep(3 * time.Second)

	page := instance.Page()

	// 检查chrome://policy/页面
	fmt.Println("📋 导航到chrome://policy/...")
	if err := page.Navigate("chrome://policy/"); err != nil {
		log.Printf("❌ 导航失败: %v", err)
		return
	}

	time.Sleep(3 * time.Second)

	// 检查企业策略
	fmt.Println("🔍 检查企业策略...")
	policyResult, err := page.Evaluate(`
		(() => {
			try {
				// 检查页面内容
				const bodyText = document.body.innerText;
				
				// 查找策略相关信息
				const hasExtensionPolicies = bodyText.includes('ExtensionInstallBlacklist') ||
											 bodyText.includes('ExtensionInstallWhitelist') ||
											 bodyText.includes('ExtensionSettings') ||
											 bodyText.includes('ExtensionInstallForcelist');
				
				const hasDeveloperModePolicy = bodyText.includes('DeveloperToolsDisabled') ||
											  bodyText.includes('ExtensionInstallBlacklist');
											  
				const isManaged = bodyText.includes('managed') || 
								 bodyText.includes('Your browser is managed') ||
								 bodyText.includes('This browser is managed');
								 
				// 尝试找到具体的策略值
				const policyEntries = [];
				const tables = document.querySelectorAll('table');
				
				for (const table of tables) {
					const rows = table.querySelectorAll('tr');
					for (const row of rows) {
						const cells = row.querySelectorAll('td');
						if (cells.length >= 2) {
							const policyName = cells[0].textContent.trim();
							const policyValue = cells[1].textContent.trim();
							if (policyName.toLowerCase().includes('extension') ||
								policyName.toLowerCase().includes('developer')) {
								policyEntries.push({
									name: policyName,
									value: policyValue
								});
							}
						}
					}
				}
				
				return {
					success: true,
					pageTitle: document.title,
					url: location.href,
					isManaged: isManaged,
					hasExtensionPolicies: hasExtensionPolicies,
					hasDeveloperModePolicy: hasDeveloperModePolicy,
					policyEntries: policyEntries,
					bodyPreview: bodyText.slice(0, 500) // 页面内容预览
				};
			} catch (error) {
				return { success: false, error: error.message };
			}
		})()
	`)

	if err != nil {
		fmt.Printf("❌ 策略检查失败: %v\n", err)
	} else {
		fmt.Printf("📊 企业策略检查结果:\n")
		if resultMap, ok := policyResult.(map[string]interface{}); ok {
			fmt.Printf("  ✅ 成功: %v\n", resultMap["success"])
			fmt.Printf("  🔗 URL: %v\n", resultMap["url"])
			fmt.Printf("  📋 标题: %v\n", resultMap["pageTitle"])
			fmt.Printf("  🏢 受管理: %v\n", resultMap["isManaged"])
			fmt.Printf("  🎯 有扩展策略: %v\n", resultMap["hasExtensionPolicies"])
			fmt.Printf("  🔧 有开发者模式策略: %v\n", resultMap["hasDeveloperModePolicy"])
			
			if policies := resultMap["policyEntries"]; policies != nil {
				fmt.Printf("  📝 相关策略: %v\n", policies)
			}
			
			if preview := resultMap["bodyPreview"]; preview != nil {
				fmt.Printf("  📄 页面内容预览:\n%v\n", preview)
			}
		}
	}

	// 也检查chrome://version/页面获取更多信息
	fmt.Println("\n📋 导航到chrome://version/...")
	if err := page.Navigate("chrome://version/"); err != nil {
		log.Printf("❌ 导航失败: %v", err)
	} else {
		time.Sleep(2 * time.Second)
		
		versionResult, err := page.Evaluate(`
			(() => {
				const bodyText = document.body.innerText;
				return {
					chromeVersion: bodyText.match(/Google Chrome\s+(\d+\.\d+\.\d+\.\d+)/)?.[1] || 'unknown',
					isOfficialBuild: bodyText.includes('Official Build'),
					bodyPreview: bodyText.slice(0, 300)
				};
			})()
		`)
		
		if err == nil {
			fmt.Printf("🔍 Chrome版本信息: %v\n", versionResult)
		}
	}

	fmt.Println("\n💡 分析:")
	fmt.Println("  1. 如果Chrome受企业策略管理，这可能解释扩展加载失败")
	fmt.Println("  2. ExtensionInstallBlacklist策略可能阻止未打包扩展")
	fmt.Println("  3. DeveloperToolsDisabled可能阻止开发者模式")

	fmt.Println("\n⏳ 保持浏览器开启30秒供手动检查...")
	time.Sleep(30 * time.Second)

	fmt.Println("✅ 检查完成")
}