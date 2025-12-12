package main

import (
	"context"
	"fmt"
	"log"

	"github.com/HNRow/puppeteer-real-browser-go/pkg/browser"
)

func main() {
	fmt.Println("🔍 基本Chrome启动测试")
	fmt.Println("===================")

	ctx := context.Background()

	// 最基本配置
	opts := &browser.ConnectOptions{
		Headless:       false,
		PersistProfile: true,
		ProfileName:    "basic_test",
		Args:           []string{"--start-maximized"},
	}

	fmt.Println("🚀 启动Chrome（无扩展）...")
	instance, err := browser.Connect(ctx, opts)
	if err != nil {
		log.Fatalf("❌ Chrome启动失败: %v", err)
	}
	defer instance.Close()

	fmt.Println("✅ Chrome启动成功")
}