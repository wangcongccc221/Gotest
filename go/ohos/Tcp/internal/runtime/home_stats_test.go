package tcp

import (
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

func resetHomeStatsForTest(t *testing.T) {
	t.Helper()

	homeStatsMu.Lock()
	homeStatsConfig = homeStatsConfigState{}
	homeStatsHistory = homeStatsHistoryState{}
	homeStatsMu.Unlock()

	t.Cleanup(func() {
		homeStatsMu.Lock()
		homeStatsConfig = homeStatsConfigState{}
		homeStatsHistory = homeStatsHistoryState{}
		homeStatsMu.Unlock()
	})
}

func updateHomeStatsForTest(at time.Time, aggregate homeStatsAggregate) {
	homeStatsMu.Lock()
	updateHomeStatsHistoryLocked(at, aggregate, 0, 0)
	homeStatsMu.Unlock()
}

func snapshotHomeStatsHistoryForTest() homeStatsHistoryState {
	homeStatsMu.Lock()
	defer homeStatsMu.Unlock()
	return homeStatsHistory
}

func resetRealtimeSaveProcessForTest(t *testing.T) {
	t.Helper()

	realtimeSaveMu.Lock()
	previous := realtimeSaveProcessHistory
	realtimeSaveProcessHistory = realtimeSaveProcessSnapshot{}
	realtimeSaveMu.Unlock()

	t.Cleanup(func() {
		realtimeSaveMu.Lock()
		realtimeSaveProcessHistory = previous
		realtimeSaveMu.Unlock()
	})
}

func TestWebSocketConnectDoesNotResetHomeStatsEfficiencyWindow(t *testing.T) {
	resetHomeStatsForTest(t)

	start := time.Date(2026, 7, 6, 10, 0, 0, 0, time.UTC)
	updateHomeStatsForTest(start, homeStatsAggregate{
		TotalCount:     100,
		TotalCupNum:    100,
		TotalExitCount: 80,
		TotalWeight:    100 * homeStatsWeightScale,
		HasStats:       true,
	})
	updateHomeStatsForTest(start.Add(homeStatsHistoryInterval), homeStatsAggregate{
		TotalCount:     190,
		TotalCupNum:    200,
		TotalExitCount: 170,
		TotalWeight:    200 * homeStatsWeightScale,
		HasStats:       true,
	})

	before := snapshotHomeStatsHistoryForTest()
	if before.CupFillEfficiency != 90 {
		t.Fatalf("setup CupFillEfficiency = %.1f, want 90.0", before.CupFillEfficiency)
	}
	if !before.HasPrev {
		t.Fatalf("setup HasPrev = false, want true")
	}

	gin.SetMode(gin.ReleaseMode)
	router := gin.New()
	router.GET("/ws/data", handleWebSocketData)
	server := httptest.NewServer(router)
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http") + "/ws/data"
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("dial websocket: %v", err)
	}
	_, _, _ = conn.ReadMessage()
	_ = conn.Close()

	after := snapshotHomeStatsHistoryForTest()
	if !after.HasPrev {
		t.Fatalf("HasPrev = false after websocket connect, want true")
	}
	if after.CupFillEfficiency != before.CupFillEfficiency {
		t.Fatalf("CupFillEfficiency = %.1f after websocket connect, want %.1f", after.CupFillEfficiency, before.CupFillEfficiency)
	}
	if !after.PrevAt.Equal(before.PrevAt) {
		t.Fatalf("PrevAt = %s after websocket connect, want %s", after.PrevAt, before.PrevAt)
	}
}

func TestHomeStatsEfficiencyUsesSortedFruitCountLike48(t *testing.T) {
	resetHomeStatsForTest(t)

	start := time.Date(2026, 7, 15, 10, 0, 0, 0, time.UTC)
	updateHomeStatsForTest(start, homeStatsAggregate{
		TotalCount:     1000,
		TotalCupNum:    2000,
		TotalExitCount: 800,
		HasStats:       true,
	})
	updateHomeStatsForTest(start.Add(homeStatsHistoryInterval), homeStatsAggregate{
		TotalCount:     1090,
		TotalCupNum:    2120,
		TotalExitCount: 800,
		HasStats:       true,
	})

	history := snapshotHomeStatsHistoryForTest()
	const want = 75.0
	if history.CupFillEfficiency != want {
		t.Fatalf("CupFillEfficiency = %.1f, want %.1f from deltaCount/deltaCup like 48", history.CupFillEfficiency, want)
	}
}

func TestRealtimeSaveProcessEfficiencyUsesSortedFruitCountLike48(t *testing.T) {
	resetRealtimeSaveProcessForTest(t)

	start := time.Date(2026, 7, 15, 10, 0, 0, 0, time.UTC)
	if process := buildRealtimeSaveProcess(start, realtimeSaveAggregate{
		TotalCount:     1000,
		TotalCupNum:    2000,
		TotalExitCount: 800,
		HasStats:       true,
	}); process != nil {
		t.Fatalf("first process = %#v, want nil baseline sample", process)
	}

	process := buildRealtimeSaveProcess(start.Add(homeStatsHistoryInterval), realtimeSaveAggregate{
		TotalCount:     1090,
		TotalCupNum:    2120,
		TotalExitCount: 800,
		HasStats:       true,
	})
	if process == nil {
		t.Fatal("process = nil, want a completed efficiency window")
	}
	const want = 75.0
	if process.SeparationEfficiency != want {
		t.Fatalf("SeparationEfficiency = %.1f, want %.1f from deltaCount/deltaCup like 48", process.SeparationEfficiency, want)
	}
}

func TestHomeStatsRealtimeOutputUsesActualElapsedWindow(t *testing.T) {
	resetHomeStatsForTest(t)

	start := time.Date(2026, 7, 6, 10, 0, 0, 0, time.UTC)
	updateHomeStatsForTest(start, homeStatsAggregate{
		TotalCount:     100,
		TotalCupNum:    100,
		TotalExitCount: 80,
		TotalWeight:    100 * homeStatsWeightScale,
		HasStats:       true,
	})
	updateHomeStatsForTest(start.Add(30*time.Second), homeStatsAggregate{
		TotalCount:     200,
		TotalCupNum:    200,
		TotalExitCount: 170,
		TotalWeight:    110 * homeStatsWeightScale,
		HasStats:       true,
	})

	history := snapshotHomeStatsHistoryForTest()
	const want = 1200.0
	if history.RealtimeOutputTonPerHour != want {
		t.Fatalf("RealtimeOutputTonPerHour = %.1f, want %.1f", history.RealtimeOutputTonPerHour, want)
	}
}
