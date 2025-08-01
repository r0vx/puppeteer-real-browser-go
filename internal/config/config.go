package config

import (
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
	}
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

// GetProxyFlags returns proxy configuration flags
func GetProxyFlags(host, port string) []string {
	if host != "" && port != "" {
		return []string{
			"--proxy-server=" + host + ":" + port,
		}
	}
	return []string{}
}

// MergeFlags combines multiple flag arrays and removes duplicates
func MergeFlags(flagArrays ...[]string) []string {
	seen := make(map[string]bool)
	var result []string

	for _, flags := range flagArrays {
		for _, flag := range flags {
			if !seen[flag] {
				seen[flag] = true
				result = append(result, flag)
			}
		}
	}

	return result
}
