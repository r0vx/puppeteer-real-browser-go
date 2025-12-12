package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"time"
)

func main() {
	fmt.Println("🧪 超级极简扩展测试")
	fmt.Println("==================")

	// 获取简单测试扩展的绝对路径
	simpleExt, _ := filepath.Abs("examples/simple_test_extension")
	fmt.Printf("📂 测试扩展: %s\n", simpleExt)

	// 检查扩展目录
	if _, err := os.Stat(simpleExt); err != nil {
		fmt.Printf("❌ 扩展目录不存在: %v\n", err)
		return
	}

	// 创建临时用户数据目录
	userDataDir := "/tmp/ultra-minimal-test"
	os.RemoveAll(userDataDir)

	// 最极简的Chrome参数 - 只保留绝对必要的
	chromeArgs := []string{
		"--user-data-dir=" + userDataDir,
		"--load-extension=" + simpleExt,
		"--enable-extensions",
		"--no-first-run",
		"--start-maximized",
		// 完全去掉 --disable-web-security, --disable-features 等限制性标志
		"chrome://extensions/",
	}

	fmt.Println("🔧 超极简Chrome参数:")
	for i, arg := range chromeArgs {
		fmt.Printf("  [%d] %s\n", i, arg)
	}

	// Chrome路径
	chromePath := "/Applications/Google Chrome.app/Contents/MacOS/Google Chrome"
	if _, err := os.Stat(chromePath); err != nil {
		fmt.Println("❌ 未找到Chrome")
		return
	}

	fmt.Println("🚀 启动超极简Chrome...")
	cmd := exec.Command(chromePath, chromeArgs...)
	
	// 启动Chrome
	if err := cmd.Start(); err != nil {
		fmt.Printf("❌ Chrome启动失败: %v\n", err)
		return
	}

	fmt.Printf("✅ Chrome已启动 (PID: %d)\n", cmd.Process.Pid)
	fmt.Println("\n💡 此测试使用最极简配置:")
	fmt.Println("  • 移除了所有--disable-*限制性标志")
	fmt.Println("  • 移除了--disable-web-security")
	fmt.Println("  • 移除了复杂的--disable-features")
	fmt.Println("  • 只保留扩展加载必需的标志")
	
	fmt.Println("\n🔍 手动检查:")
	fmt.Println("  1. Chrome应该打开chrome://extensions/")
	fmt.Println("  2. 查看是否显示'Simple Test Extension'")
	fmt.Println("  3. 如果显示，说明问题在于过多的限制性标志")

	fmt.Println("\n⏳ 等待60秒供检查...")
	time.Sleep(60 * time.Second)

	fmt.Println("\n🛑 停止Chrome...")
	cmd.Process.Kill()
	
	// 清理
	os.RemoveAll(userDataDir)
	fmt.Println("✅ 测试完成")
}