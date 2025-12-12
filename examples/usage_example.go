package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/HNRow/puppeteer-real-browser-go/pkg/browser"
)

func main() {
	fmt.Println("🎯 插件自动启用演示")
	fmt.Println("==================")

	ctx := context.Background()

	// ✅ 第一步：配置要自动安装的插件路径
	extensionPaths := []string{
		"./path/Extensions/kfjglmgfjedhhcddpfgfogkahmenikan", // 你的插件1
		"./path/Extensions/mcohilncbfahbmgdjkbpemcciiolgcge", // 你的插件2
	}

	// ✅ 第二步：创建基础配置（所有账号都会继承这些插件）
	baseOptions := &browser.ConnectOptions{
		Headless:       false,
		UseCustomCDP:   false,
		Turnstile:      true,
		Extensions:     extensionPaths,  // 🔑 关键：插件会自动安装到每个账号
		PersistProfile: true,            // 🔑 关键：启用持久化
		Args: []string{
			"--start-maximized",
			"--disable-blink-features=AutomationControlled",
			"--enable-extensions", // 确保启用插件功能
		},
	}

	// ✅ 第三步：创建账号管理器
	manager := browser.NewAccountManager(baseOptions)
	defer manager.CloseAll()

	// ✅ 第四步：创建不同的浏览器账号（插件会自动安装）
	accounts := []string{"电商账号", "工作账号", "个人账号"}

	for _, accountName := range accounts {
		fmt.Printf("\n🔄 创建账号: %s\n", accountName)
		
		// 创建账号时，系统会自动：
		// 1. 复制插件文件到该账号的用户数据目录
		// 2. 配置插件偏好设置为"启用"状态
		// 3. Chrome 启动时插件自动加载
		account, err := manager.CreateAccount(ctx, accountName, nil)
		if err != nil {
			log.Printf("创建账号失败: %v", err)
			continue
		}

		fmt.Printf("  ✅ 账号创建成功，插件已自动预装\n")

		// ✅ 第五步：验证插件是否已启用
		page := account.Instance.Page()
		
		// 导航到插件管理页面
		if err := page.Navigate("chrome://extensions/"); err != nil {
			log.Printf("无法访问插件页面: %v", err)
			continue
		}

		fmt.Printf("  🔍 正在验证插件状态...\n")
		time.Sleep(3 * time.Second)

		// 检查插件是否已安装并启用
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
				totalExtensions: extensions.length,
				enabledExtensions: extensions.filter(ext => ext.enabled).length,
				extensions: extensions
			};
		`)

		if err == nil {
			fmt.Printf("  📊 插件状态: %v\n", result)
		}

		// 设置页面标题以便识别
		page.Evaluate(fmt.Sprintf(`document.title = '%s - 插件已启用'`, accountName))
	}

	fmt.Println("\n🎉 所有账号创建完成！")
	fmt.Println("\n💡 验证方法:")
	fmt.Println("  1. 查看打开的多个浏览器窗口")
	fmt.Println("  2. 每个窗口代表一个独立账号")
	fmt.Println("  3. 在任一窗口访问 chrome://extensions/")
	fmt.Println("  4. 应该看到你的插件已安装并启用")
	fmt.Println("  5. 每个账号的插件数据完全隔离")

	fmt.Println("\n⏳ 保持浏览器打开 60 秒供检查...")
	time.Sleep(60 * time.Second)

	fmt.Println("✅ 演示完成！")
}