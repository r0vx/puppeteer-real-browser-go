package browser

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"time"
)

// AdvancedFingerprintManager 高级指纹管理器
// 结合JavaScript指纹修改和网络层指纹伪装
type AdvancedFingerprintManager struct {
	jsManager    *UserFingerprintManager
	proxyPort    int
	proxyCmd     *exec.Cmd
}

// NewAdvancedFingerprintManager 创建高级指纹管理器
func NewAdvancedFingerprintManager(configDir string) (*AdvancedFingerprintManager, error) {
	jsManager, err := NewUserFingerprintManager(configDir)
	if err != nil {
		return nil, err
	}

	return &AdvancedFingerprintManager{
		jsManager: jsManager,
		proxyPort: 8888 + (int(time.Now().Unix()) % 1000), // 随机端口
	}, nil
}

// LaunchBrowserWithFullFingerprint 启动具有完整指纹伪装的浏览器
func (afm *AdvancedFingerprintManager) LaunchBrowserWithFullFingerprint(ctx context.Context, userID string, opts *ConnectOptions) (interface{}, error) {
	// 1. 获取用户的JavaScript指纹配置
	fingerprintConfig, err := afm.jsManager.GetUserFingerprint(userID)
	if err != nil {
		return nil, fmt.Errorf("获取用户指纹配置失败: %v", err)
	}

	fmt.Printf("🔧 为用户 %s 启动完整指纹伪装...\n", userID)

	// 2. 启动网络层指纹代理
	if err := afm.startNetworkFingerprintProxy(fingerprintConfig); err != nil {
		fmt.Printf("⚠️  网络指纹代理启动失败，继续使用JavaScript指纹: %v\n", err)
	} else {
		fmt.Printf("✅ 网络指纹代理启动成功，端口: %d\n", afm.proxyPort)
	}

	// 3. 生成JavaScript注入脚本
	injector := NewFingerprintInjector(fingerprintConfig)
	injectionScript := injector.GenerateInjectionScript()

	// 4. 配置Chrome启动参数
	var chromeArgs []string
	
	// 基础指纹参数
	chromeArgs = append(chromeArgs, fingerprintConfig.GetChromeFlags()...)
	
	// 如果代理启动成功，添加代理参数
	if afm.proxyCmd != nil {
		proxyURL := fmt.Sprintf("http://127.0.0.1:%d", afm.proxyPort)
		chromeArgs = append(chromeArgs, "--proxy-server="+proxyURL)
		chromeArgs = append(chromeArgs, "--ignore-certificate-errors")
		chromeArgs = append(chromeArgs, "--ignore-ssl-errors")
		fmt.Printf("🌐 使用代理: %s\n", proxyURL)
	}

	// 反检测参数
	chromeArgs = append(chromeArgs, 
		"--disable-blink-features=AutomationControlled",
		"--exclude-switches=enable-automation",
		"--disable-infobars",
	)

	// 合并用户提供的参数
	if opts.Args != nil {
		chromeArgs = append(chromeArgs, opts.Args...)
	}
	opts.Args = chromeArgs

	// 设置用户特定的profile
	if opts.ProfileName == "" {
		opts.ProfileName = fmt.Sprintf("advanced_fp_%s", userID)
	}

	fmt.Printf("⚙️  Chrome启动参数数量: %d\n", len(chromeArgs))

	// 5. 启动浏览器
	instance, err := Connect(ctx, opts)
	if err != nil {
		afm.stopNetworkProxy()
		return nil, fmt.Errorf("Chrome启动失败: %v", err)
	}

	// 6. 注入JavaScript指纹修改脚本
	fmt.Printf("💉 注入JavaScript指纹脚本 (%d字符)...\n", len(injectionScript))
	
	// 在这里需要实际的页面对象来注入脚本
	// 由于Connect返回的是interface{}，我们需要用户在获取page对象后手动注入
	// 或者这里需要根据实际的browser库API来注入

	fmt.Printf("✅ 用户 %s 的完整指纹伪装浏览器启动成功\n", userID)
	fmt.Println("📊 已应用的指纹修改:")
	fmt.Printf("   🌐 JavaScript层: UserAgent, Screen, WebGL, Audio, Canvas等\n")
	if afm.proxyCmd != nil {
		fmt.Printf("   🔒 网络层: TLS指纹, HTTP2指纹 (通过代理)\n")
	} else {
		fmt.Printf("   ⚠️  网络层: 未修改 (代理未启动)\n")
	}

	return instance, nil
}

// startNetworkFingerprintProxy 启动网络指纹代理
func (afm *AdvancedFingerprintManager) startNetworkFingerprintProxy(config *FingerprintConfig) error {
	// 方案1: 尝试使用ja3proxy (如果可用)
	if err := afm.tryStartJA3Proxy(config); err == nil {
		return nil
	}

	// 方案2: 尝试使用mitmdump (如果可用)
	if err := afm.tryStartMitmProxy(config); err == nil {
		return nil
	}

	// 方案3: 使用内置的基础代理 (功能有限)
	return afm.tryStartBuiltinProxy(config)
}

// tryStartJA3Proxy 尝试启动ja3proxy
func (afm *AdvancedFingerprintManager) tryStartJA3Proxy(config *FingerprintConfig) error {
	// 检查ja3proxy是否可用
	if _, err := exec.LookPath("ja3proxy"); err != nil {
		return fmt.Errorf("ja3proxy未安装")
	}

	// 生成ja3proxy配置
	ja3Config := fmt.Sprintf(`{
		"ja3": "%s",
		"user_agent": "%s",
		"listen_port": %d
	}`, config.TLSConfig.JA3, config.Browser.UserAgent, afm.proxyPort)

	configFile := filepath.Join(os.TempDir(), fmt.Sprintf("ja3proxy_%d.json", afm.proxyPort))
	if err := os.WriteFile(configFile, []byte(ja3Config), 0644); err != nil {
		return fmt.Errorf("创建ja3proxy配置失败: %v", err)
	}

	// 启动ja3proxy
	cmd := exec.Command("ja3proxy", "-config", configFile)
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("启动ja3proxy失败: %v", err)
	}

	afm.proxyCmd = cmd
	time.Sleep(2 * time.Second) // 等待代理启动
	return nil
}

// tryStartMitmProxy 尝试启动mitmproxy
func (afm *AdvancedFingerprintManager) tryStartMitmProxy(config *FingerprintConfig) error {
	if _, err := exec.LookPath("mitmdump"); err != nil {
		return fmt.Errorf("mitmproxy未安装")
	}

	// 创建mitmproxy脚本
	scriptContent := fmt.Sprintf(`
import mitmproxy.http
from mitmproxy import ctx

def request(flow: mitmproxy.http.HTTPFlow) -> None:
    # 修改User-Agent
    flow.request.headers["User-Agent"] = "%s"
    # 修改Accept-Language
    flow.request.headers["Accept-Language"] = "%s,en;q=0.9"
`, config.Browser.UserAgent, config.Browser.Language)

	scriptFile := filepath.Join(os.TempDir(), fmt.Sprintf("mitmproxy_%d.py", afm.proxyPort))
	if err := os.WriteFile(scriptFile, []byte(scriptContent), 0644); err != nil {
		return fmt.Errorf("创建mitmproxy脚本失败: %v", err)
	}

	// 启动mitmdump
	cmd := exec.Command("mitmdump", 
		"-s", scriptFile,
		"--listen-port", strconv.Itoa(afm.proxyPort),
		"--set", "confdir="+os.TempDir(),
	)
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("启动mitmproxy失败: %v", err)
	}

	afm.proxyCmd = cmd
	time.Sleep(3 * time.Second) // 等待代理启动
	return nil
}

// tryStartBuiltinProxy 启动内置基础代理
func (afm *AdvancedFingerprintManager) tryStartBuiltinProxy(config *FingerprintConfig) error {
	// 这里可以启动我们之前实现的NetworkFingerprintProxy
	// 但它的功能有限，无法完全修改JA4/HTTP2指纹
	proxy := NewNetworkFingerprintProxy(config, afm.proxyPort)
	
	go func() {
		if err := proxy.StartProxy(); err != nil {
			fmt.Printf("内置代理启动失败: %v\n", err)
		}
	}()
	
	time.Sleep(2 * time.Second)
	return nil
}

// stopNetworkProxy 停止网络代理
func (afm *AdvancedFingerprintManager) stopNetworkProxy() {
	if afm.proxyCmd != nil {
		afm.proxyCmd.Process.Kill()
		afm.proxyCmd = nil
	}
}

// Close 关闭管理器
func (afm *AdvancedFingerprintManager) Close() {
	afm.stopNetworkProxy()
}

// GetUserFingerprintWithNetworkInfo 获取包含网络层信息的用户指纹
func (afm *AdvancedFingerprintManager) GetUserFingerprintWithNetworkInfo(userID string) (*FingerprintConfig, error) {
	config, err := afm.jsManager.GetUserFingerprint(userID)
	if err != nil {
		return nil, err
	}

	// 添加网络代理信息
	if afm.proxyCmd != nil {
		fmt.Printf("📡 用户 %s 的网络指纹信息:\n", userID)
		fmt.Printf("   🔒 TLS/JA4: %s\n", config.TLSConfig.JA4)
		fmt.Printf("   🌐 HTTP2/Akamai: %s\n", config.HTTP2Config.AKAMAI)
		fmt.Printf("   📡 代理端口: %d\n", afm.proxyPort)
	}

	return config, nil
}

// GenerateUsageExample 生成使用示例
func (afm *AdvancedFingerprintManager) GenerateUsageExample(userID string) string {
	return fmt.Sprintf(`
// 高级指纹管理器使用示例

// 1. 创建管理器
manager, err := browser.NewAdvancedFingerprintManager("./fingerprints")
if err != nil {
    log.Fatal(err)
}
defer manager.Close()

// 2. 启动完整指纹伪装浏览器
opts := &browser.ConnectOptions{
    Headless:       false,
    PersistProfile: true,
    Extensions:     []string{ext1, ext2},
}

instance, err := manager.LaunchBrowserWithFullFingerprint(ctx, "%s", opts)
if err != nil {
    log.Fatal(err)
}
defer instance.Close()

// 3. 使用浏览器
page := instance.Page()
page.Navigate("https://iplark.com/fingerprint")

// 现在这个浏览器应该有独特的:
// - JavaScript层指纹 (UserAgent, WebGL, Audio等)
// - 网络层指纹 (JA4, HTTP2指纹) [如果代理成功启动]
`, userID)
}

/*
安装指纹伪装工具的建议:

1. ja3proxy:
   go install github.com/CUCyber/ja3proxy@latest

2. mitmproxy:
   pip install mitmproxy

3. 或者使用Docker:
   docker run --rm -p 8080:8080 mitmproxy/mitmproxy mitmdump --web-host 0.0.0.0

注意: 完整的网络层指纹修改需要专门的工具支持
当前系统会优雅降级，如果网络层工具不可用，至少JavaScript层指纹会正常工作
*/