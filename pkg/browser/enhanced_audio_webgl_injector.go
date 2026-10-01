package browser

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
)

// EnhancedAudioWebGLInjector 增强版 Audio/WebGL 指纹注入器
// 专门解决指纹哈希相同的问题
type EnhancedAudioWebGLInjector struct {
	config *FingerprintConfig
	userHash string // 用户特定的哈希值
	noiseSeed1 int
	noiseSeed2 int
	noisePattern int
}

// NewEnhancedAudioWebGLInjector 创建增强版注入器
//
// Deprecated: Connect 不再使用；身份改由 CDP 下发（见 identity.go），保留仅为兼容。
func NewEnhancedAudioWebGLInjector(config *FingerprintConfig) *EnhancedAudioWebGLInjector {
	// 生成用户特定的哈希和种子
	userHash := generateUserHash(config.UserID)
	
	return &EnhancedAudioWebGLInjector{
		config: config,
		userHash: userHash,
		noiseSeed1: hashToInt(userHash, 0),
		noiseSeed2: hashToInt(userHash, 4),
		noisePattern: hashToInt(userHash, 8) % 17, // 0-16的模式
	}
}

// GenerateEnhancedAudioScript 生成超级增强版 Audio 指纹脚本
func (e *EnhancedAudioWebGLInjector) GenerateEnhancedAudioScript() string {
	return fmt.Sprintf(`
// ========================================
// 超级增强版 Audio 指纹修改脚本
// 用户ID: %s
// 哈希: %s
// ========================================
(function() {
    'use strict';
    
    const userHash = '%s';
    const noiseSeed1 = %d;
    const noiseSeed2 = %d;
    const noisePattern = %d;
    const audioNoiseLevel = %.8f;
    const sampleRate = %d;
    const maxChannelCount = %d;
    
    // 用户特定的噪音生成函数
    function generateUserNoise(index, type) {
        const seed = (noiseSeed1 * index + noiseSeed2) %% 1000000;
        const pattern = (seed + noisePattern * index) %% 1000000;
        
        // 多种噪音生成策略
        let noise = 0;
        switch(type %% 5) {
            case 0: // 正弦波噪音
                noise = Math.sin(pattern / 1000.0 * Math.PI) * audioNoiseLevel;
                break;
            case 1: // 余弦波噪音
                noise = Math.cos(pattern / 800.0 * Math.PI) * audioNoiseLevel * 1.2;
                break;
            case 2: // 锯齿波噪音
                noise = ((pattern %% 1000) / 1000.0 - 0.5) * audioNoiseLevel * 0.8;
                break;
            case 3: // 方波噪音
                noise = (pattern %% 2 === 0 ? 1 : -1) * audioNoiseLevel * 0.5;
                break;
            case 4: // 随机噪音
                noise = (Math.random() - 0.5) * audioNoiseLevel * 1.5;
                break;
        }
        
        return noise;
    }
    
    // 修改 AudioContext
    const OriginalAudioContext = window.AudioContext || window.webkitAudioContext;
    if (!OriginalAudioContext) {
        console.log('⚠️  AudioContext not supported');
        return;
    }
    
    function ModifiedAudioContext(options) {
        // 保留页面传入的参数；未指定采样率时按指纹配置真实创建，
        // 不再只伪造 sampleRate 读数（否则按它计算缓冲区的页面音频会变调）
        const opts = Object.assign({}, options);
        if (opts.sampleRate === undefined) opts.sampleRate = sampleRate;
        const ctx = new OriginalAudioContext(opts);

        // 修改目标通道数
        Object.defineProperty(ctx.destination, 'maxChannelCount', {
            get: () => maxChannelCount,
            configurable: true
        });
        
        // ====== 关键修改1: createOscillator ======
        const originalCreateOscillator = ctx.createOscillator.bind(ctx);
        ctx.createOscillator = function() {
            const osc = originalCreateOscillator();
            const originalFrequency = osc.frequency.value;
            
            // 用户特定的频率偏移
            const freqOffset = (noiseSeed1 %% 1000) / 10000.0; // 0-0.1 Hz
            Object.defineProperty(osc.frequency, 'defaultValue', {
                get: () => 440 + freqOffset,
                configurable: true
            });
            
            // 修改start方法
            const originalStart = osc.start.bind(osc);
            osc.start = function(when) {
                // 应用用户特定的频率偏移（不改波形：强改 type 会让页面发出的声音完全变样）
                osc.frequency.value = osc.frequency.value + freqOffset + generateUserNoise(1, 0);
                return originalStart(when);
            };
            
            return osc;
        };
        
        // ====== 关键修改2: createAnalyser（最重要）======
        const originalCreateAnalyser = ctx.createAnalyser.bind(ctx);
        ctx.createAnalyser = function() {
            const analyser = originalCreateAnalyser();
            
            // 修改 FFT 大小
            const fftSizes = [256, 512, 1024, 2048, 4096, 8192, 16384];
            analyser.fftSize = fftSizes[noiseSeed1 %% fftSizes.length];
            
            // ===== 超级增强版频域数据修改 =====
            const originalGetFloatFrequencyData = analyser.getFloatFrequencyData.bind(analyser);
            analyser.getFloatFrequencyData = function(array) {
                originalGetFloatFrequencyData(array);
                
                // 策略1: 基于位置的复杂噪音注入
                for (let i = 0; i < array.length; i++) {
                    const positionSeed = (i * noiseSeed1) %% 1000000;
                    const patternIndex = (i + noisePattern) %% 13;
                    
                    // 多层噪音叠加
                    if (i %% (7 + patternIndex) === (noiseSeed2 %% 7)) {
                        array[i] += generateUserNoise(i, 0);
                    }
                    
                    // 频率段特定噪音
                    const freqBand = Math.floor(i / array.length * 10); // 10个频段
                    if (freqBand === (noiseSeed1 %% 10)) {
                        array[i] *= (1.0 + generateUserNoise(i, 1) * 0.1);
                    }
                    
                    // 周期性波动
                    if (i %% (noisePattern + 3) === 0) {
                        const wave = Math.sin(i * (noiseSeed2 %% 100) / 100.0 * Math.PI);
                        array[i] += wave * audioNoiseLevel * 2.0;
                    }
                    
                    // 基于哈希的确定性噪音
                    const hashNoise = ((positionSeed * 31 + noiseSeed2) %% 1000) / 10000.0 - 0.05;
                    array[i] += hashNoise;
                }
                
                // 策略2: 全局频谱形状调整
                const globalShift = (noiseSeed1 %% 1000) / 100000.0;
                for (let i = 0; i < array.length; i++) {
                    array[i] += globalShift * Math.pow(-1, i);
                }
                
                // 策略3: 特定频率点的尖峰/凹陷
                const peakPoints = [
                    Math.floor(array.length * 0.1),
                    Math.floor(array.length * 0.3),
                    Math.floor(array.length * 0.6),
                    Math.floor(array.length * 0.8)
                ];
                peakPoints.forEach((point, idx) => {
                    if (point < array.length) {
                        const peakNoise = generateUserNoise(point, idx);
                        array[point] += peakNoise * 5.0;
                        // 影响周围的点
                        if (point > 0) array[point - 1] += peakNoise * 2.5;
                        if (point < array.length - 1) array[point + 1] += peakNoise * 2.5;
                    }
                });
            };
            
            // ===== 字节频域数据修改 =====
            const originalGetByteFrequencyData = analyser.getByteFrequencyData.bind(analyser);
            analyser.getByteFrequencyData = function(array) {
                originalGetByteFrequencyData(array);
                
                for (let i = 0; i < array.length; i++) {
                    // 策略1: 奇偶位置不同处理
                    if ((i %% 2) === (noiseSeed1 %% 2)) {
                        const noise = Math.floor(generateUserNoise(i, 2) * 10);
                        array[i] = Math.min(255, Math.max(0, array[i] + noise));
                    }
                    
                    // 策略2: 周期性调整
                    if (i %% (11 + noisePattern) === (noiseSeed2 %% 11)) {
                        const periodicNoise = Math.floor(Math.sin(i / 50.0 * Math.PI) * 5);
                        array[i] = Math.min(255, Math.max(0, array[i] + periodicNoise));
                    }
                    
                    // 策略3: 渐变式噪音
                    const gradientFactor = i / array.length;
                    const gradientNoise = Math.floor(gradientFactor * (noiseSeed1 %% 10));
                    array[i] = Math.min(255, Math.max(0, array[i] + gradientNoise));
                }
            };
            
            // 修改时域数据
            const originalGetFloatTimeDomainData = analyser.getFloatTimeDomainData.bind(analyser);
            analyser.getFloatTimeDomainData = function(array) {
                originalGetFloatTimeDomainData(array);
                
                for (let i = 0; i < array.length; i++) {
                    if (i %% (17 + noisePattern) === 0) {
                        array[i] += generateUserNoise(i, 3) * 0.001;
                    }
                }
            };
            
            return analyser;
        };
        
        // ====== 关键修改3: createDynamicsCompressor ======
        const originalCreateDynamicsCompressor = ctx.createDynamicsCompressor.bind(ctx);
        ctx.createDynamicsCompressor = function() {
            const compressor = originalCreateDynamicsCompressor();
            
            // 修改压缩器参数（影响音频处理）
            compressor.threshold.value = -50 + (noiseSeed1 %% 10);
            compressor.knee.value = 40 + (noiseSeed2 %% 10);
            compressor.ratio.value = 12 + (noisePattern %% 8);
            compressor.attack.value = 0.003 * (1 + (noiseSeed1 %% 100) / 1000.0);
            compressor.release.value = 0.25 * (1 + (noiseSeed2 %% 100) / 1000.0);
            
            return compressor;
        };
        
        // ====== 关键修改4: createBiquadFilter ======
        const originalCreateBiquadFilter = ctx.createBiquadFilter.bind(ctx);
        ctx.createBiquadFilter = function() {
            const filter = originalCreateBiquadFilter();
            
            // 修改滤波器参数
            filter.frequency.value = filter.frequency.value * (1 + generateUserNoise(0, 4) * 0.1);
            filter.Q.value = filter.Q.value * (1 + (noiseSeed1 %% 100) / 1000.0);
            
            return filter;
        };
        
        // ====== 关键修改5: createGain ======
        const originalCreateGain = ctx.createGain.bind(ctx);
        ctx.createGain = function() {
            const gain = originalCreateGain();
            
            // 用户特定的增益偏移
            const gainOffset = (noiseSeed2 %% 1000) / 100000.0;
            const originalGainValue = gain.gain.value;
            
            Object.defineProperty(gain.gain, 'defaultValue', {
                get: () => originalGainValue + gainOffset,
                configurable: true
            });
            
            return gain;
        };
        
        // ====== 关键修改6: createConvolver（混响效果）======
        const originalCreateConvolver = ctx.createConvolver.bind(ctx);
        ctx.createConvolver = function() {
            const convolver = originalCreateConvolver();
            
            // 如果设置了缓冲区，添加用户特定的脉冲响应修改
            const originalBufferSetter = Object.getOwnPropertyDescriptor(
                Object.getPrototypeOf(convolver), 'buffer'
            );
            
            if (originalBufferSetter && originalBufferSetter.set) {
                Object.defineProperty(convolver, 'buffer', {
                    set: function(buffer) {
                        if (buffer) {
                            // 修改脉冲响应
                            for (let channel = 0; channel < buffer.numberOfChannels; channel++) {
                                const data = buffer.getChannelData(channel);
                                for (let i = 0; i < Math.min(100, data.length); i += 10) {
                                    const idx = i + (noiseSeed1 %% 10);
                                    if (idx < data.length) {
                                        data[idx] += generateUserNoise(idx, channel) * 0.001;
                                    }
                                }
                            }
                        }
                        originalBufferSetter.set.call(this, buffer);
                    },
                    get: originalBufferSetter.get,
                    configurable: true
                });
            }
            
            return convolver;
        };
        
        return ctx;
    }
    
    // 替换全局 AudioContext；共用原型保证 instanceof 成立
    ModifiedAudioContext.prototype = OriginalAudioContext.prototype;
    Object.defineProperty(ModifiedAudioContext, 'name', { value: 'AudioContext' });
    window.AudioContext = ModifiedAudioContext;
    if (window.webkitAudioContext) {
        window.webkitAudioContext = ModifiedAudioContext;
    }
    
    // ====== 修改 OfflineAudioContext（关键！）======
    if (window.OfflineAudioContext) {
        const OriginalOfflineAudioContext = window.OfflineAudioContext;
        
        function ModifiedOfflineAudioContext(...args) {
            // 按页面请求的参数创建，只在渲染结果上加噪声：
            // 改采样率会改变渲染结果；借用实时 AudioContext 的 create* 方法会让节点属于另一个 context，连线直接报错
            const ctx = new OriginalOfflineAudioContext(...args);

            // 修改startRendering
            const originalStartRendering = ctx.startRendering.bind(ctx);
            ctx.startRendering = function() {
                return originalStartRendering().then(buffer => {
                    // 最终的音频缓冲区修改
                    for (let channel = 0; channel < buffer.numberOfChannels; channel++) {
                        const data = buffer.getChannelData(channel);
                        const step = Math.max(1, Math.floor(data.length / 1000));
                        
                        for (let i = 0; i < data.length; i += step) {
                            const noiseType = (i / step) %% 5;
                            const noise = generateUserNoise(i, noiseType);
                            
                            // 多点注入
                            if (i < data.length) data[i] += noise * 0.00001;
                            if (i + 1 < data.length) data[i + 1] += noise * 0.000005;
                            if (i + 2 < data.length) data[i + 2] += noise * 0.000002;
                        }
                        
                        // 全局偏移（非常微小但影响哈希）
                        const globalOffset = (noiseSeed2 %% 10000) / 10000000000.0;
                        for (let i = 0; i < data.length; i++) {
                            data[i] += globalOffset * Math.pow(-1, i);
                        }
                    }
                    
                    return buffer;
                });
            };

            return ctx;
        }
        ModifiedOfflineAudioContext.prototype = OriginalOfflineAudioContext.prototype;
        Object.defineProperty(ModifiedOfflineAudioContext, 'name', { value: 'OfflineAudioContext' });
        window.OfflineAudioContext = ModifiedOfflineAudioContext;
    }
    
    console.log('✅ 超级增强版 Audio 指纹修改已应用', {
        userHash: userHash.substr(0, 8) + '...',
        noiseSeed1: noiseSeed1,
        noiseSeed2: noiseSeed2,
        noisePattern: noisePattern,
        sampleRate: sampleRate
    });
})();
`,
		e.config.UserID,
		e.userHash[:16]+"...",
		e.userHash,
		e.noiseSeed1,
		e.noiseSeed2,
		e.noisePattern,
		e.config.Canvas.NoiseLevel + 0.0001 * float64(e.noiseSeed1 % 100),
		e.config.Audio.SampleRate,
		e.config.Audio.MaxChannelCount)
}

// GenerateEnhancedWebGLScript 生成超级增强版 WebGL 指纹脚本
func (e *EnhancedAudioWebGLInjector) GenerateEnhancedWebGLScript() string {
	// 生成用户特定的WebGL参数
	vendorVariant := e.generateWebGLVendorVariant()
	rendererVariant := e.generateWebGLRendererVariant()
	
	return fmt.Sprintf(`
// ========================================
// 超级增强版 WebGL 指纹修改脚本
// 用户ID: %s
// 哈希: %s
// ========================================
(function() {
    'use strict';
    
    const userHash = '%s';
    const noiseSeed1 = %d;
    const noiseSeed2 = %d;
    const noisePattern = %d;
    
    // 用户特定的WebGL渲染噪音
    function generateWebGLNoise(x, y, type) {
        const seed = (x * noiseSeed1 + y * noiseSeed2) %% 1000000;
        const pattern = (seed + noisePattern) %% 1000;
        
        switch(type %% 4) {
            case 0: return (seed %% 256) / 256.0 - 0.5;
            case 1: return Math.sin(seed / 1000.0 * Math.PI) * 0.5;
            case 2: return ((seed %% 100) - 50) / 100.0;
            case 3: return (Math.random() - 0.5) * (pattern / 1000.0);
        }
        return 0;
    }
    
    // 保存原始方法
    const originalGetContext = HTMLCanvasElement.prototype.getContext;
    const originalToDataURL = HTMLCanvasElement.prototype.toDataURL;
    
    // ====== 修改 getContext ======
    HTMLCanvasElement.prototype.getContext = function(contextType, contextAttributes) {
        const context = originalGetContext.call(this, contextType, contextAttributes);
        
        if (!context || !(contextType === 'webgl' || contextType === 'experimental-webgl' || contextType === 'webgl2')) {
            return context;
        }
        
        // ===== 关键修改1: getParameter =====
        const originalGetParameter = context.getParameter.bind(context);
        context.getParameter = function(parameter) {
            const result = originalGetParameter(parameter);
            
            // 修改基础参数
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
                    return %d + (noiseSeed1 %% 4096);
                case this.MAX_RENDERBUFFER_SIZE:
                    return %d + (noiseSeed2 %% 4096);
                case this.MAX_VIEWPORT_DIMS:
                    const baseSize = %d;
                    return new Int32Array([
                        baseSize + (noiseSeed1 %% 1024),
                        baseSize + (noiseSeed2 %% 1024)
                    ]);
                case this.MAX_VERTEX_ATTRIBS:
                    return 16 + (noisePattern %% 8);
                case this.MAX_VERTEX_UNIFORM_VECTORS:
                    return 254 + (noiseSeed1 %% 256);
                case this.MAX_FRAGMENT_UNIFORM_VECTORS:
                    return 221 + (noiseSeed2 %% 256);
                case this.MAX_VARYING_VECTORS:
                    return 8 + (noisePattern %% 4);
                case this.MAX_COMBINED_TEXTURE_IMAGE_UNITS:
                    return 32 + (noiseSeed1 %% 32);
                case this.MAX_CUBE_MAP_TEXTURE_SIZE:
                    return %d + (noisePattern %% 4096);
                case this.MAX_TEXTURE_IMAGE_UNITS:
                    return 16 + (noiseSeed2 %% 16);
                case this.MAX_VERTEX_TEXTURE_IMAGE_UNITS:
                    return 16 + (noiseSeed1 %% 16);
                case this.ALIASED_LINE_WIDTH_RANGE:
                    return new Float32Array([
                        1.0 + (noiseSeed1 %% 10) / 100.0,
                        7.375 + (noiseSeed2 %% 100) / 100.0
                    ]);
                case this.ALIASED_POINT_SIZE_RANGE:
                    return new Float32Array([
                        1.0 + (noiseSeed1 %% 10) / 100.0,
                        1024.0 + (noiseSeed2 %% 1024)
                    ]);
                case 37445: // UNMASKED_VENDOR_WEBGL
                    return '%s';
                case 37446: // UNMASKED_RENDERER_WEBGL
                    return '%s';
                case 34047: // MAX_VERTEX_UNIFORM_COMPONENTS
                    return 1024 + (noiseSeed1 %% 1024);
                case 35659: // MAX_VERTEX_UNIFORM_BLOCKS
                    return 12 + (noisePattern %% 4);
                case 35371: // MAX_VARYING_COMPONENTS
                    return 32 + (noiseSeed2 %% 32);
                default:
                    return result;
            }
        };
        
        // ===== 关键修改2: getSupportedExtensions =====
        const originalGetSupportedExtensions = context.getSupportedExtensions.bind(context);
        context.getSupportedExtensions = function() {
            const baseExtensions = [
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
            
            // 根据用户哈希选择性返回扩展
            const selectedExtensions = [];
            for (let i = 0; i < baseExtensions.length; i++) {
                const include = ((noiseSeed1 + i) %% 100) > 5; // 95%%概率包含
                if (include) {
                    selectedExtensions.push(baseExtensions[i]);
                }
            }
            
            return selectedExtensions;
        };
        
        // ===== 关键修改3: shaderSource（影响shader编译）=====
        const originalShaderSource = context.shaderSource.bind(context);
        context.shaderSource = function(shader, source) {
            // 添加用户特定的注释和precision修饰
            const userComment = '// User fingerprint hash: ' + userHash.substr(0, 32) + '\\n';
            const precisionMod = 'precision highp float;\\n// Noise: ' + noiseSeed1 + '\\n';
            const modifiedSource = userComment + precisionMod + source;
            
            return originalShaderSource(shader, modifiedSource);
        };
        
        // ===== 关键修改4: readPixels（Canvas读取）=====
        const originalReadPixels = context.readPixels.bind(context);
        context.readPixels = function(x, y, width, height, format, type, pixels) {
            originalReadPixels(x, y, width, height, format, type, pixels);
            
            // 添加用户特定的像素噪音
            if (pixels && pixels.length) {
                for (let i = 0; i < pixels.length; i += 4) {
                    if (i %% (100 + noisePattern) === (noiseSeed1 %% 100)) {
                        const noise = Math.floor(generateWebGLNoise(i, 0, 0) * 5);
                        pixels[i] = Math.min(255, Math.max(0, pixels[i] + noise));     // R
                        pixels[i+1] = Math.min(255, Math.max(0, pixels[i+1] + noise)); // G
                        pixels[i+2] = Math.min(255, Math.max(0, pixels[i+2] + noise)); // B
                    }
                }
            }
        };
        
        // ===== 关键修改5: getExtension =====
        const originalGetExtension = context.getExtension.bind(context);
        context.getExtension = function(name) {
            const ext = originalGetExtension(name);
            
            if (name === 'WEBGL_debug_renderer_info') {
                return {
                    UNMASKED_VENDOR_WEBGL: 37445,
                    UNMASKED_RENDERER_WEBGL: 37446
                };
            }
            
            if (name === 'EXT_texture_filter_anisotropic' || name === 'WEBKIT_EXT_texture_filter_anisotropic') {
                if (ext) {
                    // 修改各向异性过滤参数
                    const originalGetParam = context.getParameter.bind(context);
                    context.getParameter = function(pname) {
                        if (pname === ext.MAX_TEXTURE_MAX_ANISOTROPY_EXT) {
                            return 16.0 + (noisePattern / 10.0);
                        }
                        return originalGetParam(pname);
                    };
                }
            }
            
            return ext;
        };
        
        // ===== 关键修改6: getActiveAttrib/Uniform（影响程序信息）=====
        const originalGetActiveAttrib = context.getActiveAttrib.bind(context);
        context.getActiveAttrib = function(program, index) {
            const attrib = originalGetActiveAttrib(program, index);
            if (attrib) {
                // 微调属性信息
                attrib.size += (noiseSeed1 %% 2);
            }
            return attrib;
        };
        
        const originalGetActiveUniform = context.getActiveUniform.bind(context);
        context.getActiveUniform = function(program, index) {
            const uniform = originalGetActiveUniform(program, index);
            if (uniform) {
                // 微调uniform信息
                uniform.size += (noiseSeed2 %% 2);
            }
            return uniform;
        };
        
        // ===== 关键修改7: bufferData（影响顶点数据）=====
        const originalBufferData = context.bufferData.bind(context);
        context.bufferData = function(target, sizeOrData, usage) {
            if (sizeOrData && sizeOrData.length) {
                // 对顶点数据添加微小噪音
                for (let i = 0; i < Math.min(10, sizeOrData.length); i++) {
                    if (i %% noisePattern === 0) {
                        const noise = generateWebGLNoise(i, 0, 1) * 0.000001;
                        sizeOrData[i] += noise;
                    }
                }
            }
            return originalBufferData(target, sizeOrData, usage);
        };
        
        return context;
    };
    
    // ====== 修改 toDataURL（Canvas 导出）======
    HTMLCanvasElement.prototype.toDataURL = function(type) {
        // 对于WebGL canvas，在导出前添加噪音
        const ctx = this.getContext('webgl') || this.getContext('experimental-webgl') || 
                    this.getContext('webgl2') || this.getContext('2d');
        
        if (ctx && (ctx instanceof WebGLRenderingContext || ctx instanceof WebGL2RenderingContext)) {
            // WebGL canvas - 读取像素并添加噪音
            try {
                const width = this.width;
                const height = this.height;
                const pixels = new Uint8Array(width * height * 4);
                ctx.readPixels(0, 0, width, height, ctx.RGBA, ctx.UNSIGNED_BYTE, pixels);
                
                // 创建临时canvas来修改像素
                const tempCanvas = document.createElement('canvas');
                tempCanvas.width = width;
                tempCanvas.height = height;
                const tempCtx = tempCanvas.getContext('2d');
                const imageData = tempCtx.createImageData(width, height);
                imageData.data.set(pixels);
                tempCtx.putImageData(imageData, 0, 0);
                
                return tempCanvas.toDataURL(type);
            } catch(e) {
                // 如果出错，使用原始方法
            }
        }
        
        return originalToDataURL.call(this, type);
    };
    
    // ====== 修改 WebGL2 ======
    if (window.WebGL2RenderingContext) {
        window.WebGL2RenderingContext.prototype.getParameter = 
            window.WebGLRenderingContext.prototype.getParameter;
    }
    
    console.log('✅ 超级增强版 WebGL 指纹修改已应用', {
        userHash: userHash.substr(0, 8) + '...',
        vendor: '%s',
        renderer: '%s',
        seeds: [noiseSeed1, noiseSeed2, noisePattern]
    });
})();
`,
		e.config.UserID,
		e.userHash[:16]+"...",
		e.userHash,
		e.noiseSeed1,
		e.noiseSeed2,
		e.noisePattern,
		vendorVariant,
		rendererVariant,
		e.config.WebGL.Version,
		e.config.WebGL.ShadingLanguageVersion,
		e.config.WebGL.MaxTextureSize,
		e.config.WebGL.MaxRenderbufferSize,
		e.config.WebGL.MaxTextureSize,
		e.config.WebGL.MaxTextureSize,
		vendorVariant,
		rendererVariant,
		vendorVariant,
		rendererVariant)
}

// generateWebGLVendorVariant 生成用户特定的Vendor变体
func (e *EnhancedAudioWebGLInjector) generateWebGLVendorVariant() string {
	baseVendor := e.config.WebGL.Vendor
	
	// 根据用户哈希添加细微变化
	variants := []string{
		baseVendor,
		baseVendor + " ",
		" " + baseVendor,
		baseVendor + fmt.Sprintf(" (Build %d)", e.noiseSeed1%10000),
	}
	
	return variants[e.noiseSeed1%len(variants)]
}

// generateWebGLRendererVariant 生成用户特定的Renderer变体
func (e *EnhancedAudioWebGLInjector) generateWebGLRendererVariant() string {
	baseRenderer := e.config.WebGL.Renderer
	
	// 根据用户哈希添加细微变化
	if e.noisePattern%2 == 0 {
		// 添加版本号变化
		buildNum := 20000 + (e.noiseSeed1 % 10000)
		return fmt.Sprintf("%s (Build %d)", baseRenderer, buildNum)
	}
	
	return baseRenderer
}

// generateUserHash 生成用户特定的SHA256哈希
func generateUserHash(userID string) string {
	hasher := sha256.New()
	hasher.Write([]byte(userID + "_audio_webgl_fingerprint"))
	return hex.EncodeToString(hasher.Sum(nil))
}

// hashToInt 从哈希字符串的指定位置提取整数
func hashToInt(hash string, offset int) int {
	if offset+8 > len(hash) {
		offset = 0
	}
	
	value := 0
	for i := 0; i < 8 && offset+i < len(hash); i++ {
		char := hash[offset+i]
		var digit int
		if char >= '0' && char <= '9' {
			digit = int(char - '0')
		} else if char >= 'a' && char <= 'f' {
			digit = int(char-'a') + 10
		}
		value = value*16 + digit
	}
	
	if value < 0 {
		value = -value
	}
	
	return value
}

// CombineWithBaseStealth 将增强脚本与基础隐身脚本结合
func (e *EnhancedAudioWebGLInjector) CombineWithBaseStealth(baseStealthScript string) string {
	audioScript := e.GenerateEnhancedAudioScript()
	webglScript := e.GenerateEnhancedWebGLScript()
	
	return fmt.Sprintf(`
(() => {
    'use strict';
    
    console.log('🔒 开始注入增强版 Audio/WebGL 指纹修改...');
    
    // 1. 基础隐身脚本
    %s
    
    // 2. 增强版 Audio 指纹修改
    %s
    
    // 3. 增强版 WebGL 指纹修改
    %s
    
    console.log('✅ 所有指纹修改已完成！用户ID: %s');
})();
`, baseStealthScript, audioScript, webglScript, e.config.UserID)
}

// CalculateExpectedAudioHash 计算期望的Audio指纹哈希（用于验证）
func (e *EnhancedAudioWebGLInjector) CalculateExpectedAudioHash() string {
	data := fmt.Sprintf("%s_%d_%d_%d_%d_%f",
		e.config.UserID,
		e.config.Audio.SampleRate,
		e.config.Audio.MaxChannelCount,
		e.noiseSeed1,
		e.noiseSeed2,
		e.config.Canvas.NoiseLevel)
	
	hasher := sha256.New()
	hasher.Write([]byte(data))
	return hex.EncodeToString(hasher.Sum(nil))[:40]
}

// CalculateExpectedWebGLHash 计算期望的WebGL指纹哈希（用于验证）
func (e *EnhancedAudioWebGLInjector) CalculateExpectedWebGLHash() string {
	data := fmt.Sprintf("%s_%s_%s_%d_%d_%d",
		e.config.UserID,
		e.config.WebGL.Vendor,
		e.config.WebGL.Renderer,
		e.config.WebGL.MaxTextureSize,
		e.noiseSeed1,
		e.noiseSeed2)
	
	hasher := sha256.New()
	hasher.Write([]byte(data))
	return hex.EncodeToString(hasher.Sum(nil))[:40]
}

// GetDebugInfo 获取调试信息
func (e *EnhancedAudioWebGLInjector) GetDebugInfo() map[string]interface{} {
	return map[string]interface{}{
		"user_id":                 e.config.UserID,
		"user_hash":               e.userHash[:16] + "...",
		"noise_seed1":             e.noiseSeed1,
		"noise_seed2":             e.noiseSeed2,
		"noise_pattern":           e.noisePattern,
		"audio_sample_rate":       e.config.Audio.SampleRate,
		"audio_max_channels":      e.config.Audio.MaxChannelCount,
		"webgl_vendor":            e.config.WebGL.Vendor,
		"webgl_renderer":          e.config.WebGL.Renderer,
		"expected_audio_hash":     e.CalculateExpectedAudioHash(),
		"expected_webgl_hash":     e.CalculateExpectedWebGLHash(),
		"canvas_noise_level":      e.config.Canvas.NoiseLevel,
	}
}

