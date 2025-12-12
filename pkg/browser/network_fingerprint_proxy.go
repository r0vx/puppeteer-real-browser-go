package browser

import (
	"crypto/tls"
	"fmt"
	"net/http"
	"net/http/httputil"
)

// NetworkFingerprintProxy 网络层指纹代理
type NetworkFingerprintProxy struct {
	config       *FingerprintConfig
	proxyPort    int
	upstreamProxy string
}

// NewNetworkFingerprintProxy 创建网络指纹代理
func NewNetworkFingerprintProxy(config *FingerprintConfig, proxyPort int) *NetworkFingerprintProxy {
	return &NetworkFingerprintProxy{
		config:    config,
		proxyPort: proxyPort,
	}
}

// StartProxy 启动代理服务器
func (nfp *NetworkFingerprintProxy) StartProxy() error {
	// 创建自定义Transport用于修改TLS和HTTP2指纹
	transport := &http.Transport{
		TLSClientConfig: nfp.createCustomTLSConfig(),
		// 强制使用HTTP/2
		ForceAttemptHTTP2: true,
		// 自定义拨号器可以在这里修改TCP指纹
	}

	// 创建代理处理器
	proxy := &httputil.ReverseProxy{
		Director: func(req *http.Request) {
			// 修改请求头以模拟不同的HTTP2指纹
			nfp.modifyRequestHeaders(req)
		},
		Transport: transport,
		ModifyResponse: func(resp *http.Response) error {
			// 修改响应头
			return nil
		},
	}

	server := &http.Server{
		Addr:    fmt.Sprintf(":%d", nfp.proxyPort),
		Handler: proxy,
	}

	fmt.Printf("🔧 网络指纹代理启动在端口 %d\n", nfp.proxyPort)
	return server.ListenAndServe()
}

// createCustomTLSConfig 创建自定义TLS配置以模拟不同的JA4指纹
func (nfp *NetworkFingerprintProxy) createCustomTLSConfig() *tls.Config {
	// 基于用户指纹配置创建TLS配置
	tlsConfig := &tls.Config{
		MinVersion: tls.VersionTLS12,
		MaxVersion: tls.VersionTLS13,
	}

	// 根据指纹配置设置密码套件
	var cipherSuites []uint16
	for _, cipherName := range nfp.config.TLSConfig.CipherSuites {
		if cipherID := getCipherSuiteID(cipherName); cipherID != 0 {
			cipherSuites = append(cipherSuites, cipherID)
		}
	}
	tlsConfig.CipherSuites = cipherSuites

	// 设置支持的曲线
	var curves []tls.CurveID
	for _, curveName := range nfp.config.TLSConfig.EllipticCurves {
		if curveID := getCurveID(curveName); curveID != 0 {
			curves = append(curves, curveID)
		}
	}
	tlsConfig.CurvePreferences = curves

	return tlsConfig
}

// modifyRequestHeaders 修改请求头以模拟不同的HTTP2指纹
func (nfp *NetworkFingerprintProxy) modifyRequestHeaders(req *http.Request) {
	// 修改User-Agent
	req.Header.Set("User-Agent", nfp.config.Browser.UserAgent)

	// 根据HTTP2配置修改头部顺序和值
	// 这里可以重新排列头部顺序来改变HTTP2指纹
	
	// 添加或修改Accept-Language
	req.Header.Set("Accept-Language", nfp.config.Browser.Language+",en;q=0.9")

	// 根据配置修改其他头部
	if nfp.config.HTTP2Config.Settings["SETTINGS_ENABLE_PUSH"] == 0 {
		// 如果禁用了推送，可以在这里添加相关头部
	}

	// 设置自定义的连接属性（虽然这不能完全改变HTTP2指纹，但会有一定影响）
	req.Header.Set("Connection", "keep-alive")
}

// getCipherSuiteID 将密码套件名称转换为ID
func getCipherSuiteID(name string) uint16 {
	cipherSuites := map[string]uint16{
		"TLS_AES_128_GCM_SHA256":                      0x1301,
		"TLS_AES_256_GCM_SHA384":                      0x1302,
		"TLS_CHACHA20_POLY1305_SHA256":                0x1303,
		"TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256":     0xc02b,
		"TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256":       0xc02f,
		"TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384":     0xc02c,
		"TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384":       0xc030,
		"TLS_ECDHE_ECDSA_WITH_CHACHA20_POLY1305_SHA256": 0xcca9,
		"TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305_SHA256":   0xcca8,
		"TLS_ECDHE_RSA_WITH_AES_128_CBC_SHA":             0xc013,
		"TLS_ECDHE_RSA_WITH_AES_256_CBC_SHA":             0xc014,
		"TLS_RSA_WITH_AES_128_GCM_SHA256":                0x009c,
		"TLS_RSA_WITH_AES_256_GCM_SHA384":                0x009d,
		"TLS_RSA_WITH_AES_128_CBC_SHA":                   0x002f,
		"TLS_RSA_WITH_AES_256_CBC_SHA":                   0x0035,
	}
	return cipherSuites[name]
}

// getCurveID 将曲线名称转换为ID
func getCurveID(name string) tls.CurveID {
	curves := map[string]tls.CurveID{
		"X25519":    tls.X25519,
		"secp256r1": tls.CurveP256,
		"secp384r1": tls.CurveP384,
		"secp521r1": tls.CurveP521,
	}
	return curves[name]
}

// GetProxyURL 获取代理URL用于Chrome启动参数
func (nfp *NetworkFingerprintProxy) GetProxyURL() string {
	return fmt.Sprintf("http://127.0.0.1:%d", nfp.proxyPort)
}

// 修改Chrome启动参数以使用代理
func (config *FingerprintConfig) GetChromeArgsWithProxy(proxyURL string) []string {
	args := config.GetChromeFlags()
	
	// 添加代理参数
	args = append(args, "--proxy-server="+proxyURL)
	
	// 忽略证书错误（用于HTTPS代理）
	args = append(args, "--ignore-certificate-errors-spki-list")
	args = append(args, "--ignore-certificate-errors")
	args = append(args, "--ignore-ssl-errors")
	
	// 允许本地代理
	args = append(args, "--allow-running-insecure-content")
	
	return args
}

// 高级网络指纹修改说明
/*
完整的网络层指纹修改需要以下技术:

1. JA4/JA3指纹修改:
   - 需要在TLS握手层面修改
   - 可以通过自定义TLS库实现
   - 或者使用支持指纹伪装的代理工具

2. HTTP2指纹修改:
   - 需要修改HTTP2 SETTINGS帧
   - 需要修改WINDOW_UPDATE值
   - 需要修改头部压缩和优先级

3. TCP指纹修改:
   - 需要修改TCP选项
   - 需要修改窗口大小和缩放因子
   - 需要内核级别的修改

4. 实现建议:
   - 使用专门的指纹伪装代理 (如 ja3proxy)
   - 或集成 uTLS 库进行 TLS 指纹伪装
   - 或使用支持指纹修改的 HTTP2 库

注意: 当前这个代理只是基础框架，完整实现需要更深层的网络协议修改
*/