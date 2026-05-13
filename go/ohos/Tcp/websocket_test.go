package tcp

import (
	"encoding/json"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gorilla/websocket"
)

func testWebSocketURL(httpURL string) string {
	return "ws" + strings.TrimPrefix(httpURL, "http") + "/ws"
}

func readWebSocketFrame(t *testing.T, conn *websocket.Conn) webSocketFrame {
	t.Helper()

	_, payload, err := conn.ReadMessage()
	if err != nil {
		t.Fatalf("read websocket message: %v", err)
	}

	var frame webSocketFrame
	if err := json.Unmarshal(payload, &frame); err != nil {
		t.Fatalf("unmarshal websocket frame %s: %v", string(payload), err)
	}
	return frame
}

func TestWebSocketRouteRespondsToPing(t *testing.T) {
	server := httptest.NewServer(newRouter())
	defer server.Close()

	conn, _, err := websocket.DefaultDialer.Dial(testWebSocketURL(server.URL), nil)
	if err != nil {
		t.Fatalf("dial websocket route: %v", err)
	}
	defer conn.Close()

	ready := readWebSocketFrame(t, conn)
	if ready.Type != "ready" {
		t.Fatalf("first frame type = %q, want ready", ready.Type)
	}

	if err := conn.WriteMessage(websocket.TextMessage, []byte(`{"type":"ping","requestId":"req-1"}`)); err != nil {
		t.Fatalf("write ping frame: %v", err)
	}

	pong := readWebSocketFrame(t, conn)
	if pong.Type != "pong" {
		t.Fatalf("response type = %q, want pong", pong.Type)
	}
	if pong.RequestID != "req-1" {
		t.Fatalf("requestId = %q, want req-1", pong.RequestID)
	}
}

func TestPublishWebSocketJSONPushesRawJSONData(t *testing.T) {
	server := httptest.NewServer(newRouter())
	defer server.Close()

	conn, _, err := websocket.DefaultDialer.Dial(testWebSocketURL(server.URL), nil)
	if err != nil {
		t.Fatalf("dial websocket route: %v", err)
	}
	defer conn.Close()

	ready := readWebSocketFrame(t, conn)
	if ready.Type != "ready" {
		t.Fatalf("first frame type = %q, want ready", ready.Type)
	}

	if err := PublishWebSocketJSON(webSocketTopicStatistics, `{"speed":42,"nested":{"ok":true}}`); err != nil {
		t.Fatalf("publish websocket json: %v", err)
	}

	frame := readWebSocketFrame(t, conn)
	if frame.Type != "data" {
		t.Fatalf("published frame type = %q, want data", frame.Type)
	}
	if frame.Topic != webSocketTopicStatistics {
		t.Fatalf("topic = %q, want %q", frame.Topic, webSocketTopicStatistics)
	}

	var data map[string]any
	if err := json.Unmarshal(frame.Data, &data); err != nil {
		t.Fatalf("published data is not raw JSON object: %v", err)
	}
	if data["speed"] != float64(42) {
		t.Fatalf("speed = %v, want 42", data["speed"])
	}
}

func TestPublishWebSocketJSONRejectsInvalidJSON(t *testing.T) {
	if err := PublishWebSocketJSON("bad", `{"broken"`); err == nil {
		t.Fatal("PublishWebSocketJSON accepted invalid JSON")
	}
}
