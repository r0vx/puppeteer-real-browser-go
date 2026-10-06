package browser

import (
	"fmt"
	"math"
	"testing"
	"time"
)

// turnstilePage 模拟 Turnstile 控件：300x65、无内边距的容器，复选框在左侧；记录收到的点击
const turnstilePage = `data:text/html,<title>ts</title>
<div id="w" style="position:absolute;left:100px;top:100px;width:300px;height:65px;margin:0;padding:0">
<input type="hidden" name="cf-turnstile-response" value=""></div>
<script>window.__clicks=[];document.addEventListener('click',e=>window.__clicks.push([e.clientX,e.clientY,e.isTrusted]))</script>`

// clicks 返回页面记录的点击 [x, y, isTrusted]
func clicks(t *testing.T, p Page) [][]any {
	t.Helper()
	v, err := p.Evaluate(`window.__clicks`)
	if err != nil {
		t.Fatalf("read clicks: %v", err)
	}
	list, _ := v.([]any)
	out := make([][]any, 0, len(list))
	for _, c := range list {
		if c, ok := c.([]any); ok {
			out = append(out, c)
		}
	}
	return out
}

// TestTurnstileOption ConnectOptions.Turnstile 开启时自动点击未通过的 Turnstile 复选框
// （容器左侧 30px、垂直居中，与原版一致），通过后停止点击；关闭时不点击
func TestTurnstileOption(t *testing.T) {
	for _, chromedpPath := range []bool{true, false} {
		t.Run(fmt.Sprintf("UseChromedp=%v", chromedpPath), func(t *testing.T) {
			inst, err := Connect(t.Context(), &ConnectOptions{Headless: true, UseChromedp: chromedpPath, Turnstile: true})
			if err != nil {
				t.Fatalf("Connect: %v", err)
			}
			defer inst.Close()
			p := inst.Page()
			if err := p.Navigate(turnstilePage); err != nil {
				t.Fatalf("Navigate: %v", err)
			}

			var got [][]any
			for deadline := time.Now().Add(5 * time.Second); len(got) == 0 && time.Now().Before(deadline); time.Sleep(100 * time.Millisecond) {
				got = clicks(t, p)
			}
			if len(got) == 0 {
				t.Fatal("no click on the Turnstile widget within 5s")
			}
			x, _ := got[0][0].(float64)
			y, _ := got[0][1].(float64)
			// 容器 left=100 → 复选框 x≈130；top=100、高 65 → y≈132.5
			if math.Abs(x-130) > 2 || math.Abs(y-132.5) > 2 {
				t.Errorf("clicked at (%v, %v), want ≈(130, 132.5) on the checkbox", x, y)
			}
			if got[0][2] != true {
				t.Error("click event is not trusted (not dispatched through CDP input)")
			}

			// 拿到 token 后不应再点击
			if _, err := p.Evaluate(`document.querySelector('[name="cf-turnstile-response"]').value = 'token'`); err != nil {
				t.Fatal(err)
			}
			time.Sleep(1200 * time.Millisecond) // 等进行中的一轮结束
			before := len(clicks(t, p))
			time.Sleep(2500 * time.Millisecond)
			if after := len(clicks(t, p)); after != before {
				t.Errorf("kept clicking after the widget was solved: %d -> %d clicks", before, after)
			}
		})
	}

	t.Run("disabled", func(t *testing.T) {
		inst, err := Connect(t.Context(), &ConnectOptions{Headless: true})
		if err != nil {
			t.Fatalf("Connect: %v", err)
		}
		defer inst.Close()
		p := inst.Page()
		if err := p.Navigate(turnstilePage); err != nil {
			t.Fatalf("Navigate: %v", err)
		}
		time.Sleep(2500 * time.Millisecond)
		if n := len(clicks(t, p)); n != 0 {
			t.Errorf("Turnstile disabled but got %d clicks", n)
		}
	})
}
