# 真机基准采集（M0）Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 做一个采集页、一个本地服务和一个启动脚本。用户在小米 Chrome 与 iPhone Safari 上各采集一份浏览器表面数据，存进 `test/baseline/`，作为移动身份 M1–M3 的数据来源和测试对照。

**Architecture:**
- 单文件采集页 `test/baseline/collector.html`：导出 `window.collectBaseline()`（自动采集），记录三步交互事件，最后 `POST result`。
- Go 小服务 `cmd/baseline_server`：只监听 127.0.0.1，提供采集页并把结果存为 JSON。
- `scripts/collect-baseline.sh`：用 cloudflared quick tunnel 临时暴露成 https 地址。
- 采集页在 `pkg/browser` 的集成测试里由本库 Chrome 加载，保证每项检测都能跑通。

**Tech Stack:** Go 1.25 标准库（`net/http`、`encoding/json`），原生 JavaScript（Safari 17 / 18 与 Chrome 都支持的语法），bash，cloudflared。

**Spec:** `docs/superpowers/specs/2026-10-17-mobile-baseline-collector-design.md`

## Global Constraints

- 采集页单文件、不加载任何外部资源；每项检测单独兜底，抛错时该项记为 `{"error": "<消息>"}`，不影响其他项。
- 结果顶层：`{"schema": 1, "collectedAt": "<ISO 时间>", "auto": {…}, "interaction": {…}}`。`auto` 的组：`identity`、`screen`、`features`、`media`、`webgl`、`webgl2`、`fonts`、`engine`、`avmedia`。
- 交互记录最多 600 个事件，超出后丢弃，并标记 `truncated: true`。
- 服务只监听 `127.0.0.1:<端口>`，默认端口 8765，参数 `-port`、`-out`（默认 `test/baseline`）、`-page`（默认 `test/baseline/collector.html`）。
- 服务接口：
  - `POST /result` 请求体超过 1 MiB 返回 413；不是 JSON 或缺少 `auto` 返回 400；成功返回 `{"file": "<文件名>"}`。
  - 文件名 `<系统>-<浏览器>-<机型>.json`：系统为 `android` / `ios` / `other`，浏览器为 `chrome` / `safari` / `other`；机型取 Client Hints 的 `model`，iOS 没有时为 `iphone`；只保留小写字母、数字和连字符；同名时加 `-2`、`-3`，不覆盖。
  - 不记录请求方地址。
- 不采集 IP、地理位置、通讯录、媒体设备名称，不申请任何权限；调用栈样本中的地址替换为 `<url>`。
- 代码注释用中文，每个函数都有注释；JSON 用 `encoding/json`（与仓库现有代码一致）。
- 提交信息以 `Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>` 结尾。

## Review Focus

1. 真机上某个接口不存在或调用报错（Safari 没有 Client Hints、`permissions.query` 不认某个名称、没有 WebGL2、没有 MediaSource）：只有该项为 `null` 或 `{"error"}`，其余照常采集并上传。由 Task 1 的 `TestBaselineCollectorSurvivesMissingAPIs` 覆盖。
2. 上传失败（隧道断开、服务返回 5xx）：页面显示"复制结果"备用，文本框里是完整结果。由 Task 1 的 `TestBaselineCollectorUpload/server_error` 覆盖。
3. 滑动产生大量事件：最多记 600 个并标记 `truncated`，上传体仍远小于 1 MiB。由 Task 1 的 `TestBaselineCollectorUpload/ok` 覆盖。
4. 用户误用小米自带浏览器（或 iOS 上的 Chrome）：文件名里浏览器为 `other`，检查时能发现。由 Task 2 的 `TestSaveResultNames/xiaomi_browser` 覆盖。
5. 调用栈样本带出页面地址（cloudflared 的随机域名）：替换为 `<url>`。由 Task 1 的 `TestBaselineCollectorRuns` 覆盖。

---

### Task 1: 采集页与集成测试

**Files:**
- Create: `test/baseline/collector.html`
- Test: `pkg/browser/baseline_collector_test.go`

**Interfaces:**
- Consumes: 无
- Produces（Task 2、Task 4 和后续 M1–M3 依赖）：
  - 页面全局：`window.collectBaseline(): Promise<object>`（返回 `auto`）。
  - `window.__baselineState`：`'collecting'`、`'ready'`、`'uploaded'`、`'upload-failed'` 之一。
  - `window.__baselineAuto`：自动采集结果。`window.__baselineResult`：上传的完整结果。
  - `window.finishBaseline(): Promise<void>`：结束交互并上传。
  - 上传：`POST result`（相对地址），期望应答 `{"file": "<名>"}`。
  - 测试辅助（`pkg/browser`）：`baselineServer(t, shim string, status int) (*httptest.Server, chan []byte)`、`openBaseline(t, srv) (Page, map[string]any)`、`waitBaselineState(t, p Page, want ...string) string`、`probeErrors(v any, path string) []string`、`baselineGroup(t, auto, name) map[string]any`。

- [ ] **Step 1: 写失败的测试**

创建 `pkg/browser/baseline_collector_test.go`：

```go
package browser

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"slices"
	"strings"
	"testing"
	"time"
)

// baselineGroups collectBaseline() 必须返回的各组
var baselineGroups = []string{"identity", "screen", "features", "media", "webgl", "webgl2", "fonts", "engine", "avmedia"}

// safariShim 模拟 Safari 没有或会报错的接口：无 Client Hints、permissions.query 拒绝、无 WebGL2、无语音、无 MediaSource
const safariShim = `
delete Navigator.prototype.userAgentData;
Permissions.prototype.query = function () { return Promise.reject(new TypeError('Type error')); };
const origGetContext = HTMLCanvasElement.prototype.getContext;
HTMLCanvasElement.prototype.getContext = function (kind, ...rest) { return kind === 'webgl2' ? null : origGetContext.call(this, kind, ...rest); };
delete window.speechSynthesis;
delete window.MediaSource;
`

// baselineServer 提供采集页（shim 非空时插在 <head> 之后、先于页面脚本执行），
// /result 用 status 应答，并把收到的请求体发到返回的通道
func baselineServer(t *testing.T, shim string, status int) (*httptest.Server, chan []byte) {
	t.Helper()
	page, err := os.ReadFile("../../test/baseline/collector.html")
	if err != nil {
		t.Fatal(err)
	}
	if shim != "" {
		page = bytes.Replace(page, []byte("<head>"), []byte("<head><script>"+shim+"</script>"), 1)
	}
	posted := make(chan []byte, 1)
	mux := http.NewServeMux()
	mux.HandleFunc("GET /{$}", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Write(page)
	})
	mux.HandleFunc("POST /result", func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		select {
		case posted <- body:
		default:
		}
		w.WriteHeader(status)
		fmt.Fprint(w, `{"file": "test.json"}`)
	})
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return srv, posted
}

// openBaseline 用本库 Chrome 打开采集页，等自动采集完成，返回页面与 auto 结果
func openBaseline(t *testing.T, srv *httptest.Server) (Page, map[string]any) {
	t.Helper()
	inst, err := Connect(t.Context(), &ConnectOptions{Headless: true})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { inst.Close() })
	p := inst.Page()
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
		t.Fatalf("decode auto: %v", err)
	}
	return p, auto
}

// waitBaselineState 轮询页面的 __baselineState，直到是 want 之一（最多 20 秒），返回该状态
func waitBaselineState(t *testing.T, p Page, want ...string) string {
	t.Helper()
	deadline := time.Now().Add(20 * time.Second)
	for {
		v, _ := p.Evaluate(`window.__baselineState || ''`)
		if s := fmt.Sprint(v); slices.Contains(want, s) {
			return s
		}
		if time.Now().After(deadline) {
			t.Fatalf("baseline state %v, want one of %v", v, want)
		}
		time.Sleep(100 * time.Millisecond)
	}
}

// probeErrors 找出结果里所有兜底错误项（只有一个 error 键的对象）的路径
func probeErrors(v any, path string) []string {
	var out []string
	switch x := v.(type) {
	case map[string]any:
		if _, ok := x["error"]; ok && len(x) == 1 {
			return []string{path}
		}
		for k, child := range x {
			out = append(out, probeErrors(child, path+"."+k)...)
		}
	case []any:
		for i, child := range x {
			out = append(out, probeErrors(child, fmt.Sprintf("%s[%d]", path, i))...)
		}
	}
	return out
}

// baselineGroup 取 auto 中的一组对象，不存在或不是对象时测试失败
func baselineGroup(t *testing.T, auto map[string]any, name string) map[string]any {
	t.Helper()
	g, ok := auto[name].(map[string]any)
	if !ok {
		t.Fatalf("group %q = %v, want object", name, auto[name])
	}
	return g
}

// TestBaselineCollectorRuns 桌面 Chrome 上采集页每一项都能跑通；调用栈样本不含页面地址（Review Focus 5）
func TestBaselineCollectorRuns(t *testing.T) {
	srv, _ := baselineServer(t, "", http.StatusOK)
	_, auto := openBaseline(t, srv)
	for _, g := range baselineGroups {
		if _, ok := auto[g]; !ok {
			t.Errorf("group %q missing", g)
		}
	}
	if auto["webgl"] == nil {
		t.Error("webgl is null on desktop Chrome")
	}
	if errs := probeErrors(auto, "auto"); len(errs) > 0 {
		slices.Sort(errs)
		t.Errorf("probes failed: %v", errs)
	}
	if ua, _ := baselineGroup(t, auto, "identity")["userAgent"].(string); ua == "" {
		t.Error("identity.userAgent is empty")
	}
	stack := fmt.Sprint(baselineGroup(t, auto, "engine")["stackSample"])
	if strings.Contains(stack, "http") || strings.Contains(stack, "127.0.0.1") || !strings.Contains(stack, "<url>") {
		t.Errorf("stack sample not scrubbed: %q", stack)
	}
	if present, _ := baselineGroup(t, auto, "features")["present"].(map[string]any); len(present) < 40 {
		t.Errorf("only %d API presence checks, want >= 40", len(present))
	}
	if fonts := baselineGroup(t, auto, "fonts"); len(fonts) < 80 {
		t.Errorf("only %d fonts checked, want >= 80", len(fonts))
	}
}

// TestBaselineCollectorSurvivesMissingAPIs 接口缺失或报错时只影响对应项，其余照常采集（Review Focus 1）
func TestBaselineCollectorSurvivesMissingAPIs(t *testing.T) {
	srv, _ := baselineServer(t, safariShim, http.StatusOK)
	_, auto := openBaseline(t, srv)
	for _, g := range baselineGroups {
		if _, ok := auto[g]; !ok {
			t.Errorf("group %q missing", g)
		}
	}
	if uad := baselineGroup(t, auto, "identity")["userAgentData"]; uad != nil {
		t.Errorf("userAgentData = %v, want null", uad)
	}
	if auto["webgl2"] != nil {
		t.Errorf("webgl2 = %v, want null", auto["webgl2"])
	}
	if auto["webgl"] == nil {
		t.Error("webgl lost although only webgl2 is missing")
	}
	if voices := baselineGroup(t, auto, "avmedia")["voices"]; voices != nil {
		t.Errorf("voices = %v, want null", voices)
	}
	errs := probeErrors(auto, "auto")
	slices.Sort(errs)
	want := []string{"auto.features.permissions.camera", "auto.features.permissions.geolocation", "auto.features.permissions.notifications"}
	if !slices.Equal(errs, want) {
		t.Errorf("probe errors %v, want %v", errs, want)
	}
}

// TestBaselineCollectorUpload 上传成功时结果送达，交互事件最多 600 个并标记 truncated（Review Focus 3）；
// 上传失败时显示复制备用，文本框里是完整结果（Review Focus 2）
func TestBaselineCollectorUpload(t *testing.T) {
	t.Run("ok", func(t *testing.T) {
		srv, posted := baselineServer(t, "", http.StatusOK)
		p, _ := openBaseline(t, srv)
		if _, err := p.Evaluate(`(() => {
			const b = document.getElementById('tap-btn');
			for (let i = 0; i < 700; i++) b.dispatchEvent(new PointerEvent('pointermove', {bubbles: true}));
			window.finishBaseline();
			return true;
		})()`); err != nil {
			t.Fatal(err)
		}
		if s := waitBaselineState(t, p, "uploaded", "upload-failed"); s != "uploaded" {
			t.Fatalf("state %q, want uploaded", s)
		}
		var body struct {
			Schema      int            `json:"schema"`
			Auto        map[string]any `json:"auto"`
			Interaction struct {
				Events    []map[string]any `json:"events"`
				Truncated bool             `json:"truncated"`
			} `json:"interaction"`
		}
		select {
		case raw := <-posted:
			if err := json.Unmarshal(raw, &body); err != nil {
				t.Fatalf("decode posted result: %v", err)
			}
		case <-time.After(5 * time.Second):
			t.Fatal("nothing posted to /result")
		}
		if body.Schema != 1 || body.Auto == nil {
			t.Errorf("posted schema %d, auto present %v", body.Schema, body.Auto != nil)
		}
		if n := len(body.Interaction.Events); n != 600 || !body.Interaction.Truncated {
			t.Errorf("events %d truncated %v, want 600 / true", n, body.Interaction.Truncated)
		}
	})
	t.Run("server error", func(t *testing.T) {
		srv, _ := baselineServer(t, "", http.StatusInternalServerError)
		p, _ := openBaseline(t, srv)
		if _, err := p.Evaluate(`window.finishBaseline(), true`); err != nil {
			t.Fatal(err)
		}
		if s := waitBaselineState(t, p, "uploaded", "upload-failed"); s != "upload-failed" {
			t.Fatalf("state %q, want upload-failed", s)
		}
		if hidden, _ := p.Evaluate(`document.getElementById('fallback').classList.contains('hidden')`); hidden != false {
			t.Error("copy fallback is hidden after a failed upload")
		}
		raw, _ := p.Evaluate(`document.getElementById('result-text').value`)
		var result struct {
			Schema int            `json:"schema"`
			Auto   map[string]any `json:"auto"`
		}
		if err := json.Unmarshal([]byte(fmt.Sprint(raw)), &result); err != nil || result.Schema != 1 || result.Auto == nil {
			t.Errorf("fallback text is not the full result (%v)", err)
		}
	})
}
```

- [ ] **Step 2: 运行测试，确认失败**

Run: `go test -race -count=1 -run TestBaselineCollector ./pkg/browser/`
Expected: FAIL，三个测试都报 `open ../../test/baseline/collector.html: no such file or directory`

- [ ] **Step 3: 写采集页**

创建 `test/baseline/collector.html`：

```html
<!doctype html>
<html lang="zh-CN">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>真机基准采集</title>
<style>
  body { font: 16px/1.5 -apple-system, system-ui, sans-serif; margin: 16px; color: #222; background: #fff; }
  .step { border: 1px solid #ccc; border-radius: 8px; padding: 12px; margin: 12px 0; }
  .done { border-color: #2a2; background: #f3fff3; }
  button { font-size: 18px; padding: 10px 20px; }
  #type-input { font-size: 18px; width: 90%; padding: 8px; }
  #swipe-zone { height: 180px; overflow-y: scroll; border: 1px dashed #888; }
  #swipe-zone div { height: 900px; background: linear-gradient(#eef, #fee); }
  #result-text { width: 100%; height: 160px; font-size: 12px; }
  .hidden { display: none; }
</style>
</head>
<body>
<h1>真机基准采集</h1>
<p id="status">正在采集浏览器信息…</p>
<div id="step-tap" class="step">1. 用手指点一下：<button id="tap-btn" type="button">点我</button></div>
<div id="step-type" class="step">2. 点输入框，用手机键盘输入 <b>ab1</b>：<br>
  <input id="type-input" autocomplete="off" autocapitalize="off" autocorrect="off" spellcheck="false"></div>
<div id="step-swipe" class="step">3. 在下面的区域里上下滑动一次：<div id="swipe-zone"><div></div></div></div>
<p><button id="skip-btn" type="button">跳过剩余步骤并上传</button></p>
<div id="fallback" class="hidden">
  <p>上传失败。请点“复制结果”，把内容发给开发者。</p>
  <button id="copy-btn" type="button">复制结果</button>
  <textarea id="result-text" readonly></textarea>
</div>
<script>
(function () {
  'use strict';

  // safe 单项检测兜底：抛错时记为 {error: 消息}，不影响其他项
  function safe(fn) {
    try { return fn(); } catch (e) { return { error: String((e && e.message) || e) }; }
  }

  // safeAsync 异步检测的兜底
  async function safeAsync(fn) {
    try { return await fn(); } catch (e) { return { error: String((e && e.message) || e) }; }
  }

  // withTimeout 超时后返回 fallback
  function withTimeout(p, ms, fallback) {
    return Promise.race([p, new Promise(function (r) { setTimeout(function () { r(fallback); }, ms); })]);
  }

  // orNull undefined 记为 null，便于区分"没有"和"值为 false"
  function orNull(v) { return v === undefined ? null : v; }

  // identity 系统与浏览器标识，含 Client Hints（Safari 没有时为 null）
  async function identity() {
    const n = navigator;
    const out = {
      userAgent: n.userAgent, platform: n.platform, vendor: n.vendor, language: n.language,
      languages: Array.from(n.languages || []), hardwareConcurrency: orNull(n.hardwareConcurrency),
      deviceMemory: orNull(n.deviceMemory), maxTouchPoints: orNull(n.maxTouchPoints), webdriver: orNull(n.webdriver),
      userAgentData: null,
    };
    const uad = n.userAgentData;
    if (uad) {
      const base = { brands: uad.brands.map(function (b) { return { brand: b.brand, version: b.version }; }), mobile: uad.mobile, platform: uad.platform };
      const high = await safeAsync(function () {
        return uad.getHighEntropyValues(['architecture', 'bitness', 'model', 'platformVersion', 'fullVersionList', 'formFactors', 'wow64']);
      });
      out.userAgentData = Object.assign(base, high.error ? { highEntropy: high } : {
        architecture: orNull(high.architecture), bitness: orNull(high.bitness), model: orNull(high.model),
        platformVersion: orNull(high.platformVersion), fullVersionList: orNull(high.fullVersionList),
        formFactors: orNull(high.formFactors), wow64: orNull(high.wow64),
      });
    }
    return out;
  }

  // screenInfo 屏幕、视口、窗口与方向
  function screenInfo() {
    const s = screen, vv = window.visualViewport;
    return {
      screen: { width: s.width, height: s.height, availWidth: s.availWidth, availHeight: s.availHeight,
        availTop: orNull(s.availTop), availLeft: orNull(s.availLeft), colorDepth: s.colorDepth, pixelDepth: s.pixelDepth },
      devicePixelRatio: window.devicePixelRatio,
      innerWidth: window.innerWidth, innerHeight: window.innerHeight, outerWidth: window.outerWidth, outerHeight: window.outerHeight,
      visualViewport: vv ? { width: vv.width, height: vv.height, scale: vv.scale, offsetTop: vv.offsetTop } : null,
      orientation: s.orientation ? { type: s.orientation.type, angle: s.orientation.angle } : null,
      windowOrientation: 'orientation' in window ? orNull(window.orientation) : null,
    };
  }

  const WINDOW_APIS = ['showOpenFilePicker', 'showDirectoryPicker', 'queryLocalFonts', 'BarcodeDetector', 'DeviceMotionEvent',
    'DeviceOrientationEvent', 'ontouchstart', 'TouchEvent', 'ApplePaySession', 'chrome', 'PaymentRequest', 'Notification',
    'speechSynthesis', 'SharedWorker', 'webkitSpeechRecognition', 'EyeDropper', 'documentPictureInPicture', 'launchQueue',
    'webkitRequestFileSystem', 'openDatabase', 'GestureEvent', 'WebKitCSSMatrix', 'visualViewport', 'MediaSource',
    'ManagedMediaSource'];
  const NAVIGATOR_APIS = ['hid', 'serial', 'usb', 'bluetooth', 'keyboard', 'windowControlsOverlay', 'contacts', 'share',
    'canShare', 'getBattery', 'vibrate', 'wakeLock', 'virtualKeyboard', 'serviceWorker', 'mediaDevices', 'credentials', 'gpu',
    'xr', 'scheduling', 'userActivation', 'ink', 'presentation', 'devicePosture', 'getInstalledRelatedApps', 'setAppBadge',
    'webkitTemporaryStorage', 'connection', 'standalone'];

  // permissionStates 只查询权限状态，不申请
  async function permissionStates() {
    const out = {};
    for (const name of ['notifications', 'geolocation', 'camera']) {
      out[name] = await safeAsync(async function () {
        if (!navigator.permissions) return null;
        return (await navigator.permissions.query({ name: name })).state;
      });
    }
    return out;
  }

  // features 接口有无，以及手机与桌面行为不同的几项
  async function features() {
    const present = {};
    for (const k of WINDOW_APIS) present['window.' + k] = safe(function () { return k in window; });
    for (const k of NAVIGATOR_APIS) present['navigator.' + k] = safe(function () { return k in navigator; });
    const c = navigator.connection;
    return {
      present: present,
      plugins: safe(function () { return navigator.plugins.length; }),
      mimeTypes: safe(function () { return navigator.mimeTypes.length; }),
      pdfViewerEnabled: safe(function () { return orNull(navigator.pdfViewerEnabled); }),
      notificationConstructor: safe(function () {
        try { new Notification('baseline'); return 'constructs'; } catch (e) { return e.name + ': ' + e.message; }
      }),
      connection: c ? { type: orNull(c.type), effectiveType: orNull(c.effectiveType), downlinkMax: orNull(c.downlinkMax),
        rtt: orNull(c.rtt), saveData: orNull(c.saveData) } : null,
      chromeKeys: safe(function () { return window.chrome ? Object.keys(window.chrome) : null; }),
      permissions: await permissionStates(),
    };
  }

  const MEDIA = {
    'pointer': ['none', 'coarse', 'fine'], 'hover': ['none', 'hover'],
    'any-pointer': ['none', 'coarse', 'fine'], 'any-hover': ['none', 'hover'],
    'prefers-color-scheme': ['light', 'dark'], 'prefers-reduced-motion': ['no-preference', 'reduce'],
    'color-gamut': ['srgb', 'p3', 'rec2020'], 'dynamic-range': ['standard', 'high'],
    'display-mode': ['browser', 'standalone', 'minimal-ui', 'fullscreen'], 'forced-colors': ['none', 'active'],
    'prefers-contrast': ['no-preference', 'more', 'less', 'custom'],
  };

  // media 每个媒体特性匹配上的取值
  function media() {
    const out = {};
    for (const f of Object.keys(MEDIA)) {
      out[f] = safe(function () { return MEDIA[f].filter(function (v) { return matchMedia('(' + f + ': ' + v + ')').matches; }); });
    }
    return out;
  }

  const GL_PARAMS = ['MAX_TEXTURE_SIZE', 'MAX_CUBE_MAP_TEXTURE_SIZE', 'MAX_RENDERBUFFER_SIZE', 'MAX_VIEWPORT_DIMS',
    'MAX_VERTEX_ATTRIBS', 'MAX_VERTEX_UNIFORM_VECTORS', 'MAX_FRAGMENT_UNIFORM_VECTORS', 'MAX_VARYING_VECTORS',
    'MAX_TEXTURE_IMAGE_UNITS', 'MAX_VERTEX_TEXTURE_IMAGE_UNITS', 'MAX_COMBINED_TEXTURE_IMAGE_UNITS',
    'ALIASED_POINT_SIZE_RANGE', 'ALIASED_LINE_WIDTH_RANGE', 'RED_BITS', 'GREEN_BITS', 'BLUE_BITS', 'ALPHA_BITS',
    'DEPTH_BITS', 'STENCIL_BITS', 'SUBPIXEL_BITS', 'SAMPLES', 'SAMPLE_BUFFERS'];
  const GL2_PARAMS = ['MAX_3D_TEXTURE_SIZE', 'MAX_ARRAY_TEXTURE_LAYERS', 'MAX_DRAW_BUFFERS', 'MAX_COLOR_ATTACHMENTS',
    'MAX_SAMPLES', 'MAX_UNIFORM_BLOCK_SIZE', 'MAX_UNIFORM_BUFFER_BINDINGS', 'MAX_COMBINED_UNIFORM_BLOCKS',
    'MAX_VERTEX_UNIFORM_COMPONENTS', 'MAX_FRAGMENT_UNIFORM_COMPONENTS', 'MAX_VARYING_COMPONENTS',
    'MAX_TRANSFORM_FEEDBACK_SEPARATE_COMPONENTS', 'MAX_ELEMENT_INDEX', 'MAX_SERVER_WAIT_TIMEOUT', 'MAX_TEXTURE_LOD_BIAS',
    'MAX_VERTEX_OUTPUT_COMPONENTS', 'MAX_FRAGMENT_INPUT_COMPONENTS', 'MAX_PROGRAM_TEXEL_OFFSET', 'MIN_PROGRAM_TEXEL_OFFSET'];

  // plain 类型化数组转普通数组，便于 JSON
  function plain(v) { return v && typeof v === 'object' && 'length' in v ? Array.from(v) : v; }

  // webgl 一个上下文的标识、能力参数、着色器精度与扩展；拿不到上下文时为 null
  function webgl(kind) {
    const gl = document.createElement('canvas').getContext(kind);
    if (!gl) return null;
    const dbg = gl.getExtension('WEBGL_debug_renderer_info');
    const out = {
      vendor: gl.getParameter(gl.VENDOR), renderer: gl.getParameter(gl.RENDERER),
      version: gl.getParameter(gl.VERSION), shadingLanguageVersion: gl.getParameter(gl.SHADING_LANGUAGE_VERSION),
      unmaskedVendor: dbg ? gl.getParameter(dbg.UNMASKED_VENDOR_WEBGL) : null,
      unmaskedRenderer: dbg ? gl.getParameter(dbg.UNMASKED_RENDERER_WEBGL) : null,
      params: {}, precision: {}, extensions: gl.getSupportedExtensions(), contextAttributes: gl.getContextAttributes(),
    };
    for (const p of kind === 'webgl2' ? GL_PARAMS.concat(GL2_PARAMS) : GL_PARAMS) {
      out.params[p] = safe(function () { return plain(gl.getParameter(gl[p])); });
    }
    for (const shader of ['VERTEX_SHADER', 'FRAGMENT_SHADER']) {
      for (const t of ['LOW_FLOAT', 'MEDIUM_FLOAT', 'HIGH_FLOAT', 'LOW_INT', 'MEDIUM_INT', 'HIGH_INT']) {
        out.precision[shader + '.' + t] = safe(function () {
          const f = gl.getShaderPrecisionFormat(gl[shader], gl[t]);
          return [f.rangeMin, f.rangeMax, f.precision];
        });
      }
    }
    return out;
  }

  const FONTS = [
    // Windows
    'Arial', 'Arial Black', 'Calibri', 'Cambria', 'Candara', 'Comic Sans MS', 'Consolas', 'Constantia', 'Corbel',
    'Courier New', 'Georgia', 'Impact', 'Lucida Console', 'Lucida Sans Unicode', 'Microsoft YaHei', 'Microsoft JhengHei',
    'MS Gothic', 'Palatino Linotype', 'Segoe UI', 'Segoe UI Emoji', 'SimSun', 'SimHei', 'KaiTi', 'FangSong', 'Tahoma',
    'Times New Roman', 'Trebuchet MS', 'Verdana', 'Webdings', 'Wingdings',
    // macOS / iOS
    'Helvetica', 'Helvetica Neue', 'PingFang SC', 'PingFang TC', 'Heiti SC', 'STHeiti', 'Hiragino Sans GB',
    'Apple Color Emoji', 'Apple SD Gothic Neo', 'Avenir', 'Avenir Next', 'Menlo', 'Monaco', 'Geneva', 'Gill Sans',
    'Futura', 'Optima', 'Didot', 'American Typewriter', 'Marker Felt', 'Noteworthy', 'Songti SC', 'Kaiti SC', 'Courier',
    'Thonburi', 'Arial Hebrew', 'Damascus', 'Snell Roundhand', 'Chalkboard SE', 'Zapfino',
    // 安卓与各厂商
    'Roboto', 'Roboto Condensed', 'Roboto Mono', 'Noto Sans', 'Noto Serif', 'Noto Sans CJK SC', 'Noto Sans SC',
    'Noto Serif CJK SC', 'Noto Color Emoji', 'Droid Sans', 'Droid Sans Mono', 'Droid Serif', 'Source Sans Pro',
    'Cutive Mono', 'Coming Soon', 'Dancing Script', 'Carrois Gothic SC', 'MiSans', 'MiSans VF', 'Mi Lan Pro VF',
    'HarmonyOS Sans', 'HarmonyOS Sans SC', 'OPPOSans', 'vivo Sans', 'SamsungOne', 'Google Sans', 'Product Sans',
    // 常见中文
    'Source Han Sans SC', 'Source Han Serif SC', 'WenQuanYi Micro Hei', 'WenQuanYi Zen Hei', 'Noto Sans CJK TC',
    'DengXian', 'YouYuan', 'LiSu', 'STKaiti', 'STSong',
  ];

  // fonts 与三种基础字体比较文字宽度判断字体是否存在
  function fonts() {
    const ctx = document.createElement('canvas').getContext('2d');
    const text = 'mmmmmmmmmmlli1WwQ@中文字体';
    const width = function (f) { ctx.font = '72px ' + f; return ctx.measureText(text).width; };
    const bases = ['monospace', 'sans-serif', 'serif'];
    const baseWidths = bases.map(width);
    const out = {};
    for (const f of FONTS) {
      out[f] = bases.some(function (b, i) { return width('"' + f + '", ' + b) !== baseWidths[i]; });
    }
    return out;
  }

  // errorOf 执行一段必然报错的代码，返回错误类型与消息
  function errorOf(fn) {
    try { fn(); return null; } catch (e) { return { name: e.name, message: e.message }; }
  }

  // engine JS 引擎特征：报错文本、调用栈格式（地址替换为 <url>）、原生函数文本、数学精度、Intl 设置
  function engine() {
    return {
      nullProperty: errorOf(function () { const o = null; return o.x; }),
      callUndefined: errorOf(function () { const f = undefined; f(); }),
      jsonParse: errorOf(function () { JSON.parse('{bad'); }),
      stackSample: safe(function () {
        try { null.f(); } catch (e) {
          return String(e.stack || '').replace(/(https?|file|blob):\/\/[^\s)]+/g, '<url>').split('\n').slice(0, 4).join('\n');
        }
        return null;
      }),
      nativeToString: Function.prototype.toString.call(Math.max),
      math: { tanNeg1e300: Math.tan(-1e300), sinh1: Math.sinh(1), expm1: Math.expm1(1), cbrt100: Math.cbrt(100),
        log1p10: Math.log1p(10), acosh1e300: Math.acosh(1e300), atan2: Math.atan2(0.5, 1.5), powPi: Math.pow(Math.PI, -100) },
      intl: safe(function () {
        const d = Intl.DateTimeFormat().resolvedOptions(), n = Intl.NumberFormat().resolvedOptions();
        return { locale: d.locale, timeZone: d.timeZone, calendar: d.calendar, numberingSystem: d.numberingSystem, numberLocale: n.locale };
      }),
    };
  }

  const MIME = ['video/mp4; codecs="avc1.42E01E"', 'video/mp4; codecs="avc1.640028"', 'video/mp4; codecs="hev1.1.6.L93.B0"',
    'video/mp4; codecs="hvc1.1.6.L93.B0"', 'video/mp4; codecs="av01.0.05M.08"', 'video/webm; codecs="vp8"',
    'video/webm; codecs="vp9"', 'video/webm; codecs="av01.0.05M.08"', 'video/ogg; codecs="theora"', 'video/quicktime',
    'application/vnd.apple.mpegurl', 'application/x-mpegURL', 'audio/mp4; codecs="mp4a.40.2"', 'audio/mpeg',
    'audio/webm; codecs="opus"', 'audio/ogg; codecs="vorbis"', 'audio/ogg; codecs="opus"', 'audio/wav', 'audio/flac',
    'audio/aac', 'audio/x-m4a'];

  // voices 语音数量与前 5 个（列表为空时等 voiceschanged 最多 1 秒）；没有语音接口时为 null
  async function voices() {
    if (!('speechSynthesis' in window)) return null;
    let list = speechSynthesis.getVoices();
    if (!list.length) {
      list = await withTimeout(new Promise(function (r) {
        speechSynthesis.addEventListener('voiceschanged', function () { r(speechSynthesis.getVoices()); }, { once: true });
      }), 1000, []);
    }
    return { count: list.length, first: list.slice(0, 5).map(function (v) { return { name: v.name, lang: v.lang }; }) };
  }

  // avmedia 语音、音视频解码支持、媒体设备（只记各类数量）
  async function avmedia() {
    const video = document.createElement('video');
    const codecs = {};
    for (const m of MIME) {
      codecs[m] = {
        canPlayType: safe(function () { return video.canPlayType(m); }),
        mediaSource: safe(function () { return window.MediaSource ? MediaSource.isTypeSupported(m) : null; }),
      };
    }
    return {
      voices: await safeAsync(voices),
      codecs: codecs,
      devices: await safeAsync(async function () {
        if (!navigator.mediaDevices || !navigator.mediaDevices.enumerateDevices) return null;
        const counts = {};
        for (const d of await navigator.mediaDevices.enumerateDevices()) counts[d.kind] = (counts[d.kind] || 0) + 1;
        return counts;
      }),
    };
  }

  // collectBaseline 自动采集全部组
  async function collectBaseline() {
    return {
      identity: await safeAsync(identity),
      screen: safe(screenInfo),
      features: await safeAsync(features),
      media: safe(media),
      webgl: safe(function () { return webgl('webgl'); }),
      webgl2: safe(function () { return webgl('webgl2'); }),
      fonts: safe(fonts),
      engine: safe(engine),
      avmedia: await safeAsync(avmedia),
    };
  }
  window.collectBaseline = collectBaseline;

  // ---------- 交互记录 ----------
  const MAX_EVENTS = 600;
  const EVENT_TYPES = ['pointerover', 'pointerenter', 'pointerdown', 'pointermove', 'pointerup', 'pointercancel',
    'pointerout', 'pointerleave', 'touchstart', 'touchmove', 'touchend', 'touchcancel', 'mouseover', 'mousemove',
    'mousedown', 'mouseup', 'click', 'keydown', 'keypress', 'keyup', 'beforeinput', 'input', 'compositionstart',
    'compositionupdate', 'compositionend', 'scroll'];
  const FIELDS = ['pointerType', 'pointerId', 'isPrimary', 'width', 'height', 'pressure', 'tangentialPressure', 'tiltX',
    'tiltY', 'twist', 'button', 'buttons', 'detail', 'clientX', 'clientY', 'screenX', 'screenY', 'key', 'code', 'keyCode',
    'which', 'location', 'repeat', 'isComposing', 'inputType', 'data'];
  const interaction = { events: [], truncated: false, steps: { tap: false, type: false, swipe: false } };
  let t0 = null;

  // stepOf 事件发生在哪一步的区域
  function stepOf(el) {
    if (!el || !el.closest) return 'other';
    if (el.closest('#step-tap')) return 'tap';
    if (el.closest('#step-type')) return 'type';
    if (el.closest('#step-swipe')) return 'swipe';
    return 'other';
  }

  // touchList 触点列表转普通对象
  function touchList(list) {
    return Array.from(list).map(function (t) {
      return { identifier: t.identifier, clientX: t.clientX, clientY: t.clientY, radiusX: t.radiusX, radiusY: t.radiusY,
        rotationAngle: t.rotationAngle, force: t.force };
    });
  }

  // record 记录一个事件的属性；超过上限后只标记 truncated
  function record(e) {
    try {
      if (interaction.events.length >= MAX_EVENTS) { interaction.truncated = true; return; }
      if (t0 === null) t0 = e.timeStamp;
      const el = e.target && e.target.nodeType === 1 ? e.target : null;
      const ev = { step: stepOf(el), type: e.type, isTrusted: e.isTrusted, t: Math.round((e.timeStamp - t0) * 10) / 10,
        target: el ? (el.id || el.tagName.toLowerCase()) : 'document' };
      for (const k of FIELDS) if (e[k] !== undefined) ev[k] = e[k];
      if (e.touches) { ev.touches = touchList(e.touches); ev.changedTouches = touchList(e.changedTouches); }
      if (e.type === 'scroll' && el) ev.scrollTop = el.scrollTop;
      interaction.events.push(ev);
    } catch (err) {
      // 记录失败不影响页面
    }
  }
  for (const type of EVENT_TYPES) document.addEventListener(type, record, { capture: true, passive: true });

  // status 显示当前进度
  function status(msg) { document.getElementById('status').textContent = msg; }

  // markDone 标记一步完成；三步都完成后上传
  function markDone(step) {
    if (interaction.steps[step]) return;
    interaction.steps[step] = true;
    document.getElementById('step-' + step).classList.add('done');
    if (interaction.steps.tap && interaction.steps.type && interaction.steps.swipe) finish();
  }

  document.getElementById('tap-btn').addEventListener('click', function () { markDone('tap'); });
  document.getElementById('type-input').addEventListener('input', function (e) {
    // 等最后一个键的 keyup 也记录下来
    if (e.target.value.toLowerCase().indexOf('ab1') !== -1) setTimeout(function () { markDone('type'); }, 800);
  });
  const zone = document.getElementById('swipe-zone');
  let scrolled = false;
  zone.addEventListener('scroll', function () { scrolled = true; }, { passive: true });
  zone.addEventListener('touchend', function () {
    // 惯性滚动还会持续一会儿，等 1 秒再结束
    if (scrolled) setTimeout(function () { markDone('swipe'); }, 1000);
  }, { passive: true });

  window.__baselineState = 'collecting';
  const autoPromise = collectBaseline().catch(function (e) { return { error: String(e) }; }).then(function (r) {
    window.__baselineAuto = r;
    window.__baselineState = 'ready';
    status('浏览器信息已采集。请完成下面三步。');
    return r;
  });

  let finished = false;

  // finish 结束交互并上传；失败时显示复制备用
  async function finish() {
    if (finished) return;
    finished = true;
    status('正在上传…');
    const auto = await autoPromise;
    const result = { schema: 1, collectedAt: new Date().toISOString(), auto: auto, interaction: interaction };
    const text = JSON.stringify(result);
    window.__baselineResult = result;
    try {
      const resp = await fetch('result', { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: text });
      if (!resp.ok) throw new Error('HTTP ' + resp.status);
      const body = await resp.json();
      window.__baselineState = 'uploaded';
      status('已上传：' + body.file + '。可以关闭页面了。');
    } catch (e) {
      window.__baselineState = 'upload-failed';
      status('上传失败（' + e.message + '）');
      document.getElementById('result-text').value = text;
      document.getElementById('fallback').classList.remove('hidden');
    }
  }
  window.finishBaseline = finish;

  document.getElementById('skip-btn').addEventListener('click', finish);
  document.getElementById('copy-btn').addEventListener('click', async function () {
    const ta = document.getElementById('result-text');
    try { await navigator.clipboard.writeText(ta.value); } catch (e) { ta.select(); document.execCommand('copy'); }
    status('已复制');
  });
})();
</script>
</body>
</html>
```

- [ ] **Step 4: 运行测试，确认通过**

Run: `go test -race -count=1 -run TestBaselineCollector ./pkg/browser/`
Expected: PASS。若 `TestBaselineCollectorRuns` 报某项 `probes failed`，说明该项检测在桌面 Chrome 上抛错：修检测代码，不要放宽测试。

- [ ] **Step 5: 提交**

```bash
git add test/baseline/collector.html pkg/browser/baseline_collector_test.go
git commit -m "feat(baseline): collector page for real-phone browser baselines

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
```

---

### Task 2: 本地采集服务

**Files:**
- Create: `cmd/baseline_server/main.go`
- Test: `cmd/baseline_server/main_test.go`

**Interfaces:**
- Consumes: Task 1 的上传约定。采集页 `POST result`，请求体为完整结果 JSON；页面读应答的 `file` 字段。
- Produces（Task 3 依赖）：命令行参数 `-port`（默认 8765）、`-out`（默认 `test/baseline`）、`-page`（默认 `test/baseline/collector.html`）；只监听 `127.0.0.1`；启动后打印 `采集服务已启动：http://127.0.0.1:<端口>`。

- [ ] **Step 1: 写失败的测试**

创建 `cmd/baseline_server/main_test.go`：

```go
package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

const (
	androidChromeUA = "Mozilla/5.0 (Linux; Android 10; K) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/141.0.0.0 Mobile Safari/537.36"
	iphoneSafariUA  = "Mozilla/5.0 (iPhone; CPU iPhone OS 18_6 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/18.6 Mobile/15E148 Safari/604.1"
	miuiBrowserUA   = "Mozilla/5.0 (Linux; U; Android 14; zh-cn; 23127PN0CC Build/UKQ1.230804.001) AppleWebKit/537.36 (KHTML, like Gecko) Version/4.0 Chrome/122.0.6261.119 Mobile Safari/537.36 XiaoMi/MiuiBrowser/18.6.70515"
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

// sampleResult 构造一份最小的结果；model 为空时 userAgentData 为 null（Safari 的情况）
func sampleResult(ua, model string) string {
	uad := "null"
	if model != "" {
		uad = fmt.Sprintf(`{"model": %q}`, model)
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

// TestSaveResultNames 文件名由 UA 与机型推导；误用小米浏览器时浏览器记为 other（Review Focus 4）
func TestSaveResultNames(t *testing.T) {
	cases := []struct{ name, ua, model, want string }{
		{"android chrome", androidChromeUA, "23127PN0CC", "android-chrome-23127pn0cc.json"},
		{"iphone safari", iphoneSafariUA, "", "ios-safari-iphone.json"},
		{"xiaomi browser", miuiBrowserUA, "", "android-other-unknown.json"},
		{"model characters", androidChromeUA, "Pixel 8 Pro/中文", "android-chrome-pixel-8-pro.json"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			s, out := newTestServer(t)
			body := sampleResult(tc.ua, tc.model)
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
	body := sampleResult(androidChromeUA, "23127PN0CC")
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
```

- [ ] **Step 2: 运行测试，确认失败**

Run: `go test -race -count=1 ./cmd/baseline_server/`
Expected: FAIL，编译错误 `undefined: server` / `undefined: newServer`

- [ ] **Step 3: 写服务**

创建 `cmd/baseline_server/main.go`：

```go
// baseline_server 真机基准采集服务：提供采集页，接收手机上传的结果并存为 JSON 文件。
// 只监听本机地址，由 scripts/collect-baseline.sh 经 cloudflared 临时暴露为 https 地址
package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"regexp"
	"strings"
	"syscall"
	"time"
)

// maxResultBytes 单份结果的大小上限
const maxResultBytes = 1 << 20

func main() {
	if err := run(context.Background(), os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

// run 解析参数并运行服务，收到中断信号后优雅退出
func run(ctx context.Context, args []string) error {
	fset := flag.NewFlagSet("baseline_server", flag.ContinueOnError)
	port := fset.Int("port", 8765, "监听端口（只监听 127.0.0.1）")
	out := fset.String("out", "test/baseline", "结果保存目录")
	page := fset.String("page", "test/baseline/collector.html", "采集页路径")
	if err := fset.Parse(args); err != nil {
		return err
	}
	ctx, stop := signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM)
	defer stop()

	srv := &http.Server{
		Addr:              fmt.Sprintf("127.0.0.1:%d", *port),
		Handler:           newServer(*page, *out).routes(),
		ReadHeaderTimeout: 10 * time.Second,
	}
	errc := make(chan error, 1)
	go func() { errc <- srv.ListenAndServe() }()
	log.Printf("采集服务已启动：http://%s ，结果保存到 %s", srv.Addr, *out)
	select {
	case err := <-errc:
		return fmt.Errorf("listen: %w", err)
	case <-ctx.Done():
	}
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return srv.Shutdown(shutdownCtx)
}

// server 采集服务的依赖
type server struct {
	pagePath string // 采集页路径，每次请求重新读取，改页面不用重启
	outDir   string // 结果保存目录
}

// newServer 创建采集服务
func newServer(pagePath, outDir string) *server {
	return &server{pagePath: pagePath, outDir: outDir}
}

// routes 全部路由
func (s *server) routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /{$}", s.handlePage())
	mux.HandleFunc("POST /result", s.handleResult())
	return mux
}

// handlePage 返回采集页
func (s *server) handlePage() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		page, err := os.ReadFile(s.pagePath)
		if err != nil {
			http.Error(w, "read collector page: "+err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Header().Set("Cache-Control", "no-store")
		w.Write(page)
	}
}

// baselineSummary 结果中用于命名文件的字段
type baselineSummary struct {
	Auto *struct {
		Identity struct {
			UserAgent     string `json:"userAgent"`
			UserAgentData *struct {
				Model string `json:"model"`
			} `json:"userAgentData"`
		} `json:"identity"`
	} `json:"auto"`
}

// handleResult 接收一份结果：校验后格式化保存，返回文件名；不记录请求方地址
func (s *server) handleResult() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(http.MaxBytesReader(w, r.Body, maxResultBytes))
		if err != nil {
			var tooBig *http.MaxBytesError
			if errors.As(err, &tooBig) {
				http.Error(w, "result larger than 1 MiB", http.StatusRequestEntityTooLarge)
				return
			}
			http.Error(w, "read body: "+err.Error(), http.StatusBadRequest)
			return
		}
		var sum baselineSummary
		if err := json.Unmarshal(body, &sum); err != nil || sum.Auto == nil {
			http.Error(w, "body is not a baseline result (JSON with an auto field)", http.StatusBadRequest)
			return
		}
		var pretty bytes.Buffer
		json.Indent(&pretty, body, "", "  ") // 上面已确认是合法 JSON，不会失败
		pretty.WriteByte('\n')

		id := sum.Auto.Identity
		model := ""
		if id.UserAgentData != nil {
			model = id.UserAgentData.Model
		}
		name, err := writeUnique(s.outDir, baselineName(id.UserAgent, model), pretty.Bytes())
		if err != nil {
			http.Error(w, "save result: "+err.Error(), http.StatusInternalServerError)
			return
		}
		log.Printf("已保存 %s（%s）", name, id.UserAgent)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"file": name})
	}
}

// otherChromium UA 里也带 Chrome/ 的其他浏览器：采集必须用正版 Chrome，这些记为 other，检查时能发现用错
var otherChromium = []string{"Edg", "OPR/", "MiuiBrowser", "XiaoMi/", "SamsungBrowser", "HeyTapBrowser", "UCBrowser",
	"MQQBrowser", "HuaweiBrowser", "VivoBrowser", "Quark", "baidu"}

// baselineName 由 UA 与机型推导结果文件的基本名：<系统>-<浏览器>-<机型>
func baselineName(ua, model string) string {
	osName := "other"
	switch {
	case strings.Contains(ua, "iPhone"), strings.Contains(ua, "iPad"):
		osName = "ios"
	case strings.Contains(ua, "Android"):
		osName = "android"
	}
	browser := "other"
	switch {
	case strings.Contains(ua, "CriOS/"):
		browser = "chrome"
	case strings.Contains(ua, "Chrome/"):
		browser = "chrome"
		for _, s := range otherChromium {
			if strings.Contains(ua, s) {
				browser = "other"
			}
		}
	case strings.Contains(ua, "Version/") && strings.Contains(ua, "Safari/"):
		browser = "safari"
	}
	if model == "" && osName == "ios" {
		model = "iphone"
	}
	return osName + "-" + browser + "-" + sanitizeName(model)
}

var nonNameChars = regexp.MustCompile(`[^a-z0-9]+`)

// sanitizeName 文件名片段只保留小写字母、数字和连字符；为空时返回 unknown
func sanitizeName(s string) string {
	s = strings.Trim(nonNameChars.ReplaceAllString(strings.ToLower(s), "-"), "-")
	if s == "" {
		return "unknown"
	}
	return s
}

// writeUnique 把数据写到 dir/<base>.json；已存在时依次尝试 <base>-2.json、<base>-3.json……，返回实际文件名
func writeUnique(dir, base string, data []byte) (string, error) {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", fmt.Errorf("create %s: %w", dir, err)
	}
	for i := 1; ; i++ {
		name := base + ".json"
		if i > 1 {
			name = fmt.Sprintf("%s-%d.json", base, i)
		}
		f, err := os.OpenFile(filepath.Join(dir, name), os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
		if errors.Is(err, fs.ErrExist) {
			continue
		}
		if err != nil {
			return "", fmt.Errorf("create %s: %w", name, err)
		}
		_, werr := f.Write(data)
		if cerr := f.Close(); werr == nil {
			werr = cerr
		}
		if werr != nil {
			return "", fmt.Errorf("write %s: %w", name, werr)
		}
		return name, nil
	}
}
```

- [ ] **Step 4: 运行测试，确认通过**

Run: `go test -race -count=1 ./cmd/baseline_server/ && go vet ./cmd/baseline_server/`
Expected: PASS，vet 无输出

- [ ] **Step 5: 提交**

```bash
git add cmd/baseline_server/
git commit -m "feat(baseline): local server that serves the collector and saves results

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
```

---

### Task 3: 启动脚本、文档与隧道冒烟

**Files:**
- Create: `scripts/collect-baseline.sh`
- Modify: `CLAUDE.md`（"Auxiliary `cmd/` tools" 那一段）

**Interfaces:**
- Consumes: Task 2 的 `-port`、`-out`、`-page` 参数及其默认值；采集页在 `test/baseline/collector.html`。
- Produces（Task 4 依赖）：`scripts/collect-baseline.sh` 打印 `在手机上打开： https://<随机>.trycloudflare.com`，按 Ctrl-C 时同时停止服务和隧道。

- [ ] **Step 1: 写失败的检查**

Run: `PATH=/usr/bin:/bin bash scripts/collect-baseline.sh; echo "exit=$?"`
Expected: `bash: scripts/collect-baseline.sh: No such file or directory`，`exit=127`

- [ ] **Step 2: 写脚本**

创建 `scripts/collect-baseline.sh`，然后执行 `chmod +x scripts/collect-baseline.sh`：

```bash
#!/usr/bin/env bash
# 真机基准采集：启动本地采集服务，用 cloudflared 临时暴露为 https 地址；Ctrl-C 同时停止两者，地址随之失效
set -euo pipefail
cd "$(dirname "$0")/.."
PORT="${PORT:-8765}"

if ! command -v cloudflared >/dev/null 2>&1; then
  echo "缺少 cloudflared，请先安装：brew install cloudflared" >&2
  exit 1
fi

BIN="$(mktemp -d)/baseline_server"
go build -o "$BIN" ./cmd/baseline_server
LOG="$(mktemp)"
"$BIN" -port "$PORT" &
SERVER=$!
cloudflared tunnel --no-autoupdate --url "http://127.0.0.1:$PORT" >"$LOG" 2>&1 &
TUNNEL=$!
trap 'kill "$SERVER" "$TUNNEL" 2>/dev/null; rm -f "$LOG" "$BIN"' EXIT INT TERM

URL=""
for _ in $(seq 1 60); do
  URL="$(grep -oE 'https://[a-z0-9-]+\.trycloudflare\.com' "$LOG" | head -1 || true)"
  [ -n "$URL" ] && break
  sleep 1
done
if [ -z "$URL" ]; then
  echo "cloudflared 60 秒内没有给出地址，输出如下：" >&2
  cat "$LOG" >&2
  exit 1
fi

echo
echo "================================================================"
echo "  在手机上打开： $URL"
echo "  结果保存到：   test/baseline/"
echo "  采集完成后按 Ctrl-C 结束"
echo "================================================================"
wait "$SERVER"
```

- [ ] **Step 3: 运行检查，确认通过**

Run: `bash -n scripts/collect-baseline.sh && PATH=/usr/bin:/bin bash scripts/collect-baseline.sh; echo "exit=$?"`
Expected: 输出 `缺少 cloudflared，请先安装：brew install cloudflared`，`exit=1`

- [ ] **Step 4: 安装 cloudflared（用户已选方案 A）**

Run: `brew install cloudflared && cloudflared --version`
Expected: 打印 `cloudflared version …`

- [ ] **Step 5: 隧道冒烟（结束后删除冒烟文件）**

Run：

```bash
LOG=$(mktemp); scripts/collect-baseline.sh >"$LOG" 2>&1 & S=$!
for i in $(seq 1 60); do URL=$(grep -oE 'https://[a-z0-9-]+\.trycloudflare\.com' "$LOG" | head -1); [ -n "$URL" ] && break; sleep 1; done
sleep 5  # 新隧道的 DNS 需要几秒才生效
curl -s "$URL/" | grep -c 'collectBaseline'
curl -s -X POST -H 'Content-Type: application/json' -d '{"schema":1,"auto":{"identity":{"userAgent":"smoke-test"}}}' "$URL/result"
ls test/baseline/
kill $S; wait $S 2>/dev/null; rm -f test/baseline/other-other-unknown.json "$LOG"
```

Expected：
- 第一个 curl 输出大于 0 的计数；
- 第二个 curl 输出 `{"file":"other-other-unknown.json"}`；
- `ls` 中有 `collector.html` 和 `other-other-unknown.json`；
- 结束后 `test/baseline/` 只剩 `collector.html`。

若 curl 报 DNS 错误，多等几秒再试。

- [ ] **Step 6: 更新 CLAUDE.md**

在 `CLAUDE.md` 的 "**Auxiliary `cmd/` tools**" 一段，把这句：

```
`fingerprint_stats` (config-pool combinatorics report).
```

改成：

```
`fingerprint_stats` (config-pool combinatorics report), `baseline_server` (real-phone browser baselines for the mobile identity work: run `scripts/collect-baseline.sh`, which needs `cloudflared` and prints an https URL to open on the phone; results land in `test/baseline/`, the collector page is `test/baseline/collector.html` and is also exercised by `pkg/browser` tests).
```

- [ ] **Step 7: 提交**

```bash
git add scripts/collect-baseline.sh CLAUDE.md
git commit -m "feat(baseline): one-command collection over a cloudflared quick tunnel

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
```

---

### Task 4: 真机采集（需要用户在场）

**Files:**
- Create: `test/baseline/android-chrome-<机型>.json`、`test/baseline/ios-safari-iphone.json`（由服务生成）

**Interfaces:**
- Consumes: Task 3 的脚本；Task 1 的结果格式。
- Produces（M1–M3 依赖）：两份真机基准文件，路径在本任务的提交里。

- [ ] **Step 1: 启动采集并把地址给用户**

后台运行 `scripts/collect-baseline.sh`，等它打印出 https 地址后发给用户。请用户：
- 在**小米手机的 Google Chrome**（不是小米自带浏览器）里打开地址，按页面提示完成三步；
- 在 **iPhone 的 Safari** 里打开同一地址，同样完成三步。

- [ ] **Step 2: 确认文件名**

Run: `ls test/baseline/`
Expected: 有 `android-chrome-<机型>.json` 和 `ios-safari-iphone.json`。

文件名里浏览器是 `other`，说明用错了浏览器：删除该文件，请用户换浏览器重做。

- [ ] **Step 3: 隐私与完整性检查**

Run：

```bash
cd test/baseline
grep -l 'trycloudflare\|http://\|https://' *.json || echo "no urls"
for f in android-chrome-*.json ios-safari-iphone.json; do
  python3 - "$f" <<'EOF'
import json, sys
r = json.load(open(sys.argv[1]))
a, i = r["auto"], r["interaction"]
print(sys.argv[1])
print("  ua:", a["identity"]["userAgent"])
print("  maxTouchPoints:", a["identity"]["maxTouchPoints"], " uad mobile:", (a["identity"]["userAgentData"] or {}).get("mobile"))
print("  steps:", i["steps"], " events:", len(i["events"]), " truncated:", i["truncated"])
print("  stack:", a["engine"]["stackSample"].splitlines()[:2])
EOF
done
```

Expected：
- 输出 `no urls`；
- 两份文件的 `maxTouchPoints` 都大于 0，三步 `steps` 都是 true；
- 安卓的 `uad mobile` 为 True，iOS 的为 None；
- 调用栈里只有 `<url>`。

再人工浏览一遍两份文件，确认没有个人信息（语音名称、字体、机型代号不算）。

- [ ] **Step 4: 停止采集，提交**

在采集脚本所在的进程上按 Ctrl-C（或 kill），然后：

```bash
git add test/baseline/android-chrome-*.json test/baseline/ios-safari-iphone.json
git commit -m "test(baseline): real-phone baselines from Xiaomi Chrome and iPhone Safari

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
```

- [ ] **Step 5: 全量测试**

Run: `go test -race -count=1 ./cmd/baseline_server/ ./pkg/... ./internal/... && scripts/test-linux-chrome.sh`
Expected: 全部 PASS（Linux 容器只跑 `./pkg/... ./internal/...`，其中包含 Task 1 的采集页测试）
