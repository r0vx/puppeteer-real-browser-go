package browser

import (
	"slices"
	"testing"
)

// testHost 无界面 Linux 服务器上 Google Chrome 154 报告的真实值（与 Docker 实测一致）
func testHost() hostInfo {
	return hostInfo{
		UserAgent:           "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) HeadlessChrome/154.0.0.0 Safari/537.36",
		Platform:            "Linux x86_64",
		Languages:           []string{"en-US", "en"},
		Timezone:            "Asia/Shanghai",
		HardwareConcurrency: 12,
		Screen:              [3]float64{800, 600, 1},
		Brands:              []uaBrand{{"Chromium", "154"}, {"Google Chrome", "154"}, {"Not A(Brand", "99"}},
		FullVersionList:     []uaBrand{{"Chromium", "154.0.8037.92"}, {"Google Chrome", "154.0.8037.92"}, {"Not A(Brand", "99.0.0.0"}},
		UAPlatform:          "Linux",
		PlatformVersion:     "6.8.0",
		Architecture:        "x86",
		Bitness:             "64",
		FormFactors:         []string{"Desktop"},
		WebGLVendor:         "Google Inc. (Google)",
		WebGLRenderer:       "ANGLE (Google, Vulkan 1.3.0 (SwiftShader Device (Subzero) (0x0000C0DE)), SwiftShader driver)",
	}
}

// TestOSFromUserAgent 系统只由 UA 判定，Linux 与空 UA 按 Windows
func TestOSFromUserAgent(t *testing.T) {
	cases := []struct {
		ua   string
		want OSFamily
	}{
		{"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36", OSWindows},
		{"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36", OSMac},
		{"Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36", OSWindows}, // 旧数据里的 Linux 按 Windows
		{"", OSWindows},
	}
	for _, tc := range cases {
		if got := osFromUserAgent(tc.ua); got != tc.want {
			t.Errorf("osFromUserAgent(%q) = %q, want %q", tc.ua, got, tc.want)
		}
	}
}

// TestCanonicalUserAgent UA 为 Chrome 精简格式，主版本可从任意 UA 中取出
func TestCanonicalUserAgent(t *testing.T) {
	if got, want := canonicalUserAgent(OSWindows, "154"), "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/154.0.0.0 Safari/537.36"; got != want {
		t.Errorf("windows UA = %q, want %q", got, want)
	}
	if got, want := canonicalUserAgent(OSMac, "154"), "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/154.0.0.0 Safari/537.36"; got != want {
		t.Errorf("mac UA = %q, want %q", got, want)
	}
	if got := uaMajorVersion("... Chrome/128.0.6613.138 Safari/537.36"); got != "128" {
		t.Errorf("uaMajorVersion = %q, want 128", got)
	}
}

// TestIdentityFromConfig 版本与品牌来自真实浏览器，系统相关字段由账号系统派生，几何按系统规则推算
func TestIdentityFromConfig(t *testing.T) {
	win := &FingerprintConfig{UserID: "acct-win"}
	win.Browser.UserAgent = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/128.0.0.0 Safari/537.36"
	win.Browser.Languages = []string{"ja-JP", "ja"}
	win.Browser.Language = "ja-JP"
	win.Browser.HardwareConcurrency = 8
	win.Browser.PlatformVersion = "15.0.0"
	win.Timezone.Timezone = "Asia/Tokyo"
	win.Screen = ScreenConfig{Width: 1536, Height: 864, DevicePixelRatio: 1.25}
	win.WebGL.Vendor, win.WebGL.Renderer = windowsGPUs[0].Value.Vendor, windowsGPUs[0].Value.Renderer

	id, err := identityFromConfig(win, testHost())
	if err != nil {
		t.Fatal(err)
	}
	checks := []struct {
		name      string
		got, want any
	}{
		{"UserAgent", id.UserAgent, "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/154.0.0.0 Safari/537.36"},
		{"Platform", id.Platform, "Win32"},
		{"AcceptLanguage", id.AcceptLanguage, "ja-JP,ja"},
		{"Locale", id.Locale(), "ja-JP"},
		{"Timezone", id.Timezone, "Asia/Tokyo"},
		{"Metadata.Platform", id.Metadata.Platform, "Windows"},
		{"Metadata.PlatformVersion", id.Metadata.PlatformVersion, "15.0.0"},
		{"Metadata.Architecture", id.Metadata.Architecture, "x86"},
		{"Metadata.Bitness", id.Metadata.Bitness, "64"},
		{"Metadata.FullVersion", id.Metadata.FullVersionList[1].Version, "154.0.8037.92"},
		{"AvailHeight", id.Screen.AvailHeight, 816}, // 864 - 48：Win11 任务栏 48 CSS 像素，任何缩放比例都一样
		{"OuterHeight", id.Screen.OuterHeight, 816},
		{"InnerHeight", id.Screen.InnerHeight, 729}, // 816 - 87
		{"InnerWidth", id.Screen.InnerWidth, 1536},
		{"GPU", id.GPU.Renderer, windowsGPUs[0].Value.Renderer},
	}
	for _, c := range checks {
		if c.got != c.want {
			t.Errorf("%s = %v, want %v", c.name, c.got, c.want)
		}
	}
	// Win10 任务栏 40 CSS 像素：1920×1080 在 150% 缩放下是 1280×720，可用高度 680
	win10 := geometryFor(OSWindows, screenSpec{Width: 1280, Height: 720, DPR: 1.5}, "10.0.0")
	if win10.AvailHeight != 680 {
		t.Errorf("Win10 1280x720@1.5 availHeight = %d, want 680", win10.AvailHeight)
	}
	if !slices.Equal(id.Metadata.FormFactors, []string{"Desktop"}) {
		t.Errorf("FormFactors = %v, want [Desktop]", id.Metadata.FormFactors)
	}
	if id.NoiseSeed == 0 {
		t.Error("NoiseSeed is 0; fingerprint accounts must carry noise")
	}
	again, _ := identityFromConfig(win, testHost())
	if again.NoiseSeed != id.NoiseSeed {
		t.Error("NoiseSeed is not deterministic per account")
	}

	mac := &FingerprintConfig{UserID: "acct-mac"}
	mac.Browser.UserAgent = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36"
	mac.Browser.Languages = []string{"zh-CN", "zh"}
	mac.Browser.HardwareConcurrency = 10
	mac.Browser.PlatformVersion = "15.5.0"
	mac.Timezone.Timezone = "Asia/Shanghai"
	mac.Screen = ScreenConfig{Width: 1512, Height: 982, DevicePixelRatio: 2}
	mac.WebGL.Vendor, mac.WebGL.Renderer = macGPUs[0].Value.Vendor, macGPUs[0].Value.Renderer
	mid, err := identityFromConfig(mac, testHost())
	if err != nil {
		t.Fatal(err)
	}
	if mid.Platform != "MacIntel" || mid.Metadata.Platform != "macOS" || mid.Metadata.Architecture != "arm" {
		t.Errorf("mac identity platform fields = %q / %q / %q", mid.Platform, mid.Metadata.Platform, mid.Metadata.Architecture)
	}
	// MacBook Pro 14 有刘海，菜单栏 38
	if mid.Screen.AvailTop != 38 || mid.Screen.AvailHeight != 944 || mid.Screen.InnerHeight != 857 {
		t.Errorf("mac geometry = %+v, want availTop 38 / availHeight 944 / innerHeight 857", mid.Screen)
	}

	bad := *win
	bad.WebGL.Renderer = "ANGLE (AMD, AMD Radeon(TM) Graphics Direct3D11 vs_5_0 ps_5_0, D3D11) (Build 26453)"
	if _, err := identityFromConfig(&bad, testHost()); err == nil {
		t.Error("renderer outside the pool must be rejected (config not normalized)")
	}
}

// TestHostIdentity 不设指纹：保留真实值，只去掉无界面痕迹；有界面（真实屏幕）时不调整几何、不改 WebGL
func TestHostIdentity(t *testing.T) {
	id := hostIdentity(testHost())
	if id.UserAgent != "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/154.0.0.0 Safari/537.36" {
		t.Errorf("UserAgent = %q", id.UserAgent)
	}
	if id.Platform != "Linux x86_64" || id.Metadata.Platform != "Linux" {
		t.Errorf("platform changed: %q / %q", id.Platform, id.Metadata.Platform)
	}
	if id.Screen.Width != 1920 || id.Screen.Height != 1080 || id.Screen.InnerHeight != id.Screen.AvailHeight-browserUIHeight {
		t.Errorf("headless screen not replaced: %+v", id.Screen)
	}
	if id.GPU == nil || id.GPU.Renderer != linuxHostGPU.Renderer || id.GPU.Caps != nil {
		t.Errorf("SwiftShader not masked by vendor/renderer only: %+v", id.GPU)
	}
	if id.NoiseSeed != 0 {
		t.Error("default path must not add noise")
	}

	headful := testHost()
	headful.UserAgent = "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/154.0.0.0 Safari/537.36"
	headful.Screen = [3]float64{1920, 1080, 1}
	headful.WebGLRenderer = "ANGLE (NVIDIA, NVIDIA GeForce RTX 3060, OpenGL 4.6)"
	hid := hostIdentity(headful)
	if hid.Screen.Width != 0 {
		t.Errorf("headful host screen must be left alone, got %+v", hid.Screen)
	}
	if hid.GPU != nil {
		t.Errorf("real GPU must not be masked, got %+v", hid.GPU)
	}
}
