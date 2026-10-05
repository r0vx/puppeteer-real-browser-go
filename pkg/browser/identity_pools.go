package browser

import (
	"hash/fnv"
	"strings"
)

// weighted 带权重的取值
type weighted[T any] struct {
	Value  T
	Weight int
}

// pickWeighted 按种子确定性地加权抽取：同一种子总是抽到同一项
func pickWeighted[T any](items []weighted[T], seed uint64) T {
	total := 0
	for _, it := range items {
		total += it.Weight
	}
	r := int(seed % uint64(total))
	for _, it := range items {
		if r < it.Weight {
			return it.Value
		}
		r -= it.Weight
	}
	return items[len(items)-1].Value
}

// identitySeed 由用户 ID 和用途派生种子：同一账号每次抽到相同的值，不同用途互不相关
func identitySeed(userID, purpose string) uint64 {
	h := fnv.New64a()
	h.Write([]byte(userID + "\x00" + purpose))
	return h.Sum64()
}

// screenSpec 屏幕规格（CSS 像素）；Notch 表示带刘海的 MacBook（菜单栏更高）
type screenSpec struct {
	Width, Height int
	DPR           float64
	Notch         bool
}

// windowsScreens Windows 常见屏幕（含 125% / 150% 缩放后的 CSS 尺寸）
var windowsScreens = []weighted[screenSpec]{
	{screenSpec{1920, 1080, 1, false}, 35},
	{screenSpec{1536, 864, 1.25, false}, 15},
	{screenSpec{1366, 768, 1, false}, 12},
	{screenSpec{2560, 1440, 1, false}, 10},
	{screenSpec{1280, 720, 1.5, false}, 6},
	{screenSpec{1440, 900, 1, false}, 6},
	{screenSpec{1600, 900, 1, false}, 6},
	{screenSpec{1920, 1200, 1, false}, 5},
	{screenSpec{2048, 1152, 1.25, false}, 5},
}

// macScreens Mac 常见屏幕
var macScreens = []weighted[screenSpec]{
	{screenSpec{1440, 900, 2, false}, 15},  // MacBook Air 13（M1）
	{screenSpec{1470, 956, 2, true}, 20},   // MacBook Air 13（M2 / M3）
	{screenSpec{1512, 982, 2, true}, 20},   // MacBook Pro 14
	{screenSpec{1710, 1112, 2, true}, 10},  // MacBook Air 15
	{screenSpec{1728, 1117, 2, true}, 10},  // MacBook Pro 16
	{screenSpec{2240, 1260, 2, false}, 10}, // iMac 24
	{screenSpec{1920, 1080, 1, false}, 15}, // 外接显示器
}

// webglCaps 显卡在 Chrome（ANGLE）下暴露的 WebGL 能力参数
type webglCaps struct {
	MaxTextureSize, MaxCubeMapTextureSize, MaxRenderbufferSize int
	MaxViewportDims                                            [2]int
	MaxVertexAttribs, MaxVertexUniformVectors                  int
	MaxFragmentUniformVectors, MaxVaryingVectors               int
	MaxTextureImageUnits, MaxVertexTextureImageUnits           int
	MaxCombinedTextureImageUnits                               int
	AliasedPointSizeRange, AliasedLineWidthRange               [2]float64
}

// gpuProfile 一款显卡的 WebGL 标识与能力参数；Caps 为 nil 时只改写标识
type gpuProfile struct {
	Vendor   string // UNMASKED_VENDOR_WEBGL
	Renderer string // UNMASKED_RENDERER_WEBGL
	Caps     *webglCaps
}

// d3d11Caps Windows 上 ANGLE D3D11 后端对各厂商显卡统一暴露的能力参数
var d3d11Caps = &webglCaps{
	MaxTextureSize: 16384, MaxCubeMapTextureSize: 16384, MaxRenderbufferSize: 16384,
	MaxViewportDims: [2]int{32767, 32767}, MaxVertexAttribs: 16, MaxVertexUniformVectors: 4096,
	MaxFragmentUniformVectors: 1024, MaxVaryingVectors: 30, MaxTextureImageUnits: 16,
	MaxVertexTextureImageUnits: 16, MaxCombinedTextureImageUnits: 32,
	AliasedPointSizeRange: [2]float64{1, 1024}, AliasedLineWidthRange: [2]float64{1, 1},
}

// appleCaps Apple 芯片（ANGLE Metal 后端）实测值：Apple M2 Max，Chrome 154
var appleCaps = &webglCaps{
	MaxTextureSize: 16384, MaxCubeMapTextureSize: 16384, MaxRenderbufferSize: 16384,
	MaxViewportDims: [2]int{16384, 16384}, MaxVertexAttribs: 16, MaxVertexUniformVectors: 1024,
	MaxFragmentUniformVectors: 1024, MaxVaryingVectors: 30, MaxTextureImageUnits: 16,
	MaxVertexTextureImageUnits: 16, MaxCombinedTextureImageUnits: 32,
	AliasedPointSizeRange: [2]float64{1, 511}, AliasedLineWidthRange: [2]float64{1, 1},
}

// windowsGPUs Windows 常见显卡（renderer 串含 PCI 设备 ID，与真实 Chrome 格式一致）
var windowsGPUs = []weighted[gpuProfile]{
	{gpuProfile{"Google Inc. (Intel)", "ANGLE (Intel, Intel(R) UHD Graphics 620 (0x00005917) Direct3D11 vs_5_0 ps_5_0, D3D11)", d3d11Caps}, 20},
	{gpuProfile{"Google Inc. (Intel)", "ANGLE (Intel, Intel(R) UHD Graphics 630 (0x00003E92) Direct3D11 vs_5_0 ps_5_0, D3D11)", d3d11Caps}, 15},
	{gpuProfile{"Google Inc. (Intel)", "ANGLE (Intel, Intel(R) Iris(R) Xe Graphics (0x00009A49) Direct3D11 vs_5_0 ps_5_0, D3D11)", d3d11Caps}, 20},
	{gpuProfile{"Google Inc. (NVIDIA)", "ANGLE (NVIDIA, NVIDIA GeForce GTX 1650 (0x00001F82) Direct3D11 vs_5_0 ps_5_0, D3D11)", d3d11Caps}, 12},
	{gpuProfile{"Google Inc. (NVIDIA)", "ANGLE (NVIDIA, NVIDIA GeForce RTX 3060 (0x00002503) Direct3D11 vs_5_0 ps_5_0, D3D11)", d3d11Caps}, 13},
	{gpuProfile{"Google Inc. (AMD)", "ANGLE (AMD, AMD Radeon(TM) Graphics (0x00001638) Direct3D11 vs_5_0 ps_5_0, D3D11)", d3d11Caps}, 20},
}

// macGPUs Apple 芯片 Mac
var macGPUs = []weighted[gpuProfile]{
	{gpuProfile{"Google Inc. (Apple)", "ANGLE (Apple, ANGLE Metal Renderer: Apple M1, Unspecified Version)", appleCaps}, 30},
	{gpuProfile{"Google Inc. (Apple)", "ANGLE (Apple, ANGLE Metal Renderer: Apple M2, Unspecified Version)", appleCaps}, 30},
	{gpuProfile{"Google Inc. (Apple)", "ANGLE (Apple, ANGLE Metal Renderer: Apple M3, Unspecified Version)", appleCaps}, 20},
	{gpuProfile{"Google Inc. (Apple)", "ANGLE (Apple, ANGLE Metal Renderer: Apple M1 Pro, Unspecified Version)", appleCaps}, 10},
	{gpuProfile{"Google Inc. (Apple)", "ANGLE (Apple, ANGLE Metal Renderer: Apple M2 Pro, Unspecified Version)", appleCaps}, 10},
}

// linuxHostGPU 不设指纹的默认路径在 Linux 服务器上遮挡 SwiftShader 用的标识（只改标识，不改能力参数）
var linuxHostGPU = gpuProfile{Vendor: "Google Inc. (Intel)", Renderer: "ANGLE (Intel, Mesa Intel(R) UHD Graphics 620 (KBL GT2), OpenGL 4.6)"}

var (
	windowsCores = []weighted[int]{{4, 25}, {6, 15}, {8, 30}, {12, 15}, {16, 12}, {20, 3}}
	macCores     = []weighted[int]{{8, 60}, {10, 25}, {12, 15}}
	// 只报 Win10：生产镜像的字体取自 Windows 10，网站能按字体推断系统版本（Win11 才有 Segoe Fluent Icons），
	// 报 Win11 会与字体矛盾。有了正版 Win11 字体后，再按账号分开字体集合，恢复 15.0.0（22H2–23H2）/ 19.0.0（24H2）
	windowsPlatformVersions = []weighted[string]{{"10.0.0", 1}}
	macPlatformVersions     = []weighted[string]{{"14.7.6", 30}, {"15.5.0", 35}, {"15.6.1", 35}}
)

// screensFor 返回系统对应的屏幕池
func screensFor(os OSFamily) []weighted[screenSpec] {
	if os == OSMac {
		return macScreens
	}
	return windowsScreens
}

// gpusFor 返回系统对应的显卡池
func gpusFor(os OSFamily) []weighted[gpuProfile] {
	if os == OSMac {
		return macGPUs
	}
	return windowsGPUs
}

// coresFor 返回系统对应的 CPU 核数池
func coresFor(os OSFamily) []weighted[int] {
	if os == OSMac {
		return macCores
	}
	return windowsCores
}

// platformVersionsFor 返回系统对应的 client hints 系统版本池
func platformVersionsFor(os OSFamily) []weighted[string] {
	if os == OSMac {
		return macPlatformVersions
	}
	return windowsPlatformVersions
}

// lookupGPU 在系统的显卡池里按 renderer 精确查找
func lookupGPU(os OSFamily, renderer string) (gpuProfile, bool) {
	for _, g := range gpusFor(os) {
		if g.Value.Renderer == renderer {
			return g.Value, true
		}
	}
	return gpuProfile{}, false
}

// gpuFamily 从 vendor / renderer 串识别显卡厂商，识别不了返回空
func gpuFamily(vendor, renderer string) string {
	s := vendor + " " + renderer
	for _, f := range []string{"NVIDIA", "AMD", "Intel", "Apple"} {
		if strings.Contains(s, f) {
			return f
		}
	}
	return ""
}

// pickGPU 抽取显卡；family 非空且池中有同厂商时只在同厂商里抽（尽量保持账号原来的显卡厂商）
func pickGPU(os OSFamily, family string, seed uint64) gpuProfile {
	pool := gpusFor(os)
	var same []weighted[gpuProfile]
	for _, g := range pool {
		if family != "" && gpuFamily(g.Value.Vendor, g.Value.Renderer) == family {
			same = append(same, g)
		}
	}
	if len(same) > 0 {
		pool = same
	}
	return pickWeighted(pool, seed)
}

// screenSpecFor 在系统屏幕池里查找规格（带上刘海标记）；找不到时返回给定尺寸、不带刘海
func screenSpecFor(os OSFamily, w, h int, dpr float64) (screenSpec, bool) {
	for _, s := range screensFor(os) {
		if s.Value.Width == w && s.Value.Height == h && s.Value.DPR == dpr {
			return s.Value, true
		}
	}
	return screenSpec{Width: w, Height: h, DPR: dpr}, false
}
