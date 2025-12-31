package config

import (
	"fmt"
	"path/filepath"
	"runtime"
	"strings"
)

// DefaultChromeFlags returns Chrome flags EXACTLY matching chrome-launcher DEFAULT_FLAGS
// This is critical for anti-detection - any difference can be detected
func DefaultChromeFlags() []string {
	// CRITICAL: These must match chrome-launcher DEFAULT_FLAGS exactly
	// Source: https://github.com/GoogleChrome/chrome-launcher/blob/main/src/flags.ts
	flags := []string{
		"--disable-extensions",
		"--disable-background-networking",
		"--disable-background-timer-throttling",
		"--disable-renderer-backgrounding",
		"--disable-backgrounding-occluded-windows",
		"--disable-client-side-phishing-detection",
		"--disable-default-apps",
		"--disable-dev-shm-usage",
		"--disable-hang-monitor",
		"--disable-ipc-flooding-protection",
		"--disable-popup-blocking",
		"--disable-prompt-on-repost",
		"--disable-sync",
		"--disable-features=Translate,BackForwardCache,AcceptCHFrame,MediaRouter,OptimizationHints,DialMediaRouteProvider,CalculateNativeWinOcclusion,InterestFeedContentSuggestions,CertificateTransparencyComponentUpdater,AutofillServerCommunication,PrivacySandboxSettings4",
		"--enable-features=NetworkService,NetworkServiceLogging",
		"--disable-component-extensions-with-background-pages",
		"--disable-breakpad",
		"--force-color-profile=srgb",
		"--metrics-recording-only",
		"--no-first-run",
		"--password-store=basic",
		"--use-mock-keychain",
		"--enable-blink-features=IdleDetection",
		"--export-tagged-pdf",
		"--mute-audio",

		// CRITICAL ANTI-DETECTION FLAGS - These are absolutely essential!
		"--exclude-switches=enable-automation", // Exclude the automation switch
		"--disable-infobars",                   // Try to disable infobars (may not work in newer Chrome)
	}

	// 关键修改：添加AutomationControlled到disable-features
	for i, flag := range flags {
		if strings.HasPrefix(flag, "--disable-features=") {
			flags[i] = flag + ",AutomationControlled"
			break
		}
	}

	// 添加关键的反自动化检测标志
	flags = append(flags, "--disable-blink-features=AutomationControlled")

	// 关键修改：移除disable-component-update标志（类似原版JS）
	filteredFlags := make([]string, 0, len(flags))
	for _, flag := range flags {
		if !strings.HasPrefix(flag, "--disable-component-update") {
			filteredFlags = append(filteredFlags, flag)
		}
	}

	// Add platform-specific flags
	if runtime.GOOS == "linux" {
		filteredFlags = append(filteredFlags, "--no-sandbox")
	}

	return filteredFlags
}

// GetStealthFlags returns MINIMAL stealth flags to match original Node.js version
// The original JS version only adds --no-sandbox and --disable-dev-shm-usage as extra flags
func GetStealthFlags() []string {
	// CRITICAL: Keep this minimal to match original implementation
	// The original Node.js version doesn't add many extra stealth flags
	return []string{
		"--no-sandbox",
		"--disable-dev-shm-usage",
		"--test-type", // Suppress "unsupported command-line flag" warning
	}
}

// GetCacheFlags returns flags for optimizing browser cache
// 缓存优化参数 - 显著提升页面加载速度
func GetCacheFlags(cacheDir string, cacheSizeMB int) []string {
	flags := []string{
		// 启用激进缓存模式
		"--aggressive-cache-discard=false",
		// 磁盘缓存大小 (字节)
		"--disk-cache-size=" + fmt.Sprintf("%d", cacheSizeMB*1024*1024),
		// 媒体缓存大小
		"--media-cache-size=" + fmt.Sprintf("%d", cacheSizeMB*1024*1024/2),
	}
	
	// 如果指定了缓存目录
	if cacheDir != "" {
		flags = append(flags, "--disk-cache-dir="+cacheDir)
	}
	
	return flags
}

// GetHeadlessFlags returns flags for headless mode
func GetHeadlessFlags(headless interface{}) []string {
	var flags []string

	switch v := headless.(type) {
	case bool:
		if v {
			flags = append(flags, "--headless")
		}
	case string:
		if v != "" && v != "false" {
			flags = append(flags, "--headless="+v)
		}
	}

	return flags
}

// GetDefaultExtensionPaths returns the default extension paths
func GetDefaultExtensionPaths() []string {
	// 返回默认的扩展路径 - 使用未打包扩展目录（ChromeDP要求）
	return []string{
		"examples/path/Extensions/kfjglmgfjedhhcddpfgfogkahmenikan/1.0_0",   // Discord Token Login (unpacked)
		"examples/path/Extensions/mcohilncbfahbmgdjkbpemcciiolgcge/3.66.10_0", // OKX Wallet (unpacked)
	}
}

// GetExtensionFlags returns flags for loading Chrome extensions
func GetExtensionFlags(extensions []string) []string {
	if len(extensions) == 0 {
		return []string{}
	}

	var flags []string
	var extensionPaths []string

	for _, ext := range extensions {
		if ext != "" {
			extensionPaths = append(extensionPaths, ext)
		}
	}

	if len(extensionPaths) > 0 {
		// 当有扩展时，启用扩展功能
		flags = append(flags, "--enable-extensions")
		
		// 关键发现：--disable-extensions-except 也支持逗号分隔的多个路径！
		// 设置格式：--disable-extensions-except=/path/to/ext1,/path/to/ext2
		extensionPathsStr := strings.Join(extensionPaths, ",")
		flags = append(flags, "--disable-extensions-except="+extensionPathsStr)

		// 加载扩展的方式 - 使用绝对路径
		var absolutePaths []string
		for _, path := range extensionPaths {
			// 确保使用绝对路径
			if !strings.HasPrefix(path, "/") {
				// 如果是相对路径，转换为绝对路径
				if absPath, err := filepath.Abs(path); err == nil {
					absolutePaths = append(absolutePaths, absPath)
				} else {
					absolutePaths = append(absolutePaths, path)
				}
			} else {
				absolutePaths = append(absolutePaths, path)
			}
		}

		absolutePathsStr := strings.Join(absolutePaths, ",")
		flags = append(flags, "--load-extension="+absolutePathsStr)

		// 强制启用开发者模式相关标志
		flags = append(flags, "--enable-extension-activity-logging")
		flags = append(flags, "--enable-logging")

		// 只保留必要的标志，移除可能与冲突解决矛盾的标志
		flags = append(flags, "--allow-running-insecure-content")
		flags = append(flags, "--disable-web-security")
		flags = append(flags, "--allow-file-access-from-files")

		// 确保没有冲突的标志
		flags = append(flags, "--disable-default-apps")
	}

	return flags
}

// GetProxyFlags returns proxy configuration flags
func GetProxyFlags(host, port string) []string {
	if host != "" && port != "" {
		return []string{
			"--proxy-server=" + host + ":" + port,
		}
	}
	return []string{}
}

// MergeFlags combines multiple flag arrays and removes duplicates and conflicts
func MergeFlags(flagArrays ...[]string) []string {
	seen := make(map[string]bool)
	var result []string

	// 首先收集所有标志
	for _, flags := range flagArrays {
		for _, flag := range flags {
			if !seen[flag] {
				seen[flag] = true
				result = append(result, flag)
			}
		}
	}

	// 处理冲突的标志
	result = resolveConflictingFlags(result)

	return result
}

// resolveConflictingFlags removes conflicting flags and keeps the most appropriate ones
func resolveConflictingFlags(flags []string) []string {
	hasEnableExtensions := false
	hasLoadExtension := false

	// 检查是否有扩展相关标志
	for _, flag := range flags {
		if flag == "--enable-extensions" {
			hasEnableExtensions = true
		}
		if strings.HasPrefix(flag, "--load-extension=") {
			hasLoadExtension = true
		}
	}

	var result []string
	for _, flag := range flags {
		// 如果有扩展相关标志，跳过所有可能冲突的扩展disable标志
		if hasEnableExtensions || hasLoadExtension {
			if flag == "--disable-extensions" ||
			   flag == "--disable-component-extensions-with-background-pages" ||
			   flag == "--disable-background-timer-throttling" ||
			   flag == "--disable-backgrounding-occluded-windows" ||
			   flag == "--disable-extensions-file-access-check" ||
			   flag == "--disable-extensions-http-throttling" ||
			   flag == "--disable-component-update" ||
			   flag == "--disable-extensions-install-verification" ||
			   strings.Contains(flag, "--disable-features=") { // 移除可能包含扩展限制的features
				continue
			}
		}
		result = append(result, flag)
	}

	return result
}
