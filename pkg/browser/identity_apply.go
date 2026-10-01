package browser

import (
	"encoding/json"
	"fmt"
	"strings"
)

// cdpCaller 在某个目标会话上发送一条 CDP 命令
type cdpCaller func(method string, params any) (json.RawMessage, error)

// targetKind 需要下发身份的目标类型
type targetKind int

const (
	kindPage targetKind = iota
	kindIframe
	kindWorker
)

// targetKindOf 把 TargetInfo.type 映射为下发类型；其余类型（浏览器、扩展后台等）不处理
func targetKindOf(t string) (targetKind, bool) {
	switch t {
	case "page":
		return kindPage, true
	case "iframe":
		return kindIframe, true
	case "worker", "shared_worker", "service_worker":
		return kindWorker, true
	}
	return 0, false
}

// hostInfoJS 在未覆盖身份的页面上读取真实浏览器的值（初始标签页 chrome://newtab 或 about:blank 都是安全上下文，可读 userAgentData）
const hostInfoJS = `(async () => {
  const d = navigator.userAgentData;
  const h = await d.getHighEntropyValues(['architecture', 'bitness', 'model', 'platformVersion', 'fullVersionList', 'wow64', 'formFactors']);
  const g = document.createElement('canvas').getContext('webgl');
  let webglVendor = '', webglRenderer = '';
  if (g) {
    g.getExtension('WEBGL_debug_renderer_info');
    webglVendor = g.getParameter(37445) || '';
    webglRenderer = g.getParameter(37446) || '';
  }
  return JSON.stringify({
    userAgent: navigator.userAgent, platform: navigator.platform, languages: navigator.languages,
    timezone: Intl.DateTimeFormat().resolvedOptions().timeZone, hardwareConcurrency: navigator.hardwareConcurrency,
    screen: [screen.width, screen.height, devicePixelRatio],
    brands: d.brands, uaPlatform: d.platform, mobile: d.mobile,
    fullVersionList: h.fullVersionList, platformVersion: h.platformVersion, architecture: h.architecture,
    bitness: h.bitness, model: h.model, wow64: h.wow64, formFactors: h.formFactors || [],
    webglVendor, webglRenderer,
  });
})()`

// evaluateString 执行返回字符串的表达式（支持 Promise），脚本异常转为错误
func evaluateString(call cdpCaller, expr string) (string, error) {
	raw, err := call("Runtime.evaluate", map[string]any{"expression": expr, "awaitPromise": true, "returnByValue": true})
	if err != nil {
		return "", err
	}
	var res struct {
		Result struct {
			Value any `json:"value"`
		} `json:"result"`
		ExceptionDetails *struct {
			Text      string `json:"text"`
			Exception *struct {
				Description string `json:"description"`
			} `json:"exception"`
		} `json:"exceptionDetails"`
	}
	if err := json.Unmarshal(raw, &res); err != nil {
		return "", fmt.Errorf("decode evaluate result: %w", err)
	}
	if d := res.ExceptionDetails; d != nil {
		if d.Exception != nil {
			return "", fmt.Errorf("script error: %s", d.Exception.Description)
		}
		return "", fmt.Errorf("script error: %s", d.Text)
	}
	s, _ := res.Result.Value.(string)
	return s, nil
}

// initialPageTarget 启动时已有的第一个页面标签页，作为主页面
func initialPageTarget(conn *cdpConn) (string, error) {
	raw, err := conn.call("", "Target.getTargets", nil)
	if err != nil {
		return "", fmt.Errorf("list targets: %w", err)
	}
	var res struct {
		TargetInfos []struct {
			TargetID string `json:"targetId"`
			Type     string `json:"type"`
		} `json:"targetInfos"`
	}
	if err := json.Unmarshal(raw, &res); err != nil {
		return "", fmt.Errorf("decode targets: %w", err)
	}
	for _, ti := range res.TargetInfos {
		if ti.Type == "page" {
			return ti.TargetID, nil
		}
	}
	return "", fmt.Errorf("browser has no page target")
}

// readHostInfo 在指定标签页上读取真实浏览器的值，读完断开会话；必须在开启自动附加之前调用，
// 读取时这个页还没有被下发身份（不开临时页的原因见本任务开头）
func readHostInfo(conn *cdpConn, targetID string) (hostInfo, error) {
	raw, err := conn.call("", "Target.attachToTarget", map[string]any{"targetId": targetID, "flatten": true})
	if err != nil {
		return hostInfo{}, fmt.Errorf("attach initial page: %w", err)
	}
	var attached struct {
		SessionID string `json:"sessionId"`
	}
	if err := json.Unmarshal(raw, &attached); err != nil {
		return hostInfo{}, fmt.Errorf("decode initial page session: %w", err)
	}
	call := func(m string, p any) (json.RawMessage, error) { return conn.call(attached.SessionID, m, p) }
	value, err := evaluateString(call, hostInfoJS)
	// 先断开再处理结果：留着这个会话，自动附加后会有两个会话同时管同一个页
	if _, derr := conn.call("", "Target.detachFromTarget", map[string]any{"sessionId": attached.SessionID}); derr != nil && err == nil {
		err = fmt.Errorf("detach initial page: %w", derr)
	}
	if err != nil {
		return hostInfo{}, fmt.Errorf("read host info: %w", err)
	}
	return parseHostInfo(value)
}

// cdpStep 一条待发送的命令
type cdpStep struct {
	method string
	params any
}

// applyIdentity 在目标运行前下发身份：页面 / iframe 用 Emulation 原生覆盖并注册注入脚本；
// Worker 用 Network 覆盖 UA 与 client hints，再执行 Worker 脚本补齐 navigator
func applyIdentity(call cdpCaller, id *Identity, kind targetKind) error {
	ua := map[string]any{"userAgent": id.UserAgent, "acceptLanguage": id.AcceptLanguage, "platform": id.Platform, "userAgentMetadata": id.Metadata}
	tz := map[string]any{"timezoneId": id.Timezone}

	if kind == kindWorker {
		for _, s := range []cdpStep{{"Network.setUserAgentOverride", ua}, {"Emulation.setTimezoneOverride", tz}} {
			if _, err := call(s.method, s.params); err != nil {
				return fmt.Errorf("%s: %w", s.method, err)
			}
		}
		if _, err := evaluateString(call, identityScript(id, true)); err != nil {
			return fmt.Errorf("worker script: %w", err)
		}
		return nil
	}

	steps := []cdpStep{
		{"Page.enable", nil},
		{"Emulation.setUserAgentOverride", ua},
		{"Emulation.setTimezoneOverride", tz},
		{"Emulation.setLocaleOverride", map[string]any{"locale": id.Locale()}},
		{"Emulation.setHardwareConcurrencyOverride", map[string]any{"hardwareConcurrency": id.HardwareConcurrency}},
	}
	// iframe 不支持 setDeviceMetricsOverride（只能用于顶层目标），其屏幕与 DPR 由注入脚本补
	if kind == kindPage && id.Screen.Width > 0 {
		s := id.Screen
		steps = append(steps, cdpStep{"Emulation.setDeviceMetricsOverride", map[string]any{
			"width": s.InnerWidth, "height": s.InnerHeight, "deviceScaleFactor": s.DPR, "mobile": false,
			"screenWidth": s.Width, "screenHeight": s.Height,
		}})
	}
	steps = append(steps, cdpStep{"Page.addScriptToEvaluateOnNewDocument", map[string]any{"source": identityScript(id, false)}})

	for _, s := range steps {
		if _, err := call(s.method, s.params); err != nil {
			// 同一目标已设置过 locale 时 Chrome 会拒绝重复设置，身份已生效
			if s.method == "Emulation.setLocaleOverride" && strings.Contains(err.Error(), "Another locale override") {
				continue
			}
			return fmt.Errorf("%s: %w", s.method, err)
		}
	}
	return nil
}

// windowBounds Browser.setWindowBounds 的参数：最大化窗口的位置与外框尺寸
func windowBounds(id *Identity) map[string]any {
	return map[string]any{"left": 0, "top": id.Screen.AvailTop, "width": id.Screen.OuterWidth, "height": id.Screen.OuterHeight}
}

// identityForOptions 设了 FingerprintUserID 时由账号指纹派生身份，否则用真实浏览器身份（只去掉无界面痕迹）
func identityForOptions(opts *ConnectOptions, host hostInfo) (*Identity, error) {
	if opts == nil || opts.FingerprintUserID == "" {
		return hostIdentity(host), nil
	}
	dir := opts.FingerprintDir
	if dir == "" {
		dir = "./fingerprints"
	}
	manager, err := NewUserFingerprintManager(dir)
	if err != nil {
		return nil, fmt.Errorf("fingerprint manager: %w", err)
	}
	cfg, err := manager.GetOrCreateUserFingerprint(opts.FingerprintUserID, GetInitParamsFromOptions(opts))
	if err != nil {
		return nil, err
	}
	return identityFromConfig(cfg, host)
}
