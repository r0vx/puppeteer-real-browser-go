package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/chromedp/chromedp"
)

// CollectedFingerprint 收集到的原始指纹数据
type CollectedFingerprint struct {
	UserID      string                 `json:"user_id"`
	CollectedAt string                 `json:"collected_at"`
	Screen      map[string]interface{} `json:"screen"`
	Browser     map[string]interface{} `json:"browser"`
	System      map[string]interface{} `json:"system"`
	WebGL       map[string]interface{} `json:"webgl"`
	Audio       map[string]interface{} `json:"audio"`
	Canvas      map[string]interface{} `json:"canvas"`
	Fonts       map[string]interface{} `json:"fonts"`
	Plugins     map[string]interface{} `json:"plugins"`
	Battery     map[string]interface{} `json:"battery"`
	MediaDevices []map[string]interface{} `json:"media_devices"`
	Network     map[string]interface{} `json:"network"`
	Timezone    map[string]interface{} `json:"timezone"`
}

func main() {
	fmt.Println("🔍 浏览器指纹收集工具")
	fmt.Println("=" + string(make([]byte, 50)))
	fmt.Println()

	// 显示菜单
	fmt.Println("请选择收集方式：")
	fmt.Println("1. 启动本地HTML页面收集（推荐）")
	fmt.Println("2. 使用真实浏览器自动收集")
	fmt.Println("3. 批量收集多个设备指纹")
	fmt.Println()

	var choice int
	fmt.Print("请输入选项 (1-3): ")
	fmt.Scanln(&choice)

	switch choice {
	case 1:
		startHTMLCollector()
	case 2:
		collectWithRealBrowser()
	case 3:
		batchCollect()
	default:
		fmt.Println("❌ 无效选项")
	}
}

// startHTMLCollector 启动HTML收集页面
func startHTMLCollector() {
	fmt.Println("\n🌐 启动本地Web服务器...")

	// 获取HTML文件路径
	htmlPath := filepath.Join(".", "fingerprint_collector.html")
	if _, err := os.Stat(htmlPath); os.IsNotExist(err) {
		log.Fatalf("❌ 找不到 fingerprint_collector.html 文件")
	}

	// 创建输出目录
	outputDir := "./collected_fingerprints"
	os.MkdirAll(outputDir, 0755)

	// 设置HTTP处理器
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, htmlPath)
	})

	http.HandleFunc("/save", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		body, err := ioutil.ReadAll(r.Body)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		var collected CollectedFingerprint
		if err := json.Unmarshal(body, &collected); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		// 保存原始数据
		filename := fmt.Sprintf("%s/%s.json", outputDir, collected.UserID)
		if err := ioutil.WriteFile(filename, body, 0644); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		fmt.Printf("✅ 保存指纹: %s\n", filename)

		// 转换为项目配置格式
		convertToProjectConfig(&collected, outputDir)

		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	port := 8765
	addr := fmt.Sprintf("localhost:%d", port)
	
	fmt.Printf("✅ 服务器已启动: http://%s\n", addr)
	fmt.Println()
	fmt.Println("📋 使用说明：")
	fmt.Println("1. 在浏览器中打开: http://" + addr)
	fmt.Println("2. 页面会自动收集指纹并显示")
	fmt.Println("3. 点击「下载JSON配置」保存文件")
	fmt.Println("4. 收集到的指纹会保存在 ./collected_fingerprints/ 目录")
	fmt.Println()
	fmt.Println("💡 提示: 可以在不同的设备/浏览器中打开此页面来收集多个指纹")
	fmt.Println("按 Ctrl+C 停止服务器")
	fmt.Println()

	if err := http.ListenAndServe(addr, nil); err != nil {
		log.Fatal(err)
	}
}

// collectWithRealBrowser 使用真实浏览器自动收集
func collectWithRealBrowser() {
	fmt.Println("\n🤖 启动自动收集模式...")
	
	// 创建输出目录
	outputDir := "./collected_fingerprints"
	os.MkdirAll(outputDir, 0755)

	// 创建浏览器上下文
	ctx, cancel := chromedp.NewContext(context.Background())
	defer cancel()

	ctx, cancel = context.WithTimeout(ctx, 60*time.Second)
	defer cancel()

	// 获取HTML文件路径
	htmlPath := filepath.Join(".", "fingerprint_collector.html")
	absPath, _ := filepath.Abs(htmlPath)
	fileURL := "file://" + absPath

	fmt.Printf("📄 加载收集页面: %s\n", fileURL)

	var fingerprintJSON string

	err := chromedp.Run(ctx,
		chromedp.Navigate(fileURL),
		chromedp.Sleep(10*time.Second), // 等待收集完成
		
		// 提取收集到的数据
		chromedp.Evaluate(`JSON.stringify(window.fingerprintData || {})`, &fingerprintJSON),
	)

	if err != nil {
		log.Fatalf("❌ 收集失败: %v", err)
	}

	if fingerprintJSON == "" || fingerprintJSON == "{}" {
		log.Fatal("❌ 未收集到指纹数据")
	}

	// 解析数据
	var collected CollectedFingerprint
	if err := json.Unmarshal([]byte(fingerprintJSON), &collected); err != nil {
		log.Fatalf("❌ 解析失败: %v", err)
	}

	// 保存原始数据
	filename := fmt.Sprintf("%s/%s.json", outputDir, collected.UserID)
	data, _ := json.MarshalIndent(collected, "", "  ")
	if err := ioutil.WriteFile(filename, data, 0644); err != nil {
		log.Fatalf("❌ 保存失败: %v", err)
	}

	fmt.Printf("✅ 指纹已保存: %s\n", filename)

	// 转换为项目配置格式
	convertToProjectConfig(&collected, outputDir)
}

// batchCollect 批量收集
func batchCollect() {
	fmt.Println("\n📦 批量收集模式")
	fmt.Println("此模式需要您在多台设备上手动运行浏览器")
	fmt.Println()
	
	var count int
	fmt.Print("请输入要收集的设备数量: ")
	fmt.Scanln(&count)

	fmt.Printf("\n正在启动Web服务器，请在 %d 台设备上访问...\n", count)
	startHTMLCollector()
}

// convertToProjectConfig 转换为项目配置格式
func convertToProjectConfig(collected *CollectedFingerprint, outputDir string) {
	// 导入项目的FingerprintConfig结构
	// 这里需要手动映射字段
	
	config := map[string]interface{}{
		"user_id": collected.UserID,
		"screen": map[string]interface{}{
			"width":               getInt(collected.Screen, "width"),
			"height":              getInt(collected.Screen, "height"),
			"avail_width":         getInt(collected.Screen, "avail_width"),
			"avail_height":        getInt(collected.Screen, "avail_height"),
			"color_depth":         getInt(collected.Screen, "color_depth"),
			"pixel_depth":         getInt(collected.Screen, "pixel_depth"),
			"device_pixel_ratio":  getFloat(collected.Screen, "device_pixel_ratio"),
		},
		"browser": map[string]interface{}{
			"user_agent":           getString(collected.Browser, "user_agent"),
			"language":             getString(collected.Browser, "language"),
			"languages":            collected.Browser["languages"],
			"platform":             getString(collected.Browser, "platform"),
			"vendor":               getString(collected.Browser, "vendor"),
			"cookie_enabled":       getBool(collected.Browser, "cookie_enabled"),
			"do_not_track":         collected.Browser["do_not_track"],
			"hardware_concurrency": getInt(collected.Browser, "hardware_concurrency"),
			"max_touch_points":     getInt(collected.Browser, "max_touch_points"),
			"webdriver":            nil, // 始终设为nil以隐藏自动化
		},
		"system": collected.System,
		"webgl": map[string]interface{}{
			"vendor":                   getString(collected.WebGL, "vendor"),
			"renderer":                 getString(collected.WebGL, "renderer"),
			"version":                  getString(collected.WebGL, "version"),
			"shading_language_version": getString(collected.WebGL, "shading_language_version"),
			"max_texture_size":         getInt(collected.WebGL, "max_texture_size"),
			"max_renderbuffer_size":    getInt(collected.WebGL, "max_renderbuffer_size"),
		},
		"audio": map[string]interface{}{
			"sample_rate":       getInt(collected.Audio, "sample_rate"),
			"max_channel_count": getInt(collected.Audio, "max_channel_count"),
			"number_of_inputs":  getInt(collected.Audio, "number_of_inputs"),
			"number_of_outputs": getInt(collected.Audio, "number_of_outputs"),
		},
		"canvas": map[string]interface{}{
			"noise_level":   getFloat(collected.Canvas, "noise_level"),
			"text_variance": getInt(collected.Canvas, "text_variance"),
		},
		"fonts":         collected.Fonts,
		"plugins":       collected.Plugins,
		"battery":       collected.Battery,
		"media_devices": collected.MediaDevices,
		"network":       collected.Network,
		"timezone": map[string]interface{}{
			"offset":   getInt(collected.Timezone, "offset"),
			"timezone": getString(collected.Timezone, "timezone"),
		},
		"tls_config": map[string]interface{}{
			"ja4":         "t13d1516_8daaf6152771_e5627efa2ab1", // 默认Chrome指纹
			"tls_version": "TLS 1.3",
		},
		"http2_config": map[string]interface{}{
			"akamai": "1:65536:3:1000",
		},
	}

	// 保存转换后的配置
	configFilename := fmt.Sprintf("%s/%s_config.json", outputDir, collected.UserID)
	data, _ := json.MarshalIndent(config, "", "  ")
	if err := ioutil.WriteFile(configFilename, data, 0644); err != nil {
		fmt.Printf("⚠️  保存配置失败: %v\n", err)
		return
	}

	fmt.Printf("✅ 项目配置已保存: %s\n", configFilename)
	fmt.Println()
	fmt.Println("📋 指纹信息摘要:")
	fmt.Printf("   🌐 User-Agent: %s\n", getString(collected.Browser, "user_agent"))
	fmt.Printf("   🖥️  Platform: %s\n", getString(collected.Browser, "platform"))
	fmt.Printf("   📺 Screen: %dx%d\n", getInt(collected.Screen, "width"), getInt(collected.Screen, "height"))
	fmt.Printf("   🎨 WebGL: %s\n", getString(collected.WebGL, "renderer"))
	fmt.Printf("   🔊 Audio Sample Rate: %d Hz\n", getInt(collected.Audio, "sample_rate"))
	fmt.Println()
}

// 辅助函数
func getString(m map[string]interface{}, key string) string {
	if v, ok := m[key]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

func getInt(m map[string]interface{}, key string) int {
	if v, ok := m[key]; ok {
		switch val := v.(type) {
		case float64:
			return int(val)
		case int:
			return val
		}
	}
	return 0
}

func getFloat(m map[string]interface{}, key string) float64 {
	if v, ok := m[key]; ok {
		if f, ok := v.(float64); ok {
			return f
		}
	}
	return 0.0
}

func getBool(m map[string]interface{}, key string) bool {
	if v, ok := m[key]; ok {
		if b, ok := v.(bool); ok {
			return b
		}
	}
	return false
}

