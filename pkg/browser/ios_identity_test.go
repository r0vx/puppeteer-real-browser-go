package browser

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// loadIOSBaseline 读取 iPhone 真机基准的 auto 部分（ios-safari-iphone-2.json 为 iOS 26，ios-safari-iphone.json 为 iOS 18.5）
func loadIOSBaseline(t *testing.T, file string) map[string]any {
	t.Helper()
	raw, err := os.ReadFile("../../test/baseline/" + file)
	if err != nil {
		t.Fatal(err)
	}
	var r struct {
		Auto map[string]any `json:"auto"`
	}
	if err := json.Unmarshal(raw, &r); err != nil {
		t.Fatal(err)
	}
	return r.Auto
}

// TestIOSDevicesMatchBaselines 两款机型的 UA、屏幕、WebGPU、MAX_SAMPLES 与 WebGL 扩展都取自对应的真机基准
func TestIOSDevicesMatchBaselines(t *testing.T) {
	for _, tc := range []struct{ model, file string }{{"iphone-ios26", "ios-safari-iphone-2.json"}, {"iphone-ios18.5", "ios-safari-iphone.json"}} {
		dev, ok := lookupIOSDevice(tc.model)
		if !ok {
			t.Fatalf("%s not in the pool", tc.model)
		}
		b := loadIOSBaseline(t, tc.file)
		id, sc := b["identity"].(map[string]any), b["screen"].(map[string]any)["screen"].(map[string]any)
		if got := iosUserAgent(dev); got != id["userAgent"] {
			t.Errorf("%s ua %q, want %q", tc.model, got, id["userAgent"])
		}
		if float64(dev.Screen.Width) != sc["width"] || float64(dev.Screen.Height) != sc["height"] || dev.Screen.DPR != b["screen"].(map[string]any)["devicePixelRatio"] {
			t.Errorf("%s screen %+v, baseline %v", tc.model, dev.Screen, sc)
		}
		if gpu := b["features"].(map[string]any)["present"].(map[string]any)["navigator.gpu"]; dev.WebGPU != gpu {
			t.Errorf("%s WebGPU %v, baseline %v", tc.model, dev.WebGPU, gpu)
		}
		w1, w2 := b["webgl"].(map[string]any), b["webgl2"].(map[string]any)
		m := dev.GPU.Mobile
		if float64(m.Params2[36183].(int)) != w2["params"].(map[string]any)["MAX_SAMPLES"] {
			t.Errorf("%s MAX_SAMPLES %v, baseline %v", tc.model, m.Params2[36183], w2["params"].(map[string]any)["MAX_SAMPLES"])
		}
		for _, c := range []struct {
			got  []string
			want any
		}{{m.Extensions, w1["extensions"]}, {m.Extensions2, w2["extensions"]}} {
			want, _ := json.Marshal(c.want)
			got, _ := json.Marshal(c.got)
			if string(got) != string(want) {
				t.Errorf("%s extensions\n %s\nwant\n %s", tc.model, got, want)
			}
		}
		if m.Params[7938] != w1["version"] || m.Params2[7938] != w2["version"] || m.Params[35724] != w1["shadingLanguageVersion"] || m.Params2[35724] != w2["shadingLanguageVersion"] {
			t.Errorf("%s VERSION / SHADING_LANGUAGE_VERSION not from the baseline", tc.model)
		}
	}
}

// TestIOSNormalize iOS 指纹按机型整体补齐：Safari 版本取自机型（不跟随 UA 里的 Chrome 版本），同一用户 ID 结果固定，再次规范化没有改动
func TestIOSNormalize(t *testing.T) {
	fresh := func() *FingerprintConfig {
		c := &FingerprintConfig{UserID: "acct-ios", Device: &DeviceConfig{Kind: deviceKindIOS, DeviceMemory: 8}}
		c.Browser.UserAgent = iphoneTestUA
		c.Browser.Language = "zh-CN"
		c.Screen = ScreenConfig{Width: 1920, Height: 1080, DevicePixelRatio: 1}
		return c
	}
	c := fresh()
	if !c.Normalize() {
		t.Fatal("Normalize reported no change")
	}
	dev, ok := lookupIOSDevice(c.Device.Model)
	if !ok {
		t.Fatalf("model %q not in the pool", c.Device.Model)
	}
	b := c.Browser
	if b.UserAgent != iosUserAgent(dev) || b.Platform != iosPlatform || b.Vendor != iosVendor || b.PlatformVersion != "" ||
		b.HardwareConcurrency != iosCores || b.MaxTouchPoints != iosTouchPoints || c.Device.DeviceMemory != 0 {
		t.Errorf("browser fields not taken from the device: %+v / %+v", b, c.Device)
	}
	s := c.Screen
	if s.Width != dev.Screen.Width || s.Height != dev.Screen.Height || s.DevicePixelRatio != dev.Screen.DPR || s.AvailWidth != s.Width || s.AvailHeight != s.Height {
		t.Errorf("screen %+v does not match device %+v", s, dev.Screen)
	}
	if c.WebGL.Vendor != "Apple Inc." || c.WebGL.Renderer != "Apple GPU" || c.WebGL.Version != "WebGL 1.0" || c.WebGL.MaxTextureSize != 16384 {
		t.Errorf("webgl %+v", c.WebGL)
	}
	if c.Normalize() {
		t.Error("second Normalize changed the config")
	}
	if again := fresh(); again.Normalize() && again.Device.Model != c.Device.Model {
		t.Errorf("same user ID picked %q then %q", c.Device.Model, again.Device.Model)
	}
	keep := &FingerprintConfig{UserID: "acct-keep", Device: &DeviceConfig{Kind: deviceKindIOS, Model: "iphone-ios18.5"}}
	keep.Normalize()
	if keep.Device.Model != "iphone-ios18.5" || keep.Screen.Width != 440 || !strings.Contains(keep.Browser.UserAgent, "Version/18.5 ") {
		t.Errorf("model / screen / ua = %q %d %q", keep.Device.Model, keep.Screen.Width, keep.Browser.UserAgent)
	}
}

// TestIOSIdentityFromConfig iOS 身份：不发 Client Hints，languages 只有主语言，屏幕即视口，GPU 为 Apple GPU
func TestIOSIdentityFromConfig(t *testing.T) {
	cfg := &FingerprintConfig{UserID: "acct-ios-id", Device: &DeviceConfig{Kind: deviceKindIOS, Model: "iphone-ios26"}}
	cfg.Normalize()
	id, err := identityFromConfig(cfg, testHost())
	if err != nil {
		t.Fatal(err)
	}
	if id.OS != OSIOS || !strings.Contains(id.UserAgent, "Version/26.2 ") || id.Platform != "iPhone" || id.HardwareConcurrency != 4 ||
		id.DeviceMemory != 0 || !id.NoClientHints || id.NoWebGPU || id.Battery != 0 {
		t.Errorf("identity %+v", id)
	}
	if strings.Join(id.Languages, ",") != "zh-CN" || id.AcceptLanguage != "zh-CN" {
		t.Errorf("languages %v / accept-language %q, want only zh-CN", id.Languages, id.AcceptLanguage)
	}
	if len(id.Metadata.Brands) != 0 || id.Metadata.Platform != "" {
		t.Errorf("metadata %+v, want none", id.Metadata)
	}
	s := id.Screen
	if s.Width != 420 || s.Height != 912 || s.DPR != 3 || s.InnerWidth != 420 || s.InnerHeight != 912 || s.OuterHeight != 912 || !s.Mobile || s.ViewportTop != 0 {
		t.Errorf("screen %+v", s)
	}
	if id.GPU == nil || id.GPU.Mobile == nil || id.GPU.Renderer != "Apple GPU" {
		t.Errorf("gpu %+v", id.GPU)
	}
	old := &FingerprintConfig{UserID: "acct-ios-old", Device: &DeviceConfig{Kind: deviceKindIOS, Model: "iphone-ios18.5"}}
	old.Normalize()
	if id, _ := identityFromConfig(old, testHost()); id == nil || !id.NoWebGPU {
		t.Errorf("iOS 18.5 identity %+v, want NoWebGPU", id)
	}
}

// TestIOSHeadlessScreenFlags 无界面启动参数用 iPhone 的物理分辨率与 DPR，整屏都是可用区域
func TestIOSHeadlessScreenFlags(t *testing.T) {
	cfg := &FingerprintConfig{UserID: "acct-ios-flags", Device: &DeviceConfig{Kind: deviceKindIOS, Model: "iphone-ios26"}}
	cfg.Normalize()
	got := strings.Join(headlessScreenFlags(configGeometry(cfg)), " ")
	if want := "--screen-info={0,0 1260x2736 workAreaTop=0 workAreaBottom=0} --force-device-scale-factor=3"; got != want {
		t.Errorf("flags %q, want %q", got, want)
	}
}

// TestIOSFontConfig 只包含存在且非空的 Apple 字体目录，字体保持原名（iPhone 上能按名探测到），通用族按 Safari；没有字体时返回空
func TestIOSFontConfig(t *testing.T) {
	root, empty := t.TempDir(), t.TempDir()
	os.WriteFile(filepath.Join(root, "PingFang.ttc"), nil, 0o644)
	dir := t.TempDir()
	conf, err := writeIOSFontConfig(dir, []string{root, empty, filepath.Join(root, "missing")})
	if err != nil || conf == "" {
		t.Fatalf("conf %q, err %v", conf, err)
	}
	data, _ := os.ReadFile(conf)
	s := string(data)
	for _, want := range []string{"<dir>" + root + "</dir>", "<cachedir>" + filepath.Join(dir, "cache") + "</cachedir>",
		`<alias binding="same"><family>sans-serif</family><prefer><family>Helvetica</family><family>PingFang SC</family><family>Apple Color Emoji</family></prefer></alias>`} {
		if !strings.Contains(s, want) {
			t.Errorf("config lacks %s:\n%s", want, s)
		}
	}
	if strings.Contains(s, "<dir>"+empty+"</dir>") || strings.Contains(s, "missing") {
		t.Errorf("config lists an empty or missing directory:\n%s", s)
	}
	if conf, err := writeIOSFontConfig(t.TempDir(), []string{empty}); conf != "" || err != nil {
		t.Errorf("no Apple fonts: conf %q, err %v; want none", conf, err)
	}
}
