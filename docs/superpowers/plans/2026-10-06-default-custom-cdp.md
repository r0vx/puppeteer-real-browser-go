# 默认走 CustomCDP 通道 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 不写任何通道选项时走 CustomCDP（所有页面 / iframe / Worker / 弹窗下发身份）；chromedp 只在 `UseChromedp: true` 时使用；CustomCDP 实例上 `NewPage()` 也开 CustomCDP 页面。

**Architecture:** `ConnectOptions` 新增 `UseChromedp`，`connectToChrome` 按它分支，`UseCustomCDP` 废弃。目标管理器记录已下发身份的页面会话并按 targetId 交出，`CustomCDPPage.openTab` 经同一浏览器连接新建标签页并接管该会话；这类页面关闭时只关自己的标签页。

**Tech Stack:** Go 1.25、gorilla/websocket（cdpConn）、chromedp（旧通道）、Chrome DevTools Protocol（Target 域）。

**Spec:** `docs/superpowers/specs/2026-10-06-default-custom-cdp-design.md`

## Global Constraints

- 对外接口：不删导出符号；`UseCustomCDP` 字段保留并标 `Deprecated:`，不再影响通道选择。
- 新字段：`UseChromedp bool \`json:"useChromedp"\``，true 才走 chromedp。
- 代码注释用中文；每个函数有注释；错误带上下文（`fmt.Errorf("...: %w", err)`）；测试一律 `go test -race`。
- 构建 / vet 只针对库：`go build ./pkg/... ./internal/...`、`go vet ./pkg/... ./internal/...`；`GOOS=windows go vet` 保持通过；`cmd/example` 逐个文件编译。
- 每个任务结束提交一次，提交信息末尾加 `Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>`。

## Review Focus

1. **已显式写 `UseCustomCDP: true` 的调用方（triumph/browser-service）**：行为完全不变（Task 1 测试覆盖）。
2. **CustomCDP 实例上 `NewPage()` 开出的页面关闭后**：主页面仍可用，实例关闭时 Chrome 正常退出（Task 3 测试覆盖）。
3. **chromedp 实例（`UseChromedp: true`）上的 `NewPage()`**：仍返回 chromedp 页面，行为不变（Task 3 测试覆盖）。
4. **连续开多个新页面**：每个页面拿到自己标签页的会话，不串（Task 3 测试覆盖）。
5. **`Connect(ctx, nil)`**：与默认用例走同一分支（CustomCDP）；nil 时为有界面模式，不在自动测试里启动，由 `TestChannelSelection` 的 default 用例覆盖该分支。

## File Structure

| 文件 | 改动 |
|---|---|
| `pkg/browser/types.go` | 新增 `UseChromedp`；`UseCustomCDP` 标废弃；`BrowserContext` 增加 `custom *CustomCDPPage`；删除 `ChromeProcess.managedIdentity` |
| `pkg/browser/browser.go` | `connectToChrome` 按 `UseChromedp` 分支；`CreateBrowserContext` 记下 CustomCDP 主页面 |
| `pkg/browser/context.go` | `NewPage` 在 CustomCDP 实例上调用 `openTab`；默认选项去掉 `UseCustomCDP` |
| `pkg/browser/target_manager.go` | 记录页面会话、`waitPage`、处理 `Target.detachedFromTarget`；`startTargetManager` 返回管理器 |
| `pkg/browser/cdp_custom.go` | `CustomCDPPage` 增加 `targets`、`closeTabOnly`；`openTab`；`Close` 区分；`Connect` 去掉托管身份缓存 |
| `pkg/browser/connector.go` | 删除托管身份分支 |
| `pkg/browser/account_manager.go` | 合并逻辑改用 `UseChromedp` |
| 测试 | 各测试的通道循环改用 `UseChromedp`；新增通道选择、账号合并、`NewPage` 测试 |
| `cmd/example/*.go`、`README.md`、`CLAUDE.md` | 去掉 `UseCustomCDP`，依赖 chromedp 的示例改 `UseChromedp: true`，文档写迁移说明 |

---

### Task 1: 通道选择默认 CustomCDP

**Files:**
- Modify: `pkg/browser/types.go`（`UseCustomCDP` 处）
- Modify: `pkg/browser/browser.go`（`connectToChrome`）
- Modify: `pkg/browser/context.go`（`NewPage` 默认选项）
- Test: `pkg/browser/browser_test.go`（新增 `TestChannelSelection`）、`pkg/browser/identity_integration_test.go`、`fingerprint_audio_test.go`、`turnstile_test.go`、`proxy_test.go`、`navigation_test.go`、`cdp_custom_test.go`、`launcher_test.go`、`xvfb_test.go`

**Interfaces:**
- Produces: `ConnectOptions.UseChromedp bool`

- [ ] **Step 1: 写失败的测试**

追加到 `pkg/browser/browser_test.go`：

```go
// TestChannelSelection 默认走 CustomCDP；UseCustomCDP 已废弃不再影响通道；只有 UseChromedp 才走 chromedp
func TestChannelSelection(t *testing.T) {
	cases := []struct {
		name       string
		opts       *ConnectOptions
		wantCustom bool
	}{
		{"default", &ConnectOptions{Headless: true}, true},
		{"UseCustomCDP true (existing callers)", &ConnectOptions{Headless: true, UseCustomCDP: true}, true},
		{"UseCustomCDP false no longer selects chromedp", &ConnectOptions{Headless: true, UseCustomCDP: false}, true},
		{"UseChromedp", &ConnectOptions{Headless: true, UseChromedp: true}, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			inst, err := Connect(t.Context(), tc.opts)
			if err != nil {
				t.Fatal(err)
			}
			defer inst.Close()
			_, isCustom := inst.Page().(*CustomCDPPage)
			if isCustom != tc.wantCustom {
				t.Errorf("page type %T, want CustomCDP=%v", inst.Page(), tc.wantCustom)
			}
		})
	}
}
```

`pkg/browser/identity_integration_test.go` 的 `TestIdentityAcrossSurfaces` 中，把
`opts := &ConnectOptions{Headless: true, UseCustomCDP: true, FingerprintDir: t.TempDir()}`
改为 `opts := &ConnectOptions{Headless: true, FingerprintDir: t.TempDir()}`（测试默认通道）；同文件 `TestIdentityTargetsThatVanish`、`TestConnectRejectsCorruptFingerprint`、`TestNewPageOnCustomCDPInstance` 去掉 `UseCustomCDP: true,`；`TestCDPPageIdentity` 的 `Connect` 选项加 `UseChromedp: true`；`TestSetViewportKeepsIdentity` 的循环改为 `for _, chromedpPath := range []bool{true, false}`，子测试名 `fmt.Sprintf("UseChromedp=%v/fingerprint=%q", chromedpPath, fp)`，选项 `UseChromedp: chromedpPath`。

- [ ] **Step 2: 运行确认失败**

Run: `go test -count=1 -run 'TestChannelSelection|TestIdentityAcrossSurfaces' ./pkg/browser/`
Expected: 编译失败 `unknown field UseChromedp`

- [ ] **Step 3: 实现**

`pkg/browser/types.go`：把

```go
	// Use custom CDP client to avoid Runtime.Enable leaks (experimental)
	UseCustomCDP bool `json:"useCustomCDP"`
```

替换为：

```go
	// UseCustomCDP 不再影响通道选择：默认就走 CustomCDP（浏览器级连接，页面 / iframe / Worker / 弹窗都下发身份）。
	//
	// Deprecated: 无需再设置；要用旧的 chromedp 通道请设 UseChromedp。
	UseCustomCDP bool `json:"useCustomCDP"`

	// UseChromedp 改走旧的 chromedp 通道：只给主页面下发身份，会调用 Runtime.enable。
	// 只在需要 chromedp 专有用法（GetContext() 后直接调用 chromedp）时使用
	UseChromedp bool `json:"useChromedp"`
```

`pkg/browser/browser.go` 的 `connectToChrome` 方法体替换为：

```go
	// 默认 CustomCDP：所有目标在运行前下发身份；只有显式要求时才用旧的 chromedp 通道
	if opts.UseChromedp {
		return NewCDPConnector().Connect(ctx, chrome, opts)
	}
	return CreateCustomCDPConnector().Connect(ctx, chrome, opts)
```

`pkg/browser/context.go` 的 `NewPage` 默认选项去掉 `UseCustomCDP: false,` 这一行。

- [ ] **Step 4: 运行确认通过**

Run: `go vet ./pkg/browser/ && go test -race -count=1 -run 'TestChannelSelection|TestIdentityAcrossSurfaces|TestCDPPageIdentity|TestSetViewportKeepsIdentity' -v ./pkg/browser/`
Expected: 全部 PASS

- [ ] **Step 5: 其余测试的通道循环改用 UseChromedp**

逐个文件把"两条通道各跑一遍"的循环改为按 `UseChromedp` 区分，保证 chromedp 通道仍有覆盖：
- `fingerprint_audio_test.go`：`for _, custom := range []bool{false, true}` → `for _, chromedpPath := range []bool{true, false}`；子测试名 `fmt.Sprintf("UseChromedp=%v", chromedpPath)`；选项 `UseCustomCDP: custom,` → `UseChromedp: chromedpPath,`。
- `turnstile_test.go`、`navigation_test.go`、`proxy_test.go`（两处循环，含 `proxy=%v/UseCustomCDP=%v` 的子测试名改为 `proxy=%v/UseChromedp=%v`）：同样改法。
- `cdp_custom_test.go`、`launcher_test.go`、`xvfb_test.go`：去掉 `UseCustomCDP: true`（默认即是）。

- [ ] **Step 6: 全量运行**

Run: `go vet ./pkg/... ./internal/... && go test -race -count=1 -timeout 1500s ./pkg/... ./internal/...`
Expected: 全部 `ok`。默认通道变了，原先隐含走 chromedp 的测试（`TestConnect`、`TestStealthFeatures`、浏览器池测试等）现在走 CustomCDP，也必须通过。

- [ ] **Step 7: 提交**

```bash
git add pkg/browser/types.go pkg/browser/browser.go pkg/browser/context.go pkg/browser/*_test.go
git commit -m "feat(browser): default to the CustomCDP channel, add UseChromedp

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
```

---

### Task 2: 账号管理器按 UseChromedp 合并

**Files:**
- Modify: `pkg/browser/account_manager.go`（`NewAccountManager` 默认选项、`mergeOptions`）
- Test: `pkg/browser/browser_test.go`（新增 `TestAccountManagerMergesChannel`）

**Interfaces:**
- Consumes: Task 1 `ConnectOptions.UseChromedp`

- [ ] **Step 1: 写失败的测试**（追加到 `pkg/browser/browser_test.go`）

```go
// TestAccountManagerMergesChannel 账号级 UseChromedp 覆盖基础配置；不设时沿用基础配置
func TestAccountManagerMergesChannel(t *testing.T) {
	am := NewAccountManager(&ConnectOptions{Headless: true})
	if got := am.mergeOptions(am.baseOptions, &ConnectOptions{UseChromedp: true}); !got.UseChromedp {
		t.Error("account UseChromedp did not override the base options")
	}
	base := &ConnectOptions{Headless: true, UseChromedp: true}
	if got := am.mergeOptions(base, &ConnectOptions{}); !got.UseChromedp {
		t.Error("base UseChromedp lost when the account does not set it")
	}
	if got := am.mergeOptions(am.baseOptions, nil); got.UseChromedp {
		t.Error("default merge selected chromedp")
	}
}
```

- [ ] **Step 2: 运行确认失败**

Run: `go test -count=1 -run TestAccountManagerMergesChannel ./pkg/browser/`
Expected: FAIL（`account UseChromedp did not override` 与 `base UseChromedp lost`）

- [ ] **Step 3: 实现**（`pkg/browser/account_manager.go`）

`NewAccountManager` 默认基础配置去掉 `UseCustomCDP:   true,`。`mergeOptions` 中 `UseCustomCDP:   base.UseCustomCDP,` 改为 `UseChromedp:    base.UseChromedp,`；把

```go
		if account.UseCustomCDP {
			merged.UseCustomCDP = account.UseCustomCDP
		}
```

改为：

```go
		if account.UseChromedp {
			merged.UseChromedp = true
		}
```

- [ ] **Step 4: 运行确认通过**

Run: `gofmt -l pkg/browser/account_manager.go; go vet ./pkg/browser/ && go test -race -count=1 -run TestAccountManagerMergesChannel -v ./pkg/browser/`
Expected: PASS（gofmt 只会列出该文件原有的格式问题时，按仓库现状不整体格式化）

- [ ] **Step 5: 提交**

```bash
git add pkg/browser/account_manager.go pkg/browser/browser_test.go
git commit -m "feat(account): merge the channel choice through UseChromedp

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
```

---

### Task 3: CustomCDP 实例上 NewPage 开 CustomCDP 页面

**Files:**
- Modify: `pkg/browser/target_manager.go`、`pkg/browser/cdp_custom.go`、`pkg/browser/browser.go`（`CreateBrowserContext`）、`pkg/browser/context.go`（`NewPage`）、`pkg/browser/types.go`（`BrowserContext`、`ChromeProcess`）、`pkg/browser/connector.go`（删托管身份分支）
- Test: `pkg/browser/identity_integration_test.go`（扩展 `TestNewPageOnCustomCDPInstance`，新增 `TestNewPageOnChromedpInstance`）

**Interfaces:**
- Consumes: Task 1 `UseChromedp`
- Produces:
  - `func startTargetManager(conn *cdpConn, id *Identity, mainTarget string) (*targetManager, string, error)`
  - `func (tm *targetManager) waitPage(targetID string, timeout time.Duration) (string, error)`
  - `func (p *CustomCDPPage) openTab() (*CustomCDPPage, error)`

- [ ] **Step 1: 写失败的测试**

`pkg/browser/identity_integration_test.go` 中 `TestNewPageOnCustomCDPInstance` 的末尾（`if got, err := np.Evaluate(...)` 检查之后）追加：

```go
	if _, ok := np.(*CustomCDPPage); !ok {
		t.Errorf("NewPage on a CustomCDP instance returned %T, want *CustomCDPPage", np)
	}
	// 连续开多个新页：各自拿到自己的标签页
	var pages []Page
	for i := range 3 {
		p, err := bc.NewPage()
		if err != nil {
			t.Fatalf("NewPage #%d: %v", i, err)
		}
		pages = append(pages, p)
		if err := p.Navigate(fmt.Sprintf("%s/plain?tab=%d", ss.main.URL, i)); err != nil {
			t.Fatal(err)
		}
	}
	for i, p := range pages {
		if got, _ := p.Evaluate(`location.search`); got != fmt.Sprintf("?tab=%d", i) {
			t.Errorf("tab %d sees %v", i, got)
		}
		if err := p.Close(); err != nil {
			t.Errorf("close tab %d: %v", i, err)
		}
	}
	// 关闭新页面不能断开浏览器连接：主页面仍可用
	if err := np.Close(); err != nil {
		t.Fatal(err)
	}
	if err := inst.Page().Navigate(ss.main.URL + "/plain?main-after"); err != nil {
		t.Fatalf("main page unusable after closing new pages: %v", err)
	}
	if got, _ := inst.Page().Evaluate(`location.search`); got != "?main-after" {
		t.Errorf("main page location = %v", got)
	}
```

（该测试已有 `defer np.Close()`：把它删除，改由上面显式关闭。）

再追加新测试：

```go
// TestNewPageOnChromedpInstance chromedp 实例上的 NewPage 仍开 chromedp 页面
func TestNewPageOnChromedpInstance(t *testing.T) {
	inst, err := Connect(t.Context(), &ConnectOptions{Headless: true, UseChromedp: true})
	if err != nil {
		t.Fatal(err)
	}
	defer inst.Close()
	bc, err := inst.CreateBrowserContext(nil)
	if err != nil {
		t.Fatal(err)
	}
	np, err := bc.NewPage()
	if err != nil {
		t.Fatal(err)
	}
	defer np.Close()
	if _, ok := np.(*CDPPage); !ok {
		t.Errorf("NewPage on a chromedp instance returned %T, want *CDPPage", np)
	}
}
```

- [ ] **Step 2: 运行确认失败**

Run: `go test -count=1 -run 'TestNewPageOnCustomCDPInstance|TestNewPageOnChromedpInstance' -v ./pkg/browser/`
Expected: `TestNewPageOnCustomCDPInstance` FAIL（`returned *browser.CDPPage, want *CustomCDPPage`）；`TestNewPageOnChromedpInstance` PASS（回归保护）

- [ ] **Step 3: 实现目标管理器的页面会话登记**（`pkg/browser/target_manager.go`）

`targetManager` 结构体增加字段（import 补 `sync`）：

```go
	mu      sync.Mutex
	pages   map[string]pageSession        // 已下发身份的页面目标：targetId → 会话
	waiters map[string][]chan pageSession // 等某个页面目标的调用方
```

新增类型与方法：

```go
// pageSession 一个页面目标的会话与下发结果
type pageSession struct {
	sessionID string
	err       error
}

// recordPage 登记页面目标的会话，并唤醒等它的调用方
func (tm *targetManager) recordPage(targetID string, p pageSession) {
	tm.mu.Lock()
	defer tm.mu.Unlock()
	tm.pages[targetID] = p
	for _, ch := range tm.waiters[targetID] {
		ch <- p // 缓冲为 1，不会阻塞
	}
	delete(tm.waiters, targetID)
}

// forgetTarget 目标断开（标签页关闭等）时移除登记
func (tm *targetManager) forgetTarget(_ string, params json.RawMessage) {
	var ev struct {
		TargetID string `json:"targetId"`
	}
	if json.Unmarshal(params, &ev) != nil {
		return
	}
	tm.mu.Lock()
	delete(tm.pages, ev.TargetID)
	tm.mu.Unlock()
}

// waitPage 等页面目标下发完身份，返回它的会话；超时报错
func (tm *targetManager) waitPage(targetID string, timeout time.Duration) (string, error) {
	tm.mu.Lock()
	if p, ok := tm.pages[targetID]; ok {
		tm.mu.Unlock()
		return p.sessionID, p.err
	}
	ch := make(chan pageSession, 1)
	tm.waiters[targetID] = append(tm.waiters[targetID], ch)
	tm.mu.Unlock()
	select {
	case p := <-ch:
		return p.sessionID, p.err
	case <-time.After(timeout):
		tm.mu.Lock()
		tm.waiters[targetID] = slices.DeleteFunc(tm.waiters[targetID], func(c chan pageSession) bool { return c == ch })
		tm.mu.Unlock()
		return "", fmt.Errorf("page %s not attached within %v", targetID, timeout)
	}
}
```

（import 补 `slices`。）`startTargetManager` 改为返回管理器：

```go
// startTargetManager 在浏览器会话上开启自动附加，等主页面（mainTarget）下发完成后返回管理器与主页面会话 ID
func startTargetManager(conn *cdpConn, id *Identity, mainTarget string) (*targetManager, string, error) {
	tm := &targetManager{conn: conn, identity: id, mainTarget: mainTarget, mainPage: make(chan attachResult, 1),
		pages: map[string]pageSession{}, waiters: map[string][]chan pageSession{}}
	stopAttach := conn.subscribe("*", "Target.attachedToTarget", tm.onAttached)
	stopDetach := conn.subscribe("*", "Target.detachedFromTarget", tm.forgetTarget)
	stop := func() { stopAttach(); stopDetach() }
	if _, err := conn.call("", "Target.setAutoAttach", autoAttachParams); err != nil {
		stop()
		return nil, "", fmt.Errorf("enable auto-attach: %w", err)
	}
	select {
	case r := <-tm.mainPage:
		if r.err != nil {
			stop()
			return nil, "", fmt.Errorf("set up main page: %w", r.err)
		}
		return tm, r.sessionID, nil
	case <-time.After(15 * time.Second):
		stop()
		return nil, "", fmt.Errorf("main page %s not attached within 15s", mainTarget)
	}
}
```

`onAttached` 里，在 `// 主页面按 targetId 认定` 之前加：

```go
	// 页面目标登记会话，供 openTab 按 targetId 取用（主页面也登记，无害）
	if handled && kind == kindPage {
		tm.recordPage(ev.TargetInfo.TargetID, pageSession{sessionID: ev.SessionID, err: err})
	}
```

- [ ] **Step 4: 实现 openTab 与关闭语义**（`pkg/browser/cdp_custom.go`）

`CustomCDPPage` 结构体在 `targetID` 字段后增加：

```go
	targets      *targetManager // 本浏览器的目标管理器（openTab 用）
	closeTabOnly bool           // openTab 开出的页面：Close 只关自己的标签页，不断开浏览器连接
```

`CustomCDPConnector.Connect` 中：`session, err := startTargetManager(conn, id, mainTarget)` 改为 `tm, session, err := startTargetManager(conn, id, mainTarget)`；删除其后写 `chrome.host, chrome.managedIdentity` 的三行（含注释）；构造 `CustomCDPPage` 时加 `targets: tm,`。

`Close` 替换为：

```go
// Close closes the page. openTab 开出的页面只关闭自己的标签页；主页面断开浏览器连接（随后由实例关闭 Chrome）
func (p *CustomCDPPage) Close() error {
	if p.closeTabOnly {
		if _, err := p.client.conn.call("", "Target.closeTarget", map[string]any{"targetId": p.targetID}); err != nil {
			return fmt.Errorf("close tab: %w", err)
		}
		return nil
	}
	return p.client.Close()
}
```

新增：

```go
// openTab 在同一浏览器里新开标签页：目标管理器在它运行前下发与本页相同的身份，再交出会话
func (p *CustomCDPPage) openTab() (*CustomCDPPage, error) {
	if p.targets == nil {
		return nil, fmt.Errorf("page has no target manager")
	}
	raw, err := p.client.conn.call("", "Target.createTarget", map[string]any{"url": "about:blank"})
	if err != nil {
		return nil, fmt.Errorf("create tab: %w", err)
	}
	var created struct {
		TargetID string `json:"targetId"`
	}
	if err := json.Unmarshal(raw, &created); err != nil {
		return nil, fmt.Errorf("decode tab: %w", err)
	}
	session, err := p.targets.waitPage(created.TargetID, 15*time.Second)
	if err != nil {
		p.client.conn.call("", "Target.closeTarget", map[string]any{"targetId": created.TargetID})
		return nil, fmt.Errorf("set up tab: %w", err)
	}
	tab := &CustomCDPPage{
		client:       &CustomCDPClient{conn: p.client.conn, sessionID: session},
		chrome:       p.chrome,
		opts:         p.opts,
		ctx:          p.ctx,
		cursor:       NewGhostCursor(),
		identity:     p.identity,
		targetID:     created.TargetID,
		targets:      p.targets,
		closeTabOnly: true,
	}
	if err := tab.initialize(); err != nil {
		tab.Close()
		return nil, fmt.Errorf("initialize tab: %w", err)
	}
	return tab, nil
}
```

- [ ] **Step 5: 接入 BrowserContext，删除托管身份**

`pkg/browser/types.go` 的 `BrowserContext` 增加字段 `custom *CustomCDPPage // 实例是 CustomCDP 时的主页面（NewPage 经它开新标签页）`；`ChromeProcess` 删除 `managedIdentity` 字段，`hostMu` 注释改为 `// 保护 host`。

`pkg/browser/browser.go` 的 `CreateBrowserContext` 中，把

```go
	// Try to get options from the page if it's a CDPPage
	if cdpPage, ok := bi.page.(*CDPPage); ok {
		browserCtx.opts = cdpPage.opts
	}
```

替换为：

```go
	// 沿用实例的选项；CustomCDP 实例记下主页面，NewPage 经它在同一浏览器连接上开新标签页
	switch page := bi.page.(type) {
	case *CDPPage:
		browserCtx.opts = page.opts
	case *CustomCDPPage:
		browserCtx.opts, browserCtx.custom = page.opts, page
	}
```

`pkg/browser/context.go` 的 `NewPage` 开头加：

```go
	if bc.custom != nil {
		return bc.custom.openTab()
	}
```

`pkg/browser/connector.go` 的 `CDPPage.initialize` 中，把读取 `p.chrome.managedIdentity` 的分支（从 `// CustomCDP 实例上开的页面：target manager 已在它运行前下发了账号身份` 注释起，到 `if id == nil {` 块结束）替换为：

```go
			id, err := identityForOptions(p.opts, host)
			if err != nil {
				return err
			}
```

- [ ] **Step 6: 运行确认通过**

Run: `gofmt -l pkg/browser/target_manager.go pkg/browser/cdp_custom.go pkg/browser/context.go; go vet ./pkg/... ./internal/... && go test -race -count=1 -run 'TestNewPageOnCustomCDPInstance|TestNewPageOnChromedpInstance|TestIdentityAcrossSurfaces|TestIdentityTargetsThatVanish|TestCustomCDPPage|TestCDPPageIdentity' -v ./pkg/browser/`
Expected: 全部 PASS

- [ ] **Step 7: 提交**

```bash
git add pkg/browser/target_manager.go pkg/browser/cdp_custom.go pkg/browser/browser.go pkg/browser/context.go pkg/browser/types.go pkg/browser/connector.go pkg/browser/identity_integration_test.go
git commit -m "feat(custom-cdp): open NewPage tabs on the CustomCDP connection

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
```

---

### Task 4: 示例、文档与全量验证

**Files:**
- Modify: `cmd/example/*.go`、`cmd/*/main.go`（用到 `UseCustomCDP` 的）、`README.md`、`CLAUDE.md`

- [ ] **Step 1: 示例与工具**

- 去掉所有 `UseCustomCDP: true,` 行（连同行尾注释）。
- `UseCustomCDP: false` 的文件：若用了 `GetContext()` 后直接调用 chromedp（`douyin_network_demo.go`、`feature_test.go`），改为 `UseChromedp: true, // 本示例直接调用 chromedp，需要旧通道`；其余去掉该行。
- 改完逐个编译：

Run: `for f in cmd/example/*.go; do go build -o /dev/null "$f" || echo "FAIL $f"; done; for d in cmd/monitor cmd/stability_test cmd/fingerprint_collector cmd/fingerprint_stats; do go build -o /dev/null ./$d || echo "FAIL $d"; done; grep -rn "UseCustomCDP" cmd || echo "no UseCustomCDP left"`
Expected: 无 `FAIL`；输出 `no UseCustomCDP left`

- [ ] **Step 2: 文档**

- `README.md`：所有 `UseCustomCDP: true` 示例与说明改为"默认即是"；配置结构里列出 `UseChromedp` 与废弃的 `UseCustomCDP`；新增"迁移说明"小节：
  ```markdown
  ### 迁移：默认通道改为 CustomCDP（2026-10）
  - 不写通道选项时现在走 CustomCDP：主页面、跨站 iframe、Worker、弹窗都下发身份。`UseCustomCDP` 已废弃，可以删掉。
  - 需要旧的 chromedp 通道（例如 `GetContext()` 后直接调用 `chromedp.Run` / `chromedp.ListenTarget`）时，设 `UseChromedp: true`。chromedp 通道只给主页面下发身份，不防检测。
  - 改用 CustomCDP 接口即可不再依赖 chromedp：`EnableNetwork`、`OnNetworkRequest`、`OnNetworkResponse`、`GetResponseBody`、`EnableFetch`、`OnRequestPaused`。
  ```
- `CLAUDE.md`：`Connect — branches on opts.UseCustomCDP` 一段改为按 `opts.UseChromedp` 分支，默认 CustomCDP；`page.(*CDPPage)` 的提示改为两种页面类型都可断言；注明 `NewPage` 在 CustomCDP 实例上开 CustomCDP 页面。

- [ ] **Step 3: 全量验证**

Run（macOS）：

```bash
go vet ./pkg/... ./internal/... && GOOS=linux go vet ./pkg/... ./internal/... && GOOS=windows go vet ./pkg/... ./internal/...
go test -race -count=1 -timeout 1500s ./pkg/... ./internal/...
```

Run（Linux 同构环境）：`scripts/test-linux-chrome.sh`
Expected: 全部 `ok`

- [ ] **Step 4: 提交**

```bash
git add cmd README.md CLAUDE.md
git commit -m "docs: CustomCDP is the default channel; migrate examples to UseChromedp where needed

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
```
