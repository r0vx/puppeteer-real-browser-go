package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/HNRow/puppeteer-real-browser-go/pkg/browser"
)

func main() {
	fmt.Println("🔧 调试Chrome启动参数")
	fmt.Println("====================")

	ctx := context.Background()

	// 使用唯一的用户名
	profileName := "persistence_test_1754542063"
	opts := &browser.ConnectOptions{
		Headless:                  false,
		AutoLoadDefaultExtensions: true,
		PersistProfile:            true,
		ProfileName:               profileName,
	}

	fmt.Printf("👤 测试用户: %s\n", profileName)

	// 启动浏览器
	fmt.Println("\n🔧 正在启动Chrome...")

	// 创建launcher来获取Chrome flags
	launcher := browser.NewChromeLauncher()
	chromeProcess, err := launcher.Launch(ctx, opts)
	if err != nil {
		log.Fatalf("Chrome启动失败: %v", err)
	}
	defer chromeProcess.Kill()

	fmt.Printf("✅ Chrome启动成功 (PID: %d)\n", chromeProcess.PID)
	fmt.Printf("🔗 调试端口: %d\n", chromeProcess.Port)

	fmt.Println("\n📋 Chrome启动参数:")
	for i, flag := range chromeProcess.Flags {
		if flag == "--load-extension" && i+1 < len(chromeProcess.Flags) {
			fmt.Printf("  [%d] %s\n", i, flag)
			fmt.Printf("  [%d] %s\n", i+1, chromeProcess.Flags[i+1])
		} else if flag != chromeProcess.Flags[len(chromeProcess.Flags)-1] && chromeProcess.Flags[i+1] != "--load-extension" {
			fmt.Printf("  [%d] %s\n", i, flag)
		}
	}

	fmt.Println("\n⏳ 等待5秒...")
	time.Sleep(5 * time.Second)

	fmt.Println("✅ 调试完成")
}
