# iOS 身份（移动身份子项目 M3，表面级）设计

状态：已定稿（2026-10-17）。用户指示"完成所有 M 的任务，中途不要停下来问"，标注"实现者决定"的选择题由实现者按推荐项自行决定。

## 1. 背景与定位

用户选择 iOS 也要做，并已知晓它只能做到"表面像"（M0 规格第 1 节）。

无法伪装、因而能被识破的部分：
- **JS 引擎**：Chrome 是 V8，iPhone 上所有浏览器都是 JavaScriptCore。报错文本（`null is not an object (evaluating 'o.x')`、`JSON Parse error: Expected '}'`）、调用栈格式（`f@url:1:2`）、数学函数的末位精度都不同。
- **TLS / HTTP2 握手**：网络层就能看出是 Chrome 不是 Safari。
- **渲染引擎**：CSS 特性支持、表单控件外观、字体渲染都是 Blink 的。

M3 能做到的：浏览器内可见的标识与接口表面和 iPhone Safari 一致。也就是基准中 `identity`、`screen`、`features`、`media`、`webgl`、`engine.nativeToString` 这些项目。

真机基准：
- `test/baseline/ios-safari-iphone-2.json`：iOS 26，Safari 26.2，UA 系统段冻结为 `18_7`，420×912@3；
- `test/baseline/ios-safari-iphone.json`：iOS 18.5，Safari 18.5，440×956@3。

## 2. 开启方式

- `ConnectOptions{Mobile: true, UserAgent: <iPhone UA>}` 为新账号生成 iOS 身份。指纹 `device.kind` 为 `ios`；其余规则同 M1：已有指纹永不改系统，类型不一致时报错。
- `knownDeviceKind` 加上 `ios`。

## 3. 机型档案（实现者决定：两款，均为真机实测）

| 机型 | 屏幕 | 视口 | Safari | UA 系统段 | 来源 | 权重 |
|---|---|---|---|---|---|---|
| iPhone（iOS 26） | 420×912@3 | 420×912 | 26.2 | 18_7 | 真机实测 | 60 |
| iPhone（iOS 18.5） | 440×956@3 | 440×956 | 18.5 | 18_5 | 真机实测 | 40 |

- Safari 的版本由机型档案决定，**不**跟随宿主 Chrome。
- 共用的值：`platform` 为 `iPhone`，`vendor` 为 `Apple Computer, Inc.`，`hardwareConcurrency` 为 4，没有 `deviceMemory`，`maxTouchPoints` 为 5；`languages` 只有主语言（Safari 不追加基础语言）。
- 视口：真机采集时等于整屏（地址栏收起）。采用基准值。

## 4. 下发

### 4.1 CDP

- `Emulation.setUserAgentOverride` / `Network.setUserAgentOverride`：只给 UA 与 `platform`，**不带** `userAgentMetadata`。2026-10-17 实测：这样 Chrome 不发 `sec-ch-ua*` 请求头，与 Safari 一致。
- 移动视口、竖屏、触屏模拟：同 M1。
- `Emulation.setEmulatedMedia`：`color-gamut: p3`（实测可以模拟）。`dynamic-range` 不能模拟，列为残余风险。
- 启动参数：无界面时用 iPhone 屏幕的 `--screen-info`；不关 PDF 查看器（Safari 也报 5 个插件、`pdfViewerEnabled` 为 true，与桌面 Chrome 原生相同）；不开 WebBluetooth / NetInfo。

### 4.2 注入脚本（iOS）

- **删掉** Safari 没有的接口：
  - `navigator.userAgentData` 与 `NavigatorUAData`、`deviceMemory`；
  - `navigator` 上的 `bluetooth`、`connection`、`contacts`、`devicePosture`、`getBattery`、`getInstalledRelatedApps`、`ink`、`keyboard`、`presentation`、`scheduling`、`serial`、`setAppBadge`、`usb`、`vibrate`、`virtualKeyboard`、`webkitTemporaryStorage`、`xr`、`hid`、`windowControlsOverlay`、`gpu`（iOS 18.5 没有）；
  - `window` 上的 `BarcodeDetector`、`MediaSource`、`Notification`、`chrome`、`launchQueue`、`showDirectoryPicker`、`showOpenFilePicker`、`webkitRequestFileSystem`、`EyeDropper`、`documentPictureInPicture`、`queryLocalFonts`、`getScreenDetails` 等桌面接口；
  - `MediaDevices.prototype` 上的 `getDisplayMedia`、`setCaptureHandleConfig`。
- **补上** Safari 有、Chrome 没有的：
  - `navigator.standalone`（false）；
  - `ApplePaySession`（`canMakePayments()` 为 false）；
  - `GestureEvent`；
  - `ManagedMediaSource`（`isTypeSupported` 按基准的编解码支持）；
  - `openDatabase`（iOS 18.5 有，26 没有，按机型）。
  - 手写接口沿用 M1 的 `fakeInterface` 与 `strip`。
- **原生函数的 `toString`**：JavaScriptCore 的格式是 `function max() {\n    [native code]\n}`。iOS 身份下，伪装层的 `Function.prototype.toString` 把原生格式统一转为这种格式，包括伪装表里的条目。
- `languages`：只有主语言。
- 权限：`permissions.query({name: 'notifications'})` 为 denied。

### 4.3 WebGL

- 标识：`Apple Inc.` / `Apple GPU`。
- `VERSION`、`SHADING_LANGUAGE_VERSION` 按基准（`WebGL 1.0` 等，不带 Chromium 后缀）。
- 能力参数、精度、扩展：按基准，沿用 M1 的 `mobileWebGL` 机制。

### 4.4 字体（实现者决定：用户自备）

- Apple 字体（苹方、SF 等）受授权限制，不随库分发。做法同 Windows 字体：用户自备，放在 `fonts/apple/`（git 忽略）。
- Linux 上 iOS 身份的进程用进程级 fontconfig，只看得到这些字体：
  - `sans-serif` 为 PingFang SC / Helvetica；
  - `serif` 为 Times New Roman；
  - `monospace` 为 Courier New / Menlo。
- 目录不存在时打印一次警告，回落到 Noto（字体探测会看出不是 iPhone，列为残余风险）。
- 字体对照测试只在提供了 Apple 字体时运行。

### 4.5 输入

- **点按**（iOS 26 真机实测）：
  - 事件顺序：`pointerover` → `pointerenter` → `pointerdown` → `touchstart` → `gotpointercapture` → `pointerup` → `lostpointercapture` → `pointerout` → `pointerleave` → `touchend` → 兼容鼠标事件 → `click`；
  - 按住约 28ms，期间没有 `pointermove`；pointer 的 `pressure` 为 0、宽高 24.16；
  - **`click` 的 `pointerType` 为 `mouse`**（Chrome 为 touch），用注入脚本改写受信 `click` 的 `pointerType`；
  - 屏幕坐标与页面坐标相同（`screenY − clientY` 为 0），触摸、指针、鼠标事件都按此处理；
  - 做法：手指点按，触点半径固定 12.08、`force` 0，按住 25–60ms，不发 `touchMove`。Chrome 由半径推导 pointer 宽高、由 `force` 推导 `pressure`；iOS 的 Touch 对象 `radiusY` 0、`force` 12.08 无法同时得到，列为残余风险。
- **打字**：iOS 键盘的组字方式，`keydown` 为 `key: 'a'`、`code: 'KeyA'`、`keyCode` 229，`keyup` 为 `keyCode` 65（按键码）。复用 M2 的组字过程，只换按键字段。真机用的是拼音键盘，组字文本带 `\u2006` 分隔；这里按拉丁字母组字，列为残余风险。
- **滑动**：同 M2。

## 5. 测试

- 单元测试：
  - iOS 指纹的规范化与机型抽取；
  - UA 与 Safari 版本取自机型；
  - `knownDeviceKind("ios")`；
  - 类型不一致时报错。
- 集成测试（iOS 26 机型）：
  - 请求头里没有 `sec-ch-ua*`，UA 为 Safari；
  - 主页面、跨站 iframe、Worker、弹窗一致：`userAgentData` 不存在，`platform` 为 `iPhone`，`vendor` 为 Apple；
  - 对照基准 `ios-safari-iphone-2.json` 的 `identity`、`screen`、`features`、`media`、`webgl`、`engine.nativeToString`，逐项比较；跳过的项写明原因（引擎、编解码、语音、字体等）；
  - 打字的按键字段与组字序列和基准一致。
- macOS 与 Linux 容器全量通过。

## 6. 残余风险

- JavaScriptCore 与 V8 的差异：报错文本、调用栈格式、数学精度。
- TLS / HTTP2 握手是 Chrome 的。
- 渲染引擎：CSS 特性、表单控件。
- `dynamic-range: high`；HEVC 等编解码；语音数量（真机 68–223 个，服务器 0 个）。
- 没有自备 Apple 字体时，字体探测看得出来。
- 触点的 `radiusY`、`force`、`pressure` 组合（4.5）。
