# 真机基准采集（移动身份子项目 M0）设计

状态：设计已口头通过（2026-10-17），待用户审阅本文

## 1. 背景

需求来源：`MOBILE_FINGERPRINT_GUIDE.md`。一个账号在 PC 身份（Windows / macOS）之外，再有一份独立的移动身份（另起 user_id，如 `…_m`），用于抖音 H5 支付页。生产环境不变：无界面 Linux + 桌面版 Google Chrome（无 GPU）+ CustomCDP。

已定决策（2026-10-17）：
- 安卓与 iOS 都做。iOS 只能做到表面像：Chrome 是 Blink + V8，iPhone 上所有浏览器都是 WebKit + JavaScriptCore，报错文本、调用栈格式、TLS 与 HTTP/2 握手特征都能区分，浏览器内无法伪装。安卓 Chrome 与桌面 Chrome 同引擎、同网络栈，没有这个问题。
- 移动身份要在页面上操作（点按、输入、滑动），所以输入层也要按手机的事件模式改。
- 拆成四个子项目，依次各走一遍规格 → 计划 → 实现：
  - **M0 真机基准采集**（本文）。
  - **M1 安卓身份**：系统族、数据池、移动模拟、补齐桌面痕迹、手机 WebGL、按进程配字体。
  - **M2 移动输入**：触摸点按、手指滑动、输入法式打字。
  - **M3 iOS 身份（表面级）**。
- triumph / browser-service 侧（用 PC 会话引导、按 `_m` 存会话）等 M1 发版后再做，不在本库。

2026-10-17 实测（桌面 Chrome 开 CDP 移动模拟后仍是桌面值）：
- `navigator.plugins` 为 5，`pdfViewerEnabled` 为 true。
- `showOpenFilePicker`、`hid`、`serial`、`keyboard`、`queryLocalFonts` 都在；`navigator.contacts` 不在，`connection.type` 不在。
- `new Notification()` 能构造。
- WebGL mediump 浮点精度为 23，扩展是桌面那套。

真机上这些值是多少，靠猜不可靠。M0 的目标就是拿到真机的准确值，作为 M1–M3 的数据来源和测试对照。

## 2. 目标

1. 用户的小米手机（Google Chrome）和 iPhone（Safari）各采集一份完整的浏览器表面数据，脱敏后存进仓库。
2. 采集逻辑是一个可复用的函数：之后在本库模拟的浏览器里跑同一个函数，就能和真机逐项对比。

## 3. 设计

### 3.1 采集页 `test/baseline/collector.html`

- 单文件，不加载任何外部资源。语法限于 Safari 17 / 18 与 Chrome 都支持的范围。
- 每一项检测单独 `try`：抛错时该项记为 `{"error": "<消息>"}`，不影响其他项。
- 导出 `window.collectBaseline()`（async），返回自动采集部分（见 3.2 的 `auto`）。
- 页面流程：
  1. 打开即调用 `collectBaseline()`，显示进度。
  2. 引导三步操作，各有一个明确的区域：
     - 点一下按钮；
     - 在输入框里用手机键盘输入 `ab1`；
     - 在滑动区里上下滑动一次（滑动区内容比区域高，可以滚动）。
  3. 三步完成（或点"跳过"）后，把结果 `POST /result`。成功时显示"已上传"和文件名；失败时显示错误，并给出"复制结果"按钮。
- 交互记录最多 600 个事件，超出后丢弃，并标记 `truncated: true`。

### 3.2 结果格式

顶层：

```json
{"schema": 1, "collectedAt": "<ISO 时间>", "auto": {…}, "interaction": {…}}
```

`auto` 各组：
- **identity**：`userAgent`、`platform`、`vendor`、`language`、`languages`、`hardwareConcurrency`、`deviceMemory`、`maxTouchPoints`、`webdriver`；`userAgentData`（基础值，以及 `getHighEntropyValues` 的 `architecture`、`bitness`、`model`、`platformVersion`、`fullVersionList`、`formFactors`、`wow64`），没有时为 `null`。
- **screen**：
  - `screen` 的 `width`、`height`、`availWidth`、`availHeight`、`availTop`、`availLeft`、`colorDepth`、`pixelDepth`；
  - `devicePixelRatio`、`innerWidth`、`innerHeight`、`outerWidth`、`outerHeight`；
  - `visualViewport` 的 `width`、`height`、`scale`、`offsetTop`；
  - `screen.orientation` 的 `type`、`angle`，以及 `window.orientation`。
- **features**：约 40 个接口是否存在（名称 → 布尔）。包括：
  - 文件与设备类：`showOpenFilePicker`、`showDirectoryPicker`、`queryLocalFonts`、`navigator.hid`、`serial`、`usb`、`bluetooth`、`keyboard`、`windowControlsOverlay`；
  - 手机上常见的：`contacts`、`share`、`canShare`、`BarcodeDetector`、`DeviceMotionEvent`、`DeviceOrientationEvent`、`ontouchstart`、`TouchEvent`、`getBattery`、`vibrate`、`wakeLock`、`virtualKeyboard`、`ApplePaySession`；
  - `window.chrome`，以及 `PaymentRequest`、`Notification`、`speechSynthesis` 等。

  另记录：
  - `plugins.length`、`mimeTypes.length`、`pdfViewerEnabled`；
  - `new Notification('x')` 的结果（能构造，或错误类型加消息）；
  - `navigator.connection` 的 `type`、`effectiveType`、`downlinkMax`、`rtt`、`saveData`；
  - `Object.keys(window.chrome)`；
  - `permissions.query` 对 `notifications`、`geolocation`、`camera` 的状态。
- **media**：`pointer`、`hover`、`any-pointer`、`any-hover`、`prefers-color-scheme`、`prefers-reduced-motion`、`color-gamut`、`dynamic-range`、`display-mode`、`forced-colors`、`prefers-contrast` 各取值的匹配结果。
- **webgl** / **webgl2**：
  - 标识：`VENDOR`、`RENDERER`、`UNMASKED_VENDOR_WEBGL`、`UNMASKED_RENDERER_WEBGL`、`VERSION`、`SHADING_LANGUAGE_VERSION`；
  - 能力参数：本库 `webglCaps` 已有的全部参数，以及 WebGL2 的 `MAX_3D_TEXTURE_SIZE`、`MAX_DRAW_BUFFERS`、`MAX_SAMPLES`、`MAX_UNIFORM_BLOCK_SIZE` 等；
  - 着色器精度：顶点 / 片元 × `LOW` / `MEDIUM` / `HIGH` × `FLOAT` / `INT`，共 12 组 `rangeMin`、`rangeMax`、`precision`；
  - `getSupportedExtensions()` 全表；`getContextAttributes()`。
- **fonts**：约 80 个候选字体（Windows、macOS、安卓、iOS、常见中文字体）的有无，用与 3 种基础字体的宽度对比判断。
- **engine**：
  - 三段报错代码的错误类型和消息：读 `null` 的属性、调用 `undefined`、`JSON.parse` 非法输入；
  - 一个调用栈样本（只保留格式，文件路径替换为 `<url>`）；
  - `Function.prototype.toString.call(Math.max)`；
  - `Math.tan(-1e300)`、`Math.sinh(1)` 等几个精度样本；
  - `Intl.DateTimeFormat().resolvedOptions()` 与 `Intl.NumberFormat().resolvedOptions()` 的 `locale`、`timeZone`、`calendar`、`numberingSystem`。
- **avmedia**：
  - `speechSynthesis.getVoices()` 的数量与前 5 个 `name` / `lang`（等待 `voiceschanged` 最多 1 秒）；
  - 约 20 个 MIME 类型的 `canPlayType` 与 `MediaSource.isTypeSupported`；
  - `mediaDevices.enumerateDevices()` 只记各 `kind` 的数量。

`interaction`：三步操作期间页面收到的事件，按时间顺序记录，每个事件记：
- 通用：`step`（`tap` / `type` / `swipe`）、`type`、`isTrusted`、`t`（相对第一个事件的毫秒数）、`target`（元素 id）。
- 指针、鼠标：`pointerType`、`pointerId`、`isPrimary`、`width`、`height`、`pressure`、`tangentialPressure`、`tiltX`、`tiltY`、`twist`、`button`、`buttons`、`detail`、`clientX/Y`、`screenX/Y`。
- 触摸：`touches`、`changedTouches` 中每个触点的 `identifier`、`clientX/Y`、`radiusX/Y`、`rotationAngle`、`force`。
- 键盘、输入：`key`、`code`、`keyCode`、`which`、`location`、`repeat`、`isComposing`、`inputType`、`data`。
- 滚动：`scroll` 事件的 `scrollTop`。

监听的事件类型：`pointer*`、`touch*`、`mouse*`、`click`、`keydown`、`keypress`、`keyup`、`beforeinput`、`input`、`composition*`、`scroll`。

### 3.3 本地服务 `cmd/baseline_server/main.go`

- 只监听 `127.0.0.1:<端口>`（默认 8765，可用 `-port` 修改）。
- `GET /` 返回采集页；`POST /result` 接收结果。
  - 请求体超过 1 MiB 返回 413；不是合法 JSON 或缺少 `auto` 返回 400。
  - 文件名为 `<系统>-<浏览器>-<机型>.json`，存到 `test/baseline/`（可用 `-out` 修改目录）：
    - 系统、浏览器由 UA 判定（`android` / `ios` / `other`，`chrome` / `safari` / `other`）；
    - 机型取 Client Hints 的 `model`，没有时 iOS 记 `iphone`；
    - 只保留小写字母、数字和连字符；
    - 已存在时加 `-2`、`-3` 等序号，不覆盖。
  - 成功返回 `{"file": "<文件名>"}`，并在终端打印文件名和 UA。
- 不记录请求方地址，不写访问日志。

### 3.4 启动脚本 `scripts/collect-baseline.sh`

- 检查 `cloudflared` 是否安装，没有时提示 `brew install cloudflared` 后退出。
- 启动 `go run ./cmd/baseline_server`，再执行 `cloudflared tunnel --url http://127.0.0.1:8765`。
- 从 cloudflared 的输出里取出 `https://*.trycloudflare.com` 地址，醒目地打印出来。
- 按 Ctrl-C 时两个进程一起停止，地址随之失效。

## 4. 隐私

- 不采集 IP、地理位置、通讯录、媒体设备名称，也不采任何需要授权才能读的数据。权限只查询状态，不申请。
- 调用栈样本中的地址替换为 `<url>`。
- 结果文件提交进仓库前，由实现者逐个检查一遍，确认没有个人信息。机型代号（如 `23127PN0CC`）不算个人信息，保留。

## 5. 后续子项目怎么用

- **M1 / M3**：数据池以基准中的真机值为准。新增对照测试：在本库模拟的手机身份里用 `httptest` 加载同一采集页，调用 `collectBaseline()`，与基准比较。
  - 必须一致：接口有无、媒体查询、WebGL 精度和扩展、视口与屏幕的比例关系。
  - 按数据池检查：与机型有关的值（屏幕尺寸、GPU 名称）。
  - 比较规则在 M1 的规格里定。
- **M2**：`interaction` 作为触摸、滑动、输入法事件模拟的对照。

## 6. 测试

- `cmd/baseline_server` 单元测试：
  - 正常保存，文件名推导覆盖安卓 Chrome 与 iPhone Safari 两种 UA；
  - 同名文件追加序号；
  - 超过 1 MiB 返回 413，非法 JSON 返回 400；
  - 机型中的特殊字符被过滤。
- `pkg/browser` 集成测试：本库启动的 Chrome 用 `httptest` 加载采集页，`collectBaseline()` 不抛错，并且：
  - `identity`、`screen`、`features`、`media`、`webgl`、`fonts`、`engine`、`avmedia` 各组都存在；
  - 没有任何一项是 `{"error": …}`，桌面 Chrome 上每项检测都应能跑通。

## 7. 完成标准

- 以上测试在 macOS 与 Linux 容器中通过。
- 用户用小米 Chrome 与 iPhone Safari 各采集一次，两份结果经检查后提交到 `test/baseline/`。

## 8. 不做

- 不改动现有的 `fingerprint_collector.html` 与 `cmd/fingerprint_collector`（面向桌面的旧工具）。
- 不采 canvas / 音频指纹。
- M1–M3 的任何身份或输入改动。
