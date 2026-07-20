package tcp

import (
	"encoding/json"
	"testing"
	"time"
)

func TestRequestStGlobalRoutesToRequestedFSM(t *testing.T) {
	originalRequest := requestStGlobalFromFSM
	t.Cleanup(func() {
		requestStGlobalFromFSM = originalRequest
	})

	requestedFSM := make(chan int32, 1)
	requestStGlobalFromFSM = func(fsmID int32) int {
		requestedFSM <- fsmID
		return 0
	}

	testCases := []struct {
		name    string
		payload string
		want    int32
	}{
		{name: "fsm 2", payload: `{"type":"requestStGlobal","fsmId":512}`, want: 0x0200},
		{name: "fsm 4", payload: `{"type":"requestStGlobal","fsmId":1024}`, want: 0x0400},
		{name: "missing fsm id", payload: `{"type":"requestStGlobal"}`, want: cTCPDefaultFSMID},
		{name: "channel device id", payload: `{"type":"requestStGlobal","fsmId":529}`, want: cTCPDefaultFSMID},
		{name: "fsm outside configured range", payload: `{"type":"requestStGlobal","fsmId":1280}`, want: cTCPDefaultFSMID},
	}

	client := &webSocketClient{send: make(chan []byte, 8)}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			client.handleIncoming([]byte(testCase.payload))

			select {
			case got := <-requestedFSM:
				if got != testCase.want {
					t.Fatalf("requested FSM = 0x%04X, want 0x%04X", uint32(got), uint32(testCase.want))
				}
			case <-time.After(time.Second):
				t.Fatal("timed out waiting for requestStGlobal dispatch")
			}
		})
	}
}

func TestRequestStGlobalReplaysCachedLocalConfig(t *testing.T) {
	originalRequest := requestStGlobalFromFSM
	requestStGlobalFromFSM = func(int32) int { return 0 }
	t.Cleanup(func() {
		requestStGlobalFromFSM = originalRequest
	})

	lastExitDisplayInfoSnapshotMu.Lock()
	originalInfo := lastExitDisplayInfoSnapshot
	originalFSMID := lastExitDisplayInfoFSMID
	originalOK := lastExitDisplayInfoSnapshotOK
	lastExitDisplayInfoSnapshot = ExitDisplayInfo{}
	lastExitDisplayInfoFSMID = cTCPDefaultFSMID
	lastExitDisplayInfoSnapshotOK = true
	lastExitDisplayInfoSnapshotMu.Unlock()
	t.Cleanup(func() {
		lastExitDisplayInfoSnapshotMu.Lock()
		lastExitDisplayInfoSnapshot = originalInfo
		lastExitDisplayInfoFSMID = originalFSMID
		lastExitDisplayInfoSnapshotOK = originalOK
		lastExitDisplayInfoSnapshotMu.Unlock()
	})

	client := &webSocketClient{send: make(chan []byte, 8)}
	client.handleRequestStGlobal(webSocketControlMessage{FSMID: cTCPDefaultFSMID})

	// FSM 未连接时，requestStGlobal 必须立即回放缓存帧（出口卡片等依赖它）。
	for {
		select {
		case payload := <-client.send:
			var frame webSocketFrame
			if err := json.Unmarshal(payload, &frame); err != nil {
				t.Fatalf("invalid frame payload: %v", err)
			}
			if frame.Topic == webSocketTopicExitDisplay {
				return
			}
		default:
			t.Fatal("requestStGlobal did not replay cached exitDisplay frame")
		}
	}
}
