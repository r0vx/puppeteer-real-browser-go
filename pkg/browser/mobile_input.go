package browser

import (
	"fmt"
	"math"
	"math/rand"
	"sync"
	"time"
	"unicode"
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

// mobileTap 一次手指点按：按下、接触面两次变大、抬起（按住约 70–130ms，真机约 90ms）。
// 按住时手指有 1/24 像素级的抖动：位置完全不变时 Chrome 不为接触面变化生成 pointermove，真机会生成
func mobileTap(send cdpSend, x, y float64) error {
	x, y = fingerJitter(x), fingerJitter(y)
	waits := []time.Duration{between(30, 60), between(30, 50), between(18, 30)}
	for i, r := range tapRadii {
		typ, px, py := "touchMove", x+float64(1+rand.Intn(2))/24, y
		if i == 0 {
			typ, px = "touchStart", x
		}
		if err := send("Input.dispatchTouchEvent", touchEvent(typ, px, py, r)); err != nil {
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
	// 按 16ms 节拍发出，不等上一条的确认：确认要等浏览器处理完一帧，逐条等待会把间隔拉长到 33ms。
	// 同一连接按写入顺序处理命令，节拍间隔远大于写入耗时，顺序不会乱
	start := time.Now()
	var wg sync.WaitGroup
	errs := make(chan error, steps)
	for i := 1; i <= steps; i++ {
		time.Sleep(time.Until(start.Add(time.Duration(i) * 16 * time.Millisecond)))
		e := 1 - math.Pow(1-float64(i)/float64(steps), 3)
		move := touchEvent("touchMove", subPixel(x0+(x1-x0)*e), subPixel(y0+(y1-y0)*e), tapRadii[2])
		wg.Go(func() {
			if err := send("Input.dispatchTouchEvent", move); err != nil {
				errs <- err
			}
		})
	}
	wg.Wait()
	close(errs)
	if err := <-errs; err != nil {
		return err
	}
	time.Sleep(between(110, 160))
	return send("Input.dispatchTouchEvent", touchEvent("touchEnd", x1, y1, 0))
}

// touchSlop 浏览器在手指移动超过这段距离（CSS 像素）之后才开始滚动，这段不计入滚动；
// 短于它的拖动会被当成一次点按（实测 Chrome 移动模拟下每次拖动少滚约 14.3px）
const touchSlop = 15.0

// mobileScroll 用手指拖动滚动 (deltaX, deltaY)：手指方向与滚动方向相反；拆成等长的几次拖动，
// 每次不超过视口的 60%，中间停一下。每次多拖一个 touchSlop：滚动距离接近请求值，最短的拖动也不会变成点按
func mobileScroll(send cdpSend, viewportW, viewportH int, deltaX, deltaY float64) error {
	if math.Abs(deltaX) < 1 && math.Abs(deltaY) < 1 {
		return nil
	}
	n := int(math.Ceil(math.Max(math.Abs(deltaX)/(float64(viewportW)*0.6), math.Abs(deltaY)/(float64(viewportH)*0.6))))
	dx, dy := deltaX/float64(n), deltaY/float64(n)
	dist := math.Hypot(dx, dy)
	ex, ey := dx+dx/dist*touchSlop, dy+dy/dist*touchSlop // 沿拖动方向补上阈值
	for i := 0; i < n; i++ {
		if i > 0 {
			time.Sleep(between(120, 250))
		}
		cx := float64(viewportW)/2 + (rand.Float64()-0.5)*float64(viewportW)*0.2
		cy := float64(viewportH)/2 + (rand.Float64()-0.5)*float64(viewportH)*0.1
		if err := mobileSwipe(send, cx+ex/2, cy+ey/2, cx-ex/2, cy-ey/2); err != nil {
			return err
		}
	}
	time.Sleep(300 * time.Millisecond)
	return nil
}

// imeKey 输入法按键事件：key 为 Unidentified、code 为空、keyCode 为 229（安卓真机实测）；ts 为事件时间（秒）。
// 不带 nativeVirtualKeyCode：带上 229 后 Chrome 之后的每个 Input.dispatchTouchEvent 都收不到确认（打完字就点不了）
func imeKey(send cdpSend, typ string, ts float64) error {
	return send("Input.dispatchKeyEvent", map[string]any{"type": typ, "key": "Unidentified", "code": "", "windowsVirtualKeyCode": 229, "timestamp": ts})
}

// imeStamp 一次输入法改动的时间戳（秒）：真机上这次改动的 keydown 与 keyup 共用它（Chromium 的 ImeAdapter 行为）
func imeStamp() float64 {
	return float64(time.Now().UnixNano()) / 1e9
}

// enterKey 输入法上的回车：key Enter、code 为空、keyCode 13（真机实测），浏览器据此产生 keypress 与 insertLineBreak
func enterKey(send cdpSend) error {
	ts := imeStamp() // keydown、keypress、keyup 共用一个时间戳（真机实测）
	down := map[string]any{"type": "keyDown", "key": "Enter", "code": "", "windowsVirtualKeyCode": 13, "text": "\r", "unmodifiedText": "\r", "timestamp": ts}
	if err := send("Input.dispatchKeyEvent", down); err != nil {
		return err
	}
	time.Sleep(between(1, 4))
	return send("Input.dispatchKeyEvent", map[string]any{"type": "keyUp", "key": "Enter", "code": "", "windowsVirtualKeyCode": 13, "timestamp": ts})
}

// mobileType 按手机输入法的组字方式输入：每个字是 keydown（229）→ 组字更新为当前整个词 → keyup（229）；
// 空格与换行（\n、\r）先提交当前词，退格在组字中缩短组字；其他控制字符报错（手机键盘没有 Tab 等键）；文本结束时提交（点按已有焦点的输入框不会结束组字，不提交的话下一次输入会把这个词替换掉）
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
		case '\n', '\r':
			if err := commit(); err != nil {
				return err
			}
			if err := enterKey(send); err != nil {
				return err
			}
		case '\b':
			// 退格：组字中缩短组字（减到空即取消组字），没有组字时发退格键
			if len(word) == 0 {
				if err := backspaceKey(send); err != nil {
					return err
				}
				continue
			}
			word = word[:len(word)-1]
			if err := composeStep(send, word); err != nil {
				return err
			}
		case ' ':
			if err := commit(); err != nil {
				return err
			}
			ts := imeStamp()
			if err := imeKey(send, "rawKeyDown", ts); err != nil {
				return err
			}
			if err := send("Input.insertText", map[string]any{"text": " "}); err != nil {
				return err
			}
			time.Sleep(between(5, 10))
			if err := imeKey(send, "keyUp", ts); err != nil {
				return err
			}
		default:
			if unicode.IsControl(r) {
				return fmt.Errorf("mobile typing does not support control character %U", r)
			}
			word = append(word, r)
			if err := composeStep(send, word); err != nil {
				return err
			}
		}
	}
	return commit()
}

// composeStep 一次组字改动：keydown(229) → 组字更新为 word（空串即取消组字）→ keyup(229)，按下与抬起共用时间戳
func composeStep(send cdpSend, word []rune) error {
	ts := imeStamp()
	if err := imeKey(send, "rawKeyDown", ts); err != nil {
		return err
	}
	n := len(utf16.Encode(word))
	if err := send("Input.imeSetComposition", map[string]any{"text": string(word), "selectionStart": n, "selectionEnd": n}); err != nil {
		return err
	}
	time.Sleep(between(5, 10))
	return imeKey(send, "keyUp", ts)
}

// backspaceKey 没有组字时的退格键：key Backspace、code 为空、keyCode 8，按下与抬起共用时间戳
func backspaceKey(send cdpSend) error {
	ts := imeStamp()
	key := func(typ string) map[string]any {
		return map[string]any{"type": typ, "key": "Backspace", "code": "", "windowsVirtualKeyCode": 8, "timestamp": ts}
	}
	if err := send("Input.dispatchKeyEvent", key("rawKeyDown")); err != nil {
		return err
	}
	time.Sleep(between(1, 4))
	return send("Input.dispatchKeyEvent", key("keyUp"))
}
