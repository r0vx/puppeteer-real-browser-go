package main

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"log"
	"math/big"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/HNRow/puppeteer-real-browser-go/pkg/browser"
)

// BrowserFingerprint defines a complete browser fingerprint
type BrowserFingerprint struct {
	ProfileName string   `json:"profileName"`
	UserAgent   string   `json:"userAgent"`
	Platform    string   `json:"platform"`
	Language    []string `json:"language"`
	Timezone    string   `json:"timezone"`
	Screen      struct {
		Width  int `json:"width"`
		Height int `json:"height"`
	} `json:"screen"`
	Viewport struct {
		Width  int `json:"width"`
		Height int `json:"height"`
	} `json:"viewport"`
	WebGL struct {
		Vendor   string `json:"vendor"`
		Renderer string `json:"renderer"`
	} `json:"webgl"`
	Canvas    string               `json:"canvas"`
	Proxy     *browser.ProxyConfig `json:"proxy,omitempty"`
	CreatedAt time.Time            `json:"createdAt"`
}

// FingerprintManager manages browser fingerprints and profiles
type FingerprintManager struct {
	profiles    map[string]*BrowserFingerprint
	browsers    map[string]*FingerprintBrowser
	storageDir  string
	mutex       sync.RWMutex
	mainBrowser *browser.BrowserInstance
}

// FingerprintBrowser represents a browser instance with a specific fingerprint
type FingerprintBrowser struct {
	Profile *BrowserFingerprint
	Context *browser.BrowserContext
	Page    browser.Page
}

func main() {
	fmt.Println("🎭 Fingerprint Browser Demo")
	fmt.Println("============================")

	// 创建指纹管理器
	manager, err := NewFingerprintManager("./fingerprint_profiles")
	if err != nil {
		log.Fatalf("Failed to create fingerprint manager: %v", err)
	}
	defer manager.Close()

	// 示例1: 创建多个不同的浏览器指纹
	fmt.Println("\n🎲 Demo 1: Creating Random Fingerprints")
	profiles := []string{"user_usa_windows", "user_uk_mac", "user_de_linux", "user_jp_mobile"}

	for _, profileName := range profiles {
		fingerprint := GenerateRandomFingerprint(profileName)
		if err := manager.SaveProfile(fingerprint); err != nil {
			log.Printf("Failed to save profile %s: %v", profileName, err)
			continue
		}
		fmt.Printf("  ✅ Created profile: %s (%s %s)\n",
			profileName, fingerprint.Platform, fingerprint.UserAgent[:50]+"...")
	}

	// 示例2: 使用不同指纹并行访问网站
	fmt.Println("\n🌐 Demo 2: Parallel Browsing with Different Fingerprints")
	testURLs := []string{
		"https://httpbin.org/headers",
		"https://httpbin.org/user-agent",
		"https://httpbin.org/ip",
		"https://httpbin.org/get",
	}

	var wg sync.WaitGroup
	for i, profileName := range profiles {
		if i >= len(testURLs) {
			break
		}

		wg.Add(1)
		go func(profile string, url string) {
			defer wg.Done()
			if err := manager.BrowseWithProfile(profile, url); err != nil {
				log.Printf("Failed to browse with profile %s: %v", profile, err)
			}
		}(profileName, testURLs[i])
	}
	wg.Wait()

	// 示例3: 指纹验证和检测测试
	fmt.Println("\n🔍 Demo 3: Fingerprint Detection Test")
	if err := manager.TestFingerprintDetection(); err != nil {
		log.Printf("Fingerprint detection test failed: %v", err)
	}

	// 示例4: 电商多账号应用场景
	fmt.Println("\n🛒 Demo 4: E-commerce Multi-Account Scenario")
	if err := manager.EcommerceScenario(); err != nil {
		log.Printf("E-commerce scenario failed: %v", err)
	}

	fmt.Println("\n⏳ Keeping browsers open for 45 seconds for inspection...")
	time.Sleep(300 * time.Second)

	fmt.Println("✅ Fingerprint Browser Demo completed!")
}

// NewFingerprintManager creates a new fingerprint manager
func NewFingerprintManager(storageDir string) (*FingerprintManager, error) {
	// 确保存储目录存在
	if err := os.MkdirAll(storageDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create storage directory: %w", err)
	}

	// 创建主浏览器实例
	ctx := context.Background()
	opts := &browser.ConnectOptions{
		Headless:     false,
		UseCustomCDP: true,
		Turnstile:    true,
		Args: []string{
			"--start-maximized",
			"--disable-blink-features=AutomationControlled",
			"--exclude-switches=enable-automation",
		},
	}

	mainBrowser, err := browser.Connect(ctx, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to connect main browser: %w", err)
	}

	manager := &FingerprintManager{
		profiles:    make(map[string]*BrowserFingerprint),
		browsers:    make(map[string]*FingerprintBrowser),
		storageDir:  storageDir,
		mainBrowser: mainBrowser,
	}

	// 加载已存在的配置文件
	if err := manager.LoadProfiles(); err != nil {
		log.Printf("Warning: Failed to load existing profiles: %v", err)
	}

	return manager, nil
}

// GenerateRandomFingerprint generates a random browser fingerprint
func GenerateRandomFingerprint(profileName string) *BrowserFingerprint {
	userAgents := map[string][]string{
		"Windows": {
			"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36",
			"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/119.0.0.0 Safari/537.36",
			"Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:109.0) Gecko/20100101 Firefox/121.0",
		},
		"macOS": {
			"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36",
			"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/17.1 Safari/605.1.15",
		},
		"Linux": {
			"Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36",
			"Mozilla/5.0 (X11; Ubuntu; Linux x86_64; rv:109.0) Gecko/20100101 Firefox/121.0",
		},
	}

	platforms := []string{"Windows", "macOS", "Linux"}
	timezones := []string{
		"America/New_York", "America/Los_Angeles", "Europe/London",
		"Europe/Berlin", "Asia/Tokyo", "Asia/Shanghai", "Australia/Sydney",
	}

	languages := [][]string{
		{"en-US", "en"},
		{"en-GB", "en"},
		{"de-DE", "de", "en"},
		{"fr-FR", "fr", "en"},
		{"ja-JP", "ja", "en"},
		{"zh-CN", "zh", "en"},
	}

	// 随机选择平台
	platform := platforms[randomInt(len(platforms))]

	// 根据平台选择合适的用户代理
	userAgent := userAgents[platform][randomInt(len(userAgents[platform]))]

	// 生成合理的屏幕和视口尺寸
	var screenWidth, screenHeight, viewportWidth, viewportHeight int
	switch platform {
	case "Windows":
		screens := [][2]int{{1920, 1080}, {1366, 768}, {1536, 864}, {1440, 900}}
		screen := screens[randomInt(len(screens))]
		screenWidth, screenHeight = screen[0], screen[1]
		viewportWidth = screenWidth - randomInt(100) - 50
		viewportHeight = screenHeight - randomInt(200) - 100
	case "macOS":
		screens := [][2]int{{2560, 1600}, {1440, 900}, {1680, 1050}}
		screen := screens[randomInt(len(screens))]
		screenWidth, screenHeight = screen[0], screen[1]
		viewportWidth = screenWidth - randomInt(100) - 50
		viewportHeight = screenHeight - randomInt(200) - 100
	case "Linux":
		screens := [][2]int{{1920, 1080}, {1600, 900}, {1280, 1024}}
		screen := screens[randomInt(len(screens))]
		screenWidth, screenHeight = screen[0], screen[1]
		viewportWidth = screenWidth - randomInt(100) - 50
		viewportHeight = screenHeight - randomInt(200) - 100
	}

	fingerprint := &BrowserFingerprint{
		ProfileName: profileName,
		UserAgent:   userAgent,
		Platform:    platform,
		Language:    languages[randomInt(len(languages))],
		Timezone:    timezones[randomInt(len(timezones))],
		CreatedAt:   time.Now(),
	}

	fingerprint.Screen.Width = screenWidth
	fingerprint.Screen.Height = screenHeight
	fingerprint.Viewport.Width = viewportWidth
	fingerprint.Viewport.Height = viewportHeight

	// 生成 WebGL 指纹
	webglVendors := []string{"Google Inc.", "Apple Inc.", "Mesa"}
	webglRenderers := []string{
		"ANGLE (NVIDIA GeForce GTX 1060 6GB Direct3D11 vs_5_0 ps_5_0)",
		"Apple GPU",
		"Mesa DRI Intel(R) UHD Graphics 620",
	}
	fingerprint.WebGL.Vendor = webglVendors[randomInt(len(webglVendors))]
	fingerprint.WebGL.Renderer = webglRenderers[randomInt(len(webglRenderers))]

	// 生成 Canvas 指纹（简化版）
	fingerprint.Canvas = fmt.Sprintf("canvas_%d_%d", randomInt(1000000), time.Now().UnixNano()%1000000)

	return fingerprint
}

// SaveProfile saves a fingerprint profile to disk
func (fm *FingerprintManager) SaveProfile(fingerprint *BrowserFingerprint) error {
	fm.mutex.Lock()
	defer fm.mutex.Unlock()

	fm.profiles[fingerprint.ProfileName] = fingerprint

	// 保存到文件
	filename := filepath.Join(fm.storageDir, fingerprint.ProfileName+".json")
	data, err := json.MarshalIndent(fingerprint, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal fingerprint: %w", err)
	}

	return os.WriteFile(filename, data, 0644)
}

// LoadProfiles loads all profiles from disk
func (fm *FingerprintManager) LoadProfiles() error {
	files, err := filepath.Glob(filepath.Join(fm.storageDir, "*.json"))
	if err != nil {
		return err
	}

	for _, file := range files {
		data, err := os.ReadFile(file)
		if err != nil {
			log.Printf("Failed to read profile file %s: %v", file, err)
			continue
		}

		var fingerprint BrowserFingerprint
		if err := json.Unmarshal(data, &fingerprint); err != nil {
			log.Printf("Failed to unmarshal profile file %s: %v", file, err)
			continue
		}

		fm.mutex.Lock()
		fm.profiles[fingerprint.ProfileName] = &fingerprint
		fm.mutex.Unlock()
	}

	return nil
}

// BrowseWithProfile opens a page using a specific fingerprint profile
func (fm *FingerprintManager) BrowseWithProfile(profileName, url string) error {
	fm.mutex.RLock()
	profile := fm.profiles[profileName]
	fm.mutex.RUnlock()

	if profile == nil {
		return fmt.Errorf("profile %s not found", profileName)
	}

	// 创建专用的浏览器上下文
	contextOpts := &browser.BrowserContextOptions{
		IgnoreHTTPSErrors: true,
	}

	ctx, err := fm.mainBrowser.CreateBrowserContext(contextOpts)
	if err != nil {
		return fmt.Errorf("failed to create browser context: %w", err)
	}

	// 创建页面
	page, err := ctx.NewPage()
	if err != nil {
		return fmt.Errorf("failed to create page: %w", err)
	}

	// 应用指纹设置
	if err := fm.applyFingerprint(page, profile); err != nil {
		return fmt.Errorf("failed to apply fingerprint: %w", err)
	}

	// 导航到目标 URL
	if err := page.Navigate(url); err != nil {
		return fmt.Errorf("failed to navigate to %s: %w", url, err)
	}

	// 存储浏览器实例
	fm.mutex.Lock()
	fm.browsers[profileName] = &FingerprintBrowser{
		Profile: profile,
		Context: ctx,
		Page:    page,
	}
	fm.mutex.Unlock()

	fmt.Printf("  🌐 %s: Navigated to %s\n", profileName, url)
	return nil
}

// applyFingerprint applies fingerprint settings to a page
func (fm *FingerprintManager) applyFingerprint(page browser.Page, fingerprint *BrowserFingerprint) error {
	// 设置视口
	if err := page.SetViewport(fingerprint.Viewport.Width, fingerprint.Viewport.Height); err != nil {
		log.Printf("Failed to set viewport: %v", err)
	}

	// 注入指纹脚本
	fingerprintScript := fmt.Sprintf(`
		// Override screen properties
		Object.defineProperty(screen, 'width', {
			get: () => %d
		});
		Object.defineProperty(screen, 'height', {
			get: () => %d
		});

		// Override navigator properties
		Object.defineProperty(navigator, 'platform', {
			get: () => '%s'
		});
		Object.defineProperty(navigator, 'language', {
			get: () => '%s'
		});
		Object.defineProperty(navigator, 'languages', {
			get: () => %s
		});

		// Override timezone
		if (Intl && Intl.DateTimeFormat) {
			const originalResolvedOptions = Intl.DateTimeFormat.prototype.resolvedOptions;
			Intl.DateTimeFormat.prototype.resolvedOptions = function() {
				const options = originalResolvedOptions.call(this);
				options.timeZone = '%s';
				return options;
			};
		}

		// Override WebGL fingerprint
		const getContext = HTMLCanvasElement.prototype.getContext;
		HTMLCanvasElement.prototype.getContext = function(contextType, contextAttributes) {
			const context = getContext.call(this, contextType, contextAttributes);
			if (contextType === 'webgl' || contextType === 'experimental-webgl') {
				const getExtension = context.getExtension;
				context.getExtension = function(name) {
					if (name === 'WEBGL_debug_renderer_info') {
						const ext = getExtension.call(this, name);
						if (ext) {
							Object.defineProperty(ext, 'UNMASKED_VENDOR_WEBGL', {
								value: 37445
							});
							Object.defineProperty(ext, 'UNMASKED_RENDERER_WEBGL', {
								value: 37446
							});
						}
						return ext;
					}
					return getExtension.call(this, name);
				};

				const getParameter = context.getParameter;
				context.getParameter = function(parameter) {
					if (parameter === 37445) {
						return '%s';
					}
					if (parameter === 37446) {
						return '%s';
					}
					return getParameter.call(this, parameter);
				};
			}
			return context;
		};

		// Canvas fingerprint modification
		const toDataURL = HTMLCanvasElement.prototype.toDataURL;
		HTMLCanvasElement.prototype.toDataURL = function() {
			const result = toDataURL.apply(this, arguments);
			// Add slight variation to canvas fingerprint
			return result.replace(/.$/, '%s');
		};

		console.log('🎭 Fingerprint applied: %s');
	`,
		fingerprint.Screen.Width,
		fingerprint.Screen.Height,
		fingerprint.Platform,
		fingerprint.Language[0],
		fmt.Sprintf(`['%s']`, fingerprint.Language[0]), // 简化版语言数组
		fingerprint.Timezone,
		fingerprint.WebGL.Vendor,
		fingerprint.WebGL.Renderer,
		fingerprint.Canvas[len(fingerprint.Canvas)-1:], // 使用 canvas 指纹的最后一个字符
		fingerprint.ProfileName)

	_, err := page.Evaluate(fingerprintScript)
	return err
}

// TestFingerprintDetection tests fingerprint detection capabilities
func (fm *FingerprintManager) TestFingerprintDetection() error {
	testURL := "https://httpbin.org/headers"

	fm.mutex.RLock()
	profileNames := make([]string, 0, len(fm.profiles))
	for name := range fm.profiles {
		profileNames = append(profileNames, name)
	}
	fm.mutex.RUnlock()

	for _, profileName := range profileNames {
		browser := fm.browsers[profileName]
		if browser == nil {
			continue
		}

		// 创建新页面进行测试
		testPage, err := browser.Context.NewPage()
		if err != nil {
			continue
		}

		// 应用指纹
		if err := fm.applyFingerprint(testPage, browser.Profile); err != nil {
			continue
		}

		// 导航到测试页面
		if err := testPage.Navigate(testURL); err != nil {
			continue
		}

		time.Sleep(2 * time.Second)

		// 执行指纹检测测试
		detectionScript := `
			return {
				userAgent: navigator.userAgent,
				platform: navigator.platform,
				language: navigator.language,
				languages: navigator.languages,
				screen: {
					width: screen.width,
					height: screen.height
				},
				timezone: Intl.DateTimeFormat().resolvedOptions().timeZone,
				webdriver: navigator.webdriver,
				plugins: navigator.plugins.length
			};
		`

		result, err := testPage.Evaluate(detectionScript)
		if err != nil {
			log.Printf("Failed to run detection test for %s: %v", profileName, err)
			continue
		}

		fmt.Printf("  🔍 %s fingerprint: %v\n", profileName, result)
	}

	return nil
}

// EcommerceScenario demonstrates an e-commerce use case
func (fm *FingerprintManager) EcommerceScenario() error {
	ecommerceURL := "https://httpbin.org/anything/ecommerce"

	fm.mutex.RLock()
	profiles := make([]*BrowserFingerprint, 0, len(fm.profiles))
	for _, profile := range fm.profiles {
		profiles = append(profiles, profile)
	}
	fm.mutex.RUnlock()

	// 模拟不同用户的购物行为
	actions := []string{"browse_products", "add_to_cart", "checkout", "compare_prices"}

	var wg sync.WaitGroup
	for i, profile := range profiles {
		if i >= len(actions) {
			break
		}

		wg.Add(1)
		go func(p *BrowserFingerprint, action string) {
			defer wg.Done()

			browser := fm.browsers[p.ProfileName]
			if browser == nil {
				return
			}

			page, err := browser.Context.NewPage()
			if err != nil {
				return
			}

			// 应用指纹
			fm.applyFingerprint(page, p)

			// 模拟不同的电商行为
			actionURL := fmt.Sprintf("%s/%s", ecommerceURL, action)
			if err := page.Navigate(actionURL); err != nil {
				return
			}

			// 设置页面标题以便识别
			script := fmt.Sprintf(`document.title = '%s - %s'`, p.ProfileName, action)
			page.Evaluate(script)

			time.Sleep(time.Duration(2+randomInt(3)) * time.Second)
			fmt.Printf("  🛒 %s: Performed %s action\n", p.ProfileName, action)
		}(profile, actions[i])
	}

	wg.Wait()
	return nil
}

// Close closes all browsers and contexts
func (fm *FingerprintManager) Close() error {
	fm.mutex.Lock()
	defer fm.mutex.Unlock()

	// 关闭所有指纹浏览器
	for _, browser := range fm.browsers {
		if browser.Context != nil {
			browser.Context.Close()
		}
	}

	// 关闭主浏览器
	if fm.mainBrowser != nil {
		return fm.mainBrowser.Close()
	}

	return nil
}

// randomInt generates a random integer between 0 and max-1
func randomInt(max int) int {
	if max <= 0 {
		return 0
	}
	n, _ := rand.Int(rand.Reader, big.NewInt(int64(max)))
	return int(n.Int64())
}
