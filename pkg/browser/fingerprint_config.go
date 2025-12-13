package browser

import (
	"crypto/md5"
	"encoding/json"
	"fmt"
	"math/rand"
	"strings"
	"time"
)

// FingerprintConfig 用户指纹配置
type FingerprintConfig struct {
	UserID string `json:"user_id"`

	// 屏幕相关
	Screen ScreenConfig `json:"screen"`

	// 浏览器相关
	Browser BrowserConfig `json:"browser"`

	// 系统相关
	System SystemConfig `json:"system"`

	// WebGL相关
	WebGL WebGLConfig `json:"webgl"`

	// 音频相关
	Audio AudioConfig `json:"audio"`

	// 网络相关
	Network NetworkConfig `json:"network"`

	// 时区相关
	Timezone TimezoneConfig `json:"timezone"`

	// Canvas相关
	Canvas CanvasConfig `json:"canvas"`

	// 字体相关
	Fonts FontsConfig `json:"fonts"`

	// 插件相关
	Plugins PluginsConfig `json:"plugins"`

	// 电池相关
	Battery BatteryConfig `json:"battery"`

	// 媒体设备
	MediaDevices []MediaDevice `json:"media_devices"`

	// TLS/JA4指纹
	TLSConfig TLSConfig `json:"tls_config"`

	// HTTP2指纹
	HTTP2Config HTTP2Config `json:"http2_config"`
}

// ScreenConfig 屏幕配置
type ScreenConfig struct {
	Width            int     `json:"width"`
	Height           int     `json:"height"`
	AvailWidth       int     `json:"avail_width"`
	AvailHeight      int     `json:"avail_height"`
	ColorDepth       int     `json:"color_depth"`
	PixelDepth       int     `json:"pixel_depth"`
	DevicePixelRatio float64 `json:"device_pixel_ratio"`
}

// BrowserConfig 浏览器配置
type BrowserConfig struct {
	UserAgent           string   `json:"user_agent"`
	Language            string   `json:"language"`
	Languages           []string `json:"languages"`
	Platform            string   `json:"platform"`
	Vendor              string   `json:"vendor"`
	CookieEnabled       bool     `json:"cookie_enabled"`
	DoNotTrack          *string  `json:"do_not_track"`
	HardwareConcurrency int      `json:"hardware_concurrency"`
	MaxTouchPoints      int      `json:"max_touch_points"`
	WebDriver           *bool    `json:"webdriver"`
}

// SystemConfig 系统配置
type SystemConfig struct {
	OS           string `json:"os"`
	OSVersion    string `json:"os_version"`
	Architecture string `json:"architecture"`
}

// WebGLConfig WebGL配置
type WebGLConfig struct {
	Vendor                 string `json:"vendor"`
	Renderer               string `json:"renderer"`
	Version                string `json:"version"`
	ShadingLanguageVersion string `json:"shading_language_version"`
	MaxTextureSize         int    `json:"max_texture_size"`
	MaxRenderbufferSize    int    `json:"max_renderbuffer_size"`
}

// AudioConfig 音频配置
type AudioConfig struct {
	SampleRate      int `json:"sample_rate"`
	MaxChannelCount int `json:"max_channel_count"`
	NumberOfInputs  int `json:"number_of_inputs"`
	NumberOfOutputs int `json:"number_of_outputs"`
}

// NetworkConfig 网络配置
type NetworkConfig struct {
	EffectiveType string  `json:"effective_type"`
	Downlink      float64 `json:"downlink"`
	RTT           int     `json:"rtt"`
	SaveData      bool    `json:"save_data"`
}

// TimezoneConfig 时区配置
type TimezoneConfig struct {
	Offset   int    `json:"offset"`
	Timezone string `json:"timezone"`
}

// CanvasConfig Canvas配置
type CanvasConfig struct {
	NoiseLevel   float64 `json:"noise_level"`   // 0-1之间的噪音等级
	TextVariance int     `json:"text_variance"` // 文本渲染差异
}

// FontsConfig 字体配置
type FontsConfig struct {
	AvailableFonts []string `json:"available_fonts"`
	BlockedFonts   []string `json:"blocked_fonts"`
}

// PluginsConfig 插件配置
type PluginsConfig struct {
	EnabledPlugins  []string `json:"enabled_plugins"`
	DisabledPlugins []string `json:"disabled_plugins"`
}

// BatteryConfig 电池配置
type BatteryConfig struct {
	Charging        bool     `json:"charging"`
	ChargingTime    *float64 `json:"charging_time"`
	DischargingTime *float64 `json:"discharging_time"`
	Level           float64  `json:"level"`
}

// MediaDevice 媒体设备
type MediaDevice struct {
	Kind     string `json:"kind"`
	Label    string `json:"label"`
	DeviceID string `json:"device_id"`
}

// TLSConfig TLS指纹配置
type TLSConfig struct {
	JA4            string   `json:"ja4"`              // JA4指纹
	JA3            string   `json:"ja3"`              // JA3指纹 (向后兼容)
	TLSVersion     string   `json:"tls_version"`      // TLS版本
	CipherSuites   []string `json:"cipher_suites"`    // 密码套件
	Extensions     []string `json:"extensions"`       // TLS扩展
	EllipticCurves []string `json:"elliptic_curves"`  // 椭圆曲线
	ECPointFormats []string `json:"ec_point_formats"` // EC点格式
	SignatureAlgs  []string `json:"signature_algs"`   // 签名算法
}

// HTTP2Config HTTP2指纹配置
type HTTP2Config struct {
	AKAMAI         string         `json:"akamai"`          // Akamai指纹
	Settings       map[string]int `json:"settings"`        // HTTP2设置
	WindowUpdate   int            `json:"window_update"`   // 窗口更新
	HeaderPriority map[string]int `json:"header_priority"` // 头部优先级
	PseudoHeaders  []string       `json:"pseudo_headers"`  // 伪头部顺序
}

// FingerprintGenerator 指纹生成器
type FingerprintGenerator struct {
	rand *rand.Rand
}

// NewFingerprintGenerator 创建指纹生成器
func NewFingerprintGenerator() *FingerprintGenerator {
	return &FingerprintGenerator{
		rand: rand.New(rand.NewSource(time.Now().UnixNano())),
	}
}

// GenerateFingerprint 为用户生成独特指纹
func (fg *FingerprintGenerator) GenerateFingerprint(userID string) *FingerprintConfig {
	// 使用用户ID作为种子，确保同一用户的指纹一致
	userSeed := fg.hashUserID(userID)
	userRand := rand.New(rand.NewSource(userSeed))

	config := &FingerprintConfig{
		UserID: userID,
	}

	// 生成屏幕配置
	config.Screen = fg.generateScreenConfig(userRand)

	// 生成浏览器配置
	config.Browser = fg.generateBrowserConfig(userRand)

	// 生成系统配置
	config.System = fg.generateSystemConfig(userRand)

	// 生成WebGL配置
	config.WebGL = fg.generateWebGLConfig(userRand)

	// 生成音频配置
	config.Audio = fg.generateAudioConfig(userRand)

	// 生成网络配置
	config.Network = fg.generateNetworkConfig(userRand)

	// 生成时区配置
	config.Timezone = fg.generateTimezoneConfig(userRand)

	// 生成Canvas配置
	config.Canvas = fg.generateCanvasConfig(userRand)

	// 生成字体配置
	config.Fonts = fg.generateFontsConfig(userRand)

	// 生成插件配置
	config.Plugins = fg.generatePluginsConfig(userRand)

	// 生成电池配置
	config.Battery = fg.generateBatteryConfig(userRand)

	// 生成媒体设备配置
	config.MediaDevices = fg.generateMediaDevicesConfig(userRand)

	// 生成TLS配置
	config.TLSConfig = fg.generateTLSConfig(userRand)

	// 生成HTTP2配置
	config.HTTP2Config = fg.generateHTTP2Config(userRand)

	return config
}

// hashUserID 将用户ID转换为数值种子
func (fg *FingerprintGenerator) hashUserID(userID string) int64 {
	hasher := md5.New()
	hasher.Write([]byte(userID))
	hashBytes := hasher.Sum(nil)

	// 将前8字节转换为int64
	seed := int64(0)
	for i := 0; i < 8 && i < len(hashBytes); i++ {
		seed = (seed << 8) | int64(hashBytes[i])
	}

	return seed
}

// generateScreenConfig 生成屏幕配置
func (fg *FingerprintGenerator) generateScreenConfig(userRand *rand.Rand) ScreenConfig {
	// 大幅扩展屏幕分辨率配置 - 基于实际市场统计
	type ResolutionConfig struct {
		width      int
		height     int
		weight     int       // 权重：数字越大越常见
		dprOptions []float64 // 可能的设备像素比
	}

	resolutionConfigs := []ResolutionConfig{
		// ========== 最常见的桌面分辨率 ==========
		{1920, 1080, 35, []float64{1.0, 1.25}}, // Full HD - 最常见
		{1366, 768, 15, []float64{1.0}},        // 老笔记本标准
		{1536, 864, 10, []float64{1.0, 1.25}},  // 常见笔记本
		{1440, 900, 8, []float64{1.0}},         // 16:10笔记本
		{1600, 900, 7, []float64{1.0}},         // HD+

		// ========== 2K分辨率 ==========
		{2560, 1440, 20, []float64{1.0, 1.25, 1.5}}, // 2K - 很常见
		{2560, 1600, 5, []float64{2.0}},             // 16:10 2K
		{2048, 1152, 3, []float64{1.0}},             // 2K变体

		// ========== 高分辨率 ==========
		{3840, 2160, 12, []float64{1.0, 1.5, 2.0}}, // 4K UHD
		{3440, 1440, 4, []float64{1.0, 1.25}},      // 超宽屏
		{3840, 1600, 2, []float64{1.0}},            // 超宽屏
		{3840, 1080, 1, []float64{1.0}},            // 超宽32:9

		// ========== MacBook常见分辨率（高DPI）==========
		{1440, 900, 8, []float64{2.0}},  // MacBook Pro 13" Retina
		{1680, 1050, 6, []float64{2.0}}, // MacBook Pro 15"
		{1728, 1117, 4, []float64{2.0}}, // MacBook Air 13" M1/M2
		{1920, 1200, 7, []float64{2.0}}, // MacBook Pro 16"
		{2560, 1600, 5, []float64{2.0}}, // MacBook Pro 14"/16" (缩放)
		{3024, 1964, 3, []float64{2.0}}, // MacBook Pro 14" M1/M2 原生
		{3456, 2234, 3, []float64{2.0}}, // MacBook Pro 16" M1/M2 原生

		// ========== iMac/Studio Display ==========
		{2560, 1440, 4, []float64{2.0}}, // iMac 27"
		{2880, 1800, 2, []float64{2.0}}, // iMac 27" Retina
		{5120, 2880, 2, []float64{2.0}}, // iMac 27" 5K

		// ========== Windows高DPI笔记本 ==========
		{1920, 1080, 10, []float64{1.5, 2.0}}, // Full HD高DPI
		{2160, 1440, 4, []float64{1.5}},       // Surface Laptop
		{2256, 1504, 3, []float64{1.5}},       // Surface Pro
		{2880, 1800, 3, []float64{2.0}},       // Dell XPS 15
		{3000, 2000, 2, []float64{2.0}},       // Surface Laptop Studio
		{3200, 1800, 3, []float64{1.5, 2.0}},  // QHD+
		{3200, 2000, 2, []float64{2.0}},       // Surface Book

		// ========== 垂直显示器和特殊比例 ==========
		{1200, 1920, 1, []float64{1.0}}, // 竖屏1080p
		{1080, 1920, 1, []float64{1.0}}, // 竖屏
		{2160, 3840, 1, []float64{1.0}}, // 竖屏4K

		// ========== 老旧但仍在使用的分辨率 ==========
		{1280, 720, 5, []float64{1.0}},  // HD
		{1280, 1024, 4, []float64{1.0}}, // 5:4老显示器
		{1024, 768, 2, []float64{1.0}},  // XGA老显示器
		{1680, 1050, 3, []float64{1.0}}, // WSXGA+

		// ========== 游戏/专业显示器 ==========
		{3440, 1440, 3, []float64{1.0}}, // 21:9超宽
		{5120, 1440, 1, []float64{1.0}}, // 32:9超超宽
		{2560, 1080, 2, []float64{1.0}}, // 21:9 1080p
	}

	// 加权随机选择分辨率
	totalWeight := 0
	for _, rc := range resolutionConfigs {
		totalWeight += rc.weight
	}

	randWeight := userRand.Intn(totalWeight)
	var selectedConfig ResolutionConfig
	currentWeight := 0
	for _, rc := range resolutionConfigs {
		currentWeight += rc.weight
		if randWeight < currentWeight {
			selectedConfig = rc
			break
		}
	}

	width, height := selectedConfig.width, selectedConfig.height

	// 从该分辨率的DPR选项中选择
	devicePixelRatio := selectedConfig.dprOptions[userRand.Intn(len(selectedConfig.dprOptions))]

	// 任务栏高度变化（更真实的分布）
	taskbarHeights := []int{0, 30, 40, 48, 60, 72} // Windows/macOS不同任务栏高度
	taskbarHeight := taskbarHeights[userRand.Intn(len(taskbarHeights))]

	// 侧边栏宽度（某些系统有侧边栏）
	sidebarWidth := 0
	if userRand.Float64() < 0.1 { // 10%概率有侧边栏
		sidebarWidth = userRand.Intn(20) + 10 // 10-30px
	}

	return ScreenConfig{
		Width:            width,
		Height:           height,
		AvailWidth:       width - sidebarWidth,
		AvailHeight:      height - taskbarHeight,
		ColorDepth:       24,
		PixelDepth:       24,
		DevicePixelRatio: devicePixelRatio,
	}
}

// generateBrowserConfig 生成浏览器配置
func (fg *FingerprintGenerator) generateBrowserConfig(userRand *rand.Rand) BrowserConfig {
	// 大幅扩展Chrome版本池 - 涵盖最近2年的版本（2024年12月更新）
	chromeVersions := []string{
		// 2024年12月-2025年最新版本
		"142.0.7444.176", "141.0.7432.137", "140.0.7486.110", "139.0.7468.126",
		"138.0.7414.140", "137.0.7355.172", "136.0.7289.145", "135.0.7260.157",
		// 2024年下半年版本
		"134.0.7212.168", "133.0.7156.193", "132.0.7098.224", "131.0.6778.204",
		"130.0.6723.117", "129.0.6668.100", "128.0.6613.138", "127.0.6533.120",
		"126.0.6478.127", "125.0.6422.142", "124.0.6367.207", "123.0.6312.122",
		// 2024年上半年版本
		"122.0.6261.129", "121.0.6167.185", "120.0.6099.234", "119.0.6045.199",
		"118.0.5993.117", "117.0.5938.149", "116.0.5845.187", "115.0.5790.170",
		// 2023年稳定版本
		"114.0.5735.198", "113.0.5672.126", "112.0.5615.137", "111.0.5563.146",
		"110.0.5481.177", "109.0.5414.119", "108.0.5359.124", "107.0.5304.121",
		// 2023年早期版本
		"106.0.5249.119", "105.0.5195.125", "104.0.5112.101", "103.0.5060.134",
		"102.0.5005.115", "101.0.4951.67", "100.0.4896.127", "99.0.4844.84",
	}
	chromeVersion := chromeVersions[userRand.Intn(len(chromeVersions))]

	// 扩展操作系统平台配置
	type PlatformConfig struct {
		Platform string
		OSName   string
		Versions []string
	}

	platformConfigs := []PlatformConfig{
		// macOS - 多个版本和架构
		{
			Platform: "MacIntel",
			OSName:   "Macintosh; Intel Mac OS X",
			Versions: []string{
				"10_15_7",                    // Catalina
				"11_0_0", "11_2_3", "11_6_8", // Big Sur
				"12_0_0", "12_3_1", "12_6_9", // Monterey
				"13_0_0", "13_2_1", "13_5_2", "13_6_1", // Ventura
				"14_0_0", "14_1_2", "14_3_1", "14_5_0", // Sonoma
			},
		},
		// Apple Silicon Mac
		{
			Platform: "MacIntel", // 注意：M系列仍报告为MacIntel
			OSName:   "Macintosh; Intel Mac OS X",
			Versions: []string{
				"12_0_0", "12_4_0", "13_0_0", "13_3_0", "14_0_0", "14_2_1",
			},
		},
		// Windows 10 - 多个构建版本
		{
			Platform: "Win32",
			OSName:   "Windows NT 10.0; Win64; x64",
			Versions: []string{
				"10.0", // 多个构建号实际上UA中都显示10.0
			},
		},
		// Windows 11
		{
			Platform: "Win32",
			OSName:   "Windows NT 10.0; Win64; x64", // Win11仍报告为10.0
			Versions: []string{
				"10.0",
			},
		},
		// Linux发行版
		{
			Platform: "Linux x86_64",
			OSName:   "X11; Linux x86_64",
			Versions: []string{
				"", // Linux不在UA中显示具体版本
			},
		},
		{
			Platform: "Linux x86_64",
			OSName:   "X11; Ubuntu; Linux x86_64",
			Versions: []string{""},
		},
		{
			Platform: "Linux x86_64",
			OSName:   "X11; Fedora; Linux x86_64",
			Versions: []string{""},
		},
	}

	platformConfig := platformConfigs[userRand.Intn(len(platformConfigs))]

	// 构建UserAgent
	var userAgent string
	if len(platformConfig.Versions) > 0 && platformConfig.Versions[0] != "" {
		version := platformConfig.Versions[userRand.Intn(len(platformConfig.Versions))]
		userAgent = fmt.Sprintf("Mozilla/5.0 (%s %s) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/%s Safari/537.36",
			platformConfig.OSName, version, chromeVersion)
	} else {
		userAgent = fmt.Sprintf("Mozilla/5.0 (%s) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/%s Safari/537.36",
			platformConfig.OSName, chromeVersion)
	}

	// 扩展语言配置 - 增加更多语言和组合
	languages := [][]string{
		// 英语系
		{"en-US", "en"},
		{"en-GB", "en"},
		{"en-AU", "en"},
		{"en-CA", "en"},
		{"en-US", "en", "es"}, // 双语用户
		// 中文系
		{"zh-CN", "zh"},
		{"zh-TW", "zh"},
		{"zh-HK", "zh"},
		{"zh-CN", "zh", "en"}, // 双语用户
		// 日语系
		{"ja-JP", "ja"},
		{"ja", "ja", "en"},
		// 韩语系
		{"ko-KR", "ko"},
		{"ko", "ko", "en"},
		// 欧洲语言
		{"de-DE", "de"},
		{"de-DE", "de", "en"},
		{"fr-FR", "fr"},
		{"fr-FR", "fr", "en"},
		{"es-ES", "es"},
		{"es-ES", "es", "en"},
		{"it-IT", "it"},
		{"pt-BR", "pt"},
		{"pt-PT", "pt"},
		{"ru-RU", "ru"},
		{"pl-PL", "pl"},
		{"nl-NL", "nl"},
		{"sv-SE", "sv"},
		{"tr-TR", "tr"},
		// 其他
		{"ar-SA", "ar"},
		{"th-TH", "th"},
		{"vi-VN", "vi"},
		{"id-ID", "id"},
	}
	selectedLangs := languages[userRand.Intn(len(languages))]

	// 更真实的硬件并发数分布（基于实际设备统计）
	hardwareConcurrencyWeights := []struct {
		cores  int
		weight int // 权重，数字越大越常见
	}{
		{2, 5},   // 老旧设备
		{4, 20},  // 最常见：入门级笔记本、台式机
		{6, 15},  // 常见：中端设备
		{8, 25},  // 最常见：主流设备
		{10, 8},  // 较少：高端Intel
		{12, 12}, // 常见：高端AMD/M系列
		{14, 4},  // 较少
		{16, 8},  // 工作站
		{20, 2},  // 少见：高端工作站
		{24, 1},  // 罕见：专业工作站
	}

	// 加权随机选择
	totalWeight := 0
	for _, hw := range hardwareConcurrencyWeights {
		totalWeight += hw.weight
	}
	randWeight := userRand.Intn(totalWeight)
	hardwareConcurrency := 8 // 默认
	currentWeight := 0
	for _, hw := range hardwareConcurrencyWeights {
		currentWeight += hw.weight
		if randWeight < currentWeight {
			hardwareConcurrency = hw.cores
			break
		}
	}

	// MaxTouchPoints - 更真实的分布
	maxTouchPoints := 0
	if platformConfig.Platform == "Win32" {
		// Windows设备可能有触摸屏
		if userRand.Float64() < 0.3 { // 30%概率
			maxTouchPoints = userRand.Intn(10) + 1 // 1-10点触控
		}
	}

	return BrowserConfig{
		UserAgent:           userAgent,
		Language:            selectedLangs[0],
		Languages:           selectedLangs,
		Platform:            platformConfig.Platform,
		Vendor:              "Google Inc.",
		CookieEnabled:       true,
		DoNotTrack:          nil, // 通常为null
		HardwareConcurrency: hardwareConcurrency,
		MaxTouchPoints:      maxTouchPoints,
		WebDriver:           nil, // 反检测：设为undefined
	}
}

// generateSystemConfig 生成系统配置
func (fg *FingerprintGenerator) generateSystemConfig(userRand *rand.Rand) SystemConfig {
	systems := []SystemConfig{
		{"macOS", "14.0", "x64"},
		{"Windows", "10", "x64"},
		{"Linux", "Ubuntu 20.04", "x64"},
		{"Windows", "11", "x64"},
		{"macOS", "13.0", "x64"},
	}

	return systems[userRand.Intn(len(systems))]
}

// generateWebGLConfig 生成WebGL配置
func (fg *FingerprintGenerator) generateWebGLConfig(userRand *rand.Rand) WebGLConfig {
	// 大幅扩展WebGL配置池 - 包含各种GPU型号
	webglConfigs := []WebGLConfig{
		// ========== macOS - Apple Silicon ==========
		{
			Vendor:                 "Apple Inc.",
			Renderer:               "ANGLE (Apple, Apple M1 Pro, OpenGL 4.1 Metal - 88.1)",
			Version:                "WebGL 1.0 (OpenGL ES 2.0 Chromium)",
			ShadingLanguageVersion: "WebGL GLSL ES 1.0 (OpenGL ES GLSL ES 1.0 Chromium)",
			MaxTextureSize:         16384,
			MaxRenderbufferSize:    16384,
		},
		{
			Vendor:                 "Apple Inc.",
			Renderer:               "ANGLE (Apple, Apple M1 Max, OpenGL 4.1 Metal - 88.1)",
			Version:                "WebGL 1.0 (OpenGL ES 2.0 Chromium)",
			ShadingLanguageVersion: "WebGL GLSL ES 1.0 (OpenGL ES GLSL ES 1.0 Chromium)",
			MaxTextureSize:         16384,
			MaxRenderbufferSize:    16384,
		},
		{
			Vendor:                 "Apple Inc.",
			Renderer:               "ANGLE (Apple, Apple M2, OpenGL 4.1 Metal - 88.1)",
			Version:                "WebGL 1.0 (OpenGL ES 2.0 Chromium)",
			ShadingLanguageVersion: "WebGL GLSL ES 1.0 (OpenGL ES GLSL ES 1.0 Chromium)",
			MaxTextureSize:         16384,
			MaxRenderbufferSize:    16384,
		},
		{
			Vendor:                 "Apple Inc.",
			Renderer:               "ANGLE (Apple, Apple M2 Pro, OpenGL 4.1 Metal - 88.1)",
			Version:                "WebGL 1.0 (OpenGL ES 2.0 Chromium)",
			ShadingLanguageVersion: "WebGL GLSL ES 1.0 (OpenGL ES GLSL ES 1.0 Chromium)",
			MaxTextureSize:         16384,
			MaxRenderbufferSize:    16384,
		},
		{
			Vendor:                 "Apple Inc.",
			Renderer:               "ANGLE (Apple, Apple M2 Max, OpenGL 4.1 Metal - 88.1)",
			Version:                "WebGL 1.0 (OpenGL ES 2.0 Chromium)",
			ShadingLanguageVersion: "WebGL GLSL ES 1.0 (OpenGL ES GLSL ES 1.0 Chromium)",
			MaxTextureSize:         16384,
			MaxRenderbufferSize:    16384,
		},
		{
			Vendor:                 "Apple Inc.",
			Renderer:               "ANGLE (Apple, Apple M3, OpenGL 4.1 Metal - 88.1)",
			Version:                "WebGL 1.0 (OpenGL ES 2.0 Chromium)",
			ShadingLanguageVersion: "WebGL GLSL ES 1.0 (OpenGL ES GLSL ES 1.0 Chromium)",
			MaxTextureSize:         16384,
			MaxRenderbufferSize:    16384,
		},

		// ========== macOS - Intel集成显卡 ==========
		{
			Vendor:                 "Intel Inc.",
			Renderer:               "ANGLE (Intel, Intel(R) Iris(TM) Plus Graphics 640, OpenGL 4.1)",
			Version:                "WebGL 1.0 (OpenGL ES 2.0 Chromium)",
			ShadingLanguageVersion: "WebGL GLSL ES 1.0 (OpenGL ES GLSL ES 1.0 Chromium)",
			MaxTextureSize:         16384,
			MaxRenderbufferSize:    16384,
		},
		{
			Vendor:                 "Intel Inc.",
			Renderer:               "ANGLE (Intel, Intel(R) UHD Graphics 630, OpenGL 4.1)",
			Version:                "WebGL 1.0 (OpenGL ES 2.0 Chromium)",
			ShadingLanguageVersion: "WebGL GLSL ES 1.0 (OpenGL ES GLSL ES 1.0 Chromium)",
			MaxTextureSize:         16384,
			MaxRenderbufferSize:    16384,
		},

		// ========== macOS - AMD独立显卡 ==========
		{
			Vendor:                 "ATI Technologies Inc.",
			Renderer:               "ANGLE (AMD, AMD Radeon Pro 5500M, OpenGL 4.1)",
			Version:                "WebGL 1.0 (OpenGL ES 2.0 Chromium)",
			ShadingLanguageVersion: "WebGL GLSL ES 1.0 (OpenGL ES GLSL ES 1.0 Chromium)",
			MaxTextureSize:         16384,
			MaxRenderbufferSize:    16384,
		},
		{
			Vendor:                 "ATI Technologies Inc.",
			Renderer:               "ANGLE (AMD, AMD Radeon Pro 560X, OpenGL 4.1)",
			Version:                "WebGL 1.0 (OpenGL ES 2.0 Chromium)",
			ShadingLanguageVersion: "WebGL GLSL ES 1.0 (OpenGL ES GLSL ES 1.0 Chromium)",
			MaxTextureSize:         16384,
			MaxRenderbufferSize:    16384,
		},

		// ========== Windows - Intel集成显卡 ==========
		{
			Vendor:                 "Google Inc. (Intel)",
			Renderer:               "ANGLE (Intel, Intel(R) UHD Graphics 630 Direct3D11 vs_5_0 ps_5_0, D3D11)",
			Version:                "WebGL 1.0 (OpenGL ES 2.0 Chromium)",
			ShadingLanguageVersion: "WebGL GLSL ES 1.0 (OpenGL ES GLSL ES 1.0 Chromium)",
			MaxTextureSize:         16384,
			MaxRenderbufferSize:    16384,
		},
		{
			Vendor:                 "Google Inc. (Intel)",
			Renderer:               "ANGLE (Intel, Intel(R) UHD Graphics 620 Direct3D11 vs_5_0 ps_5_0, D3D11)",
			Version:                "WebGL 1.0 (OpenGL ES 2.0 Chromium)",
			ShadingLanguageVersion: "WebGL GLSL ES 1.0 (OpenGL ES GLSL ES 1.0 Chromium)",
			MaxTextureSize:         16384,
			MaxRenderbufferSize:    16384,
		},
		{
			Vendor:                 "Google Inc. (Intel)",
			Renderer:               "ANGLE (Intel, Intel(R) Iris(R) Xe Graphics Direct3D11 vs_5_0 ps_5_0, D3D11)",
			Version:                "WebGL 1.0 (OpenGL ES 2.0 Chromium)",
			ShadingLanguageVersion: "WebGL GLSL ES 1.0 (OpenGL ES GLSL ES 1.0 Chromium)",
			MaxTextureSize:         16384,
			MaxRenderbufferSize:    16384,
		},
		{
			Vendor:                 "Google Inc. (Intel)",
			Renderer:               "ANGLE (Intel, Intel(R) HD Graphics 530 Direct3D11 vs_5_0 ps_5_0, D3D11)",
			Version:                "WebGL 1.0 (OpenGL ES 2.0 Chromium)",
			ShadingLanguageVersion: "WebGL GLSL ES 1.0 (OpenGL ES GLSL ES 1.0 Chromium)",
			MaxTextureSize:         16384,
			MaxRenderbufferSize:    16384,
		},

		// ========== Windows - NVIDIA显卡（最丰富）==========
		{
			Vendor:                 "Google Inc. (NVIDIA)",
			Renderer:               "ANGLE (NVIDIA, NVIDIA GeForce RTX 3060 Direct3D11 vs_5_0 ps_5_0, D3D11)",
			Version:                "WebGL 1.0 (OpenGL ES 2.0 Chromium)",
			ShadingLanguageVersion: "WebGL GLSL ES 1.0 (OpenGL ES GLSL ES 1.0 Chromium)",
			MaxTextureSize:         32768,
			MaxRenderbufferSize:    32768,
		},
		{
			Vendor:                 "Google Inc. (NVIDIA)",
			Renderer:               "ANGLE (NVIDIA, NVIDIA GeForce RTX 3070 Direct3D11 vs_5_0 ps_5_0, D3D11)",
			Version:                "WebGL 1.0 (OpenGL ES 2.0 Chromium)",
			ShadingLanguageVersion: "WebGL GLSL ES 1.0 (OpenGL ES GLSL ES 1.0 Chromium)",
			MaxTextureSize:         32768,
			MaxRenderbufferSize:    32768,
		},
		{
			Vendor:                 "Google Inc. (NVIDIA)",
			Renderer:               "ANGLE (NVIDIA, NVIDIA GeForce RTX 3080 Direct3D11 vs_5_0 ps_5_0, D3D11)",
			Version:                "WebGL 1.0 (OpenGL ES 2.0 Chromium)",
			ShadingLanguageVersion: "WebGL GLSL ES 1.0 (OpenGL ES GLSL ES 1.0 Chromium)",
			MaxTextureSize:         32768,
			MaxRenderbufferSize:    32768,
		},
		{
			Vendor:                 "Google Inc. (NVIDIA)",
			Renderer:               "ANGLE (NVIDIA, NVIDIA GeForce RTX 4060 Direct3D11 vs_5_0 ps_5_0, D3D11)",
			Version:                "WebGL 1.0 (OpenGL ES 2.0 Chromium)",
			ShadingLanguageVersion: "WebGL GLSL ES 1.0 (OpenGL ES GLSL ES 1.0 Chromium)",
			MaxTextureSize:         32768,
			MaxRenderbufferSize:    32768,
		},
		{
			Vendor:                 "Google Inc. (NVIDIA)",
			Renderer:               "ANGLE (NVIDIA, NVIDIA GeForce RTX 4070 Direct3D11 vs_5_0 ps_5_0, D3D11)",
			Version:                "WebGL 1.0 (OpenGL ES 2.0 Chromium)",
			ShadingLanguageVersion: "WebGL GLSL ES 1.0 (OpenGL ES GLSL ES 1.0 Chromium)",
			MaxTextureSize:         32768,
			MaxRenderbufferSize:    32768,
		},
		{
			Vendor:                 "Google Inc. (NVIDIA)",
			Renderer:               "ANGLE (NVIDIA, NVIDIA GeForce GTX 1650 Direct3D11 vs_5_0 ps_5_0, D3D11)",
			Version:                "WebGL 1.0 (OpenGL ES 2.0 Chromium)",
			ShadingLanguageVersion: "WebGL GLSL ES 1.0 (OpenGL ES GLSL ES 1.0 Chromium)",
			MaxTextureSize:         16384,
			MaxRenderbufferSize:    16384,
		},
		{
			Vendor:                 "Google Inc. (NVIDIA)",
			Renderer:               "ANGLE (NVIDIA, NVIDIA GeForce GTX 1660 Ti Direct3D11 vs_5_0 ps_5_0, D3D11)",
			Version:                "WebGL 1.0 (OpenGL ES 2.0 Chromium)",
			ShadingLanguageVersion: "WebGL GLSL ES 1.0 (OpenGL ES GLSL ES 1.0 Chromium)",
			MaxTextureSize:         16384,
			MaxRenderbufferSize:    16384,
		},
		{
			Vendor:                 "Google Inc. (NVIDIA)",
			Renderer:               "ANGLE (NVIDIA, NVIDIA GeForce RTX 2060 Direct3D11 vs_5_0 ps_5_0, D3D11)",
			Version:                "WebGL 1.0 (OpenGL ES 2.0 Chromium)",
			ShadingLanguageVersion: "WebGL GLSL ES 1.0 (OpenGL ES GLSL ES 1.0 Chromium)",
			MaxTextureSize:         32768,
			MaxRenderbufferSize:    32768,
		},
		{
			Vendor:                 "Google Inc. (NVIDIA)",
			Renderer:               "ANGLE (NVIDIA, NVIDIA GeForce RTX 2070 SUPER Direct3D11 vs_5_0 ps_5_0, D3D11)",
			Version:                "WebGL 1.0 (OpenGL ES 2.0 Chromium)",
			ShadingLanguageVersion: "WebGL GLSL ES 1.0 (OpenGL ES GLSL ES 1.0 Chromium)",
			MaxTextureSize:         32768,
			MaxRenderbufferSize:    32768,
		},
		{
			Vendor:                 "Google Inc. (NVIDIA)",
			Renderer:               "ANGLE (NVIDIA, NVIDIA GeForce MX450 Direct3D11 vs_5_0 ps_5_0, D3D11)",
			Version:                "WebGL 1.0 (OpenGL ES 2.0 Chromium)",
			ShadingLanguageVersion: "WebGL GLSL ES 1.0 (OpenGL ES GLSL ES 1.0 Chromium)",
			MaxTextureSize:         16384,
			MaxRenderbufferSize:    16384,
		},

		// ========== Windows - AMD显卡 ==========
		{
			Vendor:                 "Google Inc. (AMD)",
			Renderer:               "ANGLE (AMD, AMD Radeon(TM) Graphics Direct3D11 vs_5_0 ps_5_0, D3D11)",
			Version:                "WebGL 1.0 (OpenGL ES 2.0 Chromium)",
			ShadingLanguageVersion: "WebGL GLSL ES 1.0 (OpenGL ES GLSL ES 1.0 Chromium)",
			MaxTextureSize:         16384,
			MaxRenderbufferSize:    16384,
		},
		{
			Vendor:                 "Google Inc. (AMD)",
			Renderer:               "ANGLE (AMD, AMD Radeon RX 6600 XT Direct3D11 vs_5_0 ps_5_0, D3D11)",
			Version:                "WebGL 1.0 (OpenGL ES 2.0 Chromium)",
			ShadingLanguageVersion: "WebGL GLSL ES 1.0 (OpenGL ES GLSL ES 1.0 Chromium)",
			MaxTextureSize:         16384,
			MaxRenderbufferSize:    16384,
		},
		{
			Vendor:                 "Google Inc. (AMD)",
			Renderer:               "ANGLE (AMD, AMD Radeon RX 6700 XT Direct3D11 vs_5_0 ps_5_0, D3D11)",
			Version:                "WebGL 1.0 (OpenGL ES 2.0 Chromium)",
			ShadingLanguageVersion: "WebGL GLSL ES 1.0 (OpenGL ES GLSL ES 1.0 Chromium)",
			MaxTextureSize:         16384,
			MaxRenderbufferSize:    16384,
		},
		{
			Vendor:                 "Google Inc. (AMD)",
			Renderer:               "ANGLE (AMD, AMD Radeon RX 5700 XT Direct3D11 vs_5_0 ps_5_0, D3D11)",
			Version:                "WebGL 1.0 (OpenGL ES 2.0 Chromium)",
			ShadingLanguageVersion: "WebGL GLSL ES 1.0 (OpenGL ES GLSL ES 1.0 Chromium)",
			MaxTextureSize:         16384,
			MaxRenderbufferSize:    16384,
		},
		{
			Vendor:                 "Google Inc. (AMD)",
			Renderer:               "ANGLE (AMD, AMD Radeon RX 580 Series Direct3D11 vs_5_0 ps_5_0, D3D11)",
			Version:                "WebGL 1.0 (OpenGL ES 2.0 Chromium)",
			ShadingLanguageVersion: "WebGL GLSL ES 1.0 (OpenGL ES GLSL ES 1.0 Chromium)",
			MaxTextureSize:         16384,
			MaxRenderbufferSize:    16384,
		},
		{
			Vendor:                 "Google Inc. (AMD)",
			Renderer:               "ANGLE (AMD, AMD Radeon RX Vega 56 Direct3D11 vs_5_0 ps_5_0, D3D11)",
			Version:                "WebGL 1.0 (OpenGL ES 2.0 Chromium)",
			ShadingLanguageVersion: "WebGL GLSL ES 1.0 (OpenGL ES GLSL ES 1.0 Chromium)",
			MaxTextureSize:         16384,
			MaxRenderbufferSize:    16384,
		},

		// ========== Linux - NVIDIA ==========
		{
			Vendor:                 "NVIDIA Corporation",
			Renderer:               "NVIDIA GeForce RTX 3060/PCIe/SSE2",
			Version:                "WebGL 1.0 (OpenGL ES 2.0 Chromium)",
			ShadingLanguageVersion: "WebGL GLSL ES 1.0 (OpenGL ES GLSL ES 1.0 Chromium)",
			MaxTextureSize:         32768,
			MaxRenderbufferSize:    32768,
		},
		{
			Vendor:                 "NVIDIA Corporation",
			Renderer:               "NVIDIA GeForce GTX 1660 Ti/PCIe/SSE2",
			Version:                "WebGL 1.0 (OpenGL ES 2.0 Chromium)",
			ShadingLanguageVersion: "WebGL GLSL ES 1.0 (OpenGL ES GLSL ES 1.0 Chromium)",
			MaxTextureSize:         32768,
			MaxRenderbufferSize:    32768,
		},

		// ========== Linux - AMD/Mesa ==========
		{
			Vendor:                 "X.Org",
			Renderer:               "AMD Radeon RX 6600 (navi23, LLVM 15.0.7, DRM 3.54, 6.5.0-28-generic)",
			Version:                "WebGL 1.0 (OpenGL ES 2.0 Chromium)",
			ShadingLanguageVersion: "WebGL GLSL ES 1.0 (OpenGL ES GLSL ES 1.0 Chromium)",
			MaxTextureSize:         16384,
			MaxRenderbufferSize:    16384,
		},
		{
			Vendor:                 "Mesa",
			Renderer:               "Mesa Intel(R) UHD Graphics 630 (CFL GT2)",
			Version:                "WebGL 1.0 (OpenGL ES 2.0 Chromium)",
			ShadingLanguageVersion: "WebGL GLSL ES 1.0 (OpenGL ES GLSL ES 1.0 Chromium)",
			MaxTextureSize:         16384,
			MaxRenderbufferSize:    16384,
		},

		// ========== WebKit (Safari风格 - 备用) ==========
		{
			Vendor:                 "WebKit",
			Renderer:               "WebKit WebGL",
			Version:                "WebGL 1.0 (OpenGL ES 2.0 Chromium)",
			ShadingLanguageVersion: "WebGL GLSL ES 1.0 (OpenGL ES GLSL ES 1.0 Chromium)",
			MaxTextureSize:         16384,
			MaxRenderbufferSize:    16384,
		},
	}

	return webglConfigs[userRand.Intn(len(webglConfigs))]
}

// generateAudioConfig 生成音频配置
func (fg *FingerprintGenerator) generateAudioConfig(userRand *rand.Rand) AudioConfig {
	sampleRates := []int{44100, 48000, 96000}
	maxChannelCounts := []int{2, 4, 6, 8}

	return AudioConfig{
		SampleRate:      sampleRates[userRand.Intn(len(sampleRates))],
		MaxChannelCount: maxChannelCounts[userRand.Intn(len(maxChannelCounts))],
		NumberOfInputs:  1,
		NumberOfOutputs: 0,
	}
}

// generateNetworkConfig 生成网络配置
func (fg *FingerprintGenerator) generateNetworkConfig(userRand *rand.Rand) NetworkConfig {
	effectiveTypes := []string{"4g", "3g", "2g", "slow-2g"}
	downlinks := []float64{0.5, 1.0, 1.55, 2.0, 3.5, 10.0, 25.0}
	rtts := []int{50, 100, 150, 200, 300, 500}

	return NetworkConfig{
		EffectiveType: effectiveTypes[userRand.Intn(len(effectiveTypes))],
		Downlink:      downlinks[userRand.Intn(len(downlinks))],
		RTT:           rtts[userRand.Intn(len(rtts))],
		SaveData:      userRand.Float64() < 0.1, // 10%概率启用省流量模式
	}
}

// generateTimezoneConfig 生成时区配置
func (fg *FingerprintGenerator) generateTimezoneConfig(userRand *rand.Rand) TimezoneConfig {
	timezones := []TimezoneConfig{
		{-480, "Asia/Shanghai"},
		{0, "UTC"},
		{-300, "America/New_York"},
		{-480, "America/Los_Angeles"},
		{60, "Europe/Berlin"},
		{540, "Asia/Tokyo"},
		{-180, "America/Sao_Paulo"},
	}

	return timezones[userRand.Intn(len(timezones))]
}

// generateCanvasConfig 生成Canvas配置
func (fg *FingerprintGenerator) generateCanvasConfig(userRand *rand.Rand) CanvasConfig {
	return CanvasConfig{
		NoiseLevel:   userRand.Float64() * 0.01, // 0-1%的噪音
		TextVariance: userRand.Intn(5) + 1,      // 1-5像素的文本变化
	}
}

// generateFontsConfig 生成字体配置
func (fg *FingerprintGenerator) generateFontsConfig(userRand *rand.Rand) FontsConfig {
	allFonts := []string{
		"Arial", "Arial Black", "Comic Sans MS", "Courier New", "Georgia",
		"Helvetica", "Impact", "Lucida Console", "Tahoma", "Times New Roman",
		"Trebuchet MS", "Verdana", "Calibri", "Cambria", "Consolas",
		"Franklin Gothic Medium", "Garamond", "Gill Sans", "Segoe UI",
	}

	// 随机选择可用字体
	numFonts := userRand.Intn(8) + 8 // 8-15个字体
	availableFonts := make([]string, 0, numFonts)

	// 打乱字体顺序并选择前N个
	shuffled := make([]string, len(allFonts))
	copy(shuffled, allFonts)
	userRand.Shuffle(len(shuffled), func(i, j int) {
		shuffled[i], shuffled[j] = shuffled[j], shuffled[i]
	})

	for i := 0; i < numFonts && i < len(shuffled); i++ {
		availableFonts = append(availableFonts, shuffled[i])
	}

	return FontsConfig{
		AvailableFonts: availableFonts,
		BlockedFonts:   []string{}, // 暂时不阻止任何字体
	}
}

// generatePluginsConfig 生成插件配置
func (fg *FingerprintGenerator) generatePluginsConfig(userRand *rand.Rand) PluginsConfig {
	return PluginsConfig{
		EnabledPlugins:  []string{"PDF Viewer", "Chrome PDF Viewer"},
		DisabledPlugins: []string{},
	}
}

// generateBatteryConfig 生成电池配置
func (fg *FingerprintGenerator) generateBatteryConfig(userRand *rand.Rand) BatteryConfig {
	charging := userRand.Float64() < 0.6  // 60%概率在充电
	level := 0.2 + userRand.Float64()*0.7 // 20%-90%电量

	var chargingTime *float64
	var dischargingTime *float64

	if charging {
		ct := float64(userRand.Intn(7200)) // 0-2小时充电时间
		chargingTime = &ct
	} else {
		dt := float64(userRand.Intn(18000)) // 0-5小时放电时间
		dischargingTime = &dt
	}

	return BatteryConfig{
		Charging:        charging,
		ChargingTime:    chargingTime,
		DischargingTime: dischargingTime,
		Level:           level,
	}
}

// generateMediaDevicesConfig 生成媒体设备配置
func (fg *FingerprintGenerator) generateMediaDevicesConfig(userRand *rand.Rand) []MediaDevice {
	devices := []MediaDevice{
		{Kind: "audioinput", Label: "Default - MacBook Pro Microphone", DeviceID: fg.generateDeviceID(userRand)},
		{Kind: "audiooutput", Label: "Default - MacBook Pro Speakers", DeviceID: fg.generateDeviceID(userRand)},
	}

	// 随机添加摄像头
	if userRand.Float64() < 0.7 { // 70%概率有摄像头
		devices = append(devices, MediaDevice{
			Kind:     "videoinput",
			Label:    "FaceTime HD Camera",
			DeviceID: fg.generateDeviceID(userRand),
		})
	}

	return devices
}

// generateDeviceID 生成设备ID
func (fg *FingerprintGenerator) generateDeviceID(userRand *rand.Rand) string {
	chars := "abcdef0123456789"
	result := make([]byte, 32)
	for i := range result {
		result[i] = chars[userRand.Intn(len(chars))]
	}
	return string(result)
}

// generateTLSConfig 生成TLS配置
func (fg *FingerprintGenerator) generateTLSConfig(userRand *rand.Rand) TLSConfig {
	// Chrome常用的密码套件
	allCipherSuites := []string{
		"TLS_AES_128_GCM_SHA256",
		"TLS_AES_256_GCM_SHA384",
		"TLS_CHACHA20_POLY1305_SHA256",
		"TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256",
		"TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256",
		"TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384",
		"TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384",
		"TLS_ECDHE_ECDSA_WITH_CHACHA20_POLY1305_SHA256",
		"TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305_SHA256",
		"TLS_ECDHE_RSA_WITH_AES_128_CBC_SHA",
		"TLS_ECDHE_RSA_WITH_AES_256_CBC_SHA",
		"TLS_RSA_WITH_AES_128_GCM_SHA256",
		"TLS_RSA_WITH_AES_256_GCM_SHA384",
		"TLS_RSA_WITH_AES_128_CBC_SHA",
		"TLS_RSA_WITH_AES_256_CBC_SHA",
	}

	// 随机选择8-12个密码套件
	numCiphers := 8 + userRand.Intn(5)
	cipherSuites := make([]string, 0, numCiphers)
	shuffled := make([]string, len(allCipherSuites))
	copy(shuffled, allCipherSuites)
	userRand.Shuffle(len(shuffled), func(i, j int) {
		shuffled[i], shuffled[j] = shuffled[j], shuffled[i]
	})

	for i := 0; i < numCiphers && i < len(shuffled); i++ {
		cipherSuites = append(cipherSuites, shuffled[i])
	}

	// TLS扩展
	extensions := []string{
		"server_name",
		"ec_point_formats",
		"supported_groups",
		"session_ticket",
		"encrypt_then_mac",
		"extended_master_secret",
		"signature_algorithms",
		"supported_versions",
		"cookie",
		"psk_key_exchange_modes",
		"certificate_authorities",
		"oid_filters",
		"post_handshake_auth",
		"signature_algorithms_cert",
		"key_share",
	}

	// 椭圆曲线
	ellipticCurves := []string{
		"X25519",
		"secp256r1",
		"secp384r1",
	}

	// 签名算法
	signatureAlgs := []string{
		"ecdsa_secp256r1_sha256",
		"rsa_pss_rsae_sha256",
		"rsa_pkcs1_sha256",
		"ecdsa_secp384r1_sha384",
		"rsa_pss_rsae_sha384",
		"rsa_pkcs1_sha384",
		"rsa_pss_rsae_sha512",
		"rsa_pkcs1_sha512",
	}

	// 生成JA4指纹（简化版）
	tlsVersion := "TLS 1.3"
	ja4 := fmt.Sprintf("t13d%04d%04d_%02x%02x_%02x%02x",
		len(cipherSuites), len(extensions),
		userRand.Intn(256), userRand.Intn(256),
		userRand.Intn(256), userRand.Intn(256))

	// 生成JA3指纹（向后兼容）
	ja3 := fmt.Sprintf("%d,%s,%s,%s,%s",
		771, // TLS 1.2
		strings.Join(cipherSuites[:min(5, len(cipherSuites))], "-"),
		strings.Join(extensions[:min(10, len(extensions))], "-"),
		strings.Join(ellipticCurves, "-"),
		strings.Join([]string{"0", "1"}, "-"))

	return TLSConfig{
		JA4:            ja4,
		JA3:            ja3,
		TLSVersion:     tlsVersion,
		CipherSuites:   cipherSuites,
		Extensions:     extensions,
		EllipticCurves: ellipticCurves,
		ECPointFormats: []string{"uncompressed"},
		SignatureAlgs:  signatureAlgs,
	}
}

// generateHTTP2Config 生成HTTP2配置
func (fg *FingerprintGenerator) generateHTTP2Config(userRand *rand.Rand) HTTP2Config {
	// HTTP2设置（Chrome常用值的变体）
	settings := map[string]int{
		"SETTINGS_HEADER_TABLE_SIZE":      4096 + userRand.Intn(8192),   // 4096-12287
		"SETTINGS_ENABLE_PUSH":            userRand.Intn(2),             // 0或1
		"SETTINGS_MAX_CONCURRENT_STREAMS": 1000 + userRand.Intn(1000),   // 1000-1999
		"SETTINGS_INITIAL_WINDOW_SIZE":    65535 + userRand.Intn(65536), // 65535-131070
		"SETTINGS_MAX_FRAME_SIZE":         16384 + userRand.Intn(32768), // 16384-49151
		"SETTINGS_MAX_HEADER_LIST_SIZE":   10240 + userRand.Intn(10240), // 10240-20479
	}

	// 窗口更新值
	windowUpdate := 15663105 + userRand.Intn(1000000) // Chrome典型值附近

	// 头部优先级
	headerPriority := map[string]int{
		"weight":     userRand.Intn(256), // 0-255
		"depends_on": userRand.Intn(10),  // 依赖的流ID
		"exclusive":  userRand.Intn(2),   // 0或1
	}

	// 伪头部顺序（Chrome特有的顺序）
	pseudoHeaders := []string{
		":method",
		":authority",
		":scheme",
		":path",
	}

	// 生成Akamai指纹（简化版）
	akamai := fmt.Sprintf("%d:%d:%d:%d:%d:%d",
		settings["SETTINGS_HEADER_TABLE_SIZE"],
		settings["SETTINGS_ENABLE_PUSH"],
		settings["SETTINGS_MAX_CONCURRENT_STREAMS"],
		settings["SETTINGS_INITIAL_WINDOW_SIZE"],
		settings["SETTINGS_MAX_FRAME_SIZE"],
		windowUpdate)

	return HTTP2Config{
		AKAMAI:         akamai,
		Settings:       settings,
		WindowUpdate:   windowUpdate,
		HeaderPriority: headerPriority,
		PseudoHeaders:  pseudoHeaders,
	}
}

// min 返回两个整数中的较小值
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// SaveToFile 保存配置到文件
func (config *FingerprintConfig) SaveToFile(filepath string) error {
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return err
	}

	return writeFile(filepath, data)
}

// LoadFromFile 从文件加载配置
func LoadFingerprintConfig(filepath string) (*FingerprintConfig, error) {
	data, err := readFile(filepath)
	if err != nil {
		return nil, err
	}

	var config FingerprintConfig
	err = json.Unmarshal(data, &config)
	if err != nil {
		return nil, err
	}

	return &config, nil
}

// ToJSON 转换为JSON字符串
func (config *FingerprintConfig) ToJSON() (string, error) {
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// GetChromeFlags 根据指纹配置生成Chrome启动参数
func (config *FingerprintConfig) GetChromeFlags() []string {
	var flags []string

	// 用户代理
	flags = append(flags, "--user-agent="+config.Browser.UserAgent)

	// 语言设置
	flags = append(flags, "--lang="+config.Browser.Language)

	// 时区设置
	flags = append(flags, "--tz="+config.Timezone.Timezone)

	// 屏幕尺寸（窗口大小）
	windowSize := fmt.Sprintf("%d,%d", config.Screen.Width, config.Screen.Height)
	flags = append(flags, "--window-size="+windowSize)

	// WebGL相关
	if config.WebGL.Vendor != "" {
		flags = append(flags, "--use-gl=desktop")
	}

	// TLS/SSL相关设置
	if len(config.TLSConfig.CipherSuites) > 0 {
		// 设置TLS版本
		if strings.Contains(config.TLSConfig.TLSVersion, "1.3") {
			flags = append(flags, "--enable-features=TLS13KeyUpdate")
		}

		// 设置SSL版本回退最小值
		flags = append(flags, "--ssl-version-fallback-min=tls1.2")

		// 启用特定的TLS特性
		flags = append(flags, "--enable-features=TLSTokenBinding")
	}

	// HTTP/2相关设置
	if config.HTTP2Config.AKAMAI != "" {
		// 启用HTTP/2
		flags = append(flags, "--enable-features=HTTP2")

		// 设置HTTP/2的初始窗口大小
		if config.HTTP2Config.WindowUpdate > 0 {
			flags = append(flags, fmt.Sprintf("--http2-initial-window-size=%d", config.HTTP2Config.WindowUpdate))
		}

		// 设置HTTP/2最大并发流数
		if maxStreams, ok := config.HTTP2Config.Settings["SETTINGS_MAX_CONCURRENT_STREAMS"]; ok && maxStreams > 0 {
			flags = append(flags, fmt.Sprintf("--http2-max-concurrent-streams=%d", maxStreams))
		}

		// 设置头部表大小
		if headerTableSize, ok := config.HTTP2Config.Settings["SETTINGS_HEADER_TABLE_SIZE"]; ok && headerTableSize > 0 {
			flags = append(flags, fmt.Sprintf("--http2-settings-header-table-size=%d", headerTableSize))
		}
	}

	// 音频采样率相关
	if config.Audio.SampleRate != 48000 { // 如果不是默认值
		// Chrome没有直接的音频采样率参数，但可以通过其他方式影响
		flags = append(flags, fmt.Sprintf("--audio-sample-rate=%d", config.Audio.SampleRate))
	}

	// Canvas和WebGL硬件加速
	if config.Canvas.NoiseLevel > 0 {
		// 启用Canvas 2D GPU加速以确保Canvas指纹正常工作
		flags = append(flags, "--enable-accelerated-2d-canvas")
	}

	return flags
}

// 辅助函数（需要实现文件读写）
func writeFile(filepath string, data []byte) error {
	// 这里需要实现文件写入逻辑
	// 在实际项目中应该使用 ioutil.WriteFile 或类似函数
	return nil
}

func readFile(filepath string) ([]byte, error) {
	// 这里需要实现文件读取逻辑
	// 在实际项目中应该使用 ioutil.ReadFile 或类似函数
	return nil, nil
}
