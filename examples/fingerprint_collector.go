package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"path/filepath"
	"time"

	"github.com/HNRow/puppeteer-real-browser-go/pkg/browser"
)

func main() {
	fmt.Println("🔍 浏览器指纹收集测试")
	fmt.Println("======================")

	ctx := context.Background()

	// 获取扩展路径
	ext1, _ := filepath.Abs("examples/path/Extensions/kfjglmgfjedhhcddpfgfogkahmenikan/1.0_0")
	ext2, _ := filepath.Abs("examples/path/Extensions/mcohilncbfahbmgdjkbpemcciiolgcge/3.66.10_0")

	fmt.Printf("📂 Discord扩展: %s\n", ext1)
	fmt.Printf("📂 OKX扩展: %s\n", ext2)

	opts := &browser.ConnectOptions{
		Headless:       false,
		PersistProfile: true,
		ProfileName:    "fingerprint_test",
		Extensions:     []string{ext1, ext2},
		Args: []string{
			"--start-maximized",
			"--no-first-run",
		},
	}

	fmt.Println("🚀 启动带扩展的Chrome...")
	instance, err := browser.Connect(ctx, opts)
	if err != nil {
		log.Fatalf("❌ Chrome启动失败: %v", err)
	}
	defer instance.Close()

	fmt.Println("✅ Chrome启动成功")
	time.Sleep(3 * time.Second)

	page := instance.Page()

	// 导航到指纹检测网站
	fmt.Println("🌐 导航到 https://iplark.com/fingerprint ...")
	if err := page.Navigate("https://iplark.com/fingerprint"); err != nil {
		log.Printf("❌ 导航失败: %v", err)
		return
	}

	// 等待页面完全加载
	fmt.Println("⏳ 等待页面加载...")
	time.Sleep(10 * time.Second)

	// 收集指纹信息
	fmt.Println("📊 收集指纹参数...")
	result, err := page.Evaluate(`
		(() => {
			const fingerprint = {};
			
			// 基本浏览器信息
			fingerprint.userAgent = navigator.userAgent;
			fingerprint.language = navigator.language;
			fingerprint.languages = navigator.languages;
			fingerprint.platform = navigator.platform;
			fingerprint.vendor = navigator.vendor;
			fingerprint.cookieEnabled = navigator.cookieEnabled;
			fingerprint.doNotTrack = navigator.doNotTrack;
			fingerprint.hardwareConcurrency = navigator.hardwareConcurrency;
			fingerprint.maxTouchPoints = navigator.maxTouchPoints;
			fingerprint.webdriver = navigator.webdriver;
			
			// 屏幕信息
			fingerprint.screen = {
				width: screen.width,
				height: screen.height,
				availWidth: screen.availWidth,
				availHeight: screen.availHeight,
				colorDepth: screen.colorDepth,
				pixelDepth: screen.pixelDepth,
				devicePixelRatio: window.devicePixelRatio
			};
			
			// 时区信息
			fingerprint.timezone = {
				offset: new Date().getTimezoneOffset(),
				timezone: Intl.DateTimeFormat().resolvedOptions().timeZone
			};
			
			// WebGL信息
			try {
				const canvas = document.createElement('canvas');
				const gl = canvas.getContext('webgl') || canvas.getContext('experimental-webgl');
				if (gl) {
					fingerprint.webgl = {
						vendor: gl.getParameter(gl.VENDOR),
						renderer: gl.getParameter(gl.RENDERER),
						version: gl.getParameter(gl.VERSION),
						shadingLanguageVersion: gl.getParameter(gl.SHADING_LANGUAGE_VERSION),
						maxTextureSize: gl.getParameter(gl.MAX_TEXTURE_SIZE),
						maxViewportDims: gl.getParameter(gl.MAX_VIEWPORT_DIMS),
						maxRenderbufferSize: gl.getParameter(gl.MAX_RENDERBUFFER_SIZE)
					};
				}
			} catch (e) {
				fingerprint.webgl = { error: e.message };
			}
			
			// Canvas指纹
			try {
				const canvas = document.createElement('canvas');
				const ctx = canvas.getContext('2d');
				ctx.textBaseline = 'top';
				ctx.font = '14px Arial';
				ctx.fillText('Canvas fingerprint', 2, 2);
				fingerprint.canvas = canvas.toDataURL();
			} catch (e) {
				fingerprint.canvas = { error: e.message };
			}
			
			// 字体检测
			try {
				const baseFonts = ['monospace', 'sans-serif', 'serif'];
				const testFonts = ['Arial', 'Arial Black', 'Comic Sans MS', 'Courier New', 
					'Georgia', 'Helvetica', 'Impact', 'Lucida Console', 'Tahoma', 
					'Times New Roman', 'Trebuchet MS', 'Verdana'];
				
				const canvas = document.createElement('canvas');
				const ctx = canvas.getContext('2d');
				const detectedFonts = [];
				
				testFonts.forEach(font => {
					const baseWidth = ctx.measureText('mmmmmmmmmmlli').width;
					ctx.font = '72px ' + font + ', monospace';
					const testWidth = ctx.measureText('mmmmmmmmmmlli').width;
					if (testWidth !== baseWidth) {
						detectedFonts.push(font);
					}
				});
				
				fingerprint.fonts = detectedFonts;
			} catch (e) {
				fingerprint.fonts = { error: e.message };
			}
			
			// 插件信息
			fingerprint.plugins = Array.from(navigator.plugins).map(plugin => ({
				name: plugin.name,
				filename: plugin.filename,
				description: plugin.description,
				mimeTypes: Array.from(plugin).map(mime => ({
					type: mime.type,
					suffixes: mime.suffixes,
					description: mime.description
				}))
			}));
			
			// 媒体设备
			if (navigator.mediaDevices) {
				navigator.mediaDevices.enumerateDevices().then(devices => {
					fingerprint.mediaDevices = devices.map(device => ({
						kind: device.kind,
						label: device.label,
						deviceId: device.deviceId
					}));
				}).catch(e => {
					fingerprint.mediaDevices = { error: e.message };
				});
			}
			
			// 电池API
			if ('getBattery' in navigator) {
				navigator.getBattery().then(battery => {
					fingerprint.battery = {
						charging: battery.charging,
						chargingTime: battery.chargingTime,
						dischargingTime: battery.dischargingTime,
						level: battery.level
					};
				}).catch(e => {
					fingerprint.battery = { error: e.message };
				});
			}
			
			// 网络信息
			if ('connection' in navigator) {
				const conn = navigator.connection;
				fingerprint.connection = {
					effectiveType: conn.effectiveType,
					downlink: conn.downlink,
					rtt: conn.rtt,
					saveData: conn.saveData
				};
			}
			
			// 权限API
			fingerprint.permissions = {};
			const permissionsToCheck = ['camera', 'microphone', 'notifications', 'geolocation'];
			permissionsToCheck.forEach(permission => {
				if ('permissions' in navigator) {
					navigator.permissions.query({name: permission}).then(result => {
						fingerprint.permissions[permission] = result.state;
					}).catch(e => {
						fingerprint.permissions[permission] = { error: e.message };
					});
				}
			});
			
			// WebRTC信息
			try {
				const pc = new RTCPeerConnection();
				pc.createDataChannel('test');
				pc.createOffer().then(offer => {
					fingerprint.webrtc = {
						sdp: offer.sdp,
						hasWebRTC: true
					};
				}).catch(e => {
					fingerprint.webrtc = { error: e.message };
				});
			} catch (e) {
				fingerprint.webrtc = { hasWebRTC: false, error: e.message };
			}
			
			// Audio Context指纹
			try {
				const audioContext = new (window.AudioContext || window.webkitAudioContext)();
				const oscillator = audioContext.createOscillator();
				const analyser = audioContext.createAnalyser();
				const gain = audioContext.createGain();
				const scriptProcessor = audioContext.createScriptProcessor(4096, 1, 1);
				
				gain.gain.value = 0;
				oscillator.type = 'triangle';
				oscillator.connect(analyser);
				analyser.connect(scriptProcessor);
				scriptProcessor.connect(gain);
				gain.connect(audioContext.destination);
				oscillator.start(0);
				
				const data = new Float32Array(analyser.frequencyBinCount);
				analyser.getFloatFrequencyData(data);
				
				fingerprint.audio = {
					sampleRate: audioContext.sampleRate,
					maxChannelCount: audioContext.destination.maxChannelCount,
					numberOfInputs: audioContext.destination.numberOfInputs,
					numberOfOutputs: audioContext.destination.numberOfOutputs,
					frequencyData: Array.from(data).slice(0, 100)
				};
				
				audioContext.close();
			} catch (e) {
				fingerprint.audio = { error: e.message };
			}
			
			// 存储信息
			fingerprint.storage = {
				localStorage: !!window.localStorage,
				sessionStorage: !!window.sessionStorage,
				indexedDB: !!window.indexedDB,
				webSQL: !!window.openDatabase
			};
			
			// CSS媒体查询
			fingerprint.mediaQueries = {
				colorGamut: {
					srgb: matchMedia('(color-gamut: srgb)').matches,
					p3: matchMedia('(color-gamut: p3)').matches,
					rec2020: matchMedia('(color-gamut: rec2020)').matches
				},
				colorScheme: {
					light: matchMedia('(prefers-color-scheme: light)').matches,
					dark: matchMedia('(prefers-color-scheme: dark)').matches
				},
				reducedMotion: matchMedia('(prefers-reduced-motion: reduce)').matches,
				invertedColors: matchMedia('(inverted-colors: inverted)').matches,
				monochrome: matchMedia('(monochrome)').matches
			};
			
			return fingerprint;
		})()
	`)

	if err != nil {
		log.Printf("❌ 指纹收集失败: %v", err)
		return
	}

	fmt.Println("📋 指纹信息收集完成！")
	
	// 将结果格式化输出
	if fingerprintData, ok := result.(map[string]interface{}); ok {
		// 将结果转换为JSON格式便于阅读
		jsonData, err := json.MarshalIndent(fingerprintData, "", "  ")
		if err != nil {
			log.Printf("❌ JSON格式化失败: %v", err)
		} else {
			fmt.Println("🔍 收集到的指纹参数:")
			fmt.Println(string(jsonData))
		}
	}

	fmt.Println("\n💡 手动验证建议:")
	fmt.Println("  1. 检查扩展是否正确加载 (chrome://extensions/)")
	fmt.Println("  2. 对比无扩展时的指纹差异")
	fmt.Println("  3. 验证反检测效果")

	fmt.Println("\n⏳ 保持浏览器开启60秒供手动检查...")
	time.Sleep(60 * time.Second)

	fmt.Println("✅ 指纹收集测试完成")
}