package browser

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

// fakeCDPServer 模拟 flatten 模式的 CDP：每收到一条命令，先推送会话 A、B 各一个事件，再回显会话和方法名作为响应
func fakeCDPServer(t *testing.T) string {
	t.Helper()
	up := websocket.Upgrader{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ws, err := up.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer ws.Close()
		for {
			var req map[string]any
			if ws.ReadJSON(&req) != nil {
				return
			}
			sess, _ := req["sessionId"].(string)
			ws.WriteJSON(map[string]any{"method": "Test.event", "sessionId": "A", "params": map[string]any{"n": 1}})
			ws.WriteJSON(map[string]any{"method": "Test.event", "sessionId": "B", "params": map[string]any{"n": 2}})
			ws.WriteJSON(map[string]any{"id": req["id"], "result": map[string]any{"session": sess, "method": req["method"]}})
		}
	}))
	t.Cleanup(srv.Close)
	return "ws" + strings.TrimPrefix(srv.URL, "http")
}

// TestCDPConnRouting 响应按 id 回到调用方并带上会话；事件只投递给对应会话的订阅者，"*" 订阅收到全部；取消订阅后不再投递
func TestCDPConnRouting(t *testing.T) {
	c, err := dialCDP(fakeCDPServer(t))
	if err != nil {
		t.Fatal(err)
	}
	defer c.close()

	gotA := make(chan string, 8)
	gotAll := make(chan string, 8)
	stopA := c.subscribe("A", "Test.event", func(s string, _ json.RawMessage) { gotA <- s })
	c.subscribe("*", "Test.event", func(s string, _ json.RawMessage) { gotAll <- s })

	raw, err := c.call("B", "Test.ping", nil)
	if err != nil {
		t.Fatalf("call: %v", err)
	}
	var res struct{ Session, Method string }
	if err := json.Unmarshal(raw, &res); err != nil {
		t.Fatal(err)
	}
	if res.Session != "B" || res.Method != "Test.ping" {
		t.Errorf("result = %+v, want session B / Test.ping", res)
	}

	// "*" 订阅应收到 A、B 两个事件；A 订阅只收到 A
	seen := map[string]bool{}
	for range 2 {
		select {
		case s := <-gotAll:
			seen[s] = true
		case <-time.After(2 * time.Second):
			t.Fatal("wildcard subscriber missed events")
		}
	}
	if !seen["A"] || !seen["B"] {
		t.Errorf("wildcard saw %v, want A and B", seen)
	}
	select {
	case s := <-gotA:
		if s != "A" {
			t.Errorf("session A subscriber got event from %q", s)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("session A subscriber missed its event")
	}

	stopA()
	if _, err := c.call("", "Test.ping", nil); err != nil {
		t.Fatal(err)
	}
	select {
	case s := <-gotA:
		t.Errorf("unsubscribed handler still received event from %q", s)
	case <-time.After(300 * time.Millisecond):
	}
}
