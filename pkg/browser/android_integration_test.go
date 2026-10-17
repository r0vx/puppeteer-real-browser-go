package browser

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"sort"
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

// androidAPIsJS 安卓与桌面不同的 JS 接口
const androidAPIsJS = `(async () => {
  let notif; try { new Notification('x'); notif = 'constructs'; } catch (e) { notif = e.name + ': ' + e.message; }
  let callNotif; try { Notification('x'); callNotif = 'ok'; } catch (e) { callNotif = e.message; }
  const b = await navigator.getBattery();
  const worker = await new Promise(r => {
    const w = new Worker(URL.createObjectURL(new Blob(['postMessage([navigator.deviceMemory, navigator.platform])'], {type: 'text/javascript'})));
    w.onmessage = e => r(e.data);
  });
  const out = {
    deviceMemory: navigator.deviceMemory, worker, notif, callNotif,
    notifPermission: typeof Notification.permission, notifSource: Function.prototype.toString.call(Notification),
    notifProtoCtor: Notification.prototype.constructor === Notification,
    hasContacts: 'contacts' in navigator, connType: navigator.connection.type,
    connKeys: Object.getOwnPropertyNames(NetworkInformation.prototype).join(), downlinkMax: String(navigator.connection.downlinkMax),
    removed: ['hid', 'windowControlsOverlay'].filter(k => k in navigator)
      .concat(['EyeDropper', 'SharedWorker', 'documentPictureInPicture', 'queryLocalFonts', 'HID', 'FontData',
        'getScreenDetails', 'ScreenDetails', 'ScreenDetailed', 'CaptureController', 'CropTarget', 'RestrictionTarget'].filter(k => k in window))
      .concat(['getDisplayMedia', 'setCaptureHandleConfig'].filter(k => window.MediaDevices && k in MediaDevices.prototype)),
    chromeApp: 'app' in window.chrome, battery: [b.charging, b.chargingTime, b.dischargingTime, b.level],
    share: [typeof navigator.share, typeof navigator.canShare], barcode: typeof BarcodeDetector,
  };
  if (out.hasContacts) {
    out.contacts = String(navigator.contacts);
    out.contactsProps = (await navigator.contacts.getProperties()).join();
    try { await navigator.contacts.select(['name']); out.contactsSelect = 'resolved'; } catch (e) { out.contactsSelect = e.name; }
    out.contactsManager = typeof ContactsManager;
  }
  if (typeof BarcodeDetector === 'function') out.barcodeFormats = (await BarcodeDetector.getSupportedFormats()).length > 0;
  return JSON.stringify(out);
})()`

// TestAndroidSurfaceAPIs 安卓身份的 JS 接口与真机一致；桌面身份不受影响（Review Focus 5）
func TestAndroidSurfaceAPIs(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { fmt.Fprint(w, `<title>plain</title>`) }))
	defer srv.Close()
	run := func(t *testing.T, p Page) map[string]any {
		if err := p.Navigate(srv.URL); err != nil {
			t.Fatal(err)
		}
		out := map[string]any{}
		if err := json.Unmarshal([]byte(awaitJS(t, p, androidAPIsJS)), &out); err != nil {
			t.Fatal(err)
		}
		return out
	}
	t.Run("android", func(t *testing.T) {
		got := run(t, connectAndroid(t, "android-apis", "PGKM10").Page())
		want := map[string]any{
			"deviceMemory": 8.0, "notif": "TypeError: Failed to construct 'Notification': Illegal constructor. Use ServiceWorkerRegistration.showNotification() instead.",
			"notifPermission": "string", "notifSource": "function Notification() { [native code] }", "notifProtoCtor": true,
			"hasContacts": true, "contacts": "[object ContactsManager]", "contactsProps": "address,email,icon,name,tel",
			"contactsSelect": "SecurityError", "contactsManager": "function", "connType": "wifi", "chromeApp": false,
			"connKeys": "onchange,effectiveType,rtt,downlink,saveData,type,downlinkMax,ontypechange,constructor", "downlinkMax": "Infinity",
			"barcode": "function", "barcodeFormats": true,
		}
		for k, v := range want {
			if got[k] != v {
				t.Errorf("%s = %v, want %v", k, got[k], v)
			}
		}
		if w := fmt.Sprint(got["worker"]); w != "[8 Linux armv81]" {
			t.Errorf("worker = %s", w)
		}
		if !strings.Contains(fmt.Sprint(got["callNotif"]), "'new' operator") {
			t.Errorf("Notification() without new = %v", got["callNotif"])
		}
		if r := got["removed"].([]any); len(r) != 0 {
			t.Errorf("desktop-only APIs still present: %v", r)
		}
		b := got["battery"].([]any)
		if b[0] != false || b[1] != nil || b[2] != nil || b[3].(float64) < 0.35 || b[3].(float64) > 0.95 {
			t.Errorf("battery = %v, want discharging with a 0.35–0.95 level (Infinity encodes as null)", b)
		}
		if s := fmt.Sprint(got["share"]); s != "[function function]" {
			t.Errorf("share = %s", s)
		}
	})
	t.Run("desktop", func(t *testing.T) {
		inst, err := Connect(t.Context(), &ConnectOptions{Headless: true, FingerprintUserID: "desktop-apis", FingerprintDir: t.TempDir()})
		if err != nil {
			t.Fatal(err)
		}
		defer inst.Close()
		got := run(t, inst.Page())
		if got["notif"] != "constructs" || len(got["removed"].([]any)) == 0 || got["chromeApp"] != true {
			t.Errorf("desktop identity changed: notif %v, removed %v, chrome.app %v", got["notif"], got["removed"], got["chromeApp"])
		}
	})
}

// loadAndroidBaseline 读取安卓真机基准的 auto 部分
func loadAndroidBaseline(t *testing.T) map[string]any {
	t.Helper()
	raw, err := os.ReadFile("../../test/baseline/android-chrome-pgkm10.json")
	if err != nil {
		t.Fatal(err)
	}
	var r struct {
		Auto map[string]any `json:"auto"`
	}
	if err := json.Unmarshal(raw, &r); err != nil {
		t.Fatal(err)
	}
	return r.Auto
}

// collectOn 在给定页面上打开采集页（见 baseline_collector_test.go），返回 auto 结果
func collectOn(t *testing.T, p Page, srv *httptest.Server) map[string]any {
	t.Helper()
	if err := p.Navigate(srv.URL); err != nil {
		t.Fatal(err)
	}
	waitBaselineState(t, p, "ready")
	raw, err := p.Evaluate(`JSON.stringify(window.__baselineAuto)`)
	if err != nil {
		t.Fatal(err)
	}
	var auto map[string]any
	if err := json.Unmarshal([]byte(fmt.Sprint(raw)), &auto); err != nil {
		t.Fatal(err)
	}
	return auto
}

// TestAndroidWebGL WebGL1 / WebGL2 的标识、能力参数、精度、扩展列表与真机一致；手机上没有的扩展取不到，也不会被悄悄启用（Review Focus 4）
func TestAndroidWebGL(t *testing.T) {
	srv, _ := baselineServer(t, "", http.StatusOK)
	p := connectAndroid(t, "android-webgl", "PGKM10").Page()
	got, want := collectOn(t, p, srv), loadAndroidBaseline(t)
	for _, kind := range []string{"webgl", "webgl2"} {
		g, w := got[kind].(map[string]any), want[kind].(map[string]any)
		for _, k := range []string{"vendor", "renderer", "unmaskedVendor", "unmaskedRenderer", "version", "shadingLanguageVersion", "precision", "extensions"} {
			if fmt.Sprint(g[k]) != fmt.Sprint(w[k]) {
				t.Errorf("%s.%s = %v, want %v", kind, k, g[k], w[k])
			}
		}
		gp, wp := g["params"].(map[string]any), w["params"].(map[string]any)
		for k, v := range wp {
			if fmt.Sprint(gp[k]) != fmt.Sprint(v) {
				t.Errorf("%s.params.%s = %v, want %v", kind, k, gp[k], v)
			}
		}
	}
	// 手机上没有的扩展：取不到，枚举值也不可用
	probe := `(() => { const gl = document.createElement('canvas').getContext('webgl');
	  const s3tc = gl.getExtension('WEBGL_compressed_texture_s3tc'), float = gl.getExtension('OES_texture_float'), aniso = gl.getExtension('EXT_texture_filter_anisotropic');
	  gl.bindTexture(gl.TEXTURE_2D, gl.createTexture());
	  gl.texImage2D(gl.TEXTURE_2D, 0, gl.RGBA, 1, 1, 0, gl.RGBA, gl.FLOAT, null);
	  return [s3tc === null, float === null, aniso !== null, gl.getError() !== 0].join(); })()`
	if v, _ := p.Evaluate(probe); v != "true,true,true,true" {
		t.Errorf("extension probe = %v, want true,true,true,true", v)
	}
}

// TestAndroidFonts Linux 上手机身份能按名字认出的字体与真机一致（只有安卓别名）；桌面身份仍用系统字体
func TestAndroidFonts(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("字体只在 Linux 上由本库控制（macOS 由 CoreText 管理）")
	}
	srv, _ := baselineServer(t, "", http.StatusOK)
	got := collectOn(t, connectAndroid(t, "android-fonts", "PGKM10").Page(), srv)["fonts"].(map[string]any)
	want := loadAndroidBaseline(t)["fonts"].(map[string]any)
	for name, v := range want {
		if got[name] != v {
			t.Errorf("font %q detected = %v, want %v", name, got[name], v)
		}
	}
	inst, err := Connect(t.Context(), &ConnectOptions{Headless: true})
	if err != nil {
		t.Fatal(err)
	}
	defer inst.Close()
	desktop := collectOn(t, inst.Page(), srv)["fonts"].(map[string]any)
	if desktop["Noto Sans CJK SC"] != true {
		t.Error("desktop identity lost the system fonts")
	}
}

// androidVersionGap 宿主 Chrome 与真机基准（Chrome 138）版本不同造成、无法判断真机新版本取值的项目（规格第 7 节）
var androidVersionGap = map[string]string{
	"engine.math.expm1": "引擎级精度差异：ARM 对 x86 或 V8 版本，无法判断",
}

// androidResidual 已知无法模拟、列为残余风险的项目（规格第 9 节）
var androidResidual = map[string]string{
	"media.dynamic-range":          "宿主屏幕不支持高动态范围",
	"screen.visualViewport.height": "真机采集时地址栏状态不同",
	// HEVC 靠手机的硬件解码器；只在接口上谎报支持，网站真推 H.265 视频时会播不了（Linux 宿主没有 HEVC 解码）
	`avmedia.codecs.video/mp4; codecs="hev1.1.6.L93.B0"`: "HEVC 硬件解码",
	`avmedia.codecs.video/mp4; codecs="hvc1.1.6.L93.B0"`: "HEVC 硬件解码",
}

// androidNotCompared 不属于身份、不比较的项目：版本号、用户设置、网络实时值
var androidNotCompared = []string{
	"identity.userAgent", "identity.userAgentData.brands", "identity.userAgentData.fullVersionList",
	"features.connection.effectiveType", "features.connection.rtt", "features.connection.saveData",
	"media.prefers-color-scheme", "avmedia.voices", "avmedia.devices",
}

// diffBaseline 递归比较基准与模拟结果，返回不一致的路径；skip 为真的路径不比较
func diffBaseline(want, got any, path string, skip func(string) bool) []string {
	if skip(path) {
		return nil
	}
	w, wok := want.(map[string]any)
	g, gok := got.(map[string]any)
	if wok && gok {
		var out []string
		for k := range w {
			out = append(out, diffBaseline(w[k], g[k], path+"."+k, skip)...)
		}
		return out
	}
	if fmt.Sprint(want) != fmt.Sprint(got) {
		return []string{fmt.Sprintf("%s = %v, want %v", path, got, want)}
	}
	return nil
}

// TestAndroidMatchesBaseline 用真机同款机型（PGKM10）跑采集页，与真机基准逐项比较；
// 只跳过版本差、残余风险与不属于身份的项目，每一项都写明原因
func TestAndroidMatchesBaseline(t *testing.T) {
	srv, _ := baselineServer(t, "", http.StatusOK)
	got := collectOn(t, connectAndroid(t, "android-baseline", "PGKM10").Page(), srv)
	want := loadAndroidBaseline(t)
	skip := func(path string) bool {
		path = strings.TrimPrefix(path, ".")
		if _, ok := androidVersionGap[path]; ok {
			return true
		}
		if _, ok := androidResidual[path]; ok {
			return true
		}
		for _, p := range androidNotCompared {
			if path == p || strings.HasPrefix(path, p+".") {
				return true
			}
		}
		return path == "fonts" && runtime.GOOS != "linux" // 字体只在 Linux 上由本库控制
	}
	diffs := diffBaseline(want, got, "", skip)
	sort.Strings(diffs)
	for _, d := range diffs {
		t.Error(d)
	}
}

// TestAndroidSetViewportClamps 手机的视口不会比屏幕大：调用方按桌面习惯传 1920×1080（triumph 默认如此）时，
// 宽度限制为屏幕宽、高度限制为手机默认视口高，不出现 innerWidth 1920、screen.width 411 的矛盾
func TestAndroidSetViewportClamps(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { fmt.Fprint(w, mobilePage) }))
	defer srv.Close()
	p := connectAndroid(t, "android-clamp", "22041211AC").Page()
	if err := p.SetViewport(1920, 1080); err != nil {
		t.Fatal(err)
	}
	if err := p.Navigate(srv.URL); err != nil {
		t.Fatal(err)
	}
	if got, _ := p.Evaluate(`[innerWidth, innerHeight, outerWidth, outerHeight, screen.width, screen.height].join()`); got != "411,810,411,810,411,914" {
		t.Errorf("viewport after SetViewport(1920, 1080) = %v, want 411,810,411,810,411,914", got)
	}
}
