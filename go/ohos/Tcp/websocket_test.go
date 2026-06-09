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

func TestParseWebSocketControlMessageReadsFruitInfoQuery(t *testing.T) {
	message, ok := parseWebSocketControlMessage(`{
		"type": "queryFruitInfo",
		"requestId": "history-query-1",
		"fruitInfoQuery": {
			"StartTime": "2026-06-01 00:00:00",
			"EndTime": "2026-06-08 23:59:59",
			"CustomerName": "客户A",
			"FarmName": "农场B",
			"FruitName": "苹果",
			"FVisible": 1,
			"PageIndex": 1,
			"PageSize": 200
		}
	}`)
	if !ok {
		t.Fatal("parseWebSocketControlMessage() rejected queryFruitInfo")
	}
	if message.RequestID != "history-query-1" {
		t.Fatalf("RequestID = %q, want history-query-1", message.RequestID)
	}
	if message.FruitInfoQuery == nil {
		t.Fatal("FruitInfoQuery is nil")
	}
	if message.FruitInfoQuery.StartTime == nil || *message.FruitInfoQuery.StartTime != "2026-06-01 00:00:00" {
		t.Fatalf("StartTime = %#v, want 2026-06-01 00:00:00", message.FruitInfoQuery.StartTime)
	}
	if message.FruitInfoQuery.EndTime == nil || *message.FruitInfoQuery.EndTime != "2026-06-08 23:59:59" {
		t.Fatalf("EndTime = %#v, want 2026-06-08 23:59:59", message.FruitInfoQuery.EndTime)
	}
	if message.FruitInfoQuery.CustomerName == nil || *message.FruitInfoQuery.CustomerName != "客户A" {
		t.Fatalf("CustomerName = %#v, want 客户A", message.FruitInfoQuery.CustomerName)
	}
	if message.FruitInfoQuery.FVisible == nil || *message.FruitInfoQuery.FVisible != 1 {
		t.Fatalf("FVisible = %#v, want 1", message.FruitInfoQuery.FVisible)
	}
	if message.FruitInfoQuery.PageSize == nil || *message.FruitInfoQuery.PageSize != 200 {
		t.Fatalf("PageSize = %#v, want 200", message.FruitInfoQuery.PageSize)
	}
}

func TestParseWebSocketControlMessageReadsFruitInfoDeleteCustomerIDs(t *testing.T) {
	message, ok := parseWebSocketControlMessage(`{
		"type": "deleteFruitInfo",
		"requestId": "history-delete-1",
		"fruitInfoDeleteCustomerIds": [12, 0, -4, 13]
	}`)
	if !ok {
		t.Fatal("parseWebSocketControlMessage() rejected deleteFruitInfo")
	}
	if message.RequestID != "history-delete-1" {
		t.Fatalf("RequestID = %q, want history-delete-1", message.RequestID)
	}
	if len(message.FruitInfoDeleteCustomerIDs) != 4 {
		t.Fatalf("FruitInfoDeleteCustomerIDs length = %d, want 4", len(message.FruitInfoDeleteCustomerIDs))
	}
	if message.FruitInfoDeleteCustomerIDs[0] != 12 || message.FruitInfoDeleteCustomerIDs[3] != 13 {
		t.Fatalf("FruitInfoDeleteCustomerIDs = %#v, want [12 0 -4 13]", message.FruitInfoDeleteCustomerIDs)
	}
}

func TestParseWebSocketControlMessageReadsExitAdditionalTextData(t *testing.T) {
	message, ok := parseWebSocketControlMessage(`{
		"type": "saveExitAdditionalText",
		"fsmId": 256,
		"exitAdditionalText": {
			"additionalTexts": ["一号备注", "二号备注"]
		}
	}`)
	if !ok {
		t.Fatal("parseWebSocketControlMessage() rejected saveExitAdditionalText")
	}
	if message.ExitAdditionalText == nil {
		t.Fatal("ExitAdditionalText is nil")
	}
	if message.FSMID != 256 {
		t.Fatalf("FSMID = %d, want 256", message.FSMID)
	}
	if len(message.ExitAdditionalText.AdditionalTexts) != 2 {
		t.Fatalf("AdditionalTexts length = %d, want 2", len(message.ExitAdditionalText.AdditionalTexts))
	}
	if message.ExitAdditionalText.AdditionalTexts[0] != "一号备注" ||
		message.ExitAdditionalText.AdditionalTexts[1] != "二号备注" {
		t.Fatalf("AdditionalTexts[0,1] = %q, %q",
			message.ExitAdditionalText.AdditionalTexts[0],
			message.ExitAdditionalText.AdditionalTexts[1])
	}
}

func TestParseWebSocketControlMessageReadsFruitTypeConfig(t *testing.T) {
	message, ok := parseWebSocketControlMessage(`{
		"type": "saveFruitTypeConfig",
		"fsmId": 256,
		"fruitTypeConfig": {
			"majorTypes": "1.脐橙(1-8);",
			"selectedFruitTypes": "1.1-新鲜脐橙;",
			"subTypeConfigs": {
				"1.脐橙(1-8)": "1.1-新鲜脐橙;"
			}
		}
	}`)
	if !ok {
		t.Fatal("parseWebSocketControlMessage() rejected saveFruitTypeConfig")
	}
	if message.FruitTypeConfig == nil {
		t.Fatal("FruitTypeConfig is nil")
	}
	if message.FruitTypeConfig.MajorTypes == nil || *message.FruitTypeConfig.MajorTypes != "1.脐橙(1-8);" {
		t.Fatalf("MajorTypes = %v, want 1.脐橙(1-8);", message.FruitTypeConfig.MajorTypes)
	}
	if message.FruitTypeConfig.SelectedFruitTypes == nil || *message.FruitTypeConfig.SelectedFruitTypes != "1.1-新鲜脐橙;" {
		t.Fatalf("SelectedFruitTypes = %v, want 1.1-新鲜脐橙;", message.FruitTypeConfig.SelectedFruitTypes)
	}
	if message.FruitTypeConfig.SubTypeConfigs == nil || (*message.FruitTypeConfig.SubTypeConfigs)["1.脐橙(1-8)"] != "1.1-新鲜脐橙;" {
		t.Fatalf("SubTypeConfigs = %#v, want configured major", message.FruitTypeConfig.SubTypeConfigs)
	}
}

func TestParseWebSocketControlMessageReadsDensityInfo(t *testing.T) {
	message, ok := parseWebSocketControlMessage(`{
		"type": "saveDensityInfo",
		"fsmId": 256,
		"densityInfo": {
			"UAnalogDensity": [1.25, 2.5, 3.75]
		}
	}`)
	if !ok {
		t.Fatal("parseWebSocketControlMessage() rejected saveDensityInfo")
	}
	if message.DensityInfo == nil {
		t.Fatal("DensityInfo is nil")
	}
	if message.DensityInfo.UAnalogDensity[0] != 1.25 ||
		message.DensityInfo.UAnalogDensity[1] != 2.5 ||
		message.DensityInfo.UAnalogDensity[2] != 3.75 {
		t.Fatalf("UAnalogDensity[0:3] = %.2f %.2f %.2f, want 1.25 2.50 3.75",
			message.DensityInfo.UAnalogDensity[0],
			message.DensityInfo.UAnalogDensity[1],
			message.DensityInfo.UAnalogDensity[2])
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

func TestApplyExitAdditionalTextUpdateOnlyReplacesProvidedTexts(t *testing.T) {
	base := defaultExitAdditionalTextInfo()
	base.AdditionalTexts[0] = "旧一号"
	base.AdditionalTexts[1] = "旧二号"

	next := applyExitAdditionalTextUpdate(base, webSocketExitAdditionalTextControl{
		AdditionalTexts: []string{"新一号"},
	})

	if next.AdditionalTexts[0] != "新一号" {
		t.Fatalf("AdditionalTexts[0] = %q, want 新一号", next.AdditionalTexts[0])
	}
	if next.AdditionalTexts[1] != "旧二号" {
		t.Fatalf("AdditionalTexts[1] = %q, want preserved", next.AdditionalTexts[1])
	}
}
