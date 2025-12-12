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
	UserID     string `json:"user_id"`
	
	// 屏幕相关
	Screen     ScreenConfig     `json:"screen"`
	
	// 浏览器相关
	Browser    BrowserConfig    `json:"browser"`
	
	// 系统相关
	System     SystemConfig     `json:"system"`
	
	// WebGL相关
	WebGL      WebGLConfig      `json:"webgl"`
	
	// 音频相关
	Audio      AudioConfig      `json:"audio"`
	
	// 网络相关
	Network    NetworkConfig    `json:"network"`
	
	// 时区相关
	Timezone   TimezoneConfig   `json:"timezone"`
	
	// Canvas相关
	Canvas     CanvasConfig     `json:"canvas"`
	
	// 字体相关
	Fonts      FontsConfig      `json:"fonts"`
	
	// 插件相关
	Plugins    PluginsConfig    `json:"plugins"`
	
	// 电池相关
	Battery    BatteryConfig    `json:"battery"`
	
	// 媒体设备
	MediaDevices []MediaDevice  `json:"media_devices"`
	
	// TLS/JA4指纹
	TLSConfig    TLSConfig      `json:"tls_config"`
	
	// HTTP2指纹
	HTTP2Config  HTTP2Config    `json:"http2_config"`
}

// ScreenConfig 屏幕配置
type ScreenConfig struct {
	Width             int     `json:"width"`
	Height            int     `json:"height"`
	AvailWidth        int     `json:"avail_width"`
	AvailHeight       int     `json:"avail_height"`
	ColorDepth        int     `json:"color_depth"`
	PixelDepth        int     `json:"pixel_depth"`
	DevicePixelRatio  float64 `json:"device_pixel_ratio"`
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
	OS          string `json:"os"`
	OSVersion   string `json:"os_version"`
	Architecture string `json:"architecture"`
}

// WebGLConfig WebGL配置
type WebGLConfig struct {
	Vendor                   string `json:"vendor"`
	Renderer                 string `json:"renderer"`
	Version                  string `json:"version"`
	ShadingLanguageVersion   string `json:"shading_language_version"`
	MaxTextureSize           int    `json:"max_texture_size"`
	MaxRenderbufferSize      int    `json:"max_renderbuffer_size"`
}

// AudioConfig 音频配置
type AudioConfig struct {
	SampleRate       int `json:"sample_rate"`
	MaxChannelCount  int `json:"max_channel_count"`
	NumberOfInputs   int `json:"number_of_inputs"`
	NumberOfOutputs  int `json:"number_of_outputs"`
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
	Charging         bool     `json:"charging"`
	ChargingTime     *float64 `json:"charging_time"`
	DischargingTime  *float64 `json:"discharging_time"`
	Level            float64  `json:"level"`
}

// MediaDevice 媒体设备
type MediaDevice struct {
	Kind     string `json:"kind"`
	Label    string `json:"label"`
	DeviceID string `json:"device_id"`
}

// TLSConfig TLS指纹配置
type TLSConfig struct {
	JA4              string   `json:"ja4"`               // JA4指纹
	JA3              string   `json:"ja3"`               // JA3指纹 (向后兼容)
	TLSVersion       string   `json:"tls_version"`       // TLS版本
	CipherSuites     []string `json:"cipher_suites"`     // 密码套件
	Extensions       []string `json:"extensions"`        // TLS扩展
	EllipticCurves   []string `json:"elliptic_curves"`   // 椭圆曲线
	ECPointFormats   []string `json:"ec_point_formats"`  // EC点格式
	SignatureAlgs    []string `json:"signature_algs"`    // 签名算法
}

// HTTP2Config HTTP2指纹配置
type HTTP2Config struct {
	AKAMAI           string            `json:"akamai"`            // Akamai指纹
	Settings         map[string]int    `json:"settings"`          // HTTP2设置
	WindowUpdate     int               `json:"window_update"`     // 窗口更新
	HeaderPriority   map[string]int    `json:"header_priority"`   // 头部优先级
	PseudoHeaders    []string          `json:"pseudo_headers"`    // 伪头部顺序
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
	// 常见屏幕分辨率
	resolutions := [][]int{
		{1920, 1080}, {1366, 768}, {1536, 864}, {1440, 900},
		{1280, 720}, {1600, 900}, {2560, 1440}, {3840, 2160},
		{1280, 1024}, {1680, 1050}, {2560, 1600}, {1920, 1200},
	}
	
	resolution := resolutions[userRand.Intn(len(resolutions))]
	width, height := resolution[0], resolution[1]
	
	// 设备像素比
	devicePixelRatios := []float64{1.0, 1.25, 1.5, 2.0, 2.5, 3.0}
	devicePixelRatio := devicePixelRatios[userRand.Intn(len(devicePixelRatios))]
	
	return ScreenConfig{
		Width:            width,
		Height:           height,
		AvailWidth:       width,
		AvailHeight:      height - userRand.Intn(100), // 任务栏高度变化
		ColorDepth:       24,
		PixelDepth:       24,
		DevicePixelRatio: devicePixelRatio,
	}
}

// generateBrowserConfig 生成浏览器配置
func (fg *FingerprintGenerator) generateBrowserConfig(userRand *rand.Rand) BrowserConfig {
	// Chrome版本
	chromeVersions := []string{
		"138.0.0.0", "137.0.0.0", "136.0.0.0", "135.0.0.0",
		"134.0.0.0", "133.0.0.0", "132.0.0.0", "131.0.0.0",
	}
	chromeVersion := chromeVersions[userRand.Intn(len(chromeVersions))]
	
	// 操作系统
	platforms := []string{"MacIntel", "Win32", "Linux x86_64"}
	platform := platforms[userRand.Intn(len(platforms))]
	
	// 构建UserAgent
	var userAgent string
	switch platform {
	case "MacIntel":
		macVersions := []string{"10_15_7", "11_0_0", "12_0_0", "13_0_0", "14_0_0"}
		macVersion := macVersions[userRand.Intn(len(macVersions))]
		userAgent = fmt.Sprintf("Mozilla/5.0 (Macintosh; Intel Mac OS X %s) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/%s Safari/537.36", macVersion, chromeVersion)
	case "Win32":
		userAgent = fmt.Sprintf("Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/%s Safari/537.36", chromeVersion)
	case "Linux x86_64":
		userAgent = fmt.Sprintf("Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/%s Safari/537.36", chromeVersion)
	}
	
	// 语言配置
	languages := [][]string{
		{"en-US", "en"},
		{"zh-CN", "zh"},
		{"ja-JP", "ja"},
		{"ko-KR", "ko"},
		{"de-DE", "de"},
		{"fr-FR", "fr"},
		{"es-ES", "es"},
		{"pt-BR", "pt"},
	}
	selectedLangs := languages[userRand.Intn(len(languages))]
	
	// 硬件并发数（CPU核心数）
	hardwareConcurrencies := []int{2, 4, 6, 8, 12, 16, 20, 24}
	hardwareConcurrency := hardwareConcurrencies[userRand.Intn(len(hardwareConcurrencies))]
	
	return BrowserConfig{
		UserAgent:           userAgent,
		Language:            selectedLangs[0],
		Languages:           selectedLangs,
		Platform:            platform,
		Vendor:              "Google Inc.",
		CookieEnabled:       true,
		DoNotTrack:          nil, // 通常为null
		HardwareConcurrency: hardwareConcurrency,
		MaxTouchPoints:      0, // 桌面设备通常为0
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
	webglConfigs := []WebGLConfig{
		{
			Vendor:                 "WebKit",
			Renderer:               "WebKit WebGL",
			Version:                "WebGL 1.0 (OpenGL ES 2.0 Chromium)",
			ShadingLanguageVersion: "WebGL GLSL ES 1.0 (OpenGL ES GLSL ES 1.0 Chromium)",
			MaxTextureSize:         16384,
			MaxRenderbufferSize:    16384,
		},
		{
			Vendor:                 "Google Inc. (Intel)",
			Renderer:               "ANGLE (Intel, Intel(R) UHD Graphics 630 Direct3D11 vs_5_0 ps_5_0, D3D11)",
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
	charging := userRand.Float64() < 0.6 // 60%概率在充电
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
		"SETTINGS_ENABLE_PUSH":            userRand.Intn(2),              // 0或1
		"SETTINGS_MAX_CONCURRENT_STREAMS": 1000 + userRand.Intn(1000),   // 1000-1999
		"SETTINGS_INITIAL_WINDOW_SIZE":    65535 + userRand.Intn(65536), // 65535-131070
		"SETTINGS_MAX_FRAME_SIZE":         16384 + userRand.Intn(32768), // 16384-49151
		"SETTINGS_MAX_HEADER_LIST_SIZE":   10240 + userRand.Intn(10240), // 10240-20479
	}
	
	// 窗口更新值
	windowUpdate := 15663105 + userRand.Intn(1000000) // Chrome典型值附近
	
	// 头部优先级
	headerPriority := map[string]int{
		"weight":    userRand.Intn(256),     // 0-255
		"depends_on": userRand.Intn(10),      // 依赖的流ID
		"exclusive":  userRand.Intn(2),       // 0或1
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