package browser

import (
	"bytes"
	"encoding/json"
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const (
	androidTestUA = "Mozilla/5.0 (Linux; Android 14; 23127PN0CC Build/UKQ1.230804.001) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.6778.39 Mobile Safari/537.36"
	iphoneTestUA  = "Mozilla/5.0 (iPhone; CPU iPhone OS 18_5 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/18.5 Mobile/15E148 Safari/604.1"
)

// TestAndroidUserAgent 安卓 UA 为 Chrome 精简格式：系统段冻结为 Android 10; K，只保留主版本
func TestAndroidUserAgent(t *testing.T) {
	want := "Mozilla/5.0 (Linux; Android 10; K) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/155.0.0.0 Mobile Safari/537.36"
	if got := androidUserAgent("155"); got != want {
		t.Errorf("androidUserAgent = %q, want %q", got, want)
	}
}

// TestAndroidDevicesCoherent 机型池每一项自洽：代号唯一，CSS 屏幕 × DPR 等于 1080 / 1440 宽的物理屏，GPU 带完整的手机 WebGL 表面
func TestAndroidDevicesCoherent(t *testing.T) {
	seen := map[string]bool{}
	for _, d := range androidDevices {
		v := d.Value
		if seen[v.Model] || d.Weight <= 0 || v.Source == "" {
			t.Errorf("%s: duplicate, zero weight or no source", v.Model)
		}
		seen[v.Model] = true
		if phys := float64(v.Screen.Width) * v.Screen.DPR; math.Abs(phys-1080) > 2 && math.Abs(phys-1440) > 2 {
			t.Errorf("%s: %dx%d@%g is not a 1080 / 1440 wide panel", v.Model, v.Screen.Width, v.Screen.Height, v.Screen.DPR)
		}
		if v.GPU.Mobile == nil || len(v.GPU.Mobile.Extensions) == 0 || len(v.GPU.Mobile.Precision) != 12 {
			t.Errorf("%s: GPU has no complete mobile WebGL surface", v.Model)
		}
	}
}

// TestAndroidNormalize 安卓指纹按机型整体补齐：同一用户 ID 结果固定，再次规范化没有改动
func TestAndroidNormalize(t *testing.T) {
	fresh := func() *FingerprintConfig {
		c := &FingerprintConfig{UserID: "acct-m", Device: &DeviceConfig{Kind: deviceKindAndroid}}
		c.Browser.UserAgent = androidTestUA
		c.Browser.Language = "zh-CN"
		c.Screen = ScreenConfig{Width: 1920, Height: 1080, DevicePixelRatio: 1}
		return c
	}
	c := fresh()
	if !c.Normalize() {
		t.Fatal("Normalize reported no change")
	}
	dev, ok := lookupAndroidDevice(c.Device.Model)
	if !ok {
		t.Fatalf("model %q not in the pool", c.Device.Model)
	}
	b := c.Browser
	if b.UserAgent != androidUserAgent("131") || b.Platform != androidPlatform || b.PlatformVersion != dev.PlatformVersion ||
		b.HardwareConcurrency != androidCores || b.MaxTouchPoints != androidTouchPoints || c.Device.DeviceMemory != androidDeviceMemory {
		t.Errorf("browser fields not taken from the device: %+v / %+v", b, c.Device)
	}
	s := c.Screen
	if s.Width != dev.Screen.Width || s.Height != dev.Screen.Height || s.DevicePixelRatio != dev.Screen.DPR || s.AvailWidth != s.Width || s.AvailHeight != s.Height {
		t.Errorf("screen %+v does not match device %+v", s, dev.Screen)
	}
	if c.WebGL.Vendor != "ARM" || c.WebGL.Renderer != "Mali-G610 MC6" || c.WebGL.MaxTextureSize != 4096 {
		t.Errorf("webgl %+v", c.WebGL)
	}
	if c.Normalize() {
		t.Error("second Normalize changed the config")
	}
	if again := fresh(); again.Normalize() && again.Device.Model != c.Device.Model {
		t.Errorf("same user ID picked %q then %q", c.Device.Model, again.Device.Model)
	}
}

// TestAndroidNormalizeKeepsModel 池中的机型保留不变
func TestAndroidNormalizeKeepsModel(t *testing.T) {
	c := &FingerprintConfig{UserID: "acct-keep", Device: &DeviceConfig{Kind: deviceKindAndroid, Model: "RMX3560"}}
	c.Normalize()
	if c.Device.Model != "RMX3560" || c.Screen.Width != 360 || c.Browser.PlatformVersion != "14.0.0" {
		t.Errorf("model / screen / version = %q %d %q", c.Device.Model, c.Screen.Width, c.Browser.PlatformVersion)
	}
}

// TestMobileUserAgentFileStaysDesktop 老指纹即使绑的是手机 UA，也仍规范化为 Windows，不会长出 device 段（Review Focus 1）
func TestMobileUserAgentFileStaysDesktop(t *testing.T) {
	for _, ua := range []string{androidTestUA, iphoneTestUA} {
		c := &FingerprintConfig{UserID: "legacy"}
		c.Browser.UserAgent = ua
		c.Normalize()
		if c.Device != nil || c.Browser.Platform != "Win32" || !strings.Contains(c.Browser.UserAgent, "Windows NT 10.0") {
			t.Errorf("legacy %q became %q / %q / device %+v", ua, c.Browser.UserAgent, c.Browser.Platform, c.Device)
		}
	}
}

// TestFingerprintForOptionsMobile Mobile 只对新账号生效；与已有文件类型不一致时报错且不改文件（Review Focus 2）；iOS 尚不支持
func TestFingerprintForOptionsMobile(t *testing.T) {
	dir := t.TempDir()
	cfg, err := fingerprintForOptions(&ConnectOptions{FingerprintUserID: "m1", FingerprintDir: dir, Mobile: true, UserAgent: androidTestUA, Width: 1920, Height: 1080})
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Device == nil || cfg.Device.Kind != deviceKindAndroid {
		t.Fatalf("new mobile account has device %+v", cfg.Device)
	}
	if _, ok := lookupAndroidDevice(cfg.Device.Model); !ok || cfg.Screen.Width == 1920 {
		t.Errorf("model %q / screen width %d: Width option leaked or model not in the pool", cfg.Device.Model, cfg.Screen.Width)
	}
	if _, err := fingerprintForOptions(&ConnectOptions{FingerprintUserID: "d1", FingerprintDir: dir}); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name string
		opts *ConnectOptions
		want string
	}{
		{"mobile file without Mobile", &ConnectOptions{FingerprintUserID: "m1", FingerprintDir: dir}, "mobile identity"},
		{"desktop file with Mobile", &ConnectOptions{FingerprintUserID: "d1", FingerprintDir: dir, Mobile: true}, "desktop identity"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			path := filepath.Join(dir, tc.opts.FingerprintUserID+".json")
			before, _ := os.ReadFile(path)
			if _, err := fingerprintForOptions(tc.opts); err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Errorf("err = %v, want it to mention %q", err, tc.want)
			}
			if after, _ := os.ReadFile(path); !bytes.Equal(before, after) {
				t.Error("fingerprint file changed")
			}
		})
	}
	if _, err := fingerprintForOptions(&ConnectOptions{FingerprintUserID: "i1", FingerprintDir: dir, Mobile: true, UserAgent: iphoneTestUA}); err == nil || !strings.Contains(err.Error(), "iOS") {
		t.Errorf("iOS mobile account: err = %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, "i1.json")); err == nil {
		t.Error("a file was written for the rejected iOS account")
	}
	var saved map[string]json.RawMessage
	raw, _ := os.ReadFile(filepath.Join(dir, "m1.json"))
	if json.Unmarshal(raw, &saved) != nil || saved["device"] == nil {
		t.Errorf("saved mobile fingerprint has no device section: %s", raw)
	}
}

// TestConnectRejectsMobileMisuse 移动身份必须有账号指纹、只支持默认通道：在启动 Chrome 之前就报错
func TestConnectRejectsMobileMisuse(t *testing.T) {
	for _, opts := range []*ConnectOptions{
		{Headless: true, Mobile: true},
		{Headless: true, Mobile: true, FingerprintUserID: "x", FingerprintDir: t.TempDir(), UseChromedp: true},
	} {
		if inst, err := Connect(t.Context(), opts); err == nil {
			inst.Close()
			t.Errorf("Connect(%+v) succeeded", opts)
		}
	}
}

// TestAndroidIdentityFromConfig 安卓身份：UA 与 Client Hints、屏幕与视口、内存与电量都由机型与真实浏览器派生
func TestAndroidIdentityFromConfig(t *testing.T) {
	cfg := &FingerprintConfig{UserID: "acct-id", Device: &DeviceConfig{Kind: deviceKindAndroid, Model: "PGKM10"}}
	cfg.Normalize()
	id, err := identityFromConfig(cfg, testHost())
	if err != nil {
		t.Fatal(err)
	}
	if id.OS != OSAndroid || id.UserAgent != androidUserAgent("154") || id.Platform != androidPlatform || id.DeviceMemory != 8 || id.HardwareConcurrency != 8 {
		t.Errorf("identity %+v", id)
	}
	m := id.Metadata
	if m.Platform != "Android" || m.PlatformVersion != "15.0.0" || m.Model != "PGKM10" || !m.Mobile || m.Architecture != "" || m.Bitness != "" ||
		len(m.FormFactors) != 1 || m.FormFactors[0] != "Mobile" || len(m.Brands) == 0 {
		t.Errorf("metadata %+v", m)
	}
	s := id.Screen
	if s.Width != 360 || s.Height != 804 || s.DPR != 3 || s.InnerWidth != 360 || s.InnerHeight != 700 || s.OuterHeight != 700 || !s.Mobile {
		t.Errorf("screen %+v", s)
	}
	if id.Battery < 0.35 || id.Battery > 0.95 || id.Battery != androidBattery("acct-id") {
		t.Errorf("battery %v", id.Battery)
	}
	if id.GPU == nil || id.GPU.Mobile == nil || id.GPU.Renderer != "Mali-G610 MC6" {
		t.Errorf("gpu %+v", id.GPU)
	}
}

// TestAndroidHeadlessScreenFlags 无界面启动参数用手机的物理分辨率与 DPR，整屏都是可用区域
func TestAndroidHeadlessScreenFlags(t *testing.T) {
	cfg := &FingerprintConfig{UserID: "acct-flags", Device: &DeviceConfig{Kind: deviceKindAndroid, Model: "PGKM10"}}
	cfg.Normalize()
	got := strings.Join(headlessScreenFlags(configGeometry(cfg)), " ")
	if want := "--screen-info={0,0 1080x2412 workAreaTop=0 workAreaBottom=0} --force-device-scale-factor=3"; got != want {
		t.Errorf("flags %q, want %q", got, want)
	}
}

// TestAndroidFontConfig 只链接各类安卓字体的文件，按类改族名，并配好通用族与安卓别名；缺的字体类会报出来
func TestAndroidFontConfig(t *testing.T) {
	root := t.TempDir()
	for _, name := range []string{"Roboto-Regular.ttf", "Roboto-Bold.ttf", "RobotoCondensed-Regular.ttf", "NotoSerif-Regular.ttf",
		"NotoSerifDisplay-Regular.ttf", "NotoSansCJK-Regular.ttc", "NotoColorEmoji.ttf", "DejaVuSans.ttf"} {
		os.WriteFile(filepath.Join(root, name), nil, 0o644)
	}
	dir := t.TempDir()
	conf, missing, err := writeAndroidFontConfig(dir, []string{root})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Join(missing, ",") != "android-mono" {
		t.Errorf("missing = %v, want [android-mono]", missing)
	}
	entries, _ := os.ReadDir(filepath.Join(dir, "fonts"))
	var linked []string
	for _, e := range entries {
		linked = append(linked, e.Name())
	}
	if want := "android-cjk--NotoSansCJK-Regular.ttc,android-emoji--NotoColorEmoji.ttf,android-sans--Roboto-Bold.ttf,android-sans--Roboto-Regular.ttf,android-serif--NotoSerif-Regular.ttf"; strings.Join(linked, ",") != want {
		t.Errorf("linked %v", linked)
	}
	data, _ := os.ReadFile(conf)
	for _, s := range []string{"<dir>" + filepath.Join(dir, "fonts") + "</dir>", `<string>/android-sans--</string>`,
		`<family>Courier New</family>`, `<family>sans-serif</family>`, `binding="same"`} {
		if !strings.Contains(string(data), s) {
			t.Errorf("config lacks %s", s)
		}
	}
}
