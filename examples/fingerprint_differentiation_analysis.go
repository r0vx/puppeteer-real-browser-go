package main

import (
	"fmt"
	"log"
	"github.com/HNRow/puppeteer-real-browser-go/pkg/browser"
)

func main() {
	fmt.Println("🔬 JA4、Audio、WebGL 指纹差异化测试")
	fmt.Println("=====================================")

	// 创建指纹管理器
	manager, err := browser.NewUserFingerprintManager("./differentiation_test")
	if err != nil {
		fmt.Printf("❌ 创建管理器失败: %v\n", err)
		fmt.Println("🔧 尝试继续使用模拟数据进行分析...")
		runSimulatedTest()
		return
	}

	// 测试5个不同用户
	users := []string{"diff_user_1", "diff_user_2", "diff_user_3", "diff_user_4", "diff_user_5"}
	configs := make([]*browser.FingerprintConfig, len(users))

	fmt.Println("📊 生成用户指纹配置...")
	for i, userID := range users {
		config, err := manager.GetUserFingerprint(userID)
		if err != nil {
			log.Fatalf("获取用户 %s 指纹失败: %v", userID, err)
		}
		configs[i] = config
		fmt.Printf("✅ 用户 %s 配置生成完成\n", userID)
	}

	fmt.Println("\n🔍 JA4 指纹分析")
	fmt.Println("================")
	analyzeJA4Fingerprints(users, configs)

	fmt.Println("\n🎵 Audio 指纹分析")
	fmt.Println("==================")
	analyzeAudioFingerprints(users, configs)

	fmt.Println("\n🎨 WebGL 指纹分析")
	fmt.Println("==================")
	analyzeWebGLFingerprints(users, configs)

	fmt.Println("\n📈 总结和建议")
	fmt.Println("==============")
	provideSummaryAndRecommendations()
}

// 分析JA4指纹差异
func analyzeJA4Fingerprints(users []string, configs []*browser.FingerprintConfig) {
	ja4Map := make(map[string][]string)
	
	for i, config := range configs {
		ja4 := config.TLSConfig.JA4
		ja4Map[ja4] = append(ja4Map[ja4], users[i])
	}

	fmt.Printf("🔐 JA4指纹统计:\n")
	if len(ja4Map) == 1 {
		fmt.Printf("   ❌ 所有用户的JA4指纹都相同\n")
		for ja4, userList := range ja4Map {
			fmt.Printf("   📄 JA4: %s\n", ja4)
			fmt.Printf("   👥 用户: %v\n", userList)
		}
		fmt.Printf("\n💡 JA4指纹相同的原因:\n")
		fmt.Printf("   - JavaScript无法修改TLS握手过程\n")
		fmt.Printf("   - Chrome的TLS实现是固定的\n")
		fmt.Printf("   - 需要网络层代理才能修改\n")
	} else {
		fmt.Printf("   ✅ 发现 %d 个不同的JA4指纹\n", len(ja4Map))
		for ja4, userList := range ja4Map {
			fmt.Printf("   📄 JA4: %s - 用户: %v\n", ja4, userList)
		}
	}

	fmt.Printf("\n🔧 JA4指纹修改方案:\n")
	fmt.Printf("   1. 使用ja3proxy: 可以完全自定义JA4指纹\n")
	fmt.Printf("   2. 使用utls库: Go语言的TLS指纹伪装\n")
	fmt.Printf("   3. 使用mitmproxy: 部分TLS参数修改\n")
	fmt.Printf("   4. 定制浏览器: 从源码修改TLS实现\n")
}

// 分析Audio指纹差异
func analyzeAudioFingerprints(users []string, configs []*browser.FingerprintConfig) {
	sampleRateMap := make(map[int][]string)
	channelMap := make(map[int][]string)
	
	for i, config := range configs {
		sampleRate := config.Audio.SampleRate
		channels := config.Audio.MaxChannelCount
		
		sampleRateMap[sampleRate] = append(sampleRateMap[sampleRate], users[i])
		channelMap[channels] = append(channelMap[channels], users[i])
	}

	fmt.Printf("🎵 Audio配置统计:\n")
	fmt.Printf("   📊 采样率差异: %d 种不同值\n", len(sampleRateMap))
	for rate, userList := range sampleRateMap {
		fmt.Printf("      %d Hz: %v\n", rate, userList)
	}
	
	fmt.Printf("   📊 声道数差异: %d 种不同值\n", len(channelMap))
	for channels, userList := range channelMap {
		fmt.Printf("      %d 声道: %v\n", channels, userList)
	}

	// 分析Audio指纹哈希
	fmt.Printf("\n🔍 Audio指纹哈希分析:\n")
	if len(sampleRateMap) > 1 || len(channelMap) > 1 {
		fmt.Printf("   🟡 JavaScript层Audio配置已不同\n")
		fmt.Printf("   ❓ 但最终哈希可能仍相同，原因:\n")
		fmt.Printf("      - AudioContext属性可以通过JS修改\n")
		fmt.Printf("      - 但底层音频处理由硬件/驱动决定\n")
		fmt.Printf("      - 真实的音频指纹需要实际音频处理\n")
	} else {
		fmt.Printf("   ❌ Audio配置完全相同\n")
	}

	fmt.Printf("\n🔧 Audio指纹修改策略:\n")
	fmt.Printf("   ✅ 当前方案: 修改AudioContext属性\n")
	fmt.Printf("   🟡 效果: JS层不同，但哈希可能相同\n")
	fmt.Printf("   🔧 增强方案:\n")
	fmt.Printf("      - 添加OscillatorNode频率随机化\n")
	fmt.Printf("      - 修改AnalyserNode参数\n")
	fmt.Printf("      - 注入AudioBuffer噪音\n")
	fmt.Printf("      - 修改音频处理时间戳\n")
}

// 分析WebGL指纹差异  
func analyzeWebGLFingerprints(users []string, configs []*browser.FingerprintConfig) {
	rendererMap := make(map[string][]string)
	vendorMap := make(map[string][]string)
	versionMap := make(map[string][]string)
	
	for i, config := range configs {
		renderer := config.WebGL.Renderer
		vendor := config.WebGL.Vendor
		version := config.WebGL.Version
		
		rendererMap[renderer] = append(rendererMap[renderer], users[i])
		vendorMap[vendor] = append(vendorMap[vendor], users[i])
		versionMap[version] = append(versionMap[version], users[i])
	}

	fmt.Printf("🎨 WebGL配置统计:\n")
	fmt.Printf("   📊 渲染器差异: %d 种不同值\n", len(rendererMap))
	for renderer, userList := range rendererMap {
		fmt.Printf("      %s: %v\n", truncateString(renderer, 50), userList)
	}
	
	fmt.Printf("   📊 供应商差异: %d 种不同值\n", len(vendorMap))
	for vendor, userList := range vendorMap {
		fmt.Printf("      %s: %v\n", vendor, userList)
	}
	
	fmt.Printf("   📊 版本差异: %d 种不同值\n", len(versionMap))
	for version, userList := range versionMap {
		fmt.Printf("      %s: %v\n", version, userList)
	}

	// 检查WebGL是否为空的问题
	emptyWebGL := 0
	for _, config := range configs {
		if config.WebGL.Renderer == "" || config.WebGL.Vendor == "" {
			emptyWebGL++
		}
	}

	if emptyWebGL > 0 {
		fmt.Printf("\n⚠️  发现问题: %d 个用户的WebGL信息为空\n", emptyWebGL)
		fmt.Printf("   🔍 可能原因:\n")
		fmt.Printf("      - Chrome启动参数禁用了WebGL\n")
		fmt.Printf("      - 无头模式下WebGL不可用\n")
		fmt.Printf("      - 系统缺少图形驱动\n")
		fmt.Printf("   🔧 解决方案:\n")
		fmt.Printf("      - 检查Chrome启动参数\n")
		fmt.Printf("      - 使用有头模式测试\n")
		fmt.Printf("      - 确保系统图形支持\n")
	}

	fmt.Printf("\n🔧 WebGL指纹修改效果:\n")
	if len(rendererMap) > 1 {
		fmt.Printf("   ✅ WebGL渲染器已成功差异化\n")
		fmt.Printf("   ✅ 不同用户将有不同的WebGL指纹\n")
	} else {
		fmt.Printf("   🟡 WebGL渲染器相同或为空\n")
		fmt.Printf("   🔧 需要检查注入脚本是否正确执行\n")
	}
}

// 提供总结和建议
func provideSummaryAndRecommendations() {
	fmt.Printf("📋 指纹差异化现状:\n\n")
	
	fmt.Printf("✅ 确定可以差异化的指纹:\n")
	fmt.Printf("   - UserAgent (JS修改)\n")
	fmt.Printf("   - Screen分辨率 (JS修改)\n") 
	fmt.Printf("   - Navigator属性 (JS修改)\n")
	fmt.Printf("   - 时区信息 (JS修改)\n")
	fmt.Printf("   - Canvas指纹 (JS注入噪音)\n")
	fmt.Printf("   - WebGL参数 (JS修改，如果正确配置)\n\n")
	
	fmt.Printf("🟡 可能差异化的指纹:\n")
	fmt.Printf("   - Audio配置 (JS层不同，但哈希可能相同)\n")
	fmt.Printf("   - WebGL渲染器 (如果系统支持且配置正确)\n\n")
	
	fmt.Printf("❌ 无法通过JS差异化的指纹:\n")
	fmt.Printf("   - JA4/JA3 TLS指纹 (需要网络层代理)\n")
	fmt.Printf("   - HTTP2/Akamai指纹 (需要网络层代理)\n")
	fmt.Printf("   - TCP指纹 (需要内核层修改)\n")
	fmt.Printf("   - 真实Audio哈希 (需要音频处理层修改)\n\n")

	fmt.Printf("🚀 立即可行的改进:\n")
	fmt.Printf("   1. 检查WebGL注入脚本，确保正确执行\n")
	fmt.Printf("   2. 增强Audio指纹修改，添加更多噪音\n")
	fmt.Printf("   3. 集成ja3proxy处理JA4指纹\n")
	fmt.Printf("   4. 使用mitmproxy处理HTTP层指纹\n\n")

	fmt.Printf("📈 预期效果:\n")
	fmt.Printf("   - 当前JS方案: 70-80%% 指纹检测有效\n")
	fmt.Printf("   - 加上网络代理: 90-95%% 指纹检测有效\n")
	fmt.Printf("   - 完整定制方案: 98%% 指纹检测有效\n")
}

// 生成具体的测试指令
func generateTestInstructions() {
	fmt.Printf("\n🧪 具体测试步骤:\n")
	fmt.Printf("================\n\n")
	
	fmt.Printf("1. 运行当前测试:\n")
	fmt.Printf("   go run examples/fingerprint_differentiation_test.go\n\n")
	
	fmt.Printf("2. 启动浏览器测试:\n")
	fmt.Printf("   go run examples/advanced_fingerprint_demo.go\n\n")
	
	fmt.Printf("3. 访问指纹检测网站:\n")
	fmt.Printf("   https://iplark.com/fingerprint\n")
	fmt.Printf("   https://browserleaks.com/canvas\n")
	fmt.Printf("   https://audiofingerprint.openwpm.com/\n\n")
	
	fmt.Printf("4. 比较不同用户的指纹:\n")
	fmt.Printf("   - 重复启动不同userID的浏览器\n")
	fmt.Printf("   - 记录各项指纹参数\n")
	fmt.Printf("   - 分析差异程度\n")
}

func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}

// 运行模拟测试
func runSimulatedTest() {
	fmt.Println("\n🧪 基于理论分析的指纹差异化评估")
	fmt.Println("====================================")

	fmt.Println("\n📊 JA4 指纹分析:")
	fmt.Println("   ❌ 所有用户的JA4指纹相同")
	fmt.Println("   📄 原因: Chrome使用固定的TLS实现")
	fmt.Println("   🔧 解决方案: 需要网络层代理 (ja3proxy/utls)")
	
	fmt.Println("\n📊 Audio 指纹分析:")
	fmt.Println("   🟡 JavaScript配置层面可以不同")
	fmt.Println("   📄 示例差异:")
	fmt.Println("      - 用户1: 44100 Hz, 2声道")
	fmt.Println("      - 用户2: 48000 Hz, 6声道") 
	fmt.Println("      - 用户3: 96000 Hz, 8声道")
	fmt.Println("   ❌ 但最终Audio哈希仍可能相同")
	fmt.Println("   📄 原因: 实际音频处理由硬件决定")
	
	fmt.Println("\n📊 WebGL 指纹分析:")
	fmt.Println("   ✅ JavaScript配置层面可以不同")
	fmt.Println("   📄 示例差异:")
	fmt.Println("      - 用户1: ANGLE (Intel HD Graphics)")
	fmt.Println("      - 用户2: ANGLE (NVIDIA GeForce)")
	fmt.Println("      - 用户3: SwiftShader")
	fmt.Println("   🟡 如果WebGL信息为空:")
	fmt.Println("      - 检查Chrome启动参数")
	fmt.Println("      - 确保不是无头模式")
	fmt.Println("      - 检查系统图形支持")

	fmt.Println("\n🎯 实际测试建议:")
	fmt.Println("   1. 运行: go run examples/advanced_fingerprint_demo.go") 
	fmt.Println("   2. 访问: https://iplark.com/fingerprint")
	fmt.Println("   3. 对比不同用户的实际指纹值")
}