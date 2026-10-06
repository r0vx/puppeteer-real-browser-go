package browser

import (
	"encoding/json"
	"fmt"
	"slices"
	"sync"
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

	mu       sync.Mutex
	pages    map[string]pageSession        // 已下发身份的页面目标：targetId → 会话
	sessions map[string]string             // 会话 → targetId（断开事件按会话清理）
	waiters  map[string][]chan pageSession // 等某个页面目标的调用方
}

// pageSession 一个页面目标的会话与下发结果
type pageSession struct {
	sessionID string
	err       error
}

// attachResult 主页面的附加结果
type attachResult struct {
	sessionID string
	err       error
}

// startTargetManager 在浏览器会话上开启自动附加，等主页面（mainTarget）下发完成后返回管理器与主页面会话 ID
func startTargetManager(conn *cdpConn, id *Identity, mainTarget string) (*targetManager, string, error) {
	tm := &targetManager{conn: conn, identity: id, mainTarget: mainTarget, mainPage: make(chan attachResult, 1),
		pages: map[string]pageSession{}, sessions: map[string]string{}, waiters: map[string][]chan pageSession{}}
	stopAttach := conn.subscribe("*", "Target.attachedToTarget", tm.onAttached)
	stopDetach := conn.subscribe("*", "Target.detachedFromTarget", tm.forgetTarget)
	stop := func() { stopAttach(); stopDetach() }
	if _, err := conn.call("", "Target.setAutoAttach", autoAttachParams); err != nil {
		stop()
		return nil, "", fmt.Errorf("enable auto-attach: %w", err)
	}
	select {
	case r := <-tm.mainPage:
		if r.err != nil {
			stop()
			return nil, "", fmt.Errorf("set up main page: %w", r.err)
		}
		return tm, r.sessionID, nil
	case <-time.After(15 * time.Second):
		stop()
		return nil, "", fmt.Errorf("main page %s not attached within 15s", mainTarget)
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
			err = setTargetWindowBounds(tm.browserCall, ev.TargetInfo.TargetID, windowBounds(tm.identity))
		}
		if _, e := call("Target.setAutoAttach", autoAttachParams); e != nil && err == nil {
			err = fmt.Errorf("Target.setAutoAttach: %w", e)
		}
	}
	// 无论下发成败都放行，否则目标会一直停在启动处
	if ev.WaitingForDebugger {
		call("Runtime.runIfWaitingForDebugger", nil)
	}

	// 页面目标登记会话，供 openTab 按 targetId 取用（主页面也登记，无害）
	if handled && kind == kindPage {
		tm.recordPage(ev.TargetInfo.TargetID, pageSession{sessionID: ev.SessionID, err: err})
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

// browserCall 在浏览器级会话上发命令
func (tm *targetManager) browserCall(method string, params any) (json.RawMessage, error) {
	return tm.conn.call("", method, params)
}

// recordPage 登记页面目标的会话，并唤醒等它的调用方
func (tm *targetManager) recordPage(targetID string, p pageSession) {
	tm.mu.Lock()
	defer tm.mu.Unlock()
	tm.pages[targetID] = p
	tm.sessions[p.sessionID] = targetID
	for _, ch := range tm.waiters[targetID] {
		ch <- p // 缓冲为 1，不会阻塞
	}
	delete(tm.waiters, targetID)
}

// forgetTarget 目标断开（标签页关闭、iframe 移除等）时移除登记，并清掉这个会话在连接上的全部事件订阅
func (tm *targetManager) forgetTarget(_ string, params json.RawMessage) {
	var ev struct {
		SessionID string `json:"sessionId"`
		TargetID  string `json:"targetId"` // CDP 已标记废弃，会话查不到时才用
	}
	if json.Unmarshal(params, &ev) != nil {
		return
	}
	tm.mu.Lock()
	target, ok := tm.sessions[ev.SessionID]
	if !ok {
		target = ev.TargetID
	}
	delete(tm.sessions, ev.SessionID)
	delete(tm.pages, target)
	tm.mu.Unlock()
	if ev.SessionID != "" {
		tm.conn.dropSession(ev.SessionID)
	}
}

// waitPage 等页面目标下发完身份，返回它的会话；超时报错
func (tm *targetManager) waitPage(targetID string, timeout time.Duration) (string, error) {
	tm.mu.Lock()
	if p, ok := tm.pages[targetID]; ok {
		tm.mu.Unlock()
		return p.sessionID, p.err
	}
	ch := make(chan pageSession, 1)
	tm.waiters[targetID] = append(tm.waiters[targetID], ch)
	tm.mu.Unlock()
	select {
	case p := <-ch:
		return p.sessionID, p.err
	case <-time.After(timeout):
		tm.mu.Lock()
		tm.waiters[targetID] = slices.DeleteFunc(tm.waiters[targetID], func(c chan pageSession) bool { return c == ch })
		tm.mu.Unlock()
		return "", fmt.Errorf("page %s not attached within %v", targetID, timeout)
	}
}
