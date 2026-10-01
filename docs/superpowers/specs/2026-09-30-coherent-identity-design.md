# 账号身份自洽（反检测子项目 1）设计

日期：2026-09-30　状态：已通过（2026-09-30）　分支：`feat/coherent-identity`

## 1. 背景与目标

生产配置：**无界面 Linux 服务器 + Google Chrome 稳定版（无 GPU）+ `UseCustomCDP` + `FingerprintUserID`**，每个账号绑定一份指纹（核心是 UA），账号对外是 Windows 用户或 Mac 用户。

目标：每个账号对外呈现**一个前后一致的 Windows / macOS Chrome 身份**——HTTP 请求头、主页面、跨站 iframe、Worker、弹窗给出的答案相同，且页面检测不到注入痕迹。

本子项目覆盖原清单 7（无界面泄露）、9（指纹不自洽）及 8（注入痕迹）中与生产路径相关的部分。chromedp 路径（`UseCustomCDP:false`）的 Runtime.enable 问题（10）属于子项目 D，不在本次范围。

## 2. 现状实测（Linux + Google Chrome 154，Docker 模拟生产）

同一账号在不同出口给出矛盾答案（`probe-user-1`）：

| 检查点 | 现状 | 真实 Chrome |
|---|---|---|
| JS `navigator.userAgent` | Chrome/128，`Windows NT 10.0; Win64; x64 10.0` | 版本 = 内核版本，格式无 ` 10.0` 尾巴 |
| 请求头 `Sec-CH-UA*`、`navigator.userAgentData` | 全空（`brands: []`） | 必有，且与 UA 一致 |
| JS 语言 / `Accept-Language` | ja-JP / en-US | 一致 |
| JS 时区 / `Date.toString()` | 圣保罗（偏移符号还反了）/ 中国时间 | 一致 |
| 屏幕 / 窗口 | 3840×1600 / 780×493 | 合理比例 |
| WebGL | `…D3D11) (Build 26453)`、`MAX_TEXTURE_SIZE 18949` | 真实型号与参数 |
| 被改写函数 `toString` | 13 个露出 JS 源码 | 全部 `[native code]` |
| 页面可读标记 | `__stealthInjected`、`_screenFixed`、`Date.name==='ModifiedDate'`、navigator 自有属性 11 个、假 `RTCPeerConnection` | 无 |

无界面本身只泄露三处：UA 中的 `HeadlessChrome`（请求头 + JS）、800×600 屏幕、WebGL 渲染器 `SwiftShader`。

覆盖范围实测（只在主页面设置 CDP 覆盖时）：跨站 iframe 的 JS 值全是原始无界面身份（Cloudflare Turnstile 就运行在这种 iframe 里）；弹窗请求头与 JS 都是原始值；Worker 的 `platform` / `languages` / `hardwareConcurrency` 未覆盖。现有注入脚本同样到不了跨站 iframe 和弹窗。

## 3. 方案概述

身份信息改由 **Chrome 自带的 CDP Emulation 接口**下发，浏览器内部真正切换身份，请求头 / JS / client hints / 时间与语言格式天然一致且不留 JS 痕迹；JS 层只保留 CDP 做不到的少数项，并统一伪装为原生函数。CustomCDP 改为浏览器级连接，对所有页面、跨站 iframe、Worker 在其执行任何代码前下发身份。

已验证可行（探针，Chrome 154）：浏览器级 `Target.setAutoAttach{autoAttach, waitForDebuggerOnStart, flatten}` 后，在每个目标暂停期间下发覆盖再 `Runtime.runIfWaitingForDebugger`，主页面、跨站 iframe、Worker、弹窗的 UA / client hints / platform / 语言 / 时区 / 核数全部一致。

## 4. 设计

### 4.1 身份档案（Identity）

由账号的指纹配置（`FingerprintConfig`）加上真实浏览器版本派生，是下发到浏览器的唯一数据源。

- **系统**：由账号绑定的 UA 判定——含 `Windows NT` → Windows，含 `Macintosh` → macOS。首次创建指纹时 UA 取 `ConnectOptions.UserAgent`；未提供时默认 Windows。
- **Chrome 版本**：始终改为真实浏览器的版本（读 `/json/version`）。UA 保持 Chrome 的精简格式：主版本号 + `.0.0.0`，例如 `Chrome/154.0.0.0`；完整版本只出现在 `fullVersionList` 里。理由：UA 版本与内核不一致可被特性检测识别；真实用户自动升级时 UA 版本本就会变。
- **由系统派生**：

| 字段 | Windows | macOS |
|---|---|---|
| UA 系统段 | `Windows NT 10.0; Win64; x64` | `Macintosh; Intel Mac OS X 10_15_7`（Chrome 冻结值） |
| `navigator.platform` | `Win32` | `MacIntel` |
| client hints `platform` / `platformVersion` | `Windows` / `10.0.0`（Win10）或 `15.0.0`、`19.0.0`（Win11） | `macOS` / `13.x`–`15.x` |
| `architecture` / `bitness` | `x86` / `64` | Apple 芯片 `arm`、Intel `x86` / `64` |
| WebGL 显卡池 | Intel / NVIDIA / AMD 的 D3D11 ANGLE 串及对应参数 | Apple M 系列 Metal 串（arm）或 Intel/AMD（x86） |
| DPR / 屏幕池 | 1、1.25、1.5；常见 Windows 分辨率 | 2；MacBook / iMac 常见分辨率 |

- **池内取值**：Win10 还是 Win11、具体的显卡和分辨率等，在池内按用户 ID 的哈希确定性选取，首次生成后写入指纹文件固定下来，之后启动不再重新抽取。
- **品牌**：`brands` / `fullVersionList` 按真实浏览器的品牌顺序和 GREASE 值生成：先在初始页面读取真实 `navigator.userAgentData`，再替换版本号，不自己编造。
- **语言**：`Language`、`Languages` 一并设置，`Accept-Language` 由同一份列表生成（Chrome 会自动补 q 值）。时区未指定时由语言推导，偏移量不再存储，由浏览器根据时区计算（这样能正确处理夏令时）。
- **其他**：`hardwareConcurrency` 来自档案；`deviceMemory` 保持原生值（Chrome 上限为 8，不伪造）。

### 4.2 目标管理（CustomCDP）

- `CustomCDPClient` 改为连接**浏览器级 WebSocket**（`/json/version` 中的 `webSocketDebuggerUrl`），消息带 `sessionId`（flatten 模式），按会话分发响应和事件。
- 启动时在浏览器会话上执行 `Target.setAutoAttach{autoAttach:true, waitForDebuggerOnStart:true, flatten:true}`，过滤掉 `browser` / `tab` 类型。主页面选第一个 `page` 目标（沿用现有筛选规则），通过自动附加得到它的会话。
- 每个新附加的目标都按同一流程处理：按类型下发身份（见 4.3）→ 注册初始化脚本 → 在该会话上开启 `setAutoAttach` 以接管它的子目标 → `Runtime.runIfWaitingForDebugger` 放行。
- `CustomCDPPage` 的公开方法和行为保持不变，内部的 `sendCommand` 改为带上主页面的会话 ID。弹窗只做身份下发，不作为 Page 对象暴露（与现状一致）。
- 全程不调用 `Runtime.enable`；`Runtime.runIfWaitingForDebugger` 不会开启 Runtime 域。

### 4.3 身份下发矩阵

| 目标类型 | 下发内容 |
|---|---|
| page | `Emulation.setUserAgentOverride`（UA、`platform`、`acceptLanguage`、`userAgentMetadata`）、`setTimezoneOverride`、`setLocaleOverride`、`setHardwareConcurrencyOverride`、`setDeviceMetricsOverride`（视口 + 屏幕尺寸 + DPR）、`Page.enable` + `addScriptToEvaluateOnNewDocument` |
| iframe（跨站） | 同 page；`setDeviceMetricsOverride` 只设置屏幕尺寸和 DPR，不改视口（需实测确认可行） |
| worker / shared_worker / service_worker | `Network.setUserAgentOverride`（UA + client hints）、`Emulation.setTimezoneOverride`；`platform` / `languages` / `hardwareConcurrency` CDP 覆盖不到，用 4.4 的原生伪装工具在 `WorkerNavigator.prototype` 上补齐（探针已验证注入早于 Worker 脚本执行） |

窗口外框尺寸（`outerWidth` / `outerHeight`）与 `screenX` / `screenY` 要与新的视口和屏幕相符，具体做法在实现阶段实测后确定（候选：按身份设置 `--window-size`，或使用 `Browser.setWindowBounds`）。

### 4.4 JS 层

**只保留以下几类**：
1. MouseEvent 的 `screenX` / `screenY` 修正（原版 puppeteer-real-browser 的核心项）。
2. WebGL：`getParameter` 返回档案中的 vendor / renderer 和对应的能力参数（WebGL1 和 WebGL2），遮掉 SwiftShader。
3. canvas、音频：按账号固定的确定性噪声（沿用已修复的音频实现，canvas 同理）。

**原生伪装工具**（所有改写统一走它）：
- 改在原型上，不在实例上新建自有属性；
- 替换后的函数保持原有的 `name`、`length`，`toString` 返回 `function x() { [native code] }`（通过统一的 `Function.prototype.toString` 代理实现，代理自身也显示为原生）；
- 不留任何全局或原型标记；
- 防重复注入用闭包内的 WeakSet，不挂到 window 上。

**删除**：默认 advanced 脚本和指纹脚本里的其余改写，包括 navigator 各属性、语言、插件、webdriver（原生值已是 false）、`permissions`、`chrome.runtime`、console 过滤、`Function.prototype.toString` 补丁、`createElement` 钩子、Date / `performance` 时间偏移、JS 时区、电池、`connection`、`mediaDevices`、假 `RTCPeerConnection`。

**WebRTC**：不再用假对象。改为启动参数 `--force-webrtc-ip-handling-policy=disable_non_proxied_udp`，在浏览器原生层面阻止绕过代理泄露真实 IP，页面看到的是真实的 `RTCPeerConnection`。

### 4.5 已保存指纹的兼容

加载旧 JSON 时规范化，修正后回写：
- UA：修正 `x64 10.0` 等错误格式，版本按 4.1 规则重写；UA 为 Linux 的账号按 Windows 处理（需确认，见第 8 节）；
- WebGL：去掉不可能的取值（`(Build N)` 后缀、加噪的能力参数）。显卡厂商系列尽量保留，若与系统冲突（如 Windows 账号配了 Apple 显卡），在同系统同厂商的池里重选；
- 屏幕、语言、时区、核数尽量保留；只修正不合理的值（如窗口比屏幕大）；
- 新增字段 `schema_version`，按版本迁移；不认识的旧字段保留不删。

### 4.6 不设指纹的默认路径（CustomCDP）

使用真实浏览器的身份，只去掉无界面痕迹：UA 和 client hints 用真实值，把 `HeadlessChrome` 替换为 `Chrome`；屏幕使用常见分辨率；JS 只保留 MouseEvent 修正和 WebGL 遮挡（遮掉 SwiftShader）。同样经过目标管理，所以对 iframe、Worker、弹窗同样生效。

### 4.7 运维

- 提供生产镜像的 Dockerfile 片段：Google Chrome 稳定版 + Windows 常用字体（雅黑、宋体、Arial、Segoe UI 等，需自备授权字体文件）+ `--init`。
- 文档注明：Mac 身份无法在 Linux 上补齐 Apple 字体（授权限制），字体探测会比 Windows 身份弱。

## 5. 对外接口

- `Connect`、`ConnectOptions`、`Page`、`PageWithSelector` 不变；`FingerprintUserID` / `FingerprintDir` / `UserAgent` / `Language` / `Timezone` 等字段的含义不变（`TimezoneOffset` 不再生效，改为由时区计算，字段保留并标注 deprecated）。
- `GetAdvancedStealthScript` 等导出函数保留名字，内容随新的 JS 层变化；不删除导出符号（删除死代码是另一项工作）。
- `NewCustomCDPClient(debugURL)` 保留签名，内部改为浏览器级连接。

## 6. 错误处理

- 目标附加后下发身份失败：记录到连接级错误回调，并照常放行该目标，避免页面卡在暂停状态。主页面下发失败时 `Connect` 返回错误。
- `Target.setAutoAttach` 不可用（极老版本 Chrome）时，`Connect` 返回明确错误，不静默降级。
- 浏览器版本读取失败时 `Connect` 返回错误（版本是身份一致性的前提）。

## 7. 测试与验收

在 Linux + Google Chrome 容器（模拟生产）和 macOS 本机上运行：

1. **一致性**：对 Windows 账号和 Mac 账号，分别在请求头、主页面、跨站 iframe、Worker（专用、共享）、弹窗中检查 UA、client hints、`platform`、语言与 `Accept-Language`、时区与 `Date.toString`、locale、核数、屏幕、DPR，各处必须完全一致且符合档案。
2. **无痕迹**：被改写函数的 `toString` 全部为 `[native code]`；navigator 上没有自有属性；没有全局或原型标记；构造函数的 `name` 正确；`RTCPeerConnection` 为原生。
3. **无界面**：任何出口都不出现 `HeadlessChrome`、`SwiftShader`、800×600。
4. **兼容**：现有全量测试继续通过；旧指纹文件（包括 Linux UA、`x64 10.0` 等坏数据）能加载并被修正。
5. **稳定性**：同一账号多次启动，指纹完全相同（canvas、音频、WebGL 的值和噪声都固定）。

外部检测页（CreepJS、browserleaks 等）只作人工参考，不纳入自动化测试。

## 8. 风险与待确认

- **已确认（2026-09-30）**：UA 版本跟随真实浏览器（第 4.1 节）。
- **已确认（2026-09-30）**：已保存的 Linux UA 账号按 Windows 处理（本地 42 份指纹中有 21 份是 Linux）。理由：
  - Windows 字体能在服务器上补齐，Mac 字体不能；
  - 服务器 CPU 是 x86-64，与 Windows 的 `x86` 一致，而 Mac 主流是 `arm`；
  - 这些账号现有的屏幕和 DPR 1 本来就符合 Windows 用户；
  - 目标站的桌面用户以 Windows 为主。
- **已知残余风险**：
  - 软件渲染（SwiftShader）出来的 canvas / WebGL 图像特征和真实显卡不同；
  - 语音列表 `speechSynthesis.getVoices` 与 Windows / Mac 不符；
  - Mac 身份的字体不全；
  - 不走代理时，服务器的 TCP / IP 指纹是 Linux。
  
  以上不在本次范围，后续可以单独评估。
- **范围外**：chromedp 路径（子项目 D）；删除死代码。
