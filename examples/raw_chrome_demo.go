package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"
)

func main() {
	fmt.Println("🚀 原始Chrome测试")
	fmt.Println("================")

	// 获取扩展的绝对路径
	ext1, _ := filepath.Abs("examples/path/Extensions/kfjglmgfjedhhcddpfgfogkahmenikan/1.0_0")
	ext2, _ := filepath.Abs("examples/path/Extensions/mcohilncbfahbmgdjkbpemcciiolgcge/3.66.10_0")

	// 创建临时用户数据目录
	userDataDir := "/tmp/chrome-extension-test"
	os.RemoveAll(userDataDir) // 清理旧数据

	fmt.Printf("📂 扩展1: %s\n", ext1)
	fmt.Printf("📂 扩展2: %s\n", ext2)
	fmt.Printf("📁 用户数据目录: %s\n", userDataDir)

	// 检查扩展目录是否存在
	if _, err := os.Stat(ext1); err != nil {
		fmt.Printf("❌ 扩展1不存在: %v\n", err)
		return
	}
	if _, err := os.Stat(ext2); err != nil {
		fmt.Printf("❌ 扩展2不存在: %v\n", err)
		return
	}

	fmt.Println("✅ 扩展目录存在")

	// 原始Chrome命令
	chromeArgs := []string{
		"--user-data-dir=" + userDataDir,
		"--load-extension=" + ext1 + "," + ext2,
		"--enable-extensions",
		"--no-first-run",
		"--start-maximized",
		"--exclude-switches=enable-automation",
		"chrome://extensions/",
	}

	fmt.Println("🔧 Chrome启动参数:")
	for i, arg := range chromeArgs {
		fmt.Printf("  [%d] %s\n", i, arg)
	}

	// 尝试多个可能的Chrome路径
	chromePaths := []string{
		"/Applications/Google Chrome.app/Contents/MacOS/Google Chrome",
		"/usr/bin/google-chrome",
		"/usr/bin/google-chrome-stable",
		"/opt/google/chrome/chrome",
	}

	var chromePath string
	for _, path := range chromePaths {
		if _, err := os.Stat(path); err == nil {
			chromePath = path
			break
		}
	}

	if chromePath == "" {
		fmt.Println("❌ 未找到Chrome可执行文件")
		return
	}

	fmt.Printf("✅ 使用Chrome路径: %s\n", chromePath)

	// 启动Chrome
	cmd := exec.Command(chromePath, chromeArgs...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	fmt.Println("\n🚀 启动Chrome...")
	if err := cmd.Start(); err != nil {
		fmt.Printf("❌ Chrome启动失败: %v\n", err)
		return
	}

	fmt.Printf("✅ Chrome已启动 (PID: %d)\n", cmd.Process.Pid)
	fmt.Println("\n💡 手动检查:")
	fmt.Println("  1. Chrome应该自动打开chrome://extensions/页面")
	fmt.Println("  2. 检查是否显示Discord Token Login和OKX Wallet扩展")
	fmt.Println("  3. 如果没有显示，启用开发者模式查看")
	fmt.Println("\n⏳ 等待60秒供检查...")

	// 等待一段时间
	time.Sleep(60 * time.Second)

	fmt.Println("\n🛑 停止Chrome进程...")
	if err := cmd.Process.Kill(); err != nil {
		fmt.Printf("❌ 无法停止Chrome: %v\n", err)
	} else {
		fmt.Println("✅ Chrome已停止")
	}

	// 清理临时目录
	os.RemoveAll(userDataDir)
	fmt.Println("🧹 已清理临时数据")
}
