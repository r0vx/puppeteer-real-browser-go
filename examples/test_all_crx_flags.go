package main

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/HNRow/puppeteer-real-browser-go/pkg/browser"
)

func main() {
	fmt.Println("🔍 显示所有CRX相关的Chrome启动参数")
	fmt.Println("==================================")

	ctx := context.Background()

	opts := &browser.ConnectOptions{
		Headless:                  false,
		AutoLoadDefaultExtensions: true,  // 使用.crx文件
		PersistProfile:            true,
		ProfileName:               "all_crx_flags",
	}

	launcher := browser.NewChromeLauncher()
	chromeProcess, err := launcher.Launch(ctx, opts)
	if err != nil {
		log.Fatalf("Chrome启动失败: %v", err)
	}
	defer chromeProcess.Kill()

	fmt.Printf("✅ Chrome启动成功\n")

	fmt.Println("\n📋 所有Chrome启动参数:")
	for i, flag := range chromeProcess.Flags {
		// 高亮显示扩展相关参数
		if strings.Contains(flag, "extension") || strings.Contains(flag, ".crx") || flag == "--load-extension" {
			fmt.Printf("  🎯 [%d] %s\n", i, flag)
		} else {
			fmt.Printf("     [%d] %s\n", i, flag)
		}
	}

	fmt.Println("\n⏳ 等待3秒...")
	time.Sleep(3 * time.Second)

	fmt.Println("✅ 完成")
}