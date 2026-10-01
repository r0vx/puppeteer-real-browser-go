package browser

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"hash/fnv"
	"regexp"
	"strings"
	_ "time/tzdata" // 时区校验不依赖服务器是否安装 tzdata
)

// OSFamily 账号对外声称的操作系统
type OSFamily string

const (
	OSWindows OSFamily = "windows"
	OSMac     OSFamily = "macos"
)

// browserUIHeight 浏览器标签栏 + 工具栏高度（outerHeight - innerHeight，Chrome 154 在 macOS / Linux 有界面实测）
const browserUIHeight = 87

// osFromUserAgent 由账号绑定的 UA 判定系统：含 Macintosh 为 macOS，其余（含旧数据里的 Linux）按 Windows
func osFromUserAgent(ua string) OSFamily {
	if strings.Contains(ua, "Macintosh") {
		return OSMac
	}
	return OSWindows
}

// platformFor 返回系统对应的 navigator.platform
func platformFor(os OSFamily) string {
	if os == OSMac {
		return "MacIntel"
	}
	return "Win32"
}

// canonicalUserAgent 返回 Chrome 精简格式的 UA：系统段为 Chrome 冻结值，版本只保留主版本
func canonicalUserAgent(os OSFamily, major string) string {
	sys := "Windows NT 10.0; Win64; x64"
	if os == OSMac {
		sys = "Macintosh; Intel Mac OS X 10_15_7"
	}
	return "Mozilla/5.0 (" + sys + ") AppleWebKit/537.36 (KHTML, like Gecko) Chrome/" + major + ".0.0.0 Safari/537.36"
}

var chromeMajorRe = regexp.MustCompile(`Chrome/(\d+)`)

// uaMajorVersion 取 UA 中 Chrome/ 后的主版本号，取不到返回空
func uaMajorVersion(ua string) string {
	if m := chromeMajorRe.FindStringSubmatch(ua); m != nil {
		return m[1]
	}
	return ""
}

// uaBrand client hints 中的一个品牌
type uaBrand struct {
	Brand   string `json:"brand"`
	Version string `json:"version"`
}

// uaMetadata 对应 Emulation.setUserAgentOverride 的 userAgentMetadata
type uaMetadata struct {
	Brands          []uaBrand `json:"brands"`
	FullVersionList []uaBrand `json:"fullVersionList"`
	Platform        string    `json:"platform"`
	PlatformVersion string    `json:"platformVersion"`
	Architecture    string    `json:"architecture"`
	Model           string    `json:"model"`
	Mobile          bool      `json:"mobile"`
	Bitness         string    `json:"bitness"`
	Wow64           bool      `json:"wow64"`
	FormFactors     []string  `json:"formFactors"`
}

// hostInfo 真实浏览器在未覆盖身份的页面上报告的值（由 hostInfoJS 读取），是身份派生的唯一外部输入
type hostInfo struct {
	UserAgent           string     `json:"userAgent"`
	Platform            string     `json:"platform"`
	Languages           []string   `json:"languages"`
	Timezone            string     `json:"timezone"`
	HardwareConcurrency int        `json:"hardwareConcurrency"`
	Screen              [3]float64 `json:"screen"` // width, height, devicePixelRatio
	Brands              []uaBrand  `json:"brands"`
	FullVersionList     []uaBrand  `json:"fullVersionList"`
	UAPlatform          string     `json:"uaPlatform"`
	PlatformVersion     string     `json:"platformVersion"`
	Architecture        string     `json:"architecture"`
	Bitness             string     `json:"bitness"`
	Model               string     `json:"model"`
	Mobile              bool       `json:"mobile"`
	Wow64               bool       `json:"wow64"`
	FormFactors         []string   `json:"formFactors"`
	WebGLVendor         string     `json:"webglVendor"`
	WebGLRenderer       string     `json:"webglRenderer"`
}

// parseHostInfo 解析 hostInfoJS 的返回值并校验必需字段
func parseHostInfo(raw string) (hostInfo, error) {
	var h hostInfo
	if err := json.Unmarshal([]byte(raw), &h); err != nil {
		return h, fmt.Errorf("parse host info: %w", err)
	}
	if h.UserAgent == "" || len(h.FullVersionList) == 0 || len(h.Languages) == 0 {
		return h, fmt.Errorf("host info incomplete: %s", raw)
	}
	return h, nil
}

// fullVersion 返回真实浏览器完整版本（优先 Google Chrome 品牌，其次 Chromium）
func (h hostInfo) fullVersion() string {
	for _, want := range []string{"Google Chrome", "Chromium"} {
		for _, b := range h.FullVersionList {
			if b.Brand == want {
				return b.Version
			}
		}
	}
	return h.FullVersionList[0].Version
}

// majorVersion 返回真实浏览器主版本号
func (h hostInfo) majorVersion() string {
	return strings.SplitN(h.fullVersion(), ".", 2)[0]
}

// screenGeometry 身份的屏幕与窗口几何（CSS 像素）；Width 为 0 表示不调整
type screenGeometry struct {
	Width, Height, AvailWidth, AvailHeight, AvailTop int
	DPR                                              float64
	OuterWidth, OuterHeight                          int // 浏览器窗口（最大化 = 可用区域）
	InnerWidth, InnerHeight                          int // 视口 = 窗口减去浏览器 UI
}

// geometryFor 由屏幕规格推算可用区域和最大化窗口：Windows 扣任务栏（Win10 40、Win11 48 CSS 像素，随缩放等比放大，
// 所以与 DPR 无关），Mac 扣菜单栏（刘海机型 38，其余 25）
func geometryFor(os OSFamily, s screenSpec, platformVersion string) screenGeometry {
	g := screenGeometry{Width: s.Width, Height: s.Height, DPR: s.DPR, AvailWidth: s.Width}
	if os == OSMac {
		bar := 25
		if s.Notch {
			bar = 38
		}
		g.AvailTop = bar
		g.AvailHeight = s.Height - bar
	} else {
		taskbar := 48
		if platformVersion == "10.0.0" {
			taskbar = 40
		}
		g.AvailHeight = s.Height - taskbar
	}
	g.OuterWidth, g.OuterHeight = g.AvailWidth, g.AvailHeight
	g.InnerWidth, g.InnerHeight = g.AvailWidth, g.AvailHeight-browserUIHeight
	return g
}

// Identity 一个账号对外呈现的完整浏览器身份，由 CDP 下发到所有目标
type Identity struct {
	OS                  OSFamily
	UserAgent           string
	Platform            string // navigator.platform
	AcceptLanguage      string // 例如 "zh-CN,zh"，Chrome 自动补 q 值
	Languages           []string
	Timezone            string
	HardwareConcurrency int
	Metadata            uaMetadata
	Screen              screenGeometry
	GPU                 *gpuProfile // nil 表示不改写 WebGL
	NoiseSeed           uint32      // 0 表示不加 canvas / 音频 / readPixels 噪声
	ScriptKey           string      // 注入脚本在同源 frame / 弹窗之间共用伪装表的口令，每次派生身份随机生成
}

// newScriptKey 生成注入脚本共用伪装表的随机口令：页面猜不到它，也就无法借它探测注入
func newScriptKey() string {
	b := make([]byte, 16)
	rand.Read(b) // crypto/rand 不会返回错误
	return hex.EncodeToString(b)
}

// Locale 返回 Intl 使用的 locale
func (id *Identity) Locale() string { return id.Languages[0] }

// noiseSeed 账号固定的非零噪声种子
func noiseSeed(userID string) uint32 {
	h := fnv.New32a()
	h.Write([]byte(userID + "\x00noise"))
	return h.Sum32() | 1
}

// identityFromConfig 由规范化后的账号指纹与真实浏览器派生身份：系统、屏幕、显卡等取自指纹，版本与品牌取自真实浏览器
func identityFromConfig(cfg *FingerprintConfig, host hostInfo) (*Identity, error) {
	os := osFromUserAgent(cfg.Browser.UserAgent)
	gpu, ok := lookupGPU(os, cfg.WebGL.Renderer)
	if !ok {
		return nil, fmt.Errorf("fingerprint %s: renderer %q is not in the %s pool (config not normalized)", cfg.UserID, cfg.WebGL.Renderer, os)
	}
	if len(cfg.Browser.Languages) == 0 {
		return nil, fmt.Errorf("fingerprint %s: no languages", cfg.UserID)
	}
	meta := uaMetadata{
		Brands:          host.Brands,
		FullVersionList: host.FullVersionList,
		PlatformVersion: cfg.Browser.PlatformVersion,
		Bitness:         "64",
		FormFactors:     []string{"Desktop"},
		Platform:        "Windows",
		Architecture:    "x86",
	}
	if os == OSMac {
		meta.Platform, meta.Architecture = "macOS", "arm"
	}
	spec, _ := screenSpecFor(os, cfg.Screen.Width, cfg.Screen.Height, cfg.Screen.DevicePixelRatio)
	return &Identity{
		OS:                  os,
		UserAgent:           canonicalUserAgent(os, host.majorVersion()),
		Platform:            platformFor(os),
		AcceptLanguage:      strings.Join(cfg.Browser.Languages, ","),
		Languages:           cfg.Browser.Languages,
		Timezone:            cfg.Timezone.Timezone,
		HardwareConcurrency: cfg.Browser.HardwareConcurrency,
		Metadata:            meta,
		Screen:              geometryFor(os, spec, cfg.Browser.PlatformVersion),
		GPU:                 &gpu,
		NoiseSeed:           noiseSeed(cfg.UserID),
		ScriptKey:           newScriptKey(),
	}, nil
}

// hostIdentity 不设指纹时的身份：沿用真实浏览器的值，只去掉无界面痕迹
// （UA 中的 HeadlessChrome、无界面默认 800×600 屏幕、SwiftShader 显卡标识）
func hostIdentity(host hostInfo) *Identity {
	formFactors := host.FormFactors
	if len(formFactors) == 0 {
		formFactors = []string{"Desktop"}
	}
	id := &Identity{
		ScriptKey:           newScriptKey(),
		OS:                  OSWindows, // 仅用于几何规则；Linux 宿主按 Windows 规则扣底部面板
		UserAgent:           strings.Replace(host.UserAgent, "HeadlessChrome/", "Chrome/", 1),
		Platform:            host.Platform,
		AcceptLanguage:      strings.Join(host.Languages, ","),
		Languages:           host.Languages,
		Timezone:            host.Timezone,
		HardwareConcurrency: host.HardwareConcurrency,
		Metadata: uaMetadata{
			Brands: host.Brands, FullVersionList: host.FullVersionList, Platform: host.UAPlatform,
			PlatformVersion: host.PlatformVersion, Architecture: host.Architecture, Model: host.Model,
			Mobile: host.Mobile, Bitness: host.Bitness, Wow64: host.Wow64, FormFactors: formFactors,
		},
	}
	if host.UAPlatform == "macOS" {
		id.OS = OSMac
	}
	if host.Screen[0] == 800 && host.Screen[1] == 600 {
		id.Screen = geometryFor(id.OS, screenSpec{Width: 1920, Height: 1080, DPR: 1}, host.PlatformVersion)
	}
	if strings.Contains(host.WebGLRenderer, "SwiftShader") {
		g := hostGPU(host.UAPlatform)
		id.GPU = &g
	}
	return id
}

// hostGPU 默认路径遮挡 SwiftShader 时使用的、与宿主系统相符的显卡标识（不改能力参数）
func hostGPU(uaPlatform string) gpuProfile {
	switch uaPlatform {
	case "Windows":
		g := windowsGPUs[0].Value
		return gpuProfile{Vendor: g.Vendor, Renderer: g.Renderer}
	case "macOS":
		g := macGPUs[0].Value
		return gpuProfile{Vendor: g.Vendor, Renderer: g.Renderer}
	default:
		return linuxHostGPU
	}
}
