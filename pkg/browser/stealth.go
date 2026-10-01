package browser

import (
	"context"

	"github.com/chromedp/chromedp"
)

// GetAdvancedStealthScript returns an advanced anti-detection script
// Based on rebrowser-patches and other anti-detection techniques
func GetAdvancedStealthScript() string {
	return `
		// Advanced anti-detection script for Go puppeteer-real-browser
		(() => {
			'use strict';
			
			// Prevent multiple injections
			if (window.__stealthInjected) return;
			window.__stealthInjected = true;
			
			// 1. Fix MouseEvent screenX and screenY properties (Critical for Cloudflare)
			if (!MouseEvent.prototype.hasOwnProperty('_screenFixed')) {
				const originalScreenX = Object.getOwnPropertyDescriptor(MouseEvent.prototype, 'screenX');
				const originalScreenY = Object.getOwnPropertyDescriptor(MouseEvent.prototype, 'screenY');
				
				Object.defineProperty(MouseEvent.prototype, 'screenX', {
					get: function() {
						return this.clientX + (window.screenX || 0);
					},
					configurable: true
				});
				
				Object.defineProperty(MouseEvent.prototype, 'screenY', {
					get: function() {
						return this.clientY + (window.screenY || 0);
					},
					configurable: true
				});
				
				MouseEvent.prototype._screenFixed = true;
			}
			
			// 2. Hide webdriver property completely (Critical)
			if (navigator.webdriver !== undefined) {
				Object.defineProperty(navigator, 'webdriver', {
					get: () => undefined,
					configurable: true
				});
			}
			
			// 3. Override automation-related properties
			// Make plugins look realistic
			Object.defineProperty(navigator, 'plugins', {
				get: () => {
					const plugins = [
						{
							name: 'Chrome PDF Plugin',
							filename: 'internal-pdf-viewer',
							description: 'Portable Document Format',
							length: 1
						},
						{
							name: 'Chrome PDF Viewer',
							filename: 'mhjfbmdgcfjbbpaeojofohoefgiehjai',
							description: '',
							length: 1
						},
						{
							name: 'Native Client',
							filename: 'internal-nacl-plugin',
							description: '',
							length: 2
						}
					];
					
					// Add array-like properties
					plugins.refresh = () => {};
					return plugins;
				},
				configurable: true
			});
			
			// 4. Override languages to be more realistic
			Object.defineProperty(navigator, 'languages', {
				get: () => ['en-US', 'en'],
				configurable: true
			});
			
			// 5. Fix permissions API
			if (navigator.permissions && navigator.permissions.query) {
				const originalQuery = navigator.permissions.query.bind(navigator.permissions);
				navigator.permissions.query = (parameters) => {
					if (parameters.name === 'notifications') {
						return Promise.resolve({ 
							state: Notification.permission,
							onchange: null
						});
					}
					return originalQuery(parameters);
				};
			}
			
			// 6. Hide automation control features and make chrome object realistic
			if (!window.chrome) {
				window.chrome = {};
			}
			
			if (!window.chrome.runtime) {
				window.chrome.runtime = {
					onConnect: undefined,
					onMessage: undefined,
					onConnectExternal: undefined,
					onMessageExternal: undefined
				};
			}
			
			// 7. Hide CDP and automation-related properties
			delete window.__nightmare;
			delete window._phantom;
			delete window.callPhantom;
			delete window.__webdriver_script_fn;
			delete window.__webdriver_evaluate;
			delete window.__selenium_unwrapped;
			delete window.__webdriver_unwrapped;
			delete window.__driver_evaluate;
			delete window.__webdriver_script_function;
			delete window.__fxdriver_evaluate;
			delete window.__driver_unwrapped;
			delete window.webdriver;
			delete window.domAutomation;
			delete window.domAutomationController;
			delete window.__lastWatirAlert;
			delete window.__lastWatirConfirm;
			delete window.__lastWatirPrompt;
			delete window._WEBDRIVER_ELEM_CACHE;
			
			// 8. Override console methods to hide automation traces
			const originalConsole = {
				debug: console.debug,
				log: console.log,
				warn: console.warn,
				error: console.error
			};
			
			const filterAutomationLogs = (method, args) => {
				const message = args.join(' ');
				// Filter out CDP and automation-related messages
				if (message.includes('DevTools') || 
					message.includes('Runtime.enable') ||
					message.includes('puppeteer') ||
					message.includes('chromedp') ||
					message.includes('automation')) {
					return;
				}
				return originalConsole[method].apply(console, args);
			};
			
			console.debug = (...args) => filterAutomationLogs('debug', args);
			console.log = (...args) => filterAutomationLogs('log', args);
			console.warn = (...args) => filterAutomationLogs('warn', args);
			console.error = (...args) => filterAutomationLogs('error', args);
			
			// 9. Override Error.stack to hide automation traces
			// DISABLED: This breaks some pages (e.g., Douyin QR code)
			// const originalPrepareStackTrace = Error.prepareStackTrace;
			// Error.prepareStackTrace = function(error, stack) { ... };
			
			// 10. Prevent detection through timing attacks
			const originalPerformanceNow = performance.now;
			let timeOffset = Math.random() * 10 - 5; // Random offset between -5 and 5ms
			performance.now = function() {
				return originalPerformanceNow.call(performance) + timeOffset;
			};
			
			// 11. Override document.createElement to hide automation iframes
			const originalCreateElement = document.createElement;
			document.createElement = function(tagName) {
				const element = originalCreateElement.call(document, tagName);
				if (tagName.toLowerCase() === 'iframe') {
					const originalSetAttribute = element.setAttribute;
					element.setAttribute = function(name, value) {
						if (name === 'src' && typeof value === 'string') {
							// Hide automation-related iframe sources
							if (value.includes('devtools') || 
								value.includes('chrome-extension') ||
								value.includes('moz-extension')) {
								return;
							}
						}
						return originalSetAttribute.call(element, name, value);
					};
				}
				return element;
			};
			
			// 12. Hide automation in user agent
			// 先取原值：getter 里再读 navigator.userAgent 会无限递归
			const originalUserAgent = navigator.userAgent;
			if (originalUserAgent.includes('HeadlessChrome')) {
				const cleanUserAgent = originalUserAgent.replace('HeadlessChrome', 'Chrome');
				Object.defineProperty(navigator, 'userAgent', {
					get: () => cleanUserAgent,
					configurable: true
				});
			}
			
			// 13. Fix vendor and product properties
			Object.defineProperty(navigator, 'vendor', {
				get: () => 'Google Inc.',
				configurable: true
			});
			
			Object.defineProperty(navigator, 'product', {
				get: () => 'Gecko',
				configurable: true
			});
			
			// 14. Add realistic hardwareConcurrency
			if (!navigator.hardwareConcurrency || navigator.hardwareConcurrency === 1) {
				Object.defineProperty(navigator, 'hardwareConcurrency', {
					get: () => 4,
					configurable: true
				});
			}
			
			// 15. Fix deviceMemory if it's suspicious
			if (navigator.deviceMemory && navigator.deviceMemory < 2) {
				Object.defineProperty(navigator, 'deviceMemory', {
					get: () => 8,
					configurable: true
				});
			}
			
			// 16. Override Notification.permission if needed
			if (Notification.permission === 'default') {
				Object.defineProperty(Notification, 'permission', {
					get: () => 'denied',
					configurable: true
				});
			}
			
			// 17. Hide automation in window.external
			if (window.external && window.external.toString().includes('Sequentum')) {
				delete window.external;
			}
			
			// 18. Add realistic connection info
			if (navigator.connection) {
				Object.defineProperty(navigator.connection, 'rtt', {
					get: () => 50 + Math.random() * 50,
					configurable: true
				});
			}
			
			// 19. Override toString methods to hide automation
			const originalToString = Function.prototype.toString;
			Function.prototype.toString = function() {
				const result = originalToString.call(this);
				if (result.includes('native code') && 
					(result.includes('puppeteer') || result.includes('chromedp'))) {
					return 'function () { [native code] }';
				}
				return result;
			};
			
			// 20. Fix automation detection in window.chrome
			if (window.chrome && window.chrome.runtime) {
				Object.defineProperty(window.chrome.runtime, 'onConnect', {
					get: () => undefined,
					configurable: true
				});
				Object.defineProperty(window.chrome.runtime, 'onMessage', {
					get: () => undefined,
					configurable: true
				});
			}
			
			// 21. Override navigator.maxTouchPoints
			Object.defineProperty(navigator, 'maxTouchPoints', {
				get: () => 0,
				configurable: true
			});
			
			// 22. Fix automation detection in window.outerHeight/outerWidth (Critical for Cloudflare)
			// This is a common detection vector - headless browsers often have outer dimensions of 0
			if (window.outerHeight === 0 || window.outerWidth === 0) {
				Object.defineProperty(window, 'outerHeight', {
					get: () => window.innerHeight + 120, // Typical browser chrome height
					configurable: true,
					enumerable: true
				});
				Object.defineProperty(window, 'outerWidth', {
					get: () => window.innerWidth + 16, // Typical browser chrome width
					configurable: true,
					enumerable: true
				});
			}
			
			// 23. Override navigator.onLine
			Object.defineProperty(navigator, 'onLine', {
				get: () => true,
				configurable: true
			});
			
			// 24. Fix automation detection in window.screen
			if (window.screen) {
				Object.defineProperty(window.screen, 'availLeft', {
					get: () => 0,
					configurable: true
				});
				Object.defineProperty(window.screen, 'availTop', {
					get: () => 0,
					configurable: true
				});
			}
			
			// 25. Canvas fingerprint protection (Critical)
			// 为 Canvas 添加微小噪声，使每次指纹不同
			(function() {
				const noiseSeed = Math.random() * 0.01;
				
				// 保存原始方法
				const originalToDataURL = HTMLCanvasElement.prototype.toDataURL;
				const originalGetImageData = CanvasRenderingContext2D.prototype.getImageData;
				const originalToBlob = HTMLCanvasElement.prototype.toBlob;
				
				// 添加噪声的辅助函数
				function addNoise(data) {
					for (let i = 0; i < data.length; i += 4) {
						// 仅修改少量像素，避免视觉差异
						if (Math.random() < 0.01) {
							data[i] = Math.max(0, Math.min(255, data[i] + Math.floor(Math.random() * 3) - 1));
							data[i + 1] = Math.max(0, Math.min(255, data[i + 1] + Math.floor(Math.random() * 3) - 1));
							data[i + 2] = Math.max(0, Math.min(255, data[i + 2] + Math.floor(Math.random() * 3) - 1));
						}
					}
				}
				
				// 重写 toDataURL
				HTMLCanvasElement.prototype.toDataURL = function(type, quality) {
					const ctx = this.getContext('2d');
					if (ctx && this.width > 0 && this.height > 0) {
						try {
							const imageData = originalGetImageData.call(ctx, 0, 0, this.width, this.height);
							addNoise(imageData.data);
							ctx.putImageData(imageData, 0, 0);
						} catch(e) {}
					}
					return originalToDataURL.call(this, type, quality);
				};
				
				// 重写 getImageData
				CanvasRenderingContext2D.prototype.getImageData = function(sx, sy, sw, sh) {
					const imageData = originalGetImageData.call(this, sx, sy, sw, sh);
					addNoise(imageData.data);
					return imageData;
				};
				
				// 重写 toBlob
				if (originalToBlob) {
					HTMLCanvasElement.prototype.toBlob = function(callback, type, quality) {
						const ctx = this.getContext('2d');
						if (ctx && this.width > 0 && this.height > 0) {
							try {
								const imageData = originalGetImageData.call(ctx, 0, 0, this.width, this.height);
								addNoise(imageData.data);
								ctx.putImageData(imageData, 0, 0);
							} catch(e) {}
						}
						return originalToBlob.call(this, callback, type, quality);
					};
				}
			})();
			
			// 26. WebGL fingerprint protection (Critical)
			// 伪装 WebGL 渲染器信息 - 在原型链上修改
			(function() {
				const vendors = ['Intel Inc.', 'Google Inc. (Intel)'];
				const renderers = [
					'ANGLE (Intel, Intel(R) UHD Graphics 630, OpenGL 4.1)',
					'ANGLE (Intel, Intel(R) Iris Plus Graphics 655, OpenGL 4.1)',
					'Intel(R) UHD Graphics 630',
					'Intel Iris OpenGL Engine'
				];
				
				// 使用固定种子确保同一会话内一致
				const seed = Date.now() % 1000;
				const selectedVendor = vendors[seed % vendors.length];
				const selectedRenderer = renderers[seed % renderers.length];
				
				// 修改 WebGLRenderingContext 原型
				if (window.WebGLRenderingContext) {
					const originalGetParameter = WebGLRenderingContext.prototype.getParameter;
					WebGLRenderingContext.prototype.getParameter = function(parameter) {
						// UNMASKED_VENDOR_WEBGL = 37445
						if (parameter === 37445) return selectedVendor;
						// UNMASKED_RENDERER_WEBGL = 37446
						if (parameter === 37446) return selectedRenderer;
						return originalGetParameter.call(this, parameter);
					};
				}
				
				// 修改 WebGL2RenderingContext 原型
				if (window.WebGL2RenderingContext) {
					const originalGetParameter2 = WebGL2RenderingContext.prototype.getParameter;
					WebGL2RenderingContext.prototype.getParameter = function(parameter) {
						if (parameter === 37445) return selectedVendor;
						if (parameter === 37446) return selectedRenderer;
						return originalGetParameter2.call(this, parameter);
					};
				}
			})();
			
			// 27. AudioContext fingerprint protection
			// 为音频处理添加微小噪声
			(function() {
				const OriginalAudioContext = window.AudioContext || window.webkitAudioContext;
				if (!OriginalAudioContext) return;
				
				// 每个会话的固定噪声因子
				const sessionNoise = (Date.now() % 10000) / 100000; // 0.00001 - 0.1

				function ModifiedAudioContext(...args) {
					const ctx = new OriginalAudioContext(...args);
					
					// 重写 createAnalyser
					const originalCreateAnalyser = ctx.createAnalyser.bind(ctx);
					ctx.createAnalyser = function() {
						const analyser = originalCreateAnalyser();
						
						const originalGetFloatFrequencyData = analyser.getFloatFrequencyData.bind(analyser);
						analyser.getFloatFrequencyData = function(array) {
							originalGetFloatFrequencyData(array);
							for (let i = 0; i < array.length; i++) {
								array[i] += (Math.random() - 0.5) * sessionNoise * 100;
							}
						};
						
						const originalGetByteFrequencyData = analyser.getByteFrequencyData.bind(analyser);
						analyser.getByteFrequencyData = function(array) {
							originalGetByteFrequencyData(array);
							for (let i = 0; i < array.length; i++) {
								array[i] = Math.max(0, Math.min(255, array[i] + Math.floor((Math.random() - 0.5) * 3)));
							}
						};
						
						// 修改时域数据
						const originalGetFloatTimeDomainData = analyser.getFloatTimeDomainData.bind(analyser);
						analyser.getFloatTimeDomainData = function(array) {
							originalGetFloatTimeDomainData(array);
							for (let i = 0; i < array.length; i++) {
								array[i] += (Math.random() - 0.5) * sessionNoise;
							}
						};
						
						return analyser;
					};
					
					// 不重写 createOscillator：在 frequency 实例上遮蔽 value 只改 JS 读数、不影响渲染，
					// 既不产生指纹噪声，又让读回值与设置值不一致（可被检测），setter 还吞掉赋值

					// 重写 createDynamicsCompressor
					const originalCreateDynamicsCompressor = ctx.createDynamicsCompressor.bind(ctx);
					ctx.createDynamicsCompressor = function() {
						const compressor = originalCreateDynamicsCompressor();
						// 微调压缩器参数
						compressor.threshold.value += sessionNoise * 10;
						compressor.knee.value += sessionNoise * 5;
						return compressor;
					};
					
					return ctx;
				}
				
				ModifiedAudioContext.prototype = OriginalAudioContext.prototype;
				Object.defineProperty(ModifiedAudioContext, 'name', { value: 'AudioContext' });
				
				window.AudioContext = ModifiedAudioContext;
				if (window.webkitAudioContext) {
					window.webkitAudioContext = ModifiedAudioContext;
				}
			})();
			
			// 28. Font fingerprint protection
			// 限制字体检测能力
			(function() {
				if (document.fonts && document.fonts.check) {
					const originalCheck = document.fonts.check.bind(document.fonts);
					document.fonts.check = function(font, text) {
						// 对于常见系统字体返回 true，减少差异性
						const commonFonts = ['Arial', 'Helvetica', 'Times New Roman', 'Georgia', 'Verdana'];
						for (const cf of commonFonts) {
							if (font.includes(cf)) return true;
						}
						return originalCheck(font, text);
					};
				}
			})();
			
			// 29. Battery API protection
			// 伪装电池 API 返回固定值
			if (navigator.getBattery) {
				const fakeBattery = {
					charging: true,
					chargingTime: 0,
					dischargingTime: Infinity,
					level: 1.0,
					addEventListener: function() {},
					removeEventListener: function() {},
					dispatchEvent: function() { return true; },
					onchargingchange: null,
					onchargingtimechange: null,
					ondischargingtimechange: null,
					onlevelchange: null
				};
				Object.defineProperty(navigator, 'getBattery', {
					value: function() { return Promise.resolve(fakeBattery); },
					writable: false,
					configurable: true
				});
			}
			
			// 30. Bluetooth/USB API protection
			// 移除蓝牙和 USB API
			delete navigator.bluetooth;
			delete navigator.usb;
			
			// 31. WebRTC IP 泄露防护 (Critical for proxy users)
			// 防止通过 WebRTC 泄露真实 IP 地址
			(function() {
				// 方案1: 完全禁用 RTCPeerConnection
				// 这是最安全的方案，但可能影响某些网站功能
				const fakeRTCPeerConnection = function() {
					return {
						createDataChannel: () => ({}),
						createOffer: () => Promise.resolve({}),
						createAnswer: () => Promise.resolve({}),
						setLocalDescription: () => Promise.resolve(),
						setRemoteDescription: () => Promise.resolve(),
						addIceCandidate: () => Promise.resolve(),
						getStats: () => Promise.resolve(new Map()),
						getSenders: () => [],
						getReceivers: () => [],
						getTransceivers: () => [],
						addTrack: () => ({ track: null }),
						removeTrack: () => {},
						addTransceiver: () => ({}),
						close: () => {},
						get localDescription() { return null; },
						get remoteDescription() { return null; },
						get signalingState() { return 'closed'; },
						get iceGatheringState() { return 'complete'; },
						get iceConnectionState() { return 'closed'; },
						get connectionState() { return 'closed'; },
						get canTrickleIceCandidates() { return null; },
						onicecandidate: null,
						onicegatheringstatechange: null,
						oniceconnectionstatechange: null,
						onconnectionstatechange: null,
						onsignalingstatechange: null,
						ontrack: null,
						ondatachannel: null,
						onnegotiationneeded: null,
						addEventListener: () => {},
						removeEventListener: () => {},
						dispatchEvent: () => true
					};
				};
				fakeRTCPeerConnection.prototype = {};
				
				// 覆盖所有 RTCPeerConnection 变体
				if (window.RTCPeerConnection) {
					window.RTCPeerConnection = fakeRTCPeerConnection;
				}
				if (window.webkitRTCPeerConnection) {
					window.webkitRTCPeerConnection = fakeRTCPeerConnection;
				}
				if (window.mozRTCPeerConnection) {
					window.mozRTCPeerConnection = fakeRTCPeerConnection;
				}
				
				// 方案2: 禁用 getUserMedia（防止摄像头/麦克风访问泄露 IP）
				if (navigator.mediaDevices) {
					navigator.mediaDevices.getUserMedia = function() {
						return Promise.reject(new DOMException('Permission denied', 'NotAllowedError'));
					};
					navigator.mediaDevices.getDisplayMedia = function() {
						return Promise.reject(new DOMException('Permission denied', 'NotAllowedError'));
					};
				}
				
				// 旧版 API
				if (navigator.getUserMedia) {
					navigator.getUserMedia = function(constraints, success, error) {
						error(new DOMException('Permission denied', 'NotAllowedError'));
					};
				}
				if (navigator.webkitGetUserMedia) {
					navigator.webkitGetUserMedia = function(constraints, success, error) {
						error(new DOMException('Permission denied', 'NotAllowedError'));
					};
				}
				if (navigator.mozGetUserMedia) {
					navigator.mozGetUserMedia = function(constraints, success, error) {
						error(new DOMException('Permission denied', 'NotAllowedError'));
					};
				}
				
				// 方案3: 移除 RTCDataChannel
				if (window.RTCDataChannel) {
					window.RTCDataChannel = function() {
						throw new DOMException('RTCDataChannel is not supported', 'NotSupportedError');
					};
				}
				
				// 方案4: 移除 RTCSessionDescription
				if (window.RTCSessionDescription) {
					window.RTCSessionDescription = function() {
						return { type: '', sdp: '' };
					};
				}
				
				// 方案5: 移除 RTCIceCandidate
				if (window.RTCIceCandidate) {
					window.RTCIceCandidate = function() {
						return { candidate: '', sdpMid: '', sdpMLineIndex: 0 };
					};
				}
			})();
			
			console.log('🛡️ Advanced stealth mode activated (with fingerprint + WebRTC protection)');
		})();
	`
}

// InjectAdvancedStealthScripts injects comprehensive anti-detection scripts
func InjectAdvancedStealthScripts() chromedp.Action {
	return chromedp.ActionFunc(func(ctx context.Context) error {
		script := GetAdvancedStealthScript()
		return chromedp.Evaluate(script, nil).Do(ctx)
	})
}

// InjectStealthOnNewDocument injects stealth scripts on every new document
func InjectStealthOnNewDocument() chromedp.Action {
	return chromedp.ActionFunc(func(ctx context.Context) error {
		script := GetAdvancedStealthScript()
		// Use Evaluate for now since AddScriptToEvaluateOnNewDocument might not be available
		return chromedp.Evaluate(script, nil).Do(ctx)
	})
}

// GetStealthScriptWithConfig generates stealth script with custom fingerprint configuration
// 使用自定义指纹配置生成 stealth 脚本
func GetStealthScriptWithConfig(config *FingerprintConfig) string {
	if config == nil {
		return GetAdvancedStealthScript()
	}

	// 使用 FingerprintInjector 生成完整的指纹注入脚本
	injector := NewFingerprintInjector(config)
	fingerprintScript := injector.GenerateInjectionScript()

	// 获取基础 stealth 脚本（去掉默认的指纹保护部分）
	baseScript := GetBaseStealthScript()

	// 合并脚本
	return baseScript + "\n" + fingerprintScript
}

// GetBaseStealthScript returns the base stealth script without fingerprint randomization
// 返回不包含指纹随机化的基础 stealth 脚本（用于自定义指纹时）
func GetBaseStealthScript() string {
	return `
		// Base anti-detection script (without fingerprint randomization)
		(() => {
			'use strict';
			
			if (window.__stealthInjected) return;
			window.__stealthInjected = true;
			
			// 1. Fix MouseEvent screenX and screenY
			if (!MouseEvent.prototype.hasOwnProperty('_screenFixed')) {
				Object.defineProperty(MouseEvent.prototype, 'screenX', {
					get: function() { return this.clientX + (window.screenX || 0); },
					configurable: true
				});
				Object.defineProperty(MouseEvent.prototype, 'screenY', {
					get: function() { return this.clientY + (window.screenY || 0); },
					configurable: true
				});
				MouseEvent.prototype._screenFixed = true;
			}
			
			// 2. Hide webdriver
			if (navigator.webdriver !== undefined) {
				Object.defineProperty(navigator, 'webdriver', {
					get: () => undefined,
					configurable: true
				});
			}
			
			// 3. Hide CDP properties
			delete window.__nightmare;
			delete window._phantom;
			delete window.callPhantom;
			delete window.__webdriver_script_fn;
			delete window.__webdriver_evaluate;
			delete window.__selenium_unwrapped;
			delete window.__webdriver_unwrapped;
			delete window.__driver_evaluate;
			delete window.__webdriver_script_function;
			delete window.__fxdriver_evaluate;
			delete window.__driver_unwrapped;
			delete window.webdriver;
			delete window.domAutomation;
			delete window.domAutomationController;
			
			// 4. Fix chrome runtime
			if (!window.chrome) window.chrome = {};
			if (!window.chrome.runtime) {
				window.chrome.runtime = {
					onConnect: undefined,
					onMessage: undefined,
					onConnectExternal: undefined,
					onMessageExternal: undefined
				};
			}
			
			// 5. Fix window dimensions
			if (window.outerHeight === 0 || window.outerWidth === 0) {
				Object.defineProperty(window, 'outerHeight', {
					get: () => window.innerHeight + 120,
					configurable: true
				});
				Object.defineProperty(window, 'outerWidth', {
					get: () => window.innerWidth + 16,
					configurable: true
				});
			}
			
			// 6. Remove Bluetooth/USB API
			delete navigator.bluetooth;
			delete navigator.usb;
			
			// 7. WebRTC IP 泄露防护 (Critical for proxy users)
			(function() {
				// 完全禁用 RTCPeerConnection 防止 IP 泄露
				const fakeRTCPeerConnection = function() {
					return {
						createDataChannel: () => ({}),
						createOffer: () => Promise.resolve({}),
						createAnswer: () => Promise.resolve({}),
						setLocalDescription: () => Promise.resolve(),
						setRemoteDescription: () => Promise.resolve(),
						addIceCandidate: () => Promise.resolve(),
						getStats: () => Promise.resolve(new Map()),
						getSenders: () => [],
						getReceivers: () => [],
						getTransceivers: () => [],
						addTrack: () => ({ track: null }),
						removeTrack: () => {},
						addTransceiver: () => ({}),
						close: () => {},
						get localDescription() { return null; },
						get remoteDescription() { return null; },
						get signalingState() { return 'closed'; },
						get iceGatheringState() { return 'complete'; },
						get iceConnectionState() { return 'closed'; },
						get connectionState() { return 'closed'; },
						get canTrickleIceCandidates() { return null; },
						onicecandidate: null,
						onicegatheringstatechange: null,
						oniceconnectionstatechange: null,
						onconnectionstatechange: null,
						onsignalingstatechange: null,
						ontrack: null,
						ondatachannel: null,
						onnegotiationneeded: null,
						addEventListener: () => {},
						removeEventListener: () => {},
						dispatchEvent: () => true
					};
				};
				fakeRTCPeerConnection.prototype = {};
				
				if (window.RTCPeerConnection) window.RTCPeerConnection = fakeRTCPeerConnection;
				if (window.webkitRTCPeerConnection) window.webkitRTCPeerConnection = fakeRTCPeerConnection;
				if (window.mozRTCPeerConnection) window.mozRTCPeerConnection = fakeRTCPeerConnection;
				
				// 禁用 getUserMedia
				if (navigator.mediaDevices) {
					navigator.mediaDevices.getUserMedia = () => Promise.reject(new DOMException('Permission denied', 'NotAllowedError'));
					navigator.mediaDevices.getDisplayMedia = () => Promise.reject(new DOMException('Permission denied', 'NotAllowedError'));
				}
				if (navigator.getUserMedia) navigator.getUserMedia = (c, s, e) => e(new DOMException('Permission denied', 'NotAllowedError'));
				if (navigator.webkitGetUserMedia) navigator.webkitGetUserMedia = (c, s, e) => e(new DOMException('Permission denied', 'NotAllowedError'));
				
				// 禁用相关 API
				if (window.RTCDataChannel) window.RTCDataChannel = function() { throw new DOMException('Not supported', 'NotSupportedError'); };
				if (window.RTCSessionDescription) window.RTCSessionDescription = function() { return { type: '', sdp: '' }; };
				if (window.RTCIceCandidate) window.RTCIceCandidate = function() { return { candidate: '', sdpMid: '', sdpMLineIndex: 0 }; };
			})();
			
			console.log('🛡️ Base stealth mode activated (with WebRTC protection)');
		})();
	`
}
