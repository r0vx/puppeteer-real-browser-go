package browser

import (
	"encoding/json"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

// legacyConfig 构造一份旧生成器风格的指纹
func legacyConfig(userID, ua, vendor, renderer, lang string, langs []string, tz string, w, h int, dpr float64, cores int) *FingerprintConfig {
	c := &FingerprintConfig{UserID: userID}
	c.Browser.UserAgent = ua
	c.Browser.Platform = "Linux x86_64"
	c.Browser.Language, c.Browser.Languages = lang, langs
	c.Browser.HardwareConcurrency = cores
	c.Timezone.Timezone = tz
	c.Screen = ScreenConfig{Width: w, Height: h, DevicePixelRatio: dpr}
	c.WebGL.Vendor, c.WebGL.Renderer = vendor, renderer
	c.TLSConfig.JA3 = "keep-me" // 与身份无关的字段必须原样保留
	return c
}

// TestNormalizeLegacyFingerprint 旧生成器的各类坏数据规范化后自洽、幂等，且不动无关字段
func TestNormalizeLegacyFingerprint(t *testing.T) {
	cases := []struct {
		name  string
		cfg   *FingerprintConfig
		check func(t *testing.T, c *FingerprintConfig)
	}{
		{"linux ua becomes windows", legacyConfig("u1",
			"Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/111.0.5563.146 Safari/537.36",
			"Google Inc. (NVIDIA)", "ANGLE (NVIDIA, NVIDIA GeForce RTX 3070 Direct3D11 vs_5_0 ps_5_0, D3D11)",
			"ja-JP", []string{"es-ES", "es", "en"}, "America/Sao_Paulo", 3840, 1600, 1, 4),
			func(t *testing.T, c *FingerprintConfig) {
				if c.Browser.UserAgent != canonicalUserAgent(OSWindows, "111") {
					t.Errorf("UA = %q", c.Browser.UserAgent)
				}
				if c.Browser.Platform != "Win32" {
					t.Errorf("Platform = %q", c.Browser.Platform)
				}
				if !slices.Equal(c.Browser.Languages, []string{"ja-JP", "ja"}) {
					t.Errorf("Languages = %v", c.Browser.Languages)
				}
				if c.Timezone.Timezone != "America/Sao_Paulo" {
					t.Errorf("valid timezone changed to %q", c.Timezone.Timezone)
				}
				if _, ok := screenSpecFor(OSWindows, c.Screen.Width, c.Screen.Height, c.Screen.DevicePixelRatio); !ok {
					t.Errorf("implausible 3840x1600 not re-picked from pool: %+v", c.Screen)
				}
				if gpuFamily(c.WebGL.Vendor, c.WebGL.Renderer) != "NVIDIA" {
					t.Errorf("GPU vendor family not kept: %q", c.WebGL.Renderer)
				}
				if c.Browser.HardwareConcurrency != 4 {
					t.Errorf("valid core count changed to %d", c.Browser.HardwareConcurrency)
				}
			}},
		{"broken windows ua fixed, plausible screen kept", legacyConfig("u2",
			"Mozilla/5.0 (Windows NT 10.0; Win64; x64 10.0) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/128.0.6613.138 Safari/537.36",
			"Google Inc. (AMD) ", "ANGLE (AMD, AMD Radeon(TM) Graphics Direct3D11 vs_5_0 ps_5_0, D3D11) (Build 26453)",
			"zh-CN", []string{"zh-CN", "zh"}, "Asia/Shanghai", 1920, 1080, 1, 8),
			func(t *testing.T, c *FingerprintConfig) {
				if c.Browser.UserAgent != canonicalUserAgent(OSWindows, "128") {
					t.Errorf("UA = %q", c.Browser.UserAgent)
				}
				if c.Screen.Width != 1920 || c.Screen.Height != 1080 || c.Screen.DevicePixelRatio != 1 {
					t.Errorf("plausible screen changed: %+v", c.Screen)
				}
				if c.Screen.AvailHeight >= c.Screen.Height {
					t.Errorf("avail height %d not below screen height", c.Screen.AvailHeight)
				}
				if strings.Contains(c.WebGL.Renderer, "Build") || gpuFamily(c.WebGL.Vendor, c.WebGL.Renderer) != "AMD" {
					t.Errorf("renderer = %q, want an AMD pool entry", c.WebGL.Renderer)
				}
			}},
		{"mac keeps mac, gpu and cores follow mac pools", legacyConfig("u3",
			"Mozilla/5.0 (Macintosh; Intel Mac OS X 13_5_2) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.6099.234 Safari/537.36",
			"Google Inc. (AMD)", "ANGLE (AMD, AMD Radeon RX 6600 XT Direct3D11 vs_5_0 ps_5_0, D3D11)",
			"en-US", []string{"en-US", "en"}, "America/New_York", 1440, 900, 2, 6),
			func(t *testing.T, c *FingerprintConfig) {
				if c.Browser.UserAgent != canonicalUserAgent(OSMac, "120") || c.Browser.Platform != "MacIntel" {
					t.Errorf("mac UA/platform = %q / %q", c.Browser.UserAgent, c.Browser.Platform)
				}
				if _, ok := lookupGPU(OSMac, c.WebGL.Renderer); !ok {
					t.Errorf("mac account has non-mac GPU %q", c.WebGL.Renderer)
				}
				if c.Browser.HardwareConcurrency != 8 && c.Browser.HardwareConcurrency != 10 && c.Browser.HardwareConcurrency != 12 {
					t.Errorf("mac cores = %d", c.Browser.HardwareConcurrency)
				}
			}},
		{"utc timezone derived from language", legacyConfig("u4",
			"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36",
			"", "", "zh-CN", nil, "UTC", 1366, 768, 1, 8),
			func(t *testing.T, c *FingerprintConfig) {
				if c.Timezone.Timezone != "Asia/Shanghai" {
					t.Errorf("Timezone = %q, want Asia/Shanghai", c.Timezone.Timezone)
				}
				if !slices.Equal(c.Browser.Languages, []string{"zh-CN", "zh"}) {
					t.Errorf("Languages = %v", c.Browser.Languages)
				}
			}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if !tc.cfg.Normalize() {
				t.Fatal("Normalize reported no change on a legacy config")
			}
			tc.check(t, tc.cfg)
			if tc.cfg.SchemaVersion != fingerprintSchemaVersion {
				t.Errorf("SchemaVersion = %d", tc.cfg.SchemaVersion)
			}
			if tc.cfg.TLSConfig.JA3 != "keep-me" {
				t.Error("unrelated field was modified")
			}
			if _, err := identityFromConfig(tc.cfg, testHost()); err != nil {
				t.Errorf("normalized config cannot build an identity: %v", err)
			}
			first, _ := json.Marshal(tc.cfg)
			if tc.cfg.Normalize() {
				t.Error("Normalize is not idempotent")
			}
			if second, _ := json.Marshal(tc.cfg); string(first) != string(second) {
				t.Error("second Normalize changed the config")
			}
		})
	}
}

// TestGenerateFingerprintFor 新指纹按系统生成、按用户 ID 确定，且能直接派生身份
func TestGenerateFingerprintFor(t *testing.T) {
	for _, osf := range []OSFamily{OSWindows, OSMac} {
		a, b := generateFingerprintFor("gen-user", osf), generateFingerprintFor("gen-user", osf)
		ja, _ := json.Marshal(a)
		jb, _ := json.Marshal(b)
		if string(ja) != string(jb) {
			t.Errorf("%s: generation is not deterministic per user", osf)
		}
		if osFromUserAgent(a.Browser.UserAgent) != osf {
			t.Errorf("%s: generated UA %q", osf, a.Browser.UserAgent)
		}
		if _, err := identityFromConfig(a, testHost()); err != nil {
			t.Errorf("%s: %v", osf, err)
		}
	}
}

// TestFingerprintManagerNormalizesAndBindsUA 新账号按绑定 UA 选系统；旧文件加载时规范化并回写；损坏文件返回错误而不是被覆盖
func TestFingerprintManagerNormalizesAndBindsUA(t *testing.T) {
	dir := t.TempDir()
	m, err := NewUserFingerprintManager(dir)
	if err != nil {
		t.Fatal(err)
	}

	cfg, err := m.GetOrCreateUserFingerprint("new-mac", &FingerprintInitParams{
		UserAgent: "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36",
		Language:  "zh-CN",
	})
	if err != nil {
		t.Fatal(err)
	}
	if osFromUserAgent(cfg.Browser.UserAgent) != OSMac || cfg.Browser.Platform != "MacIntel" {
		t.Errorf("new account did not follow bound Mac UA: %q / %q", cfg.Browser.UserAgent, cfg.Browser.Platform)
	}

	// 只设语言不设时区：时区按语言推导（不能沿用生成器默认语言对应的时区）
	ja, err := m.GetOrCreateUserFingerprint("lang-only", &FingerprintInitParams{Language: "ja-JP"})
	if err != nil {
		t.Fatal(err)
	}
	if ja.Timezone.Timezone != "Asia/Tokyo" || !slices.Equal(ja.Browser.Languages, []string{"ja-JP", "ja"}) {
		t.Errorf("language-only account: timezone %q languages %v, want Asia/Tokyo [ja-JP ja]", ja.Timezone.Timezone, ja.Browser.Languages)
	}

	legacy := legacyConfig("old-linux",
		"Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/111.0.5563.146 Safari/537.36",
		"Google Inc. (NVIDIA)", "NVIDIA GeForce RTX 3060/PCIe/SSE2", "zh-CN", []string{"zh-CN", "zh"}, "Asia/Shanghai", 1920, 1080, 1, 8)
	data, _ := json.Marshal(legacy)
	path := filepath.Join(dir, "old-linux.json")
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatal(err)
	}
	got, err := m.GetOrCreateUserFingerprint("old-linux", nil)
	if err != nil {
		t.Fatal(err)
	}
	if got.Browser.Platform != "Win32" {
		t.Errorf("legacy Linux account not normalized: %q", got.Browser.Platform)
	}
	reread, err := LoadFingerprintConfigFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if reread.SchemaVersion != fingerprintSchemaVersion || reread.Browser.Platform != "Win32" {
		t.Errorf("normalized config not written back: schema %d platform %q", reread.SchemaVersion, reread.Browser.Platform)
	}

	if err := os.WriteFile(filepath.Join(dir, "broken.json"), []byte("{not json"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := m.GetOrCreateUserFingerprint("broken", nil); err == nil {
		t.Error("corrupt fingerprint file must return an error, not be regenerated")
	}
	if b, _ := os.ReadFile(filepath.Join(dir, "broken.json")); string(b) != "{not json" {
		t.Error("corrupt fingerprint file was overwritten")
	}
}
