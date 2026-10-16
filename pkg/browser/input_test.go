package browser

import (
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"slices"
	"testing"
	"time"
)

// inputTestPage 记录页面收到的每个输入事件的属性；按钮中心落在半像素上，用来检查坐标是否对齐设备像素
const inputTestPage = `<!doctype html><html><head><title>input</title></head><body style="margin:0">
<button id=b style="position:absolute;left:300px;top:200px;width:121px;height:41px">btn</button>
<input id=i style="position:absolute;left:300px;top:300px;width:200px">
<script>
window.__log = [];
const keep = e => window.__log.push({type: e.type, trusted: e.isTrusted, t: e.timeStamp,
  pressure: e.pressure ?? null, x: e.clientX ?? null, y: e.clientY ?? null,
  key: e.key ?? null, code: e.code ?? null, keyCode: e.keyCode ?? null, inputType: e.inputType ?? null});
for (const t of ['pointermove','pointerdown','pointerup','mousedown','mouseup','click','keydown','keypress','keyup','input'])
  addEventListener(t, keep, true);
</script></body></html>`

// inputEvent 页面记录的一个输入事件
type inputEvent struct {
	Type      string   `json:"type"`
	Trusted   bool     `json:"trusted"`
	T         float64  `json:"t"`
	Pressure  *float64 `json:"pressure"`
	X         *float64 `json:"x"`
	Y         *float64 `json:"y"`
	Key       string   `json:"key"`
	Code      string   `json:"code"`
	KeyCode   int      `json:"keyCode"`
	InputType string   `json:"inputType"`
}

// recordInput 清空记录、执行动作，返回动作期间页面收到的输入事件
func recordInput(t *testing.T, p Page, act func() error) []inputEvent {
	t.Helper()
	if _, err := p.Evaluate(`window.__log = [], true`); err != nil {
		t.Fatal(err)
	}
	if err := act(); err != nil {
		t.Fatal(err)
	}
	time.Sleep(200 * time.Millisecond)
	raw, err := p.Evaluate(`JSON.stringify(window.__log)`)
	if err != nil {
		t.Fatal(err)
	}
	var evs []inputEvent
	if err := json.Unmarshal([]byte(fmt.Sprint(raw)), &evs); err != nil {
		t.Fatal(err)
	}
	return evs
}

// firstEvent 返回第一个指定类型的事件
func firstEvent(evs []inputEvent, typ string) *inputEvent {
	for i := range evs {
		if evs[i].Type == typ {
			return &evs[i]
		}
	}
	return nil
}

// dprFingerprintDir 写一个 Windows 1536x864、DPR 1.25 的账号指纹，返回指纹目录
func dprFingerprintDir(t *testing.T, userID string) string {
	t.Helper()
	dir := t.TempDir()
	m, err := NewUserFingerprintManager(dir)
	if err != nil {
		t.Fatal(err)
	}
	cfg, err := m.GetOrCreateUserFingerprint(userID, &FingerprintInitParams{
		UserAgent: "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36",
	})
	if err != nil {
		t.Fatal(err)
	}
	cfg.Screen = ScreenConfig{Width: 1536, Height: 864, DevicePixelRatio: 1.25}
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, userID+".json"), data, 0o644); err != nil {
		t.Fatal(err)
	}
	return dir
}

// TestInputLooksHuman CDP 输入不留下可识别的痕迹（两条通道）：按下时有压力、按住有时长、指针坐标落在设备像素上；
// 按键带 key / code / keyCode、有按住时长，大写与符号先按住 Shift；SendKeys 走真实按键而不是 JS 改值
func TestInputLooksHuman(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { fmt.Fprint(w, inputTestPage) }))
	defer srv.Close()
	const dpr = 1.25

	for _, chromedpPath := range []bool{true, false} {
		t.Run(fmt.Sprintf("UseChromedp=%v", chromedpPath), func(t *testing.T) {
			inst, err := Connect(t.Context(), &ConnectOptions{
				Headless: true, UseChromedp: chromedpPath,
				FingerprintUserID: "input", FingerprintDir: dprFingerprintDir(t, "input"),
			})
			if err != nil {
				t.Fatal(err)
			}
			defer inst.Close()
			p := inst.Page()
			sp := p.(PageWithSelector)
			if err := p.Navigate(srv.URL); err != nil {
				t.Fatal(err)
			}
			if got, _ := p.Evaluate(`devicePixelRatio`); got != dpr {
				t.Fatalf("devicePixelRatio = %v, want %v", got, dpr)
			}

			mouse := func(name string, act func() error) {
				evs := recordInput(t, p, act)
				down, up := firstEvent(evs, "pointerdown"), firstEvent(evs, "pointerup")
				md, mu := firstEvent(evs, "mousedown"), firstEvent(evs, "mouseup")
				if down == nil || up == nil || md == nil || mu == nil || firstEvent(evs, "click") == nil {
					t.Errorf("%s: incomplete click %+v", name, evs)
					return
				}
				// 不支持压感的鼠标：按住时 pressure 为 0.5，松开为 0
				if *down.Pressure != 0.5 || *up.Pressure != 0 {
					t.Errorf("%s: pressure down=%v up=%v, want 0.5 / 0", name, *down.Pressure, *up.Pressure)
				}
				if hold := mu.T - md.T; hold < 50 {
					t.Errorf("%s: button held %.0fms, want >= 50ms", name, hold)
				}
				for _, e := range evs {
					if e.Type != "pointermove" && e.Type != "pointerdown" && e.Type != "pointerup" {
						continue
					}
					for _, v := range []float64{*e.X, *e.Y} {
						if d := v * dpr; math.Abs(d-math.Round(d)) > 1e-3 {
							t.Errorf("%s: %s at %v,%v is off the device pixel grid", name, e.Type, *e.X, *e.Y)
							break
						}
					}
				}
			}
			mouse("Click", func() error { return p.Click(360.3, 220.7) })
			mouse("RealClick", func() error { return p.RealClick(340.3, 215.7) })
			mouse("ClickSelector", func() error { return sp.ClickSelector("#b") })

			keys := func(name string, wantDown []string, wantValue string, act func() error) {
				evs := recordInput(t, p, act)
				var down []string
				pressed := map[string]float64{}
				for _, e := range evs {
					if !e.Trusted {
						t.Errorf("%s: untrusted %s event", name, e.Type)
					}
					switch e.Type {
					case "keydown":
						if e.Key == "" || e.Code == "" || e.KeyCode == 0 {
							t.Errorf("%s: keydown without key/code/keyCode: %+v", name, e)
						}
						if e.Key == "Shift" && (e.Code != "ShiftLeft" || e.KeyCode != 16) {
							t.Errorf("%s: Shift keydown %+v, want ShiftLeft / 16", name, e)
						}
						down = append(down, e.Key)
						pressed[e.Code] = e.T
					case "keyup":
						if hold := e.T - pressed[e.Code]; hold < 20 {
							t.Errorf("%s: %q held %.0fms, want >= 20ms", name, e.Key, hold)
						}
					}
				}
				if !slices.Equal(down, wantDown) {
					t.Errorf("%s: keydown keys %q, want %q", name, down, wantDown)
				}
				if got, _ := p.Evaluate(`document.getElementById('i').value`); got != wantValue {
					t.Errorf("%s: input value %q, want %q", name, got, wantValue)
				}
			}
			if err := sp.RealClickSelector("#i"); err != nil {
				t.Fatal(err)
			}
			keys("RealSendKeys", []string{"a", "Shift", "B", "@"}, "aB@", func() error { return sp.RealSendKeys("aB@") })
			keys("SendKeys", []string{"x", "Shift", "Y"}, "aB@xY", func() error { return sp.SendKeys("#i", "xY") })
		})
	}
}
