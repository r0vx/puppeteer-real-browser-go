package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/HNRow/puppeteer-real-browser-go/pkg/browser"
)

func main() {
	fmt.Println("🔍 查看CRX文件的Chrome启动参数")
	fmt.Println("==============================")

	ctx := context.Background()

	// 使用唯一的用户名
	profileName := "crx_flags_" + fmt.Sprintf("%d", time.Now().Unix())
	opts := &browser.ConnectOptions{
		Headless:                  false,
		AutoLoadDefaultExtensions: true,  // 使用.crx文件
		PersistProfile:            true,
		ProfileName:               profileName,
	}

	fmt.Printf("👤 测试用户: %s\n", profileName)

	// 启动浏览器获取启动参数
	fmt.Println("\n🔧 正在启动Chrome...")
	
	launcher := browser.NewChromeLauncher()
	chromeProcess, err := launcher.Launch(ctx, opts)
	if err != nil {
		log.Fatalf("Chrome启动失败: %v", err)
	}
	defer chromeProcess.Kill()

	fmt.Printf("✅ Chrome启动成功 (PID: %d)\n", chromeProcess.PID)
	fmt.Printf("🔗 调试端口: %d\n", chromeProcess.Port)

	fmt.Println("\n📋 关键Chrome启动参数:")
	for i, flag := range chromeProcess.Flags {
		// 只显示扩展相关的参数
		if flag == "--load-extension" && i+1 < len(chromeProcess.Flags) {
			fmt.Printf("  📦 %s\n", flag)
			fmt.Printf("      %s\n", chromeProcess.Flags[i+1])
		} else if flag == "--enable-extensions" {
			fmt.Printf("  ✅ %s\n", flag)
		} else if flag == "--user-data-dir" && i+1 < len(chromeProcess.Flags) {
			fmt.Printf("  📁 %s=%s\n", flag, chromeProcess.Flags[i+1])
		}
	}

	fmt.Println("\n⏳ 等待5秒...")
	time.Sleep(5 * time.Second)

	fmt.Println("✅ 参数检查完成")
}