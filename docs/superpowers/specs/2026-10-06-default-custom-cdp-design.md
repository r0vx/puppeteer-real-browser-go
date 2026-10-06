# 默认走 CustomCDP 通道（子项目 D）设计

状态：已通过（2026-10-06，用户选方案 A，要求一次做完）

## 1. 背景与目标

库有两条连接 Chrome 的通道：
- **CustomCDP**（`UseCustomCDP: true`）：浏览器级连接 + 目标管理器，主页面、跨站 iframe、专用 / 共享 Worker、弹窗都在运行前下发账号身份。生产环境（triumph/browser-service）用的就是它。
- **chromedp**（不写选项时的**默认**）：只给主页面下发身份；跨站 iframe（Cloudflare Turnstile 就在里面）、Worker、弹窗看到的是 `HeadlessChrome`、Linux 的原始值。

2026-10-06 实测更正：chromedp 会调用 `Runtime.enable`，但在 Chrome 154 上，页面里检测不到（`console` 序列化相关的 4 种 getter 陷阱、`console.log` 耗时都与 CustomCDP 无差别）。所以默认通道的真实问题只剩身份覆盖不全。

**目标**：不写任何通道选项时就得到 CustomCDP 的完整身份伪装；chromedp 只作为需要显式选择的旧通道。

调研（2026-10-06）：
- 两条通道的页面公开方法：CustomCDP 包含 chromedp 页面的全部方法，还多出网络监听 / 请求拦截接口。
- 只有一种写法依赖 chromedp：`GetContext()` 拿到 context 后直接调用 chromedp（`chromedp.Run`、`chromedp.ListenTarget`、`network.GetResponseBody(...).Do`）。仓库示例里有 `douyin_network_demo.go`、`feature_test.go` 两处。
- triumph/browser-service 全部 `UseCustomCDP: true`，只用 CustomCDP 接口，没用 `NewPage`；`order-fetch`、`pkg` 只在 go.mod 中依赖。

## 2. 设计

### 2.1 通道选择
- 新增 `ConnectOptions.UseChromedp bool`：为 true 时走 chromedp（旧通道：只给主页面下发身份，会调用 `Runtime.enable`）；默认 false，走 CustomCDP。
- `UseCustomCDP` 字段保留（已有代码继续编译），注释标 `Deprecated:`，不再影响通道选择。
- `RealBrowser.connectToChrome` 按 `UseChromedp` 分支；`Connect(ctx, nil)` 走 CustomCDP。
- `AccountManager`：默认基础配置去掉 `UseCustomCDP: true`；合并账号配置时按 `UseChromedp`（账号设为 true 时覆盖基础配置），不再处理 `UseCustomCDP`。

### 2.2 CustomCDP 实例上新开页面
- `BrowserInstance.CreateBrowserContext()` 记下实例的 CustomCDP 主页面（若是），`BrowserContext.NewPage()` 在这种实例上：
  1. 经同一条浏览器级连接 `Target.createTarget{url: "about:blank"}`；
  2. 目标管理器在新标签页运行前下发账号身份（已有机制），并按 targetId 交出它的会话；
  3. 用这个会话构造 `CustomCDPPage`（启用页面所需的域、代理认证，同主页面）。
- 目标管理器记录已附加页面目标的会话，`Target.detachedFromTarget` 时移除；提供"按 targetId 等待会话"（超时 15 秒报错）。
- 新开页面 `Close()` 只关闭自己的标签页（`Target.closeTarget`），不断开浏览器连接；主页面 `Close()` 行为不变（断开连接，随后由实例关闭 Chrome）。
- chromedp 实例（`UseChromedp: true`）上的 `NewPage()` 行为不变。
- 不再需要"CustomCDP 实例上开 chromedp 页面时复用托管身份"的特殊处理（`ChromeProcess.managedIdentity` 及 `CDPPage.initialize` 中的对应分支删除）；`ChromeProcess.host` 缓存保留（chromedp 通道仍用）。

### 2.3 不做
- 不改造 chromedp，不给 chromedp 通道补 iframe / Worker / 弹窗身份。
- `CustomCDPPage.GetContext()` 仍返回 `context.Background()`（没有 chromedp 上下文），文档写明。
- 不改 triumph 的代码（它已是 `UseCustomCDP: true`，字段废弃但仍可编译、行为不变）。

## 3. 兼容与迁移
- 显式写了 `UseCustomCDP: false` 的代码：现在走 CustomCDP。需要旧行为时改成 `UseChromedp: true`。
- `GetContext()` 后直接调用 chromedp 的代码：仍能编译，但在 CustomCDP 通道上运行会出错。迁移方式：加 `UseChromedp: true`，或改用 CustomCDP 接口（`EnableNetwork` / `OnNetworkRequest` / `OnNetworkResponse` / `GetResponseBody` / `EnableFetch` / `OnRequestPaused`）。
- README、`CLAUDE.md` 改为新的默认值，并写迁移说明。

## 4. 测试
- 不写通道选项的 `Connect`：主页面、跨站 iframe、专用 / 共享 Worker、弹窗的身份一致（`TestIdentityAcrossSurfaces` 去掉 `UseCustomCDP`，即测默认通道）。
- 通道选择：默认 → `*CustomCDPPage`；`UseCustomCDP: false` → 仍是 `*CustomCDPPage`；`UseChromedp: true` → `*CDPPage`。
- 现有"两条通道各跑一遍"的测试改用 `UseChromedp` 区分，chromedp 通道继续有覆盖。
- `AccountManager` 合并：账号 `UseChromedp: true` 覆盖基础配置。
- CustomCDP 实例 `NewPage()`：新页是 `*CustomCDPPage`，身份为账号身份（用 Windows 身份，任何宿主上都能看出差别）；新页 `Close()` 后主页面仍可用；实例 `Close()` 正常。
- 所有示例、cmd 工具可编译；macOS 与 Linux 容器全量通过。

## 5. 风险
- 依赖"`GetContext()` + chromedp"写法的未知调用方会在运行时出错（编译不报错）。缓解：README 迁移说明；`UseCustomCDP` 标 Deprecated 后，静态检查会提示。
- chromedp 通道定位为旧通道，不再承诺防检测。
