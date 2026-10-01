package browser

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"
)

// navServer 导航测试站点：/page 引用一张 1.5s 后才返回的图片，load 事件因此推迟；记录每次 /page 请求的 Referer
type navServer struct {
	*httptest.Server
	mu       sync.Mutex
	referers map[string]string // query -> Referer
}

// newNavServer 启动导航测试站点，测试结束自动关闭
func newNavServer(t *testing.T) *navServer {
	t.Helper()
	ns := &navServer{referers: map[string]string{}}
	ns.Server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/page":
			ns.mu.Lock()
			ns.referers[r.URL.RawQuery] = r.Header.Get("Referer")
			ns.mu.Unlock()
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			fmt.Fprint(w, `<title>nav</title><p id=ready>ok</p><img src="/slow">`)
		case "/slow":
			select {
			case <-time.After(1500 * time.Millisecond):
			case <-r.Context().Done():
				return
			}
			w.Header().Set("Content-Type", "image/gif")
			w.Write([]byte("GIF89a\x01\x00\x01\x00\x00\x00\x00;"))
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(ns.Close)
	return ns
}

// referer 返回 /page?query 请求携带的 Referer
func (ns *navServer) referer(query string) string {
	ns.mu.Lock()
	defer ns.mu.Unlock()
	return ns.referers[query]
}

// evalString 执行 JS 并要求返回字符串
func evalString(t *testing.T, p Page, js string) string {
	t.Helper()
	v, err := p.Evaluate(js)
	if err != nil {
		t.Fatalf("Evaluate(%s): %v", js, err)
	}
	s, _ := v.(string)
	return s
}

// TestNavigation 两种实现的导航语义一致：Navigate 等到 load、网络错误返回 error、
// NavigateWithOptions 的 WaitUntil/Referrer/Timeout 生效、Refresh 等新文档加载完成
func TestNavigation(t *testing.T) {
	const docState = `document.readyState + ' ' + location.pathname + location.search`

	for _, custom := range []bool{false, true} {
		t.Run(fmt.Sprintf("UseCustomCDP=%v", custom), func(t *testing.T) {
			srv := newNavServer(t)
			ctx, cancel := context.WithTimeout(t.Context(), 120*time.Second)
			defer cancel()
			inst, err := Connect(ctx, &ConnectOptions{Headless: true, UseCustomCDP: custom})
			if err != nil {
				t.Fatalf("Connect: %v", err)
			}
			defer inst.Close()
			p := inst.Page()
			sp, ok := p.(PageWithSelector)
			if !ok {
				t.Fatalf("%T does not implement PageWithSelector", p)
			}

			t.Run("Navigate waits for load", func(t *testing.T) {
				if err := p.Navigate(srv.URL + "/page?load"); err != nil {
					t.Fatalf("Navigate: %v", err)
				}
				if got := evalString(t, p, docState); got != "complete /page?load" {
					t.Errorf("after Navigate: %q, want %q", got, "complete /page?load")
				}
			})

			t.Run("Navigate reports network errors", func(t *testing.T) {
				if err := p.Navigate("http://127.0.0.1:1/"); err == nil {
					t.Error("Navigate to a refused port returned nil error")
				}
			})

			t.Run("WaitDOMContentLoaded", func(t *testing.T) {
				err := sp.NavigateWithOptions(srv.URL+"/page?dcl", &NavigateOptions{WaitUntil: WaitDOMContentLoaded})
				if err != nil {
					t.Fatalf("NavigateWithOptions: %v", err)
				}
				// 图片 1.5s 后才返回，DOMContentLoaded 后立即返回时文档应处于 interactive
				if got := evalString(t, p, docState); got != "interactive /page?dcl" {
					t.Errorf("after DOMContentLoaded: %q, want %q", got, "interactive /page?dcl")
				}
			})

			t.Run("WaitNetworkIdle0", func(t *testing.T) {
				start := time.Now()
				err := sp.NavigateWithOptions(srv.URL+"/page?idle", &NavigateOptions{WaitUntil: WaitNetworkIdle0})
				if err != nil {
					t.Fatalf("NavigateWithOptions: %v", err)
				}
				if elapsed := time.Since(start); elapsed > 10*time.Second {
					t.Errorf("networkidle0 took %v, want < 10s", elapsed)
				}
				if got := evalString(t, p, docState); got != "complete /page?idle" {
					t.Errorf("after networkidle0: %q, want %q", got, "complete /page?idle")
				}
			})

			t.Run("Referrer", func(t *testing.T) {
				err := sp.NavigateWithOptions(srv.URL+"/page?ref", &NavigateOptions{Referrer: "http://ref.test/"})
				if err != nil {
					t.Fatalf("NavigateWithOptions: %v", err)
				}
				if got := srv.referer("ref"); got != "http://ref.test/" {
					t.Errorf("Referer = %q, want %q", got, "http://ref.test/")
				}
			})

			t.Run("Timeout", func(t *testing.T) {
				start := time.Now()
				err := sp.NavigateWithOptions(srv.URL+"/page?timeout", &NavigateOptions{Timeout: 300 * time.Millisecond})
				if err == nil {
					t.Error("NavigateWithOptions returned nil error although load takes 1.5s and timeout is 300ms")
				}
				if elapsed := time.Since(start); elapsed > 3*time.Second {
					t.Errorf("timed-out navigation took %v, want < 3s", elapsed)
				}
			})

			t.Run("Refresh waits for new document", func(t *testing.T) {
				if err := p.Navigate(srv.URL + "/page?refresh"); err != nil {
					t.Fatalf("Navigate: %v", err)
				}
				if _, err := p.Evaluate(`window.__beforeRefresh = true`); err != nil {
					t.Fatal(err)
				}
				if err := sp.Refresh(10 * time.Second); err != nil {
					t.Fatalf("Refresh: %v", err)
				}
				got := evalString(t, p, `(window.__beforeRefresh ? 'old ' : 'new ') + document.readyState`)
				if got != "new complete" {
					t.Errorf("after Refresh: %q, want %q", got, "new complete")
				}
			})
		})
	}
}
