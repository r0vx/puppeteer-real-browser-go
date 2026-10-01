package browser

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"
)

// waitLoaded 轮询直到会话中的文档加载完成
func waitLoaded(t *testing.T, call cdpCaller, url string) {
	t.Helper()
	for deadline := time.Now().Add(10 * time.Second); time.Now().Before(deadline); time.Sleep(50 * time.Millisecond) {
		if v, err := evaluateString(call, `document.readyState + ' ' + location.href`); err == nil && v == "complete "+url {
			return
		}
	}
	t.Fatalf("%s did not load", url)
}

// TestApplyIdentityOnPage 页面目标下发后，请求头、JS、client hints、屏幕、WebGL 都是档案身份；readHostInfo 读到真实无界面值
func TestApplyIdentityOnPage(t *testing.T) {
	var mu sync.Mutex
	var seen http.Header
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/" {
			mu.Lock()
			seen = r.Header.Clone()
			mu.Unlock()
		}
		fmt.Fprint(w, `<title>apply</title>`)
	}))
	defer srv.Close()

	chrome, err := NewChromeLauncher().Launch(t.Context(), &ConnectOptions{Headless: true})
	if err != nil {
		t.Fatal(err)
	}
	defer chrome.Kill()
	conn, err := dialBrowser(chrome.Port)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.close()

	initial, err := initialPageTarget(conn)
	if err != nil {
		t.Fatal(err)
	}
	host, err := readHostInfo(conn, initial)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(host.UserAgent, "HeadlessChrome/") || host.majorVersion() == "" {
		t.Fatalf("unexpected host info: %+v", host)
	}

	cfg := generateFingerprintFor("apply-user", OSWindows)
	cfg.Browser.Language, cfg.Browser.Languages = "ja-JP", nil
	cfg.Timezone.Timezone = "Asia/Tokyo"
	cfg.Normalize()
	id, err := identityFromConfig(cfg, host)
	if err != nil {
		t.Fatal(err)
	}

	raw, err := conn.call("", "Target.createTarget", map[string]any{"url": "about:blank"})
	if err != nil {
		t.Fatal(err)
	}
	var created struct{ TargetID string }
	json.Unmarshal(raw, &created)
	raw, err = conn.call("", "Target.attachToTarget", map[string]any{"targetId": created.TargetID, "flatten": true})
	if err != nil {
		t.Fatal(err)
	}
	var attached struct{ SessionID string }
	json.Unmarshal(raw, &attached)
	call := func(m string, p any) (json.RawMessage, error) { return conn.call(attached.SessionID, m, p) }

	if err := applyIdentity(call, id, kindPage); err != nil {
		t.Fatalf("applyIdentity: %v", err)
	}
	url := srv.URL + "/"
	if _, err := call("Page.navigate", map[string]any{"url": url}); err != nil {
		t.Fatal(err)
	}
	waitLoaded(t, call, url)

	s := id.Screen
	cases := []struct{ js, want string }{
		{`navigator.userAgent`, id.UserAgent},
		{`navigator.platform`, "Win32"},
		{`navigator.languages.join()`, "ja-JP,ja"},
		{`Intl.DateTimeFormat().resolvedOptions().timeZone`, "Asia/Tokyo"},
		{`String(navigator.hardwareConcurrency)`, strconv.Itoa(id.HardwareConcurrency)},
		{`navigator.userAgentData.platform`, "Windows"},
		{`navigator.userAgentData.getHighEntropyValues(['formFactors', 'platformVersion']).then(v => v.formFactors.join() + '|' + v.platformVersion)`, "Desktop|" + id.Metadata.PlatformVersion},
		{`[screen.width, screen.height, innerWidth, innerHeight, devicePixelRatio].join()`, fmt.Sprintf("%d,%d,%d,%d,%v", s.Width, s.Height, s.InnerWidth, s.InnerHeight, s.DPR)},
		{`(() => { const g = document.createElement('canvas').getContext('webgl'); g.getExtension('WEBGL_debug_renderer_info'); return g.getParameter(37446); })()`, id.GPU.Renderer},
	}
	for _, tc := range cases {
		got, err := evaluateString(call, tc.js)
		if err != nil || got != tc.want {
			t.Errorf("%s = %q (%v), want %q", tc.js, got, err, tc.want)
		}
	}

	mu.Lock()
	defer mu.Unlock()
	if ua := seen.Get("User-Agent"); ua != id.UserAgent {
		t.Errorf("User-Agent header = %q", ua)
	}
	if p := seen.Get("Sec-Ch-Ua-Platform"); p != `"Windows"` {
		t.Errorf("Sec-CH-UA-Platform header = %q", p)
	}
	if al := seen.Get("Accept-Language"); !strings.HasPrefix(al, "ja-JP,ja") {
		t.Errorf("Accept-Language header = %q", al)
	}
}
