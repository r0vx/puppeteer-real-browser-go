package browser

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"slices"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gorilla/websocket"
)

// cdpCommandTimeout 单条 CDP 命令的等待上限
const cdpCommandTimeout = 30 * time.Second

// cdpConn 一条 CDP WebSocket 连接（flatten 模式）：响应按 id 匹配，事件按 (sessionId, method) 分发。
// sessionId 为空表示浏览器级会话，或直连页面 WebSocket 时的页面本身
type cdpConn struct {
	ws      *websocket.Conn
	writeMu sync.Mutex
	nextID  atomic.Int64

	mu       sync.Mutex
	pending  map[int64]chan cdpReply
	handlers map[string][]eventSub // key 见 handlerKey；会话为 "*" 时匹配所有会话
	nextSub  int64

	ctx    context.Context
	cancel context.CancelFunc
}

// cdpReply 命令响应
type cdpReply struct {
	Result json.RawMessage
	Error  *CDPError
}

// eventSub 一个事件订阅，id 用于取消
type eventSub struct {
	id int64
	fn func(session string, params json.RawMessage)
}

// handlerKey 订阅表的键
func handlerKey(session, method string) string { return session + "\x00" + method }

// dialCDP 连接 CDP WebSocket 并启动读循环
func dialCDP(wsURL string) (*cdpConn, error) {
	ws, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		return nil, fmt.Errorf("dial %s: %w", wsURL, err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	c := &cdpConn{
		ws:       ws,
		pending:  make(map[int64]chan cdpReply),
		handlers: make(map[string][]eventSub),
		ctx:      ctx,
		cancel:   cancel,
	}
	go c.readLoop()
	return c, nil
}

// readLoop 读取消息直到连接关闭：带 id 的是响应，其余按会话和方法分发事件
func (c *cdpConn) readLoop() {
	defer c.cancel()
	for {
		_, data, err := c.ws.ReadMessage()
		if err != nil {
			return
		}
		var msg struct {
			ID        int64           `json:"id"`
			Method    string          `json:"method"`
			SessionID string          `json:"sessionId"`
			Params    json.RawMessage `json:"params"`
			Result    json.RawMessage `json:"result"`
			Error     *CDPError       `json:"error"`
		}
		if json.Unmarshal(data, &msg) != nil {
			continue
		}
		if msg.ID != 0 {
			c.mu.Lock()
			ch := c.pending[msg.ID]
			delete(c.pending, msg.ID)
			c.mu.Unlock()
			if ch != nil {
				ch <- cdpReply{Result: msg.Result, Error: msg.Error} // 缓冲为 1，不会阻塞
			}
			continue
		}
		if msg.Method == "" {
			continue
		}
		c.mu.Lock()
		subs := slices.Concat(c.handlers[handlerKey(msg.SessionID, msg.Method)], c.handlers[handlerKey("*", msg.Method)])
		c.mu.Unlock()
		if len(subs) > 0 {
			// 异步执行，避免处理器里再发命令时阻塞读循环
			go func() {
				for _, s := range subs {
					s.fn(msg.SessionID, msg.Params)
				}
			}()
		}
	}
}

// call 在指定会话上发送命令并等待响应
func (c *cdpConn) call(session, method string, params any) (json.RawMessage, error) {
	id := c.nextID.Add(1)
	ch := make(chan cdpReply, 1)
	c.mu.Lock()
	c.pending[id] = ch
	c.mu.Unlock()
	defer func() {
		c.mu.Lock()
		delete(c.pending, id)
		c.mu.Unlock()
	}()

	msg := map[string]any{"id": id, "method": method}
	if params != nil {
		msg["params"] = params
	}
	if session != "" {
		msg["sessionId"] = session
	}
	c.writeMu.Lock()
	err := c.ws.WriteJSON(msg)
	c.writeMu.Unlock()
	if err != nil {
		return nil, fmt.Errorf("send %s: %w", method, err)
	}

	timer := time.NewTimer(cdpCommandTimeout)
	defer timer.Stop()
	select {
	case r := <-ch:
		if r.Error != nil {
			return nil, fmt.Errorf("CDP error: %s", r.Error.Message)
		}
		return r.Result, nil
	case <-timer.C:
		return nil, fmt.Errorf("timeout waiting for response to %s", method)
	case <-c.ctx.Done():
		return nil, fmt.Errorf("connection closed")
	}
}

// subscribe 订阅事件，session 为 "*" 时匹配所有会话；返回取消函数
func (c *cdpConn) subscribe(session, method string, fn func(session string, params json.RawMessage)) (unsubscribe func()) {
	key := handlerKey(session, method)
	c.mu.Lock()
	c.nextSub++
	id := c.nextSub
	c.handlers[key] = append(c.handlers[key], eventSub{id: id, fn: fn})
	c.mu.Unlock()
	return func() {
		c.mu.Lock()
		defer c.mu.Unlock()
		c.handlers[key] = slices.DeleteFunc(c.handlers[key], func(s eventSub) bool { return s.id == id })
	}
}

// close 关闭连接
func (c *cdpConn) close() error {
	c.cancel()
	return c.ws.Close()
}

// dialBrowser 连接浏览器级 CDP WebSocket（/json/version 中的 webSocketDebuggerUrl）
func dialBrowser(port int) (*cdpConn, error) {
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Get(fmt.Sprintf("http://localhost:%d/json/version", port))
	if err != nil {
		return nil, fmt.Errorf("get browser endpoint: %w", err)
	}
	defer resp.Body.Close()
	var v struct {
		WebSocketDebuggerURL string `json:"webSocketDebuggerUrl"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&v); err != nil {
		return nil, fmt.Errorf("decode browser endpoint: %w", err)
	}
	if v.WebSocketDebuggerURL == "" {
		return nil, fmt.Errorf("browser endpoint has no webSocketDebuggerUrl")
	}
	return dialCDP(v.WebSocketDebuggerURL)
}
