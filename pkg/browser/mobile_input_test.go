package browser

import (
	"errors"
	"fmt"
	"math"
	"strings"
	"sync"
	"testing"
	"time"
)

// sentCDP 发出的一条 CDP 命令
type sentCDP struct {
	Method string
	Params map[string]any
}

// recordCDP 记录发出的 CDP 命令，不真正发送（滑动会并发发送，记录时加锁）
func recordCDP() (cdpSend, *[]sentCDP) {
	var mu sync.Mutex
	var log []sentCDP
	return func(method string, params any) error {
		p, _ := params.(map[string]any)
		mu.Lock()
		log = append(log, sentCDP{method, p})
		mu.Unlock()
		return nil
	}, &log
}

// failingCDP 像 recordCDP 一样记录命令，从第 from 条（从 1 数起）开始每条都返回错误
func failingCDP(from int) (cdpSend, *[]sentCDP) {
	var mu sync.Mutex
	var log []sentCDP
	return func(method string, params any) error {
		p, _ := params.(map[string]any)
		mu.Lock()
		defer mu.Unlock()
		log = append(log, sentCDP{method, p})
		if len(log) >= from {
			return errors.New("connection lost")
		}
		return nil
	}, &log
}

// touchTypes 发出的触摸事件类型
func touchTypes(log []sentCDP) string {
	var types []string
	for _, c := range log {
		types = append(types, fmt.Sprint(c.Params["type"]))
	}
	return strings.Join(types, ",")
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
	if err := mobileTap(send, OSAndroid, 100, 200, fingerSlack, fingerSlack); err != nil {
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
		// 按住时只有 1/24 像素级的抖动（真机实测位置不变；Chrome 需要一点位移才生成 pointermove）
		if p["radiusX"] != tapRadii[i] || p["radiusY"] != tapRadii[i] || math.Abs(p["x"].(float64)-first["x"].(float64)) > 0.1 || p["y"] != first["y"] {
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
	if err := mobileType(send, OSAndroid, "ab1", false); err != nil {
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
	if err := mobileType(send, OSAndroid, "a b\n", false); err != nil {
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

// TestMobileScrollSplits 长距离滚动拆成等长的几次拖动（每次不超过视口的 60%），手指与滚动方向相反；
// 每次多拖一个触摸阈值（浏览器在手指移过阈值之后才开始滚动），所以总拖动距离为请求值加上每次的阈值
func TestMobileScrollSplits(t *testing.T) {
	send, log := recordCDP()
	if err := mobileScroll(send, OSAndroid, 400, 800, 0, 1000, nil); err != nil {
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
			if drag <= touchSlop || drag > 480+touchSlop+0.1 {
				t.Errorf("swipe dragged %v px (finger must move up, past the slop, at most 60%% of the viewport plus the slop)", drag)
			}
			total += drag
		}
	}
	if starts != 3 || math.Abs(total-(1000+3*touchSlop)) > 0.5 {
		t.Errorf("%d swipes dragging %v px in total, want 3 equal swipes and %v px", starts, total, 1000+3*touchSlop)
	}
}

// TestMobileTypeControlChars 控制字符与桌面效果一致：\r 同 \n 是回车；组字中的退格缩短组字，没有组字时发退格键；其他控制字符报错
func TestMobileTypeControlChars(t *testing.T) {
	summary := func(text string) (string, error) {
		send, log := recordCDP()
		err := mobileType(send, OSAndroid, text, false)
		var got []string
		for _, c := range *log {
			switch c.Method {
			case "Input.insertText", "Input.imeSetComposition":
				got = append(got, c.Method[6:]+":"+c.Params["text"].(string))
			case "Input.dispatchKeyEvent":
				if k := c.Params["key"]; k != "Unidentified" {
					got = append(got, c.Params["type"].(string)+":"+k.(string))
				}
			}
		}
		return strings.Join(got, ","), err
	}
	cases := []struct{ text, want string }{
		{"a\rb", "imeSetComposition:a,insertText:a,keyDown:Enter,keyUp:Enter,imeSetComposition:b,insertText:b"},
		{"ab\bc", "imeSetComposition:a,imeSetComposition:ab,imeSetComposition:a,imeSetComposition:ac,insertText:ac"},
		{"a\b", "imeSetComposition:a,imeSetComposition:"},
		{"\bx", "rawKeyDown:Backspace,keyUp:Backspace,imeSetComposition:x,insertText:x"},
	}
	for _, tc := range cases {
		if got, err := summary(tc.text); err != nil || got != tc.want {
			t.Errorf("mobileType(%q) = %s (%v)\nwant %s", tc.text, got, err, tc.want)
		}
	}
	if _, err := summary("x\ty"); err == nil || !strings.Contains(err.Error(), "U+0009") {
		t.Errorf("tab: err = %v, want an unsupported control character error", err)
	}
}

// TestIOSTapSequence iOS 点按：按下后直接抬起，中间没有移动；触点半径 12.08、force 0（pointer 宽高 24.16、pressure 0），按住 25–60ms
func TestIOSTapSequence(t *testing.T) {
	send, log := recordCDP()
	start := time.Now()
	if err := mobileTap(send, OSIOS, 100, 200, fingerSlack, fingerSlack); err != nil {
		t.Fatal(err)
	}
	if d := time.Since(start); d < 25*time.Millisecond || d > 80*time.Millisecond {
		t.Errorf("tap took %v, want 25–60ms", d)
	}
	var types []string
	for _, c := range *log {
		types = append(types, c.Params["type"].(string))
	}
	if strings.Join(types, ",") != "touchStart,touchEnd" {
		t.Fatalf("touch types %v", types)
	}
	if p := touchPointOf((*log)[0]); p["radiusX"] != iosTouchRadius || p["radiusY"] != iosTouchRadius || p["force"] != 0.0 {
		t.Errorf("touch point %v", p)
	}
}

// TestIOSTypeKeys iOS 键盘：按键带真实的 key 与 code，keydown 的 keyCode 为 229、keyup 的为按键码；空格同样，回车的 code 为 Enter
func TestIOSTypeKeys(t *testing.T) {
	send, log := recordCDP()
	if err := mobileType(send, OSIOS, "aB1 \n", false); err != nil {
		t.Fatal(err)
	}
	var got []string
	for _, c := range *log {
		if c.Method == "Input.dispatchKeyEvent" {
			got = append(got, fmt.Sprintf("%s:%s:%s:%v", c.Params["type"], c.Params["key"], c.Params["code"], c.Params["windowsVirtualKeyCode"]))
		}
	}
	want := "rawKeyDown:a:KeyA:229,keyUp:a:KeyA:65,rawKeyDown:B:KeyB:229,keyUp:B:KeyB:66,rawKeyDown:1:Digit1:229,keyUp:1:Digit1:49," +
		"rawKeyDown: :Space:229,keyUp: :Space:32,keyDown:Enter:Enter:13,keyUp:Enter:Enter:13"
	if strings.Join(got, ",") != want {
		t.Errorf("keys\n %s\nwant\n %s", strings.Join(got, ","), want)
	}
}

// TestMobileTapStaysOnSmallTarget 手指的随机偏移不超过给定的范围：小元素上的点按不会落到元素外面
func TestMobileTapStaysOnSmallTarget(t *testing.T) {
	for _, os := range []OSFamily{OSAndroid, OSIOS} {
		for i := 0; i < 50; i++ {
			send, log := recordCDP()
			if err := mobileTap(send, os, 100, 200, 0.3, 0.6); err != nil {
				t.Fatal(err)
			}
			for _, c := range *log {
				p := touchPointOf(c)
				if p == nil {
					continue
				}
				// 坐标取整到 1/24 像素最多多出 1/48；按住时的 1–2/24 像素抖动只在 x 方向
				if dx, dy := math.Abs(p["x"].(float64)-100), math.Abs(p["y"].(float64)-200); dx > 0.3+2.0/24+1.0/48+1e-9 || dy > 0.6+1.0/48+1e-9 {
					t.Fatalf("%s finger landed %.3f, %.3f px off, beyond the 0.3 / 0.6 px slack", os, dx, dy)
				}
			}
		}
	}
	if got := tapSlack(4); got != 0.6 {
		t.Errorf("tapSlack(4) = %v, want 0.6 (15%% of the size)", got)
	}
	if got := tapSlack(100); got != fingerSlack {
		t.Errorf("tapSlack(100) = %v, want %v", got, fingerSlack)
	}
}

// TestMobileTapLiftsFingerOnError 手指按下之后出错：补发 touchCancel 抬起手指，返回原来的错误
func TestMobileTapLiftsFingerOnError(t *testing.T) {
	for _, tc := range []struct {
		os   OSFamily
		want string
	}{{OSAndroid, "touchStart,touchMove,touchCancel"}, {OSIOS, "touchStart,touchEnd,touchCancel"}} {
		send, log := failingCDP(2)
		if err := mobileTap(send, tc.os, 100, 200, fingerSlack, fingerSlack); err == nil {
			t.Errorf("%s: no error", tc.os)
		}
		if got := touchTypes(*log); got != tc.want {
			t.Errorf("%s: touch types %s, want %s", tc.os, got, tc.want)
		}
		if p := (*log)[len(*log)-1].Params["touchPoints"].([]map[string]any); len(p) != 0 {
			t.Errorf("%s: touchCancel carries touch points %v", tc.os, p)
		}
	}
	// 按下本身失败：手指没按下，不发 touchCancel
	send, log := failingCDP(1)
	if err := mobileTap(send, OSAndroid, 100, 200, fingerSlack, fingerSlack); err == nil || touchTypes(*log) != "touchStart" {
		t.Errorf("failed touchStart: err %v, commands %s", err, touchTypes(*log))
	}
}

// TestMobileSwipeLiftsFingerOnError 拖动中途出错：不再发 touchEnd，补发 touchCancel
func TestMobileSwipeLiftsFingerOnError(t *testing.T) {
	send, log := failingCDP(3)
	if err := mobileSwipe(send, OSAndroid, 200, 600, 200, 200); err == nil {
		t.Fatal("no error")
	}
	types := touchTypes(*log)
	if !strings.HasSuffix(types, ",touchCancel") || strings.Contains(types, "touchEnd") {
		t.Errorf("touch types %s, want the moves then touchCancel and no touchEnd", types)
	}
}

// TestMobileTypeDirect 密码、数字、电话输入框：每个字直接提交，不组字（iPhone 基准里数字就是这样：keydown 229 → insertText → keyup）
func TestMobileTypeDirect(t *testing.T) {
	for _, tc := range []struct {
		os   OSFamily
		want string
	}{
		{OSAndroid, "rawKeyDown:Unidentified,insertText:a,keyUp:Unidentified,rawKeyDown:Unidentified,insertText:1,keyUp:Unidentified,rawKeyDown:Backspace,keyUp:Backspace"},
		{OSIOS, "rawKeyDown:a,insertText:a,keyUp:a,rawKeyDown:1,insertText:1,keyUp:1,rawKeyDown:Backspace,keyUp:Backspace"},
	} {
		send, log := recordCDP()
		if err := mobileType(send, tc.os, "a1\b", true); err != nil {
			t.Fatal(err)
		}
		var got []string
		for _, c := range *log {
			switch c.Method {
			case "Input.imeSetComposition":
				t.Errorf("%s: composed %v in a direct field", tc.os, c.Params["text"])
			case "Input.insertText":
				got = append(got, "insertText:"+c.Params["text"].(string))
			case "Input.dispatchKeyEvent":
				got = append(got, fmt.Sprintf("%s:%s", c.Params["type"], c.Params["key"]))
			}
		}
		if strings.Join(got, ",") != tc.want {
			t.Errorf("%s: sequence\n %s\nwant\n %s", tc.os, strings.Join(got, ","), tc.want)
		}
	}
}

// TestMobileScrollStopsAtEdge 手指下方的滚动链没有余量时不再拖动（页面到底了真人也不会接着拖）
func TestMobileScrollStopsAtEdge(t *testing.T) {
	for _, tc := range []struct {
		room   int // 还能拖几次
		swipes int
	}{{0, 0}, {1, 1}, {10, 3}} {
		send, log := recordCDP()
		calls := 0
		room := func(x, y, dx, dy float64) bool {
			if dy <= 0 || x <= 0 || y <= 0 {
				t.Errorf("probe at %v, %v for %v, %v", x, y, dx, dy)
			}
			calls++
			return calls <= tc.room
		}
		if err := mobileScroll(send, OSAndroid, 400, 800, 0, 1000, room); err != nil {
			t.Fatal(err)
		}
		if got := strings.Count(touchTypes(*log), "touchStart"); got != tc.swipes {
			t.Errorf("room for %d swipes: %d swipes, want %d", tc.room, got, tc.swipes)
		}
	}
}
