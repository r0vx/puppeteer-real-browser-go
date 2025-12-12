package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"

	"github.com/HNRow/puppeteer-real-browser-go/pkg/browser"
)

func main() {
	fmt.Println("🌐 HTTP 代理功能演示")
	fmt.Println("===================")
	fmt.Println()

	ctx := context.Background()

	// 场景 1: 无代理（直连）
	fmt.Println("📝 场景 1: 无代理（直连）")
	fmt.Println("-------------------------------------")
	testNoProxy(ctx)

	// 场景 2: 使用真实代理进行测试
	fmt.Println("\n📝 场景 2: 真实代理测试（从 API 获取）")
	fmt.Println("-------------------------------------")
	testRealProxy(ctx)

	fmt.Println("\n✅ 所有场景演示完成！")
	fmt.Println()
	PrintProxyGuide()
}

// ProxyAPIResponse 代理 API 响应结构
type ProxyAPIResponse struct {
	Code int `json:"code"`
	Data []struct {
		IP       string `json:"ip"`
		Port     int    `json:"port"`
		ExpireAt string `json:"expire_at"`
		City     string `json:"city"`
		ISP      string `json:"isp"`
	} `json:"data"`
	Msg     string `json:"msg"`
	Success bool   `json:"success"`
}

// fetchProxyFromAPI 从 API 获取代理 IP
func fetchProxyFromAPI() (*browser.ProxyConfig, error) {
	apiURL := "http://api.shenlongip.com/ip?key=3da66g0n&area=430300&protocol=1&mr=1&pattern=json&need=1011&count=1&sign=268c0564b635a9cb201d782e96a055c2"
	
	fmt.Println("🔍 从 API 获取代理 IP...")
	
	resp, err := http.Get(apiURL)
	if err != nil {
		return nil, fmt.Errorf("请求失败: %w", err)
	}
	defer resp.Body.Close()
	
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取响应失败: %w", err)
	}
	
	var proxyResp ProxyAPIResponse
	if err := json.Unmarshal(body, &proxyResp); err != nil {
		return nil, fmt.Errorf("解析 JSON 失败: %w", err)
	}
	
	if !proxyResp.Success || len(proxyResp.Data) == 0 {
		return nil, fmt.Errorf("获取代理失败: %s", proxyResp.Msg)
	}
	
	proxyData := proxyResp.Data[0]
	proxyConfig := &browser.ProxyConfig{
		Host: proxyData.IP,
		Port: fmt.Sprintf("%d", proxyData.Port),
	}
	
	fmt.Printf("✅ 获取到代理: %s:%s\n", proxyConfig.Host, proxyConfig.Port)
	fmt.Printf("   位置: %s\n", proxyData.City)
	fmt.Printf("   运营商: %s\n", proxyData.ISP)
	fmt.Printf("   过期时间: %s\n", proxyData.ExpireAt)
	
	return proxyConfig, nil
}

// testRealProxy 测试真实代理
func testRealProxy(ctx context.Context) {
	// 获取代理
	proxyConfig, err := fetchProxyFromAPI()
	if err != nil {
		log.Printf("❌ 获取代理失败: %v", err)
		return
	}
	
	fmt.Println()
	
	// 先测试直连 IP
	fmt.Println("🔹 步骤 1: 测试直连 IP")
	directIP := getMyIP(ctx, nil)
	if directIP != "" {
		fmt.Printf("   直连 IP: %s\n", directIP)
	}
	
	fmt.Println()
	
	// 测试代理 IP
	fmt.Println("🔹 步骤 2: 测试代理 IP")
	opts := &browser.ConnectOptions{
		Headless: true, // 使用 headless 模式提高速度
		Proxy:    proxyConfig,
		Args: []string{
			"--disable-gpu",
		},
	}
	
	fmt.Println("🚀 启动浏览器（使用代理）...")
	
	instance, err := browser.Connect(ctx, opts)
	if err != nil {
		log.Printf("❌ 启动失败: %v", err)
		return
	}
	defer instance.Close()
	
	page := instance.Page()
	
	fmt.Println("📂 通过代理访问 IP 检测 API...")
	if err := page.Navigate("https://api.ipify.org?format=text"); err != nil {
		log.Printf("❌ 导航失败: %v", err)
		return
	}
	
	time.Sleep(3 * time.Second)
	
	// 获取代理 IP
	result, err := page.Evaluate(`document.body.innerText`)
	if err != nil {
		log.Printf("⚠️  获取 IP 失败: %v", err)
	} else {
		proxyIP := fmt.Sprintf("%v", result)
		fmt.Printf("   代理 IP: %s\n", proxyIP)
		
		// 验证代理是否生效
		if proxyIP != "" && proxyIP != directIP {
			fmt.Println()
			fmt.Println("✅ 代理验证成功！")
			fmt.Printf("   直连 IP: %s\n", directIP)
			fmt.Printf("   代理 IP: %s\n", proxyIP)
			fmt.Println("   IP 已改变，代理生效！")
		} else {
			fmt.Println()
			fmt.Println("⚠️  代理可能未生效")
			fmt.Printf("   直连 IP: %s\n", directIP)
			fmt.Printf("   代理 IP: %s\n", proxyIP)
		}
	}
	
	fmt.Println()
	
	// 测试访问网站
	fmt.Println("🔹 步骤 3: 测试访问网站")
	fmt.Println("📂 访问 Example.com...")
	if err := page.Navigate("https://example.com"); err != nil {
		log.Printf("❌ 导航失败: %v", err)
		return
	}
	
	time.Sleep(2 * time.Second)
	
	title, err := page.GetTitle()
	if err != nil {
		log.Printf("⚠️  获取标题失败: %v", err)
	} else {
		fmt.Printf("✅ 页面标题: %s\n", title)
	}
	
	// 截图验证
	screenshot, err := page.Screenshot()
	if err != nil {
		log.Printf("⚠️  截图失败: %v", err)
	} else {
		fmt.Printf("✅ 截图成功: %d bytes\n", len(screenshot))
	}
}

// getMyIP 获取当前 IP（用于对比）
func getMyIP(ctx context.Context, proxy *browser.ProxyConfig) string {
	opts := &browser.ConnectOptions{
		Headless: true,
		Proxy:    proxy,
	}
	
	instance, err := browser.Connect(ctx, opts)
	if err != nil {
		return ""
	}
	defer instance.Close()
	
	page := instance.Page()
	if err := page.Navigate("https://api.ipify.org?format=text"); err != nil {
		return ""
	}
	
	time.Sleep(2 * time.Second)
	
	result, err := page.Evaluate(`document.body.innerText`)
	if err != nil {
		return ""
	}
	
	return fmt.Sprintf("%v", result)
}

// testNoProxy 测试无代理（直连）
func testNoProxy(ctx context.Context) {
	opts := &browser.ConnectOptions{
		Headless: false,
		Args: []string{
			"--start-maximized",
		},
	}

	fmt.Println("🚀 启动浏览器（无代理）...")

	instance, err := browser.Connect(ctx, opts)
	if err != nil {
		log.Printf("❌ 启动失败: %v", err)
		return
	}
	defer instance.Close()

	page := instance.Page()

	fmt.Println("📂 导航到 IP 检测网站...")
	if err := page.Navigate("https://api.ipify.org?format=json"); err != nil {
		log.Printf("❌ 导航失败: %v", err)
		return
	}

	time.Sleep(2 * time.Second)

	// 获取 IP 信息
	result, err := page.Evaluate(`document.body.innerText`)
	if err != nil {
		log.Printf("⚠️  获取 IP 失败: %v", err)
	} else {
		fmt.Printf("✅ 当前 IP 信息: %v\n", result)
	}

	fmt.Println("⏳ 保持运行 3 秒...")
	time.Sleep(3 * time.Second)
}


// PrintProxyGuide 打印代理使用指南
func PrintProxyGuide() {
	fmt.Println("📘 代理使用指南")
	fmt.Println("==============")
	fmt.Println()
	
	fmt.Println("1️⃣ 基本代理配置（无认证）")
	fmt.Println("```go")
	fmt.Println("opts := &browser.ConnectOptions{")
	fmt.Println("    Proxy: &browser.ProxyConfig{")
	fmt.Println("        Host: \"proxy.example.com\",")
	fmt.Println("        Port: \"8080\",")
	fmt.Println("    },")
	fmt.Println("}")
	fmt.Println("```")
	fmt.Println()
	
	fmt.Println("2️⃣ 代理认证（用户名/密码）")
	fmt.Println("```go")
	fmt.Println("opts := &browser.ConnectOptions{")
	fmt.Println("    Proxy: &browser.ProxyConfig{")
	fmt.Println("        Host:     \"proxy.example.com\",")
	fmt.Println("        Port:     \"8080\",")
	fmt.Println("        Username: \"your_username\",")
	fmt.Println("        Password: \"your_password\",")
	fmt.Println("    },")
	fmt.Println("}")
	fmt.Println("```")
	fmt.Println()
	
	fmt.Println("3️⃣ 代理类型支持")
	fmt.Println("  ✅ HTTP 代理")
	fmt.Println("  ✅ HTTPS 代理")
	fmt.Println("  ✅ SOCKS5 代理（使用 socks5://host:port 格式）")
	fmt.Println()
	
	fmt.Println("4️⃣ 常见问题")
	fmt.Println("  Q: 代理不生效？")
	fmt.Println("  A: 检查代理服务器是否可用，防火墙设置")
	fmt.Println()
	fmt.Println("  Q: 认证失败？")
	fmt.Println("  A: 确认用户名/密码正确，代理支持认证")
	fmt.Println()
	fmt.Println("  Q: 如何验证代理生效？")
	fmt.Println("  A: 访问 https://api.ipify.org 查看 IP")
	fmt.Println()
	
	fmt.Println("5️⃣ 免费代理资源（仅供测试）")
	fmt.Println("  • https://www.proxy-list.download/")
	fmt.Println("  • https://free-proxy-list.net/")
	fmt.Println("  • https://www.sslproxies.org/")
	fmt.Println()
	fmt.Println("  ⚠️  注意：免费代理不稳定，生产环境请使用付费代理")
	fmt.Println()
	
	fmt.Println("6️⃣ 推荐付费代理服务")
	fmt.Println("  • Bright Data (原 Luminati)")
	fmt.Println("  • Oxylabs")
	fmt.Println("  • Smartproxy")
	fmt.Println("  • ProxyMesh")
	fmt.Println()
	
	fmt.Println("7️⃣ 代理池实现示例")
	fmt.Println("```go")
	fmt.Println("type ProxyPool struct {")
	fmt.Println("    proxies []*browser.ProxyConfig")
	fmt.Println("    current int")
	fmt.Println("}")
	fmt.Println()
	fmt.Println("func (p *ProxyPool) Next() *browser.ProxyConfig {")
	fmt.Println("    proxy := p.proxies[p.current]")
	fmt.Println("    p.current = (p.current + 1) % len(p.proxies)")
	fmt.Println("    return proxy")
	fmt.Println("}")
	fmt.Println("```")
}

// ExampleProxyRotation 代理轮换示例
func ExampleProxyRotation() {
	fmt.Println("\n🔄 代理轮换示例")
	fmt.Println("===============")
	
	// 代理池
	proxyList := []*browser.ProxyConfig{
		{Host: "proxy1.example.com", Port: "8080"},
		{Host: "proxy2.example.com", Port: "8080"},
		{Host: "proxy3.example.com", Port: "8080"},
	}
	
	ctx := context.Background()
	
	// 使用不同代理进行多次请求
	for i, proxy := range proxyList {
		fmt.Printf("\n第 %d 次请求 - 使用代理: %s:%s\n", i+1, proxy.Host, proxy.Port)
		
		opts := &browser.ConnectOptions{
			Headless: true,
			Proxy:    proxy,
		}
		
		instance, err := browser.Connect(ctx, opts)
		if err != nil {
			log.Printf("❌ 连接失败: %v", err)
			continue
		}
		
		page := instance.Page()
		page.Navigate("https://example.com")
		time.Sleep(1 * time.Second)
		
		instance.Close()
	}
	
	fmt.Println("\n✅ 代理轮换演示完成")
}

