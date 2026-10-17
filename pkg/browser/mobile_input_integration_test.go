package browser

import (
	"encoding/json"
	"fmt"
	"math"
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

// baselineInteraction 读取真机基准（test/baseline/ 下的文件）的交互记录
func baselineInteraction(t *testing.T, file string) []map[string]any {
	t.Helper()
	raw, err := os.ReadFile("../../test/baseline/" + file)
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
	sp := p.(*CustomCDPPage) // 手机身份只走默认通道
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
	got, want := body.Interaction.Events, baselineInteraction(t, "android-chrome-pgkm10.json")

	// RealHover 在第一次点按之前调用：在那之前不能有任何指针事件
	for _, e := range got {
		if e["step"] == "tap" {
			break
		}
		if strings.HasPrefix(fmt.Sprint(e["type"]), "pointer") {
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
	// 屏幕坐标比页面坐标多出视口上方的状态栏与地址栏（真机实测 88）
	for _, e := range []map[string]any{down, up, click, firstOf(got, "tap", "mousedown")} {
		if e == nil {
			continue
		}
		if off := e["screenY"].(float64) - e["clientY"].(float64); math.Abs(off-88) > 1 {
			t.Errorf("%s screenY - clientY = %v, want 88", e["type"], off)
		}
	}
	if hold := up["rt"].(float64) - down["rt"].(float64); hold < 50 || hold > 150 {
		t.Errorf("finger held %.0fms, want 50–150ms", hold)
	}

	var updates []string
	var downT any
	for _, e := range got {
		if e["step"] != "type" {
			continue
		}
		switch e["type"] {
		case "keydown", "keyup":
			if e["key"] != "Unidentified" || e["code"] != "" || e["keyCode"] != 229.0 {
				t.Errorf("%s %v, want Unidentified / \"\" / 229", e["type"], e)
			}
			// 真机的每次输入法改动共用一个时间戳：keydown 与 keyup 的 timeStamp 相同
			if e["type"] == "keydown" {
				downT = e["t"]
			} else if e["t"] != downT {
				t.Errorf("keyup t = %v, keydown t = %v; the phone stamps both alike", e["t"], downT)
			}
		case "compositionupdate":
			// 提交组字时 Blink 会再发一次内容相同的 compositionupdate（真机按空格、选词时也一样），连续相同的只算一次
			if d := fmt.Sprint(e["data"]); len(updates) == 0 || updates[len(updates)-1] != d {
				updates = append(updates, d)
			}
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
window.__clicks = 0;
addEventListener('click', () => window.__clicks++, true);
addEventListener('touchstart', e => window.__touchOff = e.touches[0].screenY - e.touches[0].clientY, {passive: true});
for (const type of ['touchstart', 'touchmove', 'touchend', 'pointercancel'])
  addEventListener(type, e => window.__t.push({type: e.type, cancelable: e.cancelable, t: performance.now(), y: scrollY}), {passive: true});
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
	if err := p.(*CustomCDPPage).RealScroll(0, 900); err != nil {
		t.Fatal(err)
	}
	time.Sleep(500 * time.Millisecond)
	if y, _ := p.Evaluate(`scrollY`); y.(float64) < 900-30 || y.(float64) > 900+30 {
		trail, _ := p.Evaluate(`JSON.stringify(window.__t.filter(e => e.type !== 'touchmove').map(e => e.type + '@' + Math.round(e.t) + ':' + Math.round(e.y)))`)
		t.Errorf("scrollY = %v after RealScroll(0, 900), want 900 ± 30; touch trail %v", y, trail)
	}
	if off, _ := p.Evaluate(`window.__touchOff`); off != 88.0 {
		t.Errorf("Touch screenY - clientY = %v, want 88", off)
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
	for _, want := range []string{"keydown|Enter||13|", "keypress|Enter||13|", "beforeinput|||0|insertLineBreak", "keyup|Enter||13|"} {
		if !strings.Contains(got, want) {
			t.Errorf("events %s lack %s", got, want)
		}
	}
	if v, _ := p.Evaluate(`document.getElementById('i').value`); v != "x" {
		t.Errorf("input value %v, want x", v)
	}
}

// typeThenTapPage 两个输入框与一个按钮，记录点击
const typeThenTapPage = `<!doctype html><meta name="viewport" content="width=device-width, initial-scale=1">
<input id=a style="font-size: 18px"><br><input id=b style="font-size: 18px"><br><button id=go style="font-size: 18px">go</button>
<script>window.__clicks = []; addEventListener('click', e => window.__clicks.push(e.target.id), true);</script>`

// TestAndroidTypeThenTap 打字之后照样能点按、再打字（用户名、密码、登录的常见流程）；同一个框连续输入两次是追加，不丢字
func TestAndroidTypeThenTap(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { fmt.Fprint(w, typeThenTapPage) }))
	defer srv.Close()
	p := connectAndroid(t, "android-type-tap", "PGKM10").Page()
	sp := p.(*CustomCDPPage)
	if err := p.Navigate(srv.URL); err != nil {
		t.Fatal(err)
	}
	start := time.Now()
	for _, step := range []func() error{
		func() error { return sp.SendKeys("#a", "ab") },
		func() error { return sp.SendKeys("#a", "cd") },
		func() error { return sp.SendKeys("#b", "pw") },
		func() error { return sp.ClickSelector("#go") },
	} {
		if err := step(); err != nil {
			t.Fatalf("after %v: %v", time.Since(start).Round(time.Millisecond), err)
		}
	}
	if d := time.Since(start); d > 15*time.Second {
		t.Errorf("type-then-tap flow took %v", d)
	}
	if v, _ := p.Evaluate(`[document.getElementById('a').value, document.getElementById('b').value, window.__clicks.join()].join('|')`); v != "abcd|pw|a,a,b,go" {
		t.Errorf("values / clicks = %v, want abcd|pw|a,a,b,go", v)
	}
}

// TestAndroidControlChars 退格在页面上的效果与桌面一致：组字中删组字，没有组字时删已提交的字
func TestAndroidControlChars(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { fmt.Fprint(w, enterPage) }))
	defer srv.Close()
	p := connectAndroid(t, "android-ctrl", "PGKM10").Page()
	sp := p.(*CustomCDPPage)
	if err := p.Navigate(srv.URL); err != nil {
		t.Fatal(err)
	}
	if err := sp.SendKeys("#i", "ab\bc"); err != nil {
		t.Fatal(err)
	}
	if err := sp.RealSendKeys("\b"); err != nil {
		t.Fatal(err)
	}
	if v, _ := p.Evaluate(`document.getElementById('i').value`); v != "a" {
		t.Errorf("value %v, want a (ab, backspace, c, then backspace)", v)
	}
	if err := sp.RealSendKeys("x\ty"); err == nil {
		t.Error("tab was typed instead of rejected")
	}
}

// TestAndroidShortSwipeIsNotATap 很短的滚动也是拖动：不会变成一次点按点中手指下的元素，滚动距离接近请求值（Review Focus 3）
func TestAndroidShortSwipeIsNotATap(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { fmt.Fprint(w, swipePage) }))
	defer srv.Close()
	p := connectAndroid(t, "android-short-swipe", "PGKM10").Page()
	if err := p.Navigate(srv.URL); err != nil {
		t.Fatal(err)
	}
	if err := p.(*CustomCDPPage).RealScroll(0, 8); err != nil {
		t.Fatal(err)
	}
	time.Sleep(300 * time.Millisecond)
	if v, _ := p.Evaluate(`[window.__clicks, Math.round(scrollY)].join()`); v != "0,8" && v != "0,9" && v != "0,7" {
		t.Errorf("clicks, scrollY = %v after RealScroll(0, 8), want no click and about 8 px", v)
	}
}
