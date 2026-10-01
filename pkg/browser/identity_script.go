package browser

import (
	"encoding/json"
)

// jsIdentity 注入脚本使用的身份子集，以 JSON 嵌入脚本
type jsIdentity struct {
	UserAgent           string    `json:"userAgent"`
	Platform            string    `json:"platform"`
	Languages           []string  `json:"languages"`
	HardwareConcurrency int       `json:"hardwareConcurrency"`
	Screen              *jsScreen `json:"screen,omitempty"`
	WebGL               *jsWebGL  `json:"webgl,omitempty"`
	Seed                uint32    `json:"seed"`
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

// jsWebGL 显卡标识与 getParameter 覆盖表（键为十进制 GLenum，值为数值或 [a, b]）
type jsWebGL struct {
	Vendor   string         `json:"vendor"`
	Renderer string         `json:"renderer"`
	Params   map[string]any `json:"params,omitempty"`
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
	cfg := jsIdentity{UserAgent: id.UserAgent, Platform: id.Platform, Languages: id.Languages, HardwareConcurrency: id.HardwareConcurrency, Seed: id.NoiseSeed}
	if id.Screen.Width > 0 {
		s := id.Screen
		cfg.Screen = &jsScreen{Width: s.Width, Height: s.Height, AvailWidth: s.AvailWidth, AvailHeight: s.AvailHeight, AvailTop: s.AvailTop, DPR: s.DPR}
	}
	if id.GPU != nil {
		cfg.WebGL = &jsWebGL{Vendor: id.GPU.Vendor, Renderer: id.GPU.Renderer, Params: webglParams(id.GPU.Caps)}
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
const masks = new WeakMap();
const toStringProxy = new Proxy(nativeToString, {
  apply(target, self, args) {
    return masks.has(self) ? masks.get(self) : Reflect.apply(target, self, args);
  },
});
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
  const fake = Object.getOwnPropertyDescriptor({get [key]() { return impl(original, this); }}, key).get;
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
  const W = C.webgl, P = W.params || {};
  for (const Ctx of [G.WebGLRenderingContext, G.WebGL2RenderingContext]) {
    if (!Ctx) continue;
    patchMethod(Ctx.prototype, 'getParameter', (orig, self, args) => {
      const value = Reflect.apply(orig, self, args);
      const p = args[0];
      if (p === 37445) return value === null ? value : W.vendor;
      if (p === 37446) return value === null ? value : W.renderer;
      if (Object.prototype.hasOwnProperty.call(P, p)) {
        const v = P[p];
        return Array.isArray(v) ? (p === 3386 ? new Int32Array(v) : new Float32Array(v)) : v;
      }
      return value;
    });
    if (C.seed) {
      patchMethod(Ctx.prototype, 'readPixels', (orig, self, args) => {
        const out = Reflect.apply(orig, self, args);
        const px = args[6];
        if (px && px.BYTES_PER_ELEMENT === 1 && px.length) noisePixels(px, args[2], args[0], args[1]);
        return out;
      });
    }
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
`

// jsWindowPart 页面 / iframe 专有部分：屏幕可用区域、iframe 中的屏幕与 DPR、受信鼠标事件的屏幕坐标、canvas 与音频噪声
const jsWindowPart = `
if (C.screen && G.Screen) {
  const S = C.screen;
  const fixed = {availWidth: S.availWidth, availHeight: S.availHeight, availTop: S.availTop, availLeft: 0};
  if (G.screen.width !== S.width) fixed.width = S.width;
  if (G.screen.height !== S.height) fixed.height = S.height;
  for (const [k, v] of Object.entries(fixed)) patchGetter(Screen.prototype, k, () => v);
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
