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
