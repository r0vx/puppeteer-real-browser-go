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

// surfaceInfoJS 各执行环境（主页面、跨站 iframe、专用 / 共享 Worker、弹窗）读取的身份信号
const surfaceInfoJS = `() => ({ua: navigator.userAgent, appVersion: navigator.appVersion, platform: navigator.platform, langs: navigator.languages.join(),
  hc: navigator.hardwareConcurrency, tz: Intl.DateTimeFormat().resolvedOptions().timeZone, date: new Date(0).toString(),
  locale: Intl.DateTimeFormat().resolvedOptions().locale, uadPlatform: navigator.userAgentData.platform,
  brands: navigator.userAgentData.brands.map(b => b.brand).join(),
  screen: self.document ? [screen.width, screen.height, screen.availHeight, devicePixelRatio].join() : null,
  webgl: (() => { try { const c = self.document ? document.createElement('canvas') : new OffscreenCanvas(1, 1);
    const g = c.getContext('webgl'); g.getExtension('WEBGL_debug_renderer_info'); return g.getParameter(37446); } catch (e) { return 'n/a'; } })()})`

// surfaceServers 主站（主页面 + 专用 / 共享 Worker + 弹窗）与跨站 iframe 站点，记录每个路径的请求头
type surfaceServers struct {
	main, frame *httptest.Server
	mu          sync.Mutex
	headers     map[string]http.Header
}

func newSurfaceServers(t *testing.T) *surfaceServers {
	t.Helper()
	ss := &surfaceServers{headers: map[string]http.Header{}}
	rec := func(r *http.Request) {
		ss.mu.Lock()
		ss.headers[r.URL.Path] = r.Header.Clone()
		ss.mu.Unlock()
	}
	ss.frame = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rec(r)
		fmt.Fprintf(w, `<script>parent.postMessage({frame: 1, info: (%s)()}, '*')</script>`, surfaceInfoJS)
	}))
	frameURL := strings.Replace(ss.frame.URL, "127.0.0.1", "localhost", 1) // 不同 site → 跨进程 iframe
	ss.main = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rec(r)
		switch r.URL.Path {
		case "/":
			fmt.Fprintf(w, `<title>surfaces</title><script>
const info = %s;
window.__r = {main: info()};
const w = new Worker('/worker.js');
w.onmessage = e => window.__r.worker = e.data;
const sw = new SharedWorker('/shared.js');
sw.port.onmessage = e => window.__r.shared = e.data;
sw.port.start();
addEventListener('message', e => { if (e.data && e.data.frame) window.__r.iframe = e.data.info; });
</script><iframe src="%s/f"></iframe>`, surfaceInfoJS, frameURL)
		case "/worker.js":
			w.Header().Set("Content-Type", "text/javascript")
			fmt.Fprintf(w, `postMessage((%s)())`, surfaceInfoJS)
		case "/shared.js":
			w.Header().Set("Content-Type", "text/javascript")
			fmt.Fprintf(w, `onconnect = e => e.ports[0].postMessage((%s)())`, surfaceInfoJS)
		case "/popup":
			fmt.Fprintf(w, `<title>popup</title><script>window.__info = (%s)()</script>`, surfaceInfoJS)
		default:
			fmt.Fprint(w, `<title>plain</title>`)
		}
	}))
	t.Cleanup(ss.main.Close)
	t.Cleanup(ss.frame.Close)
	return ss
}

// collectSurfaces 导航到主页面，收集主页面、专用 / 共享 Worker、iframe、弹窗的身份信号
func collectSurfaces(t *testing.T, p Page, ss *surfaceServers) map[string]map[string]any {
	t.Helper()
	if err := p.Navigate(ss.main.URL + "/"); err != nil {
		t.Fatalf("Navigate: %v", err)
	}
	out := map[string]map[string]any{}
	reported := func() bool { return out["worker"] != nil && out["shared"] != nil && out["iframe"] != nil }
	for deadline := time.Now().Add(10 * time.Second); time.Now().Before(deadline); time.Sleep(100 * time.Millisecond) {
		v, err := p.Evaluate(`JSON.stringify(window.__r)`)
		if s, ok := v.(string); err == nil && ok && json.Unmarshal([]byte(s), &out) == nil && reported() {
			break
		}
	}
	if !reported() {
		t.Fatalf("worker / shared worker / iframe never reported: %v", out)
	}
	if _, err := p.Evaluate(`window.__p = window.open('/popup'), true`); err != nil {
		t.Fatal(err)
	}
	for deadline := time.Now().Add(10 * time.Second); time.Now().Before(deadline); time.Sleep(100 * time.Millisecond) {
		v, _ := p.Evaluate(`window.__p && window.__p.__info ? JSON.stringify(window.__p.__info) : ''`)
		if s, _ := v.(string); s != "" {
			popup := map[string]any{}
			json.Unmarshal([]byte(s), &popup)
			out["popup"] = popup
			break
		}
	}
	if out["popup"] == nil {
		t.Fatal("popup never reported")
	}
	return out
}

// TestIdentityAcrossSurfaces 生产配置下，同一账号在请求头、主页面、跨站 iframe、Worker、弹窗给出完全一致的身份
func TestIdentityAcrossSurfaces(t *testing.T) {
	cases := []struct {
		name, ua, fpUser      string
		wantPlatform, wantUAD string
		wantLangs, wantTZ     string
	}{
		{"windows account", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/128.0.0.0 Safari/537.36",
			"surface-win", "Win32", "Windows", "ja-JP,ja", "Asia/Tokyo"},
		{"mac account", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36",
			"surface-mac", "MacIntel", "macOS", "ja-JP,ja", "Asia/Tokyo"},
		{"no fingerprint", "", "", "", "", "", ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ss := newSurfaceServers(t)
			opts := &ConnectOptions{Headless: true, UseCustomCDP: true, FingerprintDir: t.TempDir()}
			if tc.fpUser != "" {
				opts.FingerprintUserID, opts.UserAgent, opts.Language, opts.Timezone = tc.fpUser, tc.ua, "ja-JP", "Asia/Tokyo"
			}
			inst, err := Connect(t.Context(), opts)
			if err != nil {
				t.Fatalf("Connect: %v", err)
			}
			defer inst.Close()
			surfaces := collectSurfaces(t, inst.Page(), ss) // Connect 返回后立即导航（Review Focus 1）

			main := surfaces["main"]
			ua, _ := main["ua"].(string)
			if strings.Contains(ua, "Headless") {
				t.Errorf("headless UA leaked: %q", ua)
			}
			for name, s := range surfaces {
				for _, k := range []string{"ua", "appVersion", "platform", "langs", "hc", "tz", "date", "locale", "uadPlatform", "brands", "webgl"} {
					if fmt.Sprint(s[k]) != fmt.Sprint(main[k]) {
						t.Errorf("%s.%s = %v, main has %v", name, k, s[k], main[k])
					}
				}
				if name != "worker" && name != "shared" && fmt.Sprint(s["screen"]) != fmt.Sprint(main["screen"]) {
					t.Errorf("%s.screen = %v, main has %v", name, s["screen"], main["screen"])
				}
				if strings.Contains(fmt.Sprint(s["webgl"]), "SwiftShader") {
					t.Errorf("%s exposes SwiftShader", name)
				}
			}
			if !strings.Contains(fmt.Sprint(main["brands"]), "Google Chrome") {
				t.Errorf("brands = %v", main["brands"])
			}
			if strings.HasPrefix(fmt.Sprint(main["screen"]), "800,600") {
				t.Errorf("headless default screen leaked: %v", main["screen"])
			}
			if tc.fpUser != "" {
				for k, want := range map[string]string{"platform": tc.wantPlatform, "uadPlatform": tc.wantUAD, "langs": tc.wantLangs, "tz": tc.wantTZ} {
					if fmt.Sprint(main[k]) != want {
						t.Errorf("main.%s = %v, want %s", k, main[k], want)
					}
				}
			}

			ss.mu.Lock()
			for _, path := range []string{"/", "/worker.js", "/shared.js", "/f", "/popup"} {
				h := ss.headers[path]
				if h == nil {
					t.Errorf("no request seen for %s", path)
					continue
				}
				if h.Get("User-Agent") != ua {
					t.Errorf("%s User-Agent = %q, page says %q", path, h.Get("User-Agent"), ua)
				}
				if tc.fpUser != "" && !strings.HasPrefix(h.Get("Accept-Language"), tc.wantLangs) {
					t.Errorf("%s Accept-Language = %q", path, h.Get("Accept-Language"))
				}
			}
			for _, path := range []string{"/", "/f", "/popup"} {
				if h := ss.headers[path]; h != nil && h.Get("Sec-Ch-Ua-Platform") != `"`+fmt.Sprint(main["uadPlatform"])+`"` {
					t.Errorf("%s Sec-CH-UA-Platform = %q", path, h.Get("Sec-Ch-Ua-Platform"))
				}
			}
			ss.mu.Unlock()

			// 无注入痕迹；WebRTC 为原生
			for js, want := range map[string]any{
				`Object.getOwnPropertyNames(navigator).length`:                                  float64(0),
				`Object.prototype.toString.call(new RTCPeerConnection())`:                       "[object RTCPeerConnection]",
				`Function.prototype.toString.call(RTCPeerConnection).includes('[native code]')`: true,
			} {
				if got, err := inst.Page().Evaluate(js); err != nil || got != want {
					t.Errorf("%s = %v (%v), want %v", js, got, err, want)
				}
			}

			// 受信点击的屏幕坐标与窗口几何一致
			inst.Page().Evaluate(`window.__click = null; addEventListener('click', e => window.__click = [e.screenX - e.clientX, e.screenY - e.clientY, screenX + (outerWidth - innerWidth) / 2, screenY + outerHeight - innerHeight - (outerWidth - innerWidth) / 2].join())`)
			if err := inst.Page().Click(100, 100); err != nil {
				t.Fatal(err)
			}
			time.Sleep(200 * time.Millisecond)
			v, _ := inst.Page().Evaluate(`window.__click`)
			parts := strings.Split(fmt.Sprint(v), ",")
			if len(parts) != 4 || parts[0] != parts[2] || parts[1] != parts[3] {
				t.Errorf("trusted click screen offsets %v do not match window geometry", v)
			}
		})
	}
}

// TestIdentityTargetsThatVanish 跨站 iframe 刚插入就移除、弹窗刚打开就关闭时，页面不会卡住（Review Focus 2）
func TestIdentityTargetsThatVanish(t *testing.T) {
	ss := newSurfaceServers(t)
	inst, err := Connect(t.Context(), &ConnectOptions{Headless: true, UseCustomCDP: true, FingerprintUserID: "vanish", FingerprintDir: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	defer inst.Close()
	p := inst.Page()
	if err := p.Navigate(ss.main.URL + "/plain"); err != nil {
		t.Fatal(err)
	}
	frameURL := strings.Replace(ss.frame.URL, "127.0.0.1", "localhost", 1)
	if _, err := p.Evaluate(fmt.Sprintf(`(() => { for (let i = 0; i < 20; i++) { const f = document.createElement('iframe'); f.src = '%s/f'; document.body.appendChild(f); f.remove(); }
	  window.open('/plain?popup').close(); return true; })()`, frameURL)); err != nil {
		t.Fatal(err)
	}
	if err := p.Navigate(ss.main.URL + "/plain?after"); err != nil {
		t.Fatalf("navigation after vanishing targets: %v", err)
	}
	if title, err := p.GetTitle(); err != nil || title != "plain" {
		t.Errorf("GetTitle() = %q, %v", title, err)
	}
}

// TestConnectRejectsCorruptFingerprint 指纹文件损坏时 Connect 返回错误，而不是静默退回默认身份（Review Focus 3）
func TestConnectRejectsCorruptFingerprint(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "corrupt.json"), []byte("{"), 0o644); err != nil {
		t.Fatal(err)
	}
	inst, err := Connect(t.Context(), &ConnectOptions{Headless: true, UseCustomCDP: true, FingerprintUserID: "corrupt", FingerprintDir: dir})
	if err == nil {
		inst.Close()
		t.Fatal("Connect succeeded with a corrupt fingerprint file")
	}
}

// TestCDPPageIdentity chromedp 路径：主页面的请求头与 JS 是账号身份，没有无界面痕迹；损坏的指纹文件同样报错
func TestCDPPageIdentity(t *testing.T) {
	ss := newSurfaceServers(t)
	inst, err := Connect(t.Context(), &ConnectOptions{
		Headless: true, FingerprintUserID: "cdppage-win", FingerprintDir: t.TempDir(), Language: "ja-JP",
		UserAgent: "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36",
	})
	if err != nil {
		t.Fatal(err)
	}
	defer inst.Close()
	p := inst.Page()
	if err := p.Navigate(ss.main.URL + "/plain"); err != nil {
		t.Fatal(err)
	}
	for js, want := range map[string]any{
		`navigator.platform`:                            "Win32",
		`navigator.userAgentData.platform`:              "Windows",
		`navigator.languages.join()`:                    "ja-JP,ja",
		`navigator.userAgent.includes('Headless')`:      false,
		`screen.width === 800 && screen.height === 600`: false,
		`(() => { const g = document.createElement('canvas').getContext('webgl'); g.getExtension('WEBGL_debug_renderer_info'); return g.getParameter(37446).includes('SwiftShader'); })()`: false,
	} {
		if got, err := p.Evaluate(js); err != nil || got != want {
			t.Errorf("%s = %v (%v), want %v", js, got, err, want)
		}
	}
	ss.mu.Lock()
	ua := ss.headers["/plain"].Get("User-Agent")
	ss.mu.Unlock()
	if got, _ := p.Evaluate(`navigator.userAgent`); got != ua {
		t.Errorf("header UA %q != JS UA %v", ua, got)
	}

	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "corrupt.json"), []byte("{"), 0o644); err != nil {
		t.Fatal(err)
	}
	if bad, err := Connect(t.Context(), &ConnectOptions{Headless: true, FingerprintUserID: "corrupt", FingerprintDir: dir}); err == nil {
		bad.Close()
		t.Error("chromedp path accepted a corrupt fingerprint file")
	}
}
