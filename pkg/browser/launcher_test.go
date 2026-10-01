package browser

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"testing"
	"time"
)

// flagValue 返回 flags 中以 prefix 开头的参数值（同名参数以最后一个为准，与 Chrome 一致）
func flagValue(flags []string, prefix string) string {
	v := ""
	for _, f := range flags {
		if s, ok := strings.CutPrefix(f, prefix); ok {
			v = s
		}
	}
	return v
}

// TestCloseUserDataDirCleanup 本库创建的临时 profile 在 Close 后删除；调用方指定的目录必须保留
func TestCloseUserDataDirCleanup(t *testing.T) {
	custom, err := os.MkdirTemp("", "prbg-custom-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(custom)

	cases := []struct {
		name        string
		opts        *ConnectOptions
		wantRemoved bool
	}{
		{"temp dir", &ConnectOptions{Headless: true}, true},
		{"temp dir with IgnoreAllFlags", &ConnectOptions{Headless: true, IgnoreAllFlags: true, Args: []string{"--no-sandbox"}}, true},
		{"custom dir kept", &ConnectOptions{Headless: true, CustomConfig: map[string]any{"userDataDir": custom}}, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			inst, err := Connect(t.Context(), tc.opts)
			if err != nil {
				t.Fatalf("Connect: %v", err)
			}
			dir := flagValue(inst.Chrome().Flags, "--user-data-dir=")
			if dir == "" {
				inst.Close()
				t.Fatalf("no --user-data-dir in flags: %v", inst.Chrome().Flags)
			}
			if err := inst.Close(); err != nil {
				t.Fatalf("Close: %v", err)
			}
			_, statErr := os.Stat(dir)
			if removed := os.IsNotExist(statErr); removed != tc.wantRemoved {
				t.Errorf("user-data-dir %s removed=%v, want %v (stat err=%v)", dir, removed, tc.wantRemoved, statErr)
			}
		})
	}
}

// TestIsRunningAfterCrash Chrome 主进程被外部杀掉（模拟崩溃）后 IsRunning 应返回 false；
// 连接池健康检查依赖它，未回收的僵尸进程不能算活着
func TestIsRunningAfterCrash(t *testing.T) {
	inst, err := Connect(t.Context(), &ConnectOptions{Headless: true})
	if err != nil {
		t.Fatalf("Connect: %v", err)
	}
	defer inst.Close()

	chrome := inst.Chrome()
	if !chrome.IsRunning() {
		t.Fatal("IsRunning() = false right after Connect")
	}
	if err := chrome.Cmd.Process.Kill(); err != nil {
		t.Fatalf("kill: %v", err)
	}
	for deadline := time.Now().Add(3 * time.Second); chrome.IsRunning() && time.Now().Before(deadline); {
		time.Sleep(50 * time.Millisecond)
	}
	if chrome.IsRunning() {
		t.Error("IsRunning() = true after the Chrome process was killed")
	}
}

// TestConnectFailsFastWhenChromeExits Chrome 启动即退出时 Connect 应立即报错，而不是等满 30s 调试端口超时
func TestConnectFailsFastWhenChromeExits(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("用 shell 脚本模拟 Chrome")
	}
	fake := filepath.Join(t.TempDir(), "chrome")
	if err := os.WriteFile(fake, []byte("#!/bin/sh\nexit 1\n"), 0o755); err != nil {
		t.Fatal(err)
	}

	start := time.Now()
	inst, err := Connect(t.Context(), &ConnectOptions{Headless: true, CustomConfig: map[string]any{"chromePath": fake}})
	if err == nil {
		inst.Close()
		t.Fatal("Connect succeeded with a Chrome that exits immediately")
	}
	if elapsed := time.Since(start); elapsed > 5*time.Second {
		t.Errorf("Connect took %v to fail, want < 5s (err: %v)", elapsed, err)
	}
}

// TestCloseShutsDownGracefully Close 应让 Chrome 走正常退出流程：profile 落盘且未标记为崩溃。
// SIGTERM 退出记为 "SessionEnded"，CDP Browser.close 记为 "Normal"；
// 被 SIGKILL 时 Preferences 来不及写，或停留在启动时写入的 "Crashed"。
func TestCloseShutsDownGracefully(t *testing.T) {
	dir, err := os.MkdirTemp("", "prbg-graceful-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)

	inst, err := Connect(t.Context(), &ConnectOptions{Headless: true, CustomConfig: map[string]any{"userDataDir": dir}})
	if err != nil {
		t.Fatalf("Connect: %v", err)
	}
	if err := inst.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(dir, "Default", "Preferences"))
	if err != nil {
		t.Fatalf("read Preferences: %v", err)
	}
	var prefs struct {
		Profile struct {
			ExitType string `json:"exit_type"`
		} `json:"profile"`
	}
	if err := json.Unmarshal(data, &prefs); err != nil {
		t.Fatalf("parse Preferences: %v", err)
	}
	if got := prefs.Profile.ExitType; got != "SessionEnded" && got != "Normal" {
		t.Errorf("profile.exit_type = %q, want SessionEnded or Normal", got)
	}
}

// TestWebRTCPolicyFlag 配置代理时启动参数禁止 WebRTC 绕过代理；未配置代理时不加（不影响直连用户的 WebRTC）
func TestWebRTCPolicyFlag(t *testing.T) {
	const flag = "--force-webrtc-ip-handling-policy=disable_non_proxied_udp"
	proxy := &ProxyConfig{Host: "127.0.0.1", Port: "8080"}
	cases := []struct {
		name string
		opts *ConnectOptions
		want bool
	}{
		{"proxy", &ConnectOptions{Proxy: proxy}, true},
		{"proxy with IgnoreAllFlags", &ConnectOptions{IgnoreAllFlags: true, Proxy: proxy}, true},
		{"no proxy", &ConnectOptions{}, false},
	}
	for _, tc := range cases {
		flags := NewChromeLauncher().buildChromeFlags(tc.opts, 9222, t.TempDir())
		if got := slices.Contains(flags, flag); got != tc.want {
			t.Errorf("%s: has %s = %v, want %v", tc.name, flag, got, tc.want)
		}
	}
}
