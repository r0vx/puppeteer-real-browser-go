package main

import (
	"fmt"
	"os"
	"time"

	"github.com/HNRow/puppeteer-real-browser-go/internal/config"
	"github.com/HNRow/puppeteer-real-browser-go/pkg/browser"
)

func main() {
	fmt.Println("🕒 扩展安装时序调试")
	fmt.Println("==================")

	// 创建临时用户数据目录
	userDataDir := "/tmp/debug-timing-" + fmt.Sprintf("%d", time.Now().Unix())
	fmt.Printf("📁 用户数据目录: %s\n", userDataDir)

	// 手动安装扩展
	installer := browser.NewExtensionInstaller(userDataDir)
	extensionPaths := config.GetDefaultExtensionPaths()

	fmt.Println("\n🔧 第1步: 预安装扩展...")
	if err := installer.PreInstallExtensions(extensionPaths); err != nil {
		fmt.Printf("❌ 预安装失败: %v\n", err)
		return
	}

	fmt.Println("\n🔍 第2步: 检查安装结果...")
	extensionsDir := userDataDir + "/Default/Extensions"
	entries, err := os.ReadDir(extensionsDir)
	if err != nil {
		fmt.Printf("❌ 无法读取Extensions目录: %v\n", err)
		return
	}

	fmt.Printf("✅ 发现 %d 个扩展目录:\n", len(entries))
	for _, entry := range entries {
		fmt.Printf("  - %s\n", entry.Name())
		
		// 检查版本目录
		subDir := extensionsDir + "/" + entry.Name()
		subEntries, err := os.ReadDir(subDir)
		if err == nil {
			for _, subEntry := range subEntries {
				fmt.Printf("    └── %s\n", subEntry.Name())
				
				// 检查manifest.json
				manifestPath := subDir + "/" + subEntry.Name() + "/manifest.json"
				if _, err := os.Stat(manifestPath); err == nil {
					fmt.Printf("        ✅ manifest.json存在\n")
				}
			}
		}
	}

	fmt.Println("\n📝 第3步: 创建Preferences配置...")
	if err := installer.CreateExtensionsPreferences(extensionPaths); err != nil {
		fmt.Printf("❌ 创建配置失败: %v\n", err)
	} else {
		fmt.Println("✅ 配置创建成功")
	}

	fmt.Println("\n🔍 第4步: 再次检查扩展目录...")
	entries2, err := os.ReadDir(extensionsDir)
	if err != nil {
		fmt.Printf("❌ 无法读取Extensions目录: %v\n", err)
		return
	}

	fmt.Printf("✅ 现在有 %d 个扩展目录:\n", len(entries2))
	for _, entry := range entries2 {
		fmt.Printf("  - %s\n", entry.Name())
	}

	fmt.Println("\n⏳ 保持5秒...")
	time.Sleep(5 * time.Second)

	fmt.Println("\n🔍 第5步: 最后检查扩展目录...")
	entries3, err := os.ReadDir(extensionsDir)
	if err != nil {
		fmt.Printf("❌ 无法读取Extensions目录: %v\n", err)
		return
	}

	fmt.Printf("✅ 最终有 %d 个扩展目录:\n", len(entries3))
	for _, entry := range entries3 {
		fmt.Printf("  - %s\n", entry.Name())
	}

	// 清理
	defer os.RemoveAll(userDataDir)

	fmt.Println("\n✅ 时序调试完成")
}