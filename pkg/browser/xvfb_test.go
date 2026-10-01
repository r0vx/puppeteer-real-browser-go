//go:build linux

package browser

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"
)

// processState 返回 /proc/<pid>/stat 中的进程状态字符（R/S/Z…），进程不存在返回 0
func processState(pid int) byte {
	data, err := os.ReadFile(fmt.Sprintf("/proc/%d/stat", pid))
	if err != nil {
		return 0
	}
	i := bytes.LastIndexByte(data, ')')
	if i < 0 || i+2 >= len(data) {
		return 0
	}
	return data[i+2]
}

// TestXvfbManagerStartStop Start 拉起 Xvfb 并设置 DISPLAY；Stop 结束并回收进程
func TestXvfbManagerStartStop(t *testing.T) {
	if !IsXvfbInstalled() {
		t.Skip("Xvfb 未安装")
	}
	t.Setenv("DISPLAY", "")

	xm := NewXvfbManager()
	if err := xm.Start(); err != nil {
		t.Fatalf("Start: %v", err)
	}
	pid := xm.cmd.Process.Pid
	if got := os.Getenv("DISPLAY"); got == "" || got != xm.GetDisplay() {
		t.Errorf("DISPLAY = %q, want %q", got, xm.GetDisplay())
	}
	if !xm.IsRunning() {
		t.Error("IsRunning() = false after Start")
	}
	if err := xm.Stop(); err != nil {
		t.Fatalf("Stop: %v", err)
	}
	if st := processState(pid); st != 0 {
		t.Errorf("Xvfb pid %d still present (state %q) after Stop", pid, st)
	}
}

// TestXvfbStartFailureReapsProcess Xvfb 启动即退出时 Start 应返回错误、不设置 DISPLAY、不留僵尸进程
func TestXvfbStartFailureReapsProcess(t *testing.T) {
	bin := t.TempDir()
	if err := os.WriteFile(filepath.Join(bin, "Xvfb"), []byte("#!/bin/sh\nexit 1\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))
	t.Setenv("DISPLAY", "")

	xm := NewXvfbManager()
	if err := xm.Start(); err == nil {
		xm.Stop()
		t.Fatal("Start succeeded with an Xvfb that exits immediately")
	}
	if got := os.Getenv("DISPLAY"); got != "" {
		t.Errorf("DISPLAY = %q after failed Start, want empty", got)
	}
	pid := xm.cmd.Process.Pid
	if st := processState(pid); st != 0 {
		t.Errorf("Xvfb pid %d left in state %q after failed Start", pid, st)
	}
}

// TestXvfbExitsWithHost 宿主进程直接退出（未调用 Stop）后 Xvfb 不应残留
func TestXvfbExitsWithHost(t *testing.T) {
	if os.Getenv("PRBG_XVFB_HOST") == "1" {
		// 子进程：拉起 Xvfb 后打印 pid 并直接退出
		xm := NewXvfbManager()
		if err := xm.Start(); err != nil {
			fmt.Println("start error:", err)
			os.Exit(1)
		}
		fmt.Println(xm.cmd.Process.Pid)
		os.Exit(0)
	}
	if !IsXvfbInstalled() {
		t.Skip("Xvfb 未安装")
	}

	cmd := exec.Command(os.Args[0], "-test.run=^TestXvfbExitsWithHost$")
	cmd.Env = append(os.Environ(), "PRBG_XVFB_HOST=1", "DISPLAY=")
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("host process: %v\n%s", err, out)
	}
	pid, err := strconv.Atoi(strings.TrimSpace(string(out)))
	if err != nil {
		t.Fatalf("parse pid from %q: %v", out, err)
	}

	// 僵尸进程视为已退出（孤儿由 init 回收）
	alive := func() bool { st := processState(pid); return st != 0 && st != 'Z' }
	for deadline := time.Now().Add(3 * time.Second); alive() && time.Now().Before(deadline); {
		time.Sleep(50 * time.Millisecond)
	}
	if alive() {
		syscall.Kill(pid, syscall.SIGKILL)
		t.Errorf("Xvfb pid %d still running after host exited", pid)
	}
}

// TestConnectHeadfulStartsXvfb Linux 无 DISPLAY 时有界面模式应自动拉起 Xvfb，页面正常渲染；
// Close 后临时 profile 不能被晚退出的辅助进程（network service 落盘）重新写回
func TestConnectHeadfulStartsXvfb(t *testing.T) {
	if !IsXvfbInstalled() {
		t.Skip("Xvfb 未安装")
	}
	t.Setenv("DISPLAY", "")

	inst, err := Connect(t.Context(), &ConnectOptions{Headless: false, UseCustomCDP: true})
	if err != nil {
		t.Fatalf("Connect: %v", err)
	}
	dir := flagValue(inst.Chrome().Flags, "--user-data-dir=")

	got, err := inst.Page().Evaluate(`innerWidth > 100 && innerHeight > 100`)
	if err != nil {
		inst.Close()
		t.Fatalf("Evaluate: %v", err)
	}
	if got != true {
		t.Errorf("viewport usable = %v, want true", got)
	}

	if err := inst.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
	// 给可能残留的辅助进程留出落盘时间，再检查目录
	time.Sleep(time.Second)
	if _, err := os.Stat(dir); !os.IsNotExist(err) {
		os.RemoveAll(dir)
		t.Errorf("temp user-data-dir %s exists after Close (stat err=%v)", dir, err)
	}
}
