package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/HNRow/puppeteer-real-browser-go/pkg/browser"
)

func main() {
	fmt.Println("🏢 基于策略的扩展安装演示")
	fmt.Println("========================")

	ctx := context.Background()

	// 创建临时策略文件目录
	policyDir := "/tmp/chrome-policies"
	os.MkdirAll(policyDir, 0755)

	fmt.Printf("📁 策略目录: %s\n", policyDir)

	// 创建Chrome企业策略文件
	// 这是Chrome官方支持的扩展安装方法
	policy := map[string]interface{}{
		"ExtensionSettings": map[string]interface{}{
			"*": map[string]interface{}{
				"installation_mode": "allowed", // 默认允许安装扩展
			},
			// Discord Token Login扩展 - 使用其manifest中的key生成的ID
			"kfjglmgfjedhhcddpfgfogkahmenikan": map[string]interface{}{
				"installation_mode": "force_installed",
				"update_url":       "file://" + filepath.Join(os.Getenv("PWD"), "examples/path/Extensions/kfjglmgfjedhhcddpfgfogkahmenikan/1.0_0"),
			},
			// OKX Wallet扩展
			"mcohilncbfahbmgdjkbpemcciiolgcge": map[string]interface{}{
				"installation_mode": "force_installed", 
				"update_url":       "file://" + filepath.Join(os.Getenv("PWD"), "examples/path/Extensions/mcohilncbfahbmgdjkbpemcciiolgcge/3.66.10_0"),
			},
		},
		"ExtensionInstallForcelist": []string{
			// 另一种策略方法：直接指定要强制安装的扩展ID
			"kfjglmgfjedhhcddpfgfogkahmenikan",
			"mcohilncbfahbmgdjkbpemcciiolgcge",
		},
	}

	// 将策略写入JSON文件
	policyFile := filepath.Join(policyDir, "policies.json")
	policyJSON, _ := json.MarshalIndent(policy, "", "  ")
	if err := os.WriteFile(policyFile, policyJSON, 0644); err != nil {
		log.Fatalf("❌ 策略文件创建失败: %v", err)
	}

	fmt.Printf("📝 策略文件已创建: %s\n", policyFile)
	fmt.Println("📋 策略内容预览:")
	fmt.Printf("%s\n", string(policyJSON))

	// Chrome配置，使用策略文件
	opts := &browser.ConnectOptions{
		Headless:       false,
		PersistProfile: true,
		ProfileName:    "policy_extensions",
		Args: []string{
			"--start-maximized",
			"--enable-extensions",
			// 使用企业策略的正确方法
			"--policy-file=" + policyFile,
			// 或者使用策略目录
			"--policy-directory=" + policyDir,
			// 禁用一些可能干扰策略的标志
			"--disable-features=ChromeWhatsNewUI",
		},
	}

	fmt.Println("\n🚀 使用企业策略启动Chrome...")
	instance, err := browser.Connect(ctx, opts)
	if err != nil {
		log.Fatalf("❌ Chrome启动失败: %v", err)
	}
	defer instance.Close()

	fmt.Println("✅ Chrome启动成功")
	time.Sleep(5 * time.Second)

	page := instance.Page()

	// 首先检查策略是否被应用
	fmt.Println("📋 检查chrome://policy/...")
	if err := page.Navigate("chrome://policy/"); err != nil {
		log.Printf("❌ 导航失败: %v", err)
	} else {
		time.Sleep(3 * time.Second)
		
		policyCheck, _ := page.Evaluate(`
			(() => {
				const bodyText = document.body.innerText;
				return {
					hasExtensionSettings: bodyText.includes('ExtensionSettings'),
					hasExtensionInstallForcelist: bodyText.includes('ExtensionInstallForcelist'),
					policyCount: document.querySelectorAll('table tr').length
				};
			})()
		`)
		
		fmt.Printf("🔍 策略应用检查: %v\n", policyCheck)
	}

	// 检查扩展页面
	fmt.Println("\n📋 导航到chrome://extensions/...")
	if err := page.Navigate("chrome://extensions/"); err != nil {
		log.Printf("❌ 导航失败: %v", err)
	} else {
		time.Sleep(3 * time.Second)
		
		extensionCheck, _ := page.Evaluate(`
			(() => {
				const manager = document.querySelector('extensions-manager');
				if (!manager || !manager.shadowRoot) {
					return { error: "无法访问扩展管理器" };
				}
				
				const items = manager.shadowRoot.querySelectorAll('extensions-item');
				const extensions = Array.from(items).map(item => ({
					name: item.shadowRoot?.querySelector('#name')?.textContent?.trim() || 'Unknown',
					id: item.id || 'unknown'
				}));
				
				return {
					extensionCount: items.length,
					extensions: extensions
				};
			})()
		`)
		
		fmt.Printf("📊 扩展检查结果: %v\n", extensionCheck)
	}

	fmt.Println("\n💡 说明:")
	fmt.Println("  ✅ 这种方法使用Chrome官方的企业策略机制")
	fmt.Println("  ✅ ExtensionSettings策略控制扩展安装模式")
	fmt.Println("  ✅ force_installed模式会自动安装并禁用用户移除")
	fmt.Println("  ⚠️  需要有效的策略文件和正确的扩展ID")

	fmt.Println("\n⏳ 保持浏览器开启60秒供检查...")
	time.Sleep(60 * time.Second)

	// 清理临时文件
	os.RemoveAll(policyDir)
	fmt.Println("🧹 已清理临时策略文件")
	fmt.Println("✅ 演示完成")
}