package main

import (
	"context"
	"fmt"
	"log"

	"github.com/HNRow/puppeteer-real-browser-go/pkg/browser"
)

func main() {
	fmt.Println("🔍 Chrome启动参数对比")
	fmt.Println("=====================")

	ctx := context.Background()

	fmt.Println("\n=== 测试1: 手动指定扩展路径的启动参数 ===")
	
	// 测试1: 手动指定扩展路径
	opts1 := &browser.ConnectOptions{
		Headless:       false,
		PersistProfile: true,
		ProfileName:    "flags_manual_test",
		Extensions: []string{
			"examples/path/Extensions/kfjglmgfjedhhcddpfgfogkahmenikan/1.0_0",
			"examples/path/Extensions/mcohilncbfahbmgdjkbpemcciiolgcge/3.66.10_0",
		},
	}

	launcher1 := browser.NewChromeLauncher()
	chrome1, err := launcher1.Launch(ctx, opts1)
	if err != nil {
		log.Printf("手动加载失败: %v", err)
	} else {
		fmt.Println("📋 手动加载的Chrome启动参数:")
		for i, flag := range chrome1.Flags {
			if flag == "--load-extension" && i+1 < len(chrome1.Flags) {
				fmt.Printf("  [%d] %s\n", i, flag)
				fmt.Printf("  [%d] %s\n", i+1, chrome1.Flags[i+1])
			} else if flag != "--load-extension" {
				fmt.Printf("  [%d] %s\n", i, flag)
			}
		}
		chrome1.Kill()
	}

	fmt.Println("\n=== 测试2: 自动加载扩展的启动参数 ===")
	
	// 测试2: 自动加载扩展
	opts2 := &browser.ConnectOptions{
		Headless:                  false,
		AutoLoadDefaultExtensions: true,
		PersistProfile:            true,
		ProfileName:               "flags_auto_test",
	}

	launcher2 := browser.NewChromeLauncher()
	chrome2, err := launcher2.Launch(ctx, opts2)
	if err != nil {
		log.Printf("自动加载失败: %v", err)
	} else {
		fmt.Println("📋 自动加载的Chrome启动参数:")
		for i, flag := range chrome2.Flags {
			if flag == "--load-extension" && i+1 < len(chrome2.Flags) {
				fmt.Printf("  [%d] %s\n", i, flag)
				fmt.Printf("  [%d] %s\n", i+1, chrome2.Flags[i+1])
			} else if flag != "--load-extension" {
				fmt.Printf("  [%d] %s\n", i, flag)
			}
		}
		chrome2.Kill()
	}

	fmt.Println("\n✅ 启动参数对比完成")
}