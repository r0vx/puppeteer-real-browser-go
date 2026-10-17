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

// appleGPU iPhone 的 Apple GPU 在 Safari 上的实测值（test/baseline/ios-safari-*.json）：两台 iPhone 只有 MAX_SAMPLES 不同（iOS 26 为 8，18.5 为 4）。
// Safari 的 VERSION / SHADING_LANGUAGE_VERSION 没有 Chromium 后缀，一并覆盖
func appleGPU(maxSamples int) gpuProfile {
	params2 := map[int]any{
		32883: 2048,                 // MAX_3D_TEXTURE_SIZE
		35071: 2048,                 // MAX_ARRAY_TEXTURE_LAYERS
		34852: 8,                    // MAX_DRAW_BUFFERS
		36063: 8,                    // MAX_COLOR_ATTACHMENTS
		35376: 16384,                // MAX_UNIFORM_BLOCK_SIZE
		35375: 24,                   // MAX_UNIFORM_BUFFER_BINDINGS
		35374: 24,                   // MAX_COMBINED_UNIFORM_BLOCKS
		35658: 4096,                 // MAX_VERTEX_UNIFORM_COMPONENTS
		35657: 4096,                 // MAX_FRAGMENT_UNIFORM_COMPONENTS
		35659: 124,                  // MAX_VARYING_COMPONENTS
		35968: 4,                    // MAX_TRANSFORM_FEEDBACK_SEPARATE_COMPONENTS
		36203: 4294967294,           // MAX_ELEMENT_INDEX
		37137: 0,                    // MAX_SERVER_WAIT_TIMEOUT
		34045: 15,                   // MAX_TEXTURE_LOD_BIAS
		37154: 124,                  // MAX_VERTEX_OUTPUT_COMPONENTS
		37157: 124,                  // MAX_FRAGMENT_INPUT_COMPONENTS
		35077: 7,                    // MAX_PROGRAM_TEXEL_OFFSET
		35076: -8,                   // MIN_PROGRAM_TEXEL_OFFSET
		36183: maxSamples,           // MAX_SAMPLES
		7938:  "WebGL 2.0",          // VERSION
		35724: "WebGL GLSL ES 3.00", // SHADING_LANGUAGE_VERSION
	}
	return gpuProfile{
		Vendor: "Apple Inc.", Renderer: "Apple GPU",
		Mobile: &mobileWebGL{
			Params: map[int]any{
				3379:  16384,                     // MAX_TEXTURE_SIZE
				34076: 16384,                     // MAX_CUBE_MAP_TEXTURE_SIZE
				34024: 16384,                     // MAX_RENDERBUFFER_SIZE
				3386:  []float64{16384, 16384},   // MAX_VIEWPORT_DIMS
				34921: 16,                        // MAX_VERTEX_ATTRIBS
				36347: 1024,                      // MAX_VERTEX_UNIFORM_VECTORS
				36349: 1024,                      // MAX_FRAGMENT_UNIFORM_VECTORS
				36348: 31,                        // MAX_VARYING_VECTORS
				34930: 16,                        // MAX_TEXTURE_IMAGE_UNITS
				35660: 16,                        // MAX_VERTEX_TEXTURE_IMAGE_UNITS
				35661: 32,                        // MAX_COMBINED_TEXTURE_IMAGE_UNITS
				33901: []float64{1, 511},         // ALIASED_POINT_SIZE_RANGE
				33902: []float64{1, 1},           // ALIASED_LINE_WIDTH_RANGE
				3408:  4,                         // SUBPIXEL_BITS
				7938:  "WebGL 1.0",               // VERSION
				35724: "WebGL GLSL ES 1.0 (1.0)", // SHADING_LANGUAGE_VERSION
			},
			Params2:     params2,
			Extensions:  []string{"ANGLE_instanced_arrays", "EXT_blend_minmax", "EXT_clip_control", "EXT_color_buffer_half_float", "EXT_depth_clamp", "EXT_float_blend", "EXT_frag_depth", "EXT_polygon_offset_clamp", "EXT_shader_texture_lod", "EXT_texture_compression_bptc", "EXT_texture_compression_rgtc", "EXT_texture_filter_anisotropic", "EXT_texture_mirror_clamp_to_edge", "EXT_sRGB", "KHR_parallel_shader_compile", "OES_element_index_uint", "OES_fbo_render_mipmap", "OES_standard_derivatives", "OES_texture_float", "OES_texture_float_linear", "OES_texture_half_float", "OES_texture_half_float_linear", "OES_vertex_array_object", "WEBGL_blend_func_extended", "WEBGL_color_buffer_float", "WEBGL_compressed_texture_astc", "WEBGL_compressed_texture_etc", "WEBGL_compressed_texture_etc1", "WEBGL_compressed_texture_pvrtc", "WEBKIT_WEBGL_compressed_texture_pvrtc", "WEBGL_compressed_texture_s3tc", "WEBGL_compressed_texture_s3tc_srgb", "WEBGL_debug_renderer_info", "WEBGL_debug_shaders", "WEBGL_depth_texture", "WEBGL_draw_buffers", "WEBGL_lose_context", "WEBGL_multi_draw", "WEBGL_polygon_mode"},
			Extensions2: []string{"EXT_clip_control", "EXT_color_buffer_float", "EXT_color_buffer_half_float", "EXT_conservative_depth", "EXT_depth_clamp", "EXT_float_blend", "EXT_polygon_offset_clamp", "EXT_render_snorm", "EXT_texture_compression_bptc", "EXT_texture_compression_rgtc", "EXT_texture_filter_anisotropic", "EXT_texture_mirror_clamp_to_edge", "EXT_texture_norm16", "KHR_parallel_shader_compile", "NV_shader_noperspective_interpolation", "OES_draw_buffers_indexed", "OES_sample_variables", "OES_shader_multisample_interpolation", "OES_texture_float_linear", "WEBGL_blend_func_extended", "WEBGL_clip_cull_distance", "WEBGL_compressed_texture_astc", "WEBGL_compressed_texture_etc", "WEBGL_compressed_texture_etc1", "WEBGL_compressed_texture_pvrtc", "WEBKIT_WEBGL_compressed_texture_pvrtc", "WEBGL_compressed_texture_s3tc", "WEBGL_compressed_texture_s3tc_srgb", "WEBGL_debug_renderer_info", "WEBGL_debug_shaders", "WEBGL_lose_context", "WEBGL_multi_draw", "WEBGL_polygon_mode", "WEBGL_provoking_vertex", "WEBGL_render_shared_exponent", "WEBGL_stencil_texturing"},
			Precision: map[[2]int][3]int{
				{35633, 36336}: {127, 127, 23}, // VERTEX_SHADER.LOW_FLOAT
				{35633, 36337}: {127, 127, 23}, // VERTEX_SHADER.MEDIUM_FLOAT
				{35633, 36338}: {127, 127, 23}, // VERTEX_SHADER.HIGH_FLOAT
				{35633, 36339}: {31, 30, 0},    // VERTEX_SHADER.LOW_INT
				{35633, 36340}: {31, 30, 0},    // VERTEX_SHADER.MEDIUM_INT
				{35633, 36341}: {31, 30, 0},    // VERTEX_SHADER.HIGH_INT
				{35632, 36336}: {127, 127, 23}, // FRAGMENT_SHADER.LOW_FLOAT
				{35632, 36337}: {127, 127, 23}, // FRAGMENT_SHADER.MEDIUM_FLOAT
				{35632, 36338}: {127, 127, 23}, // FRAGMENT_SHADER.HIGH_FLOAT
				{35632, 36339}: {31, 30, 0},    // FRAGMENT_SHADER.LOW_INT
				{35632, 36340}: {31, 30, 0},    // FRAGMENT_SHADER.MEDIUM_INT
				{35632, 36341}: {31, 30, 0},    // FRAGMENT_SHADER.HIGH_INT
			},
		},
	}
}
