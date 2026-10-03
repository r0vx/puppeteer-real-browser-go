package browser

import (
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
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

// TestWebRTCDoesNotBypassProxy 配置代理后 WebRTC 不能绕过代理走 UDP（STUN 会暴露真实公网 IP）：
// 一个 ICE 候选都不应出现；不配代理时有本机 host 候选，作为对照证明本测试能看出差别
func TestWebRTCDoesNotBypassProxy(t *testing.T) {
	page := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `<title>webrtc</title><script>
window.__c = []; window.__done = false;
const pc = new RTCPeerConnection();
pc.createDataChannel('x');
pc.onicecandidate = e => { if (e.candidate) window.__c.push(e.candidate.candidate); else window.__done = true; };
pc.createOffer().then(o => pc.setLocalDescription(o));
setTimeout(() => window.__done = true, 5000);
</script>`)
	}))
	defer page.Close()
	// 代理地址只需要是个能连上的端口：测试页在本机回环地址上，Chrome 访问它本来就不走代理
	proxy := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusBadGateway) }))
	defer proxy.Close()
	_, proxyPort, _ := net.SplitHostPort(strings.TrimPrefix(proxy.URL, "http://"))

	for _, withProxy := range []bool{false, true} {
		opts := &ConnectOptions{Headless: true, UseCustomCDP: true}
		if withProxy {
			opts.Proxy = &ProxyConfig{Host: "127.0.0.1", Port: proxyPort}
		}
		inst, err := Connect(t.Context(), opts)
		if err != nil {
			t.Fatal(err)
		}
		if err := inst.Page().Navigate(page.URL + "/"); err != nil {
			inst.Close()
			t.Fatal(err)
		}
		for deadline := time.Now().Add(8 * time.Second); time.Now().Before(deadline); time.Sleep(200 * time.Millisecond) {
			if done, _ := inst.Page().Evaluate(`window.__done`); done == true {
				break
			}
		}
		got, _ := inst.Page().Evaluate(`window.__c.join('\n')`)
		inst.Close()
		cands, _ := got.(string)
		if withProxy && cands != "" {
			t.Errorf("with proxy WebRTC still gathered candidates (bypasses the proxy):\n%s", cands)
		}
		if !withProxy && !strings.Contains(cands, "typ host") {
			t.Errorf("without proxy no host candidate, the check cannot tell the difference: %q", cands)
		}
	}
}

// TestSetWebRTCPolicy 写入 WebRTC 策略：新 profile 直接创建 Preferences；已有 profile 只改 webrtc 段，
// 其余设置（含大整数）原样保留
func TestSetWebRTCPolicy(t *testing.T) {
	fresh := t.TempDir()
	if err := setWebRTCPolicy(fresh); err != nil {
		t.Fatalf("fresh profile: %v", err)
	}

	existing := t.TempDir()
	os.MkdirAll(filepath.Join(existing, "Default"), 0o755)
	const old = `{"profile":{"name":"acct","exit_type":"Normal"},"webrtc":{"multiple_routes_enabled":true},"counter":13385920394810294810}`
	if err := os.WriteFile(filepath.Join(existing, "Default", "Preferences"), []byte(old), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := setWebRTCPolicy(existing); err != nil {
		t.Fatalf("existing profile: %v", err)
	}

	for _, dir := range []string{fresh, existing} {
		data, err := os.ReadFile(filepath.Join(dir, "Default", "Preferences"))
		if err != nil {
			t.Fatal(err)
		}
		var prefs struct {
			WebRTC struct {
				Policy         string `json:"ip_handling_policy"`
				MultipleRoutes *bool  `json:"multiple_routes_enabled"`
				NonProxiedUDP  *bool  `json:"nonproxied_udp_enabled"`
			} `json:"webrtc"`
		}
		if err := json.Unmarshal(data, &prefs); err != nil {
			t.Fatalf("parse %s: %v", dir, err)
		}
		if prefs.WebRTC.Policy != "disable_non_proxied_udp" || prefs.WebRTC.MultipleRoutes == nil || *prefs.WebRTC.MultipleRoutes ||
			prefs.WebRTC.NonProxiedUDP == nil || *prefs.WebRTC.NonProxiedUDP {
			t.Errorf("%s: webrtc prefs = %s", dir, data)
		}
		if dir == existing && (!strings.Contains(string(data), `"name":"acct"`) || !strings.Contains(string(data), "13385920394810294810")) {
			t.Errorf("existing settings lost: %s", data)
		}
	}
}
