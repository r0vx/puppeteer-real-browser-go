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

// evalValue 在页面执行 JS（支持 Promise）并返回值
func evalValue(t *testing.T, c *CustomCDPClient, js string) any {
	t.Helper()
	raw, err := c.sendCommand("Runtime.evaluate", map[string]any{"expression": js, "awaitPromise": true, "returnByValue": true})
	if err != nil {
		t.Fatalf("evaluate %s: %v", js, err)
	}
	var res struct {
		Result           struct{ Value any } `json:"result"`
		ExceptionDetails any                 `json:"exceptionDetails"`
	}
	if err := json.Unmarshal(raw, &res); err != nil {
		t.Fatal(err)
	}
	if res.ExceptionDetails != nil {
		t.Fatalf("evaluate %s threw: %v", js, res.ExceptionDetails)
	}
	return res.Result.Value
}

// loadPage 导航并等待文档加载完成
func loadPage(t *testing.T, c *CustomCDPClient, url string) {
	t.Helper()
	if _, err := c.sendCommand("Page.navigate", map[string]any{"url": url}); err != nil {
		t.Fatal(err)
	}
	for deadline := time.Now().Add(5 * time.Second); time.Now().Before(deadline); time.Sleep(50 * time.Millisecond) {
		raw, err := c.sendCommand("Runtime.evaluate", map[string]any{"expression": "location.href + ' ' + document.readyState", "returnByValue": true})
		if err == nil && string(raw) != "" && json.Valid(raw) {
			var res struct{ Result struct{ Value string } }
			json.Unmarshal(raw, &res)
			if res.Result.Value == url+" complete" {
				return
			}
		}
	}
	t.Fatalf("page %s did not load", url)
}

// TestIdentityScript 注入脚本后：被改写函数显示为原生、无新增全局与 navigator 自有属性、
// WebGL / 屏幕取档案值、canvas 噪声确定且只作用于不透明像素、未受信事件保持原生、Worker 版本补齐 navigator
func TestIdentityScript(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `<title>js</title><canvas id=c width=200 height=50></canvas>`)
	}))
	defer srv.Close()
	chrome, err := NewChromeLauncher().Launch(t.Context(), &ConnectOptions{Headless: true})
	if err != nil {
		t.Fatal(err)
	}
	defer chrome.Kill()
	c, err := NewCustomCDPClient(fmt.Sprintf("http://localhost:%d", chrome.Port))
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()
	if err := c.EnablePageDomain(); err != nil {
		t.Fatal(err)
	}

	const draw = `(() => { const c = document.createElement('canvas'); c.width = 200; c.height = 50; const x = c.getContext('2d');
	  x.font = '20px Arial'; x.fillStyle = '#f60'; x.fillRect(0, 0, 60, 20); x.fillStyle = '#069'; x.fillText('identity 测试', 5, 30); return c.toDataURL(); })()`
	const blank = `document.createElement('canvas').toDataURL()`

	page := srv.URL + "/"
	loadPage(t, c, page)
	baseDraw := evalValue(t, c, draw)
	baseBlank := evalValue(t, c, blank)
	baseGlobals := evalValue(t, c, `Object.getOwnPropertyNames(window).length`)

	gpu := windowsGPUs[0].Value
	const winUA = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/154.0.0.0 Safari/537.36"
	id := &Identity{
		UserAgent: winUA, Platform: "Win32", Languages: []string{"ja-JP", "ja"}, HardwareConcurrency: 6,
		Screen: geometryFor(OSWindows, screenSpec{Width: 1536, Height: 864, DPR: 1.25}, "15.0.0"),
		GPU:    &gpu, NoiseSeed: 12345,
	}
	if _, err := c.sendCommand("Page.addScriptToEvaluateOnNewDocument", map[string]any{"source": identityScript(id, false)}); err != nil {
		t.Fatal(err)
	}
	loadPage(t, c, page)

	cases := []struct {
		name, js string
		want     any
	}{
		{"getImageData native", `Function.prototype.toString.call(CanvasRenderingContext2D.prototype.getImageData)`, "function getImageData() { [native code] }"},
		{"toDataURL native", `HTMLCanvasElement.prototype.toDataURL.toString()`, "function toDataURL() { [native code] }"},
		{"toString itself native", `Function.prototype.toString.toString()`, "function toString() { [native code] }"},
		{"getter native", `Object.getOwnPropertyDescriptor(Screen.prototype, 'availHeight').get.toString()`, "function get availHeight() { [native code] }"},
		{"name and length kept", `[WebGLRenderingContext.prototype.getParameter.name, WebGLRenderingContext.prototype.getParameter.length, CanvasRenderingContext2D.prototype.getImageData.length].join()`, "getParameter,1,4"},
		{"no navigator own props", `Object.getOwnPropertyNames(navigator).length`, float64(0)},
		{"no new globals", `Object.getOwnPropertyNames(window).length`, baseGlobals},
		{"avail area", `[screen.availWidth, screen.availHeight, screen.availTop].join()`, "1536,816,0"},
		{"webgl profile", `(() => { const g = document.createElement('canvas').getContext('webgl'); g.getExtension('WEBGL_debug_renderer_info');
		  return [g.getParameter(37445), g.getParameter(37446), g.getParameter(g.MAX_TEXTURE_SIZE), g.getParameter(g.MAX_VIEWPORT_DIMS) instanceof Int32Array, g.getParameter(g.MAX_VARYING_VECTORS)].join('|'); })()`,
			gpu.Vendor + "|" + gpu.Renderer + "|16384|true|30"},
		{"canvas noise deterministic", `(() => { const a = ` + draw + `; const b = ` + draw + `; return a === b; })()`, true},
		{"blank canvas untouched", blank, baseBlank},
		{"untrusted event native", `new MouseEvent('click', {clientX: 10, screenX: 5}).screenX`, float64(5)},
		// 改写的 getter 和原生一样检查 this：在原型上调用要抛 Illegal invocation
		{"getter checks receiver", `(() => { try { Object.getOwnPropertyDescriptor(Screen.prototype, 'availWidth').get.call(Screen.prototype); return 'no throw'; } catch (e) { return e.constructor.name; } })()`, "TypeError"},
		// 同源 iframe 的 toString 看本页的替换函数、本页的 toString 看 iframe 的替换函数，都必须是原生文本
		{"cross-realm toString native", `(() => { const f = document.createElement('iframe'); document.body.appendChild(f); const w = f.contentWindow, ts = w.Function.prototype.toString;
		  return [ts.call(WebGLRenderingContext.prototype.getParameter), ts.call(Object.getOwnPropertyDescriptor(Screen.prototype, 'availHeight').get), ts.call(Function.prototype.toString),
		    Function.prototype.toString.call(w.HTMLCanvasElement.prototype.toDataURL), Function.prototype.toString.call(ts)].join('|'); })()`,
			"function getParameter() { [native code] }|function get availHeight() { [native code] }|function toString() { [native code] }|function toDataURL() { [native code] }|function toString() { [native code] }"},
		// 对 toString 传入任意非函数 this 时仍按原生抛 TypeError（共享伪装表的口令猜不到）
		{"toString on string throws", `(() => { try { Function.prototype.toString.call(''); return 'no throw'; } catch (e) { return e.constructor.name; } })()`, "TypeError"},
		// CreepJS 识别 Proxy 版 toString 的三条检测（failed at too much recursion / reflect set proto / object toString error）
		{"toString cyclic proto throws TypeError", `(() => { const f = Function.prototype.toString, p = Object.getPrototypeOf(f);
		  try { Object.setPrototypeOf(f, Object.create(f)).toString(); return 'no throw'; } catch (e) { return e.constructor.name; } finally { Object.setPrototypeOf(f, p); } })()`, "TypeError"},
		{"toString reflect cyclic proto refused", `(() => { const f = Function.prototype.toString, p = Object.getPrototypeOf(f);
		  try { return Reflect.setPrototypeOf(f, Object.create(f)); } finally { Object.setPrototypeOf(f, p); } })()`, false},
		{"toString error stack starts at native frame", `(() => { try { Object.create(Function.prototype.toString).toString(); return 'no throw'; } catch (e) { return /at Function\.toString /.test(e.stack.split('\n')[1]); } })()`, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := evalValue(t, c, tc.js); got != tc.want {
				t.Errorf("got %v, want %v", got, tc.want)
			}
		})
	}
	if got := evalValue(t, c, draw); got == baseDraw {
		t.Error("canvas output identical to the un-noised baseline")
	}

	// Worker 版本：在真实 Worker 里先执行脚本再读取 navigator
	ws, _ := json.Marshal(identityScript(id, true) + `;postMessage([navigator.userAgent, navigator.appVersion, navigator.platform, navigator.hardwareConcurrency, navigator.languages.join(),
	  Object.getOwnPropertyDescriptor(WorkerNavigator.prototype, 'platform').get.toString(),
	  (() => { try { WorkerNavigator.prototype.userAgent; return 'no throw'; } catch (e) { return e.constructor.name; } })()].join('|'))`)
	got := evalValue(t, c, `new Promise(r => { const w = new Worker(URL.createObjectURL(new Blob([`+string(ws)+`]))); w.onmessage = e => r(e.data); })`)
	if want := winUA + "|" + strings.TrimPrefix(winUA, "Mozilla/") + "|Win32|6|ja-JP,ja|function get platform() { [native code] }|TypeError"; got != want {
		t.Errorf("worker navigator = %v, want %v", got, want)
	}
}

// TestIdentityScriptWebShare Linux 版 Chrome 没有 Web Share：Windows / Mac 身份要补上 share / canShare，
// 外观与原生一致（原生 toString、方法属性、不可 new），行为按真实 Chrome（数据校验、无用户手势拒绝、有手势当作用户取消）；
// 宿主身份（Linux）不补
func TestIdentityScriptWebShare(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `<title>share</title>`)
	}))
	defer srv.Close()
	// 模拟 Linux：在身份脚本之前删掉原生的 share / canShare
	const removeShare = `delete Navigator.prototype.share; delete Navigator.prototype.canShare;`

	run := func(t *testing.T, platform string) *CustomCDPClient {
		chrome, err := NewChromeLauncher().Launch(t.Context(), &ConnectOptions{Headless: true})
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { chrome.Kill() })
		c, err := NewCustomCDPClient(fmt.Sprintf("http://localhost:%d", chrome.Port))
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { c.Close() })
		if err := c.EnablePageDomain(); err != nil {
			t.Fatal(err)
		}
		id := &Identity{UserAgent: "Mozilla/5.0", Platform: platform, Languages: []string{"zh-CN", "zh"}, HardwareConcurrency: 8}
		for _, src := range []string{removeShare, identityScript(id, false)} {
			if _, err := c.sendCommand("Page.addScriptToEvaluateOnNewDocument", map[string]any{"source": src}); err != nil {
				t.Fatal(err)
			}
		}
		loadPage(t, c, srv.URL+"/")
		return c
	}

	t.Run("windows identity on a host without Web Share", func(t *testing.T) {
		c := run(t, "Win32")
		cases := []struct {
			name, js string
			want     any
		}{
			{"present", `typeof navigator.share + ',' + typeof navigator.canShare + ',' + ('share' in navigator)`, "function,function,true"},
			{"native toString", `[Function.prototype.toString.call(navigator.share), Function.prototype.toString.call(navigator.canShare)].join('|')`,
				"function share() { [native code] }|function canShare() { [native code] }"},
			{"method descriptor", `(d => [d.writable, d.enumerable, d.configurable].join())(Object.getOwnPropertyDescriptor(Navigator.prototype, 'share'))`, "true,true,true"},
			{"function shape", `[navigator.share.name, navigator.share.length, 'prototype' in navigator.share, Object.getOwnPropertyNames(navigator.share).sort().join()].join('|')`, "share|0|false|length,name"},
			{"not constructible", `(() => { try { new navigator.share(); return 'no throw'; } catch (e) { return e.constructor.name; } })()`, "TypeError"},
			{"canShare validates data", `[navigator.canShare(), navigator.canShare({}), navigator.canShare({url: 'https://example.com/'}), navigator.canShare({text: 'x'}), navigator.canShare({url: 'http://[bad'})].join()`,
				"false,false,true,true,false"},
			{"canShare checks receiver", `(() => { try { navigator.canShare.call({}, {text: 'x'}); return 'no throw'; } catch (e) { return e.constructor.name; } })()`, "TypeError"},
			{"share rejects bad data first", `navigator.share({}).then(() => 'resolved', e => e.name)`, "TypeError"},
			{"share needs a user gesture", `navigator.share({url: 'https://example.com/'}).then(() => 'resolved', e => e.name)`, "NotAllowedError"},
			{"share checks receiver", `navigator.share.call({}, {text: 'x'}).then(() => 'resolved', e => e.name)`, "TypeError"},
		}
		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				if got := evalValue(t, c, tc.js); got != tc.want {
					t.Errorf("got %v, want %v", got, tc.want)
				}
			})
		}
		// 有用户手势时像用户关掉了分享框
		raw, err := c.sendCommand("Runtime.evaluate", map[string]any{"expression": `navigator.share({text: 'x'}).then(() => 'resolved', e => e.name)`,
			"awaitPromise": true, "returnByValue": true, "userGesture": true})
		if err != nil {
			t.Fatal(err)
		}
		var res struct{ Result struct{ Value any } }
		json.Unmarshal(raw, &res)
		if res.Result.Value != "AbortError" {
			t.Errorf("share with a user gesture = %v, want AbortError", res.Result.Value)
		}
	})

	t.Run("linux host identity stays without Web Share", func(t *testing.T) {
		c := run(t, "Linux x86_64")
		if got := evalValue(t, c, `'share' in navigator || 'canShare' in navigator`); got != false {
			t.Errorf("Web Share added for a Linux identity")
		}
	})
}
