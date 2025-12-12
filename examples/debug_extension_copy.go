package main

import (
	"fmt"
	"os"

	"github.com/HNRow/puppeteer-real-browser-go/pkg/browser"
)

func main() {
	fmt.Println("🔍 调试扩展复制功能")
	fmt.Println("===================")

	// 创建测试目录
	userDataDir := "/tmp/debug-extension-test"
	if err := os.MkdirAll(userDataDir, 0755); err != nil {
		fmt.Printf("创建测试目录失败: %v\n", err)
		return
	}
	defer os.RemoveAll(userDataDir)

	// 测试扩展安装器
	installer := browser.NewExtensionInstaller(userDataDir)
	
	extensionPaths := []string{
		"examples/path/Extensions/kfjglmgfjedhhcddpfgfogkahmenikan/1.0_0",
		"examples/path/Extensions/mcohilncbfahbmgdjkbpemcciiolgcge/3.66.10_0",
	}

	fmt.Println("📂 测试扩展路径:")
	for i, path := range extensionPaths {
		fmt.Printf("  %d. %s\n", i+1, path)
		if _, err := os.Stat(path); err != nil {
			fmt.Printf("     ❌ 路径不存在: %v\n", err)
		} else {
			fmt.Printf("     ✅ 路径存在\n")
		}
	}

	fmt.Println("\n🔧 开始预安装扩展...")
	if err := installer.PreInstallExtensions(extensionPaths); err != nil {
		fmt.Printf("❌ 预安装失败: %v\n", err)
		return
	}

	fmt.Println("\n📋 检查安装结果...")
	extensionsDir := userDataDir + "/Default/Extensions"
	if _, err := os.Stat(extensionsDir); err != nil {
		fmt.Printf("❌ Extensions目录不存在: %v\n", err)
		return
	}

	entries, err := os.ReadDir(extensionsDir)
	if err != nil {
		fmt.Printf("❌ 读取Extensions目录失败: %v\n", err)
		return
	}

	fmt.Printf("📁 Extensions目录内容 (%d 项):\n", len(entries))
	for _, entry := range entries {
		fmt.Printf("  - %s (目录: %v)\n", entry.Name(), entry.IsDir())
		if entry.IsDir() {
			subDir := extensionsDir + "/" + entry.Name()
			subEntries, err := os.ReadDir(subDir)
			if err == nil {
				for _, subEntry := range subEntries {
					fmt.Printf("    └── %s\n", subEntry.Name())
				}
			}
		}
	}

	fmt.Println("✅ 扩展复制调试完成")
}