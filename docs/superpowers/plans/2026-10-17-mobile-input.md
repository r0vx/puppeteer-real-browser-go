# 移动输入（M2）Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 手机身份下，`Click`、`RealClick`、`ClickSelector`、`RealClickSelector`、`RealScroll`、`RealSendKeys`、`SendKeys` 按安卓真机的方式发出：手指点按、手指拖动、输入法组字。事件序列与真机基准一致。

**Architecture:**
- 新文件 `mobile_input.go` 提供只依赖"发送 CDP 命令"函数的过程：`mobileTap`、`mobileSwipe`、`mobileScroll`、`mobileType`。
- `CustomCDPPage` 的输入方法在手机身份下转到这些过程。
- 浏览器开着触屏模拟（M1），原生合成 pointer、touch、兼容鼠标、`click`、组字与 `beforeinput` / `input` 事件。

**Tech Stack:** Go 1.25，CDP `Input.dispatchTouchEvent`、`Input.dispatchKeyEvent`、`Input.imeSetComposition`、`Input.insertText`。

**Spec:** `docs/superpowers/specs/2026-10-17-mobile-input-design.md`

## Global Constraints

- 桌面身份的输入行为完全不变，现有 `TestInputLooksHuman` 照常通过。
- 只改默认通道的 `CustomCDPPage`：手机身份不支持 chromedp 通道。
- 真机节奏取自 `test/baseline/android-chrome-pgkm10.json`：
  - 点按的触点半径依次为 0.104、0.229、0.25，按住约 90ms；
  - 打字：`keyCode` 229、`key` Unidentified、`code` 为空，组字方式；
  - 回车：`key` Enter、`code` 为空、`keyCode` 13；
  - 滑动：`touchmove` 间隔约 16ms。
- 代码注释用中文，每个函数都有注释；提交信息以 `Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>` 结尾。

## Review Focus

1. 点按在页面上产生的完整事件序列（含兼容鼠标事件与 `click`）与真机一致，`click` 的 `pointerType` 为 touch。由 Task 2 的 `TestAndroidTapAndTypeMatchBaseline` 覆盖。
2. 组字输入后输入框的值正好是请求的文本，含空格与换行时没有重复或丢字。由 Task 1 的 `TestMobileTypeSequence`、`TestMobileTypeSpaceAndEnter` 与 Task 2 的 `TestAndroidEnterKey` 覆盖。
3. 长距离滚动拆成多次拖动，最终滚动距离接近请求值，不被惯性甩出很远。由 Task 2 的 `TestAndroidSwipeScrolls` 覆盖。
4. 桌面身份不受影响。由现有的 `TestInputLooksHuman` 覆盖（Task 2 Step 4 运行）。
5. 手机身份上 `RealHover` 不产生指针事件（触屏没有悬停）。由 Task 2 的 `TestAndroidTapAndTypeMatchBaseline` 覆盖。

---

### Task 1: 手机输入过程

**Files:**
- Create: `pkg/browser/mobile_input.go`
- Test: `pkg/browser/mobile_input_test.go`

**Interfaces:**
- Produces：
  - `type cdpSend func(method string, params any) error`；
  - `var tapRadii []float64`；
  - `mobileTap(send cdpSend, x, y float64) error`；
  - `mobileSwipe(send cdpSend, x0, y0, x1, y1 float64) error`；
  - `mobileScroll(send cdpSend, viewportW, viewportH int, deltaX, deltaY float64) error`；
  - `mobileType(send cdpSend, text string) error`。
  - 触点参数里的 `touchPoints` 为 `[]map[string]any`。

- [ ] **Step 1: 写失败的测试**

创建 `pkg/browser/mobile_input_test.go`：

```go
package browser

import (
	"math"
	"strings"
	"testing"
)

// sentCDP 发出的一条 CDP 命令
type sentCDP struct {
	Method string
	Params map[string]any
}

// recordCDP 记录发出的 CDP 命令，不真正发送
func recordCDP() (cdpSend, *[]sentCDP) {
	var log []sentCDP
	return func(method string, params any) error {
		p, _ := params.(map[string]any)
		log = append(log, sentCDP{method, p})
		return nil
	}, &log
}

// touchPointOf 触摸事件命令里的第一个触点；没有触点时返回 nil
func touchPointOf(c sentCDP) map[string]any {
	points, _ := c.Params["touchPoints"].([]map[string]any)
	if len(points) == 0 {
		return nil
	}
	return points[0]
}

// TestMobileTapSequence 一次点按：按下、两次接触面变大、抬起；同一个点，坐标在目标 ±1.5 像素内、精度 1/24 像素
func TestMobileTapSequence(t *testing.T) {
	send, log := recordCDP()
	if err := mobileTap(send, 100, 200); err != nil {
		t.Fatal(err)
	}
	var types []string
	for _, c := range *log {
		if c.Method != "Input.dispatchTouchEvent" {
			t.Fatalf("unexpected command %s", c.Method)
		}
		types = append(types, c.Params["type"].(string))
	}
	if strings.Join(types, ",") != "touchStart,touchMove,touchMove,touchEnd" {
		t.Fatalf("touch types %v", types)
	}
	first := touchPointOf((*log)[0])
	for i, c := range (*log)[:3] {
		p := touchPointOf(c)
		if p["radiusX"] != tapRadii[i] || p["radiusY"] != tapRadii[i] || p["x"] != first["x"] || p["y"] != first["y"] {
			t.Errorf("touch %d = %v", i, p)
		}
	}
	for _, v := range []float64{first["x"].(float64) - 100, first["y"].(float64) - 200} {
		if math.Abs(v) > 1.5 || math.Abs(v*24-math.Round(v*24)) > 1e-9 {
			t.Errorf("finger offset %v is not within ±1.5 px on a 1/24 px grid", v)
		}
	}
	if touchPointOf((*log)[3]) != nil {
		t.Error("touchEnd carries touch points")
	}
}

// imeKeyOK 输入法按键：key Unidentified、code 为空、keyCode 229
func imeKeyOK(c sentCDP, typ string) bool {
	return c.Method == "Input.dispatchKeyEvent" && c.Params["type"] == typ && c.Params["key"] == "Unidentified" &&
		c.Params["code"] == "" && c.Params["windowsVirtualKeyCode"] == 229
}

// TestMobileTypeSequence 组字输入：每个字 keydown(229) → 组字为当前整个词 → keyup(229)，最后提交整个词
func TestMobileTypeSequence(t *testing.T) {
	send, log := recordCDP()
	if err := mobileType(send, "ab1"); err != nil {
		t.Fatal(err)
	}
	l := *log
	if len(l) != 10 {
		t.Fatalf("%d commands, want 10: %v", len(l), l)
	}
	for i, want := range []string{"a", "ab", "ab1"} {
		down, comp, up := l[i*3], l[i*3+1], l[i*3+2]
		if !imeKeyOK(down, "rawKeyDown") || !imeKeyOK(up, "keyUp") {
			t.Errorf("char %d keys %v / %v", i, down, up)
		}
		if comp.Method != "Input.imeSetComposition" || comp.Params["text"] != want || comp.Params["selectionStart"] != len(want) {
			t.Errorf("char %d composition %v", i, comp)
		}
	}
	if c := l[9]; c.Method != "Input.insertText" || c.Params["text"] != "ab1" {
		t.Errorf("commit %v", c)
	}
}

// TestMobileTypeSpaceAndEnter 空格与换行先提交当前词；空格直接提交，换行是真机那样的 Enter 键
func TestMobileTypeSpaceAndEnter(t *testing.T) {
	send, log := recordCDP()
	if err := mobileType(send, "a b\n"); err != nil {
		t.Fatal(err)
	}
	var got []string
	for _, c := range *log {
		switch c.Method {
		case "Input.insertText", "Input.imeSetComposition":
			got = append(got, c.Method[6:]+":"+c.Params["text"].(string))
		case "Input.dispatchKeyEvent":
			got = append(got, c.Params["type"].(string)+":"+c.Params["key"].(string))
		}
	}
	want := "rawKeyDown:Unidentified,imeSetComposition:a,keyUp:Unidentified,insertText:a," +
		"rawKeyDown:Unidentified,insertText: ,keyUp:Unidentified," +
		"rawKeyDown:Unidentified,imeSetComposition:b,keyUp:Unidentified,insertText:b," +
		"keyDown:Enter,keyUp:Enter"
	if strings.Join(got, ",") != want {
		t.Errorf("sequence\n %s\nwant\n %s", strings.Join(got, ","), want)
	}
	for _, c := range *log {
		if c.Params["key"] == "Enter" && (c.Params["code"] != "" || c.Params["windowsVirtualKeyCode"] != 13) {
			t.Errorf("Enter key %v, want code \"\" and keyCode 13", c.Params)
		}
	}
}

// TestMobileScrollSplits 长距离滚动拆成多次拖动（每次不超过视口的 60%），手指与滚动方向相反，总拖动距离等于请求值
func TestMobileScrollSplits(t *testing.T) {
	send, log := recordCDP()
	if err := mobileScroll(send, 400, 800, 0, 1000); err != nil {
		t.Fatal(err)
	}
	var starts int
	var total, startY float64
	var lastY float64
	for _, c := range *log {
		switch c.Params["type"] {
		case "touchStart":
			starts++
			startY = touchPointOf(c)["y"].(float64)
		case "touchMove":
			lastY = touchPointOf(c)["y"].(float64)
		case "touchEnd":
			drag := startY - lastY
			if drag <= 0 || drag > 480+0.1 {
				t.Errorf("swipe dragged %v px (finger must move up, at most 60%% of the viewport)", drag)
			}
			total += drag
		}
	}
	if starts != 3 || math.Abs(total-1000) > 0.5 {
		t.Errorf("%d swipes dragging %v px in total, want 3 swipes and 1000 px", starts, total)
	}
}
```

- [ ] **Step 2: 运行，确认失败**

Run: `go test -count=1 -run 'TestMobile' ./pkg/browser/`
Expected: FAIL，编译错误（`undefined: mobileTap`、`cdpSend` 等）

- [ ] **Step 3: 实现**

创建 `pkg/browser/mobile_input.go`：

```go
package browser

import (
	"math"
	"math/rand"
	"time"
	"unicode/utf16"
)

// 移动输入（移动身份子项目 M2）：手机身份下的点按、滑动、组字输入，按真机节奏经 CDP 发出，
// 由浏览器原生合成 pointer / touch / 兼容鼠标 / 组字事件。真机实测见 docs/superpowers/specs/2026-10-17-mobile-input-design.md

// cdpSend 发送一条 CDP 命令（只关心是否出错）
type cdpSend func(method string, params any) error

// tapRadii 一次点按里触点半径的变化（真机实测：按下 0.104，接触面随按压变大到 0.229、0.25）
var tapRadii = []float64{0.104, 0.229, 0.25}

// subPixel 触点坐标按 1/24 CSS 像素取整（与真机触点精度同级）
func subPixel(v float64) float64 {
	return math.Round(v*24) / 24
}

// fingerJitter 手指不会正好落在目标点：±1.5 CSS 像素以内的随机偏移
func fingerJitter(v float64) float64 {
	return subPixel(v + (rand.Float64()-0.5)*3)
}

// between 返回 [lo, hi) 毫秒内的随机时长
func between(lo, hi int) time.Duration {
	return time.Duration(lo+rand.Intn(hi-lo)) * time.Millisecond
}

// touchEvent Input.dispatchTouchEvent 的参数：单指触点；抬起时不带触点
func touchEvent(typ string, x, y, radius float64) map[string]any {
	points := []map[string]any{}
	if typ != "touchEnd" {
		points = append(points, map[string]any{"x": x, "y": y, "radiusX": radius, "radiusY": radius, "force": 1, "id": 0})
	}
	return map[string]any{"type": typ, "touchPoints": points}
}

// mobileTap 一次手指点按：按下、接触面两次变大、抬起（按住约 70–130ms，真机约 90ms）
func mobileTap(send cdpSend, x, y float64) error {
	x, y = fingerJitter(x), fingerJitter(y)
	waits := []time.Duration{between(30, 60), between(30, 50), between(8, 25)}
	for i, r := range tapRadii {
		typ := "touchMove"
		if i == 0 {
			typ = "touchStart"
		}
		if err := send("Input.dispatchTouchEvent", touchEvent(typ, x, y, r)); err != nil {
			return err
		}
		time.Sleep(waits[i])
	}
	return send("Input.dispatchTouchEvent", touchEvent("touchEnd", x, y, 0))
}

// mobileSwipe 一次手指拖动：每 16ms 一个 touchMove，按缓出曲线从 (x0, y0) 移到 (x1, y1)；
// 到达后停住再抬起——浏览器按抬起前约 100ms 内的速度决定惯性滚动，停住就不会甩出去，滚动距离接近拖动距离
func mobileSwipe(send cdpSend, x0, y0, x1, y1 float64) error {
	if err := send("Input.dispatchTouchEvent", touchEvent("touchStart", subPixel(x0), subPixel(y0), tapRadii[0])); err != nil {
		return err
	}
	steps := 8 + rand.Intn(8) // 128–240ms
	for i := 1; i <= steps; i++ {
		time.Sleep(16 * time.Millisecond)
		e := 1 - math.Pow(1-float64(i)/float64(steps), 3)
		if err := send("Input.dispatchTouchEvent", touchEvent("touchMove", subPixel(x0+(x1-x0)*e), subPixel(y0+(y1-y0)*e), tapRadii[2])); err != nil {
			return err
		}
	}
	time.Sleep(between(110, 160))
	return send("Input.dispatchTouchEvent", touchEvent("touchEnd", x1, y1, 0))
}

// mobileScroll 用手指拖动滚动 (deltaX, deltaY)：手指方向与滚动方向相反；
// 一次拖动不超过视口的 60%，更长的距离分几次拖，中间停一下；最后等滚动停下
func mobileScroll(send cdpSend, viewportW, viewportH int, deltaX, deltaY float64) error {
	maxX, maxY := float64(viewportW)*0.6, float64(viewportH)*0.6
	clamp := func(v, limit float64) float64 { return math.Max(-limit, math.Min(limit, v)) }
	for i := 0; math.Abs(deltaX) >= 1 || math.Abs(deltaY) >= 1; i++ {
		if i > 0 {
			time.Sleep(between(120, 250))
		}
		dx, dy := clamp(deltaX, maxX), clamp(deltaY, maxY)
		cx := float64(viewportW)/2 + (rand.Float64()-0.5)*float64(viewportW)*0.2
		cy := float64(viewportH)/2 + (rand.Float64()-0.5)*float64(viewportH)*0.1
		if err := mobileSwipe(send, cx+dx/2, cy+dy/2, cx-dx/2, cy-dy/2); err != nil {
			return err
		}
		deltaX, deltaY = deltaX-dx, deltaY-dy
	}
	time.Sleep(300 * time.Millisecond)
	return nil
}

// imeKey 输入法按键事件：key 为 Unidentified、code 为空、keyCode 为 229（安卓真机实测）
func imeKey(send cdpSend, typ string) error {
	return send("Input.dispatchKeyEvent", map[string]any{"type": typ, "key": "Unidentified", "code": "", "windowsVirtualKeyCode": 229, "nativeVirtualKeyCode": 229})
}

// enterKey 输入法上的回车：key Enter、code 为空、keyCode 13（真机实测），浏览器据此产生 keypress 与 insertLineBreak
func enterKey(send cdpSend) error {
	down := map[string]any{"type": "keyDown", "key": "Enter", "code": "", "windowsVirtualKeyCode": 13, "nativeVirtualKeyCode": 13, "text": "\r", "unmodifiedText": "\r"}
	if err := send("Input.dispatchKeyEvent", down); err != nil {
		return err
	}
	time.Sleep(between(1, 4))
	return send("Input.dispatchKeyEvent", map[string]any{"type": "keyUp", "key": "Enter", "code": "", "windowsVirtualKeyCode": 13, "nativeVirtualKeyCode": 13})
}

// mobileType 按手机输入法的组字方式输入：每个字是 keydown（229）→ 组字更新为当前整个词 → keyup（229）；
// 空格与换行先提交当前词；文本结束时提交，输入框里的值就是 text
func mobileType(send cdpSend, text string) error {
	var word []rune
	commit := func() error {
		if len(word) == 0 {
			return nil
		}
		err := send("Input.insertText", map[string]any{"text": string(word)})
		word = word[:0]
		return err
	}
	first := true
	for _, r := range text {
		if !first {
			time.Sleep(between(120, 260))
		}
		first = false
		switch r {
		case '\n':
			if err := commit(); err != nil {
				return err
			}
			if err := enterKey(send); err != nil {
				return err
			}
		case ' ':
			if err := commit(); err != nil {
				return err
			}
			if err := imeKey(send, "rawKeyDown"); err != nil {
				return err
			}
			if err := send("Input.insertText", map[string]any{"text": " "}); err != nil {
				return err
			}
			time.Sleep(between(5, 10))
			if err := imeKey(send, "keyUp"); err != nil {
				return err
			}
		default:
			if err := imeKey(send, "rawKeyDown"); err != nil {
				return err
			}
			word = append(word, r)
			n := len(utf16.Encode(word))
			if err := send("Input.imeSetComposition", map[string]any{"text": string(word), "selectionStart": n, "selectionEnd": n}); err != nil {
				return err
			}
			time.Sleep(between(5, 10))
			if err := imeKey(send, "keyUp"); err != nil {
				return err
			}
		}
	}
	return commit()
}
```

- [ ] **Step 4: 运行，确认通过**

Run: `go vet ./pkg/browser/ && go test -count=1 -run 'TestMobile' ./pkg/browser/`
Expected: PASS

- [ ] **Step 5: 提交**

```bash
git add pkg/browser/mobile_input.go pkg/browser/mobile_input_test.go
git commit -m "feat(input): finger taps, swipes and IME composition for mobile identities

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
```

---

### Task 2: 接入页面方法并与真机对照

**Files:**
- Modify: `pkg/browser/cdp_custom.go`（`Click`、`RealClick`、`RealHover`、`RealScroll`、`RealSendKeys`、`SendKeys`）
- Test: `pkg/browser/mobile_input_integration_test.go`

**Interfaces:**
- Consumes：
  - Task 1 的全部 Produces；
  - M1 的 `connectAndroid`、`collectOn`（`android_integration_test.go`）；
  - M0 的 `baselineServer`（`baseline_collector_test.go`）。
- Produces: `(*CustomCDPPage).mobileSend() cdpSend`（手机身份时返回发送函数，桌面身份返回 nil）。

- [ ] **Step 1: 写失败的测试**

创建 `pkg/browser/mobile_input_integration_test.go`：

```go
package browser

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"slices"
	"sort"
	"strings"
	"testing"
	"time"
)

// tapSequence 一步里的事件类型序列：去掉 pointerrawupdate（是否触发取决于硬件采样），连续重复的类型只留一个
func tapSequence(events []map[string]any, step string) []string {
	var out []string
	for _, e := range events {
		typ := fmt.Sprint(e["type"])
		if e["step"] != step || typ == "pointerrawupdate" {
			continue
		}
		if len(out) == 0 || out[len(out)-1] != typ {
			out = append(out, typ)
		}
	}
	return out
}

// firstOf 一步里第一个指定类型的事件
func firstOf(events []map[string]any, step, typ string) map[string]any {
	for _, e := range events {
		if e["step"] == step && e["type"] == typ {
			return e
		}
	}
	return nil
}

// baselineInteraction 读取安卓真机基准的交互记录
func baselineInteraction(t *testing.T) []map[string]any {
	t.Helper()
	raw, err := os.ReadFile("../../test/baseline/android-chrome-pgkm10.json")
	if err != nil {
		t.Fatal(err)
	}
	var r struct {
		Interaction struct {
			Events []map[string]any `json:"events"`
		} `json:"interaction"`
	}
	if err := json.Unmarshal(raw, &r); err != nil {
		t.Fatal(err)
	}
	return r.Interaction.Events
}

// TestAndroidTapAndTypeMatchBaseline 在采集页上用库的方法点按钮、输入 ab1，与真机的事件序列与属性比较；
// RealHover 在手机上不产生指针事件（Review Focus 1、2、5）
func TestAndroidTapAndTypeMatchBaseline(t *testing.T) {
	srv, posted := baselineServer(t, "", http.StatusOK)
	p := connectAndroid(t, "android-input", "PGKM10").Page()
	sp := p.(PageWithSelector)
	collectOn(t, p, srv)
	if err := sp.RealHover(50, 50); err != nil {
		t.Fatal(err)
	}
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
	got, want := body.Interaction.Events, baselineInteraction(t)

	for _, e := range got {
		if e["step"] == "other" && strings.HasPrefix(fmt.Sprint(e["type"]), "pointer") {
			t.Errorf("RealHover on a phone produced %v", e["type"])
		}
	}
	if g, w := tapSequence(got, "tap"), tapSequence(want, "tap"); !slices.Equal(g, w) {
		t.Errorf("tap sequence\n %v\nwant\n %v", g, w)
	}
	down, up, click := firstOf(got, "tap", "pointerdown"), firstOf(got, "tap", "pointerup"), firstOf(got, "tap", "click")
	if down == nil || up == nil || click == nil {
		t.Fatalf("tap incomplete: down %v up %v click %v", down, up, click)
	}
	if down["pointerType"] != "touch" || down["pressure"] != 1.0 || down["width"].(float64) >= 1 || down["isPrimary"] != true {
		t.Errorf("pointerdown %v", down)
	}
	if up["pressure"] != 0.0 || click["pointerType"] != "touch" {
		t.Errorf("pointerup %v / click %v", up, click)
	}
	if hold := up["rt"].(float64) - down["rt"].(float64); hold < 50 || hold > 150 {
		t.Errorf("finger held %.0fms, want 50–150ms", hold)
	}

	var updates []string
	for _, e := range got {
		if e["step"] != "type" {
			continue
		}
		switch e["type"] {
		case "keydown", "keyup":
			if e["key"] != "Unidentified" || e["code"] != "" || e["keyCode"] != 229.0 {
				t.Errorf("%s %v, want Unidentified / \"\" / 229", e["type"], e)
			}
		case "compositionupdate":
			updates = append(updates, fmt.Sprint(e["data"]))
		case "beforeinput":
			if e["inputType"] != "insertCompositionText" && e["inputType"] != "insertFromComposition" && e["inputType"] != "deleteCompositionText" {
				t.Errorf("beforeinput %v", e["inputType"])
			}
		}
	}
	if strings.Join(updates, ",") != "a,ab,ab1" {
		t.Errorf("composition updates %v, want a,ab,ab1", updates)
	}
	if firstOf(got, "type", "compositionstart") == nil {
		t.Error("no compositionstart")
	}
}

// swipePage 可以滚动的长页面，记录触摸与指针取消事件
const swipePage = `<!doctype html><meta name="viewport" content="width=device-width, initial-scale=1">
<div style="height: 4000px; background: linear-gradient(#eef, #fee)"></div>
<script>
window.__t = [];
for (const type of ['touchstart', 'touchmove', 'touchend', 'pointercancel'])
  addEventListener(type, e => window.__t.push({type: e.type, cancelable: e.cancelable, t: performance.now()}), {passive: true});
</script>`

// TestAndroidSwipeScrolls 手机上 RealScroll 是手指拖动：有 touchstart / touchmove / touchend 与 pointercancel，
// touchmove 全部不可取消、间隔约 16ms，最终滚动距离接近请求值（Review Focus 3）
func TestAndroidSwipeScrolls(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { fmt.Fprint(w, swipePage) }))
	defer srv.Close()
	p := connectAndroid(t, "android-swipe", "PGKM10").Page()
	if err := p.Navigate(srv.URL); err != nil {
		t.Fatal(err)
	}
	if err := p.(PageWithSelector).RealScroll(0, 900); err != nil {
		t.Fatal(err)
	}
	time.Sleep(500 * time.Millisecond)
	if y, _ := p.Evaluate(`scrollY`); y.(float64) < 900*0.7 || y.(float64) > 900*1.3 {
		t.Errorf("scrollY = %v after RealScroll(0, 900)", y)
	}
	raw, _ := p.Evaluate(`JSON.stringify(window.__t)`)
	var events []map[string]any
	json.Unmarshal([]byte(fmt.Sprint(raw)), &events)
	counts := map[string]int{}
	var moves []float64
	for _, e := range events {
		counts[fmt.Sprint(e["type"])]++
		if e["type"] == "touchmove" {
			if e["cancelable"] != false {
				t.Errorf("touchmove cancelable = %v", e["cancelable"])
			}
			moves = append(moves, e["t"].(float64))
		}
	}
	if counts["touchstart"] < 2 || counts["touchend"] != counts["touchstart"] || counts["pointercancel"] == 0 || len(moves) < 10 {
		t.Errorf("event counts %v", counts)
	}
	var gaps []float64
	for i := 1; i < len(moves); i++ {
		if d := moves[i] - moves[i-1]; d < 100 { // 跳过两次拖动之间的停顿
			gaps = append(gaps, d)
		}
	}
	sort.Float64s(gaps)
	if len(gaps) == 0 || gaps[len(gaps)/2] < 12 || gaps[len(gaps)/2] > 24 {
		t.Errorf("median touchmove gap %v, want 12–24ms", gaps)
	}
}

// enterPage 一个输入框，记录按键与 beforeinput
const enterPage = `<!doctype html><meta name="viewport" content="width=device-width, initial-scale=1">
<input id=i style="font-size: 18px">
<script>
window.__k = [];
for (const type of ['keydown', 'keypress', 'keyup', 'beforeinput'])
  addEventListener(type, e => window.__k.push([e.type, e.key || '', e.code || '', e.keyCode || 0, e.inputType || ''].join('|')), true);
</script>`

// TestAndroidEnterKey 换行是真机那样的 Enter 键：key Enter、code 为空、keyCode 13，并产生 insertLineBreak
func TestAndroidEnterKey(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { fmt.Fprint(w, enterPage) }))
	defer srv.Close()
	p := connectAndroid(t, "android-enter", "PGKM10").Page()
	if err := p.Navigate(srv.URL); err != nil {
		t.Fatal(err)
	}
	if err := p.(PageWithSelector).SendKeys("#i", "x\n"); err != nil {
		t.Fatal(err)
	}
	raw, _ := p.Evaluate(`window.__k.join(',')`)
	got := fmt.Sprint(raw)
	for _, want := range []string{"keydown|Enter||13|", "keypress|Enter||13|", "beforeinput||||insertLineBreak", "keyup|Enter||13|"} {
		if !strings.Contains(got, want) {
			t.Errorf("events %s lack %s", got, want)
		}
	}
	if v, _ := p.Evaluate(`document.getElementById('i').value`); v != "x" {
		t.Errorf("input value %v, want x", v)
	}
}
```

- [ ] **Step 2: 运行，确认失败**

Run: `go test -count=1 -run 'TestAndroidTapAndTypeMatchBaseline|TestAndroidSwipeScrolls|TestAndroidEnterKey' ./pkg/browser/`
Expected: FAIL：点按序列里是鼠标事件（`pointerType` 为 mouse）；按键的 `keyCode` 是 65 等；滑动没有 `touchstart`。

- [ ] **Step 3: 实现**

`pkg/browser/cdp_custom.go`：

(a) 在 `// Click 移到 (x, y) 后点击（不走拟人轨迹）；坐标对齐设备像素` 之前加：

```go
// mobileSend 手机身份时返回发送 CDP 命令的函数（输入改走手指与输入法），桌面身份返回 nil
func (p *CustomCDPPage) mobileSend() cdpSend {
	if p.identity == nil || !p.identity.Screen.Mobile {
		return nil
	}
	return func(method string, params any) error {
		_, err := p.client.sendCommand(method, params)
		return err
	}
}
```

(b) `Click` 与 `RealClick` 的函数体开头各加：

```go
	if send := p.mobileSend(); send != nil {
		return mobileTap(send, x, y) // 手机：手指点按（没有鼠标轨迹）
	}
```

(c) `RealHover` 的函数体开头加：

```go
	if p.mobileSend() != nil {
		return nil // 触屏没有悬停
	}
```

(d) `RealScroll` 的函数体开头加：

```go
	if send := p.mobileSend(); send != nil {
		s := p.identity.Screen
		return mobileScroll(send, s.InnerWidth, s.InnerHeight, deltaX, deltaY) // 手机：手指拖动
	}
```

(e) `RealSendKeys` 改为：

```go
// RealSendKeys 向当前焦点元素逐键输入：桌面为真实按键（key / code / keyCode 齐全），手机为输入法组字，见 Type 与 mobileType
func (p *CustomCDPPage) RealSendKeys(text string) error {
	if send := p.mobileSend(); send != nil {
		return mobileType(send, text)
	}
	return p.client.Type(text)
}
```

(f) `SendKeys` 的最后一行 `return p.client.Type(text)` 改为 `return p.RealSendKeys(text)`。

- [ ] **Step 4: 运行，确认通过**

Run: `go vet ./pkg/browser/ && go test -count=1 -run 'TestAndroidTapAndTypeMatchBaseline|TestAndroidSwipeScrolls|TestAndroidEnterKey|TestInputLooksHuman|TestMobile' ./pkg/browser/`
Expected: PASS

如果点按序列与真机不同（例如浏览器多发或少发某类事件）：用 `t.Log` 打出两边的序列，判断是浏览器原生合成的差异，还是我们发的触摸事件不对。
- 我们发错了：修代码。
- 浏览器原生合成、无法控制：在 ledger 记 Ruling，并在测试里写明。

- [ ] **Step 5: 提交**

```bash
git add pkg/browser/cdp_custom.go pkg/browser/mobile_input_integration_test.go
git commit -m "feat(input): mobile identities tap, swipe and type like the real phone

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
```

---

### Task 3: 文档与全量测试

**Files:**
- Modify: `README.md`、`CLAUDE.md`

- [ ] **Step 1: 文档**

`README.md`："### 手机身份（安卓，2026-10）"一节里，把这条限制

```
  - 在手机身份上调用 `Click` / `RealSendKeys` 仍是鼠标和桌面键盘事件（子项目 M2 改为触摸与输入法）；
```

替换为一条新的要点（放在"已知限制"之前）：

```
- 手机身份上的页面操作按真机方式发出：`Click` / `RealClick` / `ClickSelector` 是手指点按（含兼容鼠标事件与 click），`RealScroll` 是手指拖动，`RealSendKeys` / `SendKeys` 是输入法组字（`keyCode` 229），换行是 Enter 键；`RealHover` 在手机上不做任何事。
```

`CLAUDE.md`："Mobile (Android) identities" 一条末尾加：

```
 Input on mobile identities goes through `mobile_input.go` (finger taps via `Input.dispatchTouchEvent`, swipes, IME composition via `Input.imeSetComposition`) — `CustomCDPPage` methods switch on `identity.Screen.Mobile`.
```

- [ ] **Step 2: 全量测试**

Run: `go vet ./pkg/... ./internal/... && GOOS=linux go vet ./pkg/... ./internal/... && GOOS=windows go vet ./pkg/... ./internal/... && go test -race -count=1 -timeout 1500s ./cmd/baseline_server/ ./pkg/... ./internal/... && scripts/test-linux-chrome.sh`
Expected: 全部 PASS

- [ ] **Step 3: 提交**

```bash
git add README.md CLAUDE.md
git commit -m "docs: mobile identities tap, swipe and type like phones

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
```
