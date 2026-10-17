# 安卓身份（M1）Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** `ConnectOptions.Mobile` 为新账号生成安卓 Chrome 身份。主页面、跨站 iframe、Worker、弹窗全部是同一台手机，与真机基准（`test/baseline/android-chrome-pgkm10.json`）在本库能控制的项目上一致。

**Architecture:**
- 沿用现有身份流程，加一个系统族 `OSAndroid`。
- 指纹文件用新的 `device` 段标记移动身份；老指纹没有这一段，永远是桌面身份。
- 机型档案（机型、屏幕、系统版本、GPU）整体抽取，GPU 统一用真机实测的 Mali-G610 MC6。
- 各项的实现方式：
  - CDP 负责 UA、Client Hints、移动视口、触屏；
  - profile 偏好关掉 PDF 查看器（插件原生为空）；
  - 注入脚本补齐其余 JS 接口与 WebGL；
  - Linux 上用进程级 fontconfig 提供安卓字体。

**Tech Stack:** Go 1.25 标准库，Chrome DevTools Protocol，注入 JS（沿用 `identity_script.go` 的原生伪装工具），fontconfig。

**Spec:** `docs/superpowers/specs/2026-10-17-android-identity-design.md`

## Global Constraints

- 已有的桌面身份行为完全不变：现有测试全部照常通过，不得为让新测试通过而改旧断言。
- 老指纹（没有 `device` 段）永不变成移动身份，即使 UA 是手机 UA。
- WebGL 的参数、扩展、精度只取真机基准里的值。
- 移动身份只支持默认 CustomCDP 通道，并且必须设 `FingerprintUserID`。
- 代码注释用中文，每个函数都有注释；提交信息以 `Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>` 结尾。
- 集成测试启动真 Chrome；字体相关断言只在 Linux 上运行（macOS 由 CoreText 管字体）。

## Review Focus

1. triumph PC 流程里绑定了手机 UA 的老指纹（没有 `device` 段）：加载后仍是 Windows 身份，文件里不会出现 `device`。由 Task 1 的 `TestMobileUserAgentFileStaysDesktop` 覆盖。
2. `Mobile` 与已有文件的类型不一致：`Connect` 报错，文件一个字节都不改。由 Task 1 的 `TestFingerprintForOptionsMobile` 覆盖。
3. 跨站 iframe 与 Worker 里也是手机：`maxTouchPoints` 为 5，`pointer: coarse` 为真，`deviceMemory`、`platform`、Client Hints 一致。由 Task 2 的 `TestAndroidIdentityAcrossSurfaces` 覆盖。
4. 页面请求一个手机上没有的 WebGL 扩展：返回 null，且该扩展没有在上下文里被悄悄启用（不能用它的枚举值）。由 Task 4 的 `TestAndroidWebGL` 覆盖。
5. 桌面身份不受安卓注入脚本影响（`hid`、`SharedWorker`、`Notification` 构造照旧）。由 Task 3 的 `TestAndroidSurfaceAPIs/desktop` 覆盖。

---

### Task 1: 移动指纹数据、机型池、规范化与 `Mobile` 开关

**Files:**
- Create: `pkg/browser/identity_android.go`
- Modify:
  - `pkg/browser/fingerprint_config.go`：`DeviceConfig`、`Normalize`、`generateFingerprintFor`；
  - `pkg/browser/identity_pools.go`：`gpuProfile.Mobile`；
  - `pkg/browser/identity.go`：`screenGeometry.Mobile`、`isIOSUserAgent`；
  - `pkg/browser/user_fingerprint_manager.go`、`pkg/browser/identity_apply.go`、`pkg/browser/types.go`、`pkg/browser/browser.go`。
- Test: `pkg/browser/android_identity_test.go`

**Interfaces:**
- Produces：
  - 常量 `OSAndroid`、`deviceKindAndroid`、`androidPlatform`、`androidViewportInset`、`androidCores`、`androidDeviceMemory`、`androidTouchPoints`。
  - 类型 `androidDevice{Model string; Screen screenSpec; PlatformVersion string; GPU gpuProfile; Source string}`、`mobileWebGL{Params, Params2 map[int]any; Extensions, Extensions2 []string; Precision map[[2]int][3]int}`、`DeviceConfig{Kind, Model string; DeviceMemory int}`。
  - 数据：`maliG610 gpuProfile`、`androidDevices []weighted[androidDevice]`。
  - 函数：
    - `lookupAndroidDevice(model string) (androidDevice, bool)`、`androidUserAgent(major string) string`、`androidGeometry(s screenSpec) screenGeometry`；
    - `configOS(cfg *FingerprintConfig) OSFamily`、`normalizeAndroid(config *FingerprintConfig, seed func(string) uint64)`、`isIOSUserAgent(ua string) bool`、`validateMobileOptions(opts *ConnectOptions) error`。
  - 字段：`FingerprintConfig.Device *DeviceConfig`、`screenGeometry.Mobile bool`、`gpuProfile.Mobile *mobileWebGL`、`ConnectOptions.Mobile bool`、`FingerprintInitParams.Mobile bool`。

- [ ] **Step 1: 写失败的测试**

创建 `pkg/browser/android_identity_test.go`：

```go
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
```

- [ ] **Step 2: 运行测试，确认失败**

Run: `go test -count=1 -run 'TestAndroid|TestMobileUserAgentFileStaysDesktop|TestFingerprintForOptionsMobile|TestConnectRejectsMobileMisuse' ./pkg/browser/`
Expected: FAIL，编译错误（`undefined: androidUserAgent`、`DeviceConfig`、`deviceKindAndroid` 等）

- [ ] **Step 3: 实现**

(a) 创建 `pkg/browser/identity_android.go`：

```go
package browser

// 安卓身份（移动身份子项目 M1）：机型档案、派生规则与 WebGL 实测值。
// 设计见 docs/superpowers/specs/2026-10-17-android-identity-design.md

// OSAndroid 安卓（Android Chrome）
const OSAndroid OSFamily = "android"

const (
	// deviceKindAndroid 指纹 device 段的类型：安卓
	deviceKindAndroid = "android"
	// androidPlatform 安卓 Chrome 的 navigator.platform（真机实测）
	androidPlatform = "Linux armv81"
	// androidViewportInset 状态栏、地址栏、手势条的总高度（CSS 像素即 dp，各机型相同；真机实测 804 − 700）
	androidViewportInset = 104
	// androidCores 天玑 8100 / 8200 都是 8 核
	androidCores = 8
	// androidDeviceMemory Chrome 把 navigator.deviceMemory 封顶为 8
	androidDeviceMemory = 8
	// androidTouchPoints 手机的 navigator.maxTouchPoints（真机实测）
	androidTouchPoints = 5
)

// androidDevice 一款安卓手机的整机档案：机型、屏幕、系统版本、GPU 是一个整体，账号整体抽取，不单独拼凑
type androidDevice struct {
	Model           string     // Client Hints 的 model（真实机型只在这里出现，UA 里冻结为 K）
	Screen          screenSpec // CSS 像素与 DPR
	PlatformVersion string     // Client Hints 的 platformVersion
	GPU             gpuProfile
	Source          string // 真机实测，或整理来源
}

// mobileWebGL 手机 GPU 在安卓 Chrome 下的完整 WebGL 表面：能力参数、扩展列表、着色器精度
type mobileWebGL struct {
	Params      map[int]any       // WebGL1 / WebGL2 共有的 getParameter 覆盖（GLenum → 数值或 [a, b]）
	Params2     map[int]any       // 只属于 WebGL2 的参数
	Extensions  []string          // WebGL1 的 getSupportedExtensions()
	Extensions2 []string          // WebGL2 的 getSupportedExtensions()
	Precision   map[[2]int][3]int // {着色器, 精度类型} → {rangeMin, rangeMax, precision}
}

// maliG610 Mali-G610 MC6（天玑 8100 / 8200）在 Chrome 138 / Android 15 上的实测值，见 test/baseline/android-chrome-pgkm10.json。
// 只覆盖能力参数；RED_BITS、SAMPLES 等随上下文属性变化的值保持原生
var maliG610 = gpuProfile{
	Vendor: "ARM", Renderer: "Mali-G610 MC6",
	Mobile: &mobileWebGL{
		Params: map[int]any{
			3379:  4096,                      // MAX_TEXTURE_SIZE
			34076: 4096,                      // MAX_CUBE_MAP_TEXTURE_SIZE
			34024: 16383,                     // MAX_RENDERBUFFER_SIZE
			3386:  []float64{16383, 16383},   // MAX_VIEWPORT_DIMS
			34921: 32,                        // MAX_VERTEX_ATTRIBS
			36347: 4096,                      // MAX_VERTEX_UNIFORM_VECTORS
			36349: 4096,                      // MAX_FRAGMENT_UNIFORM_VECTORS
			36348: 31,                        // MAX_VARYING_VECTORS
			34930: 16,                        // MAX_TEXTURE_IMAGE_UNITS
			35660: 16,                        // MAX_VERTEX_TEXTURE_IMAGE_UNITS
			35661: 96,                        // MAX_COMBINED_TEXTURE_IMAGE_UNITS
			33901: []float64{1, 1024},        // ALIASED_POINT_SIZE_RANGE
			33902: []float64{1, 4095.9375},   // ALIASED_LINE_WIDTH_RANGE
			3408:  8,                         // SUBPIXEL_BITS
		},
		Params2: map[int]any{
			32883: 16383,      // MAX_3D_TEXTURE_SIZE
			35071: 4096,       // MAX_ARRAY_TEXTURE_LAYERS
			34852: 8,          // MAX_DRAW_BUFFERS
			36063: 8,          // MAX_COLOR_ATTACHMENTS
			36183: 4,          // MAX_SAMPLES
			35376: 65536,      // MAX_UNIFORM_BLOCK_SIZE
			35375: 216,        // MAX_UNIFORM_BUFFER_BINDINGS
			35374: 216,        // MAX_COMBINED_UNIFORM_BLOCKS
			35658: 16384,      // MAX_VERTEX_UNIFORM_COMPONENTS
			35657: 16384,      // MAX_FRAGMENT_UNIFORM_COMPONENTS
			35659: 124,        // MAX_VARYING_COMPONENTS
			35968: 4,          // MAX_TRANSFORM_FEEDBACK_SEPARATE_COMPONENTS
			36203: 2147483647, // MAX_ELEMENT_INDEX
			37137: 0,          // MAX_SERVER_WAIT_TIMEOUT
			34045: 256,        // MAX_TEXTURE_LOD_BIAS
			37154: 128,        // MAX_VERTEX_OUTPUT_COMPONENTS
			37157: 124,        // MAX_FRAGMENT_INPUT_COMPONENTS
			35077: 7,          // MAX_PROGRAM_TEXEL_OFFSET
			35076: -8,         // MIN_PROGRAM_TEXEL_OFFSET
		},
		Extensions: []string{"ANGLE_instanced_arrays", "EXT_blend_minmax", "EXT_float_blend", "EXT_texture_filter_anisotropic", "EXT_sRGB",
			"OES_element_index_uint", "OES_fbo_render_mipmap", "OES_standard_derivatives", "OES_texture_float_linear", "OES_vertex_array_object",
			"WEBGL_compressed_texture_astc", "WEBGL_compressed_texture_etc", "WEBGL_compressed_texture_etc1", "WEBGL_debug_renderer_info",
			"WEBGL_debug_shaders", "WEBGL_depth_texture", "WEBGL_lose_context", "WEBGL_multi_draw"},
		Extensions2: []string{"EXT_color_buffer_float", "EXT_float_blend", "EXT_texture_filter_anisotropic", "OES_texture_float_linear",
			"WEBGL_compressed_texture_astc", "WEBGL_compressed_texture_etc", "WEBGL_compressed_texture_etc1", "WEBGL_debug_renderer_info",
			"WEBGL_debug_shaders", "WEBGL_lose_context", "WEBGL_multi_draw"},
		Precision: map[[2]int][3]int{
			{35633, 36336}: {15, 15, 10}, {35633, 36337}: {15, 15, 10}, {35633, 36338}: {127, 127, 23}, // 顶点着色器 LOW / MEDIUM / HIGH_FLOAT
			{35633, 36339}: {15, 14, 0}, {35633, 36340}: {15, 14, 0}, {35633, 36341}: {31, 30, 0}, // 顶点着色器 LOW / MEDIUM / HIGH_INT
			{35632, 36336}: {15, 15, 10}, {35632, 36337}: {15, 15, 10}, {35632, 36338}: {127, 127, 23}, // 片元着色器 LOW / MEDIUM / HIGH_FLOAT
			{35632, 36339}: {15, 14, 0}, {35632, 36340}: {15, 14, 0}, {35632, 36341}: {31, 30, 0}, // 片元着色器 LOW / MEDIUM / HIGH_INT
		},
	},
}

// androidDevices 安卓机型池：GPU 全部是 Mali-G610 MC6，WebGL 取真机实测；整理机型的 DPR 按厂商默认显示密度推算（规格 4.1）
var androidDevices = []weighted[androidDevice]{
	{androidDevice{"PGKM10", screenSpec{360, 804, 3, false}, "15.0.0", maliG610, "真机实测：一加 Ace，天玑 8100-Max，1080×2412"}, 30},
	{androidDevice{"22041211AC", screenSpec{411, 914, 3.5, false}, "14.0.0", maliG610, "整理：Redmi K50，天玑 8100，1440×3200，560dpi"}, 14},
	{androidDevice{"22041216C", screenSpec{393, 895, 2.75, false}, "14.0.0", maliG610, "整理：Redmi Note 11T Pro，天玑 8100，1080×2460，440dpi"}, 14},
	{androidDevice{"RMX3560", screenSpec{360, 804, 3, false}, "14.0.0", maliG610, "整理：realme GT Neo3，天玑 8100，1080×2412，480dpi"}, 14},
	{androidDevice{"23046PNC9C", screenSpec{393, 873, 2.75, false}, "15.0.0", maliG610, "整理：Xiaomi Civi 3，天玑 8200-Ultra，1080×2400，440dpi"}, 14},
	{androidDevice{"22122RK93C", screenSpec{411, 914, 3.5, false}, "15.0.0", maliG610, "整理：Redmi K60E，天玑 8200，1440×3200，560dpi"}, 14},
}

// lookupAndroidDevice 按机型代号在池中查找
func lookupAndroidDevice(model string) (androidDevice, bool) {
	for _, d := range androidDevices {
		if d.Value.Model == model {
			return d.Value, true
		}
	}
	return androidDevice{}, false
}

// androidUserAgent Chrome 精简格式的安卓 UA：系统段冻结为 Android 10; K，版本只保留主版本
func androidUserAgent(major string) string {
	return "Mozilla/5.0 (Linux; Android 10; K) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/" + major + ".0.0.0 Mobile Safari/537.36"
}

// androidGeometry 手机的屏幕与视口：可用区域即整屏；视口扣掉状态栏、地址栏、手势条；窗口外框等于视口（与真机一致）
func androidGeometry(s screenSpec) screenGeometry {
	inner := s.Height - androidViewportInset
	return screenGeometry{
		Width: s.Width, Height: s.Height, DPR: s.DPR, AvailWidth: s.Width, AvailHeight: s.Height,
		OuterWidth: s.Width, OuterHeight: inner, InnerWidth: s.Width, InnerHeight: inner, Mobile: true,
	}
}

// configOS 指纹声称的系统：有 device 段的按其类型（移动身份），否则按 UA 判定 Windows / macOS——老指纹永不变成移动身份
func configOS(cfg *FingerprintConfig) OSFamily {
	if cfg.Device != nil && cfg.Device.Kind == deviceKindAndroid {
		return OSAndroid
	}
	return osFromUserAgent(cfg.Browser.UserAgent)
}

// normalizeAndroid 安卓指纹的设备相关字段：机型不在池中时按用户 ID 重抽，其余字段一律取自机型
func normalizeAndroid(config *FingerprintConfig, seed func(string) uint64) {
	dev, ok := lookupAndroidDevice(config.Device.Model)
	if !ok {
		dev = pickWeighted(androidDevices, seed("device"))
	}
	major := uaMajorVersion(config.Browser.UserAgent)
	if major == "" {
		major = "120"
	}
	config.Device.Model, config.Device.DeviceMemory = dev.Model, androidDeviceMemory
	config.Browser.UserAgent = androidUserAgent(major)
	config.Browser.Platform = androidPlatform
	config.Browser.Vendor = "Google Inc."
	config.Browser.PlatformVersion = dev.PlatformVersion
	config.Browser.HardwareConcurrency = androidCores
	config.Browser.MaxTouchPoints = androidTouchPoints
	config.Screen.Width, config.Screen.Height, config.Screen.DevicePixelRatio = dev.Screen.Width, dev.Screen.Height, dev.Screen.DPR
	config.Screen.AvailWidth, config.Screen.AvailHeight = dev.Screen.Width, dev.Screen.Height
	config.Screen.ColorDepth, config.Screen.PixelDepth = 24, 24
	config.WebGL.Vendor, config.WebGL.Renderer = dev.GPU.Vendor, dev.GPU.Renderer
	config.WebGL.Version = "WebGL 1.0 (OpenGL ES 2.0 Chromium)"
	config.WebGL.ShadingLanguageVersion = "WebGL GLSL ES 1.0 (OpenGL ES GLSL ES 1.0 Chromium)"
	config.WebGL.MaxTextureSize = dev.GPU.Mobile.Params[3379].(int)
	config.WebGL.MaxRenderbufferSize = dev.GPU.Mobile.Params[34024].(int)
}
```

(b) `pkg/browser/identity_pools.go`：`gpuProfile` 加字段：

```go
type gpuProfile struct {
	Vendor   string // UNMASKED_VENDOR_WEBGL
	Renderer string // UNMASKED_RENDERER_WEBGL
	Caps     *webglCaps
	Mobile   *mobileWebGL // 手机 GPU 的完整 WebGL 表面（安卓身份）；桌面为 nil
}
```

(c) `pkg/browser/identity.go`：
- `screenGeometry` 末尾加一行字段：`Mobile bool // 手机：视口按移动端模拟（mobile: true），不调整窗口外框`。
- 在 `osFromUserAgent` 之后加：

```go
// isIOSUserAgent UA 是否为 iPhone / iPad
func isIOSUserAgent(ua string) bool {
	return strings.Contains(ua, "iPhone") || strings.Contains(ua, "iPad")
}
```

(d) `pkg/browser/fingerprint_config.go`：
- `FingerprintConfig` 在 `SchemaVersion` 之后加字段：

```go
	// Device 移动身份的设备；只有移动指纹才有这一段，老指纹没有，因而永远是桌面身份
	Device *DeviceConfig `json:"device,omitempty"`
```

- 在 `ScreenConfig` 之前加：

```go
// DeviceConfig 移动身份的设备
type DeviceConfig struct {
	Kind         string `json:"kind"`          // android（iOS 见子项目 M3）
	Model        string `json:"model"`         // Client Hints 的 model，即机型代号
	DeviceMemory int    `json:"device_memory"` // navigator.deviceMemory
}
```

- `Normalize` 整个替换为：

```go
// Normalize 把指纹规范化为自洽的身份（Windows / macOS，或有 device 段时的安卓），返回是否有改动（调用方据此回写文件）。
// 只修正身份相关字段，其余字段原样保留；对已规范化的配置幂等
func (config *FingerprintConfig) Normalize() bool {
	before, _ := json.Marshal(config)
	osf := configOS(config)
	seed := func(purpose string) uint64 { return identitySeed(config.UserID, purpose) }

	if osf == OSAndroid {
		normalizeAndroid(config, seed)
	} else {
		major := uaMajorVersion(config.Browser.UserAgent)
		if major == "" {
			major = "120"
		}
		config.Browser.UserAgent = canonicalUserAgent(osf, major)
		config.Browser.Platform = platformFor(osf)
		config.Browser.Vendor = "Google Inc."
		if !containsValue(platformVersionsFor(osf), config.Browser.PlatformVersion) {
			config.Browser.PlatformVersion = pickWeighted(platformVersionsFor(osf), seed("platform-version"))
		}
	}

	// 语言：languages 首项必须等于 language
	if config.Browser.Language == "" {
		config.Browser.Language = "zh-CN"
	}
	if len(config.Browser.Languages) == 0 || config.Browser.Languages[0] != config.Browser.Language {
		config.Browser.Languages = defaultLanguages(config.Browser.Language)
	}

	// 时区：必须是可加载的地区时区（UTC 等不像真实用户），否则由语言推导
	if !plausibleTimezone(config.Timezone.Timezone) {
		config.Timezone.Timezone, _ = getTimezoneForLanguage(config.Browser.Language)
	}

	if osf != OSAndroid {
		// 屏幕：不像该系统的真实设备时按用户 ID 重选；可用区域按系统规则重算
		if !plausibleScreen(osf, config.Screen) {
			s := pickWeighted(screensFor(osf), seed("screen"))
			config.Screen.Width, config.Screen.Height, config.Screen.DevicePixelRatio = s.Width, s.Height, s.DPR
		}
		spec, _ := screenSpecFor(osf, config.Screen.Width, config.Screen.Height, config.Screen.DevicePixelRatio)
		g := geometryFor(osf, spec, config.Browser.PlatformVersion)
		config.Screen.AvailWidth, config.Screen.AvailHeight = g.AvailWidth, g.AvailHeight
		config.Screen.ColorDepth, config.Screen.PixelDepth = 24, 24

		if !containsValue(coresFor(osf), config.Browser.HardwareConcurrency) {
			config.Browser.HardwareConcurrency = pickWeighted(coresFor(osf), seed("cores"))
		}

		// 显卡：不在池中（含 "(Build N)" 等不可能的串）时尽量在同厂商里重选，能力参数与之对齐
		gpu, ok := lookupGPU(osf, config.WebGL.Renderer)
		if !ok {
			gpu = pickGPU(osf, gpuFamily(config.WebGL.Vendor, config.WebGL.Renderer), seed("gpu"))
		}
		config.WebGL.Vendor, config.WebGL.Renderer = gpu.Vendor, gpu.Renderer
		config.WebGL.Version = "WebGL 1.0 (OpenGL ES 2.0 Chromium)"
		config.WebGL.ShadingLanguageVersion = "WebGL GLSL ES 1.0 (OpenGL ES GLSL ES 1.0 Chromium)"
		config.WebGL.MaxTextureSize, config.WebGL.MaxRenderbufferSize = gpu.Caps.MaxTextureSize, gpu.Caps.MaxRenderbufferSize
	}

	config.SchemaVersion = fingerprintSchemaVersion
	after, _ := json.Marshal(config)
	return !bytes.Equal(before, after)
}
```

- `generateFingerprintFor` 中把 `c.Browser.UserAgent = canonicalUserAgent(osf, "120")` 替换为：

```go
	if osf == OSAndroid {
		c.Device = &DeviceConfig{Kind: deviceKindAndroid}
		c.Browser.UserAgent = androidUserAgent("120")
	} else {
		c.Browser.UserAgent = canonicalUserAgent(osf, "120")
	}
```

- `fingerprintSchemaVersion` 的注释改为：`// fingerprintSchemaVersion 当前指纹格式：2 = 自洽的身份（Windows / macOS，或带 device 段的移动身份；UA 只用于判定系统，版本运行时跟随真实浏览器）`。

(e) `pkg/browser/user_fingerprint_manager.go`：
- `FingerprintInitParams` 末尾加 `Mobile bool // 新账号生成移动身份（见 ConnectOptions.Mobile）`。
- `GetInitParamsFromOptions`：判断条件末尾加 `&& !opts.Mobile`，返回的结构体加 `Mobile: opts.Mobile,`。
- `GetOrCreateUserFingerprint` 里"新账号"那段（从注释 `// 新账号：系统由绑定的 UA 决定` 到 `config := generateFingerprintFor(userID, osFamily)` 之前）替换为：

```go
	// 新账号：要求移动身份时生成安卓（iOS 见子项目 M3），否则系统由绑定的 UA 决定，未提供 UA 时为 Windows
	osFamily := OSWindows
	switch {
	case initParams != nil && initParams.Mobile:
		if isIOSUserAgent(initParams.UserAgent) {
			return nil, fmt.Errorf("fingerprint %s: iOS identities are not supported yet", userID)
		}
		osFamily = OSAndroid
	case initParams != nil && initParams.UserAgent != "":
		osFamily = osFromUserAgent(initParams.UserAgent)
	}
```

(f) `pkg/browser/identity_apply.go`：`fingerprintForOptions` 的最后一行 `return manager.GetOrCreateUserFingerprint(...)` 替换为：

```go
	cfg, err := manager.GetOrCreateUserFingerprint(opts.FingerprintUserID, GetInitParamsFromOptions(opts))
	if err != nil {
		return nil, err
	}
	// 同一账号 ID 只能是一种设备：用错类型时报错，不改文件（避免账号的设备在电脑和手机之间跳变）
	if mobile := cfg.Device != nil; mobile != opts.Mobile {
		kind := "desktop"
		if mobile {
			kind = "mobile"
		}
		return nil, fmt.Errorf("fingerprint %s is a %s identity but ConnectOptions.Mobile is %v", opts.FingerprintUserID, kind, opts.Mobile)
	}
	return cfg, nil
```

(g) `pkg/browser/types.go`：在 `FingerprintDir` 字段之后加：

```go
	// Mobile 为新账号生成移动身份（安卓 Chrome；iPhone UA 为 iOS，见子项目 M3）。只对新账号起作用：已有指纹的系统永不改变，
	// 与已有指纹的类型不一致时 Connect 报错。需要设 FingerprintUserID，且只支持默认的 CustomCDP 通道
	Mobile bool `json:"mobile"`
```

(h) `pkg/browser/browser.go`：
- 在 `RealBrowser.Connect` 的 `if opts == nil { ... }` 块之后加：

```go
	if err := validateMobileOptions(opts); err != nil {
		return nil, err
	}
```

- 文件末尾加：

```go
// validateMobileOptions 移动身份必须有账号指纹，且只支持默认的 CustomCDP 通道（chromedp 通道只给主页面下发身份）
func validateMobileOptions(opts *ConnectOptions) error {
	if !opts.Mobile {
		return nil
	}
	if opts.FingerprintUserID == "" {
		return errors.New("mobile identity needs FingerprintUserID")
	}
	if opts.UseChromedp {
		return errors.New("mobile identity needs the default CustomCDP channel; unset UseChromedp")
	}
	return nil
}
```

  并在 import 中加 `"errors"`。

- [ ] **Step 4: 运行测试，确认通过**

Run: `go vet ./pkg/browser/ && go test -count=1 -run 'TestAndroid|TestMobileUserAgentFileStaysDesktop|TestFingerprintForOptionsMobile|TestConnectRejectsMobileMisuse|TestFingerprint|TestNormalize|TestIdentity' ./pkg/browser/`
Expected: PASS（包括原有的规范化、身份测试）

- [ ] **Step 5: 提交**

```bash
git add pkg/browser/
git commit -m "feat(identity): Android device pool, mobile fingerprints and the Mobile option

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
```

---

### Task 2: 安卓身份的派生、CDP 下发与启动

**Files:**
- Modify:
  - `pkg/browser/identity_android.go`：`androidIdentity`、`androidBattery`、`portraitOrientation`；
  - `pkg/browser/identity.go`：`Identity` 字段、`identityFromConfig`、`configGeometry`；
  - `pkg/browser/identity_apply.go`：`applyIdentity`、`viewportOverride`；
  - `pkg/browser/target_manager.go`、`pkg/browser/cdp_custom.go`（`SetViewport`）、`pkg/browser/launcher.go`。
- Test: `pkg/browser/android_identity_test.go`（单元）、`pkg/browser/android_integration_test.go`（新建，集成）

**Interfaces:**
- Consumes: Task 1 的全部 Produces。
- Produces：
  - `Identity.DeviceMemory int`、`Identity.Battery float64`；
  - `androidIdentity(cfg *FingerprintConfig, host hostInfo) (*Identity, error)`、`androidBattery(userID string) float64`；
  - `disablePDFViewer(userDataDir string) error`、`updatePreferences(userDataDir string, edit func(map[string]any)) error`；
  - 测试辅助：`androidFingerprintDir(t, userID, model string) string`、`connectAndroid(t, userID, model string) *BrowserInstance`、`awaitJS(t, p Page, expr string) string`。

- [ ] **Step 1: 写失败的测试**

在 `pkg/browser/android_identity_test.go` 末尾追加：

```go
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
```

创建 `pkg/browser/android_integration_test.go`：

```go
package browser

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

// androidFingerprintDir 写一份指定机型的安卓指纹，返回指纹目录
func androidFingerprintDir(t *testing.T, userID, model string) string {
	t.Helper()
	dir := t.TempDir()
	cfg := generateFingerprintFor(userID, OSAndroid)
	cfg.Device.Model = model
	cfg.Normalize()
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, userID+".json"), data, 0o644); err != nil {
		t.Fatal(err)
	}
	return dir
}

// connectAndroid 用指定机型的安卓身份启动（无界面，默认通道）
func connectAndroid(t *testing.T, userID, model string) *BrowserInstance {
	t.Helper()
	inst, err := Connect(t.Context(), &ConnectOptions{Headless: true, Mobile: true, FingerprintUserID: userID, FingerprintDir: androidFingerprintDir(t, userID, model)})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { inst.Close() })
	return inst
}

// awaitJS 执行返回 Promise<string> 的表达式并等待结果（CustomCDP 的 Evaluate 不等 Promise）
func awaitJS(t *testing.T, p Page, expr string) string {
	t.Helper()
	if _, err := p.Evaluate(`window.__a = undefined; Promise.resolve(` + expr + `).then(v => window.__a = String(v), e => window.__a = 'ERR ' + e); true`); err != nil {
		t.Fatal(err)
	}
	for deadline := time.Now().Add(10 * time.Second); time.Now().Before(deadline); time.Sleep(50 * time.Millisecond) {
		if v, _ := p.Evaluate(`window.__a`); v != nil {
			return fmt.Sprint(v)
		}
	}
	t.Fatalf("%s never settled", expr)
	return ""
}

// androidSurfaceJS 各表面（页面、iframe、Worker、弹窗）上报的身份信号
const androidSurfaceJS = `async () => {
  const h = await navigator.userAgentData.getHighEntropyValues(['model', 'platformVersion', 'formFactors']);
  return {ua: navigator.userAgent, platform: navigator.platform, deviceMemory: navigator.deviceMemory, cores: navigator.hardwareConcurrency,
    mobile: navigator.userAgentData.mobile, uaPlatform: navigator.userAgentData.platform, model: h.model, platformVersion: h.platformVersion,
    formFactors: h.formFactors.join(),
    touch: self.document ? navigator.maxTouchPoints : null, coarse: self.document ? matchMedia('(pointer: coarse)').matches : null,
    screen: self.document ? [screen.width, screen.height, devicePixelRatio].join() : null};
}`

// newAndroidSurfaceServers 主站（主页面 + 专用 Worker + 弹窗）与跨站 iframe 站点，记录请求头
func newAndroidSurfaceServers(t *testing.T) (main *httptest.Server, headers func(path string) http.Header) {
	t.Helper()
	var mu sync.Mutex
	seen := map[string]http.Header{}
	rec := func(r *http.Request) { mu.Lock(); seen[r.URL.Path] = r.Header.Clone(); mu.Unlock() }
	frame := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rec(r)
		fmt.Fprintf(w, `<script>(%s)().then(i => parent.postMessage({frame: 1, info: i}, '*'))</script>`, androidSurfaceJS)
	}))
	frameURL := strings.Replace(frame.URL, "127.0.0.1", "localhost", 1)
	main = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rec(r)
		switch r.URL.Path {
		case "/":
			fmt.Fprintf(w, `<script>
const info = %s;
window.__r = {};
info().then(i => window.__r.main = i);
new Worker('/worker.js').onmessage = e => window.__r.worker = e.data;
addEventListener('message', e => { if (e.data && e.data.frame) window.__r.iframe = e.data.info; });
</script><iframe src="%s/f"></iframe>`, androidSurfaceJS, frameURL)
		case "/worker.js":
			w.Header().Set("Content-Type", "text/javascript")
			fmt.Fprintf(w, `(%s)().then(i => postMessage(i))`, androidSurfaceJS)
		case "/popup":
			fmt.Fprintf(w, `<script>(%s)().then(i => window.__info = i)</script>`, androidSurfaceJS)
		default:
			fmt.Fprint(w, `<title>plain</title>`)
		}
	}))
	t.Cleanup(main.Close)
	t.Cleanup(frame.Close)
	return main, func(path string) http.Header { mu.Lock(); defer mu.Unlock(); return seen[path] }
}

// TestAndroidIdentityAcrossSurfaces 主页面、跨站 iframe、专用 Worker、弹窗与请求头都是同一台手机（Review Focus 3）
func TestAndroidIdentityAcrossSurfaces(t *testing.T) {
	srv, headers := newAndroidSurfaceServers(t)
	p := connectAndroid(t, "android-surfaces", "PGKM10").Page()
	if err := p.Navigate(srv.URL + "/"); err != nil {
		t.Fatal(err)
	}
	got := map[string]map[string]any{}
	for deadline := time.Now().Add(10 * time.Second); time.Now().Before(deadline); time.Sleep(100 * time.Millisecond) {
		v, _ := p.Evaluate(`JSON.stringify(window.__r)`)
		if json.Unmarshal([]byte(fmt.Sprint(v)), &got) == nil && got["main"] != nil && got["worker"] != nil && got["iframe"] != nil {
			break
		}
	}
	if _, err := p.Evaluate(`window.__p = window.open('/popup'), true`); err != nil {
		t.Fatal(err)
	}
	for deadline := time.Now().Add(10 * time.Second); time.Now().Before(deadline) && got["popup"] == nil; time.Sleep(100 * time.Millisecond) {
		if v, _ := p.Evaluate(`window.__p && window.__p.__info ? JSON.stringify(window.__p.__info) : ''`); v != "" {
			popup := map[string]any{}
			json.Unmarshal([]byte(fmt.Sprint(v)), &popup)
			got["popup"] = popup
		}
	}
	for _, surface := range []string{"main", "iframe", "worker", "popup"} {
		info := got[surface]
		if info == nil {
			t.Errorf("%s never reported", surface)
			continue
		}
		ua := fmt.Sprint(info["ua"])
		if !strings.HasPrefix(ua, "Mozilla/5.0 (Linux; Android 10; K) ") || !strings.HasSuffix(ua, " Mobile Safari/537.36") || ua != fmt.Sprint(got["main"]["ua"]) {
			t.Errorf("%s ua %q", surface, ua)
		}
		want := map[string]any{"platform": androidPlatform, "deviceMemory": 8.0, "cores": 8.0, "mobile": true, "uaPlatform": "Android",
			"model": "PGKM10", "platformVersion": "15.0.0", "formFactors": "Mobile"}
		if surface != "worker" {
			want["touch"], want["coarse"], want["screen"] = 5.0, true, "360,804,3"
		}
		for k, v := range want {
			if info[k] != v {
				t.Errorf("%s %s = %v, want %v", surface, k, info[k], v)
			}
		}
	}
	for _, path := range []string{"/", "/f"} {
		h := headers(path)
		if h.Get("Sec-Ch-Ua-Mobile") != "?1" || h.Get("Sec-Ch-Ua-Platform") != `"Android"` || !strings.Contains(h.Get("User-Agent"), "Android 10; K") {
			t.Errorf("%s request headers %v", path, h)
		}
	}
}

// TestAndroidLaunchAndViewport 插件原生为空、PDF 查看器关闭；视口与窗口外框一致；SetViewport 后仍是手机
func TestAndroidLaunchAndViewport(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { fmt.Fprint(w, `<title>plain</title>`) }))
	defer srv.Close()
	p := connectAndroid(t, "android-launch", "22041211AC").Page()
	const js = `[navigator.plugins.length, navigator.mimeTypes.length, navigator.pdfViewerEnabled, innerWidth, innerHeight, outerWidth, outerHeight,
	  screen.width, screen.height, devicePixelRatio, screen.orientation.type, matchMedia('(pointer: coarse)').matches].join()`
	if err := p.Navigate(srv.URL); err != nil {
		t.Fatal(err)
	}
	if got, _ := p.Evaluate(js); got != "0,0,false,411,810,411,810,411,914,3.5,portrait-primary,true" {
		t.Errorf("launch state %v", got)
	}
	if err := p.SetViewport(400, 780); err != nil {
		t.Fatal(err)
	}
	if err := p.Navigate(srv.URL + "/?after"); err != nil {
		t.Fatal(err)
	}
	if got, _ := p.Evaluate(js); got != "0,0,false,400,780,400,780,411,914,3.5,portrait-primary,true" {
		t.Errorf("after SetViewport %v", got)
	}
}
```

- [ ] **Step 2: 运行测试，确认失败**

Run: `go test -count=1 -run 'TestAndroidIdentityFromConfig|TestAndroidHeadlessScreenFlags|TestAndroidIdentityAcrossSurfaces|TestAndroidLaunchAndViewport' ./pkg/browser/`
Expected: FAIL：编译错误（`id.DeviceMemory`、`id.Battery`、`androidBattery` 未定义）。

- [ ] **Step 3: 实现**

(a) `pkg/browser/identity_android.go` 末尾追加：

```go
// portraitOrientation 手机竖屏（setDeviceMetricsOverride 的 screenOrientation）
var portraitOrientation = map[string]any{"type": "portraitPrimary", "angle": 0}

// androidBattery 账号固定的电量（0.35–0.95）：宿主是服务器没有电池，原生恒为满电在充，每个手机账号都一样
func androidBattery(userID string) float64 {
	return float64(35+identitySeed(userID, "battery")%61) / 100
}

// androidIdentity 安卓身份：机型档案提供屏幕、显卡、系统版本，版本与品牌取自真实浏览器
func androidIdentity(cfg *FingerprintConfig, host hostInfo) (*Identity, error) {
	dev, ok := lookupAndroidDevice(cfg.Device.Model)
	if !ok {
		return nil, fmt.Errorf("fingerprint %s: android model %q is not in the pool (config not normalized)", cfg.UserID, cfg.Device.Model)
	}
	if len(cfg.Browser.Languages) == 0 {
		return nil, fmt.Errorf("fingerprint %s: no languages", cfg.UserID)
	}
	gpu := dev.GPU
	return &Identity{
		OS:                  OSAndroid,
		UserAgent:           androidUserAgent(host.majorVersion()),
		Platform:            androidPlatform,
		AcceptLanguage:      strings.Join(cfg.Browser.Languages, ","),
		Languages:           cfg.Browser.Languages,
		Timezone:            cfg.Timezone.Timezone,
		HardwareConcurrency: androidCores,
		DeviceMemory:        androidDeviceMemory,
		Battery:             androidBattery(cfg.UserID),
		Metadata: uaMetadata{
			Brands: host.Brands, FullVersionList: host.FullVersionList, Platform: "Android", PlatformVersion: dev.PlatformVersion,
			Model: dev.Model, Mobile: true, FormFactors: []string{"Mobile"},
		},
		Screen:    androidGeometry(dev.Screen),
		GPU:       &gpu,
		NoiseSeed: noiseSeed(cfg.UserID),
		ScriptKey: newScriptKey(),
	}, nil
}
```

  并给该文件加 import `"fmt"`、`"strings"`。

(b) `pkg/browser/identity.go`：
- `Identity` 在 `HardwareConcurrency` 之后加：

```go
	DeviceMemory        int     // navigator.deviceMemory；0 表示不改写（桌面身份沿用原生值）
	Battery             float64 // 电池电量（安卓身份）；0 表示不改写
```

- `identityFromConfig` 开头加：

```go
	if configOS(cfg) == OSAndroid {
		return androidIdentity(cfg, host)
	}
```

- `configGeometry` 替换为：

```go
// configGeometry 账号指纹对应的屏幕与窗口几何（身份与无界面启动参数共用，保证两边一致）
func configGeometry(cfg *FingerprintConfig) screenGeometry {
	os := configOS(cfg)
	if os == OSAndroid {
		return androidGeometry(screenSpec{Width: cfg.Screen.Width, Height: cfg.Screen.Height, DPR: cfg.Screen.DevicePixelRatio})
	}
	spec, _ := screenSpecFor(os, cfg.Screen.Width, cfg.Screen.Height, cfg.Screen.DevicePixelRatio)
	return geometryFor(os, spec, cfg.Browser.PlatformVersion)
}
```

(c) `pkg/browser/identity_apply.go`：
- `applyIdentity` 中 `setDeviceMetricsOverride` 那一段替换为：

```go
	// iframe 不支持 setDeviceMetricsOverride（只能用于顶层目标），其屏幕与 DPR 由注入脚本补
	if kind == kindPage && id.Screen.Width > 0 {
		s := id.Screen
		metrics := map[string]any{
			"width": s.InnerWidth, "height": s.InnerHeight, "deviceScaleFactor": s.DPR, "mobile": s.Mobile,
			"screenWidth": s.Width, "screenHeight": s.Height,
		}
		if s.Mobile {
			metrics["screenOrientation"] = portraitOrientation
		}
		steps = append(steps, cdpStep{"Emulation.setDeviceMetricsOverride", metrics})
	}
	// 手机：页面与 iframe 都开触屏模拟（maxTouchPoints、ontouchstart、pointer: coarse 按目标生效）
	if id.Screen.Mobile {
		steps = append(steps, cdpStep{"Emulation.setTouchEmulationEnabled", map[string]any{"enabled": true, "maxTouchPoints": androidTouchPoints}})
	}
```

- `viewportOverride` 替换为：

```go
// viewportOverride 改视口用的 Emulation.setDeviceMetricsOverride 参数：带上身份的屏幕尺寸与 DPR，
// 否则覆盖会把屏幕退回无界面默认值；身份不调整几何时不覆盖 DPR；手机身份保持移动端模拟与竖屏
func viewportOverride(id *Identity, width, height int) map[string]any {
	p := map[string]any{"width": width, "height": height, "deviceScaleFactor": 0, "mobile": false}
	if id != nil && id.Screen.Width > 0 {
		p["deviceScaleFactor"], p["screenWidth"], p["screenHeight"] = id.Screen.DPR, id.Screen.Width, id.Screen.Height
		if id.Screen.Mobile {
			p["mobile"], p["screenOrientation"] = true, portraitOrientation
		}
	}
	return p
}
```

(d) `pkg/browser/target_manager.go`：`onAttached` 里 `if err == nil && kind == kindPage && tm.identity.Screen.Width > 0 {` 改为 `if err == nil && kind == kindPage && tm.identity.Screen.Width > 0 && !tm.identity.Screen.Mobile {`，并在上一行加注释 `// 手机身份的窗口外框由移动端模拟决定（等于视口），不调整窗口`。

(e) `pkg/browser/cdp_custom.go`：`SetViewport` 中 `if p.identity == nil || p.identity.Screen.Width == 0 || p.targetID == "" {` 改为 `if p.identity == nil || p.identity.Screen.Width == 0 || p.identity.Screen.Mobile || p.targetID == "" {`，注释改为 `// 身份不调整几何（有界面且不设指纹）时不动用户可见的窗口；手机身份的窗口外框由移动端模拟决定`。

(f) `pkg/browser/launcher.go`：
- `Launch` 开头加载指纹的那段替换为：

```go
	// 设了账号指纹时先加载：无界面时让浏览器原生报告账号屏幕（见 headlessScreenFlags），
	// 手机身份还要改 profile 偏好；指纹文件损坏在这里就报错
	var screenFlags []string
	var fingerprint *FingerprintConfig
	if opts.FingerprintUserID != "" {
		cfg, err := fingerprintForOptions(opts)
		if err != nil {
			return nil, err
		}
		fingerprint = cfg
		if len(config.GetHeadlessFlags(opts.Headless)) > 0 {
			screenFlags = headlessScreenFlags(configGeometry(cfg))
		}
	}
	mobile := fingerprint != nil && configOS(fingerprint) == OSAndroid
```

- 写 WebRTC 偏好的那段（`if opts.Proxy != nil { if err := setWebRTCPolicy(...` ）之后加：

```go
	if mobile {
		if err := disablePDFViewer(userDataDir); err != nil {
			if tempDir != "" {
				os.RemoveAll(tempDir)
			}
			return nil, err
		}
	}
```

- `setWebRTCPolicy` 整个替换为下面三个函数：

```go
// setWebRTCPolicy 配置了代理时写进 profile 的 WebRTC 设置：禁止绕过代理直连 UDP，否则 STUN 会暴露真实公网 IP。
// 与 Chrome 设置里的 webrtc.ip_handling_policy 是同一项；--force-webrtc-ip-handling-policy 启动参数在 Chrome 154 上不生效
func setWebRTCPolicy(userDataDir string) error {
	return updatePreferences(userDataDir, func(prefs map[string]any) {
		webrtc, _ := prefs["webrtc"].(map[string]any)
		if webrtc == nil {
			webrtc = map[string]any{}
		}
		webrtc["ip_handling_policy"] = "disable_non_proxied_udp"
		webrtc["multiple_routes_enabled"] = false
		webrtc["nonproxied_udp_enabled"] = false
		prefs["webrtc"] = webrtc
	})
}

// disablePDFViewer 手机身份：关掉内置 PDF 查看器（与 Chrome 设置"下载 PDF 文件"是同一项）。
// 安卓 Chrome 没有 PDF 插件；这样 navigator.plugins / mimeTypes 原生为空、pdfViewerEnabled 原生为 false（Chrome 155 实测）
func disablePDFViewer(userDataDir string) error {
	return updatePreferences(userDataDir, func(prefs map[string]any) {
		plugins, _ := prefs["plugins"].(map[string]any)
		if plugins == nil {
			plugins = map[string]any{}
		}
		plugins["always_open_pdf_externally"] = true
		prefs["plugins"] = plugins
	})
}

// updatePreferences 修改 profile 的 Preferences：已有的设置（持久化 profile）原样保留，只改 edit 动到的部分
func updatePreferences(userDataDir string, edit func(map[string]any)) error {
	path := filepath.Join(userDataDir, "Default", "Preferences")
	prefs := map[string]any{}
	data, err := os.ReadFile(path)
	switch {
	case err == nil:
		dec := json.NewDecoder(bytes.NewReader(data))
		dec.UseNumber() // Preferences 里有超出 float64 精度的整数
		if err := dec.Decode(&prefs); err != nil {
			return fmt.Errorf("parse %s: %w", path, err)
		}
	case !errors.Is(err, fs.ErrNotExist):
		return fmt.Errorf("read %s: %w", path, err)
	}
	edit(prefs)
	out, err := json.Marshal(prefs)
	if err != nil {
		return fmt.Errorf("encode %s: %w", path, err)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("create %s: %w", filepath.Dir(path), err)
	}
	if err := os.WriteFile(path, out, 0o644); err != nil {
		return fmt.Errorf("write %s: %w", path, err)
	}
	return nil
}
```

- [ ] **Step 4: 运行测试，确认通过**

Run: `go vet ./pkg/browser/ && go test -count=1 -run 'TestAndroid|TestIdentity|TestSetViewport|TestWebRTC|TestCloseShutsDownGracefully' ./pkg/browser/`
Expected: PASS

如果 `Emulation.setTouchEmulationEnabled` 在 iframe 会话上报错：iframe 目标改为不发这条命令，并在 ledger 记 Ruling。iframe 的 `maxTouchPoints`、`pointer` 会不会跟随顶层页面，以测试结果为准。

- [ ] **Step 5: 提交**

```bash
git add pkg/browser/
git commit -m "feat(identity): apply Android identities over CDP and at launch

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
```

---

### Task 3: 安卓的 JS 接口

**Files:**
- Modify: `pkg/browser/identity_script.go`
- Test: `pkg/browser/android_integration_test.go`

**Interfaces:**
- Consumes: Task 2 的 `Identity.DeviceMemory`、`Identity.Battery`、`connectAndroid`、`awaitJS`。
- Produces: `jsAndroid{DeviceMemory int; Battery float64}`，挂在 `jsIdentity.Android`。

- [ ] **Step 1: 写失败的测试**

在 `pkg/browser/android_integration_test.go` 末尾追加：

```go
// androidAPIsJS 安卓与桌面不同的 JS 接口
const androidAPIsJS = `(async () => {
  let notif; try { new Notification('x'); notif = 'constructs'; } catch (e) { notif = e.name + ': ' + e.message; }
  let callNotif; try { Notification('x'); callNotif = 'ok'; } catch (e) { callNotif = e.message; }
  const b = await navigator.getBattery();
  const worker = await new Promise(r => {
    const w = new Worker(URL.createObjectURL(new Blob(['postMessage([navigator.deviceMemory, navigator.platform])'], {type: 'text/javascript'})));
    w.onmessage = e => r(e.data);
  });
  const out = {
    deviceMemory: navigator.deviceMemory, worker, notif, callNotif,
    notifPermission: typeof Notification.permission, notifSource: Function.prototype.toString.call(Notification),
    notifProtoCtor: Notification.prototype.constructor === Notification,
    hasContacts: 'contacts' in navigator, connType: navigator.connection.type,
    removed: ['hid', 'windowControlsOverlay'].filter(k => k in navigator)
      .concat(['EyeDropper', 'SharedWorker', 'documentPictureInPicture', 'queryLocalFonts', 'HID', 'FontData'].filter(k => k in window)),
    chromeApp: 'app' in window.chrome, battery: [b.charging, b.chargingTime, b.dischargingTime, b.level],
    share: [typeof navigator.share, typeof navigator.canShare], barcode: typeof BarcodeDetector,
  };
  if (out.hasContacts) {
    out.contacts = String(navigator.contacts);
    out.contactsProps = (await navigator.contacts.getProperties()).join();
    try { await navigator.contacts.select(['name']); out.contactsSelect = 'resolved'; } catch (e) { out.contactsSelect = e.name; }
    out.contactsManager = typeof ContactsManager;
  }
  if (typeof BarcodeDetector === 'function') out.barcodeFormats = (await BarcodeDetector.getSupportedFormats()).length > 0;
  return JSON.stringify(out);
})()`

// TestAndroidSurfaceAPIs 安卓身份的 JS 接口与真机一致；桌面身份不受影响（Review Focus 5）
func TestAndroidSurfaceAPIs(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { fmt.Fprint(w, `<title>plain</title>`) }))
	defer srv.Close()
	run := func(t *testing.T, p Page) map[string]any {
		if err := p.Navigate(srv.URL); err != nil {
			t.Fatal(err)
		}
		out := map[string]any{}
		if err := json.Unmarshal([]byte(awaitJS(t, p, androidAPIsJS)), &out); err != nil {
			t.Fatal(err)
		}
		return out
	}
	t.Run("android", func(t *testing.T) {
		got := run(t, connectAndroid(t, "android-apis", "PGKM10").Page())
		want := map[string]any{
			"deviceMemory": 8.0, "notif": "TypeError: Failed to construct 'Notification': Illegal constructor. Use ServiceWorkerRegistration.showNotification() instead.",
			"notifPermission": "string", "notifSource": "function Notification() { [native code] }", "notifProtoCtor": true,
			"hasContacts": true, "contacts": "[object ContactsManager]", "contactsProps": "address,email,icon,name,tel",
			"contactsSelect": "SecurityError", "contactsManager": "function", "connType": "wifi", "chromeApp": false,
			"barcode": "function", "barcodeFormats": true,
		}
		for k, v := range want {
			if got[k] != v {
				t.Errorf("%s = %v, want %v", k, got[k], v)
			}
		}
		if w := fmt.Sprint(got["worker"]); w != "[8 Linux armv81]" {
			t.Errorf("worker = %s", w)
		}
		if !strings.Contains(fmt.Sprint(got["callNotif"]), "'new' operator") {
			t.Errorf("Notification() without new = %v", got["callNotif"])
		}
		if r := got["removed"].([]any); len(r) != 0 {
			t.Errorf("desktop-only APIs still present: %v", r)
		}
		b := got["battery"].([]any)
		if b[0] != false || b[1] != nil || b[2] != nil || b[3].(float64) < 0.35 || b[3].(float64) > 0.95 {
			t.Errorf("battery = %v, want discharging with a 0.35–0.95 level (Infinity encodes as null)", b)
		}
		if s := fmt.Sprint(got["share"]); s != "[function function]" {
			t.Errorf("share = %s", s)
		}
	})
	t.Run("desktop", func(t *testing.T) {
		inst, err := Connect(t.Context(), &ConnectOptions{Headless: true, FingerprintUserID: "desktop-apis", FingerprintDir: t.TempDir()})
		if err != nil {
			t.Fatal(err)
		}
		defer inst.Close()
		got := run(t, inst.Page())
		if got["notif"] != "constructs" || len(got["removed"].([]any)) == 0 || got["chromeApp"] != true {
			t.Errorf("desktop identity changed: notif %v, removed %v, chrome.app %v", got["notif"], got["removed"], got["chromeApp"])
		}
	})
}
```

- [ ] **Step 2: 运行测试，确认失败**

Run: `go test -count=1 -run TestAndroidSurfaceAPIs ./pkg/browser/`
Expected: FAIL：`android` 子测试报 `deviceMemory`、`notif`、`hasContacts`、`connType`、`removed` 等不符；`desktop` 子测试通过。

- [ ] **Step 3: 实现**

`pkg/browser/identity_script.go`：

(a) `jsIdentity` 在 `Key` 之前加 `Android *jsAndroid `json:"android,omitempty"``，并新增：

```go
// jsAndroid 安卓身份专有：内存与电量
type jsAndroid struct {
	DeviceMemory int     `json:"deviceMemory"`
	Battery      float64 `json:"battery"`
}
```

(b) `identityScript` 在 `if id.GPU != nil {` 之前加：

```go
	if id.OS == OSAndroid {
		cfg.Android = &jsAndroid{DeviceMemory: id.DeviceMemory, Battery: id.Battery}
	}
```

(c) `jsCommonPart` 末尾（反引号之前）追加：

```js
if (C.android) {
  // 内存（页面与 Worker）、网络类型：安卓 Chrome 报告连接类型，桌面没有这个属性
  for (const N of [G.Navigator, G.WorkerNavigator]) if (N) patchGetter(N.prototype, 'deviceMemory', () => C.android.deviceMemory);
  if (G.NetworkInformation && !('type' in NetworkInformation.prototype)) {
    const NI = NetworkInformation.prototype;
    const check = Object.getOwnPropertyDescriptor(NI, 'effectiveType').get;
    const get = Object.getOwnPropertyDescriptor({get type() { Reflect.apply(check, this, []); return 'wifi'; }}, 'type').get;
    masks.set(get, 'function get type() { [native code] }');
    Object.defineProperty(NI, 'type', {get, enumerable: true, configurable: true});
  }
}
```

(d) `jsWindowPart` 中 Web Share 的条件 `/^(Win32|MacIntel)$/.test(C.platform)` 改为 `/^(Win32|MacIntel|Linux armv81)$/.test(C.platform)`，注释改为 `// Linux 版 Chrome 没有 Web Share，Windows / Mac / 安卓版有（CreepJS 的 noWebShare）：按真实 Chrome 的行为补上`。

(e) `jsWindowPart` 末尾（反引号之前）追加：

```js
if (C.android) {
  const A = C.android;
  // 桌面独有的接口：安卓 Chrome 没有
  for (const k of ['hid', 'windowControlsOverlay']) delete Navigator.prototype[k];
  for (const k of ['EyeDropper', 'SharedWorker', 'documentPictureInPicture', 'queryLocalFonts', 'HID', 'HIDDevice', 'HIDConnectionEvent',
    'HIDInputReportEvent', 'WindowControlsOverlay', 'WindowControlsOverlayGeometryChangeEvent', 'DocumentPictureInPicture',
    'DocumentPictureInPictureEvent', 'FontData']) delete G[k];
  if (G.chrome) delete G.chrome.app;
  // Notification：安卓只能经 ServiceWorkerRegistration.showNotification 发通知，new 时抛错；静态成员与原型保持原生
  if (G.Notification) {
    const Orig = G.Notification;
    const Fake = function Notification(title) {
      if (new.target === undefined) return Reflect.apply(Orig, this, arguments);
      throw new TypeError("Failed to construct 'Notification': Illegal constructor. Use ServiceWorkerRegistration.showNotification() instead.");
    };
    for (const [k, d] of Object.entries(Object.getOwnPropertyDescriptors(Orig))) {
      if (k !== 'length' && k !== 'name' && k !== 'prototype') Object.defineProperty(Fake, k, d);
    }
    Object.defineProperty(Fake, 'prototype', {value: Orig.prototype, writable: false});
    Object.defineProperty(Orig.prototype, 'constructor', {value: Fake, writable: true, enumerable: false, configurable: true});
    Object.setPrototypeOf(Fake, Object.getPrototypeOf(Orig));
    disguise(Fake, Orig);
    Object.defineProperty(G, 'Notification', {value: Fake, writable: true, enumerable: false, configurable: true});
  }
  // 通讯录选择器（Contact Picker）：只在安全上下文、只有安卓有
  if (G.isSecureContext && !('contacts' in Navigator.prototype)) {
    const N = Navigator.prototype;
    const receiverCheck = Object.getOwnPropertyDescriptor(N, 'userAgent').get;
    const CM = function ContactsManager() { throw new TypeError('Illegal constructor'); };
    masks.set(CM, 'function ContactsManager() { [native code] }');
    const proto = CM.prototype;
    const isManager = (self) => { if (!(self instanceof CM)) throw new TypeError('Illegal invocation'); };
    const addProtoMethod = (name, fn) => {
      masks.set(fn, 'function ' + name + '() { [native code] }');
      Object.defineProperty(proto, name, {value: fn, writable: true, enumerable: true, configurable: true});
    };
    addProtoMethod('getProperties', {getProperties() {
      try { isManager(this); } catch (e) { return Promise.reject(e); }
      return Promise.resolve(['address', 'email', 'icon', 'name', 'tel']);
    }}.getProperties);
    // 没有用户手势时与真机一样拒绝；有手势时当作用户关掉了选择框（真机返回空列表）
    addProtoMethod('select', {select(properties) {
      try { isManager(this); } catch (e) { return Promise.reject(e); }
      const activation = G.navigator.userActivation;
      if (!(activation && activation.isActive)) {
        return Promise.reject(new DOMException("Failed to execute 'select' on 'ContactsManager': A user gesture is required to call this method.", 'SecurityError'));
      }
      return new Promise((resolve) => setTimeout(() => resolve([]), 800));
    }}.select);
    Object.defineProperty(proto, Symbol.toStringTag, {value: 'ContactsManager', configurable: true});
    const manager = Object.create(proto);
    const getter = Object.getOwnPropertyDescriptor({get contacts() { Reflect.apply(receiverCheck, this, []); return manager; }}, 'contacts').get;
    masks.set(getter, 'function get contacts() { [native code] }');
    Object.defineProperty(N, 'contacts', {get: getter, enumerable: true, configurable: true});
    Object.defineProperty(G, 'ContactsManager', {value: CM, writable: true, enumerable: false, configurable: true});
  }
  // 条码识别：安卓与 Mac 版 Chrome 有，Linux 版没有。宿主没有时补上：支持安卓的格式，识别结果为空（看不到条码）
  if (!G.BarcodeDetector && G.isSecureContext) {
    const formats = ['aztec', 'code_128', 'code_39', 'code_93', 'codabar', 'data_matrix', 'ean_13', 'ean_8', 'itf', 'pdf417', 'qr_code', 'upc_a', 'upc_e'];
    const BD = function BarcodeDetector(options) {
      if (new.target === undefined) throw new TypeError("Failed to construct 'BarcodeDetector': Please use the 'new' operator, this DOM object constructor cannot be called as a function.");
      const wanted = options && options.formats;
      if (wanted !== undefined && (!wanted.length || wanted.some(f => !formats.includes(f)))) {
        throw new TypeError("Failed to construct 'BarcodeDetector': Hint option provided, but is empty or contains an unknown format.");
      }
    };
    masks.set(BD, 'function BarcodeDetector() { [native code] }');
    const getSupportedFormats = {getSupportedFormats() { return Promise.resolve(formats.slice()); }}.getSupportedFormats;
    masks.set(getSupportedFormats, 'function getSupportedFormats() { [native code] }');
    Object.defineProperty(BD, 'getSupportedFormats', {value: getSupportedFormats, writable: true, enumerable: true, configurable: true});
    const detect = {detect(image) {
      if (!(this instanceof BD)) return Promise.reject(new TypeError('Illegal invocation'));
      return Promise.resolve([]);
    }}.detect;
    masks.set(detect, 'function detect() { [native code] }');
    Object.defineProperty(BD.prototype, 'detect', {value: detect, writable: true, enumerable: true, configurable: true});
    Object.defineProperty(BD.prototype, Symbol.toStringTag, {value: 'BarcodeDetector', configurable: true});
    Object.defineProperty(G, 'BarcodeDetector', {value: BD, writable: true, enumerable: false, configurable: true});
  }
  // 电池：放电中、电量按账号固定；时间未知时为 Infinity（与真机一致）
  if (G.BatteryManager) {
    const B = BatteryManager.prototype;
    patchGetter(B, 'charging', () => false);
    patchGetter(B, 'chargingTime', () => Infinity);
    patchGetter(B, 'dischargingTime', () => Infinity);
    patchGetter(B, 'level', () => A.battery);
  }
}
```

- [ ] **Step 4: 运行测试，确认通过**

Run: `go test -count=1 -run 'TestAndroidSurfaceAPIs|TestIdentity|TestScript' ./pkg/browser/`
Expected: PASS

- [ ] **Step 5: 提交**

```bash
git add pkg/browser/identity_script.go pkg/browser/android_integration_test.go
git commit -m "feat(identity): Android-only JS surface (Notification, contacts, connection type, battery, desktop-only APIs)

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
```

---

### Task 4: 安卓的 WebGL 表面

**Files:**
- Modify: `pkg/browser/identity_script.go`
- Test: `pkg/browser/android_integration_test.go`

**Interfaces:**
- Consumes: Task 1 的 `gpuProfile.Mobile`；Task 2 的 `connectAndroid`；M0 的 `baselineServer`、`openBaseline` 在 `baseline_collector_test.go` 里，需要再加一个可传入实例的变体（见 Step 1）。
- Produces:
  - `jsWebGL.Params2`、`Extensions`、`Extensions2`、`Precision`；
  - `glParamTable(map[int]any) map[string]any`；
  - 测试辅助 `loadAndroidBaseline(t) map[string]any`、`collectOn(t, p Page, srv *httptest.Server) map[string]any`。

- [ ] **Step 1: 写失败的测试**

在 `pkg/browser/android_integration_test.go` 末尾追加：

```go
// loadAndroidBaseline 读取安卓真机基准的 auto 部分
func loadAndroidBaseline(t *testing.T) map[string]any {
	t.Helper()
	raw, err := os.ReadFile("../../test/baseline/android-chrome-pgkm10.json")
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

// collectOn 在给定页面上打开采集页（见 baseline_collector_test.go），返回 auto 结果
func collectOn(t *testing.T, p Page, srv *httptest.Server) map[string]any {
	t.Helper()
	if err := p.Navigate(srv.URL); err != nil {
		t.Fatal(err)
	}
	waitBaselineState(t, p, "ready")
	raw, err := p.Evaluate(`JSON.stringify(window.__baselineAuto)`)
	if err != nil {
		t.Fatal(err)
	}
	var auto map[string]any
	if err := json.Unmarshal([]byte(fmt.Sprint(raw)), &auto); err != nil {
		t.Fatal(err)
	}
	return auto
}

// TestAndroidWebGL WebGL1 / WebGL2 的标识、能力参数、精度、扩展列表与真机一致；手机上没有的扩展取不到，也不会被悄悄启用（Review Focus 4）
func TestAndroidWebGL(t *testing.T) {
	srv, _ := baselineServer(t, "", http.StatusOK)
	p := connectAndroid(t, "android-webgl", "PGKM10").Page()
	got, want := collectOn(t, p, srv), loadAndroidBaseline(t)
	for _, kind := range []string{"webgl", "webgl2"} {
		g, w := got[kind].(map[string]any), want[kind].(map[string]any)
		for _, k := range []string{"vendor", "renderer", "unmaskedVendor", "unmaskedRenderer", "version", "shadingLanguageVersion", "precision", "extensions"} {
			if fmt.Sprint(g[k]) != fmt.Sprint(w[k]) {
				t.Errorf("%s.%s = %v, want %v", kind, k, g[k], w[k])
			}
		}
		gp, wp := g["params"].(map[string]any), w["params"].(map[string]any)
		for k, v := range wp {
			if fmt.Sprint(gp[k]) != fmt.Sprint(v) {
				t.Errorf("%s.params.%s = %v, want %v", kind, k, gp[k], v)
			}
		}
	}
	// 手机上没有的扩展：取不到，枚举值也不可用
	probe := `(() => { const gl = document.createElement('canvas').getContext('webgl');
	  const s3tc = gl.getExtension('WEBGL_compressed_texture_s3tc'), float = gl.getExtension('OES_texture_float'), aniso = gl.getExtension('EXT_texture_filter_anisotropic');
	  gl.bindTexture(gl.TEXTURE_2D, gl.createTexture());
	  gl.texImage2D(gl.TEXTURE_2D, 0, gl.RGBA, 1, 1, 0, gl.RGBA, gl.FLOAT, null);
	  return [s3tc === null, float === null, aniso !== null, gl.getError() !== 0].join(); })()`
	if v, _ := p.Evaluate(probe); v != "true,true,true,true" {
		t.Errorf("extension probe = %v, want true,true,true,true", v)
	}
}
```

- [ ] **Step 2: 运行测试，确认失败**

Run: `go test -count=1 -run TestAndroidWebGL ./pkg/browser/`
Expected: FAIL，`params`、`precision`、`extensions` 不符，`extension probe` 不符。

- [ ] **Step 3: 实现**

`pkg/browser/identity_script.go`：

(a) `jsWebGL` 替换为：

```go
// jsWebGL 显卡标识与 getParameter 覆盖表（键为十进制 GLenum，值为数值或 [a, b]）；手机 GPU 另有只属于 WebGL2 的参数、扩展列表与着色器精度
type jsWebGL struct {
	Vendor      string            `json:"vendor"`
	Renderer    string            `json:"renderer"`
	Params      map[string]any    `json:"params,omitempty"`
	Params2     map[string]any    `json:"params2,omitempty"`
	Extensions  []string          `json:"extensions,omitempty"`
	Extensions2 []string          `json:"extensions2,omitempty"`
	Precision   map[string][3]int `json:"precision,omitempty"` // "着色器.精度类型" → [rangeMin, rangeMax, precision]
}

// glParamTable GLenum 为键的参数表转为脚本用的十进制字符串键
func glParamTable(m map[int]any) map[string]any {
	if len(m) == 0 {
		return nil
	}
	out := make(map[string]any, len(m))
	for k, v := range m {
		out[strconv.Itoa(k)] = v
	}
	return out
}
```

  并在 import 中加 `"strconv"`。

(b) `identityScript` 中 `if id.GPU != nil { cfg.WebGL = ... }` 替换为：

```go
	if id.GPU != nil {
		w := &jsWebGL{Vendor: id.GPU.Vendor, Renderer: id.GPU.Renderer, Params: webglParams(id.GPU.Caps)}
		if m := id.GPU.Mobile; m != nil {
			w.Params, w.Params2 = glParamTable(m.Params), glParamTable(m.Params2)
			w.Extensions, w.Extensions2 = m.Extensions, m.Extensions2
			w.Precision = make(map[string][3]int, len(m.Precision))
			for k, v := range m.Precision {
				w.Precision[strconv.Itoa(k[0])+"."+strconv.Itoa(k[1])] = v
			}
		}
		cfg.WebGL = w
	}
```

(c) `jsCommonPart` 中 `if (C.webgl) { ... }` 整块替换为：

```js
if (C.webgl) {
  const W = C.webgl, P = W.params || {}, P2 = W.params2 || {};
  const GL2 = G.WebGL2RenderingContext;
  const has = (o, k) => Object.prototype.hasOwnProperty.call(o, k);
  for (const Ctx of [G.WebGLRenderingContext, GL2]) {
    if (!Ctx) continue;
    const is2 = Ctx === GL2;
    patchMethod(Ctx.prototype, 'getParameter', (orig, self, args) => {
      const value = Reflect.apply(orig, self, args);
      const p = args[0];
      if (p === 37445) return value === null ? value : W.vendor;
      if (p === 37446) return value === null ? value : W.renderer;
      const table = is2 && has(P2, p) ? P2 : P;
      if (has(table, p)) {
        const v = table[p];
        return Array.isArray(v) ? (p === 3386 ? new Int32Array(v) : new Float32Array(v)) : v;
      }
      return value;
    });
    // 手机 GPU：扩展列表按真机；不在列表里的扩展取不到，也不调用原生（否则会在上下文里悄悄启用）
    const exts = is2 ? W.extensions2 : W.extensions;
    if (exts) {
      const allowed = new Set(exts.map(e => e.toLowerCase()));
      patchMethod(Ctx.prototype, 'getSupportedExtensions', (orig, self, args) => {
        const list = Reflect.apply(orig, self, args);
        return list === null ? null : exts.slice();
      });
      patchMethod(Ctx.prototype, 'getExtension', (orig, self, args) => {
        if (!(self instanceof Ctx) || !args.length || allowed.has(String(args[0]).toLowerCase())) return Reflect.apply(orig, self, args);
        return null;
      });
    }
    if (C.seed) {
      patchMethod(Ctx.prototype, 'readPixels', (orig, self, args) => {
        const out = Reflect.apply(orig, self, args);
        const px = args[6];
        if (px && px.BYTES_PER_ELEMENT === 1 && px.length) noisePixels(px, args[2], args[0], args[1]);
        return out;
      });
    }
  }
  // 手机 GPU 的着色器精度：按对象替换 WebGLShaderPrecisionFormat 的三个取值
  if (W.precision && G.WebGLShaderPrecisionFormat) {
    const over = new WeakMap();
    for (const Ctx of [G.WebGLRenderingContext, GL2]) {
      if (!Ctx) continue;
      patchMethod(Ctx.prototype, 'getShaderPrecisionFormat', (orig, self, args) => {
        const f = Reflect.apply(orig, self, args);
        const v = f && W.precision[args[0] + '.' + args[1]];
        if (v) over.set(f, v);
        return f;
      });
    }
    ['rangeMin', 'rangeMax', 'precision'].forEach((k, i) => {
      patchGetter(WebGLShaderPrecisionFormat.prototype, k, (orig, self) => over.has(self) ? over.get(self)[i] : Reflect.apply(orig, self, []));
    });
  }
}
```

- [ ] **Step 4: 运行测试，确认通过**

Run: `go test -count=1 -run 'TestAndroidWebGL|TestIdentity|TestScript|TestFingerprint' ./pkg/browser/`
Expected: PASS（桌面身份的 WebGL 测试照常通过）

- [ ] **Step 5: 提交**

```bash
git add pkg/browser/identity_script.go pkg/browser/android_integration_test.go
git commit -m "feat(identity): Android WebGL surface from the real Mali-G610 baseline

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
```

---

### Task 5: 安卓字体（Linux 进程级 fontconfig）

**Files:**
- Create: `pkg/browser/fonts_android.go`
- Modify:
  - `pkg/browser/launcher.go`：`Launch` 设置 `FONTCONFIG_FILE`，`Kill` 清理；
  - `pkg/browser/types.go`：`ChromeProcess.fontConfigDir`；
  - `test/linux-chrome.Dockerfile`、`docs/deploy/production.Dockerfile`、`docs/deploy/fontconfig/55-windows-fonts-only.conf`、`docs/deploy/README.md`。
- Test: `pkg/browser/android_identity_test.go`（单元）、`pkg/browser/android_integration_test.go`（Linux 集成）

**Interfaces:**
- Consumes: Task 2 中 `Launch` 的 `mobile` 变量。
- Produces: `writeAndroidFontConfig(dir string, roots []string) (confPath string, missing []string, err error)`、`androidFontRoles`、`androidFontAliases`。

- [ ] **Step 1: 写失败的测试**

在 `pkg/browser/android_identity_test.go` 末尾追加：

```go
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
```

在 `pkg/browser/android_integration_test.go` 末尾追加：

```go
// TestAndroidFonts Linux 上手机身份能按名字认出的字体与真机一致（只有安卓别名）；桌面身份仍用系统字体
func TestAndroidFonts(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("字体只在 Linux 上由本库控制（macOS 由 CoreText 管理）")
	}
	srv, _ := baselineServer(t, "", http.StatusOK)
	got := collectOn(t, connectAndroid(t, "android-fonts", "PGKM10").Page(), srv)["fonts"].(map[string]any)
	want := loadAndroidBaseline(t)["fonts"].(map[string]any)
	for name, v := range want {
		if got[name] != v {
			t.Errorf("font %q detected = %v, want %v", name, got[name], v)
		}
	}
	inst, err := Connect(t.Context(), &ConnectOptions{Headless: true})
	if err != nil {
		t.Fatal(err)
	}
	defer inst.Close()
	desktop := collectOn(t, inst.Page(), srv)["fonts"].(map[string]any)
	if desktop["Noto Sans CJK SC"] != true {
		t.Error("desktop identity lost the system fonts")
	}
}
```

  并在该文件 import 中加 `"runtime"`。

- [ ] **Step 2: 运行测试，确认失败**

Run: `go test -count=1 -run TestAndroidFontConfig ./pkg/browser/ ; scripts/test-linux-chrome.sh -run TestAndroidFonts`
Expected: FAIL：单元测试编译错误（`undefined: writeAndroidFontConfig`）；Linux 集成测试同样编译失败。

- [ ] **Step 3: 实现**

(a) 创建 `pkg/browser/fonts_android.go`：

```go
package browser

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// androidFontRole 一类安卓字体。配置里统一改成内部族名，只能经由通用族与别名访问：
// 安卓上 Roboto、Noto 这些名字不能按名访问（真机基准里一个都检测不到）
type androidFontRole struct {
	Name    string         // 改名后的族名，也是链接文件名的前缀
	Pattern *regexp.Regexp // 匹配字体文件名
}

// androidFontRoles 安卓 fonts.xml 的通用族（Debian 包里的对应字体）：sans-serif = Roboto，serif = Noto Serif，
// monospace = Noto Sans Mono（安卓用 Droid Sans Mono，Debian 不提供），中文与表情回退
var androidFontRoles = []androidFontRole{
	{"android-sans", regexp.MustCompile(`^Roboto-[A-Za-z]+\.ttf$`)},
	{"android-serif", regexp.MustCompile(`^NotoSerif-[A-Za-z]+\.ttf$`)},
	{"android-mono", regexp.MustCompile(`^Noto(Sans)?Mono-[A-Za-z]+\.ttf$`)},
	{"android-cjk", regexp.MustCompile(`^NotoSansCJK-[A-Za-z]+\.ttc$`)},
	{"android-emoji", regexp.MustCompile(`^NotoColorEmoji\.ttf$`)},
}

// androidFontAliases 通用族与安卓 fonts.xml 里的别名（真机基准检测到的 9 个，加上 times）→ 字体类
var androidFontAliases = []struct{ Family, Role string }{
	{"sans-serif", "android-sans"}, {"serif", "android-serif"}, {"monospace", "android-mono"},
	{"Arial", "android-sans"}, {"Helvetica", "android-sans"}, {"Tahoma", "android-sans"}, {"Verdana", "android-sans"},
	{"Times", "android-serif"}, {"Times New Roman", "android-serif"}, {"Georgia", "android-serif"},
	{"Courier", "android-mono"}, {"Courier New", "android-mono"}, {"Monaco", "android-mono"},
}

// androidFontRoots 生产与测试镜像里字体所在的目录
var androidFontRoots = []string{"/usr/share/fonts", "/usr/local/share/fonts"}

// writeAndroidFontConfig 在 dir 下生成手机身份的 fontconfig 配置：把 roots 里各类安卓字体的文件链接进 dir/fonts（文件名加类前缀），
// 扫描时按前缀改族名，再配通用族与别名；不包含系统配置，其他字体对该进程不可见。返回配置路径与缺失的字体类
func writeAndroidFontConfig(dir string, roots []string) (string, []string, error) {
	fontsDir := filepath.Join(dir, "fonts")
	if err := os.MkdirAll(fontsDir, 0o755); err != nil {
		return "", nil, fmt.Errorf("create %s: %w", fontsDir, err)
	}
	found := map[string]bool{}
	for _, root := range roots {
		filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
			if err != nil || d.IsDir() {
				return nil // 目录不存在或读不了：跳过
			}
			for _, r := range androidFontRoles {
				if !r.Pattern.MatchString(d.Name()) {
					continue
				}
				link := filepath.Join(fontsDir, r.Name+"--"+d.Name())
				if _, err := os.Lstat(link); err == nil {
					continue // 同名文件已链接（另一个根目录里的重复副本）
				}
				if err := os.Symlink(path, link); err == nil {
					found[r.Name] = true
				}
			}
			return nil
		})
	}
	var missing []string
	for _, r := range androidFontRoles {
		if !found[r.Name] {
			missing = append(missing, r.Name)
		}
	}

	var b strings.Builder
	b.WriteString("<?xml version=\"1.0\"?>\n<!DOCTYPE fontconfig SYSTEM \"urn:fontconfig:fonts.dtd\">\n<fontconfig>\n")
	fmt.Fprintf(&b, "  <dir>%s</dir>\n  <cachedir>%s</cachedir>\n", fontsDir, filepath.Join(dir, "cache"))
	for _, r := range androidFontRoles {
		fmt.Fprintf(&b, "  <match target=\"scan\"><test name=\"file\" compare=\"contains\"><string>/%s--</string></test>"+
			"<edit name=\"family\" mode=\"assign_replace\"><string>%s</string></edit></match>\n", r.Name, r.Name)
	}
	for _, a := range androidFontAliases {
		fmt.Fprintf(&b, "  <alias binding=\"same\"><family>%s</family><prefer><family>%s</family><family>android-cjk</family><family>android-emoji</family></prefer></alias>\n", a.Family, a.Role)
	}
	b.WriteString("</fontconfig>\n")
	conf := filepath.Join(dir, "fonts.conf")
	if err := os.WriteFile(conf, []byte(b.String()), 0o644); err != nil {
		return "", nil, fmt.Errorf("write %s: %w", conf, err)
	}
	return conf, missing, nil
}
```

(b) `pkg/browser/types.go`：`ChromeProcess` 在 `tempUserDataDir` 之后加字段 `fontConfigDir string // 手机身份的进程级字体配置目录，Kill 后删除`。

(c) `pkg/browser/launcher.go`：
- 在 `cmd := exec.CommandContext(ctx, chromePath, flags...)` 之前加：

```go
	// 手机身份（Linux）：进程级字体配置，只看得到安卓字体；桌面身份仍用系统配置
	fontDir, fontEnv := "", []string(nil)
	if mobile && runtime.GOOS == "linux" {
		dir, err := os.MkdirTemp("", "prbg-fonts-")
		if err != nil {
			if tempDir != "" {
				os.RemoveAll(tempDir)
			}
			return nil, fmt.Errorf("android fonts: %w", err)
		}
		conf, missing, err := writeAndroidFontConfig(dir, androidFontRoots)
		if err != nil {
			os.RemoveAll(dir)
			if tempDir != "" {
				os.RemoveAll(tempDir)
			}
			return nil, fmt.Errorf("android fonts: %w", err)
		}
		if len(missing) > 0 {
			androidFontsWarning.Do(func() {
				fmt.Printf("Warning: android fonts missing (%s): install fonts-roboto fonts-noto-core fonts-noto-cjk fonts-noto-color-emoji\n", strings.Join(missing, ", "))
			})
		}
		fontDir, fontEnv = dir, append(os.Environ(), "FONTCONFIG_FILE="+conf)
	}
```

- 在 `configureChromeCmd(cmd)` 之前加 `cmd.Env = fontEnv // nil 时继承当前环境`。
- 启动失败的分支（`if err := cmd.Start(); err != nil {`）里加 `if fontDir != "" { os.RemoveAll(fontDir) }`。
- `ChromeProcess{...}` 字面量里加 `fontConfigDir: fontDir,`。
- 文件级加 `var androidFontsWarning sync.Once // 缺安卓字体只警告一次`。
- import 加 `"runtime"`、`"strings"`、`"sync"`（已有的不重复加）。
- `Kill` 中 `if cp.tempUserDataDir != "" { defer cp.removeTempUserDataDir() }` 改为 `if cp.tempUserDataDir != "" || cp.fontConfigDir != "" { defer cp.removeTempUserDataDir() }`。
- `removeTempUserDataDir` 的最后一行改为：

```go
	if cp.tempUserDataDir != "" {
		os.RemoveAll(cp.tempUserDataDir)
	}
	if cp.fontConfigDir != "" {
		os.RemoveAll(cp.fontConfigDir)
	}
```

  注释第一行改为 `// removeTempUserDataDir 删除本库创建的临时 profile 与手机身份的字体配置目录。`。

(d) `test/linux-chrome.Dockerfile`：apt 安装列表里的 `fonts-noto-cjk` 之后加 `fonts-roboto fonts-noto-core fonts-noto-color-emoji`。注释改为 `…；含中文字体（否则中文显示成方块）与安卓身份用的 Roboto / Noto 字体`。

(e) `docs/deploy/production.Dockerfile`：第一个 `apt-get install` 列表加 `fonts-roboto fonts-noto-core fonts-noto-cjk fonts-noto-color-emoji`，并在其上方加注释：

```dockerfile
# 安卓（手机）身份的字体：开源许可，按进程单独配置，桌面身份看不到（55-windows-fonts-only.conf 已屏蔽）
```

(f) `docs/deploy/fontconfig/55-windows-fonts-only.conf`：`<rejectfont>` 里加三行，并在文件顶部注释的第 1 条里补一句"安卓身份用的 Roboto / Noto 西文字体也屏蔽（手机身份由本库按进程单独配置字体，不经过这个文件）"：

```xml
      <glob>/usr/share/fonts/truetype/roboto/*</glob>
      <glob>/usr/share/fonts/truetype/noto/*</glob>
      <glob>/usr/share/fonts/opentype/noto/NotoSerif*</glob>
```

(g) `docs/deploy/README.md`：在"## 字体"一节末尾加：

```markdown
### 手机（安卓）身份的字体
手机身份的 Chrome 进程由本库单独配置字体（`FONTCONFIG_FILE`），只看得到安卓的字体：sans-serif 为 Roboto，serif 为 Noto Serif，monospace 为 Noto Sans Mono，中文为 Noto Sans CJK，表情为 Noto Color Emoji。Arial、Helvetica、Times New Roman、Courier New 等安卓别名指向它们。镜像里装 `fonts-roboto fonts-noto-core fonts-noto-cjk fonts-noto-color-emoji`（开源许可，可以随镜像分发）。缺字体时启动不失败，但会打印一次警告。
检查：`docker run --rm <镜像> ls /usr/share/fonts/truetype/roboto` 应有 Roboto 字体文件。
```

- [ ] **Step 4: 运行测试，确认通过**

Run: `go vet ./pkg/browser/ && go test -count=1 -run 'TestAndroidFontConfig|TestCloseShutsDownGracefully|TestAndroidLaunchAndViewport' ./pkg/browser/ && scripts/test-linux-chrome.sh -run 'TestAndroidFonts|TestAndroidLaunchAndViewport|TestIdentityAcrossSurfaces'`
Expected: PASS

若 Linux 上 `TestAndroidFonts` 报某个别名检测不到，或某个内部族名被认出：用 `docker run` 在容器里执行 `FONTCONFIG_FILE=… fc-match Arial`、`fc-list` 排查。修配置，不要放宽测试，并在 ledger 记 Ruling。

- [ ] **Step 5: 提交**

```bash
git add pkg/browser/ test/linux-chrome.Dockerfile docs/deploy/
git commit -m "feat(identity): Android fonts through a per-process fontconfig on Linux

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
```

---

### Task 6: 与真机基准整体对照、文档

**Files:**
- Test: `pkg/browser/android_integration_test.go`
- Modify: `README.md`、`CLAUDE.md`

**Interfaces:**
- Consumes: Task 4 的 `collectOn`、`loadAndroidBaseline`；M0 的 `baselineServer`。
- Produces: `androidVersionGap`、`androidResidual`（路径 → 原因）、`diffBaseline(want, got any, path string, skip func(string) bool) []string`。

- [ ] **Step 1: 写对照测试**

在 `pkg/browser/android_integration_test.go` 末尾追加：

```go
// androidVersionGap 宿主 Chrome 与真机基准（Chrome 138）版本不同造成、无法判断真机新版本取值的项目（规格第 7 节）
var androidVersionGap = map[string]string{
	"engine.math.expm1": "引擎级精度差异：ARM 对 x86 或 V8 版本，无法判断",
}

// androidResidual 已知无法模拟、列为残余风险的项目（规格第 9 节）
var androidResidual = map[string]string{
	"media.dynamic-range":          "宿主屏幕不支持高动态范围",
	"screen.visualViewport.height": "真机采集时地址栏状态不同",
}

// androidNotCompared 不属于身份、不比较的项目：版本号、用户设置、网络实时值
var androidNotCompared = []string{
	"identity.userAgent", "identity.userAgentData.brands", "identity.userAgentData.fullVersionList",
	"features.connection.effectiveType", "features.connection.rtt", "features.connection.downlinkMax", "features.connection.saveData",
	"media.prefers-color-scheme", "avmedia.voices", "avmedia.devices",
}

// diffBaseline 递归比较基准与模拟结果，返回不一致的路径；skip 为真的路径不比较
func diffBaseline(want, got any, path string, skip func(string) bool) []string {
	if skip(path) {
		return nil
	}
	w, wok := want.(map[string]any)
	g, gok := got.(map[string]any)
	if wok && gok {
		var out []string
		for k := range w {
			out = append(out, diffBaseline(w[k], g[k], path+"."+k, skip)...)
		}
		return out
	}
	if fmt.Sprint(want) != fmt.Sprint(got) {
		return []string{fmt.Sprintf("%s = %v, want %v", path, got, want)}
	}
	return nil
}

// TestAndroidMatchesBaseline 用真机同款机型（PGKM10）跑采集页，与真机基准逐项比较；
// 只跳过版本差、残余风险与不属于身份的项目，每一项都写明原因
func TestAndroidMatchesBaseline(t *testing.T) {
	srv, _ := baselineServer(t, "", http.StatusOK)
	got := collectOn(t, connectAndroid(t, "android-baseline", "PGKM10").Page(), srv)
	want := loadAndroidBaseline(t)
	skip := func(path string) bool {
		path = strings.TrimPrefix(path, ".")
		if _, ok := androidVersionGap[path]; ok {
			return true
		}
		if _, ok := androidResidual[path]; ok {
			return true
		}
		for _, p := range androidNotCompared {
			if path == p || strings.HasPrefix(path, p+".") {
				return true
			}
		}
		return path == "fonts" && runtime.GOOS != "linux" // 字体只在 Linux 上由本库控制
	}
	diffs := diffBaseline(want, got, "", skip)
	sort.Strings(diffs)
	for _, d := range diffs {
		t.Error(d)
	}
}
```

  并在该文件 import 中加 `"sort"`。

- [ ] **Step 2: 运行对照测试，逐项处理差异**

Run: `go test -count=1 -run TestAndroidMatchesBaseline ./pkg/browser/ ; scripts/test-linux-chrome.sh -run TestAndroidMatchesBaseline`
Expected：首次运行可能报出前 5 个任务没覆盖到的差异（例如 Linux 与 Mac 宿主不同的接口、无界面模式的权限状态、编解码器）。逐项处理：
- 浏览器内能用 JS 或 CDP 模拟、成本不高的：修掉，按 TDD 补测试；
- 硬件或系统级、无法模拟的（例如 HEVC 硬件解码）：加进 `androidResidual`，写明原因，并在 ledger 记 Ruling；
- 宿主与基准版本不同造成的：加进 `androidVersionGap`，写明原因，并在 ledger 记 Ruling。

  直到 macOS 与 Linux 上都 PASS。

- [ ] **Step 3: 文档**

`README.md`：在"### 迁移：默认通道改为 CustomCDP（2026-10）"一节之后加：

```markdown
### 手机身份（安卓，2026-10）
- `ConnectOptions{Mobile: true, FingerprintUserID: "…"}` 为新账号生成安卓 Chrome 身份：UA、Client Hints（含机型）、移动视口与触屏、WebGL（Mali-G610 MC6 真机实测）、内存、电量、通知 / 通讯录 / 网络类型等接口，主页面、跨站 iframe、Worker、弹窗一致。
- 只对新账号起作用。已有指纹的系统永不改变，与已有指纹的类型不一致时 `Connect` 报错。手机身份需要设 `FingerprintUserID`，只支持默认通道（不能与 `UseChromedp` 同用）。
- 机型从 6 款天玑 8100 / 8200 手机中按账号固定抽取：1 款真机实测，其余为整理值。Linux 上字体由本库按进程配置，需要安装 `fonts-roboto fonts-noto-core fonts-noto-cjk fonts-noto-color-emoji`。
- 已知限制：
  - 在手机身份上调用 `Click` / `RealSendKeys` 仍是鼠标和桌面键盘事件（子项目 M2 改为触摸与输入法）；
  - WebGL 实际计算精度仍是 32 位浮点；
  - 不支持高动态范围媒体查询。
  - 详见 `docs/superpowers/specs/2026-10-17-android-identity-design.md`。
```

`CLAUDE.md`："**Identity (anti-detection)**"一节末尾加一条：

```markdown
- Mobile (Android) identities: `ConnectOptions.Mobile` on a new account writes a `device` section into the fingerprint (`identity_android.go`: device pool with the real Mali-G610 WebGL surface; `configOS` decides the OS — files without `device` are always desktop). Applied via mobile `setDeviceMetricsOverride` + touch emulation, `plugins.always_open_pdf_externally` (plugins natively empty), the `C.android` JS block in `identity_script.go`, and on Linux a per-process fontconfig (`fonts_android.go`). `TestAndroidMatchesBaseline` compares against `test/baseline/android-chrome-pgkm10.json`.
```

- [ ] **Step 4: 全量测试**

Run: `go vet ./pkg/... ./internal/... && GOOS=linux go vet ./pkg/... ./internal/... && GOOS=windows go vet ./pkg/... ./internal/... && go test -race -count=1 -timeout 1500s ./cmd/baseline_server/ ./pkg/... ./internal/... && scripts/test-linux-chrome.sh`
Expected: 全部 PASS

- [ ] **Step 5: 提交**

```bash
git add pkg/browser/android_integration_test.go README.md CLAUDE.md
git commit -m "test(identity): compare Android identities with the real-phone baseline; document mobile identities

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
```
