//go:build ignore
// +build ignore

package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/r0vx/puppeteer-real-browser-go/pkg/browser"
)

// 快手充值页面选择器
const (
	// 自定义金额按钮
	CustomAmountBtn = "#root > div > main > div > div > div.FpRKJGk3XAIr1D8qDACP > div.VK_V5n2P1cHvyLgugFEq > div:nth-child(2) > div.Y5lzdW0AOqa22YyzwvdA.Pl2xxlIxtItKADOqfMTE > div"
	// 金额输入框
	AmountInput = "#root > div > div.yO8kMoLepIjfM5ZIRM6Q > div > div.n1SnVijkShoQBxLXjI7j > div > input"
	// 确定按钮
	ConfirmBtn = "#root > div > div.yO8kMoLepIjfM5ZIRM6Q > div > div.Pc1O3eZm5SMdnuaFF3rk > button.JnjN1NsuzX0e7meKTHb8.XSrBZ0vfjO5Y1lyu05IU"

	// 账号输入框
	AccountInput = "#root > div > main > div > div > div.w3F19lRc539xYVKSHMLW > div > div > input"
	// 账号确定按钮
	AccountConfirmBtn = "#root > div > main > div > div > div.w3F19lRc539xYVKSHMLW > div > div > button"
)

func main() {
	fmt.Println("🎯 快手充值页面测试")
	fmt.Println("================================")
	fmt.Println("测试页面: https://pay.ssl.kuaishou.com/pay")
	fmt.Println()

	ctx := context.Background()

	opts := &browser.ConnectOptions{
		Headless:     false,
		UseCustomCDP: true,
		Turnstile:    false,
		Args: []string{
			"--window-size=1920,1080",
		},
	}

	fmt.Println("🚀 启动浏览器...")
	instance, err := browser.Connect(ctx, opts)
	if err != nil {
		log.Fatalf("❌ 连接失败: %v", err)
	}
	defer instance.Close()

	page := instance.Page()
	selectorPage, ok := page.(browser.PageWithSelector)
	if !ok {
		log.Fatal("❌ 页面不支持选择器方法")
	}

	fmt.Println("📂 导航到快手充值页面...")
	if err := page.Navigate("https://pay.ssl.kuaishou.com/pay"); err != nil {
		log.Fatalf("❌ 导航失败: %v", err)
	}
	time.Sleep(3 * time.Second)

	title, _ := page.GetTitle()
	fmt.Printf("✅ 页面标题: %s\n", title)

	// 步骤0: 输入账号
	fmt.Println("\n📌 步骤0: 输入快手账号")
	fmt.Printf("   选择器: %s\n", AccountInput)

	if err := selectorPage.RealClickSelector(AccountInput); err != nil {
		fmt.Printf("⚠️ 点击账号输入框失败: %v (可能页面结构不同)\n", err)
	} else {
		fmt.Println("✅ 点击账号输入框成功!")
		time.Sleep(300 * time.Millisecond)

		if err := selectorPage.RealSendKeys("6666666"); err != nil {
			fmt.Printf("❌ 输入账号失败: %v\n", err)
		} else {
			fmt.Println("✅ 输入账号成功: 6666666")
		}

		time.Sleep(500 * time.Millisecond)
		saveScreenshot(page, "step0_after_account.png")

		// 点击账号确定按钮
		fmt.Println("\n📌 点击账号确定按钮")
		fmt.Printf("   选择器: %s\n", AccountConfirmBtn)

		if err := selectorPage.RealClickSelector(AccountConfirmBtn); err != nil {
			fmt.Printf("⚠️ 点击账号确定失败: %v\n", err)
		} else {
			fmt.Println("✅ 点击账号确定成功!")
		}
		time.Sleep(2 * time.Second)
		saveScreenshot(page, "step0_after_confirm.png")
	}

	// 步骤1: 点击自定义金额
	fmt.Println("\n📌 步骤1: RealClickSelector 点击自定义金额")
	fmt.Printf("   选择器: %s\n", CustomAmountBtn)

	if err := selectorPage.RealClickSelector(CustomAmountBtn); err != nil {
		fmt.Printf("❌ 点击失败: %v\n", err)
		saveScreenshot(page, "error_step1.png")
		return
	}
	fmt.Println("✅ 点击自定义金额成功!")
	time.Sleep(1 * time.Second)
	saveScreenshot(page, "step1_after_click.png")

	// 步骤2: 点击输入框并输入金额
	fmt.Println("\n📌 步骤2: 点击输入框并输入金额 100")
	fmt.Printf("   选择器: %s\n", AmountInput)

	if err := selectorPage.RealClickSelector(AmountInput); err != nil {
		fmt.Printf("❌ 点击输入框失败: %v\n", err)
		saveScreenshot(page, "error_step2.png")
		return
	}
	fmt.Println("✅ 点击输入框成功!")

	time.Sleep(300 * time.Millisecond)

	if err := selectorPage.RealSendKeys("100"); err != nil {
		fmt.Printf("❌ RealSendKeys 输入失败: %v\n", err)
		saveScreenshot(page, "error_step2_input.png")
		return
	}
	fmt.Println("✅ 输入金额成功!")
	saveScreenshot(page, "step2_after_input.png")

	// 步骤3: 点击确定按钮
	fmt.Println("\n📌 步骤3: RealClickSelector 点击确定按钮")
	fmt.Printf("   选择器: %s\n", ConfirmBtn)

	time.Sleep(500 * time.Millisecond)

	if err := selectorPage.RealClickSelector(ConfirmBtn); err != nil {
		fmt.Printf("❌ 点击确定失败: %v\n", err)
		saveScreenshot(page, "error_step3.png")
		return
	}
	fmt.Println("✅ 点击确定成功!")

	time.Sleep(2 * time.Second)
	saveScreenshot(page, "step3_after_confirm.png")

	fmt.Println("\n🎉 快手充值测试完成!")

	fmt.Println("\n⏳ 保持浏览器打开 10 秒...")
	// time.Sleep(10 * time.Second)
	fmt.Println("✅ 测试结束!")
}

// saveScreenshot 保存截图
func saveScreenshot(page browser.Page, filename string) {
	if screenshot, err := page.Screenshot(); err == nil {
		os.WriteFile(filename, screenshot, 0644)
		fmt.Printf("   已保存截图: %s\n", filename)
	}
}
