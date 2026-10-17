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
