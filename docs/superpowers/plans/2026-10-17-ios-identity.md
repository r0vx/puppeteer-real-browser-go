# iOS 身份（M3，表面级）实现计划

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** `ConnectOptions{Mobile: true, UserAgent: <iPhone UA>}` 给新账号生成 iPhone Safari 身份，浏览器内可见的标识、接口表面、WebGL、输入事件与真机基准一致。

**Architecture:** 沿用 M1 的移动身份骨架：指纹 `device.kind = "ios"` → `normalizeIOS` 按机型档案补齐 → `iosIdentity` 派生 `Identity`（`OS = OSIOS`、`NoClientHints`）→ `applyIdentity` 下发（UA 覆盖不带 Client Hints、color-gamut p3）→ 注入脚本 `C.ios` 段改接口表面 → 输入按 `Identity.OS` 选 iOS 的触点与按键字段。

**Tech Stack:** Go 1.25，CDP（CustomCDP 通道），注入 JS，fontconfig（Linux）。

**Spec:** `docs/superpowers/specs/2026-10-17-ios-identity-design.md`

## Global Constraints

- 只支持 CustomCDP 通道，必须设 `FingerprintUserID`（沿用 `validateMobileOptions`）。
- 已有指纹永不改系统；`device` 段类型与 `ConnectOptions.Mobile` 不一致时报错且不改文件（沿用 `deviceMismatch`）。
- Safari 版本由机型档案决定，不跟随宿主 Chrome。
- Apple 字体受授权限制，不随库分发、不提交进 git（`/fonts/` 已忽略），不从网上下载。
- 代码注释用中文；新增 JS 只在 `C.ios` 下生效，桌面与安卓身份不受影响。
- 每个任务结束时 `go build ./pkg/... ./internal/...` 与 `go vet ./pkg/... ./internal/...` 通过；全量测试在 macOS 与 Linux 容器（`scripts/test-linux-chrome.sh`）各跑一遍。

## Review Focus

1. 已有 iOS 指纹不带 `Mobile` 加载、桌面指纹带 `Mobile` 与 iPhone UA 加载：报错且文件一个字节不改（Task 1 `TestFingerprintForOptionsMobile`）。
2. 跨站 iframe、专用 Worker、弹窗与主页面一致：没有 `userAgentData`、`vendor` 为 Apple、原生函数文本是 JavaScriptCore 格式；请求头没有 `sec-ch-ua*`（Task 2 / 3 `TestIOSIdentityAcrossSurfaces`）。
3. 桌面与安卓身份不受 iOS 改动影响：`Notification`、`userAgentData`、原生函数文本保持原样（Task 3 `TestIOSSurfaceAPIs/desktop`，安卓测试全部保留）。
4. 删掉 `MediaSource` 后视频播放器仍能用 MSE：`ManagedMediaSource` 能打开、能加 SourceBuffer（Task 3 `TestIOSSurfaceAPIs`）。
5. iOS 键盘打字之后照样能点按、回车产生换行（keyup 带真实按键码不能卡住后续输入）（Task 4 `TestIOSTapAndTypeMatchBaseline`、`TestIOSEnterKey`）。

---

### Task 1: iOS 机型档案、指纹规范化与身份派生

**Files:**
- Create: `pkg/browser/ios_identity.go`
- Modify: `pkg/browser/android_identity.go`（`configOS`、`knownDeviceKind`）
- Modify: `pkg/browser/fingerprint_config.go`（`DeviceConfig` 注释、`Normalize`、`generateFingerprintFor`）
- Modify: `pkg/browser/user_fingerprint_manager.go`（新账号 Mobile + iPhone UA → iOS）
- Modify: `pkg/browser/identity.go`（`Identity.NoClientHints`、`Identity.NoWebGPU`、`identityFromConfig`、`configGeometry`）
- Test: `pkg/browser/ios_identity_test.go`（新）、`pkg/browser/android_identity_test.go`（改 `TestFingerprintForOptionsMobile`）

**Interfaces:**
- Produces: `OSIOS OSFamily = "ios"`；`deviceKindIOS = "ios"`；`iosPlatform`、`iosVendor`、`iosCores`、`iosTouchPoints`；`type iosDevice struct{Model string; Screen screenSpec; UAOS, Safari string; WebGPU bool; GPU gpuProfile; Source string}`；`iosDevices []weighted[iosDevice]`；`lookupIOSDevice(model) (iosDevice, bool)`；`iosUserAgent(iosDevice) string`；`iosGeometry(screenSpec) screenGeometry`；`normalizeIOS(*FingerprintConfig, func(string) uint64)`；`iosIdentity(*FingerprintConfig, hostInfo) (*Identity, error)`；`appleGPU(maxSamples int) gpuProfile`；`Identity.NoClientHints bool`、`Identity.NoWebGPU bool`。

- [ ] **Step 1: 写失败的测试** `pkg/browser/ios_identity_test.go`

```go
package browser

import (
	"encoding/json"
	"os"
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
```

改 `pkg/browser/android_identity_test.go` 的 `TestFingerprintForOptionsMobile`：注释末尾"iOS 尚不支持"改为"iPhone UA 生成 iOS 身份"；把末尾的 iOS 报错断言（`i1` 那两段）换成：

```go
	ios, err := fingerprintForOptions(&ConnectOptions{FingerprintUserID: "i1", FingerprintDir: dir, Mobile: true, UserAgent: iphoneTestUA})
	if err != nil {
		t.Fatal(err)
	}
	if ios.Device == nil || ios.Device.Kind != deviceKindIOS || ios.Browser.Platform != iosPlatform {
		t.Fatalf("new iOS account has device %+v / platform %q", ios.Device, ios.Browser.Platform)
	}
	// 再次加载（新的管理器读文件）：认识 ios 类型；不带 Mobile 时报错且不改文件
	if _, err := fingerprintForOptions(&ConnectOptions{FingerprintUserID: "i1", FingerprintDir: dir, Mobile: true}); err != nil {
		t.Errorf("reload iOS account: %v", err)
	}
	before, _ := os.ReadFile(filepath.Join(dir, "i1.json"))
	if _, err := fingerprintForOptions(&ConnectOptions{FingerprintUserID: "i1", FingerprintDir: dir}); err == nil || !strings.Contains(err.Error(), "mobile identity") {
		t.Errorf("iOS file without Mobile: err = %v", err)
	}
	if after, _ := os.ReadFile(filepath.Join(dir, "i1.json")); !bytes.Equal(before, after) {
		t.Error("iOS fingerprint file changed")
	}
```

- [ ] **Step 2: 运行，确认失败**

Run: `go test -run 'TestIOS|TestFingerprintForOptionsMobile' ./pkg/browser/`
Expected: 编译失败（`lookupIOSDevice`、`deviceKindIOS`、`OSIOS` 等未定义）

- [ ] **Step 3: 实现** `pkg/browser/ios_identity.go`

```go
package browser

import (
	"fmt"
	"strings"
)

// iOS 身份（移动身份子项目 M3，表面级）：iPhone Safari 的机型档案与派生规则，真机实测见 test/baseline/ios-safari-*.json。
// 设计见 docs/superpowers/specs/2026-10-17-ios-identity-design.md；JS 引擎、TLS 握手、渲染引擎无法伪装（规格第 6 节）

// OSIOS iPhone Safari
const OSIOS OSFamily = "ios"

const (
	// deviceKindIOS 指纹 device 段的类型：iOS
	deviceKindIOS = "ios"
	// iosPlatform / iosVendor iPhone Safari 的 navigator.platform / vendor（真机实测）
	iosPlatform = "iPhone"
	iosVendor   = "Apple Computer, Inc."
	// iosCores Safari 对 hardwareConcurrency 统一报 4（真机实测）
	iosCores = 4
	// iosTouchPoints iPhone 的 maxTouchPoints（真机实测）
	iosTouchPoints = 5
)

// iosDevice 一款 iPhone 的档案：屏幕、系统与 Safari 版本、GPU 是一个整体
type iosDevice struct {
	Model  string     // 档案代号（iPhone 不经 UA 或 Client Hints 暴露机型）
	Screen screenSpec // CSS 像素与 DPR；真机采集时地址栏收起，视口即整屏
	UAOS   string     // UA 里的系统段，如 18_7（iOS 26 起冻结为 18_7）
	Safari string     // UA 里 Version/ 后的 Safari 版本
	WebGPU bool       // navigator.gpu（iOS 26 有，18.5 没有）
	GPU    gpuProfile
	Source string
}

// iosDevices iPhone 机型池：两款都是真机实测
var iosDevices = []weighted[iosDevice]{
	{iosDevice{"iphone-ios26", screenSpec{420, 912, 3, false}, "18_7", "26.2", true, appleGPU(8), "真机实测：iPhone，iOS 26，Safari 26.2"}, 60},
	{iosDevice{"iphone-ios18.5", screenSpec{440, 956, 3, false}, "18_5", "18.5", false, appleGPU(4), "真机实测：iPhone，iOS 18.5，Safari 18.5"}, 40},
}

// lookupIOSDevice 按档案代号查找
func lookupIOSDevice(model string) (iosDevice, bool) {
	for _, d := range iosDevices {
		if d.Value.Model == model {
			return d.Value, true
		}
	}
	return iosDevice{}, false
}

// iosUserAgent iPhone Safari 的 UA
func iosUserAgent(d iosDevice) string {
	return "Mozilla/5.0 (iPhone; CPU iPhone OS " + d.UAOS + " like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/" +
		d.Safari + " Mobile/15E148 Safari/604.1"
}

// iosGeometry iPhone 的屏幕与视口：可用区域、窗口外框、视口都是整屏（真机采集时地址栏收起），视口顶端即屏幕顶端
func iosGeometry(s screenSpec) screenGeometry {
	return screenGeometry{
		Width: s.Width, Height: s.Height, DPR: s.DPR, AvailWidth: s.Width, AvailHeight: s.Height,
		OuterWidth: s.Width, OuterHeight: s.Height, InnerWidth: s.Width, InnerHeight: s.Height, Mobile: true,
	}
}

// normalizeIOS iOS 指纹的设备相关字段：档案不在池中时按用户 ID 重抽，其余字段一律取自档案
func normalizeIOS(config *FingerprintConfig, seed func(string) uint64) {
	dev, ok := lookupIOSDevice(config.Device.Model)
	if !ok {
		dev = pickWeighted(iosDevices, seed("device"))
	}
	config.Device.Model, config.Device.DeviceMemory = dev.Model, 0
	config.Browser.UserAgent = iosUserAgent(dev)
	config.Browser.Platform = iosPlatform
	config.Browser.Vendor = iosVendor
	config.Browser.PlatformVersion = ""
	config.Browser.HardwareConcurrency = iosCores
	config.Browser.MaxTouchPoints = iosTouchPoints
	config.Screen.Width, config.Screen.Height, config.Screen.DevicePixelRatio = dev.Screen.Width, dev.Screen.Height, dev.Screen.DPR
	config.Screen.AvailWidth, config.Screen.AvailHeight = dev.Screen.Width, dev.Screen.Height
	config.Screen.ColorDepth, config.Screen.PixelDepth = 24, 24
	config.WebGL.Vendor, config.WebGL.Renderer = dev.GPU.Vendor, dev.GPU.Renderer
	config.WebGL.Version = dev.GPU.Mobile.Params[7938].(string)
	config.WebGL.ShadingLanguageVersion = dev.GPU.Mobile.Params[35724].(string)
	config.WebGL.MaxTextureSize = dev.GPU.Mobile.Params[3379].(int)
	config.WebGL.MaxRenderbufferSize = dev.GPU.Mobile.Params[34024].(int)
}

// iosIdentity iOS 身份：UA、屏幕、GPU 都取自档案（Safari 版本不跟随宿主 Chrome）；Safari 不发 Client Hints，languages 只有主语言
func iosIdentity(cfg *FingerprintConfig, host hostInfo) (*Identity, error) {
	dev, ok := lookupIOSDevice(cfg.Device.Model)
	if !ok {
		return nil, fmt.Errorf("fingerprint %s: iOS model %q is not in the pool (config not normalized)", cfg.UserID, cfg.Device.Model)
	}
	if len(cfg.Browser.Languages) == 0 {
		return nil, fmt.Errorf("fingerprint %s: no languages", cfg.UserID)
	}
	gpu := dev.GPU
	langs := cfg.Browser.Languages[:1] // Safari 的 navigator.languages 只有主语言（真机实测）
	return &Identity{
		OS:                  OSIOS,
		UserAgent:           iosUserAgent(dev),
		Platform:            iosPlatform,
		AcceptLanguage:      strings.Join(langs, ","),
		Languages:           langs,
		Timezone:            cfg.Timezone.Timezone,
		HardwareConcurrency: iosCores,
		NoClientHints:       true,
		NoWebGPU:            !dev.WebGPU,
		Screen:              iosGeometry(dev.Screen),
		GPU:                 &gpu,
		NoiseSeed:           noiseSeed(cfg.UserID),
		ScriptKey:           newScriptKey(),
	}, nil
}
```

文件末尾追加 `appleGPU`：取 `scratchpad/apple-gpu.go.txt` 的函数（参数、WebGL1 / WebGL2 扩展列表、精度表逐项抄自 `ios-safari-iphone-2.json`），注释改为：

```go
// appleGPU iPhone 的 Apple GPU 在 Safari 上的实测值（test/baseline/ios-safari-*.json）：两台 iPhone 只有 MAX_SAMPLES 不同（iOS 26 为 8，18.5 为 4）。
// Safari 的 VERSION / SHADING_LANGUAGE_VERSION 没有 Chromium 后缀，一并覆盖
func appleGPU(maxSamples int) gpuProfile {
	// ……params2 / Params / Extensions / Extensions2 / Precision 见草稿，逐项与基准一致……
}
```

`pkg/browser/android_identity.go`：

```go
// configOS 指纹声称的系统：有 device 段的按其类型（移动身份），否则按 UA 判定 Windows / macOS——老指纹永不变成移动身份
func configOS(cfg *FingerprintConfig) OSFamily {
	if cfg.Device != nil {
		switch cfg.Device.Kind {
		case deviceKindAndroid:
			return OSAndroid
		case deviceKindIOS:
			return OSIOS
		}
	}
	return osFromUserAgent(cfg.Browser.UserAgent)
}

// knownDeviceKind 本版本认识的移动设备类型；更新的版本写的类型不能按桌面规范化回写，否则会毁掉那份指纹
func knownDeviceKind(kind string) bool {
	return kind == deviceKindAndroid || kind == deviceKindIOS
}
```

`pkg/browser/fingerprint_config.go`：`DeviceConfig.Kind` 注释改为 `// android / ios`；`Normalize` 文档"或有 device 段时的安卓"改为"或有 device 段时的安卓 / iOS"；系统分支改为：

```go
	mobile := osf == OSAndroid || osf == OSIOS
	switch osf {
	case OSAndroid:
		normalizeAndroid(config, seed)
	case OSIOS:
		normalizeIOS(config, seed)
	default:
		// ……原桌面分支不变……
	}
```

`if osf != OSAndroid {` 改为 `if !mobile {`。`generateFingerprintFor`：

```go
	switch osf {
	case OSAndroid:
		c.Device = &DeviceConfig{Kind: deviceKindAndroid}
		c.Browser.UserAgent = androidUserAgent("120")
	case OSIOS:
		c.Device = &DeviceConfig{Kind: deviceKindIOS} // UA 由 Normalize 按机型给出
	default:
		c.Browser.UserAgent = canonicalUserAgent(osf, "120")
	}
```

`pkg/browser/user_fingerprint_manager.go` 新账号分支：

```go
	// 新账号：要求移动身份时生成安卓，绑的是 iPhone / iPad UA 时生成 iOS（iPhone，表面级）；否则系统由绑定的 UA 决定，未提供 UA 时为 Windows
	osFamily := OSWindows
	switch {
	case initParams != nil && initParams.Mobile:
		osFamily = OSAndroid
		if isIOSUserAgent(initParams.UserAgent) {
			osFamily = OSIOS
		}
	case initParams != nil && initParams.UserAgent != "":
		osFamily = osFromUserAgent(initParams.UserAgent)
	}
```

`pkg/browser/identity.go`：`Identity` 在 `Metadata` 后加

```go
	NoClientHints       bool // Safari 不发 Client Hints：UA 覆盖不带 userAgentMetadata，注入脚本删掉 userAgentData（iOS 身份）
	NoWebGPU            bool // 没有 navigator.gpu（iOS 18.5）；false 表示保持原生
```

`identityFromConfig` 开头改为：

```go
	switch configOS(cfg) {
	case OSAndroid:
		return androidIdentity(cfg, host)
	case OSIOS:
		return iosIdentity(cfg, host)
	}
```

`configGeometry`：

```go
	os := configOS(cfg)
	s := screenSpec{Width: cfg.Screen.Width, Height: cfg.Screen.Height, DPR: cfg.Screen.DevicePixelRatio}
	switch os {
	case OSAndroid:
		return androidGeometry(s)
	case OSIOS:
		return iosGeometry(s)
	}
```

- [ ] **Step 4: 运行，确认通过**

Run: `go test -run 'TestIOS|TestFingerprintForOptionsMobile|TestAndroid(Normalize|IdentityFromConfig|HeadlessScreenFlags)|TestMobileUserAgentFileStaysDesktop|TestFingerprintRejectsUnknownDeviceKind' ./pkg/browser/`
Expected: PASS

- [ ] **Step 5: 提交**

```bash
git add pkg/browser/ios_identity.go pkg/browser/ios_identity_test.go pkg/browser/android_identity.go pkg/browser/android_identity_test.go pkg/browser/fingerprint_config.go pkg/browser/user_fingerprint_manager.go pkg/browser/identity.go
git commit -m "feat(identity): iOS device profiles, fingerprints and identities"
```

---

### Task 2: 下发与启动（不发 Client Hints、广色域、启动参数）

**Files:**
- Modify: `pkg/browser/identity_apply.go`（`applyIdentity`）
- Modify: `pkg/browser/launcher.go`（PDF 与 WebBluetooth / NetInfo 只给安卓）
- Modify: `pkg/browser/android_integration_test.go`（`androidFingerprintDir` → `mobileFingerprintDir`，`newAndroidSurfaceServers` → `newSurfaceServers(t, js)`）
- Test: `pkg/browser/ios_integration_test.go`（新）

**Interfaces:**
- Consumes: Task 1 的 `OSIOS`、`Identity.NoClientHints`、`generateFingerprintFor(userID, OSIOS)`。
- Produces: 测试辅助 `mobileFingerprintDir(t, osf OSFamily, userID, model string) string`、`connectIOS(t, userID, model string) *BrowserInstance`、`newSurfaceServers(t, js string)`、`iosSurfaceJS`。

- [ ] **Step 1: 写失败的测试**

`android_integration_test.go`：

```go
// mobileFingerprintDir 写一份指定系统与机型的手机指纹，返回指纹目录
func mobileFingerprintDir(t *testing.T, osf OSFamily, userID, model string) string {
	t.Helper()
	dir := t.TempDir()
	cfg := generateFingerprintFor(userID, osf)
	cfg.Device.Model = model
	cfg.Normalize()
	// ……其余同原 androidFingerprintDir……
}
```

`connectAndroid` 改用 `mobileFingerprintDir(t, OSAndroid, userID, model)`；`newAndroidSurfaceServers(t)` 改为 `newSurfaceServers(t *testing.T, js string)`，函数体里的 `androidSurfaceJS` 换成 `js`，`TestAndroidIdentityAcrossSurfaces` 传 `androidSurfaceJS`。

新文件 `pkg/browser/ios_integration_test.go`：

```go
package browser

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// connectIOS 用指定机型的 iOS 身份启动（无界面，默认通道）
func connectIOS(t *testing.T, userID, model string) *BrowserInstance {
	t.Helper()
	inst, err := Connect(t.Context(), &ConnectOptions{Headless: true, Mobile: true, FingerprintUserID: userID, FingerprintDir: mobileFingerprintDir(t, OSIOS, userID, model)})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { inst.Close() })
	return inst
}

// iosSurfaceJS 各表面（页面、iframe、Worker、弹窗）上报的身份信号（Safari 没有 userAgentData，只看它在不在）
const iosSurfaceJS = `async () => ({ua: navigator.userAgent, platform: navigator.platform, cores: navigator.hardwareConcurrency,
  languages: navigator.languages.join(), uad: 'userAgentData' in navigator, vendor: self.document ? navigator.vendor : null,
  native: Function.prototype.toString.call(Math.max),
  touch: self.document ? navigator.maxTouchPoints : null, coarse: self.document ? matchMedia('(pointer: coarse)').matches : null,
  p3: self.document ? matchMedia('(color-gamut: p3)').matches : null,
  screen: self.document ? [screen.width, screen.height, devicePixelRatio].join() : null})`

// TestIOSIdentityAcrossSurfaces 主页面、跨站 iframe、专用 Worker、弹窗与请求头都是同一台 iPhone：Safari 的 UA，请求头没有 Client Hints（Review Focus 2）
func TestIOSIdentityAcrossSurfaces(t *testing.T) {
	srv, headers := newSurfaceServers(t, iosSurfaceJS)
	p := connectIOS(t, "ios-surfaces", "iphone-ios26").Page()
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
	ua := iosUserAgent(iosDevices[0].Value)
	for _, surface := range []string{"main", "iframe", "worker", "popup"} {
		info := got[surface]
		if info == nil {
			t.Errorf("%s never reported", surface)
			continue
		}
		want := map[string]any{"ua": ua, "platform": "iPhone", "cores": 4.0, "languages": "zh-CN"}
		if surface != "worker" {
			want["touch"], want["coarse"], want["p3"], want["screen"] = 5.0, true, true, "420,912,3"
		}
		for k, v := range want {
			if info[k] != v {
				t.Errorf("%s %s = %v, want %v", surface, k, info[k], v)
			}
		}
	}
	for _, path := range []string{"/", "/f", "/worker.js"} {
		h := headers(path)
		if h.Get("User-Agent") != ua {
			t.Errorf("%s User-Agent %q", path, h.Get("User-Agent"))
		}
		for k := range h {
			if strings.HasPrefix(strings.ToLower(k), "sec-ch-ua") {
				t.Errorf("%s sends %s: %v (Safari sends no client hints)", path, k, h[k])
			}
		}
	}
}

// TestIOSLaunchAndViewport 插件与 PDF 查看器保持原生（Safari 同样报 5 个插件）；视口即整屏；SetViewport 按桌面习惯传 1920×1080 时限制在屏幕内
func TestIOSLaunchAndViewport(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { fmt.Fprint(w, mobilePage) }))
	defer srv.Close()
	p := connectIOS(t, "ios-launch", "iphone-ios18.5").Page()
	const js = `[navigator.plugins.length, navigator.mimeTypes.length, navigator.pdfViewerEnabled, innerWidth, innerHeight, outerWidth, outerHeight,
	  screen.width, screen.height, devicePixelRatio, screen.orientation.type, matchMedia('(pointer: coarse)').matches].join()`
	if err := p.Navigate(srv.URL); err != nil {
		t.Fatal(err)
	}
	if got, _ := p.Evaluate(js); got != "5,2,true,440,956,440,956,440,956,3,portrait-primary,true" {
		t.Errorf("launch state %v", got)
	}
	if err := p.SetViewport(1920, 1080); err != nil {
		t.Fatal(err)
	}
	if err := p.Navigate(srv.URL + "/?after"); err != nil {
		t.Fatal(err)
	}
	if got, _ := p.Evaluate(js); got != "5,2,true,440,956,440,956,440,956,3,portrait-primary,true" {
		t.Errorf("after SetViewport(1920, 1080) %v", got)
	}
}
```

- [ ] **Step 2: 运行，确认失败**

Run: `go test -run 'TestIOSIdentityAcrossSurfaces|TestIOSLaunchAndViewport|TestAndroidIdentityAcrossSurfaces' ./pkg/browser/`
Expected: `TestIOSIdentityAcrossSurfaces` 失败于 `sec-ch-ua` 请求头与 `p3 = false`；`TestIOSLaunchAndViewport` 失败于插件数 0（PDF 查看器被关）；安卓测试仍通过。

- [ ] **Step 3: 实现**

`identity_apply.go` `applyIdentity`：

```go
	ua := map[string]any{"userAgent": id.UserAgent, "acceptLanguage": id.AcceptLanguage, "platform": id.Platform}
	// Safari 不发 Client Hints：不带 userAgentMetadata 时 Chrome 不发 sec-ch-ua* 请求头（2026-10-17 实测）
	if !id.NoClientHints {
		ua["userAgentMetadata"] = id.Metadata
	}
```

触屏模拟那段之后加：

```go
	// iPhone 的屏幕是广色域（color-gamut: p3），宿主是 sRGB；dynamic-range 不能模拟（残余风险）
	if id.OS == OSIOS {
		steps = append(steps, cdpStep{"Emulation.setEmulatedMedia", map[string]any{"features": []map[string]any{{"name": "color-gamut", "value": "p3"}}}})
	}
```

`launcher.go`：

```go
	osf := OSFamily("")
	if fingerprint != nil {
		osf = configOS(fingerprint)
	}
	android := osf == OSAndroid
	mobile := android || osf == OSIOS
```

`disablePDFViewer` 与 WebBluetooth / NetInfoDownlinkMax 两处的 `if mobile` 改为 `if android`（注释注明：Safari 的插件与 PDF 查看器和桌面 Chrome 原生相同；iPhone 没有 Web Bluetooth 与 NetInfo，由注入脚本删掉）。字体段保持 `if mobile && runtime.GOOS == "linux"`（iOS 暂用安卓字体配置，Task 5 换成 Apple 字体）。

- [ ] **Step 4: 运行，确认通过**

Run: `go test -run 'TestIOSIdentityAcrossSurfaces|TestIOSLaunchAndViewport|TestAndroidIdentityAcrossSurfaces|TestAndroidLaunchAndViewport' ./pkg/browser/`
Expected: PASS（`uad`、`vendor`、`native` 本任务不断言）

- [ ] **Step 5: 提交**

```bash
git add pkg/browser/identity_apply.go pkg/browser/launcher.go pkg/browser/android_integration_test.go pkg/browser/ios_integration_test.go
git commit -m "feat(identity): iOS identities send no client hints and report a wide-gamut screen"
```

---

### Task 3: 注入脚本的 iOS 接口表面

**Files:**
- Modify: `pkg/browser/identity_script.go`
- Test: `pkg/browser/ios_integration_test.go`

**Interfaces:**
- Consumes: Task 1 的 `Identity.OS == OSIOS`、`Identity.NoWebGPU`。
- Produces: `jsIdentity.IOS *jsIOS`（JSON `ios`），`type jsIOS struct{ NoWebGPU bool \`json:"noWebGPU"\` }`；JS 侧 `C.ios`。

- [ ] **Step 1: 写失败的测试**

`TestIOSIdentityAcrossSurfaces` 的 `want` 加上（所有表面）：

```go
		want["uad"], want["native"] = false, "function max() {\n    [native code]\n}"
		if surface != "worker" {
			want["vendor"] = "Apple Computer, Inc."
		}
```

`ios_integration_test.go` 追加：

```go
// iosAPIsJS iPhone Safari 与 Chrome 不同的 JS 接口
const iosAPIsJS = `(async () => {
  const has = (o, ks) => ks.filter(k => k in o);
  const mms = new ManagedMediaSource();
  const v = document.createElement('video');
  const opened = await new Promise(r => { mms.addEventListener('sourceopen', () => r(true)); v.src = URL.createObjectURL(mms); setTimeout(() => r(false), 3000); });
  let sb; try { sb = String(mms.addSourceBuffer('video/mp4; codecs="avc1.42E01E"')); } catch (e) { sb = e.name; }
  let ap; try { ap = ApplePaySession.canMakePayments(); } catch (e) { ap = e.name; }
  let db; try { openDatabase('a', '1', 'a', 1); db = 'opened'; } catch (e) { db = e.name; }
  const worker = await new Promise(r => {
    const src = "postMessage(['userAgentData' in navigator, 'deviceMemory' in navigator, typeof MediaSource, typeof Notification, Function.prototype.toString.call(Math.max)].join('|'))";
    const w = new Worker(URL.createObjectURL(new Blob([src], {type: 'text/javascript'})));
    w.onmessage = e => r(e.data);
  });
  return JSON.stringify({
    vendor: navigator.vendor, standalone: navigator.standalone, standaloneIn: 'standalone' in navigator,
    removedNav: has(navigator, ['userAgentData', 'deviceMemory', 'connection', 'bluetooth', 'getBattery', 'vibrate', 'usb', 'serial', 'hid', 'keyboard', 'xr', 'contacts', 'virtualKeyboard']),
    removedWin: has(window, ['chrome', 'Notification', 'MediaSource', 'BarcodeDetector', 'NavigatorUAData', 'NetworkInformation', 'showOpenFilePicker',
      'webkitRequestFileSystem', 'EyeDropper', 'launchQueue', 'queryLocalFonts', 'getScreenDetails']),
    present: ['ApplePaySession', 'GestureEvent', 'ManagedMediaSource', 'openDatabase', 'SharedWorker'].filter(k => k in window).join(),
    gesture: Object.getPrototypeOf(GestureEvent.prototype) === UIEvent.prototype,
    mms: mms instanceof ManagedMediaSource && mms.streaming === true, opened, sb,
    applePay: ap, db, dbLength: openDatabase.length, perm: (await navigator.permissions.query({name: 'notifications'})).state,
    geo: (await navigator.permissions.query({name: 'geolocation'})).state,
    patched: Function.prototype.toString.call(Object.getOwnPropertyDescriptor(Navigator.prototype, 'vendor').get),
    fake: Function.prototype.toString.call(ApplePaySession),
    user: Function.prototype.toString.call(function f() { return 1; }),
    share: typeof navigator.share, gpu: 'gpu' in navigator, worker,
  });
})()`

// TestIOSSurfaceAPIs iOS 身份的 JS 接口与 iPhone 一致：Chrome 独有的删掉，Safari 独有的补上，原生函数文本是 JavaScriptCore 格式；
// 删掉 MediaSource 后播放器仍能用 ManagedMediaSource（Review Focus 4）；桌面身份不受影响（Review Focus 3）
func TestIOSSurfaceAPIs(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { fmt.Fprint(w, `<title>plain</title>`) }))
	defer srv.Close()
	t.Run("ios", func(t *testing.T) {
		p := connectIOS(t, "ios-apis", "iphone-ios18.5").Page()
		if err := p.Navigate(srv.URL); err != nil {
			t.Fatal(err)
		}
		got := map[string]any{}
		if err := json.Unmarshal([]byte(awaitJS(t, p, iosAPIsJS)), &got); err != nil {
			t.Fatal(err)
		}
		want := map[string]any{
			"vendor": "Apple Computer, Inc.", "standalone": false, "standaloneIn": true,
			"present": "ApplePaySession,GestureEvent,ManagedMediaSource,openDatabase,SharedWorker", "gesture": true,
			"mms": true, "opened": true, "sb": "[object SourceBuffer]", "applePay": true, "db": "SecurityError", "dbLength": 4.0,
			"perm": "denied", "geo": "prompt",
			"patched": "function get vendor() {\n    [native code]\n}", "fake": "function ApplePaySession() {\n    [native code]\n}",
			"user": "function f() { return 1; }", "share": "function", "gpu": false,
			"worker": "false|false|undefined|undefined|function max() {\n    [native code]\n}",
		}
		for k, v := range want {
			if got[k] != v {
				t.Errorf("%s = %v, want %v", k, got[k], v)
			}
		}
		for _, k := range []string{"removedNav", "removedWin"} {
			if r := got[k].([]any); len(r) != 0 {
				t.Errorf("%s: Chrome-only APIs still present: %v", k, r)
			}
		}
	})
	t.Run("desktop", func(t *testing.T) {
		inst, err := Connect(t.Context(), &ConnectOptions{Headless: true, FingerprintUserID: "desktop-ios-apis", FingerprintDir: t.TempDir()})
		if err != nil {
			t.Fatal(err)
		}
		defer inst.Close()
		p := inst.Page()
		if err := p.Navigate(srv.URL); err != nil {
			t.Fatal(err)
		}
		const js = `[navigator.vendor, typeof Notification, 'userAgentData' in navigator, typeof ApplePaySession, Function.prototype.toString.call(Math.max)].join('|')`
		if got, _ := p.Evaluate(js); got != "Google Inc.|function|true|undefined|function max() { [native code] }" {
			t.Errorf("desktop identity changed: %v", got)
		}
	})
}

// TestIOSWebGL WebGL1 / WebGL2 的标识、版本串、能力参数、精度、扩展列表与 iPhone 一致
func TestIOSWebGL(t *testing.T) {
	srv, _ := baselineServer(t, "", http.StatusOK)
	got, want := collectOn(t, connectIOS(t, "ios-webgl", "iphone-ios26").Page(), srv), loadIOSBaseline(t, "ios-safari-iphone-2.json")
	for _, kind := range []string{"webgl", "webgl2"} {
		for _, d := range diffBaseline(want[kind], got[kind], kind, func(string) bool { return false }) {
			t.Error(d)
		}
	}
}
```

- [ ] **Step 2: 运行，确认失败**

Run: `go test -run 'TestIOSIdentityAcrossSurfaces|TestIOSSurfaceAPIs|TestIOSWebGL' ./pkg/browser/`
Expected: FAIL：`uad = true`、`vendor = Google Inc.`、`ManagedMediaSource is not defined`、版本串带 Chromium 后缀等

- [ ] **Step 3: 实现** `identity_script.go`

`jsIdentity` 加 `IOS *jsIOS \`json:"ios,omitempty"\``，并定义：

```go
// jsIOS iOS 身份专有：没有 navigator.gpu 的机型（iOS 18.5）
type jsIOS struct {
	NoWebGPU bool `json:"noWebGPU"`
}
```

`identityScript` 在安卓那段后加：

```go
	if id.OS == OSIOS {
		cfg.IOS = &jsIOS{NoWebGPU: id.NoWebGPU}
	}
```

`jsPrelude` 的 `toStringProxy` 改为：

```js
// iOS 身份：原生函数文本是 JavaScriptCore 的格式（function max() {\n    [native code]\n}），伪装表里的条目一并转换；页面自己的函数原样
const jsc = (s) => C.ios && s.endsWith(' { [native code] }') ? s.slice(0, -18) + ' {\n    [native code]\n}' : s;
const toStringProxy = {toString() {
  if (this === C.key) return masks;
  return jsc(masks.has(this) ? masks.get(this) : Reflect.apply(nativeToString, this, []));
}}.toString;
```

`jsCommonPart` 末尾（`if (C.android) {…}` 之后）加：

```js
if (C.ios) {
  // Safari 没有的接口（页面与 Worker）：Client Hints、内存、网络信息、电池、振动、硬件接口；navigator.gpu 只有 iOS 26 有
  const navKeys = ['userAgentData', 'deviceMemory', 'connection', 'hid', 'serial', 'usb', 'bluetooth', 'keyboard', 'ink', 'scheduling',
    'presentation', 'getInstalledRelatedApps', 'setAppBadge', 'clearAppBadge', 'virtualKeyboard', 'webkitTemporaryStorage',
    'webkitPersistentStorage', 'xr', 'devicePosture', 'windowControlsOverlay', 'contacts', 'getBattery', 'vibrate'];
  if (C.ios.noWebGPU) navKeys.push('gpu');
  for (const N of [G.Navigator, G.WorkerNavigator]) if (N) for (const k of navKeys) delete N.prototype[k];
  // ManagedMediaSource：iPhone 上没有 MediaSource，只有它（iOS 17.1 起）。由宿主的 MediaSource 实现，播放器照常可用；
  // 编解码支持沿用宿主（谎报宿主解不了的格式会让网站推来放不了的视频）
  if (G.MediaSource && G.document && !G.ManagedMediaSource) {
    const MS = G.MediaSource;
    const MMS = function ManagedMediaSource() {
      if (new.target === undefined) throw strip(new TypeError('calling ManagedMediaSource constructor without new is invalid'), MMS);
      return Reflect.construct(MS, [], new.target);
    };
    Object.setPrototypeOf(MMS.prototype, MS.prototype);
    Object.setPrototypeOf(MMS, MS);
    const streaming = Object.getOwnPropertyDescriptor({get streaming() {
      if (!(this instanceof MMS)) throw strip(new TypeError('The ManagedMediaSource.streaming getter can only be used on instances of ManagedMediaSource'), streaming);
      return true;
    }}, 'streaming').get;
    masks.set(streaming, 'function get streaming() { [native code] }');
    Object.defineProperty(MMS.prototype, 'streaming', {get: streaming, enumerable: true, configurable: true});
    fakeInterface(MMS, 'ManagedMediaSource', {});
  }
  for (const k of ['NavigatorUAData', 'Notification', 'BarcodeDetector', 'NetworkInformation', 'MediaSource', 'HID', 'HIDDevice', 'Serial',
    'SerialPort', 'USB', 'USBDevice', 'Bluetooth', 'Keyboard', 'Scheduler', 'scheduler', 'BatteryManager', 'ContactsManager']) delete G[k];
  if (C.ios.noWebGPU) for (const k of ['GPU', 'GPUAdapter', 'GPUDevice']) delete G[k];
  for (const N of [G.Navigator, G.WorkerNavigator]) if (N && 'vendor' in N.prototype) patchGetter(N.prototype, 'vendor', () => 'Apple Computer, Inc.');
  // 通知权限：Safari 网页（不是加到主屏幕的网页应用）为 denied（真机实测）
  if (G.Permissions && G.PermissionStatus) {
    const denied = new WeakSet();
    patchMethod(Permissions.prototype, 'query', (orig, self, args) => {
      const p = Reflect.apply(orig, self, args);
      const d = args[0];
      return d !== null && typeof d === 'object' && d.name === 'notifications' ? p.then(s => (denied.add(s), s)) : p;
    });
    patchGetter(PermissionStatus.prototype, 'state', (orig, self) => denied.has(self) ? 'denied' : Reflect.apply(orig, self, []));
  }
}
```

`jsWindowPart`：Web Share 的正则改为 `/^(Win32|MacIntel|Linux armv81|iPhone)$/`，注释改为"Windows / Mac / 安卓 / iPhone 都有"。文件末尾（`if (C.android) {…}` 之后）加：

```js
if (C.ios) {
  // Chrome 与桌面独有的接口：iPhone Safari 没有
  for (const k of ['chrome', 'launchQueue', 'showDirectoryPicker', 'showOpenFilePicker', 'showSaveFilePicker', 'webkitRequestFileSystem',
    'webkitResolveLocalFileSystemURL', 'EyeDropper', 'documentPictureInPicture', 'queryLocalFonts', 'getScreenDetails', 'ScreenDetails',
    'ScreenDetailed', 'CaptureController', 'CropTarget', 'RestrictionTarget', 'DocumentPictureInPicture', 'DocumentPictureInPictureEvent',
    'FontData', 'IdleDetector', 'WindowControlsOverlay', 'WindowControlsOverlayGeometryChangeEvent', 'HIDConnectionEvent', 'HIDInputReportEvent']) delete G[k];
  if (G.MediaDevices) for (const k of ['getDisplayMedia', 'setCaptureHandleConfig']) delete MediaDevices.prototype[k];
  const N = Navigator.prototype;
  const receiverCheck = Object.getOwnPropertyDescriptor(N, 'userAgent').get; // this 不是 Navigator 时和原生一样抛错
  // navigator.standalone：Safari 独有，不是主屏幕网页应用时为 false
  if (!('standalone' in N)) {
    const get = Object.getOwnPropertyDescriptor({get standalone() { Reflect.apply(receiverCheck, this, []); return false; }}, 'standalone').get;
    masks.set(get, 'function get standalone() { [native code] }');
    Object.defineProperty(N, 'standalone', {get, enumerable: true, configurable: true});
  }
  // GestureEvent：Safari 的双指手势事件
  if (!G.GestureEvent) {
    const GE = function GestureEvent(type) {
      if (new.target === undefined) throw strip(new TypeError('calling GestureEvent constructor without new is invalid'), GE);
      return Reflect.construct(UIEvent, arguments, new.target);
    };
    Object.setPrototypeOf(GE.prototype, UIEvent.prototype);
    Object.setPrototypeOf(GE, UIEvent);
    fakeInterface(GE, 'GestureEvent', {});
  }
  // ApplePaySession：iPhone 都支持 Apple Pay（canMakePayments 不检查绑卡，为 true）；不安全的页面里调用会报错
  if (!G.ApplePaySession) {
    const insecure = () => new DOMException('Trying to call an ApplePaySession API from an insecure document.', 'InvalidAccessError');
    const AP = function ApplePaySession(version, request) {
      if (new.target === undefined) throw strip(new TypeError('calling ApplePaySession constructor without new is invalid'), AP);
      if (arguments.length < 2) throw strip(new TypeError('Not enough arguments'), AP);
      throw strip(G.isSecureContext ? new DOMException('Must create a new ApplePaySession from a user gesture handler.', 'InvalidAccessError') : insecure(), AP);
    };
    const statics = {
      canMakePayments() {
        if (!G.isSecureContext) throw strip(insecure(), statics.canMakePayments);
        return true;
      },
      canMakePaymentsWithActiveCard(merchantIdentifier) {
        return G.isSecureContext ? Promise.resolve(false) : Promise.reject(insecure());
      },
      supportsVersion(version) {
        if (!G.isSecureContext) throw strip(insecure(), statics.supportsVersion);
        return Number(version) >= 1 && Number(version) <= 14;
      },
    };
    for (const [k, fn] of Object.entries(statics)) {
      masks.set(fn, 'function ' + k + '() { [native code] }');
      Object.defineProperty(AP, k, {value: fn, writable: true, enumerable: true, configurable: true});
    }
    Object.setPrototypeOf(AP.prototype, EventTarget.prototype);
    Object.setPrototypeOf(AP, EventTarget);
    fakeInterface(AP, 'ApplePaySession', {});
  }
  // openDatabase：iPhone 上 WebSQL 的入口还在（两台真机都有）；不实现数据库，调用时报 SecurityError（Safari 禁用 WebSQL 时的表现）
  if (!('openDatabase' in G)) {
    const openDatabase = {openDatabase(name, version, displayName, estimatedSize) {
      if (arguments.length < 4) throw strip(new TypeError('Not enough arguments'), openDatabase);
      throw strip(new DOMException('The operation is insecure.', 'SecurityError'), openDatabase);
    }}.openDatabase;
    masks.set(openDatabase, 'function openDatabase() { [native code] }');
    Object.defineProperty(G, 'openDatabase', {value: openDatabase, writable: true, enumerable: true, configurable: true});
  }
}
```

- [ ] **Step 4: 运行，确认通过**

Run: `go test -run 'TestIOS|TestAndroid(SurfaceAPIs|FakeAPIShapes|IdentityAcrossSurfaces)' ./pkg/browser/`
Expected: PASS

- [ ] **Step 5: 提交**

```bash
git add pkg/browser/identity_script.go pkg/browser/ios_integration_test.go
git commit -m "feat(identity): iPhone Safari API surface for iOS identities"
```

---

### Task 4: iOS 输入（点按、按键字段、click 的 pointerType）

**Files:**
- Modify: `pkg/browser/mobile_input.go`
- Modify: `pkg/browser/cdp_custom.go`（传 `p.identity.OS`）
- Modify: `pkg/browser/identity_script.go`（`C.ios` 窗口段：click 的 `pointerType`）
- Test: `pkg/browser/mobile_input_test.go`、`pkg/browser/mobile_input_integration_test.go`（`baselineInteraction` 加文件参数）、`pkg/browser/ios_integration_test.go`

**Interfaces:**
- Consumes: Task 1 的 `OSIOS`。
- Produces: `iosTouchRadius = 12.079470187425613`；`touchEvent(typ string, x, y, radius, force float64) map[string]any`；`mobileTap(send cdpSend, os OSFamily, x, y float64) error`；`mobileSwipe(send, os, x0, y0, x1, y1)`；`mobileScroll(send, os, viewportW, viewportH int, deltaX, deltaY float64)`；`mobileType(send, os, text)`；`imeKey(send, os, typ string, r rune, ts float64)`；`enterKey(send, os)`；`backspaceKey(send, os)`；`composeStep(send, os, r rune, word []rune)`；`baselineInteraction(t, file string)`。

- [ ] **Step 1: 写失败的测试**

`mobile_input_test.go`：现有调用加上 `OSAndroid`（`mobileTap(send, OSAndroid, 100, 200)`、`mobileType(send, OSAndroid, …)`、`mobileScroll(send, OSAndroid, 400, 800, 0, 1000)`）。追加：

```go
// TestIOSTapSequence iOS 点按：按下后直接抬起，中间没有移动；触点半径 12.08、force 0（pointer 宽高 24.16、pressure 0），按住 25–60ms
func TestIOSTapSequence(t *testing.T) {
	send, log := recordCDP()
	start := time.Now()
	if err := mobileTap(send, OSIOS, 100, 200); err != nil {
		t.Fatal(err)
	}
	if d := time.Since(start); d < 25*time.Millisecond || d > 80*time.Millisecond {
		t.Errorf("tap took %v, want 25–60ms", d)
	}
	var types []string
	for _, c := range *log {
		types = append(types, c.Params["type"].(string))
	}
	if strings.Join(types, ",") != "touchStart,touchEnd" {
		t.Fatalf("touch types %v", types)
	}
	if p := touchPointOf((*log)[0]); p["radiusX"] != iosTouchRadius || p["radiusY"] != iosTouchRadius || p["force"] != 0.0 {
		t.Errorf("touch point %v", p)
	}
}

// TestIOSTypeKeys iOS 键盘：按键带真实的 key 与 code，keydown 的 keyCode 为 229、keyup 的为按键码；空格同样，回车的 code 为 Enter
func TestIOSTypeKeys(t *testing.T) {
	send, log := recordCDP()
	if err := mobileType(send, OSIOS, "aB1 \n"); err != nil {
		t.Fatal(err)
	}
	var got []string
	for _, c := range *log {
		if c.Method == "Input.dispatchKeyEvent" {
			got = append(got, fmt.Sprintf("%s:%s:%s:%v", c.Params["type"], c.Params["key"], c.Params["code"], c.Params["windowsVirtualKeyCode"]))
		}
	}
	want := "rawKeyDown:a:KeyA:229,keyUp:a:KeyA:65,rawKeyDown:B:KeyB:229,keyUp:B:KeyB:66,rawKeyDown:1:Digit1:229,keyUp:1:Digit1:49," +
		"rawKeyDown: :Space:229,keyUp: :Space:32,keyDown:Enter:Enter:13,keyUp:Enter:Enter:13"
	if strings.Join(got, ",") != want {
		t.Errorf("keys\n %s\nwant\n %s", strings.Join(got, ","), want)
	}
}
```

（`mobile_input_test.go` 的 import 加 `fmt`、`time`。）

`mobile_input_integration_test.go`：`baselineInteraction(t)` 改为 `baselineInteraction(t *testing.T, file string)`，读 `"../../test/baseline/" + file`；安卓测试传 `"android-chrome-pgkm10.json"`。

`ios_integration_test.go` 追加：

```go
// TestIOSTapAndTypeMatchBaseline 在采集页上点按钮、输入 ab1，与 iPhone（iOS 26）的事件序列与属性比较（Review Focus 5）
func TestIOSTapAndTypeMatchBaseline(t *testing.T) {
	srv, posted := baselineServer(t, "", http.StatusOK)
	p := connectIOS(t, "ios-input", "iphone-ios26").Page()
	sp := p.(*CustomCDPPage)
	collectOn(t, p, srv)
	if err := sp.ClickSelector("#tap-btn"); err != nil {
		t.Fatal(err)
	}
	if err := sp.SendKeys("#type-input", "ab1"); err != nil {
		t.Fatal(err)
	}
	time.Sleep(300 * time.Millisecond)
	if v, _ := p.Evaluate(`document.getElementById('type-input').value`); v != "ab1" {
		t.Errorf("input value %v, want ab1", v)
	}
	if _, err := p.Evaluate(`window.finishBaseline(), true`); err != nil {
		t.Fatal(err)
	}
	var body struct {
		Interaction struct {
			Events []map[string]any `json:"events"`
		} `json:"interaction"`
	}
	select {
	case raw := <-posted:
		if err := json.Unmarshal(raw, &body); err != nil {
			t.Fatal(err)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("nothing posted")
	}
	got, want := body.Interaction.Events, baselineInteraction(t, "ios-safari-iphone-2.json")
	if g, w := tapSequence(got, "tap"), tapSequence(want, "tap"); !slices.Equal(g, w) {
		t.Errorf("tap sequence\n %v\nwant\n %v", g, w)
	}
	down, up, click := firstOf(got, "tap", "pointerdown"), firstOf(got, "tap", "pointerup"), firstOf(got, "tap", "click")
	if down == nil || up == nil || click == nil {
		t.Fatalf("tap incomplete: down %v up %v click %v", down, up, click)
	}
	if down["pointerType"] != "touch" || down["pressure"] != 0.0 || math.Abs(down["width"].(float64)-24.16) > 0.01 || down["isPrimary"] != true {
		t.Errorf("pointerdown %v", down)
	}
	if up["pressure"] != 0.0 || click["pointerType"] != "mouse" || click["width"] != 1.0 {
		t.Errorf("pointerup %v / click %v", up, click)
	}
	// Safari 的屏幕坐标与页面坐标相同
	for _, e := range []map[string]any{down, up, click, firstOf(got, "tap", "mousedown")} {
		if e != nil && e["screenY"] != e["clientY"] {
			t.Errorf("%s screenY %v, clientY %v", e["type"], e["screenY"], e["clientY"])
		}
	}
	if hold := up["rt"].(float64) - down["rt"].(float64); hold < 20 || hold > 110 {
		t.Errorf("finger held %.0fms, want 20–110ms (iPhone about 28ms)", hold)
	}
	// 按键字段与真机一致：keydown 为 229，keyup 为按键码，二者共用时间戳
	var keys, updates []string
	for _, e := range got {
		if e["step"] != "type" {
			continue
		}
		switch e["type"] {
		case "keydown", "keyup":
			keys = append(keys, fmt.Sprintf("%v:%v:%v:%v", e["type"], e["key"], e["code"], e["keyCode"]))
		case "compositionupdate":
			if d := fmt.Sprint(e["data"]); len(updates) == 0 || updates[len(updates)-1] != d {
				updates = append(updates, d)
			}
		}
	}
	if w := "keydown:a:KeyA:229,keyup:a:KeyA:65,keydown:b:KeyB:229,keyup:b:KeyB:66,keydown:1:Digit1:229,keyup:1:Digit1:49"; strings.Join(keys, ",") != w {
		t.Errorf("keys\n %s\nwant\n %s", strings.Join(keys, ","), w)
	}
	if strings.Join(updates, ",") != "a,ab,ab1" {
		t.Errorf("composition updates %v, want a,ab,ab1", updates)
	}
}

// TestIOSEnterKey 回车是 iOS 键盘的 Enter 键（code Enter、keyCode 13），产生换行；打字之后照样能点按（Review Focus 5）
func TestIOSEnterKey(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { fmt.Fprint(w, enterPage) }))
	defer srv.Close()
	p := connectIOS(t, "ios-enter", "iphone-ios26").Page()
	sp := p.(*CustomCDPPage)
	if err := p.Navigate(srv.URL); err != nil {
		t.Fatal(err)
	}
	if err := sp.SendKeys("#i", "x\n"); err != nil {
		t.Fatal(err)
	}
	if err := sp.ClickSelector("#i"); err != nil {
		t.Fatal(err)
	}
	raw, _ := p.Evaluate(`window.__k.join(',')`)
	got := fmt.Sprint(raw)
	for _, want := range []string{"keydown|x|KeyX|229|", "keyup|x|KeyX|88|", "keydown|Enter|Enter|13|", "beforeinput|||0|insertLineBreak", "keyup|Enter|Enter|13|"} {
		if !strings.Contains(got, want) {
			t.Errorf("events %s lack %s", got, want)
		}
	}
	if v, _ := p.Evaluate(`document.getElementById('i').value`); v != "x" {
		t.Errorf("input value %v, want x", v)
	}
}

// TestIOSSwipeScrolls iOS 上 RealScroll 同样是手指拖动，滚动距离接近请求值；触点的屏幕坐标与页面坐标相同
func TestIOSSwipeScrolls(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { fmt.Fprint(w, swipePage) }))
	defer srv.Close()
	p := connectIOS(t, "ios-swipe", "iphone-ios26").Page()
	if err := p.Navigate(srv.URL); err != nil {
		t.Fatal(err)
	}
	if err := p.(*CustomCDPPage).RealScroll(0, 900); err != nil {
		t.Fatal(err)
	}
	time.Sleep(500 * time.Millisecond)
	if y, _ := p.Evaluate(`scrollY`); y.(float64) < 900-30 || y.(float64) > 900+30 {
		t.Errorf("scrollY = %v after RealScroll(0, 900), want 900 ± 30", y)
	}
	if off, _ := p.Evaluate(`window.__touchOff`); off != 0.0 {
		t.Errorf("Touch screenY - clientY = %v, want 0", off)
	}
}
```

（`ios_integration_test.go` 的 import 加 `math`、`slices`。）

- [ ] **Step 2: 运行，确认失败**

Run: `go test -run 'TestIOSTap|TestIOSTypeKeys|TestMobile|TestIOSEnterKey|TestIOSSwipeScrolls' ./pkg/browser/`
Expected: 编译失败（`mobileTap` 等参数个数不符、`iosTouchRadius` 未定义）

- [ ] **Step 3: 实现** `mobile_input.go`

import 加 `"github.com/chromedp/chromedp/kb"`。

```go
// iosTouchRadius iPhone 的触点半径（真机实测 12.08，pointer 宽高为它的两倍）
const iosTouchRadius = 12.079470187425613

// touchEvent Input.dispatchTouchEvent 的参数：单指触点；抬起时不带触点
func touchEvent(typ string, x, y, radius, force float64) map[string]any {
	points := []map[string]any{}
	if typ != "touchEnd" {
		points = append(points, map[string]any{"x": x, "y": y, "radiusX": radius, "radiusY": radius, "force": force, "id": 0})
	}
	return map[string]any{"type": typ, "touchPoints": points}
}

// finger 系统的手指触点：按下与移动时的半径、force。安卓接触面随按压变大、force 1；iOS 半径固定、force 0（pointer 的 pressure 为 0，真机实测）
func finger(os OSFamily) (down, move, force float64) {
	if os == OSIOS {
		return iosTouchRadius, iosTouchRadius, 0
	}
	return tapRadii[0], tapRadii[2], 1
}
```

`mobileTap(send cdpSend, os OSFamily, x, y float64) error`：在 `x, y = fingerJitter(x), fingerJitter(y)` 之后加：

```go
	// iOS：按下后直接抬起，中间没有移动（真机按住约 28ms，没有 pointermove）；从按下被确认时起算
	if os == OSIOS {
		if err := send("Input.dispatchTouchEvent", touchEvent("touchStart", x, y, iosTouchRadius, 0)); err != nil {
			return err
		}
		time.Sleep(between(25, 60))
		return send("Input.dispatchTouchEvent", touchEvent("touchEnd", x, y, 0, 0))
	}
```

安卓路径的 `touchEvent(typ, px, py, r)` 改为 `touchEvent(typ, px, py, r, 1)`，`touchEvent("touchEnd", x, y, 0)` 改为 `touchEvent("touchEnd", x, y, 0, 0)`；函数注释开头加"安卓"。

`mobileSwipe(send cdpSend, os OSFamily, x0, y0, x1, y1 float64) error`：开头 `downR, moveR, force := finger(os)`，`touchStart` 用 `touchEvent("touchStart", subPixel(x0), subPixel(y0), downR, force)`，`touchMove` 用 `moveR, force`，`touchEnd` 加 `, 0`。`mobileScroll(send cdpSend, os OSFamily, viewportW, viewportH int, deltaX, deltaY float64) error` 把 `os` 传给 `mobileSwipe`。

按键：

```go
// imeKey 组字按键；r 为按下的字符，ts 为事件时间（秒）。安卓输入法：key 为 Unidentified、code 为空、keyCode 为 229（真机实测）；
// iOS 键盘：key / code 为真实按键（a / KeyA），keydown 的 keyCode 为 229、keyup 的为按键码 65（真机实测），键盘上没有的字符同安卓。
// 不带 nativeVirtualKeyCode：带上 229 后 Chrome 之后的每个 Input.dispatchTouchEvent 都收不到确认（打完字就点不了）
func imeKey(send cdpSend, os OSFamily, typ string, r rune, ts float64) error {
	p := map[string]any{"type": typ, "key": "Unidentified", "code": "", "windowsVirtualKeyCode": 229, "timestamp": ts}
	if k, ok := kb.Keys[r]; ok && os == OSIOS {
		p["key"], p["code"] = k.Key, k.Code
		if typ == "keyUp" {
			p["windowsVirtualKeyCode"] = int(k.Windows)
		}
	}
	return send("Input.dispatchKeyEvent", p)
}

// functionCode 回车、退格这类功能键的 code：安卓输入法为空，iOS 键盘为按键名（Enter、Backspace）
func functionCode(os OSFamily, r rune) string {
	if os == OSIOS {
		return kb.Keys[r].Code
	}
	return ""
}
```

`enterKey(send cdpSend, os OSFamily)` 与 `backspaceKey(send cdpSend, os OSFamily)` 的 `"code": ""` 改为 `"code": functionCode(os, '\r')` / `functionCode(os, '\b')`，注释注明"安卓 code 为空（真机实测），iOS 为 Enter / Backspace"。`composeStep(send cdpSend, os OSFamily, r rune, word []rune)` 两次 `imeKey(send, "rawKeyDown", ts)` / `imeKey(send, "keyUp", ts)` 改为带 `os, …, r`。`mobileType(send cdpSend, os OSFamily, text string)`：`enterKey(send, os)`；退格分支 `backspaceKey(send, os)` 与 `composeStep(send, os, '\b', word)`；空格分支 `imeKey(send, os, "rawKeyDown", ' ', ts)` / `imeKey(send, os, "keyUp", ' ', ts)`；默认分支 `composeStep(send, os, r, word)`。

`cdp_custom.go`：`mobileTap(send, x, y)` → `mobileTap(send, p.identity.OS, x, y)`（两处）；`mobileScroll(send, s.InnerWidth, …)` → `mobileScroll(send, p.identity.OS, s.InnerWidth, …)`；`mobileType(send, text)` → `mobileType(send, p.identity.OS, text)`。

`identity_script.go` 的 `jsWindowPart` `if (C.ios) {` 段末尾加：

```js
  // 点按产生的 click 在 Safari 上 pointerType 为 mouse（Chrome 为 touch，真机实测）
  if (G.PointerEvent) patchGetter(PointerEvent.prototype, 'pointerType', (orig, self) => self.isTrusted && self.type === 'click' ? 'mouse' : Reflect.apply(orig, self, []));
```

- [ ] **Step 4: 运行，确认通过**

Run: `go test -run 'TestMobile|TestIOSTap|TestIOSTypeKeys|TestIOSEnterKey|TestIOSSwipeScrolls|TestAndroid(TapAndType|Swipe|Enter|TypeThenTap|ControlChars|ShortSwipe)' ./pkg/browser/`
Expected: PASS

- [ ] **Step 5: 提交**

```bash
git add pkg/browser/mobile_input.go pkg/browser/cdp_custom.go pkg/browser/identity_script.go pkg/browser/mobile_input_test.go pkg/browser/mobile_input_integration_test.go pkg/browser/ios_integration_test.go
git commit -m "feat(input): iPhone taps, keyboard key fields and mouse-typed clicks for iOS identities"
```

---

### Task 5: iOS 字体（用户自备的 Apple 字体）

**Files:**
- Modify: `pkg/browser/ios_identity.go`（`iosFontRoots`、`iosFontAliases`、`writeIOSFontConfig`）
- Modify: `pkg/browser/launcher.go`（iOS 用 Apple 字体配置，没有时警告并回落）
- Modify: `docs/deploy/fontconfig/55-windows-fonts-only.conf`、`docs/deploy/production.Dockerfile`、`docs/deploy/README.md`
- Test: `pkg/browser/ios_identity_test.go`、`pkg/browser/ios_integration_test.go`

**Interfaces:**
- Produces: `iosFontRoots []string`；`writeIOSFontConfig(dir string, roots []string) (string, error)`（没有任何 Apple 字体目录时返回空路径）。

- [ ] **Step 1: 写失败的测试**

`ios_identity_test.go`：

```go
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
```

（import 加 `path/filepath`。）

`ios_integration_test.go`：

```go
// TestIOSFonts Linux 上提供了 Apple 字体时，iOS 身份能按名字认出的字体与 iPhone 一致（Apple 字体受授权限制，由用户自备）
func TestIOSFonts(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("字体只在 Linux 上由本库控制（macOS 由 CoreText 管理）")
	}
	if conf, _ := writeIOSFontConfig(t.TempDir(), iosFontRoots); conf == "" {
		t.Skip("没有 Apple 字体（/usr/share/fonts/apple），iOS 身份回落到安卓字体")
	}
	srv, _ := baselineServer(t, "", http.StatusOK)
	got := collectOn(t, connectIOS(t, "ios-fonts", "iphone-ios26").Page(), srv)["fonts"].(map[string]any)
	for name, v := range loadIOSBaseline(t, "ios-safari-iphone-2.json")["fonts"].(map[string]any) {
		if got[name] != v {
			t.Errorf("font %q detected = %v, want %v", name, got[name], v)
		}
	}
}
```

（import 加 `runtime`。）

- [ ] **Step 2: 运行，确认失败**

Run: `go test -run 'TestIOSFont' ./pkg/browser/`
Expected: 编译失败（`writeIOSFontConfig`、`iosFontRoots` 未定义）

- [ ] **Step 3: 实现**

`ios_identity.go` import 加 `"os"`、`"path/filepath"`，追加：

```go
// iosFontRoots 用户自备的 Apple 字体所在目录（受授权限制，不随库分发，见 docs/deploy/README.md）
var iosFontRoots = []string{"/usr/share/fonts/apple", "/usr/local/share/fonts/apple"}

// iosFontAliases iPhone Safari 的通用族：西文为 Helvetica / Times New Roman / Courier New，中文回落到苹方，表情为 Apple Color Emoji
var iosFontAliases = []struct {
	Family string
	Prefer []string
}{
	{"sans-serif", []string{"Helvetica", "PingFang SC"}},
	{"serif", []string{"Times New Roman", "PingFang SC"}},
	{"monospace", []string{"Courier New", "PingFang SC"}},
}

// writeIOSFontConfig 在 dir 下生成 iOS 身份的 fontconfig 配置：只包含 roots 里存在且非空的 Apple 字体目录，字体保持原名
// （iPhone 上 PingFang SC、Helvetica 等都能按名探测到），再配通用族；不包含系统配置。没有任何 Apple 字体时返回空路径
func writeIOSFontConfig(dir string, roots []string) (string, error) {
	var b strings.Builder
	b.WriteString("<?xml version=\"1.0\"?>\n<!DOCTYPE fontconfig SYSTEM \"urn:fontconfig:fonts.dtd\">\n<fontconfig>\n")
	found := false
	for _, r := range roots {
		if entries, err := os.ReadDir(r); err == nil && len(entries) > 0 {
			fmt.Fprintf(&b, "  <dir>%s</dir>\n", r)
			found = true
		}
	}
	if !found {
		return "", nil
	}
	fmt.Fprintf(&b, "  <cachedir>%s</cachedir>\n", filepath.Join(dir, "cache"))
	for _, a := range iosFontAliases {
		fmt.Fprintf(&b, "  <alias binding=\"same\"><family>%s</family><prefer>", a.Family)
		for _, f := range a.Prefer {
			fmt.Fprintf(&b, "<family>%s</family>", f)
		}
		b.WriteString("<family>Apple Color Emoji</family></prefer></alias>\n")
	}
	b.WriteString("</fontconfig>\n")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", fmt.Errorf("create %s: %w", dir, err)
	}
	conf := filepath.Join(dir, "fonts.conf")
	if err := os.WriteFile(conf, []byte(b.String()), 0o644); err != nil {
		return "", fmt.Errorf("write %s: %w", conf, err)
	}
	return conf, nil
}
```

`launcher.go` 字体段：

```go
	// 手机身份（Linux）：进程级字体配置。iOS 只看得到用户自备的 Apple 字体，没有时警告并回落到安卓字体；安卓只看得到安卓字体；桌面身份仍用系统配置
	fontDir, fontEnv := "", []string(nil)
	if mobile && runtime.GOOS == "linux" {
		dir, err := os.MkdirTemp("", "prbg-fonts-")
		// ……创建失败的处理不变，错误前缀改为 "mobile fonts: "……
		conf := ""
		if osf == OSIOS {
			if conf, err = writeIOSFontConfig(dir, iosFontRoots); err == nil && conf == "" {
				iosFontsWarning.Do(func() {
					fmt.Printf("Warning: no Apple fonts in %s: iOS identities fall back to Android fonts (font probing will not look like an iPhone)\n", strings.Join(iosFontRoots, ", "))
				})
			}
		}
		if err == nil && conf == "" {
			var missing []string
			conf, missing, err = writeAndroidFontConfig(dir, androidFontRoots)
			if err == nil && len(missing) > 0 {
				androidFontsWarning.Do(func() { /* 原警告不变 */ })
			}
		}
		if err != nil {
			os.RemoveAll(dir)
			if tempDir != "" {
				os.RemoveAll(tempDir)
			}
			return nil, fmt.Errorf("mobile fonts: %w", err)
		}
		fontDir, fontEnv = dir, append(os.Environ(), "FONTCONFIG_FILE="+conf)
	}
```

文件末尾：

```go
// iosFontsWarning 没有 Apple 字体只警告一次
var iosFontsWarning sync.Once
```

`55-windows-fonts-only.conf` 的 `<rejectfont>` 加 `<glob>/usr/share/fonts/apple/*</glob>`、`<glob>/usr/local/share/fonts/apple/*</glob>`，注释第 1 条补"iOS 身份用的 Apple 字体（用户自备）也屏蔽"。`production.Dockerfile` 在 Windows 字体 COPY 之后加注释行：

```dockerfile
# iOS 账号（可选）：自备合法授权的 Apple 字体（苹方、Helvetica 等，从自己的 Mac 拷贝），放在构建上下文的 fonts/apple/ 下再取消注释；
# 只有 iOS 身份的进程看得到它们（55-windows-fonts-only.conf 已对桌面身份屏蔽）
# COPY fonts/apple/ /usr/share/fonts/apple/
```

并把构建检查的 `grep -iqE 'noto|roboto'` 改为 `grep -iqE 'noto|roboto|pingfang'`（提示文字补"/ 苹方"）。`docs/deploy/README.md` 手机身份字体一段后加 iOS 字体说明：放置位置、授权要求、没有时的回落与警告、检查命令 `docker run --rm <镜像> ls /usr/share/fonts/apple`。

- [ ] **Step 4: 运行，确认通过**

Run: `go test -run 'TestIOSFont|TestAndroidFont' ./pkg/browser/`
Expected: PASS（macOS 上 `TestIOSFonts` 跳过）

- [ ] **Step 5: 提交**

```bash
git add pkg/browser/ios_identity.go pkg/browser/ios_identity_test.go pkg/browser/launcher.go pkg/browser/ios_integration_test.go docs/deploy/
git commit -m "feat(fonts): iOS identities see only user-supplied Apple fonts on Linux"
```

---

### Task 6: 对照真机基准与文档

**Files:**
- Test: `pkg/browser/ios_integration_test.go`
- Modify: `README.md`、`CLAUDE.md`、`pkg/browser/types.go`（`Mobile` 注释）、`docs/superpowers/specs/2026-10-17-ios-identity-design.md`（按实测更正）

- [ ] **Step 1: 写测试**

```go
// iosResidual 已知无法模拟、列为残余风险的项目（规格第 6 节）
var iosResidual = map[string]string{
	"engine.nullProperty":              "JavaScriptCore 与 V8 的报错文本不同",
	"engine.callUndefined":             "JavaScriptCore 与 V8 的报错文本不同",
	"engine.jsonParse":                 "JavaScriptCore 与 V8 的报错文本不同",
	"engine.stackSample":               "JavaScriptCore 与 V8 的调用栈格式不同",
	"features.notificationConstructor": "ReferenceError 的文本由引擎决定",
	"media.dynamic-range":              "宿主屏幕不支持高动态范围，CDP 不能模拟",
	"avmedia.codecs":                   "编解码支持沿用宿主（谎报会让网站推来放不了的视频）",
}

// iosNotCompared 不属于身份、不比较的项目：用户设置、设备上的语音与媒体设备
var iosNotCompared = []string{"media.prefers-color-scheme", "avmedia.voices", "avmedia.devices"}

// TestIOSMatchesBaseline 用 iOS 26 机型跑采集页，与 iPhone 真机基准逐项比较；只跳过残余风险与不属于身份的项目，每一项都写明原因
func TestIOSMatchesBaseline(t *testing.T) {
	srv, _ := baselineServer(t, "", http.StatusOK)
	got := collectOn(t, connectIOS(t, "ios-baseline", "iphone-ios26").Page(), srv)
	_, appleFonts := os.Stat(iosFontRoots[0])
	skip := func(path string) bool {
		path = strings.TrimPrefix(path, ".")
		for p := range iosResidual {
			if path == p || strings.HasPrefix(path, p+".") {
				return true
			}
		}
		for _, p := range iosNotCompared {
			if path == p || strings.HasPrefix(path, p+".") {
				return true
			}
		}
		return path == "fonts" && (runtime.GOOS != "linux" || appleFonts != nil) // 字体只在 Linux 上、提供了 Apple 字体时比较
	}
	diffs := diffBaseline(loadIOSBaseline(t, "ios-safari-iphone-2.json"), got, "", skip)
	sort.Strings(diffs)
	for _, d := range diffs {
		t.Error(d)
	}
}
```

（import 加 `os`、`sort`。）

- [ ] **Step 2: 运行**

Run: `go test -run TestIOSMatchesBaseline ./pkg/browser/`
Expected: PASS。出现上表之外的差异时：能修的回到对应任务修（TDD），确实无法模拟的加进 `iosResidual` 并写明原因，每一处都记入台账的 Ruling。

- [ ] **Step 3: 文档**

- `README.md` 移动身份一节加 iOS：开启方式（`Mobile: true` + iPhone UA）、两款机型、能做到与做不到（JS 引擎、TLS、渲染引擎、Apple 字体自备）。
- `CLAUDE.md` Identity 段：`ios_identity.go`（机型档案、`normalizeIOS`、`iosIdentity`、Apple 字体配置），`C.ios` 注入段，`mobile_input.go` 按 `Identity.OS` 区分安卓与 iOS。
- `types.go` `Mobile` 注释：去掉"见子项目 M3"，写明 iPhone / iPad UA 生成 iOS（iPhone）身份。
- 规格：`openDatabase` 两台真机都有（规格原写"26 没有"）；`ManagedMediaSource` 的编解码沿用宿主；`ApplePaySession.canMakePayments()` 为 true；残余风险补上 `pointerId` 取值范围、`Accept-Language` 只有主语言（真机格式未采集）、ManagedSourceBuffer。

- [ ] **Step 4: 全量**

Run: `go build ./pkg/... ./internal/... && go vet ./pkg/... ./internal/... && go test -race -count=1 ./cmd/baseline_server ./pkg/... ./internal/...`，再跑 `scripts/test-linux-chrome.sh`
Expected: 两边全部通过

- [ ] **Step 5: 提交**

```bash
git add pkg/browser/ios_integration_test.go README.md CLAUDE.md pkg/browser/types.go docs/superpowers/specs/2026-10-17-ios-identity-design.md
git commit -m "test(identity): iOS identity matches the iPhone baseline; docs"
```
