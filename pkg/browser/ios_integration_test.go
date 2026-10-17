package browser

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// connectIOS 用指定机型的 iOS 身份启动（无界面，默认通道）
func connectIOS(t *testing.T, userID, model string) *BrowserInstance {
	t.Helper()
	inst, err := Connect(t.Context(), &ConnectOptions{Headless: true, Mobile: true, FingerprintUserID: userID, FingerprintDir: mobileFingerprintDir(t, OSIOS, userID, model)})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { inst.Close() })
	return inst
}

// iosSurfaceJS 各表面（页面、iframe、Worker、弹窗）上报的身份信号（Safari 没有 userAgentData，只看它在不在）
const iosSurfaceJS = `async () => ({ua: navigator.userAgent, platform: navigator.platform, cores: navigator.hardwareConcurrency,
  languages: navigator.languages.join(), uad: 'userAgentData' in navigator, vendor: self.document ? navigator.vendor : null,
  native: Function.prototype.toString.call(Math.max),
  touch: self.document ? navigator.maxTouchPoints : null, coarse: self.document ? matchMedia('(pointer: coarse)').matches : null,
  p3: self.document ? matchMedia('(color-gamut: p3)').matches : null,
  screen: self.document ? [screen.width, screen.height, devicePixelRatio].join() : null})`

// TestIOSIdentityAcrossSurfaces 主页面、跨站 iframe、专用 Worker、弹窗与请求头都是同一台 iPhone：Safari 的 UA，请求头没有 Client Hints（Review Focus 2）
func TestIOSIdentityAcrossSurfaces(t *testing.T) {
	srv, headers := newMobileSurfaceServers(t, iosSurfaceJS)
	p := connectIOS(t, "ios-surfaces", "iphone-ios26").Page()
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
	ua := iosUserAgent(iosDevices[0].Value)
	for _, surface := range []string{"main", "iframe", "worker", "popup"} {
		info := got[surface]
		if info == nil {
			t.Errorf("%s never reported", surface)
			continue
		}
		want := map[string]any{"ua": ua, "platform": "iPhone", "cores": 4.0, "languages": "zh-CN"}
		if surface != "worker" {
			want["touch"], want["coarse"], want["p3"], want["screen"] = 5.0, true, true, "420,912,3"
		}
		for k, v := range want {
			if info[k] != v {
				t.Errorf("%s %s = %v, want %v", surface, k, info[k], v)
			}
		}
	}
	for _, path := range []string{"/", "/f", "/worker.js"} {
		h := headers(path)
		if h.Get("User-Agent") != ua {
			t.Errorf("%s User-Agent %q", path, h.Get("User-Agent"))
		}
		for k := range h {
			if strings.HasPrefix(strings.ToLower(k), "sec-ch-ua") {
				t.Errorf("%s sends %s: %v (Safari sends no client hints)", path, k, h[k])
			}
		}
	}
}

// TestIOSLaunchAndViewport 插件与 PDF 查看器保持原生（Safari 同样报 5 个插件）；视口即整屏；SetViewport 按桌面习惯传 1920×1080 时限制在屏幕内
func TestIOSLaunchAndViewport(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { fmt.Fprint(w, mobilePage) }))
	defer srv.Close()
	p := connectIOS(t, "ios-launch", "iphone-ios18.5").Page()
	const js = `[navigator.plugins.length, navigator.mimeTypes.length, navigator.pdfViewerEnabled, innerWidth, innerHeight, outerWidth, outerHeight,
	  screen.width, screen.height, devicePixelRatio, screen.orientation.type, matchMedia('(pointer: coarse)').matches].join()`
	if err := p.Navigate(srv.URL); err != nil {
		t.Fatal(err)
	}
	if got, _ := p.Evaluate(js); got != "5,2,true,440,956,440,956,440,956,3,portrait-primary,true" {
		t.Errorf("launch state %v", got)
	}
	if err := p.SetViewport(1920, 1080); err != nil {
		t.Fatal(err)
	}
	if err := p.Navigate(srv.URL + "/?after"); err != nil {
		t.Fatal(err)
	}
	if got, _ := p.Evaluate(js); got != "5,2,true,440,956,440,956,440,956,3,portrait-primary,true" {
		t.Errorf("after SetViewport(1920, 1080) %v", got)
	}
}
