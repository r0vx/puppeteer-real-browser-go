package browser

import (
	"fmt"
	"strings"
)

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
	// androidViewportTop 视口上方的状态栏与地址栏高度：触摸、指针、鼠标事件的 screenY − clientY（真机实测 88）
	androidViewportTop = 88
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
			3379:  4096,                    // MAX_TEXTURE_SIZE
			34076: 4096,                    // MAX_CUBE_MAP_TEXTURE_SIZE
			34024: 16383,                   // MAX_RENDERBUFFER_SIZE
			3386:  []float64{16383, 16383}, // MAX_VIEWPORT_DIMS
			34921: 32,                      // MAX_VERTEX_ATTRIBS
			36347: 4096,                    // MAX_VERTEX_UNIFORM_VECTORS
			36349: 4096,                    // MAX_FRAGMENT_UNIFORM_VECTORS
			36348: 31,                      // MAX_VARYING_VECTORS
			34930: 16,                      // MAX_TEXTURE_IMAGE_UNITS
			35660: 16,                      // MAX_VERTEX_TEXTURE_IMAGE_UNITS
			35661: 96,                      // MAX_COMBINED_TEXTURE_IMAGE_UNITS
			33901: []float64{1, 1024},      // ALIASED_POINT_SIZE_RANGE
			33902: []float64{1, 4095.9375}, // ALIASED_LINE_WIDTH_RANGE
			3408:  8,                       // SUBPIXEL_BITS
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
		OuterWidth: s.Width, OuterHeight: inner, InnerWidth: s.Width, InnerHeight: inner, Mobile: true, ViewportTop: androidViewportTop,
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

// appendSwitchValue 给逗号分隔的开关（--enable-features=、--enable-blink-features=）追加一项：
// 合并进最后一个同名开关（Chrome 只认最后一个），没有时新加
func appendSwitchValue(flags []string, prefix, value string) []string {
	for i := len(flags) - 1; i >= 0; i-- {
		if strings.HasPrefix(flags[i], prefix) {
			out := append([]string(nil), flags...)
			out[i] = flags[i] + "," + value
			return out
		}
	}
	return append(flags, prefix+value)
}

// knownDeviceKind 本版本认识的移动设备类型；更新的版本写的类型不能按桌面规范化回写，否则会毁掉那份指纹
func knownDeviceKind(kind string) bool {
	return kind == deviceKindAndroid
}

// deviceMismatch 账号指纹的设备类型与 ConnectOptions.Mobile 不一致时的错误（同一账号 ID 只能是一种设备）
func deviceMismatch(userID string, cfg *FingerprintConfig, mobile bool) error {
	if isMobile := cfg.Device != nil; isMobile != mobile {
		kind := "desktop"
		if isMobile {
			kind = "mobile"
		}
		return fmt.Errorf("fingerprint %s is a %s identity but ConnectOptions.Mobile is %v", userID, kind, mobile)
	}
	return nil
}
