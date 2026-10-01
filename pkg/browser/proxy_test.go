package browser

import (
	"context"
	"encoding/base64"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"
)

const (
	testProxyUser = "prbg-user"
	testProxyPass = "prbg-pass"
)

// authProxy 要求 Basic 认证的 HTTP 代理：认证通过后直接返回测试页面（不真正转发），记录请求
type authProxy struct {
	*httptest.Server
	mu         sync.Mutex
	paths      []string // 通过代理认证的请求路径
	siteAuths  []string // /private 收到的 Authorization 头
	credential string   // 代理凭据对应的 Basic 头
}

// newAuthProxy 启动测试代理，测试结束自动关闭
func newAuthProxy(t *testing.T) *authProxy {
	t.Helper()
	ap := &authProxy{credential: "Basic " + base64.StdEncoding.EncodeToString([]byte(testProxyUser+":"+testProxyPass))}
	ap.Server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Proxy-Authorization") != ap.credential {
			w.Header().Set("Proxy-Authenticate", `Basic realm="prbg-proxy"`)
			w.WriteHeader(http.StatusProxyAuthRequired)
			return
		}
		ap.mu.Lock()
		ap.paths = append(ap.paths, r.URL.Path)
		if r.URL.Path == "/private" && r.Header.Get("Authorization") != "" {
			ap.siteAuths = append(ap.siteAuths, r.Header.Get("Authorization"))
		}
		ap.mu.Unlock()

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		switch r.URL.Path {
		case "/private": // 目标站要求 HTTP 认证：代理凭据绝不能被当作站点凭据发出去
			w.Header().Set("WWW-Authenticate", `Basic realm="site"`)
			w.WriteHeader(http.StatusUnauthorized)
			fmt.Fprint(w, "<title>401</title>")
		case "/popup": // 延迟发出子请求，保证此时新标签页的 Fetch 拦截已经生效
			fmt.Fprint(w, `<title>popup</title><script>setTimeout(() => { new Image().src = '/late' }, 1500)</script>`)
		default:
			fmt.Fprint(w, `<title>proxied</title><p id=ok>ok</p>`)
		}
	}))
	t.Cleanup(ap.Close)
	return ap
}

// config 返回指向该代理的 ProxyConfig
func (ap *authProxy) config(t *testing.T) *ProxyConfig {
	host, port, err := net.SplitHostPort(ap.Listener.Addr().String())
	if err != nil {
		t.Fatal(err)
	}
	return &ProxyConfig{Host: host, Port: port, Username: testProxyUser, Password: testProxyPass}
}

// waitPath 等代理收到指定路径的已认证请求
func (ap *authProxy) waitPath(path string, timeout time.Duration) bool {
	for deadline := time.Now().Add(timeout); time.Now().Before(deadline); time.Sleep(50 * time.Millisecond) {
		ap.mu.Lock()
		ok := slices.Contains(ap.paths, path)
		ap.mu.Unlock()
		if ok {
			return true
		}
	}
	return false
}

// TestURLPatternRegexp Fetch urlPattern 语义：* 任意串、? 单个字符、\ 转义，其余按字面匹配整条 URL
func TestURLPatternRegexp(t *testing.T) {
	cases := []struct {
		pattern, url string
		want         bool
	}{
		{"*", "https://a.test/x?y=1", true},
		{"*api/pay*", "https://a.test/api/pay?id=1", true},
		{"*api/pay*", "https://a.test/api/refund", false},
		{"https://a.test/?", "https://a.test/x", true},
		{"https://a.test/?", "https://a.test/xy", false},
		{"*.js", "https://a.test/app.js", true},
		{"*.js", "https://a.test/appXjs", false},             // . 按字面
		{`*a\*b*`, "https://a.test/a*b", true},               // \* 为字面星号
		{`*a\*b*`, "https://a.test/aXXb", false},             //
		{"https://a.test/(x)+", "https://a.test/(x)+", true}, // 正则元字符按字面
	}
	for _, tc := range cases {
		re, err := urlPatternRegexp(tc.pattern)
		if err != nil {
			t.Fatalf("urlPatternRegexp(%q): %v", tc.pattern, err)
		}
		if got := re.MatchString(tc.url); got != tc.want {
			t.Errorf("pattern %q match %q = %v, want %v", tc.pattern, tc.url, got, tc.want)
		}
	}
}

// TestProxyAuth 带账号密码的代理：两种实现都应自动应答代理 407 并加载页面，
// 且不能把代理凭据交给要求 HTTP 认证的目标站
func TestProxyAuth(t *testing.T) {
	for _, custom := range []bool{false, true} {
		t.Run(fmt.Sprintf("UseCustomCDP=%v", custom), func(t *testing.T) {
			proxy := newAuthProxy(t)
			ctx, cancel := context.WithTimeout(t.Context(), 60*time.Second)
			defer cancel()
			inst, err := Connect(ctx, &ConnectOptions{Headless: true, UseCustomCDP: custom, Proxy: proxy.config(t)})
			if err != nil {
				t.Fatalf("Connect: %v", err)
			}
			defer inst.Close()
			p := inst.Page()

			// 代理不会绕过非 loopback 域名，prbg.test 由测试代理直接应答
			if err := p.Navigate("http://prbg.test/"); err != nil {
				t.Fatalf("Navigate: %v", err)
			}
			if err := p.WaitForSelector("#ok"); err != nil {
				t.Fatalf("WaitForSelector: %v", err)
			}
			if title, err := p.GetTitle(); err != nil || title != "proxied" {
				t.Fatalf("GetTitle() = %q, %v; want %q", title, err, "proxied")
			}

			// 站点认证交给浏览器默认处理：headless 下原生行为就是 ERR_INVALID_AUTH_CREDENTIALS，不检查导航错误
			_ = p.Navigate("http://prbg.test/private")
			if !proxy.waitPath("/private", 10*time.Second) {
				t.Fatal("proxy never received /private")
			}
			time.Sleep(time.Second) // 留出浏览器带凭据重试的时间
			proxy.mu.Lock()
			defer proxy.mu.Unlock()
			for _, a := range proxy.siteAuths {
				if a == proxy.credential {
					t.Errorf("proxy credentials were sent to the target site: %q", a)
					break
				}
			}
		})
	}
}

// TestRequestInterception 两种实现的拦截行为一致：开启时处理器可伪造响应，关闭后请求正常放行；
// 配置了认证代理时（拦截与代理认证共用 Fetch）开关拦截不能破坏代理认证
func TestRequestInterception(t *testing.T) {
	for _, withProxy := range []bool{false, true} {
		for _, custom := range []bool{false, true} {
			t.Run(fmt.Sprintf("proxy=%v/UseCustomCDP=%v", withProxy, custom), func(t *testing.T) {
				opts := &ConnectOptions{Headless: true, UseCustomCDP: custom}
				base := "http://prbg.test"
				if withProxy {
					opts.Proxy = newAuthProxy(t).config(t)
				} else {
					srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
						fmt.Fprint(w, `<title>direct</title><p id=ok>ok</p>`)
					}))
					defer srv.Close()
					base = srv.URL
				}

				ctx, cancel := context.WithTimeout(t.Context(), 60*time.Second)
				defer cancel()
				inst, err := Connect(ctx, opts)
				if err != nil {
					t.Fatalf("Connect: %v", err)
				}
				defer inst.Close()
				p := inst.Page()

				if err := p.SetRequestInterception(true); err != nil {
					t.Fatalf("SetRequestInterception(true): %v", err)
				}
				if err := p.OnRequest(func(req *InterceptedRequest) error {
					if strings.HasSuffix(req.URL, "/mock") {
						return req.Respond(&RequestResponse{Status: 200, Body: `<title>mocked</title><p id=mocked>m</p>`})
					}
					return req.Continue()
				}); err != nil {
					t.Fatalf("OnRequest: %v", err)
				}
				if err := p.Navigate(base + "/mock"); err != nil {
					t.Fatalf("Navigate /mock: %v", err)
				}
				if err := p.WaitForSelector("#mocked"); err != nil {
					t.Fatalf("mocked response not rendered: %v", err)
				}

				if err := p.SetRequestInterception(false); err != nil {
					t.Fatalf("SetRequestInterception(false): %v", err)
				}
				if err := p.Navigate(base + "/after"); err != nil {
					t.Fatalf("Navigate /after: %v", err)
				}
				if err := p.WaitForSelector("#ok"); err != nil {
					t.Fatalf("page not loaded after disabling interception: %v", err)
				}
			})
		}
	}
}

// TestProxyAuthPopup 页面新开的标签页（chromedp 路径由 TargetHandler 接管）走认证代理时请求不能被挂起
func TestProxyAuthPopup(t *testing.T) {
	proxy := newAuthProxy(t)
	ctx, cancel := context.WithTimeout(t.Context(), 60*time.Second)
	defer cancel()
	inst, err := Connect(ctx, &ConnectOptions{Headless: true, Proxy: proxy.config(t)})
	if err != nil {
		t.Fatalf("Connect: %v", err)
	}
	defer inst.Close()
	p := inst.Page()

	if err := p.Navigate("http://prbg.test/"); err != nil {
		t.Fatalf("Navigate: %v", err)
	}
	if _, err := p.Evaluate(`window.open('http://prbg.test/popup'), true`); err != nil {
		t.Fatalf("window.open: %v", err)
	}
	if !proxy.waitPath("/popup", 10*time.Second) {
		t.Fatal("popup never loaded through the proxy")
	}
	if !proxy.waitPath("/late", 10*time.Second) {
		t.Error("request issued by the popup after load never reached the proxy (paused and never continued)")
	}
}
