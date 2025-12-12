package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/HNRow/puppeteer-real-browser-go/pkg/browser"
)

func main() {
	fmt.Println("🔍 验证Chrome实际启动参数")
	fmt.Println("========================")

	ctx := context.Background()

	opts := &browser.ConnectOptions{
		Headless:                  false,
		AutoLoadDefaultExtensions: true,
		PersistProfile:            true,
		ProfileName:               "verify_args",
	}

	// 直接启动Chrome launcher来获取参数
	launcher := browser.NewChromeLauncher()
	chrome, err := launcher.Launch(ctx, opts)
	if err != nil {
		log.Fatalf("Chrome启动失败: %v", err)
	}
	defer chrome.Kill()

	fmt.Printf("✅ Chrome PID: %d\n", chrome.PID)
	fmt.Printf("🔗 调试端口: %d\n", chrome.Port)

	fmt.Println("\n📋 完整的Chrome启动参数:")
	for i, arg := range chrome.Flags {
		if arg == "--load-extension" {
			fmt.Printf("  [%d] 🎯 %s\n", i, arg)
			if i+1 < len(chrome.Flags) {
				fmt.Printf("  [%d] 📂 %s\n", i+1, chrome.Flags[i+1])
			}
		} else if arg == "--enable-extensions" {
			fmt.Printf("  [%d] ✅ %s\n", i, arg)
		} else if arg == "--user-data-dir" {
			fmt.Printf("  [%d] 📁 %s\n", i, arg)
			if i+1 < len(chrome.Flags) {
				fmt.Printf("  [%d] 📂 %s\n", i+1, chrome.Flags[i+1])
			}
		} else {
			fmt.Printf("  [%d] %s\n", i, arg)
		}
	}

	fmt.Println("\n🔍 关键参数分析:")

	// 检查是否包含--load-extension
	hasLoadExtension := false
	loadExtensionValue := ""
	for _, arg := range chrome.Flags {
		if strings.HasPrefix(arg, "--load-extension=") {
			hasLoadExtension = true
			loadExtensionValue = strings.TrimPrefix(arg, "--load-extension=")
			break
		} else if arg == "--load-extension" {
			// 处理分离的参数格式（如果存在）
			// 这个分支保留以防万一
			hasLoadExtension = true
			break
		}
	}

	if hasLoadExtension {
		fmt.Printf("✅ 找到 --load-extension 参数\n")
		fmt.Printf("📂 扩展路径: %s\n", loadExtensionValue)

		// 分析路径
		if loadExtensionValue != "" {
			paths := strings.Split(loadExtensionValue, ",")
			fmt.Printf("📋 扩展数量: %d\n", len(paths))
			for i, path := range paths {
				fmt.Printf("  %d. %s\n", i+1, path)

				// 检查路径是否存在
				if _, err := os.Stat(path); err != nil {
					fmt.Printf("     ❌ 路径不存在或无法访问: %v\n", err)
				} else {
					fmt.Printf("     ✅ 路径存在\n")
				}
			}
		}
	} else {
		fmt.Printf("❌ 未找到 --load-extension 参数！\n")
	}

	// 检查--enable-extensions
	hasEnableExtensions := false
	for _, arg := range chrome.Flags {
		if arg == "--enable-extensions" {
			hasEnableExtensions = true
			break
		}
	}

	if hasEnableExtensions {
		fmt.Printf("✅ 找到 --enable-extensions 参数\n")
	} else {
		fmt.Printf("❌ 未找到 --enable-extensions 参数\n")
	}

	fmt.Println("\n⏳ 等待5秒...")
	time.Sleep(10 * time.Second)
}
