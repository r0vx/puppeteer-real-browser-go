package browser

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

// androidFingerprintDir 写一份指定机型的安卓指纹，返回指纹目录
func androidFingerprintDir(t *testing.T, userID, model string) string {
	t.Helper()
	dir := t.TempDir()
	cfg := generateFingerprintFor(userID, OSAndroid)
	cfg.Device.Model = model
	cfg.Normalize()
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, userID+".json"), data, 0o644); err != nil {
		t.Fatal(err)
	}
	return dir
}

// connectAndroid 用指定机型的安卓身份启动（无界面，默认通道）
func connectAndroid(t *testing.T, userID, model string) *BrowserInstance {
	t.Helper()
	inst, err := Connect(t.Context(), &ConnectOptions{Headless: true, Mobile: true, FingerprintUserID: userID, FingerprintDir: androidFingerprintDir(t, userID, model)})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { inst.Close() })
	return inst
}

// awaitJS 执行返回 Promise<string> 的表达式并等待结果（CustomCDP 的 Evaluate 不等 Promise）
func awaitJS(t *testing.T, p Page, expr string) string {
	t.Helper()
	if _, err := p.Evaluate(`window.__a = undefined; Promise.resolve(` + expr + `).then(v => window.__a = String(v), e => window.__a = 'ERR ' + e); true`); err != nil {
		t.Fatal(err)
	}
	for deadline := time.Now().Add(10 * time.Second); time.Now().Before(deadline); time.Sleep(50 * time.Millisecond) {
		if v, _ := p.Evaluate(`window.__a`); v != nil {
			return fmt.Sprint(v)
		}
	}
	t.Fatalf("%s never settled", expr)
	return ""
}

// androidSurfaceJS 各表面（页面、iframe、Worker、弹窗）上报的身份信号
const androidSurfaceJS = `async () => {
  const h = await navigator.userAgentData.getHighEntropyValues(['model', 'platformVersion', 'formFactors']);
  return {ua: navigator.userAgent, platform: navigator.platform, deviceMemory: navigator.deviceMemory, cores: navigator.hardwareConcurrency,
    mobile: navigator.userAgentData.mobile, uaPlatform: navigator.userAgentData.platform, model: h.model, platformVersion: h.platformVersion,
    formFactors: h.formFactors.join(),
    touch: self.document ? navigator.maxTouchPoints : null, coarse: self.document ? matchMedia('(pointer: coarse)').matches : null,
    screen: self.document ? [screen.width, screen.height, devicePixelRatio].join() : null};
}`

// newAndroidSurfaceServers 主站（主页面 + 专用 Worker + 弹窗）与跨站 iframe 站点，记录请求头
func newAndroidSurfaceServers(t *testing.T) (main *httptest.Server, headers func(path string) http.Header) {
	t.Helper()
	var mu sync.Mutex
	seen := map[string]http.Header{}
	rec := func(r *http.Request) { mu.Lock(); seen[r.URL.Path] = r.Header.Clone(); mu.Unlock() }
	frame := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rec(r)
		fmt.Fprintf(w, `<script>(%s)().then(i => parent.postMessage({frame: 1, info: i}, '*'))</script>`, androidSurfaceJS)
	}))
	frameURL := strings.Replace(frame.URL, "127.0.0.1", "localhost", 1)
	main = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rec(r)
		switch r.URL.Path {
		case "/":
			fmt.Fprintf(w, `<script>
const info = %s;
window.__r = {};
info().then(i => window.__r.main = i);
new Worker('/worker.js').onmessage = e => window.__r.worker = e.data;
addEventListener('message', e => { if (e.data && e.data.frame) window.__r.iframe = e.data.info; });
</script><iframe src="%s/f"></iframe>`, androidSurfaceJS, frameURL)
		case "/worker.js":
			w.Header().Set("Content-Type", "text/javascript")
			fmt.Fprintf(w, `(%s)().then(i => postMessage(i))`, androidSurfaceJS)
		case "/popup":
			fmt.Fprintf(w, `<script>(%s)().then(i => window.__info = i)</script>`, androidSurfaceJS)
		default:
			fmt.Fprint(w, `<title>plain</title>`)
		}
	}))
	t.Cleanup(main.Close)
	t.Cleanup(frame.Close)
	return main, func(path string) http.Header { mu.Lock(); defer mu.Unlock(); return seen[path] }
}

// TestAndroidIdentityAcrossSurfaces 主页面、跨站 iframe、专用 Worker、弹窗与请求头都是同一台手机（Review Focus 3）
func TestAndroidIdentityAcrossSurfaces(t *testing.T) {
	srv, headers := newAndroidSurfaceServers(t)
	p := connectAndroid(t, "android-surfaces", "PGKM10").Page()
	if err := p.Navigate(srv.URL + "/"); err != nil {
		t.Fatal(err)
	}
	got := map[string]map[string]any{}
	for deadline := time.Now().Add(10 * time.Second); time.Now().Before(deadline); time.Sleep(100 * time.Millisecond) {
		v, _ := p.Evaluate(`JSON.stringify(window.__r)`)
		if json.Unmarshal([]byte(fmt.Sprint(v)), &got) == nil && got["main"] != nil && got["worker"] != nil && got["iframe"] != nil {
			break
		}
	}
	if _, err := p.Evaluate(`window.__p = window.open('/popup'), true`); err != nil {
		t.Fatal(err)
	}
	for deadline := time.Now().Add(10 * time.Second); time.Now().Before(deadline) && got["popup"] == nil; time.Sleep(100 * time.Millisecond) {
		if v, _ := p.Evaluate(`window.__p && window.__p.__info ? JSON.stringify(window.__p.__info) : ''`); v != "" {
			popup := map[string]any{}
			json.Unmarshal([]byte(fmt.Sprint(v)), &popup)
			got["popup"] = popup
		}
	}
	for _, surface := range []string{"main", "iframe", "worker", "popup"} {
		info := got[surface]
		if info == nil {
			t.Errorf("%s never reported", surface)
			continue
		}
		ua := fmt.Sprint(info["ua"])
		if !strings.HasPrefix(ua, "Mozilla/5.0 (Linux; Android 10; K) ") || !strings.HasSuffix(ua, " Mobile Safari/537.36") || ua != fmt.Sprint(got["main"]["ua"]) {
			t.Errorf("%s ua %q", surface, ua)
		}
		want := map[string]any{"platform": androidPlatform, "deviceMemory": 8.0, "cores": 8.0, "mobile": true, "uaPlatform": "Android",
			"model": "PGKM10", "platformVersion": "15.0.0", "formFactors": "Mobile"}
		if surface != "worker" {
			want["touch"], want["coarse"], want["screen"] = 5.0, true, "360,804,3"
		}
		for k, v := range want {
			if info[k] != v {
				t.Errorf("%s %s = %v, want %v", surface, k, info[k], v)
			}
		}
	}
	for _, path := range []string{"/", "/f"} {
		h := headers(path)
		if h.Get("Sec-Ch-Ua-Mobile") != "?1" || h.Get("Sec-Ch-Ua-Platform") != `"Android"` || !strings.Contains(h.Get("User-Agent"), "Android 10; K") {
			t.Errorf("%s request headers %v", path, h)
		}
	}
}

// mobilePage 带 viewport meta 的测试页：没有它时手机浏览器按 980 像素宽的布局视口排版（真机同样如此）
const mobilePage = `<!doctype html><meta name="viewport" content="width=device-width, initial-scale=1"><title>plain</title>`

// TestAndroidLaunchAndViewport 插件原生为空、PDF 查看器关闭；视口与窗口外框一致；SetViewport 后仍是手机
func TestAndroidLaunchAndViewport(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { fmt.Fprint(w, mobilePage) }))
	defer srv.Close()
	p := connectAndroid(t, "android-launch", "22041211AC").Page()
	const js = `[navigator.plugins.length, navigator.mimeTypes.length, navigator.pdfViewerEnabled, innerWidth, innerHeight, outerWidth, outerHeight,
	  screen.width, screen.height, devicePixelRatio, screen.orientation.type, matchMedia('(pointer: coarse)').matches].join()`
	if err := p.Navigate(srv.URL); err != nil {
		t.Fatal(err)
	}
	if got, _ := p.Evaluate(js); got != "0,0,false,411,810,411,810,411,914,3.5,portrait-primary,true" {
		t.Errorf("launch state %v", got)
	}
	if err := p.SetViewport(400, 780); err != nil {
		t.Fatal(err)
	}
	if err := p.Navigate(srv.URL + "/?after"); err != nil {
		t.Fatal(err)
	}
	if got, _ := p.Evaluate(js); got != "0,0,false,400,780,400,780,411,914,3.5,portrait-primary,true" {
		t.Errorf("after SetViewport %v", got)
	}
}
