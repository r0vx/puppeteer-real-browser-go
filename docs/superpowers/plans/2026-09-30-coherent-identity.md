# 账号身份自洽 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 生产配置（无界面 Linux + Google Chrome + `UseCustomCDP` + `FingerprintUserID`）下，每个账号对外呈现一个在请求头、主页面、跨站 iframe、Worker、弹窗中完全一致的 Windows / macOS Chrome 身份，且不留可检测的注入痕迹。

**Architecture:** 身份（`Identity`）由账号指纹 + 真实浏览器信息纯函数派生；CustomCDP 改为浏览器级 WebSocket（flatten 会话），由 `targetManager` 自动附加每个新目标，在其运行前用 CDP Emulation 下发身份、注入最小 JS 层（原生伪装工具 + WebGL / 屏幕 / Worker navigator / 噪声），再放行。chromedp 路径只在主页面下发同一身份。

**Tech Stack:** Go 1.25、gorilla/websocket、chromedp + cdproto（已有依赖，不新增）、Chrome DevTools Protocol（Target / Emulation / Network / Page / Browser 域）。

**Spec:** `docs/superpowers/specs/2026-09-30-coherent-identity-design.md`

## Global Constraints

- Go 版本以 `go.mod` 为准（`go 1.25`）；不新增第三方依赖（`time/tzdata` 属标准库，可用）。
- 代码注释用中文，风格与所在文件一致；每个函数有注释；返回的错误带上下文（`fmt.Errorf("...: %w", err)`）。
- CustomCDP 路径**绝不调用 `Runtime.enable`**（`Runtime.evaluate` / `Runtime.runIfWaitingForDebugger` 可以）。
- 对外接口不变：`Connect`、`ConnectOptions`、`Page`、`PageWithSelector`、所有现有导出函数 / 类型保留（不删导出符号）。
- UA 格式逐字使用：Windows `Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/<主版本>.0.0.0 Safari/537.36`；macOS `Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/<主版本>.0.0.0 Safari/537.36`。
- 账号系统只由绑定 UA 判定：含 `Macintosh` → macOS，其余（含旧数据里的 Linux）→ Windows。UA 版本始终跟随真实浏览器。
- 浏览器 UI 高度（`outerHeight - innerHeight`）取 87（Chrome 154 在 macOS / Linux 有界面实测值）。
- 测试一律 `go test -race`；集成测试启动真实 Chrome；涉及无界面 Linux 的结论必须在 `test/linux-chrome.Dockerfile` 镜像（amd64 + Google Chrome 稳定版）里验证。
- 每个任务结束提交一次，提交信息末尾加 `Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>`。
- 构建 / vet 只针对库：`go build ./pkg/... ./internal/...`、`go vet ./pkg/... ./internal/...`（`cmd/example` 有多个 `main`，不能 `./...`）；Windows 交叉编译需保持通过（`GOOS=windows go vet ./pkg/... ./internal/...`）。

## Review Focus

1. **Connect 返回后立即导航**：主页面身份必须在 `Connect` 返回前下发完成，第一次 `Navigate` 就带新身份（Task 6 测试覆盖）。
2. **附加过程中目标消失**（跨站 iframe 刚插入就被移除、弹窗立刻关闭）：该目标的下发失败只打印警告，不阻塞其他目标、不让页面卡在暂停状态（Task 6 测试覆盖）。
3. **指纹文件损坏 / 目录不可写**：`Connect` 返回明确错误，不 panic、不静默退回默认身份（Task 6 测试覆盖）。
4. **有界面模式（Xvfb / 本机）不设指纹**：真实屏幕不是无界面默认的 800×600 时不调整窗口和视口，避免把用户可见的窗口拉伸（Task 2 单测覆盖）。
5. **旧指纹文件里的未知字段和无关字段**（TLS、HTTP2、电池等）：规范化后原样保留，只修正身份相关字段（Task 3 单测覆盖）。

## 实测约束（2026-10-01 在 macOS Chrome 154 与 Linux amd64 Google Chrome 154.0.8037.92 上探针验证）

- **不要用临时页读取真实浏览器信息**：`Target.closeTarget` 是异步的，紧接着开启的自动附加 5/5 次会先附加到正在关闭的临时页（即使 `detachedFromTarget` 已到、`getTargets` 已不列出），会被误当成主页面。改为在启动时已有的标签页上读取并 `detachFromTarget`，主页面按 targetId 认定（Task 5、6）。
- **不要用 `Target.createTarget` 直接带目标 URL**：首个文档和首个请求拿到的是原始值（竞态）；先建 `about:blank`（或用已有标签页）下发身份后再导航则稳定正确。库内新开页面（`BrowserContext.NewPage`）走 chromedp，本来就是先空白页再导航。
- **SharedWorker**：会被浏览器级自动附加接管；CDP 覆盖了它的请求头、`userAgentData`、时区、locale，但 `navigator.userAgent` / `appVersion` / `platform` 仍是原值，由 Worker 脚本补齐（Task 4）。专用 Worker 的 `languages` 原生也不跟随，同样由脚本补齐。
- **弹窗**：带 opener 的 `window.open` 弹窗身份正确（与 opener 同进程，`setLocaleOverride` 会报 "Another locale override is already in effect"，已按成功处理）。无界面 Chrome 154 下 `noopener` 弹窗和 `target=_blank` 链接根本不打开，测试只覆盖前者。
- **ServiceWorker**：`targetKindOf` 按 Worker 处理，但无界面探针里没能跑起来，未验证；不在规格验收清单内。

## 与规格的偏差

- 4.4"防重复注入用闭包内的 WeakSet"：不做。每个目标只经自己的会话注册一次脚本；同站 iframe 由父目标的注册覆盖，跨站 iframe / 弹窗 / Worker 各是独立目标，不存在同一文档执行两次。
- 6"下发失败记录到连接级错误回调"：改为打印警告（与现有代码风格一致），不新增公开 API（第 5 节要求接口不变）。
- 5"`GetAdvancedStealthScript` 等导出函数内容随新的 JS 层变化"：改为保留原内容并标注 Deprecated——新 JS 层需要 `Identity`，旧函数签名里没有。
- 5"`NewCustomCDPClient(debugURL)` 内部改为浏览器级连接"：它接收的是页面调试地址，保留页面级直连；`Connect` 自己用 `dialBrowser` 走浏览器级连接。
- 4.1"Chrome 版本读 `/json/version`"：改为在初始标签页用 `userAgentData.getHighEntropyValues` 读取，同时拿到品牌顺序和 GREASE 值（4.1"品牌"一条本就要求这样读）。
- 4.1 表中 macOS 的 Intel / AMD 显卡：取值池只收 Apple Silicon（Chrome 冻结 UA 下 Intel Mac 无法与 arm 区分，Apple Silicon 是当前主流）。

## File Structure

| 文件 | 职责 |
|---|---|
| `pkg/browser/cdp_conn.go`（新） | CDP WebSocket 连接：请求 ID、按 sessionId 路由响应和事件、可取消订阅 |
| `pkg/browser/identity.go`（新） | `Identity` 类型、系统判定、UA 规范化、由指纹 / 真实浏览器派生身份 |
| `pkg/browser/identity_pools.go`（新） | Windows / macOS 的屏幕、显卡（含 WebGL 能力参数）、核数、系统版本取值池与确定性抽取 |
| `pkg/browser/identity_js.go`（新） | 注入脚本生成：原生伪装工具、WebGL、屏幕、MouseEvent、Worker navigator、canvas / 音频噪声 |
| `pkg/browser/identity_apply.go`（新） | 选定初始标签页并读取真实浏览器信息（`initialPageTarget`、`readHostInfo`）、按目标类型下发身份（`applyIdentity`）、窗口外框参数 |
| `pkg/browser/target_manager.go`（新） | 浏览器级自动附加：每个目标下发身份后放行；提供主页面会话 |
| `pkg/browser/cdp_custom.go`（改） | `CustomCDPClient` 改走 `cdpConn` + sessionId；`CustomCDPConnector.Connect` 改用 target manager；删除旧的 UA / 脚本注入 |
| `pkg/browser/connector.go`（改） | `CDPPage.initialize` 改为下发同一身份（仅主页面）；去掉固定 1920×1080 视口 |
| `pkg/browser/fingerprint_config.go`（改） | 生成器改为 Windows / macOS 自洽取值；新增 `Normalize`、`SchemaVersion`、`PlatformVersion` |
| `pkg/browser/user_fingerprint_manager.go`（改） | 加载时规范化并回写；新建时按绑定 UA 选系统 |
| `pkg/browser/stealth.go`、`script_cache.go`、`fingerprint_injector.go`、`enhanced_audio_webgl_injector.go`、`timestamp_fingerprint_injector.go`（改） | 导出函数保留，标注 Deprecated（连接流程不再使用） |
| `internal/config/config.go`、`pkg/browser/launcher.go`（改） | 配置代理时加 `--force-webrtc-ip-handling-policy=disable_non_proxied_udp` |
| `test/linux-chrome.Dockerfile`、`scripts/test-linux-chrome.sh`（新） | 模拟生产的 Linux 测试环境 |
| `docs/deploy/production.Dockerfile`、`docs/deploy/README.md`（新） | 生产镜像（Google Chrome + Windows 字体 + tini）与部署说明 |
| 测试：`cdp_conn_test.go`、`identity_test.go`、`fingerprint_normalize_test.go`、`identity_js_test.go`、`identity_integration_test.go`（新） | 各任务的测试 |

---
### Task 1: CDP 连接层（按 sessionId 路由）

把 `CustomCDPClient` 的 WebSocket 收发抽成 `cdpConn`，支持 flatten 会话：同一条连接上，响应按 id 匹配，事件按 `(sessionId, method)` 分发，订阅 `"*"` 匹配所有会话。`CustomCDPClient` 变成"连接 + 会话 ID"，现有行为不变（直连页面时会话 ID 为空）。

**Files:**
- Create: `pkg/browser/cdp_conn.go`
- Create: `pkg/browser/cdp_conn_test.go`
- Modify: `pkg/browser/cdp_custom.go`（`CustomCDPClient` 结构体、`eventSub`、`NewCustomCDPClient` 末尾、`handleMessages`、`OnEvent`、`subscribe`、`sendCommand`、`Close`）

**Interfaces:**
- Produces:
  - `func dialCDP(wsURL string) (*cdpConn, error)`
  - `func (c *cdpConn) call(session, method string, params any) (json.RawMessage, error)`
  - `func (c *cdpConn) subscribe(session, method string, fn func(session string, params json.RawMessage)) (unsubscribe func())`
  - `func (c *cdpConn) close() error`
  - `type CustomCDPClient struct { conn *cdpConn; sessionID string; url string }`（后续任务直接构造它）

- [ ] **Step 1: 写失败的测试** `pkg/browser/cdp_conn_test.go`

```go
package browser

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

// fakeCDPServer 模拟 flatten 模式的 CDP：每收到一条命令，先推送会话 A、B 各一个事件，再回显会话和方法名作为响应
func fakeCDPServer(t *testing.T) string {
	t.Helper()
	up := websocket.Upgrader{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ws, err := up.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer ws.Close()
		for {
			var req map[string]any
			if ws.ReadJSON(&req) != nil {
				return
			}
			sess, _ := req["sessionId"].(string)
			ws.WriteJSON(map[string]any{"method": "Test.event", "sessionId": "A", "params": map[string]any{"n": 1}})
			ws.WriteJSON(map[string]any{"method": "Test.event", "sessionId": "B", "params": map[string]any{"n": 2}})
			ws.WriteJSON(map[string]any{"id": req["id"], "result": map[string]any{"session": sess, "method": req["method"]}})
		}
	}))
	t.Cleanup(srv.Close)
	return "ws" + strings.TrimPrefix(srv.URL, "http")
}

// TestCDPConnRouting 响应按 id 回到调用方并带上会话；事件只投递给对应会话的订阅者，"*" 订阅收到全部；取消订阅后不再投递
func TestCDPConnRouting(t *testing.T) {
	c, err := dialCDP(fakeCDPServer(t))
	if err != nil {
		t.Fatal(err)
	}
	defer c.close()

	gotA := make(chan string, 8)
	gotAll := make(chan string, 8)
	stopA := c.subscribe("A", "Test.event", func(s string, _ json.RawMessage) { gotA <- s })
	c.subscribe("*", "Test.event", func(s string, _ json.RawMessage) { gotAll <- s })

	raw, err := c.call("B", "Test.ping", nil)
	if err != nil {
		t.Fatalf("call: %v", err)
	}
	var res struct{ Session, Method string }
	if err := json.Unmarshal(raw, &res); err != nil {
		t.Fatal(err)
	}
	if res.Session != "B" || res.Method != "Test.ping" {
		t.Errorf("result = %+v, want session B / Test.ping", res)
	}

	// "*" 订阅应收到 A、B 两个事件；A 订阅只收到 A
	seen := map[string]bool{}
	for range 2 {
		select {
		case s := <-gotAll:
			seen[s] = true
		case <-time.After(2 * time.Second):
			t.Fatal("wildcard subscriber missed events")
		}
	}
	if !seen["A"] || !seen["B"] {
		t.Errorf("wildcard saw %v, want A and B", seen)
	}
	select {
	case s := <-gotA:
		if s != "A" {
			t.Errorf("session A subscriber got event from %q", s)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("session A subscriber missed its event")
	}

	stopA()
	if _, err := c.call("", "Test.ping", nil); err != nil {
		t.Fatal(err)
	}
	select {
	case s := <-gotA:
		t.Errorf("unsubscribed handler still received event from %q", s)
	case <-time.After(300 * time.Millisecond):
	}
}
```

- [ ] **Step 2: 运行确认失败**

Run: `go test -count=1 -run TestCDPConnRouting ./pkg/browser/`
Expected: 编译失败，`undefined: dialCDP`

- [ ] **Step 3: 实现** `pkg/browser/cdp_conn.go`

```go
package browser

import (
	"context"
	"encoding/json"
	"fmt"
	"slices"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gorilla/websocket"
)

// cdpCommandTimeout 单条 CDP 命令的等待上限
const cdpCommandTimeout = 30 * time.Second

// cdpConn 一条 CDP WebSocket 连接（flatten 模式）：响应按 id 匹配，事件按 (sessionId, method) 分发。
// sessionId 为空表示浏览器级会话，或直连页面 WebSocket 时的页面本身
type cdpConn struct {
	ws      *websocket.Conn
	writeMu sync.Mutex
	nextID  atomic.Int64

	mu       sync.Mutex
	pending  map[int64]chan cdpReply
	handlers map[string][]eventSub // key 见 handlerKey；会话为 "*" 时匹配所有会话
	nextSub  int64

	ctx    context.Context
	cancel context.CancelFunc
}

// cdpReply 命令响应
type cdpReply struct {
	Result json.RawMessage
	Error  *CDPError
}

// eventSub 一个事件订阅，id 用于取消
type eventSub struct {
	id int64
	fn func(session string, params json.RawMessage)
}

// handlerKey 订阅表的键
func handlerKey(session, method string) string { return session + "\x00" + method }

// dialCDP 连接 CDP WebSocket 并启动读循环
func dialCDP(wsURL string) (*cdpConn, error) {
	ws, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		return nil, fmt.Errorf("dial %s: %w", wsURL, err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	c := &cdpConn{
		ws:       ws,
		pending:  make(map[int64]chan cdpReply),
		handlers: make(map[string][]eventSub),
		ctx:      ctx,
		cancel:   cancel,
	}
	go c.readLoop()
	return c, nil
}

// readLoop 读取消息直到连接关闭：带 id 的是响应，其余按会话和方法分发事件
func (c *cdpConn) readLoop() {
	defer c.cancel()
	for {
		_, data, err := c.ws.ReadMessage()
		if err != nil {
			return
		}
		var msg struct {
			ID        int64           `json:"id"`
			Method    string          `json:"method"`
			SessionID string          `json:"sessionId"`
			Params    json.RawMessage `json:"params"`
			Result    json.RawMessage `json:"result"`
			Error     *CDPError       `json:"error"`
		}
		if json.Unmarshal(data, &msg) != nil {
			continue
		}
		if msg.ID != 0 {
			c.mu.Lock()
			ch := c.pending[msg.ID]
			delete(c.pending, msg.ID)
			c.mu.Unlock()
			if ch != nil {
				ch <- cdpReply{Result: msg.Result, Error: msg.Error} // 缓冲为 1，不会阻塞
			}
			continue
		}
		if msg.Method == "" {
			continue
		}
		c.mu.Lock()
		subs := slices.Concat(c.handlers[handlerKey(msg.SessionID, msg.Method)], c.handlers[handlerKey("*", msg.Method)])
		c.mu.Unlock()
		if len(subs) > 0 {
			// 异步执行，避免处理器里再发命令时阻塞读循环
			go func() {
				for _, s := range subs {
					s.fn(msg.SessionID, msg.Params)
				}
			}()
		}
	}
}

// call 在指定会话上发送命令并等待响应
func (c *cdpConn) call(session, method string, params any) (json.RawMessage, error) {
	id := c.nextID.Add(1)
	ch := make(chan cdpReply, 1)
	c.mu.Lock()
	c.pending[id] = ch
	c.mu.Unlock()
	defer func() {
		c.mu.Lock()
		delete(c.pending, id)
		c.mu.Unlock()
	}()

	msg := map[string]any{"id": id, "method": method}
	if params != nil {
		msg["params"] = params
	}
	if session != "" {
		msg["sessionId"] = session
	}
	c.writeMu.Lock()
	err := c.ws.WriteJSON(msg)
	c.writeMu.Unlock()
	if err != nil {
		return nil, fmt.Errorf("send %s: %w", method, err)
	}

	timer := time.NewTimer(cdpCommandTimeout)
	defer timer.Stop()
	select {
	case r := <-ch:
		if r.Error != nil {
			return nil, fmt.Errorf("CDP error: %s", r.Error.Message)
		}
		return r.Result, nil
	case <-timer.C:
		return nil, fmt.Errorf("timeout waiting for response to %s", method)
	case <-c.ctx.Done():
		return nil, fmt.Errorf("connection closed")
	}
}

// subscribe 订阅事件，session 为 "*" 时匹配所有会话；返回取消函数
func (c *cdpConn) subscribe(session, method string, fn func(session string, params json.RawMessage)) (unsubscribe func()) {
	key := handlerKey(session, method)
	c.mu.Lock()
	c.nextSub++
	id := c.nextSub
	c.handlers[key] = append(c.handlers[key], eventSub{id: id, fn: fn})
	c.mu.Unlock()
	return func() {
		c.mu.Lock()
		defer c.mu.Unlock()
		c.handlers[key] = slices.DeleteFunc(c.handlers[key], func(s eventSub) bool { return s.id == id })
	}
}

// close 关闭连接
func (c *cdpConn) close() error {
	c.cancel()
	return c.ws.Close()
}
```

- [ ] **Step 4: 改造 `CustomCDPClient`**（`pkg/browser/cdp_custom.go`）

结构体整体替换为：

```go
// CustomCDPClient implements a custom CDP client that avoids Runtime.Enable leaks.
// 它是"连接 + 会话"：sessionID 为空表示直连页面 WebSocket，非空表示浏览器级连接上的 flatten 会话
type CustomCDPClient struct {
	conn      *cdpConn
	sessionID string
	url       string
}
```

删除 `cdp_custom.go` 里原来的 `eventSub` 类型（已移到 `cdp_conn.go`）和整个 `handleMessages` 方法。`NewCustomCDPClient` 中从 `// Connect to WebSocket` 到函数结尾替换为：

```go
	conn, err := dialCDP(wsURL)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to WebSocket: %w", err)
	}
	return &CustomCDPClient{conn: conn, url: wsURL}, nil
}
```

`OnEvent`、`subscribe`、`sendCommand`、`Close` 替换为：

```go
// OnEvent subscribes to a CDP event
func (c *CustomCDPClient) OnEvent(method string, handler func(json.RawMessage)) {
	c.subscribe(method, handler)
}

// subscribe 订阅本会话的事件并返回取消函数（用于导航等待这类一次性监听）
func (c *CustomCDPClient) subscribe(method string, handler func(json.RawMessage)) (unsubscribe func()) {
	return c.conn.subscribe(c.sessionID, method, func(_ string, params json.RawMessage) { handler(params) })
}

// sendCommand sends a CDP command on this session and waits for the response
func (c *CustomCDPClient) sendCommand(method string, params interface{}) (json.RawMessage, error) {
	return c.conn.call(c.sessionID, method, params)
}

// Close closes the CDP connection
func (c *CustomCDPClient) Close() error {
	return c.conn.close()
}
```

`cdp_custom.go` 中不再使用的 import（`sync` 若只剩 `fetchMu` 仍在用则保留，`websocket` 移除）按 `go vet` 提示清理。

- [ ] **Step 5: 运行测试**

Run: `go vet ./pkg/... ./internal/... && go test -race -count=1 -run 'TestCDPConnRouting|TestCustomCDPPage|TestNavigation|TestProxyAuth|TestRequestInterception|TestTurnstileOption' ./pkg/browser/`
Expected: 全部 PASS（CustomCDP 现有行为不变）

- [ ] **Step 6: 提交**

```bash
git add pkg/browser/cdp_conn.go pkg/browser/cdp_conn_test.go pkg/browser/cdp_custom.go
git commit -m "refactor(custom-cdp): route CDP messages by session over a shared connection

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
```

---
### Task 2: 身份模型与取值池

纯函数：由（规范化后的）账号指纹 + 真实浏览器信息派生 `Identity`；不设指纹时由真实浏览器信息派生（只去无界面痕迹）。取值池按系统分开，按用户 ID 确定性抽取。

**Files:**
- Create: `pkg/browser/identity_pools.go`
- Create: `pkg/browser/identity.go`
- Create: `pkg/browser/identity_test.go`

**Interfaces:**
- Consumes: `FingerprintConfig`（现有类型；Task 3 会新增 `Browser.PlatformVersion` 字段，本任务先在 `fingerprint_config.go` 的 `BrowserConfig` 里加上该字段：`PlatformVersion string \`json:"platform_version,omitempty"\``）
- Produces:
  - `type OSFamily string`；`OSWindows`、`OSMac`
  - `func osFromUserAgent(ua string) OSFamily`、`func canonicalUserAgent(os OSFamily, major string) string`、`func uaMajorVersion(ua string) string`、`func platformFor(os OSFamily) string`
  - `type hostInfo struct`、`func parseHostInfo(raw string) (hostInfo, error)`、`func (h hostInfo) majorVersion() string`
  - `type Identity struct`、`func (id *Identity) Locale() string`
  - `func identityFromConfig(cfg *FingerprintConfig, host hostInfo) (*Identity, error)`、`func hostIdentity(host hostInfo) *Identity`
  - 池：`type weighted[T any]`、`func pickWeighted[T any](items []weighted[T], seed uint64) T`、`func identitySeed(userID, purpose string) uint64`、`screensFor / gpusFor / coresFor / platformVersionsFor(os)`、`func lookupGPU(os OSFamily, renderer string) (gpuProfile, bool)`、`func pickGPU(os OSFamily, family string, seed uint64) gpuProfile`、`func gpuFamily(vendor, renderer string) string`、`func screenSpecFor(os OSFamily, w, h int, dpr float64) (screenSpec, bool)`、`func geometryFor(os OSFamily, s screenSpec, platformVersion string) screenGeometry`

- [ ] **Step 1: 写失败的测试** `pkg/browser/identity_test.go`

```go
package browser

import (
	"slices"
	"testing"
)

// testHost 无界面 Linux 服务器上 Google Chrome 154 报告的真实值（与 Docker 实测一致）
func testHost() hostInfo {
	return hostInfo{
		UserAgent:           "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) HeadlessChrome/154.0.0.0 Safari/537.36",
		Platform:            "Linux x86_64",
		Languages:           []string{"en-US", "en"},
		Timezone:            "Asia/Shanghai",
		HardwareConcurrency: 12,
		Screen:              [3]float64{800, 600, 1},
		Brands:              []uaBrand{{"Chromium", "154"}, {"Google Chrome", "154"}, {"Not A(Brand", "99"}},
		FullVersionList:     []uaBrand{{"Chromium", "154.0.8037.92"}, {"Google Chrome", "154.0.8037.92"}, {"Not A(Brand", "99.0.0.0"}},
		UAPlatform:          "Linux",
		PlatformVersion:     "6.8.0",
		Architecture:        "x86",
		Bitness:             "64",
		FormFactors:         []string{"Desktop"},
		WebGLVendor:         "Google Inc. (Google)",
		WebGLRenderer:       "ANGLE (Google, Vulkan 1.3.0 (SwiftShader Device (Subzero) (0x0000C0DE)), SwiftShader driver)",
	}
}

func TestOSFromUserAgent(t *testing.T) {
	cases := []struct {
		ua   string
		want OSFamily
	}{
		{"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36", OSWindows},
		{"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36", OSMac},
		{"Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36", OSWindows}, // 旧数据里的 Linux 按 Windows
		{"", OSWindows},
	}
	for _, tc := range cases {
		if got := osFromUserAgent(tc.ua); got != tc.want {
			t.Errorf("osFromUserAgent(%q) = %q, want %q", tc.ua, got, tc.want)
		}
	}
}

func TestCanonicalUserAgent(t *testing.T) {
	if got, want := canonicalUserAgent(OSWindows, "154"), "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/154.0.0.0 Safari/537.36"; got != want {
		t.Errorf("windows UA = %q, want %q", got, want)
	}
	if got, want := canonicalUserAgent(OSMac, "154"), "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/154.0.0.0 Safari/537.36"; got != want {
		t.Errorf("mac UA = %q, want %q", got, want)
	}
	if got := uaMajorVersion("... Chrome/128.0.6613.138 Safari/537.36"); got != "128" {
		t.Errorf("uaMajorVersion = %q, want 128", got)
	}
}

// TestIdentityFromConfig 版本与品牌来自真实浏览器，系统相关字段由账号系统派生，几何按系统规则推算
func TestIdentityFromConfig(t *testing.T) {
	win := &FingerprintConfig{UserID: "acct-win"}
	win.Browser.UserAgent = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/128.0.0.0 Safari/537.36"
	win.Browser.Languages = []string{"ja-JP", "ja"}
	win.Browser.Language = "ja-JP"
	win.Browser.HardwareConcurrency = 8
	win.Browser.PlatformVersion = "15.0.0"
	win.Timezone.Timezone = "Asia/Tokyo"
	win.Screen = ScreenConfig{Width: 1536, Height: 864, DevicePixelRatio: 1.25}
	win.WebGL.Vendor, win.WebGL.Renderer = windowsGPUs[0].Value.Vendor, windowsGPUs[0].Value.Renderer

	id, err := identityFromConfig(win, testHost())
	if err != nil {
		t.Fatal(err)
	}
	checks := []struct {
		name      string
		got, want any
	}{
		{"UserAgent", id.UserAgent, "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/154.0.0.0 Safari/537.36"},
		{"Platform", id.Platform, "Win32"},
		{"AcceptLanguage", id.AcceptLanguage, "ja-JP,ja"},
		{"Locale", id.Locale(), "ja-JP"},
		{"Timezone", id.Timezone, "Asia/Tokyo"},
		{"Metadata.Platform", id.Metadata.Platform, "Windows"},
		{"Metadata.PlatformVersion", id.Metadata.PlatformVersion, "15.0.0"},
		{"Metadata.Architecture", id.Metadata.Architecture, "x86"},
		{"Metadata.Bitness", id.Metadata.Bitness, "64"},
		{"Metadata.FullVersion", id.Metadata.FullVersionList[1].Version, "154.0.8037.92"},
		{"AvailHeight", id.Screen.AvailHeight, 826}, // 864 - round(48/1.25)
		{"OuterHeight", id.Screen.OuterHeight, 826},
		{"InnerHeight", id.Screen.InnerHeight, 739}, // 826 - 87
		{"InnerWidth", id.Screen.InnerWidth, 1536},
		{"GPU", id.GPU.Renderer, windowsGPUs[0].Value.Renderer},
	}
	for _, c := range checks {
		if c.got != c.want {
			t.Errorf("%s = %v, want %v", c.name, c.got, c.want)
		}
	}
	if !slices.Equal(id.Metadata.FormFactors, []string{"Desktop"}) {
		t.Errorf("FormFactors = %v, want [Desktop]", id.Metadata.FormFactors)
	}
	if id.NoiseSeed == 0 {
		t.Error("NoiseSeed is 0; fingerprint accounts must carry noise")
	}
	again, _ := identityFromConfig(win, testHost())
	if again.NoiseSeed != id.NoiseSeed {
		t.Error("NoiseSeed is not deterministic per account")
	}

	mac := &FingerprintConfig{UserID: "acct-mac"}
	mac.Browser.UserAgent = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36"
	mac.Browser.Languages = []string{"zh-CN", "zh"}
	mac.Browser.HardwareConcurrency = 10
	mac.Browser.PlatformVersion = "15.5.0"
	mac.Timezone.Timezone = "Asia/Shanghai"
	mac.Screen = ScreenConfig{Width: 1512, Height: 982, DevicePixelRatio: 2}
	mac.WebGL.Vendor, mac.WebGL.Renderer = macGPUs[0].Value.Vendor, macGPUs[0].Value.Renderer
	mid, err := identityFromConfig(mac, testHost())
	if err != nil {
		t.Fatal(err)
	}
	if mid.Platform != "MacIntel" || mid.Metadata.Platform != "macOS" || mid.Metadata.Architecture != "arm" {
		t.Errorf("mac identity platform fields = %q / %q / %q", mid.Platform, mid.Metadata.Platform, mid.Metadata.Architecture)
	}
	// MacBook Pro 14 有刘海，菜单栏 38
	if mid.Screen.AvailTop != 38 || mid.Screen.AvailHeight != 944 || mid.Screen.InnerHeight != 857 {
		t.Errorf("mac geometry = %+v, want availTop 38 / availHeight 944 / innerHeight 857", mid.Screen)
	}

	bad := *win
	bad.WebGL.Renderer = "ANGLE (AMD, AMD Radeon(TM) Graphics Direct3D11 vs_5_0 ps_5_0, D3D11) (Build 26453)"
	if _, err := identityFromConfig(&bad, testHost()); err == nil {
		t.Error("renderer outside the pool must be rejected (config not normalized)")
	}
}

// TestHostIdentity 不设指纹：保留真实值，只去掉无界面痕迹；有界面（真实屏幕）时不调整几何、不改 WebGL
func TestHostIdentity(t *testing.T) {
	id := hostIdentity(testHost())
	if id.UserAgent != "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/154.0.0.0 Safari/537.36" {
		t.Errorf("UserAgent = %q", id.UserAgent)
	}
	if id.Platform != "Linux x86_64" || id.Metadata.Platform != "Linux" {
		t.Errorf("platform changed: %q / %q", id.Platform, id.Metadata.Platform)
	}
	if id.Screen.Width != 1920 || id.Screen.Height != 1080 || id.Screen.InnerHeight != id.Screen.AvailHeight-browserUIHeight {
		t.Errorf("headless screen not replaced: %+v", id.Screen)
	}
	if id.GPU == nil || id.GPU.Renderer != linuxHostGPU.Renderer || id.GPU.Caps != nil {
		t.Errorf("SwiftShader not masked by vendor/renderer only: %+v", id.GPU)
	}
	if id.NoiseSeed != 0 {
		t.Error("default path must not add noise")
	}

	headful := testHost()
	headful.UserAgent = "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/154.0.0.0 Safari/537.36"
	headful.Screen = [3]float64{1920, 1080, 1}
	headful.WebGLRenderer = "ANGLE (NVIDIA, NVIDIA GeForce RTX 3060, OpenGL 4.6)"
	hid := hostIdentity(headful)
	if hid.Screen.Width != 0 {
		t.Errorf("headful host screen must be left alone, got %+v", hid.Screen)
	}
	if hid.GPU != nil {
		t.Errorf("real GPU must not be masked, got %+v", hid.GPU)
	}
}
```

- [ ] **Step 2: 运行确认失败**

Run: `go test -count=1 -run 'TestOSFromUserAgent|TestCanonicalUserAgent|TestIdentityFromConfig|TestHostIdentity' ./pkg/browser/`
Expected: 编译失败（`undefined: hostInfo` 等）

- [ ] **Step 3: 实现取值池** `pkg/browser/identity_pools.go`

```go
package browser

import (
	"hash/fnv"
	"strings"
)

// weighted 带权重的取值
type weighted[T any] struct {
	Value  T
	Weight int
}

// pickWeighted 按种子确定性地加权抽取：同一种子总是抽到同一项
func pickWeighted[T any](items []weighted[T], seed uint64) T {
	total := 0
	for _, it := range items {
		total += it.Weight
	}
	r := int(seed % uint64(total))
	for _, it := range items {
		if r < it.Weight {
			return it.Value
		}
		r -= it.Weight
	}
	return items[len(items)-1].Value
}

// identitySeed 由用户 ID 和用途派生种子：同一账号每次抽到相同的值，不同用途互不相关
func identitySeed(userID, purpose string) uint64 {
	h := fnv.New64a()
	h.Write([]byte(userID + "\x00" + purpose))
	return h.Sum64()
}

// screenSpec 屏幕规格（CSS 像素）；Notch 表示带刘海的 MacBook（菜单栏更高）
type screenSpec struct {
	Width, Height int
	DPR           float64
	Notch         bool
}

// windowsScreens Windows 常见屏幕（含 125% / 150% 缩放后的 CSS 尺寸）
var windowsScreens = []weighted[screenSpec]{
	{screenSpec{1920, 1080, 1, false}, 35},
	{screenSpec{1536, 864, 1.25, false}, 15},
	{screenSpec{1366, 768, 1, false}, 12},
	{screenSpec{2560, 1440, 1, false}, 10},
	{screenSpec{1280, 720, 1.5, false}, 6},
	{screenSpec{1440, 900, 1, false}, 6},
	{screenSpec{1600, 900, 1, false}, 6},
	{screenSpec{1920, 1200, 1, false}, 5},
	{screenSpec{2048, 1152, 1.25, false}, 5},
}

// macScreens Mac 常见屏幕
var macScreens = []weighted[screenSpec]{
	{screenSpec{1440, 900, 2, false}, 15},  // MacBook Air 13（M1）
	{screenSpec{1470, 956, 2, true}, 20},   // MacBook Air 13（M2 / M3）
	{screenSpec{1512, 982, 2, true}, 20},   // MacBook Pro 14
	{screenSpec{1710, 1112, 2, true}, 10},  // MacBook Air 15
	{screenSpec{1728, 1117, 2, true}, 10},  // MacBook Pro 16
	{screenSpec{2240, 1260, 2, false}, 10}, // iMac 24
	{screenSpec{1920, 1080, 1, false}, 15}, // 外接显示器
}

// webglCaps 显卡在 Chrome（ANGLE）下暴露的 WebGL 能力参数
type webglCaps struct {
	MaxTextureSize, MaxCubeMapTextureSize, MaxRenderbufferSize int
	MaxViewportDims                                            [2]int
	MaxVertexAttribs, MaxVertexUniformVectors                  int
	MaxFragmentUniformVectors, MaxVaryingVectors               int
	MaxTextureImageUnits, MaxVertexTextureImageUnits           int
	MaxCombinedTextureImageUnits                               int
	AliasedPointSizeRange, AliasedLineWidthRange               [2]float64
}

// gpuProfile 一款显卡的 WebGL 标识与能力参数；Caps 为 nil 时只改写标识
type gpuProfile struct {
	Vendor   string // UNMASKED_VENDOR_WEBGL
	Renderer string // UNMASKED_RENDERER_WEBGL
	Caps     *webglCaps
}

// d3d11Caps Windows 上 ANGLE D3D11 后端对各厂商显卡统一暴露的能力参数
var d3d11Caps = &webglCaps{
	MaxTextureSize: 16384, MaxCubeMapTextureSize: 16384, MaxRenderbufferSize: 16384,
	MaxViewportDims: [2]int{32767, 32767}, MaxVertexAttribs: 16, MaxVertexUniformVectors: 4096,
	MaxFragmentUniformVectors: 1024, MaxVaryingVectors: 30, MaxTextureImageUnits: 16,
	MaxVertexTextureImageUnits: 16, MaxCombinedTextureImageUnits: 32,
	AliasedPointSizeRange: [2]float64{1, 1024}, AliasedLineWidthRange: [2]float64{1, 1},
}

// appleCaps Apple 芯片（ANGLE Metal 后端）实测值：Apple M2 Max，Chrome 154
var appleCaps = &webglCaps{
	MaxTextureSize: 16384, MaxCubeMapTextureSize: 16384, MaxRenderbufferSize: 16384,
	MaxViewportDims: [2]int{16384, 16384}, MaxVertexAttribs: 16, MaxVertexUniformVectors: 1024,
	MaxFragmentUniformVectors: 1024, MaxVaryingVectors: 30, MaxTextureImageUnits: 16,
	MaxVertexTextureImageUnits: 16, MaxCombinedTextureImageUnits: 32,
	AliasedPointSizeRange: [2]float64{1, 511}, AliasedLineWidthRange: [2]float64{1, 1},
}

// windowsGPUs Windows 常见显卡（renderer 串含 PCI 设备 ID，与真实 Chrome 格式一致）
var windowsGPUs = []weighted[gpuProfile]{
	{gpuProfile{"Google Inc. (Intel)", "ANGLE (Intel, Intel(R) UHD Graphics 620 (0x00005917) Direct3D11 vs_5_0 ps_5_0, D3D11)", d3d11Caps}, 20},
	{gpuProfile{"Google Inc. (Intel)", "ANGLE (Intel, Intel(R) UHD Graphics 630 (0x00003E92) Direct3D11 vs_5_0 ps_5_0, D3D11)", d3d11Caps}, 15},
	{gpuProfile{"Google Inc. (Intel)", "ANGLE (Intel, Intel(R) Iris(R) Xe Graphics (0x00009A49) Direct3D11 vs_5_0 ps_5_0, D3D11)", d3d11Caps}, 20},
	{gpuProfile{"Google Inc. (NVIDIA)", "ANGLE (NVIDIA, NVIDIA GeForce GTX 1650 (0x00001F82) Direct3D11 vs_5_0 ps_5_0, D3D11)", d3d11Caps}, 12},
	{gpuProfile{"Google Inc. (NVIDIA)", "ANGLE (NVIDIA, NVIDIA GeForce RTX 3060 (0x00002503) Direct3D11 vs_5_0 ps_5_0, D3D11)", d3d11Caps}, 13},
	{gpuProfile{"Google Inc. (AMD)", "ANGLE (AMD, AMD Radeon(TM) Graphics (0x00001638) Direct3D11 vs_5_0 ps_5_0, D3D11)", d3d11Caps}, 20},
}

// macGPUs Apple 芯片 Mac
var macGPUs = []weighted[gpuProfile]{
	{gpuProfile{"Google Inc. (Apple)", "ANGLE (Apple, ANGLE Metal Renderer: Apple M1, Unspecified Version)", appleCaps}, 30},
	{gpuProfile{"Google Inc. (Apple)", "ANGLE (Apple, ANGLE Metal Renderer: Apple M2, Unspecified Version)", appleCaps}, 30},
	{gpuProfile{"Google Inc. (Apple)", "ANGLE (Apple, ANGLE Metal Renderer: Apple M3, Unspecified Version)", appleCaps}, 20},
	{gpuProfile{"Google Inc. (Apple)", "ANGLE (Apple, ANGLE Metal Renderer: Apple M1 Pro, Unspecified Version)", appleCaps}, 10},
	{gpuProfile{"Google Inc. (Apple)", "ANGLE (Apple, ANGLE Metal Renderer: Apple M2 Pro, Unspecified Version)", appleCaps}, 10},
}

// linuxHostGPU 不设指纹的默认路径在 Linux 服务器上遮挡 SwiftShader 用的标识（只改标识，不改能力参数）
var linuxHostGPU = gpuProfile{Vendor: "Google Inc. (Intel)", Renderer: "ANGLE (Intel, Mesa Intel(R) UHD Graphics 620 (KBL GT2), OpenGL 4.6)"}

var (
	windowsCores            = []weighted[int]{{4, 25}, {6, 15}, {8, 30}, {12, 15}, {16, 12}, {20, 3}}
	macCores                = []weighted[int]{{8, 60}, {10, 25}, {12, 15}}
	windowsPlatformVersions = []weighted[string]{{"10.0.0", 40}, {"15.0.0", 30}, {"19.0.0", 30}} // Win10 / Win11 22H2–23H2 / Win11 24H2
	macPlatformVersions     = []weighted[string]{{"14.7.6", 30}, {"15.5.0", 35}, {"15.6.1", 35}}
)

// screensFor 返回系统对应的屏幕池
func screensFor(os OSFamily) []weighted[screenSpec] {
	if os == OSMac {
		return macScreens
	}
	return windowsScreens
}

// gpusFor 返回系统对应的显卡池
func gpusFor(os OSFamily) []weighted[gpuProfile] {
	if os == OSMac {
		return macGPUs
	}
	return windowsGPUs
}

// coresFor 返回系统对应的 CPU 核数池
func coresFor(os OSFamily) []weighted[int] {
	if os == OSMac {
		return macCores
	}
	return windowsCores
}

// platformVersionsFor 返回系统对应的 client hints 系统版本池
func platformVersionsFor(os OSFamily) []weighted[string] {
	if os == OSMac {
		return macPlatformVersions
	}
	return windowsPlatformVersions
}

// lookupGPU 在系统的显卡池里按 renderer 精确查找
func lookupGPU(os OSFamily, renderer string) (gpuProfile, bool) {
	for _, g := range gpusFor(os) {
		if g.Value.Renderer == renderer {
			return g.Value, true
		}
	}
	return gpuProfile{}, false
}

// gpuFamily 从 vendor / renderer 串识别显卡厂商，识别不了返回空
func gpuFamily(vendor, renderer string) string {
	s := vendor + " " + renderer
	for _, f := range []string{"NVIDIA", "AMD", "Intel", "Apple"} {
		if strings.Contains(s, f) {
			return f
		}
	}
	return ""
}

// pickGPU 抽取显卡；family 非空且池中有同厂商时只在同厂商里抽（尽量保持账号原来的显卡厂商）
func pickGPU(os OSFamily, family string, seed uint64) gpuProfile {
	pool := gpusFor(os)
	var same []weighted[gpuProfile]
	for _, g := range pool {
		if family != "" && gpuFamily(g.Value.Vendor, g.Value.Renderer) == family {
			same = append(same, g)
		}
	}
	if len(same) > 0 {
		pool = same
	}
	return pickWeighted(pool, seed)
}

// screenSpecFor 在系统屏幕池里查找规格（带上刘海标记）；找不到时返回给定尺寸、不带刘海
func screenSpecFor(os OSFamily, w, h int, dpr float64) (screenSpec, bool) {
	for _, s := range screensFor(os) {
		if s.Value.Width == w && s.Value.Height == h && s.Value.DPR == dpr {
			return s.Value, true
		}
	}
	return screenSpec{Width: w, Height: h, DPR: dpr}, false
}
```

- [ ] **Step 4: 实现身份** `pkg/browser/identity.go`

```go
package browser

import (
	"encoding/json"
	"fmt"
	"hash/fnv"
	"math"
	"regexp"
	"strings"
	_ "time/tzdata" // 时区校验不依赖服务器是否安装 tzdata
)

// OSFamily 账号对外声称的操作系统
type OSFamily string

const (
	OSWindows OSFamily = "windows"
	OSMac     OSFamily = "macos"
)

// browserUIHeight 浏览器标签栏 + 工具栏高度（outerHeight - innerHeight，Chrome 154 在 macOS / Linux 有界面实测）
const browserUIHeight = 87

// osFromUserAgent 由账号绑定的 UA 判定系统：含 Macintosh 为 macOS，其余（含旧数据里的 Linux）按 Windows
func osFromUserAgent(ua string) OSFamily {
	if strings.Contains(ua, "Macintosh") {
		return OSMac
	}
	return OSWindows
}

// platformFor 返回系统对应的 navigator.platform
func platformFor(os OSFamily) string {
	if os == OSMac {
		return "MacIntel"
	}
	return "Win32"
}

// canonicalUserAgent 返回 Chrome 精简格式的 UA：系统段为 Chrome 冻结值，版本只保留主版本
func canonicalUserAgent(os OSFamily, major string) string {
	sys := "Windows NT 10.0; Win64; x64"
	if os == OSMac {
		sys = "Macintosh; Intel Mac OS X 10_15_7"
	}
	return "Mozilla/5.0 (" + sys + ") AppleWebKit/537.36 (KHTML, like Gecko) Chrome/" + major + ".0.0.0 Safari/537.36"
}

var chromeMajorRe = regexp.MustCompile(`Chrome/(\d+)`)

// uaMajorVersion 取 UA 中 Chrome/ 后的主版本号，取不到返回空
func uaMajorVersion(ua string) string {
	if m := chromeMajorRe.FindStringSubmatch(ua); m != nil {
		return m[1]
	}
	return ""
}

// uaBrand client hints 中的一个品牌
type uaBrand struct {
	Brand   string `json:"brand"`
	Version string `json:"version"`
}

// uaMetadata 对应 Emulation.setUserAgentOverride 的 userAgentMetadata
type uaMetadata struct {
	Brands          []uaBrand `json:"brands"`
	FullVersionList []uaBrand `json:"fullVersionList"`
	Platform        string    `json:"platform"`
	PlatformVersion string    `json:"platformVersion"`
	Architecture    string    `json:"architecture"`
	Model           string    `json:"model"`
	Mobile          bool      `json:"mobile"`
	Bitness         string    `json:"bitness"`
	Wow64           bool      `json:"wow64"`
	FormFactors     []string  `json:"formFactors"`
}

// hostInfo 真实浏览器在未覆盖身份的页面上报告的值（由 hostInfoJS 读取），是身份派生的唯一外部输入
type hostInfo struct {
	UserAgent           string     `json:"userAgent"`
	Platform            string     `json:"platform"`
	Languages           []string   `json:"languages"`
	Timezone            string     `json:"timezone"`
	HardwareConcurrency int        `json:"hardwareConcurrency"`
	Screen              [3]float64 `json:"screen"` // width, height, devicePixelRatio
	Brands              []uaBrand  `json:"brands"`
	FullVersionList     []uaBrand  `json:"fullVersionList"`
	UAPlatform          string     `json:"uaPlatform"`
	PlatformVersion     string     `json:"platformVersion"`
	Architecture        string     `json:"architecture"`
	Bitness             string     `json:"bitness"`
	Model               string     `json:"model"`
	Mobile              bool       `json:"mobile"`
	Wow64               bool       `json:"wow64"`
	FormFactors         []string   `json:"formFactors"`
	WebGLVendor         string     `json:"webglVendor"`
	WebGLRenderer       string     `json:"webglRenderer"`
}

// parseHostInfo 解析 hostInfoJS 的返回值并校验必需字段
func parseHostInfo(raw string) (hostInfo, error) {
	var h hostInfo
	if err := json.Unmarshal([]byte(raw), &h); err != nil {
		return h, fmt.Errorf("parse host info: %w", err)
	}
	if h.UserAgent == "" || len(h.FullVersionList) == 0 || len(h.Languages) == 0 {
		return h, fmt.Errorf("host info incomplete: %s", raw)
	}
	return h, nil
}

// fullVersion 返回真实浏览器完整版本（优先 Google Chrome 品牌，其次 Chromium）
func (h hostInfo) fullVersion() string {
	for _, want := range []string{"Google Chrome", "Chromium"} {
		for _, b := range h.FullVersionList {
			if b.Brand == want {
				return b.Version
			}
		}
	}
	return h.FullVersionList[0].Version
}

// majorVersion 返回真实浏览器主版本号
func (h hostInfo) majorVersion() string {
	return strings.SplitN(h.fullVersion(), ".", 2)[0]
}

// screenGeometry 身份的屏幕与窗口几何（CSS 像素）；Width 为 0 表示不调整
type screenGeometry struct {
	Width, Height, AvailWidth, AvailHeight, AvailTop int
	DPR                                              float64
	OuterWidth, OuterHeight                          int // 浏览器窗口（最大化 = 可用区域）
	InnerWidth, InnerHeight                          int // 视口 = 窗口减去浏览器 UI
}

// geometryFor 由屏幕规格推算可用区域和最大化窗口：Windows 扣任务栏（Win10 40、Win11 48 物理像素），Mac 扣菜单栏（刘海机型 38，其余 25）
func geometryFor(os OSFamily, s screenSpec, platformVersion string) screenGeometry {
	g := screenGeometry{Width: s.Width, Height: s.Height, DPR: s.DPR, AvailWidth: s.Width}
	if os == OSMac {
		bar := 25
		if s.Notch {
			bar = 38
		}
		g.AvailTop = bar
		g.AvailHeight = s.Height - bar
	} else {
		taskbar := 48.0
		if platformVersion == "10.0.0" {
			taskbar = 40
		}
		g.AvailHeight = s.Height - int(math.Round(taskbar/s.DPR))
	}
	g.OuterWidth, g.OuterHeight = g.AvailWidth, g.AvailHeight
	g.InnerWidth, g.InnerHeight = g.AvailWidth, g.AvailHeight-browserUIHeight
	return g
}

// Identity 一个账号对外呈现的完整浏览器身份，由 CDP 下发到所有目标
type Identity struct {
	OS                  OSFamily
	UserAgent           string
	Platform            string // navigator.platform
	AcceptLanguage      string // 例如 "zh-CN,zh"，Chrome 自动补 q 值
	Languages           []string
	Timezone            string
	HardwareConcurrency int
	Metadata            uaMetadata
	Screen              screenGeometry
	GPU                 *gpuProfile // nil 表示不改写 WebGL
	NoiseSeed           uint32      // 0 表示不加 canvas / 音频 / readPixels 噪声
}

// Locale 返回 Intl 使用的 locale
func (id *Identity) Locale() string { return id.Languages[0] }

// noiseSeed 账号固定的非零噪声种子
func noiseSeed(userID string) uint32 {
	h := fnv.New32a()
	h.Write([]byte(userID + "\x00noise"))
	return h.Sum32() | 1
}

// identityFromConfig 由规范化后的账号指纹与真实浏览器派生身份：系统、屏幕、显卡等取自指纹，版本与品牌取自真实浏览器
func identityFromConfig(cfg *FingerprintConfig, host hostInfo) (*Identity, error) {
	os := osFromUserAgent(cfg.Browser.UserAgent)
	gpu, ok := lookupGPU(os, cfg.WebGL.Renderer)
	if !ok {
		return nil, fmt.Errorf("fingerprint %s: renderer %q is not in the %s pool (config not normalized)", cfg.UserID, cfg.WebGL.Renderer, os)
	}
	if len(cfg.Browser.Languages) == 0 {
		return nil, fmt.Errorf("fingerprint %s: no languages", cfg.UserID)
	}
	meta := uaMetadata{
		Brands:          host.Brands,
		FullVersionList: host.FullVersionList,
		PlatformVersion: cfg.Browser.PlatformVersion,
		Bitness:         "64",
		FormFactors:     []string{"Desktop"},
		Platform:        "Windows",
		Architecture:    "x86",
	}
	if os == OSMac {
		meta.Platform, meta.Architecture = "macOS", "arm"
	}
	spec, _ := screenSpecFor(os, cfg.Screen.Width, cfg.Screen.Height, cfg.Screen.DevicePixelRatio)
	return &Identity{
		OS:                  os,
		UserAgent:           canonicalUserAgent(os, host.majorVersion()),
		Platform:            platformFor(os),
		AcceptLanguage:      strings.Join(cfg.Browser.Languages, ","),
		Languages:           cfg.Browser.Languages,
		Timezone:            cfg.Timezone.Timezone,
		HardwareConcurrency: cfg.Browser.HardwareConcurrency,
		Metadata:            meta,
		Screen:              geometryFor(os, spec, cfg.Browser.PlatformVersion),
		GPU:                 &gpu,
		NoiseSeed:           noiseSeed(cfg.UserID),
	}, nil
}

// hostIdentity 不设指纹时的身份：沿用真实浏览器的值，只去掉无界面痕迹
// （UA 中的 HeadlessChrome、无界面默认 800×600 屏幕、SwiftShader 显卡标识）
func hostIdentity(host hostInfo) *Identity {
	formFactors := host.FormFactors
	if len(formFactors) == 0 {
		formFactors = []string{"Desktop"}
	}
	id := &Identity{
		OS:                  OSWindows, // 仅用于几何规则；Linux 宿主按 Windows 规则扣底部面板
		UserAgent:           strings.Replace(host.UserAgent, "HeadlessChrome/", "Chrome/", 1),
		Platform:            host.Platform,
		AcceptLanguage:      strings.Join(host.Languages, ","),
		Languages:           host.Languages,
		Timezone:            host.Timezone,
		HardwareConcurrency: host.HardwareConcurrency,
		Metadata: uaMetadata{
			Brands: host.Brands, FullVersionList: host.FullVersionList, Platform: host.UAPlatform,
			PlatformVersion: host.PlatformVersion, Architecture: host.Architecture, Model: host.Model,
			Mobile: host.Mobile, Bitness: host.Bitness, Wow64: host.Wow64, FormFactors: formFactors,
		},
	}
	if host.UAPlatform == "macOS" {
		id.OS = OSMac
	}
	if host.Screen[0] == 800 && host.Screen[1] == 600 {
		id.Screen = geometryFor(id.OS, screenSpec{Width: 1920, Height: 1080, DPR: 1}, host.PlatformVersion)
	}
	if strings.Contains(host.WebGLRenderer, "SwiftShader") {
		g := hostGPU(host.UAPlatform)
		id.GPU = &g
	}
	return id
}

// hostGPU 默认路径遮挡 SwiftShader 时使用的、与宿主系统相符的显卡标识（不改能力参数）
func hostGPU(uaPlatform string) gpuProfile {
	switch uaPlatform {
	case "Windows":
		g := windowsGPUs[0].Value
		return gpuProfile{Vendor: g.Vendor, Renderer: g.Renderer}
	case "macOS":
		g := macGPUs[0].Value
		return gpuProfile{Vendor: g.Vendor, Renderer: g.Renderer}
	default:
		return linuxHostGPU
	}
}
```

- [ ] **Step 5: 运行测试**

Run: `go vet ./pkg/browser/ && go test -race -count=1 -run 'TestOSFromUserAgent|TestCanonicalUserAgent|TestIdentityFromConfig|TestHostIdentity' -v ./pkg/browser/`
Expected: 全部 PASS

- [ ] **Step 6: 提交**

```bash
git add pkg/browser/identity.go pkg/browser/identity_pools.go pkg/browser/identity_test.go pkg/browser/fingerprint_config.go
git commit -m "feat(identity): derive a coherent per-account browser identity

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
```

---
### Task 3: 指纹规范化、生成器与管理器

旧指纹文件（Linux UA、`x64 10.0`、`(Build N)` 显卡、语言 / 时区不一致等）在加载时规范化并回写；新账号按绑定 UA 选系统，取值来自 Task 2 的池。只修正身份相关字段，其余字段（TLS、HTTP2、电池等）原样保留。

**Files:**
- Modify: `pkg/browser/fingerprint_config.go`（新增 `SchemaVersion`、`Normalize` 及辅助函数；`GenerateFingerprint` 改为调用 `generateFingerprintFor`；删除旧的随机生成函数）
- Modify: `pkg/browser/user_fingerprint_manager.go`（`GetOrCreateUserFingerprint`）
- Create: `pkg/browser/fingerprint_normalize_test.go`

**Interfaces:**
- Consumes: Task 2 的 `osFromUserAgent`、`canonicalUserAgent`、`uaMajorVersion`、`platformFor`、`pickWeighted`、`identitySeed`、`screensFor`、`coresFor`、`platformVersionsFor`、`lookupGPU`、`pickGPU`、`gpuFamily`、`screenSpecFor`、`geometryFor`、`identityFromConfig`；现有 `getTimezoneForLanguage(lang) (string, int)`
- Produces:
  - `const fingerprintSchemaVersion = 2`；`FingerprintConfig.SchemaVersion int \`json:"schema_version,omitempty"\``
  - `func (config *FingerprintConfig) Normalize() bool`
  - `func generateFingerprintFor(userID string, os OSFamily) *FingerprintConfig`

- [ ] **Step 1: 写失败的测试** `pkg/browser/fingerprint_normalize_test.go`

```go
package browser

import (
	"encoding/json"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

// legacyConfig 构造一份旧生成器风格的指纹
func legacyConfig(userID, ua, vendor, renderer, lang string, langs []string, tz string, w, h int, dpr float64, cores int) *FingerprintConfig {
	c := &FingerprintConfig{UserID: userID}
	c.Browser.UserAgent = ua
	c.Browser.Platform = "Linux x86_64"
	c.Browser.Language, c.Browser.Languages = lang, langs
	c.Browser.HardwareConcurrency = cores
	c.Timezone.Timezone = tz
	c.Screen = ScreenConfig{Width: w, Height: h, DevicePixelRatio: dpr}
	c.WebGL.Vendor, c.WebGL.Renderer = vendor, renderer
	c.TLSConfig.JA3 = "keep-me" // 与身份无关的字段必须原样保留
	return c
}

func TestNormalizeLegacyFingerprint(t *testing.T) {
	cases := []struct {
		name  string
		cfg   *FingerprintConfig
		check func(t *testing.T, c *FingerprintConfig)
	}{
		{"linux ua becomes windows", legacyConfig("u1",
			"Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/111.0.5563.146 Safari/537.36",
			"Google Inc. (NVIDIA)", "ANGLE (NVIDIA, NVIDIA GeForce RTX 3070 Direct3D11 vs_5_0 ps_5_0, D3D11)",
			"ja-JP", []string{"es-ES", "es", "en"}, "America/Sao_Paulo", 3840, 1600, 1, 4),
			func(t *testing.T, c *FingerprintConfig) {
				if c.Browser.UserAgent != canonicalUserAgent(OSWindows, "111") {
					t.Errorf("UA = %q", c.Browser.UserAgent)
				}
				if c.Browser.Platform != "Win32" {
					t.Errorf("Platform = %q", c.Browser.Platform)
				}
				if !slices.Equal(c.Browser.Languages, []string{"ja-JP", "ja"}) {
					t.Errorf("Languages = %v", c.Browser.Languages)
				}
				if c.Timezone.Timezone != "America/Sao_Paulo" {
					t.Errorf("valid timezone changed to %q", c.Timezone.Timezone)
				}
				if _, ok := screenSpecFor(OSWindows, c.Screen.Width, c.Screen.Height, c.Screen.DevicePixelRatio); !ok {
					t.Errorf("implausible 3840x1600 not re-picked from pool: %+v", c.Screen)
				}
				if gpuFamily(c.WebGL.Vendor, c.WebGL.Renderer) != "NVIDIA" {
					t.Errorf("GPU vendor family not kept: %q", c.WebGL.Renderer)
				}
				if c.Browser.HardwareConcurrency != 4 {
					t.Errorf("valid core count changed to %d", c.Browser.HardwareConcurrency)
				}
			}},
		{"broken windows ua fixed, plausible screen kept", legacyConfig("u2",
			"Mozilla/5.0 (Windows NT 10.0; Win64; x64 10.0) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/128.0.6613.138 Safari/537.36",
			"Google Inc. (AMD) ", "ANGLE (AMD, AMD Radeon(TM) Graphics Direct3D11 vs_5_0 ps_5_0, D3D11) (Build 26453)",
			"zh-CN", []string{"zh-CN", "zh"}, "Asia/Shanghai", 1920, 1080, 1, 8),
			func(t *testing.T, c *FingerprintConfig) {
				if c.Browser.UserAgent != canonicalUserAgent(OSWindows, "128") {
					t.Errorf("UA = %q", c.Browser.UserAgent)
				}
				if c.Screen.Width != 1920 || c.Screen.Height != 1080 || c.Screen.DevicePixelRatio != 1 {
					t.Errorf("plausible screen changed: %+v", c.Screen)
				}
				if c.Screen.AvailHeight >= c.Screen.Height {
					t.Errorf("avail height %d not below screen height", c.Screen.AvailHeight)
				}
				if strings.Contains(c.WebGL.Renderer, "Build") || gpuFamily(c.WebGL.Vendor, c.WebGL.Renderer) != "AMD" {
					t.Errorf("renderer = %q, want an AMD pool entry", c.WebGL.Renderer)
				}
			}},
		{"mac keeps mac, gpu and cores follow mac pools", legacyConfig("u3",
			"Mozilla/5.0 (Macintosh; Intel Mac OS X 13_5_2) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.6099.234 Safari/537.36",
			"Google Inc. (AMD)", "ANGLE (AMD, AMD Radeon RX 6600 XT Direct3D11 vs_5_0 ps_5_0, D3D11)",
			"en-US", []string{"en-US", "en"}, "America/New_York", 1440, 900, 2, 6),
			func(t *testing.T, c *FingerprintConfig) {
				if c.Browser.UserAgent != canonicalUserAgent(OSMac, "120") || c.Browser.Platform != "MacIntel" {
					t.Errorf("mac UA/platform = %q / %q", c.Browser.UserAgent, c.Browser.Platform)
				}
				if _, ok := lookupGPU(OSMac, c.WebGL.Renderer); !ok {
					t.Errorf("mac account has non-mac GPU %q", c.WebGL.Renderer)
				}
				if c.Browser.HardwareConcurrency != 8 && c.Browser.HardwareConcurrency != 10 && c.Browser.HardwareConcurrency != 12 {
					t.Errorf("mac cores = %d", c.Browser.HardwareConcurrency)
				}
			}},
		{"utc timezone derived from language", legacyConfig("u4",
			"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36",
			"", "", "zh-CN", nil, "UTC", 1366, 768, 1, 8),
			func(t *testing.T, c *FingerprintConfig) {
				if c.Timezone.Timezone != "Asia/Shanghai" {
					t.Errorf("Timezone = %q, want Asia/Shanghai", c.Timezone.Timezone)
				}
				if !slices.Equal(c.Browser.Languages, []string{"zh-CN", "zh"}) {
					t.Errorf("Languages = %v", c.Browser.Languages)
				}
			}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if !tc.cfg.Normalize() {
				t.Fatal("Normalize reported no change on a legacy config")
			}
			tc.check(t, tc.cfg)
			if tc.cfg.SchemaVersion != fingerprintSchemaVersion {
				t.Errorf("SchemaVersion = %d", tc.cfg.SchemaVersion)
			}
			if tc.cfg.TLSConfig.JA3 != "keep-me" {
				t.Error("unrelated field was modified")
			}
			if _, err := identityFromConfig(tc.cfg, testHost()); err != nil {
				t.Errorf("normalized config cannot build an identity: %v", err)
			}
			first, _ := json.Marshal(tc.cfg)
			if tc.cfg.Normalize() {
				t.Error("Normalize is not idempotent")
			}
			if second, _ := json.Marshal(tc.cfg); string(first) != string(second) {
				t.Error("second Normalize changed the config")
			}
		})
	}
}

// TestGenerateFingerprintFor 新指纹按系统生成、按用户 ID 确定，且能直接派生身份
func TestGenerateFingerprintFor(t *testing.T) {
	for _, osf := range []OSFamily{OSWindows, OSMac} {
		a, b := generateFingerprintFor("gen-user", osf), generateFingerprintFor("gen-user", osf)
		ja, _ := json.Marshal(a)
		jb, _ := json.Marshal(b)
		if string(ja) != string(jb) {
			t.Errorf("%s: generation is not deterministic per user", osf)
		}
		if osFromUserAgent(a.Browser.UserAgent) != osf {
			t.Errorf("%s: generated UA %q", osf, a.Browser.UserAgent)
		}
		if _, err := identityFromConfig(a, testHost()); err != nil {
			t.Errorf("%s: %v", osf, err)
		}
	}
}

// TestFingerprintManagerNormalizesAndBindsUA 新账号按绑定 UA 选系统；旧文件加载时规范化并回写；损坏文件返回错误而不是被覆盖
func TestFingerprintManagerNormalizesAndBindsUA(t *testing.T) {
	dir := t.TempDir()
	m, err := NewUserFingerprintManager(dir)
	if err != nil {
		t.Fatal(err)
	}

	cfg, err := m.GetOrCreateUserFingerprint("new-mac", &FingerprintInitParams{
		UserAgent: "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36",
		Language:  "zh-CN",
	})
	if err != nil {
		t.Fatal(err)
	}
	if osFromUserAgent(cfg.Browser.UserAgent) != OSMac || cfg.Browser.Platform != "MacIntel" {
		t.Errorf("new account did not follow bound Mac UA: %q / %q", cfg.Browser.UserAgent, cfg.Browser.Platform)
	}

	legacy := legacyConfig("old-linux",
		"Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/111.0.5563.146 Safari/537.36",
		"Google Inc. (NVIDIA)", "NVIDIA GeForce RTX 3060/PCIe/SSE2", "zh-CN", []string{"zh-CN", "zh"}, "Asia/Shanghai", 1920, 1080, 1, 8)
	data, _ := json.Marshal(legacy)
	path := filepath.Join(dir, "old-linux.json")
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatal(err)
	}
	got, err := m.GetOrCreateUserFingerprint("old-linux", nil)
	if err != nil {
		t.Fatal(err)
	}
	if got.Browser.Platform != "Win32" {
		t.Errorf("legacy Linux account not normalized: %q", got.Browser.Platform)
	}
	reread, err := LoadFingerprintConfigFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if reread.SchemaVersion != fingerprintSchemaVersion || reread.Browser.Platform != "Win32" {
		t.Errorf("normalized config not written back: schema %d platform %q", reread.SchemaVersion, reread.Browser.Platform)
	}

	if err := os.WriteFile(filepath.Join(dir, "broken.json"), []byte("{not json"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := m.GetOrCreateUserFingerprint("broken", nil); err == nil {
		t.Error("corrupt fingerprint file must return an error, not be regenerated")
	}
	if b, _ := os.ReadFile(filepath.Join(dir, "broken.json")); string(b) != "{not json" {
		t.Error("corrupt fingerprint file was overwritten")
	}
}
```

- [ ] **Step 2: 运行确认失败**

Run: `go test -count=1 -run 'TestNormalizeLegacyFingerprint|TestGenerateFingerprintFor|TestFingerprintManagerNormalizesAndBindsUA' ./pkg/browser/`
Expected: 编译失败（`undefined: fingerprintSchemaVersion`、`LoadFingerprintConfigFile` 等）

- [ ] **Step 3: 实现规范化**（`pkg/browser/fingerprint_config.go`）

在 `FingerprintConfig` 结构体第一行 `UserID` 之后加：

```go
	// SchemaVersion 文件格式版本，见 fingerprintSchemaVersion
	SchemaVersion int `json:"schema_version,omitempty"`
```

在文件末尾追加：

```go
// fingerprintSchemaVersion 当前指纹格式：2 = 自洽的 Windows / macOS 身份（UA 只用于判定系统，版本运行时跟随真实浏览器）
const fingerprintSchemaVersion = 2

// Normalize 把指纹规范化为自洽的 Windows / macOS 身份，返回是否有改动（调用方据此回写文件）。
// 只修正身份相关字段，其余字段原样保留；对已规范化的配置幂等
func (config *FingerprintConfig) Normalize() bool {
	before, _ := json.Marshal(config)
	osf := osFromUserAgent(config.Browser.UserAgent)
	seed := func(purpose string) uint64 { return identitySeed(config.UserID, purpose) }

	major := uaMajorVersion(config.Browser.UserAgent)
	if major == "" {
		major = "120"
	}
	config.Browser.UserAgent = canonicalUserAgent(osf, major)
	config.Browser.Platform = platformFor(osf)
	config.Browser.Vendor = "Google Inc."
	if !containsValue(platformVersionsFor(osf), config.Browser.PlatformVersion) {
		config.Browser.PlatformVersion = pickWeighted(platformVersionsFor(osf), seed("platform-version"))
	}

	// 语言：languages 首项必须等于 language
	if config.Browser.Language == "" {
		config.Browser.Language = "zh-CN"
	}
	if len(config.Browser.Languages) == 0 || config.Browser.Languages[0] != config.Browser.Language {
		config.Browser.Languages = defaultLanguages(config.Browser.Language)
	}

	// 时区：必须是可加载的地区时区（UTC 等不像真实用户），否则由语言推导
	if !plausibleTimezone(config.Timezone.Timezone) {
		config.Timezone.Timezone, _ = getTimezoneForLanguage(config.Browser.Language)
	}

	// 屏幕：不像该系统的真实设备时按用户 ID 重选；可用区域按系统规则重算
	if !plausibleScreen(osf, config.Screen) {
		s := pickWeighted(screensFor(osf), seed("screen"))
		config.Screen.Width, config.Screen.Height, config.Screen.DevicePixelRatio = s.Width, s.Height, s.DPR
	}
	spec, _ := screenSpecFor(osf, config.Screen.Width, config.Screen.Height, config.Screen.DevicePixelRatio)
	g := geometryFor(osf, spec, config.Browser.PlatformVersion)
	config.Screen.AvailWidth, config.Screen.AvailHeight = g.AvailWidth, g.AvailHeight
	config.Screen.ColorDepth, config.Screen.PixelDepth = 24, 24

	if !containsValue(coresFor(osf), config.Browser.HardwareConcurrency) {
		config.Browser.HardwareConcurrency = pickWeighted(coresFor(osf), seed("cores"))
	}

	// 显卡：不在池中（含 "(Build N)" 等不可能的串）时尽量在同厂商里重选，能力参数与之对齐
	gpu, ok := lookupGPU(osf, config.WebGL.Renderer)
	if !ok {
		gpu = pickGPU(osf, gpuFamily(config.WebGL.Vendor, config.WebGL.Renderer), seed("gpu"))
	}
	config.WebGL.Vendor, config.WebGL.Renderer = gpu.Vendor, gpu.Renderer
	config.WebGL.Version = "WebGL 1.0 (OpenGL ES 2.0 Chromium)"
	config.WebGL.ShadingLanguageVersion = "WebGL GLSL ES 1.0 (OpenGL ES GLSL ES 1.0 Chromium)"
	config.WebGL.MaxTextureSize, config.WebGL.MaxRenderbufferSize = gpu.Caps.MaxTextureSize, gpu.Caps.MaxRenderbufferSize

	config.SchemaVersion = fingerprintSchemaVersion
	after, _ := json.Marshal(config)
	return !bytes.Equal(before, after)
}

// containsValue 池中是否有该取值
func containsValue[T comparable](items []weighted[T], v T) bool {
	return slices.ContainsFunc(items, func(w weighted[T]) bool { return w.Value == v })
}

// defaultLanguages 由主语言推导 navigator.languages：带地区时追加基础语言（zh-CN → zh-CN, zh）
func defaultLanguages(lang string) []string {
	if base, _, ok := strings.Cut(lang, "-"); ok {
		return []string{lang, base}
	}
	return []string{lang}
}

// plausibleTimezone 是否为可加载的地区时区（排除 UTC、Etc/* 这类不像真实用户的值）
func plausibleTimezone(tz string) bool {
	if !strings.Contains(tz, "/") || strings.HasPrefix(tz, "Etc/") {
		return false
	}
	_, err := time.LoadLocation(tz)
	return err == nil
}

// plausibleScreen 屏幕是否像该系统的真实设备：池中的规格直接通过，否则按尺寸与 DPR 范围判断
func plausibleScreen(osf OSFamily, s ScreenConfig) bool {
	if _, ok := screenSpecFor(osf, s.Width, s.Height, s.DevicePixelRatio); ok {
		return true
	}
	if s.Width <= s.Height || s.Width < 1280 || s.Width > 2560 || s.Height < 720 || s.Height > 1600 {
		return false
	}
	if osf == OSMac {
		return s.DevicePixelRatio == 1 || s.DevicePixelRatio == 2
	}
	return s.DevicePixelRatio == 1 || s.DevicePixelRatio == 1.25 || s.DevicePixelRatio == 1.5
}

// LoadFingerprintConfigFile 从 JSON 文件读取指纹（不做规范化）
func LoadFingerprintConfigFile(path string) (*FingerprintConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read fingerprint %s: %w", path, err)
	}
	var c FingerprintConfig
	if err := json.Unmarshal(data, &c); err != nil {
		return nil, fmt.Errorf("parse fingerprint %s: %w", path, err)
	}
	return &c, nil
}

// generateFingerprintFor 生成指定系统的新指纹；身份字段由 Normalize 按用户 ID 确定性补齐
func generateFingerprintFor(userID string, osf OSFamily) *FingerprintConfig {
	c := &FingerprintConfig{UserID: userID}
	c.Browser.UserAgent = canonicalUserAgent(osf, "120")
	c.Browser.Language = "zh-CN"
	c.Browser.CookieEnabled = true
	c.Normalize()
	return c
}
```

`fingerprint_config.go` 的 import 补上 `bytes`、`os`、`slices`、`time`（已有 `encoding/json`、`fmt`、`strings`）。

`GenerateFingerprint` 方法体替换为：

```go
// GenerateFingerprint 为用户生成指纹：默认 Windows 身份，取值按用户 ID 确定性抽取
func (fg *FingerprintGenerator) GenerateFingerprint(userID string) *FingerprintConfig {
	return generateFingerprintFor(userID, OSWindows)
}
```

删除旧的随机生成实现（不再被调用，均为未导出方法）：`hashUserID`、`generateScreenConfig`、`generateBrowserConfig`、`generateSystemConfig`、`generateWebGLConfig`、`generateAudioConfig`、`generateNetworkConfig`、`generateTimezoneConfig`、`generateCanvasConfig`、`generateFontsConfig`、`generatePluginsConfig`、`generateBatteryConfig`、`generateMediaDevicesConfig`、`generateDeviceID`、`generateTLSConfig`、`generateHTTP2Config`，以及包级函数 `min`（之后使用内置 `min`）。删除后按 `go vet` 清理不再使用的 import（`crypto/md5`、`math/rand` 若无其他引用）。

- [ ] **Step 4: 改造管理器**（`pkg/browser/user_fingerprint_manager.go`）

`GetOrCreateUserFingerprint` 中从 `// 尝试从文件加载` 到 `// 保存到文件` 之前的代码替换为：

```go
	// 从文件加载；旧格式或不自洽的指纹在这里规范化并回写。损坏的文件直接报错，绝不覆盖
	configPath := ufm.getUserConfigPath(userID)
	if _, err := os.Stat(configPath); err == nil {
		config, err := LoadFingerprintConfigFile(configPath)
		if err != nil {
			return nil, fmt.Errorf("load fingerprint %s: %w", userID, err)
		}
		if config.Normalize() {
			if err := ufm.saveConfigToFile(config, configPath); err != nil {
				return nil, fmt.Errorf("save normalized fingerprint %s: %w", userID, err)
			}
		}
		ufm.mutex.Lock()
		ufm.cache[userID] = config
		ufm.mutex.Unlock()
		return config, nil
	}

	// 新账号：系统由绑定的 UA 决定，未提供 UA 时为 Windows
	osFamily := OSWindows
	if initParams != nil && initParams.UserAgent != "" {
		osFamily = osFromUserAgent(initParams.UserAgent)
	}
	config := generateFingerprintFor(userID, osFamily)
	if initParams != nil {
		if initParams.Width > 0 {
			config.Screen.Width = initParams.Width
		}
		if initParams.Height > 0 {
			config.Screen.Height = initParams.Height
		}
		if initParams.UserAgent != "" {
			config.Browser.UserAgent = initParams.UserAgent
		}
		if initParams.Language != "" {
			config.Browser.Language = initParams.Language
			config.Browser.Languages = nil
		}
		if len(initParams.Languages) > 0 {
			config.Browser.Languages = initParams.Languages
		}
		if initParams.Timezone != "" {
			config.Timezone.Timezone = initParams.Timezone
		}
		config.Normalize()
	}

```

（`TimezoneOffset` 不再使用：偏移量由浏览器按时区计算。在 `types.go` 的 `TimezoneOffset` 字段注释末尾加 `Deprecated: 不再生效，偏移量由浏览器按 Timezone 计算。`）

- [ ] **Step 5: 运行测试**

Run: `go vet ./pkg/... ./internal/... && go test -race -count=1 -run 'TestNormalizeLegacyFingerprint|TestGenerateFingerprintFor|TestFingerprintManagerNormalizesAndBindsUA|TestIdentityFromConfig|TestHostIdentity' -v ./pkg/browser/`
Expected: 全部 PASS

另外确认依赖生成器的 demo 仍能编译：`for f in cmd/example/fingerprint_demo.go cmd/example/fingerprint_persist_demo.go; do go build -o /dev/null $f; done; go build -o /dev/null ./cmd/fingerprint_stats ./cmd/fingerprint_collector`
Expected: 无输出（编译通过）

- [ ] **Step 6: 提交**

```bash
git add pkg/browser/fingerprint_config.go pkg/browser/user_fingerprint_manager.go pkg/browser/fingerprint_normalize_test.go pkg/browser/types.go
git commit -m "feat(fingerprint): normalize stored fingerprints into coherent Windows/macOS identities

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
```

---
### Task 4: 最小 JS 层（原生伪装）

生成注入脚本：页面 / iframe 版本和 Worker 版本。所有改写都经由"原生伪装工具"：改在原型（或属性真正所在的对象）上，替换函数保持原名、原参数个数，`toString` 返回原生文本（统一的 `Function.prototype.toString` 代理，代理自身也显示原生），不留任何全局或原型标记。噪声只依赖账号种子与位置，同一账号同一内容每次结果相同。

**Files:**
- Create: `pkg/browser/identity_js.go`
- Create: `pkg/browser/identity_js_test.go`

**Interfaces:**
- Consumes: Task 2 的 `Identity`、`screenGeometry`、`gpuProfile`、`webglCaps`、`geometryFor`、`windowsGPUs`
- Produces: `func identityScript(id *Identity, worker bool) string`

- [ ] **Step 1: 写失败的测试** `pkg/browser/identity_js_test.go`

```go
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
		{"avail area", `[screen.availWidth, screen.availHeight, screen.availTop].join()`, "1536,826,0"},
		{"webgl profile", `(() => { const g = document.createElement('canvas').getContext('webgl'); g.getExtension('WEBGL_debug_renderer_info');
		  return [g.getParameter(37445), g.getParameter(37446), g.getParameter(g.MAX_TEXTURE_SIZE), g.getParameter(g.MAX_VIEWPORT_DIMS) instanceof Int32Array, g.getParameter(g.MAX_VARYING_VECTORS)].join('|'); })()`,
			gpu.Vendor + "|" + gpu.Renderer + "|16384|true|30"},
		{"canvas noise deterministic", `(() => { const a = ` + draw + `; const b = ` + draw + `; return a === b; })()`, true},
		{"blank canvas untouched", blank, baseBlank},
		{"untrusted event native", `new MouseEvent('click', {clientX: 10, screenX: 5}).screenX`, float64(5)},
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
	  Object.getOwnPropertyDescriptor(WorkerNavigator.prototype, 'platform').get.toString()].join('|'))`)
	got := evalValue(t, c, `new Promise(r => { const w = new Worker(URL.createObjectURL(new Blob([`+string(ws)+`]))); w.onmessage = e => r(e.data); })`)
	if want := winUA + "|" + strings.TrimPrefix(winUA, "Mozilla/") + "|Win32|6|ja-JP,ja|function get platform() { [native code] }"; got != want {
		t.Errorf("worker navigator = %v, want %v", got, want)
	}
}
```

- [ ] **Step 2: 运行确认失败**

Run: `go test -count=1 -run TestIdentityScript ./pkg/browser/`
Expected: 编译失败，`undefined: identityScript`

- [ ] **Step 3: 实现** `pkg/browser/identity_js.go`

```go
package browser

import (
	"encoding/json"
)

// jsIdentity 注入脚本使用的身份子集，以 JSON 嵌入脚本
type jsIdentity struct {
	UserAgent           string    `json:"userAgent"`
	Platform            string    `json:"platform"`
	Languages           []string  `json:"languages"`
	HardwareConcurrency int       `json:"hardwareConcurrency"`
	Screen              *jsScreen `json:"screen,omitempty"`
	WebGL               *jsWebGL  `json:"webgl,omitempty"`
	Seed                uint32    `json:"seed"`
}

// jsScreen 屏幕与可用区域
type jsScreen struct {
	Width       int     `json:"width"`
	Height      int     `json:"height"`
	AvailWidth  int     `json:"availWidth"`
	AvailHeight int     `json:"availHeight"`
	AvailTop    int     `json:"availTop"`
	DPR         float64 `json:"dpr"`
}

// jsWebGL 显卡标识与 getParameter 覆盖表（键为十进制 GLenum，值为数值或 [a, b]）
type jsWebGL struct {
	Vendor   string         `json:"vendor"`
	Renderer string         `json:"renderer"`
	Params   map[string]any `json:"params,omitempty"`
}

// webglParams 能力参数转为 getParameter 覆盖表
func webglParams(c *webglCaps) map[string]any {
	if c == nil {
		return nil
	}
	return map[string]any{
		"3379":  c.MaxTextureSize,                        // MAX_TEXTURE_SIZE
		"34076": c.MaxCubeMapTextureSize,                 // MAX_CUBE_MAP_TEXTURE_SIZE
		"34024": c.MaxRenderbufferSize,                   // MAX_RENDERBUFFER_SIZE
		"3386":  c.MaxViewportDims[:],                    // MAX_VIEWPORT_DIMS（Int32Array）
		"34921": c.MaxVertexAttribs,                      // MAX_VERTEX_ATTRIBS
		"36347": c.MaxVertexUniformVectors,               // MAX_VERTEX_UNIFORM_VECTORS
		"36349": c.MaxFragmentUniformVectors,             // MAX_FRAGMENT_UNIFORM_VECTORS
		"36348": c.MaxVaryingVectors,                     // MAX_VARYING_VECTORS
		"34930": c.MaxTextureImageUnits,                  // MAX_TEXTURE_IMAGE_UNITS
		"35660": c.MaxVertexTextureImageUnits,            // MAX_VERTEX_TEXTURE_IMAGE_UNITS
		"35661": c.MaxCombinedTextureImageUnits,          // MAX_COMBINED_TEXTURE_IMAGE_UNITS
		"33901": c.AliasedPointSizeRange[:],              // ALIASED_POINT_SIZE_RANGE（Float32Array）
		"33902": c.AliasedLineWidthRange[:],              // ALIASED_LINE_WIDTH_RANGE（Float32Array）
	}
}

// identityScript 生成注入脚本；worker 为 true 时生成 Worker 版本（补 WorkerNavigator，不含 DOM / Web Audio 部分）
func identityScript(id *Identity, worker bool) string {
	cfg := jsIdentity{UserAgent: id.UserAgent, Platform: id.Platform, Languages: id.Languages, HardwareConcurrency: id.HardwareConcurrency, Seed: id.NoiseSeed}
	if id.Screen.Width > 0 {
		s := id.Screen
		cfg.Screen = &jsScreen{Width: s.Width, Height: s.Height, AvailWidth: s.AvailWidth, AvailHeight: s.AvailHeight, AvailTop: s.AvailTop, DPR: s.DPR}
	}
	if id.GPU != nil {
		cfg.WebGL = &jsWebGL{Vendor: id.GPU.Vendor, Renderer: id.GPU.Renderer, Params: webglParams(id.GPU.Caps)}
	}
	data, _ := json.Marshal(cfg) // 只含基本类型，不会失败
	body := jsWindowPart
	if worker {
		body = jsWorkerPart
	}
	return "(() => {\n'use strict';\nconst C = " + string(data) + ";\n" + jsPrelude + jsCommonPart + body + "\n})();"
}

// jsPrelude 原生伪装工具与确定性噪声
const jsPrelude = `
const G = globalThis;
const nativeToString = Function.prototype.toString;
const masks = new WeakMap();
const toStringProxy = new Proxy(nativeToString, {
  apply(target, self, args) {
    return masks.has(self) ? masks.get(self) : Reflect.apply(target, self, args);
  },
});
masks.set(toStringProxy, Reflect.apply(nativeToString, nativeToString, []));
Object.defineProperty(Function.prototype, 'toString', {value: toStringProxy});
const disguise = (fake, original) => {
  masks.set(fake, Reflect.apply(nativeToString, original, []));
  Object.defineProperty(fake, 'name', {value: original.name});
  Object.defineProperty(fake, 'length', {value: original.length});
  return fake;
};
const owner = (obj, key) => {
  while (obj && !Object.getOwnPropertyDescriptor(obj, key)) obj = Object.getPrototypeOf(obj);
  return obj;
};
const patchMethod = (obj, key, impl) => {
  const target = owner(obj, key);
  const desc = target && Object.getOwnPropertyDescriptor(target, key);
  if (!desc || typeof desc.value !== 'function') return;
  const original = desc.value;
  const fake = {[key](...args) { return impl(original, this, args); }}[key];
  Object.defineProperty(target, key, {value: disguise(fake, original)});
};
const patchGetter = (obj, key, impl) => {
  const target = owner(obj, key);
  const desc = target && Object.getOwnPropertyDescriptor(target, key);
  if (!desc || typeof desc.get !== 'function') return;
  const original = desc.get;
  const fake = Object.getOwnPropertyDescriptor({get [key]() { return impl(original, this); }}, key).get;
  Object.defineProperty(target, key, {get: disguise(fake, original)});
};
const hash = (n) => {
  let h = Math.imul(n ^ C.seed, 2654435761) >>> 0;
  h ^= h >>> 15;
  h = Math.imul(h, 2246822519) >>> 0;
  h ^= h >>> 13;
  return h;
};
const noisePixels = (data, width, x0, y0) => {
  for (let i = 0; i < data.length; i += 4) {
    if (data[i + 3] === 0) continue;
    const p = i >> 2;
    const h = hash((y0 + Math.floor(p / width)) * 8192 + x0 + (p % width));
    if ((h & 63) === 0) data[i + ((h >>> 6) % 3)] ^= 1;
  }
};
`

// jsCommonPart 页面与 Worker 都有的部分：WebGL 标识 / 能力参数与 readPixels 噪声、OffscreenCanvas 噪声
const jsCommonPart = `
if (C.webgl) {
  const W = C.webgl, P = W.params || {};
  for (const Ctx of [G.WebGLRenderingContext, G.WebGL2RenderingContext]) {
    if (!Ctx) continue;
    patchMethod(Ctx.prototype, 'getParameter', (orig, self, args) => {
      const value = Reflect.apply(orig, self, args);
      const p = args[0];
      if (p === 37445) return value === null ? value : W.vendor;
      if (p === 37446) return value === null ? value : W.renderer;
      if (Object.prototype.hasOwnProperty.call(P, p)) {
        const v = P[p];
        return Array.isArray(v) ? (p === 3386 ? new Int32Array(v) : new Float32Array(v)) : v;
      }
      return value;
    });
    if (C.seed) {
      patchMethod(Ctx.prototype, 'readPixels', (orig, self, args) => {
        const out = Reflect.apply(orig, self, args);
        const px = args[6];
        if (px && px.BYTES_PER_ELEMENT === 1 && px.length) noisePixels(px, args[2], args[0], args[1]);
        return out;
      });
    }
  }
}
if (C.seed && G.OffscreenCanvas && G.OffscreenCanvasRenderingContext2D) {
  const offGetContext = OffscreenCanvas.prototype.getContext;
  const offGetImageData = OffscreenCanvasRenderingContext2D.prototype.getImageData;
  patchMethod(OffscreenCanvasRenderingContext2D.prototype, 'getImageData', (orig, self, args) => {
    const img = Reflect.apply(orig, self, args);
    noisePixels(img.data, img.width, args[0] | 0, args[1] | 0);
    return img;
  });
  patchMethod(OffscreenCanvas.prototype, 'convertToBlob', (orig, self, args) => {
    if (!self.width || !self.height) return Reflect.apply(orig, self, args);
    const copy = new OffscreenCanvas(self.width, self.height);
    const ctx = Reflect.apply(offGetContext, copy, ['2d']);
    ctx.drawImage(self, 0, 0);
    const img = Reflect.apply(offGetImageData, ctx, [0, 0, self.width, self.height]);
    noisePixels(img.data, self.width, 0, 0);
    ctx.putImageData(img, 0, 0);
    return Reflect.apply(orig, copy, args);
  });
}
`

// jsWindowPart 页面 / iframe 专有部分：屏幕可用区域、iframe 中的屏幕与 DPR、受信鼠标事件的屏幕坐标、canvas 与音频噪声
const jsWindowPart = `
if (C.screen && G.Screen) {
  const S = C.screen;
  const fixed = {availWidth: S.availWidth, availHeight: S.availHeight, availTop: S.availTop, availLeft: 0};
  if (G.screen.width !== S.width) fixed.width = S.width;
  if (G.screen.height !== S.height) fixed.height = S.height;
  for (const [k, v] of Object.entries(fixed)) patchGetter(Screen.prototype, k, () => v);
  if (G.devicePixelRatio !== S.dpr) patchGetter(G, 'devicePixelRatio', () => S.dpr);
}
if (G.MouseEvent && G === G.top) {
  const uiX = () => (G.outerWidth - G.innerWidth) / 2;
  const uiY = () => G.outerHeight - G.innerHeight - uiX();
  patchGetter(MouseEvent.prototype, 'screenX', (orig, self) => self.isTrusted ? self.clientX + G.screenX + uiX() : Reflect.apply(orig, self, []));
  patchGetter(MouseEvent.prototype, 'screenY', (orig, self) => self.isTrusted ? self.clientY + G.screenY + uiY() : Reflect.apply(orig, self, []));
}
if (C.seed && G.HTMLCanvasElement) {
  const createElement = Document.prototype.createElement;
  const getContext = HTMLCanvasElement.prototype.getContext;
  const getImageData = CanvasRenderingContext2D.prototype.getImageData;
  patchMethod(CanvasRenderingContext2D.prototype, 'getImageData', (orig, self, args) => {
    const img = Reflect.apply(orig, self, args);
    noisePixels(img.data, img.width, args[0] | 0, args[1] | 0);
    return img;
  });
  const exportNoisy = (canvas, orig, args) => {
    const w = canvas.width, h = canvas.height;
    if (!w || !h) return Reflect.apply(orig, canvas, args);
    const copy = Reflect.apply(createElement, document, ['canvas']);
    copy.width = w;
    copy.height = h;
    const ctx = Reflect.apply(getContext, copy, ['2d']);
    ctx.drawImage(canvas, 0, 0);
    const img = Reflect.apply(getImageData, ctx, [0, 0, w, h]);
    noisePixels(img.data, w, 0, 0);
    ctx.putImageData(img, 0, 0);
    return Reflect.apply(orig, copy, args);
  };
  patchMethod(HTMLCanvasElement.prototype, 'toDataURL', (orig, self, args) => exportNoisy(self, orig, args));
  patchMethod(HTMLCanvasElement.prototype, 'toBlob', (orig, self, args) => exportNoisy(self, orig, args));
}
if (C.seed && G.AudioBuffer) {
  const noised = new WeakSet();
  patchMethod(AudioBuffer.prototype, 'getChannelData', (orig, self, args) => {
    const data = Reflect.apply(orig, self, args);
    if (!noised.has(data)) {
      noised.add(data);
      const ch = args[0] | 0;
      for (let i = 0; i < data.length; i += 97) {
        const h = hash(i * 8 + ch);
        if ((h & 7) === 0) data[i] += ((h >>> 3) & 1 ? 1 : -1) * 1e-7;
      }
    }
    return data;
  });
  const getChannelData = AudioBuffer.prototype.getChannelData;
  patchMethod(AudioBuffer.prototype, 'copyFromChannel', (orig, self, args) => {
    Reflect.apply(getChannelData, self, [args[1] | 0]);
    return Reflect.apply(orig, self, args);
  });
}
`

// jsWorkerPart Worker 专有部分：CDP 覆盖不到 Worker 的 platform / languages / hardwareConcurrency，
// 也覆盖不到 SharedWorker 的 userAgent / appVersion（实测仍是 HeadlessChrome；userAgentData 与请求头已由 CDP 覆盖）
const jsWorkerPart = `
if (G.WorkerNavigator) {
  const N = WorkerNavigator.prototype;
  const languages = Object.freeze([...C.languages]);
  const appVersion = C.userAgent.replace(/^Mozilla\//, '');
  patchGetter(N, 'userAgent', () => C.userAgent);
  patchGetter(N, 'appVersion', () => appVersion);
  patchGetter(N, 'platform', () => C.platform);
  patchGetter(N, 'hardwareConcurrency', () => C.hardwareConcurrency);
  patchGetter(N, 'languages', () => languages);
  patchGetter(N, 'language', () => languages[0]);
}
`
```

- [ ] **Step 4: 运行测试**

Run: `go vet ./pkg/browser/ && go test -race -count=1 -run TestIdentityScript -v ./pkg/browser/`
Expected: 全部子测试 PASS

- [ ] **Step 5: 提交**

```bash
git add pkg/browser/identity_js.go pkg/browser/identity_js_test.go
git commit -m "feat(identity): minimal injected JS layer disguised as native code

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
```

---
### Task 5: 读取真实浏览器信息与下发身份

`readHostInfo` 在启动时已有的标签页（之后的主页面）上读取真实浏览器的值，读完立即断开该会话（在开启自动附加前调用）。**不要改成开临时页再关掉**：实测关闭是异步的，紧接着开启的自动附加 5/5 次会先附加到这个正在关闭的页（`Target.detachedFromTarget` 已到、`Target.getTargets` 已不列出也一样），它会被误当成主页面。`applyIdentity` 按目标类型下发：页面 / iframe 用 Emulation 原生覆盖并注册注入脚本，Worker 用 Network UA 覆盖并执行 Worker 脚本。两条连接路径共用 `applyIdentity`。

**Files:**
- Create: `pkg/browser/identity_apply.go`
- Create: `pkg/browser/identity_apply_test.go`
- Modify: `pkg/browser/cdp_conn.go`（追加 `dialBrowser`）

**Interfaces:**
- Consumes: Task 1 `dialCDP`、`cdpConn.call`；Task 2 `Identity`、`parseHostInfo`、`identityFromConfig`；Task 3 `generateFingerprintFor`；Task 4 `identityScript`
- Produces:
  - `type cdpCaller func(method string, params any) (json.RawMessage, error)`
  - `type targetKind int`；`kindPage`、`kindIframe`、`kindWorker`；`func targetKindOf(t string) (targetKind, bool)`
  - `const hostInfoJS`；`func initialPageTarget(conn *cdpConn) (string, error)`；`func readHostInfo(conn *cdpConn, targetID string) (hostInfo, error)`；`func evaluateString(call cdpCaller, expr string) (string, error)`
  - `func applyIdentity(call cdpCaller, id *Identity, kind targetKind) error`
  - `func windowBounds(id *Identity) map[string]any`
  - `func dialBrowser(port int) (*cdpConn, error)`

- [ ] **Step 1: 写失败的测试** `pkg/browser/identity_apply_test.go`

```go
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
```

- [ ] **Step 2: 运行确认失败**

Run: `go test -count=1 -run TestApplyIdentityOnPage ./pkg/browser/`
Expected: 编译失败，`undefined: dialBrowser`

- [ ] **Step 3: 实现** — 在 `pkg/browser/cdp_conn.go` 末尾追加（import 补 `net/http`）：

```go
// dialBrowser 连接浏览器级 CDP WebSocket（/json/version 中的 webSocketDebuggerUrl）
func dialBrowser(port int) (*cdpConn, error) {
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Get(fmt.Sprintf("http://localhost:%d/json/version", port))
	if err != nil {
		return nil, fmt.Errorf("get browser endpoint: %w", err)
	}
	defer resp.Body.Close()
	var v struct {
		WebSocketDebuggerURL string `json:"webSocketDebuggerUrl"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&v); err != nil {
		return nil, fmt.Errorf("decode browser endpoint: %w", err)
	}
	if v.WebSocketDebuggerURL == "" {
		return nil, fmt.Errorf("browser endpoint has no webSocketDebuggerUrl")
	}
	return dialCDP(v.WebSocketDebuggerURL)
}
```

`pkg/browser/identity_apply.go`：

```go
package browser

import (
	"encoding/json"
	"fmt"
	"strings"
)

// cdpCaller 在某个目标会话上发送一条 CDP 命令
type cdpCaller func(method string, params any) (json.RawMessage, error)

// targetKind 需要下发身份的目标类型
type targetKind int

const (
	kindPage targetKind = iota
	kindIframe
	kindWorker
)

// targetKindOf 把 TargetInfo.type 映射为下发类型；其余类型（浏览器、扩展后台等）不处理
func targetKindOf(t string) (targetKind, bool) {
	switch t {
	case "page":
		return kindPage, true
	case "iframe":
		return kindIframe, true
	case "worker", "shared_worker", "service_worker":
		return kindWorker, true
	}
	return 0, false
}

// hostInfoJS 在未覆盖身份的页面上读取真实浏览器的值（初始标签页 chrome://newtab 或 about:blank 都是安全上下文，可读 userAgentData）
const hostInfoJS = `(async () => {
  const d = navigator.userAgentData;
  const h = await d.getHighEntropyValues(['architecture', 'bitness', 'model', 'platformVersion', 'fullVersionList', 'wow64', 'formFactors']);
  const g = document.createElement('canvas').getContext('webgl');
  let webglVendor = '', webglRenderer = '';
  if (g) {
    g.getExtension('WEBGL_debug_renderer_info');
    webglVendor = g.getParameter(37445) || '';
    webglRenderer = g.getParameter(37446) || '';
  }
  return JSON.stringify({
    userAgent: navigator.userAgent, platform: navigator.platform, languages: navigator.languages,
    timezone: Intl.DateTimeFormat().resolvedOptions().timeZone, hardwareConcurrency: navigator.hardwareConcurrency,
    screen: [screen.width, screen.height, devicePixelRatio],
    brands: d.brands, uaPlatform: d.platform, mobile: d.mobile,
    fullVersionList: h.fullVersionList, platformVersion: h.platformVersion, architecture: h.architecture,
    bitness: h.bitness, model: h.model, wow64: h.wow64, formFactors: h.formFactors || [],
    webglVendor, webglRenderer,
  });
})()`

// evaluateString 执行返回字符串的表达式（支持 Promise），脚本异常转为错误
func evaluateString(call cdpCaller, expr string) (string, error) {
	raw, err := call("Runtime.evaluate", map[string]any{"expression": expr, "awaitPromise": true, "returnByValue": true})
	if err != nil {
		return "", err
	}
	var res struct {
		Result struct {
			Value any `json:"value"`
		} `json:"result"`
		ExceptionDetails *struct {
			Text      string `json:"text"`
			Exception *struct {
				Description string `json:"description"`
			} `json:"exception"`
		} `json:"exceptionDetails"`
	}
	if err := json.Unmarshal(raw, &res); err != nil {
		return "", fmt.Errorf("decode evaluate result: %w", err)
	}
	if d := res.ExceptionDetails; d != nil {
		if d.Exception != nil {
			return "", fmt.Errorf("script error: %s", d.Exception.Description)
		}
		return "", fmt.Errorf("script error: %s", d.Text)
	}
	s, _ := res.Result.Value.(string)
	return s, nil
}

// initialPageTarget 启动时已有的第一个页面标签页，作为主页面
func initialPageTarget(conn *cdpConn) (string, error) {
	raw, err := conn.call("", "Target.getTargets", nil)
	if err != nil {
		return "", fmt.Errorf("list targets: %w", err)
	}
	var res struct {
		TargetInfos []struct {
			TargetID string `json:"targetId"`
			Type     string `json:"type"`
		} `json:"targetInfos"`
	}
	if err := json.Unmarshal(raw, &res); err != nil {
		return "", fmt.Errorf("decode targets: %w", err)
	}
	for _, ti := range res.TargetInfos {
		if ti.Type == "page" {
			return ti.TargetID, nil
		}
	}
	return "", fmt.Errorf("browser has no page target")
}

// readHostInfo 在指定标签页上读取真实浏览器的值，读完断开会话；必须在开启自动附加之前调用，
// 读取时这个页还没有被下发身份（不开临时页的原因见本任务开头）
func readHostInfo(conn *cdpConn, targetID string) (hostInfo, error) {
	raw, err := conn.call("", "Target.attachToTarget", map[string]any{"targetId": targetID, "flatten": true})
	if err != nil {
		return hostInfo{}, fmt.Errorf("attach initial page: %w", err)
	}
	var attached struct {
		SessionID string `json:"sessionId"`
	}
	if err := json.Unmarshal(raw, &attached); err != nil {
		return hostInfo{}, fmt.Errorf("decode initial page session: %w", err)
	}
	call := func(m string, p any) (json.RawMessage, error) { return conn.call(attached.SessionID, m, p) }
	value, err := evaluateString(call, hostInfoJS)
	// 先断开再处理结果：留着这个会话，自动附加后会有两个会话同时管同一个页
	if _, derr := conn.call("", "Target.detachFromTarget", map[string]any{"sessionId": attached.SessionID}); derr != nil && err == nil {
		err = fmt.Errorf("detach initial page: %w", derr)
	}
	if err != nil {
		return hostInfo{}, fmt.Errorf("read host info: %w", err)
	}
	return parseHostInfo(value)
}

// cdpStep 一条待发送的命令
type cdpStep struct {
	method string
	params any
}

// applyIdentity 在目标运行前下发身份：页面 / iframe 用 Emulation 原生覆盖并注册注入脚本；
// Worker 用 Network 覆盖 UA 与 client hints，再执行 Worker 脚本补齐 navigator
func applyIdentity(call cdpCaller, id *Identity, kind targetKind) error {
	ua := map[string]any{"userAgent": id.UserAgent, "acceptLanguage": id.AcceptLanguage, "platform": id.Platform, "userAgentMetadata": id.Metadata}
	tz := map[string]any{"timezoneId": id.Timezone}

	if kind == kindWorker {
		for _, s := range []cdpStep{{"Network.setUserAgentOverride", ua}, {"Emulation.setTimezoneOverride", tz}} {
			if _, err := call(s.method, s.params); err != nil {
				return fmt.Errorf("%s: %w", s.method, err)
			}
		}
		if _, err := evaluateString(call, identityScript(id, true)); err != nil {
			return fmt.Errorf("worker script: %w", err)
		}
		return nil
	}

	steps := []cdpStep{
		{"Page.enable", nil},
		{"Emulation.setUserAgentOverride", ua},
		{"Emulation.setTimezoneOverride", tz},
		{"Emulation.setLocaleOverride", map[string]any{"locale": id.Locale()}},
		{"Emulation.setHardwareConcurrencyOverride", map[string]any{"hardwareConcurrency": id.HardwareConcurrency}},
	}
	// iframe 不支持 setDeviceMetricsOverride（只能用于顶层目标），其屏幕与 DPR 由注入脚本补
	if kind == kindPage && id.Screen.Width > 0 {
		s := id.Screen
		steps = append(steps, cdpStep{"Emulation.setDeviceMetricsOverride", map[string]any{
			"width": s.InnerWidth, "height": s.InnerHeight, "deviceScaleFactor": s.DPR, "mobile": false,
			"screenWidth": s.Width, "screenHeight": s.Height,
		}})
	}
	steps = append(steps, cdpStep{"Page.addScriptToEvaluateOnNewDocument", map[string]any{"source": identityScript(id, false)}})

	for _, s := range steps {
		if _, err := call(s.method, s.params); err != nil {
			// 同一目标已设置过 locale 时 Chrome 会拒绝重复设置，身份已生效
			if s.method == "Emulation.setLocaleOverride" && strings.Contains(err.Error(), "Another locale override") {
				continue
			}
			return fmt.Errorf("%s: %w", s.method, err)
		}
	}
	return nil
}

// windowBounds Browser.setWindowBounds 的参数：最大化窗口的位置与外框尺寸
func windowBounds(id *Identity) map[string]any {
	return map[string]any{"left": 0, "top": id.Screen.AvailTop, "width": id.Screen.OuterWidth, "height": id.Screen.OuterHeight}
}
```

- [ ] **Step 4: 运行测试**

Run: `go vet ./pkg/browser/ && go test -race -count=1 -run TestApplyIdentityOnPage -v ./pkg/browser/`
Expected: PASS

- [ ] **Step 5: 提交**

```bash
git add pkg/browser/identity_apply.go pkg/browser/identity_apply_test.go pkg/browser/cdp_conn.go
git commit -m "feat(identity): read host browser info and apply identity per target

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
```

---
### Task 6: 目标管理器接入 CustomCDP

CustomCDP 改为浏览器级连接：读取真实浏览器信息 → 派生身份 → 开启自动附加，每个页面 / 跨站 iframe / Worker / 弹窗在运行前下发身份再放行 → 主页面会话交给 `CustomCDPPage`。删除 `CustomCDPPage.initialize` 里旧的 UA 覆盖与脚本注入。

**Files:**
- Create: `pkg/browser/target_manager.go`
- Create: `pkg/browser/identity_integration_test.go`
- Modify: `pkg/browser/identity_apply.go`（追加 `identityForOptions`）
- Modify: `pkg/browser/cdp_custom.go`（`CustomCDPConnector.Connect`、`CustomCDPPage.initialize`）

**Interfaces:**
- Consumes: Task 1 `cdpConn`、`dialBrowser`（Task 5）、`CustomCDPClient{conn, sessionID}`；Task 2 `identityFromConfig`、`hostIdentity`；Task 5 `initialPageTarget`、`readHostInfo`、`applyIdentity`、`targetKindOf`、`windowBounds`
- Produces:
  - `func identityForOptions(opts *ConnectOptions, host hostInfo) (*Identity, error)`
  - `func startTargetManager(conn *cdpConn, id *Identity, mainTarget string) (mainSession string, err error)`

- [ ] **Step 1: 写失败的测试** `pkg/browser/identity_integration_test.go`

```go
package browser

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

// surfaceInfoJS 各执行环境（主页面、跨站 iframe、专用 / 共享 Worker、弹窗）读取的身份信号
const surfaceInfoJS = `() => ({ua: navigator.userAgent, appVersion: navigator.appVersion, platform: navigator.platform, langs: navigator.languages.join(),
  hc: navigator.hardwareConcurrency, tz: Intl.DateTimeFormat().resolvedOptions().timeZone, date: new Date(0).toString(),
  locale: Intl.DateTimeFormat().resolvedOptions().locale, uadPlatform: navigator.userAgentData.platform,
  brands: navigator.userAgentData.brands.map(b => b.brand).join(),
  screen: self.document ? [screen.width, screen.height, screen.availHeight, devicePixelRatio].join() : null,
  webgl: (() => { try { const c = self.document ? document.createElement('canvas') : new OffscreenCanvas(1, 1);
    const g = c.getContext('webgl'); g.getExtension('WEBGL_debug_renderer_info'); return g.getParameter(37446); } catch (e) { return 'n/a'; } })()})`

// surfaceServers 主站（主页面 + 专用 / 共享 Worker + 弹窗）与跨站 iframe 站点，记录每个路径的请求头
type surfaceServers struct {
	main, frame *httptest.Server
	mu          sync.Mutex
	headers     map[string]http.Header
}

func newSurfaceServers(t *testing.T) *surfaceServers {
	t.Helper()
	ss := &surfaceServers{headers: map[string]http.Header{}}
	rec := func(r *http.Request) {
		ss.mu.Lock()
		ss.headers[r.URL.Path] = r.Header.Clone()
		ss.mu.Unlock()
	}
	ss.frame = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rec(r)
		fmt.Fprintf(w, `<script>parent.postMessage({frame: 1, info: (%s)()}, '*')</script>`, surfaceInfoJS)
	}))
	frameURL := strings.Replace(ss.frame.URL, "127.0.0.1", "localhost", 1) // 不同 site → 跨进程 iframe
	ss.main = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rec(r)
		switch r.URL.Path {
		case "/":
			fmt.Fprintf(w, `<title>surfaces</title><script>
const info = %s;
window.__r = {main: info()};
const w = new Worker('/worker.js');
w.onmessage = e => window.__r.worker = e.data;
const sw = new SharedWorker('/shared.js');
sw.port.onmessage = e => window.__r.shared = e.data;
sw.port.start();
addEventListener('message', e => { if (e.data && e.data.frame) window.__r.iframe = e.data.info; });
</script><iframe src="%s/f"></iframe>`, surfaceInfoJS, frameURL)
		case "/worker.js":
			w.Header().Set("Content-Type", "text/javascript")
			fmt.Fprintf(w, `postMessage((%s)())`, surfaceInfoJS)
		case "/shared.js":
			w.Header().Set("Content-Type", "text/javascript")
			fmt.Fprintf(w, `onconnect = e => e.ports[0].postMessage((%s)())`, surfaceInfoJS)
		case "/popup":
			fmt.Fprintf(w, `<title>popup</title><script>window.__info = (%s)()</script>`, surfaceInfoJS)
		default:
			fmt.Fprint(w, `<title>plain</title>`)
		}
	}))
	t.Cleanup(ss.main.Close)
	t.Cleanup(ss.frame.Close)
	return ss
}

// collectSurfaces 导航到主页面，收集主页面、专用 / 共享 Worker、iframe、弹窗的身份信号
func collectSurfaces(t *testing.T, p Page, ss *surfaceServers) map[string]map[string]any {
	t.Helper()
	if err := p.Navigate(ss.main.URL + "/"); err != nil {
		t.Fatalf("Navigate: %v", err)
	}
	out := map[string]map[string]any{}
	reported := func() bool { return out["worker"] != nil && out["shared"] != nil && out["iframe"] != nil }
	for deadline := time.Now().Add(10 * time.Second); time.Now().Before(deadline); time.Sleep(100 * time.Millisecond) {
		v, err := p.Evaluate(`JSON.stringify(window.__r)`)
		if s, ok := v.(string); err == nil && ok && json.Unmarshal([]byte(s), &out) == nil && reported() {
			break
		}
	}
	if !reported() {
		t.Fatalf("worker / shared worker / iframe never reported: %v", out)
	}
	if _, err := p.Evaluate(`window.__p = window.open('/popup'), true`); err != nil {
		t.Fatal(err)
	}
	for deadline := time.Now().Add(10 * time.Second); time.Now().Before(deadline); time.Sleep(100 * time.Millisecond) {
		v, _ := p.Evaluate(`window.__p && window.__p.__info ? JSON.stringify(window.__p.__info) : ''`)
		if s, _ := v.(string); s != "" {
			popup := map[string]any{}
			json.Unmarshal([]byte(s), &popup)
			out["popup"] = popup
			break
		}
	}
	if out["popup"] == nil {
		t.Fatal("popup never reported")
	}
	return out
}

// TestIdentityAcrossSurfaces 生产配置下，同一账号在请求头、主页面、跨站 iframe、Worker、弹窗给出完全一致的身份
func TestIdentityAcrossSurfaces(t *testing.T) {
	cases := []struct {
		name, ua, fpUser       string
		wantPlatform, wantUAD  string
		wantLangs, wantTZ      string
	}{
		{"windows account", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/128.0.0.0 Safari/537.36",
			"surface-win", "Win32", "Windows", "ja-JP,ja", "Asia/Tokyo"},
		{"mac account", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36",
			"surface-mac", "MacIntel", "macOS", "ja-JP,ja", "Asia/Tokyo"},
		{"no fingerprint", "", "", "", "", "", ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ss := newSurfaceServers(t)
			opts := &ConnectOptions{Headless: true, UseCustomCDP: true, FingerprintDir: t.TempDir()}
			if tc.fpUser != "" {
				opts.FingerprintUserID, opts.UserAgent, opts.Language, opts.Timezone = tc.fpUser, tc.ua, "ja-JP", "Asia/Tokyo"
			}
			inst, err := Connect(t.Context(), opts)
			if err != nil {
				t.Fatalf("Connect: %v", err)
			}
			defer inst.Close()
			surfaces := collectSurfaces(t, inst.Page(), ss) // Connect 返回后立即导航（Review Focus 1）

			main := surfaces["main"]
			ua, _ := main["ua"].(string)
			if strings.Contains(ua, "Headless") {
				t.Errorf("headless UA leaked: %q", ua)
			}
			for name, s := range surfaces {
				for _, k := range []string{"ua", "appVersion", "platform", "langs", "hc", "tz", "date", "locale", "uadPlatform", "brands", "webgl"} {
					if fmt.Sprint(s[k]) != fmt.Sprint(main[k]) {
						t.Errorf("%s.%s = %v, main has %v", name, k, s[k], main[k])
					}
				}
				if name != "worker" && name != "shared" && fmt.Sprint(s["screen"]) != fmt.Sprint(main["screen"]) {
					t.Errorf("%s.screen = %v, main has %v", name, s["screen"], main["screen"])
				}
				if strings.Contains(fmt.Sprint(s["webgl"]), "SwiftShader") {
					t.Errorf("%s exposes SwiftShader", name)
				}
			}
			if !strings.Contains(fmt.Sprint(main["brands"]), "Google Chrome") {
				t.Errorf("brands = %v", main["brands"])
			}
			if strings.HasPrefix(fmt.Sprint(main["screen"]), "800,600") {
				t.Errorf("headless default screen leaked: %v", main["screen"])
			}
			if tc.fpUser != "" {
				for k, want := range map[string]string{"platform": tc.wantPlatform, "uadPlatform": tc.wantUAD, "langs": tc.wantLangs, "tz": tc.wantTZ} {
					if fmt.Sprint(main[k]) != want {
						t.Errorf("main.%s = %v, want %s", k, main[k], want)
					}
				}
			}

			ss.mu.Lock()
			for _, path := range []string{"/", "/worker.js", "/shared.js", "/f", "/popup"} {
				h := ss.headers[path]
				if h == nil {
					t.Errorf("no request seen for %s", path)
					continue
				}
				if h.Get("User-Agent") != ua {
					t.Errorf("%s User-Agent = %q, page says %q", path, h.Get("User-Agent"), ua)
				}
				if tc.fpUser != "" && !strings.HasPrefix(h.Get("Accept-Language"), tc.wantLangs) {
					t.Errorf("%s Accept-Language = %q", path, h.Get("Accept-Language"))
				}
			}
			for _, path := range []string{"/", "/f", "/popup"} {
				if h := ss.headers[path]; h != nil && h.Get("Sec-Ch-Ua-Platform") != `"`+fmt.Sprint(main["uadPlatform"])+`"` {
					t.Errorf("%s Sec-CH-UA-Platform = %q", path, h.Get("Sec-Ch-Ua-Platform"))
				}
			}
			ss.mu.Unlock()

			// 无注入痕迹；WebRTC 为原生
			for js, want := range map[string]any{
				`Object.getOwnPropertyNames(navigator).length`:                    float64(0),
				`Object.prototype.toString.call(new RTCPeerConnection())`:         "[object RTCPeerConnection]",
				`Function.prototype.toString.call(RTCPeerConnection).includes('[native code]')`: true,
			} {
				if got, err := inst.Page().Evaluate(js); err != nil || got != want {
					t.Errorf("%s = %v (%v), want %v", js, got, err, want)
				}
			}

			// 受信点击的屏幕坐标与窗口几何一致
			inst.Page().Evaluate(`window.__click = null; addEventListener('click', e => window.__click = [e.screenX - e.clientX, e.screenY - e.clientY, screenX + (outerWidth - innerWidth) / 2, screenY + outerHeight - innerHeight - (outerWidth - innerWidth) / 2].join())`)
			if err := inst.Page().Click(100, 100); err != nil {
				t.Fatal(err)
			}
			time.Sleep(200 * time.Millisecond)
			v, _ := inst.Page().Evaluate(`window.__click`)
			parts := strings.Split(fmt.Sprint(v), ",")
			if len(parts) != 4 || parts[0] != parts[2] || parts[1] != parts[3] {
				t.Errorf("trusted click screen offsets %v do not match window geometry", v)
			}
		})
	}
}

// TestIdentityTargetsThatVanish 跨站 iframe 刚插入就移除、弹窗刚打开就关闭时，页面不会卡住（Review Focus 2）
func TestIdentityTargetsThatVanish(t *testing.T) {
	ss := newSurfaceServers(t)
	inst, err := Connect(t.Context(), &ConnectOptions{Headless: true, UseCustomCDP: true, FingerprintUserID: "vanish", FingerprintDir: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	defer inst.Close()
	p := inst.Page()
	if err := p.Navigate(ss.main.URL + "/plain"); err != nil {
		t.Fatal(err)
	}
	frameURL := strings.Replace(ss.frame.URL, "127.0.0.1", "localhost", 1)
	if _, err := p.Evaluate(fmt.Sprintf(`(() => { for (let i = 0; i < 20; i++) { const f = document.createElement('iframe'); f.src = '%s/f'; document.body.appendChild(f); f.remove(); }
	  window.open('/plain?popup').close(); return true; })()`, frameURL)); err != nil {
		t.Fatal(err)
	}
	if err := p.Navigate(ss.main.URL + "/plain?after"); err != nil {
		t.Fatalf("navigation after vanishing targets: %v", err)
	}
	if title, err := p.GetTitle(); err != nil || title != "plain" {
		t.Errorf("GetTitle() = %q, %v", title, err)
	}
}

// TestConnectRejectsCorruptFingerprint 指纹文件损坏时 Connect 返回错误，而不是静默退回默认身份（Review Focus 3）
func TestConnectRejectsCorruptFingerprint(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "corrupt.json"), []byte("{"), 0o644); err != nil {
		t.Fatal(err)
	}
	inst, err := Connect(t.Context(), &ConnectOptions{Headless: true, UseCustomCDP: true, FingerprintUserID: "corrupt", FingerprintDir: dir})
	if err == nil {
		inst.Close()
		t.Fatal("Connect succeeded with a corrupt fingerprint file")
	}
}
```

- [ ] **Step 2: 运行确认失败**

Run: `go test -count=1 -run 'TestIdentityAcrossSurfaces|TestIdentityTargetsThatVanish|TestConnectRejectsCorruptFingerprint' ./pkg/browser/`
Expected: FAIL（iframe / Worker / 弹窗的值与主页面不一致、请求头含 `HeadlessChrome`、损坏文件时 Connect 成功）

- [ ] **Step 3: 实现** `pkg/browser/target_manager.go`

```go
package browser

import (
	"encoding/json"
	"fmt"
	"time"
)

// autoAttachParams 自动附加：新目标暂停在启动处，下发身份后再放行；flatten 让子会话共用同一条连接
var autoAttachParams = map[string]any{"autoAttach": true, "waitForDebuggerOnStart": true, "flatten": true}

// targetManager 浏览器级自动附加：每个新目标（页面、跨站 iframe、Worker、弹窗）在运行任何代码前下发身份
type targetManager struct {
	conn       *cdpConn
	identity   *Identity
	mainTarget string // 主页面的 targetId（启动时已有的标签页）
	mainPage   chan attachResult
}

// attachResult 主页面的附加结果
type attachResult struct {
	sessionID string
	err       error
}

// startTargetManager 在浏览器会话上开启自动附加，等主页面（mainTarget）下发完成后返回它的会话 ID
func startTargetManager(conn *cdpConn, id *Identity, mainTarget string) (string, error) {
	tm := &targetManager{conn: conn, identity: id, mainTarget: mainTarget, mainPage: make(chan attachResult, 1)}
	stop := conn.subscribe("*", "Target.attachedToTarget", tm.onAttached)
	if _, err := conn.call("", "Target.setAutoAttach", autoAttachParams); err != nil {
		stop()
		return "", fmt.Errorf("enable auto-attach: %w", err)
	}
	select {
	case r := <-tm.mainPage:
		if r.err != nil {
			stop()
			return "", fmt.Errorf("set up main page: %w", r.err)
		}
		return r.sessionID, nil
	case <-time.After(15 * time.Second):
		stop()
		return "", fmt.Errorf("main page %s not attached within 15s", mainTarget)
	}
}

// onAttached 处理一个新附加的目标：按类型下发身份、接管它的子目标，最后一定放行
func (tm *targetManager) onAttached(_ string, params json.RawMessage) {
	var ev struct {
		SessionID  string `json:"sessionId"`
		TargetInfo struct {
			TargetID string `json:"targetId"`
			Type     string `json:"type"`
			URL      string `json:"url"`
		} `json:"targetInfo"`
		WaitingForDebugger bool `json:"waitingForDebugger"`
	}
	if json.Unmarshal(params, &ev) != nil {
		return
	}
	call := func(m string, p any) (json.RawMessage, error) { return tm.conn.call(ev.SessionID, m, p) }

	kind, handled := targetKindOf(ev.TargetInfo.Type)
	var err error
	if handled {
		err = applyIdentity(call, tm.identity, kind)
		if err == nil && kind == kindPage && tm.identity.Screen.Width > 0 {
			err = tm.setWindowBounds(ev.TargetInfo.TargetID)
		}
		if _, e := call("Target.setAutoAttach", autoAttachParams); e != nil && err == nil {
			err = fmt.Errorf("Target.setAutoAttach: %w", e)
		}
	}
	// 无论下发成败都放行，否则目标会一直停在启动处
	if ev.WaitingForDebugger {
		call("Runtime.runIfWaitingForDebugger", nil)
	}

	// 主页面按 targetId 认定，不取"第一个附加的页面"：启动时可能有别的页面先被附加
	if ev.TargetInfo.TargetID == tm.mainTarget {
		tm.mainPage <- attachResult{sessionID: ev.SessionID, err: err}
		return
	}
	if err != nil {
		// 目标在下发过程中消失（iframe 被移除、弹窗被关闭）时会走到这里，只告警不影响其他目标
		fmt.Printf("Warning: identity setup for %s target %q failed: %v\n", ev.TargetInfo.Type, ev.TargetInfo.URL, err)
	}
}

// setWindowBounds 把页面所在窗口设为身份的最大化窗口（决定 outerWidth / outerHeight / screenX / screenY）
func (tm *targetManager) setWindowBounds(targetID string) error {
	raw, err := tm.conn.call("", "Browser.getWindowForTarget", map[string]any{"targetId": targetID})
	if err != nil {
		return fmt.Errorf("Browser.getWindowForTarget: %w", err)
	}
	var w struct {
		WindowID int `json:"windowId"`
	}
	if err := json.Unmarshal(raw, &w); err != nil {
		return fmt.Errorf("decode window: %w", err)
	}
	if _, err := tm.conn.call("", "Browser.setWindowBounds", map[string]any{"windowId": w.WindowID, "bounds": windowBounds(tm.identity)}); err != nil {
		return fmt.Errorf("Browser.setWindowBounds: %w", err)
	}
	return nil
}
```

在 `pkg/browser/identity_apply.go` 末尾追加：

```go
// identityForOptions 设了 FingerprintUserID 时由账号指纹派生身份，否则用真实浏览器身份（只去掉无界面痕迹）
func identityForOptions(opts *ConnectOptions, host hostInfo) (*Identity, error) {
	if opts == nil || opts.FingerprintUserID == "" {
		return hostIdentity(host), nil
	}
	dir := opts.FingerprintDir
	if dir == "" {
		dir = "./fingerprints"
	}
	manager, err := NewUserFingerprintManager(dir)
	if err != nil {
		return nil, fmt.Errorf("fingerprint manager: %w", err)
	}
	cfg, err := manager.GetOrCreateUserFingerprint(opts.FingerprintUserID, GetInitParamsFromOptions(opts))
	if err != nil {
		return nil, err
	}
	return identityFromConfig(cfg, host)
}
```

`pkg/browser/cdp_custom.go`：`CustomCDPConnector.Connect` 方法体替换为：

```go
	conn, err := dialBrowser(chrome.Port)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to browser: %w", err)
	}
	mainTarget, err := initialPageTarget(conn)
	if err != nil {
		conn.close()
		return nil, err
	}
	host, err := readHostInfo(conn, mainTarget)
	if err != nil {
		conn.close()
		return nil, err
	}
	id, err := identityForOptions(opts, host)
	if err != nil {
		conn.close()
		return nil, err
	}
	session, err := startTargetManager(conn, id, mainTarget)
	if err != nil {
		conn.close()
		return nil, err
	}

	page := &CustomCDPPage{
		client: &CustomCDPClient{conn: conn, sessionID: session},
		chrome: chrome,
		opts:   opts,
		ctx:    ctx,
		cursor: NewGhostCursor(),
	}
	if err := page.initialize(); err != nil {
		conn.close()
		return nil, fmt.Errorf("failed to initialize custom CDP page: %w", err)
	}
	return page, nil
```

`CustomCDPPage.initialize`：保留 `EnablePageDomain`、`EnableDOMDomain`、`Page.setLifecycleEventsEnabled`、代理认证四段；删除从 `// CRITICAL: Inject stealth script on new document WITHOUT Runtime.Enable` 到函数结尾的旧 UA 覆盖与脚本注入代码，改为 `return nil`。方法注释改为：`// initialize 启用页面所需的域与代理认证；身份与注入脚本已由 target manager 在页面运行前下发`。

- [ ] **Step 4: 运行测试**

Run: `go vet ./pkg/... ./internal/... && go test -race -count=1 -timeout 600s -run 'TestIdentityAcrossSurfaces|TestIdentityTargetsThatVanish|TestConnectRejectsCorruptFingerprint|TestCustomCDPPage|TestNavigation|TestProxyAuth|TestRequestInterception|TestTurnstileOption|TestFingerprintAudio' -v ./pkg/browser/ 2>&1 | grep -E '^(---|    ---|ok|FAIL)|_test.go'`
Expected: 全部 PASS。若"受信点击"一项失败，说明原生 CDP 点击的 screenX/Y 与几何不符而注入脚本的修正未生效，检查 `jsWindowPart` 中 MouseEvent 段是否在顶层页面执行。

- [ ] **Step 5: 提交**

```bash
git add pkg/browser/target_manager.go pkg/browser/identity_integration_test.go pkg/browser/identity_apply.go pkg/browser/cdp_custom.go
git commit -m "feat(custom-cdp): apply the account identity to every page, iframe, worker and popup

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
```

---
### Task 7: chromedp 路径主页面下发同一身份

`UseCustomCDP:false` 时主页面也下发同一身份（避免旧的 JS 改写被移除后出现倒退）。该路径不接管 iframe / Worker / 弹窗（子项目 D）。去掉固定的 1920×1080 视口模拟（由身份几何取代）。

**Files:**
- Modify: `pkg/browser/connector.go`（`initialize` 中注入 stealth 脚本与 UA 的 `ActionFunc`、`setupAdditionalStealth`）
- Modify: `pkg/browser/identity_integration_test.go`（追加 `TestCDPPageIdentity`）

**Interfaces:**
- Consumes: Task 2 `parseHostInfo`；Task 5 `hostInfoJS`、`applyIdentity`、`windowBounds`、`kindPage`；Task 6 `identityForOptions`
- Produces: `func setChromedpWindowBounds(ctx context.Context, id *Identity) error`

- [ ] **Step 1: 写失败的测试**（追加到 `identity_integration_test.go`）

```go
// TestCDPPageIdentity chromedp 路径：主页面的请求头与 JS 是账号身份，没有无界面痕迹；损坏的指纹文件同样报错
func TestCDPPageIdentity(t *testing.T) {
	ss := newSurfaceServers(t)
	inst, err := Connect(t.Context(), &ConnectOptions{
		Headless: true, FingerprintUserID: "cdppage-win", FingerprintDir: t.TempDir(), Language: "ja-JP",
		UserAgent: "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36",
	})
	if err != nil {
		t.Fatal(err)
	}
	defer inst.Close()
	p := inst.Page()
	if err := p.Navigate(ss.main.URL + "/plain"); err != nil {
		t.Fatal(err)
	}
	for js, want := range map[string]any{
		`navigator.platform`:                 "Win32",
		`navigator.userAgentData.platform`:   "Windows",
		`navigator.languages.join()`:         "ja-JP,ja",
		`navigator.userAgent.includes('Headless')`: false,
		`screen.width === 800 && screen.height === 600`: false,
		`(() => { const g = document.createElement('canvas').getContext('webgl'); g.getExtension('WEBGL_debug_renderer_info'); return g.getParameter(37446).includes('SwiftShader'); })()`: false,
	} {
		if got, err := p.Evaluate(js); err != nil || got != want {
			t.Errorf("%s = %v (%v), want %v", js, got, err, want)
		}
	}
	ss.mu.Lock()
	ua := ss.headers["/plain"].Get("User-Agent")
	ss.mu.Unlock()
	if got, _ := p.Evaluate(`navigator.userAgent`); got != ua {
		t.Errorf("header UA %q != JS UA %v", ua, got)
	}

	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "corrupt.json"), []byte("{"), 0o644); err != nil {
		t.Fatal(err)
	}
	if bad, err := Connect(t.Context(), &ConnectOptions{Headless: true, FingerprintUserID: "corrupt", FingerprintDir: dir}); err == nil {
		bad.Close()
		t.Error("chromedp path accepted a corrupt fingerprint file")
	}
}
```

- [ ] **Step 2: 运行确认失败**

Run: `go test -count=1 -run TestCDPPageIdentity ./pkg/browser/`
Expected: FAIL（`navigator.userAgentData.platform` 不是 Windows、WebGL 含 SwiftShader 等）

- [ ] **Step 3: 实现**（`pkg/browser/connector.go`）

`initialize` 里 `chromedp.Run` 的动作列表中，删除 `p.setupAdditionalStealth(),`，并把 `// CRITICAL: Inject stealth script and set UserAgent` 下的整个 `chromedp.ActionFunc` 替换为：

```go
		// 身份：读取真实浏览器信息，派生并下发到主页面（chromedp 路径不接管 iframe / Worker / 弹窗，见子项目 D）
		chromedp.ActionFunc(func(ctx context.Context) error {
			var raw string
			if err := chromedp.Evaluate(hostInfoJS, &raw, func(ep *runtime.EvaluateParams) *runtime.EvaluateParams {
				return ep.WithAwaitPromise(true)
			}).Do(ctx); err != nil {
				return fmt.Errorf("read host info: %w", err)
			}
			host, err := parseHostInfo(raw)
			if err != nil {
				return err
			}
			id, err := identityForOptions(p.opts, host)
			if err != nil {
				return err
			}
			call := func(method string, params any) (json.RawMessage, error) {
				return nil, cdp.Execute(ctx, method, params, nil)
			}
			if err := applyIdentity(call, id, kindPage); err != nil {
				return err
			}
			if id.Screen.Width > 0 {
				return setChromedpWindowBounds(ctx, id)
			}
			return nil
		}),
```

删除不再使用的 `setupAdditionalStealth` 方法，在文件末尾追加：

```go
// setChromedpWindowBounds chromedp 路径：把主页面所在窗口设为身份的最大化窗口
func setChromedpWindowBounds(ctx context.Context, id *Identity) error {
	c := chromedp.FromContext(ctx)
	bctx := cdp.WithExecutor(ctx, c.Browser)
	var w struct {
		WindowID int64 `json:"windowId"`
	}
	if err := cdp.Execute(bctx, "Browser.getWindowForTarget", map[string]any{"targetId": c.Target.TargetID}, &w); err != nil {
		return fmt.Errorf("Browser.getWindowForTarget: %w", err)
	}
	if err := cdp.Execute(bctx, "Browser.setWindowBounds", map[string]any{"windowId": w.WindowID, "bounds": windowBounds(id)}, nil); err != nil {
		return fmt.Errorf("Browser.setWindowBounds: %w", err)
	}
	return nil
}
```

import 补 `github.com/chromedp/cdproto/cdp`、`github.com/chromedp/cdproto/runtime`；移除 `github.com/chromedp/cdproto/emulation`（此后 connector.go 不再使用）。

- [ ] **Step 4: 运行测试**

Run: `go vet ./pkg/... ./internal/... && go test -race -count=1 -run 'TestCDPPageIdentity|TestConnect|TestBrowserScreenshot|TestStealthFeatures|TestNavigation|TestProxyAuth|TestRequestInterception' ./pkg/browser/`
Expected: 全部 PASS

- [ ] **Step 5: 提交**

```bash
git add pkg/browser/connector.go pkg/browser/identity_integration_test.go
git commit -m "feat(cdppage): apply the same identity on the chromedp main page

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
```

---

### Task 8: WebRTC 原生防泄露

去掉假 `RTCPeerConnection` 后（Task 6 / 7 已不再注入旧脚本），改用浏览器原生策略：配置了代理时禁止 WebRTC 绕过代理直连 UDP。

**Files:**
- Modify: `internal/config/config.go`（新增 `GetWebRTCFlags`）
- Modify: `pkg/browser/launcher.go`（`buildChromeFlags` 两个分支的代理段）
- Modify: `pkg/browser/launcher_test.go`（追加 `TestWebRTCPolicyFlag`）

**Interfaces:**
- Produces: `func GetWebRTCFlags(proxyConfigured bool) []string`（`internal/config`）

- [ ] **Step 1: 写失败的测试**（追加到 `launcher_test.go`，import 补 `slices`）

```go
// TestWebRTCPolicyFlag 配置代理时启动参数禁止 WebRTC 绕过代理；未配置代理时不加（不影响直连用户的 WebRTC）
func TestWebRTCPolicyFlag(t *testing.T) {
	const flag = "--force-webrtc-ip-handling-policy=disable_non_proxied_udp"
	proxy := &ProxyConfig{Host: "127.0.0.1", Port: "8080"}
	cases := []struct {
		name string
		opts *ConnectOptions
		want bool
	}{
		{"proxy", &ConnectOptions{Proxy: proxy}, true},
		{"proxy with IgnoreAllFlags", &ConnectOptions{IgnoreAllFlags: true, Proxy: proxy}, true},
		{"no proxy", &ConnectOptions{}, false},
	}
	for _, tc := range cases {
		flags := NewChromeLauncher().buildChromeFlags(tc.opts, 9222, t.TempDir())
		if got := slices.Contains(flags, flag); got != tc.want {
			t.Errorf("%s: has %s = %v, want %v", tc.name, flag, got, tc.want)
		}
	}
}
```

- [ ] **Step 2: 运行确认失败**

Run: `go test -count=1 -run TestWebRTCPolicyFlag ./pkg/browser/`
Expected: FAIL（`proxy: has ... = false, want true`）

- [ ] **Step 3: 实现**

`internal/config/config.go` 追加：

```go
// GetWebRTCFlags 配置了代理时禁止 WebRTC 绕过代理直连 UDP，避免泄露真实 IP；
// 这是浏览器原生策略，页面看到的仍是原生 RTCPeerConnection
func GetWebRTCFlags(proxyConfigured bool) []string {
	if !proxyConfigured {
		return nil
	}
	return []string{"--force-webrtc-ip-handling-policy=disable_non_proxied_udp"}
}
```

`pkg/browser/launcher.go` 的 `buildChromeFlags` 中，两处 `if opts.Proxy != nil {` 代码块内、`flags = append(flags, proxyFlags...)` 之后各加一行：

```go
			flags = append(flags, config.GetWebRTCFlags(true)...)
```

- [ ] **Step 4: 运行测试**

Run: `go vet ./pkg/... ./internal/... && go test -race -count=1 -run 'TestWebRTCPolicyFlag|TestProxyAuth|TestIdentityAcrossSurfaces' ./pkg/browser/`
Expected: 全部 PASS

- [ ] **Step 5: 提交**

```bash
git add internal/config/config.go pkg/browser/launcher.go pkg/browser/launcher_test.go
git commit -m "feat(launcher): block non-proxied WebRTC UDP natively when a proxy is set

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
```

---

### Task 9: 旧脚本标注、部署文档与 Linux 生产环境验证

**Files:**
- Modify: `pkg/browser/stealth.go`、`pkg/browser/script_cache.go`、`pkg/browser/fingerprint_injector.go`、`pkg/browser/enhanced_audio_webgl_injector.go`、`pkg/browser/timestamp_fingerprint_injector.go`（导出函数加 Deprecated 注释）
- Create: `test/linux-chrome.Dockerfile`、`scripts/test-linux-chrome.sh`
- Create: `docs/deploy/production.Dockerfile`、`docs/deploy/README.md`
- Modify: `CLAUDE.md`

- [ ] **Step 1: 标注旧脚本**

在以下导出函数 / 构造函数的注释最后一行追加 `// Deprecated: Connect 不再使用；身份改由 CDP 下发（见 identity.go），保留仅为兼容。`：`GetAdvancedStealthScript`、`InjectAdvancedStealthScripts`、`InjectStealthOnNewDocument`、`GetStealthScriptWithConfig`、`GetBaseStealthScript`（stealth.go）；`GetCachedAdvancedStealthScript`、`GetCachedSimpleStealthScript`、`GetCachedBaseStealthScript`、`GetCachedStealthScriptWithConfig`（script_cache.go）；`NewFingerprintInjector`（fingerprint_injector.go）；`NewEnhancedAudioWebGLInjector`（enhanced_audio_webgl_injector.go）；`NewTimestampFingerprintInjector`（timestamp_fingerprint_injector.go）。`GetSimpleStealthScript` 仍被 chromedp 路径的 `TargetHandler` 使用，不标注。

- [ ] **Step 2: Linux 测试环境**

`test/linux-chrome.Dockerfile`：

```dockerfile
# 模拟生产的测试环境：Linux amd64 + Google Chrome 稳定版 + 无 GPU；含 Xvfb 供有界面测试
FROM golang:1.25-bookworm
RUN apt-get update \
 && apt-get install -y --no-install-recommends wget ca-certificates xvfb fonts-liberation procps \
 && wget -q -O /tmp/chrome.deb https://dl.google.com/linux/direct/google-chrome-stable_current_amd64.deb \
 && apt-get install -y --no-install-recommends /tmp/chrome.deb \
 && rm -rf /var/lib/apt/lists/* /tmp/chrome.deb
```

`scripts/test-linux-chrome.sh`（`chmod +x`）：

```bash
#!/usr/bin/env bash
# 在模拟生产的 Linux 环境（amd64 + Google Chrome 稳定版、无 GPU）里跑库的全部测试；额外参数透传给 go test
set -euo pipefail
cd "$(dirname "$0")/.."
docker build --platform linux/amd64 -t prbg-linux-chrome -f test/linux-chrome.Dockerfile test
docker run --rm --init --platform linux/amd64 --shm-size=1g \
  -v "$PWD":/src:ro -v prbg-gomod-amd64:/go/pkg/mod -v prbg-gocache-amd64:/root/.cache/go-build -w /src \
  prbg-linux-chrome go test -race -count=1 -timeout 1800s "$@" ./pkg/... ./internal/...
```

- [ ] **Step 3: 生产部署示例**

`docs/deploy/production.Dockerfile`：

```dockerfile
# 生产镜像示例：Google Chrome 稳定版 + Windows 字体 + tini
FROM golang:1.25-bookworm AS build
WORKDIR /src
COPY . .
# 换成你的程序入口
RUN go build -o /out/app ./cmd/yourapp

FROM debian:bookworm-slim
RUN apt-get update \
 && apt-get install -y --no-install-recommends ca-certificates wget tini fontconfig \
 && wget -q -O /tmp/chrome.deb https://dl.google.com/linux/direct/google-chrome-stable_current_amd64.deb \
 && apt-get install -y --no-install-recommends /tmp/chrome.deb \
 && rm -rf /var/lib/apt/lists/* /tmp/chrome.deb
# Windows 账号需要的字体：自备合法授权的字体文件，放在构建上下文的 fonts/windows/ 下
COPY fonts/windows/ /usr/share/fonts/windows/
RUN fc-cache -f
COPY --from=build /out/app /usr/local/bin/app
# tini 作为 1 号进程回收 Chrome 的辅助进程，否则关闭浏览器时会多等 3 秒
ENTRYPOINT ["tini", "--", "/usr/local/bin/app"]
```

`docs/deploy/README.md`：

```markdown
# 生产部署说明

## 环境
- Linux amd64 + Google Chrome 稳定版（不是发行版的 chromium 包：品牌与编解码器不同）。
- 无 GPU 时 WebGL 走 SwiftShader 软件渲染；库会把显卡标识与能力参数换成账号档案里的真实显卡，但渲染出来的图像特征仍是软件渲染（已知残余风险）。
- 容器用 tini 或 `docker run --init` 作 1 号进程，回收 Chrome 辅助进程。

## 字体
Windows 账号会被字体探测检查，需要安装 Windows 常用字体：微软雅黑（msyh）、宋体（simsun）、黑体（simhei）、Arial、Times New Roman、Segoe UI、Calibri、Consolas。字体需自备合法授权，放在 `fonts/windows/` 后按 `production.Dockerfile` 构建。

Mac 账号：苹方、SF 等 Apple 字体的授权只限 Apple 硬件，无法在 Linux 上合规安装，Mac 身份在字体探测上弱于 Windows 身份。

## 验证
`scripts/test-linux-chrome.sh` 在同构环境里跑全部测试，其中 `TestIdentityAcrossSurfaces` 校验请求头、主页面、跨站 iframe、Worker、弹窗的身份一致。

## 已知残余风险
软件渲染的 canvas / WebGL 图像特征、`speechSynthesis.getVoices()` 语音列表、Mac 身份字体、不走代理时服务器的 TCP / IP 指纹为 Linux。
```

- [ ] **Step 4: 更新 CLAUDE.md**

在 `CLAUDE.md` 的 **Stealth + fingerprint injection** 小节开头加一段：

```markdown
- **Identity (current design, `docs/superpowers/specs/2026-09-30-coherent-identity-design.md`)**: `identity.go` derives an `Identity` from the account fingerprint (OS comes from the bound UA; Chrome version always follows the real browser) or from the real browser when no fingerprint is set. On the CustomCDP path `target_manager.go` connects at browser level (flatten sessions, `cdp_conn.go`), auto-attaches every page / cross-site iframe / worker / popup with `waitForDebuggerOnStart`, applies the identity through CDP Emulation (`identity_apply.go`) plus a minimal JS layer disguised as native code (`identity_js.go`), then resumes it. The chromedp path applies the identity to its main page only. The older stealth / injector scripts below are deprecated and no longer injected by `Connect`.
- Linux production-like tests: `scripts/test-linux-chrome.sh` (amd64 Google Chrome in Docker).
```

- [ ] **Step 5: 全量验证**

Run（macOS 本机）：

```bash
go vet ./pkg/... ./internal/...
GOOS=linux go vet ./pkg/... ./internal/...
GOOS=windows go vet ./pkg/... ./internal/... && GOOS=windows go test -c -o /dev/null ./pkg/browser/
go test -race -count=1 -timeout 1200s ./pkg/... ./internal/...
for f in cmd/example/*.go; do go build -o /dev/null "$f" || echo "FAIL $f"; done
for d in cmd/monitor cmd/stability_test cmd/fingerprint_collector cmd/fingerprint_stats; do go build -o /dev/null ./$d || echo "FAIL $d"; done
```

Expected: vet 无输出；测试全部 `ok`；demo / 工具无 `FAIL`。

Run（Linux 生产同构环境）：`scripts/test-linux-chrome.sh`
Expected: `ok  github.com/r0vx/puppeteer-real-browser-go/pkg/browser` 与 `internal/utils` 全部通过（包括 Xvfb 有界面测试与 `TestIdentityAcrossSurfaces`）。

- [ ] **Step 6: 提交**

```bash
git add pkg/browser/stealth.go pkg/browser/script_cache.go pkg/browser/fingerprint_injector.go pkg/browser/enhanced_audio_webgl_injector.go pkg/browser/timestamp_fingerprint_injector.go test/linux-chrome.Dockerfile scripts/test-linux-chrome.sh docs/deploy CLAUDE.md
git commit -m "docs: deprecate legacy stealth scripts, add Linux test env and deployment guide

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
```
