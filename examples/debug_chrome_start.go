package main

import (
	"context"
	"fmt"
	"log"
	"path/filepath"

	"github.com/HNRow/puppeteer-real-browser-go/pkg/browser"
)

func main() {
	fmt.Println("🔍 Chrome启动调试")
	fmt.Println("================")

	ctx := context.Background()

	// 获取扩展路径
	ext1, _ := filepath.Abs("examples/path/Extensions/kfjglmgfjedhhcddpfgfogkahmenikan/1.0_0")
	ext2, _ := filepath.Abs("examples/path/Extensions/mcohilncbfahbmgdjkbpemcciiolgcge/3.66.10_0")

	fmt.Printf("📂 扩展1: %s\n", ext1)
	fmt.Printf("📂 扩展2: %s\n", ext2)

	// 最基本的配置 - 不使用 AutoLoadDefaultExtensions
	opts := &browser.ConnectOptions{
		Headless:       false,
		PersistProfile: true,
		ProfileName:    "basic_start",
		// 直接指定扩展而不是使用 AutoLoadDefaultExtensions
		Extensions: []string{ext1, ext2},
		Args: []string{
			"--no-first-run",
			"--start-maximized",
		},
	}

	fmt.Println("🚀 尝试最基本配置启动Chrome...")
	instance, err := browser.Connect(ctx, opts)
	if err != nil {
		fmt.Printf("❌ 基本配置启动失败: %v\n", err)
		
		// 尝试更基本的配置
		fmt.Println("\n🚀 尝试超基本配置（无扩展）...")
		opts2 := &browser.ConnectOptions{
			Headless:       false,
			PersistProfile: true,
			ProfileName:    "ultra_basic",
			Args: []string{
				"--no-first-run",
			},
		}
		
		instance2, err2 := browser.Connect(ctx, opts2)
		if err2 != nil {
			log.Fatalf("❌ 连超基本配置都无法启动: %v", err2)
		} else {
			fmt.Println("✅ 超基本配置启动成功")
			instance2.Close()
		}
		return
	}
	defer instance.Close()

	fmt.Println("✅ Chrome启动成功")
	fmt.Println("🎯 这表明扩展加载配置本身是可以工作的")
}