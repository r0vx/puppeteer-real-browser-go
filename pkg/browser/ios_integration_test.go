package browser

import (
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"net/http/httptest"
	"slices"
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
		want := map[string]any{"ua": ua, "platform": "iPhone", "cores": 4.0, "languages": "zh-CN", "uad": false, "native": "function max() {\n    [native code]\n}"}
		if surface != "worker" {
			want["touch"], want["coarse"], want["p3"], want["screen"], want["vendor"] = 5.0, true, true, "420,912,3", "Apple Computer, Inc."
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

// iosAPIsJS iPhone Safari 与 Chrome 不同的 JS 接口
const iosAPIsJS = `(async () => {
  const has = (o, ks) => ks.filter(k => k in o);
  const mms = new ManagedMediaSource();
  const v = document.createElement('video');
  const opened = await new Promise(r => { mms.addEventListener('sourceopen', () => r(true)); v.src = URL.createObjectURL(mms); setTimeout(() => r(false), 3000); });
  let sb; try { sb = String(mms.addSourceBuffer('video/mp4; codecs="avc1.42E01E"')); } catch (e) { sb = e.name; }
  let ap; try { ap = ApplePaySession.canMakePayments(); } catch (e) { ap = e.name; }
  let db; try { openDatabase('a', '1', 'a', 1); db = 'opened'; } catch (e) { db = e.name; }
  const worker = await new Promise(r => {
    const src = "postMessage(['userAgentData' in navigator, 'deviceMemory' in navigator, typeof MediaSource, typeof Notification, Function.prototype.toString.call(Math.max)].join('|'))";
    const w = new Worker(URL.createObjectURL(new Blob([src], {type: 'text/javascript'})));
    w.onmessage = e => r(e.data);
  });
  return JSON.stringify({
    vendor: navigator.vendor, standalone: navigator.standalone, standaloneIn: 'standalone' in navigator,
    removedNav: has(navigator, ['userAgentData', 'deviceMemory', 'connection', 'bluetooth', 'getBattery', 'vibrate', 'usb', 'serial', 'hid', 'keyboard', 'xr', 'contacts', 'virtualKeyboard']),
    removedWin: has(window, ['chrome', 'Notification', 'MediaSource', 'BarcodeDetector', 'NavigatorUAData', 'NetworkInformation', 'showOpenFilePicker',
      'webkitRequestFileSystem', 'EyeDropper', 'launchQueue', 'queryLocalFonts', 'getScreenDetails']),
    present: ['ApplePaySession', 'GestureEvent', 'ManagedMediaSource', 'openDatabase', 'SharedWorker'].filter(k => k in window).join(),
    gesture: Object.getPrototypeOf(GestureEvent.prototype) === UIEvent.prototype,
    mms: mms instanceof ManagedMediaSource && mms.streaming === true, opened, sb,
    applePay: ap, db, dbLength: openDatabase.length, perm: (await navigator.permissions.query({name: 'notifications'})).state,
    geo: (await navigator.permissions.query({name: 'geolocation'})).state,
    patched: Function.prototype.toString.call(Object.getOwnPropertyDescriptor(Navigator.prototype, 'vendor').get),
    fake: Function.prototype.toString.call(ApplePaySession),
    user: Function.prototype.toString.call(function f() { return 1; }),
    share: typeof navigator.share, gpu: 'gpu' in navigator, worker,
  });
})()`

// TestIOSSurfaceAPIs iOS 身份的 JS 接口与 iPhone 一致：Chrome 独有的删掉，Safari 独有的补上，原生函数文本是 JavaScriptCore 格式；
// 删掉 MediaSource 后播放器仍能用 ManagedMediaSource（Review Focus 4）；桌面身份不受影响（Review Focus 3）
func TestIOSSurfaceAPIs(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { fmt.Fprint(w, `<title>plain</title>`) }))
	defer srv.Close()
	t.Run("ios", func(t *testing.T) {
		p := connectIOS(t, "ios-apis", "iphone-ios18.5").Page()
		if err := p.Navigate(srv.URL); err != nil {
			t.Fatal(err)
		}
		got := map[string]any{}
		if err := json.Unmarshal([]byte(awaitJS(t, p, iosAPIsJS)), &got); err != nil {
			t.Fatal(err)
		}
		want := map[string]any{
			"vendor": "Apple Computer, Inc.", "standalone": false, "standaloneIn": true,
			"present": "ApplePaySession,GestureEvent,ManagedMediaSource,openDatabase,SharedWorker", "gesture": true,
			"mms": true, "opened": true, "sb": "[object SourceBuffer]", "applePay": true, "db": "SecurityError", "dbLength": 4.0,
			"perm": "denied", "geo": "prompt",
			"patched": "function get vendor() {\n    [native code]\n}", "fake": "function ApplePaySession() {\n    [native code]\n}",
			"user": "function f() { return 1; }", "share": "function", "gpu": false,
			"worker": "false|false|undefined|undefined|function max() {\n    [native code]\n}",
		}
		for k, v := range want {
			if got[k] != v {
				t.Errorf("%s = %v, want %v", k, got[k], v)
			}
		}
		for _, k := range []string{"removedNav", "removedWin"} {
			if r, _ := got[k].([]any); len(r) != 0 {
				t.Errorf("%s: Chrome-only APIs still present: %v", k, r)
			}
		}
	})
	t.Run("desktop", func(t *testing.T) {
		inst, err := Connect(t.Context(), &ConnectOptions{Headless: true, FingerprintUserID: "desktop-ios-apis", FingerprintDir: t.TempDir()})
		if err != nil {
			t.Fatal(err)
		}
		defer inst.Close()
		p := inst.Page()
		if err := p.Navigate(srv.URL); err != nil {
			t.Fatal(err)
		}
		const js = `[navigator.vendor, typeof Notification, 'userAgentData' in navigator, typeof ApplePaySession, Function.prototype.toString.call(Math.max)].join('|')`
		if got, _ := p.Evaluate(js); got != "Google Inc.|function|true|undefined|function max() { [native code] }" {
			t.Errorf("desktop identity changed: %v", got)
		}
	})
}

// TestIOSWebGL WebGL1 / WebGL2 的标识、版本串、能力参数、精度、扩展列表与 iPhone 一致
func TestIOSWebGL(t *testing.T) {
	srv, _ := baselineServer(t, "", http.StatusOK)
	got, want := collectOn(t, connectIOS(t, "ios-webgl", "iphone-ios26").Page(), srv), loadIOSBaseline(t, "ios-safari-iphone-2.json")
	for _, kind := range []string{"webgl", "webgl2"} {
		for _, d := range diffBaseline(want[kind], got[kind], kind, func(string) bool { return false }) {
			t.Error(d)
		}
	}
}

// TestIOSTapAndTypeMatchBaseline 在采集页上点按钮、输入 ab1，与 iPhone（iOS 26）的事件序列与属性比较（Review Focus 5）
func TestIOSTapAndTypeMatchBaseline(t *testing.T) {
	srv, posted := baselineServer(t, "", http.StatusOK)
	p := connectIOS(t, "ios-input", "iphone-ios26").Page()
	sp := p.(*CustomCDPPage)
	collectOn(t, p, srv)
	if err := sp.ClickSelector("#tap-btn"); err != nil {
		t.Fatal(err)
	}
	if err := sp.SendKeys("#type-input", "ab1"); err != nil {
		t.Fatal(err)
	}
	time.Sleep(300 * time.Millisecond)
	if v, _ := p.Evaluate(`document.getElementById('type-input').value`); v != "ab1" {
		t.Errorf("input value %v, want ab1", v)
	}
	if _, err := p.Evaluate(`window.finishBaseline(), true`); err != nil {
		t.Fatal(err)
	}
	var body struct {
		Interaction struct {
			Events []map[string]any `json:"events"`
		} `json:"interaction"`
	}
	select {
	case raw := <-posted:
		if err := json.Unmarshal(raw, &body); err != nil {
			t.Fatal(err)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("nothing posted")
	}
	got, want := body.Interaction.Events, baselineInteraction(t, "ios-safari-iphone-2.json")
	if g, w := tapSequence(got, "tap"), tapSequence(want, "tap"); !slices.Equal(g, w) {
		t.Errorf("tap sequence\n %v\nwant\n %v", g, w)
	}
	down, up, click := firstOf(got, "tap", "pointerdown"), firstOf(got, "tap", "pointerup"), firstOf(got, "tap", "click")
	if down == nil || up == nil || click == nil {
		t.Fatalf("tap incomplete: down %v up %v click %v", down, up, click)
	}
	if down["pointerType"] != "touch" || down["pressure"] != 0.0 || math.Abs(down["width"].(float64)-24.16) > 0.01 || down["isPrimary"] != true {
		t.Errorf("pointerdown %v", down)
	}
	if up["pressure"] != 0.0 || click["pointerType"] != "mouse" || click["width"] != 1.0 {
		t.Errorf("pointerup %v / click %v", up, click)
	}
	// Safari 的屏幕坐标与页面坐标相同
	for _, e := range []map[string]any{down, up, click, firstOf(got, "tap", "mousedown")} {
		if e != nil && e["screenY"] != e["clientY"] {
			t.Errorf("%s screenY %v, clientY %v", e["type"], e["screenY"], e["clientY"])
		}
	}
	if hold := up["rt"].(float64) - down["rt"].(float64); hold < 20 || hold > 110 {
		t.Errorf("finger held %.0fms, want 20–110ms (iPhone about 28ms)", hold)
	}
	// 按键字段与真机一致：keydown 为 229，keyup 为按键码
	var keys, updates []string
	for _, e := range got {
		if e["step"] != "type" {
			continue
		}
		switch e["type"] {
		case "keydown", "keyup":
			keys = append(keys, fmt.Sprintf("%v:%v:%v:%v", e["type"], e["key"], e["code"], e["keyCode"]))
		case "compositionupdate":
			if d := fmt.Sprint(e["data"]); len(updates) == 0 || updates[len(updates)-1] != d {
				updates = append(updates, d)
			}
		}
	}
	if w := "keydown:a:KeyA:229,keyup:a:KeyA:65,keydown:b:KeyB:229,keyup:b:KeyB:66,keydown:1:Digit1:229,keyup:1:Digit1:49"; strings.Join(keys, ",") != w {
		t.Errorf("keys\n %s\nwant\n %s", strings.Join(keys, ","), w)
	}
	if strings.Join(updates, ",") != "a,ab,ab1" {
		t.Errorf("composition updates %v, want a,ab,ab1", updates)
	}
}

// TestIOSEnterKey 回车是 iOS 键盘的 Enter 键（code Enter、keyCode 13），产生换行；打字之后照样能点按（Review Focus 5）
func TestIOSEnterKey(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { fmt.Fprint(w, enterPage) }))
	defer srv.Close()
	p := connectIOS(t, "ios-enter", "iphone-ios26").Page()
	sp := p.(*CustomCDPPage)
	if err := p.Navigate(srv.URL); err != nil {
		t.Fatal(err)
	}
	if err := sp.SendKeys("#i", "x\n"); err != nil {
		t.Fatal(err)
	}
	if err := sp.ClickSelector("#i"); err != nil {
		t.Fatal(err)
	}
	raw, _ := p.Evaluate(`window.__k.join(',')`)
	got := fmt.Sprint(raw)
	for _, want := range []string{"keydown|x|KeyX|229|", "keyup|x|KeyX|88|", "keydown|Enter|Enter|13|", "beforeinput|||0|insertLineBreak", "keyup|Enter|Enter|13|"} {
		if !strings.Contains(got, want) {
			t.Errorf("events %s lack %s", got, want)
		}
	}
	if v, _ := p.Evaluate(`document.getElementById('i').value`); v != "x" {
		t.Errorf("input value %v, want x", v)
	}
}

// TestIOSSwipeScrolls iOS 上 RealScroll 同样是手指拖动，滚动距离接近请求值；触点的屏幕坐标与页面坐标相同
func TestIOSSwipeScrolls(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { fmt.Fprint(w, swipePage) }))
	defer srv.Close()
	p := connectIOS(t, "ios-swipe", "iphone-ios26").Page()
	if err := p.Navigate(srv.URL); err != nil {
		t.Fatal(err)
	}
	if err := p.(*CustomCDPPage).RealScroll(0, 900); err != nil {
		t.Fatal(err)
	}
	time.Sleep(500 * time.Millisecond)
	if y, _ := p.Evaluate(`scrollY`); y.(float64) < 900-30 || y.(float64) > 900+30 {
		t.Errorf("scrollY = %v after RealScroll(0, 900), want 900 ± 30", y)
	}
	if off, _ := p.Evaluate(`window.__touchOff`); off != 0.0 {
		t.Errorf("Touch screenY - clientY = %v, want 0", off)
	}
}
