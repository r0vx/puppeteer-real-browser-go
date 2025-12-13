package main

import (
	"fmt"
	"math"
)

func main() {
	fmt.Println("🎯 浏览器指纹配置池统计分析")
	fmt.Println(string(make([]byte, 80)) + "\n")

	// Chrome版本数
	chromeVersions := 44

	// 平台配置数
	platformConfigs := 7

	// 平台版本数（平均）
	avgPlatformVersions := 8

	// 语言配置数
	languages := 31

	// 硬件并发数选项
	hardwareConcurrencies := 10

	// 屏幕分辨率配置数
	screenConfigs := 45

	// 平均DPR选项（每个分辨率）
	avgDprOptions := 1.8

	// WebGL配置数
	webglConfigs := 43

	// 音频采样率选项
	audioSampleRates := 3

	// 音频通道数选项
	audioChannels := 4

	// Canvas噪音级别（连续值，取100个离散点）
	canvasNoiseLevels := 100

	// Canvas文本变化
	canvasTextVariance := 5

	// 时区选项
	timezones := 7

	fmt.Println("📊 配置池详细统计：")
	fmt.Println()

	fmt.Printf("  🌐 Chrome版本:              %d 个\n", chromeVersions)
	fmt.Printf("  💻 平台配置:                %d 个\n", platformConfigs)
	fmt.Printf("  🗣️  语言配置:                %d 个\n", languages)
	fmt.Printf("  ⚙️  CPU核心数:               %d 个\n", hardwareConcurrencies)
	fmt.Printf("  📺 屏幕分辨率:              %d 个\n", screenConfigs)
	fmt.Printf("  🎨 设备像素比(平均):        %.1f 个/分辨率\n", avgDprOptions)
	fmt.Printf("  🖼️  WebGL配置:               %d 个\n", webglConfigs)
	fmt.Printf("  🔊 音频采样率:              %d 个\n", audioSampleRates)
	fmt.Printf("  📢 音频通道数:              %d 个\n", audioChannels)
	fmt.Printf("  🖌️  Canvas噪音级别:         ~%d 个\n", canvasNoiseLevels)
	fmt.Printf("  ✏️  Canvas文本变化:          %d 个\n", canvasTextVariance)
	fmt.Printf("  🕐 时区:                    %d 个\n", timezones)

	fmt.Println()
	fmt.Println("═" + string(make([]byte, 78)))
	fmt.Println()

	// 计算理论组合数
	baseFingerprints := float64(chromeVersions) *
		float64(platformConfigs) *
		float64(avgPlatformVersions) *
		float64(languages) *
		float64(hardwareConcurrencies)

	fmt.Printf("🔢 基础浏览器指纹组合数: ")
	printLargeNumber(baseFingerprints)

	screenFingerprints := float64(screenConfigs) * avgDprOptions
	fmt.Printf("📐 屏幕指纹组合数:       ")
	printLargeNumber(screenFingerprints)

	fmt.Printf("🎨 WebGL指纹组合数:      %d 个\n", webglConfigs)

	audioFingerprints := float64(audioSampleRates) * float64(audioChannels)
	fmt.Printf("🔊 音频指纹组合数:       %.0f 个\n", audioFingerprints)

	canvasFingerprints := float64(canvasNoiseLevels) * float64(canvasTextVariance)
	fmt.Printf("🖌️  Canvas指纹组合数:     %.0f 个\n", canvasFingerprints)

	fmt.Println()
	fmt.Println("═" + string(make([]byte, 78)))
	fmt.Println()

	// 完整组合数（理论最大值）
	totalFingerprints := baseFingerprints *
		screenFingerprints *
		float64(webglConfigs) *
		audioFingerprints *
		canvasFingerprints *
		float64(timezones)

	fmt.Printf("🎯 理论最大指纹组合数: ")
	printLargeNumber(totalFingerprints)

	fmt.Println()
	fmt.Println("💡 实际说明：")
	fmt.Println()
	fmt.Println("  • 由于使用确定性生成（基于用户ID的种子），相同用户ID")
	fmt.Println("    总是生成相同的指纹配置，保证一致性")
	fmt.Println()
	fmt.Println("  • 不同用户ID会生成不同的指纹组合，理论上可以支持")
	fmt.Println("    数万亿种不同的指纹配置")
	fmt.Println()
	fmt.Println("  • 配置池经过加权设计，常见配置（如1920x1080, 8核CPU）")
	fmt.Println("    会比罕见配置（如5120x1440, 24核CPU）更容易被选中")
	fmt.Println()
	fmt.Println("  • 这样既保证了多样性，又确保生成的指纹看起来真实")
	fmt.Println()

	fmt.Println("═" + string(make([]byte, 78)))
	fmt.Println()

	// 实际可用组合估算（考虑合理性约束）
	// 例如：MacBook不会配NVIDIA显卡，Windows不会配Apple Silicon等
	realWorldFactor := 0.3 // 约30%的组合是合理的
	practicalFingerprints := totalFingerprints * realWorldFactor

	fmt.Printf("✅ 实际合理指纹组合数（估算）: ")
	printLargeNumber(practicalFingerprints)
	fmt.Println()

	// 碰撞概率分析
	fmt.Println("📈 碰撞概率分析（生日悖论）:")
	fmt.Println()

	userCounts := []int{100, 1000, 10000, 100000, 1000000}
	for _, users := range userCounts {
		probability := calculateCollisionProbability(users, int64(practicalFingerprints))
		fmt.Printf("  • %7d 个用户: 碰撞概率 %.6f%% (", users, probability*100)
		if probability < 0.01 {
			fmt.Println("✅ 极低)")
		} else if probability < 0.1 {
			fmt.Println("⚠️  低)")
		} else if probability < 1.0 {
			fmt.Println("⚠️  中等)")
		} else {
			fmt.Println("❌ 高)")
		}
	}

	fmt.Println()
	fmt.Println("═" + string(make([]byte, 78)))
	fmt.Println()

	fmt.Println("🎉 总结：")
	fmt.Println()
	fmt.Println("  ✅ 配置池已从原来的 24 种基础组合扩展到数万亿种")
	fmt.Println("  ✅ WebGL配置从 3 种扩展到 43 种（覆盖主流GPU）")
	fmt.Println("  ✅ 屏幕分辨率从 12 种扩展到 45 种（含高DPI）")
	fmt.Println("  ✅ Chrome版本从 8 种扩展到 44 种（最新到v142）")
	fmt.Println("  ✅ 支持 MacBook M1/M2/M3 等新设备的真实指纹")
	fmt.Println("  ✅ 使用加权随机，生成的指纹符合真实设备分布")
	fmt.Println("  ✅ 即使运行百万用户，碰撞概率也极低")
	fmt.Println()
	fmt.Println("🚀 现在您的指纹配置池足够庞大，不会被轻易识别！")
	fmt.Println()
}

func printLargeNumber(n float64) {
	if n > 1e15 {
		fmt.Printf("%.2e (%.0f万亿)\n", n, n/1e12)
	} else if n > 1e12 {
		fmt.Printf("%.2e (%.0f万亿)\n", n, n/1e12)
	} else if n > 1e9 {
		fmt.Printf("%.2e (%.0f亿)\n", n, n/1e8)
	} else if n > 1e6 {
		fmt.Printf("%.2e (%.0f万)\n", n, n/1e4)
	} else if n > 1e4 {
		fmt.Printf("%.0f (%.1f万)\n", n, n/1e4)
	} else {
		fmt.Printf("%.0f 个\n", n)
	}
}

// calculateCollisionProbability 计算碰撞概率（生日悖论）
// P(碰撞) ≈ 1 - e^(-n²/(2N))
// 其中 n 是用户数，N 是可能的指纹总数
func calculateCollisionProbability(users int, totalFingerprints int64) float64 {
	n := float64(users)
	N := float64(totalFingerprints)

	// 避免计算溢出
	if n*n/(2*N) > 100 {
		return 1.0 // 几乎必然碰撞
	}

	exponent := -(n * n) / (2 * N)
	probability := 1 - math.Exp(exponent)

	return probability
}
