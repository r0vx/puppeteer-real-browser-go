package main

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"syscall"
	"testing"
	"time"
)

const (
	androidChromeUA = "Mozilla/5.0 (Linux; Android 10; K) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/141.0.0.0 Mobile Safari/537.36"
	iphoneSafariUA  = "Mozilla/5.0 (iPhone; CPU iPhone OS 18_6 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/18.6 Mobile/15E148 Safari/604.1"
	miuiBrowserUA   = "Mozilla/5.0 (Linux; U; Android 14; zh-cn; 23127PN0CC Build/UKQ1.230804.001) AppleWebKit/537.36 (KHTML, like Gecko) Version/4.0 Chrome/122.0.6261.119 Mobile Safari/537.36 XiaoMi/MiuiBrowser/18.6.70515"
	wechatAndroidUA = "Mozilla/5.0 (Linux; Android 14; 23127PN0CC Build/UKQ1.230804.001; wv) AppleWebKit/537.36 (KHTML, like Gecko) Version/4.0 Chrome/130.0.6723.103 Mobile Safari/537.36 XWEB/1300333 MMWEBSDK/20241103 MicroMessenger/8.0.55.2780(0x28003737) WeChat/arm64 Weixin NetType/WIFI Language/zh_CN ABI/arm64"
	douyinAndroidUA = "Mozilla/5.0 (Linux; Android 14; 23127PN0CC Build/UKQ1.230804.001; wv) AppleWebKit/537.36 (KHTML, like Gecko) Version/4.0 Chrome/126.0.6478.122 Mobile Safari/537.36 aweme_330100 JsSdk/1.0 NetType/WIFI Channel/xiaomi_1128_64 app_version/33.1.0 ByteLocale/zh-CN Region/CN AppSkin/white AppTheme/light BytedanceWebview/d8a21c6"
	chromeIOSUA     = "Mozilla/5.0 (iPhone; CPU iPhone OS 18_6 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) CriOS/141.0.7390.41 Mobile/15E148 Safari/604.1"
	edgeIOSUA       = "Mozilla/5.0 (iPhone; CPU iPhone OS 18_6 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/18.0 EdgiOS/141.3537.60 Mobile/15E148 Safari/605.1.15"
	wechatIOSUA     = "Mozilla/5.0 (iPhone; CPU iPhone OS 18_6 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Mobile/15E148 MicroMessenger/8.0.61(0x18003d2e) NetType/WIFI Language/zh_CN"
)

// newTestServer 用临时采集页与临时结果目录创建服务，返回服务与结果目录
func newTestServer(t *testing.T) (*server, string) {
	t.Helper()
	dir := t.TempDir()
	page := filepath.Join(dir, "collector.html")
	if err := os.WriteFile(page, []byte("<script>window.collectBaseline = () => ({})</script>"), 0o644); err != nil {
		t.Fatal(err)
	}
	out := filepath.Join(dir, "out")
	return newServer(page, out), out
}

// postResult 向 /result 提交请求体
func postResult(h http.Handler, body string) *httptest.ResponseRecorder {
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/result", strings.NewReader(body)))
	return rec
}

// sampleResult 构造一份最小的结果；model 与 brands 都为空时 userAgentData 为 null（Safari 的情况）
func sampleResult(ua, model string, brands ...string) string {
	uad := "null"
	if model != "" || len(brands) > 0 {
		list := make([]map[string]string, 0, len(brands))
		for _, b := range brands {
			list = append(list, map[string]string{"brand": b, "version": "141"})
		}
		data, _ := json.Marshal(map[string]any{"model": model, "brands": list})
		uad = string(data)
	}
	return fmt.Sprintf(`{"schema": 1, "auto": {"identity": {"userAgent": %q, "userAgentData": %s}}, "interaction": {"events": []}}`, ua, uad)
}

// savedFile 解出应答中的文件名
func savedFile(t *testing.T, rec *httptest.ResponseRecorder) string {
	t.Helper()
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d: %s", rec.Code, rec.Body)
	}
	var resp struct {
		File string `json:"file"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	return resp.File
}

// TestSaveResultNames 文件名由 UA、Client Hints 品牌与机型推导：只有正版安卓 Chrome 与 iOS Safari 记为 chrome / safari，
// 厂商浏览器、App 内嵌页、iOS 上的第三方浏览器（包括 Chrome）都记为 other（Review Focus 4）
func TestSaveResultNames(t *testing.T) {
	chrome := []string{"Google Chrome", "Chromium", "Not=A?Brand"}
	cases := []struct {
		name, ua, model string
		brands          []string
		want            string
	}{
		{"android chrome", androidChromeUA, "23127PN0CC", chrome, "android-chrome-23127pn0cc.json"},
		{"iphone safari", iphoneSafariUA, "", nil, "ios-safari-iphone.json"},
		{"xiaomi browser", miuiBrowserUA, "", nil, "android-other-unknown.json"},
		{"model characters", androidChromeUA, "Pixel 8 Pro/中文", chrome, "android-chrome-pixel-8-pro.json"},
		{"android chrome without brands", androidChromeUA, "23127PN0CC", nil, "android-other-23127pn0cc.json"},
		{"wechat webview", wechatAndroidUA, "23127PN0CC", []string{"Android WebView", "Chromium"}, "android-other-23127pn0cc.json"},
		{"douyin webview", douyinAndroidUA, "23127PN0CC", []string{"Android WebView", "Chromium"}, "android-other-23127pn0cc.json"},
		{"webview claiming chrome", wechatAndroidUA, "23127PN0CC", chrome, "android-other-23127pn0cc.json"},
		{"chrome on ios", chromeIOSUA, "", nil, "ios-other-iphone.json"},
		{"edge on ios", edgeIOSUA, "", nil, "ios-other-iphone.json"},
		{"wechat on ios", wechatIOSUA, "", nil, "ios-other-iphone.json"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			s, out := newTestServer(t)
			body := sampleResult(tc.ua, tc.model, tc.brands...)
			if got := savedFile(t, postResult(s.routes(), body)); got != tc.want {
				t.Errorf("file = %q, want %q", got, tc.want)
			}
			saved, err := os.ReadFile(filepath.Join(out, tc.want))
			if err != nil {
				t.Fatal(err)
			}
			// 保存的是格式化后的同一份 JSON
			var got, want any
			if err := json.Unmarshal(saved, &got); err != nil {
				t.Fatal(err)
			}
			json.Unmarshal([]byte(body), &want)
			if !reflect.DeepEqual(got, want) {
				t.Errorf("saved content differs from the posted result")
			}
			if !bytes.Contains(saved, []byte("\n  \"auto\"")) {
				t.Errorf("saved JSON is not indented:\n%s", saved)
			}
		})
	}
}

// TestSaveResultNoOverwrite 同一台手机采两次不覆盖，第二份加序号
func TestSaveResultNoOverwrite(t *testing.T) {
	s, out := newTestServer(t)
	body := sampleResult(androidChromeUA, "23127PN0CC", "Google Chrome")
	first := savedFile(t, postResult(s.routes(), body))
	second := savedFile(t, postResult(s.routes(), body))
	if first != "android-chrome-23127pn0cc.json" || second != "android-chrome-23127pn0cc-2.json" {
		t.Errorf("files %q, %q", first, second)
	}
	entries, _ := os.ReadDir(out)
	if len(entries) != 2 {
		t.Errorf("%d files in output, want 2", len(entries))
	}
}

// TestSaveResultRejects 过大、不是 JSON、缺少 auto 的请求被拒绝，且不写文件
func TestSaveResultRejects(t *testing.T) {
	cases := []struct {
		name string
		body string
		want int
	}{
		{"too large", `{"auto": {"pad": "` + strings.Repeat("x", 1<<20) + `"}}`, http.StatusRequestEntityTooLarge},
		{"not json", "not json", http.StatusBadRequest},
		{"no auto", `{"schema": 1}`, http.StatusBadRequest},
		{"null auto", `{"schema": 1, "auto": null}`, http.StatusBadRequest},
		// 几 KB 的深层嵌套，格式化缩进后有约 9 MB：上限要约束落盘大小，不只是请求体
		{"amplified by indent", `{"auto": {"x": ` + strings.Repeat("[", 3000) + strings.Repeat("]", 3000) + `}}`, http.StatusRequestEntityTooLarge},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			s, out := newTestServer(t)
			if rec := postResult(s.routes(), tc.body); rec.Code != tc.want {
				t.Errorf("status %d, want %d", rec.Code, tc.want)
			}
			if entries, _ := os.ReadDir(out); len(entries) != 0 {
				t.Errorf("%d files written for a rejected request", len(entries))
			}
		})
	}
}

// TestServesPage 根路径返回采集页，其他路径 404
func TestServesPage(t *testing.T) {
	s, _ := newTestServer(t)
	rec := httptest.NewRecorder()
	s.routes().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), "collectBaseline") ||
		!strings.HasPrefix(rec.Header().Get("Content-Type"), "text/html") {
		t.Errorf("GET / = %d %q %q", rec.Code, rec.Header().Get("Content-Type"), rec.Body)
	}
	rec = httptest.NewRecorder()
	s.routes().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/favicon.ico", nil))
	if rec.Code != http.StatusNotFound {
		t.Errorf("GET /favicon.ico = %d, want 404", rec.Code)
	}
}

// TestCollectScriptRefusesBusyPort 端口被别的程序占用时，启动脚本报错退出且不开隧道，免得把那个程序暴露到公网
func TestCollectScriptRefusesBusyPort(t *testing.T) {
	busy := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { fmt.Fprint(w, "another local service") }))
	defer busy.Close()
	port := busy.Listener.Addr().(*net.TCPAddr).Port

	// 假的 cloudflared：被启动就留下标记
	stubDir := t.TempDir()
	marker := filepath.Join(stubDir, "tunnel-started")
	stub := "#!/bin/sh\ntouch '" + marker + "'\nexec sleep 30\n"
	if err := os.WriteFile(filepath.Join(stubDir, "cloudflared"), []byte(stub), 0o755); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "bash", "../../scripts/collect-baseline.sh")
	cmd.Env = append(os.Environ(), "PATH="+stubDir+string(os.PathListSeparator)+os.Getenv("PATH"), fmt.Sprintf("PORT=%d", port))
	out, err := cmd.CombinedOutput()
	if err == nil {
		t.Errorf("script succeeded on a busy port:\n%s", out)
	}
	if _, statErr := os.Stat(marker); statErr == nil {
		t.Errorf("tunnel was started although port %d belongs to another service:\n%s", port, out)
	}
}

// TestCollectScriptStartsTunnel 正常路径：服务就绪后才开隧道并打印地址；中断后服务、隧道都停止
func TestCollectScriptStartsTunnel(t *testing.T) {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	port := l.Addr().(*net.TCPAddr).Port
	l.Close()

	// 假的 cloudflared：确认此时本服务已在监听，再报出一个隧道地址
	stubDir := t.TempDir()
	stub := fmt.Sprintf("#!/bin/sh\ncurl -sf http://127.0.0.1:%d/ | grep -q collectBaseline || exit 1\necho 'https://stub-tunnel.trycloudflare.com' >&2\nexec sleep 30\n", port)
	if err := os.WriteFile(filepath.Join(stubDir, "cloudflared"), []byte(stub), 0o755); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "bash", "../../scripts/collect-baseline.sh")
	cmd.Env = append(os.Environ(), "PATH="+stubDir+string(os.PathListSeparator)+os.Getenv("PATH"), fmt.Sprintf("PORT=%d", port))
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		t.Fatal(err)
	}
	cmd.Stderr = cmd.Stdout
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	found := false
	for sc := bufio.NewScanner(stdout); sc.Scan(); {
		if strings.Contains(sc.Text(), "在手机上打开： https://stub-tunnel.trycloudflare.com") {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("script never printed the tunnel URL")
	}
	cmd.Process.Signal(syscall.SIGINT)
	go func() { _, _ = io.Copy(io.Discard, stdout) }()
	cmd.Wait()
	// 服务停止后端口应已释放
	deadline := time.Now().Add(5 * time.Second)
	for {
		c, err := net.Dial("tcp", fmt.Sprintf("127.0.0.1:%d", port))
		if err != nil {
			break
		}
		c.Close()
		if time.Now().After(deadline) {
			t.Fatal("collector server still listening after the script was interrupted")
		}
		time.Sleep(100 * time.Millisecond)
	}
}
