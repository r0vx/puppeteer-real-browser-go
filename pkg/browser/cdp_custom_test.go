package browser

import (
	"bytes"
	"testing"
)

// TestCustomCDPPage 用真实 headless Chrome 覆盖 CustomCDP（默认通道）主链路：
// 挂载到真实 page 目标、stealth 脚本在新文档生效且不破坏页面 API、截图返回 PNG 原始字节。
func TestCustomCDPPage(t *testing.T) {
	inst, err := Connect(t.Context(), &ConnectOptions{Headless: true})
	if err != nil {
		t.Fatalf("Connect: %v", err)
	}
	defer inst.Close()
	p := inst.Page()

	if err := p.Navigate("data:text/html,<title>custom-cdp</title><p id=ready>ok</p>"); err != nil {
		t.Fatalf("Navigate: %v", err)
	}
	// CustomCDP 的 Navigate 不等加载完成，用选择器确认新文档已就绪
	if err := p.WaitForSelector("#ready"); err != nil {
		t.Fatalf("WaitForSelector: %v", err)
	}

	cases := []struct {
		name string
		js   string
		want any
	}{
		// /json 首项可能是 browser_ui（omnibox 弹层，视口 1x1），必须挂到 type=page 的标签页
		{"attached to page target", `innerWidth > 100 && innerHeight > 100`, true},
		// 身份在新文档生效：无界面默认的 800×600 屏幕已被替换，可用区域扣除了任务栏 / 菜单栏
		{"identity applied on new document", `screen.width === 1920 && screen.height === 1080 && screen.availHeight < screen.height`, true},
		// 脚本自己构造的（未受信）鼠标事件保持原生：screenX 就是构造参数里的值
		{"untrusted MouseEvent native", `new MouseEvent('click', {clientX: 100}).screenX`, float64(0)},
		// headless UA 伪装：读取不能递归爆栈，也不能残留 HeadlessChrome
		{"userAgent readable", `navigator.userAgent.includes('Chrome/') && !navigator.userAgent.includes('HeadlessChrome')`, true},
		// 设置振荡器频率不能抛错，读回值必须等于设置值
		{"oscillator frequency settable", `(() => { const o = new AudioContext().createOscillator(); o.frequency.value = 500; return o.frequency.value; })()`, float64(500)},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := p.Evaluate(tc.js)
			if err != nil {
				t.Fatalf("Evaluate(%s): %v", tc.js, err)
			}
			if got != tc.want {
				t.Errorf("Evaluate(%s) = %v (%T), want %v", tc.js, got, got, tc.want)
			}
		})
	}

	t.Run("title", func(t *testing.T) {
		title, err := p.GetTitle()
		if err != nil {
			t.Fatalf("GetTitle: %v", err)
		}
		if title != "custom-cdp" {
			t.Errorf("GetTitle() = %q, want %q", title, "custom-cdp")
		}
	})

	t.Run("set viewport", func(t *testing.T) {
		if err := p.SetViewport(1280, 800); err != nil {
			t.Fatalf("SetViewport: %v", err)
		}
		got, err := p.Evaluate(`innerWidth + 'x' + innerHeight`)
		if err != nil {
			t.Fatalf("Evaluate: %v", err)
		}
		if got != "1280x800" {
			t.Errorf("viewport = %v, want 1280x800", got)
		}
	})

	t.Run("screenshot is PNG", func(t *testing.T) {
		shot, err := p.Screenshot()
		if err != nil {
			t.Fatalf("Screenshot: %v", err)
		}
		if !bytes.HasPrefix(shot, []byte("\x89PNG\r\n\x1a\n")) {
			t.Errorf("Screenshot() head = %q, want PNG signature", shot[:min(len(shot), 8)])
		}
	})
}
