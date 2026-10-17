package browser

import (
	"encoding/json"
	"strconv"
)

// jsIdentity 注入脚本使用的身份子集，以 JSON 嵌入脚本
type jsIdentity struct {
	UserAgent           string     `json:"userAgent"`
	Platform            string     `json:"platform"`
	Languages           []string   `json:"languages"`
	HardwareConcurrency int        `json:"hardwareConcurrency"`
	Screen              *jsScreen  `json:"screen,omitempty"`
	WebGL               *jsWebGL   `json:"webgl,omitempty"`
	Seed                uint32     `json:"seed"`
	Android             *jsAndroid `json:"android,omitempty"`
	Key                 string     `json:"key"` // 见 Identity.ScriptKey
}

// jsAndroid 安卓身份专有：内存与电量
type jsAndroid struct {
	DeviceMemory int     `json:"deviceMemory"`
	Battery      float64 `json:"battery"`
}

// jsScreen 屏幕与可用区域
type jsScreen struct {
	Width       int     `json:"width"`
	Height      int     `json:"height"`
	AvailWidth  int     `json:"availWidth"`
	AvailHeight int     `json:"availHeight"`
	AvailTop    int     `json:"availTop"`
	DPR         float64 `json:"dpr"`
}

// jsWebGL 显卡标识与 getParameter 覆盖表（键为十进制 GLenum，值为数值或 [a, b]）；手机 GPU 另有只属于 WebGL2 的参数、扩展列表与着色器精度
type jsWebGL struct {
	Vendor      string            `json:"vendor"`
	Renderer    string            `json:"renderer"`
	Params      map[string]any    `json:"params,omitempty"`
	Params2     map[string]any    `json:"params2,omitempty"`
	Extensions  []string          `json:"extensions,omitempty"`
	Extensions2 []string          `json:"extensions2,omitempty"`
	Precision   map[string][3]int `json:"precision,omitempty"` // "着色器.精度类型" → [rangeMin, rangeMax, precision]
}

// glParamTable GLenum 为键的参数表转为脚本用的十进制字符串键
func glParamTable(m map[int]any) map[string]any {
	if len(m) == 0 {
		return nil
	}
	out := make(map[string]any, len(m))
	for k, v := range m {
		out[strconv.Itoa(k)] = v
	}
	return out
}

// webglParams 能力参数转为 getParameter 覆盖表
func webglParams(c *webglCaps) map[string]any {
	if c == nil {
		return nil
	}
	return map[string]any{
		"3379":  c.MaxTextureSize,               // MAX_TEXTURE_SIZE
		"34076": c.MaxCubeMapTextureSize,        // MAX_CUBE_MAP_TEXTURE_SIZE
		"34024": c.MaxRenderbufferSize,          // MAX_RENDERBUFFER_SIZE
		"3386":  c.MaxViewportDims[:],           // MAX_VIEWPORT_DIMS（Int32Array）
		"34921": c.MaxVertexAttribs,             // MAX_VERTEX_ATTRIBS
		"36347": c.MaxVertexUniformVectors,      // MAX_VERTEX_UNIFORM_VECTORS
		"36349": c.MaxFragmentUniformVectors,    // MAX_FRAGMENT_UNIFORM_VECTORS
		"36348": c.MaxVaryingVectors,            // MAX_VARYING_VECTORS
		"34930": c.MaxTextureImageUnits,         // MAX_TEXTURE_IMAGE_UNITS
		"35660": c.MaxVertexTextureImageUnits,   // MAX_VERTEX_TEXTURE_IMAGE_UNITS
		"35661": c.MaxCombinedTextureImageUnits, // MAX_COMBINED_TEXTURE_IMAGE_UNITS
		"33901": c.AliasedPointSizeRange[:],     // ALIASED_POINT_SIZE_RANGE（Float32Array）
		"33902": c.AliasedLineWidthRange[:],     // ALIASED_LINE_WIDTH_RANGE（Float32Array）
	}
}

// identityScript 生成注入脚本；worker 为 true 时生成 Worker 版本（补 WorkerNavigator，不含 DOM / Web Audio 部分）
func identityScript(id *Identity, worker bool) string {
	key := id.ScriptKey
	if key == "" {
		key = newScriptKey()
	}
	cfg := jsIdentity{UserAgent: id.UserAgent, Platform: id.Platform, Languages: id.Languages, HardwareConcurrency: id.HardwareConcurrency, Seed: id.NoiseSeed, Key: key}
	if id.Screen.Width > 0 {
		s := id.Screen
		cfg.Screen = &jsScreen{Width: s.Width, Height: s.Height, AvailWidth: s.AvailWidth, AvailHeight: s.AvailHeight, AvailTop: s.AvailTop, DPR: s.DPR}
	}
	if id.OS == OSAndroid {
		cfg.Android = &jsAndroid{DeviceMemory: id.DeviceMemory, Battery: id.Battery}
	}
	if id.GPU != nil {
		w := &jsWebGL{Vendor: id.GPU.Vendor, Renderer: id.GPU.Renderer, Params: webglParams(id.GPU.Caps)}
		if m := id.GPU.Mobile; m != nil {
			w.Params, w.Params2 = glParamTable(m.Params), glParamTable(m.Params2)
			w.Extensions, w.Extensions2 = m.Extensions, m.Extensions2
			w.Precision = make(map[string][3]int, len(m.Precision))
			for k, v := range m.Precision {
				w.Precision[strconv.Itoa(k[0])+"."+strconv.Itoa(k[1])] = v
			}
		}
		cfg.WebGL = w
	}
	data, _ := json.Marshal(cfg) // 只含基本类型，不会失败
	body := jsWindowPart
	if worker {
		body = jsWorkerPart
	}
	return "(() => {\n'use strict';\nconst C = " + string(data) + ";\n" + jsPrelude + jsCommonPart + body + "\n})();"
}

// jsPrelude 原生伪装工具与确定性噪声
const jsPrelude = `
const G = globalThis;
const nativeToString = Function.prototype.toString;
// 伪装表在同源的父窗口 / opener 之间共用：否则拿 iframe 或弹窗自己的 toString 就能看到这里替换函数的源码。
// 以随机口令 C.key 作 this 调用对方的 toString 取表；跨源、对方没注入或口令不同时会抛错，退回用自己的表
let masks = new WeakMap();
for (const w of [G.parent, G.opener]) {
  try {
    const shared = w && w !== G ? Reflect.apply(w.Function.prototype.toString, C.key, []) : null;
    if (shared && typeof shared === 'object') {
      masks = shared;
      break;
    }
  } catch (e) {}
}
// 替换后的 toString 用方法写法：和原生一样不可 new、没有 prototype。不用 Proxy：Proxy 会被原型循环检查
// （CreepJS 的 too much recursion / reflect set proto）和报错调用栈识破
const toStringProxy = {toString() {
  if (this === C.key) return masks;
  return masks.has(this) ? masks.get(this) : Reflect.apply(nativeToString, this, []);
}}.toString;
masks.set(toStringProxy, Reflect.apply(nativeToString, nativeToString, []));
Object.defineProperty(Function.prototype, 'toString', {value: toStringProxy});
const disguise = (fake, original) => {
  masks.set(fake, Reflect.apply(nativeToString, original, []));
  Object.defineProperty(fake, 'name', {value: original.name});
  Object.defineProperty(fake, 'length', {value: original.length});
  return fake;
};
const owner = (obj, key) => {
  while (obj && !Object.getOwnPropertyDescriptor(obj, key)) obj = Object.getPrototypeOf(obj);
  return obj;
};
const patchMethod = (obj, key, impl) => {
  const target = owner(obj, key);
  const desc = target && Object.getOwnPropertyDescriptor(target, key);
  if (!desc || typeof desc.value !== 'function') return;
  const original = desc.value;
  const fake = {[key](...args) { return impl(original, this, args); }}[key];
  Object.defineProperty(target, key, {value: disguise(fake, original)});
};
const patchGetter = (obj, key, impl) => {
  const target = owner(obj, key);
  const desc = target && Object.getOwnPropertyDescriptor(target, key);
  if (!desc || typeof desc.get !== 'function') return;
  const original = desc.get;
  // 先调原生 getter：this 不对时和原生一样抛 Illegal invocation
  const fake = Object.getOwnPropertyDescriptor({get [key]() { Reflect.apply(original, this, []); return impl(original, this); }}, key).get;
  Object.defineProperty(target, key, {get: disguise(fake, original)});
};
const hash = (n) => {
  let h = Math.imul(n ^ C.seed, 2654435761) >>> 0;
  h ^= h >>> 15;
  h = Math.imul(h, 2246822519) >>> 0;
  h ^= h >>> 13;
  return h;
};
const noisePixels = (data, width, x0, y0) => {
  for (let i = 0; i < data.length; i += 4) {
    if (data[i + 3] === 0) continue;
    const p = i >> 2;
    const h = hash((y0 + Math.floor(p / width)) * 8192 + x0 + (p % width));
    if ((h & 63) === 0) data[i + ((h >>> 6) % 3)] ^= 1;
  }
};
`

// jsCommonPart 页面与 Worker 都有的部分：WebGL 标识 / 能力参数与 readPixels 噪声、OffscreenCanvas 噪声
const jsCommonPart = `
if (C.webgl) {
  const W = C.webgl, P = W.params || {}, P2 = W.params2 || {};
  const GL2 = G.WebGL2RenderingContext;
  const has = (o, k) => Object.prototype.hasOwnProperty.call(o, k);
  for (const Ctx of [G.WebGLRenderingContext, GL2]) {
    if (!Ctx) continue;
    const is2 = Ctx === GL2;
    patchMethod(Ctx.prototype, 'getParameter', (orig, self, args) => {
      const value = Reflect.apply(orig, self, args);
      const p = args[0];
      if (p === 37445) return value === null ? value : W.vendor;
      if (p === 37446) return value === null ? value : W.renderer;
      const table = is2 && has(P2, p) ? P2 : P;
      if (has(table, p)) {
        const v = table[p];
        return Array.isArray(v) ? (p === 3386 ? new Int32Array(v) : new Float32Array(v)) : v;
      }
      return value;
    });
    // 手机 GPU：扩展列表按真机；不在列表里的扩展取不到，也不调用原生（否则会在上下文里悄悄启用）
    const exts = is2 ? W.extensions2 : W.extensions;
    if (exts) {
      const allowed = new Set(exts.map(e => e.toLowerCase()));
      patchMethod(Ctx.prototype, 'getSupportedExtensions', (orig, self, args) => {
        const list = Reflect.apply(orig, self, args);
        return list === null ? null : exts.slice();
      });
      patchMethod(Ctx.prototype, 'getExtension', (orig, self, args) => {
        if (!(self instanceof Ctx) || !args.length || allowed.has(String(args[0]).toLowerCase())) return Reflect.apply(orig, self, args);
        return null;
      });
    }
    if (C.seed) {
      patchMethod(Ctx.prototype, 'readPixels', (orig, self, args) => {
        const out = Reflect.apply(orig, self, args);
        const px = args[6];
        if (px && px.BYTES_PER_ELEMENT === 1 && px.length) noisePixels(px, args[2], args[0], args[1]);
        return out;
      });
    }
  }
  // 手机 GPU 的着色器精度：按对象替换 WebGLShaderPrecisionFormat 的三个取值
  if (W.precision && G.WebGLShaderPrecisionFormat) {
    const over = new WeakMap();
    for (const Ctx of [G.WebGLRenderingContext, GL2]) {
      if (!Ctx) continue;
      patchMethod(Ctx.prototype, 'getShaderPrecisionFormat', (orig, self, args) => {
        const f = Reflect.apply(orig, self, args);
        const v = f && W.precision[args[0] + '.' + args[1]];
        if (v) over.set(f, v);
        return f;
      });
    }
    ['rangeMin', 'rangeMax', 'precision'].forEach((k, i) => {
      patchGetter(WebGLShaderPrecisionFormat.prototype, k, (orig, self) => over.has(self) ? over.get(self)[i] : Reflect.apply(orig, self, []));
    });
  }
}
if (C.seed && G.OffscreenCanvas && G.OffscreenCanvasRenderingContext2D) {
  const offGetContext = OffscreenCanvas.prototype.getContext;
  const offGetImageData = OffscreenCanvasRenderingContext2D.prototype.getImageData;
  patchMethod(OffscreenCanvasRenderingContext2D.prototype, 'getImageData', (orig, self, args) => {
    const img = Reflect.apply(orig, self, args);
    noisePixels(img.data, img.width, args[0] | 0, args[1] | 0);
    return img;
  });
  patchMethod(OffscreenCanvas.prototype, 'convertToBlob', (orig, self, args) => {
    if (!self.width || !self.height) return Reflect.apply(orig, self, args);
    const copy = new OffscreenCanvas(self.width, self.height);
    const ctx = Reflect.apply(offGetContext, copy, ['2d']);
    ctx.drawImage(self, 0, 0);
    const img = Reflect.apply(offGetImageData, ctx, [0, 0, self.width, self.height]);
    noisePixels(img.data, self.width, 0, 0);
    ctx.putImageData(img, 0, 0);
    return Reflect.apply(orig, copy, args);
  });
}
if (C.android) {
  // 内存（页面与 Worker）
  for (const N of [G.Navigator, G.WorkerNavigator]) if (N) patchGetter(N.prototype, 'deviceMemory', () => C.android.deviceMemory);
  // 连接类型：手机身份启动时打开了原生的 type / downlinkMax（NetInfoDownlinkMax），这里只把服务器的 ethernet 换成 wifi
  if (G.NetworkInformation && 'type' in NetworkInformation.prototype) patchGetter(NetworkInformation.prototype, 'type', () => 'wifi');
}
`

// jsWindowPart 页面 / iframe 专有部分：屏幕可用区域、iframe 中的屏幕与 DPR、受信鼠标事件的屏幕坐标、canvas 与音频噪声
const jsWindowPart = `
if (C.screen && G.Screen) {
  const S = C.screen;
  // 只在原生值不符时改写（无界面 + 指纹时启动参数已让原生值正确，见 headlessScreenFlags）
  const want = {width: S.width, height: S.height, availWidth: S.availWidth, availHeight: S.availHeight, availTop: S.availTop, availLeft: 0};
  for (const [k, v] of Object.entries(want)) if (G.screen[k] !== v) patchGetter(Screen.prototype, k, () => v);
  if (G.devicePixelRatio !== S.dpr) patchGetter(G, 'devicePixelRatio', () => S.dpr);
}
if (G.MouseEvent && G === G.top) {
  const uiX = () => (G.outerWidth - G.innerWidth) / 2;
  const uiY = () => G.outerHeight - G.innerHeight - uiX();
  patchGetter(MouseEvent.prototype, 'screenX', (orig, self) => self.isTrusted ? self.clientX + G.screenX + uiX() : Reflect.apply(orig, self, []));
  patchGetter(MouseEvent.prototype, 'screenY', (orig, self) => self.isTrusted ? self.clientY + G.screenY + uiY() : Reflect.apply(orig, self, []));
}
if (C.seed && G.HTMLCanvasElement) {
  const createElement = Document.prototype.createElement;
  const getContext = HTMLCanvasElement.prototype.getContext;
  const getImageData = CanvasRenderingContext2D.prototype.getImageData;
  patchMethod(CanvasRenderingContext2D.prototype, 'getImageData', (orig, self, args) => {
    const img = Reflect.apply(orig, self, args);
    noisePixels(img.data, img.width, args[0] | 0, args[1] | 0);
    return img;
  });
  const exportNoisy = (canvas, orig, args) => {
    const w = canvas.width, h = canvas.height;
    if (!w || !h) return Reflect.apply(orig, canvas, args);
    const copy = Reflect.apply(createElement, document, ['canvas']);
    copy.width = w;
    copy.height = h;
    const ctx = Reflect.apply(getContext, copy, ['2d']);
    ctx.drawImage(canvas, 0, 0);
    const img = Reflect.apply(getImageData, ctx, [0, 0, w, h]);
    noisePixels(img.data, w, 0, 0);
    ctx.putImageData(img, 0, 0);
    return Reflect.apply(orig, copy, args);
  };
  patchMethod(HTMLCanvasElement.prototype, 'toDataURL', (orig, self, args) => exportNoisy(self, orig, args));
  patchMethod(HTMLCanvasElement.prototype, 'toBlob', (orig, self, args) => exportNoisy(self, orig, args));
}
if (G.Navigator && G.isSecureContext && /^(Win32|MacIntel|Linux armv81)$/.test(C.platform) && !('share' in Navigator.prototype)) {
  // Linux 版 Chrome 没有 Web Share，Windows / Mac / 安卓版有（CreepJS 的 noWebShare）：按真实 Chrome 的行为补上
  const N = Navigator.prototype;
  const receiverCheck = Object.getOwnPropertyDescriptor(N, 'userAgent').get; // this 不是 Navigator 时和原生一样抛 Illegal invocation
  let allowed = G === G.top;
  if (!allowed) {
    try { allowed = !!G.top.location.href; } catch (e) {} // 跨域 iframe 默认没有 web-share 权限
  }
  const baseURL = G.location.href;
  const shareable = (data) => {
    if (data === null || typeof data !== 'object') return false;
    const has = data.title !== undefined || data.text !== undefined || data.url !== undefined || !!(data.files && data.files.length);
    if (!has) return false;
    if (data.url !== undefined) {
      try { new URL(String(data.url), baseURL); } catch (e) { return false; }
    }
    return true;
  };
  let pending = false;
  const addMethod = (name, fn) => {
    masks.set(fn, 'function ' + name + '() { [native code] }');
    Object.defineProperty(N, name, {value: fn, writable: true, enumerable: true, configurable: true});
  };
  addMethod('canShare', {canShare() {
    Reflect.apply(receiverCheck, this, []);
    return allowed && shareable(arguments[0]);
  }}.canShare);
  // 检查顺序与 Chrome 一致：接收者、权限、数据、进行中的分享、用户手势；有手势时当作用户关掉了分享框
  addMethod('share', {share() {
    try { Reflect.apply(receiverCheck, this, []); } catch (e) { return Promise.reject(e); }
    if (!allowed) return Promise.reject(new DOMException('Permission denied', 'NotAllowedError'));
    if (!shareable(arguments[0])) {
      return Promise.reject(new TypeError('No known share data fields supplied. If using only new fields (other than title, text and url), you must feature-detect them first.'));
    }
    if (pending) return Promise.reject(new DOMException('An earlier share has not yet completed.', 'InvalidStateError'));
    const activation = G.navigator.userActivation;
    if (!(activation && activation.isActive)) {
      return Promise.reject(new DOMException('Must be handling a user gesture to perform a share request.', 'NotAllowedError'));
    }
    pending = true;
    return new Promise((resolve, reject) => setTimeout(() => {
      pending = false;
      reject(new DOMException('Share canceled', 'AbortError'));
    }, 800));
  }}.share);
}
if (C.seed && G.AudioBuffer) {
  const noised = new WeakSet();
  patchMethod(AudioBuffer.prototype, 'getChannelData', (orig, self, args) => {
    const data = Reflect.apply(orig, self, args);
    if (!noised.has(data)) {
      noised.add(data);
      const ch = args[0] | 0;
      for (let i = 0; i < data.length; i += 97) {
        const h = hash(i * 8 + ch);
        if ((h & 7) === 0) data[i] += ((h >>> 3) & 1 ? 1 : -1) * 1e-7;
      }
    }
    return data;
  });
  const getChannelData = AudioBuffer.prototype.getChannelData;
  patchMethod(AudioBuffer.prototype, 'copyFromChannel', (orig, self, args) => {
    Reflect.apply(getChannelData, self, [args[1] | 0]);
    return Reflect.apply(orig, self, args);
  });
}
if (C.android) {
  const A = C.android;
  // 桌面独有的接口：安卓 Chrome 没有
  for (const k of ['hid', 'windowControlsOverlay']) delete Navigator.prototype[k];
  for (const k of ['EyeDropper', 'SharedWorker', 'documentPictureInPicture', 'queryLocalFonts', 'HID', 'HIDDevice', 'HIDConnectionEvent',
    'HIDInputReportEvent', 'WindowControlsOverlay', 'WindowControlsOverlayGeometryChangeEvent', 'DocumentPictureInPicture',
    'DocumentPictureInPictureEvent', 'FontData', 'getScreenDetails', 'ScreenDetails', 'ScreenDetailed', 'CaptureController', 'CropTarget',
    'RestrictionTarget']) delete G[k];
  // 屏幕录制（getDisplayMedia）与采集句柄只有桌面 Chrome 有
  if (G.MediaDevices) for (const k of ['getDisplayMedia', 'setCaptureHandleConfig']) delete MediaDevices.prototype[k];
  if (G.chrome) delete G.chrome.app;
  // Notification：安卓只能经 ServiceWorkerRegistration.showNotification 发通知，new 时抛错；静态成员与原型保持原生
  if (G.Notification) {
    const Orig = G.Notification;
    const Fake = function Notification(title) {
      if (new.target === undefined) return Reflect.apply(Orig, this, arguments);
      throw new TypeError("Failed to construct 'Notification': Illegal constructor. Use ServiceWorkerRegistration.showNotification() instead.");
    };
    for (const [k, d] of Object.entries(Object.getOwnPropertyDescriptors(Orig))) {
      if (k !== 'length' && k !== 'name' && k !== 'prototype') Object.defineProperty(Fake, k, d);
    }
    Object.defineProperty(Fake, 'prototype', {value: Orig.prototype, writable: false});
    Object.defineProperty(Orig.prototype, 'constructor', {value: Fake, writable: true, enumerable: false, configurable: true});
    Object.setPrototypeOf(Fake, Object.getPrototypeOf(Orig));
    disguise(Fake, Orig);
    Object.defineProperty(G, 'Notification', {value: Fake, writable: true, enumerable: false, configurable: true});
  }
  // 通讯录选择器（Contact Picker）：只在安全上下文、只有安卓有
  if (G.isSecureContext && !('contacts' in Navigator.prototype)) {
    const N = Navigator.prototype;
    const receiverCheck = Object.getOwnPropertyDescriptor(N, 'userAgent').get;
    const CM = function ContactsManager() { throw new TypeError('Illegal constructor'); };
    masks.set(CM, 'function ContactsManager() { [native code] }');
    const proto = CM.prototype;
    const isManager = (self) => { if (!(self instanceof CM)) throw new TypeError('Illegal invocation'); };
    const addProtoMethod = (name, fn) => {
      masks.set(fn, 'function ' + name + '() { [native code] }');
      Object.defineProperty(proto, name, {value: fn, writable: true, enumerable: true, configurable: true});
    };
    addProtoMethod('getProperties', {getProperties() {
      try { isManager(this); } catch (e) { return Promise.reject(e); }
      return Promise.resolve(['address', 'email', 'icon', 'name', 'tel']);
    }}.getProperties);
    // 没有用户手势时与真机一样拒绝；有手势时当作用户关掉了选择框（真机返回空列表）
    addProtoMethod('select', {select(properties) {
      try { isManager(this); } catch (e) { return Promise.reject(e); }
      const activation = G.navigator.userActivation;
      if (!(activation && activation.isActive)) {
        return Promise.reject(new DOMException("Failed to execute 'select' on 'ContactsManager': A user gesture is required to call this method.", 'SecurityError'));
      }
      return new Promise((resolve) => setTimeout(() => resolve([]), 800));
    }}.select);
    Object.defineProperty(proto, Symbol.toStringTag, {value: 'ContactsManager', configurable: true});
    const manager = Object.create(proto);
    const getter = Object.getOwnPropertyDescriptor({get contacts() { Reflect.apply(receiverCheck, this, []); return manager; }}, 'contacts').get;
    masks.set(getter, 'function get contacts() { [native code] }');
    Object.defineProperty(N, 'contacts', {get: getter, enumerable: true, configurable: true});
    Object.defineProperty(G, 'ContactsManager', {value: CM, writable: true, enumerable: false, configurable: true});
  }
  // 条码识别：安卓与 Mac 版 Chrome 有，Linux 版没有。宿主没有时补上：支持安卓的格式，识别结果为空（看不到条码）
  if (!G.BarcodeDetector && G.isSecureContext) {
    const formats = ['aztec', 'code_128', 'code_39', 'code_93', 'codabar', 'data_matrix', 'ean_13', 'ean_8', 'itf', 'pdf417', 'qr_code', 'upc_a', 'upc_e'];
    const BD = function BarcodeDetector(options) {
      if (new.target === undefined) throw new TypeError("Failed to construct 'BarcodeDetector': Please use the 'new' operator, this DOM object constructor cannot be called as a function.");
      const wanted = options && options.formats;
      if (wanted !== undefined && (!wanted.length || wanted.some(f => !formats.includes(f)))) {
        throw new TypeError("Failed to construct 'BarcodeDetector': Hint option provided, but is empty or contains an unknown format.");
      }
    };
    masks.set(BD, 'function BarcodeDetector() { [native code] }');
    const getSupportedFormats = {getSupportedFormats() { return Promise.resolve(formats.slice()); }}.getSupportedFormats;
    masks.set(getSupportedFormats, 'function getSupportedFormats() { [native code] }');
    Object.defineProperty(BD, 'getSupportedFormats', {value: getSupportedFormats, writable: true, enumerable: true, configurable: true});
    const detect = {detect(image) {
      if (!(this instanceof BD)) return Promise.reject(new TypeError('Illegal invocation'));
      return Promise.resolve([]);
    }}.detect;
    masks.set(detect, 'function detect() { [native code] }');
    Object.defineProperty(BD.prototype, 'detect', {value: detect, writable: true, enumerable: true, configurable: true});
    Object.defineProperty(BD.prototype, Symbol.toStringTag, {value: 'BarcodeDetector', configurable: true});
    Object.defineProperty(G, 'BarcodeDetector', {value: BD, writable: true, enumerable: false, configurable: true});
  }
  // 电池：放电中、电量按账号固定；时间未知时为 Infinity（与真机一致）
  if (G.BatteryManager) {
    const B = BatteryManager.prototype;
    patchGetter(B, 'charging', () => false);
    patchGetter(B, 'chargingTime', () => Infinity);
    patchGetter(B, 'dischargingTime', () => Infinity);
    patchGetter(B, 'level', () => A.battery);
  }
}
`

// jsWorkerPart Worker 专有部分：CDP 覆盖不到 Worker 的 platform / languages / hardwareConcurrency，
// 也覆盖不到 SharedWorker 的 userAgent / appVersion（实测仍是 HeadlessChrome；userAgentData 与请求头已由 CDP 覆盖）
const jsWorkerPart = `
if (G.WorkerNavigator) {
  const N = WorkerNavigator.prototype;
  const languages = Object.freeze([...C.languages]);
  const appVersion = C.userAgent.replace(/^Mozilla\//, '');
  patchGetter(N, 'userAgent', () => C.userAgent);
  patchGetter(N, 'appVersion', () => appVersion);
  patchGetter(N, 'platform', () => C.platform);
  patchGetter(N, 'hardwareConcurrency', () => C.hardwareConcurrency);
  patchGetter(N, 'languages', () => languages);
  patchGetter(N, 'language', () => languages[0]);
}
`
