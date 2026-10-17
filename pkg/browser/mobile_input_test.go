package browser

import (
	"math"
	"strings"
	"sync"
	"testing"
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

// TestMobileTypeControlChars 控制字符与桌面效果一致：\r 同 \n 是回车；组字中的退格缩短组字，没有组字时发退格键；其他控制字符报错
func TestMobileTypeControlChars(t *testing.T) {
	summary := func(text string) (string, error) {
		send, log := recordCDP()
		err := mobileType(send, text)
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
