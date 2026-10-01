package browser

import (
	"encoding/json"
	"fmt"
	"time"
)

// autoAttachParams 自动附加：新目标暂停在启动处，下发身份后再放行；flatten 让子会话共用同一条连接
var autoAttachParams = map[string]any{"autoAttach": true, "waitForDebuggerOnStart": true, "flatten": true}

// targetManager 浏览器级自动附加：每个新目标（页面、跨站 iframe、Worker、弹窗）在运行任何代码前下发身份
type targetManager struct {
	conn       *cdpConn
	identity   *Identity
	mainTarget string // 主页面的 targetId（启动时已有的标签页）
	mainPage   chan attachResult
}

// attachResult 主页面的附加结果
type attachResult struct {
	sessionID string
	err       error
}

// startTargetManager 在浏览器会话上开启自动附加，等主页面（mainTarget）下发完成后返回它的会话 ID
func startTargetManager(conn *cdpConn, id *Identity, mainTarget string) (string, error) {
	tm := &targetManager{conn: conn, identity: id, mainTarget: mainTarget, mainPage: make(chan attachResult, 1)}
	stop := conn.subscribe("*", "Target.attachedToTarget", tm.onAttached)
	if _, err := conn.call("", "Target.setAutoAttach", autoAttachParams); err != nil {
		stop()
		return "", fmt.Errorf("enable auto-attach: %w", err)
	}
	select {
	case r := <-tm.mainPage:
		if r.err != nil {
			stop()
			return "", fmt.Errorf("set up main page: %w", r.err)
		}
		return r.sessionID, nil
	case <-time.After(15 * time.Second):
		stop()
		return "", fmt.Errorf("main page %s not attached within 15s", mainTarget)
	}
}

// onAttached 处理一个新附加的目标：按类型下发身份、接管它的子目标，最后一定放行
func (tm *targetManager) onAttached(_ string, params json.RawMessage) {
	var ev struct {
		SessionID  string `json:"sessionId"`
		TargetInfo struct {
			TargetID string `json:"targetId"`
			Type     string `json:"type"`
			URL      string `json:"url"`
		} `json:"targetInfo"`
		WaitingForDebugger bool `json:"waitingForDebugger"`
	}
	if json.Unmarshal(params, &ev) != nil {
		return
	}
	call := func(m string, p any) (json.RawMessage, error) { return tm.conn.call(ev.SessionID, m, p) }

	kind, handled := targetKindOf(ev.TargetInfo.Type)
	var err error
	if handled {
		err = applyIdentity(call, tm.identity, kind)
		if err == nil && kind == kindPage && tm.identity.Screen.Width > 0 {
			err = tm.setWindowBounds(ev.TargetInfo.TargetID)
		}
		if _, e := call("Target.setAutoAttach", autoAttachParams); e != nil && err == nil {
			err = fmt.Errorf("Target.setAutoAttach: %w", e)
		}
	}
	// 无论下发成败都放行，否则目标会一直停在启动处
	if ev.WaitingForDebugger {
		call("Runtime.runIfWaitingForDebugger", nil)
	}

	// 主页面按 targetId 认定，不取"第一个附加的页面"：启动时可能有别的页面先被附加
	if ev.TargetInfo.TargetID == tm.mainTarget {
		tm.mainPage <- attachResult{sessionID: ev.SessionID, err: err}
		return
	}
	if err != nil {
		// 目标在下发过程中消失（iframe 被移除、弹窗被关闭）时会走到这里，只告警不影响其他目标
		fmt.Printf("Warning: identity setup for %s target %q failed: %v\n", ev.TargetInfo.Type, ev.TargetInfo.URL, err)
	}
}

// setWindowBounds 把页面所在窗口设为身份的最大化窗口（决定 outerWidth / outerHeight / screenX / screenY）
func (tm *targetManager) setWindowBounds(targetID string) error {
	raw, err := tm.conn.call("", "Browser.getWindowForTarget", map[string]any{"targetId": targetID})
	if err != nil {
		return fmt.Errorf("Browser.getWindowForTarget: %w", err)
	}
	var w struct {
		WindowID int `json:"windowId"`
	}
	if err := json.Unmarshal(raw, &w); err != nil {
		return fmt.Errorf("decode window: %w", err)
	}
	if _, err := tm.conn.call("", "Browser.setWindowBounds", map[string]any{"windowId": w.WindowID, "bounds": windowBounds(tm.identity)}); err != nil {
		return fmt.Errorf("Browser.setWindowBounds: %w", err)
	}
	return nil
}
