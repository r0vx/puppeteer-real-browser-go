package main

import (
	"context"
	"fmt"
	"log"
	"path/filepath"
	"strings"
	"time"

	"github.com/HNRow/puppeteer-real-browser-go/pkg/browser"
)

func main() {
	fmt.Println("🔍 验证双标志设置")
	fmt.Println("================")

	ctx := context.Background()

	// 获取扩展路径
	ext1, _ := filepath.Abs("examples/path/Extensions/kfjglmgfjedhhcddpfgfogkahmenikan/1.0_0")
	ext2, _ := filepath.Abs("examples/path/Extensions/mcohilncbfahbmgdjkbpemcciiolgcge/3.66.10_0")

	opts := &browser.ConnectOptions{
		Headless:       false,
		PersistProfile: true,
		ProfileName:    "verify_flags4",
		Extensions:     []string{ext1, ext2},
		Args:           []string{"--start-maximized"},
	}

	// 直接启动Chrome launcher来获取参数
	launcher := browser.NewChromeLauncher()
	chrome, err := launcher.Launch(ctx, opts)
	if err != nil {
		log.Fatalf("Chrome启动失败: %v", err)
	}
	defer chrome.Kill()

	fmt.Printf("✅ Chrome PID: %d\n", chrome.PID)

	fmt.Println("\n🔍 查找关键标志:")

	hasLoadExtension := false
	hasDisableExtensionsExcept := false
	loadExtensionValue := ""
	disableExtensionsExceptValue := ""

	for i, flag := range chrome.Flags {
		if strings.HasPrefix(flag, "--load-extension=") {
			hasLoadExtension = true
			loadExtensionValue = strings.TrimPrefix(flag, "--load-extension=")
			fmt.Printf("  [%d] 🎯 找到 --load-extension\n", i)
			fmt.Printf("      📂 值: %s\n", loadExtensionValue)
		}

		if strings.HasPrefix(flag, "--disable-extensions-except=") {
			hasDisableExtensionsExcept = true
			disableExtensionsExceptValue = strings.TrimPrefix(flag, "--disable-extensions-except=")
			fmt.Printf("  [%d] 🎯 找到 --disable-extensions-except\n", i)
			fmt.Printf("      📂 值: %s\n", disableExtensionsExceptValue)
		}
	}

	fmt.Println("\n📊 检查结果:")
	if hasLoadExtension {
		fmt.Println("  ✅ --load-extension 已设置")
		paths := strings.Split(loadExtensionValue, ",")
		fmt.Printf("  📦 加载扩展数量: %d\n", len(paths))
	} else {
		fmt.Println("  ❌ --load-extension 未设置")
	}

	if hasDisableExtensionsExcept {
		fmt.Println("  ✅ --disable-extensions-except 已设置")
		fmt.Printf("  📂 排除扩展: %s\n", disableExtensionsExceptValue)
	} else {
		fmt.Println("  ❌ --disable-extensions-except 未设置")
	}

	fmt.Println("\n💡 chromedp 要求的两个标志:")
	fmt.Printf("  --disable-extensions-except: %v\n", hasDisableExtensionsExcept)
	fmt.Printf("  --load-extension: %v\n", hasLoadExtension)

	if hasLoadExtension && hasDisableExtensionsExcept {
		fmt.Println("  🎉 两个标志都已正确设置！")
	} else {
		fmt.Println("  ⚠️ 缺少必需的标志")
	}

	time.Sleep(60 * time.Second)
}
