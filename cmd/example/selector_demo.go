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

// 抖音充值页面选择器
const (
	// 自定义金额按钮
	CustomAmountBtn = "#extra"
	// 金额输入框（弹出层）
	AmountInput = "body > div:nth-child(20) > div > div.ant-popover-content > div > div > div > span > input"
	// 确定按钮
	ConfirmBtn = "body > div:nth-child(20) > div > div.ant-popover-content > div > div > div > div > button.ant-btn.css-18h3yg2.ant-btn-primary.combo_list_custom_popover_content_button_group_confirm_button-znqIQc"
)

func main() {
	fmt.Println("🎯 UseCustomCDP: true 功能测试")
	fmt.Println("================================")
	fmt.Println("测试页面: https://pay.ssl.kuaishou.com/pay")
	fmt.Println()

	ctx := context.Background()

	opts := &browser.ConnectOptions{
		Headless:     false,
		UseCustomCDP: true, // 测试自定义 CDP 模式
		Turnstile:    false,
		Args: []string{
			"--window-size=1280,900",
		},
	}

	fmt.Println("🚀 启动浏览器 (UseCustomCDP: true)...")
	instance, err := browser.Connect(ctx, opts)
	if err != nil {
		log.Fatalf("❌ 连接失败: %v", err)
	}
	defer instance.Close()

	page := instance.Page()

	fmt.Println("📂 导航到抖音充值页面...")
	if err := page.Navigate("https://www.douyin.com/pay"); err != nil {
		log.Fatalf("❌ 导航失败: %v", err)
	}
	time.Sleep(3 * time.Second)

	title, _ := page.GetTitle()
	fmt.Printf("✅ 页面标题: %s\n", title)

	// 测试点击自定义金额
	fmt.Println("\n========== 测试：点击自定义金额 ==========")
	testClickCustomAmount(page)

	// 截图
	fmt.Println("\n📸 保存最终截图...")
	if screenshot, err := page.Screenshot(); err == nil {
		os.WriteFile("final_result.png", screenshot, 0644)
		fmt.Println("✅ 已保存: final_result.png")
	}

	fmt.Println("\n⏳ 保持浏览器打开 10 秒...")
	time.Sleep(10 * time.Second)
	fmt.Println("✅ 测试完成!")
}

// 测试点击自定义金额
func testClickCustomAmount(page browser.Page) {
	selectorPage, ok := page.(browser.PageWithSelector)
	if !ok {
		fmt.Println("❌ 不支持选择器方法")
		return
	}

	// 步骤1: RealClickSelector 点击自定义金额
	fmt.Println("\n📌 步骤1: RealClickSelector 点击 #extra")
	if err := selectorPage.RealClickSelector(CustomAmountBtn); err != nil {
		fmt.Printf("❌ 点击失败: %v\n", err)
		return
	}
	fmt.Println("✅ 点击自定义金额成功!")
	time.Sleep(1 * time.Second)

	// 截图
	if screenshot, err := page.Screenshot(); err == nil {
		os.WriteFile("step1_after_click_extra.png", screenshot, 0644)
		fmt.Println("   已保存截图: step1_after_click_extra.png")
	}

	// 步骤2: RealClickSelector 点击输入框 + RealType 输入金额
	fmt.Println("\n📌 步骤2: 点击输入框并输入金额 100")
	fmt.Printf("   选择器: %s\n", AmountInput)

	if err := selectorPage.RealClickSelector(AmountInput); err != nil {
		fmt.Printf("❌ 点击输入框失败: %v\n", err)
		return
	}
	fmt.Println("✅ 点击输入框成功!")

	time.Sleep(300 * time.Millisecond)

	if err := selectorPage.RealSendKeys("100"); err != nil {
		fmt.Printf("❌ RealSendKeys 输入失败: %v\n", err)
		return
	}
	fmt.Println("✅ 输入金额成功!")

	// 截图
	if screenshot, err := page.Screenshot(); err == nil {
		os.WriteFile("step2_after_input.png", screenshot, 0644)
		fmt.Println("   已保存截图: step2_after_input.png")
	}

	// 步骤3: RealClickSelector 点击确定按钮
	fmt.Println("\n📌 步骤3: RealClickSelector 点击确定按钮")
	fmt.Printf("   选择器: %s\n", ConfirmBtn)

	time.Sleep(500 * time.Millisecond)

	if err := selectorPage.RealClickSelector(ConfirmBtn); err != nil {
		fmt.Printf("❌ 点击确定失败: %v\n", err)
	} else {
		fmt.Println("✅ 点击确定成功!")
	}

	time.Sleep(2 * time.Second)

	// 截图
	if screenshot, err := page.Screenshot(); err == nil {
		os.WriteFile("step3_after_confirm.png", screenshot, 0644)
		fmt.Println("   已保存截图: step3_after_confirm.png")
	}

	fmt.Println("\n🎉 测试完成!")
}
