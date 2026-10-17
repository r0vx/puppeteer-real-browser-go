# 安卓身份（移动身份子项目 M1）设计

状态：已定稿（2026-10-17）。用户指示"完成所有 M 的任务，中途不要停下来问"，文中标注"实现者决定"的选择题均由实现者按推荐项自行决定。

## 1. 背景

整体决策与真机基准见 `docs/superpowers/specs/2026-10-17-mobile-baseline-collector-design.md`（M0）。

- M1 在现有身份流程里增加安卓（Android Chrome）身份：
  - 主页面、跨站 iframe、Worker、弹窗都在页面脚本运行前下发；
  - 生产环境仍是无界面 Linux 上的桌面版 Google Chrome。
- 真机基准：`test/baseline/android-chrome-pgkm10.json`。机型为一加 Ace（OnePlus Ace，`PGKM10`，天玑 8100-Max，GPU Mali-G610 MC6），系统 Android 15，浏览器 Chrome 138。

2026-10-17 对照实测：桌面 Chrome 155 只开 CDP 移动模拟、不加本库伪装时，与真机的差异分三类。

**1. CDP 就能改的**：UA、Client Hints（`formFactors`、`bitness`、`architecture`、`model`）、视口、触屏、横竖屏。

**2. 需要额外处理的**：
- `navigator.plugins` 为 5（真机 0）、`mimeTypes` 为 2（真机 0）、`pdfViewerEnabled` 为 true（真机 false）。
- `new Notification()` 能构造。真机抛出 `TypeError: Failed to construct 'Notification': Illegal constructor. Use ServiceWorkerRegistration.showNotification() instead.`
- 真机有、模拟没有：`navigator.contacts`、`navigator.connection.type`（真机为 `wifi`）。
- 模拟有、真机没有：
  - `navigator.hid`、`navigator.windowControlsOverlay`；
  - `window.EyeDropper`、`window.SharedWorker`、`window.documentPictureInPicture`、`window.queryLocalFonts`；
  - `window.chrome.app`。
- `navigator.deviceMemory`、`hardwareConcurrency` 是宿主的值。

**3. WebGL**：
- 模拟多出 21 个桌面扩展。
- 能力参数差 12 项。例如 `MAX_TEXTURE_SIZE` 为 16384（真机 4096），`MAX_VERTEX_ATTRIBS` 为 16（真机 32），`SUBPIXEL_BITS` 为 4（真机 8）。
- 着色器 low / medium 精度是桌面的 `[127,127,23]`（浮点）和 `[31,30,0]`（整数），真机为 `[15,15,10]` 和 `[15,14,0]`。

另外两项差异：
- **字体**：模拟能认出 34 个桌面字体，真机只认出 9 个名字，都是安卓字体别名。
- **`Math.expm1(1)`**：末位不同。原因可能是 CPU 架构，也可能是版本，见第 7 节。

## 2. 目标

1. 调用方可以为账号开启安卓身份。同一账号每次启动都是同一台"手机"。
2. 第 1 节列出的差异，凡是本库能控制的，都与真机一致。对照测试以真机基准为准；不能控制的列为残余风险（第 9 节）。
3. 已有账号的身份不受影响：不改系统，不改任何值。

## 3. 开启方式与兼容（实现者决定：显式开关）

- `ConnectOptions` 新增 `Mobile bool`。只有设了 `Mobile: true`，新账号才生成移动身份；系统由 UA 决定：iPhone / iPad UA 为 iOS（M3），其余为安卓。
- 指纹文件新增 `device` 段（见 4.2）。有 `device` 段的才是移动指纹。
  - 已有的指纹都没有这一段，所以永远按原规则（Windows / macOS）规范化。
  - 即使原始数据里的 UA 是手机 UA，也不会变成安卓。这正是 triumph PC 流程依赖的行为：PC 流程按 `user_agents` 表的 UA 建指纹，表里有手机 UA。
- 防误用：
  - `Mobile: true` 加载到一份已有的桌面指纹，`Connect` 报错（`fingerprint X is a desktop identity`）；
  - `Mobile: false` 加载到一份移动指纹，也报错；
  - 都不改写文件。
- 移动身份只支持默认的 CustomCDP 通道。`UseChromedp: true` 与 `Mobile: true` 同时设置时，`Connect` 报错。
- `Mobile: true` 但没有设 `FingerprintUserID` 时报错：移动身份必须有账号指纹。
- `Width` / `Height` 选项对移动身份无效：屏幕由机型决定。

## 4. 数据

### 4.1 机型档案池

GPU 统一用 Mali-G610 MC6（实现者决定）。

- 真机基准只有一款 GPU。WebGL 的扩展列表、驱动相关参数靠公开资料拼凑容易出错。例如真机 `MAX_TEXTURE_SIZE` 只有 4096，像是 Chrome 针对该 GPU 设的限制，公开资料里查不到。
- 因此整理的机型都选同 GPU 的天玑 8100 / 8200 手机，WebGL 全部取真机实测值；机型、屏幕、系统版本各不相同。
- 以后采到高通等其他 GPU 的真机数据，再扩充。

| 机型 | `model` | SoC | 物理分辨率 | DPR | CSS 屏幕 | Android | 来源 | 权重 |
|---|---|---|---|---|---|---|---|---|
| 一加 Ace | `PGKM10` | 天玑 8100-Max | 1080×2412 | 3 | 360×804 | 15 | 真机实测 | 30 |
| Redmi K50 | `22041211AC` | 天玑 8100 | 1440×3200 | 3.5 | 411×914 | 14 | 整理 | 14 |
| Redmi Note 11T Pro | `22041216C` | 天玑 8100 | 1080×2460 | 2.75 | 393×895 | 14 | 整理 | 14 |
| realme GT Neo3 | `RMX3560` | 天玑 8100 | 1080×2412 | 3 | 360×804 | 14 | 整理 | 14 |
| Xiaomi Civi 3 | `23046PNC9C` | 天玑 8200-Ultra | 1080×2400 | 2.75 | 393×873 | 15 | 整理 | 14 |
| Redmi K60E | `22122RK93C` | 天玑 8200 | 1440×3200 | 3.5 | 411×914 | 15 | 整理 | 14 |

- 整理来的机型：
  - `model` 与 SoC 取自厂商公开信息；
  - DPR 按厂商默认显示密度：小米、红米的 FHD 屏为 440dpi（2.75），2K 屏为 560dpi（3.5）；一加、realme 为 480dpi（3）；
  - CSS 屏幕 = 物理分辨率 ÷ DPR，四舍五入。
- 所有机型共用的值：
  - `navigator.platform` 为 `Linux armv81`（真机实测）；
  - `hardwareConcurrency` 为 8；
  - `deviceMemory` 为 8（Chrome 把内存上报值封顶为 8）；
  - `maxTouchPoints` 为 5。
- 视口：`innerHeight` = 屏幕高 − 104。104 是真机实测的状态栏、地址栏、手势条总高度（CSS 像素即 dp，所以各机型相同）。`innerWidth` = 屏幕宽，`outerWidth`、`outerHeight` 等于视口。
- `screen` 的 `avail*` 等于屏幕，`availTop` 为 0。方向为竖屏 `portrait-primary`，`window.orientation` 为 0。
- 每个账号按用户 ID 确定性地抽一款机型（`identitySeed(userID, "device")`），之后固定不变。

### 4.2 指纹文件

新增 `device` 段：

```json
"device": {"kind": "android", "model": "PGKM10", "device_memory": 8}
```

`Normalize` 遇到 `kind: "android"` 时走安卓分支：
- `model` 不在池中时，按用户 ID 重抽一款；然后用该机型的数据覆盖屏幕、显卡、系统版本、核数。
- UA 为 Chrome 精简格式的安卓 UA：`Mozilla/5.0 (Linux; Android 10; K) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/<主版本>.0.0.0 Mobile Safari/537.36`。
  - 安卓 UA 里的系统段被 Chrome 冻结为 `Android 10; K`，不带真实机型。
  - 真实机型只在 Client Hints 的 `model` 里出现。
- `Browser.Platform` 为 `Linux armv81`；`MaxTouchPoints` 为 5；语言、时区规则与桌面相同。
- `WebGL.Vendor` / `Renderer` 是 `UNMASKED_*` 的值：`ARM` / `Mali-G610 MC6`。
- 新账号（`Mobile: true` 且没有文件）：`generateFingerprintFor(userID, OSAndroid)` 生成，`device.kind` 为 `android`。

`osFromUserAgent` 不变，仍只区分 Windows / macOS。移动身份的系统由 `device.kind` 决定，与 UA 判定互不干扰。

## 5. 下发

### 5.1 身份（`Identity`）

`OSFamily` 新增 `OSAndroid`。`identityFromConfig` 的安卓分支：
- UA 为 4.2 的安卓 UA，主版本跟随真实浏览器；
- `Platform` 为 `Linux armv81`；
- Client Hints：
  - `platform` 为 `Android`，`platformVersion` 为机型的版本（如 `15.0.0`），`model` 为机型代号；
  - `mobile` 为 true，`formFactors` 为 `["Mobile"]`；
  - `architecture` 与 `bitness` 为空字符串，`wow64` 为 false；
  - 品牌取自真实浏览器（同桌面）。
- 屏幕几何为 4.1 的值，`Mobile` 标记为 true。
- GPU 为 Mali-G610 MC6 档案，含扩展与精度（5.4）。
- 内存 8，电池见 5.3。

### 5.2 CDP（页面与 iframe 在运行前，Worker 照旧）

- 页面：
  - `Emulation.setDeviceMetricsOverride`：`mobile: true`；宽高为视口；`deviceScaleFactor` 为 DPR；`screenWidth/Height` 为屏幕；`screenOrientation` 为 `{type: portraitPrimary, angle: 0}`。
  - `Emulation.setTouchEmulationEnabled {enabled: true, maxTouchPoints: 5}`。
- 跨站 iframe：也开启触屏模拟（`maxTouchPoints`、`ontouchstart`、`pointer: coarse` 都按目标生效）。屏幕与 DPR 仍由注入脚本补（现有机制）。
- `Emulation.setUserAgentOverride` / `Network.setUserAgentOverride` 带安卓元数据，`platform` 为 `Linux armv81`。时区、语言、核数沿用现有步骤。
- 改视口（`SetViewport`）时保留 `mobile: true` 与屏幕。

### 5.3 注入脚本（`identity_script.go`，只在安卓身份下启用）

- `navigator.deviceMemory` 为 8（页面与 Worker）。
- `Notification`：
  - `new Notification(…)` 抛出真机那段 `TypeError`；不加 `new` 直接调用时保持原生报错；
  - `Notification.permission`、`requestPermission`、`prototype`、`maxActions` 等静态成员保持原生；
  - 函数的 `toString` 显示为原生。
- `navigator.contacts`：
  - 提供 `ContactsManager` 接口（`window.ContactsManager`）与 `navigator.contacts` 实例；
  - `getProperties()` 返回 `['address', 'email', 'icon', 'name', 'tel']`；
  - `select()` 在没有用户手势时与真机一样拒绝（`SecurityError`）。
- `navigator.connection.type` 为 `wifi`。
- 删除桌面独有接口：
  - `Navigator.prototype` 上的 `hid`、`windowControlsOverlay`；
  - `window` 上的 `EyeDropper`、`SharedWorker`、`documentPictureInPicture`、`queryLocalFonts`；
  - 接口对象 `HID`、`HIDDevice`、`HIDConnectionEvent`、`HIDInputReportEvent`、`WindowControlsOverlay`、`WindowControlsOverlayGeometryChangeEvent`、`DocumentPictureInPicture`、`DocumentPictureInPictureEvent`、`FontData`；
  - `window.chrome.app`。
- 电池：
  - `getBattery()` 的 `level` 按账号种子固定在 0.35–0.95，`charging` 为 false；
  - `chargingTime`、`dischargingTime` 都为 `Infinity`（时间未知；真机基准没有采集这两项，按安卓 Chrome 的常见取值）。
  - 宿主是服务器，没有电池，原生恒为"100%、在充电"，每个手机账号都一样，属于聚类特征。

### 5.4 WebGL（安卓身份）

- `getParameter` 覆盖表扩展为任意 GLenum。WebGL1 与 WebGL2 分别取真机的能力参数（第 1 节中的 22 项，加上 WebGL2 的 19 项）。
- `getSupportedExtensions()` 返回真机列表（WebGL1、WebGL2 各一份）。
- `getExtension(name)`：
  - 不在列表中的返回 null；
  - 在列表中、宿主也有的，返回原生对象；
  - 在列表中、宿主没有的，返回 null。比如 ASTC / ETC 压缩纹理：无法真的支持，返回 null 比返回假对象更安全。这一点列为残余风险。
- `getShaderPrecisionFormat` 返回真机的 12 组精度（通过 `WebGLShaderPrecisionFormat.prototype` 的取值器按对象替换）。
- `VENDOR` / `RENDERER` 为 `WebKit` / `WebKit WebGL`（与真机一致）。`UNMASKED_*` 为 `ARM` / `Mali-G610 MC6`。

### 5.5 启动时（安卓身份）

- 无界面时，`--screen-info` 与 `--force-device-scale-factor` 用手机屏幕（物理像素与 DPR）。所有 frame 原生报告手机屏幕。
- profile 偏好写入 `plugins.always_open_pdf_externally: true`。2026-10-17 在 Chrome 155 上实测：这样 `navigator.plugins`、`mimeTypes` 原生为空，`pdfViewerEnabled` 原生为 false，`plugins[0]` 为 undefined，不需要 JS 伪装。
- 字体见第 6 节。
- 为此，`Launch` 在设了 `FingerprintUserID` 时总是先加载指纹（现在只在无界面时加载）。

## 6. 字体（实现者决定：按进程配置）

- 安卓身份的 Chrome 进程用单独的 fontconfig 配置：启动时生成到临时目录，通过 `FONTCONFIG_FILE` 环境变量指定。桌面身份不受影响，仍用系统配置（生产镜像里的 Windows 字体）。
- 配置内容对齐安卓 `fonts.xml`：
  - `sans-serif` 为 Roboto；`serif` 为 Noto Serif；`monospace` 为 Noto Sans Mono（安卓用 Droid Sans Mono，Debian 不提供，见第 9 节）。
  - 中文回退 Noto Sans CJK SC；表情 Noto Color Emoji。
  - 别名：Arial、Helvetica、Tahoma、Verdana 指向 sans-serif；Times、Times New Roman、Georgia 指向 serif；Courier、Courier New、Monaco 指向 monospace。
  - 只扫描上述字体所在的目录，其他字体对该进程不可见。
- 字体来源：
  - Debian 包 `fonts-roboto`、`fonts-noto-core`、`fonts-noto-cjk`、`fonts-noto-color-emoji fonts-noto-mono`。全部为 Apache 或 OFL 许可，可以随镜像分发。
  - 生产镜像（`docs/deploy/production.Dockerfile`）与测试镜像（`test/linux-chrome.Dockerfile`）都安装这些包。
- 只对 Linux 生效。macOS 开发机由 CoreText 管理字体，不受控制，字体相关测试只在 Linux 容器中跑。
- 找不到 Roboto 时：启动不失败，但打印一次警告。缺字体时页面能看出来，但不该让浏览器起不来。

## 7. Chrome 版本差（138 对 155）

真机基准是 Chrome 138，宿主是 Chrome 155，相差 17 个大版本。处理原则：

1. **以真机为准的项目**：同一版本里桌面与安卓本来就不同的项目，即第 1 节列出的那些。我们按真机值伪装。
2. **只删已知的桌面独有接口**：5.3 删除的接口都是 Chrome 只在桌面提供的（设备、窗口控件、取色器、画中画文档、本地字体、`SharedWorker`）。
   - 宿主 155 上有、基准 138 上没有的其他接口，可能是 139–155 新加的跨平台接口，删掉反而会和真机 155 不一致。
   - 所以一律保留，并在对照测试的"版本差"清单里逐项列出、写明原因。
3. **引擎细节**：`Math.expm1` 等引擎级差异无法判断原因（ARM 对 x86，或 V8 版本），不做伪装，列为残余风险。
4. **UA 与 Client Hints 的版本号**：始终跟随宿主（155），不用基准的 138。这是现有"版本跟随真实浏览器"的规则。
5. **更新基准**：用户把手机上的 Chrome 升级后，用 `scripts/collect-baseline.sh` 重采。对照测试读的就是基准文件，届时"版本差"清单应能缩短。

## 8. 测试

- 单元测试：
  - 安卓 `Normalize`：机型不在池中时重抽；同一用户 ID 结果固定；屏幕、GPU、UA、Client Hints 字段与机型一致；对已规范化的配置幂等。
  - 带手机 UA 的旧桌面文件仍规范化为 Windows。
  - `Mobile` 与文件类型不一致时报错。
  - 安卓 UA 格式。
- 集成测试（默认通道）：
  - **跨表面一致**：主页面、跨站 iframe、专用 Worker、弹窗里的 `navigator.platform`、`userAgentData`（含高熵值）、`deviceMemory`、`hardwareConcurrency`、UA 一致，且都是安卓值。iframe 里 `maxTouchPoints` 为 5，`matchMedia('(pointer: coarse)')` 为真。
  - **对照真机基准**：用 `PGKM10` 机型的账号加载采集页，调用 `collectBaseline()`，与 `android-chrome-pgkm10.json` 比较：
    - `identity`：除 UA 与品牌版本号外的全部字段；
    - `screen`：除 `visualViewport.height` 外（真机采集时地址栏状态不同）；
    - `features.present`：除"版本差"清单外的全部；
    - `features` 中的插件、MIME、PDF、Notification、`connection.type`、`chromeKeys`；
    - `media` 中的 `pointer`、`hover`、`any-pointer`、`any-hover`；
    - `webgl` 与 `webgl2`：厂商、渲染器、能力参数、精度、扩展；
    - `fonts`（只在 Linux 上比较）。
  - **报错路径**：`Mobile` 加 `UseChromedp`、`Mobile` 且没有 `FingerprintUserID`、`Mobile` 加载桌面文件，都返回错误。
  - **启动**：安卓身份启动后 `navigator.plugins.length` 为 0。
  - **SetViewport**：改视口后仍是移动视口，屏幕不变。
- macOS 与 Linux 容器全量通过。

## 9. 残余风险（不处理，写入 README）

- **触摸与输入**：M1 之后，在手机身份上调用 `Click`、`RealSendKeys` 仍是鼠标和桌面键盘事件，M2 才改。M2 完成前，不要用手机身份在页面上操作。
- **WebGL 计算精度**：着色器精度只改了查询结果，实际渲染仍按 32 位浮点计算。用着色器实测 mediump 精度的检测能识破。ASTC / ETC 压缩纹理扩展在列表里，但取不到对象。
- **`dynamic-range: high`**：真机屏幕支持高动态范围，宿主不支持，媒体查询无法原生模拟，不伪装。
- **引擎细节**：`Math.expm1` 等（第 7 节）。
- **系统字体的字形**：一加用的是 OPPO Sans 等厂商字体，我们用 Roboto，文字宽度可能略有不同；等宽字体用 Noto Sans Mono 而不是 Droid Sans Mono。
- **整理来的机型**：DPR 按厂商默认显示密度推算；用户改过显示大小的真机会不同，这是正常的个体差异。
- **TLS / HTTP2**：安卓 Chrome 与桌面 Chrome 同用 BoringSSL，握手基本一致；没有逐字节核对过。
- **群体聚类**：所有安卓账号都是 Mali-G610 MC6、8 核、8 GB，在风控的群体统计里是同一类设备；机型、屏幕、系统版本、电量各不相同。采到更多 GPU 的真机数据后再扩充。
- **蓝牙可用性**：`navigator.bluetooth.getAvailability()` 在服务器上为 false（没有蓝牙适配器），真机通常为 true。
- **HEVC 解码**：真机靠硬件解码支持 H.265，宿主没有，不谎报支持（谎报会让网站推 H.265 视频而播不了）。
- **原生取值器里的报错栈**：被替换的原生方法 / 取值器在原生抛错时，调用栈多一帧注入代码（桌面身份同样如此，已有设计）；手写的新接口（Notification、ContactsManager、BarcodeDetector）已去掉这一帧。

## 10. 不做

- 触摸点按、滑动、输入法式打字（M2）。
- iOS 身份（M3）。
- chromedp 通道的移动身份。
- 高通 Adreno、联发科 Immortalis 等其他 GPU（需要真机数据）。
- triumph 侧的改动。
