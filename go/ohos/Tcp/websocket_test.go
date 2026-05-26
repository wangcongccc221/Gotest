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

func TestWebSocketRouteDoesNotTreatBusinessJSONAsControlRequest(t *testing.T) {
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

	if err := conn.WriteMessage(websocket.TextMessage, []byte(`{"Type":"MachineData","Speed":42}`)); err != nil {
		t.Fatalf("write business json frame: %v", err)
	}

	echo := readWebSocketFrame(t, conn)
	if echo.Type != "echo" {
		t.Fatalf("response type = %q, want echo", echo.Type)
	}

	var data map[string]any
	if err := json.Unmarshal(echo.Data, &data); err != nil {
		t.Fatalf("echo data is not raw json object: %v", err)
	}
	if data["Type"] != "MachineData" {
		t.Fatalf("Type = %v, want MachineData", data["Type"])
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

func TestParseWebSocketControlMessageReadsExitInfosFieldsByPresence(t *testing.T) {
	message, ok := parseWebSocketControlMessage(`{
		"type": "saveExitInfos",
		"fsmId": 256,
		"exitInfos": {
			"ExitName": [65],
			"ExitBoxType": [1]
		}
	}`)
	if !ok {
		t.Fatal("parseWebSocketControlMessage() rejected saveExitInfos")
	}
	if message.ExitInfos == nil {
		t.Fatal("ExitInfos is nil")
	}
	if message.ExitInfos.ExitName == nil || len(message.ExitInfos.ExitName) != 1 || message.ExitInfos.ExitName[0] != 'A' {
		t.Fatalf("ExitName = %#v, want [65]", message.ExitInfos.ExitName)
	}
	if message.ExitInfos.ExitControlMode != nil {
		t.Fatalf("ExitControlMode = %#v, want omitted nil", message.ExitInfos.ExitControlMode)
	}
	if message.ExitInfos.ExitBoxType == nil || len(message.ExitInfos.ExitBoxType) != 1 || message.ExitInfos.ExitBoxType[0] != 1 {
		t.Fatalf("ExitBoxType = %#v, want [1]", message.ExitInfos.ExitBoxType)
	}
}

func TestParseWebSocketControlMessageReadsExitDisplayData(t *testing.T) {
	message, ok := parseWebSocketControlMessage(`{
		"type": "saveExitDisplay",
		"fsmId": 7,
		"exitDisplay": {
			"displayType": 3,
			"displayNames": ["一号屏", "二号屏"]
		}
	}`)
	if !ok {
		t.Fatal("parseWebSocketControlMessage() rejected saveExitDisplay")
	}
	if message.ExitDisplay == nil {
		t.Fatal("ExitDisplay is nil")
	}
	if message.FSMID != 7 {
		t.Fatalf("FSMID = %d, want 7", message.FSMID)
	}
	if message.ExitDisplay.DisplayType == nil || *message.ExitDisplay.DisplayType != 3 {
		t.Fatalf("DisplayType = %v, want 3", message.ExitDisplay.DisplayType)
	}
	if len(message.ExitDisplay.DisplayNames) != 2 {
		t.Fatalf("DisplayNames length = %d, want 2", len(message.ExitDisplay.DisplayNames))
	}
	if message.ExitDisplay.DisplayNames[0] != "一号屏" || message.ExitDisplay.DisplayNames[1] != "二号屏" {
		t.Fatalf("DisplayNames[0,1] = %q, %q", message.ExitDisplay.DisplayNames[0], message.ExitDisplay.DisplayNames[1])
	}
}

func TestApplyExitDisplayUpdatePreservesOmittedFieldsAndAllowsZeroDisplayType(t *testing.T) {
	base := defaultExitDisplayInfo()
	base.DisplayType = 7
	base.DisplayNames[2] = "保留"
	zero := int64(0)

	next := applyExitDisplayUpdate(base, webSocketExitDisplayControl{DisplayType: &zero})

	if next.DisplayType != 0 {
		t.Fatalf("DisplayType = %d, want explicit zero", next.DisplayType)
	}
	if next.DisplayNames[2] != "保留" {
		t.Fatalf("DisplayNames[2] = %q, want preserved", next.DisplayNames[2])
	}
}

func TestApplyExitDisplayUpdateOnlyReplacesProvidedDisplayNames(t *testing.T) {
	base := defaultExitDisplayInfo()
	base.DisplayNames[0] = "旧一号"
	base.DisplayNames[1] = "旧二号"

	next := applyExitDisplayUpdate(base, webSocketExitDisplayControl{
		DisplayNames: []string{"新一号"},
	})

	if next.DisplayNames[0] != "新一号" {
		t.Fatalf("DisplayNames[0] = %q, want 新一号", next.DisplayNames[0])
	}
	if next.DisplayNames[1] != "旧二号" {
		t.Fatalf("DisplayNames[1] = %q, want preserved", next.DisplayNames[1])
	}
}
