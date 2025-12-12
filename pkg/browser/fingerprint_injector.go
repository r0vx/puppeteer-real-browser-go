package browser

import (
	"crypto/md5"
	"fmt"
	"strings"
	"sync"
)

// FingerprintInjector JavaScript注入器，用于修改浏览器指纹
type FingerprintInjector struct {
	config                *FingerprintConfig
	audioWebGLInjector    *EnhancedAudioWebGLInjector
	timestampInjector     *TimestampFingerprintInjector
	scriptCache           string
	scriptCacheMu         sync.RWMutex
	cacheValid            bool
}

// NewFingerprintInjector 创建指纹注入器
func NewFingerprintInjector(config *FingerprintConfig) *FingerprintInjector {
	return &FingerprintInjector{
		config:             config,
		audioWebGLInjector: NewEnhancedAudioWebGLInjector(config),
		timestampInjector:  NewTimestampFingerprintInjector(config),
		cacheValid:         false,
	}
}

// GenerateInjectionScript 生成完整的JavaScript注入脚本（带缓存）
func (fi *FingerprintInjector) GenerateInjectionScript() string {
	// 快速路径：检查缓存
	fi.scriptCacheMu.RLock()
	if fi.cacheValid && fi.scriptCache != "" {
		cached := fi.scriptCache
		fi.scriptCacheMu.RUnlock()
		return cached
	}
	fi.scriptCacheMu.RUnlock()
	
	// 慢速路径：生成脚本并缓存
	fi.scriptCacheMu.Lock()
	defer fi.scriptCacheMu.Unlock()
	
	// 双重检查（避免重复生成）
	if fi.cacheValid && fi.scriptCache != "" {
		return fi.scriptCache
	}
	
	// 生成脚本
	script := fi.GenerateInjectionScriptEnhanced()
	fi.scriptCache = script
	fi.cacheValid = true
	
	return script
}

// InvalidateCache 清除脚本缓存（当配置改变时调用）
func (fi *FingerprintInjector) InvalidateCache() {
	fi.scriptCacheMu.Lock()
	defer fi.scriptCacheMu.Unlock()
	fi.cacheValid = false
	fi.scriptCache = ""
}

// GenerateInjectionScriptEnhanced 生成增强版注入脚本（内部使用，不带缓存）
func (fi *FingerprintInjector) GenerateInjectionScriptEnhanced() string {
	// 使用预先创建的注入器（避免重复创建）
	var scripts []string
	
	// ===== 第一部分：时间戳修改（必须最先执行！）=====
	scripts = append(scripts, fi.timestampInjector.GenerateTimestampInjectionScript())
	
	// ===== 第二部分：基础属性修改 =====
	// 注入navigator对象修改
	scripts = append(scripts, fi.generateNavigatorScript())
	
	// 注入screen对象修改
	scripts = append(scripts, fi.generateScreenScript())
	
	// ===== 第三部分：增强版 Audio/WebGL 修改 =====
	// 注入增强版 WebGL 修改（替换原版本）
	scripts = append(scripts, fi.audioWebGLInjector.GenerateEnhancedWebGLScript())
	
	// 注入Canvas修改
	scripts = append(scripts, fi.generateCanvasScript())
	
	// 注入增强版 AudioContext 修改（替换原版本）
	scripts = append(scripts, fi.audioWebGLInjector.GenerateEnhancedAudioScript())
	
	// ===== 第四部分：其他指纹修改 =====
	// 注入时区修改（注意：已在时间戳脚本中处理，这里可能重复但确保兼容性）
	scripts = append(scripts, fi.generateTimezoneScript())
	
	// 注入字体修改
	scripts = append(scripts, fi.generateFontsScript())
	
	// 注入插件修改
	scripts = append(scripts, fi.generatePluginsScript())
	
	// 注入电池API修改
	scripts = append(scripts, fi.generateBatteryScript())
	
	// 注入媒体设备修改
	scripts = append(scripts, fi.generateMediaDevicesScript())
	
	// 注入网络信息修改
	scripts = append(scripts, fi.generateNetworkScript())
	
	// 包装所有脚本
	fullScript := fmt.Sprintf(`
(function() {
    'use strict';
    
    console.log('🔒 开始注入完整增强版指纹修改脚本（包括TS1时间戳）...');
    
    // 防止脚本被检测
    const originalDefineProperty = Object.defineProperty;
    const originalGetOwnPropertyDescriptor = Object.getOwnPropertyDescriptor;
    
    %s
    
    // 清理痕迹
    delete window.fingerprintConfig;
    
    console.log('✅ 完整指纹注入完成 - 用户: %s');
    console.log('   🕐 时间戳哈希: %s');
    console.log('   🔊 预期Audio哈希: %s');
    console.log('   🎨 预期WebGL哈希: %s');
})();
`, strings.Join(scripts, "\n\n    "), fi.config.UserID,
		fi.timestampInjector.CalculateExpectedTimestampHash()[:16]+"...",
		fi.audioWebGLInjector.CalculateExpectedAudioHash()[:16]+"...",
		fi.audioWebGLInjector.CalculateExpectedWebGLHash()[:16]+"...")
	
	return fullScript
}

// GenerateInjectionScriptLegacy 生成传统版本的注入脚本（不使用增强版）
func (fi *FingerprintInjector) GenerateInjectionScriptLegacy() string {
	var scripts []string
	
	// 注入navigator对象修改
	scripts = append(scripts, fi.generateNavigatorScript())
	
	// 注入screen对象修改
	scripts = append(scripts, fi.generateScreenScript())
	
	// 注入WebGL修改
	scripts = append(scripts, fi.generateWebGLScript())
	
	// 注入Canvas修改
	scripts = append(scripts, fi.generateCanvasScript())
	
	// 注入AudioContext修改
	scripts = append(scripts, fi.generateAudioScript())
	
	// 注入时区修改
	scripts = append(scripts, fi.generateTimezoneScript())
	
	// 注入字体修改
	scripts = append(scripts, fi.generateFontsScript())
	
	// 注入插件修改
	scripts = append(scripts, fi.generatePluginsScript())
	
	// 注入电池API修改
	scripts = append(scripts, fi.generateBatteryScript())
	
	// 注入媒体设备修改
	scripts = append(scripts, fi.generateMediaDevicesScript())
	
	// 注入网络信息修改
	scripts = append(scripts, fi.generateNetworkScript())
	
	// 包装所有脚本
	fullScript := fmt.Sprintf(`
(function() {
    'use strict';
    
    // 防止脚本被检测
    const originalDefineProperty = Object.defineProperty;
    const originalGetOwnPropertyDescriptor = Object.getOwnPropertyDescriptor;
    
    %s
    
    // 清理痕迹
    delete window.fingerprintConfig;
    
    console.log('🔒 Fingerprint injection completed for user: %s');
})();
`, strings.Join(scripts, "\n\n    "), fi.config.UserID)
	
	return fullScript
}

// generateNavigatorScript 生成navigator对象修改脚本
func (fi *FingerprintInjector) generateNavigatorScript() string {
	webdriverValue := "undefined"
	if fi.config.Browser.WebDriver != nil {
		if *fi.config.Browser.WebDriver {
			webdriverValue = "true"
		} else {
			webdriverValue = "false"
		}
	}
	
	doNotTrackValue := "null"
	if fi.config.Browser.DoNotTrack != nil {
		doNotTrackValue = fmt.Sprintf("'%s'", *fi.config.Browser.DoNotTrack)
	}
	
	languagesArray := "'" + strings.Join(fi.config.Browser.Languages, "', '") + "'"
	
	return fmt.Sprintf(`
    // 修改navigator属性
    originalDefineProperty(navigator, 'userAgent', {
        get: () => '%s',
        configurable: true
    });
    
    originalDefineProperty(navigator, 'language', {
        get: () => '%s',
        configurable: true
    });
    
    originalDefineProperty(navigator, 'languages', {
        get: () => [%s],
        configurable: true
    });
    
    originalDefineProperty(navigator, 'platform', {
        get: () => '%s',
        configurable: true
    });
    
    originalDefineProperty(navigator, 'vendor', {
        get: () => '%s',
        configurable: true
    });
    
    originalDefineProperty(navigator, 'hardwareConcurrency', {
        get: () => %d,
        configurable: true
    });
    
    originalDefineProperty(navigator, 'maxTouchPoints', {
        get: () => %d,
        configurable: true
    });
    
    originalDefineProperty(navigator, 'webdriver', {
        get: () => %s,
        configurable: true
    });
    
    originalDefineProperty(navigator, 'doNotTrack', {
        get: () => %s,
        configurable: true
    });
    
    originalDefineProperty(navigator, 'cookieEnabled', {
        get: () => %t,
        configurable: true
    });`,
		fi.config.Browser.UserAgent,
		fi.config.Browser.Language,
		languagesArray,
		fi.config.Browser.Platform,
		fi.config.Browser.Vendor,
		fi.config.Browser.HardwareConcurrency,
		fi.config.Browser.MaxTouchPoints,
		webdriverValue,
		doNotTrackValue,
		fi.config.Browser.CookieEnabled)
}

// generateScreenScript 生成screen对象修改脚本
func (fi *FingerprintInjector) generateScreenScript() string {
	return fmt.Sprintf(`
    // 修改screen属性
    originalDefineProperty(screen, 'width', {
        get: () => %d,
        configurable: true
    });
    
    originalDefineProperty(screen, 'height', {
        get: () => %d,
        configurable: true
    });
    
    originalDefineProperty(screen, 'availWidth', {
        get: () => %d,
        configurable: true
    });
    
    originalDefineProperty(screen, 'availHeight', {
        get: () => %d,
        configurable: true
    });
    
    originalDefineProperty(screen, 'colorDepth', {
        get: () => %d,
        configurable: true
    });
    
    originalDefineProperty(screen, 'pixelDepth', {
        get: () => %d,
        configurable: true
    });
    
    originalDefineProperty(window, 'devicePixelRatio', {
        get: () => %.2f,
        configurable: true
    });`,
		fi.config.Screen.Width,
		fi.config.Screen.Height,
		fi.config.Screen.AvailWidth,
		fi.config.Screen.AvailHeight,
		fi.config.Screen.ColorDepth,
		fi.config.Screen.PixelDepth,
		fi.config.Screen.DevicePixelRatio)
}

// generateWebGLScript 生成WebGL修改脚本
func (fi *FingerprintInjector) generateWebGLScript() string {
	return fmt.Sprintf(`
    // 修改WebGL属性 - 确保在Canvas创建前就修改
    (function() {
        // 先检查WebGL支持
        if (!window.WebGLRenderingContext) {
            console.log('⚠️ WebGL not supported, skipping WebGL fingerprint modification');
            return;
        }
        
        const originalGetContext = HTMLCanvasElement.prototype.getContext;
        HTMLCanvasElement.prototype.getContext = function(contextType, contextAttributes) {
            const context = originalGetContext.call(this, contextType, contextAttributes);
            
            if (context && (contextType === 'webgl' || contextType === 'experimental-webgl' || contextType === 'webgl2')) {
                // 修改getParameter方法
                const originalGetParameter = context.getParameter;
                context.getParameter = function(parameter) {
                    switch(parameter) {
                        case context.VENDOR:
                            return '%s';
                        case context.RENDERER: 
                            return '%s';
                        case context.VERSION:
                            return '%s';
                        case context.SHADING_LANGUAGE_VERSION:
                            return '%s';
                        case context.MAX_TEXTURE_SIZE:
                            return %d;
                        case context.MAX_RENDERBUFFER_SIZE:
                            return %d;
                        case context.MAX_VIEWPORT_DIMS:
                            return new Int32Array([%d, %d]);
                        case context.MAX_VERTEX_ATTRIBS:
                            return 16;
                        case context.MAX_VERTEX_UNIFORM_VECTORS:
                            return 254;
                        case context.MAX_FRAGMENT_UNIFORM_VECTORS:
                            return 221;
                        case context.MAX_VARYING_VECTORS:
                            return 8;
                        case context.ALIASED_LINE_WIDTH_RANGE:
                            return new Float32Array([1, 1]);
                        case context.ALIASED_POINT_SIZE_RANGE:
                            return new Float32Array([1, 1024]);
                        case context.MAX_CUBE_MAP_TEXTURE_SIZE:
                            return %d;
                        case context.UNMASKED_VENDOR_WEBGL:
                            return '%s';
                        case context.UNMASKED_RENDERER_WEBGL:
                            return '%s';
                        default:
                            return originalGetParameter.call(this, parameter);
                    }
                };

                // 修改getSupportedExtensions
                const originalGetSupportedExtensions = context.getSupportedExtensions;
                context.getSupportedExtensions = function() {
                    return [
                        'ANGLE_instanced_arrays',
                        'EXT_blend_minmax',
                        'EXT_color_buffer_half_float',
                        'EXT_disjoint_timer_query',
                        'EXT_float_blend',
                        'EXT_frag_depth',
                        'EXT_shader_texture_lod',
                        'EXT_texture_compression_bptc',
                        'EXT_texture_compression_rgtc',
                        'EXT_texture_filter_anisotropic',
                        'WEBKIT_EXT_texture_filter_anisotropic',
                        'EXT_sRGB',
                        'KHR_parallel_shader_compile',
                        'OES_element_index_uint',
                        'OES_fbo_render_mipmap',
                        'OES_standard_derivatives',
                        'OES_texture_float',
                        'OES_texture_float_linear',
                        'OES_texture_half_float',
                        'OES_texture_half_float_linear',
                        'OES_vertex_array_object',
                        'WEBGL_color_buffer_float',
                        'WEBGL_compressed_texture_s3tc',
                        'WEBKIT_WEBGL_compressed_texture_s3tc',
                        'WEBGL_compressed_texture_s3tc_srgb',
                        'WEBGL_debug_renderer_info',
                        'WEBGL_debug_shaders',
                        'WEBGL_depth_texture',
                        'WEBKIT_WEBGL_depth_texture',
                        'WEBGL_draw_buffers',
                        'WEBGL_lose_context',
                        'WEBKIT_WEBGL_lose_context'
                    ];
                };

                // 修改getExtension
                const originalGetExtension = context.getExtension;
                context.getExtension = function(name) {
                    if (name === 'WEBGL_debug_renderer_info') {
                        return {
                            UNMASKED_VENDOR_WEBGL: 37445,
                            UNMASKED_RENDERER_WEBGL: 37446
                        };
                    }
                    if (name === 'EXT_texture_filter_anisotropic' || name === 'WEBKIT_EXT_texture_filter_anisotropic') {
                        const ext = originalGetExtension.call(this, name);
                        if (ext) {
                            // 修改各向异性过滤参数
                            originalDefineProperty(ext, 'MAX_TEXTURE_MAX_ANISOTROPY_EXT', {
                                get: () => 16.0,
                                configurable: true
                            });
                        }
                        return ext;
                    }
                    return originalGetExtension.call(this, name);
                };
                
                // 添加shader编译修改以影响指纹
                const originalShaderSource = context.shaderSource;
                context.shaderSource = function(shader, source) {
                    // 为不同用户添加不同的注释（不影响功能但影响指纹）
                    const userComment = '// User fingerprint: %s\n';
                    const modifiedSource = userComment + source;
                    return originalShaderSource.call(this, shader, modifiedSource);
                };
            }
            
            return context;
        };
    })();
    
    // 全局WebGL修改
    const getParameter = WebGLRenderingContext.prototype.getParameter;
    WebGLRenderingContext.prototype.getParameter = function(parameter) {
        switch(parameter) {
            case this.VENDOR:
                return '%s';
            case this.RENDERER:
                return '%s';
            case this.VERSION:
                return '%s';
            case this.SHADING_LANGUAGE_VERSION:
                return '%s';
            case this.MAX_TEXTURE_SIZE:
                return %d;
            case this.MAX_RENDERBUFFER_SIZE:
                return %d;
            case this.MAX_VIEWPORT_DIMS:
                return new Int32Array([%d, %d]);
            case 37445: // UNMASKED_VENDOR_WEBGL
                return '%s';
            case 37446: // UNMASKED_RENDERER_WEBGL  
                return '%s';
            default:
                return getParameter.call(this, parameter);
        }
    };
    
    // 修改WebGL2
    if (window.WebGL2RenderingContext) {
        const getParameter2 = WebGL2RenderingContext.prototype.getParameter;
        WebGL2RenderingContext.prototype.getParameter = WebGLRenderingContext.prototype.getParameter;
    }
    
    console.log('✅ WebGL fingerprint modification applied for user: %s');`,
		fi.config.WebGL.Vendor,
		fi.config.WebGL.Renderer,
		fi.config.WebGL.Version,
		fi.config.WebGL.ShadingLanguageVersion,
		fi.config.WebGL.MaxTextureSize,
		fi.config.WebGL.MaxRenderbufferSize,
		fi.config.WebGL.MaxTextureSize, fi.config.WebGL.MaxTextureSize,
		fi.config.WebGL.MaxTextureSize,
		fi.config.WebGL.Vendor,
		fi.config.WebGL.Renderer,
		fi.config.UserID, // 添加用户ID到shader注释中
		fi.config.WebGL.Vendor,
		fi.config.WebGL.Renderer,
		fi.config.WebGL.Version,
		fi.config.WebGL.ShadingLanguageVersion,
		fi.config.WebGL.MaxTextureSize,
		fi.config.WebGL.MaxRenderbufferSize,
		fi.config.WebGL.MaxTextureSize, fi.config.WebGL.MaxTextureSize,
		fi.config.WebGL.Vendor,
		fi.config.WebGL.Renderer,
		fi.config.UserID)
}

// generateCanvasScript 生成Canvas修改脚本
func (fi *FingerprintInjector) generateCanvasScript() string {
	return fmt.Sprintf(`
    // 修改Canvas指纹
    const toDataURL = HTMLCanvasElement.prototype.toDataURL;
    const getImageData = CanvasRenderingContext2D.prototype.getImageData;
    
    HTMLCanvasElement.prototype.toDataURL = function(type) {
        const originalResult = toDataURL.call(this, type);
        
        // 添加随机噪音
        const canvas = document.createElement('canvas');
        const ctx = canvas.getContext('2d');
        canvas.width = this.width;
        canvas.height = this.height;
        
        const img = new Image();
        img.onload = function() {
            ctx.drawImage(img, 0, 0);
            
            // 添加微小的噪音
            const imageData = ctx.getImageData(0, 0, canvas.width, canvas.height);
            const data = imageData.data;
            const noiseLevel = %.6f;
            
            for (let i = 0; i < data.length; i += 4) {
                if (Math.random() < noiseLevel) {
                    data[i] += Math.floor(Math.random() * %d) - %d;     // R
                    data[i + 1] += Math.floor(Math.random() * %d) - %d; // G  
                    data[i + 2] += Math.floor(Math.random() * %d) - %d; // B
                }
            }
            
            ctx.putImageData(imageData, 0, 0);
        };
        img.src = originalResult;
        
        return originalResult;
    };
    
    CanvasRenderingContext2D.prototype.getImageData = function(sx, sy, sw, sh) {
        const imageData = getImageData.call(this, sx, sy, sw, sh);
        
        // 为getImageData也添加少量噪音
        const data = imageData.data;
        const noiseLevel = %.6f * 0.1; // 更小的噪音
        
        for (let i = 0; i < data.length; i += 4) {
            if (Math.random() < noiseLevel) {
                data[i] += Math.floor(Math.random() * 3) - 1;     // R
                data[i + 1] += Math.floor(Math.random() * 3) - 1; // G
                data[i + 2] += Math.floor(Math.random() * 3) - 1; // B
            }
        }
        
        return imageData;
    };`,
		fi.config.Canvas.NoiseLevel,
		fi.config.Canvas.TextVariance*2, fi.config.Canvas.TextVariance,
		fi.config.Canvas.TextVariance*2, fi.config.Canvas.TextVariance,
		fi.config.Canvas.TextVariance*2, fi.config.Canvas.TextVariance,
		fi.config.Canvas.NoiseLevel)
}

// generateAudioScript 生成AudioContext修改脚本
func (fi *FingerprintInjector) generateAudioScript() string {
	return fmt.Sprintf(`
    // 修改AudioContext属性和指纹 - 增强版
    const AudioContext = window.AudioContext || window.webkitAudioContext;
    if (AudioContext) {
        const originalAudioContext = AudioContext;
        
        // 用户特定的音频噪音种子和模式
        const audioNoiseSeed = %.6f;
        const userAudioSeed = %d;
        const userNoisePattern = %d %% 13; // 用户特定的噪音模式
        const userFreqOffset = (userAudioSeed %% 1000) / 10000.0; // 0-0.1的频率偏移
        
        window.AudioContext = function() {
            const ctx = new originalAudioContext();
            
            // 修改基础属性
            originalDefineProperty(ctx, 'sampleRate', {
                get: () => %d,
                configurable: true
            });
            
            originalDefineProperty(ctx.destination, 'maxChannelCount', {
                get: () => %d,
                configurable: true
            });
            
            originalDefineProperty(ctx.destination, 'numberOfInputs', {
                get: () => %d,
                configurable: true
            });
            
            originalDefineProperty(ctx.destination, 'numberOfOutputs', {
                get: () => %d,
                configurable: true
            });
            
            // 修改createOscillator以生成不同的音频指纹
            const originalCreateOscillator = ctx.createOscillator;
            ctx.createOscillator = function() {
                const oscillator = originalCreateOscillator.call(this);
                const originalStart = oscillator.start;
                const originalConnect = oscillator.connect;
                
                // 修改频率属性
                originalDefineProperty(oscillator.frequency, 'defaultValue', {
                    get: () => 440 + userFreqOffset,
                    configurable: true
                });
                
                oscillator.start = function(when) {
                    // 为每个用户添加不同的频率偏移模式
                    if (oscillator.frequency) {
                        const originalFreq = oscillator.frequency.value;
                        let offset = 0;
                        
                        switch(userNoisePattern) {
                            case 0: offset = userFreqOffset * 10; break;
                            case 1: offset = userFreqOffset * -10; break;
                            case 2: offset = userFreqOffset * 5; break;
                            default: offset = userFreqOffset * (userNoisePattern - 6);
                        }
                        
                        oscillator.frequency.value = originalFreq + offset;
                    }
                    
                    // 修改oscillator类型以增加差异
                    const types = ['sine', 'square', 'sawtooth', 'triangle'];
                    oscillator.type = types[userAudioSeed %% types.length];
                    
                    return originalStart.call(this, when);
                };
                
                // 修改connect方法以添加增益节点
                oscillator.connect = function(destination, output, input) {
                    if (Math.random() < 0.1) { // 10%%%%的概率添加增益调整
                        const gainNode = ctx.createGain();
                        gainNode.gain.value = 0.98 + (userAudioSeed %% 100) / 5000; // 微小增益调整
                        originalConnect.call(this, gainNode);
                        return gainNode.connect(destination, output, input);
                    }
                    return originalConnect.call(this, destination, output, input);
                };
                
                return oscillator;
            };
            
            // 修改createAnalyser以影响频域数据
            const originalCreateAnalyser = ctx.createAnalyser;
            ctx.createAnalyser = function() {
                const analyser = originalCreateAnalyser.call(this);
                const originalGetFloatFrequencyData = analyser.getFloatFrequencyData;
                const originalGetByteFrequencyData = analyser.getByteFrequencyData;
                
                // 修改频域数据以产生不同的音频指纹 - 增强版
                analyser.getFloatFrequencyData = function(array) {
                    originalGetFloatFrequencyData.call(this, array);
                    
                    // 基于用户模式的多种噪音注入策略
                    const noiseIntensity = audioNoiseSeed * (0.05 + userNoisePattern * 0.01);
                    
                    for (let i = 0; i < array.length; i++) {
                        // 策略1: 基于位置的噪音
                        if (i %% (7 + userNoisePattern) === userAudioSeed %% (7 + userNoisePattern)) {
                            array[i] += noiseIntensity * (1 + Math.sin(i * userFreqOffset));
                        }
                        
                        // 策略2: 基于频率段的调整
                        if (i > array.length * 0.1 && i < array.length * 0.9) { // 中频段
                            const adjustment = (userAudioSeed %% 100) / 10000.0;
                            array[i] *= (1.0 + adjustment);
                        }
                        
                        // 策略3: 周期性噪音注入
                        if (i %% (userNoisePattern + 5) === 0) {
                            array[i] += Math.sin(i * userFreqOffset * Math.PI) * noiseIntensity * 0.5;
                        }
                    }
                };
                
                analyser.getByteFrequencyData = function(array) {
                    originalGetByteFrequencyData.call(this, array);
                    
                    // 字节数据的噪音注入策略
                    for (let i = 0; i < array.length; i++) {
                        // 策略1: 交替噪音模式
                        if (i %% (11 + userNoisePattern) === userAudioSeed %% (11 + userNoisePattern)) {
                            const noise = Math.floor(audioNoiseSeed * (5 + userNoisePattern));
                            array[i] = Math.min(255, Math.max(0, array[i] + noise));
                        }
                        
                        // 策略2: 基于奇偶性的微调
                        if ((i %% 2) === (userAudioSeed %% 2)) {
                            const microNoise = Math.floor((userAudioSeed %% 10) / 10.0);
                            array[i] = Math.min(255, Math.max(0, array[i] + microNoise));
                        }
                        
                        // 策略3: 渐变噪音
                        if (i %% 50 < userNoisePattern) {
                            const gradientNoise = Math.floor(Math.sin(i / 50.0 * Math.PI) * 3);
                            array[i] = Math.min(255, Math.max(0, array[i] + gradientNoise));
                        }
                    }
                };
                
                return analyser;
            };
            
            // 修改createScriptProcessor以影响时域数据 - 增强版
            const originalCreateScriptProcessor = ctx.createScriptProcessor;
            ctx.createScriptProcessor = function(bufferSize, numberOfInputChannels, numberOfOutputChannels) {
                const scriptProcessor = originalCreateScriptProcessor.call(this, bufferSize, numberOfInputChannels, numberOfOutputChannels);
                
                const originalAddEventListener = scriptProcessor.addEventListener;
                scriptProcessor.addEventListener = function(type, listener, useCapture) {
                    if (type === 'audioprocess') {
                        const wrappedListener = function(event) {
                            // 多层次音频处理修改
                            const inputBuffer = event.inputBuffer;
                            const outputBuffer = event.outputBuffer;
                            
                            if (inputBuffer && outputBuffer) {
                                for (let channel = 0; channel < Math.min(inputBuffer.numberOfChannels, outputBuffer.numberOfChannels); channel++) {
                                    const inputData = inputBuffer.getChannelData(channel);
                                    const outputData = outputBuffer.getChannelData(channel);
                                    
                                    for (let i = 0; i < inputData.length; i++) {
                                        let sample = inputData[i];
                                        
                                        // 策略1: 周期性微调
                                        if (i %% (100 + userNoisePattern * 10) === userAudioSeed %% 1000) {
                                            sample += audioNoiseSeed * 0.0001 * Math.sin(i * userFreqOffset);
                                        }
                                        
                                        // 策略2: 基于样本值的调整
                                        if (Math.abs(sample) > 0.1 && i %% userNoisePattern === 0) {
                                            sample *= (1.0 + (userAudioSeed %% 1000) / 1000000.0);
                                        }
                                        
                                        // 策略3: 时间戳相关的微调
                                        if (ctx.currentTime && i %% 1000 === Math.floor(ctx.currentTime * 1000) %% 1000) {
                                            sample += (userAudioSeed %% 1000 - 500) / 10000000.0;
                                        }
                                        
                                        outputData[i] = sample;
                                    }
                                }
                            }
                            return listener.call(this, event);
                        };
                        return originalAddEventListener.call(this, type, wrappedListener, useCapture);
                    }
                    return originalAddEventListener.call(this, type, listener, useCapture);
                };
                
                return scriptProcessor;
            };
            
            // 修改createBuffer以影响缓冲区创建
            const originalCreateBuffer = ctx.createBuffer;
            ctx.createBuffer = function(numberOfChannels, length, sampleRate) {
                const buffer = originalCreateBuffer.call(this, numberOfChannels, length, sampleRate);
                
                // 为每个通道添加用户特定的微小噪音
                for (let channel = 0; channel < numberOfChannels; channel++) {
                    const channelData = buffer.getChannelData(channel);
                    for (let i = 0; i < Math.min(100, channelData.length); i += 10) { // 只修改前100个样本
                        const index = i + (userAudioSeed %% 10);
                        if (index < channelData.length) {
                            channelData[index] = (userAudioSeed %% 1000 - 500) / 100000.0;
                        }
                    }
                }
                
                return buffer;
            };
            
            // 修改createGain以添加微小的增益差异
            const originalCreateGain = ctx.createGain;
            ctx.createGain = function() {
                const gainNode = originalCreateGain.call(this);
                const baseGain = gainNode.gain.value;
                const gainOffset = (userAudioSeed %% 1000) / 100000.0; // 微小偏移
                
                originalDefineProperty(gainNode.gain, 'defaultValue', {
                    get: () => baseGain + gainOffset,
                    configurable: true
                });
                
                return gainNode;
            };
            
            return ctx;
        };
        
        if (window.webkitAudioContext) {
            window.webkitAudioContext = window.AudioContext;
        }
        
        // 修改OfflineAudioContext
        if (window.OfflineAudioContext) {
            const originalOfflineAudioContext = window.OfflineAudioContext;
            window.OfflineAudioContext = function(numberOfChannels, length, sampleRate) {
                // 使用修改后的采样率
                const modifiedSampleRate = %d + (userAudioSeed %% 100); // 微小的采样率差异
                const ctx = new originalOfflineAudioContext(numberOfChannels, length, modifiedSampleRate);
                
                // 应用相同的修改逻辑到offline context
                const originalStartRendering = ctx.startRendering;
                ctx.startRendering = function() {
                    // 在渲染前添加最终的音频指纹修改
                    const renderingPromise = originalStartRendering.call(this);
                    
                    return renderingPromise.then(buffer => {
                        // 对渲染结果添加微小修改
                        for (let channel = 0; channel < buffer.numberOfChannels; channel++) {
                            const channelData = buffer.getChannelData(channel);
                            const step = Math.floor(channelData.length / 10);
                            
                            for (let i = 0; i < channelData.length; i += step) {
                                if (i %% userNoisePattern === 0) {
                                    const noiseIndex = i + (userAudioSeed %% step);
                                    if (noiseIndex < channelData.length) {
                                        channelData[noiseIndex] += audioNoiseSeed * 0.00001;
                                    }
                                }
                            }
                        }
                        return buffer;
                    });
                };
                
                return ctx;
            };
        }
        
        console.log('✅ Enhanced Audio fingerprint modification applied for user:', '%s', {
            noiseSeed: audioNoiseSeed,
            userSeed: userAudioSeed,
            noisePattern: userNoisePattern,
            freqOffset: userFreqOffset
        });
    }`,
		fi.config.Canvas.NoiseLevel, // 重用Canvas噪音级别作为音频噪音
		fi.hashUserID(fi.config.UserID), // 生成用户特定的种子
		fi.hashUserID(fi.config.UserID+"pattern"), // 噪音模式种子
		fi.config.Audio.SampleRate,
		fi.config.Audio.MaxChannelCount,
		fi.config.Audio.NumberOfInputs,
		fi.config.Audio.NumberOfOutputs,
		fi.config.Audio.SampleRate,
		fi.config.UserID)
}

// generateTimezoneScript 生成时区修改脚本
func (fi *FingerprintInjector) generateTimezoneScript() string {
	return fmt.Sprintf(`
    // 修改时区相关
    const originalGetTimezoneOffset = Date.prototype.getTimezoneOffset;
    Date.prototype.getTimezoneOffset = function() {
        return %d;
    };
    
    const originalResolvedOptions = Intl.DateTimeFormat.prototype.resolvedOptions;
    Intl.DateTimeFormat.prototype.resolvedOptions = function() {
        const options = originalResolvedOptions.call(this);
        options.timeZone = '%s';
        return options;
    };`,
		fi.config.Timezone.Offset,
		fi.config.Timezone.Timezone)
}

// generateFontsScript 生成字体修改脚本
func (fi *FingerprintInjector) generateFontsScript() string {
	availableFonts := "['" + strings.Join(fi.config.Fonts.AvailableFonts, "', '") + "']"
	
	return fmt.Sprintf(`
    // 修改字体检测
    const originalMeasureText = CanvasRenderingContext2D.prototype.measureText;
    const availableFonts = %s;
    
    CanvasRenderingContext2D.prototype.measureText = function(text) {
        const result = originalMeasureText.call(this, text);
        
        // 检查当前字体设置
        const fontFamily = this.font.split(' ').pop().replace(/['"]/g, '');
        
        // 如果字体不在可用列表中，返回默认宽度
        if (!availableFonts.includes(fontFamily)) {
            // 模拟字体不可用的情况
            return {
                width: result.width * 0.95, // 略微不同的宽度
                actualBoundingBoxLeft: result.actualBoundingBoxLeft,
                actualBoundingBoxRight: result.actualBoundingBoxRight,
                fontBoundingBoxAscent: result.fontBoundingBoxAscent,
                fontBoundingBoxDescent: result.fontBoundingBoxDescent,
                actualBoundingBoxAscent: result.actualBoundingBoxAscent,
                actualBoundingBoxDescent: result.actualBoundingBoxDescent,
                emHeightAscent: result.emHeightAscent,
                emHeightDescent: result.emHeightDescent,
                hangingBaseline: result.hangingBaseline,
                alphabeticBaseline: result.alphabeticBaseline,
                ideographicBaseline: result.ideographicBaseline
            };
        }
        
        return result;
    };`,
		availableFonts)
}

// generatePluginsScript 生成插件修改脚本
func (fi *FingerprintInjector) generatePluginsScript() string {
	return `
    // 修改插件信息（保持默认的PDF插件）
    const plugins = [
        {
            name: 'PDF Viewer',
            filename: 'internal-pdf-viewer',
            description: 'Portable Document Format',
            length: 2,
            0: { type: 'application/pdf', suffixes: 'pdf', description: 'Portable Document Format' },
            1: { type: 'text/pdf', suffixes: 'pdf', description: 'Portable Document Format' }
        },
        {
            name: 'Chrome PDF Viewer',
            filename: 'internal-pdf-viewer', 
            description: 'Portable Document Format',
            length: 2,
            0: { type: 'application/pdf', suffixes: 'pdf', description: 'Portable Document Format' },
            1: { type: 'text/pdf', suffixes: 'pdf', description: 'Portable Document Format' }
        }
    ];
    
    originalDefineProperty(navigator, 'plugins', {
        get: () => plugins,
        configurable: true
    });`
}

// generateBatteryScript 生成电池API修改脚本
func (fi *FingerprintInjector) generateBatteryScript() string {
	chargingTimeValue := "Infinity"
	if fi.config.Battery.ChargingTime != nil {
		chargingTimeValue = fmt.Sprintf("%.2f", *fi.config.Battery.ChargingTime)
	}
	
	dischargingTimeValue := "Infinity"
	if fi.config.Battery.DischargingTime != nil {
		dischargingTimeValue = fmt.Sprintf("%.2f", *fi.config.Battery.DischargingTime)
	}
	
	return fmt.Sprintf(`
    // 修改电池API
    if ('getBattery' in navigator) {
        const originalGetBattery = navigator.getBattery;
        navigator.getBattery = function() {
            return Promise.resolve({
                charging: %t,
                chargingTime: %s,
                dischargingTime: %s,
                level: %.2f,
                addEventListener: function() {},
                removeEventListener: function() {},
                dispatchEvent: function() { return true; }
            });
        };
    }`,
		fi.config.Battery.Charging,
		chargingTimeValue,
		dischargingTimeValue,
		fi.config.Battery.Level)
}

// generateMediaDevicesScript 生成媒体设备修改脚本
func (fi *FingerprintInjector) generateMediaDevicesScript() string {
	var devices []string
	for _, device := range fi.config.MediaDevices {
		deviceStr := fmt.Sprintf(`{kind: '%s', label: '%s', deviceId: '%s'}`, 
			device.Kind, device.Label, device.DeviceID)
		devices = append(devices, deviceStr)
	}
	
	devicesArray := "[" + strings.Join(devices, ", ") + "]"
	
	return fmt.Sprintf(`
    // 修改媒体设备
    if (navigator.mediaDevices) {
        const originalEnumerateDevices = navigator.mediaDevices.enumerateDevices;
        navigator.mediaDevices.enumerateDevices = function() {
            return Promise.resolve(%s);
        };
    }`,
		devicesArray)
}

// generateNetworkScript 生成网络信息修改脚本
func (fi *FingerprintInjector) generateNetworkScript() string {
	return fmt.Sprintf(`
    // 修改网络连接信息
    if ('connection' in navigator) {
        const connectionInfo = {
            effectiveType: '%s',
            downlink: %.2f,
            rtt: %d,
            saveData: %t,
            addEventListener: function() {},
            removeEventListener: function() {},
            dispatchEvent: function() { return true; }
        };
        
        originalDefineProperty(navigator, 'connection', {
            get: () => connectionInfo,
            configurable: true
        });
    }`,
		fi.config.Network.EffectiveType,
		fi.config.Network.Downlink,
		fi.config.Network.RTT,
		fi.config.Network.SaveData)
}

// GetPreloadScript 获取预加载脚本（在页面加载前注入）
func (fi *FingerprintInjector) GetPreloadScript() string {
	return `
// 预加载脚本 - 在任何其他脚本执行前运行
(function() {
    'use strict';
    
    // 隐藏webdriver属性
    Object.defineProperty(navigator, 'webdriver', {
        get: () => undefined,
        configurable: true
    });
    
    // 移除automation相关属性
    delete window.chrome.runtime.onConnect;
    
    // 禁用自动化检测相关的事件
    window.addEventListener = new Proxy(window.addEventListener, {
        apply: function(target, thisArg, args) {
            if (args[0] === 'chrome-extension-onconnect') {
                return;
            }
            return target.apply(thisArg, args);
        }
    });
    
    console.log('🛡️  Anti-detection preload script executed');
})();`
}

// GenerateExtensionManifest 生成指纹修改扩展的manifest.json
func (fi *FingerprintInjector) GenerateExtensionManifest() string {
	return `{
    "manifest_version": 3,
    "name": "Fingerprint Modifier",
    "version": "1.0.0",
    "description": "Modify browser fingerprint for privacy protection",
    "permissions": ["activeTab", "scripting"],
    "host_permissions": ["<all_urls>"],
    "content_scripts": [
        {
            "matches": ["<all_urls>"],
            "js": ["content.js"],
            "run_at": "document_start",
            "all_frames": true
        }
    ],
    "web_accessible_resources": [
        {
            "resources": ["injected.js"],
            "matches": ["<all_urls>"]
        }
    ]
}`
}

// GenerateContentScript 生成content script
func (fi *FingerprintInjector) GenerateContentScript() string {
	return fmt.Sprintf(`
// Content Script
(function() {
    'use strict';
    
    // 注入主脚本
    const script = document.createElement('script');
    script.textContent = %s;
    (document.head || document.documentElement).appendChild(script);
    script.remove();
    
    console.log('🔧 Fingerprint modification content script loaded');
})();`,
		"`"+fi.GenerateInjectionScript()+"`")
}

// hashUserID 生成用户ID的数值哈希
func (fi *FingerprintInjector) hashUserID(userID string) int {
	hasher := md5.New()
	hasher.Write([]byte(userID))
	hashBytes := hasher.Sum(nil)
	
	// 将前4字节转换为int
	hash := int(0)
	for i := 0; i < 4 && i < len(hashBytes); i++ {
		hash = (hash << 8) | int(hashBytes[i])
	}
	
	if hash < 0 {
		hash = -hash
	}
	
	return hash
}