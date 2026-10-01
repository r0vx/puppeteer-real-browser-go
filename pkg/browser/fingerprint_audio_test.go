package browser

import (
	"fmt"
	"testing"
	"time"
)

// TestFingerprintAudioKeepsWebAudioWorking 指纹模式下的音频脚本只能加噪声，不能破坏 Web Audio 本身：
// 振荡器波形按页面设置、构造参数生效、instanceof 成立、OfflineAudioContext 能按请求的采样率渲染
func TestFingerprintAudioKeepsWebAudioWorking(t *testing.T) {
	cases := []struct {
		name string
		js   string
		want any
	}{
		{"oscillator type kept", `(() => {
			const ctx = new AudioContext(), bad = [];
			for (const type of ['sine', 'square', 'sawtooth', 'triangle']) {
				const o = ctx.createOscillator();
				o.type = type;
				o.start();
				if (o.type !== type) bad.push(type + '->' + o.type);
				o.stop();
			}
			return bad.join(',');
		})()`, ""},
		{"AudioContext options honoured", `new AudioContext({sampleRate: 22050}).sampleRate`, float64(22050)},
		{"instanceof AudioContext", `new AudioContext() instanceof AudioContext`, true},
		{"instanceof OfflineAudioContext", `new OfflineAudioContext(1, 128, 44100) instanceof OfflineAudioContext`, true},
		{"OfflineAudioContext renders", `(() => {
			window.__render = 'pending';
			try {
				// FingerprintJS 式音频指纹：振荡器 → 压缩器 → 离线渲染
				const c = new OfflineAudioContext(1, 5000, 44100);
				const o = c.createOscillator();
				o.type = 'triangle';
				o.frequency.value = 10000;
				const comp = c.createDynamicsCompressor();
				o.connect(comp);
				comp.connect(c.destination);
				o.start(0);
				c.startRendering().then(
					b => { window.__render = c.sampleRate + '/' + b.length + '/' + b.sampleRate; },
					e => { window.__render = 'reject: ' + e; });
			} catch (e) {
				window.__render = 'throw: ' + e;
			}
			return true;
		})()`, true},
	}

	for _, custom := range []bool{false, true} {
		t.Run(fmt.Sprintf("UseCustomCDP=%v", custom), func(t *testing.T) {
			inst, err := Connect(t.Context(), &ConnectOptions{
				Headless:          true,
				UseCustomCDP:      custom,
				FingerprintUserID: "audio-test-user",
				FingerprintDir:    t.TempDir(),
			})
			if err != nil {
				t.Fatalf("Connect: %v", err)
			}
			defer inst.Close()
			p := inst.Page()
			if err := p.Navigate("data:text/html,<title>audio</title>"); err != nil {
				t.Fatalf("Navigate: %v", err)
			}

			for _, tc := range cases {
				t.Run(tc.name, func(t *testing.T) {
					got, err := p.Evaluate(tc.js)
					if err != nil {
						t.Fatalf("Evaluate: %v", err)
					}
					if got != tc.want {
						t.Errorf("got %v (%T), want %v", got, got, tc.want)
					}
				})
			}

			t.Run("OfflineAudioContext render result", func(t *testing.T) {
				var got any
				for deadline := time.Now().Add(5 * time.Second); time.Now().Before(deadline); time.Sleep(100 * time.Millisecond) {
					if got, err = p.Evaluate(`window.__render`); err != nil || got != "pending" {
						break
					}
				}
				if got != "44100/5000/44100" {
					t.Errorf("render = %v (err %v), want 44100/5000/44100", got, err)
				}
			})
		})
	}
}
